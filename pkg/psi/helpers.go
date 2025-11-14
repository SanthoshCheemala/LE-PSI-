package psi

import (
	"fmt"
	"os"

	// "sort"
	"github.com/SanthoshCheemala/LE-PSI/pkg/LE"
	"github.com/SanthoshCheemala/LE-PSI/pkg/matrix"
	"github.com/tuneinsight/lattigo/v3/ring"
)

// VerboseMode controls detailed logging output. Set PSI_VERBOSE=false to suppress.
var VerboseMode = os.Getenv("PSI_VERBOSE") == "false"

type Cxtx struct {
	C0 []*matrix.Vector
	C1 []*matrix.Vector
	C  *matrix.Vector
	D  *ring.Poly
}

// ReduceToTreeIndex reduces a hash value to a tree index based on the number of layers.
func ReduceToTreeIndex(rawHash uint64, layers int) uint64 {
	var mask uint64
	bits := uint(layers)
	if bits == 0 || bits >= 64 {
		mask = ^uint64(0)
	} else {
		mask = (uint64(1) << bits) - 1
	}
	return rawHash & mask
}

// CorrectnessCheck verifies decryption correctness using threshold-based matching.
// Returns true if at least 95% of coefficients match.
func CorrectnessCheck(decrypted, original *ring.Poly, le *LE.LE) bool {
	q14 := le.Q / 4
	q34 := (le.Q / 4) * 3
	binaryDecrypted := le.R.NewPoly()
	
	for i := 0; i < le.R.N; i++ {
		if decrypted.Coeffs[0][i] < q14 || decrypted.Coeffs[0][i] > q34 {
			binaryDecrypted.Coeffs[0][i] = 0
		} else {
			binaryDecrypted.Coeffs[0][i] = 1
		}
	}
	
	matchCount := 0
	for i := 0; i < le.R.N; i++ {
		if binaryDecrypted.Coeffs[0][i] == original.Coeffs[0][i] {
			matchCount++
		}
	}
	
	if VerboseMode {
		matchPercentage := float64(matchCount) / float64(le.R.N)
		fmt.Printf("Match rate: %.2f%% (%d/%d coefficients)\n", matchPercentage*100, matchCount, le.R.N)
	}
	
	return float64(matchCount)/float64(le.R.N) >= 0.95
}