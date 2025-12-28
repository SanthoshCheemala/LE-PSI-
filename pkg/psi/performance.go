// Package psi provides performance monitoring and noise measurement utilities
// for Laconic Private Set Intersection operations.
package psi

import (
	"fmt"
	"runtime"
	"time"

	"github.com/tuneinsight/lattigo/v3/ring"
)

// PerformanceMonitor tracks timing and throughput metrics for PSI operations.
// It records execution time for each phase (key generation, hashing, witness
// generation, intersection detection) and calculates overall throughput.
//
// Fields:
//   - StartTime: Timestamp when monitoring began
//   - KeyGenTime: Duration of key generation phase
//   - HashingTime: Duration of data hashing phase
//   - WitnessTime: Duration of witness generation phase
//   - IntersectionTime: Duration of intersection detection phase
//   - TotalOperations: Number of operations performed
//   - NumWorkers: Number of CPU cores/workers used
//
// Example:
//
//	monitor := psi.NewPerformanceMonitor()
//	// ... perform PSI operations ...
//	monitor.PrintReport()
type PerformanceMonitor struct {
	StartTime        time.Time
	KeyGenTime       time.Duration
	HashingTime      time.Duration
	WitnessTime      time.Duration
	IntersectionTime time.Duration
	TotalOperations  int
	NumWorkers       int
}

// NewPerformanceMonitor creates a new performance monitor initialized with
// the current timestamp and auto-detected CPU core count.
//
// Returns:
//   - *PerformanceMonitor: Initialized monitor ready for tracking
//
// Example:
//
//	monitor := psi.NewPerformanceMonitor()
//	defer monitor.PrintReport()
func NewPerformanceMonitor() *PerformanceMonitor {
	return &PerformanceMonitor{
		StartTime:  time.Now(),
		NumWorkers: runtime.NumCPU(),
	}
}

// TrackKeyGeneration records the duration of the key generation phase.
//
// Parameters:
//   - start: Timestamp when key generation started
//
// Example:
//
//	start := time.Now()
//	// ... generate keys ...
//	monitor.TrackKeyGeneration(start)
func (pm *PerformanceMonitor) TrackKeyGeneration(start time.Time) {
	pm.KeyGenTime = time.Since(start)
}

// TrackHashing records the duration of the data hashing phase.
//
// Parameters:
//   - start: Timestamp when hashing started
func (pm *PerformanceMonitor) TrackHashing(start time.Time) {
	pm.HashingTime = time.Since(start)
}

// TrackWitnessGeneration records the duration of the witness generation phase.
//
// Parameters:
//   - start: Timestamp when witness generation started
func (pm *PerformanceMonitor) TrackWitnessGeneration(start time.Time) {
	pm.WitnessTime = time.Since(start)
}

// TrackIntersectionDetection records the duration of the intersection detection phase.
//
// Parameters:
//   - start: Timestamp when intersection detection started
func (pm *PerformanceMonitor) TrackIntersectionDetection(start time.Time) {
	pm.IntersectionTime = time.Since(start)
}

// PrintReport prints a comprehensive performance report to stdout.
// The report includes timing breakdowns, percentages, throughput, and worker count.
//
// Parameters:
//   - verbose: Optional boolean to control output (default: true)
//
// Example:
//
//	monitor.PrintReport()        // Verbose output
//	monitor.PrintReport(false)   // Silent mode
func (pm *PerformanceMonitor) PrintReport(verbose ...bool) {
	verboseMode := true
	if len(verbose) > 0 {
		verboseMode = verbose[0]
	}

	if !verboseMode {
		return
	}

	totalTime := time.Since(pm.StartTime)

	fmt.Println("\nLE-PSI Performance Report (Parallelized)")
	fmt.Println("==================================================")
	fmt.Printf("CPU Cores Used: %d\n", pm.NumWorkers)
	fmt.Printf("Total Execution Time: %v\n", totalTime)
	fmt.Printf("Key Generation Time: %v (%.1f%%)\n", pm.KeyGenTime, float64(pm.KeyGenTime)/float64(totalTime)*100)
	fmt.Printf("Hashing Time: %v (%.1f%%)\n", pm.HashingTime, float64(pm.HashingTime)/float64(totalTime)*100)
	fmt.Printf("Witness Generation Time: %v (%.1f%%)\n", pm.WitnessTime, float64(pm.WitnessTime)/float64(totalTime)*100)
	fmt.Printf("Intersection Detection Time: %v (%.1f%%)\n", pm.IntersectionTime, float64(pm.IntersectionTime)/float64(totalTime)*100)

	if pm.TotalOperations > 0 {
		throughput := float64(pm.TotalOperations) / totalTime.Seconds()
		fmt.Printf("Throughput: %.2f operations/second\n", throughput)
		fmt.Printf("Parallel Efficiency: %.1fx speedup potential\n", float64(pm.NumWorkers))
	}

	fmt.Println("==================================================")
}

