package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/SanthoshCheemala/PSI/pkg/psi"
	"github.com/SanthoshCheemala/PSI/utils"
	_ "github.com/mattn/go-sqlite3"
)

// ScalabilityTest represents a single test configuration
type ScalabilityTest struct {
	Name           string
	ServerSize     int
	ClientSize     int
	OverlapPercent float64
	Description    string
}

// TestResult stores the results of a scalability test
type TestResult struct {
	TestName             string        `json:"test_name"`
	ServerDatasetSize    int           `json:"server_dataset_size"`
	ClientDatasetSize    int           `json:"client_dataset_size"`
	OverlapSize          int           `json:"overlap_size"`
	OverlapPercent       float64       `json:"overlap_percent"`
	MatchesFound         int           `json:"matches_found"`
	Accuracy             float64       `json:"accuracy"`
	InitializationTime   time.Duration `json:"initialization_time_ns"`
	EncryptionTime       time.Duration `json:"encryption_time_ns"`
	IntersectionTime     time.Duration `json:"intersection_time_ns"`
	TotalTime            time.Duration `json:"total_time_ns"`
	Throughput           float64       `json:"throughput_ops_per_sec"`
	MemoryEstimate       int64         `json:"memory_estimate_bytes"`
	Success              bool          `json:"success"`
	ErrorMessage         string        `json:"error_message,omitempty"`
	CryptographicParams  CryptoParams  `json:"cryptographic_params"`
	GoRuntimeStats       GoStats       `json:"go_runtime_stats"`
}

// GoStats stores Go runtime performance metrics
type GoStats struct {
	// Memory Statistics
	AllocatedMemoryMB    float64 `json:"allocated_memory_mb"`
	TotalAllocatedMB     float64 `json:"total_allocated_mb"`
	SystemMemoryMB       float64 `json:"system_memory_mb"`
	HeapAllocMB          float64 `json:"heap_alloc_mb"`
	HeapSysMB            float64 `json:"heap_sys_mb"`
	HeapIdleMB           float64 `json:"heap_idle_mb"`
	HeapInUseMB          float64 `json:"heap_inuse_mb"`
	StackInUseMB         float64 `json:"stack_inuse_mb"`
	
	// Garbage Collection Statistics
	NumGC                uint32  `json:"num_gc"`
	GCCPUPercentage      float64 `json:"gc_cpu_percentage"`
	LastGCPauseMs        float64 `json:"last_gc_pause_ms"`
	TotalGCPauseMs       float64 `json:"total_gc_pause_ms"`
	
	// Goroutine and CPU Statistics
	NumGoroutines        int     `json:"num_goroutines"`
	NumCPU               int     `json:"num_cpu"`
	GOMAXPROCS           int     `json:"gomaxprocs"`
	
	// Memory Allocation Statistics
	Mallocs              uint64  `json:"mallocs"`
	Frees                uint64  `json:"frees"`
	LiveObjects          uint64  `json:"live_objects"`
}

// CryptoParams stores cryptographic parameters
type CryptoParams struct {
	RingDimension int     `json:"ring_dimension"`
	Modulus       uint64  `json:"modulus"`
	MatrixSize    int     `json:"matrix_size"`
	TreeLayers    int     `json:"tree_layers"`
	NumSlots      int     `json:"num_slots"`
	LoadFactor    float64 `json:"load_factor"`
}

// ScalabilityReport aggregates all test results
type ScalabilityReport struct {
	Timestamp      string       `json:"timestamp"`
	TotalTests     int          `json:"total_tests"`
	SuccessfulTests int         `json:"successful_tests"`
	FailedTests    int          `json:"failed_tests"`
	TestResults    []TestResult `json:"test_results"`
	Summary        Summary      `json:"summary"`
}

