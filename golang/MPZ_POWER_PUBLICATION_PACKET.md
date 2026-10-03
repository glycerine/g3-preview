# Integer and rational Power/shift source publication

Status: selected implementation complete, source-reviewed and frozen for parent
integration. Production changed only `mpz_power_shift.go`, `mpq_power.go` and
public Copy/Mul in `mpq.go`. Shared normalization needed no implementation
change. This is selected executable internal32-bit coverage, not completion of
whole numeric source units, allocator topology, or solver consumers.

Initial behavioral RED used `292eb4a`; intermediate integration used `e3a76a6`;
final isolated gates use committed `51e5436` plus exactly 13 agent-owned Go
files and the parent-owned `util_params_ref_test.go`. All writes, caches, logs
and compiled outputs remain under `golang`. No generics, unsafe, cgo, or
production math/big implementation was introduced. Go bytes remain frozen.

## Source methods and public routes

| Source | Final concrete route and action order |
| --- | --- |
| `mpz.cpp:1453–1503 power` | `IntegerManager.Power`: original inline2 destructive fast path retained; inline0 debug assertion followed by release Reset; inline1 Set1; otherwise Copy local power, Set caller1, conditional caller Mul, unconditional local square, uint32 mask shift, ordinary lexical cleanup. |
| `mpq.cpp:283–295 power` | `RationalManager.Power`: no integer fast paths; public Copy local power, Set caller1, public Mul into caller/local in source order, final unused square retained, lexical cleanup. |
| `mpz.h:784–790 machine_div2k/mul2k` | Existing output overloads Copy caller output before the in-place operation, including k0. New `MachineDiv2kInPlace(*Int,uint32) error` and `Mul2kInPlace(*Int,uint32) error` expose actual source in-place overloads directly. |
| `mpz.cpp:1629–1684 machine_div2k` | Small signed divide/reset paths; big word/bit shift; source size publication then normalize. Source new-size/equality/bit assertions are debug-only at their actual sites. |
| `mpz.cpp:1691–1748 mul2k` | Small SetInt64 path; source uint32 new-size arithmetic; preserving ensure-capacity; promotion assertion; zero filling; size publication; backward word shift and zero prefix; debug zero-prefix assertions; bit propagation; normalize. |
| `mpz.cpp:1556–1594 ensure_capacity` | Existing single preserving helper retained; newly allocated requested-capacity equality diagnostic added at the source site. Actual Words pool class stays distinct from requested source cell capacity. |
| `mpq.h:666–669 set(mpq,mpq)` | Public `RationalManager.Copy` sets caller numerator before caller denominator; preserves raw shape and partial component publication. No pair work/Move staging. |
| `mpq.h:324–332 mul`, `140–145 rat_mul` | Public `RationalManager.Mul` chooses integer branch before alias mutations. Integer branch: numerator Mul then reset denominator. Other branch: numerator Mul, denominator Mul, normalize caller. No private Power-only duplicate. |
| `mpq.h:56–77 normalize` | Existing actual SYNCH local scratch or unsynchronized retained scratch; integer GCD then numerator Div then denominator Div. Cleanup is unchanged. Inherited GCD/Div internal staging remains separately open. |

Source inline integer zero^zero is **Reset0 in release** and a typed invalid-
power error at its SASSERT in g3debug, before Reset. Rational zero^zero is1.
Generic p>=2^31 is no longer rejected at entry. Execute completed source
iterations through mask31, including the last extra square; return existing
`PowerDomainError{Reason:PowerMaskOverflow}` only on the next actual mask0
continuation. Earlier finite arithmetic/allocation errors take precedence.
Bounded -1, and rational0/1/-1, exercise that actual wrap without huge operands.
The committed integer base2/0/1 finite fast paths remain separate source routes.

Ordinary aliases and reachable raw negative/unreduced rational shapes remain
supported. Copy never normalizes. Mul uses the original branch choice even
when caller aliases either input, and normalization does not silently repair
negative denominator signs. The source INT_MIN,k31 small right-shift +1 defect
remains its explicit independently inferred fixture, not a mathematical oracle.

## Typed storage and cleanup boundaries

All persistent arithmetic storage and scratch still use the existing concrete
cells, Words and Budget owners. No new manager, header, per-object pointer,
private arithmetic algorithm or hidden allocation was added. On typed Go
allocation/domain errors, local power/normalization temporaries are explicitly
cleaned: this is an exception adapter beyond the source raw-mpz normal-path
`del` calls. Completed caller components, manager scratch changes, missing
scratch and exact owned failed descriptors remain observable. No rollback,
implicit scratch repair or retry guarantee was added.

