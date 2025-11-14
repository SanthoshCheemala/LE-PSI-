#!/usr/bin/env python3
"""
LE-PSI Scalability Analysis - Graph Generator for Research Papers

This script reads the JSON output from scalability tests and generates
publication-quality graphs suitable for inclusion in academic papers.

Requirements:
    pip install matplotlib numpy pandas

Usage:
    python generate_graphs.py scalability_results/scalability_test_*.json
"""

import json
import sys
import os
from pathlib import Path

try:
    import matplotlib.pyplot as plt
    import matplotlib
    import numpy as np
except ImportError:
    print("Error: Required packages not installed")
    print("Please run: pip install matplotlib numpy pandas")
    sys.exit(1)

# Use a professional style
plt.style.use('seaborn-v0_8-paper' if 'seaborn-v0_8-paper' in plt.style.available else 'default')
matplotlib.rcParams['font.size'] = 10
matplotlib.rcParams['figure.dpi'] = 300

def load_test_data(json_file):
    """Load test results from JSON file"""
    with open(json_file, 'r') as f:
        return json.load(f)

def plot_execution_time_vs_dataset_size(data, output_dir):
    """Plot execution time scaling with dataset size"""
    successful_tests = [t for t in data['test_results'] if t['success']]
    
    server_sizes = [t['server_dataset_size'] for t in successful_tests]
    client_sizes = [t['client_dataset_size'] for t in successful_tests]
    times = [t['total_time_ns'] / 1e9 for t in successful_tests]  # Convert to seconds
    
    fig, (ax1, ax2) = plt.subplots(1, 2, figsize=(12, 4))
    
    # Server size vs time
    ax1.plot(server_sizes, times, 'o-', linewidth=2, markersize=8, color='#667eea')
    ax1.set_xlabel('Server Dataset Size (records)', fontweight='bold')
    ax1.set_ylabel('Execution Time (seconds)', fontweight='bold')
    ax1.set_title('Server Dataset Size vs Execution Time')
    ax1.grid(True, alpha=0.3)
    ax1.set_xscale('log')
    ax1.set_yscale('log')
    
    # Client size vs time
    ax2.plot(client_sizes, times, 's-', linewidth=2, markersize=8, color='#764ba2')
    ax2.set_xlabel('Client Dataset Size (records)', fontweight='bold')
    ax2.set_ylabel('Execution Time (seconds)', fontweight='bold')
    ax2.set_title('Client Dataset Size vs Execution Time')
    ax2.grid(True, alpha=0.3)
    ax2.set_xscale('log')
    ax2.set_yscale('log')
    
    plt.tight_layout()
    output_file = os.path.join(output_dir, 'execution_time_scaling.pdf')
    plt.savefig(output_file, bbox_inches='tight')
    plt.savefig(output_file.replace('.pdf', '.png'), bbox_inches='tight')
    print(f"✓ Saved: {output_file}")
    plt.close()

def plot_throughput_analysis(data, output_dir):
    """Plot throughput across different dataset sizes"""
    successful_tests = [t for t in data['test_results'] if t['success']]
    
    test_names = [t['test_name'] for t in successful_tests]
    throughputs = [t['throughput_ops_per_sec'] for t in successful_tests]
    
    fig, ax = plt.subplots(figsize=(10, 6))
    
    bars = ax.bar(range(len(test_names)), throughputs, color='#667eea', alpha=0.8)
    ax.set_xlabel('Test Case', fontweight='bold')
    ax.set_ylabel('Throughput (operations/second)', fontweight='bold')
    ax.set_title('PSI Throughput Across Different Dataset Sizes')
    ax.set_xticks(range(len(test_names)))
    ax.set_xticklabels(test_names, rotation=45, ha='right')
    ax.grid(True, alpha=0.3, axis='y')
    
    # Add value labels on bars
    for bar in bars:
        height = bar.get_height()
        ax.text(bar.get_x() + bar.get_width()/2., height,
                f'{height:.1f}',
                ha='center', va='bottom', fontsize=9)
    
    plt.tight_layout()
    output_file = os.path.join(output_dir, 'throughput_analysis.pdf')
    plt.savefig(output_file, bbox_inches='tight')
    plt.savefig(output_file.replace('.pdf', '.png'), bbox_inches='tight')
    print(f"✓ Saved: {output_file}")
    plt.close()

