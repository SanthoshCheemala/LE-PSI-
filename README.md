# Laconic PSI - Making Laconic Private Set Intersection Practical

The first practical, end-to-end implementation of Laconic Private Set Intersection based on Ring Learning With Errors (Ring-LWE). This system provides **post-quantum security** with **O(n log m) communication complexity** versus classical PSI's O(n + m).

## The Problem

Financial institutions must screen customers against sanctions lists to prevent money laundering, yet privacy regulations (GDPR) prohibit sharing customer data with third parties. Classical PSI protocols address this, but they rely on cryptography that quantum computers can break. Organizations protecting sensitive data for decades face a critical vulnerability: adversaries can intercept encrypted communications today and store them until quantum computers become available.

## Our Solution

We built the first complete implementation of Laconic PSI and developed memory-efficient batching and parallelization techniques that reduce peak memory from **312 GB to 6.5 GB**, enabling deployment on standard servers.

## Features

- **Post-quantum secure** based on Ring-LWE lattice problems
- **O(n log m) communication complexity** - sublinear in server dataset size
- **Memory-efficient architecture** - 97.9% reduction in peak RAM requirements
- **Adaptive threading** with auto-detection of CPU/RAM
- **Constant memory** - 6.5 GB peak RAM with batching (for 500+ records)
- **Distributed scaling** - 15× speedup across 25 AWS nodes
- **FLARE** - Proof-of-concept sanctions screening system for GDPR-compliant deployment

## Installation

```bash
go get anonymous.4open.science/anonymize/LE-PSI--5C28
```

## Quick Start

```go
package main

import (
    "log"
    "anonymous.4open.science/anonymize/LE-PSI--5C28/pkg/psi"
    "anonymous.4open.science/anonymize/LE-PSI--5C28/utils"
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

| Dataset Size | Time (min) | Peak RAM (GB) |
|--------------|------------|---------------|
| 50           | 0.2        | 1.5           |
| 100          | 0.6        | 3.6           |
| 500          | 6.3        | 6.5           |
| 1,000        | 15.7       | 6.5           |
| 2,000        | 31.1       | 6.5           |
| 5,000        | 76.1       | 6.5           |
| 10,000       | 152.8      | 6.5           |

**Scaling:** ~15 minutes per 1,000 records (linear scaling)  
**Throughput:** ~1.0 ops/sec for datasets exceeding 1,000 records  
**Worker count:** Auto-scales based on system (96-core → 77 workers)  
**Parallelization:** 80% CPU utilization, 35 MB per worker for Ring-LWE buffers

### Memory Optimization

Without batching, 10,000 records requires **312 GB RAM**. Our constant-memory batching reduces this to **6.5 GB** (97.9% reduction), enabling execution on standard hardware.

### Distributed Scaling

| Configuration | Time | Cost/hr | Total Cost |
|---------------|------|---------|------------|
| Single node (c5.24xlarge) | 153 min | $4.08 | $10.40 |
| Distributed (25×c5.2xlarge) | 10.2 min | $8.50 | $1.45 |

**86% cost reduction** with distributed deployment.

## Project Structure

```
Laconic-PSI/
├── pkg/psi/          # Core PSI implementation
├── pkg/LE/           # Laconic Encryption primitives (Ring-LWE)
├── pkg/matrix/       # Matrix operations for lattice crypto
├── utils/            # Data preprocessing utilities
├── internal/storage/ # Merkle tree storage operations
├── cmd/Flare/        # FLARE CLI tool for sanctions screening
├── simulation/       # HTTP server/client demo
├── scalability_tests/# Performance benchmarks
└── Documentation/    # API guides and research findings
```

## FLARE: Sanctions Screening System

FLARE is a proof-of-concept sanctions screening system demonstrating GDPR-compliant deployment:

- **Scale:** 100 customer records against 10,000 sanctions entries
- **Performance:** 153-minute screening latency (acceptable for overnight batch processing)
- **Communication:** 415 KB total bandwidth (4.15 KB per record)
- **Distributed:** 25 AWS EC2 nodes enable screening against 100,000 entries in 10 minutes

**GDPR Compliance by Design:**
- **Data Minimization (Art. 5.1c):** Only 256-bit encrypted hashes transmitted
- **Purpose Limitation (Art. 5.1b):** Provider learns only aggregate match count
- **Storage Minimization (Art. 5.1e):** No persistent customer data stored by third parties
- **Security (Art. 32):** Post-quantum encryption protects against future quantum attacks

## CLI Tool

```bash
cd cmd/Flare && go build -o flare

