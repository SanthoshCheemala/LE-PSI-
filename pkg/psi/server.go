package psi

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/SanthoshCheemala/LE-PSI/internal/storage"
	"github.com/SanthoshCheemala/LE-PSI/pkg/LE"
	"github.com/SanthoshCheemala/LE-PSI/pkg/matrix"
	_ "github.com/mattn/go-sqlite3"
	"github.com/tuneinsight/lattigo/v3/ring"
)

// ServerInitContext holds the initialized server-side state for PSI operations.
// This context is created by ServerInitialize and contains all cryptographic
// materials needed for intersection detection.
//
// Fields:
//   - PublicParams: Public parameter matrix shared with clients
//   - Message: Message polynomial for encryption
//   - LEParams: Lattice encryption parameters
//   - PrivateKeys: Server's private key vectors for decryption
//   - WitnessVectors1: First set of witness vectors for tree navigation
//   - WitnessVectors2: Second set of witness vectors for tree navigation
//   - TreeIndices: Mapped indices of server's dataset in the witness tree
//   - OriginalHashes: Original hash values of server's dataset elements
//   - DBPath: Path to the witness tree database file
//
// The context should be cleaned up after use by calling Cleanup() method
// to properly close database connections and free resources.
type ServerInitContext struct {
	PublicParams    *matrix.Vector
	Message         *ring.Poly
	LEParams        *LE.LE
	PrivateKeys     []*matrix.Vector
	WitnessVectors1 [][]*matrix.Vector
	WitnessVectors2 [][]*matrix.Vector
	MemoryTree      *LE.MemoryTree
	TreeIndices     []uint64
	OriginalHashes  []uint64
	DBPath          string
	CuckooRebuilds  int
}

type CuckooPlacementStats struct {
	Rebuilds int
	Failures int
}

type ChunkedDetectionOptions struct {
	ChunkSize   int
	WorkerCount int
	ForceGC     bool
}

type ChunkedDetectionStats struct {
	Mode                 string
	ChunkSize            int
	WorkerCount          int
	ChunksProcessed      int
	LeafIndexedFiltering bool
	TargetedDecCalls     int
	AllPairsDecCalls     int
}

func configureTreeBuildDB(db *sql.DB) error {
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	pragmas := []string{
		"PRAGMA journal_mode = MEMORY",
		"PRAGMA synchronous = OFF",
		"PRAGMA temp_store = MEMORY",
	}
	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			return fmt.Errorf("%s: %w", pragma, err)
		}
	}
	return nil
}

func placeCuckooLeaves(privateSet []uint64, layers int) ([]uint64, CuckooPlacementStats, error) {
	placement := make([]uint64, len(privateSet))
	occupied := make(map[uint64]int, len(privateSet))

	var assign func(record int, seen map[uint64]bool) bool
	assign = func(record int, seen map[uint64]bool) bool {
		candidates := [2]uint64{
			ReduceToTreeIndex(privateSet[record], layers),
			ReduceToTreeIndex2(privateSet[record], layers),
		}

		for _, leaf := range candidates {
			if seen[leaf] {
				continue
			}
			seen[leaf] = true

			owner, taken := occupied[leaf]
			if !taken || assign(owner, seen) {
				occupied[leaf] = record
				placement[record] = leaf
				return true
			}
		}
		return false
	}

	for i := range privateSet {
		if !assign(i, make(map[uint64]bool)) {
			return nil, CuckooPlacementStats{Failures: 1}, fmt.Errorf(
				"cuckoo placement failed for record %d: both candidate leaves are saturated; increase tree expansion or add a stash",
				i,
			)
		}
	}

	return placement, CuckooPlacementStats{}, nil
}

