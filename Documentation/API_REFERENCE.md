# LE-PSI API Reference

Complete API documentation for the Lattice Encryption Private Set Intersection (LE-PSI) library.

## Table of Contents
- [Package: psi](#package-psi)
  - [Core Functions](#core-functions)
  - [Performance Monitoring](#performance-monitoring)
  - [Helper Functions](#helper-functions)
  - [Types & Structures](#types--structures)
- [Usage Examples](#usage-examples)

---

## Package: psi

Import: `github.com/SanthoshCheemala/LE-PSI/pkg/psi`

### Core Functions

#### 1. `SetupLEParameters`
```go
func SetupLEParameters(size int) (*LE.LE, error)
```
**Description**: Initializes Lattice Encryption parameters for PSI operations based on dataset size.

**Parameters**:
- `size` (int): Expected size of the dataset (number of elements)

**Returns**:
- `*LE.LE`: Configured lattice encryption parameters
- `error`: Error if ring dimension is unsupported or initialization fails

**Example**:
```go
leParams, err := psi.SetupLEParameters(1000)
if err != nil {
    log.Fatal(err)
}
// Configured with: Ring Dimension=256, Q=180143985094819841, N=4
// Layers automatically calculated based on size
```

**Configuration Details**:
- Ring Dimension: 256 (default)
- Modulus Q: 180143985094819841
- qBits: 58
- Matrix Dimension N: 4
- Layers: Auto-computed (log2(16 × size))
- Security Level: ~128-bit

---

#### 2. `ServerInitialize`
```go
func ServerInitialize(private_set_X []uint64, Treepath string) (*ServerInitContext, error)
```
**Description**: Prepares server-side PSI context with the server's private dataset.

**Parameters**:
- `private_set_X` ([]uint64): Server's private dataset
- `Treepath` (string): Database file path for witness tree storage

**Returns**:
- `*ServerInitContext`: Initialized server context
- `error`: Error if setup fails

**Example**:
```go
serverData := []uint64{100, 200, 300, 400, 500}
ctx, err := psi.ServerInitialize(serverData, "./data/tree.db")
if err != nil {
    log.Fatal(err)
}
defer ctx.Cleanup()
```

**What it does**:
1. Sets up lattice encryption parameters
2. Generates cryptographic keys
3. Creates witness tree structure
4. Stores tree in SQLite database

---

#### 3. `GetPublicParameters`
```go
func GetPublicParameters(ctx *ServerInitContext) (*matrix.Vector, *ring.Poly, *LE.LE)
```
**Description**: Extracts public parameters from server context to share with clients.

**Parameters**:
- `ctx` (*ServerInitContext): Server initialization context

**Returns**:
- `*matrix.Vector`: Public parameter matrix (PP)
- `*ring.Poly`: Message polynomial
- `*LE.LE`: Lattice encryption parameters

**Example**:
```go
pp, msg, le := psi.GetPublicParameters(ctx)
// Send these to client via network/API
```

---

#### 4. `SerializeParameters`
```go
func SerializeParameters(pp *matrix.Vector, msg *ring.Poly, le *LE.LE) *SerializableParams
```
**Description**: Converts public parameters to JSON-serializable format.

**Parameters**:
- `pp` (*matrix.Vector): Public parameter matrix
- `msg` (*ring.Poly): Message polynomial
- `le` (*LE.LE): Lattice encryption parameters

**Returns**:
- `*SerializableParams`: JSON-serializable parameters

**Example**:
```go
params := psi.SerializeParameters(pp, msg, le)
jsonData, _ := json.Marshal(params)
// Send jsonData to client
```

---

#### 5. `DeserializeParameters`
```go
func DeserializeParameters(params *SerializableParams) (*matrix.Vector, *ring.Poly, *LE.LE, error)
```
**Description**: Reconstructs public parameters from serialized format (client-side).

**Parameters**:
- `params` (*SerializableParams): Serialized parameters from server

**Returns**:
- `*matrix.Vector`: Reconstructed public parameter matrix
- `*ring.Poly`: Reconstructed message polynomial
- `*LE.LE`: Reconstructed lattice parameters
- `error`: Error if deserialization fails

**Example**:
```go
// Client side
var params psi.SerializableParams
json.Unmarshal(receivedData, &params)
pp, msg, le, err := psi.DeserializeParameters(&params)
```

---

#### 6. `ClientEncrypt`
```go
func ClientEncrypt(private_set_Y []uint64, pp *matrix.Vector, msg *ring.Poly, le *LE.LE) []Cxtx
```
**Description**: Encrypts client's private dataset using server's public parameters.

**Parameters**:
- `private_set_Y` ([]uint64): Client's private dataset
- `pp` (*matrix.Vector): Public parameters from server
- `msg` (*ring.Poly): Message polynomial from server
- `le` (*LE.LE): Lattice parameters from server

**Returns**:
- `[]Cxtx`: Slice of encrypted ciphertexts

**Example**:
```go
clientData := []uint64{150, 200, 250, 350}
ciphertexts := psi.ClientEncrypt(clientData, pp, msg, le)
// Send ciphertexts to server
```

**Performance**: Automatically uses parallel processing with optimal worker threads.

---

#### 7. `DetectIntersectionWithContext`
```go
func DetectIntersectionWithContext(ctx *ServerInitContext, clientCiphertexts []Cxtx) ([]uint64, error)
```
**Description**: Computes intersection between server and client datasets.

**Parameters**:
- `ctx` (*ServerInitContext): Server initialization context
- `clientCiphertexts` ([]Cxtx): Encrypted client dataset

**Returns**:
- `[]uint64`: Intersection set (common elements)
- `error`: Error if decryption or witness lookup fails

**Example**:
```go
intersection, err := psi.DetectIntersectionWithContext(ctx, ciphertexts)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Found %d common elements: %v\n", len(intersection), intersection)
```

---

### Performance Monitoring

#### 8. `NewPerformanceMonitor`
```go
func NewPerformanceMonitor() *PerformanceMonitor
```
**Description**: Creates a new performance monitor for tracking PSI metrics.

**Returns**:
- `*PerformanceMonitor`: New monitor instance

**Example**:
```go
monitor := psi.NewPerformanceMonitor()
// Use monitor throughout PSI operations
```

---

#### 9. `GetMetrics`
```go
func (pm *PerformanceMonitor) GetMetrics() map[string]interface{}
```
**Description**: Returns all performance metrics in frontend-friendly format.

**Returns**:
- `map[string]interface{}`: Comprehensive metrics

**Response Structure**:
```json
{
  "total_time_seconds": 1.234,
  "total_time_formatted": "1.234s",
  "key_gen_time_seconds": 0.234,
  "key_gen_time_formatted": "234ms",
  "key_gen_percent": 18.96,
  "hashing_time_seconds": 0.123,
  "hashing_time_formatted": "123ms",
  "hashing_percent": 9.97,
  "witness_time_seconds": 0.456,
  "witness_time_formatted": "456ms",
  "witness_percent": 36.95,
  "intersection_time_seconds": 0.421,
  "intersection_time_formatted": "421ms",
  "intersection_percent": 34.12,
  "num_workers": 8,
  "total_operations": 1000,
  "throughput_ops_per_sec": 810.37
}
```

**Example**:
```go
metrics := monitor.GetMetrics()
jsonData, _ := json.Marshal(metrics)
// Send to frontend dashboard
```

---

#### 10. `GetMemoryUsage`
```go
func (pm *PerformanceMonitor) GetMemoryUsage() map[string]interface{}
```
**Description**: Returns current runtime memory statistics.

**Returns**:
- `map[string]interface{}`: Memory metrics

**Response Structure**:
```json
{
  "alloc_mb": 45.23,
  "total_alloc_mb": 128.45,
  "sys_mb": 256.78,
  "num_gc": 12,
  "goroutines": 48
}
```

**Example**:
```go
memStats := monitor.GetMemoryUsage()
fmt.Printf("Current memory: %.2f MB\n", memStats["alloc_mb"])
```

---

#### 11. `GetThroughput`
```go
func (pm *PerformanceMonitor) GetThroughput() float64
```
**Description**: Calculates operations per second.

**Returns**:
- `float64`: Operations per second

**Example**:
```go
throughput := monitor.GetThroughput()
fmt.Printf("Throughput: %.2f ops/sec\n", throughput)
```

---

#### 12. `GetTotalTime`
```go
func (pm *PerformanceMonitor) GetTotalTime() time.Duration
```
**Description**: Returns total execution time since monitor creation.

**Returns**:
- `time.Duration`: Elapsed time

**Example**:
```go
totalTime := monitor.GetTotalTime()
fmt.Printf("Total time: %v\n", totalTime)
```

---

### Helper Functions

#### 13. `CalculateOptimalWorkers`
```go
func CalculateOptimalWorkers(datasetSize int) int
```
**Description**: Determines optimal number of worker goroutines based on dataset size and hardware.

**Parameters**:
- `datasetSize` (int): Number of elements to process

**Returns**:
- `int`: Optimal worker count (8-48)

**Example**:
```go
workers := psi.CalculateOptimalWorkers(5000)
fmt.Printf("Using %d workers\n", workers)
```

**Optimization Factors**:
- Available RAM (117 GB out of 251 GB)
- Memory per record (~35 MB)
- Hardware cores (48 on dual-socket Xeon)
- Cache optimization for large datasets

---

#### 14. `ReduceToTreeIndex`
```go
func ReduceToTreeIndex(rawHash uint64, layers int) uint64
```
**Description**: Maps hash value to valid tree index.

**Parameters**:
- `rawHash` (uint64): Raw hash value
- `layers` (int): Number of tree layers

**Returns**:
- `uint64`: Tree index in range [0, 2^layers - 1]

**Example**:
```go
treeIdx := psi.ReduceToTreeIndex(12345678, 10)
// Returns index in [0, 1023]
```

---

#### 15. `CorrectnessCheck`
```go
func CorrectnessCheck(decrypted, original *ring.Poly, le *LE.LE) bool
```
**Description**: Verifies decryption correctness (95% threshold).

**Parameters**:
- `decrypted` (*ring.Poly): Decrypted polynomial
- `original` (*ring.Poly): Original plaintext
- `le` (*LE.LE): Lattice parameters

**Returns**:
- `bool`: true if match rate >= 95%

**Example**:
```go
isCorrect := psi.CorrectnessCheck(decrypted, original, le)
if !isCorrect {
    log.Println("Decryption verification failed")
}
```

---

### Types & Structures

#### `ServerInitContext`
```go
type ServerInitContext struct {
    PublicParams    *matrix.Vector
    Message         *ring.Poly
    LEParams        *LE.LE
    PrivateKeys     []*matrix.Vector
    WitnessVectors1 [][]*matrix.Vector
    WitnessVectors2 [][]*matrix.Vector
    TreeIndices     []uint64
    OriginalHashes  []uint64
    DBPath          string
}
```
**Description**: Server-side state container for PSI operations.

---

#### `Cxtx`
```go
type Cxtx struct {
    C0 []*matrix.Vector
    C1 []*matrix.Vector
    C  *matrix.Vector
    D  *ring.Poly
}
```
**Description**: Encrypted ciphertext structure for a single data element.

---

#### `SerializableParams`
```go
type SerializableParams struct {
    PP     [][]uint64   `json:"pp"`
    Msg    []uint64     `json:"msg"`
    Q      uint64       `json:"q"`
    D      int          `json:"d"`
    N      int          `json:"n"`
    Layers int          `json:"layers"`
    M      int          `json:"m"`
    M2     int          `json:"m2"`
    A0NTT  [][][]uint64 `json:"a0ntt"`
    A1NTT  [][][]uint64 `json:"a1ntt"`
    BNTT   [][][]uint64 `json:"bntt"`
    GNTT   [][][]uint64 `json:"gntt"`
}
```
**Description**: JSON-serializable public parameters.

---

#### `PerformanceMonitor`
```go
type PerformanceMonitor struct {
    StartTime        time.Time
    KeyGenTime       time.Duration
    HashingTime      time.Duration
    WitnessTime      time.Duration
    IntersectionTime time.Duration
    TotalOperations  int
    NumWorkers       int
}
```
**Description**: Tracks PSI performance metrics.

---

## Usage Examples

### Complete PSI Workflow

#### Server Side:
```go
package main

import (
    "encoding/json"
    "log"
    "github.com/SanthoshCheemala/LE-PSI/pkg/psi"
)

func main() {
    // 1. Initialize server with dataset
    serverData := []uint64{100, 200, 300, 400, 500}
    ctx, err := psi.ServerInitialize(serverData, "./data/tree.db")
    if err != nil {
        log.Fatal(err)
    }
    defer ctx.Cleanup()
    
    // 2. Get public parameters
    pp, msg, le := psi.GetPublicParameters(ctx)
    
    // 3. Serialize and send to client
    params := psi.SerializeParameters(pp, msg, le)
    paramsJSON, _ := json.Marshal(params)
    // Send paramsJSON to client via API/network
    
    // 4. Receive encrypted client data
    var clientCiphertexts []psi.Cxtx
    // Receive from client...
    
    // 5. Compute intersection
    monitor := psi.NewPerformanceMonitor()
    intersection, err := psi.DetectIntersectionWithContext(ctx, clientCiphertexts)
    if err != nil {
        log.Fatal(err)
    }
    
    // 6. Get performance metrics
    metrics := monitor.GetMetrics()
    memStats := monitor.GetMemoryUsage()
    
    log.Printf("Intersection: %v\n", intersection)
    log.Printf("Metrics: %+v\n", metrics)
    log.Printf("Memory: %+v\n", memStats)
}
```

#### Client Side:
```go
package main

import (
    "encoding/json"
    "log"
    "github.com/SanthoshCheemala/LE-PSI/pkg/psi"
)

func main() {
    // 1. Receive public parameters from server
    var paramsJSON []byte
    // Receive from server...
    
    var params psi.SerializableParams
    json.Unmarshal(paramsJSON, &params)
    
    // 2. Deserialize parameters
    pp, msg, le, err := psi.DeserializeParameters(&params)
    if err != nil {
        log.Fatal(err)
    }
    
    // 3. Encrypt client dataset
    clientData := []uint64{150, 200, 250, 350}
    ciphertexts := psi.ClientEncrypt(clientData, pp, msg, le)
    
    // 4. Send ciphertexts to server
    cipherJSON, _ := json.Marshal(ciphertexts)
    // Send to server...
    
    log.Printf("Encrypted %d elements\n", len(ciphertexts))
}
```

---

## Error Handling

All functions return errors that should be checked:

```go
ctx, err := psi.ServerInitialize(data, path)
if err != nil {
    // Handle specific errors
    switch {
    case strings.Contains(err.Error(), "ring dimension"):
        log.Fatal("Unsupported ring dimension")
    case strings.Contains(err.Error(), "database"):
        log.Fatal("Database error")
    default:
        log.Fatal(err)
    }
}
```

---

## Performance Tips

1. **Worker Threads**: Use `CalculateOptimalWorkers()` for automatic optimization
2. **Dataset Size**: Larger datasets benefit from more workers (up to hardware limit)
3. **Memory**: Monitor with `GetMemoryUsage()` to prevent OOM
4. **Throughput**: Track with `GetThroughput()` for performance tuning
5. **Database Path**: Use SSD storage for witness tree database

---

## Security Considerations

- **Security Level**: ~128-bit with default parameters (D=256, Q=180143985094819841)
- **Privacy**: Server learns only intersection, client reveals nothing about non-matching elements
- **Parameters**: Do not modify default cryptographic parameters without expert review
- **Network**: Use TLS/HTTPS for parameter and ciphertext transmission

---

## Support

For issues or questions:
- GitHub: [SanthoshCheemala/LE-PSI-](https://github.com/SanthoshCheemala/LE-PSI-)
- Documentation: `/Documentation/API_GUIDE.md`

---

**Last Updated**: December 3, 2025
**Version**: 1.0.0
