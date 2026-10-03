# Selected integer/rational bitwise source publication — final packet

Status: implemented and frozen based on committed `01492fe` (Power packet
`96e3058`). The five approved Go files are immutable; parent owns integration,
metadata and Git. Full selected internal32 OR/AND/XOR/NOT bodies and rational
forwards retain the source arithmetic routes, completed caller publications,
source debug assertions and explicit finite-domain adaptations below. Service
production is unchanged. Whole source units remain partial.

All work, caches and artifacts remain under `golang`; no Go generics, unsafe,
cgo, production math/big or alternate bitwise arithmetic was introduced.
The design/test inventory below records the approved scope; final observed
results and exact frozen inventory follow at the end.

## Audit corrections to the older remaining-publication document

`MPZ_REMAINING_PUBLICATION_PACKET.md` is a historical audit, not current method
status. Its bitwise staging and universal negative/integrality findings were real
on baseline01492fe and are addressed by this selected chunk. Its Power/shift entry-rejection/staging and public Rational Copy/Mul
staging findings were reconciled in the committed Power packet; do not repeat
them as missing. Shared normalization already exists and remains the same
algorithm. Current Integer Del preserves m_val while clearing owned cell;
Reset additionally sets0. No change to that correct committed Del is needed.

The complete source bodies were read at `mpz.cpp:1018–1185`, declarations in
`mpz.h:654–660`, actual Del/getters in `mpz.h:340–347,230–243` and
`mpz.cpp:1262–1342`, rational forwards `mpq.h:632–662`, and all four
`rational.h:370–392` value-return wrappers. Existing Go bitwise production and
all associated test bodies were read, together with their shared Copy/Mul,
Mod/Div/getter and service ownership routes.

## Complete selected methods and source actions

| Method | Actual complete internal32-bit body |
| --- | --- |
| Integer OR (`mpz.cpp:1018–1066`) | Debug SASSERT nonnegative a then b. Small/small: Del caller then inline signed32 OR. Otherwise copy local a1, copy b1, Set multiplier1, Reset caller0; while both nonzero: Mod each by retained two64, GetUint64 both, Set tmp OR, Mul tmp by multiplier, Add directly into caller, Mul multiplier by two64, Div a1 then b1 by two64. After loop, multiply and Add each nonzero remaining tail in source order. Del a1/b1/a2/b2/m/tmp in that order. |
| Integer AND (`1068–1104`) | Same debug assertions, direct small Del+AND and six locals/loop. No remaining-tail actions. |
| Integer XOR (`1106–1149`) | Same debug assertions; small path uses source signed64 XOR then SetInt64. Same locals/loop and ordered OR-style remaining tails. |
| Integer NOT (`1151–1185`) | Debug nonnegative assertion. Small and width<=63: signed64 mask and SetInt64. Otherwise Copy a1, Set multiplier1, Set caller0; while width>0: Mod/GetUint64, complement, source inverse assertion, optional low-width mask, Set tmp, actual GetUint64(tmp)==v assertion, Mul tmp/m, Add caller, Mul multiplier/two64, Div a1/two64, decrement width by min(width,64). Del a1/a2/m/tmp. Retain final unused multiplier AND division actions. |
| Rational OR/AND/XOR (`mpq.h:634–653`) | Debug-only SASSERT(is_int(a)&&is_int(b)), then actual integer method into caller numerator, then reset denominator only after successful integer action. |
| Rational NOT (`658–662`) | Debug-only integrality assertion, actual integer NOT into caller numerator, then denominator reset. |
| Rational inherited integer overloads (`632,640,648,656`) | Existing explicit `Integers()` composition; no duplicate algorithm or fictional rational shifts. |
| Service value-return OR/AND/XOR/NOT (`rational.h:370–392`) | Construct actual default rational result0/1, invoke raw manager into that result, return named result. Existing locked service paths already do this through one raw algorithm and owning transfer, with lexical cleanup on errors. No copy-before-compound route belongs to these four source wrappers. |