def plot_accuracy_analysis(data, output_dir):
    """Plot accuracy across different test cases"""
    successful_tests = [t for t in data['test_results'] if t['success']]
    
    test_names = [t['test_name'] for t in successful_tests]
    accuracies = [t['accuracy'] for t in successful_tests]
    
    fig, ax = plt.subplots(figsize=(10, 5))
    
    bars = ax.bar(range(len(test_names)), accuracies, color='#2ecc71', alpha=0.8)
    ax.set_xlabel('Test Case', fontweight='bold')
    ax.set_ylabel('Accuracy (%)', fontweight='bold')
    ax.set_title('PSI Accuracy Across Different Dataset Sizes')
    ax.set_xticks(range(len(test_names)))
    ax.set_xticklabels(test_names, rotation=45, ha='right')
    ax.set_ylim([0, 105])
    ax.axhline(y=100, color='r', linestyle='--', alpha=0.5, label='Perfect Accuracy')
    ax.grid(True, alpha=0.3, axis='y')
    ax.legend()
    
    # Add value labels
    for bar in bars:
        height = bar.get_height()
        ax.text(bar.get_x() + bar.get_width()/2., height,
                f'{height:.1f}%',
                ha='center', va='bottom', fontsize=9)
    
    plt.tight_layout()
    output_file = os.path.join(output_dir, 'accuracy_analysis.pdf')
    plt.savefig(output_file, bbox_inches='tight')
    plt.savefig(output_file.replace('.pdf', '.png'), bbox_inches='tight')
    print(f"✓ Saved: {output_file}")
    plt.close()

def plot_time_breakdown(data, output_dir):
    """Plot breakdown of time spent in each phase"""
    successful_tests = [t for t in data['test_results'] if t['success']]
    
    test_names = [t['test_name'] for t in successful_tests]
    init_times = [t['initialization_time_ns'] / 1e9 for t in successful_tests]
    enc_times = [t['encryption_time_ns'] / 1e9 for t in successful_tests]
    int_times = [t['intersection_time_ns'] / 1e9 for t in successful_tests]
    
    fig, ax = plt.subplots(figsize=(12, 6))
    
    x = np.arange(len(test_names))
    width = 0.25
    
    bars1 = ax.bar(x - width, init_times, width, label='Initialization', color='#3498db', alpha=0.8)
    bars2 = ax.bar(x, enc_times, width, label='Encryption', color='#e74c3c', alpha=0.8)
    bars3 = ax.bar(x + width, int_times, width, label='Intersection', color='#f39c12', alpha=0.8)
    
    ax.set_xlabel('Test Case', fontweight='bold')
    ax.set_ylabel('Time (seconds)', fontweight='bold')
    ax.set_title('Execution Time Breakdown by Phase')
    ax.set_xticks(x)
    ax.set_xticklabels(test_names, rotation=45, ha='right')
    ax.legend()
    ax.grid(True, alpha=0.3, axis='y')
    
    plt.tight_layout()
    output_file = os.path.join(output_dir, 'time_breakdown.pdf')
    plt.savefig(output_file, bbox_inches='tight')
    plt.savefig(output_file.replace('.pdf', '.png'), bbox_inches='tight')
    print(f"✓ Saved: {output_file}")
    plt.close()

