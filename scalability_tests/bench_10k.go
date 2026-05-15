// bench_10k.go runs the tracked single-node 10K benchmark in explicit
// chunked mode. It intentionally uses psi.ServerInitializeChunked so witness
// material is generated only for the active chunk during intersection.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/SanthoshCheemala/LE-PSI/pkg/psi"
)

type BenchmarkResult struct {
	RunID                string  `json:"run_id"`
	ServerSize           int     `json:"m"`
	ClientSize           int     `json:"n"`
	Mode                 string  `json:"mode"`
	ChunkSize            int     `json:"chunk_size"`
	WorkerCount          int     `json:"worker_count"`
	LeafIndexedFiltering bool    `json:"leaf_indexed_filtering"`
	TargetedDecCalls     int     `json:"targeted_dec_calls"`
	AllPairsDecCalls     int     `json:"all_pairs_dec_calls"`
	CuckooRebuilds       int     `json:"cuckoo_rebuilds"`
	D                    int     `json:"D"`
	Q                    uint64  `json:"q"`
	Sigma                float64 `json:"sigma"`
	PeakHeapMB           uint64  `json:"peak_heap_mb"`
	InitSec              float64 `json:"init_sec"`
	EncSec               float64 `json:"enc_sec"`
	IntersectSec         float64 `json:"intersect_sec"`
	TotalSec             float64 `json:"total_sec"`
	MatchesFound         int     `json:"matches_found"`
}

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

func heapMB() uint64 {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	return mem.HeapAlloc / 1024 / 1024
}

func updatePeak(peak *uint64) {
	if current := heapMB(); current > *peak {
		*peak = current
	}
}

func main() {
	serverSize := envInt("M", 10000)
	clientSize := envInt("N", 100)
	chunkSize := envInt("LEPSI_CHUNK_SIZE", 256)
	workerCount := envInt("LEPSI_WORKERS", runtime.NumCPU())
	runID := fmt.Sprintf("bench_10k_chunked_%s", time.Now().UTC().Format("20060102_150405"))

	fmt.Println("==================================================")
	fmt.Printf("  SINGLE-NODE CHUNKED BENCHMARK\n")
	fmt.Printf("  m (server size) : %d\n", serverSize)
	fmt.Printf("  n (client size) : %d\n", clientSize)
	fmt.Printf("  chunk_size      : %d\n", chunkSize)
	fmt.Printf("  worker_count    : %d\n", workerCount)
	fmt.Printf("  mode            : chunked\n")
	fmt.Println("==================================================")

	serverSet := make([]uint64, serverSize)
	for i := range serverSet {
		serverSet[i] = uint64(i + 1)
	}

	clientSet := make([]uint64, clientSize)
	overlap := 10
	if overlap > clientSize {
		overlap = clientSize
	}
	for i := 0; i < clientSize-overlap; i++ {
		clientSet[i] = uint64(serverSize + i + 1000)
	}
	for i := 0; i < overlap; i++ {
		clientSet[clientSize-overlap+i] = serverSet[i]
	}

	dbPath := "_10k_bench.db"
	os.Remove(dbPath)
	defer os.Remove(dbPath)

	start := time.Now()
	peakHeap := heapMB()

	fmt.Printf("\n[Phase 1] Server init without eager witness materialization...\n")
	initStart := time.Now()
	ctx, err := psi.ServerInitializeChunked(serverSet, dbPath)
	if err != nil {
		panic(err)
	}
	initSec := time.Since(initStart).Seconds()
	updatePeak(&peakHeap)
	fmt.Printf("  ✓ Init done: %.1f s\n", initSec)

	fmt.Printf("\n[Phase 2] Client encryption...\n")
	encStart := time.Now()
	pp, msg, le := psi.GetPublicParameters(ctx)
	ciphertexts := psi.ClientEncrypt(clientSet, pp, msg, le)
	encSec := time.Since(encStart).Seconds()
	updatePeak(&peakHeap)
	fmt.Printf("  ✓ Encrypt done: %.1f s (%d ciphertexts)\n", encSec, len(ciphertexts))

	fmt.Printf("\n[Phase 3] Chunked leaf-indexed intersection...\n")
	intersectStart := time.Now()
	matches, stats, err := psi.DetectIntersectionChunkedWithContext(ctx, ciphertexts, psi.ChunkedDetectionOptions{
		ChunkSize:   chunkSize,
		WorkerCount: workerCount,
		ForceGC:     true,
	})
	if err != nil {
		panic(err)
	}
	intersectSec := time.Since(intersectStart).Seconds()
	updatePeak(&peakHeap)
	fmt.Printf("  ✓ Intersection done: %.1f s\n", intersectSec)

	totalSec := time.Since(start).Seconds()
	resultObj := BenchmarkResult{
		RunID:                runID,
		ServerSize:           serverSize,
		ClientSize:           clientSize,
		Mode:                 stats.Mode,
		ChunkSize:            stats.ChunkSize,
		WorkerCount:          stats.WorkerCount,
		LeafIndexedFiltering: stats.LeafIndexedFiltering,
		TargetedDecCalls:     stats.TargetedDecCalls,
		AllPairsDecCalls:     stats.AllPairsDecCalls,
		CuckooRebuilds:       ctx.CuckooRebuilds,
		D:                    ctx.LEParams.D,
		Q:                    ctx.LEParams.Q,
		Sigma:                ctx.LEParams.Sigma,
		PeakHeapMB:           peakHeap,
		InitSec:              initSec,
		EncSec:               encSec,
		IntersectSec:         intersectSec,
		TotalSec:             totalSec,
		MatchesFound:         len(matches),
	}

	fmt.Println("\n==================================================")
	fmt.Printf("  TOTAL WALL TIME : %.2f min (%.1f sec)\n", totalSec/60, totalSec)
	fmt.Printf("  PEAK HEAP       : %d MB\n", peakHeap)
	fmt.Printf("  MATCHES         : %d / %d expected\n", len(matches), overlap)
	fmt.Printf("  TARGETED DEC    : %d vs %d all-pairs\n", stats.TargetedDecCalls, stats.AllPairsDecCalls)
	fmt.Println("==================================================")

	os.MkdirAll("scalability_results", 0755)
	fileName := fmt.Sprintf("bench_10k_%s.json", time.Now().UTC().Format("20060102_150405"))
	outPath := filepath.Join("scalability_results", fileName)
	file, err := os.Create(outPath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(resultObj); err != nil {
		panic(err)
	}

	fmt.Printf("\n✓ Saved benchmark data to: %s\n\n", outPath)
}
