#!/usr/bin/env bash
# ============================================================
#  01_create_vms.sh — Provision GCE VMs for Distributed PSI
#  1 coordinator + K shard nodes
# ============================================================
set -euo pipefail

PROJECT="${PROJECT:-distributed-sim}"
ZONE="${ZONE:-us-central1-a}"
K="${K:-7}"                          # number of shard VMs
MACHINE="${MACHINE:-n2-highcpu-16}"  # 16 vCPU, ~16 GB RAM, ~$0.57/hr
IMAGE_FAMILY="debian-12"
IMAGE_PROJECT="debian-cloud"
LABEL="experiment=lepsi-dist"

echo "=================================================="
echo "  LE-PSI GCE Cluster Provisioning"
echo "  Project : $PROJECT"
echo "  Zone    : $ZONE"
echo "  Shards  : $K  (+ 1 coordinator)"
echo "  Machine : $MACHINE"
echo "=================================================="

# ── Coordinator ──────────────────────────────────────────
echo "[1/3] Creating coordinator VM..."
gcloud compute instances create "lepsi-coord" \
  --project="$PROJECT" \
  --zone="$ZONE" \
  --machine-type="$MACHINE" \
  --image-family="$IMAGE_FAMILY" \
  --image-project="$IMAGE_PROJECT" \
  --boot-disk-size=20GB \
  --boot-disk-type=pd-ssd \
  --labels="${LABEL},role=coordinator" \
  --tags="lepsi-psi" \
  --metadata="startup-script=apt-get update -y && apt-get install -y golang-go sqlite3"

# ── Shard VMs in parallel ────────────────────────────────
echo "[2/3] Creating $K shard VMs in parallel..."
for i in $(seq 0 $((K - 1))); do
  gcloud compute instances create "lepsi-shard-${i}" \
    --project="$PROJECT" \
    --zone="$ZONE" \
    --machine-type="$MACHINE" \
    --image-family="$IMAGE_FAMILY" \
    --image-project="$IMAGE_PROJECT" \
    --boot-disk-size=20GB \
    --boot-disk-type=pd-ssd \
    --labels="${LABEL},role=shard,shard_id=${i}" \
    --tags="lepsi-psi" \
    --metadata="startup-script=apt-get update -y && apt-get install -y golang-go sqlite3" &
done
wait
echo "  ✓ All shard VMs created"

# ── Firewall rule ────────────────────────────────────────
echo "[3/3] Creating firewall rule for inter-node HTTP..."
gcloud compute firewall-rules create lepsi-internal-http \
  --project="$PROJECT" \
  --network=default \
  --allow=tcp:8080,tcp:8081 \
  --source-tags=lepsi-psi \
  --target-tags=lepsi-psi \
  --description="LE-PSI shard <-> coordinator HTTP" 2>/dev/null || \
  echo "  (firewall rule already exists, skipping)"

echo ""
echo "✓ Cluster ready. VM list:"
gcloud compute instances list \
  --project="$PROJECT" \
  --filter="labels.experiment=lepsi-dist" \
  --format="table(name,zone,networkInterfaces[0].networkIP,status,machineType.scope())"
