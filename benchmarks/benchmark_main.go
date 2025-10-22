package benchmarks
// FLARE PSI Benchmarking Tool
// Uses new distributed PSI architecture for accurate performance measurement

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	psi "github.com/SanthoshCheemala/FLARE/internal/crypto/PSI"
	"github.com/SanthoshCheemala/FLARE/utils"
)

// BenchmarkConfig holds benchmark configuration
type BenchmarkConfig struct {
	ServerSize    int
	ClientSize    int
	RingDimension int
	OutputDir     string
	Verbose       bool
	Iterations    int
}

// BenchmarkResult holds benchmark results
type BenchmarkResult struct {
	Config              BenchmarkConfig       `json:"config"`
	InitializationTime  time.Duration         `json:"initializationTime"`
	EncryptionTime      time.Duration         `json:"encryptionTime"`
	DetectionTime       time.Duration         `json:"detectionTime"`
	TotalTime           time.Duration         `json:"totalTime"`
	Throughput          float64               `json:"throughput"`
	IntersectionSize    int                   `json:"intersectionSize"`
	MemoryUsageMB       uint64                `json:"memoryUsageMB"`
	CPUCores            int                   `json:"cpuCores"`
	Timestamp           string                `json:"timestamp"`
}

func main() {
	// Parse flags
	config := BenchmarkConfig{}
	flag.IntVar(&config.ServerSize, "server-size", 50, "Number of items in server dataset")
	flag.IntVar(&config.ClientSize, "client-size", 20, "Number of items in client dataset")
	flag.IntVar(&config.RingDimension, "ring-dimension", 256, "Ring dimension: 256, 512, 1024, or 2048")
	flag.StringVar(&config.OutputDir, "output-dir", "benchmark_results", "Output directory for results")
	flag.BoolVar(&config.Verbose, "verbose", false, "Enable verbose logging")
	flag.IntVar(&config.Iterations, "iterations", 1, "Number of benchmark iterations")
	flag.Parse()

	// Validate config
	if err := validateConfig(&config); err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}

	// Create output directory
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Run benchmarks
	fmt.Println("=== FLARE PSI Benchmark ===")
	fmt.Printf("Server Set Size: %d\n", config.ServerSize)
	fmt.Printf("Client Set Size: %d\n", config.ClientSize)
	fmt.Printf("Ring Dimension: %d\n", config.RingDimension)
	fmt.Printf("Iterations: %d\n", config.Iterations)
	fmt.Printf("CPU Cores: %d\n\n", runtime.NumCPU())

	// Run multiple iterations and average
	var totalInit, totalEncrypt, totalDetect, totalOverall time.Duration
	var totalIntersection int
	var avgMemory uint64

	for i := 0; i < config.Iterations; i++ {
		if config.Iterations > 1 {
			fmt.Printf("Iteration %d/%d...\n", i+1, config.Iterations)
		}

		result := runBenchmark(&config)
		
		totalInit += result.InitializationTime
		totalEncrypt += result.EncryptionTime
		totalDetect += result.DetectionTime
		totalOverall += result.TotalTime
		totalIntersection += result.IntersectionSize
		avgMemory += result.MemoryUsageMB
	}

	// Calculate averages
	avgResult := BenchmarkResult{
		Config:             config,
		InitializationTime: totalInit / time.Duration(config.Iterations),
		EncryptionTime:     totalEncrypt / time.Duration(config.Iterations),
		DetectionTime:      totalDetect / time.Duration(config.Iterations),
		TotalTime:          totalOverall / time.Duration(config.Iterations),
		IntersectionSize:   totalIntersection / config.Iterations,
		MemoryUsageMB:      avgMemory / uint64(config.Iterations),
		CPUCores:           runtime.NumCPU(),
		Timestamp:          time.Now().Format("2006-01-02 15:04:05"),
	}

	// Calculate throughput (operations per second)
	totalOps := float64(config.ServerSize * config.ClientSize)
	avgResult.Throughput = totalOps / avgResult.TotalTime.Seconds()

	// Display results
	displayResults(&avgResult)

	// Save results
	saveResults(&avgResult, config.OutputDir)

	fmt.Printf("\nBenchmark complete! Results saved to %s/\n", config.OutputDir)
}