def plot_memory_usage(data, output_dir):
    """Plot actual RAM usage analysis"""
    successful_tests = [t for t in data['test_results'] if t['success']]
    
    dataset_sizes = [t['server_dataset_size'] for t in successful_tests]
    peak_ram = [t['ram_analysis']['peak_ram_mb'] for t in successful_tests]
    server_init_ram = [t['ram_analysis']['server_init_ram_delta_mb'] for t in successful_tests]
    
    fig, (ax1, ax2) = plt.subplots(1, 2, figsize=(14, 5))
    
    # Plot 1: Peak RAM usage
    ax1.plot(dataset_sizes, peak_ram, 'o-', linewidth=2, markersize=8, color='#9b59b6', label='Peak RAM')
    ax1.plot(dataset_sizes, server_init_ram, 's-', linewidth=2, markersize=8, color='#e74c3c', label='Server Init RAM')
    ax1.set_xlabel('Server Dataset Size (records)', fontweight='bold')
    ax1.set_ylabel('RAM Usage (MB)', fontweight='bold')
    ax1.set_title('RAM Consumption vs Dataset Size')
    ax1.grid(True, alpha=0.3)
    ax1.legend()
    
    # Add linear trend line for peak RAM
    if len(dataset_sizes) > 1:
        z = np.polyfit(dataset_sizes, peak_ram, 1)
        p = np.poly1d(z)
        ax1.plot(dataset_sizes, p(dataset_sizes), "--", alpha=0.5, color='purple',
                label=f'Linear fit: {z[0]:.3f}*n + {z[1]:.1f}')
        ax1.legend()
    
    # Plot 2: RAM per record
    ram_per_record = [t['ram_analysis']['ram_per_server_record_mb'] for t in successful_tests]
    ax2.bar(range(len(dataset_sizes)), ram_per_record, color='#3498db', alpha=0.8)
    ax2.set_xlabel('Test Case', fontweight='bold')
    ax2.set_ylabel('RAM per Server Record (MB)', fontweight='bold')
    ax2.set_title('Memory Efficiency: RAM per Record')
    ax2.set_xticks(range(len(dataset_sizes)))
    ax2.set_xticklabels([f'{s}' for s in dataset_sizes], rotation=45)
    ax2.grid(True, alpha=0.3, axis='y')
    
    # Add average line
    avg_ram_per_record = np.mean(ram_per_record)
    ax2.axhline(y=avg_ram_per_record, color='red', linestyle='--', alpha=0.7,
                label=f'Avg: {avg_ram_per_record:.3f} MB/record')
    ax2.legend()
    
    plt.tight_layout()
    output_file = os.path.join(output_dir, 'memory_usage.pdf')
    plt.savefig(output_file, bbox_inches='tight')
    plt.savefig(output_file.replace('.pdf', '.png'), bbox_inches='tight')
    print(f"✓ Saved: {output_file}")
    plt.close()

def plot_ram_breakdown(data, output_dir):
    """Plot RAM breakdown by stage"""
    successful_tests = [t for t in data['test_results'] if t['success']]
    
    test_names = [t['test_name'] for t in successful_tests]
    baseline_ram = [t['ram_analysis']['baseline_ram_mb'] for t in successful_tests]
    data_load_delta = [t['ram_analysis']['data_load_ram_delta_mb'] for t in successful_tests]
    server_init_delta = [t['ram_analysis']['server_init_ram_delta_mb'] for t in successful_tests]
    encryption_delta = [t['ram_analysis']['encryption_ram_delta_mb'] for t in successful_tests]
    
    fig, ax = plt.subplots(figsize=(12, 6))
    
    x = np.arange(len(test_names))
    width = 0.6
    
    # Stacked bar chart
    p1 = ax.bar(x, baseline_ram, width, label='Baseline', color='#95a5a6', alpha=0.7)
    p2 = ax.bar(x, data_load_delta, width, bottom=baseline_ram, label='Data Loading', color='#3498db', alpha=0.7)
    p3 = ax.bar(x, server_init_delta, width, 
                bottom=np.array(baseline_ram) + np.array(data_load_delta),
                label='Server Init (Witnesses)', color='#e74c3c', alpha=0.7)
    p4 = ax.bar(x, encryption_delta, width,
                bottom=np.array(baseline_ram) + np.array(data_load_delta) + np.array(server_init_delta),
                label='Client Encryption', color='#2ecc71', alpha=0.7)
    
    ax.set_xlabel('Test Case', fontweight='bold')
    ax.set_ylabel('RAM Usage (MB)', fontweight='bold')
    ax.set_title('RAM Breakdown by PSI Stage', fontweight='bold', fontsize=14)
    ax.set_xticks(x)
    ax.set_xticklabels(test_names, rotation=45, ha='right')
    ax.legend(loc='upper left')
    ax.grid(True, alpha=0.3, axis='y')
    
    plt.tight_layout()
    output_file = os.path.join(output_dir, 'ram_breakdown_stages.pdf')
    plt.savefig(output_file, bbox_inches='tight')
    plt.savefig(output_file.replace('.pdf', '.png'), bbox_inches='tight')
    print(f"✓ Saved: {output_file}")
    plt.close()

