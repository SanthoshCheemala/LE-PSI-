# LE-PSI Paper Support Dossier

Date prepared: 2026-05-16

Purpose: this is a technical support document for manuscript writing. It is not a journal-paper draft. It answers the audit questions in `/Users/santhoshcheemala/ALL_IN_ONE/doubts.txt` using the current repository, final benchmark logs, and final evidence bundle.

Repository state used for this dossier:

- Working branch for cleanup: `codex/paper-artifact-cleanup`
- Final optimized branch before cleanup: `optimizations-v2`
- Latest optimization commit on that branch: `bae5e17`
- Single-node result commit recorded in JSON: `cea7d83`
- Distributed chunked shard code commit: `b52740c`
- Remote distributed suite runner commit: `bae5e17`
- Evidence root: `comparative_baselines/results/evidence/psi_repro_20260515_145900/`

## Urgent Answers First

1. Final single-node GCE logs and JSON:
   - Summary: `comparative_baselines/results/evidence/psi_repro_20260515_145900/lepsi_single/summary.json`
   - 10K JSON: `comparative_baselines/results/evidence/psi_repro_20260515_145900/lepsi_single/lepsi_m10000_n100.json`
   - 10K log: `comparative_baselines/results/evidence/psi_repro_20260515_145900/lepsi_single/run_m10000.log`
   - Full run log: `comparative_baselines/results/evidence/psi_repro_20260515_145900/lepsi_single/full_run.log`

2. Final distributed GCE logs and JSON:
   - Root: `comparative_baselines/results/evidence/psi_repro_20260515_145900/lepsi_distributed/chunked_b52740c_20260515_170750/`
   - 10K coordinator log: `.../20260515_172245_m10000_n100_K7_b52740c_chunked/coordinator.log`
   - 10K JSON: `.../20260515_172245_m10000_n100_K7_b52740c_chunked/distributed_2026-05-15_17-22-45_m10000_n100_K7.json`
   - Shard logs: `.../shard_logs/shard_0.log` through `shard_6.log`
   - Distributed summary: `.../summary.tsv`
   - Shard decryption-call summary: `.../shard_dec_calls.tsv`

3. Final machine types:
   - Single-node: `psi-compare`, `e2-highmem-8`, `us-east1-b`, 8 vCPUs, 62.8059 GiB RAM.
   - Distributed: one coordinator plus seven `e2-highmem-4` shard VMs across `us-east1-c` and `us-east1-d`.

4. Final code parameters:
   - `D=256`, `qBits=58`, `q=180143985094819841`, `N=4`, `sigma=1073741824 = 2^30`, Lattigo `v3.0.6`, SHA-256-derived `uint64` identifiers.
   - `D=2048` exists behind `PSI_SECURITY_LEVEL=128`, but no final 1K-10K GCE benchmark was run at `D=2048`.

5. Final APSI result:
   - Same VM as LE-PSI single-node: `psi-compare`, `e2-highmem-8`.
   - 10K online time: `319 ms`.
   - Receiver peak RSS: `19,956 KB`.
   - Communication: `1,085 KB` total.
   - Matches: `10`.
   - Evidence: `comparative_baselines/results/evidence/psi_repro_20260515_145900/comparative/apsi/apsi_m10000_n100.json`.

6. Claim wording:
   - Safe: "implementation and empirical evaluation of DKLLMR23-style Ring-LWE laconic PSI."
   - Avoid: "first post-quantum PSI", "first Ring-LWE PSI", "first PQ PSI implementation."
   - If a "first" claim is necessary, use only a cautious form after literature verification: "to the best of our knowledge, the first implementation-oriented evaluation of DKLLMR23-style Ring-LWE laconic PSI."

7. Suggested technical title:
   - "Laconic Private Set Intersection from Ring-LWE: Implementation and Empirical Evaluation"

## A. Final Single-Node GCE Result

Final single-node machine:

