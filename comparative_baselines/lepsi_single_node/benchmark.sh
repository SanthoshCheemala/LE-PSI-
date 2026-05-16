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
BENCH_SRC="/tmp/lepsi_single_bench.go"

mkdir -p "$RESULTS_DIR"

CPU_COUNT="$(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo unknown)"
RAM_GB="$(free -g 2>/dev/null | awk '/Mem/{print $2}' || true)"
if [[ -z "$RAM_GB" ]]; then
  RAM_BYTES="$(sysctl -n hw.memsize 2>/dev/null || echo 0)"
  RAM_GB="$((RAM_BYTES / 1024 / 1024 / 1024))"
fi
if /usr/bin/time -v true >/dev/null 2>&1; then
  TIME_ARGS=(-v)
  RSS_DIVISOR=1024
else
  TIME_ARGS=(-l)
  RSS_DIVISOR=1048576
fi

echo "════════════════════════════════════════════════════════"
echo "  LE-PSI SINGLE-NODE BENCHMARK (Comparative)"
echo "  Machine: $(hostname) | CPUs: $CPU_COUNT | RAM: ${RAM_GB}G"
echo "  Date: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo "════════════════════════════════════════════════════════"

# ── Build the bench binary inside the repo module ─────────
echo ""
echo "[Step 1] Building LE-PSI benchmark binary..."

# Write the Go benchmark source to /tmp and build it from the module root.
cat > "$BENCH_SRC" <<'GOEOF'
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/SanthoshCheemala/LE-PSI/pkg/psi"
)

func envInt(name string, fallback int) int {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func envString(name string, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}

func gitCommit() string {
	out, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

func machineType() string {
	client := &http.Client{Timeout: 700 * time.Millisecond}
	req, err := http.NewRequest("GET", "http://metadata.google.internal/computeMetadata/v1/instance/machine-type", nil)
	if err != nil {
		return os.Getenv("MACHINE_TYPE")
	}
	req.Header.Set("Metadata-Flavor", "Google")
	resp, err := client.Do(req)
	if err != nil {
		return os.Getenv("MACHINE_TYPE")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return os.Getenv("MACHINE_TYPE")
	}
	parts := strings.Split(strings.TrimSpace(string(body)), "/")
	return parts[len(parts)-1]
}

func ramGB() float64 {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 && fields[0] == "MemTotal:" {
			kb, _ := strconv.ParseFloat(fields[1], 64)
			return kb / 1024 / 1024
		}
	}
	return 0
}

func currentRSSMB() uint64 {
	file, err := os.Open("/proc/self/status")
	if err != nil {
		return 0
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 && fields[0] == "VmRSS:" {
			kb, _ := strconv.ParseUint(fields[1], 10, 64)
			return kb / 1024
		}
	}
	return 0
}

func startMemoryMonitor(done <-chan struct{}) (*uint64, *uint64) {
	var peakRSS uint64
	var peakHeap uint64
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			rss := currentRSSMB()
			var memStats runtime.MemStats
			runtime.ReadMemStats(&memStats)
			heap := memStats.HeapAlloc / 1024 / 1024

			for {
				old := atomic.LoadUint64(&peakRSS)
				if rss <= old || atomic.CompareAndSwapUint64(&peakRSS, old, rss) {
					break
				}
			}
			for {
				old := atomic.LoadUint64(&peakHeap)
				if heap <= old || atomic.CompareAndSwapUint64(&peakHeap, old, heap) {
					break
				}
			}
			select {
			case <-done:
				return
			case <-ticker.C:
			}
		}
	}()
	return &peakRSS, &peakHeap
}