Do not replace the 64-bit Mod/Div/Mul/Add algorithm with direct limb Boolean
loops, a host big integer, bit scanning, or a second replay. Source small
OR/AND publish via Del+scalar assignment. Cached scalar reads may preserve
ordinary output aliases, but do not invent a resetting Del. Input local copies
finish before output reset, so copy refusal leaves caller unchanged; later
failures preserve actual reset, completed chunk additions or setter failedRef.
Tail publication differs from the common loop and must remain ordered.

## Debug/release domain and nontermination contract

The source negative checks are SASSERT, not universal release guards. Preserve:

* Release small/small negative OR/AND/XOR is finite signed arithmetic, including
  INT_MIN and output aliases. Debug refuses at its source assertion before any
  numeric action. Existing ErrBitwiseDomain can remain the debug assertion
  category; it must no longer reject all finite release cases.
* Mixed-sign big binary operands have finite source paths: the nonnegative
  quotient eventually reaches0, the loop ends, and OR/XOR execute their actual
  negative tail while AND omits tails. Zero with a big negative can take the
  no-loop tail/no-tail path. Compare finite results to independent test-only
  two's-complement big.Int expectations, and freeze any source-specific
  departure separately instead of changing the algorithm.
* Two negative operands in the big branch eventually reach a1=b1=-1; source
  while remains true forever. This is distinct from the finite small/small
  negative fast path. Parent source review accepts a typed nontermination
  boundary at the FIRST actual continuation where BOTH quotients are inline
  -1, after all preceding copy/reset/mod/set/multiply/add/div actions.
  Proposed `ErrNonterminatingBitwise` and concrete
  `BitwiseNonterminationError{Operation BitwiseOperation}` distinguish this
  from source debug assertions. The operation enum identifies OR/AND/XOR.
  Earlier real allocation errors take precedence. This stops an infinite
  domain whose multiplier/output can still grow; it does NOT claim equality
  to a future source budget-refusal prefix. No entry sign rejection or arbitrary
  iteration cap substitutes for reaching that actual boundary.
* NOT is finite by its uint32 width even for negative release operands.
  Negative input alone must not trigger the binary-loop nontermination guard.
  Width0,63,64,65,127,128,129 and larger source chunk boundaries remain actual
  paths. Huge widths use real finite allocation refusal in tests, not a cap.
* Source small NOT width63 signed shift/subtraction is undefined signed
  arithmetic in portable C++. Retain the existing explicitly documented
  unsigned low63-bit mask adapter rather than pretending it is a new source
  assertion or universally portable native observation.
* Rational release fractional/negative-denominator/unreduced inputs invoke
  bitwise operations on NUMERATORS and reset output denominator on success.
  Do not apply mathematical rational bitwise semantics or silently normalize
  input. Debug integrality refusal happens before integer negative checks.
  Existing initialized-numeral/provenance safety checks remain API boundaries.

Add NOT's two actual diagnostic expressions at their source sites, under
g3debug: ~v==n BEFORE masking, and actual manager Uint64(tmp)==v AFTER the
successful SetUint64 and BEFORE multiplication. Existing ErrIntegerAssertion
can report these source invariant failures. Do not fabricate behavioral RED
for tautologies or inject fake arithmetic callbacks to make them fail.
g3checked handle validation is independent of g3debug.

## Approved design history: proposed production and test ownership

Arithmetic signatures remain unchanged. Only the nontermination error/operation
types are new public control metadata; no new numerical owner is required.