- VM: `psi-compare`
- Project: `lepsi-distributed-493617`
- Zone: `us-east1-b`
- Machine type: `e2-highmem-8`
- vCPUs: `8`
- RAM: `62.80588912963867 GiB` in JSON, `62Gi` in `free -h`
- OS: the scripts provision Debian 12. The final evidence records machine type and Go version, but not `/etc/os-release`.
- Go: `go version go1.24.1 linux/amd64`
- Lattigo: `github.com/tuneinsight/lattigo/v3 v3.0.6` in `go.mod:5-8`

Final single-node mode:

- `explicit_chunked=true`
- `leaf_indexed_filtering=true`
- `cuckoo_mode=true`
- Final benchmark calls `psi.ServerInitializeChunked` and `psi.DetectIntersectionChunkedWithContext` in `scalability_tests/bench_10k.go:146-173`.

Final 10K parameters:

| Field | Value |
|---|---:|
| `m` | 10000 |
| `n` | 100 |
| `D` | 256 |
| `qBits` | 58 |
| `q` | 180143985094819841 |
| `N` | 4 |
| `sigma` | 1073741824 |
| `chunk_size` | 256 |
| `worker_count` | 8 |

Final 10K measured times:

| Field | Value |
|---|---:|
| `init_sec` | 56.496360652 |
| `enc_sec` | 9.364191907 |
| `intersect_sec` | 28.655249636 |
| `total_sec` | 94.516296865 |
| `total_min` | 1.575271614 |

Final 10K memory:

| Field | Value | Meaning |
|---|---:|---|
| `peak_rss_mb` | 20764 | OS resident set size sampled from `/proc/self/status` `VmRSS` by the benchmark process. |
| `peak_heap_mb` | 20175 | Go `runtime.MemStats.HeapAlloc` sampled by the benchmark process. |
| `total_alloc_mb` | 55059 | Cumulative Go allocation across the run, not live memory. |

Memory recording code:

- The GCE benchmark source is generated inside `comparative_baselines/lepsi_single_node/benchmark.sh`.
- `currentRSSMB()` reads `/proc/self/status` and extracts `VmRSS` in `comparative_baselines/lepsi_single_node/benchmark.sh:131-147`.
- `startMemoryMonitor()` samples RSS and Go heap in `comparative_baselines/lepsi_single_node/benchmark.sh:149-180`.
- `scalability_tests/bench_10k.go:57-66` records peak heap for the tracked standalone benchmark.

Correctness:

- Final 10K single-node result completed successfully.
- Expected matches: `10`.
- Found matches: `10`.
- False positives: `0` for the controlled benchmark generator.
- False negatives: `0` for the controlled benchmark generator.
- Important caveat: the single-node benchmark deliberately chooses non-overlap client items whose candidate leaves avoid occupied server leaves. This makes `actual_dec_calls` equal the intended overlap count. This is valid for proving the optimized path ran, but the paper should describe it as a controlled leaf-filtered benchmark.

Single-node final table:

| m | n | total_sec | init_sec | enc_sec | intersect_sec | peak_rss_mb | matches | actual_dec_calls |
|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| 1000 | 100 | 19.944 | 6.509 | 12.316 | 1.120 | 6141 | 10 | 10 |
| 2000 | 100 | 24.908 | 11.520 | 10.775 | 2.612 | 7532 | 10 | 10 |
| 4000 | 100 | 41.877 | 23.438 | 11.278 | 7.161 | 11007 | 10 | 10 |
| 8000 | 100 | 74.639 | 46.274 | 9.409 | 18.956 | 17162 | 10 | 10 |
| 10000 | 100 | 94.516 | 56.496 | 9.364 | 28.655 | 20764 | 10 | 10 |

## B. Single-Node Architecture Clarity

The final single-node benchmark uses explicit chunk-batched witness processing.

Evidence:

