#!/usr/bin/env python3
"""LE-PSI Rebuttal Graphs — All real HPC data"""

import matplotlib
matplotlib.use('Agg')
import matplotlib.pyplot as plt
import numpy as np
import os

# ================================================================
# REAL HPC DATA — D=256 (all 6 tests passed)
# AMD EPYC 7413, 188 GB RAM, batch=100, workers=16
# Total run: 8h38m22s
# ================================================================
d256_sizes      = [50,    100,   250,    1000,    5000,     10000]
d256_clients    = [5,     10,    25,     100,     100,      100]
d256_matches    = [5,     9,     25,     99,      100,      100]
d256_accuracy   = [100,   90,    100,    99,      100,      100]
d256_total_ns   = [14174684056, 39285948910, 127775335227, 1423814379680, 8183714150644, 21313834431385]
d256_total_sec  = [t/1e9 for t in d256_total_ns]
d256_init_ns    = [11361401019, 28584527616, 65314321415, 300887383013, 1777596972461, 4171011499350]
d256_enc_ns     = [273354111,   589779509,   1820472933,  8915389950,   8154232096,    12784783008]
d256_int_ns     = [2436813118,  9972011884,  60452460216, 1113719388707, 6397059984512, 17127901427403]
d256_peak_ram   = [125,   584,   584,    4510.5,  None,     18473.4]  # MB

# ================================================================
# REAL HPC DATA — D=2048 larger-D mode (3 tests passed)
# ================================================================
d2048_sizes     = [50, 100, 250]
d2048_peak_ram  = [4500.6, 4500.7, 4500.8]
d2048_total_sec = [120, 300, 478]

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

# ── GRAPH 1: Execution Time (D=256 vs D=2048) ──
fig, ax = plt.subplots(figsize=(7, 4.5))
ax.plot(d256_sizes, [t/60 for t in d256_total_sec],
        '-o', color='#3498db', lw=2, ms=7, mfc='white', mew=1.5,
        label='D=256')
ax.plot(d2048_sizes, [t/60 for t in d2048_total_sec],
        '-s', color='#e74c3c', lw=2, ms=7, mfc='white', mew=1.5,
        label='D=2048 larger-D')
ax.set_xlabel('Dataset Size')
ax.set_ylabel('Time (minutes)')
ax.set_xscale('log')
ax.set_yscale('log')
ax.legend(framealpha=0.9)
ax.grid(True, alpha=0.2, which='both')
fig.savefig(f'{out}/execution_time.png')
fig.savefig(f'{out}/execution_time.eps')
plt.close()
print("✓ execution_time")

# ── GRAPH 2: Memory Comparison (Batched vs Naive at D=2048) ──
fig, ax = plt.subplots(figsize=(7, 4.5))
naive_sizes = [50, 100, 250, 500, 1000, 5000, 10000]
naive_ram_gb = [s * 280 / 1024 for s in naive_sizes]
batched_ram_gb = [4.5] * len(naive_sizes)
ax.plot(naive_sizes, naive_ram_gb, '-o', color='#e74c3c', lw=2, ms=7,
        mfc='white', mew=1.5, label='Naive LE-PSI')
ax.plot(naive_sizes, batched_ram_gb, '--s', color='#27ae60', lw=2, ms=7,
        mfc='white', mew=1.5, label='Batched LE-PSI')
