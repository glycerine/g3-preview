# Rational parser direct-publication proposal

Read-only complete source audit based on committed `0c9d695`. Rounding is
committed8b5dc06 and its three Go hashes are unchanged. Only this NEW packet
is written now. Production/test/metadata/Git edits require parent scope review.
Concrete Go only: no generics, unsafe, cgo, production math/big, numerical
replay, scratch repair or private replacement parsing/arithmetic algorithms.
Current user gates are native, checked/debug, vet and parent source/evidence
no-generics audit; race and platform gates remain deferred.

## Full selected source and actual debt

Read complete `src/util/mpq.cpp:186–276`, mpq.h48–77normalize,
696–708pair setter,666–669Copy, and actual current mpq_parse.go/mpq.go plus
mp_parse_test.go and registered mpq.cpp tests. The entire selected string
setter already has its permissive grammar and arithmetic routes implemented.
The debt is top-level transaction publication, not a new parser or grammar.

Source begins Reset(a.num), then creates inline ten10 and scoped integer tmp.
Go instead constructs a new rational work, Copies the old caller denominator
into work BEFORE resetting work.num, executes all grammar/actions on work,
and Moves only after success. This adds a fallible denominator copy that source
never performs and hides all source caller mutation on exceptions/OOM. It also
retains the original caller BIG numerator until the final Move, where source
frees it before reading any input bytes. An old BIG denominator must remain
physically the same actual source payload until a branch mutates it, not an
extra copied payload before numeral reset.

Proposed production scope: existing `mpnum/mpq_parse.go` ONLY. SetString retains
its existing entry validation policy, Resets actual caller.num, and passes the
actual caller into the single existing suffix helper. Remove only the work
construction/old-denominator Copy/work drop/final Move and update stale atomic
publication comments. The suffix helper may rename its parameter from work
to out/a for clarity but retains its algorithm. ParseError/Reason/sentinels and
public SetString signature stay unchanged. No missing-API RED is appropriate.
No mpz parser/core, pair setter, constructor/destructor, output helper, service
production or arena/backend edit is proposed.

## Exact full grammar and action inventory to preserve

| Phase / condition | Actual source actions and current faithful implementation |
| --- | --- |
| Entry | Reset actual numerator before any byte inspection; source inline ten10 and scoped tmp0. No denominator Copy or reset yet. |
| Leading/sign | Skip only ASCII space0x20 for sign detection. Remember negative only if the first remaining byte is '-'; sign byte is subsequently ignored by digit scanning. Tabs/newlines/Unicode and later signs do not alter this flag. |
| Integer bytes | Until NUL/end, '/', '.', 'e' or 'E': ignore nondigits, each ASCII digit calls integer Mul(num,10,tmp), then Add(tmp,digit,num). No strict whole-token grammar or strconv substitution. |
| No suffix | reset_denominator caller only after integer bytes; final optional Neg(num), then normalize caller. Empty/junk parses0. |
| '/' or '.' | Create scoped tmp2, set1; Reset actual den before suffix digit scanning. Every ASCII digit calls Mul(den,10,tmp), Add(tmp,digit,den). For decimal only, Mul(tmp2,10,tmp2) follows that denominator digit publication. Ignore all other bytes except e/E/NUL; '/' inside this scan is ignored. |
| Scientific marker | Slash suffix plus e/E throws mixed-notation BEFORE exponent reading or zero-denominator check. Otherwise advance past marker; exponent negative only for immediately following '-'; all later nondigits ignored except '/' throws mixed-notation. |
| Exponent digits | uint64 wrap on exp=10*exp+digit; do not replace with overflow rejection or signed conversion. Source<=9 SASSERTs are already implied by the ASCII digit condition, not a manufactured behavioral failure. |
| Non-slash completion | If wrapped exp>UINT_MAX, throw exponent-too-big BEFORE creating scoped b. Then create scoped b; when decimal digits existed, SetParts(b,current den,tmp2), integer Set caller.den1 (not reset_denominator), then actual public Add(caller,b,caller). |
| Nonzero exponent | Create scoped exp then scoped ten rational, Set ten10, call actual public Power(ten,exp,expTmp), then public Div or Mul on actual caller depending remembered exponent sign. Source first-growth failures before actual mask wrap retain precedence. No giant unbounded test or early power cap. |
| Slash zero | After mixed-notation checks, if actual caller.den is zero throw division-by-zero parser category. This is a source exception, not normalize's later integer unsafe division. |
| Final | Suffix scoped locals die before outer optional numerator Neg; then actual normalize caller; outer scoped tmp dies at function exit. No successful or exceptional final whole-rational rollback. |

