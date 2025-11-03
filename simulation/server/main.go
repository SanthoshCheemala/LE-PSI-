package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/SanthoshCheemala/LE-PSI/pkg/psi"
	"github.com/SanthoshCheemala/LE-PSI/utils"
)

var (
	serverCtx     *psi.ServerInitContext
	serverData    []interface{}
	mu            sync.RWMutex
	serverStarted time.Time
	requestCount  int
)

type StatusResponse struct {
	Status     string    `json:"status"`
	Message    string    `json:"message"`
	DataSize   int       `json:"data_size"`
	Uptime     string    `json:"uptime"`
	Requests   int       `json:"requests_handled"`
	ServerTime time.Time `json:"server_time"`
}

type ParamsResponse struct {
	Params    *psi.SerializableParams `json:"params"`
	Message   string                   `json:"message"`
	Timestamp time.Time                `json:"timestamp"`
}

type IntersectionRequest struct {
	Ciphertexts  []psi.Cxtx `json:"ciphertexts"`
	ClientHashes []uint64   `json:"client_hashes"`
	ClientID     string     `json:"client_id"`
}

type IntersectionResponse struct {
	Matches        []uint64  `json:"matches"`
	Count          int       `json:"count"`
	Message        string    `json:"message"`
	ProcessingTime string    `json:"processing_time"`
	Timestamp      time.Time `json:"timestamp"`
}

const serverDataFilePath = "../../data/server_data.json"

func main() {
	fmt.Println("=== LE-PSI Server Simulation ===")

	// Load server dataset (generic JSON array)
	items, err := loadArrayFromJSON(serverDataFilePath)
	if err != nil {
		log.Fatalf("failed to load server dataset from %s: %v", serverDataFilePath, err)
	}
	fmt.Printf("Server dataset size: %d items\n", len(items))

	// Use data as-is for PSI utils
	serverData = items

	serializedData, err := utils.PrepareDataForPSI(serverData)
	if err != nil {
		log.Fatal("Data preparation failed:", err)
	}

	serverHashes := utils.HashDataPoints(serializedData)

	ctx, err := psi.ServerInitialize(serverHashes, "simulation_server.db")
	if err != nil {
		log.Fatal("Server initialization failed:", err)
	}

	serverCtx = ctx
	serverStarted = time.Now()

	fmt.Println("Server initialized successfully")
	fmt.Println("Server listening on http://localhost:8080")

	http.HandleFunc("/api/status", handleStatus)
	http.HandleFunc("/api/params", handleGetParams)
	http.HandleFunc("/api/intersect", handleIntersection)

	log.Fatal(http.ListenAndServe(":8080", nil))
}

// loadArrayFromJSON loads a generic JSON array ([]interface{})
func loadArrayFromJSON(path string) ([]interface{}, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var items []interface{}
	if err := json.Unmarshal(b, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	mu.RLock()
	response := StatusResponse{
		Status:     "running",
		Message:    "Server is healthy",
		DataSize:   len(serverData),
		Uptime:     time.Since(serverStarted).String(),
		Requests:   requestCount,
		ServerTime: time.Now(),
	}
	mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleGetParams(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	requestCount++
	mu.Unlock()

	pp, msg, le := psi.GetPublicParameters(serverCtx)
	serializedParams := psi.SerializeParameters(pp, msg, le)

	response := ParamsResponse{
		Params:    serializedParams,
		Message:   "Parameters retrieved successfully",
		Timestamp: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	log.Printf("Sending parameters to [%s]", r.RemoteAddr)
	json.NewEncoder(w).Encode(response)
}

func handleIntersection(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	requestCount++
	mu.Unlock()

	startTime := time.Now()
	var request IntersectionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request", 400)
		log.Printf("❌ Failed to decode request: %v", err)
		return
	}

	clientID := request.ClientID
	if clientID == "" {
		clientID = r.RemoteAddr
	}

	log.Printf("Received %d ciphertexts from [%s]", len(request.Ciphertexts), clientID)

	matches, err := psi.DetectIntersectionWithContext(serverCtx, request.Ciphertexts)
	if err != nil {
		http.Error(w, "Detection failed", 500)
		log.Printf("Error: %v", err)
		return
	}

	response := IntersectionResponse{
		Matches:        matches,
		Count:          len(matches),
		Message:        fmt.Sprintf("Found %d matches", len(matches)),
		ProcessingTime: time.Since(startTime).String(),
		Timestamp:      time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	log.Printf("Found %d matches for [%s] in %v\n", len(matches), clientID, time.Since(startTime))
}