// GetPublicParameters extracts the public parameters from the server context.
// These parameters need to be shared with the client for encryption.
//
// Parameters:
//   - ctx: Server initialization context from ServerInitialize
//
// Returns:
//   - *matrix.Vector: Public parameter matrix (PP)
//   - *ring.Poly: Message polynomial
//   - *LE.LE: Lattice encryption parameters
//
// Example:
//
//	pp, msg, le := psi.GetPublicParameters(ctx)
//	// Send pp, msg, le to client for encryption
func GetPublicParameters(ctx *ServerInitContext) (*matrix.Vector, *ring.Poly, *LE.LE) {
	return ctx.PublicParams, ctx.Message, ctx.LEParams
}

// SerializableParams represents PSI public parameters in a JSON-serializable format.
// Use SerializeParameters to create and DeserializeParameters to reconstruct.
type SerializableParams struct {
	PP     [][]uint64   `json:"pp"`
	Msg    []uint64     `json:"msg"`
	Q      uint64       `json:"q"`
	D      int          `json:"d"`
	N      int          `json:"n"`
	Layers int          `json:"layers"`
	M      int          `json:"m"`
	M2     int          `json:"m2"`
	A0NTT  [][][]uint64 `json:"a0ntt"`
	A1NTT  [][][]uint64 `json:"a1ntt"`
	BNTT   [][][]uint64 `json:"bntt"`
	GNTT   [][][]uint64 `json:"gntt"`
}

// SerializeParameters converts public parameters into a serializable format for network transmission.
// Use this to send parameters from server to client over network or file storage.
//
// Parameters:
//   - pp: Public parameter matrix vector
//   - msg: Message polynomial
//   - le: Lattice encryption parameters
//
// Returns:
//   - *SerializableParams: Serialized parameters ready for JSON/network transmission
//
// Example:
//
//	params := psi.SerializeParameters(pp, msg, le)
//	// Send params over network or save to file
func SerializeParameters(pp *matrix.Vector, msg *ring.Poly, le *LE.LE) *SerializableParams {
	ppCoeffs := make([][]uint64, len(pp.Elements))
	for i, poly := range pp.Elements {
		ppCoeffs[i] = poly.Coeffs[0]
	}

	msgCoeffs := msg.Coeffs[0]

	serializeMatrix := func(mat *matrix.Matrix) [][][]uint64 {
		if mat == nil || mat.Elements == nil {
			return nil
		}
		result := make([][][]uint64, len(mat.Elements))
		for i, row := range mat.Elements {
			if row != nil {
				result[i] = make([][]uint64, len(row))
				for j, poly := range row {
					if poly != nil && poly.Coeffs != nil && len(poly.Coeffs) > 0 {
						result[i][j] = poly.Coeffs[0]
					}
				}
			}
		}
		return result
	}

	return &SerializableParams{
		PP:     ppCoeffs,
		Msg:    msgCoeffs,
		Q:      le.Q,
		D:      le.D,
		N:      le.N,
		Layers: le.Layers,
		M:      le.M,
		M2:     le.M2,
		A0NTT:  serializeMatrix(le.A0NTT),
		A1NTT:  serializeMatrix(le.A1NTT),
		BNTT:   serializeMatrix(le.BNTT),
		GNTT:   serializeMatrix(le.GNTT),
	}
}

