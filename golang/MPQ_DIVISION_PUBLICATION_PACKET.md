# Selected rational division and reciprocal publication — final packet

Implemented and frozen, based on HEAD45f5d4b. The initial source audit was
on57d0c64 and tests-first RED on6ffd717. Seven approved Go files are immutable;
parent owns the three root sexpr fixture files, central metadata, integration
and Git. No IntegerManager, Natural division, service, allocator or shared
coefficient-store production changed. All artifacts remain under golang; no
Go generics, unsafe, cgo, production math/big, alternate/replayed arithmetic or
missing-scratch repair was introduced.

The approved historical audit/design below is retained for precise source
mapping. Final executed scope, chosen zero-domain contract, RED/GREEN evidence,
immutable hashes and named inventory follow after it. Whole source units and
full solver/consumer goal remain partial/open.

## Current evidence and stale claims

The entire selected division bodies were read: mpz.h machine_div_rem,
machine_div, rem, div, mod at426–498; mpz.cpp big arithmetic345–549;
mpq.h normalize56–77, additive/multiplicative76–155, division443–560 and
setters666–717. Current Go mpz.go publication/addition/setters,
mpz_div.go, mpq.go, mpq_ops.go and actual service operator routes were read,
as were the complete registered src/test/mpq.cpp, relevant src/test/mpz.cpp
and rational.cpp tstmod/tst9 plus corresponding permanent Go tests.

**Integer division is already direct.** The historical staged transaction
from c3b3e9a was removed by `fdf6f3c` (git log for mpz_div.go). At current
mpz_div.go139–183, quotRemCore uses the actual two retained scratch cells,
executes one Natural Algorithm D, publishes quotient via publishTmpAt0 then
remainder via publishTmpAt1. The shorter-numerator branch Copies r then
Resets q. mpz.go579–605 publishes sign/pointer/size and transfers the actual
scratch before any replacement allocation. It neither prepares both outputs
atomically nor Move-publishes a private result pair. These match
mpz.cpp474–507 and source setter transfer semantics, with documented storage
and typed failed-pointer adaptations.

Current Div at mpz_div.go89–112 publishes MachineDivRem into c and a real
local remainder before reading **actual b.val** for Add/Sub adjustment.
Mod at115–131 publishes Rem into c then reads actual b for Add/Sub. Therefore
c==b retains the actual source overwritten-divisor defects; there is no shadow
transaction here. Source mpz.h466–496 uses exactly those rereads. The existing
TestIntegerEuclideanStagedFailurePointsAndAliases name is historical; its
current assertions recognize real scratch/free/publication prefixes.
The committed bitwise packet's inherited-Mod/Div-staging caveat is stale and
must be corrected by parent during reviewed documentation integration.
Do not modify working integer division or invent a new RED for absent staging.

Power/shifts, public rational Copy/Mul and bitwise staging/entry-domain debts
were closed in96e3058/57d0c64. Earlier remaining-publication prose about those
methods is historical. Integer normalizeOwned exists. Rational normalize
already performs actual GCD, numerator Div then denominator Div, using local
GCD scratch for SYNCH and retained nTmp otherwise. It is not missing and
must not be duplicated. Its caller can still hide its public actions by
normalizing a staged pair; that remains a real defect below.

## Baseline audit: method families before this selected closure

