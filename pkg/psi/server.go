package psi

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"github.com/SanthoshCheemala/LE-PSI/internal/storage"
	"github.com/SanthoshCheemala/LE-PSI/pkg/LE"
	"github.com/SanthoshCheemala/LE-PSI/pkg/matrix"
	_ "github.com/mattn/go-sqlite3"
	"github.com/tuneinsight/lattigo/v3/ring"
)

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

func GetPublicParameters(ctx *ServerInitContext) (*matrix.Vector, *ring.Poly, *LE.LE) {
	return ctx.PublicParams, ctx.Message, ctx.LEParams
}

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

// calculateOptimalWorkers returns the optimal number of worker threads based on dataset size.
// This balances CPU utilization with memory constraints to prevent swapping.
//
// Algorithm:
//  1. Memory constraint: Available RAM / (dataset_size × memory_per_record × safety_margin)
//  2. Cache constraint: sqrt(dataset_size) capped at hardware limit (48 physical cores)
//  3. Hardware constraint: 48 physical cores (Intel Xeon Gold 5418Y × 2 sockets)
//  4. Returns minimum of all constraints, with practical minimum of 4 workers
//
// Examples:
//   100 records  → 32 workers (cache optimal)
//   500 records  → 22 workers (memory begins to constrain)
//   1000 records → 16 workers (balanced)
//   2000 records → 12 workers (memory constrained)
//   4000 records → 8 workers (heavily memory constrained)
func calculateOptimalWorkers(datasetSize int) int {
	// System constraints for dual-socket Intel Xeon Gold 5418Y
	const (
		availableRAM_GB  = 117.0 // Available RAM (251 GB total - 134 GB used)
		memPerRecord_GB  = 0.035 // 35 MB per record (12 MB witness + 13 MB thread + 10 MB overhead)
		safetyMargin     = 1.15  // 15% safety margin (reduced from 20% - more aggressive)
		hardwareLimit    = 48    // Physical cores (24 per socket × 2 sockets)
		practicalMinimum = 8     // Increased from 4 - better for multi-socket systems
	)

	// Calculate memory-constrained limit (TUNED: More aggressive)
	// Formula: available_memory / (records × memory_per_record × safety_margin)
	// More records → less available memory per worker → fewer workers
	estimatedMemory := float64(datasetSize) * memPerRecord_GB * safetyMargin
	memoryLimit := hardwareLimit // Default to hardware limit
	if estimatedMemory > availableRAM_GB*0.6 {
		// Changed from 0.5 to 0.6 - allow using more RAM before scaling down
		// Changed from 0.8 to 0.85 - utilize more available RAM
		memoryLimit = int((availableRAM_GB * 0.85) / estimatedMemory * float64(hardwareLimit))
	}

	// Calculate cache efficiency limit (TUNED: More aggressive)
	// Use 1.5×sqrt for better parallelism while maintaining cache efficiency
	// L3 cache: 90 MB total, L2: 96 MB - can handle more parallel workers
	cacheLimit := hardwareLimit
	if datasetSize > 100 {
		// Scale up by 1.5× for better CPU utilization
		cacheLimit = int(1.5 * math.Sqrt(float64(datasetSize)))
		if cacheLimit > hardwareLimit {
			cacheLimit = hardwareLimit
		}
		if cacheLimit < 16 {
			cacheLimit = 16 // Increased from 8 - better for dual-socket NUMA
		}
	}

	// Take the minimum of all constraints
	optimal := memoryLimit
	if cacheLimit < optimal {
		optimal = cacheLimit
	}
	if hardwareLimit < optimal {
		optimal = hardwareLimit
	}

	// Ensure practical minimum for performance
	if optimal < practicalMinimum {
		optimal = practicalMinimum
	}

	// Log the decision for monitoring and debugging
	estimatedRAM_GB := float64(datasetSize) * memPerRecord_GB
	log.Printf("🚀 Adaptive Threading (TUNED): %d records → %d workers (est. RAM: %.1f GB, memory limit: %d, cache limit: %d)",
		datasetSize, optimal, estimatedRAM_GB, memoryLimit, cacheLimit)

	return optimal
}

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

	numWorkers := calculateOptimalWorkers(X_size)
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

	witnessStart := time.Now()
	witnessesVec1 := make([][]*matrix.Vector, X_size)
	witnessesVec2 := make([][]*matrix.Vector, X_size)
	
	witnessChan := make(chan int, X_size)
	var witnessWg sync.WaitGroup

	for w := 0; w < numWorkers; w++ {
		witnessWg.Add(1)
		go func() {
			defer witnessWg.Done()
			for i := range witnessChan {
				witnessesVec1[i], witnessesVec2[i] = LE.WitGen(db, leParams, hashedClient[i])
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

	// Return server context with all initialization data
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

func DetectIntersectionWithContext(ctx *ServerInitContext, clientCiphertexts []Cxtx) ([]uint64, error) {
	monitor := NewPerformanceMonitor()
	intersectionStart := time.Now()

	X_size := len(ctx.OriginalHashes)

	numWorkers := calculateOptimalWorkers(X_size)
	if numWorkers > X_size {
		numWorkers = X_size
	}

	var Z []uint64
	intersectionMap := make(map[int]bool)
	var resultMutex sync.Mutex

	type workItem struct {
		j, k int
	}
	workItems := make(chan workItem, len(clientCiphertexts)*X_size)
	var detectionWg sync.WaitGroup

	for w := 0; w < numWorkers; w++ {
		detectionWg.Add(1)
		go func() {
			defer detectionWg.Done()
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
	monitor.TrackIntersectionDetection(intersectionStart)

	monitor.TotalOperations = len(clientCiphertexts) * X_size
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