func validateConfig(config *BenchmarkConfig) error {
	if config.ServerSize < 1 {
		return fmt.Errorf("server size must be positive")
	}
	if config.ClientSize < 1 {
		return fmt.Errorf("client size must be positive")
	}
	if config.RingDimension != 256 && config.RingDimension != 512 && 
	   config.RingDimension != 1024 && config.RingDimension != 2048 {
		return fmt.Errorf("ring dimension must be 256, 512, 1024, or 2048")
	}
	if config.Iterations < 1 {
		return fmt.Errorf("iterations must be positive")
	}
	return nil
}

func runBenchmark(config *BenchmarkConfig) BenchmarkResult {
	var memStats runtime.MemStats
	
	// Generate synthetic datasets
	serverData := generateSyntheticData(config.ServerSize, "server")
	clientData := generateSyntheticData(config.ClientSize, "client")
	
	// Add some overlapping items
	overlapCount := min(config.ServerSize/4, config.ClientSize/2)
	for i := 0; i < overlapCount; i++ {
		clientData[i] = serverData[i]
	}

	dbPath := fmt.Sprintf("%s/benchmark_tree_%d.db", config.OutputDir, time.Now().UnixNano())

	// Phase 1: Server Initialization
	runtime.ReadMemStats(&memStats)
	memBefore := memStats.Alloc
	
	startInit := time.Now()
	serverHashes, err := preprocessData(serverData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error preprocessing server data: %v\n", err)
		os.Exit(1)
	}
	
	serverCtx, err := psi.ServerInitialize(serverHashes, dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing server: %v\n", err)
		os.Exit(1)
	}
	initTime := time.Since(startInit)

	// Phase 2: Client Encryption
	startEncrypt := time.Now()
	clientHashes, err := preprocessData(clientData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error preprocessing client data: %v\n", err)
		os.Exit(1)
	}
	
	ciphertexts := psi.Client(clientHashes, serverCtx.PublicParams, serverCtx.Message, serverCtx.LEParams)
	encryptTime := time.Since(startEncrypt)

	// Phase 3: Intersection Detection
	startDetect := time.Now()
	intersectionHashes, err := psi.DetectIntersectionWithContext(serverCtx, ciphertexts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error detecting intersection: %v\n", err)
		os.Exit(1)
	}
	detectTime := time.Since(startDetect)

	// Calculate memory usage
	runtime.ReadMemStats(&memStats)
	memAfter := memStats.Alloc
	memUsedMB := (memAfter - memBefore) / (1024 * 1024)

	// Total time
	totalTime := initTime + encryptTime + detectTime

	// Clean up
	os.Remove(dbPath)

	return BenchmarkResult{
		Config:             *config,
		InitializationTime: initTime,
		EncryptionTime:     encryptTime,
		DetectionTime:      detectTime,
		TotalTime:          totalTime,
		IntersectionSize:   len(intersectionHashes),
		MemoryUsageMB:      memUsedMB,
		CPUCores:           runtime.NumCPU(),
		Timestamp:          time.Now().Format("2006-01-02 15:04:05"),
	}
}

func generateSyntheticData(count int, prefix string) []interface{} {
	data := make([]interface{}, count)
	for i := 0; i < count; i++ {
		switch i % 5 {
		case 0:
			data[i] = fmt.Sprintf("%s_user_%d@example.com", prefix, i)
		case 1:
			data[i] = fmt.Sprintf("%s_item_%d", prefix, i)
		case 2:
			data[i] = i * 12345 + 67890
		case 3:
			data[i] = map[string]interface{}{
				"id":   i,
				"type": prefix,
				"value": fmt.Sprintf("data_%d", i),
			}
		case 4:
			data[i] = fmt.Sprintf("%s_%d_%d", prefix, i, time.Now().UnixNano()%1000)
		}
	}
	return data
}

