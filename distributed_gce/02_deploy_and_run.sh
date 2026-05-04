#!/usr/bin/env bash
# ============================================================
#  02_deploy_and_run.sh — Deploy + Run Distributed LE-PSI
#  Mirrors all-in-one-run.sh from the leader election project
# ============================================================
set -euo pipefail

PROJECT="${PROJECT:-distributed-sim}"
ZONE="${ZONE:-us-central1-a}"
K="${K:-7}"
M="${M:-10000}"
N="${N:-100}"
WORKDIR='/tmp/lepsi'
PSI_SRC="/Users/santhoshcheemala/ALL_IN_ONE/Research_Implimentation/PSI"
RESULTS_DIR="$(cd "$(dirname "$0")" && pwd)/results"
RUN_ID="$(date +%Y%m%d_%H%M%S)_m${M}_n${N}_K${K}"
mkdir -p "$RESULTS_DIR/$RUN_ID"

PARALLEL_JOBS=4

echo "============================================================"
echo "  LE-PSI DISTRIBUTED BENCHMARK"
echo "  Date    : $(date)"
echo "  Project : $PROJECT / $ZONE"
echo "  Config  : m=$M, n=$N, K=$K shards"
echo "  Results : $RESULTS_DIR/$RUN_ID"
echo "============================================================"

# ── Get VM list from GCP ─────────────────────────────────
COORD_ROW="$(gcloud compute instances list \
  --project="$PROJECT" \
  --filter="labels.experiment=lepsi-dist AND labels.role=coordinator" \
  --format='csv[no-heading](name,zone,networkInterfaces[0].networkIP)' | head -1)"

SHARD_ROWS="$(gcloud compute instances list \
  --project="$PROJECT" \
  --filter="labels.experiment=lepsi-dist AND labels.role=shard" \
  --sort-by="labels.shard_id" \
  --format='csv[no-heading](name,zone,networkInterfaces[0].networkIP)')"

COORD_NAME=$(echo "$COORD_ROW" | cut -d, -f1)
COORD_ZONE=$(echo "$COORD_ROW" | cut -d, -f2)
COORD_IP=$(echo "$COORD_ROW"   | cut -d, -f3)

echo "Coordinator: $COORD_NAME ($COORD_IP)"

# Build comma-separated shard URLs
SHARD_URLS=""
while IFS=, read -r name zone ip; do
  [[ -z "$ip" ]] && continue
  SHARD_URLS="${SHARD_URLS:+$SHARD_URLS,}http://${ip}:8081"
done <<< "$SHARD_ROWS"
echo "Shard URLs: $SHARD_URLS"

# ── SSH/SCP helpers (identical to leader project) ────────
ssh_cmd() {
  local name="$1" zone="$2" cmd="$3"
  gcloud compute ssh "$name" \
    --project="$PROJECT" --zone="$zone" \
    --ssh-flag='-T' \
    --ssh-flag='-o BatchMode=yes' \
    --ssh-flag='-o ConnectTimeout=30' \
    --command="$cmd" < /dev/null
}

scp_dir() {
  local src="$1" name="$2" zone="$3" dst="$4"
  gcloud compute scp --recurse "$src" "$name:$dst" \
    --project="$PROJECT" --zone="$zone"
}

wait_for_slot() {
  while [[ $(jobs -pr | wc -l | tr -d ' ') -ge "$PARALLEL_JOBS" ]]; do sleep 1; done
}

# ── Step 1: Install Go on all VMs ────────────────────────
echo ""
echo "[1/5] Installing Go on all VMs..."
install_go() {
  local name="$1" zone="$2"
  ssh_cmd "$name" "$zone" 'set -e
    if ! command -v go &>/dev/null || [[ $(go version | grep -oP "go1\.\d+" | head -1) < "go1.21" ]]; then
      echo "Installing Go 1.24..."
      sudo apt-get update -y -q
      sudo apt-get install -y -q gcc git sqlite3 libsqlite3-dev wget
      wget -q https://go.dev/dl/go1.24.1.linux-amd64.tar.gz -O /tmp/go.tar.gz
      sudo rm -rf /usr/local/go
      sudo tar -C /usr/local -xzf /tmp/go.tar.gz
      echo "export PATH=\$PATH:/usr/local/go/bin" >> ~/.bashrc
      rm /tmp/go.tar.gz
    fi
    /usr/local/go/bin/go version'
}

install_go "$COORD_NAME" "$COORD_ZONE" &
while IFS=, read -r name zone ip; do
  [[ -z "$name" ]] && continue
  wait_for_slot
  install_go "$name" "$zone" &
done <<< "$SHARD_ROWS"
wait
echo "  ✓ Go installed on all VMs"

# ── Step 2: Upload source code ───────────────────────────
echo ""
echo "[2/5] Uploading PSI source code..."
upload_source() {
  local name="$1" zone="$2"
  ssh_cmd "$name" "$zone" "mkdir -p $WORKDIR/bin"
  for pkg in pkg internal utils distributed_gce go.mod go.sum; do
    scp_dir "$PSI_SRC/$pkg" "$name" "$zone" "$WORKDIR/" 2>/dev/null || true
  done
}

upload_source "$COORD_NAME" "$COORD_ZONE" &
while IFS=, read -r name zone ip; do
  [[ -z "$name" ]] && continue
  wait_for_slot
  upload_source "$name" "$zone" &
done <<< "$SHARD_ROWS"
wait
echo "  ✓ Source uploaded"

