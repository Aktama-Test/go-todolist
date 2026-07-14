#!/usr/bin/env bash
# Generates a Gradle multi-project testcontainers benchmark mirroring GoodNotes'
# run-test: parallel workers, 1 fork/task, -Xmx4g, many DB-backed test modules,
# each standing up a testcontainer (no reuse) AND doing real DB work
# (schema + batched inserts + queries) so we measure I/O, not just startup.
set -euo pipefail
ROOT=${ROOT:-genproj}
NCOCKROACH=${NCOCKROACH:-19}
NPOSTGRES=${NPOSTGRES:-4}
NVALKEY=${NVALKEY:-2}
ROWS=${ROWS:-5000}
QUERIES=${QUERIES:-50}
TC=1.20.4
rm -rf "$ROOT"; mkdir -p "$ROOT"

cat > "$ROOT/gradle.properties" <<'EOF'
org.gradle.parallel=true
org.gradle.caching=false
org.gradle.jvmargs=-Xmx4096m -XX:MaxMetaspaceSize=512m
EOF

echo "rootProject.name = 'tc-bench'" > "$ROOT/settings.gradle"

cat > "$ROOT/build.gradle" <<EOF
subprojects {
  apply plugin: 'java'
  repositories { mavenCentral() }
  dependencies {
    testImplementation 'org.junit.jupiter:junit-jupiter:5.10.2'
    testImplementation 'org.testcontainers:testcontainers:$TC'
    testImplementation 'org.postgresql:postgresql:42.7.4'
    testRuntimeOnly 'org.junit.platform:junit-platform-launcher'
    testRuntimeOnly 'org.slf4j:slf4j-simple:2.0.13'
  }
  test {
    useJUnitPlatform()
    testLogging { showStandardStreams = true; exceptionFormat 'full' }
  }
}
EOF

# sql module: start container + create table + batch insert ROWS + QUERIES selects
gen_sql() {
  local name=$1 image=$2 port=$3 extra=$4 db=$5 user=$6 pass=$7
  local dir="$ROOT/$name"; mkdir -p "$dir/src/test/java/bench"
  echo "include '$name'" >> "$ROOT/settings.gradle"; : > "$dir/build.gradle"
  cat > "$dir/src/test/java/bench/T.java" <<JAVA
package bench;
import org.junit.jupiter.api.Test;
import org.testcontainers.containers.GenericContainer;
import org.testcontainers.containers.wait.strategy.Wait;
import java.sql.*;
public class T {
  @Test void run() throws Exception {
    try (var c = new GenericContainer<>("$image")
        .withExposedPorts($port)$extra
        .waitingFor(Wait.forListeningPort())) {
      long s = System.nanoTime();
      c.start();
      System.out.println("CONTAINER_START_MS $name " + (System.nanoTime()-s)/1_000_000);
      String url = "jdbc:postgresql://" + c.getHost() + ":" + c.getMappedPort($port) + "/$db?sslmode=disable";
      long w = System.nanoTime();
      Connection cn = null;
      for (int a=0; a<40 && cn==null; a++) {
        try { cn = DriverManager.getConnection(url, "$user", "$pass"); }
        catch (SQLException e) { Thread.sleep(500); }
      }
      if (cn == null) throw new IllegalStateException("no db connection");
      try (Connection conn = cn) {
        try (Statement st = conn.createStatement()) { st.execute("CREATE TABLE bench (id INT PRIMARY KEY, v TEXT)"); }
        conn.setAutoCommit(false);
        try (PreparedStatement ps = conn.prepareStatement("INSERT INTO bench(id,v) VALUES(?,?)")) {
          for (int i=0;i<$ROWS;i++){ ps.setInt(1,i); ps.setString(2,"row-"+i+"-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"); ps.addBatch(); if(i%500==499){ps.executeBatch();} }
          ps.executeBatch();
        }
        conn.commit();
        for (int q=0;q<$QUERIES;q++){ try (Statement st=conn.createStatement(); ResultSet rs=st.executeQuery("SELECT count(*), max(id) FROM bench")){ rs.next(); } }
      }
      System.out.println("DB_WORK_MS $name " + (System.nanoTime()-w)/1_000_000);
    }
  }
}
JAVA
}

# plain module: just start container (valkey/redis)
gen_plain() {
  local name=$1 image=$2 port=$3
  local dir="$ROOT/$name"; mkdir -p "$dir/src/test/java/bench"
  echo "include '$name'" >> "$ROOT/settings.gradle"; : > "$dir/build.gradle"
  cat > "$dir/src/test/java/bench/T.java" <<JAVA
package bench;
import org.junit.jupiter.api.Test;
import org.testcontainers.containers.GenericContainer;
import org.testcontainers.containers.wait.strategy.Wait;
public class T {
  @Test void run() {
    try (var c = new GenericContainer<>("$image").withExposedPorts($port).waitingFor(Wait.forListeningPort())) {
      long s = System.nanoTime();
      c.start();
      System.out.println("CONTAINER_START_MS $name " + (System.nanoTime()-s)/1_000_000);
    }
  }
}
JAVA
}

for n in $(seq 1 "$NCOCKROACH"); do gen_sql "cockroach$n" "cockroachdb/cockroach:v24.1.6" 26257 '.withCommand("start-single-node --insecure")' defaultdb root ''; done
for n in $(seq 1 "$NPOSTGRES"); do gen_sql "postgres$n" "postgres:16" 5432 '.withEnv("POSTGRES_PASSWORD","test")' postgres postgres test; done
for n in $(seq 1 "$NVALKEY"); do gen_plain "valkey$n" "valkey/valkey:8" 6379; done
echo "generated $ROOT: $NCOCKROACH cockroach + $NPOSTGRES postgres (sql, $ROWS rows/$QUERIES queries) + $NVALKEY valkey"
