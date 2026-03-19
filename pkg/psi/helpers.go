// Package psi implements Private Set Intersection (PSI) using Lattice Encryption.
//
// This package provides a complete implementation of privacy-preserving set intersection
// based on lattice-based cryptography. It allows two parties (server and client) to
// compute the intersection of their private datasets without revealing non-matching elements.
//
// Basic Usage:
//
//  1. Server Setup:
//     serverData := []uint64{100, 200, 300, 400}
//     ctx, err := psi.ServerInitialize(serverData, "./tree.db")
//     if err != nil {
//     log.Fatal(err)
//     }
//     defer ctx.Cleanup()
//
//  2. Share Public Parameters (server → client):
//     pp, msg, le := psi.GetPublicParameters(ctx)
//     // Transmit pp, msg, le to client
//
//  3. Client Encryption:
//     clientData := []uint64{150, 200, 250}
//     ciphertexts := psi.ClientEncrypt(clientData, pp, msg, le)
//     // Send ciphertexts to server
//
//  4. Server Computes Intersection:
//     intersection, err := psi.DetectIntersectionWithContext(ctx, ciphertexts)
//     // intersection = [200] (common element)
//
// Security:
//   - Based on Ring Learning With Errors (Ring-LWE) hardness assumption
//   - 128-bit security level with ring dimension D=256
//   - Server learns only the intersection (privacy-preserving)
//   - Client reveals nothing about non-matching elements
//
// Performance:
//   - Parallel processing with automatic worker optimization
//   - Witness tree data structure for O(log n) lookup
//   - Optimized for datasets ranging from 100 to 100,000+ elements
package psi

import (
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/SanthoshCheemala/LE-PSI/pkg/LE"
	"github.com/SanthoshCheemala/LE-PSI/pkg/matrix"
	"github.com/tuneinsight/lattigo/v3/ring"
)

// VerboseMode controls detailed logging output. Set PSI_VERBOSE=false to suppress.
var VerboseMode = os.Getenv("PSI_VERBOSE") == "false"

// Cxtx represents an encrypted ciphertext structure for PSI operations.
// It contains the components of a lattice-based encryption of a single data element.
//
// Fields:
//   - C0: Vector of encrypted path components (first part of dual ciphertext)
//   - C1: Vector of encrypted path components (second part of dual ciphertext)
//   - C: Compressed vector representation for efficient transmission
//   - D: Polynomial component for message encoding
//
// This structure is produced by ClientEncrypt and consumed by DetectIntersectionWithContext.
type Cxtx struct {
	C0 []*matrix.Vector
	C1 []*matrix.Vector
	C  *matrix.Vector
	D  *ring.Poly
}

// ReduceToTreeIndex reduces a hash value to a tree index based on the number of layers.
// This function maps raw hash values to valid tree indices for witness tree lookup.
//
// Parameters:
//   - rawHash: Raw 64-bit hash value of the data element
//   - layers: Number of layers in the witness tree
//
// Returns:
//   - uint64: Tree index (masked hash value) in range [0, 2^layers - 1]
//
// Example:
//
//	treeIdx := psi.ReduceToTreeIndex(12345678, 10)  // Returns index in [0, 1023]
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
// Returns true if at least 95% of coefficients match between decrypted and original.
//
// Parameters:
//   - decrypted: Polynomial resulting from decryption
//   - original: Original plaintext polynomial
//   - le: Lattice encryption parameters for modulus Q and ring R
//
// Returns:
//   - bool: true if match rate >= 95%, false otherwise
//
// Note: Enables verbose logging with PSI_VERBOSE=false environment variable
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

// CalculateOptimalWorkers determines the optimal number of worker goroutines
// based on dataset size and available system resources.
//
// All PSI operations (KeyGen, WitGen, Dec) are CPU-bound compute tasks with
// no I/O on the hot path. The Merkle tree is pre-loaded into RAM before
// parallel processing begins, so workers perform pure in-memory arithmetic.
//
// Parameters:
//   - datasetSize: Number of elements to process
//
// Returns:
//   - int: Optimal number of worker goroutines (up to runtime.NumCPU())
//
// The function considers:
//   - Available CPU cores (uses all detected cores for compute-bound work)
//   - Available RAM (85% of total system RAM)
//   - Memory per record (~35 MB empirically measured)
//   - Safety margin (15%) to prevent memory exhaustion
//
// Algorithm:
//  1. Detect system resources (CPU cores, available RAM)
//  2. Set hardware limit to all available CPU cores (work is compute-bound)
//  3. Calculate memory-based limit to avoid exceeding available RAM
//  4. Take the minimum of hardware and memory limits
//  5. Ensure practical minimum of 4 workers
//  6. Cap at dataset size (no point having more workers than records)
//
// Example:
//
//	workers := psi.CalculateOptimalWorkers(5000)  // Auto-detects system, returns optimal count
//	// On 96-core system: Returns 96 workers (all cores, compute-bound)
//	// On 8-core system: Returns 8 workers
func CalculateOptimalWorkers(datasetSize int) int {
	// Dynamic system detection
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	totalCores := runtime.NumCPU()
	totalRAM_GB := float64(memStats.Sys) / (1024 * 1024 * 1024)
	availableRAM_GB := totalRAM_GB * 0.85 // Use 85% of total RAM

	const (
		memPerRecord_GB  = 0.035 // 35 MB per record (empirically measured)
		safetyMargin     = 1.15  // 15% safety margin
		practicalMinimum = 4     // Minimum workers for parallelism
	)

	// All PSI operations are compute-bound (lattice arithmetic, NTT transforms).
	// Use all available CPU cores — there is no I/O during the parallel phase
	// since the Merkle tree is already loaded into memory.
	hardwareLimit := totalCores
	if hardwareLimit < practicalMinimum {
		hardwareLimit = practicalMinimum
	}

	// Memory-based throttling: if estimated memory exceeds 60% of available RAM,
	// scale down worker count proportionally to avoid OOM
	estimatedMemory := float64(datasetSize) * memPerRecord_GB * safetyMargin
	memoryLimit := hardwareLimit
	if estimatedMemory > availableRAM_GB*0.6 {
		memoryLimit = int((availableRAM_GB * 0.85) / estimatedMemory * float64(hardwareLimit))
	}

	optimal := memoryLimit
	if hardwareLimit < optimal {
		optimal = hardwareLimit
	}

	if optimal < practicalMinimum {
		optimal = practicalMinimum
	}

	if optimal > datasetSize {
		optimal = datasetSize
	}

	estimatedRAM_GB := float64(datasetSize) * memPerRecord_GB
	log.Printf("🚀 Auto-Tuned: %d cores detected, %.1f GB RAM available → %d workers for %d records (est. RAM: %.1f GB)",
		totalCores, availableRAM_GB, optimal, datasetSize, estimatedRAM_GB)

	return optimal
}