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

Use the same machine type as the distributed shards for consistency:

```bash
gcloud compute instances create psi-compare \
  --machine-type=e2-highmem-4 \
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
  nohup bash /tmp/lepsi-repo/comparative_baselines/apsi/setup_and_benchmark.sh \
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

Each benchmark produces JSON files like:

```json
{
  "protocol": "Microsoft APSI (BFV/SEAL)",
  "server_dataset_size": 10000,
  "client_dataset_size": 100,
  "matches_found": 10,
  "receiver_online_time_ms": 1234,
  "sender_peak_rss_kb": 512000
}
```

## Paper Table Format

| m     | Protocol | Init (s) | Online (s) | Peak RAM (MB) |
|-------|----------|----------|------------|---------------|
| 1,000 | LE-PSI   | ...      | ...        | ...           |
| 1,000 | APSI     | ...      | ...        | ...           |
| 10,000| LE-PSI   | ...      | ...        | ...           |
| 10,000| APSI     | ...      | ...        | ...           |
