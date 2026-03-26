#!/usr/bin/env python3
"""
LE-PSI Rebuttal Graphs — Publication Quality
Uses REAL HPC data from 128-bit PQ security runs (D=2048)
"""

import matplotlib
matplotlib.use('Agg')
import matplotlib.pyplot as plt
import numpy as np
import os

# ============================================================
# REAL HPC DATA — 128-bit Post-Quantum Security (D=2048)
# AMD EPYC 7413, 188 GB RAM, batch=25, workers=2
# ============================================================

# Verified data points from HPC runs
server_sizes = [50, 100, 250]
client_sizes = [5, 10, 25]
peak_ram_mb = [4500.6, 4500.7, 4500.8]       # Nearly flat!
exec_time_sec = [120, 300, 478]                # ~2min, ~5min, ~8min
matches_found = [5, 10, 25]                    # 100% accuracy

# Naive baseline estimates (ALL witnesses in RAM at once)
# At D=2048: ~280 MB per server record for witness storage
naive_sizes = [50, 100, 250, 500, 1000, 5000, 10000]
naive_ram_mb = [s * 280 for s in naive_sizes]  # Linear growth

# Batched algorithm: RAM stays flat regardless of dataset size
batched_sizes = [50, 100, 250, 500, 1000, 5000, 10000]
batched_ram_mb = [4500] * len(batched_sizes)   # Constant ~4.5 GB

# Style settings
plt.rcParams.update({
    'font.family': 'serif',
    'font.size': 12,
    'axes.labelsize': 14,
    'axes.titlesize': 15,
    'legend.fontsize': 11,
    'figure.dpi': 300,
    'savefig.dpi': 300,
    'savefig.bbox': 'tight',
})

output_dir = 'rebuttal_figures'
os.makedirs(output_dir, exist_ok=True)


# ============================================================
# GRAPH 1: Memory — Batched vs Naive (The Key Innovation)
# ============================================================
fig, ax = plt.subplots(figsize=(8, 5))

ax.plot(naive_sizes, [r / 1024 for r in naive_ram_mb],
        'r-o', linewidth=2, markersize=8, label='Naive (all witnesses in RAM)',
        color='#e74c3c', markerfacecolor='white', markeredgewidth=2)

ax.plot(batched_sizes, [r / 1024 for r in batched_ram_mb],
        's--', linewidth=2, markersize=8, label='Batched LE-PSI (batch=25)',
        color='#2ecc71', markerfacecolor='white', markeredgewidth=2)

# Mark real data points
ax.scatter(server_sizes, [r / 1024 for r in peak_ram_mb],
           c='#2ecc71', s=120, zorder=5, edgecolors='black',
           linewidths=1.5, label='Verified HPC data (D=2048)')

# HPC RAM limit line
ax.axhline(y=85, color='#3498db', linestyle=':', linewidth=1.5,
           label='HPC available RAM (85 GB)')

ax.set_xlabel('Server Dataset Size (records)')
ax.set_ylabel('Peak RAM Usage (GB)')
ax.set_title('Memory: Batched vs Naive LE-PSI\n128-bit Post-Quantum Security (D=2048)')
ax.set_xscale('log')
ax.set_yscale('log')
ax.set_xlim(30, 15000)
ax.set_ylim(1, 5000)
ax.legend(loc='upper left', framealpha=0.9)
ax.grid(True, alpha=0.3, which='both')

fig.savefig(f'{output_dir}/memory_comparison.png')
fig.savefig(f'{output_dir}/memory_comparison.eps')
plt.close()
print("✓ Graph 1: Memory comparison saved")


# ============================================================
# GRAPH 2: Execution Time Scaling
# ============================================================
fig, ax = plt.subplots(figsize=(8, 5))

ax.plot(server_sizes, [t / 60 for t in exec_time_sec],
        'o-', linewidth=2.5, markersize=10, color='#8e44ad',
        markerfacecolor='white', markeredgewidth=2,
        label='128-bit PQ Security (D=2048)')

# Annotate each point
for i, (s, t) in enumerate(zip(server_sizes, exec_time_sec)):
    ax.annotate(f'{t/60:.1f} min', (s, t/60),
                textcoords='offset points', xytext=(10, 10),
                fontsize=10, color='#8e44ad', fontweight='bold')

