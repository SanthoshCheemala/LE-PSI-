package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/SanthoshCheemala/LE-PSI/pkg/psi"
	"github.com/SanthoshCheemala/LE-PSI/utils"
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

const clientDataFilePath = "../../data/client_data.json"

type CustomerRecord struct {
	CustomerID          string  `json:"customer_id"`
	Name                string  `json:"name"`
	DOB                 string  `json:"dob"`
	Country             string  `json:"country"`
	Email               string  `json:"email"`
	Phone               string  `json:"phone"`
	Address             string  `json:"address"`
	AccountNumber       string  `json:"account_number"`
	AccountBalance      float64 `json:"account_balance"`
	AccountOpened       string  `json:"account_opened"`
	MonthlyTransactions int     `json:"monthly_transactions"`
	AvgTransactionAmount float64 `json:"avg_transaction_amount"`
	PSIKey              string  `json:"psi_key"`
	PSIHash             string  `json:"psi_hash"`
	IsMatch             bool    `json:"is_match"`
}

func main() {
	fmt.Println("=== LE-PSI Client Simulation ===")

	// Load client dataset with proper structure
	customers, err := loadCustomerRecords(clientDataFilePath)
	if err != nil {
		log.Fatalf("failed to load client dataset (%s): %v", clientDataFilePath, err)
	}

	fmt.Printf("Client dataset: %d items\n", len(customers))

	fmt.Println("Checking server status...")
	if !checkServerStatus() {
		log.Fatal("Server unavailable")
	}
	fmt.Println("Server is healthy")

	fmt.Println("Getting PSI parameters...")
	paramsResp := requestParameters()
	if paramsResp == nil {
		log.Fatal("Failed to get parameters")
	}
	fmt.Printf("Parameters received (D=%d, Layers=%d)\n", paramsResp.Params.D, paramsResp.Params.Layers)

	fmt.Println("Preparing and hashing data...")
	// Extract PSI keys for hashing (matching generator logic)
	psiKeys := make([]string, len(customers))
	for i, customer := range customers {
		psiKeys[i] = customer.PSIKey
	}
	
	clientHashes := utils.HashDataPoints(psiKeys)
	fmt.Printf("Prepared and hashed %d items\n", len(clientHashes))
	
	// Store client data for display
	clientData := make([]interface{}, len(customers))
	for i, c := range customers {
		clientData[i] = c
	}

	fmt.Println("Deserializing parameters...")
	pp, msg, le, err := psi.DeserializeParameters(paramsResp.Params)
	if err != nil {
		log.Fatal("Parameter deserialization failed:", err)
	}
	fmt.Println("Parameters deserialized")

	fmt.Println("Encrypting data...")
	start := time.Now()
	ciphertexts := psi.ClientEncrypt(clientHashes, pp, msg, le)
	fmt.Printf("Encrypted in %v\n", time.Since(start))

	fmt.Println("Requesting intersection...")
	matches := requestIntersection(ciphertexts, clientHashes)

	fmt.Println("\n============================================================")
	fmt.Println("RESULTS:")
	fmt.Printf("Matches found: %d/%d (%.2f%%)\n", len(matches), len(customers), float64(len(matches))/float64(len(customers))*100)
	fmt.Println("============================================================")
	
	if len(matches) > 0 {
		hashMap := make(map[uint64]*CustomerRecord)
		for i, h := range clientHashes {
			hashMap[h] = &customers[i]
		}
		fmt.Println("\n🚨 MATCHED CUSTOMERS (Sanctions Hits):")
		for i, h := range matches {
			if customer, ok := hashMap[h]; ok {
				fmt.Printf("\n  Match #%d:\n", i+1)
				fmt.Printf("    Customer ID: %s\n", customer.CustomerID)
				fmt.Printf("    Name: %s\n", customer.Name)
				fmt.Printf("    DOB: %s\n", customer.DOB)
				fmt.Printf("    Country: %s\n", customer.Country)
				fmt.Printf("    Account: %s\n", customer.AccountNumber)
				fmt.Printf("    Balance: $%.2f\n", customer.AccountBalance)
				fmt.Printf("    PSI Key: %s\n", customer.PSIKey)
			}
		}
		fmt.Println("\n============================================================")
	} else {
		fmt.Println("\n✅ No matches found - All customers are clean")
		fmt.Println("============================================================")
	}
}

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

// loadCustomerRecords loads customer records from JSON
func loadCustomerRecords(path string) ([]CustomerRecord, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var customers []CustomerRecord
	if err := json.Unmarshal(b, &customers); err != nil {
		return nil, err
	}
	return customers, nil
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
	fmt.Printf("Found %d matches\n", result.Count)
	return result.Matches
}
