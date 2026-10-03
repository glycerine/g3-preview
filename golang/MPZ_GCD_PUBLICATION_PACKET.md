# Selected GCD and number-theory publication packet

Implemented and frozen on a snapshot based on committed `0324159`. Eight Go
files are ready for parent integration; this document makes no Git or whole
source-unit completion claim. The parent owns central metadata and the final
source/evidence/no-generics audit. Production, tests and tools use concrete
Go types; no generics, unsafe, cgo, production math/big, arithmetic replay or
scratch repair was added.

## Implemented scope and source action order

Read the complete selected `_MP_INTERNAL`, 32-bit LEHMER_GCD bodies in
`src/util/mpz.cpp:560–844`, array/extended GCD and LCM `:845–1015`, Divides
`:2054–2070`, `mpz.h:393–425,632–648`, `mpq.cpp:102–120`, and
`mpq.h:595–629`, alongside the actual registered test bodies and Go methods.
Public GCD/GCDMany/ExtendedGCD/LCM/Divides signatures and scalar U32GCD/U64GCD
are unchanged. No numeral record or manager layout changed.

* `mpnum/mpz_gcd.go`: the one-zero branches now Copy directly into the caller,
  then Abs that caller, matching source `mpz.cpp:581–590`. A failed INT_MIN
  promotion preserves the completed caller Copy as valid inline INT_MIN:
  its existing setter allocates before assigning the new pointer/sign. A
  failed BIG Copy preserves the existing source sign/free/failedRef prefix.
  The general Lehmer branch retains its actual source local copies, matrix
  and remainder arithmetic, final Set/Swap and normal a1/b1/r/t/tmp Del order.
* The same file executes all twelve selected Lehmer SASSERT predicates at
  their actual source sites in `g3debug`. New concrete tag-selected helpers
  use actual manager Compare for ge and source scalar predicates for matrix
  and size bounds. Release does not evaluate debug expressions.
* `mpnum/mpz_number_theory.go`: source ExtendedGCD `:937` ge(tmp1,tmp2)
  executes before aux Copy/division in debug. The already-implemented final
  Bezout diagnostic still performs actual Mul/Mul/Add/Compare with its own
  `errExtendedGCDInvariant` category. GCDMany, signed LCM, Divides, coefficients
  and final publication algorithms remain unchanged.
* `mpnum/mpq_ops.go`: only `integerOperands` integrality becomes debug-only.
  Its four actual raw number-theory callers preserve ordinary owner/value
  checks, operate on source numerators in release, and retain their existing
  denominator-reset and output/alias order. Release fractional GCD/LCM/
  Divides/ExtendedGCD are source raw actions, not mathematical rational gcd.

Unchanged selected algorithms are independently exercised by the full suites:
scalar binary gcd, small signed GCD and unsigned32 boundary, general Lehmer,
GCDMany empty/singleton/early-one/future-item aliases, signed/noncanonical LCM,
Divides zero convention, manager ExtendedGCD and its distinct source-debug
coefficient diagnosis. Rational singleton GCDMany Copy+Abs does not acquire an
integrality assertion. Normalize keeps the single existing GCD/Div algorithms
and real local/retained scratch; no private replacement or rollback was added.

## Exact diagnostic sites

The added Lehmer sites are `mpz.cpp:738` outer-loop ge; `:747` small-tail ge;
`:756–757` BIG predicates; `:760` b_sz<=a_sz before top-limb access;
`:769–772` four source matrix bounds against UINT_MAX+1; `:789` ge before
B==0 selection; `:794` ge after fallback remainder/swaps; and `:813` ge after
matrix updates. ExtendedGCD `:937` precedes the actual division. False selected
source assertions return existing `ErrIntegerAssertion`; ownership/physical
access errors remain distinct. These assertions follow source short-circuits.
The supplemental valid-numeral 1>=2 helper test checks a real false predicate,
not a claim that public canonical GCD can violate its proven loop invariants.
No fake tautology RED or independent postcheck replaces a source assertion.

## Tests-first history and precise dependent corrections

Initial new tests ran on isolated committed `45a32d7` before production edits:
`.cache/gcd-behavior-red.log` (71108) and
`.cache/gcd-behavior-debug-red.log` (28438) failed. Corrected
`.cache/gcd-corrected-red.log` (12229) confirms genuine caller-copy/refusal
and release raw-rational guard differences. The initial MIN expectation
incorrectly anticipated failedRef; complete source setter review corrected
that test model to allocation-before-pointer/sign. Aliases to a distinct old
caller or zero demonstrate actual RED; an alias already containing MIN was
already GREEN. No additional setter/backend defect or missing API was invented.