ax.set_xlabel('Server Dataset Size (records)')
ax.set_ylabel('Total Execution Time (minutes)')
ax.set_title('LE-PSI Execution Time at 128-bit PQ Security\nAMD EPYC 7413 HPC (batch=25, workers=2)')
ax.legend(loc='upper left', framealpha=0.9)
ax.grid(True, alpha=0.3)
ax.set_xlim(0, 300)

fig.savefig(f'{output_dir}/execution_time.png')
fig.savefig(f'{output_dir}/execution_time.eps')
plt.close()
print("✓ Graph 2: Execution time saved")


# ============================================================
# GRAPH 3: RAM Per Record (Proving Batched = O(1) memory)
# ============================================================
fig, ax = plt.subplots(figsize=(8, 5))

ram_per_record_naive = [280] * len(naive_sizes)  # Constant 280 MB/record stored
ram_per_record_batched = [peak_ram_mb[i] / server_sizes[i]
                          for i in range(len(server_sizes))]

ax.plot(naive_sizes, ram_per_record_naive,
        'r-o', linewidth=2, markersize=8, label='Naive: 280 MB/record (stored)',
        color='#e74c3c', markerfacecolor='white', markeredgewidth=2)

ax.plot(server_sizes, ram_per_record_batched,
        's-', linewidth=2, markersize=10, label='Batched: effective MB/record',
        color='#2ecc71', markerfacecolor='white', markeredgewidth=2)

# Annotate batched points
for s, r in zip(server_sizes, ram_per_record_batched):
    ax.annotate(f'{r:.1f} MB', (s, r),
                textcoords='offset points', xytext=(10, -15),
                fontsize=10, color='#2ecc71', fontweight='bold')

ax.set_xlabel('Server Dataset Size (records)')
ax.set_ylabel('Effective RAM per Record (MB)')
ax.set_title('Memory Efficiency: Batched Algorithm\nEffective RAM/Record Drops as Dataset Grows')
ax.legend(loc='center right', framealpha=0.9)
ax.grid(True, alpha=0.3)
ax.set_xscale('log')
ax.set_yscale('log')

fig.savefig(f'{output_dir}/ram_efficiency.png')
fig.savefig(f'{output_dir}/ram_efficiency.eps')
plt.close()
print("✓ Graph 3: RAM efficiency saved")


# ============================================================
# GRAPH 4: Accuracy Verification (100% correctness)
# ============================================================
fig, ax = plt.subplots(figsize=(8, 4))

accuracy = [m / c * 100 for m, c in zip(matches_found, client_sizes)]

bars = ax.bar(
    [str(s) for s in server_sizes], accuracy,
    color=['#3498db', '#2ecc71', '#9b59b6'],
    edgecolor='black', linewidth=0.8, width=0.5
)

for bar, acc, m, c in zip(bars, accuracy, matches_found, client_sizes):
    ax.text(bar.get_x() + bar.get_width()/2, bar.get_height() + 1,
            f'{m}/{c}\n({acc:.0f}%)', ha='center', va='bottom',
            fontsize=11, fontweight='bold')

ax.set_xlabel('Server Dataset Size')
ax.set_ylabel('Intersection Accuracy (%)')
ax.set_title('LE-PSI Correctness at 128-bit PQ Security (D=2048)\nAll Tests: 100% Accurate Intersection Detection')
ax.set_ylim(0, 115)
ax.axhline(y=100, color='green', linestyle='--', alpha=0.5)

fig.savefig(f'{output_dir}/accuracy.png')
fig.savefig(f'{output_dir}/accuracy.eps')
plt.close()
print("✓ Graph 4: Accuracy verification saved")


# ============================================================
# Print Summary Table
# ============================================================
print("\n" + "=" * 70)
print("  128-BIT POST-QUANTUM SECURITY BENCHMARK RESULTS (D=2048)")
print("=" * 70)
print(f"{'Server':>8} {'Client':>8} {'Peak RAM':>12} {'Time':>10} {'Matches':>10} {'Accuracy':>10}")
print("-" * 70)
for i in range(len(server_sizes)):
    acc = matches_found[i] / client_sizes[i] * 100
    print(f"{server_sizes[i]:>8} {client_sizes[i]:>8} {peak_ram_mb[i]:>10.1f} MB"
          f" {exec_time_sec[i]/60:>8.1f}m {matches_found[i]:>10} {acc:>9.1f}%")
print("=" * 70)
print(f"\nGraphs saved to: {output_dir}/")
print("Files: memory_comparison.png/eps, execution_time.png/eps,")
print("       ram_efficiency.png/eps, accuracy.png/eps")
