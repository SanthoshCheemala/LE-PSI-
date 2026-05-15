# LE-PSI: Laconic Private Set Intersection from Ring-LWE

LE-PSI is a Go implementation and evaluation artifact for DKLLMR-style
laconic private set intersection using Ring-LWE-based laconic encryption.

This repository is an implementation and benchmarking artifact. The default
benchmark parameters are chosen for fast evaluation; they are not a claim of
production 128-bit post-quantum security.

## What This Repository Contains

- Core laconic encryption and PSI implementation in `pkg/`.
- Explicit chunk-batched single-node benchmark path.
- Leaf-indexed filtering that reduces decryption attempts by routing
  ciphertexts to matching Merkle leaves.
- Distributed GCE coordinator/shard benchmark scripts.
- Same-machine Microsoft APSI comparison artifacts.
- Reproducibility evidence for the final 2026-05-15 runs.

## Important Security and Leakage Notes

The final reported runs use `D=256`, which is a reduced-parameter evaluation
mode. The code supports `PSI_SECURITY_LEVEL=128` to select `D=2048`, but the
final 1K-10K GCE results in this artifact were not collected at `D=2048`.

The optimized implementation uses leaf-indexed filtering. Each ciphertext
contains a visible target leaf so the server can avoid all-pairs decryption.
This exposes candidate leaf/bucket indices as an implementation leakage term.
The current client emits two ciphertexts per client item and does not shuffle
those adjacent pairs.

Input records are canonicalized and represented as `uint64` identifiers. The
utility preprocessing path hashes serialized records with SHA-256 and uses the
first 64 bits.

## Parameters Used in Final Evaluation Runs

| Parameter | Value |
|---|---:|
| Ring dimension `D` | 256 |
| Secure-mode option | `PSI_SECURITY_LEVEL=128` selects `D=2048` |
| Modulus `q` | 180143985094819841 |
| `qBits` | 58 |
| Matrix dimension `N` | 4 |
| Gaussian width `sigma` | 1073741824 (`2^30`) |
| Tree expansion | `ceil(log2(16*m))` layers |
| Lattigo | `github.com/tuneinsight/lattigo/v3 v3.0.6` |
| Identifier width | SHA-256-derived `uint64` |

## Final Evidence Bundle

The main reproducibility bundle is:

`comparative_baselines/results/evidence/psi_repro_20260515_145900/`

It contains:

- `lepsi_single/`: LE-PSI single-node logs and JSON for 1K, 2K, 4K, 8K, and 10K.
- `lepsi_distributed/chunked_b52740c_20260515_170750/`: distributed coordinator
  logs, result JSONs, shard logs, and summaries.
- `comparative/apsi/`: Microsoft APSI 10K same-machine baseline.
- `comparative/kkrt_libpsi/STATUS.md`: attempted KKRT/libPSI build status.
- `comparative/alos22_relic/STATUS.md`: attempted ALOS22/RELIC status.
- `MANIFEST` and `SHA256SUMS`.

Additional paper-support material is in:

`docs/paper_artifact/`

That folder contains an audit-style dossier answering code/result questions and
selected source-data-backed figures. It is not manuscript text.

## Single-Node LE-PSI Results

Hardware: GCE `psi-compare`, `e2-highmem-8`, 8 vCPUs, about 62.8 GiB RAM,
`us-east1-b`.

Mode: `explicit_chunked`, `chunk_size=256`, `leaf_indexed_filtering=true`.

| m | n | total_sec | init_sec | enc_sec | intersect_sec | peak_rss_mb | matches | actual_dec_calls |
|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| 1000 | 100 | 19.944 | 6.509 | 12.316 | 1.120 | 6141 | 10 | 10 |
| 2000 | 100 | 24.908 | 11.520 | 10.775 | 2.612 | 7532 | 10 | 10 |
| 4000 | 100 | 41.877 | 23.438 | 11.278 | 7.161 | 11007 | 10 | 10 |
| 8000 | 100 | 74.639 | 46.274 | 9.409 | 18.956 | 17162 | 10 | 10 |
| 10000 | 100 | 94.516 | 56.496 | 9.364 | 28.655 | 20764 | 10 | 10 |

