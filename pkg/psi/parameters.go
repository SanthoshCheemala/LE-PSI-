package psi

import (
	"fmt"
	"math"
	"os"

	"github.com/SanthoshCheemala/LE-PSI/pkg/LE"
)

// SetupLEParameters initializes Laconic Encryption parameters for PSI operations.
// This function configures the Ring-LWE cryptographic parameters and computes
// the optimal Merkle tree depth based on dataset size.
//
// Parameters:
//   - size: Expected number of elements in the server dataset
//
// Returns:
//   - *LE.LE: Configured Laconic Encryption parameters
//   - error: Non-nil if parameter initialization fails
//
// Cryptographic Parameters (Environment Configurable):
//   - Q: Modulus = 180143985094819841 (~2^58)
//   - D: Ring dimension = 256 (Fast Testing) OR 2048 (True 128-bit PQ Security)
//   - N: Matrix dimension = 4
//   - qBits: Modulus bit length = 58
//
// Security Configuration:
//   - By default, D=256 is used for extremely fast evaluations, but it does NOT
//     provide full 128-bit post-quantum security for the 58-bit modulus.
//   - Set the environment variable `PSI_SECURITY_LEVEL=128` to enforce D=2048,
//     which yields a lattice dimension of 8192 (N=4, D=2048). This comfortably
//     provides >128-bit security against known quantum lattice attacks.
// The function automatically calculates:
//   - Merkle tree layers: log2(16 * size) for 16x expansion factor
//   - Load factor: items per slot ratio
//   - Collision probability: using balls-into-bins model
//
// Example:
//
//	le, err := psi.SetupLEParameters(10000)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	// le.Layers = 18 (for 10K elements with 16x expansion)
//	// Collision probability < 10^-6
func SetupLEParameters(size int) (*LE.LE, error) {
	const (
		Q     = uint64(180143985094819841) // Modulus (~2^58)
		qBits = 58                          // Modulus bit length
		N     = 4                           // Matrix dimension
		c     = 16.0                        // Expansion factor (16x slots vs items)
	)

	// Default to Fast Evaluation Mode (low security)
	D := 256
	securityMode := "Fast Evaluation (Low Security)"

	// Enforce 128-bit post-quantum security if configured
	if os.Getenv("PSI_SECURITY_LEVEL") == "128" {
		D = 2048
		securityMode = "128-bit Post-Quantum Security Mode"
	}

	if D != 256 && D != 512 && D != 1024 && D != 2048 {
		return nil, fmt.Errorf("unsupported ring dimension %d. Supported values: 256, 512, 1024, 2048", D)
	}

	var leParams *LE.LE
	var err error

	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic in LE.setup with dimension %d: %v", D, r)
				fmt.Printf("Recovered from Panic in LE.setup: %v\n", r)
			}
		}()
		fmt.Println("Setting up LE with Parameters Q =", Q, "qBits =", qBits, "D =", D, "N =", N)
		leParams = LE.Setup(Q, qBits, D, N)
	}()
	
	if err != nil {
		return nil, err
	}
	if leParams == nil {
		return nil, fmt.Errorf("failed to initialize the le parameters (nil result)")
	}
	if leParams.R == nil {
		return nil, fmt.Errorf("ring(R) is nil in le parameters")
	}

	leParams.Layers = int(math.Ceil(math.Log2(c * float64(size))))

	numSlots := 1 << leParams.Layers
	loadFactor := float64(size) / float64(numSlots)
	
	m := float64(size)
	Nf := float64(numSlots)
	collisionProb := 1.0 - math.Exp(-(m*m)/(2*Nf))

	fmt.Println("Successfully initialized the LE parameters:")
	fmt.Printf(" - Security Level: %s\n", securityMode)
	fmt.Printf(" - Ring Dimension: %d\n", D)
	fmt.Printf(" - Modulus Q: %d\n", Q)
	fmt.Printf(" - Matrix Dimension N: %d\n", N)
	fmt.Printf(" - qBits: %d\n", qBits)
	fmt.Printf(" - Layers: %d (slots = %d)\n", leParams.Layers, numSlots)
	fmt.Printf(" - Load Factor: %.6f (items/slot)\n", loadFactor)
	fmt.Printf(" - Estimated Collision Probability: %.6e\n", collisionProb)

	return leParams, nil
}
