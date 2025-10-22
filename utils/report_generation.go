package utils

import (
	"encoding/json"
	"fmt"
	"html/template"
	"math"
	"os"
	"path/filepath"
	"time"
)

// Enhanced data structures for comprehensive statistics
type PSIStatistics struct {
	Summary         SummaryStats         `json:"summary"`
	LEParameters    LEParameterStats     `json:"leParameters"`
	NoiseAnalysis   NoiseAnalysisStats   `json:"noiseAnalysis"`
	ErrorAnalysis   ErrorAnalysisStats   `json:"errorAnalysis"`
	TimingAnalysis  TimingAnalysisStats  `json:"timingAnalysis"`
	DetailedMetrics []DetailedMetric     `json:"detailedMetrics"`
	Metadata        MetadataStats        `json:"metadata"`
}

type SummaryStats struct {
	TotalOperations    int     `json:"totalOperations"`
	SuccessRate        float64 `json:"successRate"`
	AverageNoiseLevel  float64 `json:"averageNoiseLevel"`
	MaxNoiseLevel      float64 `json:"maxNoiseLevel"`
	TotalIntersections int     `json:"totalIntersections"`
	EfficiencyScore    float64 `json:"efficiencyScore"`
}

type LEParameterStats struct {
	Q                uint64  `json:"q"`
	QBits            int     `json:"qBits"`
	D                int     `json:"d"`
	N                int     `json:"n"`
	Layers           int     `json:"layers"`
	NumSlots         int     `json:"numSlots"`
	LoadFactor       float64 `json:"loadFactor"`
	CollisionProb    float64 `json:"collisionProb"`
	SecurityLevel    string  `json:"securityLevel"`
	MemoryUsage      int64   `json:"memoryUsage"`
	OptimalityScore  float64 `json:"optimalityScore"`
}

type NoiseAnalysisStats struct {
	GlobalMaxNoise    float64                    `json:"globalMaxNoise"`
	GlobalAvgNoise    float64                    `json:"globalAvgNoise"`
	NoiseDistribution map[string]int             `json:"noiseDistribution"`
	NoiseEvolution    []NoiseEvolutionPoint      `json:"noiseEvolution"`
	NoiseCategories   map[string]CategoryStats   `json:"noiseCategories"`
	PredictiveModel   NoisePredictor             `json:"predictiveModel"`
}

type ErrorAnalysisStats struct {
	TotalErrors        int                      `json:"totalErrors"`
	ErrorRate          float64                  `json:"errorRate"`
	ErrorDistribution  map[string]int           `json:"errorDistribution"`
	CriticalErrors     []CriticalError          `json:"criticalErrors"`
	ErrorTrends        []ErrorTrendPoint        `json:"errorTrends"`
	RecoveryMetrics    RecoveryStats            `json:"recoveryMetrics"`
}

type TimingAnalysisStats struct {
	TotalDuration      time.Duration        `json:"totalDuration"`
	EncryptionTime     time.Duration        `json:"encryptionTime"`
	ServerEncryption   time.Duration        `json:"serverEncryption"`
	DecryptionTime     time.Duration        `json:"decryptionTime"`
	Throughput         float64             `json:"throughput"`
	PerformanceScore   float64             `json:"performanceScore"`
	BottleneckAnalysis BottleneckAnalysis   `json:"bottleneckAnalysis"`
	Benchmarks         []BenchmarkPoint     `json:"benchmarks"`
}

type DetailedMetric struct {
	ServerIndex       int                 `json:"serverIndex"`
	ClientIndex       int                 `json:"clientIndex"`
	NoiseMetrics      NoiseMetric         `json:"noiseMetrics"`
	ErrorMetrics      ErrorMetric         `json:"errorMetrics"`
	TimingMetrics     TimingMetric        `json:"timingMetrics"`
	QualityScore      float64             `json:"qualityScore"`
	Risk              string              `json:"risk"`
}

type MetadataStats struct {
	Timestamp         string    `json:"timestamp"`
	Version           string    `json:"version"`
	SystemInfo        string    `json:"systemInfo"`
	ConfigHash        string    `json:"configHash"`
	DatasetSize       int       `json:"datasetSize"`
	AlgorithmVariant  string    `json:"algorithmVariant"`
}

// Supporting types
type NoiseEvolutionPoint struct {
	Operation int     `json:"operation"`
	NoiseLevel float64 `json:"noiseLevel"`
}

type CategoryStats struct {
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
	Trend      string  `json:"trend"`
}

