# FLARE PSI Benchmarks

This directory contains benchmarking and testing tools for the FLARE PSI framework.

## Files

- **`psi_benchmark.go`** - Core benchmarking functions using new distributed PSI architecture
- **`benchmark_main.go`** - Main executable for running benchmarks
- **`run_benchmarks.sh`** - Shell script to run various benchmark scenarios

## Usage

### Build Benchmark Tool
```bash
cd benchmarks
go build -o psi_benchmark benchmark_main.go
```

### Run Benchmarks
```bash
# Run with default parameters
./psi_benchmark

# Run with custom parameters
./psi_benchmark -server-size=100 -client-size=50 -ring-dimension=512 -verbose

# Run comprehensive benchmark suite
./run_benchmarks.sh
```

### Benchmark Parameters

- `-server-size` - Number of items in server dataset (default: 50)
- `-client-size` - Number of items in client dataset (default: 20)
- `-ring-dimension` - Lattice ring dimension: 256, 512, 1024, or 2048 (default: 256)
- `-output-dir` - Directory for output files (default: "benchmark_results")
- `-verbose` - Enable verbose logging (default: false)
- `-iterations` - Number of benchmark iterations (default: 1)

## Output

Benchmarks generate several output files in the `benchmark_results/` directory:

- `performance_metrics.json` - Detailed performance metrics
- `timing_breakdown.json` - Timing breakdown by phase
- `parameter_analysis.json` - Analysis of cryptographic parameters
- `benchmark_report.html` - Visual HTML report

## What's Measured

1. **Initialization Time** - Server setup and key generation
2. **Encryption Time** - Client data encryption
3. **Detection Time** - Server intersection detection
4. **Total Time** - End-to-end PSI execution
5. **Throughput** - Operations per second
6. **Memory Usage** - Peak memory consumption
7. **CPU Utilization** - Multi-core efficiency

## Example Results

```
=== FLARE PSI Benchmark ===
Server Set Size: 50
Client Set Size: 20
Ring Dimension: 256

Results:
- Initialization: 312ms
- Encryption: 89ms
- Detection: 156ms
- Total: 557ms
- Throughput: 1795 ops/sec
- Intersection Found: 8 items
```

## Notes

- Benchmarks use the new distributed PSI architecture
- All timings are averaged over multiple iterations
- Results may vary based on hardware and dataset characteristics
- For production benchmarks, use larger datasets and more iterations