// Summary provides aggregate statistics
type Summary struct {
	TotalDataProcessed     int     `json:"total_data_processed"`
	TotalMatchesFound      int     `json:"total_matches_found"`
	AverageAccuracy        float64 `json:"average_accuracy"`
	AverageThroughput      float64 `json:"average_throughput_ops_per_sec"`
	TotalExecutionTime     string  `json:"total_execution_time"`
	FastestTest            string  `json:"fastest_test"`
	SlowestTest            string  `json:"slowest_test"`
	LargestDatasetTested   int     `json:"largest_dataset_tested"`
	ScalabilityScore       float64 `json:"scalability_score"`
}

// Transaction represents a data record
type Transaction struct {
	TransactionID string  `json:"transaction_id"`
	Amount        float64 `json:"amount"`
	Currency      string  `json:"currency"`
	Merchant      string  `json:"merchant"`
	Timestamp     string  `json:"timestamp"`
}

func main() {
	fmt.Println("=================================================")
	fmt.Println("  LE-PSI SCALABILITY TESTING FRAMEWORK")
	fmt.Println("  Testing PSI on Large Datasets")
	fmt.Println("=================================================\n")

	// Define test cases for scalability analysis - ALL USING REAL DATABASE
	tests := []ScalabilityTest{
		{
			Name:           "Small-Scale",
			ServerSize:     100,
			ClientSize:     25,
			OverlapPercent: 0.0, // Will be calculated from real data
			Description:    "100 records from transactions.db",
		},
		{
			Name:           "Medium-Scale-1",
			ServerSize:     500,
			ClientSize:     100,
			OverlapPercent: 0.0, // Will be calculated from real data
			Description:    "500 records from transactions.db",
		},
		{
			Name:           "Medium-Scale-2",
			ServerSize:     1000,
			ClientSize:     200,
			OverlapPercent: 0.0, // Will be calculated from real data
			Description:    "1K records from transactions.db",
		},
		{
			Name:           "Large-Scale-1",
			ServerSize:     5000,
			ClientSize:     500,
			OverlapPercent: 0.0, // Will be calculated from real data
			Description:    "5K records from transactions.db",
		},
		{
			Name:           "Large-Scale-2",
			ServerSize:     10000,
			ClientSize:     1000,
			OverlapPercent: 0.0, // Will be calculated from real data
			Description:    "10K records from transactions.db",
		},
		{
			Name:           "Very-Large-Scale",
			ServerSize:     20000,
			ClientSize:     2000,
			OverlapPercent: 0.0, // Will be calculated from real data
			Description:    "20K records from transactions.db",
		},
		{
			Name:           "Max-Scale",
			ServerSize:     50000,
			ClientSize:     5000,
			OverlapPercent: 0.0, // Will be calculated from real data
			Description:    "50K records from transactions.db - maximum scale",
		},
	}

	// Create results directory
	resultsDir := "scalability_results"
	if err := os.MkdirAll(resultsDir, 0755); err != nil {
		log.Fatalf("Failed to create results directory: %v", err)
	}

	// Run all tests
	report := ScalabilityReport{
		Timestamp:   time.Now().Format("2006-01-02_15-04-05"),
		TestResults: make([]TestResult, 0),
	}

	fmt.Printf("Starting %d scalability tests...\n\n", len(tests))

	for i, test := range tests {
		fmt.Printf("[%d/%d] Running: %s\n", i+1, len(tests), test.Name)
		fmt.Printf("       %s\n", test.Description)

		result := runScalabilityTest(test)
		report.TestResults = append(report.TestResults, result)

		if result.Success {
			report.SuccessfulTests++
			fmt.Printf("       ✓ Success - Found %d matches in %v\n", result.MatchesFound, result.TotalTime)
			fmt.Printf("       Throughput: %.2f ops/sec\n", result.Throughput)
		} else {
			report.FailedTests++
			fmt.Printf("       ✗ Failed - %s\n", result.ErrorMessage)
		}
		fmt.Println()
	}

	report.TotalTests = len(tests)
	report.Summary = generateSummary(report.TestResults)

	// Save results
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	jsonPath := filepath.Join(resultsDir, fmt.Sprintf("scalability_test_%s.json", timestamp))
	htmlPath := filepath.Join(resultsDir, fmt.Sprintf("scalability_report_%s.html", timestamp))

	// Save JSON report
	if err := saveJSONReport(jsonPath, report); err != nil {
		log.Printf("Error saving JSON report: %v", err)
	} else {
		fmt.Printf("✓ JSON report saved: %s\n", jsonPath)
	}

	// Generate HTML report
	if err := generateHTMLReport(htmlPath, jsonPath); err != nil {
		log.Printf("Error generating HTML report: %v", err)
	} else {
		fmt.Printf("✓ HTML report saved: %s\n", htmlPath)
	}

	// Print summary
	fmt.Println("\n=================================================")
	fmt.Println("  SCALABILITY TEST SUMMARY")
	fmt.Println("=================================================")
	fmt.Printf("Total Tests:           %d\n", report.TotalTests)
	fmt.Printf("Successful:            %d\n", report.SuccessfulTests)
	fmt.Printf("Failed:                %d\n", report.FailedTests)
	fmt.Printf("Total Data Processed:  %d records\n", report.Summary.TotalDataProcessed)
	fmt.Printf("Total Matches Found:   %d\n", report.Summary.TotalMatchesFound)
	fmt.Printf("Average Accuracy:      %.2f%%\n", report.Summary.AverageAccuracy)
	fmt.Printf("Average Throughput:    %.2f ops/sec\n", report.Summary.AverageThroughput)
	fmt.Printf("Largest Dataset:       %d records\n", report.Summary.LargestDatasetTested)
	fmt.Printf("Scalability Score:     %.2f/100\n", report.Summary.ScalabilityScore)
	fmt.Printf("Total Execution Time:  %s\n", report.Summary.TotalExecutionTime)
	fmt.Println("=================================================")
}

