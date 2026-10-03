# Extended-GCD source debug diagnostic

The src/util/mpz.cpp:969 Z3DEBUG branch runs under the g3debug build tag,
independently of g3checked handle validation. It multiplies a by the actual r1
into nexta, b by the actual r2 into nextb, adds into nexta, then compares with
tmp1 before publishing the gcd. All six existing source locals are cleaned in
the caller on arithmetic failure or assertion failure. No extra manager fields,
generic types, heap records, or unbudgeted numerical storage are introduced.
The release helper performs no arithmetic and preserves existing alias behavior.

Two tests began missing-diagnostic RED. They cover 625 independently checked
signed Bezout cases, an input/coefficient alias whose source assertion fails
before publishing gcd (coefficients 3/-11 survive, prior gcd 17 survives), and
40 allocation prefixes of large exact diagnostic arithmetic. Immediate root
audits precede formatting or cleanup; every completed owner fully refunds its
budget. Source assertion failure is adapted to an internal classified error,
which propagates through the existing rational manager and service forwarding.

This tag implements this audited diagnostic branch; it does not claim every
source Z3DEBUG or trace branch is translated. Existing finite release-alias
tests retain their release expectations; the affected alias fixtures now also
check the source debug failure and pre-publication outputs under g3debug.

The final isolated snapshot passed the full native default suite and full
g3checked+g3debug suite. Checked release number-theory tests, checked debug race
tests for both number-theory and diagnostic groups, checked debug vet, and
checked debug WASI compilation passed. Actual checked debug JavaScript/Wasm
execution passed the full mpnum suite, including explicit RUN/PASS records for
both new groups and all nine existing number-theory groups in
`.cache/extended-gcd-debug-final-js.log`. WASI runtime is not claimed.
Source inventory and no-generics syntax checks passed; source-unit coverage
remains 21 ported, 42 partial, 2039 unported of 2102, with 677 API operations.
