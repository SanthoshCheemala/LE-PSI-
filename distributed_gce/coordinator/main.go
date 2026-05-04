// coordinator.go — Distributed LE-PSI Coordinator
// Runs on lepsi-coord VM. Orchestrates K shard VMs:
//   1. Pushes S_k records to each shard via POST /init
//   2. Collects serialized public params from shard-0 for client
//   3. Receives client ciphertexts and fans out to all shards in parallel
//   4. Aggregates match lists and returns intersection to client
//   5. Logs timing to results JSON in the existing scalability_results format

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/SanthoshCheemala/LE-PSI/pkg/psi"
)

// ── Config ────────────────────────────────────────────────

type Config struct {
	K         int      // number of shards
	M         int      // total server records
	N         int      // client records
	ShardURLs []string // "http://10.x.x.x:8081" for each shard
	ResultDir string
}

// ── Wire types ────────────────────────────────────────────

type ShardInitReq struct {
	ShardID int      `json:"shard_id"`
	Records []uint64 `json:"records"`
}

type ShardInitResp struct {
	ShardID   int                     `json:"shard_id"`
	Records   int                     `json:"records"`
	InitSec   float64                 `json:"init_sec"`
	PeakRAMMB float64                 `json:"peak_ram_mb"`
	Params    *psi.SerializableParams `json:"params"`
}

type ShardIntersectReq struct {
	Ciphertexts []psi.SerializableCxtx `json:"ciphertexts"`
}

type MatchEntry struct {
	ServerIdx int `json:"i"`
	ClientIdx int `json:"j"`
}

type ShardIntersectResp struct {
	ShardID      int          `json:"shard_id"`
	Matches      []MatchEntry `json:"matches"`
	IntersectSec float64      `json:"intersect_sec"`
	PeakRAMMB    float64      `json:"peak_ram_mb"`
}

// ── Benchmark result (same schema as scalability_tests/main.go) ──
type DistributedResult struct {
	Timestamp         string  `json:"timestamp"`
	M                 int     `json:"server_dataset_size"`
	N                 int     `json:"client_dataset_size"`
	K                 int     `json:"shards"`
	TotalTimeNS       int64   `json:"total_time_ns"`
	InitTimeNS        int64   `json:"init_time_ns"`
	IntersectTimeNS   int64   `json:"intersect_time_ns"`
	MatchesFound      int     `json:"matches_found"`
	PeakRAMPerShardMB float64 `json:"peak_ram_per_shard_mb"`
	Success           bool    `json:"success"`
	ErrorMessage      string  `json:"error_message,omitempty"`
}

// ── HTTP helpers ──────────────────────────────────────────

// longClient has a 24-hour timeout so shard init/intersect can
// run for hours without the coordinator dropping the connection.
var longClient = &http.Client{
	Timeout: 24 * time.Hour,
}

func postJSON(url string, body any, out any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	resp, err := longClient.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, raw)
	}
	return json.Unmarshal(raw, out)
}

// ── Phase 1: Init all shards ──────────────────────────────

func initShards(cfg Config, serverSet []uint64) ([]ShardInitResp, error) {
	// Partition records into K equal slices
	sliceSize := (cfg.M + cfg.K - 1) / cfg.K
	results := make([]ShardInitResp, cfg.K)
	errs := make([]error, cfg.K)
	var wg sync.WaitGroup

	for k := 0; k < cfg.K; k++ {
		k := k
		lo := k * sliceSize
		hi := lo + sliceSize
		if hi > cfg.M {
			hi = cfg.M
		}
		slice := serverSet[lo:hi]

		wg.Add(1)
		go func() {
			defer wg.Done()
			req := ShardInitReq{ShardID: k, Records: slice}
			log.Printf("[coord] → shard-%d /init: %d records", k, len(slice))
			err := postJSON(cfg.ShardURLs[k]+"/init", req, &results[k])
			if err != nil {
				errs[k] = fmt.Errorf("shard-%d init: %w", k, err)
				return
			}
			log.Printf("[coord] ✓ shard-%d init: %.1fs, %.0f MB",
				k, results[k].InitSec, results[k].PeakRAMMB)
		}()
	}
	wg.Wait()

	for k, e := range errs {
		if e != nil {
			return nil, fmt.Errorf("shard-%d failed: %w", k, e)
		}
	}
	return results, nil
}

// ── Phase 2: Client encryption (using shard-0 params) ────

func runClient(initResps []ShardInitResp, clientSet []uint64, cfg Config) ([]psi.SerializableCxtx, error) {
	// Use public params from shard-0 (all shards share same LE params)
	pp0, msg0, le0, err := psi.DeserializeParameters(initResps[0].Params)
	if err != nil {
		return nil, fmt.Errorf("deserialize params: %w", err)
	}

	// Build tree indices for client set
	treeIndices := make([]uint64, cfg.N)
	for j, c := range clientSet {
		treeIndices[j] = psi.ReduceToTreeIndex(c, le0.Layers)
	}

	// Encrypt
	rawCts := psi.Client(treeIndices, pp0, msg0, le0)

	// Serialize for HTTP transport
	serialized := make([]psi.SerializableCxtx, len(rawCts))
	for j, ct := range rawCts {
		serialized[j] = psi.SerializeCxtx(ct)
	}
	log.Printf("[coord] client: encrypted %d ciphertexts", len(serialized))
	return serialized, nil
}

// ── Phase 3: Fan out to all shards ───────────────────────

