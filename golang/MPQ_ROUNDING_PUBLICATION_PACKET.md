# Rational rounding and scalar-projection source audit

Read-only proposal based on committed `c7bed52`. Only this new document is
written; no production, tests, metadata or Git edits. The GCD packet is
committed and its physical accessor regions now belong to the arena agent.
All implementation remains concrete Go, with no generics, unsafe, cgo,
production math/big, algorithm replay or scratch repair. Current gates are
native, checked/debug and vet; platform and race verification is deferred by
user instruction.

## Recommended next complete executable family: rounding

The actual remaining rounding debt is caller publication, not the mathematical
algorithm. `src/util/mpq.cpp:78–99` and `mpq.h:239–251` are complete, short
source bodies. Current `mpnum/mpq_round.go` implements their arithmetic in
roundCore correctly, but roundToInt and round allocate extra work values and
Move only after all actions succeed. These wrappers conceal completed source
caller Copy/quotient/sign-correction and retain an old rational denominator
past its actual source reset on success. Removing only those additional
transactional wrappers closes all four actual output overloads; the complete
PrevPowerOfTwo consumer stays an actual scoped integer Floor then integer
PrevPowerOfTwo, without a replacement logarithm or redundant division.

| Public Go route | Exact source action | Current state / proposed change |
| --- | --- | --- |
| FloorToInt | If denominator is exactly small1, Copy caller from numerator and return. Otherwise save numerator-negative flag, MachineDiv(num,den,caller), then Sub(caller,1,caller) if saved flag. | roundCore agrees; roundToInt adds nonsource staging. Direct caller route. |
| CeilToInt | Same integer Copy. Otherwise save numerator-positive flag, MachineDiv into caller, then Add1 if saved flag. | Arithmetic agrees; same extra staging. Direct caller route. |
| Floor | Floor into caller.num, then Del caller.den / scalar1. | Current extra rational staging. Denominator reset only after successful integer route, exactly as source. |
| Ceil | Ceil into caller.num, then reset denominator. | Same correction, no whole-rational rollback. |
| PrevPowerOfTwo | Actual source scoped integer local; Floor(a,tmp), integer prev_power_of_two(tmp), lexical Del. | Local is actual source, not staging to remove. Existing route inherits Floor dependency. Preserve returned0 for nonpositive integer. |

The numerator sign MUST be read before division, including numerator/denominator
output aliases. There is no denominator-sign test or remainder test in source
rounding. Raw unreduced2/2 Ceil produces2, raw-2/2 Floor produces-2; raw1/-2 Ceil
produces1 and raw-1/-2 Floor produces-1. These are finite source action quirks,
not changes to fix with mathematical rational rounding. Independent mathematical
floor/ceil oracles apply only to canonical fractions with positive denominator;
raw fixtures need literal source action expectations. Ordinary invalid-owner,
zero-denominator and dead-handle guards remain explicit Go boundaries. No
underlying integer division/shift/GCD/setter or allocator change is proposed.

### Exact proposed file/API ownership

After parent review, production scope is existing `mpnum/mpq_round.go` only:
remove extra work/Move in roundToInt and round, call existing roundCore against
the actual caller, and retain source final resetDenominator. Keep roundCore's
single Copy/MachineDiv/Add/Sub paths and PrevPowerOfTwo actual source local.
Public signatures FloorToInt, CeilToInt, Floor, Ceil and PrevPowerOfTwo are
unchanged; there is no missing-API RED to fabricate. No new record/layout,
owner wrapper, implicit allocation service or numeral namespace is needed.

New tests first: `mpnum/mp_round_publication_test.go`; this packet document.
Existing tests remain unchanged until actual named dependent RED and parent
approval. Expected actual old fixture debt is
TestRationalRoundFailureCleanup in mpq_round_test.go, which demands unchanged
all outputs after every error. Its source-success oracles, initializer/Close
checks,16 registered PrevPower assertions and other groups remain untouched.
Service value-return Floor/Ceil already create a source rational result then
invoke raw manager and transfer it. Their local is actual rational.h:278–287
value-return storage, so service production is read-only, not another staging
rewrite. Any service fault fixture exposed by the changed dependency requires
separate named RED/source review before editing.

