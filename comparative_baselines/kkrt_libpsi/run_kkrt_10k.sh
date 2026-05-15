#!/usr/bin/env bash
# Run KKRT16 PSI from osu-crypto/libPSI for a 10K x 100 comparison.
set -euo pipefail

RESULTS="${RESULTS:-/tmp/psi_comparative_10k/kkrt_libpsi}"
LIBPSI_DIR="${LIBPSI_DIR:-/tmp/libPSI}"
M="${M:-10000}"
N="${N:-100}"
OVERLAP="${OVERLAP:-10}"
THREADS="${THREADS:-$(nproc)}"
PORT="${PORT:-1212}"

mkdir -p "$RESULTS"

echo "==== KKRT16/libPSI 10K benchmark ===="
echo "results=$RESULTS m=$M n=$N overlap=$OVERLAP threads=$THREADS"
echo "machine=$(hostname) cpus=$(nproc) ram_gb=$(free -g | awk '/Mem/{print $2}') date=$(date -u +%Y-%m-%dT%H:%M:%SZ)"

sudo apt-get update -qq
sudo apt-get install -y -qq git cmake g++ make python3 libssl-dev libboost-all-dev

if [[ ! -d "$LIBPSI_DIR/.git" ]]; then
  git clone --recursive https://github.com/osu-crypto/libPSI.git "$LIBPSI_DIR"
fi

cd "$LIBPSI_DIR"
python3 build.py --par="$THREADS" 2>&1 | tee "$RESULTS/build.log"

FRONTEND="$(find "$LIBPSI_DIR/out/build" -path '*/frontend/frontend.exe' -type f | head -1)"
if [[ -z "$FRONTEND" ]]; then
  echo "frontend.exe not found after build" >&2
  exit 2
fi

python3 - "$RESULTS/sender.csv" "$RESULTS/receiver.csv" "$M" "$N" "$OVERLAP" <<'PY'
import sys
sender, receiver, m, n, overlap = sys.argv[1], sys.argv[2], int(sys.argv[3]), int(sys.argv[4]), int(sys.argv[5])
def hx(x): return f"{x:032x}"
with open(sender, "w") as s:
    for i in range(1, m + 1):
        s.write(hx(i) + "\n")
with open(receiver, "w") as r:
    for i in range(1, overlap + 1):
        r.write(hx(i) + "\n")
    for i in range(overlap + 1, n + 1):
        r.write(hx(m + 1000 + i) + "\n")
PY

rm -f "$RESULTS/output.csv"

START_NS=$(date +%s%N)
/usr/bin/time -v -o "$RESULTS/receiver_time.txt" \
  "$FRONTEND" -kkrt -r 1 -server 1 -csv -ip "127.0.0.1:$PORT" \
  -in "$RESULTS/receiver.csv" -out "$RESULTS/output.csv" \
  -senderSize "$M" -receiverSize "$N" -t "$THREADS" \
  > "$RESULTS/receiver.log" 2>&1 &
RECEIVER_PID=$!

sleep 2
/usr/bin/time -v -o "$RESULTS/sender_time.txt" \
  "$FRONTEND" -kkrt -r 0 -server 0 -csv -ip "127.0.0.1:$PORT" \
  -in "$RESULTS/sender.csv" \
  -senderSize "$M" -receiverSize "$N" -t "$THREADS" \
  > "$RESULTS/sender.log" 2>&1

wait "$RECEIVER_PID"
END_NS=$(date +%s%N)

MATCHES=$(wc -l < "$RESULTS/output.csv" 2>/dev/null || echo 0)
TOTAL_MS=$(( (END_NS - START_NS) / 1000000 ))
SENDER_RSS_KB=$(awk 'tolower($0) ~ /maximum resident/ {print $NF}' "$RESULTS/sender_time.txt" 2>/dev/null || echo 0)
RECEIVER_RSS_KB=$(awk 'tolower($0) ~ /maximum resident/ {print $NF}' "$RESULTS/receiver_time.txt" 2>/dev/null || echo 0)

cat > "$RESULTS/kkrt_libpsi_m${M}_n${N}.json" <<EOF
{
  "protocol": "KKRT16 (osu-crypto/libPSI)",
  "source_url": "https://github.com/osu-crypto/libPSI",
  "source_commit": "$(git rev-parse --short HEAD)",
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "machine": "$(hostname)",
  "machine_type": "$(curl -fs -H 'Metadata-Flavor: Google' http://metadata.google.internal/computeMetadata/v1/instance/machine-type 2>/dev/null | awk -F/ '{print $NF}')",
  "vcpus": $(nproc),
  "ram_gb": $(free -g | awk '/Mem/{print $2}'),
  "server_dataset_size": $M,
  "client_dataset_size": $N,
  "expected_intersection": $OVERLAP,
  "matches_found": $MATCHES,
  "total_time_ms": $TOTAL_MS,
  "sender_peak_rss_kb": ${SENDER_RSS_KB:-0},
  "receiver_peak_rss_kb": ${RECEIVER_RSS_KB:-0},
  "threads": $THREADS,
  "notes": "File-based localhost run; communication bytes are not emitted by libPSI frontend."
}
EOF

cat "$RESULTS/kkrt_libpsi_m${M}_n${N}.json"
