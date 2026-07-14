#!/usr/bin/env bash
# Generates a Kotlin-compile-heavy Gradle build to mirror the CPU/daemon-bound
# compile phase of GoodNotes' `gradlew check` (which our container/DB benchmark
# did not exercise). Many Kotlin files across parallel modules; matches their
# Kotlin daemon heap (-Xmx4096M).
set -euo pipefail
ROOT=${ROOT:-kproj}
MODULES=${MODULES:-8}
FILES_PER_MODULE=${FILES_PER_MODULE:-300}
KOTLIN=2.0.21
rm -rf "$ROOT"; mkdir -p "$ROOT"

cat > "$ROOT/gradle.properties" <<'EOF'
org.gradle.parallel=true
org.gradle.caching=false
org.gradle.jvmargs=-Xmx4096m -Dkotlin.daemon.jvm.options="-Xmx4096M" -XX:MaxMetaspaceSize=512m
EOF

cat > "$ROOT/build.gradle" <<EOF
buildscript {
  repositories { mavenCentral() }
  dependencies { classpath 'org.jetbrains.kotlin:kotlin-gradle-plugin:$KOTLIN' }
}
subprojects {
  apply plugin: 'org.jetbrains.kotlin.jvm'
  repositories { mavenCentral() }
}
EOF

echo "rootProject.name = 'kbench'" > "$ROOT/settings.gradle"

mkmod() {
  local m=$1 dir="$ROOT/mod$m"
  mkdir -p "$dir/src/main/kotlin/bench/m$m"
  echo "include 'mod$m'" >> "$ROOT/settings.gradle"
  : > "$dir/build.gradle"
  for f in $(seq 1 "$FILES_PER_MODULE"); do
    cat > "$dir/src/main/kotlin/bench/m$m/F${f}.kt" <<KT
package bench.m$m
data class D${f}(val id: Int, val name: String, val vals: List<Int>, val tags: Map<String, Int>)
class C${f} {
  fun hash(x: Int): Int { var s = x; for (i in 0..20) { s = s * 31 + i xor (s ushr 3) }; return s }
  fun reduce(l: List<Int>): Int = l.asSequence().map { it * $f }.filter { it % 2 == 0 }.fold(0) { a, v -> a + v }
  fun group(items: List<D${f}>): Map<String, Int> = items.groupBy { it.name }.mapValues { e -> e.value.sumOf { it.id } }
  fun <T : Comparable<T>> maxOfOr(list: List<T>, dflt: T): T = list.maxOrNull() ?: dflt
  fun make(n: Int): List<D${f}> = (0 until n).map { D${f}(it, "n\$it", listOf(1, 2, it, $f), mapOf("a" to it, "b" to $f)) }
}
KT
  done
}
for m in $(seq 1 "$MODULES"); do mkmod "$m"; done
echo "generated $ROOT: $MODULES modules x $FILES_PER_MODULE kotlin files = $((MODULES*FILES_PER_MODULE)) files"
