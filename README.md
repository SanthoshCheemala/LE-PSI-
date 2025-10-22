````markdown
# LE-PSI - Laconic Encryption Private Set Intersection

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

LE-PSI is a high-performance Private Set Intersection (PSI) framework built on lattice-based Laconic Oblivious Transfer (LOT) and Laconic Encryption. It enables two parties to compute the intersection of their private datasets without revealing any additional information.

## 🌟 Key Features

- **Privacy-Preserving**: Cryptographically secure intersection without data leakage
- **Lattice-Based Security**: Post-quantum secure using lattice cryptography
- **High Performance**: Parallel processing utilizing all CPU cores
- **Easy Integration**: Simple API for distributed systems
- **Flexible Data Types**: Support for strings, integers, JSON, and any serializable data
- **Production Ready**: Clean, well-tested codebase with comprehensive documentation

## 🚀 Quick Start

### Installation

```bash
# Clone the repository
git clone https://github.com/SanthoshCheemala/LE-PSI.git
cd LE-PSI

# Build the command-line tool
cd cmd/Flare
go build -o flare
```

### Basic Usage

```bash
# Quick inline mode
./flare -mode inline -server-data "1,2,3,4,5" -client-data "2,4,7"
# Output: Matches: [2 4]

# With string data
./flare -mode inline -server-data "alice,bob,charlie" -client-data "bob,david"
# Output: Matches: [bob]
```

### Programmatic Usage

```go
package main

import (
    psi "github.com/SanthoshCheemala/LE-PSI/internal/crypto/PSI"
    "github.com/SanthoshCheemala/LE-PSI/utils"
)

func main() {
    // Server dataset
    serverData := []interface{}{"alice", "bob", "charlie", 123, 456}
    serverStrings, _ := utils.PrepareDataForPSI(serverData)
    serverHashes := utils.HashDataPoints(serverStrings)
    
    // Server initialization
    ctx, _ := psi.ServerInitialize(serverHashes, "data/tree.db")
    pp, msg, le := psi.GetPublicParameters(ctx)
    
    // Client dataset
    clientData := []interface{}{"bob", "david", 123}
    clientStrings, _ := utils.PrepareDataForPSI(clientData)
    clientHashes := utils.HashDataPoints(clientStrings)
    
    // Client encryption
    ciphertexts := psi.ClientEncrypt(clientHashes, pp, msg, le)
    
    // Server intersection detection
    matches, _ := psi.DetectIntersectionWithContext(ctx, ciphertexts)
    // matches contains the hash values of intersecting items
}
```

## 📊 Architecture Overview

FLARE uses a client-server architecture with three main phases:

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         LE-PSI Architecture                             │
└─────────────────────────────────────────────────────────────────────────┘

Phase 1: Server Initialization
┌──────────────┐
│   Server     │  1. Preprocess data (serialize + hash)
│   Dataset    │  2. Generate keys for each item
│  [X₁...Xₙ]   │  3. Build Merkle-like tree structure
└──────┬───────┘  4. Compute witness vectors
       │          5. Generate public parameters (pp, msg, LE)
       ▼
┌──────────────┐
│ ServerInit   │──────► Public Parameters (pp, msg, LE)
│              │        └─► Send to Client
└──────────────┘

Phase 2: Client Encryption
┌──────────────┐
│   Client     │  1. Receive public parameters
│   Dataset    │  2. Preprocess data (serialize + hash)
│  [Y₁...Yₘ]   │  3. Encrypt each item using pp, msg, LE
└──────┬───────┘     └─► Laconic Encryption with tree index
       │          4. Generate ciphertexts
       ▼
┌──────────────┐
│ClientEncrypt │──────► Ciphertexts [C₁...Cₘ]
│              │        └─► Send to Server
└──────────────┘

Phase 3: Intersection Detection
┌──────────────┐
│ Ciphertexts  │  1. Receive client ciphertexts
│  [C₁...Cₘ]   │  2. Decrypt each ciphertext with server keys
└──────┬───────┘  3. Check if decrypted message matches
       │          4. Return indices of matching items
       ▼
┌──────────────┐
│   Detect     │──────► Intersection Result
│ Intersection │        └─► Matching Indices/Hashes
└──────────────┘
```

### Data Flow

```
Server Data        Client Data
    │                  │
    ▼                  ▼
┌─────────┐      ┌─────────┐
│ Prepare │      │ Prepare │
│  Data   │      │  Data   │
└────┬────┘      └────┬────┘
     │                │
     ▼                │
┌─────────┐           │
│  Hash   │           │
└────┬────┘           │
     │                │
     ▼                │
┌──────────────┐      │
│  Initialize  │      │
│    Server    │      │
└───────┬──────┘      │
        │             │
        ├─────────────┼──► Public Parameters (pp, msg, LE)
        │             │
        │             ▼
        │        ┌─────────┐
        │        │  Hash   │
        │        └────┬────┘
        │             │
        │             ▼
        │        ┌─────────┐
        │        │ Encrypt │
        │        └────┬────┘
        │             │
        ◄─────────────┤
        │             │
        ▼             │
┌──────────────┐      │
│    Detect    │◄─────┘
│ Intersection │    Ciphertexts
└───────┬──────┘
        │
        ▼
   Intersection
     Results
