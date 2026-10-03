# Dense upolynomial selected core: implementation held for coordinator review

Source: `src/math/polynomial/upolynomial.h/.cpp`, `util/vector.h`, `util/mpz.h`, `_scoped_numeral_vector`, actual registered `src/test/upolynomial.cpp`. The selected concrete storage/arithmetic/sign-evaluation packet is implemented and frozen after the gates below. This is a selected algorithm/ownership packet in the full upolynomial→algebraic→arithmetic plugin backend; full GCD, factorization, Sturm/isolation/refinement, multivariate conversion and algebraic operations remain OPEN.

## Selected source surface

Complete actual coefficient-vector storage/lifetime and selected core/derived manager ctor/temp/destructor actions, checkpoint, mode access/configuration, reset/trim/set_size/degree and trim-mutating is_zero/is_const, raw integer and rational coefficient set, in-place/output negation, add/sub/mul, derivative, equality, power, scalar coefficient multiplication, and all three finite-point sign-evaluation overloads plus zero/±infinity. Include source helper bodies and precise aliases. Primitive/content normalization, division/exact division/GCD/monic/factors/sequences and the other transformation/root algorithms are explicitly later packets; do not advertise fake methods for them.

Power is included because actual registered basic `tst1` calls it and it needs only the selected arithmetic: source uses repeated multiplication, not binary exponentiation. General polynomial display and its printer/TRACE events remain OPEN pending a separately audited owned formatter (including source SMT2 single-term fallthrough defect); printing examples is not an assertion port.

## Concrete public API

`NewUPolynomialManager(UPolynomialConfig{Integer *IntegerManager, Checkpoint func() bool, Cooperation func() error}) (*UPolynomialManager,error)` owns actual source core/derived state and borrows one real IntegerManager and limit service. Checkpoint is required and bound to actual g3 utilReslimit.inc; no nil/default-success substitute. Cooperation is also required: it binds the coordinator's actual source `_NO_OMP_` cooperation service, preserving runtime enable state and call order.

Public polynomial owners are scalar `UPolynomial` tokens with private origin/slot/generation fields. Explicit `NewPolynomial`, `DelPolynomial(*UPolynomial)`, `Copy(dst,src)`, `Swap(a,b)`; size/capacity/degree, source `Reserve`, scoped `Resize`/`Shrink`/`PushCoefficient`/`PopCoefficient` adapters and scalar coefficient read/write actions. No public owning Go slices or raw numeric pointers; `ReadCoefficient(p,i,*Int)` deep-copies through the actual numeral manager, `SetCoefficient` performs actual modular set without automatically trimming. The caller owns read results. `Reset`, `Trim`, `SetSize`, `IsZero`, `IsConst`, `NegInPlace`, `Neg`, `Add`, `Sub`, `Mul`, `Derivative`, `Equal`, `Power`, `MulScalar`, plus source mode forwarding `SetZ`, `SetZP`, `SetZPUint64`, `Modular`, `Field`, `NumeralManager`, `IntegerManager` and private borrowed modulus access.

Explicit raw-source array overloads use a scalar `UPolynomialView` (origin, physical backing slot/generation, start/count), obtained by `View` while owning polynomial remains valid. `SetView`, `NegViewInPlace`, `MulScalarView`, `NegView`, `AddViews`, `SubViews`, `MulViews`, `DerivativeView`, `EqualViews`, `PowerView`, and private borrowed actual coefficient access preserve source base-pointer aliases and captured physical lifetime. Read guards are at actual dereferences, not a blanket logical-size restriction; source g3debug subscript assertions versus initialized physical extent are distinct. Relocation/free invalidates views, swap transfers backing ownership without invalidating physical views. Public whole-polynomial methods take source size/base once before operation and reuse these exact algorithms.

`SetInts(p,[]Int)` and `SetRationals(p,[]Rational)` borrow caller arrays only for the call and require the raw common integer owner contract. The latter source copies numerators, only debug-checks integrality, and does not silently multiply denominators or use alternate coefficients. Owned RationalService values need their separate explicit raw-copy boundary later; no illegal shallow owning rational copy.