| Current methods | Source/current conclusion |
| --- | --- |
| Integer Add/Sub | Source big_add_sub algorithm is present and publishes through actual source scratch transfer; inline signed64 arithmetic routes exist. No whole-result staging remains. Source real_sz<=sz and borrow==0 debug assertions are incompletely selected: Go has an unconditional final borrow panic and different physical bounds adapters. Keep these diagnostic/backend debts distinct from arithmetic absence. |
| Integer MachineDivRem/MachineDiv/Rem/Div/Mod | Algorithms, all modes, signs, shorter-numerator order and finite alias defects are present and direct. Dummy-zero SASSERT after big_div/big_rem is not explicitly selected. Small MachineDivRem preallocates the promoting INT_MIN/-1 quotient before invoking SetInt64; currently ordinary failure outcome agrees but literal setter action/field ordering needs later backend review, not invented behavioral RED. Integer DivideByZero is an explicit guard before actual undefined /,% or Algorithm D domain. |
| Integer scalar SetInt32/Uint32/Int64/Uint64/SetLimbs/Copy | Actual routes and ordinary allocation-prefix corrections exist. Capacity/setter assertions are selected in the committed setter packet. Pure infallible sign/pointer/header ordering in setBig64 and payload zero clearing versus source untouched capacity remain acknowledged representation debts. No blanket claim of missing setters. |
| Integer Inv | Source body SASSERT(false), release no-op. Current always typed ErrIntegerInvUnsupported is a documented unsupported-domain adapter, not a field inverse. Keep it open as an exact release diagnostic-profile difference, without providing fake arithmetic. |
| Rational Copy/Mul/Power/bitwise | Direct source component/caller routes now selected. Safety/provenance/initialized-numeral boundaries still exist. No staged-pair debt should be assigned to these methods on this HEAD. |
| Rational Add/Sub | addSub stages a full work pair, including integer fast paths. Source rat_add/rat_sub cross-products, denominator publication, numerator Add/Sub and normalize should affect caller. This coherent additive family is a separate follow-up. |
| Rational SetFraction/SetParts | Full staged pair hides sequential source setter/sign/normalize prefixes. SetParts tests scalar signed denominator before temporary Copies/Neg but source branch is chosen before Copies; component alias and read-after-publication semantics need closure in a separate complete setter family. |
| Rational Inv/Div | Actual successful reciprocal/division algorithms are present; whole work pair + Move hides source Copy, component Mul, Neg, Swap and normalization prefixes. Div's real b==c local must be preserved, rather than eliminating all locals. |
| DivInt/AccDiv/RatDiv | Whole-pair staging plus emulated component-alias rereads is real. Remove that staging using actual public integer operations and actual caller components, not private equivalents. |
| IDiv/MachineIDiv/Rem/Mod and ToInt variants | Already direct integer forwards and denominator-after-success. Shared integerOperands currently applies integrality in release although source only SASSERTs; change only these source forwards and reviewed dependent callers, keeping unrelated number-theory policies explicit. |
| Mixed AddInt/MulInts/MulInt/fused family | Mixed primitive pair staging remains. Fused AddMul/SubMul actually use single public Mul then Add/Sub with the source SYNCH retained/local temporary; they inherit staged additive/mixed primitive publication. Do not advertise the whole fused family closed by changing its division neighbors. |
| Floor/Ceil, parse/root/number-theory | Separate stages remain in some routes, documented per existing source bodies. These are not replaced by division replay/private algorithms in this packet. GCD's zero-operand branch stages Copy/Abs; inherited GCD publication/diagnostic debts remain separate even when called by normalize. |

## Approved design history: complete executable division/reciprocal scope

Existing signatures stay unchanged:

- RationalManager.Inv(a,out), Div(a,b,out).
- RationalManager.DivInt(a,bInt,out), AccDiv(a,bInt), RatDiv(aInt,bInt,out).
- IDiv, MachineIDiv, Rem, Mod and IDivToInt, MachineIDivToInt, RemToInt,
  ModToInt: all actual source integer-result overloads and rational result variants.
- Add concrete InvInPlace(a *Rational) error for actual inv(mpq&) source overload.
  Observe missing API RED separately before implementation. Output Inv must
  use public Copy(out,a), then InvInPlace(out), rather than a special substitute.

Actual action order:

| Method | Required single source route |
| --- | --- |
| InvInPlace | Debug !is_zero before numerics; if numerator negative, Neg num then Neg den; Swap num/den. No GCD, denominator repair or allocation work pair. |
| Inv(a,out) | Copy caller numerator then denominator using already corrected public Copy; call actual InvInPlace. Thus debug zero assertion occurs AFTER Copy, not at method entry. Self output retains source Copy self no-action. |
| Div, b!=out | Mul(a.num,b.den,out.num); Mul(a.den,b.num,out.den); if actual out.den negative, Neg num then den; normalize(out). |
| Div, b==out | Actual local tmp: Mul(a.num,b.den,tmp); Mul(a.den,b.num,out.den); Copy(out.num,tmp); Del tmp at source statement before signs/normalize. Error cleanup is lexical Go adaptation. |
| DivInt | Copy(out.num,a.num); Mul(a.den,b,out.den); read actual b sign AFTER these publications; Neg num then den if negative; normalize caller. Component b==out.num/out.den follows actual overwritten object automatically. |
| AccDiv | Mul(a.den,b,a.den); reread actual b sign, Neg a.num then a.den if negative; normalize a. No inserted numerator Copy. |
| RatDiv | Copy out.num from a then out.den from actual b, normalize. Deliberately no sign repair: raw negative denominator remains reachable. |
| IDiv/MachineIDiv/Rem/Mod | Source debug integrality at the actual entry; actual Integer operation into caller num or Int output; rational denominator reset only after success. Preserve finite release fractional/raw input action on numerator, existing divisor/output defects and all error prefixes. |
| normalize | Reuse actual current public GCD and Div once each in source order; no precomputation, second division replay, private GCD or scratch repair. Test both storage policies and actual second component prefix. |

Inv currently tests zero before Copy and stages all changes. Even debug
out-of-place zero therefore has a meaningful source-prefix RED: source first
Copies0/1 into caller, then fails the actual Inv assertion; it does not preserve
the destination sentinel. Release zero and division zero are distinct below.

## Approved design history: zero-domain and owning-result contract

Source inv(mpq&) has only SASSERT(!is_zero(a)). Release Inv(0/1) executes Swap
and finitely leaves1/0. Output Inv must first Copy0/1 into destination; debug
assertion then retains copied0/1. For source INT_MIN numeral Neg, actual
promoting allocation can fail BEFORE denominator Neg/Swap. No whole-pair
rollback or retry using destroyed/missing scratch is valid.

Source rational Div/DivInt/AccDiv/RatDiv have no early zero-divisor guard.
For nonzero numerator and zero divisor, real component operations can leave
n/0; normalize computes gcd(n,0)=abs(n), then returns immediately if gcd1 or
Divs numerator to±1 and denominator to0. These are finite raw results.
For0/0, real fields become0/0 before gcd0 and the first Integer Div(num,0,num)
reaches unsafe arithmetic; ErrDivideByZero belongs there. If an earlier
Copy/Mul/GCD allocation fails, that failure and its completed prefixes precede
any later domain condition. Do not blanket-reject all zero divisors at entry.
Component aliases can change what zero/sign is subsequently read; derive
fixtures from exact action routes, rather than original scalar values alone.

Current RationalManager.check rejects denominator0 as ErrUninitializedRational;
checkCleanup does too. These guards intentionally distinguish NewRational0/1
from raw Go zero and prohibit ordinary invalid-numeral reads. They cannot be
silently equated with a source arithmetic exception. Full owning cleanup is a
prerequisite once caller fields can hold source-finite zero-denominator results.
Proposed alternatives for parent review:

1. **Preferred raw action fidelity:** return nil from source-finite Inv/Div
   actions and retain exact1/0 or±1/0; ordinary check/String/next arithmetic
   still returns ErrUninitializedRational. Cleanup-only Del validation checks
   actual integer field ownership but permits denominator0, so callers can
   destroy all live/failed components. This also permits harmless cleanup of
   raw Go0/0; it does NOT initialize, lazily normalize or allow reads of it.
2. **Explicit Go post-action domain boundary:** return ErrUninitializedRational
   AFTER all source-finite actions/publications, with the same cleanup contract.
   This is a Go adapter; source does not throw there. It can alter service
   value-result behavior and must be documented/tested as such.

Option1 was approved and implemented; Option2 remains historical and was
not used. The distinctions below remain real ownership boundaries. Neither alternative
claims true source wrapper equivalence for denominator0. Source rational value
operators can copy raw1/0 through their ordinary field setters. Current service
value-result Copy/Move/check routes reject it as an initialized-numeral boundary.
Keeping service production unchanged would preserve a safe empty owning result
on that Go boundary error with local drop cleanup, while compound assignment
retains its actual invalid raw destination. Those differences must remain
explicitly open; do not add an unreviewed private unchecked-copy surrogate or
new initialized marker to pretend source equivalence. A later full owner-value
representation/bridge contract could reconcile these transfers separately.