// DeserializeParameters reconstructs public parameters from serialized format.
// Use this on the client side to receive parameters from the server.
//
// Parameters:
//   - params: Serialized parameters from SerializeParameters
//
// Returns:
//   - *matrix.Vector: Reconstructed public parameter matrix
//   - *ring.Poly: Reconstructed message polynomial
//   - *LE.LE: Reconstructed lattice encryption parameters
//   - error: Returns error if deserialization fails
//
// Example:
//
//	pp, msg, le, err := psi.DeserializeParameters(receivedParams)
//	if err != nil {
//	    log.Fatal(err)
//	}
func DeserializeParameters(params *SerializableParams) (*matrix.Vector, *ring.Poly, *LE.LE, error) {
	r, err := ring.NewRing(params.D, []uint64{params.Q})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create ring: %w", err)
	}

	ppVec := &matrix.Vector{Elements: make([]*ring.Poly, len(params.PP))}
	for i, coeffs := range params.PP {
		poly := r.NewPoly()
		copy(poly.Coeffs[0], coeffs)
		ppVec.Elements[i] = poly
	}

	msgPoly := r.NewPoly()
	copy(msgPoly.Coeffs[0], params.Msg)

	deserializeMatrix := func(serialized [][][]uint64) *matrix.Matrix {
		if len(serialized) == 0 {
			return nil
		}
		n := len(serialized)
		m := 0
		if len(serialized[0]) > 0 {
			m = len(serialized[0])
		}
		mat := matrix.NewMatrix(n, m, r)
		for i := 0; i < n; i++ {
			for j := 0; j < m && j < len(serialized[i]); j++ {
				if len(serialized[i][j]) > 0 {
					copy(mat.Elements[i][j].Coeffs[0], serialized[i][j])
				}
			}
		}
		return mat
	}

	le := &LE.LE{
		Q:      params.Q,
		D:      params.D,
		N:      params.N,
		Layers: params.Layers,
		M:      params.M,
		M2:     params.M2,
		R:      r,
		A0NTT:  deserializeMatrix(params.A0NTT),
		A1NTT:  deserializeMatrix(params.A1NTT),
		BNTT:   deserializeMatrix(params.BNTT),
		GNTT:   deserializeMatrix(params.GNTT),
	}

	return ppVec, msgPoly, le, nil
}

// ServerInitialize prepares the server-side PSI context with the server's private dataset.
// This function must be called before performing any intersection operations.
//
// Parameters:
//   - private_set_X: Server's private dataset (slice of uint64 values)
//   - Treepath: Path to the database file for storing the witness tree structure
//
// Returns:
//   - *ServerInitContext: Initialized server context containing:
//   - Lattice encryption parameters (LE)
//   - Public parameters (PP)
//   - Message polynomial (Msg)
//   - Witness tree for efficient lookup
//   - error: Returns error if parameter setup fails or tree creation fails
//
// Example:
//
//	serverData := []uint64{100, 200, 300, 400}
//	ctx, err := psi.ServerInitialize(serverData, "./data/tree.db")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer ctx.Cleanup()
func ServerInitialize(private_set_X []uint64, Treepath string) (*ServerInitContext, error) {
	return serverInitialize(private_set_X, Treepath, true)
}

func ServerInitializeChunked(private_set_X []uint64, Treepath string) (*ServerInitContext, error) {
	return serverInitialize(private_set_X, Treepath, false)
}