Derived sign methods: `EvalSignInt(view,*Int)`, `EvalSignRational(view,*Rational)`, `EvalSignBinary(view,*BinaryRational)` and `EvalSignZero/PlusInfinity/MinusInfinity(view)` return `(int,error)`. Inputs must borrow the same real underlying IntegerManager as source's raw numeral pointers. Sign access is source is_zero then is_pos, not a mathematical sign computed by an unrelated manager. No general value-evaluation method is invented: source exposes sign evaluation here.

`Close` requires no live external polynomial owners, refusing before destruction while they exist; no implicit external-reference ownership release or closing the borrowed integer/limit owner. Actual users explicitly close their vectors first. Zero/nil/shallow owner copies are typed rejected domains via charged self metadata. Stats expose actual owner, metadata/page/directory, backing and child-manager bytes without double-counting the borrowed integer graph.

## Concrete storage and physical budget

Each coefficient is an actual named `Int` record; no token re-encoding, boxed value, per-coefficient pointer, unsafe or generics. Manager-owned scalar vector metadata (size/capacity/initialized/backing ID) lives in concrete typed descriptor pages. External scalar handles select descriptor rows; scratch vector descriptors are embedded in the actual charged cold owner. A cold backing directory owns each actual variable `[]Int` allocation, never exposing a slice as a polynomial. This is the source vector's one owned physical coefficient array, not COW or slice-copy polynomial semantics. The directory holds one slice header per physical array, not one Go object/pointer per coefficient; all directory capacities and row pages are charged, including overlap. Scalar backing IDs/generations model physical pointer validity.

Actual source `mpz` has a user-defined move constructor, so it is NOT trivially copyable. Source `svector<mpz>` growth allocates new capacity, publishes new backing, transfers logical numeral ownership in ascending order, destroys moved-from logical records, then frees old backing. Only logical entries move: initialized inactive capacity is not memcpy-preserved. Preserve old m_val after pointer transfer; stale unused cells are not owning roots. Source resize publishes the new logical size before ascending default construction. Reserve is logical growth, not capacity-only. Capacities are0→2→3→5…, with exact uint32 source multiplication/header arithmetic and selected LP64 sizeof(mpz)=16/header8 bounds independently of actual Go Int stride and host representability. Actual backing/cold headers, rather than fictional source bytes, determine budget charges. Source capacity's real default-initialized prefix is tracked to permit release retained accesses safely and detect uninitialized reads.

Source manager owns16 core vectors: basic1, division2, exact-division1, GCD2, CRA1, modular-GCD6, square-free2, power1. The actual derived manager adds5 more: db, dbab1, dbab2, translation, push. All21 real empty descriptors exist from constructor; pending algorithms are not callable stubs. Core uses the committed real IntegerModularManager; its actual cold child header is charged separately from the parent pointer field once. Parent constructor reservation/source-layout adaptation and child allocation prefixes are documented.

Destruction order matters: derived destructor resets its5 vectors in declaration order, then member destructors free their backings in reverse order. Base destructor resets its16 vectors in explicit source order, then member destructors free their backings in reverse order; its embedded modular manager destructor follows and releases its seven actual numerals. Do not simply iterate all21 vectors in one convenient order. Local scoped evaluation numerals use actual lexical reverse destruction; safe Go failure cleanup is recorded separately from raw MPZ locals lacking destructors.

## Actual selected algorithms/action order