def plot_ram_scaling_analysis(data, output_dir):
    """Plot RAM scaling factor analysis"""
    successful_tests = [t for t in data['test_results'] if t['success']]
    
    dataset_sizes = [t['server_dataset_size'] for t in successful_tests]
    total_ram_delta = [t['ram_analysis']['total_ram_delta_mb'] for t in successful_tests]
    
    fig, ax = plt.subplots(figsize=(10, 6))
    
    ax.scatter(dataset_sizes, total_ram_delta, s=100, alpha=0.6, color='#e74c3c', edgecolors='black', linewidth=1)
    
    # Fit and plot linear regression
    if len(dataset_sizes) > 1:
        z = np.polyfit(dataset_sizes, total_ram_delta, 1)
        p = np.poly1d(z)
        ax.plot(dataset_sizes, p(dataset_sizes), "--", linewidth=2, color='#c0392b',
                label=f'Linear fit: {z[0]:.3f} MB/record')
        
        # Calculate R²
        y_mean = np.mean(total_ram_delta)
        ss_tot = np.sum((np.array(total_ram_delta) - y_mean) ** 2)
        ss_res = np.sum((np.array(total_ram_delta) - p(dataset_sizes)) ** 2)
        r_squared = 1 - (ss_res / ss_tot)
        
        ax.text(0.05, 0.95, f'R² = {r_squared:.4f}\nRAM Scaling: {z[0]:.3f} MB/record', 
                transform=ax.transAxes, fontsize=11, verticalalignment='top',
                bbox=dict(boxstyle='round', facecolor='wheat', alpha=0.5))
    
    ax.set_xlabel('Server Dataset Size (records)', fontweight='bold')
    ax.set_ylabel('Total RAM Increase from Baseline (MB)', fontweight='bold')
    ax.set_title('RAM Scaling Factor Analysis', fontweight='bold', fontsize=14)
    ax.grid(True, alpha=0.3)
    ax.legend()
    
    plt.tight_layout()
    output_file = os.path.join(output_dir, 'ram_scaling_factor.pdf')
    plt.savefig(output_file, bbox_inches='tight')
    plt.savefig(output_file.replace('.pdf', '.png'), bbox_inches='tight')
    print(f"✓ Saved: {output_file}")
    plt.close()

