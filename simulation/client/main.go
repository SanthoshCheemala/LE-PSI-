package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	psi "github.com/SanthoshCheemala/FLARE/internal/crypto/PSI"
	"github.com/SanthoshCheemala/FLARE/utils"
)

const serverURL = "http://localhost:8080"

type StatusResponse struct {
	Status     string `json:"status"`
	Message    string `json:"message"`
	DataSize   int    `json:"data_size"`
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
	Matches []uint64 `json:"matches"`
	Count   int      `json:"count"`
	Message string   `json:"message"`
}

func main() {
	fmt.Println("=== FLARE PSI Client Simulation ===")
	fmt.Println()

	clientData := []interface{}{
		"alice@example.com",
		"bob@example.com",
		"charlie@example.com",
		"zoe@example.com",
		"adam@example.com",
		"kate@example.com",
		"leo@example.com",
		"unknown@test.com",
	}

	fmt.Printf("📊 Client dataset: %d items\n", len(clientData))
	fmt.Println()

	fmt.Println("🔍 Step 1: Checking server...")
	if !checkServerStatus() {
		log.Fatal("❌ Server unavailable")
	}
	fmt.Println("✅ Server is healthy")
	fmt.Println()

	fmt.Println("🔍 Step 2: Getting PSI parameters...")
	paramsResp := requestParameters()
	if paramsResp == nil {
		log.Fatal("❌ Failed to get parameters")
	}
	fmt.Printf("✅ Got params (D=%d, Layers=%d)\n", paramsResp.Params.D, paramsResp.Params.Layers)
	fmt.Println()

	fmt.Println("🔍 Step 3: Preparing and hashing data...")
	serializedData, err := utils.PrepareDataForPSI(clientData)
	if err != nil {
		log.Fatal("❌ Data preparation failed:", err)
	}
	
	clientHashes := utils.HashDataPoints(serializedData)
	fmt.Printf("✅ Prepared and hashed %d items\n", len(clientHashes))
	fmt.Println()

	fmt.Println("🔍 Step 4: Deserializing parameters...")
	pp, msg, le, err := psi.DeserializeParameters(paramsResp.Params)
	if err != nil {
		log.Fatal("❌ Parameter deserialization failed:", err)
	}
	fmt.Println("✅ Parameters deserialized")
	fmt.Println()

	fmt.Println("🔍 Step 5: Encrypting data...")
	start := time.Now()
	ciphertexts := psi.ClientEncrypt(clientHashes, pp, msg, le)
	fmt.Printf("✅ Encrypted in %v\n", time.Since(start))
	fmt.Println()

	fmt.Println("🔍 Step 6: Requesting intersection...")
	matches := requestIntersection(ciphertexts, clientHashes)
	fmt.Println()

	fmt.Println("📊 Results:")
	fmt.Printf("   Matches: %d/%d\n", len(matches), len(clientData))
	if len(matches) > 0 {
		hashMap := make(map[uint64]string)
		for i, h := range clientHashes {
			hashMap[h] = serializedData[i]
		}
		fmt.Println("🎯 Matched:")
		for _, h := range matches {
			if data, ok := hashMap[h]; ok {
				fmt.Printf("   - %s\n", data)
			}
		}
	}
}

func checkServerStatus() bool {
	resp, err := http.Get(serverURL + "/api/status")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	var status StatusResponse
	json.NewDecoder(resp.Body).Decode(&status)
	return status.Status == "running"
}

func requestParameters() *ParamsResponse {
	resp, err := http.Get(serverURL + "/api/params")
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	var params ParamsResponse
	json.NewDecoder(resp.Body).Decode(&params)
	return &params
}

func requestIntersection(ciphertexts []psi.Cxtx, hashes []uint64) []uint64 {
	req := IntersectionRequest{
		Ciphertexts: ciphertexts,
		ClientHashes: hashes,
		ClientID: "test-client",
	}
	jsonData, _ := json.Marshal(req)
	resp, err := http.Post(serverURL+"/api/intersect", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	var result IntersectionResponse
	json.NewDecoder(resp.Body).Decode(&result)
	fmt.Printf("✅ Found %d matches\n", result.Count)
	return result.Matches
}
