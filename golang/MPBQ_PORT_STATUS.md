# Binary-rational selected implementation

The approved packet is implemented in the new files listed below, with fourteen permanent tests. The approved source audit and contract are retained here; observed RED/GREEN and final verification are recorded at the end. `src/util/mpbq.h`, all of `src/util/mpbq.cpp`, and all of `src/test/mpbq.cpp` were read. Parent independently reviewed the frozen implementation and records selected method coverage centrally; whole MPBQ source units remain partial for trace and full custom ostream integration. Existing MPZ/MPQ arithmetic, roots, rounding, owner-local services, cross-owner copying, and budgeted formatting are actual dependencies already implemented; this proposal does not reopen their frozen files.

The source is Copyright (c) 2011 Microsoft Corporation, Leonardo de Moura, with the 2012 notice in the test file. Preserve those notices in subsequent translations. `golang/AGENTS.md` overrides generic sketches: every proposed Go type and method is concrete; no generics, unsafe, production math/big, interface-boxed hot records, or global numerical manager.

## Complete proposed packet and file ownership

Implement the full selected internal-32-bit `mpbq`/`mpbq_manager` contract, including inline methods, all four interval-selection overloads, conversions, formatting, source debug conditionals, and the concrete scoped/vector specializations and scalar operator overloads used by the active tests. Do not call the unit complete after only addition/approximation. `mpbqi.h`, general interval managers, polynomial algorithms, algebraic-number isolation, and their AST adapters remain separate unported dependencies.

Approved new files only:

- `mpnum/mpbq.go`: scalar representation, constructor/lifecycle/set/query APIs, manager ownership and stats.
- `mpnum/mpbq_arithmetic.go`: add/sub/mul/fused operations, shifts, power, comparisons and magnitudes.
- `mpnum/mpbq_root.go`: root bounds, floor/ceil and conversions.
- `mpnum/mpbq_select.go`: four integer-selection helpers, four small-selection paths, directed approximation/division, refinement.
- `mpnum/mpbq_format.go`: plain/pretty/SMT2/decimal formatting into existing owner workspace.
- `mpnum/mpbq_debug.go` and `mpnum/mpbq_debug_release.go`: `g3debug` source diagnostics versus release selection.
- `mpnum/scoped_binary.go`: concrete scoped numeral, owning vector, and int32/Int overload routes.
- `mpnum/mpbq_test.go`, `mpnum/mpbq_select_test.go`, `mpnum/mpbq_format_test.go`, `mpnum/scoped_binary_test.go`: permanent source traces, independent arithmetic, storage and failure tests.
- This status file. No existing manager/service files, central ledger/manifest, or git edits.

## Representation, owner graph, and proposed constructor

`BinaryRational` is a pointer-free scalar `{ num Int; k uint32 }`, denoting `num/2^k`. Raw Go zero is the valid source default 0/1. `NewBinaryRational()`, `NewBinaryInt(int32)`, and `NewBinaryIntPower(int32,uint32)` reproduce source constructors; the last **does not normalize**. Manager setters with a denominator power do normalize. Owning big numerators may not be shallow-copied; `Copy` deep-copies, `Swap` transfers ownership, and the lexical scoped owners reject copied owner headers. There is no invented denominator allocation.

Proposed cold-owner constructor:

```go
type BinaryRationalConfig struct { Diagnostics *RationalService }
func NewBinaryRationalManager(z *IntegerManager, c BinaryRationalConfig) (*BinaryRationalManager, error)
```

The manager borrows `z`; caller binary values and scoped/vector owners must be destroyed before this manager closes, and it must close before `z.Close`. It never closes `z`. Operations remain owner-confined, matching the source unsynchronized manager. A self pointer in the cold manager detects shallow copying; it does not make raw numeric handles detect foreign managers. `Close` is idempotent; nil/copied/closed owner and nil numerical arguments receive existing typed owner/storage errors before arithmetic.