func fanOutIntersect(cfg Config, cts []psi.SerializableCxtx) ([]ShardIntersectResp, error) {
	results := make([]ShardIntersectResp, cfg.K)
	errs := make([]error, cfg.K)
	var wg sync.WaitGroup

	req := ShardIntersectReq{Ciphertexts: cts}

	for k := 0; k < cfg.K; k++ {
		k := k
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Printf("[coord] → shard-%d /intersect", k)
			err := postJSON(cfg.ShardURLs[k]+"/intersect", req, &results[k])
			if err != nil {
				errs[k] = fmt.Errorf("shard-%d intersect: %w", k, err)
				return
			}
			log.Printf("[coord] ✓ shard-%d: %d matches in %.1fs",
				k, len(results[k].Matches), results[k].IntersectSec)
		}()
	}
	wg.Wait()

	for _, e := range errs {
		if e != nil {
			return nil, e
		}
	}
	return results, nil
}

// ── Main benchmark run ────────────────────────────────────

func runBenchmark(cfg Config) (*DistributedResult, error) {
	res := &DistributedResult{
		Timestamp: time.Now().Format("2006-01-02_15-04-05"),
		M:         cfg.M,
		N:         cfg.N,
		K:         cfg.K,
	}

	// Generate synthetic datasets (same as scalability_tests/main.go)
	rng := rand.New(rand.NewSource(42))
	serverSet := make([]uint64, cfg.M)
	for i := range serverSet {
		serverSet[i] = rng.Uint64()
	}
	clientSet := make([]uint64, cfg.N)
	// First 10% of client set overlaps with server (matches)
	overlap := cfg.N / 10
	for j := 0; j < overlap; j++ {
		clientSet[j] = serverSet[j]
	}
	for j := overlap; j < cfg.N; j++ {
		clientSet[j] = rng.Uint64()
	}

	totalStart := time.Now()

	// Phase 1: Init
	initStart := time.Now()
	log.Printf("[coord] Phase 1: Initializing %d shards (m=%d, m/K=%d)...", cfg.K, cfg.M, cfg.M/cfg.K)
	initResps, err := initShards(cfg, serverSet)
	if err != nil {
		res.ErrorMessage = err.Error()
		return res, err
	}
	res.InitTimeNS = time.Since(initStart).Nanoseconds()
	log.Printf("[coord] ✓ All shards initialized in %.1f min", float64(res.InitTimeNS)/1e9/60)

	// Phase 2: Client encrypt
	log.Printf("[coord] Phase 2: Client encryption (n=%d)...", cfg.N)
	cts, err := runClient(initResps, clientSet, cfg)
	if err != nil {
		res.ErrorMessage = err.Error()
		return res, err
	}

	// Phase 3: Fan-out intersection
	intersectStart := time.Now()
	log.Printf("[coord] Phase 3: Fan-out intersection to %d shards...", cfg.K)
	shardResps, err := fanOutIntersect(cfg, cts)
	if err != nil {
		res.ErrorMessage = err.Error()
		return res, err
	}
	res.IntersectTimeNS = time.Since(intersectStart).Nanoseconds()

	// Aggregate
	totalMatches := 0
	var maxRAM float64
	for _, sr := range shardResps {
		totalMatches += len(sr.Matches)
		if sr.PeakRAMMB > maxRAM {
			maxRAM = sr.PeakRAMMB
		}
	}
	res.TotalTimeNS = time.Since(totalStart).Nanoseconds()
	res.MatchesFound = totalMatches
	res.PeakRAMPerShardMB = maxRAM
	res.Success = true

	log.Printf("[coord] ══════════════════════════════════════════")
	log.Printf("[coord] RESULT: m=%d n=%d K=%d", cfg.M, cfg.N, cfg.K)
	log.Printf("[coord]   Total  : %.2f min", float64(res.TotalTimeNS)/1e9/60)
	log.Printf("[coord]   Init   : %.2f min", float64(res.InitTimeNS)/1e9/60)
	log.Printf("[coord]   Intersect: %.2f min", float64(res.IntersectTimeNS)/1e9/60)
	log.Printf("[coord]   Matches: %d", res.MatchesFound)
	log.Printf("[coord]   Peak RAM/shard: %.0f MB", res.PeakRAMPerShardMB)
	log.Printf("[coord] ══════════════════════════════════════════")

	return res, nil
}

// ── Entry point ───────────────────────────────────────────

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Read config from env
	m, _ := strconv.Atoi(os.Getenv("M"))
	n, _ := strconv.Atoi(os.Getenv("N"))
	if m == 0 { m = 10000 }
	if n == 0 { n = 100   }

	// SHARD_URLS: comma-separated "http://10.x.x.x:8081,http://10.x.x.y:8081,..."
	shardURLsRaw := os.Getenv("SHARD_URLS")
	if shardURLsRaw == "" {
		log.Fatal("SHARD_URLS env var required")
	}
	shardURLs := strings.Split(shardURLsRaw, ",")
	k := len(shardURLs)

	resultDir := os.Getenv("RESULT_DIR")
	if resultDir == "" { resultDir = "/tmp/lepsi_results" }
	os.MkdirAll(resultDir, 0755)

	cfg := Config{
		K:         k,
		M:         m,
		N:         n,
		ShardURLs: shardURLs,
		ResultDir: resultDir,
	}

	log.Printf("LE-PSI Coordinator: m=%d n=%d K=%d", m, n, k)

	result, err := runBenchmark(cfg)
	if err != nil {
		log.Printf("Benchmark failed: %v", err)
		result.Success = false
	}

	// Save JSON in same format as scalability_results/
	outFile := fmt.Sprintf("%s/distributed_%s_m%d_n%d_K%d.json",
		resultDir, result.Timestamp, m, n, k)
	data, _ := json.MarshalIndent(result, "", "  ")
	os.WriteFile(outFile, data, 0644)
	log.Printf("Results saved to: %s", outFile)
	fmt.Println(string(data))
}