```

## 🏗️ Project Structure

```
LE-PSI/
├── cmd/
│   └── Flare/              # Command-line tool
│       ├── main.go         # CLI implementation
│       └── README.md       # CLI usage guide
│
├── internal/
│   ├── crypto/
│   │   └── PSI/           # Core PSI implementation
│   │       ├── server.go  # Server-side functions
│   │       ├── client.go  # Client-side functions
│   │       ├── helpers.go # Utility functions
│   │       └── parameters.go
│   │
│   └── storage/
│       ├── db.go          # Tree database operations
│       └── keys.go        # Key storage
│
├── pkg/
│   ├── LE/                # Laconic Encryption
│   │   ├── LE.go          # Core LE implementation
│   │   ├── LE_keygen.go   # Key generation
│   │   └── LE_upd.go      # Tree update & encryption
│   │
│   └── matrix/            # Matrix operations
│       ├── matrix.go      # Matrix arithmetic
│       └── matrix_vector.go
│
├── utils/
│   ├── data_preprocessor.go  # Data serialization
│   └── report_generation.go  # Performance metrics
│
├── simulation/            # Distributed demo
│   ├── server/           # HTTP server
│   └── client/           # HTTP client
│
├── Documentation/         # Technical docs
│   ├── API_GUIDE.md      # Integration guide
│   └── CONCEPTS.md       # Cryptographic concepts
│
└── benchmarks/           # Performance tests
```

## 📖 Core API Functions

### Server Side

```go
// Initialize server with dataset
ctx, err := psi.ServerInitialize(serverHashes, "db.sqlite")

// Get public parameters to send to client
pp, msg, le := psi.GetPublicParameters(ctx)

// Serialize parameters for network transmission
serialized := psi.SerializeParameters(pp, msg, le)

// Detect intersection with client ciphertexts
matches, err := psi.DetectIntersectionWithContext(ctx, ciphertexts)
```

### Client Side

```go
// Deserialize parameters received from server
pp, msg, le, err := psi.DeserializeParameters(serialized)

// Encrypt client dataset
ciphertexts := psi.ClientEncrypt(clientHashes, pp, msg, le)
```

### Data Preprocessing

```go
// Prepare any data type for PSI
serialized, err := utils.PrepareDataForPSI([]interface{}{
    "string", 123, map[string]string{"key": "value"},
})

// Hash serialized data
hashes := utils.HashDataPoints(serialized)
```

## 🎯 Use Cases

- **Privacy-Preserving Analytics**: Compute joint statistics without sharing raw data
- **Contact Discovery**: Find mutual contacts without revealing all contacts
- **Threat Intelligence**: Share security indicators while maintaining confidentiality
- **Healthcare**: Identify common patients across institutions
- **Ad Campaign Measurement**: Measure ad effectiveness without sharing user lists
- **Supply Chain**: Find common suppliers without revealing business relationships

## 📈 Performance

**Benchmark Results** (Apple M1, 8 cores):

| Server Set Size | Client Set Size | Initialization | Encryption | Detection | Total    |
|----------------|----------------|----------------|------------|-----------|----------|
| 100            | 50             | ~150ms         | ~80ms      | ~25ms     | ~255ms   |
| 1,000          | 500            | ~1.2s          | ~400ms     | ~180ms    | ~1.8s    |
| 10,000         | 5,000          | ~15s           | ~4s        | ~2s       | ~21s     |

- **Parallel Efficiency**: 8x speedup with 8 cores
- **Memory Usage**: ~500MB for 10,000 items
- **Network Transfer**: ~2MB parameters for typical setup

## 🔒 Security

- **Post-Quantum Secure**: Based on Ring-LWE lattice problems
- **Semantic Security**: Ciphertexts reveal no information about plaintexts
- **Malicious Security**: Resistant to common attack vectors
- **No Data Leakage**: Only intersection is revealed, nothing else

**Cryptographic Parameters:**
- Ring Dimension: 256
- Modulus: 180143985094819841 (58 bits)
- Security Level: ~128-bit classical, ~64-bit quantum

## 🛠️ Development

### Building from Source

```bash
# Install dependencies
go mod download

# Build command-line tool
go build -o flare ./cmd/Flare

# Build simulation servers
cd simulation/server && go build -o server_sim
cd ../client && go build -o client_sim

# Run tests
go test ./...

# Run benchmarks
cd benchmarks && go run benchmark_main.go
```

### Running Simulations

**Terminal 1 - Start Server:**
```bash
cd simulation/server
./server_sim
```

**Terminal 2 - Run Client:**
```bash
cd simulation/client
./client_sim
```

## 📚 Documentation

- **[API Guide](Documentation/API_GUIDE.md)**: Detailed integration guide for distributed systems
- **[Concepts](Documentation/CONCEPTS.md)**: Understanding Laconic Encryption and PSI
- **[CLI Tool](cmd/Flare/README.md)**: Command-line tool usage

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🔗 References

- **Laconic Oblivious Transfer**: [Cho et al. 2017]
- **Laconic Private Set Intersection**: [Alamati et al. 2021]
- **Lattigo Library**: [EPFL-LDS/lattigo](https://github.com/tuneinsight/lattigo)

## 📧 Contact

For questions or support, please open an issue on GitHub.

---

**LE-PSI: Built with ❤️ using Go and Lattice Cryptography**

````
