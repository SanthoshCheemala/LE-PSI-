package psi

import (
	"runtime"
	"sync"

	"github.com/SanthoshCheemala/FLARE/pkg/LE"
	"github.com/SanthoshCheemala/FLARE/pkg/matrix"
	"github.com/tuneinsight/lattigo/v3/ring"
	"github.com/tuneinsight/lattigo/v3/utils"
)

func ClientEncrypt(private_set_Y []uint64, pp *matrix.Vector, msg *ring.Poly, le *LE.LE) []Cxtx {
	return Client(private_set_Y, pp, msg, le)
}

func Client(private_set_Y []uint64, pp *matrix.Vector, msg *ring.Poly, le *LE.LE) []Cxtx {
	Y_size := len(private_set_Y)

	treeIndices := make([]uint64, Y_size)
	for i := 0; i < Y_size; i++ {
		treeIndices[i] = ReduceToTreeIndex(private_set_Y[i], le.Layers)
	}

	C := make([]Cxtx, Y_size)
	cipherChan := make(chan int, Y_size)
	var cipherWg sync.WaitGroup

	numWorkers := runtime.NumCPU()
	if numWorkers > Y_size {
		numWorkers = Y_size
	}

	for w := 0; w < numWorkers; w++ {
		cipherWg.Add(1)
		go func() {
			defer cipherWg.Done()
			
			workerPRNG, _ := utils.NewPRNG()
			workerGaussianSampler := ring.NewGaussianSampler(workerPRNG, le.R, le.Sigma, le.Bound)
			
			for i := range cipherChan {
				r := make([]*matrix.Vector, le.Layers+1)
				for j := 0; j < le.Layers+1; j++ {
					r[j] = matrix.NewRandomVec(le.N, le.R, workerPRNG).NTT(le.R)
				}

				e := workerGaussianSampler.ReadNew()
				e0 := make([]*matrix.Vector, le.Layers+1)
				e1 := make([]*matrix.Vector, le.Layers+1)
				for j := 0; j < le.Layers+1; j++ {
					if j == le.Layers {
						e0[j] = matrix.NewNoiseVec(le.M2, le.R, workerPRNG, le.Sigma, le.Bound).NTT(le.R)
					} else {
						e0[j] = matrix.NewNoiseVec(le.M, le.R, workerPRNG, le.Sigma, le.Bound).NTT(le.R)
					}
					e1[j] = matrix.NewNoiseVec(le.M, le.R, workerPRNG, le.Sigma, le.Bound).NTT(le.R)
				}

				c0, c1, cvec, dpoly := LE.Enc(le, pp, treeIndices[i], msg, r, e0, e1, e)
				C[i] = Cxtx{C0: c0, C1: c1, C: cvec, D: dpoly}
			}
		}()
	}

	for i := 0; i < Y_size; i++ {
		cipherChan <- i
	}
	close(cipherChan)
	cipherWg.Wait()

	return C
}