| File | Proposed scope |
| --- | --- |
| Existing `mpnum/mpz_bitwise.go` | Complete OR/AND/XOR/NOT direct caller publication; source small paths and debug assertions; real -1 continuation guard; NOT diagnostics; remove extra work/Move temporary while preserving all six/four actual locals. |
| Existing `mpnum/mpq_bitwise.go` | Restrict integrality checks to actual debug assertions while preserving owner checks and numerator-before-denominator actions. Do NOT change global integerOperands, which other unrelated families still use. |
| Existing `mpnum/rational_service_extra.go` | Read-only by default: current value-result/owning-transfer/cleanup routes are already appropriate. Change only if a concrete independently observed lifetime defect requires parent review; no LCM/root/projection/formatting edits. |
| New `mpnum/mp_bitwise_publication_test.go` | New namespaced complete source-action, profile, alias, finite model, fault-prefix, raw rational and service tests. |
| Existing `mpnum/mpz_bitwise_test.go` | Preserve source bug4 and all successful models. Only observed obsolete universal-negative/error-publication expectations may be updated after actual RED and parent approval. |
| Existing `mpnum/mpq_bitwise_test.go` | After actual RED, change only bitwise debug/release domain branches and failure-output expectations; numerator can publish before error, denominator retains its old value until integer success. Immediate graph must precede fault disable. |
| Existing `mpnum/rational_service_extra_test.go` | Existing bitwise-only portions of NumberTheoryAndBitwise, FailurePrefixesAndOwnership, FormattingPublicationAndProvenance and WarmReuse are validation targets; only demonstrated changed assertions may be edited. Preserve all other sourceTST8/root/number-theory/formatting/bridge/projection checks. |
| New proposal/status document | This file; no old status, metadata, manifests or Git modifications. |

No existing `mpz.go`, division, normalization, raw allocator or global registry
edits are requested in this packet. Report any genuine dependency failure before
changing its file or weakening a shared root audit.

## Registered source tests and TDD inventory

`src/test/mpz.cpp::bug4` is actually called by `tst_mpz`. It contributes exactly
ONE ENSURE: compare OR result with the source copy-then-aliased OR result.
Existing TestIntegerSourceBitwiseBug4 ports that route and an added exact
`4294967295 4294967295\n` print expectation. Preserve it. Search of the actual
MPQ/rational registered test files found no additional direct bitwise ENSURE;
the service TST8 root assertions are separate and must not be credited to this
packet. Independent/random/fault loop counts are not source assertion counts.

Existing complete test groups to retain/run:

* IntegerSourceBitwiseBug4, IntegerBitwiseIndependentOracleAndAliases,
  IntegerBitwiseFailurePublicationAndWarm, IntegerBitwiseSourcePreconditions,
  IntegerBitwiseAliasedFailurePublication,
  IntegerBitwiseSmallPathsReleaseBigOutput.
* RationalBitwiseForwardingIndependentAndAliases,
  RationalBitwiseFailureOwnershipAndDenominatorPublication.
* Service extra NumberTheoryAndBitwise, FailurePrefixesAndOwnership,
  FormattingPublicationAndProvenance and WarmReuse, plus unchanged TST8 and
  other families in the full module suite.

Approved tests-first design history (actual executed evidence is below):

1. Public caller-reset/partial-chunk behavioral RED for all four integer
   methods: real reservation sweeps must find an error after source output0 or
   a completed low chunk. Current staged work instead retains its sentinel.
   Include out==a, out==b, a==b/out, and operand-copy refusal BEFORE Reset.
   Derive finite expected prefix candidates from actual 64-bit source actions
   and independent input arithmetic, not a second production algorithm.
2. Release negative small paths and mixed-sign finite big paths, with g3debug
   unchanged caller at actual assertion. Include zero/tails and operand order.
   Current universal guard supplies real RED for defined release cases.
3. Both-negative big continuation: exact completed low-chunk prefix at first
   two -1 quotients, typed operation-specific error and immediate complete
   ownership. Arm real earlier allocation refusal on a fresh owner to prove
   error precedence, without claiming an infinite arithmetic result.
4. NOT finite negative width profiles, big-input width0 copy-before-zero,
   low-width masking of larger input, width63 explicit adapter, and retained
   final unused multiplier/division errors AFTER mathematically complete
   output. Source diagnostics are actual runtime expressions; no invented
   assertion-count increase or test hook.
5. Raw rational fractional/negative/unreduced release wrappers: source
   numerator action and successful den1; debug integrality before negative
   assertion; error leaves EXACT old denominator even if numerator changed
   or became failedRef. Both SYNCH settings and ordinary output aliases.
