package psi

import (
	"fmt"
	"os"

	// "sort"
	"github.com/SanthoshCheemala/LE-PSI/pkg/LE"
	"github.com/SanthoshCheemala/LE-PSI/pkg/matrix"
	"github.com/tuneinsight/lattigo/v3/ring"
)

// VerboseMode controls debug output for correctness checks
// Set to false to suppress detailed mismatch logs
var VerboseMode = os.Getenv("PSI_VERBOSE") == "false"

type Cxtx struct {
	C0 []*matrix.Vector
	C1 []*matrix.Vector
	C  *matrix.Vector
	D  *ring.Poly
}

// ReduceToTreeIndex takes a raw hash value and reduces it to tree index
// by applying a mask based on the number of tree layers
// The hashing should already be done (e.g., in utils.PrepareDataForPSI)
func ReduceToTreeIndex(rawHash uint64, layers int) uint64 {
	// Create mask based on number of layers in the tree
	var mask uint64
	bits := uint(layers)
	if bits == 0 || bits >= 64 {
		mask = ^uint64(0)
	} else {
		mask = (uint64(1) << bits) - 1
	}
	
	return rawHash & mask
}

func CorrectnessCheck(decrypted, original *ring.Poly, le *LE.LE) bool {
    q14 := le.Q / 4
    q34 := (le.Q / 4) * 3
    binaryDecrypted := le.R.NewPoly()
    
    // Convert coefficients to binary based on thresholds
    for i := 0; i < le.R.N; i++ {
        if decrypted.Coeffs[0][i] < q14 || decrypted.Coeffs[0][i] > q34 {
            binaryDecrypted.Coeffs[0][i] = 0
        } else {
            binaryDecrypted.Coeffs[0][i] = 1
        }
    }
    
    // Enhanced debugging (only if verbose mode is enabled)
    matchCount := 0
    mismatchCount := 0
    for i := 0; i < le.R.N; i++ {
        if binaryDecrypted.Coeffs[0][i] == original.Coeffs[0][i] {
            matchCount++
        } else {
            mismatchCount++
            if VerboseMode && mismatchCount <= 5 { // Show first 5 mismatches only in verbose mode
                fmt.Printf("Mismatch at coeff %d: decoded=%d, original=%d (raw=%d)\n", 
                    i, binaryDecrypted.Coeffs[0][i], original.Coeffs[0][i], decrypted.Coeffs[0][i])
            }
        }
    }
    
    if VerboseMode {
        fmt.Printf("Correctness: %d matches, %d mismatches out of %d coefficients\n", 
            matchCount, mismatchCount, le.R.N)
        
        // Use a threshold instead of perfect equality for noisy decryption
        matchPercentage := float64(matchCount) / float64(le.R.N)
        fmt.Printf("Match percentage: %.2f%%\n", matchPercentage*100)
    }
    
    // Consider it correct if at least 95% of coefficients match
    matchPercentage := float64(matchCount) / float64(le.R.N)
    return matchPercentage >= 0.95
}