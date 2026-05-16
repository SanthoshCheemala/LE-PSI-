# Comparative Baselines

This directory contains scripts for running same-setup comparisons between
LE-PSI and Microsoft APSI on a single GCE VM.

## Why Single-Node?

The paper's reviewer specifically asked for a **fair comparison** — running both
protocols on identical hardware eliminates infrastructure advantages. The distributed
benchmarks (in `distributed_gce/`) show LE-PSI's scalability, while these single-node
results show how the core protocol compares head-to-head.

## Setup

### 1. Create a GCE VM

Use the final single-node comparison machine type:

```bash
gcloud compute instances create psi-compare \
  --machine-type=e2-highmem-8 \
  --zone=us-east1-b \
  --image-family=debian-12 \
  --image-project=debian-cloud \
  --boot-disk-size=100GB \
  --project=lepsi-distributed-493617
```

### 2. Upload Code

```bash
# Upload the entire repo (needed for LE-PSI Go build)
gcloud compute scp --recurse \
  /Users/santhoshcheemala/ALL_IN_ONE/Research_Implimentation/PSI/ \
  psi-compare:/tmp/lepsi-repo/ \
  --zone=us-east1-b

# Also ensure Go is installed on the VM
gcloud compute ssh psi-compare --zone=us-east1-b --command="
  wget -q https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
  sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz
  echo 'export PATH=\$PATH:/usr/local/go/bin' >> ~/.bashrc
"
```

### 3. Run APSI Benchmark

```bash
gcloud compute ssh psi-compare --zone=us-east1-b --command="
  nohup bash /tmp/lepsi-repo/comparative_baselines/apsi/run_apsi_10k.sh \
    > /tmp/apsi_full.log 2>&1 &
"
```

### 4. Run LE-PSI Benchmark (after APSI finishes)

```bash
gcloud compute ssh psi-compare --zone=us-east1-b --command="
  export PATH=\$PATH:/usr/local/go/bin
  cd /tmp/lepsi-repo
  nohup bash comparative_baselines/lepsi_single_node/benchmark.sh \
    > /tmp/lepsi_single_full.log 2>&1 &
"
```

The final 2026-05-15 LE-PSI evidence uses the default controlled client
generator. For a follow-up random-client stress run that includes natural
target-leaf collisions, set `CLIENT_MODE=random`:

```bash
gcloud compute ssh psi-compare --zone=us-east1-b --command="
  export PATH=\$PATH:/usr/local/go/bin
  cd /tmp/lepsi-repo
  CLIENT_MODE=random CLIENT_SEED=20260515 LEPSI_SIZES='10000' \
    bash comparative_baselines/lepsi_single_node/benchmark.sh
"
```

The 2026-05-16 same-machine random diagnostic produced
`correctness_passed=false` with `false_positive_count=3`, so use this mode as a
correctness stress test for the current leaf-only optimized path, not as a
headline performance row.

### 5. Collect Results

```bash
mkdir -p comparative_baselines/results
gcloud compute scp psi-compare:/tmp/apsi_results/*.json comparative_baselines/results/
gcloud compute scp psi-compare:/tmp/lepsi_single_results/*.json comparative_baselines/results/
```

### 6. Stop VM

```bash
gcloud compute instances stop psi-compare --zone=us-east1-b --quiet
```

## Expected Output

Final evidence from the 2026-05-15 run is stored under
`comparative_baselines/results/evidence/psi_repro_20260515_145900/`.

The APSI runner defaults to `M=10000` and `N=100` so comparative baselines stay
focused on the same 10K setting used in the final table. It produces JSON files
like:

```json
{
  "protocol": "Microsoft APSI (BFV/SEAL)",
  "server_dataset_size": 10000,
  "client_dataset_size": 100,
  "matches_found": 10,
  "online_time_ms": 319,
  "receiver_peak_rss_kb": 19956,
  "communication_total_kb": 1085
}
```

## Paper Table Format

Use phase labels carefully. APSI reports online time; LE-PSI reports server
initialization, client encryption, and intersection separately.
Also label LE-PSI single-node results by `client_mode`; controlled and random
client workloads should not be mixed in one row without explanation.

| m | Protocol | Machine | Reported time field | Peak memory field | Communication |
|---:|---|---|---|---|---|
| 10000 | LE-PSI | `e2-highmem-8` | `total_sec`, plus phase fields | `peak_rss_mb` | protocol-specific, not measured in this runner |
| 10000 | APSI | `e2-highmem-8` | `online_time_ms` | `receiver_peak_rss_kb` | `communication_total_kb` |