func runScalabilityTest(test ScalabilityTest) TestResult {
	result := TestResult{
		TestName: test.Name,
		Success:  false,
	}

	startTime := time.Now()

	// Load data from database ONLY - no synthetic data
	serverData, clientData, expectedMatches := loadFromDatabase(test.ServerSize, test.ClientSize)
	result.ServerDatasetSize = len(serverData)
	result.ClientDatasetSize = len(clientData)

	result.OverlapSize = expectedMatches
	if result.ClientDatasetSize > 0 {
		result.OverlapPercent = float64(expectedMatches) / float64(result.ClientDatasetSize) * 100
	}

	// Prepare data
	serverStrings, err := utils.PrepareDataForPSI(serverData)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Server data preparation failed: %v", err)
		return result
	}

	clientStrings, err := utils.PrepareDataForPSI(clientData)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Client data preparation failed: %v", err)
		return result
	}

	serverHashes := utils.HashDataPoints(serverStrings)
	clientHashes := utils.HashDataPoints(clientStrings)

	// Step 1: Server Initialization
	initStart := time.Now()
	dbPath := fmt.Sprintf("test_%s.db", test.Name)
	ctx, err := psi.ServerInitialize(serverHashes, dbPath)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Server initialization failed: %v", err)
		return result
	}
	result.InitializationTime = time.Since(initStart)

	// Clean up database after test
	defer os.Remove(dbPath)

	// Get cryptographic parameters
	pp, msg, le := psi.GetPublicParameters(ctx)
	result.CryptographicParams = extractCryptoParams(ctx)

	// Step 2: Client Encryption
	encStart := time.Now()
	ciphertexts := psi.ClientEncrypt(clientHashes, pp, msg, le)
	result.EncryptionTime = time.Since(encStart)

	// Step 3: Intersection Detection
	intStart := time.Now()
	matches, err := psi.DetectIntersectionWithContext(ctx, ciphertexts)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Intersection detection failed: %v", err)
		return result
	}
	result.IntersectionTime = time.Since(intStart)

	// Calculate metrics
	result.TotalTime = time.Since(startTime)
	result.MatchesFound = len(matches)
	
	if expectedMatches > 0 {
		result.Accuracy = float64(result.MatchesFound) / float64(expectedMatches) * 100
	} else {
		result.Accuracy = 100.0
	}

	if result.TotalTime.Seconds() > 0 {
		result.Throughput = float64(result.ClientDatasetSize) / result.TotalTime.Seconds()
	}

	// Estimate memory usage
	result.MemoryEstimate = estimateMemoryUsage(
		result.CryptographicParams.RingDimension,
		result.CryptographicParams.MatrixSize,
		result.CryptographicParams.TreeLayers,
		result.ServerDatasetSize,
	)

	// Collect Go runtime statistics
	result.GoRuntimeStats = collectGoRuntimeStats()

	result.Success = true
	return result
}

