#!/bin/bash
# ============================================================
#  LE-PSI Single-Node Benchmark (for fair comparison with APSI)
#  Run on the SAME GCE VM as the APSI benchmark.
#
#  Usage:
#    1. Upload entire PSI repo to GCE VM:
#       gcloud compute scp --recurse \
#         /Users/santhoshcheemala/ALL_IN_ONE/Research_Implimentation/PSI/ \
#         psi-compare:/tmp/lepsi-repo/ --zone=us-east1-b
#
#    2. SSH in and run:
#       gcloud compute ssh psi-compare --zone=us-east1-b
#       export PATH=$PATH:/usr/local/go/bin
#       cd /tmp/lepsi-repo
#       bash comparative_baselines/lepsi_single_node/benchmark.sh
# ============================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
RESULTS_DIR="/tmp/lepsi_single_results"
BENCH_DIR="$REPO_ROOT/comparative_baselines/lepsi_single_node"

mkdir -p "$RESULTS_DIR"

echo "════════════════════════════════════════════════════════"
echo "  LE-PSI SINGLE-NODE BENCHMARK (Comparative)"
echo "  Machine: $(hostname) | CPUs: $(nproc) | RAM: $(free -g | awk '/Mem/{print $2}')G"
echo "  Date: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo "════════════════════════════════════════════════════════"

# ── Build the bench binary inside the repo module ─────────
echo ""
echo "[Step 1] Building LE-PSI benchmark binary..."

# Write Go benchmark source inside the repo so it can resolve module imports
cat > "$BENCH_DIR/bench_main.go" <<'GOEOF'
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/SanthoshCheemala/LE-PSI/pkg/psi"
)

func main() {
	if len(os.Args) < 3 {
		log.Fatal("Usage: lepsi_bench <m> <n>")
	}
	m, _ := strconv.Atoi(os.Args[1])
	n, _ := strconv.Atoi(os.Args[2])

	log.Printf("LE-PSI single-node benchmark: m=%d, n=%d", m, n)

	// Generate server dataset: {1, 2, ..., m}
	serverSet := make([]uint64, m)
	for i := range serverSet {
		serverSet[i] = uint64(i + 1)
	}

	// Generate client dataset: 10 overlap with server, rest unique
	clientSet := make([]uint64, n)
	overlap := 10
	if overlap > n { overlap = n }
	for i := 0; i < n-overlap; i++ {
		clientSet[i] = uint64(m + i + 1000)
	}
	for i := 0; i < overlap; i++ {
		clientSet[n-overlap+i] = serverSet[i]
	}

	dbPath := fmt.Sprintf("/tmp/lepsi_bench_m%d.db", m)
	os.Remove(dbPath)
	defer os.Remove(dbPath)

	totalStart := time.Now()

	// Phase 1: Server init
	log.Println("  Phase 1: Server init...")
	initStart := time.Now()
	ctx, err := psi.ServerInitialize(serverSet, dbPath)
	if err != nil {
		log.Fatalf("ServerInitialize: %v", err)
	}
	initTime := time.Since(initStart)
	log.Printf("  Init done: %v", initTime)

	// Phase 2: Client encrypt
	log.Println("  Phase 2: Client encrypt...")
	encStart := time.Now()
	pp, msg, le := psi.GetPublicParameters(ctx)
	ciphertexts := psi.ClientEncrypt(clientSet, pp, msg, le)
	encTime := time.Since(encStart)
	log.Printf("  Encrypt done: %v (%d ciphertexts)", encTime, len(ciphertexts))

	// Phase 3: Intersection
	log.Println("  Phase 3: Intersection...")
	intStart := time.Now()
	Z, err := psi.DetectIntersectionWithContext(ctx, ciphertexts)
	if err != nil {
		log.Fatalf("DetectIntersection: %v", err)
	}
	intTime := time.Since(intStart)
	log.Printf("  Intersection done: %v, matches=%d", intTime, len(Z))

	totalTime := time.Since(totalStart)

	// Memory stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	result := map[string]interface{}{
		"protocol":              "LE-PSI (single-node, leaf-filtered)",
		"timestamp":             time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		"machine":               func() string { h, _ := os.Hostname(); return h }(),
		"cpus":                  runtime.NumCPU(),
		"server_dataset_size":   m,
		"client_dataset_size":   n,
		"expected_intersection": overlap,
		"matches_found":         len(Z),
		"init_time_ms":          initTime.Milliseconds(),
		"encrypt_time_ms":       encTime.Milliseconds(),
		"intersect_time_ms":     intTime.Milliseconds(),
		"total_time_ms":         totalTime.Milliseconds(),
		"heap_alloc_mb":         memStats.HeapAlloc / 1024 / 1024,
		"total_alloc_mb":        memStats.TotalAlloc / 1024 / 1024,
	}

	outFile := fmt.Sprintf("/tmp/lepsi_single_results/lepsi_m%d_n%d.json", m, n)
	f, err := os.Create(outFile)
	if err != nil {
		log.Fatal(err)
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	enc.Encode(result)
	f.Close()

	log.Printf("✓ Results: %s", outFile)
	b, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(b))
}
GOEOF

cd "$REPO_ROOT"
go build -o /tmp/lepsi_bench ./comparative_baselines/lepsi_single_node/
chmod +x /tmp/lepsi_bench
echo "✓ Built and ready: /tmp/lepsi_bench"

# ── Run benchmarks ────────────────────────────────────────
echo ""
echo "[Step 2] Running benchmarks..."

SIZES=(1000 2000 4000 8000 10000)
N=100

for M in "${SIZES[@]}"; do
  echo ""
  echo "╔══════════════════════════════════════════════════╗"
  echo "║  LE-PSI single-node: m=$M, n=$N                  "
  echo "╚══════════════════════════════════════════════════╝"

  # Run and capture both stdout/stderr to a run log
  /usr/bin/time -v /tmp/lepsi_bench "$M" "$N" > "$RESULTS_DIR/run_m${M}.log" 2>&1 || {
    echo "  ❌ Run failed for m=$M. Check $RESULTS_DIR/run_m${M}.log"
    continue
  }

  RSS=$(grep "Maximum resident" "$RESULTS_DIR/run_m${M}.log" 2>/dev/null | awk '{print $NF}' || echo "0")
  echo "  Peak RSS: $((RSS/1024)) MB"

  # Print the last few lines of the run log (results summary)
  tail -n 5 "$RESULTS_DIR/run_m${M}.log" | grep -v "Maximum resident"
done

echo ""
echo "════════════════════════════════════════════════════════"
echo "  ALL LE-PSI SINGLE-NODE BENCHMARKS COMPLETE"
echo "  Results: $RESULTS_DIR/"
echo "════════════════════════════════════════════════════════"
ls -la "$RESULTS_DIR/"*.json 2>/dev/null || echo "No JSON results found"
