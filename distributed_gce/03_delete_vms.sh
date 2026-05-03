#!/usr/bin/env bash
# ============================================================
#  03_delete_vms.sh — Teardown GCE cluster after benchmark
#  Run this immediately after collecting results to stop billing
# ============================================================
set -euo pipefail

PROJECT="${PROJECT:-distributed-sim}"

echo "Deleting all lepsi-dist VMs in project $PROJECT..."

INSTANCE_LIST="$(gcloud compute instances list \
  --project="$PROJECT" \
  --filter="labels.experiment=lepsi-dist" \
  --format='csv[no-heading](name,zone)')"

if [[ -z "$INSTANCE_LIST" ]]; then
  echo "No instances found. Nothing to delete."
  exit 0
fi

echo "Will delete:"
echo "$INSTANCE_LIST"
echo ""
read -rp "Are you sure? (yes/no): " CONFIRM
[[ "$CONFIRM" != "yes" ]] && { echo "Cancelled."; exit 0; }

while IFS=, read -r name zone; do
  [[ -z "$name" ]] && continue
  gcloud compute instances delete "$name" \
    --project="$PROJECT" \
    --zone="$zone" \
    --quiet &
done <<< "$INSTANCE_LIST"
wait

echo "✓ All VMs deleted. Billing stopped."