func loadFromDatabase(serverSize, clientSize int) ([]interface{}, []interface{}, int) {
	dbPath := "../data/transactions.db"
	
	// Check if database exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		log.Fatalf("ERROR: Database %s not found! Cannot run tests without real data.", dbPath)
	}

	// Open database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("ERROR: Failed to open database: %v", err)
	}
	defer db.Close()

	// Load server data from database with specified limit
	fmt.Printf("Loading %d records from transactions.db...\n", serverSize)
	
	query := fmt.Sprintf("SELECT * FROM finanical_transactions LIMIT %d", serverSize)
	rows, err := db.Query(query)
	if err != nil {
		log.Fatalf("ERROR: Failed to query database: %v", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		log.Fatalf("ERROR: Failed to get columns: %v", err)
	}

	// Read all rows
	var serverData []interface{}
	for rows.Next() {
		// Create a slice to hold column values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		// Scan row
		if err := rows.Scan(valuePtrs...); err != nil {
			log.Printf("Warning: Failed to scan row: %v", err)
			continue
		}

		// Convert to map
		rowData := make(map[string]interface{})
		for i, col := range columns {
			var v interface{}
			val := values[i]
			b, ok := val.([]byte)
			if ok {
				v = string(b)
			} else {
				v = val
			}
			rowData[col] = v
		}
		
		serverData = append(serverData, rowData)
	}
	
	if len(serverData) == 0 {
		log.Fatalf("ERROR: No data loaded from database!")
	}

	fmt.Printf("✓ Loaded %d server records from database\n", len(serverData))

	// Create client dataset as a subset of server data (for realistic overlap)
	clientData := make([]interface{}, clientSize)
	overlapSize := clientSize // All client data overlaps with server
	
	for i := 0; i < clientSize; i++ {
		if i < len(serverData) {
			clientData[i] = serverData[i]
		} else {
			log.Fatalf("ERROR: Not enough data in database! Need at least %d records but only have %d", clientSize, len(serverData))
		}
	}

	return serverData, clientData, overlapSize
}

func extractCryptoParams(ctx *psi.ServerInitContext) CryptoParams {
	// Extract parameters from context
	// These would normally come from the LE parameters
	return CryptoParams{
		RingDimension: 256, // From your LE parameters
		Modulus:       180143985094819841,
		MatrixSize:    4,
		TreeLayers:    4,
		NumSlots:      0, // Would be calculated from dataset size
		LoadFactor:    0.5,
	}
}

