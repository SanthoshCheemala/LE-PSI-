#!/bin/bash

# FLARE PSI Comprehensive Benchmark Suite
# Tests various dataset sizes and ring dimensions

echo "======================================"
echo "FLARE PSI Comprehensive Benchmark Suite"
echo "======================================"
echo ""

# Create results directory
mkdir -p benchmark_results

# Small datasets
echo "[1/6] Running small dataset benchmark (Server:20, Client:10, Ring:256)..."
go run benchmark_main.go -server-size=20 -client-size=10 -ring-dimension=256 -iterations=3
mv benchmark_results benchmark_results_small_256
echo ""

# Medium datasets with 256
echo "[2/6] Running medium dataset benchmark (Server:50, Client:20, Ring:256)..."
go run benchmark_main.go -server-size=50 -client-size=20 -ring-dimension=256 -iterations=3
mv benchmark_results benchmark_results_medium_256
echo ""

# Medium datasets with 512
echo "[3/6] Running medium dataset benchmark (Server:50, Client:20, Ring:512)..."
go run benchmark_main.go -server-size=50 -client-size=20 -ring-dimension=512 -iterations=3
mv benchmark_results benchmark_results_medium_512
echo ""

# Large datasets with 256
echo "[4/6] Running large dataset benchmark (Server:100, Client:50, Ring:256)..."
go run benchmark_main.go -server-size=100 -client-size=50 -ring-dimension=256 -iterations=2
mv benchmark_results benchmark_results_large_256
echo ""

# Large datasets with 512
echo "[5/6] Running large dataset benchmark (Server:100, Client:50, Ring:512)..."
go run benchmark_main.go -server-size=100 -client-size=50 -ring-dimension=512 -iterations=2
mv benchmark_results benchmark_results_large_512
echo ""

# Extra large with 1024
echo "[6/6] Running extra large benchmark (Server:200, Client:100, Ring:1024)..."
go run benchmark_main.go -server-size=200 -client-size=100 -ring-dimension=1024 -iterations=1
mv benchmark_results benchmark_results_xlarge_1024
echo ""

echo "======================================"
echo "Benchmark Suite Complete!"
echo "======================================"
echo ""
echo "Results saved in:"
echo "  - benchmark_results_small_256/"
echo "  - benchmark_results_medium_256/"
echo "  - benchmark_results_medium_512/"
echo "  - benchmark_results_large_256/"
echo "  - benchmark_results_large_512/"
echo "  - benchmark_results_xlarge_1024/"
echo ""
echo "View results with:"
echo "  cat benchmark_results_*/benchmark_result.json"
