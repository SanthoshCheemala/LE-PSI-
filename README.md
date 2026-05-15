# LE-PSI: Laconic Private Set Intersection from Ring-LWE

A practical implementation of Laconic Private Set Intersection based on Ring Learning With Errors (Ring-LWE), providing lattice-based security with **O(n log m) communication complexity**.

## Overview

Private Set Intersection (PSI) allows two parties to discover shared elements without revealing their full datasets. Classical PSI protocols rely on discrete-log or factoring assumptions vulnerable to quantum attacks. LE-PSI implements the laconic encryption framework of [DKLLMR23] using Ring-LWE, offering security under lattice hardness assumptions.

**Key properties:**
- **Lattice-based security** — Ring-LWE hardness (conjectured post-quantum resistant)
- **O(n log m) communication** — Sublinear in server dataset size `m`
- **Leaf-indexed filtering** — Server performs only O(n) decryptions instead of O(n·m) by matching ciphertexts to their target Merkle leaves
- **2-choice Cuckoo placement** — Reduces collisions vs naive modular hashing
- **Distributed scaling** — Horizontal sharding across GCE VMs

## Cryptographic Parameters

| Parameter | Value | Notes |
|-----------|-------|-------|
| Ring dimension (D) | 256 (fast) / 2048 (secure) | Set via `PSI_SECURITY_LEVEL=128` |
| Modulus (Q) | 2^58 | 180143985094819841 |
| Matrix dimension (N) | 4 | |
| Gaussian width (σ) | 2^30 | Lattigo `SamplerGaussian` |
| Tree expansion | 16× | `layers = ceil(log2(16·m))` |
| Hash truncation | 64-bit | SHA-256 → uint64 leaf indices |
| Lattigo version | v3 | `github.com/tuneinsight/lattigo/v3` |

> **Security note:** D=256 is used for fast evaluation but does NOT provide 128-bit post-quantum security for the 58-bit modulus. Set `PSI_SECURITY_LEVEL=128` to enforce D=2048 for full security.

## Installation

```bash
git clone https://github.com/SanthoshCheemala/LE-PSI-.git
cd LE-PSI-
go mod download
```

Requires Go 1.21+ and CGO (for SQLite).

## Quick Start

```go
package main

import (
    "log"
    "github.com/SanthoshCheemala/LE-PSI/pkg/psi"
)

func main() {
    serverSet := []uint64{100, 200, 300, 400, 500}
    clientSet := []uint64{200, 400, 600}

    ctx, err := psi.ServerInitialize(serverSet, "tree.db")
    if err != nil { log.Fatal(err) }

    pp, msg, le := psi.GetPublicParameters(ctx)
    ciphertexts := psi.ClientEncrypt(clientSet, pp, msg, le)

    matches, err := psi.DetectIntersectionWithContext(ctx, ciphertexts)
    if err != nil { log.Fatal(err) }

    log.Printf("Intersection: %v", matches) // [200 400]
}
```

## How It Works

1. **Server** initializes a Merkle tree over its dataset and generates LE public parameters
2. **Client** encrypts each element toward its two Cuckoo leaf positions using Laconic Encryption
3. **Server** uses leaf-indexed filtering to attempt decryption only on matching leaves
4. **Server** returns the intersection elements

### Leaf-Indexed Filtering (Optimization)

The naive intersection requires O(m × 2n) decryption attempts. Since LE decryption only succeeds when the ciphertext's target leaf matches the server record's leaf, we build a `map[leaf] → []ciphertext_indices` and skip all non-matching pairs:

```
Before: 143 server records × 200 ciphertexts = 28,600 Dec calls
After:  Leaf-indexed filtering → 17 targeted Dec calls (99.94% reduction)
```

This is protocol-safe: the target leaf is already encoded in the ciphertext structure (path through the Merkle tree).

## Project Structure

```
LE-PSI/
├── pkg/psi/              # Core PSI: client, server, intersection logic
│   ├── server.go          # ServerInitialize, DetectIntersectionWithContext
│   ├── client.go          # ClientEncrypt (2-choice Cuckoo)
│   ├── helpers.go         # Cxtx struct, hash functions, CorrectnessCheck
│   ├── parameters.go      # SetupLEParameters (Ring-LWE config)
│   └── cxtx_serial.go     # JSON serialization for distributed mode
├── pkg/LE/               # Laconic Encryption primitives
│   ├── le.go              # LE.Setup (key generation)
│   └── LE_upd.go          # Enc, Dec, TreeHash, WitGen, MemoryTree
├── pkg/matrix/           # Ring-LWE matrix/vector operations
├── scalability_tests/    # Single-node benchmarks (1K–10K)
├── distributed_gce/      # Multi-shard distributed benchmarks on GCE
│   ├── coordinator/       # Fan-out coordinator
│   ├── shard/             # Per-shard intersection server
│   └── results/           # Benchmark JSON results
├── comparative_baselines/ # APSI comparison scripts
└── cmd/Flare/            # CLI demo tool
```

## Performance (Single Node, D=256)

| Server (m) | Client (n) | Ciphertexts | Init (s) | Intersect (s) | Total (min) |
|-----------|-----------|-------------|----------|---------------|-------------|
| 1,000     | 100       | 200         | ~55      | < 1           | ~1          |
| 2,000     | 100       | 200         | ~110     | < 1           | ~2          |
| 4,000     | 100       | 200         | ~220     | < 1           | ~4          |

> Note: Init time (key generation + Merkle tree) dominates. With leaf-indexed filtering, intersection is near-instant.

## Distributed Mode (GCE)

For large datasets, the server set is sharded across K VMs:

```bash
# Deploy to 7 shards
PROJECT=lepsi-distributed-493617 ZONE=us-east1-b bash distributed_gce/deploy_latest.sh

# Run benchmarks
PROJECT=lepsi-distributed-493617 ZONE=us-east1-b K=7 bash distributed_gce/run_all_benchmarks.sh
```

Each shard initializes independently and processes intersection in parallel via streaming JSON.

## Security Model

- **Semi-honest** (honest-but-curious) adversary model
- **Client privacy:** Server learns only |C| (client set size)
- **Server privacy:** Client learns only C ∩ S
- **Leakage:** Target leaf indices are visible to server (inherent in laconic encryption)

## Comparison with Classical PSI

| Protocol | Security | Comm. Complexity | Time (m=10⁴) | Post-Quantum |
|----------|----------|-----------------|---------------|--------------|
| KKRT     | DL-based | O(n + m)        | ~0.04 s       | No           |
| OKVS/VOLE| DL-based | O(n + m)        | ~0.02 s       | No           |
| Microsoft APSI | BFV/SEAL | O(n)     | TBD           | Lattice-based|
| **LE-PSI (Ours)** | Ring-LWE | **O(n log m)** | TBD    | **Yes**      |

> LE-PSI is slower than classical PSI due to Ring-LWE arithmetic overhead. The advantage is in the asymptotic communication savings when m ≫ n, and lattice-based security.

## References

- **[DKLLMR23]** Döttling, Garg, Ishai, Malavolta, Mour, Ostrovsky. "Trapdoor Hash Functions and Their Applications." Crypto 2019 / ePrint 2023/404.
- **[Lattigo]** Mouchet et al. "Lattigo: A Multiparty Homomorphic Encryption Library in Go." EPFL-LDS. v3.
- **[Ring-LWE]** Lyubashevsky, Peikert, Regev. "On Ideal Lattices and Learning with Errors over Rings." EUROCRYPT 2010.
- **[APSI]** Microsoft. "Asymmetric PSI." ePrint 2021/1116.

## License

MIT License — see [LICENSE](LICENSE).