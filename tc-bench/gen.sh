#!/usr/bin/env bash
# Generates a Gradle multi-project testcontainers benchmark that mirrors
# GoodNotes' run-test concurrency: parallel workers, 1 fork/task, -Xmx4g,
# many DB-backed test modules each standing up a testcontainer (no reuse).
set -euo pipefail
ROOT=${ROOT:-genproj}
NCOCKROACH=${NCOCKROACH:-19}
NPOSTGRES=${NPOSTGRES:-4}
NVALKEY=${NVALKEY:-2}
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
    testRuntimeOnly 'org.junit.platform:junit-platform-launcher'
    testRuntimeOnly 'org.slf4j:slf4j-simple:2.0.13'
  }
  test {
    useJUnitPlatform()
    testLogging { showStandardStreams = true; exceptionFormat 'full' }
    environment 'DOCKER_API_VERSION', '1.40'
  }
}
EOF

gen() {
  local name=$1 image=$2 port=$3 extra=$4
  local dir="$ROOT/$name"
  mkdir -p "$dir/src/test/java/bench"
  echo "include '$name'" >> "$ROOT/settings.gradle"
  : > "$dir/build.gradle"
  cat > "$dir/src/test/java/bench/T.java" <<JAVA
package bench;
import org.junit.jupiter.api.Test;
import org.testcontainers.containers.GenericContainer;
import org.testcontainers.containers.wait.strategy.Wait;
public class T {
  @Test void start() {
    try (var c = new GenericContainer<>("$image")
        .withExposedPorts($port)$extra
        .waitingFor(Wait.forListeningPort())) {
      long s = System.nanoTime();
      c.start();
      long ms = (System.nanoTime() - s) / 1_000_000;
      System.out.println("CONTAINER_START_MS $name " + ms);
    }
  }
}
JAVA
}

for n in $(seq 1 "$NCOCKROACH"); do gen "cockroach$n" "cockroachdb/cockroach:v24.1.6" 26257 '.withCommand("start-single-node --insecure")'; done
for n in $(seq 1 "$NPOSTGRES"); do gen "postgres$n" "postgres:16" 5432 '.withEnv("POSTGRES_PASSWORD","test")'; done
for n in $(seq 1 "$NVALKEY"); do gen "valkey$n" "valkey/valkey:8" 6379 ''; done
echo "generated $ROOT: $NCOCKROACH cockroach, $NPOSTGRES postgres, $NVALKEY valkey"