Left-shift newSize uses source uint32 addition. Physical guards now occur at
actual zero/backward/prefix/bit accesses. Concrete Words access length is
prepared within actual retained capacity without publishing numeric cell size;
the source cell size is published only after successful zero filling and before
backward accesses. Existing Words zero-clearing and access-length setup remain
explicit representation adapters. No early uint64 whole-span rejection hides
the source completed size prefix.

A bounded malformed-header fixture (requested capacity6, forged sizeUINT_MAX,
k32) demonstrates newSize wrap1, empty zero loop, published size1, then typed
Bounds at the first unsafe backward access. It is not a claim that a normal
public numeral can own billions of limbs. Only this deliberately forged
fixture restores its original header for teardown; normal error tests do not
repair source-invalid numerals.

## Tests-first evidence and dependent corrections

The first seven behavioral groups observed native RED38838 (all seven) and
checked-debug RED33810 (six; zero^zero debug already correct), logs
`.cache/mp-power-publication-{native,debug}-red.log`. Added public Copy/Mul
component-prefix regressions observed RED49066 native and RED71500 debug:
caller17/5 was incorrectly retained rather than completed numerator3/21;
logs `.cache/mp-power-copy-mul-{native,debug}-red.log`.
Both new in-place APIs produced actual missing-method compile RED before
implementation. Bounded wrapped-header regression observed behavioral
RED36780: eager guard retained sizeUINT_MAX rather than the completed size1.
It then passed after actual-access guard placement.

All existing-file changes were approved only after actual dependency RED:

| Test file / named groups | Exact correction |
| --- | --- |
| `mpz_power_shift_test.go`: SourceShiftDefectAndPowerDomains, RationalPowerIndependentAndDomains, PowerDomainAliases, ShiftUnsignedBoundsAndPromotions | Debug/release0^0 distinction; bounded mask-wrap parity/output; real first-growth error precedence for enormous bases; fresh independent owner graphs after nonretryable source scratch failures; oversized left shift retains completed Copy(MaxInt64). Other math/source fixtures unchanged. |
| `mpq_test.go`: RationalArithmeticFailureCleanup op2, RationalCopySetterFailureCleanup op0 | Immediate actual named-root audit; concrete prior/copied/raw product/normalized component candidates or exact failedRef; denominator cannot act before numerator completes; unchanged non-destination inputs. Other ops retain their assertions and all success oracles. |
| `inf_int_rational_test.go`: InfIntRationalFaultPrefixesTemporaryCleanup cold Copy | Exact owned numerator failedRef, denominator1 and unchanged epsilon7; immediate caller/filler/service graph; source unchanged; cleanup through real InfIntDel. |
| `rational_service_test.go`: RationalServiceCopyFailureAndWarmReuse cold Copy | Exact failedRef/den1, actual guarded read and immediate named graph; unchanged independent source rational; real Del and NewValue before separate successful warm scenario. |
| `mp_parse_test.go`: ParserExceptionCategoriesAndFailurePublication high-mask strings only | Arm real first-growth failure; allocation error precedes mask wrap and retains its error category; parser's separately staged caller remains unchanged. Other exception/domain/math assertions unchanged. |
| `mpbq_select_test.go`: BinarySourceShapeDomainsAndPrefixes huge numerator3 Power only | Real first-growth refusal after source denominator exponent publication and underlying numerator1; immediate full binary/debug graphs. Other domains/math unchanged. |
| `scoped_binary_interval_test.go`: BinaryIntervalFailurePrefixes0^0 only | Lower1 publication; release upper Reset0 succeeds, debug upper source assertion refuses. Same endpoint prefixes and actual ownership audit. |
| `scoped_numeral_test.go`: ScopedRationalVectorIndependentLifecycleAndFailures cold Push only | Source append publishes slot; Copy owns exact failedRef with denominator1; immediate full caller/vector/filler graph; real Close succeeds and refunds exact cold header/backing bytes. Independent random successful model unchanged. |
| `rational_service_ops_test.go`: RationalFacadeCompoundFailureAliases op2 only | Concrete old/raw/normalized component prefixes or failedRef, denominator sequencing, unchanged other operand; actual graph before disabling fault. All other facade operations retain their guarantees. |
| Parent `util_params_ref_test.go`: ParamsReferenceReservationFailureSweeps audit | Recognize only same-service exact owned positive failedRef with denominator1; all other invalid roots remain fatal. Compare separately measured actual cells/payload baseline and audit before disabling faults. No Params production change. |

