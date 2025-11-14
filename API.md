# LE-PSI Library API Documentation

## Overview

LE-PSI is a Go library implementing Private Set Intersection using Lattice-based Laconic Encryption. This library provides a clean API for integrating PSI functionality into your applications.

## Installation

```bash
go get github.com/SanthoshCheemala/LE-PSI
```

## Quick Start

```go
import (
    "github.com/SanthoshCheemala/LE-PSI/pkg/psi"
    "github.com/SanthoshCheemala/LE-PSI/utils"
)

// Server side
serverData := []interface{}{"alice", "bob", "charlie"}
serverStrings, _ := utils.PrepareDataForPSI(serverData)
serverHashes := utils.HashDataPoints(serverStrings)

ctx, _ := psi.ServerInitialize(serverHashes, "tree.db")
pp, msg, le := psi.GetPublicParameters(ctx)

// Client side
clientData := []interface{}{"bob", "david"}
clientStrings, _ := utils.PrepareDataForPSI(clientData)
clientHashes := utils.HashDataPoints(clientStrings)

ciphertexts := psi.ClientEncrypt(clientHashes, pp, msg, le)

// Detect intersection
matches, _ := psi.DetectIntersectionWithContext(ctx, ciphertexts)
```

## Core API

### Data Preprocessing (`utils` package)

#### `PrepareDataForPSI(dataset []interface{}) ([]string, error)`
Converts any data type array to serialized strings for PSI processing.

**Parameters:**
- `dataset`: Array of any serializable data types (strings, integers, structs, maps, etc.)

**Returns:**
- `[]string`: Serialized string representations
- `error`: Error if serialization fails

**Example:**
```go
data := []interface{}{"user@email.com", 12345, map[string]string{"key": "value"}}
serialized, err := utils.PrepareDataForPSI(data)
```

#### `HashDataPoints(serializedData []string) []uint64`
Converts serialized strings to uint64 hashes using SHA-256.

**Parameters:**
- `serializedData`: Array of serialized strings

**Returns:**
- `[]uint64`: Array of hash values

### Server API (`psi` package)

#### `ServerInitialize(privateSet []uint64, dbPath string) (*ServerInitContext, error)`
Initializes the server with a private dataset.

**Parameters:**
- `privateSet`: Array of hashed uint64 values
- `dbPath`: Path to SQLite database for tree storage

**Returns:**
- `*ServerInitContext`: Server context containing keys and witnesses
- `error`: Error if initialization fails

**Features:**
- Adaptive threading based on dataset size
- Automatic optimization for memory and CPU
- Progress logging

#### `GetPublicParameters(ctx *ServerInitContext) (*matrix.Vector, *ring.Poly, *LE.LE)`
Extracts public parameters to send to clients.

**Parameters:**
- `ctx`: Server initialization context

**Returns:**
- `*matrix.Vector`: Public parameter vector
- `*ring.Poly`: Message polynomial
- `*LE.LE`: Laconic Encryption parameters

#### `DetectIntersectionWithContext(ctx *ServerInitContext, ciphertexts []Cxtx) ([]uint64, error)`
Detects intersection between server and client datasets.

**Parameters:**
- `ctx`: Server initialization context
- `ciphertexts`: Encrypted client queries

**Returns:**
- `[]uint64`: Hash values of intersecting items
- `error`: Error if detection fails

**Performance:**
- Adaptive worker threads (8-48 based on dataset size)
- Parallel decryption for optimal speed
- Memory-aware processing

### Client API (`psi` package)

#### `ClientEncrypt(privateSet []uint64, pp *matrix.Vector, msg *ring.Poly, le *LE.LE) []Cxtx`
Encrypts client dataset for private queries.

**Parameters:**
- `privateSet`: Array of hashed uint64 values
- `pp`: Public parameter vector from server
- `msg`: Message polynomial from server
- `le`: Laconic Encryption parameters from server

**Returns:**
- `[]Cxtx`: Array of ciphertexts

**Features:**
- Multi-threaded encryption
- Deterministic output for same inputs
- No data leakage

### Parameter Serialization

#### `SerializeParameters(pp *matrix.Vector, msg *ring.Poly, le *LE.LE) *SerializableParams`
Serializes parameters for network transmission.

#### `DeserializeParameters(params *SerializableParams) (*matrix.Vector, *ring.Poly, *LE.LE, error)`
Deserializes parameters received over network.

**Use Case:**
```go
// Server side
serialized := psi.SerializeParameters(pp, msg, le)
// Send serialized over HTTP/gRPC

// Client side
pp, msg, le, err := psi.DeserializeParameters(serialized)
```

## Configuration

### Adaptive Threading
The library automatically optimizes worker threads based on:
- Dataset size
- Available RAM (configurable: default 117 GB)
- CPU cores (configurable: default 48)
- Cache efficiency

To customize, modify constants in `pkg/psi/server.go`:
```go
const (
    availableRAM_GB = 117.0  // Your available RAM
    hardwareLimit   = 48     // Your CPU cores
)
```

### Verbose Logging
Enable detailed logging:
```bash
export PSI_VERBOSE=true
```

## Performance Characteristics

| Dataset Size | Memory Usage | Estimated Time | Workers |
|-------------|--------------|----------------|---------|
| 100         | 3.5 GB       | ~30s           | 32-48   |
| 500         | 15 GB        | ~4m            | 28-34   |
| 1,000       | 30 GB        | ~1h            | 32-38   |
| 2,000       | 55 GB        | ~2h            | 28-34   |
| 4,000       | 95 GB        | ~5h            | 24-29   |

**Memory per record:** ~35 MB (includes witnesses, threads, overhead)

## Security

- **Post-quantum secure**: Based on Ring-LWE lattice problems
- **Semantic security**: Ciphertexts reveal no information beyond intersection
- **128-bit classical security**: 58-bit modulus provides strong security guarantees
- **No data leakage**: Only intersection results are revealed

## Error Handling

All public functions return errors that should be checked:

```go
ctx, err := psi.ServerInitialize(hashes, "tree.db")
if err != nil {
    log.Fatalf("Server initialization failed: %v", err)
}
```

Common errors:
- Empty datasets
- Database access failures
- Insufficient memory
- Invalid parameters

## Advanced Usage

### Custom Data Types

The library supports any serializable Go type:

```go
type User struct {
    Email string
    ID    int
}

users := []interface{}{
    User{Email: "alice@example.com", ID: 1},
    User{Email: "bob@example.com", ID: 2},
}

serialized, _ := utils.PrepareDataForPSI(users)
hashes := utils.HashDataPoints(serialized)
```

### Performance Monitoring

Enable performance tracking:

```go
import "github.com/SanthoshCheemala/LE-PSI/pkg/psi"

// Performance metrics are logged automatically
ctx, _ := psi.ServerInitialize(hashes, "tree.db")
// Logs: "Adaptive Threading: 1000 records → 32 workers (est. RAM: 35.0 GB)"
```

### Memory Management

For large datasets, the library automatically:
- Scales worker threads down to prevent swap
- Uses 85% of available RAM maximum
- Optimizes cache usage with 1.5×√n workers

## Examples

See the `/examples` directory for:
- Basic PSI workflow
- HTTP server/client implementation
- Batch processing
- Custom data types

## License

MIT License - see LICENSE file for details.

## Support

- GitHub Issues: https://github.com/SanthoshCheemala/LE-PSI/issues
- Documentation: https://github.com/SanthoshCheemala/LE-PSI/blob/main/README.md