func buildBenchmarkClientSet(serverSet []uint64, treeIndices []uint64, layers int, n int, desiredOverlap int) ([]uint64, int) {
	occupiedLeaves := make(map[uint64]bool, len(treeIndices))
	for _, leaf := range treeIndices {
		occupiedLeaves[leaf] = true
	}

	overlapValues := make([]uint64, 0, desiredOverlap)
	for i, value := range serverSet {
		leaf1 := psi.ReduceToTreeIndex(value, layers)
		leaf2 := psi.ReduceToTreeIndex2(value, layers)
		otherLeaf := leaf1
		if treeIndices[i] == leaf1 {
			otherLeaf = leaf2
		}
		if otherLeaf != treeIndices[i] && !occupiedLeaves[otherLeaf] {
			overlapValues = append(overlapValues, value)
			if len(overlapValues) == desiredOverlap {
				break
			}
		}
	}

	clientSet := make([]uint64, n)
	nonOverlap := n - len(overlapValues)
	candidate := uint64(len(serverSet) + 1000)
	for i := 0; i < nonOverlap; i++ {
		for {
			leaf1 := psi.ReduceToTreeIndex(candidate, layers)
			leaf2 := psi.ReduceToTreeIndex2(candidate, layers)
			if !occupiedLeaves[leaf1] && !occupiedLeaves[leaf2] {
				clientSet[i] = candidate
				candidate++
				break
			}
			candidate++
		}
	}
	copy(clientSet[nonOverlap:], overlapValues)
	return clientSet, len(overlapValues)
}

func buildRandomClientSet(serverSet []uint64, n int, desiredOverlap int, seed int64) ([]uint64, int) {
	if desiredOverlap > n {
		desiredOverlap = n
	}
	if desiredOverlap > len(serverSet) {
		desiredOverlap = len(serverSet)
	}

	clientSet := make([]uint64, n)
	copy(clientSet, serverSet[:desiredOverlap])

	rng := rand.New(rand.NewSource(seed))
	seen := make(map[uint64]bool, n)
	for i := 0; i < desiredOverlap; i++ {
		seen[clientSet[i]] = true
	}

	minNonServerValue := uint64(len(serverSet) + 1)
	for i := desiredOverlap; i < n; i++ {
		for {
			candidate := rng.Uint64()
			if candidate < minNonServerValue {
				continue
			}
			if seen[candidate] {
				continue
			}
			seen[candidate] = true
			clientSet[i] = candidate
			break
		}
	}

	return clientSet, desiredOverlap
}

func matchCorrectness(clientSet []uint64, matches []uint64, expectedIntersection int) (int, int, bool) {
	clientValues := make(map[uint64]bool, len(clientSet))
	for _, value := range clientSet {
		clientValues[value] = true
	}

	matchedExpected := 0
	falsePositives := 0
	for _, match := range matches {
		if clientValues[match] {
			matchedExpected++
		} else {
			falsePositives++
		}
	}

	falseNegatives := expectedIntersection - matchedExpected
	if falseNegatives < 0 {
		falseNegatives = 0
	}
	return falsePositives, falseNegatives, falsePositives == 0 && falseNegatives == 0 && len(matches) == expectedIntersection
}

