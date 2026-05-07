#!/bin/bash
# ======================================================================
# LE-PSI 128-bit Security Benchmark — HPC Version
# Target: AMD EPYC 7763, 96 cores, 256 GB RAM
#
# Memory budget at D=2048:
#   Per-witness:       ~280 MB
#   Merkle tree (m=1000, D=2048): ~12 GB  (8x larger than D=256)
#   Available for witnesses: ~200 GB
#   Safe worker count: min(96 × 0.8, 200000/280) = min(77, 714) = 77
#
# Sizes to test: 50, 100, 250, 500, 1000
#   (Do NOT go beyond 1000 at D=2048 — tree alone needs ~12+ GB per 1K records)
# ======================================================================

set -e

cd "$(dirname "$0")"
PROJECT_DIR="$PWD"

# ── Security & Runtime Settings ──────────────────────────────────────────────
export PSI_SECURITY_LEVEL="128"     # Enables D=2048 in the codebase
export GOGC=25                       # Very aggressive GC (frees memory quickly)
export GOMEMLIMIT=220GiB             # Soft limit — triggers GC before OOM
export GOMAXPROCS=77                 # Use 77 of 96 cores (leave 20% for GC/OS)
export PSI_MAX_WORKERS=77            # Max concurrent witnesses (77 × 280 MB = ~21 GB)

# ── Output ───────────────────────────────────────────────────────────────────
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
RESULTS_DIR="scalability_results"
mkdir -p "$RESULTS_DIR"
LOG="$RESULTS_DIR/hpc_128bit_${TIMESTAMP}.txt"

echo "==========================================================" | tee "$LOG"
echo "  LE-PSI 128-BIT SECURITY BENCHMARK — HPC RUN"            | tee -a "$LOG"
echo "  Date:     $(date)"                                        | tee -a "$LOG"
echo "  Hardware: AMD EPYC 7763, 96 cores, 256 GB RAM"           | tee -a "$LOG"
echo "  Security: D=2048 (128-bit quantum)"                       | tee -a "$LOG"
echo "  Workers:  77 (peak witness RAM: ~21 GB)"                  | tee -a "$LOG"
echo "  Sizes:    50, 100, 250, 500, 1000"                        | tee -a "$LOG"
echo "==========================================================" | tee -a "$LOG"
echo "" | tee -a "$LOG"

# ── Build binary ─────────────────────────────────────────────────────────────
echo "[BUILD] Compiling main_128bit.go..." | tee -a "$LOG"
go build -o psi_128bit_bench ./scalability_tests/main_128bit.go
echo "        ✓ Build OK" | tee -a "$LOG"
echo "" | tee -a "$LOG"

# ── Run each size independently ───────────────────────────────────────────────
# Running sizes independently means a crash at m=1000 doesn't lose m=50/100/250 results
SIZES=(50 100 250 500 1000)

for SIZE in "${SIZES[@]}"; do
    echo "----------------------------------------------------------" | tee -a "$LOG"
    echo "  Running m=$SIZE, n=$(($SIZE / 10)) | $(date)"            | tee -a "$LOG"
    echo "----------------------------------------------------------" | tee -a "$LOG"

    T_START=$SECONDS
    export PSI_SINGLE_SIZE=$SIZE

    # 3-hour timeout per size (D=2048 at m=1000 may take ~10-14 hours single-node)
    if timeout 10800 ./psi_128bit_bench 2>&1 | tee -a "$LOG"; then
        ELAPSED=$((SECONDS - T_START))
        echo ""                                             | tee -a "$LOG"
        echo "  ✓ m=$SIZE DONE — Wall time: ${ELAPSED}s"  | tee -a "$LOG"
    else
        EXIT=$?
        echo ""                                            | tee -a "$LOG"
        if [ $EXIT -eq 124 ]; then
            echo "  ✗ m=$SIZE TIMEOUT (>3 hrs)"           | tee -a "$LOG"
        else
            echo "  ✗ m=$SIZE FAILED (exit $EXIT)"        | tee -a "$LOG"
        fi
        echo "  Continuing to next size..."                | tee -a "$LOG"
    fi

    echo "" | tee -a "$LOG"

    # Let the OS reclaim memory between runs
    sleep 10
done

# ── Cleanup & Summary ─────────────────────────────────────────────────────────
rm -f psi_128bit_bench

echo "==========================================================" | tee -a "$LOG"
echo "  ALL SIZES COMPLETE"                                        | tee -a "$LOG"
echo "  End: $(date)"                                              | tee -a "$LOG"
echo "  Results: $LOG"                                             | tee -a "$LOG"
echo "==========================================================" | tee -a "$LOG"
echo ""
echo "Copy results back with:"
echo "  scp hpc:/path/to/PSI/scalability_results/hpc_128bit_${TIMESTAMP}.txt ."