- Trim scans highest coefficient downward, tests source numeral is_zero, DELs each zero, then publishes shrunk size. `set_size` asserts oldsize>=newsize only in debug, DELs old suffix ascending, shrinks, then trims. `Reset` calls set_size0; it retains vector capacity.
- Set(Int view) detects nonnull physical base pointer equality first and returns after debug equality-size assertion. Even a zero-count view of a nonempty same-base vector is release no-op, not a reset. Otherwise reserve, copy/normalize each coefficient ascending, then set_size/trim. Rational set reserves first, checks each rational integral in debug, copies its actual numerator, then trims. Numeric failure keeps copied prefix and real expanded size.
- In-place negation simply negates/normalizes each coefficient ascending, without trimming. Output negation uses basic scratch, copying then negating each coefficient, set_size/trim, then swaps vector ownership. Add/sub reserve max size, operate overlap ascending, copy each remaining left then right tail (right sub tail copied then negated), trim, then swap. Scratch partial failure remains owned; output stays until actual swap. No whole-operation rollback or result slice is invented.
- Mul resets basic scratch for either zero-size input. Otherwise computes source unsigned size1+size2−1, reserves, RESETs every result coefficient, swaps input order only when first size<second, then performs checkpoint BEFORE each outer row including zero rows. Zero input coefficients skip arithmetic; every nonzero pair executes actual AddMul(result,a,b,result). Trim then swap; partial cancellation/failure leaves scratch coefficient prefixes intact.
- Derivative writes directly to output (not basic scratch): reserve size−1, for i1…size−1 construct raw local d, modular Set(d,i), Mul(p[i],d,out[i−1]), then set_size/trim. In-place ascending writes do not clobber later reads. Source raw d has no explicit DEL; physically representable selected LP64 vector bounds keep i below INT32_MAX, so it remains inline; no invented destructor action. This invariant requires a scalar source-extent test.
- Equality first size comparison, then raw numeral EQ ascending, stopping at first mismatch. It does not trim operands or equate differently sized all-zero forms.
- Power k0 debug-asserts nonempty input, reserve1/set1/set_size; release empty^0 still follows finite source actions. k1/empty input/constant1 branch uses Set. Otherwise Set input into power scratch, then repeat actual Mul(powerScratch,input,powerScratch) for i1…k−1 and swap into output. The `#if 0` binary-power block is inactive and explicitly not ported as the active algorithm.
- Scalar multiplication zero resets vector; nonzero tests one before ascending Mul and does not trim automatically.
- Integer sign evaluation: empty0, singleton sign directly; scoped r initialized from last coefficient, Horner descending using Mul for zero coefficient or AddMul(coefficient,r,point,r), final sign, lexical local DEL. No checkpoint exists in this loop.
- Rational sign evaluation computes numerator-scaled Horner: scoped denominatorPower first, then r, then ak; each nonzero coefficient Mul(a,denPower,ak)/AddMul(ak,r,num,r); every iteration also multiplies denominatorPower by denominator, including final unused multiplication. It returns sign of scaled value, even for noncanonical negative denominators; no RationalManager Horner replacement.
- Binary sign evaluation scopes r then ak, uses actual borrowed numerator and exponent, k_i uint32 initialized k and increments k_i+=k AFTER each row, including final row. Each nonzero coefficient Mul2k(a,k_i,ak)/AddMul; zero coefficient Mul only. Preserve uint32 wrap, typed actual shift bounds, and source operand/lifetime actions.
- Zero/infinity signs access first/last coefficient as source; minus infinity flips by degree parity. No trim/default canonicalization is added.

## Resource/cooperation prerequisite

core_manager ctor always borrows real reslimit&. It cannot have a fake nil/default-success limit; Go mpnum cannot import g3 (g3 already imports mpnum), so one concrete cold callback binds to actual utilReslimit.inc. Root package tests can bind its real private service. The coordinator approved the NEW root `upolynomial_source_test.go` in this packet; package mpnum scalar tests may cover pure allocation arithmetic without inventing a limit-bound source owner.

