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
	RunID                 string  `json:"run_id"`
	ServerSize            int     `json:"m"`
	ClientSize            int     `json:"n"`
	Mode                  string  `json:"mode"`
	ChunkSize             int     `json:"chunk_size"`
	WorkerCount           int     `json:"worker_count"`
	LeafIndexedFiltering  bool    `json:"leaf_indexed_filtering"`
	TargetedDecCalls      int     `json:"targeted_dec_calls"`
	AllPairsDecCalls      int     `json:"all_pairs_dec_calls"`
	CuckooRebuilds        int     `json:"cuckoo_rebuilds"`
	D                     int     `json:"D"`
	Q                     uint64  `json:"q"`
	QBits                 int     `json:"qBits"`
	NMatrix               int     `json:"N"`
	Sigma                 float64 `json:"sigma"`
	PeakHeapMB            uint64  `json:"peak_heap_mb"`
	InitSec               float64 `json:"init_sec"`
	EncSec                float64 `json:"enc_sec"`
	IntersectSec          float64 `json:"intersect_sec"`
	TotalSec              float64 `json:"total_sec"`
	MatchesFound          int     `json:"matches_found"`
	ActualDecCalls        int     `json:"actual_dec_calls"`
	TotalPossibleDecCalls int     `json:"total_possible_dec_calls"`
	ReductionFactor       float64 `json:"decryption_reduction_factor"`
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
	fmt.Printf("  mode            : explicit_chunked\n")
	fmt.Println("==================================================")

	serverSet := make([]uint64, serverSize)
	for i := range serverSet {
		serverSet[i] = uint64(i + 1)
	}

	overlap := 10
	if overlap > clientSize {
		overlap = clientSize
	}
	if overlap > serverSize {
		overlap = serverSize
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

	clientSet, actualOverlap := buildBenchmarkClientSet(serverSet, ctx.TreeIndices, ctx.LEParams.Layers, clientSize, overlap)
	fmt.Printf("  Client set prepared: expected_intersection=%d, non-overlap leaves avoid occupied server leaves\n", actualOverlap)

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
		RunID:                 runID,
		ServerSize:            serverSize,
		ClientSize:            clientSize,
		Mode:                  stats.Mode,
		ChunkSize:             stats.ChunkSize,
		WorkerCount:           stats.WorkerCount,
		LeafIndexedFiltering:  stats.LeafIndexedFiltering,
		TargetedDecCalls:      stats.TargetedDecCalls,
		AllPairsDecCalls:      stats.AllPairsDecCalls,
		CuckooRebuilds:        ctx.CuckooRebuilds,
		D:                     ctx.LEParams.D,
		Q:                     ctx.LEParams.Q,
		QBits:                 ctx.LEParams.QBits,
		NMatrix:               ctx.LEParams.N,
		Sigma:                 ctx.LEParams.Sigma,
		PeakHeapMB:            peakHeap,
		InitSec:               initSec,
		EncSec:                encSec,
		IntersectSec:          intersectSec,
		TotalSec:              totalSec,
		MatchesFound:          len(matches),
		ActualDecCalls:        stats.ActualDecCalls,
		TotalPossibleDecCalls: stats.TotalPossibleDecCalls,
		ReductionFactor:       stats.ReductionFactor,
	}

	fmt.Println("\n==================================================")
	fmt.Printf("  TOTAL WALL TIME : %.2f min (%.1f sec)\n", totalSec/60, totalSec)
	fmt.Printf("  PEAK HEAP       : %d MB\n", peakHeap)
	fmt.Printf("  MATCHES         : %d / %d expected\n", len(matches), actualOverlap)
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
