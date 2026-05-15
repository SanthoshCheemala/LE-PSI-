# KKRT/libPSI status

Attempted on the same psi-compare e2-highmem-8 VM. The build fails in the libPSI dependency stack while compiling macoro unit tests under GCC 12/C++20 before frontend.exe is produced. No KKRT runtime result is reported.

For paper fairness, do not compare a hand-modified or partially built KKRT binary unless this upstream build issue is resolved cleanly and reproducibly.