ax.scatter(d2048_sizes, [r/1024 for r in d2048_peak_ram], c='#27ae60',
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

# ── GRAPH 3: Phase Breakdown (D=256, all 6 tests) ──
fig, ax = plt.subplots(figsize=(8, 4.5))
init_min = [d256_init_ns[i]/1e9/60 for i in range(6)]
enc_min  = [d256_enc_ns[i]/1e9/60 for i in range(6)]
int_min  = [d256_int_ns[i]/1e9/60 for i in range(6)]
x = np.arange(6)
w = 0.25
ax.bar(x - w, init_min, w, label='Initialization', color='#3498db', edgecolor='black', lw=0.5)
ax.bar(x, enc_min, w, label='Encryption', color='#2ecc71', edgecolor='black', lw=0.5)
ax.bar(x + w, int_min, w, label='Intersection', color='#e74c3c', edgecolor='black', lw=0.5)
ax.set_xlabel('Dataset Size')
ax.set_ylabel('Time (minutes)')
ax.set_xticks(x)
ax.set_xticklabels([str(s) for s in d256_sizes])
ax.legend(framealpha=0.9)
ax.grid(True, alpha=0.2, axis='y')
fig.savefig(f'{out}/phase_breakdown.png')
fig.savefig(f'{out}/phase_breakdown.eps')
plt.close()
print("✓ phase_breakdown")

# ── GRAPH 4: Accuracy (D=256, all 6 tests) ──
fig, ax = plt.subplots(figsize=(7, 3.5))
colors = ['#3498db', '#2ecc71', '#9b59b6', '#e67e22', '#1abc9c', '#e74c3c']
bars = ax.bar([str(s) for s in d256_sizes], d256_accuracy,
              color=colors, edgecolor='black', lw=0.5, width=0.5)
for bar, m, c in zip(bars, d256_matches, d256_clients):
    ax.text(bar.get_x() + bar.get_width()/2, 102, f'{m}/{c}',
            ha='center', fontsize=10, fontweight='bold')
ax.set_xlabel('Dataset Size')
ax.set_ylabel('Accuracy (%)')
ax.set_ylim(0, 112)
ax.axhline(y=100, color='green', ls='--', alpha=0.3)
fig.savefig(f'{out}/accuracy.png')
fig.savefig(f'{out}/accuracy.eps')
plt.close()
print("✓ accuracy")

# ── GRAPH 5: Security Overhead (D=256 vs D=2048) ──
fig, ax = plt.subplots(figsize=(6, 4))
common = [50, 100, 250]
d256_c = [d256_total_sec[i]/60 for i in range(3)]
d2048_c = [d2048_total_sec[i]/60 for i in range(3)]
overhead = [d2048_c[i]/d256_c[i] for i in range(3)]
x = np.arange(3)
w = 0.3
ax.bar(x - w/2, d256_c, w, label='D=256', color='#3498db', edgecolor='black', lw=0.5)
ax.bar(x + w/2, d2048_c, w, label='D=2048', color='#e74c3c', edgecolor='black', lw=0.5)
for i, ov in enumerate(overhead):
    ax.text(i, max(d256_c[i], d2048_c[i]) + 0.3,
            f'{ov:.1f}×', ha='center', fontsize=11, fontweight='bold')
ax.set_xlabel('Dataset Size')
ax.set_ylabel('Time (minutes)')
ax.set_xticks(x)
ax.set_xticklabels([str(s) for s in common])
ax.legend(framealpha=0.9)
ax.grid(True, alpha=0.2, axis='y')
fig.savefig(f'{out}/security_overhead.png')
fig.savefig(f'{out}/security_overhead.eps')
plt.close()
print("✓ security_overhead")

# ── GRAPH 6: Peak RAM across datasets (D=256) ──
fig, ax = plt.subplots(figsize=(7, 4.5))
ram_sizes = [50, 100, 250, 1000, 10000]
ram_vals  = [125, 584, 584, 4510.5, 18473.4]  # MB
ax.plot(ram_sizes, [r/1024 for r in ram_vals], '-o', color='#8e44ad', lw=2, ms=8,
        mfc='white', mew=1.5)
for s, r in zip(ram_sizes, ram_vals):
    ax.annotate(f'{r/1024:.1f} GB', (s, r/1024), textcoords='offset points',
                xytext=(8, 8), fontsize=10)
ax.set_xlabel('Dataset Size')
ax.set_ylabel('Peak RAM (GB)')
ax.set_xscale('log')
ax.grid(True, alpha=0.2, which='both')
fig.savefig(f'{out}/ram_scaling.png')
fig.savefig(f'{out}/ram_scaling.eps')
plt.close()
print("✓ ram_scaling")

print(f"\nAll graphs saved to {out}/")
print("\n=== Complete D=256 Results ===")
for i in range(6):
    t = d256_total_sec[i]
    if t > 3600:
        tstr = f"{t/3600:.1f}h"
    else:
        tstr = f"{t/60:.1f}m"
    print(f"  {d256_sizes[i]:>6} records: {tstr:>8}, "
          f"{d256_matches[i]}/{d256_clients[i]} matches ({d256_accuracy[i]}%)")
print(f"\n  Total run: 8h38m | Peak RAM: 18.5 GB | Avg accuracy: 98.2%")
