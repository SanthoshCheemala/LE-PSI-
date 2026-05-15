#!/usr/bin/env bash
# ============================================================
#  run_all_benchmarks.sh — Run m=1K,2K,4K,8K,10K benchmarks
#  Assumes binaries are already deployed (run deploy_latest.sh first)
#  If a benchmark fails, continues to next size.
# ============================================================
set -uo pipefail

PROJECT="${PROJECT:-lepsi-distributed-493617}"
ZONE="${ZONE:-us-east1-b}"
K="${K:-7}"
N="${N:-100}"
WORKDIR='/tmp/lepsi'
RESULTS_DIR="$(cd "$(dirname "$0")" && pwd)/results"

SIZES=(1000 2000 4000 8000 10000)

echo "══════════════════════════════════════════════════"
echo "  LE-PSI DISTRIBUTED BENCHMARK SUITE"
echo "  Sizes  : ${SIZES[*]}"
echo "  K=$K shards, n=$N"
echo "══════════════════════════════════════════════════"

# ── Get VM info ──────────────────────────────────────────
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

SHARD_URLS=""
while IFS=, read -r name zone ip; do
  [[ -z "$ip" ]] && continue
  SHARD_URLS="${SHARD_URLS:+$SHARD_URLS,}http://${ip}:8081"
done <<< "$SHARD_ROWS"

ssh_cmd() {
  local name="$1" zone="$2" cmd="$3"
  gcloud compute ssh "$name" --project="$PROJECT" --zone="$zone" \
    --ssh-flag='-T' --ssh-flag='-o BatchMode=yes' --ssh-flag='-o ConnectTimeout=30' \
    --command="$cmd" < /dev/null
}

# ── Run each benchmark ───────────────────────────────────
for M in "${SIZES[@]}"; do
  echo ""
  echo "╔══════════════════════════════════════════════════╗"
  echo "║  m=$M, n=$N, K=$K                              "
  echo "╚══════════════════════════════════════════════════╝"

  RUN_ID="$(date +%Y%m%d_%H%M%S)_m${M}_n${N}_K${K}"
  mkdir -p "$RESULTS_DIR/$RUN_ID"

  # Kill stale processes
  ssh_cmd "$COORD_NAME" "$COORD_ZONE" \
    "pkill -9 lepsi_coord 2>/dev/null; rm -f /tmp/coord.log /tmp/coord.pid /tmp/cts_payload.json" || true
  while IFS=, read -r name zone ip; do
    [[ -z "$name" ]] && continue
    ssh_cmd "$name" "$zone" "pkill -9 lepsi_shard 2>/dev/null" || true
  done <<< "$SHARD_ROWS"
  sleep 3

  # Start shards
  echo "  Starting shards..."
  SHARD_ID=0
  SHARD_OK=true
  while IFS=, read -r name zone ip; do
    [[ -z "$name" ]] && continue
    for attempt in 1 2 3; do
      if ssh_cmd "$name" "$zone" \
        "cd $WORKDIR && pkill lepsi_shard 2>/dev/null || true; sleep 1; \
         SHARD_ID=$SHARD_ID PORT=8081 nohup ./bin/lepsi_shard > /tmp/shard_${SHARD_ID}.log 2>&1 & \
         sleep 3; curl -sf http://localhost:8081/health && echo OK || echo FAIL" 2>&1 | grep -q "OK"; then
        echo "    ✓ shard-$SHARD_ID"
        break
      fi
      sleep 10
      [[ $attempt -eq 3 ]] && { echo "    ✗ shard-$SHARD_ID FAILED"; SHARD_OK=false; }
    done
    SHARD_ID=$((SHARD_ID + 1))
  done <<< "$SHARD_ROWS"

  [[ "$SHARD_OK" != "true" ]] && { echo "  ✗ Skipping m=$M"; continue; }
  sleep 5

  # Run coordinator
  echo "  Running (m=$M)..."
  BENCH_START=$(date +%s)

  ssh_cmd "$COORD_NAME" "$COORD_ZONE" \
    "cd $WORKDIR && \
     M=$M N=$N SHARD_URLS='$SHARD_URLS' RESULT_DIR=/tmp/lepsi_results \
     nohup ./bin/lepsi_coord > /tmp/coord.log 2>&1 & \
     echo \$! > /tmp/coord.pid"

  # Poll for completion
  TIMEOUT=$((3 * 3600))
  ELAPSED=0
  while true; do
    if ssh_cmd "$COORD_NAME" "$COORD_ZONE" \
      "grep -q -E 'Results saved to|Benchmark failed' /tmp/coord.log 2>/dev/null"; then
      break
    fi
    if ! ssh_cmd "$COORD_NAME" "$COORD_ZONE" \
      "kill -0 \$(cat /tmp/coord.pid 2>/dev/null) 2>/dev/null"; then
      echo "  ✗ Coordinator died for m=$M"
      break
    fi
    sleep 30
    ELAPSED=$((ELAPSED + 30))
    [[ $ELAPSED -ge $TIMEOUT ]] && { echo "  ✗ Timeout m=$M"; break; }
    MINS=$((ELAPSED / 60))
    LAST=$(ssh_cmd "$COORD_NAME" "$COORD_ZONE" "tail -1 /tmp/coord.log 2>/dev/null" 2>/dev/null || echo "...")
    echo "  [${MINS}m] $LAST"
  done

  WALL=$(( $(date +%s) - BENCH_START ))
  echo "  Wall: $(echo "scale=1; $WALL/60" | bc) min"

  # Collect results
  gcloud compute scp "$COORD_NAME:/tmp/coord.log" \
    "$RESULTS_DIR/$RUN_ID/coordinator.log" \
    --project="$PROJECT" --zone="$COORD_ZONE" 2>/dev/null || true
  gcloud compute scp "$COORD_NAME:/tmp/lepsi_results/*.json" \
    "$RESULTS_DIR/$RUN_ID/" \
    --project="$PROJECT" --zone="$COORD_ZONE" 2>/dev/null || true
  SHARD_ID=0
  while IFS=, read -r name zone ip; do
    [[ -z "$name" ]] && continue
    gcloud compute scp "$name:/tmp/shard_${SHARD_ID}.log" \
      "$RESULTS_DIR/$RUN_ID/shard_${SHARD_ID}.log" \
      --project="$PROJECT" --zone="$zone" 2>/dev/null || true
    SHARD_ID=$((SHARD_ID + 1))
  done <<< "$SHARD_ROWS"

  # Summary
  if grep -q "RESULT:" "$RESULTS_DIR/$RUN_ID/coordinator.log" 2>/dev/null; then
    echo "  ✅ m=$M DONE"
    grep -E "Total|Init|Intersect|Matches|Peak" "$RESULTS_DIR/$RUN_ID/coordinator.log" | tail -5
  else
    echo "  ❌ m=$M FAILED"
    tail -3 "$RESULTS_DIR/$RUN_ID/coordinator.log" 2>/dev/null
  fi
done

echo ""
echo "══════════════════════════════════════════════════"
echo "  ALL BENCHMARKS COMPLETE — Results: $RESULTS_DIR/"
echo "══════════════════════════════════════════════════"
