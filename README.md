# LE-PSI - Lattice-based Private Set Intersection

A high-performance Go library for Private Set Intersection using lattice-based cryptography. Two parties can securely find common elements in their datasets without revealing any other information.

## What is Private Set Intersection?

Private Set Intersection allows two parties to find common elements between their datasets without either party revealing their full dataset to the other. For example, two companies can find shared customers without exposing their entire customer lists.

## Features

- **Post-quantum secure** based on Ring-LWE lattice problems
- **High performance** with adaptive threading (auto-detects CPU/RAM)
- **Supports any serializable data types** (strings, integers, structs, maps)
- **Clean API** for easy integration
- **Scalable** - handles datasets from 100 to 10,000+ records
- **Constant memory** - 6.5 GB peak RAM with batching (regardless of dataset size)

## Installation

```bash
go get github.com/SanthoshCheemala/LE-PSI
```

## Quick Start

```go
package main

import (
    "log"
    "github.com/SanthoshCheemala/LE-PSI/pkg/psi"
    "github.com/SanthoshCheemala/LE-PSI/utils"
)

func main() {
    // Server setup
    serverData := []interface{}{"alice", "bob", "charlie"}
    serverStrings, _ := utils.PrepareDataForPSI(serverData)
    serverHashes := utils.HashDataPoints(serverStrings)
    
    ctx, err := psi.ServerInitialize(serverHashes, "tree.db")
    if err != nil {
        log.Fatal(err)
    }
    
    pp, msg, le := psi.GetPublicParameters(ctx)
    
    // Client query
    clientData := []interface{}{"bob", "david"}
    clientStrings, _ := utils.PrepareDataForPSI(clientData)
    clientHashes := utils.HashDataPoints(clientStrings)
    
    ciphertexts := psi.ClientEncrypt(clientHashes, pp, msg, le)
    
    // Find intersection
    matches, _ := psi.DetectIntersectionWithContext(ctx, ciphertexts)
    log.Printf("Found %d matches", len(matches))
}
```

## How It Works

1. **Server initializes** with their dataset and generates public parameters
2. **Server sends** public parameters to client
3. **Client encrypts** their dataset using the public parameters
4. **Client sends** encrypted data back to server
5. **Server detects** intersection without learning client's full dataset
6. **Server returns** matching elements

## Performance

| Records | Memory  | Time    | Workers (96-core) |
|---------|---------|---------|-------------------|
| 100     | 3.5 GB  | ~30s    | 77                |
| 500     | 15 GB   | ~4m     | 77                |
| 1,000   | 30 GB   | ~1h     | 77                |
| 2,000   | 55 GB   | ~2h     | 77                |
| 5,000   | 140 GB  | ~6h     | 77                |
| 10,000  | 313 GB  | ~2h33m  | 77                |

**Memory per record:** ~35 MB (includes witnesses, threads, overhead)  
**Worker count:** Auto-scales based on your system (8-core → ~6 workers, 96-core → 77 workers)  
**Parallelization:** 80% CPU utilization for optimal cache performance

### Auto-Detection
The system **automatically detects** your hardware and optimizes performance:
- Detects CPU cores using `runtime.NumCPU()`
- Detects available RAM using `runtime.MemStats`
- Scales worker threads to 80% of CPU cores
- No manual configuration needed!

## Project Structure

```
LE-PSI/
├── pkg/psi/          # Core PSI implementation
├── pkg/LE/           # Laconic Encryption primitives
├── pkg/matrix/       # Matrix operations for lattice crypto
├── utils/            # Data preprocessing utilities
├── internal/storage/ # Database operations for tree structure
├── cmd/Flare/        # Command-line tool
├── simulation/       # HTTP server/client demo
├── scalability_tests/# Performance benchmarks
└── Documentation/    # API guides and research papers
```

## CLI Tool

```bash
cd cmd/Flare && go build -o flare

./flare -mode inline -server-data "1,2,3,4,5" -client-data "2,4,7"
```

## Use Cases

- **Privacy-preserving analytics** - Find common users without sharing full datasets
- **Contact discovery** - Messaging apps finding mutual contacts
- **Threat intelligence sharing** - Security teams sharing indicators without exposing sources
- **Healthcare data matching** - Hospitals finding common patients for research
- **Ad campaign measurement** - Measuring ad effectiveness without sharing user data
- **Supply chain verification** - Verifying suppliers without revealing full supply chain

## Security

- **Post-quantum resistant** - Based on Ring-LWE lattice cryptography
- **Semantic security** - No information leakage beyond intersection
- **128-bit classical security** - Ring dimension D=256, modulus Q≈2^58
- **64-bit quantum security** - Resistant to Shor's algorithm
- **Semi-honest adversary** - Secure against honest-but-curious parties

## Building from Source

```bash
git clone https://github.com/SanthoshCheemala/LE-PSI.git
cd LE-PSI
go mod download
go build -o flare ./cmd/Flare
```

## Documentation

- [API Documentation](API.md) - Complete API reference
- [Scalability Tests](scalability_tests/README.md) - Performance benchmarks
- [CLI Tool Guide](cmd/Flare/README.md) - Command-line usage

## Performance Monitoring

Built-in performance tracking:

```go
monitor := psi.NewPerformanceMonitor()
// ... perform PSI operations ...
monitor.PrintReport()

// Output:
// LE-PSI Performance Report (Parallelized)
// CPU Cores Used: 96
// Total Execution Time: 2h33m
// Throughput: 1.08 operations/second
```

## License

MIT License - See [LICENSE](LICENSE) for details.

## References

- **Laconic PSI:** Döttling et al., "Efficient Laconic Cryptography from Learning with Errors," Cryptology ePrint Archive, Paper 2023/404, 2023. [Link](https://eprint.iacr.org/2023/404)
- **Lattigo:** Lattice-based cryptographic library in Go. [GitHub](https://github.com/tuneinsight/lattigo)
- **Ring-LWE:** Lyubashevsky, Peikert, Regev, "On Ideal Lattices and Learning with Errors over Rings," EUROCRYPT 2010.

## Citation

If you use LE-PSI in your research, please cite:

```bibtex
@software{le_psi_2024,
  title={LE-PSI: Efficient Implementation of Laconic Private Set Intersection},
  author={Cheemala, Santhosh},
  year={2024},
  url={https://github.com/SanthoshCheemala/LE-PSI}
}
```

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.

## Support

- **GitHub Issues:** [Report bugs or request features](https://github.com/SanthoshCheemala/LE-PSI/issues)
- **Documentation:** [Full documentation](https://github.com/SanthoshCheemala/LE-PSI/blob/main/API.md)