- Benchmark entry point: `scalability_tests/bench_10k.go:110-232`.
- It calls chunked server initialization at `scalability_tests/bench_10k.go:146-153`.
- It calls chunked intersection at `scalability_tests/bench_10k.go:167-178`.
- The chunk loop is in `pkg/psi/server.go:786-840`.
- Witnesses are generated inside the chunk worker at `pkg/psi/server.go:813-816`.
- Chunk-local state is released after `wg.Wait()`, and `runtime.GC()` is optionally called at `pkg/psi/server.go:835-839`.

It is not the old "one goroutine per server record plus global semaphore" path for the final result.

Legacy/eager witness path:

- `ServerInitialize` still exists and calls `serverInitialize(..., true)` at `pkg/psi/server.go:320-322`.
- The eager witness arrays are allocated at `pkg/psi/server.go:543-560`.
- This path is legacy/API-compatible, not the final benchmark path.
- The final path uses `ServerInitializeChunked`, which calls `serverInitialize(..., false)` at `pkg/psi/server.go:324-326`.
- In normal chunked mode, `serverInitialize` delegates to direct in-memory tree build at `pkg/psi/server.go:445-448`.

Recommended paper wording:

- Use: "Memory is governed by the active chunk configuration, the in-memory Merkle tree, private keys, and temporary witness state."
- Avoid: "constant memory" or "only governed by worker count."

## C. CPU Usage and Worker Configuration

Final hot path does not intentionally reserve 70% or 80% of CPU cores.

Evidence:

- Single-node benchmark default worker count is `runtime.NumCPU()` unless `LEPSI_WORKERS` is set: `scalability_tests/bench_10k.go:111-115`.
- Distributed shard intersection default worker count is `runtime.NumCPU()` unless `LEPSI_WORKERS` is set: `distributed_gce/shard/main.go:222-228`.
- Distributed shard server calls `runtime.GOMAXPROCS(runtime.NumCPU())`: `distributed_gce/shard/main.go:286-287`.
- Distributed coordinator calls `runtime.GOMAXPROCS(runtime.NumCPU())`: `distributed_gce/coordinator/main.go:368-370`.

Remaining CPU/memory logic:

- `CalculateOptimalWorkers` still exists for default library paths in `pkg/psi/helpers.go:192-240`.
- It uses all detected CPU cores by default at `pkg/psi/helpers.go:197-207`.
- In `PSI_SECURITY_LEVEL=128`, it caps workers at 4 at `pkg/psi/helpers.go:208-211`.
- It also uses an 85% RAM safety calculation at `pkg/psi/helpers.go:213-239`.

Recommended paper wording:

"The optimized benchmark uses available cores by default, with worker count explicitly logged and overridable through `LEPSI_WORKERS`; chunk size controls temporary witness pressure."

## D. Final Distributed GCE Result

Final distributed hardware:

- Coordinator: `e2-highmem-4`.
- Shards: seven `e2-highmem-4` VMs.
- Zones: `us-east1-c` and `us-east1-d`.
- Evidence of VM stop state: `comparative_baselines/results/evidence/psi_repro_20260515_145900/metadata/vm_status_after_distributed_stop.txt`.

Final distributed parameters:

| Field | Value |
|---|---:|
| `m` | 1000, 2000, 4000, 8000, 10000 |
| `n` | 100 |
| `K` | 7 |
| `D` | 256 |
| `qBits` | 58 |
| `chunk_size` per shard | 256 |
| `worker_count` per shard | 4 |

Final distributed results:

| m | n | shards | total_sec | init_sec | intersect_sec | wall_ms | matches |
|---:|---:|---:|---:|---:|---:|---:|---:|
| 1000 | 100 | 7 | 158.108 | 6.070 | 98.064 | 158133 | 16 |
| 2000 | 100 | 7 | 164.080 | 10.519 | 103.077 | 164103 | 18 |
| 4000 | 100 | 7 | 187.506 | 20.134 | 111.825 | 187550 | 20 |
| 8000 | 100 | 7 | 237.135 | 40.517 | 126.434 | 237163 | 18 |
| 10000 | 100 | 7 | 234.835 | 48.335 | 122.164 | 234865 | 24 |

