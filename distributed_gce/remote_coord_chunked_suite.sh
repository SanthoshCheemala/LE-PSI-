#!/usr/bin/env bash
set -euo pipefail

WORKDIR="${WORKDIR:-/tmp/lepsi}"
RESULT_ROOT="${RESULT_ROOT:-/tmp/lepsi_remote_chunked_results}"
SIZES="${SIZES:-1000 2000 4000 8000 10000}"
N="${N:-100}"
K="${K:-7}"
RUN_LABEL="${RUN_LABEL:-chunked_remote}"

if [[ -z "${SHARD_URLS:-}" ]]; then
  echo "SHARD_URLS is required" >&2
  exit 1
fi

mkdir -p "$RESULT_ROOT"
cd "$WORKDIR"

echo "remote chunked distributed suite"
echo "timestamp=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo "run_label=$RUN_LABEL"
echo "sizes=$SIZES"
echo "n=$N k=$K"
echo "shard_urls=$SHARD_URLS"
echo "workdir=$WORKDIR result_root=$RESULT_ROOT"

for M in $SIZES; do
  RUN_ID="$(date -u +%Y%m%d_%H%M%S)_m${M}_n${N}_K${K}_${RUN_LABEL}"
  RUN_DIR="$RESULT_ROOT/$RUN_ID"
  mkdir -p "$RUN_DIR"

  echo ""
  echo "=== m=$M n=$N K=$K run_id=$RUN_ID ==="
  rm -f /tmp/coord.log /tmp/coord.pid /tmp/cts_payload.json

  START_NS="$(date +%s%N)"
  M="$M" N="$N" SHARD_URLS="$SHARD_URLS" RESULT_DIR="$RUN_DIR" \
    ./bin/lepsi_coord > "$RUN_DIR/coordinator.log" 2>&1
  END_NS="$(date +%s%N)"

  WALL_MS=$(( (END_NS - START_NS) / 1000000 ))
  echo "$WALL_MS" > "$RUN_DIR/wall_ms.txt"

  if grep -q "Results saved to" "$RUN_DIR/coordinator.log"; then
    echo "DONE m=$M wall_ms=$WALL_MS"
    grep -E "RESULT:|Total  :|Init   :|Intersect:|Matches:" "$RUN_DIR/coordinator.log" | tail -6 || true
  else
    echo "FAILED m=$M wall_ms=$WALL_MS" >&2
    tail -40 "$RUN_DIR/coordinator.log" >&2 || true
  fi
done

echo ""
echo "all remote chunked runs finished at $(date -u +%Y-%m-%dT%H:%M:%SZ)"
