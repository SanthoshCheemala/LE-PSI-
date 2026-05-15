#!/usr/bin/env bash
# Run only the 10K Microsoft APSI baseline on the same GCE VM.
set -euo pipefail

RESULTS="${RESULTS:-/tmp/psi_comparative_10k/apsi}"
M="${M:-10000}"
N="${N:-100}"
OVERLAP="${OVERLAP:-10}"
ITEM_BYTES="${ITEM_BYTES:-8}"
PORT="${PORT:-1213}"
THREADS="${THREADS:-$(nproc)}"
APSI_DIR="${APSI_DIR:-/tmp/apsi_build}"
VCPKG_DIR="${VCPKG_DIR:-/tmp/vcpkg}"

mkdir -p "$RESULTS"

echo "==== Microsoft APSI 10K benchmark ===="
echo "results=$RESULTS m=$M n=$N overlap=$OVERLAP threads=$THREADS"
echo "machine=$(hostname) cpus=$(nproc) ram_gb=$(free -g | awk '/Mem/{print $2}') date=$(date -u +%Y-%m-%dT%H:%M:%SZ)"

sudo apt-get update -qq
sudo apt-get install -y -qq cmake g++ git curl zip unzip tar pkg-config \
  python3 libssl-dev ninja-build autoconf automake libtool net-tools

if [[ ! -d "$VCPKG_DIR" ]]; then
  git clone https://github.com/microsoft/vcpkg.git "$VCPKG_DIR"
  "$VCPKG_DIR/bootstrap-vcpkg.sh" -disableMetrics
fi
"$VCPKG_DIR/vcpkg" install seal[no-throw-tran] kuku log4cplus cppzmq flatbuffers jsoncpp tclap

if [[ ! -d "$APSI_DIR" ]]; then
  git clone --recursive https://github.com/microsoft/APSI.git "$APSI_DIR"
fi

cd "$APSI_DIR"
git checkout -- CMakeLists.txt
sed -i 's/find_package(SEAL 4\.1/find_package(SEAL 4.3/' CMakeLists.txt
sed -i 's/find_package(Kuku 2\.1/find_package(Kuku 3.0/' CMakeLists.txt

mkdir -p build
cd build
cmake .. \
  -DCMAKE_BUILD_TYPE=Release \
  -DAPSI_BUILD_CLI=ON \
  -DAPSI_BUILD_TESTS=OFF \
  -DCMAKE_TOOLCHAIN_FILE="$VCPKG_DIR/scripts/buildsystems/vcpkg.cmake"
make -j"$THREADS"

SENDER="$APSI_DIR/build/bin/sender_cli"
RECEIVER="$APSI_DIR/build/bin/receiver_cli"
PARAMS="$APSI_DIR/parameters/1M-256.json"

cd /tmp
python3 "$APSI_DIR/tools/scripts/test_data_creator.py" "$M" "$N" "$OVERLAP" 0 "$ITEM_BYTES"

pkill -9 -f sender_cli 2>/dev/null || true
sleep 2

/usr/bin/time -v -o "$RESULTS/sender_time_m${M}.txt" \
  "$SENDER" -d /tmp/db.csv -p "$PARAMS" -t "$THREADS" --port "$PORT" -l info \
  -f "$RESULTS/sender_m${M}.log" &
SENDER_PID=$!

for _ in $(seq 1 300); do
  if ss -tlnp 2>/dev/null | grep -q ":$PORT"; then
    break
  fi
  if ! kill -0 "$SENDER_PID" 2>/dev/null; then
    echo "sender exited before listening" >&2
    break
  fi
  sleep 1
done
sleep 2

START_NS=$(date +%s%N)
/usr/bin/time -v -o "$RESULTS/receiver_time_m${M}.txt" \
  "$RECEIVER" -q /tmp/query.csv -o "$RESULTS/matches_m${M}.txt" \
  -a 127.0.0.1 --port "$PORT" -t "$THREADS" -l info \
  -f "$RESULTS/receiver_m${M}.log"
END_NS=$(date +%s%N)

kill -TERM "$SENDER_PID" 2>/dev/null || true
wait "$SENDER_PID" 2>/dev/null || true

ONLINE_MS=$(( (END_NS - START_NS) / 1000000 ))
MATCHES=$(wc -l < "$RESULTS/matches_m${M}.txt" 2>/dev/null || echo 0)
SENDER_RSS_KB=$(awk 'tolower($0) ~ /maximum resident/ {print $NF}' "$RESULTS/sender_time_m${M}.txt" 2>/dev/null || echo 0)
RECEIVER_RSS_KB=$(awk 'tolower($0) ~ /maximum resident/ {print $NF}' "$RESULTS/receiver_time_m${M}.txt" 2>/dev/null || echo 0)
COMM_R2S_KB=$(awk '/Communication R->S:/ {print $(NF-1)}' "$RESULTS/receiver_m${M}.log" | tail -1)
COMM_S2R_KB=$(awk '/Communication S->R:/ {print $(NF-1)}' "$RESULTS/receiver_m${M}.log" | tail -1)
COMM_TOTAL_KB=$(awk '/Communication total:/ {print $(NF-1)}' "$RESULTS/receiver_m${M}.log" | tail -1)

cat > "$RESULTS/apsi_m${M}_n${N}.json" <<EOF
{
  "protocol": "Microsoft APSI (BFV/SEAL)",
  "source_url": "https://github.com/microsoft/APSI",
  "source_commit": "$(cd "$APSI_DIR" && git rev-parse --short HEAD)",
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "machine": "$(hostname)",
  "machine_type": "$(curl -fs -H 'Metadata-Flavor: Google' http://metadata.google.internal/computeMetadata/v1/instance/machine-type 2>/dev/null | awk -F/ '{print $NF}')",
  "vcpus": $(nproc),
  "ram_gb": $(free -g | awk '/Mem/{print $2}'),
  "server_dataset_size": $M,
  "client_dataset_size": $N,
  "expected_intersection": $OVERLAP,
  "matches_found": $MATCHES,
  "online_time_ms": $ONLINE_MS,
  "sender_peak_rss_kb": ${SENDER_RSS_KB:-0},
  "receiver_peak_rss_kb": ${RECEIVER_RSS_KB:-0},
  "communication_r_to_s_kb": ${COMM_R2S_KB:-0},
  "communication_s_to_r_kb": ${COMM_S2R_KB:-0},
  "communication_total_kb": ${COMM_TOTAL_KB:-0},
  "threads": $THREADS,
  "params_file": "1M-256.json"
}
EOF

cat "$RESULTS/apsi_m${M}_n${N}.json"