Coordinator log coverage:

- Logs shard initialization start and success: `distributed_gce/coordinator/main.go:300-310`.
- Logs client encryption of 200 ciphertexts: `distributed_gce/coordinator/main.go:311-325`.
- Logs fan-out to shards and per-shard completion: `distributed_gce/coordinator/main.go:327-335` and `distributed_gce/coordinator/main.go:258-259`.
- Logs total runtime, init, intersect, matches, and result path: `distributed_gce/coordinator/main.go:354-410`.
- The exact phrase "all shard intersections completed" is not printed, but all shard completion lines and the final result are present.

Distributed memory:

- `peak_ram_per_shard_mb` in the final coordinator JSON is `0` and should not be cited.
- Shard `PeakRAMMB` is currently `runtime.MemStats.Sys` delta around init/intersect in `distributed_gce/shard/main.go:110-130` and `distributed_gce/shard/main.go:217-257`.
- Because this is a Go `Sys` delta, not OS peak RSS, it can be `0` or `1 MB` and is not a reliable RAM measurement.
- Use single-node RSS for memory claims, or rerun distributed with `/proc/self/status`/`/usr/bin/time -v` per shard if distributed memory is required.

Code commit relationship:

- Single-node final JSON records `cea7d83`.
- Distributed final shard optimized path is commit `b52740c`.
- Distributed suite runner is commit `bae5e17`.
- The protocol implementation changed between `cea7d83` and `b52740c` only to make distributed shards use the same chunked optimized path.

Do not report the old `12.7 min` number as the final clean distributed rerun. The final 10K distributed rerun is `3.91 min`.

## E. GCE Hardware Consistency

Final hardware to report:

- Single-node LE-PSI and APSI: `e2-highmem-8`, 8 vCPUs, about 62.8 GiB RAM, `us-east1-b`.
- Distributed LE-PSI: coordinator plus seven `e2-highmem-4` shard VMs, `us-east1-c`/`us-east1-d`.

Legacy hardware mentions:

- `n1-highmem-4`, `n1-highmem-8`, and `n2-highcpu-16` are not final result hardware.
- If scripts still mention them, treat them as old defaults or legacy. The cleaned README should not cite them as final hardware.

Supplementary artifact handling:

- Failed/orphaned runs should be excluded from final result tables or clearly separated as "attempted/legacy".
- The active evidence bundle already separates final runs and attempted baselines.

## F. Cryptographic Parameter Consistency

Final Lattigo version:

- `github.com/tuneinsight/lattigo/v3 v3.0.6` in `go.mod:5-8`.

Final sigma:

- `LE.Setup` sets `Sigma = math.Pow(2, 30)` at `pkg/LE/LE.go:52-68`.
- JSON records `sigma: 1073741824`.
- Do not write `sigma=3.2` unless the code is changed and rerun.

Final paper/security statement:

- `D=256` is a reduced-parameter evaluation mode.
- `D=2048` is available through `PSI_SECURITY_LEVEL=128` in `pkg/psi/parameters.go:55-63`.
- No final `D=2048` GCE benchmark was collected for the 1K-10K tables.
- Any Lattice Estimator statement must match `D`, `q`, `qBits`, and `sigma=2^30` actually used, or be clearly separated as a secure-parameter projection.

Hash length:

- The implementation canonicalizes records and hashes serialized values with SHA-256, then truncates to the first 64 bits using `binary.BigEndian.Uint64(hash[:8])` in `utils/data_preprocessor.go:163-172`.
- The PSI API and benchmark paths operate on `uint64` identifiers.
- Do not claim full 256-bit item matching without explaining this 64-bit derivation.

