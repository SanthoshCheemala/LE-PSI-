package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// PSIReport represents a simplified PSI analysis report.
// This structure contains essential metrics for PSI operations.
type PSIReport struct {
	Summary    SummaryMetrics    `json:"summary"`
	Parameters ParameterMetrics  `json:"parameters"`
	Timing     TimingMetrics     `json:"timing"`
	Noise      NoiseMetrics      `json:"noise"`
	Metadata   ReportMetadata    `json:"metadata"`
}

// SummaryMetrics contains high-level PSI operation statistics.
type SummaryMetrics struct {
	TotalOperations int     `json:"totalOperations"`
	TotalMatches    int     `json:"totalMatches"`
	TotalErrors     int     `json:"totalErrors"`
	SuccessRate     float64 `json:"successRate"`
}

// ParameterMetrics contains cryptographic parameter information.
type ParameterMetrics struct {
	Q             uint64  `json:"q"`
	QBits         int     `json:"qBits"`
	D             int     `json:"d"`
	N             int     `json:"n"`
	Layers        int     `json:"layers"`
	NumSlots      int     `json:"numSlots"`
	LoadFactor    float64 `json:"loadFactor"`
	CollisionProb float64 `json:"collisionProb"`
}

// TimingMetrics contains execution time breakdowns.
type TimingMetrics struct {
	TotalDuration    string  `json:"totalDuration"`
	EncryptionTime   string  `json:"encryptionTime"`
	ServerEncryption string  `json:"serverEncryption"`
	DecryptionTime   string  `json:"decryptionTime"`
	Throughput       float64 `json:"throughput"`
}

// NoiseMetrics contains noise analysis statistics.
type NoiseMetrics struct {
	MaxNoise      float64        `json:"maxNoise"`
	AvgNoise      float64        `json:"avgNoise"`
	Distribution  map[string]int `json:"distribution"`
}

// ReportMetadata contains report generation information.
type ReportMetadata struct {
	Timestamp   string `json:"timestamp"`
	Version     string `json:"version"`
	DatasetSize int    `json:"datasetSize"`
}

// WritePSIReport generates a simplified JSON report for PSI analysis.
// This function exports essential metrics without complex HTML generation.
//
// Parameters:
//   - jsonPath: Output path for JSON report
//   - totalMatches: Number of successful intersections
//   - totalErrors: Number of failed operations
//   - totalMaxNoise: Maximum noise level observed
//   - totalAvgNoise: Average noise level
//   - duration: Total execution time
//   - encDuration: Client encryption time
//   - serverEncDuration: Server encryption time
//   - decDuration: Decryption time
//   - leAnalysis: Lattice encryption parameters
//
// Example:
//
//	err := utils.WritePSIReport("report.json", matches, errors, maxNoise, avgNoise,
//	    totalTime, encTime, serverTime, decTime, leParams)
func WritePSIReport(
	jsonPath string,
	totalMatches int,
	totalErrors int,
	totalMaxNoise, totalAvgNoise float64,
	duration, encDuration, serverEncDuration, decDuration time.Duration,
	leAnalysis map[string]interface{},
) error {
	totalOps := totalMatches + totalErrors
	successRate := 0.0
	if totalOps > 0 {
		successRate = float64(totalMatches) / float64(totalOps) * 100
	}

	throughput := 0.0
	if duration.Seconds() > 0 {
		throughput = float64(totalOps) / duration.Seconds()
	}

	// Extract parameters with safe type assertions
	params := ParameterMetrics{}
	if val, ok := leAnalysis["Q"].(uint64); ok {
		params.Q = val
	}
	if val, ok := leAnalysis["qBits"].(int); ok {
		params.QBits = val
	}
	if val, ok := leAnalysis["D"].(int); ok {
		params.D = val
	}
	if val, ok := leAnalysis["N"].(int); ok {
		params.N = val
	}
	if val, ok := leAnalysis["Layers"].(int); ok {
		params.Layers = val
	}
	if val, ok := leAnalysis["NumSlots"].(int); ok {
		params.NumSlots = val
	}
	if val, ok := leAnalysis["LoadFactor"].(float64); ok {
		params.LoadFactor = val
	}
	if val, ok := leAnalysis["CollisionProb"].(float64); ok {
		params.CollisionProb = val
	}

	report := PSIReport{
		Summary: SummaryMetrics{
			TotalOperations: totalOps,
			TotalMatches:    totalMatches,
			TotalErrors:     totalErrors,
			SuccessRate:     successRate,
		},
		Parameters: params,
		Timing: TimingMetrics{
			TotalDuration:    duration.String(),
			EncryptionTime:   encDuration.String(),
			ServerEncryption: serverEncDuration.String(),
			DecryptionTime:   decDuration.String(),
			Throughput:       throughput,
		},
		Noise: NoiseMetrics{
			MaxNoise:     totalMaxNoise,
			AvgNoise:     totalAvgNoise,
			Distribution: make(map[string]int),
		},
		Metadata: ReportMetadata{
			Timestamp:   time.Now().Format("2006-01-02 15:04:05"),
			Version:     "LE-PSI v1.0",
			DatasetSize: totalOps,
		},
	}

	file, err := os.Create(jsonPath)
	if err != nil {
		return fmt.Errorf("failed to create report file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		return fmt.Errorf("failed to encode report: %w", err)
	}

	fmt.Printf("✓ Report saved: %s\n", jsonPath)
	return nil
}

// WriteEnhancedPSIReport is a legacy wrapper for backward compatibility.
// Use WritePSIReport for new code.
//
// Deprecated: Use WritePSIReport instead.
func WriteEnhancedPSIReport(
	filepath, jsonPath string,
	noiseStats []map[string]interface{},
	errorStats []map[string]interface{},
	totalMatches int,
	totalMaxNoise, totalAvgNoise float64,
	totalErrors int,
	duration, encDuration, serverEncDuration, decDuration time.Duration,
	leAnalysis map[string]interface{},
) {
	if err := WritePSIReport(jsonPath, totalMatches, totalErrors, totalMaxNoise, totalAvgNoise,
		duration, encDuration, serverEncDuration, decDuration, leAnalysis); err != nil {
		fmt.Printf("Error writing report: %v\n", err)
	}
}