Initial existing RED is in `.cache/mp-power-existing-red.log`. Broader actual
RED30374/22240 and exploratory43741/42943 logs are retained;
the latter were deliberately interrupted when an obsolete huge Binary fixture
attempted real enormous squaring, and are not final PASS evidence. No production
leak was inferred from a test Fatal skipping its vector Close: the corrected
fixture proves real cleanup/refund. No common ownership helper was relaxed.

## Final test scope and gates

New `mp_power_publication_test.go` has exactly14 added groups. They cover both
in-place overloads, source0^0 profiles, finite error-before-wrap, real mask-wrap
aliases, final-unused-square refusal after completed caller46341, big input
Copy refusal before output1, raw rational finite independent models and Mul
aliases, copy-first shifts, public sequential Copy/Mul refusal and the bounded
header guard. Independent math/big is test-only.

These add **zero new directly registered C++ ENSURE assertions**. Existing
active `tst_2k` assertions continue through actual production parser, Power and
shift routes, with finite independent models. The source `tst_pw2`128 print-only
values have exact independent value checks; complete persistent stream traces
are unclaimed. MPQ's registered prev-power assertions remain their separate
unit; its test file has no direct Power ENSURE.

The final named inventory is exactly31 mpnum groups plus one Params root group:

```text
TestBinaryIntervalFailurePrefixes
TestBinarySourceShapeDomainsAndPrefixes
TestInfIntRationalFaultPrefixesTemporaryCleanup
TestIntegerPowerAndShiftIndependent
TestIntegerSourceShiftTable
TestParamsReferenceReservationFailureSweeps
TestParserExceptionCategoriesAndFailurePublication
TestPowerDomainAliases
TestPowerPublicationBigInputCopyPrecedesOutputOne
TestPowerPublicationFinalUnusedSquareRefusal
TestPowerPublicationInPlaceShiftOverloads
TestPowerPublicationIntegerActualWrapPrefix
TestPowerPublicationIntegerBudgetBeforeMaskWrap
TestPowerPublicationIntegerZeroZeroProfile
TestPowerPublicationLeftShiftCopyPrecedesPromotionRefusal
TestPowerPublicationRationalActualWrapPrefix
TestPowerPublicationRationalBudgetBeforeMaskWrap
TestPowerPublicationRationalCopyComponentRefusal
TestPowerPublicationRationalMulComponentRefusal
TestPowerPublicationRawRationalShapesAndMulAliases
TestPowerPublicationRightShiftCopyRefusalPrefix
TestPowerPublicationWrappedShiftHeaderAccessOrder
TestPowerShiftFailureCleanup
TestPowerShiftWarmStorageAndLifetime
TestRationalArithmeticFailureCleanup
TestRationalCopySetterFailureCleanup
TestRationalFacadeCompoundFailureAliases
TestRationalPowerIndependentAndDomains
TestRationalServiceCopyFailureAndWarmReuse
TestScopedRationalVectorIndependentLifecycleAndFailures
TestShiftUnsignedBoundsAndPromotions
TestSourceShiftDefectAndPowerDomains
```

Final snapshot: `.cache/release/mp-power-publication-frozen`, baseline51e5436,
exact frozen14 Go files. Terminal gates:

| Gate | Handle and result |
| --- | --- |
| Full native `./...` | 61596 exit0; root2.822s, mpnum21.668s |
| Full `g3checked,g3debug ./...` | 98171 exit0; root3.145s, mpnum22.675s |
| Named32 checked-debug race `./...` | 8899 exit0; exact32 RUN/PASS, mpnum22.768s |
| Actual named32 checked-debug JS/Wasm `./...` | 96459 exit0; exact32 RUN/PASS, mpnum6.214s; absolute existing js-exec wrapper |
| Full checked-debug vet | 53039 exit0 |
| Full checked-debug WASI build plus mpnum test compile | 23269 exit0 |
| Source fingerprint, evidence and no-generics audit | 73031 exit0 on HEAD ledger,2102 units/677 APIs,33 ported/145 partial/1924 unported; integrated-ledger audit68048 exit0 with the same2102/677/33/145/1924 inventory |

