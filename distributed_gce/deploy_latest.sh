#!/usr/bin/env bash
# ============================================================
#  deploy_latest.sh — Upload latest code + rebuild binaries
#  Run this ONCE after code changes. Then run run_all_benchmarks.sh
# ============================================================
set -euo pipefail

PROJECT="${PROJECT:-lepsi-distributed-493617}"
ZONE="${ZONE:-us-east1-b}"
WORKDIR='/tmp/lepsi'
PSI_SRC="/Users/santhoshcheemala/ALL_IN_ONE/Research_Implimentation/PSI"

echo "[deploy] Getting VM list..."
COORD_ROW="$(gcloud compute instances list \
  --project="$PROJECT" \
  --filter="labels.experiment=lepsi-dist AND labels.role=coordinator AND status=RUNNING" \
  --format='csv[no-heading](name,zone)' | head -1)"
SHARD_ROWS="$(gcloud compute instances list \
  --project="$PROJECT" \
  --filter="labels.experiment=lepsi-dist AND labels.role=shard AND status=RUNNING" \
  --sort-by="labels.shard_id" \
  --format='csv[no-heading](name,zone)')"

COORD_NAME=$(echo "$COORD_ROW" | cut -d, -f1)
COORD_ZONE=$(echo "$COORD_ROW" | cut -d, -f2)

ssh_cmd() {
  gcloud compute ssh "$1" --project="$PROJECT" --zone="$2" \
    --ssh-flag='-T' --ssh-flag='-o BatchMode=yes' --ssh-flag='-o ConnectTimeout=30' \
    --command="$3" < /dev/null
}

upload_and_build() {
  local name="$1" zone="$2" binary="$3" buildpath="$4"
  echo "  [$name] Uploading..."
  ssh_cmd "$name" "$zone" "mkdir -p $WORKDIR/bin $WORKDIR/distributed_gce"
  for pkg in pkg internal utils go.mod go.sum; do
    gcloud compute scp --recurse "$PSI_SRC/$pkg" "$name:$WORKDIR/" \
      --project="$PROJECT" --zone="$zone" 2>/dev/null || true
  done
  gcloud compute scp --recurse "$PSI_SRC/distributed_gce/coordinator" \
    "$name:$WORKDIR/distributed_gce/" --project="$PROJECT" --zone="$zone" 2>/dev/null || true
  gcloud compute scp --recurse "$PSI_SRC/distributed_gce/shard" \
    "$name:$WORKDIR/distributed_gce/" --project="$PROJECT" --zone="$zone" 2>/dev/null || true
  echo "  [$name] Building $binary..."
  ssh_cmd "$name" "$zone" \
    "cd $WORKDIR && PATH=/usr/local/go/bin:\$PATH go build -o bin/$binary $buildpath 2>&1"
  echo "  [$name] ✓ Done"
}

echo ""
echo "[deploy] Coordinator: $COORD_NAME"
upload_and_build "$COORD_NAME" "$COORD_ZONE" "lepsi_coord" "./distributed_gce/coordinator/"

echo ""
echo "[deploy] Shards:"
while IFS=, read -r name zone; do
  [[ -z "$name" ]] && continue
  upload_and_build "$name" "$zone" "lepsi_shard" "./distributed_gce/shard/"
done <<< "$SHARD_ROWS"

echo ""
echo "✓ All binaries deployed. Now run:"
echo "  PROJECT=$PROJECT ZONE=$ZONE K=7 bash distributed_gce/run_all_benchmarks.sh"