## Source tests and complete tests-first inventory

Actual registered src/test/mpq.cpp contributions: tst0 has one ENSURE dividing
2/3 by4/3 into b and comparing against mk_q1/2; bug1 and bug2 have one each
for in-place Inv±2→±1/2. Exactly **three existing source ENSURE routes** are
in this raw reciprocal/rational-division scope; current permanent
TestRationalSourceDivAndInverseAliases verifies their numeric expectations in
both scratch modes. Refine new source-route tests through actual manager Equal
and real setup/destruction actions without counting mode repetition as new
source assertions.

Registered rational.cpp tst9 calls tstmod for eight concrete signed cases,
three ENSUREs each (reconstruction, nonnegative remainder, less than abs(divisor)):
**24 existing assertions**, routed through actual service IDiv/Mod value-result
and multiplication/addition/comparison primitives. Current
TestRationalOpsSourceTst9All24Assertions is a raw-manager adaptation; preserve
its24 credit and source route distinction. Existing facade source TST1/TST2
coverage remains independently counted; no source assertion inflation from
random/fault loops or print-only source mpz tst2/tst2b.

Proposed FIRST new namespaced `mp_div_publication_test.go`, isolated actual
HEAD57d0c64 plus only tests (no arena/default-backend/coefficient drafts):

- Missing InvInPlace API RED separate from behavioral tests.
- Div actual b==out and b!=out action-prefix fault sweeps for both SYNCH modes,
  destination all aliases including a==b==out; demonstrate caller numerator
  or denominator publication before later failure and actual local Copy order.
- Inv Copy-before-assert, direct INT_MIN promoting Neg failure prefix, successful
  positive/negative reciprocal, raw negative/unreduced denominator, self versus
  independent result; precise owned failedRef versus untouched later component.
- DivInt/AccDiv/RatDiv all ordinary output aliases and component b aliases;
  actual sign reread quirks, numerator-before-denominator failure, raw negative
  RatDiv denominator; no mathematical canonicalization assumption on raw shape.
- All eight integer-valued forwards: release fractional/raw inputs use actual
  numerator, debug integrality first; denominator remains old on integer error,
  resets only after integer success. Direct divisor/output bugs frozen separately.
- Zero source finite actions versus0/0 endangered Div0 and earlier actual budget
  failure precedence, chosen read/cleanup/value-return boundary, no early guard
  disguised as source behavior and no fake C++ observation.
- Normalize chosen inputs with nontrivial common divisor to expose actual first
  component Div publication then second; immediate complete live cells/Words
  audit BEFORE formatting, fault disable, Del or Close. Retired failed descriptor
  is not live; missing scratch is not repaired. Fresh owners for independent
  remaining cases after cleanup, not automatic arithmetic retry.
- Independent bounded big.Int/Rat arithmetic only in tests for valid nonzero
  mathematical cases; explicit source-action expectations for raw shapes and
  defective aliases. All reservation sweeps require actual successful completion.
- Full real service value-return/compound/scalar/IDivInto/ModInto routes,
  provenance first, copy-before-compound/returned-lvalue copy lifetimes; no direct
  manager shortcut. Empty returned owner on Go error and all local roots cleaned.
- Warm exact zero-allocation tests, actual retained scratch/peak graph, realistic
  warmed timing benchmark in cache-only tree if storage performance is discussed.

This section records the approved tests-first plan. Actual observed RED/GREEN
and bounded final fixtures are recorded below; no hypothetical source fault
is credited as observed execution.

## Approved design history: file and dependency boundaries

Production candidates after approval:

- `mpnum/mpq.go`: ONLY Inv/Div, actual new InvInPlace, minimal selected cleanup
  validation if chosen; normalize unchanged except an actual proven defect.
- `mpnum/mpq_ops.go`: ONLY DivInt/AccDiv/RatDiv and integer division/rem/mod
  entry checks. Do not accidentally change global integerOperands semantics for
  unrelated number-theory callers; use source-specific entry guard when necessary.
- New `mpnum/mp_div_publication_test.go`; this proposal/status document.