The first source-table transcription omitted 463. The added independent
trial-division enumeration caught this at position89 before final freeze.
The frozen literal now contains all 168 source primes through997; earlier
campaign runs with that transcription do not count as full source-table
coverage. Corrected expanded selected gate83140 passed all nine groups.

Fresh full mpnum dependent runs on `0324159` ended with exactly the reviewed
old fixture failures: native37834 (23.718s) had two groups, debug72950 (24.966s)
had only the zero-promotion group. Logs are
`.cache/gcd-dependent-native.log` and `.cache/gcd-dependent-debug.log`.
After parent source review, only these old regions changed:

* `TestIntegerGCDZeroPromotionFailure`: retains ErrSlotLimit and untouched
  inputs, but expects actual copied inline MIN/cell0 instead of unchanged17.
  It audits the complete real RationalManager/IntegerManager roots immediately
  before cleanup; no ordinary failure guarantee or helper was weakened.
* `TestNumberTheoryArraySourceOrderAndRationalSingleton`: the final visited
  fractional item returns debug ErrRationalNotInteger after caller gcd2;
  release returns raw numerator gcd1/den1. It audits exact roots before
  formatting and retains all singleton/early-one/future-alias/source fixtures.

Other old tests, common audits, successful independent math/source checks,
service and polynomial production, setters/Copy/Abs, final Bezout helper and
physical storage are unchanged. Tests do not retry arithmetic with repaired
scratch or treat a retired failed descriptor as live numeral storage.

## Executed source and independent coverage

`src/test/mpz.cpp` actually registers `rand_tst_gcd(50,3,2)` and `tst_gcd_bug`.
The new full campaign follows all 168 source primes, the conditional draw/call
shape, actual production Set/Power/Mul/Neg, literal test-only slow_gcd through
real owner operations, then manager GCD and fifty conditional UNREACHABLE
comparisons. A local concrete LCG supplies nonnegative31-bit values before the
source modulo operations; its sequence is explicitly not libc rand. A third
independent big.Int oracle checks input products and gcd. The old 16-prime
coherence property remains useful independent coverage, not credit for this
full source action path. There are zero ENSURE executions in the registered
random body; fifty source comparisons and the added independent checks are
not relabeled ENSURE. The registered MIN/MIN print-only fixture has an added
exact numeral output expectation2147483648; complete ostream traces remain open.
The commented/unregistered 50000-repeat source benchmark is not claimed.

The seven new groups cover both scratch policies, zero directions and output
aliases, real cell exhaustion and BIG-copy refusal with immediate complete
owned-root checks before fault disabling, raw signed/fractional denominator
profiles, actually visited array fractions, source RNG/prime generation,
independent full prime enumeration, debug predicate boundaries, finite signed
GCD/LCM aliases and the source future-element array alias. Existing suites
retain general Lehmer fault cleanup, matrix/small-tail arithmetic, coefficient
aliases/debug diagnosis, normalized rational/service and actual polynomial
content/exact-division consumers. No branch-counter instrumentation or new
claim of public invariant violation was introduced.

Registered rational tst1's existing gcd(12,8)==4 assertion, tst6's nine manager
ExtendedGCD calls (no ENSURE), and tst7's distinct util/ext_gcd.h wrapper999
ENSURE executions remain their existing ports. Manager arithmetic properties
are independent strengthening and do not inflate those source counts.

## Frozen verification and performance

Immutable snapshot `.cache/release/mpz-gcd-final-0324159` contains committed
`0324159:golang` plus exactly the eight files below, excluding other agents'
backend/polynomial/SAT drafts. Single final full native51543 and checked/debug
49510 both terminated0; mpnum times23.156s and24.706s. Full checked/debug vet7229
terminated0. Logs `.cache/gcd-final-native.log`, `.cache/gcd-final-debug.log`,
`.cache/gcd-final-vet.log` record these results; native/debug logs contain
exact nine RUN/PASS registrations verified
against `.cache/gcd-final-tests.json`; all eight SHA256 values stayed unchanged.
Per current user instructions, no race, JS/WASM or WASI checks were launched.
Their verification remains deferred, not silently credited as current gates.
Parent source/evidence/no-generics audit passed after central metadata overlay.