### Tests-first behavioral and source inventory

1. Integral BIG input, distinct SMALL/BIG integer/rational caller, actual
   first Copy growth refusal. Source publishes caller sign/free/failedRef;
   current work wrapper leaves old caller. Observe this bounded actual RED
   for Floor and Ceil, both output types and scratch policies. Self numerator
   Copy is already no-action: do not invent a refusal for that alias.
2. Noninteger MachineDiv with wide quotient, actual first output/scratch growth
   faults and post-quotient sign-correction promotion faults. Use finite
   reservation sweeps/real storage preparation and classify actual source
   prefixes: old until publication, caller quotient then corrected value or
   owned failedRef, and exact unchanged denominator until successful integer
   route. No arbitrary mutation allowances or imagined future OOM equivalence.
3. All source output routes and aliases: separate caller, caller==input,
   integer output==num, integer output==den, actual same component identities.
   Save source sign before division; denominator mutation invalidating the raw
   input is an observed finite publication followed by ordinary read boundary,
   not a reason to restore the original numeral. Lexical Del/explicit Reset
   cleanup remains an action, never hidden arithmetic repair.
4. Canonical finite signed/small/BIG fractions checked with independent test-only
   big.Int QuoRem rounding; integer MIN/width edges; raw negative/unreduced
   nonzero denominator fixtures with source sign-only correction; old BIG output
   denominator storage freed exactly at final reset.
5. Immediate complete qopsAuditLiveOwnership with all caller/components/cache
   roots BEFORE faults are disabled, formatting or teardown. Sweep completion
   must be observed, untouched nondestination operands compared exactly, then
   real cleanup/Close refunds actual graph. No common audit changes.
6. Actual source integer local in PrevPowerOfTwo: success/failure cleanup,
   both policies, retained plateau and warmed zero allocation; real service
   value-return routes and source TST1/TST2 continue on final native/debug suites.
   A cache-only warmed workload may report timing and actual graph separately.

Registered src/test/mpq.cpp tst_prev_power_2 has exactly16 ENSURE executions,
already in TestRationalSourcePrevPowerOfTwo. Registered rational.cpp TST1
contains11 floor/ceil assertions (36–42,66,76–79), TST2 contains5
(128–133), already routed through the full facade tests. New fault/alias/raw
fixtures are independent strengthening, not more source assertions. Preserve
all existing counts and actual statement lifetimes; no full ostream or TRACE
claim is needed for this arithmetic family.

## Complete adjacent setter/inspection/parser audit

The following findings are real but intentionally outside the recommended
rounding production scope until their full contracts are separately reviewed.

