# Concrete infinitesimal integer rational

`src/util/inf_int_rational.h` and `.cpp` have complete concrete owner-local
method equivalents in `mpnum/inf_int_rational.go`. Owned files are that file,
`mpnum/inf_int_rational_test.go`, and this status. Parent owns existing
RationalService constant fields and hook integration; this packet does not edit
service/manager files, manifests or git metadata. Other numeral/specialization
units are outside this completion claim.

## Complete source mapping

All references below are to `src/util/inf_int_rational.h` unless specified.

| Source declarations/actions | Concrete equivalent |
| --- | --- |
| Fields `:30-34`; cpp static definitions | `OwnedInfIntRational{first Rational, second int32, ownerID uint64}`. Three constants belong to service fields `infIntZero`, `infIntOne`, `infIntMinusOne`, never mutable globals. |
| `init/finalize :36-37`; cpp `:42-60` | `initInfIntRationalConstants() error`, `destroyInfIntRationalConstants()`: first constants copied zero→one→minus-one; epsilon initialized zero. Partial/uninitialized constants have ownerID0 and are skipped by destructor unwind. Parent calls after InfRational initialization; destroys InfRational, InfInt, then power cache and rational constants/manager. |
| Hash/functor `:39-45` | `InfIntHash`: rational first hash then xor `(uint32(second)+1)` with defined unsigned wrap. `InfIntEqual` supplies concrete equality functor action; no per-record stored functions/interfaces. |
| `swap :47-50` | `InfIntSwap` swaps first via manager before scalar second, preserving same-owner source ordering. |
| `to_string :52`; cpp `:26-40`; stream `:374-378` | `InfIntString`: first rational display, scalar0 returns first, negative emits `(first -e*abs(second))`, positive `(first +e*second)`. Caller owns result string; stream composition maps to this exact String output. |
| Default/copy/int/signed-pair/rational/bool/scalar constructors `:54-86` | `InfIntNewValue`, `InfIntClone`, `InfIntFromInt`, `InfIntFromFraction`, `InfIntFromRational(r,0)`, `InfIntFromSignedInf(r,bool)`, `InfIntFromRational(r,epsilon)`. Signed pair preserves inclusive signed setters for both components, sign flips before setters, and explicit widened INT_MIN magnitude adapter inherited from service. |
| Destructor `:88` | `InfIntDel` destroys sole owned first component and invalidates caller token. Scalar second has no owned storage. |
| Reset `:93-96` | `InfIntReset` resets first, then publishes scalar0. |
| Int/int64/uint64/rational predicates `:98-110`; signed/unsigned getters `:112-120` | `InfIntIsInt`, `InfIntIsInt64`, `InfIntIsUint64`, `InfIntIsRational`, `InfIntInt64`, `InfIntUint64`. First predicate action precedes epsilon0 check. Source assertions become typed conversion errors. |
| First/rational/infinitesimal getters `:122-130` | `InfIntRational`, `InfIntFirst`, `InfIntInfinitesimal`: source const-reference first getters become independent owned copies so no raw mutable numerical borrow escapes. Infinitesimal is constructed from signed int32 second. |
| Assignments `:132-142` | `InfIntCopy`, `InfIntSetRational`: deep-copy first before copying/resetting second. `InfIntSetParts` and `InfIntSetInt` are explicit constructor/setter convenience adapters using the same ordering. |
| Numerator/denominator `:144-152` | `InfIntNumerator`, `InfIntDenominator`: require epsilon0 before actions; construct rational component then deep-copy into result InfInt, preserving source two copy actions and temporary destruction. |
| InfInt add/sub compound `:154-158,171-175` | `InfIntAddAssign`, `InfIntSubAssign`: numerical operation first, then checked signed epsilon action/publication. Self alias supported. |
| Rational scalar multiply `:160-167` | `InfIntMulAssign`: inclusive rational `IsInt32` validation before multiplication; rational first mutates; scalar is reread AFTER mutation, then epsilon product publishes. Source alias with first is exercised through private kernel under lock. |
| Mixed rational add/sub `:177-185` | `InfIntAddRationalAssign`, `InfIntSubRationalAssign`: first alone changes, epsilon remains. |
| Prefix/post inc/dec `:187-199`; rational wrapper `rational.h:220-233` | `InfIntInc`, `InfIntDec`, `InfIntPostInc`, `InfIntPostDec`: source rational Add/Sub with local mk_q(1) pair, not distinct manager mixed-int Inc/Dec. Post forms deep-copy old first before mutation; named return copy is elided/moved through Go ownership transfer. |
| Three EQ and LT overloads `:201-229`, three NE/GT/LE/GE overloads each `:308-354` | All18 methods: InfIntEqual/NotEqual/Less/Greater/LessEqual/GreaterEqual, same names suffix Rational, and InfIntRationalEqual/NotEqual/Less/Greater/LessEqual/GreaterEqual. LT executes first rational LT; only when false, separate rational EQ; only when equal, epsilon order. GT reverses operands; LE/GE/NE negate their exact source comparison route. |
| Neg/sign/zero/one predicates `:231-270` | `InfIntNeg`, `InfIntIsZero/One/MinusOne/Neg/Pos/NonNeg/NonPos`. First sign dominates; epsilon resolves first0. Neg mutates rational first before scalar negation. |
| Floor/ceil `:272-292` | `InfIntFloor`, `InfIntCeil`: exactinteger first corrected by epsilon sign only; noninteger first delegates manager floor/ceil. Integer correction preserves rational free-operator clone→compound→return deep-copy actions. |
| Constant getters `:294-304` | `InfIntZero`, `InfIntOne`, `InfIntMinusOne`: independent owned copies; service constants remain private. |
| Binary add/mul/sub `:356-366` | `InfIntAdd`, `InfIntMul`, `InfIntSub`: operand clone→compound lvalue→return deep copy→temporary destruction. No elision of the required return-lvalue copy. Invalid scalar multiplication domain is checked after initial operand copy, as source expression order dictates. |
| Unary neg/abs `:368-372,381-387` | `InfIntNegCopy`, `InfIntAbs`: deep-copy named result then mutate; named return moved/elided (explicit adapter) without an invented additional copy. Temporary first is destroyed on error. |