The source constructor initializes only default scalar scratch, with no integer arithmetic or eager limb allocations. Retain exactly its eight Int and three binary scratch fields (eleven owning numerators): `tmp`, `tmp2`, `addMulTmp`, `selectIntTmp1/2`, `selectSmallTmp`, `selectSmallTmp1/2`, and `divTmp1/2/3`. Charge the actual manager header through `z.budget`; inline scratch lives in that header. Big payloads use existing numeric cells/Words and the same Budget. Close reproduces source destructor order: addMulTmp; tmp; tmp2; selectIntTmp1; selectIntTmp2; selectSmallTmp; selectSmallTmp1; selectSmallTmp2; divTmp1; divTmp2; divTmp3. Report owned versus borrowed bytes and live numerator payloads separately; never double-charge the borrowed integer owner or numeral header fields.

For `g3debug`, borrow the supplied `Diagnostics` service instead of inventing a global service. Require a valid service with the same Budget at constructor entry in that build; release can accept nil. The parent approved this explicit borrowed-diagnostic context adaptation. Diagnostic arithmetic follows source `to_rational` and wrapper actions, not a production mathematical oracle. Service locking is confined to its own actions; no service callback acquires a binary-manager lock. Both owners and their shared Budget are externally confined. Diagnostics contribute real additional scratch/cache/peak costs; no claim of release-equivalent allocation profile.

## Proposed concrete API mapping

Names below represent ordinary methods on `*BinaryRationalManager`; numerical inputs/outputs are pointers. Errors are explicit even for source void/bool methods. Queries returning a bool use `(bool,error)`; magnitudes use `(int32,error)`; powers use uint32. Families are listed rather than replaced by a generic or interface dispatcher.

| Source methods | Proposed Go names and shape |
| --- | --- |
| precise, field | `Precise() bool`, `Field() bool` (true, false) |
| swap, del, reset scalar | `Swap(a,b) error`, `Del(a) error`, `Reset(a) error` |
| reset(mpbq_vector&) | `ResetVector(v *ScopedBinaryVector) error`, ascending **Reset** then clear logical length, retaining backing capacity |
| set(int), set(unsigned) | `SetInt32(dst,n)`, `SetUint32(dst,n)` |
| set(int,k), set(int64,k), set(mpz,k) | `SetInt32Power(dst,n,k)`, `SetInt64Power(dst,n,k)`, `SetIntegerPower(dst,n,k)` |
| set(mpz), set(mpbq) | `SetInteger(dst,n)`, `Copy(dst,src)` |
| is_int, numerator, denominator power | `IsInteger(a)`, `Numerator(a,out *Int) error`, `DenominatorPower(a) (uint32,error)` |
| zero/nonzero/one/pos/neg/nonpos/nonneg | `IsZero`, `IsNonzero`, `IsOne`, `IsPos`, `IsNeg`, `IsNonPos`, `IsNonNeg` |
| add/sub/mul(BQ,BQ) | `Add(a,b,out)`, `Sub(a,b,out)`, `Mul(a,b,out)` |
| add/sub/mul(BQ,Int) | `AddInteger(a,b,out)`, `SubInteger(a,b,out)`, `MulInteger(a,b,out)` |
| addmul(a,b,c,r), mixed overload | `AddMul(a,b,c,out)`, `AddMulInteger(a,b *Int,c,out)` |
| neg, div2/div2k, mul2/mul2k | `Neg(a)`, `Div2(a)`, `Div2k(a,k)`, `Mul2(a)`, `Mul2k(a,k)` |
| power in-place and copy overload | `PowerInPlace(a,p)`, `Power(a,p,out)` |
| root lower/upper in-place and copy overload | `RootLowerInPlace(a,n)`, `RootLower(a,n,out)`, `RootUpperInPlace(a,n)`, `RootUpper(a,n,out)` returning `(bool,error)` |
| eq/neq/lt/gt/le/ge(BQ,BQ) | `Equal`, `NotEqual`, `Less`, `Greater`, `LessEqual`, `GreaterEqual` |
| six comparisons(BQ,Int) | same six names with `Integer` suffix |
| six comparisons(BQ,Rat) | same six names with `Rational` suffix, signature `(a *BinaryRational, qm *RationalManager, b *Rational)` |
| magnitude_lb/ub, lt_1div2k | `MagnitudeLower`, `MagnitudeUpper`, `LessOneDiv2k(a,k)` |
| to_mpbq | `FromRational(qm,q,dst) (bool,error)` |
| template to_mpq, to_rational | `ToRational(qm,src,dst) error`, `ToOwnedRational(service,src) (OwnedRational,error)` |
| floor/ceil with external integer manager | `Floor(a,out)`, `Ceil(a,out)` plus `FloorInto(a,zm,out)`, `CeilInto(a,zm,out)` |
| select_small, closed select_small_core | `SelectSmall(lower,upper,out) (bool,error)`, `SelectSmallCore(lower,upper,out) error` |
| three mixed select_small_core overloads | `SelectSmallRationalLower(qm,lower,upper,out)`, `SelectSmallRationalUpper(qm,lower,upper,out)`, `SelectSmallRationalBounds(qm,lower,upper,out)` |
| refine_upper/lower | `RefineUpper(qm,q,lower,upper)`, `RefineLower(qm,q,lower,upper)` |
| approx, approx_div | `Approx(a,k,toPlusInf)`, `ApproxDiv(a,b,out,k,toPlusInf)`; explicit-default forwarding method selects source k=32,false |
| to_string/display | `String(a)`, `FormatInto(a,dst)` |
| display_pp | `PrettyString(a)`, `FormatPrettyInto(a,dst)` |
| display_smt2 | `SMT2String(a,decimal)`, `FormatSMT2Into(a,decimal,dst)` |
| display_decimal(single and pair) | `DecimalString(a,precision)`, `FormatDecimalInto(a,precision,dst)`, `IntervalDecimalString(a,b,precision)`, `FormatIntervalDecimalInto(a,b,precision,dst)`; default forwarder selects source precision 8 |

