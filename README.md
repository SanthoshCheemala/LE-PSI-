# LE-PSI - Lattice-based Private Set Intersection

A high-performance Go library for Private Set Intersection using lattice-based cryptography. Two parties can securely find common elements in their datasets without revealing any other information.

## What is Private Set Intersection?

Private Set Intersection allows two parties to find common elements between their datasets without either party revealing their full dataset to the other. For example, two companies can find shared customers without exposing their entire customer lists.

## Features

- Post-quantum secure based on Ring-LWE lattice problems
- High performance with adaptive threading
- Supports any serializable data types
- Clean API for easy integration
- Handles datasets from 100 to 4,000+ records

## Installation

```bash
go get github.com/SanthoshCheemala/LE-PSI
```

## Quick Start

```go
package main

import (
    "github.com/SanthoshCheemala/LE-PSI/pkg/psi"
    "github.com/SanthoshCheemala/LE-PSI/utils"
)

func main() {
    // Server setup
    serverData := []interface{}{"alice", "bob", "charlie"}
    serverHashes := utils.HashDataPoints(utils.PrepareDataForPSI(serverData))
    
    ctx, _ := psi.ServerInitialize(serverHashes, "tree.db")
    pp, msg, le := psi.GetPublicParameters(ctx)
    
    // Client query
    clientData := []interface{}{"bob", "david"}
    clientHashes := utils.HashDataPoints(utils.PrepareDataForPSI(clientData))
    
    ciphertexts := psi.ClientEncrypt(clientHashes, pp, msg, le)
    
    // Find intersection
    matches, _ := psi.DetectIntersectionWithContext(ctx, ciphertexts)
}
```

## How It Works

1. Server initializes with their dataset and generates public parameters
2. Server sends public parameters to client
3. Client encrypts their dataset using the public parameters
4. Client sends encrypted data back to server
5. Server detects intersection without learning client's full dataset
6. Server returns matching elements

## Performance

| Records | Memory  | Time    | Workers |
|---------|---------|---------|---------|
| 100     | 3.5 GB  | 30s     | 48      |
| 500     | 15 GB   | 4m      | 34      |
| 1,000   | 30 GB   | 1h      | 38      |
| 2,000   | 55 GB   | 2h      | 34      |

Memory usage is approximately 35 MB per record with adaptive threading using 8-48 workers.

## Project Structure

```
LE-PSI/
├── pkg/psi/          Core PSI implementation
├── pkg/LE/           Laconic Encryption primitives
├── pkg/matrix/       Matrix operations for lattice crypto
├── utils/            Data preprocessing utilities
├── internal/storage/ Database operations for tree structure
├── cmd/Flare/        Command-line tool
├── simulation/       HTTP server/client demo
└── benchmarks/       Performance tests
```

## CLI Tool

```bash
cd cmd/Flare && go build -o flare

./flare -mode inline -server-data "1,2,3,4,5" -client-data "2,4,7"
```

## Use Cases

- Privacy-preserving analytics
- Contact discovery
- Threat intelligence sharing
- Healthcare data matching
- Ad campaign measurement
- Supply chain verification

## Security

- Post-quantum resistant based on lattice cryptography
- Semantic security with no information leakage beyond intersection
- 256-ring dimension with 58-bit modulus
- 128-bit classical security level

## Building from Source

```bash
git clone https://github.com/SanthoshCheemala/LE-PSI.git
cd LE-PSI
go mod download
go build -o flare ./cmd/Flare
```

## Documentation

- [API Guide](Documentation/API_GUIDE.md) - Integration guide for distributed systems
- [CLI Tool](cmd/Flare/README.md) - Command-line usage

## License

MIT License

## References

- Efficieny Laconic Cryptography 
- Lattigo 