## G. Cuckoo Hashing Correctness

Final behavior:

- `placeCuckooLeaves` implements 2-choice placement with recursive displacement in `pkg/psi/server.go:94-131`.
- If placement cannot assign an item, it returns an error at `pkg/psi/server.go:121-127`.
- It does not silently overwrite a leaf.
- It does not currently rebuild with fresh salts.
- It does not use a stash.

Final logs:

- Single-node JSON prints `cuckoo_rebuilds`; final 10K value is `0`.
- It does not print `cuckoo_failures` or `stash_size`.
- Since all final runs succeeded, no Cuckoo placement failure occurred in those runs.

Recommended wording:

"The implementation uses two-choice Cuckoo placement with recursive displacement. If placement fails, the benchmark aborts rather than overwriting an occupied leaf. The evaluated runs had zero placement failures/rebuilds."

Do not claim "rebuild with fresh salts" unless implemented and rerun.

## H. Leaf-Indexed Filtering and Privacy Leakage

Target leaf exposure:

- `Cxtx` contains visible `TargetLeaf` at `pkg/psi/helpers.go:56-77`.
- Serialization includes `target_leaf` at `pkg/psi/cxtx_serial.go:17-21` and `pkg/psi/cxtx_serial.go:49-50`.
- Client encryption writes `TargetLeaf` into every ciphertext at `pkg/psi/client.go:80-81`.

Two ciphertexts per client item:

- Each client item produces two candidate leaves in `pkg/psi/client.go:37-42`.
- Total ciphertexts are `2 * Y_size` in `pkg/psi/client.go:44-45`.

Ciphertext order:

- The current client code does not shuffle ciphertexts. It emits adjacent pairs in `C[2*i]` and `C[2*i+1]`.
- Therefore, if network serialization preserves order, the server may infer that adjacent ciphertexts correspond to the two candidate leaves of one client item.

What the optimized server learns:

- Client set size through ciphertext count.
- Candidate/target leaf indices for client ciphertexts.
- Ciphertext ordering, including likely adjacent-pair relation unless shuffled later.
- Which server records matched.

What it should not learn directly:

- Raw client identifiers for non-matching items.
- Non-matching client values.

Security/privacy section recommendation:

State explicit implementation leakage:

"The optimized implementation exposes candidate leaf/bucket indices to the server for leaf-indexed routing. Under hash preimage resistance this does not reveal raw identifiers, but it is an explicit leakage term of the implementation. The current artifact also does not shuffle the two ciphertexts per client item."

Decryption-count reduction:

- All-pairs count: `m * 2n`.
- Targeted count: sum of ciphertexts whose `TargetLeaf` matches occupied server leaves.
- Code builds `leafToCts` in `pkg/psi/server.go:755-758`.
- It logs targeted/all-pairs counts in `pkg/psi/server.go:760-765`.
- Stats are recorded in `pkg/psi/server.go:767-779`.

Final single-node decryption reduction:

| m | all-pairs calls | actual targeted calls | reduction factor |
|---:|---:|---:|---:|
| 1000 | 200000 | 10 | 20000x |
| 2000 | 400000 | 10 | 40000x |
| 4000 | 800000 | 10 | 80000x |
| 8000 | 1600000 | 10 | 160000x |
| 10000 | 2000000 | 10 | 200000x |

Final distributed 10K decryption reduction:

- Actual targeted calls across shards: `87`.
- All-pairs possible across shards: `2,000,000`.
- Reduction factor: about `22,988.5x`.

These numbers are useful as an optimization result, with the controlled single-node caveat.

## I. Comparative Baseline: Microsoft APSI

APSI did run successfully on the same machine as LE-PSI single-node:

- VM: `psi-compare`
- Machine type: `e2-highmem-8`
- vCPUs: 8
- RAM: about 62 GiB
- Evidence JSON: `comparative_baselines/results/evidence/psi_repro_20260515_145900/comparative/apsi/apsi_m10000_n100.json`

