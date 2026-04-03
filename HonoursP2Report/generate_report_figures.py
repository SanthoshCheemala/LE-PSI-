#!/usr/bin/env python3
"""
generate_report_figures.py
Generates all figures for the Honours Report as PNG files.
Run once, then upload the PNGs to Overleaf.

Usage:
    cd /Users/santhoshcheemala/ALL_IN_ONE/Research_Implimentation/PSI
    python3 HonoursP2Report/generate_report_figures.py
"""

import os, matplotlib
matplotlib.use('Agg')
import matplotlib.pyplot as plt
import numpy as np

OUT = os.path.dirname(os.path.abspath(__file__))

plt.rcParams.update({
    'figure.dpi': 150, 'figure.facecolor': 'white',
    'axes.spines.top': False, 'axes.spines.right': False,
    'axes.grid': True, 'grid.color': '#e2e8f0',
    'font.size': 11, 'axes.titlesize': 12, 'axes.labelsize': 11,
    'axes.titleweight': 'bold',
})

sizes = [50, 100, 500, 1000, 2000, 5000, 10000]
times = [0.2, 0.6, 6.3, 15.7, 31.1, 76.1, 152.8]
ram   = [1.5, 3.6,  6.5,  6.5,  6.5,  6.5,   6.5]
tput  = [4.17, 2.78, 1.32, 1.06, 1.07, 1.10, 1.09]

# ── Fig 1: Execution Time ─────────────────────────────────────────────────────
fig, ax = plt.subplots(figsize=(7, 4))
ax.plot(sizes, times, 'o-', color='#2563EB', lw=2, ms=6, label='Laconic PSI (measured)')
ax.plot([0, 10000], [0, 152.8], '--', color='#DC2626', lw=1.5,
        label='Linear fit (≈15.3 min/1K)')
for x, y in zip(sizes, times):
    ax.annotate(f'{y}', (x, y), textcoords='offset points',
                xytext=(4, 5), fontsize=8, color='#2563EB')
ax.set(xlabel='Server Dataset Size m (records)', ylabel='Execution Time (minutes)',
       title='Execution Time Scaling', xlim=(0,10500), ylim=(0,175))
ax.legend(framealpha=0.8)
fig.tight_layout()
fig.savefig(f'{OUT}/fig1_time_scaling.png'); plt.close()

# ── Fig 2: Peak RAM ───────────────────────────────────────────────────────────
fig, ax = plt.subplots(figsize=(7, 4))
ax.plot(sizes, ram, 's-', color='#0D9488', lw=2, ms=6, label='Batched (this work)')
ax.plot([50,100,500], [0.16,0.31,1.55], '^--', color='#DC2626', lw=1.5, ms=5,
        label='Naïve (truncated; 312 GB at m=10K)')
ax.axhline(6.5, color='#0D9488', ls=':', lw=1, alpha=0.5)
ax.annotate('6.5 GB\nconstant plateau', xy=(3000,6.5), xytext=(3500,8),
            fontsize=9, color='#0D9488', ha='center',
            arrowprops=dict(arrowstyle='->', color='#0D9488', lw=1.2))
ax.set(xlabel='Server Dataset Size m (records)', ylabel='Peak RAM (GB)',
       title='Peak Memory Usage: Batching vs. Naïve', xlim=(0,10500), ylim=(0,10.5))
ax.legend(framealpha=0.8)
fig.tight_layout()
fig.savefig(f'{OUT}/fig2_ram_scaling.png'); plt.close()

# ── Fig 3: Throughput ─────────────────────────────────────────────────────────
fig, ax = plt.subplots(figsize=(7, 4))
ax.plot(sizes, tput, 'o-', color='#7C3AED', lw=2, ms=6, label='Laconic PSI')
ax.axhline(1.0, ls='--', color='gray', lw=1.3, label='≈1.0 ops/sec ceiling')
ax.fill_between(sizes, tput, 1.0,
                where=[t > 1.0 for t in tput], alpha=0.12, color='#7C3AED',
                label='Startup overhead region')
ax.annotate('Memory-bandwidth\nceiling', xy=(5000, 1.0), xytext=(2500, 1.8),
            fontsize=9, color='gray',
            arrowprops=dict(arrowstyle='->', color='gray', lw=1.2))
ax.set(xlabel='Server Dataset Size m (records)', ylabel='Throughput (ops/sec)',
       title='Throughput Analysis: Memory Bandwidth Ceiling',
       xlim=(0,10500), ylim=(0,4.5))