Direct Int arguments use the same implicit-owner contract as existing IntegerManager. Rational methods explicitly identify `qm`; if `qm.z==z`, read fields directly after checked initialization. Otherwise use budgeted lexical `z.CopyFrom` numerators/denominators, preserving raw shape; never interpret foreign cell handles through z. Mixed selection still executes the source qm Ceil/Floor/Mul actions using qm-owned lexical temporaries, then copies results into z as the source boundary requires. `ToRational` performs source **Power(2,k,target), Inv(target), MulInteger(numerator,target,target)** in that order, with a cross-owner numeral copy where necessary; it does not substitute SetParts. `ToOwnedRational` preserves source owning r(numerator), twok assignment from power(rational(2),k), and value-return division copy routes. Existing private service helpers can be used within new package files under the service lock, without new exported raw borrows.

`FloorInto`/`CeilInto` use the requested output integer manager for actual arithmetic, with a checked lexical input copy only when owners differ. Foreign scoped bridges validate their self/owner first. Explicit cross-owner copies are storage adapters, not new source arithmetic algorithms.

## Exact arithmetic/control-flow obligations

- Normalize: early return for k=0; set zero numerator's k=0; otherwise compute numerator's power-of-two multiplicity, cap by k, source MachineDiv2k, subtract k. Preserve the inherited small INT_MIN/shift31 sign defect. Del calls Int.Del only and **leaves k and source inline m_val intact**; Reset calls Int.Reset and sets k=0. Copy retains raw representation and source requested numerator capacity; it does not normalize.
- Add/Sub align denominators into retained tmp and preserve operand order in all three k branches, then normalize. Mul multiplies numerators first, writes `r.k=a.k+b.k` at uint32 width, then **rereads** `a.k==0 || b.k==0` after publication. Output aliases can therefore suppress normalization; do not snapshot that condition or force normalization. Mixed Mul always normalizes. AddMul uses retained tmp and source order `Mul(b,c,tmp)` or `Mul(c,b,tmp)`, then Add(a,tmp,out).
- Div2/Div2k increment/add exponent at uint32 width and normalize only if its old value was zero. Mul2 decrements nonzero k or shifts the integer numerator; Mul2k either subtracts k or shifts numerator by the excess. Magnitudes compute source unsigned-width log2-minus-k(+1), then explicitly convert to int32. Source equality is representation equality; a raw constructor 2/2 need not Equal normalized 1 although order comparisons consider their values equal. One likewise requires k=0.
- Power multiplies exponent **before** integer Power. Preserve source copy-before-in-place overload. Source uint64 exponent-product assertion becomes a build-aware diagnostic; release multiplication wraps. Inherited typed PowerDomainError/0^0 errors remain distinct. Proposed failure policy is source statement-prefix publication, including exponent changes before failing inherited Power; do not silently add a whole-BQ rollback transaction.
- RootLower uses integer root upper bound, Dec numerator when result is false, then rounds denominator exponent according to the resulting numerator sign. RootUpper does not Dec and uses the complementary exponent rounding. Both normalize and retain the source bool, including raw zero with nondivisible k reporting false. Degree zero is undefined in BQ `%n` even if integer-root zero returned true; classify before that undefined division. Negative-even root uses existing typed domain error. Copy overload publishes source Copy before in-place actions.
- Floor/Ceil use MachineDiv2k and then unconditional Sub(1) for negative fractional representation / Add(1) for positive fractional representation. Raw unnormalized -2/2 thus floors to -2, and raw 2/2 ceils to 2. Do not apply conventional mathematical rounding to these source fixtures.
- FromRational is exact for denominator1 or positive power2, otherwise stores numerator/2^(Log2(den)+1) and returns false. It is not a generic nearest approximation. Rational comparison cross products do not flip order for raw negative denominators; preserve reachable finite raw-shape results, document the normalized-denominator expectation of interval algorithms.
- SelectInteger chooses source first applicable endpoint/integer bound; closed BQ/BQ, open Rat/BQ, open BQ/Rat, and open Rat/Rat variants have distinct ±1 endpoint adjustments. SelectSmallCore BQ/BQ first tries integer; then linear search when `min(kLower,kUpper)<=8`, otherwise source binary search with final VERIFY call. Mixed variants always use source linear doubled-interval loops and scoped qm temporaries. SelectSmall returns false without changing out for reversed closed bounds.
- Refinement repeatedly Add endpoints into lexical mid, Div2, compare with q, Swap one endpoint, and only Del mid on the successful return (lexical cleanup also handles Go errors). Source preconditions require strict enclosure and a non-power-two denominator. No additional denominator-sign or reducedness guard was added: finite raw negative/unreduced examples retain the source action results. Source strict-enclosure/non-power2 preconditions have the approved typed adapters. Normalized positive-denominator nonbinary rationals and valid intervals terminate without an arbitrary iteration cap.
- Approx retains input when k is already small enough; otherwise uses Abs, truncating MachineDiv2k, increment iff sign differs from direction, restore sign, publish k, Normalize. Increment is unconditional in that direction even for raw exact-divisible input. ApproxDiv has three exact source branches: positive power-two divisor numerator; divisor numerator divides numerator; otherwise scaled absolute numerator and truncating division plus directional increment. Preserve all k arithmetic widths, numerator aliases, exponent publication and inherited Div divisor-alias defects. Never replace by exact rational division plus generic rounding.

