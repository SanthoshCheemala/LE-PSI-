// shard_server.go — HTTP shard server for Distributed Laconic PSI
// Each shard VM runs this binary. It:
//   1. Receives its slice of server records S_k via POST /init
//   2. Runs ServerInitialize (offline phase)
//   3. Exposes POST /intersect — receives client ciphertexts, returns match list
//   4. Exposes GET /health   — readiness probe

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/SanthoshCheemala/LE-PSI/pkg/psi"
	_ "github.com/mattn/go-sqlite3"
)

// ── Shared state ─────────────────────────────────────────
var (
	serverCtx   *psi.ServerInitContext
	shardID     int
	mu          sync.Mutex
	initialized bool
)

// ── Wire types ───────────────────────────────────────────

// InitRequest: coordinator pushes S_k and serialised public params
type InitRequest struct {
	ShardID     int           `json:"shard_id"`
	Records     []uint64      `json:"records"` // server element hashes for this shard
}

type InitResponse struct {
	ShardID    int                    `json:"shard_id"`
	Records    int                    `json:"records"`
	InitSec    float64                `json:"init_sec"`
	PeakRAMMB  float64                `json:"peak_ram_mb"`
	Params     *psi.SerializableParams `json:"params"` // coordinator forwards to client
}

// IntersectRequest: client ciphertexts forwarded by coordinator
type IntersectRequest struct {
	Ciphertexts []psi.SerializableCxtx `json:"ciphertexts"`
}

type MatchEntry struct {
	ServerIdx int    `json:"i"` // index within this shard's records
	ClientIdx int    `json:"j"`
}

type IntersectResponse struct {
	ShardID      int          `json:"shard_id"`
	Matches      []MatchEntry `json:"matches"`
	IntersectSec float64      `json:"intersect_sec"`
	PeakRAMMB    float64      `json:"peak_ram_mb"`
}

// ── Handlers ─────────────────────────────────────────────

func handleHealth(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	ready := initialized
	recs := 0
	if serverCtx != nil {
		recs = len(serverCtx.OriginalHashes)
	}
	mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"shard_id":%d,"ready":%v,"records":%d}`, shardID, ready, recs)
}

func handleInit(w http.ResponseWriter, r *http.Request) {
	var req InitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad JSON: "+err.Error(), 400)
		return
	}

	log.Printf("[shard %d] /init received: %d records", req.ShardID, len(req.Records))
	shardID = req.ShardID

	var memBefore runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memBefore)
	start := time.Now()

	dbPath := fmt.Sprintf("/tmp/shard_%d.db", req.ShardID)
	os.Remove(dbPath) // fresh start

	ctx, err := psi.ServerInitialize(req.Records, dbPath)
	if err != nil {
		http.Error(w, "ServerInitialize: "+err.Error(), 500)
		return
	}

	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	mu.Lock()
	serverCtx = ctx
	initialized = true
	mu.Unlock()

	pp, msg, le := psi.GetPublicParameters(ctx)
	params := psi.SerializeParameters(pp, msg, le)

	resp := InitResponse{
		ShardID:   req.ShardID,
		Records:   len(req.Records),
		InitSec:   time.Since(start).Seconds(),
		PeakRAMMB: float64(memAfter.Sys-memBefore.Sys) / (1024 * 1024),
		Params:    params,
	}

	log.Printf("[shard %d] init done in %.1fs", shardID, resp.InitSec)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleIntersect(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	if !initialized || serverCtx == nil {
		mu.Unlock()
		http.Error(w, "shard not initialized", 503)
		return
	}
	ctx := serverCtx
	mu.Unlock()

	var req IntersectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad JSON: "+err.Error(), 400)
		return
	}

	log.Printf("[shard %d] /intersect: %d ciphertexts vs %d records",
		shardID, len(req.Ciphertexts), len(ctx.OriginalHashes))

	// Deserialize ciphertexts
	ciphertexts := make([]psi.Cxtx, len(req.Ciphertexts))
	for i, sc := range req.Ciphertexts {
		ct, err := psi.DeserializeCxtx(sc, ctx.LEParams)
		if err != nil {
			http.Error(w, fmt.Sprintf("deserialize ct[%d]: %v", i, err), 400)
			return
		}
		ciphertexts[i] = ct
	}

	var memBefore runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memBefore)
	start := time.Now()

	// Run intersection using existing DetectIntersectionWithContext
	intersection, err := psi.DetectIntersectionWithContext(ctx, ciphertexts)
	if err != nil {
		http.Error(w, "intersection: "+err.Error(), 500)
		return
	}

	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	// Map intersection hash values back to (server_idx, client_idx) pairs
	// Build reverse map: hash -> shard index
	hashToShardIdx := make(map[uint64]int, len(ctx.OriginalHashes))
	for idx, h := range ctx.OriginalHashes {
		hashToShardIdx[h] = idx
	}
	// Build reverse map: hash -> client indices
	// We need to track which client ciphertext matched — use the existing detection
	// For now report shard-level matches (server element indices that matched)
	var matches []MatchEntry
	for _, h := range intersection {
		if si, ok := hashToShardIdx[h]; ok {
			matches = append(matches, MatchEntry{ServerIdx: si, ClientIdx: -1})
		}
	}

	resp := IntersectResponse{
		ShardID:      shardID,
		Matches:      matches,
		IntersectSec: time.Since(start).Seconds(),
		PeakRAMMB:    float64(memAfter.Sys-memBefore.Sys) / (1024 * 1024),
	}

	log.Printf("[shard %d] intersection done: %d matches in %.1fs",
		shardID, len(matches), resp.IntersectSec)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ── Main ─────────────────────────────────────────────────

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	sidStr := os.Getenv("SHARD_ID")
	if sidStr != "" {
		shardID, _ = strconv.Atoi(sidStr)
	}

	// Use all available CPUs
	runtime.GOMAXPROCS(runtime.NumCPU())

	mux := http.NewServeMux()
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/init", handleInit)
	mux.HandleFunc("/intersect", handleIntersect)

	log.Printf("LE-PSI shard server starting on :%s (GOMAXPROCS=%d)",
		port, runtime.NumCPU())

	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("ListenAndServe: %v", err)
	}
}
