# LE-PSI Paper Support Artifact

This folder contains support material for writing the journal paper. It is not
the paper text.

## Contents

- `PAPER_SUPPORT_DOSSIER.md` answers the audit questions from
  `/Users/santhoshcheemala/ALL_IN_ONE/doubts.txt` using code paths, line
  numbers, logs, JSON outputs, and explicit caveats.
- `LE_PSI_Paper_Support_Dossier.docx` is a shareable DOCX copy of the dossier.
- `DOCX_QA.md` records the DOCX render-check limitation on this machine.
- `figures/generate_figures.py` regenerates the selected support figures.
- `figures/source_data/` contains CSV data used by the figures.
- `figures/output/` contains PNG and PDF figure outputs.

## Evidence Root

The run evidence used by the dossier is stored in:

`comparative_baselines/results/evidence/psi_repro_20260515_145900/`

That evidence bundle contains:

- LE-PSI single-node results for `m=1000,2000,4000,8000,10000`.
- LE-PSI distributed chunked results for `m=1000,2000,4000,8000,10000`.
- Microsoft APSI 10K same-machine baseline.
- KKRT/libPSI and ALOS22/RELIC attempted-baseline status files.
- `MANIFEST` and `SHA256SUMS`.

## Regenerate Figures

```bash
python3 docs/paper_artifact/figures/generate_figures.py
```

## Verify Evidence Checksums

```bash
cd comparative_baselines/results/evidence/psi_repro_20260515_145900
sha256sum -c SHA256SUMS
```

On macOS without `sha256sum`, use:

```bash
shasum -a 256 -c SHA256SUMS
```

## Caveats to Preserve in the Manuscript

- The final `D=256` runs are reduced-parameter evaluation runs, not full
  128-bit post-quantum security runs.
- The optimized implementation uses visible target leaves for leaf-indexed
  filtering; this is an explicit implementation leakage term.
- The single-node benchmark uses a controlled non-overlap client generator, so
  the targeted-decryption count equals the intended overlap count.
- The benchmark scripts support `CLIENT_MODE=random`. The 2026-05-16
  same-machine 10K diagnostic found `13` reported matches for `10` expected
  matches (`false_positive_count=3`, `correctness_passed=false`), so it should
  be treated as a correctness warning for the current leaf-only optimized path.
- The distributed workload is generated differently from the controlled
  single-node workload; do not compare the 24 distributed 10K matches as the
  same workload as the 10 controlled single-node matches.
- With `sigma=2^30`, do not reuse older exact 140-bit classical / 70-bit
  quantum estimates unless the estimator is rerun with the actual code
  parameters.
- Distributed coordinator JSON does not provide a reliable RAM metric; use
  single-node RSS for memory claims unless distributed RSS is rerun.