Logs are `.cache/mp-power-frozen-{native,debug,race,js,vet,audit}.log` and
`.cache/mp-power-frozen-integrated-audit.log`; WASI test output is
`.cache/mp-power-frozen-mpnum-wasi.test`. Independent extraction verified exact
RUN and PASS sets against `.cache/mp-power-final-tests.json` in both race/JS
logs. All14 current and snapshot hashes were verified unchanged.
The parent also independently reports full native12345/debug15696,
actualJS32643, vet30432, fullWASI31103 and root/mpnum WASI compiles33790/26883,
all terminal0; parent exact32race60512 and integrated audit79699 are GREEN.
Parent owns central manifest/progress and the Params test.

## Frozen ordered file inventory

The first13 files belong to this numerical packet; the final root test is the
parent's approved dependency correction. Source production is only the first3.

| File | SHA256 |
| --- | --- |
| `mpnum/mpz_power_shift.go` | `d46ce8c9c8a611e03b9338cedcfd215f05a998eae2b570721ee1f3c290d98ac2` |
| `mpnum/mpq_power.go` | `5e2dd0fed4a55e3fd0767e807eea9d1f2d0cf21e8f23f2997a8b0953d6de537a` |
| `mpnum/mpq.go` | `7e0b16d2a84677a7c3cc04b43ef4f7be10352f587dfea625033c92babeee2f58` |
| `mpnum/mp_power_publication_test.go` | `4e6726e46e2c0e31dd8f843105777e6f9f99b7b71e71fe43c771d65b21793c8b` |
| `mpnum/mpz_power_shift_test.go` | `9fb759e7ca742e7deb502968c85ead854d5188a171bccd5ff796c85ca5767606` |
| `mpnum/mpq_test.go` | `454df3f6e20331a2415436804c6be7bca329d5a41d2a85a5ab13057cec4a9502` |
| `mpnum/inf_int_rational_test.go` | `9639a2d7ca28b90bdb998450b3cb629d7c09f079a504d2b80788029192b09b17` |
| `mpnum/rational_service_test.go` | `5c1dd2c1387e0f43964f58220ad1647cce38e32790d9e25149ed5f96c3e51392` |
| `mpnum/mp_parse_test.go` | `663645c980602cdadd90df7edb74909f3c33b112990981ffdc2633e54fd7303e` |
| `mpnum/mpbq_select_test.go` | `4397937acc80c563dc4492c83c739f3a3b865509a023c459f97b00b1e72b8286` |
| `mpnum/scoped_binary_interval_test.go` | `4e969138e9b3ca05a508ec18d6ff939a2476674c006df5a96f828fda0dd8c6b6` |
| `mpnum/scoped_numeral_test.go` | `3d873b070567fde77bbd16f6c60d09cdd06df32a8c60630d1daff8b860aed486` |
| `mpnum/rational_service_ops_test.go` | `5bfcbef478c22c4bb493e5594009b4573b0dd8b5d6ba00dfb92fcf308446c758` |
| `util_params_ref_test.go` | `4b30348a457d21d41b3523cca3709782fa357e27c9ff59143b6bc88a230ead3d` |

## Remaining boundaries

Selected public caller publication is reconciled; inherited Integer GCD/Div
staging is not. Other Add/Sub/Div/Inv, roots, bitwise and wrapper failure-action
debts stay open where previously identified. Parser staging stays explicit.
Pure infallible scalar setter pointer/sign/header order, Words tail clearing,
small-object allocator/global/TLS accounting composition and complete source
numerical allocator topology remain prior review boundaries. The actual new
raw allocator service is not silently substituted for Budget ownership here.
GMP/nonselected backends, TRACE, custom persistent ostream and complete
solver/AST/polynomial consumers remain open. Missing scratch/failedRef is a
source-prefix lifetime adapter with guarded reads and explicit cleanup, not a
valid arithmetic value or implicit numerical recovery contract.


## Parent final commit verification

Parent frozen tree .cache/review/mp-power-final-51e5436 contains the exact14 Go
files above plus integrated manifest and status documents. Full native12345
and checked/debug15696 exited0 (mpnum21.586s/22.073s). Race60512 and actual
JavaScript32643 exited0; independently extracted RUN/PASS sets exactly match
all32 named declarations across root and mpnum packages. Vet30432, full WASI
build31103 and root/mpnum WASI test compilations33790/26883 exited0. Integrated
source-evidence/no-generics audit79699 exited0 at2102units/677APIs with the
unchanged33ported/145partial/1924unported ledger. Agent integrated relative-source
audit68048 also exited0. Source SHA values and final Go hashes were independently
checked; only this document changed after the final gates. Stage/index hashes
will be checked against .cache/mp-power-parent-final-inventory.json before the
coherent commit.