Source checkpoint executes limit.inc first; false throws cancellation and cooperation is skipped. True then executes actual `cooperate("upolynomial")`. Full util/cooperate.cpp/h and coordinator z3_omp.h audit distinguish runtime enable state from the genuine `_NO_OMP_` source profile: thread ID0, in_parallel false, lock primitives no-ops, while source section/checkpoint/task state and debug assertions still execute. Coordinator committed actual utilCooperation (`fb08673`) implementing those source state transitions and sites; root tests bind both actual utilReslimit.inc and actual utilCooperation.Cooperate("upolynomial"). Both cold bindings are required. Limit refusal precedes and skips cooperation; enabled source no-OMP checkpoint publishes owner0, while init_task's enabled debug parallel assertion remains distinct. OpenMP multilane lock scheduling/TRACE stays OPEN. This is not a fabricated always-disabled profile; runtime enabled/disabled cases both have real source actions. Scalar/caller storage is charged by the actual owner.

## Source assertions and TDD evidence

Registered `tst_upolynomial` calls basic `tst1`, Zp, Zp2 and two extended-GCD examples. Basic t1 constructs a multivariate polynomial, converts, mutates coefficients, trims, squares, powers and prints; it has no ENSURE/VERIFY. Zp examples likewise perform real polynomial GCD/subresultant algorithms and print, with no standalone basic-core assertion. Their multivariate conversion/GCD/printer algorithms remain OPEN; recreating finite coefficient input vectors is an explicitly adapted input fixture, not completion of those tests.

The active entry also invokes `tst_gcd`, lower_bound, fact, rem, exact_div, isolate_roots5, isolate_roots (twice), sturm2, convert_q2bq, translate_q, refine, refinable, sturm, remove_one_half, basic tst1, Zp, Zp2, ext_gcd and ext_gcd_z7. Those complete consumer paths remain OPEN. The unconditional early return inside `tst_gcd` leaves its later examples inactive; the power body's `#if 0` binary algorithm is likewise inactive. This selected packet does not imply full registration/test coverage.

The active `tst_translate_q` has nine ENSUREs before actual translation (four rational roots, three nonroots, two fractional nonroots). These nine actions are ported using the exact source factor polynomial through actual dense arithmetic and source rational input construction, with the multivariate input-conversion adaptation explicit. Its later translation/reciprocal ENSUREs remain OPEN until the actual transformation algorithms execute. Root/refinement tests' evaluation assertions require actual preceding isolation/refinement and are not counted by substitute points. Existing registered consumer totals must not change until coordinator audits exact selected assertion routes.

The first missing-API RED contained the actual nine pretranslation ENSURE routes and basic coefficient construction/mutation/square/power routes with the declared dense input adapter. The later API and behavioral REDs below preceded their corresponding overload, source-width, read-timing and predicate corrections. Independent bounded convolution/derivative and exact rational/binary sign oracles, failure ownership and warm-allocation fixtures were then added and verified against the implementation; they are model/ownership coverage, not additional registered source assertion counts. No later GCD/Zp print route, isolation, translation or reciprocal assertion is claimed from those substitute inputs.

The approved scalar/cold layout and root test ownership are implemented in eight new Go files. Existing numerics/services/metadata/Git were not edited. Whole source units remain PARTIAL; the following map and gate evidence describe this selected packet only.

## Final source method and ownership map