Predictable errors in underlying integer operations retain their existing staged guarantees; the BQ compound method preserves previously completed source statements. Every error path still releases lexical numeral roots immediately. Source exceptions do not define safe ownership cleanup; cleanup is the explicit Go adaptation, not a claim of identical C++ throw behavior.

## Formatting, diagnostics, and scoped specialization

Use z's existing budgeted NaturalManager text workspace and repeated-decimal-division helpers. Append offsets preserve existing prefixes through workspace growth. No per-call limb slices, digit slices, fmt intermediate strings, or decimal-power Go strings. Formatting prepares full owned text before caller-buffer publication; short buffers/budget failures leave caller bytes unchanged. Returned Go strings contain one final caller-owned allocation; do not claim that escaped lifetime is manager-budgeted. Length excludes NUL.

Plain formatting distinguishes k=0, /2, /2^k. Pretty formatting uses `/2<sup>k</sup>` for k>1. SMT2 uses exact `(/ NUM (^ 2 K))`, decimal suffixes on NUM, 2 and K, and the existing negative integer SMT2 route. Single decimal follows source absolute numerator/Power(2,k)/Rem/Div; exact termination omits `?`, exhausted precision appends `?`. Integer fast path has no decimal point. Raw noninteger representation with zero remainder violates the source SASSERT; release still performs the source loop. Interval decimal emits `?` immediately for differing signs; for equal negative signs its leading `-` precedes a possible `?`; mismatched integer parts or digits end at their exact source prefix. If both remainders become zero, return precise text, otherwise append `?` at exhausted precision. All local numeral cleanup reproduces source sequence on success and lexical cleanup on error.

