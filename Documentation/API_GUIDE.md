# FLARE API Guide - Integration for Distributed Systems

This guide explains how to integrate FLARE PSI into your distributed applications, microservices, or REST APIs.

## Table of Contents

1. [Overview](#overview)
2. [Architecture Patterns](#architecture-patterns)
3. [Server-Side Integration](#server-side-integration)
4. [Client-Side Integration](#client-side-integration)
5. [Network Protocol](#network-protocol)
6. [Complete Examples](#complete-examples)
7. [Best Practices](#best-practices)
8. [Troubleshooting](#troubleshooting)

---

## Overview

FLARE provides a clean, stateful API designed for distributed systems where:
- **Server** holds a private dataset and initializes once
- **Multiple clients** can query the server without server reinitialization
- **Parameters** are transmitted over the network (HTTP, gRPC, WebSocket, etc.)
- **Privacy** is maintained throughout the process

### Key Design Principles

1. **Separation of Concerns**: Crypto logic is encapsulated in the PSI package
2. **Stateful Server**: Server initializes once, serves multiple clients
3. **Stateless Client**: Client receives parameters, encrypts, sends ciphertexts
4. **Framework Agnostic**: Works with any network protocol
5. **Type Safe**: Strong typing prevents common integration errors

---

## Architecture Patterns

### Pattern 1: REST API (Recommended)

```
┌─────────────┐                           ┌─────────────┐
│   Client    │                           │   Server    │
│ Application │                           │ Application │
└──────┬──────┘                           └──────┬──────┘
       │                                         │
       │  GET /api/psi/parameters                │
       ├────────────────────────────────────────►│
       │                                         │
       │  ◄─── Public Parameters (pp, msg, LE)  │
       │                                         │
       │  [Client encrypts local data]          │
       │                                         │
       │  POST /api/psi/intersect                │
       │       Body: {ciphertexts: [...]}        │
       ├────────────────────────────────────────►│
       │                                         │
       │  ◄─── Intersection Result               │
       │       Body: {matches: [...]}            │
       │                                         │
```

### Pattern 2: Message Queue

```
┌─────────┐     ┌───────────┐     ┌─────────┐
│ Client  │────►│   Queue   │────►│ Server  │
└─────────┘     │ (RabbitMQ)│     └─────────┘
                │  (Kafka)  │
                └───────────┘
     
Messages:
1. client → queue: {type: "get_params", client_id: "..."}
2. queue → server: Process parameter request
3. server → queue: {type: "params", data: {...}}
4. queue → client: Receive parameters
5. client → queue: {type: "compute", ciphertexts: [...]}
6. queue → server: Process intersection
7. server → queue: {type: "result", matches: [...]}
8. queue → client: Receive result
```

### Pattern 3: gRPC

```protobuf
service PSIService {
    rpc GetParameters(ParameterRequest) returns (ParameterResponse);
    rpc ComputeIntersection(IntersectionRequest) returns (IntersectionResponse);
}
```

---

## Server-Side Integration

### Step 1: Import Required Packages

```go
package main

import (
    "encoding/json"
    "net/http"
    "sync"
    
    psi "github.com/SanthoshCheemala/FLARE/internal/crypto/PSI"
    "github.com/SanthoshCheemala/FLARE/utils"
)
```

### Step 2: Initialize Server State

```go
type PSIServer struct {
    ctx          *psi.ServerInitContext
    mu           sync.RWMutex
    initialized  bool
}

func NewPSIServer(dataset []interface{}, dbPath string) (*PSIServer, error) {
    // Prepare data using framework utilities
    serialized, err := utils.PrepareDataForPSI(dataset)
    if err != nil {
        return nil, err
    }
    
    // Hash serialized data
    hashes := utils.HashDataPoints(serialized)
    
    // Initialize PSI server (expensive operation - do once)
    ctx, err := psi.ServerInitialize(hashes, dbPath)
    if err != nil {
        return nil, err
    }
    
    return &PSIServer{
        ctx:         ctx,
        initialized: true,
    }, nil
}
```

### Step 3: Expose Parameters Endpoint

```go
func (s *PSIServer) HandleGetParameters(w http.ResponseWriter, r *http.Request) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    if !s.initialized {
        http.Error(w, "Server not initialized", http.StatusServiceUnavailable)
        return
    }
    
    // Extract public parameters from server context
    pp, msg, le := psi.GetPublicParameters(s.ctx)
    
    // Serialize for network transmission
    serialized := psi.SerializeParameters(pp, msg, le)
    
    // Send to client
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "params": serialized,
        "status": "success",
    })
}
```

### Step 4: Expose Intersection Endpoint

```go
type IntersectionRequest struct {
    Ciphertexts []psi.Cxtx `json:"ciphertexts"`
    ClientID    string     `json:"client_id"`
}

type IntersectionResponse struct {
    Matches []uint64 `json:"matches"`
    Count   int      `json:"count"`
}

func (s *PSIServer) HandleIntersection(w http.ResponseWriter, r *http.Request) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    // Decode client request
    var req IntersectionRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }
    
    // Compute intersection using server context
    matches, err := psi.DetectIntersectionWithContext(s.ctx, req.Ciphertexts)
    if err != nil {
        http.Error(w, "Intersection detection failed", http.StatusInternalServerError)
        return
    }
    
    // Send result
    response := IntersectionResponse{
        Matches: matches,
        Count:   len(matches),
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}
```

### Step 5: Complete Server Setup

```go
func main() {
    // Your server dataset
    serverData := []interface{}{
        "user1@example.com",
        "user2@example.com",
        "user3@example.com",
        // ... more data
    }
    
    // Initialize PSI server
    psiServer, err := NewPSIServer(serverData, "data/psi_tree.db")
    if err != nil {
        log.Fatal("Server initialization failed:", err)
    }
    
    // Register HTTP handlers
    http.HandleFunc("/api/psi/parameters", psiServer.HandleGetParameters)
    http.HandleFunc("/api/psi/intersect", psiServer.HandleIntersection)
    
    // Start server
    log.Println("PSI Server listening on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

---

## Client-Side Integration

### Step 1: Fetch Parameters from Server

```go
package main

import (
    "bytes"
    "encoding/json"
    "net/http"
    
    psi "github.com/SanthoshCheemala/FLARE/internal/crypto/PSI"
    "github.com/SanthoshCheemala/FLARE/utils"
)

func fetchParameters(serverURL string) (*psi.SerializableParams, error) {
    resp, err := http.Get(serverURL + "/api/psi/parameters")
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result struct {
        Params *psi.SerializableParams `json:"params"`
        Status string                   `json:"status"`
    }
    
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    
    return result.Params, nil
}
```

### Step 2: Prepare Client Data

```go
func prepareClientData(data []interface{}) ([]uint64, error) {
    // Serialize data using framework utility
    serialized, err := utils.PrepareDataForPSI(data)
    if err != nil {
        return nil, err
    }
    
    // Hash serialized data
    hashes := utils.HashDataPoints(serialized)
    
    return hashes, nil
}
```

### Step 3: Encrypt Client Data

```go
func encryptClientData(hashes []uint64, params *psi.SerializableParams) ([]psi.Cxtx, error) {
    // Deserialize parameters received from server
    pp, msg, le, err := psi.DeserializeParameters(params)
    if err != nil {
        return nil, err
    }
    
    // Encrypt client data using server's public parameters
    ciphertexts := psi.ClientEncrypt(hashes, pp, msg, le)
    
    return ciphertexts, nil
}
```

### Step 4: Request Intersection

```go
func requestIntersection(serverURL string, ciphertexts []psi.Cxtx, clientID string) ([]uint64, error) {
    request := map[string]interface{}{
        "ciphertexts": ciphertexts,
        "client_id":   clientID,
    }
    
    jsonData, err := json.Marshal(request)
    if err != nil {
        return nil, err
    }
    
    resp, err := http.Post(
        serverURL+"/api/psi/intersect",
        "application/json",
        bytes.NewBuffer(jsonData),
    )
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result struct {
        Matches []uint64 `json:"matches"`
        Count   int      `json:"count"`
    }
    
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    
    return result.Matches, nil
}
```

### Step 5: Complete Client Flow

```go
func main() {
    serverURL := "http://localhost:8080"
    
    // Client's private dataset
    clientData := []interface{}{
        "user1@example.com",
        "user4@example.com",
        "user5@example.com",
    }
    
    // Step 1: Fetch parameters from server
    params, err := fetchParameters(serverURL)
    if err != nil {
        log.Fatal("Failed to fetch parameters:", err)
    }
    
    // Step 2: Prepare client data
    clientHashes, err := prepareClientData(clientData)
    if err != nil {
        log.Fatal("Failed to prepare data:", err)
    }
    
    // Step 3: Encrypt client data
    ciphertexts, err := encryptClientData(clientHashes, params)
    if err != nil {
        log.Fatal("Failed to encrypt data:", err)
    }
    
    // Step 4: Request intersection
    matches, err := requestIntersection(serverURL, ciphertexts, "client-123")
    if err != nil {
        log.Fatal("Failed to compute intersection:", err)
    }
    
    // Step 5: Process results
    fmt.Printf("Found %d matches: %v\n", len(matches), matches)
}
```

---

## Network Protocol

### Request/Response Format

#### GET /api/psi/parameters

**Response:**
```json
{
  "params": {
    "pp": [[...], [...]],
    "msg": [...],
    "q": 180143985094819841,
    "d": 256,
    "n": 4,
    "layers": 9,
    "m": 232,
    "m2": 512,
    "a0ntt": [[[...]]],
    "a1ntt": [[[...]]],
    "bntt": [[[...]]],
    "gntt": [[[...]]]
  },
  "status": "success"
}
```

#### POST /api/psi/intersect

**Request:**
```json
{
  "ciphertexts": [
    {
      "c0": [[...]],
      "c1": [[...]],
      "c": [[...]],
      "d": [...]
    }
  ],
  "client_id": "client-123"
}
```

**Response:**
```json
{
  "matches": [12345, 67890],
  "count": 2
}
```

---

## Complete Examples

### Example 1: Microservice Architecture

```go
// Service 1: User Service (Server)
type UserService struct {
    psiServer *PSIServer
}

func (s *UserService) Init() error {
    users := s.fetchAllUsers() // Get from database
    psiServer, err := NewPSIServer(users, "data/users_psi.db")
    if err != nil {
        return err
    }
    s.psiServer = psiServer
    return nil
}

// Service 2: Analytics Service (Client)
type AnalyticsService struct {
    userServiceURL string
}

func (s *AnalyticsService) FindCommonUsers(campaignUsers []string) ([]string, error) {
    // Convert to interface{}
    data := make([]interface{}, len(campaignUsers))
    for i, u := range campaignUsers {
        data[i] = u
    }
    
    // Use PSI to find common users
    params, _ := fetchParameters(s.userServiceURL)
    hashes, _ := prepareClientData(data)
    ciphertexts, _ := encryptClientData(hashes, params)
    matches, _ := requestIntersection(s.userServiceURL, ciphertexts, "analytics")
    
    return s.hashesToUsers(matches), nil
}
```

### Example 2: WebSocket Real-Time PSI

```go
// Server side
func handleWebSocket(conn *websocket.Conn, psiServer *PSIServer) {
    for {
        var msg Message
        conn.ReadJSON(&msg)
        
        switch msg.Type {
        case "get_params":
            pp, msg, le := psi.GetPublicParameters(psiServer.ctx)
            serialized := psi.SerializeParameters(pp, msg, le)
            conn.WriteJSON(Response{Type: "params", Data: serialized})
            
        case "compute":
            var req IntersectionRequest
            json.Unmarshal(msg.Data, &req)
            matches, _ := psi.DetectIntersectionWithContext(psiServer.ctx, req.Ciphertexts)
            conn.WriteJSON(Response{Type: "result", Data: matches})
        }
    }
}
```

---

## Best Practices

### 1. Server Initialization

✅ **DO:**
- Initialize server once at startup
- Cache ServerInitContext in memory
- Use a persistent database path
- Handle initialization errors gracefully

❌ **DON'T:**
- Re-initialize for every client request
- Use temporary database paths
- Ignore initialization errors

### 2. Data Preprocessing

✅ **DO:**
```go
// Always use framework utilities
serialized, err := utils.PrepareDataForPSI(data)
hashes := utils.HashDataPoints(serialized)
```

❌ **DON'T:**
```go
// Don't implement custom hashing
hash := sha256.Sum256([]byte(fmt.Sprint(data)))
```

### 3. Error Handling

```go
// Comprehensive error handling
ctx, err := psi.ServerInitialize(hashes, dbPath)
if err != nil {
    log.Printf("PSI initialization failed: %v", err)
    // Return appropriate HTTP status
    http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
    return
}
```

### 4. Concurrency

```go
// Use read locks for concurrent access
func (s *PSIServer) HandleRequest(w http.ResponseWriter, r *http.Request) {
    s.mu.RLock()  // Multiple readers allowed
    defer s.mu.RUnlock()
    
    // Safe concurrent reads
    pp, msg, le := psi.GetPublicParameters(s.ctx)
    // ...
}
```

### 5. Resource Management

```go
// Proper cleanup
defer resp.Body.Close()
defer db.Close()

// Use context for timeouts
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
```

---

## Troubleshooting

### Issue 1: "Database locked" Error

**Cause**: SQLite database accessed concurrently without proper locking

**Solution**:
```go
// Use mutex for database operations
s.mu.Lock()
defer s.mu.Unlock()
psi.ServerInitialize(hashes, dbPath)
```

### Issue 2: Out of Memory

**Cause**: Large datasets consuming too much RAM

**Solution**:
- Batch processing for large datasets
- Increase system memory
- Use smaller ring dimensions (trade security for performance)

### Issue 3: Slow Performance

**Cause**: Not utilizing parallel processing

**Solution**:
```go
// Let the framework use all CPU cores automatically
// PSI functions already parallelize internally
runtime.GOMAXPROCS(runtime.NumCPU())
```

### Issue 4: Nil Pointer on Client Decrypt

**Cause**: Missing LE matrices in serialized parameters

**Solution**:
```go
// Always use framework serialization
serialized := psi.SerializeParameters(pp, msg, le)  // ✓ Includes matrices
```

---

## Performance Optimization

### 1. Connection Pooling

```go
client := &http.Client{
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 100,
        IdleConnTimeout:     90 * time.Second,
    },
}
```

### 2. Compression

```go
// Compress large parameter payloads
import "compress/gzip"

func compressJSON(data interface{}) ([]byte, error) {
    var buf bytes.Buffer
    gzWriter := gzip.NewWriter(&buf)
    json.NewEncoder(gzWriter).Encode(data)
    gzWriter.Close()
    return buf.Bytes(), nil
}
```

### 3. Caching

```go
// Cache parameters on client side
type PSIClient struct {
    paramsCache *psi.SerializableParams
    cacheExpiry time.Time
}

func (c *PSIClient) GetParameters() (*psi.SerializableParams, error) {
    if c.paramsCache != nil && time.Now().Before(c.cacheExpiry) {
        return c.paramsCache, nil
    }
    // Fetch new parameters
}
```

---

## Security Considerations

1. **Use HTTPS**: Always use TLS for network communication
2. **Rate Limiting**: Implement rate limiting to prevent abuse
3. **Authentication**: Verify client identity before processing
4. **Input Validation**: Validate all incoming data
5. **Logging**: Log security-relevant events (audit trail)

```go
// Example with authentication
func (s *PSIServer) HandleIntersection(w http.ResponseWriter, r *http.Request) {
    // Verify API key
    apiKey := r.Header.Get("X-API-Key")
    if !s.validateAPIKey(apiKey) {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    
    // Rate limiting
    if !s.rateLimiter.Allow(apiKey) {
        http.Error(w, "Too many requests", http.StatusTooManyRequests)
        return
    }
    
    // Process request...
}
```

---

## Summary

This guide covered:
- ✅ Complete server-side integration
- ✅ Complete client-side integration
- ✅ Network protocol specifications
- ✅ Best practices and common pitfalls
- ✅ Performance optimization techniques
- ✅ Security considerations

For cryptographic details, see [CONCEPTS.md](./CONCEPTS.md)

For command-line usage, see [cmd/Flare/README.md](../cmd/Flare/README.md)