def plot_scalability_score(data, output_dir):
    """Plot overall scalability metrics"""
    summary = data['summary']
    
    metrics = {
        'Avg Accuracy': summary['average_accuracy'],
        'Avg Throughput\n(normalized)': min(summary['average_throughput_ops_per_sec'] * 10, 100),
        'Dataset Scale\n(normalized)': min(summary['largest_dataset_tested'] / 200, 100),
        'Overall Score': summary['scalability_score']
    }
    
    fig, ax = plt.subplots(figsize=(8, 6))
    
    bars = ax.bar(range(len(metrics)), list(metrics.values()), 
                   color=['#2ecc71', '#3498db', '#e74c3c', '#f39c12'], alpha=0.8)
    ax.set_ylabel('Score (0-100)', fontweight='bold')
    ax.set_title('LE-PSI Scalability Metrics', fontweight='bold', fontsize=14)
    ax.set_xticks(range(len(metrics)))
    ax.set_xticklabels(list(metrics.keys()))
    ax.set_ylim([0, 105])
    ax.grid(True, alpha=0.3, axis='y')
    
    # Add value labels
    for bar in bars:
        height = bar.get_height()
        ax.text(bar.get_x() + bar.get_width()/2., height,
                f'{height:.1f}',
                ha='center', va='bottom', fontsize=10, fontweight='bold')
    
    plt.tight_layout()
    output_file = os.path.join(output_dir, 'scalability_score.pdf')
    plt.savefig(output_file, bbox_inches='tight')
    plt.savefig(output_file.replace('.pdf', '.png'), bbox_inches='tight')
    print(f"✓ Saved: {output_file}")
    plt.close()

def plot_go_runtime_analysis(data, output_dir):
    """Plot Go runtime performance metrics"""
    successful_tests = [t for t in data['test_results'] if t['success'] and 'go_runtime_stats' in t]
    
    if not successful_tests:
        print("⚠ No Go runtime statistics available, skipping Go runtime graphs")
        return
    
    test_names = [t['test_name'] for t in successful_tests]
    dataset_sizes = [t['server_dataset_size'] for t in successful_tests]
    
    # Memory usage comparison
    heap_alloc = [t['go_runtime_stats']['heap_alloc_mb'] for t in successful_tests]
    system_mem = [t['go_runtime_stats']['system_memory_mb'] for t in successful_tests]
    
    # GC statistics
    num_gc = [t['go_runtime_stats']['num_gc'] for t in successful_tests]
    gc_cpu = [t['go_runtime_stats']['gc_cpu_percentage'] for t in successful_tests]
    
    # Goroutines
    goroutines = [t['go_runtime_stats']['num_goroutines'] for t in successful_tests]
    
    # Create a 2x2 subplot
    fig, ((ax1, ax2), (ax3, ax4)) = plt.subplots(2, 2, figsize=(14, 10))
    
    # 1. Memory Usage
    ax1.plot(dataset_sizes, heap_alloc, 'o-', linewidth=2, markersize=6, label='Heap Allocated', color='#3498db')
    ax1.plot(dataset_sizes, system_mem, 's-', linewidth=2, markersize=6, label='System Memory', color='#e74c3c')
    ax1.set_xlabel('Dataset Size (records)', fontweight='bold')
    ax1.set_ylabel('Memory (MB)', fontweight='bold')
    ax1.set_title('Go Memory Usage Across Dataset Sizes')
    ax1.set_xscale('log')
    ax1.legend()
    ax1.grid(True, alpha=0.3)
    
    # 2. Garbage Collection
    ax2_twin = ax2.twinx()
    bars = ax2.bar(range(len(test_names)), num_gc, alpha=0.7, color='#9b59b6', label='GC Runs')
    line = ax2_twin.plot(range(len(test_names)), gc_cpu, 'ro-', linewidth=2, markersize=8, label='GC CPU %')
    ax2.set_xlabel('Test Case', fontweight='bold')
    ax2.set_ylabel('Number of GC Runs', fontweight='bold', color='#9b59b6')
    ax2_twin.set_ylabel('GC CPU Percentage', fontweight='bold', color='red')
    ax2.set_title('Garbage Collection Performance')
    ax2.set_xticks(range(len(test_names)))
    ax2.set_xticklabels(test_names, rotation=45, ha='right')
    ax2.tick_params(axis='y', labelcolor='#9b59b6')
    ax2_twin.tick_params(axis='y', labelcolor='red')
    ax2.grid(True, alpha=0.3)
    
    # 3. Goroutine Usage
    ax3.bar(range(len(test_names)), goroutines, color='#2ecc71', alpha=0.8)
    ax3.set_xlabel('Test Case', fontweight='bold')
    ax3.set_ylabel('Number of Goroutines', fontweight='bold')
    ax3.set_title('Goroutine Concurrency Usage')
    ax3.set_xticks(range(len(test_names)))
    ax3.set_xticklabels(test_names, rotation=45, ha='right')
    ax3.grid(True, alpha=0.3, axis='y')
    
    # Add value labels
    for i, v in enumerate(goroutines):
        ax3.text(i, v, str(v), ha='center', va='bottom', fontweight='bold')
    
    # 4. Memory Efficiency (Heap per Record)
    mem_per_record = [heap_alloc[i] * 1024 / dataset_sizes[i] for i in range(len(successful_tests))]
    ax4.plot(dataset_sizes, mem_per_record, 'o-', linewidth=2, markersize=8, color='#f39c12')
    ax4.set_xlabel('Dataset Size (records)', fontweight='bold')
    ax4.set_ylabel('Memory per Record (KB)', fontweight='bold')
    ax4.set_title('Memory Efficiency - Heap per Record')
    ax4.set_xscale('log')
    ax4.grid(True, alpha=0.3)
    
    plt.tight_layout()
    output_file = os.path.join(output_dir, 'go_runtime_analysis.pdf')
    plt.savefig(output_file, bbox_inches='tight')
    plt.savefig(output_file.replace('.pdf', '.png'), bbox_inches='tight')
    print(f"✓ Saved: {output_file}")
    plt.close()