// GetTotalTime returns the total execution time since monitor creation.
//
// Returns:
//   - time.Duration: Elapsed time since StartTime
func (pm *PerformanceMonitor) GetTotalTime() time.Duration {
	return time.Since(pm.StartTime)
}

// GetThroughput calculates operations per second based on total operations and elapsed time.
// Returns 0 if no time has elapsed or no operations have been recorded.
//
// Returns:
//   - float64: Operations per second (ops/sec)
func (pm *PerformanceMonitor) GetThroughput() float64 {
	totalTime := time.Since(pm.StartTime)
	if totalTime.Seconds() == 0 || pm.TotalOperations == 0 {
		return 0
	}
	return float64(pm.TotalOperations) / totalTime.Seconds()
}

// GetMetrics returns all performance metrics as a structured map suitable for JSON serialization.
// This is useful for sending metrics to frontends or logging systems.
//
// Returns:
//   - map[string]interface{}: Comprehensive metrics including:
//     - total_time_seconds, total_time_formatted
//     - key_gen_time_seconds, key_gen_time_formatted, key_gen_percent
//     - hashing_time_seconds, hashing_time_formatted, hashing_percent
//     - witness_time_seconds, witness_time_formatted, witness_percent
//     - intersection_time_seconds, intersection_time_formatted, intersection_percent
//     - num_workers, total_operations, throughput_ops_per_sec
//
// Example:
//
//	metrics := monitor.GetMetrics()
//	json.NewEncoder(w).Encode(metrics) // Send to HTTP response
func (pm *PerformanceMonitor) GetMetrics() map[string]interface{} {
	totalTime := time.Since(pm.StartTime)

	metrics := map[string]interface{}{
		"total_time_seconds":          totalTime.Seconds(),
		"total_time_formatted":        totalTime.String(),
		"key_gen_time_seconds":        pm.KeyGenTime.Seconds(),
		"key_gen_time_formatted":      pm.KeyGenTime.String(),
		"hashing_time_seconds":        pm.HashingTime.Seconds(),
		"hashing_time_formatted":      pm.HashingTime.String(),
		"witness_time_seconds":        pm.WitnessTime.Seconds(),
		"witness_time_formatted":      pm.WitnessTime.String(),
		"intersection_time_seconds":   pm.IntersectionTime.Seconds(),
		"intersection_time_formatted": pm.IntersectionTime.String(),
		"num_workers":                 pm.NumWorkers,
		"total_operations":            pm.TotalOperations,
		"throughput_ops_per_sec":      pm.GetThroughput(),
	}

	if totalTime.Seconds() > 0 {
		metrics["key_gen_percent"] = (pm.KeyGenTime.Seconds() / totalTime.Seconds()) * 100
		metrics["hashing_percent"] = (pm.HashingTime.Seconds() / totalTime.Seconds()) * 100
		metrics["witness_percent"] = (pm.WitnessTime.Seconds() / totalTime.Seconds()) * 100
		metrics["intersection_percent"] = (pm.IntersectionTime.Seconds() / totalTime.Seconds()) * 100
	}

	return metrics
}

