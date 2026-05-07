#!/usr/bin/env bash
# ============================================================
#  run_all_benchmarks.sh — Run distributed benchmarks at
#  multiple m values sequentially on existing VMs.
#  VMs must already be created via 01_create_vms.sh
# ============================================================
set -euo pipefail

PROJECT="${PROJECT:-lepsi-distributed-493617}"
ZONE="${ZONE:-us-central1-c}"
K="${K:-7}"
N=100
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# m values to benchmark (skip 1000 if already done)
M_VALUES=(1000 2000 4000 8000 10000)

echo "============================================================"
echo "  LE-PSI MULTI-BENCHMARK RUNNER"
echo "  Date    : $(date)"
echo "  Config  : K=$K, n=$N"
echo "  m values: ${M_VALUES[*]}"
echo "  Est time: ~1.5 hours total"
echo "============================================================"
echo ""

for m in "${M_VALUES[@]}"; do
  echo ""
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo "  STARTING: m=$m, n=$N, K=$K"
  echo "  Time: $(date)"
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  
  K=$K M=$m N=$N PROJECT="$PROJECT" ZONE="$ZONE" \
    bash "$SCRIPT_DIR/02_deploy_and_run.sh"
  
  echo ""
  echo "  ✓ Completed m=$m at $(date)"
  echo ""
  
  # Brief pause between runs to let shards fully clean up
  sleep 10
done

echo ""
echo "============================================================"
echo "  ALL BENCHMARKS COMPLETE at $(date)"
echo "  Results:"
ls -dt "$SCRIPT_DIR"/results/*/ | head -${#M_VALUES[@]}
echo "============================================================"

# Auto-delete VMs after all benchmarks
echo ""
echo "Deleting VMs to stop billing..."
bash "$SCRIPT_DIR/03_delete_vms.sh"