The single-node benchmark uses a controlled client generator whose non-overlap
items avoid occupied server leaves. This is useful for auditing the
leaf-filtered optimized path, but should be described as a controlled benchmark.

## Distributed LE-PSI Results

Hardware: one coordinator plus seven `e2-highmem-4` shard VMs across
`us-east1-c` and `us-east1-d`.

| m | n | shards | total_sec | init_sec | intersect_sec | matches |
|---:|---:|---:|---:|---:|---:|---:|
| 1000 | 100 | 7 | 158.108 | 6.070 | 98.064 | 16 |
| 2000 | 100 | 7 | 164.080 | 10.519 | 103.077 | 18 |
| 4000 | 100 | 7 | 187.506 | 20.134 | 111.825 | 20 |
| 8000 | 100 | 7 | 237.135 | 40.517 | 126.434 | 18 |
| 10000 | 100 | 7 | 234.835 | 48.335 | 122.164 | 24 |

The distributed path uses the same explicit chunked intersection logic on each
shard. For the 10K distributed run, shard logs show 87 actual targeted
decryptions versus 2,000,000 possible all-pairs decryptions.

The distributed coordinator JSON field `peak_ram_per_shard_mb` is not a
reliable RAM measurement in the final evidence bundle and should not be cited.

## Microsoft APSI Baseline

Microsoft APSI was run on the same `psi-compare` `e2-highmem-8` VM for the 10K
comparison.

| protocol | m | n | online_ms | receiver_peak_rss_kb | communication_total_kb | matches |
|---|---:|---:|---:|---:|---:|---:|
| Microsoft APSI | 10000 | 100 | 319 | 19956 | 1085 | 10 |

APSI is an HE-based asymmetric PSI baseline, not a laconic PSI baseline. Compare
phase definitions carefully: APSI online time is not the same metric as full
LE-PSI setup plus query time.

## Reproduce Core Checks

Run package tests:

```bash
go test ./pkg/LE ./pkg/matrix ./pkg/psi
```

Regenerate support figures:

```bash
python3 docs/paper_artifact/figures/generate_figures.py
```

Verify the evidence bundle:

```bash
cd comparative_baselines/results/evidence/psi_repro_20260515_145900
sha256sum -c SHA256SUMS
```

On macOS:

```bash
shasum -a 256 -c SHA256SUMS
```

## Run Benchmarks

Single-node LE-PSI:

```bash
bash comparative_baselines/lepsi_single_node/benchmark.sh
```

Microsoft APSI baseline, run on the GCE comparison VM:

```bash
bash comparative_baselines/apsi/run_apsi_10k.sh
```

Distributed remote chunked suite, run from the configured coordinator after
shard servers are deployed and `SHARD_URLS` is set:

```bash
SIZES="1000 2000 4000 8000 10000" N=100 K=7 \
  RUN_LABEL=b52740c_chunked \
  bash distributed_gce/remote_coord_chunked_suite.sh
```

## Repository Layout

```text
pkg/LE/                  Laconic encryption primitives
pkg/matrix/              Ring/matrix helpers
pkg/psi/                 PSI server, client, chunked intersection, serialization
comparative_baselines/   Same-machine baseline scripts and final evidence
distributed_gce/         GCE coordinator/shard implementation and run scripts
docs/paper_artifact/     Paper-support dossier, figures, and source data
docs/legacy/             Notes about removed superseded documents
cmd/Flare/               Proof-of-concept demo CLI
```

## References

- DKLLMR-style laconic PSI and laconic encryption literature.
- Lattigo v3: `github.com/tuneinsight/lattigo/v3`.
- Microsoft APSI: `https://github.com/microsoft/APSI`.

## License

MIT License. See `LICENSE`.
