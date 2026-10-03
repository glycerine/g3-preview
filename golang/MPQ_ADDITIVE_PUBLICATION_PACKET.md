# MPQ additive and mixed-product publication packet

Implemented and frozen, based on committed `facaeb4`. The selected public
methods now write caller fields in source order. Eight Go files are frozen;
Git, source metadata and integration commits remain parent-owned. No Go
parameterized types/functions, unsafe operations, cgo or production
`math/big` were introduced.

## Selected source and implementation

Read complete `src/util/mpq.h:76–155` and `271–442`. The selected source methods
are both `rat_add` overloads, `rat_sub`, public rational Add/Sub, mixed Add,
both integer-product rational-result Mul overloads, Inc/Dec forwarding, and
all four rational/mixed AddMul/SubMul overloads. SYNCH selects actual local
numerals; the other profile retains the existing manager caches.

Production edits are restricted to:

* `mpnum/mpq.go`: Add/Sub's shared addSub implementation.
* `mpnum/mpq_ops.go`: AddInt, MulInt and MulInts.

Add/Sub's integer fastpath writes caller.num with the existing Integer
Add/Sub, then destroys caller.den and publishes inline denominator1. General
Add/Sub computes tmp1=a.num*b.den and tmp2=b.num*a.den, writes caller.den
with a.den*b.den, writes caller.num using tmp1 +/- tmp2, and normalizes once.
Ordinary SYNCH completion explicitly deletes tmp1 then tmp2, as the source
statements require. Deferred per-local cleanup remains only for interrupted
Go ownership; it does not claim C++ plain mpz locals are exception-safe.

AddInt uses the integer fastpath or computes b*a.den into the actual single
local/retained tmp, Copies a.den to caller.den, Adds a.num/tmp to caller.num,
then normalizes. MulInts writes caller.num through the actual Integer Mul,
then resets caller.den. MulInt selects whether b is integer-valued before
any aliased output mutation, writes caller.num through Integer Mul(a,b.num),
then either resets caller.den or Copies b.den and normalizes. There is no
new pair staging, source algorithm replay or private fused-only substitute.

The existing fused/fusedInt implementations are unchanged: select the actual
b=+1/-1 shortcuts before mutation, or call public Mul/MulInt on the source
local/retained rational temporary, then public Add/Sub exactly once. Their
local deferred destruction already occurs after Add/Sub returns. Closing their
public dependencies corrects their caller publication without duplicating
arithmetic. Raw Inc/Dec still call AddInt with integer +1/-1; service prefix
operators still construct source rational-one and use rational Add. The two
routes are deliberately preserved.

Public Rational Copy/Mul, normalize, Integer Add/Sub/Mul/Div/Mod/GCD, Natural
algorithms, service implementations, allocator and coefficient store are
unchanged. All existing method signatures and cold owner layouts remain
unchanged. There is no selected raw sub(mpq,mpz,mpq) overload, so no invented
SubInt or fabricated missing-API RED was added.

## Tests-first evidence and scope

NEW `mpnum/mp_add_publication_test.go` contains nine groups:

1. TestAdditivePublicationBoundedModelsAndAliases: 504 independent signed
   exact operation/mode/output-alias cases through all nine public routes.
2. TestAdditivePublicationCallerFailurePrefixes: actual reservation sweeps,
   complete immediate named cells/Words audits, exact old/raw/normalized
   component candidates and owned failedRef, unchanged inputs, finite success
   detection and complete Close refund.
3. TestAdditivePublicationFusedUnitDispatch: 64 actual +/-1 dispatch cases with
   every rational output alias and both policies.
4. TestAdditivePublicationIntegerFastpathOldDenominator: real cold integer
   action refusal with nontrivial old BIG denominator; denominator destruction
   happens only after the integer action succeeds.
5. TestAdditivePublicationRawDenominatorsAndIdentity: unreduced, negative
   nonzero denominator states and all three operands referring to one value.
   Raw mixed methods use the original scalar numerator, not oracle-normalized
   numerator. Exact sign-preserving fields are distinct from rational math.
6. TestAdditivePublicationMixedScalarComponentAliases: scalar borrows of both
   caller components, caller equal/different from the rational input, and all
   three mixed overloads.
7. TestAdditivePublicationSynchronizedTemporaryDeleteOrder: literal existing
   Integer-operation source transcript; compare the entire concrete recycled
   cell chain after actual tmp1 then tmp2 Del. This checks observable ordinary
   lifetime order, not a replacement rational implementation.
8. TestAdditivePublicationWarmPlateau: all nine methods, both policies,
   100 warm chains followed by 100 allocation measurements; zero host heap
   allocations and unchanged actual Budget bytes/live cells/live Words.