ax.legend(framealpha=0.8)
fig.tight_layout()
fig.savefig(f'{OUT}/fig3_throughput.png'); plt.close()

# ── Fig 4: Communication Scaling (log-log) ───────────────────────────────────
m_vals = [100, 1_000, 10_000, 100_000, 1_000_000]
lac = [2.7 + np.log2(m) * 0.032 for m in m_vals]
kk  = [(1000 + m) * 9.83e-5   for m in m_vals]

fig, ax = plt.subplots(figsize=(7, 4.5))
ax.loglog(m_vals, lac, 'o-', color='#2563EB', lw=2, ms=6, label='Laconic PSI — O(n log m)')
ax.loglog(m_vals, kk,  '^-', color='#DC2626', lw=2, ms=6, label='KKRT Classical PSI — O(n+m)')
ax.axvspan(1e4, 1e5, alpha=0.07, color='gray', label='Crossover region (10⁴–10⁵)')
ax.annotate('29× reduction\nat m=10⁶', xy=(1e6, 3.34), xytext=(1e5, 0.5),
            fontsize=9, color='#2563EB',
            arrowprops=dict(arrowstyle='->', color='gray', lw=1.2))
ax.set(xlabel='Server Dataset Size m', ylabel='Communication (MB)',
       title='Communication Scaling: Laconic PSI vs. KKRT (n=1,000 client records)',
       xlim=(80, 2e6), ylim=(0.05, 200))
ax.legend(framealpha=0.8)
fig.tight_layout()
fig.savefig(f'{OUT}/fig4_comm_scaling.png'); plt.close()

# ── Fig 5: Bottleneck ─────────────────────────────────────────────────────────
phases = ['Witness\nFetch\n(35%)', 'NTT\nOps\n(25%)',
          'Decryption\n(10%)', 'Other\n(30%)']
pct    = [35, 25, 10, 30]
colors = ['#F97316', '#EAB308', '#22C55E', '#94A3B8']

fig, (ax1, ax2) = plt.subplots(1, 2, figsize=(9, 4))

# Bar chart
bars = ax1.bar(phases, pct, color=colors, edgecolor='white', lw=1.5)
for b, p in zip(bars, pct):
    ax1.text(b.get_x() + b.get_width()/2, b.get_height() + 0.4,
             f'{p}%', ha='center', fontsize=11, fontweight='bold')
ax1.set(ylabel='Time Share (%)', title='Bottleneck: Bar View', ylim=(0, 45))
ax1.annotate('Memory-bound\n(all workers compete\nfor same RAM bus)',
             xy=(0, 35), xytext=(1.2, 40),
             fontsize=8, color='#DC2626',
             arrowprops=dict(arrowstyle='->', color='#DC2626'))

# Pie chart
ax2.pie(pct, labels=phases, colors=colors, autopct='%1.0f%%',
        startangle=90, pctdistance=0.75,
        wedgeprops=dict(edgecolor='white', linewidth=1.5))
ax2.set_title('Bottleneck: Pie View')

fig.suptitle('Fig 5: Execution Breakdown (Go pprof, m=10,000)',
             fontsize=12, fontweight='bold')
fig.tight_layout()
fig.savefig(f'{OUT}/fig5_bottleneck.png'); plt.close()

# ── Fig 6: Ablation Study ─────────────────────────────────────────────────────
configs = ['Batching only\n(1 worker,\n6.5 GB)',
           'Parallel only\n(OOM >256 GB)',
           'Both: ours\n(77 workers,\n6.5 GB)']
t_vals  = [1760, None, 153]
colors  = ['#F97316', '#DC2626', '#0D9488']

fig, ax = plt.subplots(figsize=(7, 4.5))
bars = ax.bar([0, 1, 2], [1760, 1900, 153], color=colors,
              edgecolor='white', lw=1.5, width=0.5)
ax.text(0, 1760+20, '1,760 min', ha='center', fontsize=10, fontweight='bold')
ax.text(1, 1900+20, 'OOM\n(N/A)', ha='center', fontsize=10,
        fontweight='bold', color='#DC2626')
ax.text(2, 153+20,  '153 min', ha='center', fontsize=10, fontweight='bold')
ax.annotate('11.5× speedup', xy=(2,153), xytext=(1.1, 900),
            fontsize=10, color='#0D9488',
            arrowprops=dict(arrowstyle='->', color='#0D9488', lw=1.5))
