#!/usr/bin/env bash
# ============================================================
#  auto_cleanup.sh — Watch for benchmark completion, then
#  delete all GCE VMs to stop billing.
#
#  Run this in a SEPARATE terminal tab while 02_deploy_and_run.sh
#  is running in the first tab.
# ============================================================
set -euo pipefail

PROJECT="${PROJECT:-lepsi-distributed-493617}"
ZONE="${ZONE:-us-central1-a}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CHECK_INTERVAL=60  # seconds between checks

echo "============================================================"
echo "  LE-PSI Auto-Cleanup Watcher"
echo "  Project : $PROJECT"
echo "  Zone    : $ZONE"
echo "  Checking every ${CHECK_INTERVAL}s for benchmark completion"
echo "  Started : $(date)"
echo "============================================================"
echo ""

# Wait for the deploy/benchmark script to finish
echo "[watcher] Watching for 02_deploy_and_run.sh to finish..."
while pgrep -f "02_deploy_and_run.sh" > /dev/null 2>&1; do
  echo "[watcher] $(date '+%H:%M:%S') — benchmark still running..."
  sleep "$CHECK_INTERVAL"
done

echo ""
echo "[watcher] ✓ Benchmark process finished at $(date)"
echo "[watcher] Waiting 30s for any final file writes..."
sleep 30

# Show results if they exist
LATEST_RESULT=$(ls -td "$SCRIPT_DIR"/results/*/ 2>/dev/null | head -1)
if [[ -n "$LATEST_RESULT" ]]; then
  echo ""
  echo "[watcher] Latest results in: $LATEST_RESULT"
  ls -lh "$LATEST_RESULT" 2>/dev/null
  echo ""
  # Print coordinator log if it has real results
  if [[ -f "$LATEST_RESULT/coordinator.log" ]]; then
    echo "[watcher] Coordinator log:"
    cat "$LATEST_RESULT/coordinator.log"
    echo ""
  fi
fi

# Delete all VMs
echo "[watcher] ══════════════════════════════════════════"
echo "[watcher] DELETING ALL GCE VMs..."
echo "[watcher] ══════════════════════════════════════════"
bash "$SCRIPT_DIR/03_delete_vms.sh"

echo ""
echo "[watcher] ✓ All VMs deleted. Billing stopped."
echo "[watcher] Finished at $(date)"