9. TestAdditivePublicationServiceValueAndRationalOneRoutes: actual source
   value copies, scalar construction, postfix old result and rational-one
   prefix route; immediate complete ownership and independence after mutation.

Initial missing source publication was observed on isolated `70067bd` plus
only the new test file. Native73317 and checked/debug 56028 failed the caller
publication requirement for the nine methods in both policies; independent
bounded math/alias and unit-dispatch groups passed. Logs:
`.cache/additive-behavior-red.log` and `additive-behavior-debug-red.log`.
Expanded integer fastpath native 93802 also observed staging RED while raw
signed/identity and scalar component alias groups passed, log
`.cache/additive-expanded-red.log`.

Initial post-translation runs showed that a free concrete cell could satisfy
Integer publishTmpAt's missing-scratch replacement without a reservation.
That is a valid source/current-backend path, not a production defect. The
cold failure fixtures now fill only actual spare slots with named owned
numerals, using Live+Retired < Slots (retired metadata is not a live numeral).
They audit those roots too and do not alter the arithmetic algorithm. Revised
baseline 20313 is actual RED; direct 1932 is GREEN, logs
`.cache/additive-cold-red.log` and `additive-cold-green.log`. All general routes
and integer fastpaths now demonstrate real publication before a later cold
refusal. The synchronized deletion transcript and warm checks were added as
supplemental lifecycle verification after the first source translation;
no fictitious first-RED claim is made for these checks.

A corrected test compilation mistake (Close has no return value), and a
corrected oracle parsing mistake (big.Rat.SetString does not accept signed
denominators; SetFrac does), are not counted as behavioral or API RED.

## Approved existing-test corrections

Full mpnum tests on `965949a` plus the selected production and new tests
observed six dependent groups RED, native 50633/debug 46072. Logs are
`.cache/additive-dependent-native.log` and `additive-dependent-debug.log`.
The parent reviewed source routes before approving these five test files:

* `mpq_test.go`, ArithmeticFailureCleanup: only Add/Sub ops 0/1 now enumerate
  original components, exact cross sum/difference and denominator product,
  sign-preserving normalized components and actual failedRef. Unchanged
  non-destination operands and all existing Copy/Mul/Div/Inv expectations
  remain checked.
* `mpq_ops_test.go`, AllPredictableFailurePrefixes: only AddInt/MulInt/
  AddMul/SubMulInt/MulInts ops 0/1/4/5/6 changed to actual component prefixes.
  Direct division ops 2/3/7, integer forwards and successful math remain.
* `mpq_inspect_test.go`, EveryFailurePrefix: only raw Inc/Dec ops 7/8 may publish
  old numerator +/- old denominator, then existing normalization. Other
  extraction/setter assertions, untouched output/scalar input and ownership
  audits remain. The independent limb construction supplies original values
  without dereferencing a subsequently freed caller pointer.
* `inf_int_rational_test.go`, PostFailureAndGetterIndependence: failed postfix
  first component may have completed Add/Sub prefix; second component stays
  -4, returned owner remains zero, and real cleanup consumes the remaining
  owner. Successful math, getter independence and overflow checks remain.
* `rational_service_ops_test.go`, CompoundFailureAliases: only AddAssign/
  SubAssign/AddMulAssign/SubMulAssign ops 0/1/5/6 changed. Fused products use
  the actual normalized copy-then-Mul temporary and operand order. Existing
  MulAssign/DivAssign/other source failure expectations remain.
* The same file's AllocationFailureOwnership: only actual observed
  PostInc/AddMulAssign/SubMulAssign ops 14/15/16 changed. op 15 and op 16 were
  separately observed RED using cache-only extracted loop ranges (75830 and
  41473; `.cache/additive-fused-dependent-red.log` and
  `additive-subfused-dependent-red.log`) before their separate approval.
  The group's full ownership audit now precedes fault disabling; unchanged
  operands and returned owner 0 remain checked. Other operations are unchanged.

All immediate audits run before rendering, Del, Close or disabling fault
injection. The common parent-owned qops ownership helper is unchanged. Exact
old/raw/reduced component candidates retain the source denominator sign:
normalization divides by positive GCD and does not independently repair a raw
negative denominator. No arithmetic retry, missing-scratch repair, blanket
arbitrary mutation allowance, forged root or silent source-count increase was
introduced. Consumed failure owners use real cleanup and fresh services for
independent later cases.

## Frozen files and verified gates

Final tree `.cache/release/mpq-additive-final-facaeb4` is committed `facaeb4`
plus only these eight Go files. Snapshot and worktree hashes are identical:

| File | SHA256 |
| --- | --- |
| mpnum/mpq.go | b9eb7be7f2cf7d815172df34ca9668a79df9291e878084e63065c466684789dd |
| mpnum/mpq_ops.go | 526050baebb153069796127a938f5a7913627f6b0fc6a6cdee78f15e12251fee |
| mpnum/mp_add_publication_test.go | b39a35927c5295ced8be502ed775188ebd70d7370653ae13fc6148fd6ea0a142 |
| mpnum/mpq_test.go | 2a745bed6ddd1f397efbfe0d8b9e00f0bd4775e7f23c91ed655c832460ebe7ce |
| mpnum/mpq_ops_test.go | b50fae0bd0f836b414190c19d1fa108a2b63f4b7cfbc4e4ee59515e161f94bce |
| mpnum/mpq_inspect_test.go | 4c27c1d36093bf3ab8b0aeba68233610e5f2a83d1d5d3dc15de5205f86e01c97 |
| mpnum/inf_int_rational_test.go | def481c3ad0da4c2a1c7ab8dbc52399d807839e92bdf534931ce8f18c340f176 |
| mpnum/rational_service_ops_test.go | f3b222db2fe3999608a648fc3698c868b4d981fcf6a77ac4f7fe36ee6d5864e1 |

Cache inventories: `.cache/additive-final-hashes.json` and
`additive-final-tests.json` (nine new names plus six modified groups).
Selected native 76106 and checked/debug 75933 passed (2.430s/2.529s).
Final immutable-tree gates:

* Full native ./...34932: terminal PASS, mpnum 23.721s.
* Full g3checked,g3debug ./...96736: terminal PASS, mpnum 24.106s.
* Exact fifteen named g3checked,g3debug race 81536: terminal PASS30.679s,
  verified 15 RUN and 15 PASS names against the independent inventory.
* Actual JS/WASM 23747 with matching Node wrapper and checked/debug tags:
  terminal PASS9.102s, same exact 15 RUN/PASS set verified.
* Full checked/debug vet42044: terminal PASS.
* Full WASI build50310 and mpnum checked/debug WASI test compilation58550:
  terminal PASS; output remains under golang/.cache.
* Source/no-generics/evidence audit39066: terminal PASS on the frozen tree;
  the preserved source is read-only and the ledger remains parent-owned.

Logs use `.cache/additive-final-{native,debug,race,js,vet,wasi,wasi-mpnum,audit}.log`.
Other agents' codec, backend, source-buffer and API drafts are excluded.

## Accurate source credit and remaining boundaries

`src/test/mpq.cpp` tst1's nine integer-add ENSURE assertions exercise inherited
MPZ setup; they do not establish rational Add/Sub allocation order. The actual
registered rational.cpp tst1/tst2 121 facade assertions already had faithful
operator routes and are rerun by the full suites. New bounded/fault/lifetime
cases are independent additions. No extra source ENSURE credit or whole-source
unit completion is claimed.

The selected additive/mixed/fused algorithms and their public caller component
order are now translated. Existing arithmetic has actual one-pass source
algorithms; no historical transactional claims are retained for Integer
Div/Mod, Rational Copy/Mul/Div/Inv, Power/shift or bitwise. Underlying source
GCD diagnostics/action debts, remaining Integer diagnostics and scalar
representation adaptations, physical source allocator topology/global memory
integration and exceptional lifetime equivalence remain separate work. The
existing normalize implementation is reused once; this packet does not claim
all inherited GCD failure paths reproduce source allocator exceptions.

Ordinary input guards still reject zero-denominator rational values. Explicit
cleanup/Reset accepts actual owned invalid raw outputs from the division
family; service returned-value copying of such outputs remains an explicit
owning boundary. This additive packet does not invent unchecked transfers.
TRACE, persistent C++ stream state, other numerical backends, complete AST/
polynomial/solver consumers and the full solver remain open. No existing
service or shared coefficient-store production was changed by this packet.

## Parent integration verification

Parent-reviewed integration is based on committed coefficient-store adoption
45a32d7, with exactly eight immutable Go files, this packet and two source
evidence files overlaid. Full native/debug ./... and checked/debug vet passed.
Exact54 arithmetic regression groups passed race and actual JavaScript/WASM;
both complete RUN/PASS sets match additive-broad-54-tests.json. Full WASI
build and root/mpnum checked/debug test compilation passed; WASI execution
remains unverified. Source/evidence/no-generics audit passed: 2102 units,
677 APIs, 33 ported, 148 partial, 1921 unported. No whole-unit promotion.
Incomplete GCD, factor and allocator drafts were excluded.

Logs: .cache/additive-integration-{native,debug,race,js,vet,wasi,audit}.log
and additive-integration-wasi-{root,mpnum}.log. Parent verified frozen Go
hashes before and after the gates; terminal packet/progress updates change
documentation only.
