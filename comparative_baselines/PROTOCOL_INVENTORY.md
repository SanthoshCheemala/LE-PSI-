# Comparative PSI Protocol Inventory

Target VM for same-machine runs: `psi-compare` (`e2-highmem-8`, Debian 12).
Target size for external baselines: `m=10000`, `n=100`, expected overlap `10`.

| Protocol | Public code status | Runner |
| --- | --- | --- |
| Microsoft APSI | Public: https://github.com/microsoft/APSI | `comparative_baselines/apsi/run_apsi_10k.sh` |
| KKRT16 | Public: https://github.com/osu-crypto/libPSI | `comparative_baselines/kkrt_libpsi/run_kkrt_10k.sh` |
| ALOS22 laconic PSI from pairings | Public RELIC demo: https://github.com/relic-toolkit/relic/tree/main/demo/psi-client-server | `comparative_baselines/alos22_relic/run_alos22_10k.sh` |
| HE-PSI / Chen-Laine-Rindal 2017 | Paper and description are public, but I did not find a separate maintained runnable repository. Microsoft APSI is the maintained Microsoft HE-based PSI implementation used as the runnable HE baseline. | No separate runner unless a source repository is identified. |