// collectGoRuntimeStats gathers Go runtime performance metrics
func collectGoRuntimeStats() GoStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	stats := GoStats{
		// Memory Statistics (convert to MB)
		AllocatedMemoryMB: float64(m.Alloc) / 1024 / 1024,
		TotalAllocatedMB:  float64(m.TotalAlloc) / 1024 / 1024,
		SystemMemoryMB:    float64(m.Sys) / 1024 / 1024,
		HeapAllocMB:       float64(m.HeapAlloc) / 1024 / 1024,
		HeapSysMB:         float64(m.HeapSys) / 1024 / 1024,
		HeapIdleMB:        float64(m.HeapIdle) / 1024 / 1024,
		HeapInUseMB:       float64(m.HeapInuse) / 1024 / 1024,
		StackInUseMB:      float64(m.StackInuse) / 1024 / 1024,
		
		// Garbage Collection Statistics
		NumGC:         m.NumGC,
		GCCPUPercentage: m.GCCPUFraction * 100,
		
		// Goroutine and CPU Statistics
		NumGoroutines: runtime.NumGoroutine(),
		NumCPU:        runtime.NumCPU(),
		GOMAXPROCS:    runtime.GOMAXPROCS(0),
		
		// Memory Allocation Statistics
		Mallocs:      m.Mallocs,
		Frees:        m.Frees,
		LiveObjects:  m.Mallocs - m.Frees,
	}
	
	// Calculate GC pause times
	if m.NumGC > 0 {
		// Last GC pause
		stats.LastGCPauseMs = float64(m.PauseNs[(m.NumGC+255)%256]) / 1000000
		
		// Total GC pause time
		for _, pause := range m.PauseNs {
			stats.TotalGCPauseMs += float64(pause) / 1000000
		}
	}
	
	return stats
}

func estimateMemoryUsage(ringDim, matrixSize, layers, datasetSize int) int64 {
	// Rough memory estimation
	polySize := int64(ringDim * 8) // 8 bytes per coefficient
	matrixMemory := polySize * int64(matrixSize*matrixSize)
	treeMemory := polySize * int64(1<<layers)
	datasetMemory := int64(datasetSize * 32) // Rough estimate per data point
	
	return matrixMemory*6 + treeMemory + datasetMemory
}