func serverInitialize(private_set_X []uint64, Treepath string, precomputeWitnesses bool) (*ServerInitContext, error) {
	monitor := NewPerformanceMonitor()

	X_size := len(private_set_X)
	if X_size == 0 {
		return nil, errors.New("server set is empty")
	}

	leParams, err := SetupLEParameters(len(private_set_X))
	if err != nil {
		return nil, fmt.Errorf("SetupLEParameters: %w", err)
	}

	db, err := sql.Open("sqlite3", Treepath)
	if err != nil {
		return nil, fmt.Errorf("open tree db: %w", err)
	}
	defer db.Close()

	if err := configureTreeBuildDB(db); err != nil {
		return nil, fmt.Errorf("configure tree db: %w", err)
	}

	if err := storage.InitializeTreeDB(db, leParams.Layers); err != nil {
		log.Printf("warning: InitializeTreeDB returned: %v\n", err)
	}

	publicKeys := make([]*matrix.Vector, X_size)
	privateKeys := make([]*matrix.Vector, X_size)
	hashedClient := make([]uint64, X_size)
	keyGenStart := time.Now()

	numWorkers := CalculateOptimalWorkers(X_size)
	if numWorkers > X_size {
		numWorkers = X_size
	}

	workChan := make(chan int, X_size)
	var wg sync.WaitGroup

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range workChan {
				publicKeys[i], privateKeys[i] = leParams.KeyGen()
			}
		}()
	}

	for i := 0; i < X_size; i++ {
		workChan <- i
	}
	close(workChan)
	wg.Wait()
	monitor.TrackKeyGeneration(keyGenStart)

	placement, placementStats, err := placeCuckooLeaves(private_set_X, leParams.Layers)
	if err != nil {
		return nil, err
	}
	copy(hashedClient, placement)

	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin tree build transaction: %w", err)
	}
	treeBuildTxOpen := true
	defer func() {
		if treeBuildTxOpen {
			_ = tx.Rollback()
		}
	}()

	for i := 0; i < X_size; i++ {
		LE.Upd(tx, hashedClient[i], leParams.Layers, publicKeys[i], leParams)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tree build transaction: %w", err)
	}
	treeBuildTxOpen = false

	pp := LE.ReadFromDB(db, 0, 0, leParams).NTT(leParams.R)
	msg := matrix.NewRandomPolyBinary(leParams.R)

	fmt.Println("       💾 Loading Merkle Tree into RAM for fast witness generation...")
	loadStart := time.Now()
	memoryTree, err := LE.LoadTreeFromDB(db, leParams.Layers, leParams)
	if err != nil {
		return nil, fmt.Errorf("failed to load tree into memory: %w", err)
	}
	fmt.Printf("       ✓ Tree loaded in %v\n", time.Since(loadStart))

	var witnessesVec1 [][]*matrix.Vector
	var witnessesVec2 [][]*matrix.Vector
	if precomputeWitnesses {
		witnessStart := time.Now()
		witnessesVec1 = make([][]*matrix.Vector, X_size)
		witnessesVec2 = make([][]*matrix.Vector, X_size)

		witnessChan := make(chan int, X_size)
		var witnessWg sync.WaitGroup

		numWorkers = CalculateOptimalWorkers(X_size)
		for w := 0; w < numWorkers; w++ {
			witnessWg.Add(1)
			go func() {
				defer witnessWg.Done()
				for i := range witnessChan {
					witnessesVec1[i], witnessesVec2[i] = LE.WitGenMemory(memoryTree, leParams, hashedClient[i])
				}
			}()
		}

		for i := 0; i < X_size; i++ {
			witnessChan <- i
		}
		close(witnessChan)
		witnessWg.Wait()
		monitor.TrackWitnessGeneration(witnessStart)
	} else {
		log.Printf("       Chunked mode: witness generation deferred to active chunks")
	}

	monitor.PrintReport()

	ctx := &ServerInitContext{
		PublicParams:    pp,
		Message:         msg,
		LEParams:        leParams,
		PrivateKeys:     privateKeys,
		WitnessVectors1: witnessesVec1,
		WitnessVectors2: witnessesVec2,
		MemoryTree:      memoryTree,
		TreeIndices:     hashedClient,
		OriginalHashes:  private_set_X,
		DBPath:          Treepath,
		CuckooRebuilds:  placementStats.Rebuilds,
	}

	return ctx, nil
}