func main() {
	if len(os.Args) < 3 {
		log.Fatal("Usage: lepsi_bench <m> <n>")
	}
	m, _ := strconv.Atoi(os.Args[1])
	n, _ := strconv.Atoi(os.Args[2])
	chunkSize := envInt("LEPSI_CHUNK_SIZE", 256)
	workerCount := envInt("LEPSI_WORKERS", runtime.NumCPU())
	clientMode := envString("CLIENT_MODE", "controlled")
	clientSeed := int64(envInt("CLIENT_SEED", 20260515))
	runID := fmt.Sprintf("lepsi_single_%s_m%d_n%d", time.Now().UTC().Format("20060102_150405"), m, n)

	doneRSS := make(chan struct{})
	peakRSS, peakHeap := startMemoryMonitor(doneRSS)
	defer close(doneRSS)

	log.Printf("LE-PSI single-node benchmark: run_id=%s m=%d n=%d mode=explicit_chunked chunk_size=%d workers=%d client_mode=%s",
		runID, m, n, chunkSize, workerCount, clientMode)

	// Generate server dataset: {1, 2, ..., m}
	serverSet := make([]uint64, m)
	for i := range serverSet {
		serverSet[i] = uint64(i + 1)
	}

	overlap := 10
	if overlap > n {
		overlap = n
	}
	if overlap > m {
		overlap = m
	}

	dbPath := fmt.Sprintf("/tmp/lepsi_bench_m%d.db", m)
	os.Remove(dbPath)
	defer os.Remove(dbPath)

	totalStart := time.Now()

	// Phase 1: Server init
	log.Println("  Phase 1: Server init...")
	initStart := time.Now()
	ctx, err := psi.ServerInitializeChunked(serverSet, dbPath)
	if err != nil {
		log.Fatalf("ServerInitializeChunked: %v", err)
	}
	initTime := time.Since(initStart)
	log.Printf("  Init done: %v", initTime)

	nonOverlapAvoidsLeaves := false
	var clientSet []uint64
	var actualOverlap int
	switch clientMode {
	case "controlled":
		clientSet, actualOverlap = buildBenchmarkClientSet(serverSet, ctx.TreeIndices, ctx.LEParams.Layers, n, overlap)
		nonOverlapAvoidsLeaves = true
		log.Printf("  Client set prepared: mode=controlled expected_intersection=%d, non-overlap leaves avoid occupied server leaves", actualOverlap)
	case "random":
		clientSet, actualOverlap = buildRandomClientSet(serverSet, n, overlap, clientSeed)
		log.Printf("  Client set prepared: mode=random seed=%d expected_intersection=%d, non-overlap leaves are random", clientSeed, actualOverlap)
	default:
		log.Fatalf("unsupported CLIENT_MODE=%q; use controlled or random", clientMode)
	}

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
	Z, detectStats, err := psi.DetectIntersectionChunkedWithContext(ctx, ciphertexts, psi.ChunkedDetectionOptions{
		ChunkSize:   chunkSize,
		WorkerCount: workerCount,
		ForceGC:     true,
	})
	if err != nil {
		log.Fatalf("DetectIntersectionChunked: %v", err)
	}
	intTime := time.Since(intStart)
	log.Printf("  Intersection done: %v, matches=%d", intTime, len(Z))
	falsePositives, falseNegatives, correctnessPassed := matchCorrectness(clientSet, Z, actualOverlap)

	totalTime := time.Since(totalStart)

	// Memory stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	peakRSSMB := atomic.LoadUint64(peakRSS)
	if current := currentRSSMB(); current > peakRSSMB {
		peakRSSMB = current
	}
	peakHeapMB := atomic.LoadUint64(peakHeap)
	if current := memStats.HeapAlloc / 1024 / 1024; current > peakHeapMB {
		peakHeapMB = current
	}

	result := map[string]interface{}{
		"protocol":               "LE-PSI (single-node, chunked, leaf-filtered)",
		"run_id":                 runID,
		"timestamp":              time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		"git_commit":             gitCommit(),
		"machine":                func() string { h, _ := os.Hostname(); return h }(),
		"machine_type":           machineType(),
		"vcpu":                   runtime.NumCPU(),
		"vcpus":                  runtime.NumCPU(),
		"ram_gb":                 ramGB(),
		"m":                      m,
		"n":                      n,
		"D":                      ctx.LEParams.D,
		"q":                      ctx.LEParams.Q,
		"qBits":                  ctx.LEParams.QBits,
		"N":                      ctx.LEParams.N,
		"sigma":                  ctx.LEParams.Sigma,
		"mode":                   detectStats.Mode,
		"chunk_size":             detectStats.ChunkSize,
		"worker_count":           detectStats.WorkerCount,
		"chunks_processed":       detectStats.ChunksProcessed,
		"client_mode":            clientMode,
		"client_seed":            clientSeed,
		"non_overlap_avoids_occupied_leaves": nonOverlapAvoidsLeaves,
		"expected_intersection":  actualOverlap,
		"matches_found":          len(Z),
		"false_positive_count":   falsePositives,
		"false_negative_count":   falseNegatives,
		"correctness_passed":     correctnessPassed,
		"init_sec":               initTime.Seconds(),
		"enc_sec":                encTime.Seconds(),
		"intersect_sec":          intTime.Seconds(),
		"total_sec":              totalTime.Seconds(),
		"peak_heap_mb":           peakHeapMB,
		"peak_rss_mb":            peakRSSMB,
		"total_alloc_mb":         memStats.TotalAlloc / 1024 / 1024,
		"cuckoo_rebuilds":        ctx.CuckooRebuilds,
		"leaf_indexed_filtering": detectStats.LeafIndexedFiltering,
		"targeted_dec_calls":     detectStats.TargetedDecCalls,
		"all_pairs_dec_calls":    detectStats.AllPairsDecCalls,
		"actual_dec_calls":       detectStats.ActualDecCalls,
		"total_possible_dec_calls": detectStats.TotalPossibleDecCalls,
		"decryption_reduction_factor": detectStats.ReductionFactor,
	}

	outFile := fmt.Sprintf("/tmp/lepsi_single_results/lepsi_m%d_n%d.json", m, n)
	if clientMode != "controlled" {
		outFile = fmt.Sprintf("/tmp/lepsi_single_results/lepsi_m%d_n%d_%s.json", m, n, clientMode)
	}
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
go build -o /tmp/lepsi_bench "$BENCH_SRC"
chmod +x /tmp/lepsi_bench
echo "✓ Built and ready: /tmp/lepsi_bench"

# ── Run benchmarks ────────────────────────────────────────
echo ""
echo "[Step 2] Running benchmarks..."

if [[ -n "${LEPSI_SIZES:-}" ]]; then
  read -r -a SIZES <<< "$LEPSI_SIZES"
else
  SIZES=(1000 2000 4000 8000 10000)
fi
N="${N:-100}"
LEPSI_CHUNK_SIZE="${LEPSI_CHUNK_SIZE:-256}"
LEPSI_WORKERS="${LEPSI_WORKERS:-$CPU_COUNT}"
CLIENT_MODE="${CLIENT_MODE:-controlled}"
CLIENT_SEED="${CLIENT_SEED:-20260515}"

for M in "${SIZES[@]}"; do
  LOG_SUFFIX=""
  if [[ "$CLIENT_MODE" != "controlled" ]]; then
    LOG_SUFFIX="_${CLIENT_MODE}"
  fi
  echo ""
  echo "╔══════════════════════════════════════════════════╗"
  echo "║  LE-PSI chunked: m=$M, n=$N, chunk=$LEPSI_CHUNK_SIZE, workers=$LEPSI_WORKERS, client_mode=$CLIENT_MODE"
  echo "╚══════════════════════════════════════════════════╝"

  # Run and capture both stdout/stderr to a run log
  LEPSI_CHUNK_SIZE="$LEPSI_CHUNK_SIZE" LEPSI_WORKERS="$LEPSI_WORKERS" CLIENT_MODE="$CLIENT_MODE" CLIENT_SEED="$CLIENT_SEED" \
    /usr/bin/time "${TIME_ARGS[@]}" /tmp/lepsi_bench "$M" "$N" > "$RESULTS_DIR/run_m${M}${LOG_SUFFIX}.log" 2>&1 || {
    echo "  ❌ Run failed for m=$M. Check $RESULTS_DIR/run_m${M}${LOG_SUFFIX}.log"
    continue
  }

  RSS=$(awk 'tolower($0) ~ /maximum resident/ { for (i=1; i<=NF; i++) if ($i ~ /^[0-9]+$/) value=$i } END { print value+0 }' "$RESULTS_DIR/run_m${M}${LOG_SUFFIX}.log" 2>/dev/null || echo "0")
  echo "  Peak RSS: $((RSS/RSS_DIVISOR)) MB"

  # Print the last few lines of the run log (results summary)
  tail -n 5 "$RESULTS_DIR/run_m${M}${LOG_SUFFIX}.log" | grep -v "Maximum resident"
done

echo ""
echo "════════════════════════════════════════════════════════"
echo "  ALL LE-PSI SINGLE-NODE BENCHMARKS COMPLETE"
echo "  Results: $RESULTS_DIR/"
echo "════════════════════════════════════════════════════════"
ls -la "$RESULTS_DIR/"*.json 2>/dev/null || echo "No JSON results found"