func preprocessData(dataset []interface{}) ([]uint64, error) {
	serialized, err := utils.PrepareDataForPSI(dataset)
	if err != nil {
		return nil, err
	}
	return utils.HashDataPoints(serialized), nil
}

func displayResults(result *BenchmarkResult) {
	fmt.Println("\n=== Benchmark Results ===")
	fmt.Printf("Initialization:  %v\n", result.InitializationTime)
	fmt.Printf("Encryption:      %v\n", result.EncryptionTime)
	fmt.Printf("Detection:       %v\n", result.DetectionTime)
	fmt.Printf("Total Time:      %v\n", result.TotalTime)
	fmt.Printf("Throughput:      %.2f ops/sec\n", result.Throughput)
	fmt.Printf("Memory Used:     %d MB\n", result.MemoryUsageMB)
	fmt.Printf("Intersection:    %d items found\n", result.IntersectionSize)
	
	// Calculate percentages
	initPct := float64(result.InitializationTime) / float64(result.TotalTime) * 100
	encryptPct := float64(result.EncryptionTime) / float64(result.TotalTime) * 100
	detectPct := float64(result.DetectionTime) / float64(result.TotalTime) * 100
	
	fmt.Println("\nTime Breakdown:")
	fmt.Printf("  Initialization: %.1f%%\n", initPct)
	fmt.Printf("  Encryption:     %.1f%%\n", encryptPct)
	fmt.Printf("  Detection:      %.1f%%\n", detectPct)
}

func saveResults(result *BenchmarkResult, outputDir string) {
	// Save JSON results
	jsonPath := fmt.Sprintf("%s/benchmark_result.json", outputDir)
	file, err := os.Create(jsonPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating results file: %v\n", err)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding results: %v\n", err)
	}

	// Generate detailed breakdown
	breakdown := map[string]interface{}{
		"phases": map[string]interface{}{
			"initialization": map[string]interface{}{
				"duration_ms":  result.InitializationTime.Milliseconds(),
				"duration_sec": result.InitializationTime.Seconds(),
				"percentage":   float64(result.InitializationTime) / float64(result.TotalTime) * 100,
			},
			"encryption": map[string]interface{}{
				"duration_ms":  result.EncryptionTime.Milliseconds(),
				"duration_sec": result.EncryptionTime.Seconds(),
				"percentage":   float64(result.EncryptionTime) / float64(result.TotalTime) * 100,
			},
			"detection": map[string]interface{}{
				"duration_ms":  result.DetectionTime.Milliseconds(),
				"duration_sec": result.DetectionTime.Seconds(),
				"percentage":   float64(result.DetectionTime) / float64(result.TotalTime) * 100,
			},
		},
		"metrics": map[string]interface{}{
			"total_operations":     result.Config.ServerSize * result.Config.ClientSize,
			"throughput_ops_sec":   result.Throughput,
			"memory_usage_mb":      result.MemoryUsageMB,
			"cpu_cores":            result.CPUCores,
			"intersection_found":   result.IntersectionSize,
			"intersection_percent": float64(result.IntersectionSize) / float64(result.Config.ClientSize) * 100,
		},
		"timestamp": result.Timestamp,
	}

	breakdownPath := fmt.Sprintf("%s/timing_breakdown.json", outputDir)
	breakdownFile, err := os.Create(breakdownPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating breakdown file: %v\n", err)
		return
	}
	defer breakdownFile.Close()

	encoder = json.NewEncoder(breakdownFile)
	encoder.SetIndent("", "  ")
	encoder.Encode(breakdown)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