`g3debug` reproduces source before/after rational conversions in Normalize, six Add/Sub/Mul bodies, and Approx strict-bound checks, plus ordinary source assertion sites (exponent product, interval conditions, division conditions, decimal remainder). Failed diagnostics use a typed invariant error and retain source publication prefix; real diagnostics are not replaced by empty tag handlers or test-only arithmetic. The VERIFY selection action executes in both builds. TRACE/STRACE logging is observational instrumentation and remains explicitly separate, not an invented logger prerequisite.

Concrete `ScopedBinary` owns one numerator, borrows its manager, charges its actual cold header, and supplies checked Get/Manager/Set/Assign/Clone/Reset/Swap/Close, source predicate/Neg, comparisons and Add/Sub/Mul compound/value methods. Int32/Int overloads create scoped `_b`, call Set, then the BQ/BQ operation; clean `_b` at the source expression boundary. Generic source BQ/BQ value operators use copy-before-compound and returned-lvalue-copy semantics as existing scoped/facade translations do. Test expressions destroy every temporary at each source statement. Do not expose Div/Rem/Abs methods: those generic template functions cannot instantiate for this manager because it provides no div/rem/abs member.

`ScopedBinaryVector` is a concrete cold owner with contiguous pointer-free BinaryRational payloads, exact source vector growth 2 then (3n+1)/2, actual-capacity byte charges, old/new allocation peak, checked size/host-length arithmetic, and explicit transfer/invalidation on growth. The one Go slice belongs to the vector owner, not each hot numeral. Push publishes a default element before Copy as source scoped_vector does; a failing Copy leaves its default/prefix slot logically present. Shrink/Pop/Reset destroy ascending numerators via Del; manager ResetVector instead uses Reset before retaining vector backing. Final Close frees backing/header and closes no borrowed manager. Source vector's raw scalar storage destruction never automatically owns/destroys big numerators; ownership routes must be tested separately.

## Active source tests and independent fixtures

`src/test/mpbq.cpp` registers tst1 and tst2. There are **zero ENSURE/test assertions** and six printed result events. Port exact source constructor/set/operator/assignment order, output alias in tst2, and statement/destructor lifetimes FIRST, before production implementation. Add the following fixed expectations, independently derived from integer powers/truncation/normalization; these are added regression assertions on source print events, not six original assertions and not observed C++ executions:

```text
36637340735244577447646647796940134554460176108377786095865669439633741578249/2^67
1066286953144141515008134528672675792897940150822331957049483591681/2^32
496528555240547989964547813251929261208484019535748268035/2
-496528555240547989964547813251929261208484019535748268035/2
-1066286953144141515008134528672675792897940150822331957049483591681/2^32
283568639100782052886145506193140176213/2^127
```

Tst1 starts at INT_MAX, then +1, square, *3-1, square-5, square-7, Div2k(67). Approx directions are positive upper/lower then negative upper/lower, each after Copy of the original. Tst2 computes aliased ApproxDiv(5,3,a,128,false); its truncated numerator has one factor of two, yielding exponent127. Expected values above were checked by finite Python integer arithmetic during this proposal audit; no C++ reference run was needed.

Permanent added tests cover finite signed dyadics using test-only big.Int/big.Rat, all denominator-alignment branches, raw/normalized constructor differences, source Eq versus value ordering, denominator wrap and Power prefixes, mixed Int/Rat/cross-owner copies, every output alias, root true/false bounds, Floor/Ceil raw defects, all three ApproxDiv branches/directions/signs, and scalar magnitude wrap. Select tests force integer fast paths, linear threshold8, binary threshold9 including both final branches, open-endpoint adjustments, and both refinement endpoint-update branches; independently verify enclosure and minimum exponent for bounded valid normalized intervals. Formatting covers precision0, exact termination, digit mismatch, negative `-?`, raw remainder0 release/debug differences, and all SMT2/pretty forms.

