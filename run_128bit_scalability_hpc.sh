#!/usr/bin/env bash
# =============================================================================
#  run_128bit_scalability_hpc.sh
#  128-bit Quantum Security — Scalability Test (D=2048, n=100 FIXED)
#
#  PURPOSE:
#    Scalability test with 128-bit security and batched witness generation.
#    Client size is FIXED at n=100 (larger than previous n=m/10 runs).
#    This isolates how server-side m-scaling behaves under a realistic
#    client workload, while keeping the test tractable on HPC.
#
#  HARDWARE : AMD EPYC 7763, 96 cores, 256 GB RAM
#  SECURITY : D=2048 (128-bit quantum)
#  BATCHING : semaphore limits concurrent witnesses to 77
#             (77 × 280 MB = ~21 GB witness RAM — safe on 256 GB)
#
#  ESTIMATED RUNTIMES (n=100 fixed):
#    m=100  : init~55s,  encrypt~270s, intersect~3000s   → ~55  min  ✓
#    m=250  : init~140s, encrypt~270s, intersect~7500s   → ~130 min  ✓
#    m=500  : init~320s, encrypt~270s, intersect~15430s  → ~267 min  ✓  ← KEY target
#    m=1000 : init~640s, encrypt~270s, intersect~30860s  → ~530 min  ✗  (too slow)
#
#  NOTE: m=1000, n=100 would take ~8.8 hrs single-node.
#        It can be enabled by adding 1000 to SIZES if you have a 12 hr slot.
#
#  TOTAL ESTIMATED: ~7.5 hours for m=100..500
# =============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/scalability_tests"

# ── Configuration ──────────────────────────────────────────────────────────────
WORKERS=77
CLIENT_SIZE=100          # FIXED at 100 as requested
MEM_LIMIT_GB=220         # leaves 36 GB headroom for OS on 256 GB node
BINARY="./bench_128bit_scalability"
RESULTS_DIR="scalability_results/128bit_n100_$(date +%Y%m%d_%H%M%S)"

# Server sizes — m=1000 commented out (see estimated runtimes above)
SIZES=(100 250 500)
# SIZES=(100 250 500 1000)   # uncomment if you have a 12-hour HPC slot

# Per-size timeout (9 hours — safe upper bound for m=500, n=100)
TIMEOUT_SEC=32400

# ── Banner ─────────────────────────────────────────────────────────────────────
echo "=========================================================="
echo "  LE-PSI 128-BIT SCALABILITY BENCHMARK — HPC RUN"
echo "  Date:     $(date)"
echo "  Hardware: AMD EPYC 7763, 96 cores, 256 GB RAM"
echo "  Security: D=2048 (128-bit quantum security)"
echo "  Workers:  ${WORKERS}  (batching — peak witness RAM: ~21 GB)"
echo "  Client:   n=${CLIENT_SIZE} FIXED"
echo "  MemLimit: ${MEM_LIMIT_GB} GB (of 256 GB)"
echo "  Sizes:    ${SIZES[*]}"
echo "=========================================================="
echo ""

# ── Build ──────────────────────────────────────────────────────────────────────
echo "[BUILD] Compiling main_128bit.go..."
if go build -o "$BINARY" main_128bit.go; then
    echo "        ✓ Build OK"
else
    echo "        ✗ Build FAILED — aborting"
    exit 1
fi

mkdir -p "$RESULTS_DIR"

# ── Master log ─────────────────────────────────────────────────────────────────
MASTER_LOG="${RESULTS_DIR}/hpc_128bit_n${CLIENT_SIZE}_$(date +%Y%m%d_%H%M%S).txt"
{
    echo "=========================================================="
    echo "  LE-PSI 128-BIT SCALABILITY BENCHMARK — HPC RUN"
    echo "  Date:     $(date)"
    echo "  Hardware: AMD EPYC 7763, 96 cores, 256 GB RAM"
    echo "  Security: D=2048 (128-bit quantum)"
    echo "  Workers:  ${WORKERS}"
    echo "  Client:   n=${CLIENT_SIZE} FIXED"
    echo "  Sizes:    ${SIZES[*]}"
    echo "=========================================================="
} | tee -a "$MASTER_LOG"

# ── Run each size independently ────────────────────────────────────────────────
PASSED=0; FAILED=0; TIMED_OUT=0

for M in "${SIZES[@]}"; do

    echo "" | tee -a "$MASTER_LOG"
    echo "----------------------------------------------------------" | tee -a "$MASTER_LOG"
    echo "  Running m=${M}, n=${CLIENT_SIZE} | $(date)"           | tee -a "$MASTER_LOG"
    echo "----------------------------------------------------------" | tee -a "$MASTER_LOG"

    SIZE_LOG="${RESULTS_DIR}/run_m${M}.log"
    START_TS=$(date +%s)

    if timeout "$TIMEOUT_SEC" env \
        PSI_SINGLE_SIZE="$M"         \
        PSI_CLIENT_SIZE="$CLIENT_SIZE" \
        PSI_MAX_WORKERS="$WORKERS"   \
        PSI_MEM_LIMIT_GB="$MEM_LIMIT_GB" \
        GOGC=25                      \
        "$BINARY" 2>&1 | tee -a "$SIZE_LOG" "$MASTER_LOG"; then

        END_TS=$(date +%s)
        WALL=$((END_TS - START_TS))
        echo "  ✓ m=${M} DONE — Wall time: ${WALL}s" | tee -a "$MASTER_LOG"
        cp "$SIZE_LOG" "${RESULTS_DIR}/SUCCESS_m${M}_n${CLIENT_SIZE}.log"
        PASSED=$((PASSED + 1))

    else
        EXIT_CODE=$?
        END_TS=$(date +%s)
        WALL=$((END_TS - START_TS))

        if [ "$EXIT_CODE" -eq 124 ]; then
            echo "  ✗ m=${M} TIMEOUT after ${WALL}s" | tee -a "$MASTER_LOG"
            cp "$SIZE_LOG" "${RESULTS_DIR}/TIMEOUT_m${M}_n${CLIENT_SIZE}.log"
            TIMED_OUT=$((TIMED_OUT + 1))
        else
            echo "  ✗ m=${M} FAILED (exit ${EXIT_CODE}) after ${WALL}s" | tee -a "$MASTER_LOG"
            cp "$SIZE_LOG" "${RESULTS_DIR}/FAILED_m${M}_n${CLIENT_SIZE}.log"
            FAILED=$((FAILED + 1))
        fi
        # continue to next size regardless
    fi

    # Brief pause to let GC settle between sizes
    sleep 15
done

# Cleanup binary
rm -f "$BINARY"

# ── Summary ────────────────────────────────────────────────────────────────────
echo "" | tee -a "$MASTER_LOG"
echo "==========================================================" | tee -a "$MASTER_LOG"
echo "  ALL SIZES COMPLETE" | tee -a "$MASTER_LOG"
echo "  Passed:    ${PASSED}" | tee -a "$MASTER_LOG"
echo "  Timed out: ${TIMED_OUT}" | tee -a "$MASTER_LOG"
echo "  Failed:    ${FAILED}" | tee -a "$MASTER_LOG"
echo "  End:       $(date)" | tee -a "$MASTER_LOG"
echo "  Results:   ${RESULTS_DIR}/" | tee -a "$MASTER_LOG"
echo "==========================================================" | tee -a "$MASTER_LOG"