./flare -mode inline -server-data "1,2,3,4,5" -client-data "2,4,7"
```

## Bottleneck Analysis

Profiling reveals the intersection phase consumes 53-70% of execution time:

| Component | % of Time | Bound |
|-----------|-----------|-------|
| Witness fetching (Merkle proofs) | 35% | Memory-bound |
| NTT operations | 25% | CPU-bound (parallelizable) |
| Ring-LWE decryption | 10% | CPU-bound |

The memory-bound witness fetching bottleneck explains why throughput stabilizes at ~1.0 ops/sec—adding more workers cannot overcome memory bandwidth limits.

## Security

- **Post-quantum resistant** - Based on Ring-LWE lattice cryptography
- **Ring dimension:** n = 256
- **Modulus:** q ≈ 2^58
- **Error standard deviation:** σ = 3.2
- **64-bit quantum security** - Trade-off between security and performance
- **Semi-honest adversary model** - Secure against honest-but-curious parties
- **Client Privacy:** Server learns only |C| (number of client elements)
- **Server Privacy:** Client learns only C ∩ S (intersection)

> **Note:** For 128-bit quantum security, larger ring dimensions (n = 512 or 1024) would be required, resulting in ~4× larger ciphertexts and ~4× slower computation.

## Use Cases

- **FLARE Sanctions Screening** - GDPR-compliant regulatory verification for financial institutions
- **Privacy-preserving compliance** - Screen customers against watchlists without exposing customer data
- **Long-term data protection** - Security guarantees that remain valid as quantum computers advance
- **Healthcare data matching** - Hospitals finding common patients with post-quantum security
- **Government archives** - Protecting sensitive data with 30+ year lifespans
- **Financial records** - Quantum-resistant screening for regulated institutions

## Comparison with Classical PSI

| Protocol | Time (10⁴ records) | Communication | Memory | Post-Quantum |
|----------|-------------------|---------------|--------|--------------|
| KKRT     | 0.04 s            | 0.5 MB        | 0.1 GB | No           |
| OKVS     | 0.02 s            | 0.4 MB        | 0.1 GB | No           |
| **Laconic PSI (Ours)** | 153 min | 41 MB | 6.5 GB | **Yes** |

Laconic PSI is ~10,000× slower than classical PSI due to:
- Ring-LWE operations (~1,000× overhead)
- Laconic protocol structure (~10× overhead)

### When to Use Laconic PSI

1. **Long-term data protection:** Applications with 30+ year data lifespans (medical records, government archives)
2. **Highly unbalanced datasets:** When m ≫ n (e.g., 10,000 queries against 1 million records), O(n log m) provides 5× communication savings
3. **Overnight batch processing:** Applications with loose latency requirements where quantum resistance is mandatory

## Building from Source

```bash
git clone anonymous.4open.science/anonymize/LE-PSI--5C28
cd LE-PSI--5C28
go mod download
go build -o flare ./cmd/Flare
```

## Technical Details

### Communication Analysis

For 10,000 records:
- Ciphertext: 3.7 KB per record
- Merkle proof: 0.45 KB per record (⌈log₂ m⌉ hashes)
- **Total:** 4.15 KB per record → 41 MB total

This reflects O(n log m) complexity, providing significant savings when m ≫ n.

### Worker Calculation

Worker count is determined dynamically:
```
w = min(CPUs × 0.8, Available RAM / 35MB)
```

On a 96-core, 256 GB system with 6.5 GB batch memory: 77 workers (CPU-limited, not memory-limited).

## Documentation

- [API Documentation](API.md) - Complete API reference
- [Scalability Tests](scalability_tests/README.md) - Performance benchmarks
- [Research Findings](Documentation/RESEARCH_PAPER_FINDINGS.md) - Detailed analysis

## Future Work

- **Hardware Acceleration:** GPU/FPGA implementations for NTT operations (25% of execution time)
- **Parameter Optimization:** Exploring n = 512/1024 for 128-bit quantum security
- **Scalability Research:** Testing beyond 10⁴ records to validate asymptotic advantages
- **Malicious Security:** Extensions to defend against protocol deviations

## License

MIT License - See [LICENSE](LICENSE) for details.

## References

- **Laconic PSI:** Döttling et al., "Trapdoor Hash Functions and Their Applications," Cryptology ePrint Archive, Paper 2023/404, 2023. [Link](https://eprint.iacr.org/2023/404)
- **Lattigo:** Mouchet et al., "Lattigo: A Multiparty Homomorphic Encryption Library in Go," EPFL-LDS. [GitHub](https://github.com/tuneinsight/lattigo)
- **Ring-LWE:** Lyubashevsky, Peikert, Regev, "On Ideal Lattices and Learning with Errors over Rings," EUROCRYPT 2010.
- **Classical PSI (KKRT):** Kolesnikov et al., "Efficient Batched Oblivious PRF with Applications to Private Set Intersection," CCS 2016.

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.