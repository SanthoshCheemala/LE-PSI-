# Scalability Testing

Tests LE-PSI performance on datasets ranging from 100 to 50,000 records.

## Quick Start

```bash
./run_tests.sh
```

Choose from:
1. Run all tests (7 scenarios, 30-60 minutes)
2. Quick test (100 records, ~10 seconds)
3. Database test (50,000 real transactions)
4. Build executable
5. View results

## Database Integration

Uses `../data/transactions.db` with 6.36M financial transactions.
Tests load first 50,000 records.

## Outputs

- `scalability_results/*.json` - Test data
- `scalability_results/*.html` - Visual reports

## Generate Graphs

```bash
pip install matplotlib numpy pandas
python generate_graphs.py scalability_results/*.json
```

Generates publication-ready PDFs and LaTeX tables for research papers.