Current IntegerManager, Natural Algorithm D, allocator/backend, coefficient
store and service production are read-only validation dependencies. No new
owner/header/scratch fields or Go slices per hot numeral are needed. Six-field
or four-field bitwise locals are irrelevant to this division family's actual
single local tmp/GCD-local ownership. Removing work pairs changes real peak
and reservation count; account actual existing cell/Words storage, never claim
source single-object allocation topology from pooled budget reservations.

Only AFTER actual dependent RED, request narrow review for existing tests:
mpq_test.go op3Div/op4Inv failure sweeps, cold inverse assertion/retry assumptions,
source-zero guards; mpq_ops_test.go operation2DivInt/3RatDiv/7AccDiv and debug/
release integer-domain guards; rational_service_ops_test.go DivAssign failure
branch and any observed owned-value result behavior. Preserve Add/Sub/fused
atomic expectations while those algorithms remain staged; preserve all source
and successful independent oracles. Any other dependent real RED needs its own
source-action mapping before editing. Common qopsAuditLiveOwnership is parent
owned and must not be weakened. Service tests remain test-only adaptations.

Final freeze: exact approved file hashes/names, fresh committed HEAD plus only
selected overlays, full native and checked/debug, actual named race and JS
RUN/PASS, vet, full WASI and test compiles, integrated source fingerprint/
evidence/no-generics audit. Parent owns metadata/Git and stale old-doc correction.
Whole mpz/mpq/rational source units remain partial; GMP/64-bit-limb, TRACE,
persistent ostream, some source diagnostics, infallible scalar/physical field
ordering, true allocator/TLS/global-memory composition, initialization hooks,
full solver and proof/tactic consumers remain open.

## Executed final scope and zero-domain contract

Production changed ONLY mpq.go Inv/Div/newInvInPlace and the denominator check
in cleanup-only validation; mpq_ops.go ONLY DivInt/AccDiv/RatDiv and integerOp
entry checks for the eight actual division/rem/mod overloads. Shared
integerOperands and unrelated number-theory policies were left unchanged.
Public Copy, normalize, GCD, Integer Div/Mod, Natural Algorithm D and service
production remain the actual existing single algorithms. The real b==out
numerator local is destroyed at its source statement BEFORE sign repair and
normalize; a lexical deferred cleanup handles earlier Go errors only.
No staged work pair, shadow divisor, private arithmetic or replay remains in
these selected routes. Source SYNCH local versus retained normalize scratch
is preserved; shared concurrent arithmetic is not newly claimed.

Option1 is executed: finite source Inv0/1→1/0 and division n/0→±1/0 returns
nil with exact raw fields. Ordinary String/read/arithmetic still returns
ErrUninitializedRational for denominator0. Explicit Del/Reset checks actual
integer ownership, including own failedRef, but permits denominator0. It can
also safely destroy/reset raw Go0/0; this is an explicit cleanup/reset action,
not lazy numeral initialization or permission for ordinary raw-zero reads.
0/0 publishes real components, computes GCD0 and reaches ErrDivideByZero
only at the actual first Integer Div0 call. Actual earlier allocation errors
retain precedence and caller fields. Debug inverse zero assertion uses existing
ErrIntegerAssertion after public Copy; in-place assertions precede Neg/Swap.

The source-finite invalid-pair contract is intentionally distinct from the
existing service value-result transfer boundary. Service Div value-return still
Copies left, performs actual raw compound Div, then its result Copy rejects
denominator0, returns an empty owned token and destroys the raw local's owned
components. Compound DivAssign finitely publishes1/0 and returnsnil; its token
still has provenance and can be Del'd, while normal reads reject it. No source
wrapper equivalence is claimed for this Go invalid-numeral boundary. A real
source owning-value representation/bridge review remains separate; no unchecked
private copy or initialized marker was invented here.

## Actual TDD and precise dependent corrections

Isolated baseline6ffd717 with only new tests observed behavioral RED:
release Inv zero early DivideByZero, rational zero-divisor early rejection,
fully staged caller failure result and universal release fractional integer
preconditions. Debug RED additionally retained the old caller sentinel71 rather
than public Copy0/1 before the actual inverse assertion. Logs
`.cache/division-behavior-red.log` / `division-behavior-debug-red.log`.
Separate missing InvInPlace API RED is `division-api-red.log`.

