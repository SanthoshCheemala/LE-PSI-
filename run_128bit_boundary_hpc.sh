#!/usr/bin/env bash
# =============================================================================
#  run_128bit_boundary_hpc.sh
#  128-bit Quantum Security — Boundary Exploration (m=500–600, n=75–100)
#
#  PURPOSE:
#    Characterise performance in the boundary zone just above the previously
#    confirmed safe point (m=500, n=50 → 136 min, 9.5 GB — SUCCESS).
#    No OOM crashes are expected: ciphertexts are ~2.7 KB each, so doubling n
#    adds only ~135 KB of RAM.  Extra time is purely computational.
#
#  HARDWARE : AMD EPYC 7763, 96 cores, 256 GB RAM
#  SECURITY : D=2048 (128-bit quantum)
#  BATCHING : semaphore caps concurrent witnesses at 77
#             (77 × 280 MB = ~21.6 GB — safe on 256 GB)
#
#  TEST MATRIX:
#    Run 1 — m=500, n=75   estimated: ~200 min  (~3.3 hrs)
#    Run 2 — m=500, n=100  estimated: ~265 min  (~4.4 hrs) ← KEY TARGET
#    Run 3 — m=600, n=100  estimated: ~317 min  (~5.3 hrs) ← PUSH m FURTHER
#
#  TOTAL ESTIMATED WALL TIME:  ~13 hours (submit as overnight HPC job)
#  PER-SIZE TIMEOUT:           7 hours (25200 s) — safe upper bound
# =============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/scalability_tests"

# ── Configuration ──────────────────────────────────────────────────────────────
WORKERS=77
MEM_LIMIT_GB=220
TIMEOUT_SEC=25200          # 7-hour per-run hard limit
BINARY="./bench_128bit_boundary"
RESULTS_DIR="scalability_results/128bit_boundary_$(date +%Y%m%d_%H%M%S)"

# Test matrix: "m:n" pairs — ordered from fastest to slowest
TEST_PAIRS=("500:75" "500:100" "600:100")

# ── Banner ─────────────────────────────────────────────────────────────────────
echo "=========================================================="
echo "  LE-PSI 128-BIT BOUNDARY BENCHMARK — HPC RUN"
echo "  Date:     $(date)"
echo "  Hardware: AMD EPYC 7763, 96 cores, 256 GB RAM"
echo "  Security: D=2048 (128-bit quantum)"
echo "  Workers:  ${WORKERS}  (batched witness — peak ~21 GB)"
echo "  MemLimit: ${MEM_LIMIT_GB} GB"
echo "  Test pairs (m:n): ${TEST_PAIRS[*]}"
echo ""
echo "  Estimated total wall time: ~13 hours"
echo "  Previously stable:  m=500, n=50 — 136 min, 9.5 GB ✓"
echo "=========================================================="
echo ""

# ── Build ──────────────────────────────────────────────────────────────────────
echo "[BUILD] Compiling main_128bit.go..."
if go build -o "$BINARY" main_128bit.go; then
    echo "        ✓ Build OK"
else
    echo "        ✗ Build FAILED"
    exit 1
fi

mkdir -p "$RESULTS_DIR"
MASTER_LOG="${RESULTS_DIR}/master_$(date +%Y%m%d_%H%M%S).txt"

# Write banner to master log too
{
echo "=========================================================="
echo "  LE-PSI 128-BIT BOUNDARY BENCHMARK"
echo "  Date: $(date) | Workers: ${WORKERS} | MemLimit: ${MEM_LIMIT_GB} GB"
echo "  Test pairs: ${TEST_PAIRS[*]}"
echo "=========================================================="
} | tee -a "$MASTER_LOG"

# ── Estimated time reference table ────────────────────────────────────────────
{
echo ""
echo "  Estimated runtimes (extrapolated from m=500,n=50=128min):"
echo "    m=500, n=75  → ~200 min"
echo "    m=500, n=100 → ~265 min  ← key target"
echo "    m=600, n=100 → ~317 min"
echo ""
} | tee -a "$MASTER_LOG"

# ── Run each (m, n) pair ──────────────────────────────────────────────────────
PASSED=0; FAILED=0; TIMED_OUT=0

for PAIR in "${TEST_PAIRS[@]}"; do
    M="${PAIR%%:*}"
    N="${PAIR##*:}"

    echo "" | tee -a "$MASTER_LOG"
    echo "----------------------------------------------------------" | tee -a "$MASTER_LOG"
    echo "  Running m=${M}, n=${N} | $(date)" | tee -a "$MASTER_LOG"
    echo "----------------------------------------------------------" | tee -a "$MASTER_LOG"

    SIZE_LOG="${RESULTS_DIR}/run_m${M}_n${N}.log"
    START_TS=$(date +%s)

    if timeout "$TIMEOUT_SEC" env \
        PSI_SINGLE_SIZE="$M"         \
        PSI_CLIENT_SIZE="$N"         \
        PSI_MAX_WORKERS="$WORKERS"   \
        PSI_MEM_LIMIT_GB="$MEM_LIMIT_GB" \
        PSI_SECURITY_LEVEL=128       \
        GOGC=25                      \
        "$BINARY" 2>&1 | tee -a "$SIZE_LOG" "$MASTER_LOG"; then

        END_TS=$(date +%s)
        WALL=$(( END_TS - START_TS ))
        echo "" | tee -a "$MASTER_LOG"
        echo "  ✓ m=${M}, n=${N} DONE — Wall time: ${WALL}s ($(( WALL/60 )) min)" | tee -a "$MASTER_LOG"
        cp "$SIZE_LOG" "${RESULTS_DIR}/SUCCESS_m${M}_n${N}.log"
        PASSED=$(( PASSED + 1 ))

    else
        EXIT_CODE=$?
        END_TS=$(date +%s)
        WALL=$(( END_TS - START_TS ))

        if [ "$EXIT_CODE" -eq 124 ]; then
            echo "  ✗ m=${M}, n=${N} TIMEOUT after ${WALL}s" | tee -a "$MASTER_LOG"
            cp "$SIZE_LOG" "${RESULTS_DIR}/TIMEOUT_m${M}_n${N}.log"
            TIMED_OUT=$(( TIMED_OUT + 1 ))
        else
            echo "  ✗ m=${M}, n=${N} FAILED (exit ${EXIT_CODE}) after ${WALL}s" | tee -a "$MASTER_LOG"
            cp "$SIZE_LOG" "${RESULTS_DIR}/FAILED_m${M}_n${N}.log"
            FAILED=$(( FAILED + 1 ))
        fi
        # always continue to next pair
    fi

    # Brief pause between runs so Go GC and OS can reclaim memory cleanly
    echo "  [pause 20s before next run]" | tee -a "$MASTER_LOG"
    sleep 20
done

# Cleanup
rm -f "$BINARY"

# ── Summary ────────────────────────────────────────────────────────────────────
echo "" | tee -a "$MASTER_LOG"
echo "==========================================================" | tee -a "$MASTER_LOG"
echo "  BOUNDARY TEST COMPLETE" | tee -a "$MASTER_LOG"
echo "  Passed:    ${PASSED}" | tee -a "$MASTER_LOG"
echo "  Timed out: ${TIMED_OUT}" | tee -a "$MASTER_LOG"
echo "  Failed:    ${FAILED}" | tee -a "$MASTER_LOG"
echo "  End:       $(date)" | tee -a "$MASTER_LOG"
echo "  Results:   ${RESULTS_DIR}/" | tee -a "$MASTER_LOG"
echo "==========================================================" | tee -a "$MASTER_LOG"