| Adjacent source family | Evidence and current Go behavior | Action / dependency boundary |
| --- | --- | --- |
| Copy rational, raw components | mpq.h666–669 sequential numerator Copy then denominator Copy; current mpq.go Copy is now direct since Power/publication chunk. | Already corrected; no imaginary transaction debt. |
| Integral scalar / MPZ / limb setters | mpq.h673–676,712–745 numerator setter then reset_denominator; current SetInt64,SetUint64,SetInteger,SetLimbs are direct. | Already source ordered subject inherited physical setter adapters. Stale mpq_inspect.go comment claiming allocation preparation before publication is historical prose, not another staged implementation. |
| Numerator/denominator extraction | mpq.h191–197 actual component Copy then reset result denominator for rational overload; Go Numerator/Denominator and rational counterparts route those same methods. | Already direct, including aliases. Do not infer a defect merely from an old atomic fixture that currently passes due real capacity reuse. |
| Pair scalar setter | mpq.h689–693 asserts d!=0 only debug, writes caller.num then caller.den then normalize. Current SetFraction has universal d0 check and whole-rational work/Move. | Actual staged/diagnostic debt. Signed32 pair overload mpq.h678–687 also has signed-negative flip and INT_MIN unary-negation UB to classify. Not bundled into rounding; arena owns related physical/output helpers. |
| MPZ-pair setter | mpq.h696–708 tests denominator sign BEFORE numerator/denominator Copies, then negates both if original sign negative, then normalize. Current SetParts stages and reads d.val after copies into work. | Actual caller publication/alias debt; source0den has no initial assertion and normalize can produce finite +/-1/0 or 0/0 endangered Div. Requires complete field-alias/error/root/domain review, no blanket initial Div0 guard. |
| String parser | mpq.cpp186–276 Reset actual caller numerator immediately; source numerator/denominator digit actions, old-denominator scientific quirk and three source exceptions. Current mpq_parse.go stages work with copied old denominator. | Complete grammar/action algorithms already present; caller source exceptions/OOM prefixes remain transaction debt. New direct parser must reuse public setters/Power/Add/Div/Mul rather than private duplicates. Separate full parser packet after pair setters; preserveuint64 exponent wrap and actual mask-wrap precedence. |
| Scalar predicates, Hash, Bitsize, StorageSize, parity/power query | mpq.h predicates and inspection source branches match current mpq_inspect.go, subject ordinary safe read boundary. | No staged output to remove; source radix/statistics quirks retained. |
| Uint64/Int64 getters | mpq.h776/778 has only SASSERT(rational range), then actual inherited integer getter. Go universally uses IsUint64/IsInt64 then ErrConversion. Finite7/2 should return numerator7 in release, debug rejects. Inline negative Uint64 cast is also actual source. | Real diagnostic-profile debt. Integer get_int64 mpz.cpp1335–1360 and big_abs_to_uint64 diagnostic/host cast/limb truncation must be audited first; no broad integer getter guard change or private conversion algorithm in rounding. |
| Unsigned/Int32 wrapper projections | rational.h107–122 asserts wrapper range only debug, then inherited64-bit getter and actual narrowing. Go universally rejects. | Real layered diagnostic debt; inclusive predicate boundaries themselves are already right. Cast profile/unsafe signed-negation and inherited getter guards require full selected-profile audit before implementation. |
| Float64 | mpq.cpp297–305 converts num then den with inherited actualUINT_MAX radix; only SASSERT(d>0), then n/d. Go universally returns ErrRationalFloatDomain for negative/NaN converted denominator. Raw1/-2 release returns-0.5; NaN denominator release yieldsNaN. | Real debug-only assertion debt. Preserve numerator→den conversion order, IEEE results and existing radix defect; ordinary den0 read guard separate. No integer Float64 rewrite justified. This finite, read-only diagnostic closure can follow rounding after parent reviews exact source category. |
| Inc/Dec | mpq.h313/315 inc adds scalar1, dec adds scalar-1. Current inspection forwards single AddInt. | Additive dependencies now direct; no duplicate algorithms needed. |

The getter predicates are intentionally different from getters: IsUint64,
IsInt64,IsUnsigned,IsInt32 must continue to report false for source fractional
inputs in every profile. Debug getter assertions occur before the inherited
integer accessor, while release skips the predicate expression. Existing
Float64 domain tests in TestRationalInspectionStatisticsFloatAndDomains and
TestRationalInspectionFloatExceptionalBranches assert universal errors; those
are likely future named dependent RED, not approved edits now. No current
getter production change is proposed until complete selected integer access
and source host conversion boundaries are reviewed with the parent/arena.

## Ownership and remaining completion boundary

This recommendation changes only a redundant owner-local transaction. Actual
Integer MachineDiv/Copy/Add/Sub scratch and physical storage are borrowed from
the existing RationalManager and remain its single source algorithms. Source
normal Del order, cached TMP ownership, failedRef cleanup-only guards and
consumed/retired metadata are retained, with no manual numeric token fabrication.
The arena's memory-backed MPZ/Natural/constructor/destructor/output integration
is separate and serialized; no claims that Budget equals source memory/TLS
history. The new family can be source-action correct before that topology lands.

Whole MPQ headers remain partial: pair setters, parser transaction publication,
layered scalar getter diagnostics and other documented source profiles/streams
are actual remaining obligations. This is an implementation proposal, not an
observed RED or GREEN. Production/test authorization and final isolated base
come from parent review; parent owns central evidence/no-generics audit after
metadata overlay. Full solver, AST polynomial consumers and proof obligations
remain in the continuing goal.