Added action-focused baseline regressions in `division-extra-red.log` observed
out-of-place INT_MIN inverse retaining sentinel71 on Neg allocation refusal
instead of the completed Copy MIN prefix, and zero division incorrectly taking
precedence over actual first growth refusal. These are real bounded arithmetic
errors, not fabricated allocator callbacks. Slot-fill fixtures use actual
Live+Retired<Slots and charged manager storage. The source-three-assertion group
was immediately GREEN; no new defect or RED is claimed for those already
correct successful routes. New raw-sign/alias, zero-output alias, inverse
Copy-before-debug-assertion and double-MIN Neg fixtures retain finite source
prefixes. Ordinary positive big in-place inverse may reserve nothing: it is
not required to manufacture a budget failure merely to satisfy a test sweep.
All failing sweep cases receive exact component candidates and immediate roots.

After the approved production change, full mpnum native26573/debug24940
observed the following dependent REDs; only their parent-approved branches
changed (logs `division-dependent-native-red.log` / `...debug-red.log`):

- mpq_test CheckedInitializationAndZero: source debug inverse assertion versus
  release1/0 and finite nonzero division byzero. Ordinary raw-zero String and
  SetInt64 guards remain; fixtures explicitly Reset between invalid finite results
  and test actual Del. SetFraction zero policy is unchanged outside this scope.
- mpq_test ArithmeticFailureCleanup: ONLY op3Div/op4Inv failure branches permit
  actual old/cross-product/sign/Copy/Neg/Swap/reduced component values or exact
  owner failedRef, and preserve non-destination inputs. All Add/Sub and Mul
  guarantees/oracles remain. Cold inverse test was already GREEN and unchanged.
- mpq_ops IntegerProductsRatDivQuirksAndGuards: finite zero denominator raw fields,
  explicit Reset fixture actions and debug-only fractional division precondition.
- mpq_ops AllPredictableFailurePrefixes: ONLY op2DivInt/op3RatDiv/op7AccDiv
  component-prefix expectations. Add/multiply/fused and all successful oracles
  remain. Immediate full roots precede fault disable/rendering.
- service CompoundFailureAliases: ONLY op3DivAssign raw cross-product/sign/reduced
  caller fields. Other source wrapper operations and all source assertion counts
  remain. Service checks actual provenance before numeric actions.
- ScopedNumeralScalarFailureAndAllThinOperations fractional Rem: debug leaves3
  with ErrRationalNotInteger; release uses raw divisor numerator1, publishes0/1.
  Scope roots are audited immediately and the owner closes normally.

The full45f5d4b native module exposed one additional parent-owned root RED:
TestSexprBVSourceWidthQuirks for BV1/2,width4. Source sexpr.cpp138–145 takes
Rem→IDiv→unsigned digit. Release now prints#x1 then the existing
ErrSexprDisplayContinuation; debug fails the actual Rem integrality assertion
following#x. Parent independently reproduced RED93250 and owns the narrow
common/tag-selected test helper correction in three root test files. Sexpr
production and all other width cases are unchanged. This is a dependent source
profile fixture correction, not a new arithmetic or printer algorithm.

No scratch is repaired after a source transfer/free failure, no invalid source
numeral is silently made readable, and no common audit is weakened. Tests
observe all public/scratch cells and Words BEFORE disabling faults, rendering,
Del or Close. Only own exact retired failedRef is nonlive metadata; any other
invalid named root is fatal. Source-style normal integer locals lack C++
destructors; lexical Go error cleanup remains an explicit ownership adapter.

Source credit remains the three actual registered MPQ ENSUREs tst0/bug1/bug2,
and existing24 rational tst9 assertions with the documented raw-manager
adaptation versus source service routes. Sixteen new groups are added action/
profile/math/ownership tests, not sixteen newly ported C++ ENSUREs. All existing
facade TST1/TST2 source counts remain independently unchanged.

## Frozen files and verification evidence

