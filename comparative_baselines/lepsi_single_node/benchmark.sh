#!/bin/bash
# ============================================================
#  LE-PSI Single-Node Benchmark (for fair comparison with APSI)
#  Run on the SAME GCE VM as the APSI benchmark.
#
#  This runs our LE-PSI protocol on a single node (no sharding)
#  to produce apples-to-apples timing against Microsoft APSI.
#
#  Usage:
#    1. Upload the entire PSI repo to the VM
#    2. SSH in and run: bash comparative_baselines/lepsi_single_node/benchmark.sh
# ============================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
RESULTS_DIR="/tmp/lepsi_single_results"

mkdir -p "$RESULTS_DIR"

echo "════════════════════════════════════════════════════════"
echo "  LE-PSI SINGLE-NODE BENCHMARK (Comparative)"
echo "  Machine: $(hostname) | CPUs: $(nproc) | RAM: $(free -g | awk '/Mem/{print $2}')G"
echo "  Date: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo "════════════════════════════════════════════════════════"

# ── Build LE-PSI bench binary ─────────────────────────────
echo ""
echo "[Step 1] Building LE-PSI benchmark binary..."
cd "$REPO_ROOT"

cat > /tmp/lepsi_single_bench.go <<'GOEOF'
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

	// Generate server dataset
	serverSet := make([]uint64, m)
	for i := range serverSet {
		serverSet[i] = uint64(i + 1)
	}
	// Generate client dataset (last 10 overlap with server)
	clientSet := make([]uint64, n)
	overlap := 10
	for i := 0; i < n-overlap; i++ {
		clientSet[i] = uint64(m + i + 1000) // non-overlapping
	}
	for i := 0; i < overlap; i++ {
		clientSet[n-overlap+i] = serverSet[i] // overlapping
	}

	dbPath := fmt.Sprintf("/tmp/lepsi_bench_m%d.db", m)
	os.Remove(dbPath)
	defer os.Remove(dbPath)

	totalStart := time.Now()

	// Phase 1: Server init (keygen + tree build)
	initStart := time.Now()
	ctx, err := psi.ServerInitialize(serverSet, dbPath)
	if err != nil {
		log.Fatalf("ServerInitialize: %v", err)
	}
	initTime := time.Since(initStart)
	log.Printf("  Init: %v", initTime)

	// Phase 2: Client encrypt
	encStart := time.Now()
	pp, msg, le := psi.GetPublicParameters(ctx)
	ciphertexts := psi.ClientEncrypt(clientSet, pp, msg, le)
	encTime := time.Since(encStart)
	log.Printf("  Client encrypt: %v (%d ciphertexts)", encTime, len(ciphertexts))

	// Phase 3: Intersection
	intStart := time.Now()
	Z, err := psi.DetectIntersectionWithContext(ctx, ciphertexts)
	if err != nil {
		log.Fatalf("DetectIntersection: %v", err)
	}
	intTime := time.Since(intStart)
	log.Printf("  Intersection: %v", intTime)

	totalTime := time.Since(totalStart)
	log.Printf("  Total: %v, matches=%d", totalTime, len(Z))

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

	log.Printf("✓ Results saved to %s", outFile)
	b, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(b))
}
GOEOF

go build -o /tmp/lepsi_bench /tmp/lepsi_single_bench.go
echo "✓ Built /tmp/lepsi_bench"

# ── Run benchmarks ────────────────────────────────────────
echo ""
echo "[Step 2] Running benchmarks..."

SIZES=(1000 2000 4000 8000 10000)
N=100

for M in "${SIZES[@]}"; do
  echo ""
  echo "╔══════════════════════════════════════════════════╗"
  echo "║  LE-PSI: m=$M, n=$N (single-node)               "
  echo "╚══════════════════════════════════════════════════╝"

  /usr/bin/time -v /tmp/lepsi_bench "$M" "$N" \
    2>"$RESULTS_DIR/time_m${M}.txt"

  RSS=$(grep "Maximum resident" "$RESULTS_DIR/time_m${M}.txt" 2>/dev/null | awk '{print $NF}' || echo "0")
  echo "  Peak RSS: $((RSS/1024)) MB"
done

echo ""
echo "════════════════════════════════════════════════════════"
echo "  ALL LE-PSI SINGLE-NODE BENCHMARKS COMPLETE"
echo "  Results: $RESULTS_DIR/"
echo "════════════════════════════════════════════════════════"
ls -la "$RESULTS_DIR/"*.json