APSI parameters:

- Source URL: `https://github.com/microsoft/APSI`
- Source commit recorded: `b967a12`
- Parameter file: `1M-256.json`
- Threads: `8`
- Protocol family: Microsoft APSI, BFV/SEAL-based asymmetric PSI.
- Same-setup comparison: yes for the 10K APSI run versus single-node LE-PSI.

Measured APSI values:

| Field | Value |
|---|---:|
| `m` | 10000 |
| `n` | 100 |
| `expected_intersection` | 10 |
| `matches_found` | 10 |
| `online_time_ms` | 319 |
| `receiver_peak_rss_kb` | 19956 |
| `sender_peak_rss_kb` | 0, unreliable/not measured |
| `communication_r_to_s_kb` | 1049 |
| `communication_s_to_r_kb` | 35 |
| `communication_total_kb` | 1085 |

APSI caveats:

- Use APSI as an HE-based asymmetric PSI baseline, not as a laconic PSI baseline.
- The JSON reports online time; do not compare against LE-PSI total/offline setup without explaining phase definitions.
- Raw logs exist in `comparative_baselines/results/evidence/psi_repro_20260515_145900/comparative/apsi/`.

## J. Comparative Analysis: Literature Baselines

Recommended treatment:

- KKRT: classical context only unless the upstream libPSI build is repaired reproducibly. Evidence: `comparative_baselines/results/evidence/psi_repro_20260515_145900/comparative/kkrt_libpsi/STATUS.md`.
- ALOS22/RELIC: related-work or published-artifact context only; no fair 10K runtime result in this artifact. Evidence: `comparative_baselines/results/evidence/psi_repro_20260515_145900/comparative/alos22_relic/STATUS.md`.
- HE-PSI Chen-Laine-Rindal: cite as related work; Microsoft APSI is the runnable Microsoft HE-family implementation used here.
- LEAP/RLWE/OKVS PSI: related work only unless a maintained source repository is identified and run on the same VM.

Recommended comparison table columns:

- Protocol
- Type/family
- Security assumption
- Laconic?
- Same setup?
- Source of numbers
- Runtime
- Memory
- Communication
- Caveat

## K. Final Claims and Wording

Safe claims:

- "Implementation and empirical evaluation of DKLLMR23-style Ring-LWE laconic PSI."
- "Lattice-based / plausibly post-quantum under Ring-LWE assumptions."
- "Reduced-parameter evaluation at `D=256`; secure-parameter mode available but not benchmarked at scale in the final GCE run."
- "Leaf-indexed routing reduces decryption attempts in the implementation."
- "FLARE is a proof-of-concept case study."

Avoid:

- "First post-quantum PSI."
- "First Ring-LWE PSI."
- "First PQ PSI implementation."
- "Constant memory."
- "Server learns only |C|" for the optimized implementation.
- "GDPR-compliant." Prefer "supports data-minimization-oriented workflows."

## L. Figures and Tables

Generated support figures:

- Runtime breakdown: `docs/paper_artifact/figures/output/single_node_runtime_breakdown.png` and `.pdf`.
- Peak RSS scaling: `docs/paper_artifact/figures/output/single_node_peak_rss.png` and `.pdf`.
- Decryption reduction: `docs/paper_artifact/figures/output/decryption_reduction.png` and `.pdf`.

Source data:

- `docs/paper_artifact/figures/source_data/single_node_summary.csv`
- `docs/paper_artifact/figures/source_data/distributed_dec_call_summary.csv`

Use tables rather than graphs for:

- Distributed runtime: coordinator/network/file-stream overhead makes a simple table clearer.
- APSI comparison: APSI online time is not phase-equivalent to LE-PSI total time.

Old profiling/bottleneck charts should be removed or clearly marked as historical. The final optimized evidence supersedes older "70% witness generation" and "77 worker" stories.

## M. Supplementary Artifact Status

