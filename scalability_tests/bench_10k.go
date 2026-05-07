// bench_10k.go
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	"github.com/SanthoshCheemala/LE-PSI/pkg/LE"
	"github.com/SanthoshCheemala/LE-PSI/pkg/matrix"
	"github.com/SanthoshCheemala/LE-PSI/pkg/psi"
	"github.com/SanthoshCheemala/LE-PSI/utils"
	_ "github.com/mattn/go-sqlite3"
	"github.com/tuneinsight/lattigo/v3/ring"
	lattigo_utils "github.com/tuneinsight/lattigo/v3/utils"
)

type BenchmarkResult struct {
	ServerSize int     `json:"server_size"`
	ClientSize int     `json:"client_size"`
	MaxWorkers int     `json:"max_workers"`
	PeakRAM_MB float64 `json:"peak_ram_mb"`
	InitSec    float64 `json:"init_time_sec"`
	EncSec     float64 `json:"enc_time_sec"`
	IntSec     float64 `json:"int_time_sec"`
	TotalSec   float64 `json:"total_time_sec"`
}

func main() {
	serverSize := 10000
	clientSize := 100
	maxWorkers := 77 // The paper's bounded batching limit

	fmt.Println("==================================================")
	fmt.Printf("  SINGLE-NODE BENCHMARK (As reported in paper)\n")
	fmt.Printf("  m (server size) : %d\n", serverSize)
	fmt.Printf("  n (client size) : %d\n", clientSize)
	fmt.Printf("  Workers capped  : %d\n", maxWorkers)
	fmt.Printf("  Security        : D=256 (Fast Evaluation)\n")
	fmt.Println("==================================================")

	// Set a reasonable GC memory limit to prevent OOM
	debug.SetMemoryLimit(16 * 1024 * 1024 * 1024) // 16 GB soft limit

	start := time.Now()
	peakRAM := getRAM_MB()

	// ── Generate synthetic data
	serverData := make([]interface{}, serverSize)
	for i := 0; i < serverSize; i++ {
		serverData[i] = map[string]interface{}{"id": fmt.Sprintf("TXN-%d", i)}
	}
	clientData := serverData[:clientSize]

	serverStrings, _ := utils.PrepareDataForPSI(serverData)
	clientStrings, _ := utils.PrepareDataForPSI(clientData)
	serverHashes := utils.HashDataPoints(serverStrings)
	clientHashes := utils.HashDataPoints(clientStrings)
	X_size := len(serverHashes)

	// ── Phase 1: Server Initialization
	fmt.Printf("\n[Phase 1] Server init (m=%d)...\n", X_size)
	initStart := time.Now()

	// Force D=256
	os.Setenv("PSI_SECURITY_LEVEL", "64")
	leParams, _ := psi.SetupLEParameters(X_size)

	dbPath := fmt.Sprintf("_10k_bench.db")
	os.Remove(dbPath)
	db, _ := sql.Open("sqlite3", dbPath)
	defer db.Close()

	for i := 0; i <= leParams.Layers; i++ {
		db.Exec(fmt.Sprintf("CREATE TABLE IF NOT EXISTS tree_%d (p1 BLOB, p2 BLOB, P3 BLOB, p4 BLOB, y_def BOOLEAN)", i))
	}

	privateKeys := make([]*matrix.Vector, X_size)
	hashedServer := make([]uint64, X_size)
	pkSlice := make([]*matrix.Vector, X_size)

	var initWG sync.WaitGroup
	initSem := make(chan struct{}, runtime.NumCPU())
	for i := 0; i < X_size; i++ {
		initWG.Add(1)
		go func(idx int) {
			defer initWG.Done()
			initSem <- struct{}{}
			pk, sk := leParams.KeyGen()
			hash := psi.ReduceToTreeIndex(serverHashes[idx], leParams.Layers)
			pkSlice[idx] = pk
			privateKeys[idx] = sk
			hashedServer[idx] = hash
			<-initSem
		}(i)
	}
	initWG.Wait()

	for i := 0; i < X_size; i++ {
		LE.Upd(db, hashedServer[i], leParams.Layers, pkSlice[i], leParams)
	}
	pkSlice = nil
	runtime.GC()

	pp := LE.ReadFromDB(db, 0, 0, leParams).NTT(leParams.R)
	msg := matrix.NewRandomPolyBinary(leParams.R)

	fmt.Printf("  Loading Merkle tree into RAM...\n")
	memoryTree, _ := LE.LoadTreeFromDB(db, leParams.Layers, leParams)

	updatePeak(&peakRAM)
	initTime := time.Since(initStart).Seconds()
	fmt.Printf("  ✓ Init done: %.1f s | Peak RAM: %.0f MB\n", initTime, peakRAM)

	// ── Phase 2: Client Encryption
	fmt.Printf("\n[Phase 2] Encrypting %d client queries...\n", clientSize)
	encStart := time.Now()

	ciphertexts := make([]psi.Cxtx, len(clientHashes))
	var encWG sync.WaitGroup
	for i := 0; i < len(clientHashes); i++ {
		encWG.Add(1)
		go func(idx int) {
			defer encWG.Done()
			treeIdx := psi.ReduceToTreeIndex(clientHashes[idx], leParams.Layers)
			prng, _ := lattigo_utils.NewPRNG()
			gaussianSampler := ring.NewGaussianSampler(prng, leParams.R, leParams.Sigma, leParams.Bound)

			r := make([]*matrix.Vector, leParams.Layers+1)
			for j := 0; j < leParams.Layers+1; j++ {
				r[j] = matrix.NewRandomVec(leParams.N, leParams.R, prng).NTT(leParams.R)
			}
			e := gaussianSampler.ReadNew()
			e0 := make([]*matrix.Vector, leParams.Layers+1)
			e1 := make([]*matrix.Vector, leParams.Layers+1)
			for j := 0; j < leParams.Layers+1; j++ {
				if j == leParams.Layers {
					e0[j] = matrix.NewNoiseVec(leParams.M2, leParams.R, prng, leParams.Sigma, leParams.Bound).NTT(leParams.R)
				} else {
					e0[j] = matrix.NewNoiseVec(leParams.M, leParams.R, prng, leParams.Sigma, leParams.Bound).NTT(leParams.R)
				}
				e1[j] = matrix.NewNoiseVec(leParams.M, leParams.R, prng, leParams.Sigma, leParams.Bound).NTT(leParams.R)
			}
			c0, c1, cvec, dpoly := LE.Enc(leParams, pp, treeIdx, msg, r, e0, e1, e)
			ciphertexts[idx] = psi.Cxtx{C0: c0, C1: c1, C: cvec, D: dpoly}
		}(i)
	}
	encWG.Wait()

	updatePeak(&peakRAM)
	encTime := time.Since(encStart).Seconds()
	fmt.Printf("  ✓ Encrypt done: %.1f s\n", encTime)

	// ── Phase 3: Intersection (BOUNDED BATCHING)
	fmt.Printf("\n[Phase 3] Intersection (Workers Capped at %d)...\n", maxWorkers)
	intStart := time.Now()

	var matchMu sync.Mutex
	matches := make([]uint64, 0)
	matchMap := make(map[int]bool)

	// HERE IS THE BOUNDED BATCHING:
	workerSem := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup

	for i := 0; i < X_size; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			workerSem <- struct{}{} // Cap concurrent goroutines
			defer func() {
				<-workerSem
				runtime.GC() // Clean up witness immediately
			}()

			w1, w2 := LE.WitGenMemory(memoryTree, leParams, hashedServer[idx])
			sk := privateKeys[idx]

			for j := 0; j < len(ciphertexts); j++ {
				msg2 := LE.Dec(leParams, sk, w1, w2, ciphertexts[j].C0, ciphertexts[j].C1, ciphertexts[j].C, ciphertexts[j].D)

				if psi.CorrectnessCheck(msg2, msg, leParams) {
					matchMu.Lock()
					if !matchMap[idx] {
						matches = append(matches, serverHashes[idx])
						matchMap[idx] = true
					}
					matchMu.Unlock()
				}
			}
		}(i)
	}
	wg.Wait()

	updatePeak(&peakRAM)
	intTime := time.Since(intStart).Seconds()
	fmt.Printf("  ✓ Intersection done: %.1f s\n", intTime)

	// ── Summary
	totalSec := time.Since(start).Seconds()
	fmt.Println("\n==================================================")
	fmt.Printf("  TOTAL WALL TIME : %.2f min (%.1f sec)\n", totalSec/60, totalSec)
	fmt.Printf("  PEAK RAM        : %.2f GB (%.1f MB)\n", peakRAM/1024, peakRAM)
	fmt.Printf("  MATCHES         : %d / %d expected\n", len(matches), clientSize)
	fmt.Println("==================================================")

	// Save to JSON
	resultObj := BenchmarkResult{
		ServerSize: serverSize,
		ClientSize: clientSize,
		MaxWorkers: maxWorkers,
		PeakRAM_MB: peakRAM,
		InitSec:    initTime,
		EncSec:     encTime,
		IntSec:     intTime,
		TotalSec:   totalSec,
	}

	os.MkdirAll("scalability_results", 0755)
	fileName := fmt.Sprintf("bench_10k_%s.json", time.Now().Format("20060102_150405"))
	outPath := filepath.Join("scalability_results", fileName)
	file, _ := os.Create(outPath)
	defer file.Close()
	
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	encoder.Encode(resultObj)

	fmt.Printf("\n✓ Saved benchmark data to: %s\n\n", outPath)
}

func getRAM_MB() float64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return float64(m.HeapAlloc) / 1024 / 1024
}

func updatePeak(peak *float64) {
	cur := getRAM_MB()
	if cur > *peak {
		*peak = cur
	}
}