Storage tests audit actual owner headers, eleven retained numeral roots, all constructor/refusal prefixes, whole-graph budget totals without double charges, vector old/new peaks, warm zero allocations/reservations, alias failure publication at source statement boundaries, copied-owner rejection, cross-owner provenance checks, repeated Reset/Del/Close, and immediate live cells/Words before rendering or teardown so Close cannot conceal lost temporaries. Failure sweeps continue to the measured successful reservation prefix; do not assume a fixed small number. All finite tests remain hermetic.

Observe initial RED compilation for the real proposed APIs, then GREEN implementation. Final gates: targeted and full native, actual g3checked+g3debug race, named RUN/PASS actual checked-debug JS, vet, WASI compilation, and the no-generics/source inventory audit. Parent centrally reviews and commits. The parent verification below records those selected-profile gates; the coverage ledger keeps remaining source integration open. Completing this prerequisite does not complete MPBQI, algebraic numbers, polynomial managers, or arithmetic AST plugins.


## Observed implementation and review evidence

All API groups in the mapping above are implemented, including private Normalize and all four SelectInteger algorithms. Public source scalar k access is `BinaryRational.K`; numerator extraction is an explicit owning copy. Concrete scoped specialization includes all six BQ comparisons, all twelve int32/Int comparison overloads, six int32/Int value overloads, source BQ copy-before-compound/value-return-copy, assignment/self-assignment, scalar predicates/Neg and owning vectors. The generic source Div/Rem/Abs functions cannot instantiate for MPBQ and are deliberately not invented. The raw vector-reset overload uses the concrete vector owner while retaining its distinct ascending Reset action; generic raw-vector shallow-copy utilities are not advertised as numerical ownership APIs.

Initial source-fixture RED was observed before any implementation: `go test -run '^TestBinarySource' ./mpnum` failed on undefined manager/config/scoped APIs. Additional test fixtures were added before selection/root/format implementation. Both active source bodies and their six independently derived output assertions became GREEN. Source test assertion count remains **0**; the six print expectations and all mathematical/storage tests are added regressions. The source review found a diagnostic `&&` short-circuit issue; `TestBinaryDebugEnclosureShortCircuit` observed RED (`ErrInjectedFailure` from wrongly evaluating the upper comparison), then GREEN after restoring source short-circuit evaluation. The two source-test named owners destruct b before a; full-expression intermediate results destruct in reverse creation order.

Fourteen named tests cover finite arithmetic and aliases, independent signed root bounds, all selected comparison/conversion families, source equality and alias representation quirks, source INT_MIN/shift31 inheritance, uint32 exponent and magnitude wrapping, power and copy/root error prefixes, both approximation directions, three approximation-division branches, all four selection paths including threshold8/9 and the binary-search VERIFY action, real same-owner and foreign-owner rational bridges, formatting branches/atomic output, pointer-free record shape, actual cold-header charges, vector growth and publication prefixes, warm zero allocations under FailAfter(0), and immediate fault ownership.

Every allocation-failure prefix is checked before formatter/Del/Close: all eleven retained binary scratch roots, caller roots, six integer-manager fixed roots, NaturalManager Words scratch, and diagnostic-service roots are accounted for against live cells and live Words. Successful ToOwnedRational results are explicitly retained and audited as caller-owned before Del. Constructor refusal leaves borrowed owner storage and budget unchanged; successful constructor charges exactly the Go manager header once. Close of manager and vector releases only owned roots/header/backing; borrowed integer and diagnostic owners are closed separately by the test graph. Reservation sweeps terminate on observed success rather than assuming a small fixed success prefix.

Reachable raw-shape interval regressions preserve source results, without a blanket invalid-denominator restriction:

- q=-1/-3, l=1,u=0: RefineUpper returns l=1/2,u=1/4; RefineLower returns l=1/2,u=0. A mixed rational-lower selection with closed BQ endpoint0 returns0. These are finite source cross-product paths with reversed real order.
- q=3/6, l=0,u=1: RefineUpper publishes l=1/2,u=3/4; RefineLower publishes l=1/4,u=1/2. Release returns normally; g3debug returns ErrBinaryInvariant at the final strict-enclosure assertion after that publication prefix.
- Approx on raw0/2^8 toward minus infinity publishes0; its source debug strict inequality fails. Single decimal of raw2/2 renders1.0 in release and triggers its source remainder assertion in g3debug. Raw floor/ceil and aliased Mul normalization differences remain explicit fixtures.

