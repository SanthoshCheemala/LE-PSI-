#!/bin/bash
# ============================================================
#  APSI Comparative Baseline — Single-Node Build & Benchmark
#  Run on the SAME single GCE VM as LE-PSI for fair comparison.
#
#  Microsoft APSI: BFV-based asymmetric PSI (Microsoft SEAL).
#  Lattice-family HE baseline — fairest PQ comparison.
#
#  Usage:
#    1. Create a GCE VM (e2-highmem-4 or similar):
#       gcloud compute instances create apsi-bench \
#         --machine-type=e2-highmem-4 --zone=us-east1-b \
#         --image-family=debian-12 --image-project=debian-cloud \
#         --boot-disk-size=50GB
#
#    2. Upload this script:
#       gcloud compute scp comparative_baselines/apsi/setup_and_benchmark.sh \
#         apsi-bench:/tmp/ --zone=us-east1-b
#
#    3. SSH in and run:
#       gcloud compute ssh apsi-bench --zone=us-east1-b
#       chmod +x /tmp/setup_and_benchmark.sh
#       nohup bash /tmp/setup_and_benchmark.sh > /tmp/apsi_full.log 2>&1 &
#
#    4. Collect results:
#       gcloud compute scp apsi-bench:/tmp/apsi_results/* ./comparative_baselines/apsi/results/
# ============================================================
set -euo pipefail

APSI_DIR="/tmp/apsi_build"
RESULTS_DIR="/tmp/apsi_results"
VCPKG_DIR="/tmp/vcpkg"
DATA_DIR="/tmp/apsi_data"

mkdir -p "$RESULTS_DIR" "$DATA_DIR"

echo "════════════════════════════════════════════════════════"
echo "  APSI COMPARATIVE BASELINE BENCHMARK"
echo "  Machine: $(hostname) | CPUs: $(nproc) | RAM: $(free -g | awk '/Mem/{print $2}')G"
echo "  Date: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo "════════════════════════════════════════════════════════"

# ── Step 1: Install build dependencies ────────────────────
echo ""
echo "[Step 1/5] Installing build dependencies..."
sudo apt-get update -qq
sudo apt-get install -y -qq \
  cmake g++ git curl zip unzip tar pkg-config python3 \
  libssl-dev ninja-build autoconf automake libtool net-tools

# ── Step 2: Install vcpkg + APSI dependencies ────────────
echo ""
echo "[Step 2/5] Setting up vcpkg and dependencies..."
echo "  (This takes 15-30 min on first run)"
if [ ! -d "$VCPKG_DIR" ]; then
  git clone https://github.com/microsoft/vcpkg.git "$VCPKG_DIR"
  "$VCPKG_DIR/bootstrap-vcpkg.sh" -disableMetrics
fi

"$VCPKG_DIR/vcpkg" install seal[no-throw-tran] kuku log4cplus cppzmq flatbuffers jsoncpp tclap

# ── Step 3: Clone and build APSI ──────────────────────────
echo ""
echo "[Step 3/5] Building APSI..."
if [ ! -d "$APSI_DIR" ]; then
  git clone --recursive https://github.com/microsoft/APSI.git "$APSI_DIR"
fi

cd "$APSI_DIR"
mkdir -p build && cd build
cmake .. \
  -DCMAKE_BUILD_TYPE=Release \
  -DAPSI_BUILD_CLI=ON \
  -DAPSI_BUILD_TESTS=OFF \
  -DCMAKE_TOOLCHAIN_FILE="$VCPKG_DIR/scripts/buildsystems/vcpkg.cmake"
make -j$(nproc)

SENDER_BIN="$APSI_DIR/build/bin/sender_cli"
RECEIVER_BIN="$APSI_DIR/build/bin/receiver_cli"

if [ ! -f "$SENDER_BIN" ] || [ ! -f "$RECEIVER_BIN" ]; then
  echo "ERROR: APSI CLI binaries not found after build"
  ls -la "$APSI_DIR/build/bin/" 2>/dev/null || echo "bin/ does not exist"
  exit 1
fi
echo "✓ APSI built: $SENDER_BIN"

# ── Step 4: Generate test data ────────────────────────────
echo ""
echo "[Step 4/5] Generating test data..."

SIZES=(1000 2000 4000 8000 10000)
RECEIVER_SIZE=100
INTERSECTION_SIZE=10   # ~10% intersection rate
ITEM_BYTES=8           # 64-bit items (matches our uint64 hashes)

cd "$DATA_DIR"
for M in "${SIZES[@]}"; do
  echo "  m=$M n=$RECEIVER_SIZE intersection=$INTERSECTION_SIZE..."
  python3 "$APSI_DIR/tools/scripts/test_data_creator.py" \
    "$M" "$RECEIVER_SIZE" "$INTERSECTION_SIZE" 0 "$ITEM_BYTES"
  mv db.csv "db_m${M}.csv"
  mv query.csv "query_m${M}.csv"
done
echo "✓ Test data generated"

# ── Step 5: Run benchmarks ────────────────────────────────
echo ""
echo "[Step 5/5] Running APSI benchmarks..."

# Find best matching parameter file
# APSI ships params for various sender/receiver sizes
PARAMS_FILE=""
for f in "$APSI_DIR/parameters/"*; do
  PARAMS_FILE="$f"