| Source selected body/accessor | Concrete implementation |
|---|---|
| core/derived ctor, their explicit reset and implicit reverse vector destruction | `NewUPolynomialManager`, 16 core +5 derived embedded `upolyVectorState` members; `Close` executes both source phases and closes the actual modular child afterward |
| `zm()` / `m()` | `IntegerManager()` / borrowed `NumeralManager()`; neither borrowed owner is silently copied or closed |
| `lim()` | required concrete callback configuration borrows the actual caller `utilReslimit`; there is no replacement limit object or default-success service in mpnum |
| `p()` | private actual `borrowedModulus()`; public `Modulus(*Int)` explicitly adds an owned copy boundary |
| `modular/field/set_z/set_zp` overloads, `checkpoint` | `Modular`, `Field`, `SetZ`, `SetZP`, `SetZPUint64`, `Checkpoint`; limit refusal precedes and skips actual cooperation |
| `trim`, `set_size`, `reset`, `degree`, `is_zero`, `is_const` | `Trim`, `SetSize`, `Reset`, `Degree`, `IsZero`, `IsConst`; degree uses size without trim, both predicates actually trim |
| integer/rational `set` and vector `set` | `SetInts`, `SetRationals`, `SetView`, `Copy`; source same-base return precedes reserve, otherwise reserve precedes every source input dereference |
| raw in-place `neg`; `neg_core` / output `neg` | `NegViewInPlace` / `NegInPlace`; `NegView` / `Neg`, with actual basic scratch and source alias assertion |
| `add_core/add`, `sub_core/sub`, `mul_core/mul` | `AddViews/Add`, `SubViews/Sub`, `MulViews/Mul`; literal source scratch actions, checkpoints, trim and final swaps |
| `derivative` raw/vector overload | `DerivativeView/Derivative`, direct output writes; raw per-row multiplier remains inline within source physical extent |
| `eq` | `EqualViews/Equal`; different counts return before reading inputs; matching zero counts read no backing |
| active `pw` | `PowerView/Power`; actual repeated multiplication, not the inactive binary-power block |
| scalar `mul(unsigned,p,b)` / vector wrapper | `MulScalarView` / `MulScalar`; raw release zero multiplication leaves size, debug asserts nonzero; owning wrapper resets on zero |
| three finite `eval_sign_at` overloads | `EvalSignInt/Rational/Binary`; actual source scoped numeral actions, reverse cleanup, final unused denominator multiplication/exponent update preserved |
| zero/positive-infinity/negative-infinity | `EvalSignZero/PlusInfinity/MinusInfinity`; source no-read zero-size branches and degree parity |
| actual coefficient `svector<mpz>` instantiated storage | scalar descriptors, typed Int backings, literal source capacity action, ascending ownership moves, publication order, initialized-prefix guards |
| `_scoped_numeral_vector<mpzzp>` constructor/manager accessor/reset/shrink/resize/push/pop/destructor | external scalar owner `NewPolynomial`, `NumeralManager`, `Reset`, `Shrink`, `Resize`, `PushCoefficient`, `PopCoefficient`, `DelPolynomial`; general template family remains OPEN |

`UPolynomial` is16 bytes with private scalar provenance; `UPolynomialView` borrows scalar physical backing identity. Actual target `Int`/descriptor/page/backing directory strides use `reflect.TypeOf(...).Size()`. There are no public coefficient slices, boxed records, per-coefficient objects or generic types. The source profile uses LP64 mpz16/header8 independently of Go release/checked strides and target host bounds.

The literal source uint32 check is independently represented by `upolySourceCapacity`: old/new byte extents wrap in uint32, then `new_capacity <= old_capacity || new_capacity_T <= old_capacity_T` refuses. The actual empty-constructor geometric endpoint is204324849; the next306487274 fails the source check. A synthetic old268435456 passes the literal scalar source check (old bytes8, new bytes2147483656) but its wide allocation cannot contain402653184 source elements; a separate Go guard rejects that undefined source allocation/write continuation. This is distinct from source overflow behavior and from actual typed-Go host representability. Cold directories use their own checked physical doubling adapter; successful metadata/growth prefixes remain charged and retained after a later reservation refusal.

Source raw pointers are represented by borrow tokens resolved **at actual coefficient reads**, not eagerly copied into a Go slice. A zero-count stale view performs no backing read. SetView reserves/publishes destination size before a stale first coefficient causes the safe Go error. If a captured input has moved into basic scratch and subsequent source growth frees it, the next real read reports a typed lifetime error after the completed scratch prefix; it does not keep an uncharged old array readable. Debug alias assertions execute at the actual source helper entry. No whole-operation transaction or rollback replaces these source action prefixes.

## Explicit Go/source domains and remaining source scope