ax.set_xticks([0,1,2]); ax.set_xticklabels(configs, fontsize=9)
ax.set(ylabel='Execution Time (minutes)',
       title='Fig 6: Ablation Study — Optimisation Impact (m=10,000)',
       ylim=(0, 2100))
fig.tight_layout()
fig.savefig(f'{OUT}/fig6_ablation.png'); plt.close()

# ── Fig 7: Protocol Comparison (grouped bar) — replaces comparison table ─────
protocols = ['KKRT\n(Classical)', 'ALOS22\n(Pairing)', 'Ours D=256\n(Ring-LWE)']
runtimes  = [0.26/60, 0.1, 153]      # in minutes (KKRT: 0.26s, ALOS22: ~6s estimated)
memories  = [8.3/1024, 0.5, 6500]    # in MB (normalised differently — use GB)

fig, (ax1, ax2) = plt.subplots(1, 2, figsize=(9, 4.5))
col = ['#94A3B8', '#F97316', '#2563EB']

# Runtime bar (log scale)
ax1.bar(protocols, [0.26/60, 0.10, 153], color=col, edgecolor='white', lw=1.3)
ax1.set_yscale('log')
ax1.set(ylabel='Runtime (minutes, log scale)',
        title='Runtime Comparison (m=10K)', ylim=(0.001, 1000))
for i, v in enumerate([0.26/60, 0.10, 153]):
    ax1.text(i, v*1.4, f'{v:.3g} min', ha='center', fontsize=9, fontweight='bold')
ax1.axhspan(0.001, 1, alpha=0.06, color='green')
ax1.text(1, 0.003, 'Classical speed zone', ha='center', fontsize=8, color='green')

# Memory bar (log scale)
mem_gb = [8.3/1024, 0.1, 6.5]
ax2.bar(protocols, mem_gb, color=col, edgecolor='white', lw=1.3)
ax2.set_yscale('log')
ax2.set(ylabel='Peak RAM (GB, log scale)',
        title='Memory Comparison (m=10K)', ylim=(0.001, 100))
for i, v in enumerate(mem_gb):
    ax2.text(i, v*1.5, f'{v:.3g} GB', ha='center', fontsize=9, fontweight='bold')

# PQ annotation
for ax in (ax1, ax2):
    ax.get_children()[2].set_hatch('///')  # hatching for PQ protocol
    ax.text(2.0, ax.get_ylim()[0]*3, '✓ PQ Secure', ha='center',
            fontsize=8, color='#2563EB', fontstyle='italic')

fig.suptitle('Fig 7: Protocol Comparison — Runtime and Memory at m=10,000',
             fontsize=12, fontweight='bold')
fig.tight_layout()
fig.savefig(f'{OUT}/fig7_protocol_comparison.png'); plt.close()

# ── Fig 8: Security Validation Bar Chart — replaces security table ────────────
metrics  = ['Encryption\nTiming\nVariation', 'Decryption\nTiming\nDifference',
            'Randomness\nDeviation']
values   = [0.5, 3.33, 0.43]
thresh   = [2.0, 5.0, 1.0]   # acceptable threshold

fig, ax = plt.subplots(figsize=(7, 4))
x = np.arange(len(metrics))
w = 0.35
b1 = ax.bar(x - w/2, values,  w, color='#2563EB', label='Measured (%)',  edgecolor='white')
b2 = ax.bar(x + w/2, thresh,  w, color='#94A3B8', label='Acceptable threshold (%)',
            edgecolor='white', alpha=0.7)
for b, v in zip(b1, values):
    ax.text(b.get_x()+b.get_width()/2, v+0.05, f'{v}%',
            ha='center', fontsize=10, fontweight='bold', color='#1E3A8A')
ax.set_xticks(x); ax.set_xticklabels(metrics, fontsize=10)
ax.set(ylabel='Percentage (%)',
       title='Fig 8: Security Validation — Measured vs. Acceptable Threshold',
       ylim=(0, 7))
ax.legend(framealpha=0.8)
ax.annotate('All measured values\nbelow safe threshold ✓',
            xy=(2, 0.43), xytext=(1.2, 4.5), fontsize=9, color='green',
            arrowprops=dict(arrowstyle='->', color='green', lw=1.2))
fig.tight_layout()
fig.savefig(f'{OUT}/fig8_security_validation.png'); plt.close()

print("All 8 figures saved to:", OUT)
for i, name in enumerate(['time_scaling','ram_scaling','throughput','comm_scaling',
                           'bottleneck','ablation','protocol_comparison',
                           'security_validation'], 1):
    print(f"  fig{i}_{name}.png")