done
# Prefer the 1M-1024 unlabeled params if it exists
for f in "$APSI_DIR/parameters/"*1M*1024*; do
  [ -f "$f" ] && PARAMS_FILE="$f" && break
done
echo "  Using params: $(basename "$PARAMS_FILE")"

THREADS=$(nproc)

for M in "${SIZES[@]}"; do
  echo ""
  echo "╔══════════════════════════════════════════════════╗"
  echo "║  APSI: m=$M, n=$RECEIVER_SIZE, threads=$THREADS"
  echo "╚══════════════════════════════════════════════════╝"

  DB_FILE="$DATA_DIR/db_m${M}.csv"
  QUERY_FILE="$DATA_DIR/query_m${M}.csv"
  RESULT_JSON="$RESULTS_DIR/apsi_m${M}_n${RECEIVER_SIZE}.json"

  # Kill any leftover sender
  pkill -f sender_cli 2>/dev/null || true
  sleep 1

  # Start sender (server) in background
  echo "  Starting sender..."
  /usr/bin/time -v "$SENDER_BIN" \
    -d "$DB_FILE" \
    -p "$PARAMS_FILE" \
    -t "$THREADS" \
    --port 1212 \
    -l info \
    -f "$RESULTS_DIR/sender_m${M}.log" \
    2>"$RESULTS_DIR/sender_time_m${M}.txt" &
  SENDER_PID=$!

  # Wait for sender to bind port
  echo "  Waiting for sender to listen..."
  for i in $(seq 1 600); do
    if ss -tlnp 2>/dev/null | grep -q ':1212' || \
       netstat -tlnp 2>/dev/null | grep -q ':1212'; then
      break
    fi
    # Check sender didn't crash
    if ! kill -0 "$SENDER_PID" 2>/dev/null; then
      echo "  ERROR: Sender process died. Check $RESULTS_DIR/sender_m${M}.log"
      cat "$RESULTS_DIR/sender_time_m${M}.txt" 2>/dev/null
      break
    fi
    sleep 1
  done
  sleep 2

  # Run receiver (client) and time it
  echo "  Running receiver query..."
  RECV_START_NS=$(date +%s%N)

  /usr/bin/time -v "$RECEIVER_BIN" \
    -q "$QUERY_FILE" \
    -o "$RESULTS_DIR/matches_m${M}.txt" \
    -a "127.0.0.1" \
    --port 1212 \
    -t "$THREADS" \
    -l info \
    -f "$RESULTS_DIR/receiver_m${M}.log" \
    2>"$RESULTS_DIR/receiver_time_m${M}.txt"

  RECV_END_NS=$(date +%s%N)
  RECV_MS=$(( (RECV_END_NS - RECV_START_NS) / 1000000 ))

  # Stop sender
  kill "$SENDER_PID" 2>/dev/null || true
  wait "$SENDER_PID" 2>/dev/null || true
  sleep 1

  # Extract metrics
  SENDER_RSS=$(grep "Maximum resident" "$RESULTS_DIR/sender_time_m${M}.txt" 2>/dev/null | awk '{print $NF}' || echo "0")
  RECEIVER_RSS=$(grep "Maximum resident" "$RESULTS_DIR/receiver_time_m${M}.txt" 2>/dev/null | awk '{print $NF}' || echo "0")
  SENDER_WALL=$(grep "wall clock" "$RESULTS_DIR/sender_time_m${M}.txt" 2>/dev/null | awk '{print $NF}' || echo "?")
  RECEIVER_WALL=$(grep "wall clock" "$RESULTS_DIR/receiver_time_m${M}.txt" 2>/dev/null | awk '{print $NF}' || echo "?")
  MATCHES=$(wc -l < "$RESULTS_DIR/matches_m${M}.txt" 2>/dev/null || echo "0")

  # Write structured JSON result
  cat > "$RESULT_JSON" <<ENDJSON
{
  "protocol": "Microsoft APSI (BFV/SEAL)",
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "machine": "$(hostname)",
  "cpus": $(nproc),
  "ram_gb": $(free -g | awk '/Mem/{print $2}'),
  "server_dataset_size": $M,
  "client_dataset_size": $RECEIVER_SIZE,
  "expected_intersection": $INTERSECTION_SIZE,
  "matches_found": $MATCHES,
  "receiver_online_time_ms": $RECV_MS,
  "sender_wall_time": "$SENDER_WALL",
  "receiver_wall_time": "$RECEIVER_WALL",
  "sender_peak_rss_kb": $SENDER_RSS,
  "receiver_peak_rss_kb": $RECEIVER_RSS,
  "threads": $THREADS,
  "params_file": "$(basename "$PARAMS_FILE")",
  "item_bytes": $ITEM_BYTES
}
ENDJSON

  echo "  ✓ m=$M done | online=${RECV_MS}ms | matches=$MATCHES"
  echo "    Sender RSS: $((SENDER_RSS/1024))MB | Receiver RSS: $((RECEIVER_RSS/1024))MB"
done

echo ""
echo "════════════════════════════════════════════════════════"
echo "  ALL APSI BENCHMARKS COMPLETE"
echo "  Results: $RESULTS_DIR/"
echo "════════════════════════════════════════════════════════"
ls -la "$RESULTS_DIR/"*.json
