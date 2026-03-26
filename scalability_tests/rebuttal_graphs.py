#!/usr/bin/env python3
"""LE-PSI Rebuttal Graphs — Clean academic style"""

import matplotlib
matplotlib.use('Agg')
import matplotlib.pyplot as plt
import numpy as np
import os

# === REAL HPC DATA (D=2048, 128-bit PQ Security) ===
server_sizes = [50, 100, 250]
client_sizes = [5, 10, 25]
peak_ram_mb = [4500.6, 4500.7, 4500.8]
exec_time_sec = [120, 300, 478]
matches_found = [5, 10, 25]

# Naive baseline (all witnesses in RAM)
naive_sizes = [50, 100, 250, 500, 1000, 5000, 10000]
naive_ram_gb = [s * 280 / 1024 for s in naive_sizes]

# Batched: flat
batched_sizes = [50, 100, 250, 500, 1000, 5000, 10000]
batched_ram_gb = [4.5] * len(batched_sizes)

# Style
plt.rcParams.update({
    'font.family': 'serif',
    'font.size': 13,
    'axes.labelsize': 14,
    'axes.titlesize': 14,
    'legend.fontsize': 11,
    'figure.dpi': 300,
    'savefig.dpi': 300,
    'savefig.bbox': 'tight',
})

out = 'rebuttal_figures'
os.makedirs(out, exist_ok=True)

# ── GRAPH 1: Memory Comparison ──
fig, ax = plt.subplots(figsize=(7, 4.5))
ax.plot(naive_sizes, naive_ram_gb, '-o', color='#e74c3c', lw=2, ms=7,
        mfc='white', mew=1.5, label='Naive LE-PSI')
ax.plot(batched_sizes, batched_ram_gb, '--s', color='#27ae60', lw=2, ms=7,
        mfc='white', mew=1.5, label='Batched LE-PSI')
ax.scatter(server_sizes, [r/1024 for r in peak_ram_mb], c='#27ae60',
           s=80, zorder=5, edgecolors='black', lw=1, label='Measured (D=2048)')
ax.axhline(y=85, color='#3498db', ls=':', lw=1.2, alpha=0.7, label='HPC limit (85 GB)')
ax.set_xlabel('Dataset Size')
ax.set_ylabel('Peak RAM (GB)')
ax.set_xscale('log')
ax.set_yscale('log')
ax.legend(framealpha=0.9, loc='upper left')
ax.grid(True, alpha=0.2, which='both')
fig.savefig(f'{out}/memory_comparison.png')
fig.savefig(f'{out}/memory_comparison.eps')
plt.close()
print("✓ memory_comparison")

# ── GRAPH 2: Execution Time ──
fig, ax = plt.subplots(figsize=(6, 4))
ax.plot(server_sizes, [t/60 for t in exec_time_sec], '-o', color='#8e44ad',
        lw=2, ms=8, mfc='white', mew=1.5)
for s, t in zip(server_sizes, exec_time_sec):
    ax.annotate(f'{t/60:.1f}m', (s, t/60), textcoords='offset points',
                xytext=(8, 8), fontsize=10)
ax.set_xlabel('Dataset Size')
ax.set_ylabel('Time (minutes)')
ax.grid(True, alpha=0.2)
fig.savefig(f'{out}/execution_time.png')
fig.savefig(f'{out}/execution_time.eps')
plt.close()
print("✓ execution_time")

# ── GRAPH 3: RAM Efficiency ──
fig, ax = plt.subplots(figsize=(7, 4.5))
ram_per_rec = [peak_ram_mb[i]/server_sizes[i] for i in range(len(server_sizes))]
ax.plot(naive_sizes, [280]*len(naive_sizes), '-o', color='#e74c3c', lw=2, ms=7,
        mfc='white', mew=1.5, label='Naive (280 MB/record)')
ax.plot(server_sizes, ram_per_rec, '-s', color='#27ae60', lw=2, ms=8,
        mfc='white', mew=1.5, label='Batched (effective)')
for s, r in zip(server_sizes, ram_per_rec):
    ax.annotate(f'{r:.0f}', (s, r), textcoords='offset points',
                xytext=(8, -12), fontsize=10)
ax.set_xlabel('Dataset Size')
ax.set_ylabel('RAM per Record (MB)')
ax.set_xscale('log')
ax.set_yscale('log')
ax.legend(framealpha=0.9)
ax.grid(True, alpha=0.2, which='both')
fig.savefig(f'{out}/ram_efficiency.png')
fig.savefig(f'{out}/ram_efficiency.eps')
plt.close()
print("✓ ram_efficiency")

# ── GRAPH 4: Accuracy ──
fig, ax = plt.subplots(figsize=(5, 3.5))
colors = ['#3498db', '#2ecc71', '#9b59b6']
bars = ax.bar([str(s) for s in server_sizes],
              [100]*3, color=colors, edgecolor='black', lw=0.5, width=0.5)
for bar, m, c in zip(bars, matches_found, client_sizes):
    ax.text(bar.get_x() + bar.get_width()/2, 102, f'{m}/{c}',
            ha='center', fontsize=11, fontweight='bold')
ax.set_xlabel('Dataset Size')
ax.set_ylabel('Accuracy (%)')
ax.set_ylim(0, 112)
ax.axhline(y=100, color='green', ls='--', alpha=0.3)
fig.savefig(f'{out}/accuracy.png')
fig.savefig(f'{out}/accuracy.eps')
plt.close()
print("✓ accuracy")

print(f"\nAll graphs saved to {out}/")