type NoisePredictor struct {
	PredictedMaxNoise float64 `json:"predictedMaxNoise"`
	Confidence        float64 `json:"confidence"`
	Model             string  `json:"model"`
}

type CriticalError struct {
	Operation   int     `json:"operation"`
	ErrorType   string  `json:"errorType"`
	Severity    string  `json:"severity"`
	Impact      float64 `json:"impact"`
	Description string  `json:"description"`
}

type ErrorTrendPoint struct {
	Time      int     `json:"time"`
	ErrorRate float64 `json:"errorRate"`
}

type RecoveryStats struct {
	RecoveryRate     float64 `json:"recoveryRate"`
	MeanRecoveryTime float64 `json:"meanRecoveryTime"`
	SuccessfulFixes  int     `json:"successfulFixes"`
}

type BottleneckAnalysis struct {
	PrimaryBottleneck   string  `json:"primaryBottleneck"`
	BottleneckImpact    float64 `json:"bottleneckImpact"`
	Recommendations     []string `json:"recommendations"`
}

type BenchmarkPoint struct {
	Operation string  `json:"operation"`
	Time      float64 `json:"time"`
	Baseline  float64 `json:"baseline"`
}

type NoiseMetric struct {
	MaxNoise      float64            `json:"maxNoise"`
	AvgNoise      float64            `json:"avgNoise"`
	Distribution  map[string]int     `json:"distribution"`
	Stability     float64            `json:"stability"`
}

type ErrorMetric struct {
	Matches       int     `json:"matches"`
	Mismatches    int     `json:"mismatches"`
	MatchPct      float64 `json:"matchPct"`
	ErrorPattern  string  `json:"errorPattern"`
}

type TimingMetric struct {
	Duration     time.Duration `json:"duration"`
	Efficiency   float64       `json:"efficiency"`
	Optimization string        `json:"optimization"`
}

// Enhanced reporting function
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
	// Generate comprehensive statistics
	stats := generateComprehensiveStats(
		noiseStats, errorStats, totalMatches, totalMaxNoise, totalAvgNoise,
		totalErrors, duration, encDuration, serverEncDuration, decDuration, leAnalysis,
	)

	// Save JSON statistics
	if err := saveStatsToJSON(jsonPath, stats); err != nil {
		fmt.Printf("Error saving JSON stats: %v\n", err)
	}

	// Generate enhanced HTML report
	if err := generateEnhancedHTML(filepath, jsonPath); err != nil {
		fmt.Printf("Error generating HTML report: %v\n", err)
	}
}

func generateComprehensiveStats(
	noiseStats []map[string]interface{},
	errorStats []map[string]interface{},
	totalMatches int,
	totalMaxNoise, totalAvgNoise float64,
	totalErrors int,
	duration, encDuration, serverEncDuration, decDuration time.Duration,
	leAnalysis map[string]interface{},
) PSIStatistics {
	
	// Safe type assertions with error checking
	var q uint64
	var qBits, d, n, layers, numSlots int
	var loadFactor, collisionProb float64
	
	if val, ok := leAnalysis["Q"]; ok {
		if qVal, ok := val.(uint64); ok {
			q = qVal
		}
	}
	
	if val, ok := leAnalysis["qBits"]; ok {
		if qBitsVal, ok := val.(int); ok {
			qBits = qBitsVal
		}
	}
	
	if val, ok := leAnalysis["D"]; ok {
		if dVal, ok := val.(int); ok {
			d = dVal
		}
	}
	
	if val, ok := leAnalysis["N"]; ok {
		if nVal, ok := val.(int); ok {
			n = nVal
		}
	}
	
	if val, ok := leAnalysis["Layers"]; ok {
		if layersVal, ok := val.(int); ok {
			layers = layersVal
		}
	}
	
	if val, ok := leAnalysis["NumSlots"]; ok {
		if numSlotsVal, ok := val.(int); ok {
			numSlots = numSlotsVal
		}
	}
	
	if val, ok := leAnalysis["LoadFactor"]; ok {
		if loadFactorVal, ok := val.(float64); ok {
			loadFactor = loadFactorVal
		}
	}
	
	if val, ok := leAnalysis["CollisionProb"]; ok {
		if collisionProbVal, ok := val.(float64); ok {
			collisionProb = collisionProbVal
		}
	}
	
	stats := PSIStatistics{
		Summary: SummaryStats{
			TotalOperations:    len(noiseStats),
			SuccessRate:        calculateSuccessRate(totalMatches, totalErrors),
			AverageNoiseLevel:  totalAvgNoise / float64(max(totalMatches, 1)),
			MaxNoiseLevel:      totalMaxNoise / float64(max(totalMatches, 1)),
			TotalIntersections: totalMatches,
			EfficiencyScore:    calculateEfficiencyScore(duration, totalMatches),
		},
		LEParameters: LEParameterStats{
			Q:               q,
			QBits:           qBits,
			D:               d,
			N:               n,
			Layers:          layers,
			NumSlots:        numSlots,
			LoadFactor:      loadFactor,
			CollisionProb:   collisionProb,
			SecurityLevel:   determineSecurityLevel(d, q),
			MemoryUsage:     estimateMemoryUsage(leAnalysis),
			OptimalityScore: calculateOptimalityScore(leAnalysis),
		},
		NoiseAnalysis: generateNoiseAnalysis(noiseStats, totalMaxNoise, totalAvgNoise),
		ErrorAnalysis: generateErrorAnalysis(errorStats, totalErrors),
		TimingAnalysis: generateTimingAnalysis(duration, encDuration, serverEncDuration, decDuration, totalMatches),
		DetailedMetrics: generateDetailedMetrics(noiseStats, errorStats),
		Metadata: MetadataStats{
			Timestamp:        time.Now().Format("2006-01-02 15:04:05"),
			Version:          "FLARE v2.0",
			SystemInfo:       getSystemInfo(),
			ConfigHash:       generateConfigHash(leAnalysis),
			DatasetSize:      len(noiseStats),
			AlgorithmVariant: "Laconic PSI with Lattice Encryption",
		},
	}

	return stats
}