def plot_go_memory_breakdown(data, output_dir):
    """Plot detailed Go memory breakdown"""
    successful_tests = [t for t in data['test_results'] if t['success'] and 'go_runtime_stats' in t]
    
    if not successful_tests:
        return
    
    test_names = [t['test_name'] for t in successful_tests]
    
    # Memory components
    heap_alloc = [t['go_runtime_stats']['heap_alloc_mb'] for t in successful_tests]
    heap_idle = [t['go_runtime_stats']['heap_idle_mb'] for t in successful_tests]
    heap_inuse = [t['go_runtime_stats']['heap_inuse_mb'] for t in successful_tests]
    stack_inuse = [t['go_runtime_stats']['stack_inuse_mb'] for t in successful_tests]
    
    fig, ax = plt.subplots(figsize=(12, 6))
    
    x = np.arange(len(test_names))
    width = 0.2
    
    bars1 = ax.bar(x - 1.5*width, heap_alloc, width, label='Heap Allocated', color='#3498db', alpha=0.8)
    bars2 = ax.bar(x - 0.5*width, heap_inuse, width, label='Heap In Use', color='#e74c3c', alpha=0.8)
    bars3 = ax.bar(x + 0.5*width, heap_idle, width, label='Heap Idle', color='#2ecc71', alpha=0.8)
    bars4 = ax.bar(x + 1.5*width, stack_inuse, width, label='Stack In Use', color='#f39c12', alpha=0.8)
    
    ax.set_xlabel('Test Case', fontweight='bold')
    ax.set_ylabel('Memory (MB)', fontweight='bold')
    ax.set_title('Go Memory Breakdown by Component')
    ax.set_xticks(x)
    ax.set_xticklabels(test_names, rotation=45, ha='right')
    ax.legend()
    ax.grid(True, alpha=0.3, axis='y')
    
    plt.tight_layout()
    output_file = os.path.join(output_dir, 'go_memory_breakdown.pdf')
    plt.savefig(output_file, bbox_inches='tight')
    plt.savefig(output_file.replace('.pdf', '.png'), bbox_inches='tight')
    print(f"✓ Saved: {output_file}")
    plt.close()