A separate cache-only copy `.cache/perf/gcd-final` adds one benchmark file,
never altering frozen production/tests. Native benchmark42077 terminated0;
`.cache/gcd-benchmark-native.log` records three 200ms repeats per policy on
linux/amd64 AMD Ryzen Threadripper3960X. Each single-goroutine iteration does
actual GCD(a,b,g), LCM(a,b,l), Divides(g,a), with fixed BIG operands representing
6*(2^256-1) and10*(2^192-1). Real RationalManager uses cellPageSize2, WordsPageSize16,
descriptorPageSize2 and a16MiB Budget. Both policies warm100 iterations before
timing, retain6384 actual charged bytes,10 live cells/16 slots/1 retired
metadata slot and14 Words payloads, and refund the whole graph on cleanup.
Local policy repeats63843/59216/57391ns/op; synchronized policy
62518/62635/58544ns/op. Every repeat is0B/op and0allocs/op, with unchanged
post-timing owned Budget. This is a Go warmed-workload measurement, not a
source allocator comparison or whole solver performance claim.

## Remaining boundaries

The numericCellSlab/Words backend, retired failure metadata and source physical
cell/TLS/global-memory topology remain separate parent/arena work. Budget
statistics are not source allocator counters. This selected algorithm/diagnostic
chunk does not change initialization/destruction topology or claim that Go
exception cleanup is literal C++ exception recovery. Underlying source callback,
allocation and failed scratch prefixes remain observable. Source GMP and
inactive BINARY_GCD/LS_BINARY_GCD/EUCLID_GCD profiles, disabled array sorting,
TRACE and persistent external streams stay open. Remaining arithmetic domains,
other source units and full solver/proof integration remain partial.

## Exact frozen inventory

* `mpnum/mpz_gcd.go` — `ba6bcf653236c51946ba314b11f7f6b05d52ce05be7b2507974add085583d706`
* `mpnum/mpz_number_theory.go` — `71a427d3704f89b7dca8111d20038c7f8d3fb7b6f36507bc944be028b0a0116e`
* `mpnum/mpq_ops.go` — `4e2f21c006e5152242dfdacf59e39d6faa84a253e0aacb950e55572da824a18e`
* `mpnum/mpz_gcd_diagnostics_debug.go` — `c95a2f4ded648108741398ab2db6a29150f20f69fce0cea62bfcf123acaabd05`
* `mpnum/mpz_gcd_diagnostics_release.go` — `7aa7930dda533f0b066d1f873345553cfc1979b03af14a52dff0a1191add7718`
* `mpnum/mp_gcd_publication_test.go` — `b46c3aaadb00fe07e888eff59e3b9536554da6fd8b8ea8edc8cb5d484f15d454`
* `mpnum/mpz_gcd_test.go` — `d3f39511334401822a3ddec0d571af73182ef41c598cd3ae70b5e4c6ffbd304e`
* `mpnum/mp_number_theory_test.go` — `9bf333e2ec4c6972a4cbd8a2926cae6e674929d069bf7e8198c31ec17fb6a15e`

Exact nine new/modified native and debug test registrations:

* `TestGCDPublicationZeroCopyBeforeMinimumPromotion`
* `TestGCDPublicationZeroBigCopyRefusal`
* `TestGCDPublicationRationalIntegralityProfiles`
* `TestGCDPublicationRationalArrayVisitedFraction`
* `TestGCDPublicationSourceFull168PrimeCampaign`
* `TestGCDPublicationPrimeTableAndPredicateBoundary`
* `TestGCDPublicationFullFamilySignedAliases`
* `TestIntegerGCDZeroPromotionFailure`
* `TestNumberTheoryArraySourceOrderAndRationalSingleton`

The prior additive broad54 inventory remains `.cache/additive-broad-54-tests.json`
(prior43 division names plus9 additive plus2 inspection/InfInt names). It is a
cache regression inventory, not extra source assertion credit, and no additive
files or central metadata were edited in this task.

## Parent integration evidence

The parent reviewed the source action order, new tests and the two narrowly
corrected old fixtures. Eight live and frozen Go hashes match before and after
the native/debug/vet gates; the complete named registration sets were checked.
Four existing source units retain their partial status. Integrated portcheck
passed after the two central evidence files were overlaid: 2102 source/tool
units, 677 APIs, 33 ported, 148 partial, 1921 unported. Log:
.cache/gcd-parent-audit.log. Final status/progress overlays are documentation
only. Incomplete factorization, allocator and SAT drafts were excluded.