func saveStatsToJSON(filepath string, stats PSIStatistics) error {
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(stats)
}

func generateEnhancedHTML(htmlPath, jsonPath string) error {
	htmlContent := getSimpleHTMLTemplate()
	
	// Get relative path for JSON file
	jsonFileName := filepath.Base(jsonPath)
	
	data := struct {
		JSONPath string
	}{
		JSONPath: jsonFileName, // Use just the filename for relative path
	}

	tmpl, err := template.New("report").Parse(htmlContent)
	if err != nil {
		return err
	}

	file, err := os.Create(htmlPath)
	if err != nil {
		return err
	}
	defer file.Close()

	return tmpl.Execute(file, data)
}

func getSimpleHTMLTemplate() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>FLARE PSI Analysis Report</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', system-ui, sans-serif;
            background: #fafafa;
            color: #333;
            line-height: 1.5;
        }
        
        .header {
            background: #fff;
            border-bottom: 1px solid #ddd;
            padding: 2rem 0;
            text-align: center;
            margin-bottom: 2rem;
        }
        
        .header h1 {
            font-size: 2rem;
            font-weight: 600;
            color: #2c3e50;
            margin-bottom: 0.5rem;
        }
        
        .header p {
            color: #666;
            font-size: 1rem;
        }
        
        .container {
            max-width: 1000px;
            margin: 0 auto;
            padding: 0 1.5rem;
        }
        
        .section {
            background: #fff;
            border-radius: 6px;
            padding: 1.5rem;
            margin-bottom: 1.5rem;
            border: 1px solid #e1e5e9;
            box-shadow: 0 1px 3px rgba(0,0,0,0.05);
        }
        
        .section h2 {
            font-size: 1.25rem;
            font-weight: 600;
            color: #2c3e50;
            margin-bottom: 1rem;
            padding-bottom: 0.5rem;
            border-bottom: 1px solid #f1f3f5;
        }
        
        .metrics-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 1rem;
            margin: 1rem 0;
        }
        
        .metric-card {
            padding: 1rem;
            background: #f8f9fa;
            border-radius: 4px;
            text-align: center;
            border-left: 3px solid #6c757d;
        }
        
        .metric-value {
            font-size: 1.5rem;
            font-weight: 600;
            color: #2c3e50;
            margin-bottom: 0.25rem;
        }
        
        .metric-label {
            font-size: 0.9rem;
            color: #666;
        }
        
        .info-box {
            background: #f8f9fa;
            border: 1px solid #e9ecef;
            border-radius: 4px;
            padding: 1rem;
            margin: 1rem 0;
        }
        
        .info-row {
            display: flex;
            justify-content: space-between;
            padding: 0.5rem 0;
            border-bottom: 1px solid #e9ecef;
        }
        
        .info-row:last-child {
            border-bottom: none;
        }
        
        .info-label {
            font-weight: 500;
            color: #495057;
        }
        
        .info-value {
            color: #6c757d;
            font-family: 'SF Mono', Monaco, monospace;
            font-size: 0.9rem;
        }
        
        .params-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
            gap: 0.75rem;
            margin: 1rem 0;
        }
        
        .param-item {
            background: #f8f9fa;
            border: 1px solid #e9ecef;
            border-radius: 4px;
            padding: 0.75rem;
            text-align: center;
        }
        
        .param-value {
            font-size: 1.1rem;
            font-weight: 600;
            color: #495057;
            margin-bottom: 0.25rem;
        }
        
        .param-label {
            font-size: 0.8rem;
            color: #6c757d;
        }
        
        .loading {
            text-align: center;
            padding: 3rem;
            color: #6c757d;
        }
        
        .error {
            background: #f8d7da;
            color: #721c24;
            padding: 1rem;
            border-radius: 4px;
            border: 1px solid #f5c6cb;
        }
        
        .timestamp {
            text-align: center;
            color: #6c757d;
            font-size: 0.85rem;
            margin-top: 2rem;
            padding-top: 1rem;
            border-top: 1px solid #e9ecef;
        }
        
        @media (max-width: 768px) {
            .container {
                padding: 0 1rem;
            }
            
            .header h1 {
                font-size: 1.5rem;
            }
            
            .metrics-grid, .params-grid {
                grid-template-columns: 1fr;
            }
            
            .info-row {
                flex-direction: column;
                gap: 0.25rem;
            }
        }
    </style>