- External scalar polynomial owners must be explicitly released before manager Close; a live-owner refusal leaves the owner open. Borrowed Integer/resource/cooperation services must outlive the manager. Closing its modular child independently violates this ownership contract. Shallow cold-manager copies and zero cold owners return typed owner errors; copying an owning scalar token is an alias, not a second owner. Use actual `Copy` for numeric ownership duplication.
- Callback values are stored in the charged owner header. Their supplied environments and actual resource/cooperation owners are caller-borrowed, must outlive the manager, and are not fabricated/duplicated by it. All packet-owned arrays/pages/directories/cold headers are charged, including old/new overlap. `Stats.TotalBytes` covers that physical owner graph and its modular child's header; the borrowed IntegerManager's physical cell/limb graph is observed separately and is not charged twice.
- Captured raw views remain valid across ownership swaps and initialized retained-slot operations; relocation/free invalidates the physical generation. Dereference of source freed/uninitialized storage becomes a typed guard at the actual access, retaining completed source publication/actions. Owner-local scalar views reject foreign nonempty storage at reads; borrowed external `[]Int`/`[]Rational` inputs require the same implicit integer domain. This is a concrete owner-local specialization, not a general raw-pointer ABI.
- Source uint32 arithmetic is preserved where defined. Checked host representability and source physically undersized wrapped allocations return typed bounds errors. Generation/origin exhaustion does not wrap provenance into old live tokens.
- Local scoped eval numerals always release in actual lexical reverse order. Go error cleanup also releases partial locals on allocation/lifetime returns, an explicit safe ownership adapter rather than a claim about C++ invalid-pointer or allocator-exception continuation. Source derivative's raw local has no added normal destructor action; the source extent keeps every multiplier inline.
- Selected g3debug source assertions execute: alias checks, same-base size, set_size/shrink size, rational integrality, raw scalar nonzero, empty power0, underlying real modular normalization assertions. General source TRACE/printing remains OPEN. The actual cooperation `_NO_OMP_` profile is used, not claimed OpenMP scheduling.
- General vector/template families, polynomial display/SMT2/TRACE formatting (including the source missing-return single-term SMT2 behavior), primitive/content normalization, division/exact division, modular/subresultant/extended GCD, square-free/monic, factors/sequences, Descartes/root bounds, translations/reciprocal transforms, Sturm, root isolation/refinement, multivariate conversion, algebraic-number manager and actual AST arithmetic-plugin dispatch remain OPEN. No callable placeholder algorithms or rational-only algebraic substitute were added.
- Only the nine **pretranslation** ENSURE executions are selected source assertions here. Later translation/reciprocal/isolation assertions and real registered GCD/Zp consumer paths remain OPEN. The basic print-only `tst1` routes exercise actual coefficient mutation/trim/square/repeated power with an explicit dense input adapter; they are not counted as source assertions or completion of its multivariate conversion/printer.

## Observed RED and final immutable gate evidence

All files/cache/temp/build artifacts were written only under golang, and every Go build invocation used subtree GOCACHE/GOTMPDIR/TMPDIR. No external reference program was needed for this packet.

Missing/behavioral RED logs beneath `.cache/`:

- `upolynomial-initial-red.log`: absent scalar owner/source extent helper in isolated snapshot.
- `upolynomial-api-red.log`: actual source-nine/root fixture and real cooperation service, missing manager/config/API.
- `upolynomial-views-red.log`, `upolynomial-scoped-red.log`, `upolynomial-direct-red.log`, `upolynomial-scalar-view-red.log`: missing approved concrete overload/scoped/direct APIs before implementation.
- `upolynomial-source-extent-red.log`, `upolynomial-directory-red.log`: missing separate literal source arithmetic/physical cold bounds helpers before translation.
- `upolynomial-read-timing-red.log`: stale zero-count view wrongly read backing; subsequent fixtures cover reserve-before-read and scratch-transferred relocation.
- `upolynomial-predicates-red.log`: is_const originally omitted source trim; source header audit also corrected is_zero.
- `upolynomial-root-audit-red.log`: missing immediate cell/Words observation API before numeric failure-prefix coverage.

