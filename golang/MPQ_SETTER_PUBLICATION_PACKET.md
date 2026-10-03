# Rational paired/scalar setter publication proposal

Read-only source audit and NEW tests based on committed `e9f911f`. This tests-first checkpoint changes no existing Go production or test bodies.
The implementation scope below remains a proposal for review. Concrete Go only; no generics, unsafe, numerical replay, scratch
repair or production math/big. Current native/checked-debug/vet scope retains
visible physical-backend baseline RED; race/platform gates remain deferred.

## Complete selected source/API inventory

Read mpq.h48–77 normalize and all setters/factories at180–182,664–747,
current mpq.go and mpq_inspect.go, RationalService setters, registered
src/test/mpq.cpp bodies and matching manager/service failure assertions.

| Source overload | Current concrete route | Actual action debt |
| --- | --- | --- |
| set(target,mpq)666–669 | Copy | Already NUM then DEN direct; source invalid-den ordinary-read boundary remains separate. |
| set(a,int)673–676 | SetInt64 for currently representable int32 input | Direct NUM then reset_denominator; narrower entry currently rejects den0 initializer destinations. Explicit SetInt32 can bind actual Integer.SetInt32. |
| set(a,int,int)678–687 | RationalService.SetFractionSigned | Private work transaction; raw manager lacks named signed32 pair overload. Source d0 assertion is debug only. |
| set(a,int64,uint64)689–694 | SetFraction | Private work transaction plus universal d0 early error; source writes NUM then DEN then normalize. |
| set(a,mpz,mpz)696–708 | SetParts | Private work, universal d0 early error, negative-DEN branch read after copies instead of before actual caller actions. |
| set(a,unsigned)712–715 | Service.SetUnsigned via SetInt64 | Existing valid unsigned32 value route has correct INT_MAX inclusive shape. Raw SetUint32 should use actual Integer.SetUint32, distinct from uint64 strict threshold. |
| set(a,char const*)719 | SetString | Direct source caller actions committed e9f911f; no parser edit proposed. |
| set(a,int64)723–726 | SetInt64 | Direct NUM setter then DEN reset, ordinary entry invalid-den restriction. |
| set(a,uint64)730–733 | SetUint64 | Direct source NUM setter then DEN reset; entry restriction. |
| set(a,mpz)735–738 | SetInteger | Direct NUM Copy then DEN reset; entry restriction. Actual operand aliases are reread only where source uses them. |
| set(a,digits)742–745 | SetLimbs | Direct source integer digit setter then DEN reset; entry restriction and Integer backend physical guards remain inherited. |
| dup749–752 | Existing independent owning service copies | Not a new raw manager Dup claim in this family; actual rational Copy/value-return boundaries audited separately. |

No float setter exists in this source group. Wrapper double routes remain
unreachable adaptations, not a substitute fraction overload. Scalar int32 MIN
negation in negative-den signed-pair normalization has source signed unary UB;
existing widened service behavior defines intended magnitude explicitly. It
must remain a documented adapter, not a claimed observed C++ result.

## Actual source algorithms and aliases

SetParts first inspects is_neg(d), before either NUM or DEN setter. That branch
is latched across mutations. It then Copies actual NUM from n, Copies actual DEN
from d, conditionally Negs NUM then DEN, and calls the one existing normalize.
Every source reference is read at that action; no pre-copied pair or later sign
recheck may replace it. Self Copy stays the actual Integer no-op. Failed BIG Copy
publishes its actual sign/failedRef and leaves prior completed component actions.

Concrete fully defined crossed-input examples:

| Old raw a | n / d references | Exact source result | Current staged result |
| --- | --- | --- | --- |
| 5/-3 | a.den / a.num | -1/-1 | -3/5 |
| -5/-3 | a.den / a.num | 1/1 | 3/5 |
| 5/3 | a.den / a.num | 1/1 | 3/5 |
| -5/3 | a.den / a.num | -1/-1 | -3/5 |

These outcomes use real post-NUM input rereads and normalize GCD/division. They
are raw field expectations, not a mathematical rational constructor oracle.
Negative denominator can survive a branch chosen before the aliased numerator
write; never normalize signs a second time or reject that finite raw shape.

The signed32 scalar pair performs source debug d!=0 assertion, then optional
local n=-n followed by d=-d, then signed NUM setter, signed DEN setter, normalize.
The int64/uint64 pair asserts d!=0 only in debug, then its two distinct scalar
setters and normalize. A NUM publication is never withheld because DEN later
refuses allocation. Scalar one-field overloads set NUM then reset_denominator;
DEN does not change if NUM fails. reset_denominator Del occurs before its scalar1
publication, including actual BIG old-DEN destruction. Do not rewrite Integer
setter/Copy/normalize algorithms or their actual retained source storage.

## Zero denominator and initializer boundaries

SetParts has NO source d0 assertion. With nonzero supplied NUM and DEN0, its
Copies and normalize produce finite ±1/0 (GCD abs NUM, NUM division then DEN
zero division by nonzero). Supplied0/0 reaches actual Integer.Div0 only after
both source component Copies. A typed divide-by-zero guard belongs at that
endangered arithmetic, not at entry. Ordinary rational reads/arithmetic still
reject resulting den0; real Del/explicit Reset must retain owning cleanup.