// DetectIntersectionWithContext computes the intersection between server and client datasets.
// It decrypts the client's ciphertexts and identifies matching elements.
//
// Parameters:
//   - ctx: Server initialization context from ServerInitialize
//   - clientCiphertexts: Encrypted client dataset from ClientEncrypt
//
// Returns:
//   - []uint64: Intersection set (elements present in both datasets)
//   - error: Returns error if decryption or witness lookup fails
//
// Example:
//
//	intersection, err := psi.DetectIntersectionWithContext(ctx, ciphertexts)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Found %d common elements\n", len(intersection))
func DetectIntersectionWithContext(ctx *ServerInitContext, clientCiphertexts []Cxtx) ([]uint64, error) {
	runtime.GC()

	monitor := NewPerformanceMonitor()
	intersectionStart := time.Now()

	X_size := len(ctx.OriginalHashes)

	numWorkers := CalculateOptimalWorkers(X_size)
	if numWorkers < 1 {
		numWorkers = 1
	}

	// ── Leaf-indexed filtering ──────────────────────────────
	// Build a map from leaf index → list of ciphertext indices.
	// In DKLLMR23, decryption only succeeds when the ciphertext's target
	// leaf matches the server record's leaf. All other pairs produce
	// noise/garbage. This reduces Dec calls from O(m × 2n) to O(m + 2n).
	leafToCts := make(map[uint64][]int)
	for j, ct := range clientCiphertexts {
		leafToCts[ct.TargetLeaf] = append(leafToCts[ct.TargetLeaf], j)
	}

	totalChecks := 0
	for k := 0; k < X_size; k++ {
		totalChecks += len(leafToCts[ctx.TreeIndices[k]])
	}
	log.Printf("   Leaf-indexed filtering: %d server records × %d ciphertexts → %d targeted Dec calls (was %d all-pairs)",
		X_size, len(clientCiphertexts), totalChecks, X_size*len(clientCiphertexts))

	var Z []uint64
	intersectionMap := make(map[int]bool)
	var resultMutex sync.Mutex

	// Job granularity: one job = one server record (not one pair).
	// Each worker processes all matching ciphertexts for its assigned record.
	jobs := make(chan int, numWorkers*2)
	var detectionWg sync.WaitGroup

	var processedCount uint64
	doneChan := make(chan struct{})

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				current := atomic.LoadUint64(&processedCount)
				percent := float64(current) / float64(X_size) * 100
				log.Printf("   ... Progress: %d/%d server records (%.1f%%)", current, X_size, percent)
			case <-doneChan:
				return
			}
		}
	}()

	for w := 0; w < numWorkers; w++ {
		detectionWg.Add(1)
		go func() {
			defer detectionWg.Done()
			defer func() {
				if r := recover(); r != nil {
					log.Printf("CRITICAL: Worker panic: %v", r)
				}
			}()

			for k := range jobs {
				serverLeaf := ctx.TreeIndices[k]
				ctIndices := leafToCts[serverLeaf]

				// Skip server records whose leaf has no targeting ciphertexts
				if len(ctIndices) == 0 {
					atomic.AddUint64(&processedCount, 1)
					continue
				}

				for _, j := range ctIndices {
					msg2 := LE.Dec(ctx.LEParams, ctx.PrivateKeys[k], ctx.WitnessVectors1[k], ctx.WitnessVectors2[k],
						clientCiphertexts[j].C0, clientCiphertexts[j].C1, clientCiphertexts[j].C, clientCiphertexts[j].D)

					if CorrectnessCheck(msg2, ctx.Message, ctx.LEParams) {
						resultMutex.Lock()
						if !intersectionMap[k] {
							Z = append(Z, ctx.OriginalHashes[k])
							intersectionMap[k] = true
						}
						resultMutex.Unlock()
					}
				}
				atomic.AddUint64(&processedCount, 1)
			}
		}()
	}

	for k := 0; k < X_size; k++ {
		jobs <- k
	}
	close(jobs)
	detectionWg.Wait()
	close(doneChan)

	monitor.TrackIntersectionDetection(intersectionStart)

	monitor.TotalOperations = totalChecks
	monitor.PrintReport()

	return Z, nil
}

