#!/usr/bin/env bash
# Run the RELIC demo for Aranha-Lin-Orlandi-Simkin CCS'22 laconic PSI.
set -euo pipefail

RESULTS="${RESULTS:-/tmp/psi_comparative_10k/alos22_relic}"
RELIC_DIR="${RELIC_DIR:-/tmp/relic}"
M="${M:-10000}"
N="${N:-100}"
THREADS="${THREADS:-$(nproc)}"

mkdir -p "$RESULTS"

echo "==== ALOS22/RELIC laconic PSI 10K benchmark ===="
echo "results=$RESULTS m=$M n=$N threads=$THREADS"
echo "machine=$(hostname) cpus=$(nproc) ram_gb=$(free -g | awk '/Mem/{print $2}') date=$(date -u +%Y-%m-%dT%H:%M:%SZ)"

sudo apt-get update -qq
sudo apt-get install -y -qq git cmake make gcc libgmp-dev

if [[ ! -d "$RELIC_DIR/.git" ]]; then
  git clone --depth 1 https://github.com/relic-toolkit/relic.git "$RELIC_DIR"
fi

cd "$RELIC_DIR/demo/psi-client-server"
cat > params.h <<EOF
#define M $M
#define N $N

#define SK "258B8F5E39671B337C1E3B87559B579D3F878E5293DF2B01DE1B9E10CA9EC9D0"
EOF

make clean >/dev/null 2>&1 || true
make -j"$THREADS" 2>&1 | tee "$RESULTS/build.log"

START_NS=$(date +%s%N)
/usr/bin/time -v -o "$RESULTS/test_bench_time.txt" ./test-bench > "$RESULTS/test_bench.log" 2>&1
END_NS=$(date +%s%N)

TOTAL_MS=$(( (END_NS - START_NS) / 1000000 ))
PEAK_RSS_KB=$(awk 'tolower($0) ~ /maximum resident/ {print $NF}' "$RESULTS/test_bench_time.txt" 2>/dev/null || echo 0)

cat > "$RESULTS/alos22_relic_m${M}_n${N}.json" <<EOF
{
  "protocol": "ALOS22 laconic PSI from pairings (RELIC demo)",
  "source_url": "https://github.com/relic-toolkit/relic/tree/main/demo/psi-client-server",
  "source_commit": "$(git rev-parse --short HEAD)",
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "machine": "$(hostname)",
  "machine_type": "$(curl -fs -H 'Metadata-Flavor: Google' http://metadata.google.internal/computeMetadata/v1/instance/machine-type 2>/dev/null | awk -F/ '{print $NF}')",
  "vcpus": $(nproc),
  "ram_gb": $(free -g | awk '/Mem/{print $2}'),
  "server_dataset_size": $M,
  "client_dataset_size": $N,
  "total_time_ms": $TOTAL_MS,
  "peak_rss_kb": ${PEAK_RSS_KB:-0},
  "notes": "RELIC test-bench reports primitive timings for cp_pbpsi_gen/ask/ans/int with params.h set to M and N; it is not a CSV/file PSI wrapper."
}
EOF

cat "$RESULTS/alos22_relic_m${M}_n${N}.json"