// GetMemoryUsage returns current memory statistics from the Go runtime.
// Useful for monitoring resource consumption during PSI operations.
//
// Returns:
//   - map[string]interface{}: Memory metrics including:
//     - alloc_mb: Currently allocated heap memory in MB
//     - total_alloc_mb: Cumulative allocated memory in MB
//     - sys_mb: Total memory obtained from OS in MB
//     - num_gc: Number of completed GC cycles
//     - goroutines: Current number of goroutines
//
// Example:
//
//	memStats := monitor.GetMemoryUsage()
//	fmt.Printf("Memory: %.2f MB\n", memStats["alloc_mb"])
func (pm *PerformanceMonitor) GetMemoryUsage() map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return map[string]interface{}{
		"alloc_mb":       float64(m.Alloc) / 1024 / 1024,
		"total_alloc_mb": float64(m.TotalAlloc) / 1024 / 1024,
		"sys_mb":         float64(m.Sys) / 1024 / 1024,
		"num_gc":         m.NumGC,
		"goroutines":     runtime.NumGoroutine(),
	}
}

// MeasureNoiseLevel calculates the noise level between an original message and its decrypted version.
// This is useful for analyzing the quality of lattice-based encryption/decryption.
//
// Parameters:
//   - r: Ring structure for polynomial operations
//   - original: Original plaintext polynomial
//   - decrypted: Decrypted polynomial (may contain noise)
//   - Q: Modulus value
//
// Returns:
//   - maxNoiseFraction: Maximum noise as fraction of Q (e.g., 0.01 = 1% of Q)
//   - avgNoiseFraction: Average noise as fraction of Q
//   - noiseDistribution: Map showing distribution of noise levels across bins
//
// The noise distribution bins are:
//   - "0-0.1%Q": Very low noise (< 0.1% of Q)
//   - "0.1-1%Q": Low noise (0.1-1% of Q)
//   - "1-5%Q": Moderate noise (1-5% of Q)
//   - "5-10%Q": High noise (5-10% of Q)
//   - "10-25%Q": Very high noise (10-25% of Q)
//   - ">25%Q": Excessive noise (> 25% of Q)
//
// Example:
//
//	maxNoise, avgNoise, dist := psi.MeasureNoiseLevel(r, original, decrypted, Q)
//	fmt.Printf("Max noise: %.2f%%, Avg noise: %.2f%%\n", maxNoise*100, avgNoise*100)
func MeasureNoiseLevel(r *ring.Ring, original, decrypted *ring.Poly, Q uint64) (maxNoiseFraction, avgNoiseFraction float64, noiseDistribution map[string]int) {
	diff := r.NewPoly()
	r.Sub(decrypted, original, diff)

	totalCoeffs := len(diff.Coeffs[0])
	maxNoise := uint64(0)
	totalNoise := uint64(0)

	noiseDistribution = map[string]int{
		"0-0.1%Q": 0,
		"0.1-1%Q": 0,
		"1-5%Q":   0,
		"5-10%Q":  0,
		"10-25%Q": 0,
		">25%Q":   0,
	}

	for _, coeff := range diff.Coeffs[0] {
		var noise uint64
		if coeff > Q/2 {
			noise = Q - coeff
		} else {
			noise = coeff
		}

		if noise > maxNoise {
			maxNoise = noise
		}

		totalNoise += noise

		noiseFraction := float64(noise) / float64(Q)
		switch {
		case noiseFraction <= 0.001:
			noiseDistribution["0-0.1%Q"]++
		case noiseFraction <= 0.01:
			noiseDistribution["0.1-1%Q"]++
		case noiseFraction <= 0.05:
			noiseDistribution["1-5%Q"]++
		case noiseFraction <= 0.1:
			noiseDistribution["5-10%Q"]++
		case noiseFraction <= 0.25:
			noiseDistribution["10-25%Q"]++
		default:
			noiseDistribution[">25%Q"]++
		}
	}

	maxNoiseFraction = float64(maxNoise) / float64(Q)
	avgNoiseFraction = float64(totalNoise) / float64(totalCoeffs) / float64(Q)

	return maxNoiseFraction, avgNoiseFraction, noiseDistribution
}