func DetectIntersectionChunkedWithContext(ctx *ServerInitContext, clientCiphertexts []Cxtx, opts ChunkedDetectionOptions) ([]uint64, ChunkedDetectionStats, error) {
	runtime.GC()

	if ctx.MemoryTree == nil {
		return nil, ChunkedDetectionStats{}, errors.New("chunked detection requires ServerInitializeChunked or a context with MemoryTree")
	}

	X_size := len(ctx.OriginalHashes)
	if X_size == 0 {
		return nil, ChunkedDetectionStats{}, errors.New("server context is empty")
	}

	workerCount := opts.WorkerCount
	if workerCount <= 0 {
		workerCount = CalculateOptimalWorkers(X_size)
	}
	if workerCount < 1 {
		workerCount = 1
	}
	if workerCount > X_size {
		workerCount = X_size
	}

	chunkSize := opts.ChunkSize
	if chunkSize <= 0 {
		chunkSize = workerCount * 16
	}
	if chunkSize < workerCount {
		chunkSize = workerCount
	}
	if chunkSize > X_size {
		chunkSize = X_size
	}

	leafToCts := make(map[uint64][]int)
	for j, ct := range clientCiphertexts {
		leafToCts[ct.TargetLeaf] = append(leafToCts[ct.TargetLeaf], j)
	}

	totalChecks := 0
	for k := 0; k < X_size; k++ {
		totalChecks += len(leafToCts[ctx.TreeIndices[k]])
	}
	log.Printf("   Chunked leaf-indexed filtering: m=%d, ciphertexts=%d, chunk_size=%d, workers=%d, targeted_dec_calls=%d (all_pairs=%d)",
		X_size, len(clientCiphertexts), chunkSize, workerCount, totalChecks, X_size*len(clientCiphertexts))

	stats := ChunkedDetectionStats{
		Mode:                 "chunked",
		ChunkSize:            chunkSize,
		WorkerCount:          workerCount,
		LeafIndexedFiltering: true,
		TargetedDecCalls:     totalChecks,
		AllPairsDecCalls:     X_size * len(clientCiphertexts),
	}

	var Z []uint64
	intersectionMap := make(map[int]bool)
	var resultMutex sync.Mutex
	intersectionStart := time.Now()

	for start := 0; start < X_size; start += chunkSize {
		end := start + chunkSize
		if end > X_size {
			end = X_size
		}
		stats.ChunksProcessed++
		log.Printf("   ... Chunk %d: records [%d,%d)", stats.ChunksProcessed, start, end)

		jobs := make(chan int, workerCount*2)
		var wg sync.WaitGroup

		for w := 0; w < workerCount; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						log.Printf("CRITICAL: chunked worker panic: %v", r)
					}
				}()

				for k := range jobs {
					ctIndices := leafToCts[ctx.TreeIndices[k]]
					if len(ctIndices) == 0 {
						continue
					}

					witness1, witness2 := LE.WitGenMemory(ctx.MemoryTree, ctx.LEParams, ctx.TreeIndices[k])
					for _, j := range ctIndices {
						msg2 := LE.Dec(ctx.LEParams, ctx.PrivateKeys[k], witness1, witness2,
							clientCiphertexts[j].C0, clientCiphertexts[j].C1, clientCiphertexts[j].C, clientCiphertexts[j].D)

						if CorrectnessCheck(msg2, ctx.Message, ctx.LEParams) {
							resultMutex.Lock()
							if !intersectionMap[k] {
								Z = append(Z, ctx.OriginalHashes[k])
								intersectionMap[k] = true
							}
							resultMutex.Unlock()
						}
					}
				}
			}()
		}

		for k := start; k < end; k++ {
			jobs <- k
		}
		close(jobs)
		wg.Wait()

		if opts.ForceGC {
			runtime.GC()
		}
	}

	log.Printf("   Chunked intersection complete in %v, matches=%d", time.Since(intersectionStart), len(Z))
	return Z, stats, nil
}

func Server(private_set_X []uint64, Treepath string) ([]uint64, error) {
	ctx, err := ServerInitialize(private_set_X, Treepath)
	if err != nil {
		return nil, err
	}

	ciphertexts := Client(ctx.TreeIndices, ctx.PublicParams, ctx.Message, ctx.LEParams)

	Z, err := DetectIntersectionWithContext(ctx, ciphertexts)
	if err != nil {
		return nil, err
	}

	return Z, nil
}