## Ownership, budgets, source failure actions

The hot record is pointer-free in checked and release layouts; all methods
borrow the existing service and take its mutex. There are no per-value Go heap
owners, raw slices, raw public numeral borrows, generics, unsafe or production
math/big. A raw zero token is invalid; ownerID identifies service provenance,
not a per-value ledger. Owning shallow Go copies remain forbidden, use Clone or
Copy. Foreign tokens and closed owners reject before mutation. Del invalidates
the caller token; service Close invalidates all outstanding tokens.

Caller token/string storage is external. Numerical first-component allocations
and old+new temporary peaks use existing manager/Budget graph accounting. Three
constant records are part of the reflect-charged parent service header, not
precharged again. Partial constructor hook failure destroys initialized fields
without touching uninitialized zero-denominator records. Successful Budget
reservations retain their normal fault-prefix accounting; no counter restore or
transaction bypass is introduced.

Predictable numeral failures inherit existing manager staged publication and
storage-prefix policy. If first numerical mutation succeeds but epsilon's signed
arithmetic is outside int32, the API returns `ErrInfIntOverflow` AFTER retaining
that first mutation, leaving old epsilon unchanged. This is an explicit typed
adapter for C++ signed UB, not an assertion of observed overflow behavior.
INT_MIN numerical negation follows that same first-prefix policy. Negative
textual INT_MIN would also negate a signed int in source; String first performs
rational formatting, then returns this typed overflow error. Valid finite signed
cases remain unrestricted. Scalar domain errors use `ErrInfIntScalar` before
first multiply; signed32 inclusive boundaries are accepted. In source scalar
alias use, a post-multiplication coefficient outside int32 becomes existing
checked `ErrConversion` after the rational prefix. Public deep getters avoid
introducing raw mutable aliases; the real private source alias kernel is tested.

## Tests-first and independent evidence

There are **no registered source inf_int_rational tests or active ENSUREs**.
Source algorithms are translated from the fully read header/cpp, and no invented
source test/assertion count is reported.

