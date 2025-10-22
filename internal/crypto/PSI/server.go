package psi

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/SanthoshCheemala/FLARE/internal/storage"
	"github.com/SanthoshCheemala/FLARE/pkg/LE"
	"github.com/SanthoshCheemala/FLARE/pkg/matrix"
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
	numWorkers := runtime.NumCPU()
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
	numWorkers := runtime.NumCPU()
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