The finalized snapshot is `.cache/release/upolynomial-final-2f9931b`, archived committed HEAD `2f9931b49f54cece187019c7eb667a892722eed3` plus **only eight packet Go files**. Other live SAT/AST/numeric drafts were excluded. All seven gates finished exit0:

| Gate/session | Log | Evidence |
|---|---|---|
| full native88335 | `.cache/upolynomial-final-native.log` | root2.560s, mpnum22.480s, all modules PASS |
| full checked-debug56977 | `.cache/upolynomial-final-debug.log` | root2.869s, mpnum23.402s, all modules PASS |
| checked-debug race46847 | `.cache/upolynomial-final-race.log` | actual21 named RUN/PASS; root1.244s |
| actual checked-debug JS86114 | `.cache/upolynomial-final-js.log` | actual21 named RUN/PASS; root0.251s; real js-exec preserves subtree TMPDIR |
| checked-debug vet62281 | `.cache/upolynomial-final-vet.log` | PASS |
| checked-debug WASI build13989 | `.cache/upolynomial-final-wasi.log` | PASS |
| source/no-generics audit23355 | `.cache/upolynomial-final-audit.log` |2102units677APIs,33ported97partial1972unported; source statuses unchanged by this packet |

The21 groups comprise18 root source/model/ownership tests and three scalar-layout/source-growth/physical-directory tests. Independent finite evaluation executes216 rational,270 binary,54 integer point checks; convolution covers108 integer/modular/alias cases. Numeric error tests audit actual cells and Words references immediately before formatting/DEL/Close across24 evaluation and112 selected-output failure ordinal attempts, allowing only actual source retained cache and published owner roots. Cold FailAfter0 construction and warmed arithmetic both measure0 Go allocations. The real production destructor is observed through immediate borrowed live-cell accounting and LIFO cell reuse: power scratch is released after basic scratch, before borrowed integer teardown.

Immutable Go hashes are recorded in `.cache/upolynomial-frozen-go.sha256`:

```
510a2d0691b2807c015ae45e04a559e1909a46821362bbb2849b37f41fd2f053  mpnum/upolynomial.go
d5d0370c880c45c52ef50b1e0a8cf70e8851fc178439d793d58f104cd866d37f  mpnum/upolynomial_eval.go
357b801a5fd9dadcc98477abcee127bfb5c62d0a8b2e671c77cffca05823fa06  mpnum/upolynomial_modes.go
6a8d7eec23bdd58c7a76166fb5c438ab486675d802be5ecca50f47252826d039  mpnum/upolynomial_ops.go
c6fc923066bf2844cf4604b766d39c44521995f04bbd3617f60336e712369fc0  mpnum/upolynomial_storage.go
430dece6484dadb170437a66a596e8b165634298c7f4c36557ad1558521d4ec0  mpnum/upolynomial_test.go
00f3e1d1beb0f466d73fa7d0233c02fece058811ae345705a89a69e8df380c35  mpnum/upolynomial_vector.go
ea27fc20700a89bc15588958262872d3b15d8b1162664f66180360b1f0294ea4  upolynomial_source_test.go
```

## Coordinator integration

Snapshot `.cache/review/upolynomial-integration-a4df8ed` adds only the eight
frozen Go files to committed real registry/configa4df8ed. Coordinator verified
all eight live/frozen hashes and exact twenty-one race/actual JS RUN/PASS names.
Full native99616 and checked-debug6211 passed (mpnum22.862s/23.463s). Initial
ledger audit70898 exposed root-package test references assigned to mpnum-only
targets; corrected metadata separates three mpnum declaration groups from
eighteen root integration groups. No Go source changed. Final source/no-generics
audit65261 passed, ledger33ported/102partial/1967unported. Full upolynomial
and algebraic/solver/proof/interpolation obligations remain open.