Seven Go SHA values and43 mpnum named groups are in
`.cache/division-final-hashes.json` / `division-final-tests.json`.
Authoritative agent snapshot `.cache/release/mpq-division-final-45f5d4b` is
45f5d4b plus exactly those seven Go. Initial6ffd717 final native/debug/race
sessions were interrupted130 to adopt the parent-required committed shared-store
base; old completed JS/vet/WASI are superseded, not claimed as final integration.

On exact45f5d4b frozen bytes: full native86408 mpnum22.705s, with sole root
sexpr expectation RED described above; full checked/debug62458 exited0
(mpnum23.335s). Exact43 RUN/PASS sets match the inventory for checked/debug
race94316 exit0 (39.571s) and actual checked/debug JS25473 exit0 (12.755s).
Vet46548 and full checked/debug WASI build plus mpnum test compilation96292
exited0. Logs `.cache/division-final-{native,debug,race,js,vet}.log`, WASI
`.cache/division-mpnum.wasm`. Parent's corrected ten-Go integration adds the
three root profile fixture files; its full module/44-name evidence is separate
and will be appended after terminal verification. No further Go edits are planned.

| Frozen approved Go file | SHA256 |
| --- | --- |
| `mpnum/mpq.go` | `1633a39d950bd75c217c854182248acec2afcfe4431b95bfe7fc2d2cf3528ffa` |
| `mpnum/mpq_ops.go` | `e7d8e21d7ca72996787baf12124c66d59305341e3a7ebf94e376c43903a20600` |
| `mpnum/mp_div_publication_test.go` | `3d70894aa5dc355ae55f2ceb9d5e70b9a52b429ef5565dbe480767c42eb7e761` |
| `mpnum/mpq_test.go` | `343862bb1b4b6a3bb07ece9b6ff34957787c786137e4ab5a2e8340af366266a8` |
| `mpnum/mpq_ops_test.go` | `035d6608b521278b453137bc4f8fae61b5b7e0438238931f56083a205b7ab529` |
| `mpnum/rational_service_ops_test.go` | `0722b52568e308f679c492fcce739b14e6149d455d2bbca461a15a77226f9012` |
| `mpnum/scoped_numeral_test.go` | `426cd3096a2fe4d36d06c4b6939fa95bf74e39308fc4294ad674df150843c79a` |

Exact43 named mpnum groups (16new+7core rational+8mixed+11facade+1scoped):

- `TestDivisionPublicationInverseZeroProfileAndCleanup`
- `TestDivisionPublicationZeroDivisorFiniteActions`
- `TestDivisionPublicationCallerFailurePrefixes`
- `TestDivisionPublicationSourceThreeAssertions`
- `TestDivisionPublicationFractionalIntegerForwardProfiles`
- `TestDivisionPublicationInverseInPlaceOverload`
- `TestDivisionPublicationInverseMinNegFailureOrder`
- `TestDivisionPublicationEarlierAllocationPrecedesZeroDomain`
- `TestDivisionPublicationComponentAliasSourceActions`
- `TestDivisionPublicationRawUnreducedAndDenominatorSigns`
- `TestDivisionPublicationServiceInvalidResultCopyAndCleanup`
- `TestDivisionPublicationIntegerForwardRawSignsAliases`
- `TestDivisionPublicationWarmStorage`
- `TestDivisionPublicationZeroDivisorOutputAliases`
- `TestDivisionPublicationInverseCopyFailureBeforeDebugAssertion`
- `TestDivisionPublicationInverseTwoMinNegActions`
- `TestRationalSourceDivAndInverseAliases`
- `TestRationalCheckedInitializationAndZero`
- `TestRationalIndependentSignedArithmetic`
- `TestRationalNormalizationAndIntegerBranches`
- `TestRationalArithmeticFailureCleanup`
- `TestRationalWarmStorageAndScratchModes`
- `TestRationalColdInverseFailureCleanup`
- `TestRationalOpsSourceTst9All24Assertions`
- `TestRationalOpsMixedFiniteOraclesAndAliases`
- `TestRationalOpsFusedAllOutputAliases`
- `TestRationalOpsIntegerWrappersOracleAndSourceAliasQuirks`
- `TestRationalOpsIntegerProductsRatDivQuirksAndGuards`
- `TestRationalOpsSourceComponentAliasActionOrder`
- `TestRationalOpsLargeLimbScratchBranchesAndWarm`
- `TestRationalOpsAllPredictableFailurePrefixes`
- `TestRationalFacadeSourceTST1`
- `TestRationalFacadeSourceTST2`
- `TestRationalFacadeIndependentArithmeticAndAliases`
- `TestRationalFacadeScalarOverloadsPostfixAndExtraction`
- `TestRationalFacadeFusedSourceOperandRoutes`
- `TestRationalFacadeModHatNumBitsAndPowerDomains`
- `TestRationalFacadeAllocationFailureOwnership`
- `TestRationalFacadeProvenanceAndNoncanonicalSourceShapes`
- `TestRationalFacadeWarmZeroAllocations`
- `TestRationalFacadeInspectionForwardingAndSourceOutputAliases`
- `TestRationalFacadeCompoundFailureAliases`
- `TestScopedNumeralScalarFailureAndAllThinOperations`


