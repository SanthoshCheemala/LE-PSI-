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
	TreeIndices     []uint64
	OriginalHashes  []uint64
	DBPath          string
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
//     - Lattice encryption parameters (LE)
//     - Public parameters (PP)
//     - Message polynomial (Msg)
//     - Witness tree for efficient lookup
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
				hashedClient[i] = ReduceToTreeIndex(private_set_X[i], leParams.Layers)
			}
		}()
	}

	for i := 0; i < X_size; i++ {
		workChan <- i
	}
	close(workChan)
	wg.Wait()
	monitor.TrackKeyGeneration(keyGenStart)

	for i := 0; i < X_size; i++ {
		LE.Upd(db, hashedClient[i], leParams.Layers, publicKeys[i], leParams)
	}

	pp := LE.ReadFromDB(db, 0, 0, leParams).NTT(leParams.R)
	msg := matrix.NewRandomPolyBinary(leParams.R)

	fmt.Println("       💾 Loading Merkle Tree into RAM for fast witness generation...")
	loadStart := time.Now()
	memoryTree, err := LE.LoadTreeFromDB(db, leParams.Layers, leParams)
	if err != nil {
		return nil, fmt.Errorf("failed to load tree into memory: %w", err)
	}
	fmt.Printf("       ✓ Tree loaded in %v\n", time.Since(loadStart))

	witnessStart := time.Now()
	witnessesVec1 := make([][]*matrix.Vector, X_size)
	witnessesVec2 := make([][]*matrix.Vector, X_size)
	
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

	monitor.PrintReport()

	ctx := &ServerInitContext{
		PublicParams:    pp,
		Message:         msg,
		LEParams:        leParams,
		PrivateKeys:     privateKeys,
		WitnessVectors1: witnessesVec1,
		WitnessVectors2: witnessesVec2,
		TreeIndices:     hashedClient,
		OriginalHashes:  private_set_X,
		DBPath:          Treepath,
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

	var Z []uint64
	intersectionMap := make(map[int]bool)
	var resultMutex sync.Mutex

	type workItem struct {
		j, k int
	}
	totalWork := len(clientCiphertexts) * X_size
	workItems := make(chan workItem, totalWork)
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
				percent := float64(current) / float64(totalWork) * 100
				log.Printf("   ... Progress: %d/%d (%.1f%%)", current, totalWork, percent)
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
			
			for item := range workItems {
				j, k := item.j, item.k
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
				atomic.AddUint64(&processedCount, 1)
			}
		}()
	}

	for j := range clientCiphertexts {
		for k := 0; k < X_size; k++ {
			workItems <- workItem{j: j, k: k}
		}
	}
	close(workItems)
	detectionWg.Wait()
	close(doneChan)
	
	monitor.TrackIntersectionDetection(intersectionStart)

	monitor.TotalOperations = totalWork
	monitor.PrintReport()

	return Z, nil
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