No whole-binary-operation rollback transaction was introduced. Underlying MPZ methods keep their existing budget/failure adaptations; the BQ layer preserves completed source statements, exponent updates and aliases. Cross-owner CopyFrom, cold Go scope/owner headers, safe vector transfer and staging caller text are declared Go storage/ownership adapters. No per-call unaccounted limb/digit/intermediate-string scratch was introduced. Escaped final Go strings remain caller-owned; they are not falsely charged for an unknown escaped lifetime. Scoped/vector raw borrows follow the existing lexical validity contract and may not survive growth/shrink/reset/close or be shallow-copied as owners.

MPBQ_DEBUG under g3debug performs real source wrapper-rational conversions before/after Normalize and Add/Sub/Mul, with source expression cleanup; Approx keeps a scoped binary snapshot and compares through the binary Less route. Plain SASSERT sites map to build-aware invariant errors, including unsigned product checks and raw-decimal assertions; source interval/nonzero/undefined-root domain adapters are explicit. VERIFY always executes. TRACE/STRACE logging remains separate unported observational instrumentation, so whole-fork diagnostic-service integration is not claimed complete. GMP/alternate limb backends remain inherited unported numerical configurations; this packet implements the selected internal32 arithmetic path only. MPBQI, interval manager, polynomial/algebraic managers and AST integration remain open.

Final gates on frozen Go bytes (all build/cache/temp/output paths under golang):

- Full native `go test -count=1 ./...`: PASS, session91106, mpnum10.211s.
- Targeted actual g3checked+g3debug race `-run '^TestBinary' ./mpnum`: PASS, session92899,8.040s.
- Actual g3checked+g3debug JS execution: PASS, session43657, all14 named RUN/PASS records,2.584s.
- Full g3checked+g3debug vet: PASS (completed direct call).
- g3checked+g3debug WASI test compilation: PASS, session29704.
- No-generics/source inventory `go run ./cmd/portcheck`: PASS, session11638;2102units/677APIs,32ported/58partial/2012unported before central MPBQ metadata update.
- Full actual g3checked+g3debug module gate: PASS, session59035, mpnum10.289s.
- Full actual g3checked+g3debug race module gate: PASS, session70853;mpnum138.812s/utilhash3.309s, all modules.

No git, central manifest, source-tree, external-temp or existing numerical-file edits were made by this packet. Parent owns central metadata and commit. Source whole-unit status must remain honest about open logging/integration/configurations even though the selected mathematical APIs and build-aware arithmetic diagnostics are implemented.


Whole-header completeness boundary: ScopedBinary.IsNonPos and IsNonNeg are implemented direct manager forwarders (frozen scoped_binary.go lines155/161); the source free-standing predicate friends map to these ordinary concrete methods. No missing numerical primitive is implied. Persistent ostream state/failbit integration and stream operator registration remain open: current buffer/string APIs stage text and publish atomically at an explicit Go output boundary. Source logging, persistent stream integration, the full generic header surface beyond instantiated MPBQ routes, and dependent interval/AST integrations keep whole-source headers partial; selected numerical algorithms and concrete member forwarders must not be marked missing because of those separate gaps.

## Independent parent verification

Parent snapshot `.cache/release/mpbq-integrated` contains committed HEAD plus
exactly these twelve frozen Go files; unfinished AST tests and other drafts
are excluded. All twelve SHA256 hashes match the agent handoff and authoritative
worktree. Parent full native session57335 PASS (mpnum10.560s); full checked/debug
session14038 PASS (10.583s); targeted actual checked/debug race74472 PASS;
actual checked/debug JS36795 PASS (2.634s). Both race/JS logs have exactly
fourteen distinct expected RUN/PASS groups and the PASS footer. Checked/debug
vet PASS, WASI compilation82353 PASS. Central source/no-generics audit35987 PASS:
2102 units/677 APIs,33ported/60partial/2009unported. Exactly five central source
rows changed; all source hashes/assertions/conditionals/registrations and API
entries are unchanged. This numerical prerequisite does not implement the
remaining solver, proof or interpolation objective.