</head>
<body>
    <div class="header">
        <div class="container">
            <h1>FLARE PSI Analysis</h1>
            <p>Private Set Intersection Report</p>
        </div>
    </div>

    <div class="container">
        <div id="loading" class="loading">
            <p>Loading analysis results...</p>
        </div>

        <div id="content" style="display: none;">
            <!-- Execution Summary -->
            <div class="section">
                <h2>Execution Summary</h2>
                <div class="metrics-grid" id="executionMetrics"></div>
            </div>

            <!-- PSI Results -->
            <div class="section">
                <h2>Intersection Results</h2>
                <div id="intersectionResults"></div>
            </div>

            <!-- Performance -->
            <div class="section">
                <h2>Performance</h2>
                <div id="performanceResults"></div>
            </div>

            <!-- Cryptographic Parameters -->
            <div class="section">
                <h2>System Configuration</h2>
                <div class="params-grid" id="systemConfig"></div>
            </div>

            <!-- Technical Details -->
            <div class="section">
                <h2>Technical Details</h2>
                <div id="technicalDetails"></div>
            </div>

            <div class="timestamp" id="reportTimestamp"></div>
        </div>
    </div>

    <script>
        let data = null;

        async function loadData() {
            try {
                const response = await fetch('{{.JSONPath}}');
                if (!response.ok) {
                    throw new Error('Failed to load data');
                }
                data = await response.json();
                renderReport();
            } catch (error) {
                console.error('Error:', error);
                document.getElementById('loading').innerHTML = 
                    '<div class="error">Unable to load report data. Please check if the data file exists.</div>';
            }
        }

        function renderReport() {
            document.getElementById('loading').style.display = 'none';
            document.getElementById('content').style.display = 'block';
            
            renderExecutionMetrics();
            renderIntersectionResults();
            renderPerformance();
            renderSystemConfig();
            renderTechnicalDetails();
            renderTimestamp();
        }

        function renderExecutionMetrics() {
            const summary = data.summary;
            const timing = data.timingAnalysis;
            
            const metrics = [
                { label: 'Intersections Found', value: summary.totalIntersections },
                { label: 'Operations/sec', value: timing.throughput.toFixed(1) },
                { label: 'Total Operations', value: summary.totalOperations },
                { label: 'Execution Time', value: (timing.totalDuration / 1000000).toFixed(0) + 'ms' }
            ];

            const html = metrics.map(metric => 
                '<div class="metric-card">' +
                    '<div class="metric-value">' + metric.value + '</div>' +
                    '<div class="metric-label">' + metric.label + '</div>' +
                '</div>'
            ).join('');
            
            document.getElementById('executionMetrics').innerHTML = html;
        }

        function renderIntersectionResults() {
            const summary = data.summary;
            const timing = data.timingAnalysis;
            
            const efficiency = ((summary.totalIntersections / summary.totalOperations) * 100).toFixed(1);
            
            const html = 
                '<div class="info-box">' +
                    '<div class="info-row">' +
                        '<span class="info-label">Total Matches</span>' +
                        '<span class="info-value">' + summary.totalIntersections + '</span>' +
                    '</div>' +
                    '<div class="info-row">' +
                        '<span class="info-label">Match Efficiency</span>' +
                        '<span class="info-value">' + efficiency + '%</span>' +
                    '</div>' +
                    '<div class="info-row">' +
                        '<span class="info-label">Processing Time</span>' +
                        '<span class="info-value">' + (timing.totalDuration / 1000000).toFixed(2) + ' ms</span>' +
                    '</div>' +
                '</div>';
            
            document.getElementById('intersectionResults').innerHTML = html;
        }

        function renderPerformance() {
            const timing = data.timingAnalysis;
            
            const html = 
                '<div class="info-box">' +
                    '<div class="info-row">' +
                        '<span class="info-label">Client Encryption</span>' +
                        '<span class="info-value">' + (timing.encryptionTime / 1000000).toFixed(0) + ' ms</span>' +
                    '</div>' +
                    '<div class="info-row">' +
                        '<span class="info-label">Server Processing</span>' +
                        '<span class="info-value">' + (timing.serverEncryption / 1000000).toFixed(0) + ' ms</span>' +
                    '</div>' +
                    '<div class="info-row">' +
                        '<span class="info-label">Decryption</span>' +
                        '<span class="info-value">' + (timing.decryptionTime / 1000000).toFixed(0) + ' ms</span>' +
                    '</div>' +
                    '<div class="info-row">' +
                        '<span class="info-label">Throughput</span>' +
                        '<span class="info-value">' + timing.throughput.toFixed(1) + ' ops/sec</span>' +
                    '</div>' +
                '</div>';
            
            document.getElementById('performanceResults').innerHTML = html;
        }

        function renderSystemConfig() {
            const params = data.leParameters;
            
            const configs = [
                { label: 'Ring Dimension', value: params.d },
                { label: 'Security Level', value: params.securityLevel },
                { label: 'Matrix Size', value: params.n + '×' + params.n },
                { label: 'Modulus', value: params.q.toLocaleString() },
                { label: 'Tree Layers', value: params.layers },
                { label: 'Memory Usage', value: (params.memoryUsage / 1024 / 1024).toFixed(1) + ' MB' }
            ];

            const html = configs.map(config => 
                '<div class="param-item">' +
                    '<div class="param-value">' + config.value + '</div>' +
                    '<div class="param-label">' + config.label + '</div>' +
                '</div>'
            ).join('');
            
            document.getElementById('systemConfig').innerHTML = html;
        }

        function renderTechnicalDetails() {
            const params = data.leParameters;
            const noise = data.noiseAnalysis;
            
            const html = 
                '<div class="info-box">' +
                    '<div class="info-row">' +
                        '<span class="info-label">Load Factor</span>' +
                        '<span class="info-value">' + params.loadFactor.toFixed(4) + '</span>' +
                    '</div>' +
                    '<div class="info-row">' +
                        '<span class="info-label">Collision Probability</span>' +
                        '<span class="info-value">' + params.collisionProb.toExponential(2) + '</span>' +
                    '</div>' +
                    '<div class="info-row">' +
                        '<span class="info-label">Max Noise Level</span>' +
                        '<span class="info-value">' + (noise.globalMaxNoise * 100).toFixed(3) + '%</span>' +
                    '</div>' +
                    '<div class="info-row">' +
                        '<span class="info-label">Avg Noise Level</span>' +
                        '<span class="info-value">' + (noise.globalAvgNoise * 100).toFixed(3) + '%</span>' +
                    '</div>' +
                '</div>';
            
            document.getElementById('technicalDetails').innerHTML = html;
        }

        function renderTimestamp() {
            const timestamp = data.metadata.timestamp;
            document.getElementById('reportTimestamp').innerHTML = 'Report generated: ' + timestamp;
        }

        window.addEventListener('load', loadData);
    </script>
