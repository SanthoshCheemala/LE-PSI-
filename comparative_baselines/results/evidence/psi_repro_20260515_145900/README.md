# PSI Reproducibility Evidence - 2026-05-15

This folder is the active reproducibility bundle for the LE-PSI optimization audit and reruns.

## Code State

- LE-PSI single-node benchmark commit: `cea7d83`
- Distributed chunked rerun code commit: `b52740c`
- Current branch after remote distributed suite runner: `bae5e17`
- `b52740c` switches distributed shards to the same chunked optimized PSI path; `bae5e17` adds the remote suite runner.
- Correctness test added: `pkg/psi/tree_build_test.go`
- `go test ./pkg/LE ./pkg/matrix ./pkg/psi` passed locally and on `psi-compare`.

## Single-Node LE-PSI Rerun

Hardware: `psi-compare`, `e2-highmem-8`, 8 vCPUs, ~62.8 GiB RAM.

| m | n | total_sec | init_sec | enc_sec | intersect_sec | peak_rss_mb | matches | actual_dec_calls |
|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| 1000 | 100 | 19.944 | 6.509 | 12.316 | 1.120 | 6141 | 10 | 10 |
| 2000 | 100 | 24.908 | 11.520 | 10.775 | 2.612 | 7532 | 10 | 10 |
| 4000 | 100 | 41.877 | 23.438 | 11.278 | 7.161 | 11007 | 10 | 10 |
| 8000 | 100 | 74.639 | 46.274 | 9.409 | 18.956 | 17162 | 10 | 10 |
| 10000 | 100 | 94.516 | 56.496 | 9.364 | 28.655 | 20764 | 10 | 10 |

Important caveat: the benchmark client generator uses controlled non-overlap items whose candidate leaves avoid occupied server leaves. This makes the targeted-decryption count exactly the intended overlap count. The runtime is real, but the paper should describe this as a controlled leaf-filtered benchmark or rerun with random non-overlap clients for a less favorable collision profile.

## Single-Node Random-Client Diagnostic

A follow-up random-client 10K diagnostic was run on 2026-05-16 on the same
`psi-compare` `e2-highmem-8` machine after adding explicit correctness counters
in commit `80a0527`.

Evidence:

- `lepsi_single_random/lepsi_single_random_20260516_rerun/lepsi_m10000_n100_random.json`
- `lepsi_single_random/lepsi_single_random_20260516_rerun/run_m10000_random.log`
- `lepsi_single_random/lepsi_single_random_20260516_rerun/full_run.log`

| m | n | client_mode | total_sec | peak_rss_mb | expected_intersection | matches_found | false_positive_count | false_negative_count | correctness_passed | actual_dec_calls |
|---:|---:|---|---:|---:|---:|---:|---:|---:|---|---:|
| 10000 | 100 | random | 89.553 | 21938 | 10 | 13 | 3 | 0 | false | 13 |

This is diagnostic evidence, not a replacement final performance row. It shows
that random non-overlap client items can collide with occupied target leaves and
be counted as matches by the current leaf-only optimized path. The paper should
not claim random-workload PSI correctness for this optimized path unless an
additional item-equality check or collision-handling layer is implemented and
rerun.

## Comparative Baselines

Microsoft APSI 10K was rerun on the same `psi-compare` VM.

| protocol | m | n | online_ms | receiver_peak_rss_kb | communication_total_kb | matches |
|---|---:|---:|---:|---:|---:|---:|
| Microsoft APSI | 10000 | 100 | 319 | 19956 | 1085 | 10 |

KKRT/libPSI was attempted but failed during upstream dependency build before a runnable `frontend.exe` was produced. ALOS22/RELIC is marked as not rerun here because previous attempts built the demo but failed at runtime; invasive fixes would not be a fair published-code comparison.

## Distributed LE-PSI Rerun

Hardware: one coordinator plus seven `e2-highmem-4` shard VMs across `us-east1-c` and `us-east1-d`.

The final remote chunked run completed for all requested sizes. Evidence is in `lepsi_distributed/chunked_b52740c_20260515_170750/`, including coordinator logs, JSON outputs, wall-clock files, copied shard logs, and summary TSV files.

| m | n | shards | total_sec | init_sec | intersect_sec | wall_ms | matches |
|---:|---:|---:|---:|---:|---:|---:|---:|
| 1000 | 100 | 7 | 158.108 | 6.070 | 98.064 | 158133 | 16 |
| 2000 | 100 | 7 | 164.080 | 10.519 | 103.077 | 164103 | 18 |
| 4000 | 100 | 7 | 187.506 | 20.134 | 111.825 | 187550 | 20 |
| 8000 | 100 | 7 | 237.135 | 40.517 | 126.434 | 237163 | 18 |
| 10000 | 100 | 7 | 234.835 | 48.335 | 122.164 | 234865 | 24 |

Shard logs confirm the optimized path with `mode=explicit_chunked`, `chunk_size=256`, `workers=4`, and leaf-indexed targeted decryptions. For the 10K run, shard logs sum to 87 actual targeted decryptions versus 2,000,000 possible all-pairs decryptions. The coordinator JSON field `peak_ram_per_shard_mb` remains `0` and should not be used as a RAM measurement.

Important caveat: these distributed runs use the distributed dataset path, not the controlled 10-match single-node generator, so the observed match counts differ from the single-node table.
