#!/usr/bin/env python3
"""Generate journal-support figures from collected LE-PSI evidence.

The figures are intentionally limited to plots that add information beyond the
tables: phase breakdown, memory scaling, and decryption reduction. APSI and
distributed comparisons stay tabular in the dossier because those measurements
come from different execution models.
"""

from __future__ import annotations

import csv
import json
from pathlib import Path

import matplotlib.pyplot as plt


ROOT = Path(__file__).resolve().parents[3]
EVIDENCE = ROOT / "comparative_baselines/results/evidence/psi_repro_20260515_145900"
SINGLE = EVIDENCE / "lepsi_single/summary.json"
DIST_DEC = EVIDENCE / "lepsi_distributed/chunked_b52740c_20260515_170750/shard_dec_calls.tsv"
OUT = ROOT / "docs/paper_artifact/figures/output"
DATA = ROOT / "docs/paper_artifact/figures/source_data"


def load_single() -> list[dict]:
    with SINGLE.open() as f:
        rows = json.load(f)
    return sorted(rows, key=lambda r: r["m"])


def write_csv(path: Path, rows: list[dict], fields: list[str]) -> None:
    with path.open("w", newline="") as f:
        writer = csv.DictWriter(f, fieldnames=fields)
        writer.writeheader()
        for row in rows:
            writer.writerow({field: row.get(field, "") for field in fields})


def save(fig: plt.Figure, name: str) -> None:
    OUT.mkdir(parents=True, exist_ok=True)
    fig.tight_layout()
    fig.savefig(OUT / f"{name}.png", dpi=220)
    fig.savefig(OUT / f"{name}.pdf")
    plt.close(fig)


def runtime_breakdown(rows: list[dict]) -> None:
    labels = [f"{r['m']:,}" for r in rows]
    x = list(range(len(rows)))
    init = [r["init_sec"] for r in rows]
    enc = [r["enc_sec"] for r in rows]
    inter = [r["intersect_sec"] for r in rows]

    fig, ax = plt.subplots(figsize=(7.2, 4.2))
    ax.bar(x, init, label="Server init", color="#3B6FB6")
    ax.bar(x, enc, bottom=init, label="Client encryption", color="#54A24B")
    bottom2 = [a + b for a, b in zip(init, enc)]
    ax.bar(x, inter, bottom=bottom2, label="Intersection", color="#E45756")
    ax.set_xticks(x, labels)
    ax.set_xlabel("Server set size m")
    ax.set_ylabel("Runtime (seconds)")
    ax.set_title("LE-PSI single-node runtime breakdown")
    ax.legend(frameon=False)
    ax.grid(axis="y", alpha=0.25)
    save(fig, "single_node_runtime_breakdown")


def memory_scaling(rows: list[dict]) -> None:
    m = [r["m"] for r in rows]
    rss = [r["peak_rss_mb"] for r in rows]

    fig, ax = plt.subplots(figsize=(7.2, 4.2))
    ax.plot(m, rss, marker="o", color="#2F6F6D", linewidth=2.2)
    ax.set_xlabel("Server set size m")
    ax.set_ylabel("Peak RSS (MB)")
    ax.set_title("LE-PSI single-node peak RSS")
    ax.grid(alpha=0.25)
    save(fig, "single_node_peak_rss")


def decryption_reduction(rows: list[dict]) -> None:
    m = [r["m"] for r in rows]
    all_pairs = [r["total_possible_dec_calls"] for r in rows]
    actual = [r["actual_dec_calls"] for r in rows]

    fig, ax = plt.subplots(figsize=(7.2, 4.2))
    ax.plot(m, all_pairs, marker="o", label="All-pairs Dec calls", color="#7B3F98", linewidth=2.2)
    ax.plot(m, actual, marker="o", label="Leaf-targeted Dec calls", color="#D9892B", linewidth=2.2)
    ax.set_yscale("log")
    ax.set_xlabel("Server set size m")
    ax.set_ylabel("Decryption calls (log scale)")
    ax.set_title("Leaf-indexed filtering reduces decryption attempts")
    ax.legend(frameon=False)
    ax.grid(alpha=0.25, which="both")
    save(fig, "decryption_reduction")


def distributed_dec_summary() -> list[dict]:
    totals: dict[int, dict[str, int]] = {}
    with DIST_DEC.open() as f:
        reader = csv.DictReader(f, delimiter="\t")
        for row in reader:
            m = int(row["m"])
            entry = totals.setdefault(m, {"m": m, "matches_sum": 0, "actual_dec_calls_sum": 0, "total_possible_dec_calls_sum": 0})
            entry["matches_sum"] += int(row["matches"])
            entry["actual_dec_calls_sum"] += int(row["actual_dec_calls"])
            entry["total_possible_dec_calls_sum"] += int(row["total_possible_dec_calls"])

    out = []
    for m in sorted(totals):
        row = totals[m]
        actual = row["actual_dec_calls_sum"]
        row["reduction_factor"] = round(row["total_possible_dec_calls_sum"] / actual, 3) if actual else ""
        out.append(row)
    return out


def main() -> None:
    DATA.mkdir(parents=True, exist_ok=True)
    OUT.mkdir(parents=True, exist_ok=True)

    rows = load_single()
    write_csv(
        DATA / "single_node_summary.csv",
        rows,
        [
            "m",
            "n",
            "total_sec",
            "init_sec",
            "enc_sec",
            "intersect_sec",
            "peak_rss_mb",
            "matches_found",
            "expected_intersection",
            "actual_dec_calls",
            "total_possible_dec_calls",
            "git_commit",
            "machine_type",
            "vcpus",
            "ram_gb",
        ],
    )

    dist_rows = distributed_dec_summary()
    write_csv(
        DATA / "distributed_dec_call_summary.csv",
        dist_rows,
        ["m", "matches_sum", "actual_dec_calls_sum", "total_possible_dec_calls_sum", "reduction_factor"],
    )

    runtime_breakdown(rows)
    memory_scaling(rows)
    decryption_reduction(rows)


if __name__ == "__main__":
    main()