Current artifact root:

- `comparative_baselines/results/evidence/psi_repro_20260515_145900/`

Exists:

- `README.md`
- `MANIFEST`
- `SHA256SUMS`
- `lepsi_single/` logs and JSON
- `lepsi_distributed/` final chunked logs and JSON
- `comparative/apsi/` APSI logs and JSON
- `comparative/kkrt_libpsi/STATUS.md`
- `comparative/alos22_relic/STATUS.md`
- `metadata/git_commit.txt`
- `metadata/vm_status_after_distributed_stop.txt`

Does not currently exist as named:

- `README_Supplementary.md`
- `MANIFEST.csv`
- `SHA256SUMS.txt`
- A zipped artifact

Recommendation:

- The current `README.md`, `MANIFEST`, and `SHA256SUMS` are enough for the repository artifact.
- If the journal requires a zip supplement, create it after final review from the evidence root and `docs/paper_artifact/`.

## N. GitHub/Repository Cleanup

Final branch strategy:

- Backup branch created before cleanup: `codex/backup-before-paper-artifact-20260516`.
- Cleanup branch: `codex/paper-artifact-cleanup`.
- Push target: PR branch, not direct push to `main`.

README cleanup requirements:

- Replace stale performance table with final 2026-05-15 values.
- State leaf-index leakage.
- State D=256 evaluation mode.
- State Lattigo `v3.0.6`, sigma `2^30`, SHA-256-derived `uint64`.
- Provide reproducibility commands for single-node, distributed, APSI, and checksum verification.

Stale claims to remove from GitHub-facing docs:

- `312GB to 6.5GB`
- `840GB`
- `25-node AWS`
- `70%/80% CPU reservation`
- `77 workers` as final behavior
- `constant memory`
- `first post-quantum PSI`
- `server learns only |C|`

## O. Paper Metadata

These items are not discoverable from the repository and should be filled by the author:

- Final title
- Author names and order
- Affiliations
- Corresponding author
- Target journal
- Highlights
- Graphical abstract
- Declaration of competing interest
- Data availability
- Code availability
- Funding statement
- Author contribution statement

Suggested title if useful:

"Laconic Private Set Intersection from Ring-LWE: Implementation and Empirical Evaluation"

## Reproducibility Commands

Single-node LE-PSI on `psi-compare`:

```bash
export PROJECT=lepsi-distributed-493617
export ZONE=us-east1-b
gcloud compute instances start psi-compare --project="$PROJECT" --zone="$ZONE"
gcloud compute ssh psi-compare --project="$PROJECT" --zone="$ZONE" --command='
  export PATH=$PATH:/usr/local/go/bin
  cd /tmp/lepsi-repo
  bash comparative_baselines/lepsi_single_node/benchmark.sh
'
```

Distributed remote chunked suite:

```bash
export PROJECT=lepsi-distributed-493617
export RUN_LABEL=b52740c_chunked
export SIZES="1000 2000 4000 8000 10000"
export N=100
export K=7
# Start shard servers, set SHARD_URLS to the seven internal shard URLs, then run:
gcloud compute ssh lepsi-coord-rerun --project="$PROJECT" --zone=us-east1-c --command='
  cd /tmp/lepsi
  SIZES="1000 2000 4000 8000 10000" N=100 K=7 RUN_LABEL=b52740c_chunked \
  SHARD_URLS="$SHARD_URLS" \
  bash distributed_gce/remote_coord_chunked_suite.sh
'
```

APSI 10K baseline:

```bash
gcloud compute ssh psi-compare --project=lepsi-distributed-493617 --zone=us-east1-b --command='
  cd /tmp/lepsi-repo
  bash comparative_baselines/apsi/run_apsi_10k.sh
'
```

Verify artifact checksums:

```bash
cd comparative_baselines/results/evidence/psi_repro_20260515_145900
sha256sum -c SHA256SUMS
```