def generate_latex_table(data, output_dir):
    """Generate LaTeX table for research paper"""
    successful_tests = [t for t in data['test_results'] if t['success']]
    
    latex_content = r"""\begin{table}[h]
\centering
\caption{LE-PSI Performance Evaluation on Various Dataset Sizes}
\label{tab:lepsi_performance}
\begin{tabular}{|l|r|r|r|r|r|}
\hline
\textbf{Test Case} & \textbf{Server} & \textbf{Client} & \textbf{Time (s)} & \textbf{Throughput} & \textbf{Accuracy} \\
 & \textbf{Size} & \textbf{Size} &  & \textbf{(ops/s)} & \textbf{(\%)} \\
\hline
"""
    
    for test in successful_tests:
        latex_content += f"{test['test_name']} & "
        latex_content += f"{test['server_dataset_size']:,} & "
        latex_content += f"{test['client_dataset_size']:,} & "
        latex_content += f"{test['total_time_ns']/1e9:.2f} & "
        latex_content += f"{test['throughput_ops_per_sec']:.2f} & "
        latex_content += f"{test['accuracy']:.1f} \\\\\n"
    
    latex_content += r"""\hline
\end{tabular}
\end{table}
"""
    
    output_file = os.path.join(output_dir, 'performance_table.tex')
    with open(output_file, 'w') as f:
        f.write(latex_content)
    print(f"✓ Saved: {output_file}")

def main():
    if len(sys.argv) < 2:
        print("Usage: python generate_graphs.py <json_file>")
        print("Example: python generate_graphs.py scalability_results/scalability_test_*.json")
        sys.exit(1)
    
    json_file = sys.argv[1]
    
    if not os.path.exists(json_file):
        print(f"Error: File not found: {json_file}")
        sys.exit(1)
    
    print("=================================================")
    print("  LE-PSI Scalability Graph Generator")
    print("=================================================\n")
    
    # Load data
    print(f"Loading data from: {json_file}")
    data = load_test_data(json_file)
    
    # Create output directory
    output_dir = os.path.join(os.path.dirname(json_file), 'graphs')
    os.makedirs(output_dir, exist_ok=True)
    print(f"Output directory: {output_dir}\n")
    
    # Generate all graphs
    print("Generating graphs...")
    plot_execution_time_vs_dataset_size(data, output_dir)
    plot_throughput_analysis(data, output_dir)
    plot_accuracy_analysis(data, output_dir)
    plot_time_breakdown(data, output_dir)
    plot_memory_usage(data, output_dir)
    plot_scalability_score(data, output_dir)
    
    # Generate RAM analysis graphs
    print("\nGenerating RAM analysis...")
    plot_ram_breakdown(data, output_dir)
    plot_ram_scaling_analysis(data, output_dir)
    
    # Generate Go runtime analysis graphs
    print("\nGenerating Go runtime analysis...")
    plot_go_runtime_analysis(data, output_dir)
    plot_go_memory_breakdown(data, output_dir)
    
    # Generate LaTeX table
    print("\nGenerating LaTeX table...")
    generate_latex_table(data, output_dir)
    
    print("\n=================================================")
    print("  All graphs generated successfully!")
    print(f"  Location: {output_dir}/")
    print("=================================================")
    print("\nGenerated files:")
    print("  - execution_time_scaling.pdf/png")
    print("  - throughput_analysis.pdf/png")
    print("  - accuracy_analysis.pdf/png")
    print("  - time_breakdown.pdf/png")
    print("  - memory_usage.pdf/png (RAM analysis)")
    print("  - ram_breakdown_stages.pdf/png (RAM by stage)")
    print("  - ram_scaling_factor.pdf/png (RAM scaling)")
    print("  - scalability_score.pdf/png")
    print("  - go_runtime_analysis.pdf/png (Go performance)")
    print("  - go_memory_breakdown.pdf/png (Go memory)")
    print("  - performance_table.tex")
    print("\nThese files are ready for inclusion in your research paper!")

if __name__ == "__main__":
    main()