6. Service all four real value-return paths: exact finite results, provenance
   before numeric actions, fresh local0/1, empty owning return on error and
   lexical cleanup before immediate caller/manager/cache graph audit. Source
   inputs remain unchanged because these source wrappers return a new value.
7. Full fault sweeps with explicit completion markers, all ordinary aliases,
   public failedRef cleanup, permanent retired metadata distinct from live
   cells, guarded missing scratch and owner Close/refund. Audit cells AND
   Words before String, fault disable, Del or Close. Never repair/retry invalid
   numerical state; independent scenarios use fresh complete owners.
8. Existing warm zero-allocation/source arithmetic checks and bounded exact
   models stay intact. If physical storage is changed beyond removing extra
   staged work, obtain parent storage review and measured timing/peak evidence.

Existing tests first become RED only after the actual new source routes are
implemented. Inventory the precise changed expectations before editing them;
do not preemptively weaken successful models or unrelated wrapper guarantees.
Final freeze: exact owned file hashes and named inventory; isolated current
approved HEAD plus only approved files; full native/g3checked+g3debug, named
race and actual JS/Wasm RUN/PASS, vet, full WASI plus test compilation,
integrated source fingerprint/evidence/no-generics audit. Parent owns commits
and central unit/method coverage; whole headers stay partial.

## Ownership, physical allocation and dependencies left open

Binary general paths own six source Int locals; NOT owns four. They are raw
mpz locals in C++, whose source body explicitly deletes them only on normal
completion. Existing Go lexical cleanup on typed errors is an explicit safer
exception-lifetime adapter. It must free valid/owned-failed local components
without losing live cells or suppressing original arithmetic errors. Public
output and retained manager scratch are separate known roots. Service result
locals additionally have the actual owning rational destructor contract.

Use the existing concrete numericCellSlab/Words/NaturalManager and actual
Budget graph. This source publication packet neither claims source malloc
topology nor emits synthetic raw-allocator events for slab records. Source
cell is one small-object allocation; Go metadata and payload are separate
charged backing owners with possible old/new overlap. The real committed
root-package raw and small-object allocator owners are not currently composed
into mpnum, which cannot import parent g3 cyclically. Parent must coordinate a
concrete backend/accounting boundary before any such integration; no fake
callbacks or Budget.Used-as-global-memory substitution is proposed.

Actual bitwise calls use the existing single Copy, SetUint64, Add, Mul, Mod,
Div and Uint64 methods. Copy/Mul caller publication is now reconciled.
Integer Mod/Div and quotRemCore have been direct since fdf6f3c: current
mpz_div.go89–183 follows mpz.h466–496/mpz.cpp474–507, including actual
overwritten-divisor rereads and quotient-before-remainder publication. Remaining
boundaries include omitted generic getter/arithmetic diagnostics, source scalar
infallible field ordering and Words tail clearing. Do not
silently bypass these public algorithms with private bitwise duplicates.
Selected chunk residues/SetUint64 values fit the actual Uint64 accessor;
this packet does not claim a complete unrelated raw getter-domain rewrite.

GMP and 64-bit-limb conditional branches, TRACE/custom persistent ostream,
full source allocator/TLS profile, unrelated bitwise consumers and full solver
integration remain open. The accepted release negative continuation guard is
a typed source-invalid-domain termination adapter, not a source mathematical
replacement or an assertion that future finite allocation refusal is identical.

## Final implementation and observed TDD evidence

Only `mpz_bitwise.go` and `mpq_bitwise.go` production changed. Integer general
paths removed the extra staged `work` numeral, Reset/Set the actual caller
at the source point, Add every actual chunk/tail directly to it and retain the
final multiplier and Div actions. Small OR/AND use actual Del plus scalar
publication; XOR retains SetInt64. Source negative and rational-integrality
assertions are build-selected by g3debug. Both NOT internal invariants use
existing ErrIntegerAssertion and execute the real accessor where source does.
No artificial hook or behavioral RED is claimed for tautological `~v==n`.