Embedded NUL ends every loop; bytes beyond it cannot create later markers,
exceptions or digits. Go strings are borrowed only for the call, never retained
in numeral storage. Empty Go string models an empty C string, not nullptr.
The existing numeralByte helper provides the finite host buffer boundary.
The selected source char comparisons use only ASCII digit/sign/marker values;
signed high-byte behavior does not broaden their acceptance.

## Exception publication and lexical ownership

From valid old-17/13, before normalization/sign adjustment:

| Input | Exact actual caller state at source exception |
| --- | --- |
| 1/2e0 or1/2E | num1,den2; mixed notation before exponent handling. |
| 1/0e0 | num1,den0; mixed notation wins over slash-zero exception. |
| 1e2/3 or1e/3 | num1,old den13; mixed notation while exponent scan. |
| 1e999999999999/0 | num1,old den13; mixed notation precedes too-big check. |
| 1e4294967296 | num1,old den13; wrapped exponent too big before b. |
| 1.25e4294967296 | num1,den25; decimal suffix digits published, but b/den1/Add not reached. |
| 1/ or1/00 | num1,den0; parser zero-denominator exception after accumulation. |
| / | num0,den0; same exception, no hidden normalization or sign. |

A leading negative flag applies only after suffix success, so '-1/2e0' still
has positive raw numerator1 at the mixed-notation error. Errors from digit
Mul/Add, denominator scaling, public Power/Mul/Div or final normalize retain
actual earlier fields and owning failedRef/TMP prefixes. They remain their
original allocation/arithmetic categories; do not relabel them ParseError.

Keep actual lexical locals and reverse destructor order: outer tmp at final
return; suffix tmp2 after b; within exp>0, ten then exp before b/tmp2. Existing
Go defer order already matches these scopes. Deleting source temporaries on
typed failure is the existing explicit Go exception-cleanup adapter; never
drop caller fields or silently repair missing manager scratch. Their real
physical roots and source completed allocation/cache histories remain visible
in immediate audits.

SetParts(b,caller.den,tmp2) is an actual public dependency, not to duplicate.
Its current general pair-setter staging is a separately documented debt, but
this parser call always has a fresh private b0/1, digit-built nonnegative
numerator and positive10^digits denominator, with no input/output component
aliases. The staged private pair has the same persistent payload allocations
on ordinary success/failure and its cleanup handles partial copied fields;
no source caller-prefix distinction is created merely by changing its private
scalar address. Keep this single public route and explicitly retain the pair
setter's remaining full-contract debt rather than claiming it corrected here.
If tests reveal a real private-local cleanup/allocation discrepancy, report
that concrete dependency before widening production scope.

### Invalid denominator / failed-reference boundary

Direct parser exceptions can leave finite source raw1/0 or0/0. Ordinary Go
reads/arithmetic retain ErrUninitializedRational; existing Del/Reset cleanup
validates actual integer ownership and accepts den0. Caller may inspect fields
in tests and explicitly Reset/Del, not render invalid rationals or pretend the
parser restored the old value. FailedRef cleanup remains cleanup-only.

Entry SetString currently uses check and rejects raw Go zero or prior den0.
Source can reparse some prior zero-denominator outputs with e.g plain '2',
while a no-dot scientific input can carry den0 into actual later arithmetic.
Retaining the current entry initialized-numeral policy is an explicit Go API
boundary, not full equivalence for these source-finite reparsing continuations.
Small raw Go zero and parser-produced0/0 have identical scalar representation;
there is no existing initialization sidecar that can distinguish them. This
packet proposes no sentinel/lazy initialization, new record field or blanket
checkCleanup arithmetic guard. If parent chooses to extend explicit parser
reinitialization, that requires a separately reviewed entry contract/test
change; existing raw-zero initializer fixture cannot be silently weakened.

## Tests FIRST and small checkpoint plan

NEW `mpnum/mp_parse_publication_test.go` first, actual behavioral RED on isolated
0c9d695 plus that file. Preserve all old Go until RED; after approval production
mpq_parse.go only. Existing test changes require named dependent RED and
parent source review, not broad atomicity removal.

