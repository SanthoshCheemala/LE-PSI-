#!/usr/bin/env bash
# =============================================================================
#  run_10k_hpc.sh
#  Standalone runner for the 10,000 server record benchmark
#
#  This script correctly replicates the conditions reported in the paper:
#   - m = 10,000 (server items)
#   - n = 100 (client items)
#   - D = 256 (64-bit Fast Evaluation Security)
#   - Bounded Batching: 77 concurrent workers max
#
#  Usage on HPC:
#    ./run_10k_hpc.sh
# =============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "=========================================================="
echo "  LE-PSI 10K PAPER BENCHMARK"
echo "  Date: $(date)"
echo "=========================================================="

# Run the dedicated benchmark Go script
go run scalability_tests/bench_10k.go