For scalar pair d0, debug ErrIntegerAssertion belongs before first NUM action;
release nonzero/0 normalizes to±1/0, release0/0 reaches actual Integer.Div0 after
publication. This is distinct from parser's explicit zero-den slash exception.
A source finite invalid result must be returned as raw owning fields without a
fabricated post-action initializer error or rollback.

Propose explicit setter initializer/reinitializer entry identical in principle
to reviewed SetString: safe manager/closed/destination validation, ordinary
Integer NUM then DEN handle check (not checkCleanup), accepting inline den0 and
raw Go zero as the explicit Go initialization adaptation. Scalar/pair setters
then perform their actual source actions. Ordinary arithmetic, Copy source-read
policy and service checks are not globally weakened. A failed owning descriptor
still fails ordinary entry checks; no hidden repair or lazy initialization.
At the SetParts sign read validate d ordinarily; Copy validates n at its actual
first use and d again at its later Copy, so source-prefix diagnostics do not
require eager snapshots. Later stale alias reads are explicit lifetime guards,
not source-valid new arithmetic cases.

## Proposed owned production scope after review

- mpnum/mpq.go: SetFraction, SetParts, narrowly shared explicit setter-entry
  validation, new SetFractionSigned(a *Rational, n, d int32) error and SetInt32 /
  SetUint32 typed scalar overloads. Existing SetInt64 uses same entry only;
  normalize stays the existing single algorithm and scratch policy.
- mpnum/mpq_inspect.go: only existing SetInteger/SetUint64/SetLimbs entry binding
  and removal of stale comments claiming Integer allocation-before-publication.
- mpnum/rational_service.go: ONLY SetFractionSigned replacing private work with
  the new single raw manager signed-pair route after existing provenance lock/
  checks. Other service setter wrappers already call the actual manager and
  remain unchanged; service invalid-read/entry policy is separately documented.
- NEW mpnum/mp_setter_publication_test.go; existing tests held until exact named
  RED and parent source-action approval. This NEW packet only.

No existing production has been edited. Arena owns Natural/Integer physical
storage and teardown; no backend/core/allocator change or scratch recovery.
A staged private Copy alternative is prohibited: public setters must be the
actual shared paths used by parser, service, polynomial and numeral consumers.

## Tests-first action plan and source evidence

Initial NEW three groups expose crossed aliases, SetParts d0, and public scalar
NUM publication before an actual refused DEN allocation. Source independent
numeral root audits occur before fault disable/rendering, using committed
actual physical backing/owner helper; no Words-per-numeral fictional charge.
The scalar refusal fixture fills actual cells (Live+Retired<Slots), arms first
growth failure, calls SetFraction(1,UINT64_MAX), and expects completed NUM1 with
old DEN19. Typed Memout requires actual Runtime.OutOfMemory; parser/source
categories are never substituted. Caller and bank are destroyed through real
owner callbacks and Budget returns0.

Before further production, expand NEW tests for complete input identities:
external n/d, each component source, crossed components, n==d, full self-pair,
BIG signs/shape/copy-capacity, and valid same/different component aliases. Use
an independently simulated sequential source raw action model for aliases and
math/big.Rat only for genuinely nondefective independent constructor inputs.
Fault sweeps must compare paired action states (CopyNUM→CopyDEN→NegNUM→NegDEN→
normalizeNUM→normalizeDEN) and exact owning failedRef at actual Copy-growth,
including immediate caches and source physical roots before cleanup. Scalar
pair shapes INT_MAX versus uint64 threshold, signed MIN adapter, typed source
assertion placement, finite den0/0/0 and explicit initializer routes get separate
fixtures. Warm pooled zero-allocation and Direct actual allocation events with
stable real object/Budget graphs follow current source allocator profiles.
Service exact value/compound/provenance routes are exercised without unowned
raw handle transfer; no new global wrapper completion claim.

Registered mpq.cpp uses signed-pair setup in tst0/bug1/bug2/tst2/set_str_bug and
int64/uint64 setup in16 tst_prev_power_2 assertions. Existing source counts
remain unchanged: their setup routing is refined, not new ENSURE executions.
Alias/fault/domain fixtures are added independent/source-derived expectations;
zero assertion inflation. Full printed ostream/TRACE traces remain open.

The first compiler run caught test-helper argument shapes (IntegerManager and
[]*Int), corrected before behavioral execution; not credited as source RED.
Actual behavioral RED is terminal on committed e9f911f plus only the NEW test:
native94899 (0.005s), checked/debug25865 (0.007s), all three groups fail with
source-action distinctions in both scratch policies. Logs are
`.cache/setter-publication-red-native.log` and
`.cache/setter-publication-red-debug.log`; snapshot
`.cache/release/mpq-setter-red-e9f911f`. Crossed aliases currently retain staged
3/5 variants, zero-den inputs are rejected with unchanged17/13, and the armed
scalar DEN refusal leaves old NUM17 rather than source completedNUM1. Immediate
physical/owning graph and real teardown/refund checks pass in every fixture.
No production or existing test edit has occurred. These are ordinary bounded
behavioral failures, not missing-API or exception-format credit.
