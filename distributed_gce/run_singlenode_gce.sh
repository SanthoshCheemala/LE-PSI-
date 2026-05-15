#!/usr/bin/env bash
# ============================================================
#  run_singlenode_gce.sh — Single-node benchmark on dedicated GCE VM
#  Usage: bash distributed_gce/run_singlenode_gce.sh
# ============================================================
set -euo pipefail

PROJECT="${PROJECT:-lepsi-distributed-493617}"
ZONE="${ZONE:-us-east1-b}"
MACHINE="${MACHINE:-e2-highmem-8}"
VM_NAME="lepsi-singlenode"
WORKDIR="/tmp/lepsi"
PSI_SRC="/Users/santhoshcheemala/ALL_IN_ONE/Research_Implimentation/PSI"
RESULTS_DIR="$PSI_SRC/scalability_results"

echo "══════════════════════════════════════════════════"
echo "  Single-Node GCE Benchmark"
echo "  Project : $PROJECT"
echo "  Zone    : $ZONE"
echo "  Machine : $MACHINE (8 vCPUs, ~64 GB RAM)"
echo "══════════════════════════════════════════════════"

# ── Step 1: Create VM ────────────────────────────────────
echo ""
echo "[1/5] Creating VM..."
if gcloud compute instances describe "$VM_NAME" \
    --project="$PROJECT" --zone="$ZONE" &>/dev/null; then
  echo "  VM already exists, starting..."
  gcloud compute instances start "$VM_NAME" \
    --project="$PROJECT" --zone="$ZONE" 2>/dev/null || true
else
  gcloud compute instances create "$VM_NAME" \
    --project="$PROJECT" --zone="$ZONE" \
    --machine-type="$MACHINE" \
    --image-family=debian-12 --image-project=debian-cloud \
    --boot-disk-size=30GB \
    --scopes=default
fi
echo "  ✓ VM ready"

# Wait for SSH
echo "  Waiting for SSH..."
for i in $(seq 1 30); do
  if gcloud compute ssh "$VM_NAME" --project="$PROJECT" --zone="$ZONE" \
    --ssh-flag='-o ConnectTimeout=5' --command="echo ok" &>/dev/null; then
    break
  fi
  sleep 5
done
echo "  ✓ SSH ready"

# ── Helper ───────────────────────────────────────────────
ssh_cmd() {
  gcloud compute ssh "$VM_NAME" \
    --project="$PROJECT" --zone="$ZONE" \
    --ssh-flag='-T' \
    --ssh-flag='-o BatchMode=yes' \
    --ssh-flag='-o ConnectTimeout=30' \
    --command="$1" < /dev/null
}

# ── Step 2: Install Go ──────────────────────────────────
echo ""
echo "[2/5] Installing Go..."
ssh_cmd 'set -e
  if /usr/local/go/bin/go version 2>/dev/null | grep -q "go1.24"; then
    echo "Go 1.24 already installed"
    exit 0
  fi
  for i in $(seq 1 12); do
    if sudo apt-get update -y -q 2>/dev/null; then break; fi
    sleep 5
  done
  sudo apt-get install -y -q gcc git wget 2>/dev/null || true
  wget -q https://go.dev/dl/go1.24.1.linux-amd64.tar.gz -O /tmp/go.tar.gz
  sudo rm -rf /usr/local/go
  sudo tar -C /usr/local -xzf /tmp/go.tar.gz
  rm /tmp/go.tar.gz
  /usr/local/go/bin/go version'
echo "  ✓ Go installed"

# ── Step 3: Upload source (minimal) ─────────────────────
echo ""
echo "[3/5] Uploading source code..."
ssh_cmd "mkdir -p $WORKDIR/scalability_tests"
for pkg in pkg internal utils go.mod go.sum; do
  gcloud compute scp --recurse "$PSI_SRC/$pkg" "$VM_NAME:$WORKDIR/" \
    --project="$PROJECT" --zone="$ZONE" 2>/dev/null || true
done
gcloud compute scp "$PSI_SRC/scalability_tests/bench_10k_gce.go" \
  "$VM_NAME:$WORKDIR/scalability_tests/" \
  --project="$PROJECT" --zone="$ZONE" 2>/dev/null
echo "  ✓ Source uploaded"

# ── Step 4: Build + Run ─────────────────────────────────
echo ""
echo "[4/5] Building and running benchmark..."
ssh_cmd "cd $WORKDIR && PATH=/usr/local/go/bin:\$PATH \
  go build -o bin/bench_10k_gce ./scalability_tests/bench_10k_gce.go 2>&1"
echo "  ✓ Built"

echo ""
echo "  Running benchmark (this will take ~20-40 min)..."
BENCH_START=$(date +%s)

ssh_cmd "cd $WORKDIR && \
  MACHINE_TYPE=$MACHINE \
  nohup ./bin/bench_10k_gce > /tmp/bench.log 2>&1 & \
  echo \$! > /tmp/bench.pid"

# Poll for completion
while true; do
  if ssh_cmd "grep -q -E 'Saved to:|failed' /tmp/bench.log 2>/dev/null"; then
    break
  fi
  # Show progress if available
  LAST=$(ssh_cmd "tail -1 /tmp/bench.log 2>/dev/null" 2>/dev/null || echo "...")
  echo "  ... $LAST"
  sleep 30
done

BENCH_END=$(date +%s)
WALL=$((BENCH_END - BENCH_START))
echo ""
echo "  Wall time: ${WALL}s ($(echo "scale=1; $WALL/60" | bc) min)"

# ── Step 5: Collect results ──────────────────────────────
echo ""
echo "[5/5] Collecting results..."
mkdir -p "$RESULTS_DIR"
gcloud compute scp "$VM_NAME:/tmp/bench.log" "$RESULTS_DIR/singlenode_gce.log" \
  --project="$PROJECT" --zone="$ZONE" 2>/dev/null || true
gcloud compute scp "$VM_NAME:$WORKDIR/scalability_results/*.json" "$RESULTS_DIR/" \
  --project="$PROJECT" --zone="$ZONE" 2>/dev/null || true

echo ""
echo "══════════════════════════════════════════════════"
echo "  RESULTS:"
cat "$RESULTS_DIR/singlenode_gce.log" | tail -15
echo ""
echo "  Files saved to: $RESULTS_DIR/"
echo "══════════════════════════════════════════════════"

# ── Cleanup ──────────────────────────────────────────────
echo ""
read -p "Delete VM $VM_NAME? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
  gcloud compute instances delete "$VM_NAME" \
    --project="$PROJECT" --zone="$ZONE" --quiet
  echo "  ✓ VM deleted"
fi
