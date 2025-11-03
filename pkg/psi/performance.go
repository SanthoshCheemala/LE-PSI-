package psi

import (
	"fmt"
	"runtime"
	"time"

	"github.com/tuneinsight/lattigo/v3/ring"
)

// PerformanceMonitor tracks PSI performance metrics
type PerformanceMonitor struct {
	StartTime      time.Time
	KeyGenTime     time.Duration
	HashingTime    time.Duration
	WitnessTime    time.Duration
	IntersectionTime time.Duration
	TotalOperations int
	NumWorkers     int
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor() *PerformanceMonitor {
	return &PerformanceMonitor{
		StartTime:  time.Now(),
		NumWorkers: runtime.NumCPU(),
	}
}

// TrackKeyGeneration records key generation timing
func (pm *PerformanceMonitor) TrackKeyGeneration(start time.Time) {
	pm.KeyGenTime = time.Since(start)
}

// TrackHashing records hashing timing
func (pm *PerformanceMonitor) TrackHashing(start time.Time) {
	pm.HashingTime = time.Since(start)
}

// TrackWitnessGeneration records witness generation timing
func (pm *PerformanceMonitor) TrackWitnessGeneration(start time.Time) {
	pm.WitnessTime = time.Since(start)
}

// TrackIntersectionDetection records intersection detection timing
func (pm *PerformanceMonitor) TrackIntersectionDetection(start time.Time) {
	pm.IntersectionTime = time.Since(start)
}

// PrintReport prints a comprehensive performance report
func (pm *PerformanceMonitor) PrintReport(verbose ...bool) {
	// Default to verbose mode if not specified
	verboseMode := true
	if len(verbose) > 0 {
		verboseMode = verbose[0]
	}
	
	if !verboseMode {
		return // Skip printing if in silent mode
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

// MeasureNoiseLevel calculates the noise level between an original message and its decrypted version.
// It returns:
// - maxNoiseFraction: maximum noise as a fraction of Q (e.g., 0.01 means 1% of Q)
// - avgNoiseFraction: average noise as a fraction of Q
// - noiseDistribution: a map showing the distribution of noise levels
func MeasureNoiseLevel(r *ring.Ring, original, decrypted *ring.Poly, Q uint64) (maxNoiseFraction, avgNoiseFraction float64, noiseDistribution map[string]int) {
    diff := r.NewPoly()
    r.Sub(decrypted, original, diff)
    
    totalCoeffs := len(diff.Coeffs[0])
    maxNoise := uint64(0)
    totalNoise := uint64(0)
    
    // Initialize noise distribution bins
    noiseDistribution = map[string]int{
        "0-0.1%Q": 0,
        "0.1-1%Q": 0,
        "1-5%Q": 0,
        "5-10%Q": 0,
        "10-25%Q": 0,
        ">25%Q": 0,
    }
    
    // Calculate noise for each coefficient
    for _, coeff := range diff.Coeffs[0] {
        // Convert coefficient to its absolute distance from 0
        // Consider both directions of noise (coeff could be close to Q when noise is negative)
        var noise uint64
        if coeff > Q/2 {
            noise = Q - coeff // negative noise (coeff close to Q)
        } else {
            noise = coeff // positive noise
        }
        
        // Track maximum noise
        if noise > maxNoise {
            maxNoise = noise
        }
        
        // Accumulate total noise for average calculation
        totalNoise += noise
        
        // Add to distribution buckets
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
    
    // Calculate max and average noise as fraction of Q
    maxNoiseFraction = float64(maxNoise) / float64(Q)
    avgNoiseFraction = float64(totalNoise) / float64(totalCoeffs) / float64(Q)
    
    return maxNoiseFraction, avgNoiseFraction, noiseDistribution
}