func generateSummary(results []TestResult) Summary {
	var summary Summary
	
	var totalAccuracy float64
	var totalThroughput float64
	var totalExecTime time.Duration
	var fastestTime time.Duration = time.Hour * 999
	var slowestTime time.Duration
	var fastestTest, slowestTest string
	var maxDataset int
	
	successCount := 0
	
	for _, result := range results {
		if !result.Success {
			continue
		}
		
		successCount++
		summary.TotalDataProcessed += result.ClientDatasetSize
		summary.TotalMatchesFound += result.MatchesFound
		totalAccuracy += result.Accuracy
		totalThroughput += result.Throughput
		totalExecTime += result.TotalTime
		
		if result.TotalTime < fastestTime {
			fastestTime = result.TotalTime
			fastestTest = result.TestName
		}
		
		if result.TotalTime > slowestTime {
			slowestTime = result.TotalTime
			slowestTest = result.TestName
		}
		
		if result.ServerDatasetSize > maxDataset {
			maxDataset = result.ServerDatasetSize
		}
	}
	
	if successCount > 0 {
		summary.AverageAccuracy = totalAccuracy / float64(successCount)
		summary.AverageThroughput = totalThroughput / float64(successCount)
	}
	
	summary.TotalExecutionTime = totalExecTime.String()
	summary.FastestTest = fmt.Sprintf("%s (%v)", fastestTest, fastestTime)
	summary.SlowestTest = fmt.Sprintf("%s (%v)", slowestTest, slowestTime)
	summary.LargestDatasetTested = maxDataset
	
	// Calculate scalability score (0-100)
	// Based on: throughput, accuracy, and ability to handle large datasets
	baseScore := (summary.AverageThroughput / 100.0) * 30 // Max 30 points for throughput
	accuracyScore := (summary.AverageAccuracy / 100.0) * 40 // Max 40 points for accuracy
	scaleScore := float64(min(maxDataset, 20000)) / 20000.0 * 30 // Max 30 points for scale
	
	summary.ScalabilityScore = minFloat(baseScore+accuracyScore+scaleScore, 100.0)
	
	return summary
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func saveJSONReport(filepath string, report ScalabilityReport) error {
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func generateHTMLReport(htmlPath, jsonPath string) error {
	htmlContent := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>LE-PSI Scalability Report</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', system-ui, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: #333;
            line-height: 1.6;
            min-height: 100vh;
            padding: 2rem;
        }
        
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            border-radius: 12px;
            box-shadow: 0 20px 60px rgba(0,0,0,0.3);
            overflow: hidden;
        }
        
        .header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 3rem 2rem;
            text-align: center;
        }
        
        .header h1 {
            font-size: 2.5rem;
            font-weight: 700;
            margin-bottom: 0.5rem;
        }
        
        .header p {
            font-size: 1.1rem;
            opacity: 0.9;
        }
        
        .content {
            padding: 2rem;
        }
        
        .summary-cards {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 1.5rem;
            margin-bottom: 2rem;
        }
        
        .card {
            background: linear-gradient(135deg, #f5f7fa 0%, #c3cfe2 100%);
            border-radius: 8px;
            padding: 1.5rem;
            text-align: center;
            box-shadow: 0 4px 6px rgba(0,0,0,0.1);
            transition: transform 0.2s;
        }
        
        .card:hover {
            transform: translateY(-5px);
        }
        
        .card-value {
            font-size: 2rem;
            font-weight: 700;
            color: #667eea;
            margin-bottom: 0.5rem;
        }
        
        .card-label {
            font-size: 0.9rem;
            color: #666;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }
        
        .section {
            background: #f8f9fa;
            border-radius: 8px;
            padding: 1.5rem;
            margin-bottom: 1.5rem;
        }
        
        .section h2 {
            font-size: 1.5rem;
            color: #667eea;
            margin-bottom: 1rem;
            padding-bottom: 0.5rem;
            border-bottom: 2px solid #667eea;
        }
        
        .test-results {
            display: grid;
            gap: 1rem;
        }
        
        .test-card {
            background: white;
            border-radius: 6px;
            padding: 1.5rem;
            border-left: 4px solid #667eea;
            box-shadow: 0 2px 4px rgba(0,0,0,0.05);
        }
        
        .test-card.failed {
            border-left-color: #e74c3c;
        }
        
        .test-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 1rem;
        }
        
        .test-name {
            font-size: 1.2rem;
            font-weight: 600;
            color: #2c3e50;
        }
        
        .test-status {
            padding: 0.25rem 0.75rem;
            border-radius: 20px;
            font-size: 0.85rem;
            font-weight: 600;
        }
        
        .test-status.success {
            background: #d4edda;
            color: #155724;
        }
        
        .test-status.failed {
            background: #f8d7da;
            color: #721c24;
        }
        
        .test-metrics {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 1rem;
            margin-top: 1rem;
        }
        
        .metric {
            display: flex;
            justify-content: space-between;
            padding: 0.5rem;
            background: #f8f9fa;
            border-radius: 4px;
        }
        
        .metric-label {
            color: #666;
            font-size: 0.9rem;
        }
        
        .metric-value {
            font-weight: 600;
            color: #2c3e50;
            font-family: 'SF Mono', Monaco, monospace;
            font-size: 0.9rem;
        }
        
        .chart-container {
            margin-top: 1rem;
            height: 300px;
        }
        
        .loading {
            text-align: center;
            padding: 3rem;
            color: #667eea;
            font-size: 1.2rem;
        }
        
        .timestamp {
            text-align: center;
            padding: 1rem;
            color: #999;
            font-size: 0.9rem;
            border-top: 1px solid #e9ecef;
        }
        
        @media (max-width: 768px) {
            body { padding: 1rem; }
            .header h1 { font-size: 1.8rem; }
            .summary-cards { grid-template-columns: 1fr; }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🔐 LE-PSI Scalability Report</h1>
            <p>Private Set Intersection - Performance Analysis</p>
        </div>

        <div class="content">
            <div id="loading" class="loading">
                Loading test results...
            </div>

            <div id="report" style="display: none;">
                <!-- Summary Cards -->
                <div class="summary-cards" id="summaryCards"></div>

                <!-- Detailed Results -->
                <div class="section">
                    <h2>📊 Test Results</h2>
                    <div class="test-results" id="testResults"></div>
                </div>

                <!-- Performance Analysis -->
                <div class="section">
                    <h2>⚡ Performance Analysis</h2>
                    <div id="performanceAnalysis"></div>
                </div>

                <div class="timestamp" id="timestamp"></div>
            </div>
        </div>
    </div>

    <script>
        async function loadData() {
            try {
                const jsonFile = '` + filepath.Base(jsonPath) + `';
                const response = await fetch(jsonFile);
                const data = await response.json();
                renderReport(data);
            } catch (error) {
                document.getElementById('loading').innerHTML = 
                    '<div style="color: #e74c3c;">Error loading test results</div>';
            }
        }

        function renderReport(data) {
            document.getElementById('loading').style.display = 'none';
            document.getElementById('report').style.display = 'block';
            
            renderSummaryCards(data);
            renderTestResults(data);
            renderPerformanceAnalysis(data);
            
            document.getElementById('timestamp').innerHTML = 
                'Report generated: ' + data.timestamp;
        }

        function renderSummaryCards(data) {
            const cards = [
                { label: 'Total Tests', value: data.total_tests },
                { label: 'Success Rate', value: ((data.successful_tests / data.total_tests) * 100).toFixed(1) + '%' },
                { label: 'Total Matches', value: data.summary.total_matches_found.toLocaleString() },
                { label: 'Avg Accuracy', value: data.summary.average_accuracy.toFixed(2) + '%' },
                { label: 'Avg Throughput', value: data.summary.average_throughput_ops_per_sec.toFixed(1) + ' ops/s' },
                { label: 'Scalability Score', value: data.summary.scalability_score.toFixed(1) + '/100' }
            ];

            const html = cards.map(card => 
                '<div class="card">' +
                    '<div class="card-value">' + card.value + '</div>' +
                    '<div class="card-label">' + card.label + '</div>' +
                '</div>'
            ).join('');
            
            document.getElementById('summaryCards').innerHTML = html;
        }

        function renderTestResults(data) {
            const html = data.test_results.map(test => {
                const statusClass = test.success ? 'success' : 'failed';
                const status = test.success ? '✓ Success' : '✗ Failed';
                
                return '<div class="test-card ' + (test.success ? '' : 'failed') + '">' +
                    '<div class="test-header">' +
                        '<div class="test-name">' + test.test_name + '</div>' +
                        '<div class="test-status ' + statusClass + '">' + status + '</div>' +
                    '</div>' +
                    (test.success ? renderTestMetrics(test) : 
                        '<div style="color: #e74c3c;">' + test.error_message + '</div>') +
                '</div>';
            }).join('');
            
            document.getElementById('testResults').innerHTML = html;
        }

        function renderTestMetrics(test) {
            let html = '<div class="test-metrics">' +
                '<div class="metric">' +
                    '<span class="metric-label">Dataset Size</span>' +
                    '<span class="metric-value">' + test.server_dataset_size.toLocaleString() + ' / ' + 
                    test.client_dataset_size.toLocaleString() + '</span>' +
                '</div>' +
                '<div class="metric">' +
                    '<span class="metric-label">Matches Found</span>' +
                    '<span class="metric-value">' + test.matches_found + ' / ' + test.overlap_size + '</span>' +
                '</div>' +
                '<div class="metric">' +
                    '<span class="metric-label">Accuracy</span>' +
                    '<span class="metric-value">' + test.accuracy.toFixed(2) + '%</span>' +
                '</div>' +
                '<div class="metric">' +
                    '<span class="metric-label">Total Time</span>' +
                    '<span class="metric-value">' + (test.total_time_ns / 1000000).toFixed(0) + ' ms</span>' +
                '</div>' +
                '<div class="metric">' +
                    '<span class="metric-label">Throughput</span>' +
                    '<span class="metric-value">' + test.throughput_ops_per_sec.toFixed(1) + ' ops/s</span>' +
                '</div>' +
                '<div class="metric">' +
                    '<span class="metric-label">Memory Est.</span>' +
                    '<span class="metric-value">' + (test.memory_estimate_bytes / 1024 / 1024).toFixed(1) + ' MB</span>' +
                '</div>' +
            '</div>';
            
            // Add Go Runtime Statistics if available
            if (test.go_runtime_stats) {
                html += '<h4 style="margin-top: 20px; color: #2c3e50;">🔧 Go Runtime Performance</h4>';
                html += '<div class="test-metrics">' +
                    '<div class="metric">' +
                        '<span class="metric-label">Heap Memory</span>' +
                        '<span class="metric-value">' + test.go_runtime_stats.heap_alloc_mb.toFixed(2) + ' MB</span>' +
                    '</div>' +
                    '<div class="metric">' +
                        '<span class="metric-label">System Memory</span>' +
                        '<span class="metric-value">' + test.go_runtime_stats.system_memory_mb.toFixed(2) + ' MB</span>' +
                    '</div>' +
                    '<div class="metric">' +
                        '<span class="metric-label">Goroutines</span>' +
                        '<span class="metric-value">' + test.go_runtime_stats.num_goroutines + '</span>' +
                    '</div>' +
                    '<div class="metric">' +
                        '<span class="metric-label">GC Runs</span>' +
                        '<span class="metric-value">' + test.go_runtime_stats.num_gc + '</span>' +
                    '</div>' +
                    '<div class="metric">' +
                        '<span class="metric-label">GC CPU %</span>' +
                        '<span class="metric-value">' + test.go_runtime_stats.gc_cpu_percentage.toFixed(2) + '%</span>' +
                    '</div>' +
                    '<div class="metric">' +
                        '<span class="metric-label">Live Objects</span>' +
                        '<span class="metric-value">' + test.go_runtime_stats.live_objects.toLocaleString() + '</span>' +
                    '</div>' +
                    '<div class="metric">' +
                        '<span class="metric-label">CPUs Used</span>' +
                        '<span class="metric-value">' + test.go_runtime_stats.gomaxprocs + ' / ' + test.go_runtime_stats.num_cpu + '</span>' +
                    '</div>' +
                    '<div class="metric">' +
                        '<span class="metric-label">Last GC Pause</span>' +
                        '<span class="metric-value">' + test.go_runtime_stats.last_gc_pause_ms.toFixed(2) + ' ms</span>' +
                    '</div>' +
                '</div>';
            }
            
            return html;
        }

        function renderPerformanceAnalysis(data) {
            const html = '<div class="test-metrics">' +
                '<div class="metric">' +
                    '<span class="metric-label">Largest Dataset</span>' +
                    '<span class="metric-value">' + data.summary.largest_dataset_tested.toLocaleString() + ' records</span>' +
                '</div>' +
                '<div class="metric">' +
                    '<span class="metric-label">Total Data Processed</span>' +
                    '<span class="metric-value">' + data.summary.total_data_processed.toLocaleString() + ' records</span>' +
                '</div>' +
                '<div class="metric">' +
                    '<span class="metric-label">Fastest Test</span>' +
                    '<span class="metric-value">' + data.summary.fastest_test + '</span>' +
                '</div>' +
                '<div class="metric">' +
                    '<span class="metric-label">Slowest Test</span>' +
                    '<span class="metric-value">' + data.summary.slowest_test + '</span>' +
                '</div>' +
            '</div>';
            
            document.getElementById('performanceAnalysis').innerHTML = html;
        }

        window.addEventListener('load', loadData);
    </script>
</body>
</html>`;

	return os.WriteFile(htmlPath, []byte(htmlContent), 0644)
}