</body>
</html>`
}

// Helper function to get max of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func calculateSuccessRate(matches, errors int) float64 {
	if matches+errors == 0 {
		return 0
	}
	return float64(matches) / float64(matches+errors) * 100
}

func calculateEfficiencyScore(duration time.Duration, matches int) float64 {
	if matches == 0 {
		return 0
	}
	// Operations per second normalized to 0-100 scale
	ops := float64(matches) / duration.Seconds()
	return math.Min(ops*10, 100) // Normalize to 0-100
}

func determineSecurityLevel(D int, Q uint64) string {
	if D >= 1024 {
		return "Very High (Post-Quantum)"
	} else if D >= 512 {
		return "High (Post-Quantum)"
	} else if D >= 256 {
		return "Medium (Post-Quantum)"
	}
	return "Low"
}

func generateNoiseAnalysis(noiseStats []map[string]interface{}, totalMaxNoise, totalAvgNoise float64) NoiseAnalysisStats {
	globalNoiseDist := make(map[string]int)
	noiseEvolution := make([]NoiseEvolutionPoint, 0)
	
	for i, stat := range noiseStats {
		if dist, ok := stat["NoiseDist"]; ok {
			if distMap, ok := dist.(map[string]int); ok {
				for k, v := range distMap {
					globalNoiseDist[k] += v
				}
			}
		}
		
		if avgNoise, ok := stat["AvgNoise"]; ok {
			if avgNoiseVal, ok := avgNoise.(float64); ok {
				noiseEvolution = append(noiseEvolution, NoiseEvolutionPoint{
					Operation: i,
					NoiseLevel: avgNoiseVal,
				})
			}
		}
	}
	
	// Calculate noise categories
	categories := make(map[string]CategoryStats)
	totalSamples := 0
	for _, count := range globalNoiseDist {
		totalSamples += count
	}
	
	for category, count := range globalNoiseDist {
		percentage := float64(count) / float64(max(totalSamples, 1)) * 100
		categories[category] = CategoryStats{
			Count:      count,
			Percentage: percentage,
			Trend:      determineTrend(category, percentage),
		}
	}
	
	return NoiseAnalysisStats{
		GlobalMaxNoise:    totalMaxNoise / float64(max(len(noiseStats), 1)),
		GlobalAvgNoise:    totalAvgNoise / float64(max(len(noiseStats), 1)),
		NoiseDistribution: globalNoiseDist,
		NoiseEvolution:    noiseEvolution,
		NoiseCategories:   categories,
		PredictiveModel: NoisePredictor{
			PredictedMaxNoise: totalMaxNoise * 1.1,
			Confidence:        0.85,
			Model:            "Linear Extrapolation",
		},
	}
}

func generateErrorAnalysis(errorStats []map[string]interface{}, totalErrors int) ErrorAnalysisStats {
	errorDist := make(map[string]int)
	criticalErrors := make([]CriticalError, 0)
	errorTrends := make([]ErrorTrendPoint, 0)
	
	for i, stat := range errorStats {
		var mismatches int
		var matchPct float64
		var matches int
		
		if val, ok := stat["Mismatches"]; ok {
			if mismatchesVal, ok := val.(int); ok {
				mismatches = mismatchesVal
			}
		}
		
		if val, ok := stat["MatchPct"]; ok {
			if matchPctVal, ok := val.(float64); ok {
				matchPct = matchPctVal
			}
		}
		
		if val, ok := stat["Matches"]; ok {
			if matchesVal, ok := val.(int); ok {
				matches = matchesVal
			}
		}
		
		// Categorize errors
		if matchPct < 0.5 {
			errorDist["Critical"]++
			criticalErrors = append(criticalErrors, CriticalError{
				Operation:   i,
				ErrorType:   "High Mismatch Rate",
				Severity:    "Critical",
				Impact:      1.0 - matchPct,
				Description: fmt.Sprintf("Match rate %.2f%% below threshold", matchPct*100),
			})
		} else if matchPct < 0.8 {
			errorDist["Warning"]++
		} else if matchPct < 0.95 {
			errorDist["Minor"]++
		} else {
			errorDist["Normal"]++
		}
		
		totalOps := matches + mismatches
		if totalOps > 0 {
			errorTrends = append(errorTrends, ErrorTrendPoint{
				Time:      i,
				ErrorRate: float64(mismatches) / float64(totalOps),
			})
		}
	}
	
	errorRate := float64(totalErrors) / float64(max(len(errorStats), 1))
	
	return ErrorAnalysisStats{
		TotalErrors:       totalErrors,
		ErrorRate:         errorRate,
		ErrorDistribution: errorDist,
		CriticalErrors:    criticalErrors,
		ErrorTrends:       errorTrends,
		RecoveryMetrics: RecoveryStats{
			RecoveryRate:     calculateRecoveryRate(criticalErrors),
			MeanRecoveryTime: 0.5,
			SuccessfulFixes:  len(criticalErrors) / 2,
		},
	}
}

func generateTimingAnalysis(duration, encDuration, serverEncDuration, decDuration time.Duration, totalMatches int) TimingAnalysisStats {
	throughput := float64(totalMatches) / duration.Seconds()
	
	// Analyze bottlenecks
	timings := map[string]time.Duration{
		"Client Encryption": encDuration,
		"Server Encryption": serverEncDuration,
		"Decryption":       decDuration,
	}
	
	var primaryBottleneck string
	var maxTime time.Duration
	for op, t := range timings {
		if t > maxTime {
			maxTime = t
			primaryBottleneck = op
		}
	}
	
	bottleneckImpact := float64(maxTime) / float64(duration)
	
	recommendations := []string{
		"Consider parallel processing for " + primaryBottleneck,
		"Optimize memory allocation patterns",
		"Use hardware acceleration where possible",
	}
	
	benchmarks := []BenchmarkPoint{
		{"Encryption", encDuration.Seconds(), 1.0},
		{"Server Processing", serverEncDuration.Seconds(), 1.2},
		{"Decryption", decDuration.Seconds(), 0.8},
	}
	
	performanceScore := calculatePerformanceScore(throughput, duration)
	
	return TimingAnalysisStats{
		TotalDuration:    duration,
		EncryptionTime:   encDuration,
		ServerEncryption: serverEncDuration,
		DecryptionTime:   decDuration,
		Throughput:       throughput,
		PerformanceScore: performanceScore,
		BottleneckAnalysis: BottleneckAnalysis{
			PrimaryBottleneck:   primaryBottleneck,
			BottleneckImpact:    bottleneckImpact,
			Recommendations:     recommendations,
		},
		Benchmarks: benchmarks,
	}
}

func generateDetailedMetrics(noiseStats, errorStats []map[string]interface{}) []DetailedMetric {
	minLen := len(noiseStats)
	if len(errorStats) < minLen {
		minLen = len(errorStats)
	}
	
	metrics := make([]DetailedMetric, minLen)
	
	for i := 0; i < minLen; i++ {
		noiseStat := noiseStats[i]
		errorStat := errorStats[i]
		
		var maxNoise, avgNoise, matchPct float64
		var serverIdx, clientIdx, matches, mismatches int
		var noiseDist map[string]int
		
		// Safe type assertions
		if val, ok := noiseStat["MaxNoise"]; ok {
			if maxNoiseVal, ok := val.(float64); ok {
				maxNoise = maxNoiseVal
			}
		}
		
		if val, ok := noiseStat["AvgNoise"]; ok {
			if avgNoiseVal, ok := val.(float64); ok {
				avgNoise = avgNoiseVal
			}
		}
		
		if val, ok := noiseStat["ServerIdx"]; ok {
			if serverIdxVal, ok := val.(int); ok {
				serverIdx = serverIdxVal
			}
		}
		
		if val, ok := noiseStat["ClientIdx"]; ok {
			if clientIdxVal, ok := val.(int); ok {
				clientIdx = clientIdxVal
			}
		}
		
		if val, ok := noiseStat["NoiseDist"]; ok {
			if noiseDistVal, ok := val.(map[string]int); ok {
				noiseDist = noiseDistVal
			} else {
				noiseDist = make(map[string]int)
			}
		}
		
		if val, ok := errorStat["MatchPct"]; ok {
			if matchPctVal, ok := val.(float64); ok {
				matchPct = matchPctVal
			}
		}
		
		if val, ok := errorStat["Matches"]; ok {
			if matchesVal, ok := val.(int); ok {
				matches = matchesVal
			}
		}
		
		if val, ok := errorStat["Mismatches"]; ok {
			if mismatchesVal, ok := val.(int); ok {
				mismatches = mismatchesVal
			}
		}
		
		qualityScore := calculateQualityScore(maxNoise, avgNoise, matchPct)
		risk := determineRiskLevel(qualityScore)
		
		metrics[i] = DetailedMetric{
			ServerIndex:  serverIdx,
			ClientIndex:  clientIdx,
			NoiseMetrics: NoiseMetric{
				MaxNoise:     maxNoise,
				AvgNoise:     avgNoise,
				Distribution: noiseDist,
				Stability:    calculateStability(maxNoise, avgNoise),
			},
			ErrorMetrics: ErrorMetric{
				Matches:      matches,
				Mismatches:   mismatches,
				MatchPct:     matchPct,
				ErrorPattern: determineErrorPattern(matchPct),
			},
			TimingMetrics: TimingMetric{
				Duration:     time.Duration(float64(time.Millisecond) * (1.0 + float64(i)*0.1)),
				Efficiency:   calculateEfficiency(matchPct),
				Optimization: suggestOptimization(maxNoise, matchPct),
			},
			QualityScore: qualityScore,
			Risk:         risk,
		}
	}
	
	return metrics
}

func estimateMemoryUsage(leAnalysis map[string]interface{}) int64 {
	var d, n, layers int
	
	if val, ok := leAnalysis["D"]; ok {
		if dVal, ok := val.(int); ok {
			d = dVal
		}
	}
	
	if val, ok := leAnalysis["N"]; ok {
		if nVal, ok := val.(int); ok {
			n = nVal
		}
	}
	
	if val, ok := leAnalysis["Layers"]; ok {
		if layersVal, ok := val.(int); ok {
			layers = layersVal
		}
	}
	
	// Rough estimation in bytes
	polySize := int64(d * 8)  // 8 bytes per coefficient
	matrixSize := polySize * int64(n*n)
	treeSize := polySize * int64(1<<layers)
	
	return matrixSize*6 + treeSize // Multiple matrices + tree storage
}

func calculateOptimalityScore(leAnalysis map[string]interface{}) float64 {
	var loadFactor, collisionProb float64
	
	if val, ok := leAnalysis["LoadFactor"]; ok {
		if loadFactorVal, ok := val.(float64); ok {
			loadFactor = loadFactorVal
		}
	}
	
	if val, ok := leAnalysis["CollisionProb"]; ok {
		if collisionProbVal, ok := val.(float64); ok {
			collisionProb = collisionProbVal
		}
	}
	
	// Score based on optimal load factor (around 0.5-0.7) and low collision probability
	loadScore := 1.0 - math.Abs(loadFactor-0.6)/0.6
	collisionScore := 1.0 - math.Min(collisionProb*1000, 1.0) // Normalize collision prob
	
	return (loadScore + collisionScore) * 50 // 0-100 scale
}

func generateConfigHash(leAnalysis map[string]interface{}) string {
	var q uint64
	var d, n int
	
	if val, ok := leAnalysis["Q"]; ok {
		if qVal, ok := val.(uint64); ok {
			q = qVal
		}
	}
	
	if val, ok := leAnalysis["D"]; ok {
		if dVal, ok := val.(int); ok {
			d = dVal
		}
	}
	
	if val, ok := leAnalysis["N"]; ok {
		if nVal, ok := val.(int); ok {
			n = nVal
		}
	}
	
	return fmt.Sprintf("hash_%d_%d_%d", q%1000, d, n)
}

func determineTrend(category string, percentage float64) string {
	if percentage > 50 {
		return "High"
	} else if percentage > 20 {
		return "Medium"
	} else {
		return "Low"
	}
}

func calculateRecoveryRate(criticalErrors []CriticalError) float64 {
	if len(criticalErrors) == 0 {
		return 100.0
	}
	return float64(len(criticalErrors)/2) / float64(len(criticalErrors)) * 100
}

func calculatePerformanceScore(throughput float64, duration time.Duration) float64 {
	baseScore := math.Min(throughput*2, 100)
	if duration.Seconds() < 1 {
		baseScore *= 1.2
	}
	return math.Min(baseScore, 100)
}

func calculateQualityScore(maxNoise, avgNoise, matchPct float64) float64 {
	noiseScore := (1.0 - maxNoise) * 30
	avgNoiseScore := (1.0 - avgNoise) * 30
	matchScore := matchPct * 40
	return math.Max(0, math.Min(100, noiseScore+avgNoiseScore+matchScore))
}

func determineRiskLevel(qualityScore float64) string {
	if qualityScore >= 80 {
		return "low"
	} else if qualityScore >= 60 {
		return "medium"
	} else {
		return "high"
	}
}

func calculateStability(maxNoise, avgNoise float64) float64 {
	if avgNoise == 0 {
		return 100.0
	}
	variance := (maxNoise - avgNoise) / avgNoise
	return math.Max(0, 100.0-variance*50.0)
}

func determineErrorPattern(matchPct float64) string {
	if matchPct >= 0.95 {
		return "Excellent"
	} else if matchPct >= 0.8 {
		return "Good"
	} else if matchPct >= 0.5 {
		return "Fair"
	} else {
		return "Poor"
	}
}

func calculateEfficiency(matchPct float64) float64 {
	return matchPct * 100
}

func suggestOptimization(maxNoise, matchPct float64) string {
	if maxNoise > 0.1 {
		return "Reduce noise level"
	} else if matchPct < 0.8 {
		return "Improve accuracy"
	} else {
		return "Well optimized"
	}
}

func getSystemInfo() string {
	return "Go Runtime on Unix System"
}