# ── Step 3: Build on each VM ─────────────────────────────
echo ""
echo "[3/5] Building binaries..."
build_shard() {
  local name="$1" zone="$2"
  ssh_cmd "$name" "$zone" \
    "cd $WORKDIR && PATH=/usr/local/go/bin:\$PATH go build -o bin/lepsi_shard ./distributed_gce/shard/ 2>&1"
}
build_coord() {
  local name="$1" zone="$2"
  ssh_cmd "$name" "$zone" \
    "cd $WORKDIR && PATH=/usr/local/go/bin:\$PATH go build -o bin/lepsi_coord ./distributed_gce/coordinator/ 2>&1"
}

build_coord "$COORD_NAME" "$COORD_ZONE" &
while IFS=, read -r name zone ip; do
  [[ -z "$name" ]] && continue
  wait_for_slot
  build_shard "$name" "$zone" &
done <<< "$SHARD_ROWS"
wait
echo "  ✓ Binaries built"

# ── Step 4: Start shard servers (with retry) ─────────────
echo ""
echo "[4/5] Starting shard servers..."

start_shard() {
  local name="$1" zone="$2" sid="$3"
  ssh_cmd "$name" "$zone" \
    "cd $WORKDIR && pkill lepsi_shard 2>/dev/null || true; sleep 1; \
     SHARD_ID=$sid PORT=8081 nohup ./bin/lepsi_shard > /tmp/shard_${sid}.log 2>&1 & \
     sleep 3; curl -sf http://localhost:8081/health && echo ' shard $sid OK' || echo ' shard $sid FAILED'"
}

MAX_RETRIES=3
SHARD_ID=0
while IFS=, read -r name zone ip; do
  [[ -z "$name" ]] && continue
  for attempt in $(seq 1 $MAX_RETRIES); do
    echo "  [shard-$SHARD_ID] attempt $attempt on $name ($ip)..."
    if start_shard "$name" "$zone" "$SHARD_ID" 2>&1 | grep -q "OK"; then
      echo "  [shard-$SHARD_ID] ✓ started"
      break
    fi
    echo "  [shard-$SHARD_ID] ✗ failed, retrying in 15s..."
    sleep 15
    if [[ $attempt -eq $MAX_RETRIES ]]; then
      echo "  ERROR: shard-$SHARD_ID failed after $MAX_RETRIES attempts. Aborting."
      exit 1
    fi
  done
  SHARD_ID=$((SHARD_ID + 1))
done <<< "$SHARD_ROWS"

echo "  Waiting 10s for shard servers to stabilize..."
sleep 10

# ── Verify ALL shards healthy from coordinator ───────────
echo "  Verifying all shards from coordinator..."
SHARD_URLS_ARRAY=(${SHARD_URLS//,/ })
ALL_HEALTHY=true
for i in "${!SHARD_URLS_ARRAY[@]}"; do
  url="${SHARD_URLS_ARRAY[$i]}/health"
  if ssh_cmd "$COORD_NAME" "$COORD_ZONE" "curl -sf $url" 2>/dev/null | grep -q "shard_id"; then
    echo "  ✓ shard-$i reachable from coordinator"
  else
    echo "  ✗ shard-$i NOT reachable from coordinator at $url"
    ALL_HEALTHY=false
  fi
done

if [[ "$ALL_HEALTHY" != "true" ]]; then
  echo "  ERROR: Not all shards are healthy. Aborting benchmark."
  exit 1
fi
echo "  ✓ All shard servers verified and running"

# ── Step 5: Run coordinator benchmark ────────────────────
echo ""
echo "[5/5] Running coordinator benchmark (m=$M, n=$N, K=$K)..."
BENCH_START=$(date +%s)

ssh_cmd "$COORD_NAME" "$COORD_ZONE" \
  "cd $WORKDIR && \
   M=$M N=$N SHARD_URLS='$SHARD_URLS' RESULT_DIR=/tmp/lepsi_results \
   ./bin/lepsi_coord 2>&1 | tee /tmp/coord.log"

BENCH_END=$(date +%s)
WALL_SEC=$((BENCH_END - BENCH_START))

echo ""
echo "  Wall time: ${WALL_SEC}s ($(echo "scale=1; $WALL_SEC/60" | bc) min)"

# ── Step 6: Collect results ───────────────────────────────
echo ""
echo "[6/5] Collecting results..."
gcloud compute scp \
  "$COORD_NAME:/tmp/lepsi_results/*.json" \
  "$RESULTS_DIR/$RUN_ID/" \
  --project="$PROJECT" --zone="$COORD_ZONE" 2>/dev/null || true

gcloud compute scp \
  "$COORD_NAME:/tmp/coord.log" \
  "$RESULTS_DIR/$RUN_ID/coordinator.log" \
  --project="$PROJECT" --zone="$COORD_ZONE" 2>/dev/null || true

# Collect shard logs
SHARD_ID=0
while IFS=, read -r name zone ip; do
  [[ -z "$name" ]] && continue
  gcloud compute scp \
    "$name:/tmp/shard_${SHARD_ID}.log" \
    "$RESULTS_DIR/$RUN_ID/shard_${SHARD_ID}.log" \
    --project="$PROJECT" --zone="$zone" 2>/dev/null || true
  SHARD_ID=$((SHARD_ID + 1))
done <<< "$SHARD_ROWS"

echo ""
echo "============================================================"
echo "  COMPLETED"
echo "  Results: $RESULTS_DIR/$RUN_ID/"
ls -lh "$RESULTS_DIR/$RUN_ID/" 2>/dev/null || true
echo "============================================================"