Binary nontermination is reported only at the first next continuation where
both real local quotients are inline -1. Completed output chunks and prior
allocation errors are preserved. This guard ends an infinite source domain;
it does not assert equivalence to a possible later budget exhaustion as the
source multiplier continues growing. Small/small negative binary operations,
mixed-sign finite binary paths and finite-width negative NOT remain executable
in release. The width63 unsigned-mask adapter avoids source signed shift/
subtraction UB while defining intended low63 bits. Owning cleanup on typed
errors remains the documented Go exception-lifetime adapter.

Initial isolated HEAD01492fe behavioral RED (`.cache/bitwise-behavior-red.log`)
showed three real differences: finite negative-inline rejection, entirely
staged caller failure publication, and release fractional rational rejection.
Separate missing-API RED (`.cache/bitwise-api-red.log`) observed absent
BitwiseOperation, BitwiseNonterminationError and ErrNonterminatingBitwise.
First test build also exposed a fixture misuse of void Close, corrected before
behavioral evidence. No source defect was inferred from that compile error.

After approved production changes, `.cache/bitwise-dependent-native.log`
observed exactly the obsolete expectations in IntegerBitwiseSourcePreconditions,
RationalBitwiseForwardingIndependentAndAliases and
RationalBitwiseFailureOwnershipAndDenominatorPublication. Parent approved
only those narrow corrections. Every positive independent oracle, original
bug4 route, aliases, warm allocation and cleanup check remains. Release cases
now assert exact finite results; debug asserts unchanged output and source
error. Rational failure sweeps audit named cells/Words before fault disable,
then check identical denominator scalar and a source-derived numerator prefix
or the owner's exact failedRef.

An initially overstrong new test required every alias to numerically change
on a failure. AND with output aliasing b has the same mathematical result as
b in this fixture: reservation failures can precede Reset or follow publication
of that identical value. Retained cell/header equality also cannot observe
payload writes. The corrected test requires changed failure publication for
each distinct-output operation, successful completion for every alias, and
exact source prefix values for every failing alias. Binary prefix models stop
after the shorter operand's actual 64-bit loop checkpoints, then permit one
complete OR/XOR tail; they do not fabricate intermediate high chunks. NOT
uses its actual width-limited checkpoints. Immediate root audits always precede
formatting/cleanup. Big NOT width0 copy refusal required exhausting actual
cold cell slots to force a real Copy allocation; no fake allocation callback
was added. This source ordering already existed and is confirmation, not an
invented production defect/RED. Existing positive tests retain widths0,63,64,
65,127,128 and other boundaries; added tests cover INT_MIN and both-negative
small/small output aliases.

Source assertion count remains **one actual registered ENSURE** from
src/test/mpz.cpp::bug4. Its Copy/OR routes are unchanged. The twelve new test
groups are added action/profile/oracle checks, not newly ported C++ ENSUREs.
No direct bitwise ENSURE was found in registered mpq/rational test units;
separate TST8's34 root assertions receive no bitwise credit. Service forwards
are checked through actual owning value-return routes, with empty returned
token and lexical cleanup on error, unchanged inputs and immediate full roots.

## Final frozen evidence

Isolated `.cache/release/bitwise-red` is HEAD01492fe plus only five approved Go
files; despite its initial name it contains the final frozen GREEN bytes.
Full native7926 and checked/debug24151 exited0 (mpnum22.032s/22.634s).
Named23 checked/debug race77582 exited0 (mpnum9.524s); actual checked/debug
JS74481 exited0 (mpnum2.536s). Independent log parsing confirms exactly23
RUN names and23 PASS names equal the frozen inventory in both logs.
Vet66046 and full checked/debug WASI build plus mpnum test compilation29870
exited0. Logs are `.cache/bitwise-full.log`, `bitwise-full-debug.log`, and
`bitwise-final-{race,js,vet,audit}.log`; WASI output is
`.cache/bitwise-mpnum.wasm`. The source/evidence/no-generics audit includes
the parent's current integrated manifest; its terminal result follows below.
No production/tests changed after these frozen hashes were recorded.