Initial tests preceded implementation; observed RED was missing concrete
OwnedInfIntRational/hooks/methods (concurrent other-agent RED also present).
An explicit source rational increment route fault test later observed behavioral
RED: old InfIntInc succeeded at prefix3 where exact manager rational Add failed.
Inc/Dec/post methods were corrected to the source rational pair route; both
32-prefix route sweeps now agree on errors, immediate Used charges/cells/Words,
values and epsilon. Explicit Clone copy-constructor fixture preceded its method
and observed missing-API RED. Remaining additional lifecycle checks against
existing algorithms passed without manufacturing a failure.

Ten named groups:

- `TestInfIntRationalSourceConstructorsConstantsLifetime`
- `TestInfIntRationalIndependentArithmeticAndAliases`
- `TestInfIntRationalOverflowSourcePrefixesAndScalarDomain`
- `TestInfIntRationalComparisonsSignsAndRounding`
- `TestInfIntRationalAccessorsHashDisplayPostOperations`
- `TestInfIntRationalConversionsAndMixedComparisons`
- `TestInfIntRationalFaultPrefixesTemporaryCleanup`
- `TestInfIntRationalAliasReReadWarmReuseAndGuards`
- `TestInfIntRationalIncrementSourceRationalRoute`
- `TestInfIntRationalPostFailureAndGetterIndependence`

Coverage:150 independently seeded exact rational/epsilon arithmetic cases;
225 first/epsilon comparison pairs plus all18 mixed declarations; floor/ceil and
sign grids; signed/unsigned64 bounds; INT_MAX signed fraction shape; scalar
INT_MIN/INT_MAX acceptance; noninteger/out-of-range scalar rejection;
add/sub/mul/neg signed-UB typed prefix effects; source scalar alias reread
(2+3e times ownfirst becomes4+12e);80 binary operator fault prefixes and80 post
prefixes; independent immediate live cells/Words checks accounting retained
NaturalManager scratch; failed Copy leaves rational and epsilon unchanged;
unary-result overflow cleans already copied first immediately; deep getter
and constant independence; foreign/destroyed/closed provenance; warm Copy/Swap
zero Go allocations under FailAfter(0); budget releases checked before and at
owner close. Parent owns separate actual complete hook/lifecycle/order tests.

## Source SHA256

- `src/util/inf_int_rational.h`: `6f2ce95052ff18cc2e6fd7103741c1959c6004a4207fa9b366492d4e94ba95c8`
- `src/util/inf_int_rational.cpp`: `ff015c7ba05fbc9dd77a58e4226b803004e573c72b8fd77e1d2022e40a343486`

Final gate evidence follows when all complete. All writes/temp/cache/artifacts
remain within golang; no external writes or commits are made by this packet.


## Final gates

Production/tests frozen; all these actual final gates PASS:

- Targeted native `go test ./mpnum -run '^TestInfIntRational' -count=1 -v`, session83989, all10 explicit RUN/PASS.
- Targeted checked race `go test -tags=g3checked -race ./mpnum -run '^TestInfIntRational' -count=1 -v`, session90421, all10 explicit RUN/PASS.
- Checked vet `go vet -tags=g3checked ./mpnum`, exit0.
- Fresh checked JS cross-build77223 to `.cache/inf-int-rational-js.test.wasm`.
- Actual minimal-environment Node execution with `-test.count=1 -test.v`, exit0; all10 names including source rational-route and post-failure groups observed RUN/PASS. Compilation or cached summary is not treated as runtime evidence.
- Checked WASI cross-build27482 to `.cache/inf-int-rational-wasi.test.wasm`, exit0.
- `go run ./cmd/portcheck`12069, exit0, no-generics/source audit (parent in-progress inventory14ported41partial2047unported).

Parent owns final combined full-module/isolated integration gates and metadata.
Earlier selected-file feedback explicitly excluded concurrently incomplete root
facade/ExtendedGCD files; all final gates above compile the actual complete
shared mpnum package without exclusions.

Parent final combined frozen native30734, full checked race63802, and full actual
checked JS67573 passed across every module package. All41 new packet tests have
explicit JS RUN/PASS evidence in
`.cache/rational-facade-inf-ext-gcd-frozen-checked-js.log`; source/test checksums
remained unchanged. Checked full vet, WASI build and no-generics/source audit
also passed. WASI runtime execution is not claimed.
