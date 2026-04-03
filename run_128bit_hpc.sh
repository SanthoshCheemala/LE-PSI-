#!/bin/bash

# ======================================================================
# LE-PSI 128-bit Quantum Security (D=2048) Scalability Tests
# Run this on the HPC to get the REAL exact numbers for the paper
# ======================================================================

set -e

# Move to the directory where the script is located
cd "$(dirname "$0")"
PROJECT_DIR="$PWD"
cd "$PROJECT_DIR"

# Enforce 128-bit security via the environment variable we built
export PSI_SECURITY_LEVEL="128"
export GOGC=50 # Aggressive GC to keep RAM tight

# Output file
timestamp=$(date +"%Y%m%d_%H%M%S")
log_file="scalability_results/hpc_128bit_run_${timestamp}.txt"
mkdir -p scalability_results

echo "==========================================================" | tee -a "$log_file"
echo "Starting 128-bit (D=2048) LE-PSI Benchmarks" | tee -a "$log_file"
echo "Time: $(date)" | tee -a "$log_file"
echo "Security: 128-bit (PSI_SECURITY_LEVEL=128)" | tee -a "$log_file"
echo "Log file: $log_file" | tee -a "$log_file"
echo "==========================================================" | tee -a "$log_file"

# We only run the small sizes because 10k at 128-bit will take 10+ hours on 1 node
# We just need 50, 100, and 250 to PROVE the 4x overhead factor empirically
DATASET_SIZES=(50 100 250)

echo "Building test binary..." | tee -a "$log_file"
go build -o psi_test ./scalability_tests/main.go

for size in "${DATASET_SIZES[@]}"; do
    echo "" | tee -a "$log_file"
    echo "----------------------------------------------------------" | tee -a "$log_file"
    echo "Running benchmark at m=${size}" | tee -a "$log_file"
    echo "----------------------------------------------------------" | tee -a "$log_file"
    
    # Use the standard shell builtin 'time' since /usr/bin/time might not be installed
    time ./psi_test -size=$size -mode=full 2>&1 | tee -a "$log_file"
    
    echo "Finished m=${size} benchmark." | tee -a "$log_file"
    
    # Rest to let memory clear
    sleep 5
done

# Cleanup
rm psi_test

echo "" | tee -a "$log_file"
echo "==========================================================" | tee -a "$log_file"
echo "ALL TESTS COMPLETED." | tee -a "$log_file"
echo "Check $log_file for the exact runtime and Peak Memory limits!" | tee -a "$log_file"
echo "==========================================================" | tee -a "$log_file"