1. All three real source exceptions with exact raw fields above, both scratch
   policies, old SMALL/BIG numerator and denominator, remembered negative sign
   not yet applied; immediate full owned-root audit and typed categories.
   Cases with den0 assert ordinary read guard, then real Del/explicit Reset.
2. Old BIG denominator with fault0 and short no-allocation mixed-notation path
   proves source has no preparatory den Copy. Source Reset frees old numerator
   first and reaches mixed exception; staged baseline may instead allocate
   while copying denominator and return injected OOM. Use real backing demand,
   not fabricated allocator callbacks or assuming every Allocate reserves.
3. Integer/fraction/decimal/scientific digit fault prefixes and source sign timing:
   real bounded reservation sweeps with fresh owner per fault; decode actual
   limbs before disabling faults/rendering; permit only completed source digit/
   arithmetic component values and actual owner failedRef, never any mutation.
   Observe sweep success boundaries. Audit all caller/cache roots before
   teardown, no success retry after failed arithmetic and no scratch repair.
4. Stateful scientific sequences retain original actual denominator until
   source resets/scales it; confirm expuint64 wrap, UINT_MAX comparison,
   immediate versus later exponent '-', NUL, ignored nondigits/signed bytes,
   slash/mixed category precedence. Current successful grammar fixtures are
   controls, not rewritten expectations. Independent test-only mathematical
   oracle applies to canonical finite cases; quirks use source action models.
5. Public Power high-mask branch with armed first real growth refusal remains
   bounded and wins before wrap; retain already-source-faithful Power loop,
   public Add/Mul/Div/normalize. Never test enormous exponent by unbounded
   allocation or replace it with an entry-domain guard.
6. Real RationalService SetString forwards under actual provenance check and
   manager lock, source value-return/string constructor cleanup, lexical local
   refund/retained plateau and existing warm-zeroallocation controls. Service
   production stays unchanged; any exposed old service fixture gets separate
   named RED review. Scope constructors continue actual source lifetime routes.

Likely old named dependent debts in mp_parse_test.go:
TestParserExceptionCategoriesAndFailurePublication (unchanged old-17/13 after
all parser exceptions/highmask faults) and TestParserBudgetFailureCleanup
(atomic rational result). Only rational parser branch assertions are candidates;
IntegerManager.SetString has its own transaction debt and remains unchanged.
The latter group mixes both parsers, so integer failures/success oracles must
not be weakened during rational correction. TestParserWarmLifetimeAndInitialization
raw-zero/closed/borrowed-string checks stay unchanged under proposed entry policy.

Registered src/test/mpq.cpp set_str_bug has exactly3ENSURE cases ('1.0','1.1',
'1/3'), already ported through production SetString. Preserve its real repeated
caller state and source scope routes. Existing source integer string/assertion
fixtures, facade TST1/TST2 and literal inventory remain independent consumers,
not new source credit. New prefix/domain fixtures add no ENSURE executions;
full persistent ostream/TRACE output remains open.

## Exact owned scope and completion boundary

Proposed future owned files: NEW mpnum/mp_parse_publication_test.go, existing
mpnum/mpq_parse.go only the caller transaction/parameter comments, and this
packet. Old mp_parse_test.go changes held until actual named RED; no private
normalization, public pair setter, scalar conversion, MPZ/Natural/backend,
constructor/cleanup, service, metadata or Git edits. Parent owns checkpoints
and audits; arena owns physical MPZ and transitive numerical owner helpers.

This closes the selected valid-initialized caller parser publication/actions
and its actual source exception categories, not all MPQ setter contracts.
Pair-setter, initial-invalid-den reparsing, scalar getter diagnostics, integer
parser publication, TRACE/external streams and source memory/TLS topology
remain explicit obligations. Complete grammar loops already faithful are
preserved, and the full solver/AST/proof/polynomial goal remains open.


## Checkpoint review status

Saved as a proposal checkpoint, without parser production changes. Tests-first
valid-caller transaction removal is approved. The narrower initialized-only
entry policy described above is not a permanent completion boundary: audit
SetString as an explicit initializer/reinitializer using owner and ordinary
Integer handle validation, then the actual source numerator reset. Tests must
first expose source-produced 1/0 and 0/0 destinations reparsed as plain, decimal
and fraction inputs. Raw Go-zero initialization is an explicit adaptation.
Ordinary arithmetic guards remain unchanged. Any existing initialization
expectation needs a named RED and source review before modification; scientific
zero-denominator continuations and public arithmetic dependencies remain open.