## Cache-only warmed workload measurement

Separate `.cache/perf/division-final`, based on45f5d4b plus the same seven
frozen Go and one cache-only benchmark, measured the actual chain
Div(a,b,out)→Inv(a,out)→DivInt(a,b.num,out)→AccDiv(out,b.num)→RatDiv(a.num,b.num,out).
Inputs were18446744073709551617/4294967297 and4294967297/65537; both input
owners remain stable and the same output is reused. Cold construction uses the
actual Rational/Integer managers, shared1MiB Budget, two-cell slab pages,
16-word backing pages and two-descriptor pages. Actual parser setup and100
full warm chains precede timing. Final result is checked after StopTimer.
Manager Close and exact full budget refund are outside the measured interval.
No hidden intermediate text/host oracle or allocator callback is measured.

AMD Ryzen Threadripper3960X, linux/amd64, three200ms repetitions per policy:

| Profile/policy | ns per five-operation chain | B/op | allocs/op | warmed actual graph |
| --- | --- | --- | --- | --- |
| Release retained GCD scratch | 35791–37300 | 0 | 0 | 11cells,15Words refs,5344charged bytes |
| Release SYNCH local GCD scratch | 33617–36614 | 0 | 0 | 11cells,15Words refs,5344charged bytes |
| Checked/debug retained | 33516–34987 | 0 | 0 | 11cells,15Words refs,5448charged bytes |
| Checked/debug SYNCH local | 32828–35831 | 0 | 0 | 11cells,15Words refs,5448charged bytes |

Native13591 and checked/debug15902 benchmark sessions exited0; logs
`.cache/division-perf.log` / `division-perf-debug.log`. These are measured Go
workload timings, not source-C++ comparisons, source allocation-topology
claims or a controlled before/after speedup. Persistent storage is actual
charged manager/page/payload storage; no phantom source stack charge is used.

Remaining source boundaries: staged additive/mixed-multiplicative/setter/round
families and inherited selected GCD branches/diagnostics, integer infallible
setter/header ordering and payload clearing, true allocator/TLS/global-memory
composition, invalid-denominator owning value-result bridge policy, GMP/64-bit
profiles, TRACE/persistent stream effects and real solver/tactic/proof consumer
closure. Integer Div/Mod and paired division are already direct and must not
again be listed as staged debts.


## Parent terminal integrated verification

Parent snapshot `.cache/review/mpq-division-parent-45f5d4b` overlays exactly ten Go files and four documentation/ledger files, excluding concurrent prime/factor/codec/backend/SAT drafts. Full native73490 and checked/debug11037 passed (root3.284s/3.869s; mpnum22.131s/22.943s). Checked/debug race7937 and actual JS76986 each executed the identical44 RUN/PASS set: all43 mpnum names above plus TestSexprBVSourceWidthQuirks. Vet14530, full WASI build5505, checked/debug root70896 and mpnum82508 test compilation passed. Source/evidence/no-generics audit50088 exited0 with33ported/145partial/1924unported. Original parent native93250 reproduced the fractional BV expectation RED before the narrow profile fixture fix; its result is superseded by73490. Ten Go hashes remained unchanged through final gates. Documentation-only terminal evidence was appended after gates; source/Go bytes were unchanged. No whole source unit is promoted.