| Frozen Go file | SHA256 |
| --- | --- |
| `mpnum/mpz_bitwise.go` | `da2f014671636da28aa87a6a25022821f1bc6ff91093f67885658936b362e7d6` |
| `mpnum/mpq_bitwise.go` | `50ca62b5e3d7b3cbe75dde5e0ea6154d0862a1297a8984cc904f5e0951c42283` |
| `mpnum/mp_bitwise_publication_test.go` | `efd4faa02feefed40b9aec1083263dd26b710f3c2160f60be1b4a8fd61b6c223` |
| `mpnum/mpz_bitwise_test.go` | `2e51909126fbc8a6b5969c6f8d98bd9b6b697eb6778ad528819716f67d743b02` |
| `mpnum/mpq_bitwise_test.go` | `a9c025db949b79b0eb6fa169439a1257f21487f869dc3bdd7f9c5c0e9fdae53b` |

Exact23 named groups (twelve new, six integer, two rational, three existing
service groups; broader service groups retain their other source checks):

- `TestBitwisePublicationNegativeInlineProfile`
- `TestBitwisePublicationCallerPrefixes`
- `TestBitwisePublicationRationalRawProfile`
- `TestBitwisePublicationActualNonterminationBoundary`
- `TestBitwisePublicationFiniteMixedSignsAndWidthAliases`
- `TestBitwisePublicationFinalUnusedActions`
- `TestBitwisePublicationRationalDenominatorFailureAliases`
- `TestBitwisePublicationEarlyFailurePrecedesBoundary`
- `TestBitwisePublicationServiceValueRoutes`
- `TestBitwisePublicationServiceFailureRoots`
- `TestBitwisePublicationBigNotZeroWidthCopyOrder`
- `TestBitwisePublicationSignedSmallBoundaryAliases`
- `TestIntegerSourceBitwiseBug4`
- `TestIntegerBitwiseIndependentOracleAndAliases`
- `TestIntegerBitwiseFailurePublicationAndWarm`
- `TestIntegerBitwiseSourcePreconditions`
- `TestIntegerBitwiseAliasedFailurePublication`
- `TestIntegerBitwiseSmallPathsReleaseBigOutput`
- `TestRationalBitwiseForwardingIndependentAndAliases`
- `TestRationalBitwiseFailureOwnershipAndDenominatorPublication`
- `TestRationalServiceExtraNumberTheoryAndBitwise`
- `TestRationalServiceExtraFailurePrefixesAndOwnership`
- `TestRationalServiceExtraWarmReuse`


Remaining boundaries include other arithmetic/getter diagnostics, infallible
scalar/header field ordering,
Words clearing, concrete allocator/backend/global memory composition, GMP/
64-bit-limb profiles, TRACE, persistent ostream state and real solver consumers.
This packet adds no allocator/backend API or service production changes,
no source-unit promotion and no full solver claim.

Parent independently froze the five Go files plus integrated manifest/progress
in `.cache/review/bitwise-final-01492fe` and verified unchanged hashes. Full
native23226 exited0 (mpnum20.910s); checked/debug51702 exited0 (21.607s).
Exactly23RUN/23PASS were verified for checked/debug race60873 (9.661s) and
actual JS41996 (2.472s). Vet83398, full WASI build60840 and root/mpnum test
compiles75367/91810 exited0. Parent integrated source/evidence/no-generics audit20630 exited0:
2102units,677APIs,33ported145partial1924unported; log
`.cache/bitwise-parent-audit.log`. Parent independently verified all2102 unit
SHA values and the exact1687-file source fingerprint
`f8480997c14e2d6c414f40c60ce5095ef40a89ce6d1bcf1ed196c5d682d42d8f`.
No source-unit promotion occurred. Agent supplementary audit65989 also exited0 with the same counts; log
`.cache/bitwise-final-audit.log`.
