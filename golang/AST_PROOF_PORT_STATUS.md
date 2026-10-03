# AST proof manager work in progress

This packet implements the builder/reader bodies inventoried in
AST_PROOF_PACKET.md against committed AST core 07a6182 and SAT storage
0da9cbe. It covers all 44 distinct out-of-line names, their overloads and
inline mode/fact/parent inspection. It does not complete the entire AST manager,
proof checking, solver proof production, diagnostics or interpolation.
The frozen integration contains 21 concrete Go files and 31 named test groups;
parallel marks, dependencies and solver drafts are excluded.

Source method tests observed missing-API RED before initial mode/readers,
Boolean/quantifier/NNF rules, transitivity arrays/star/congruence, and parameter
builders were implemented. Rewrite/unary matcher tests also observed missing
APIs before implementation. A behavioral regression observed RED before fixing
release modus-ponens and reflexivity-transitivity short circuits: release must
not evaluate fact readers used only by source assertions. Source debug performs
those checks before its identity branches.

Current implemented bodies cover proof mode and inline fact/parent inspection;
true/asserted/goal/hypothesis/definition introduction; EQ/OEQ reflexivity,
symmetry, two/three/four/array transitivity and endpoint transitivity-star;
monotonicity and both congruence variants; modus ponens; commutativity,
iff-true/false, lemma/distributivity, quantifier introduction, rewrite/star,
pull/push/eliminate-unused/DER/skolemization, definition axiom/applications,
NNF validator and positive/negative constructors, iff-to-OEQ, and/not-or
elimination; both unit-resolution constructors; quantifier instantiation and
theory-lemma parameter builders, rewrite extraction and all three unary matcher
overloads. Rewrite's VERIFY checks EQ/IFF in every profile; OEQ is excluded.
Failure reaches the selected release compiler's builtin-unreachable boundary
and is reported as a typed error; diagnostic stream transport remains open.
Source fixed goal parent-count and the protected
three-input helper's actual two-input forwarding remain explicit fixtures.
Disabled-mode checks are per source body rather than imposed on every builder.
NNF validation executes even though the source ignores its false result.

The concrete lexical proof argument buffer stores its first sixteen scalar
handles directly in an inline array. It does not store a self-referential slice.
A runtime allocation regression first reported one host allocation; after the
correction, AllocsPerRun(100) reports zero. Compiler escape inspection confirms
no proof buffer moved to the heap. Overflow doubles, reserves the full new array
while old backing remains charged, and refunds lexical scratch with the injector
armed. Immediate failure-prefix tests cover inline16, spill32 and spill64.
Parameter scratch uses source 2-to-(3n+1)/2 backing growth, concrete owned
parameter records and ascending destruction; declaration registration owns the
real retained AST/rational parameter payloads. A new theory-lemma fault sweep
uses a rational with two large integer cells/payloads. Before rendering or
cleanup, every live parameter record must be reachable from the caller input or
an actual live info record, and numeral live-cell/payload counts must equal the
count implied by those owners. Both native and checked/debug runs pass; this
checks actual ownership counts rather than every private numeric cell identity.

An independent three-variable Boolean oracle covers all ordered three-literal
clauses and all nonempty removal subsets drawn from each clause: 1,116 cases,
with all eight truth assignments checked per case. The test also verifies the
actual registered conclusion including duplicate/order preservation and the
empty/singleton/disjunction selection. This validates the builders; it does not
execute the separate source proof_checker.

Nineteen named non-hyper proof groups have passed native and checked/debug
runs against the available core prerequisite. Subsequent review tests first
observed behavioral RED for five source mismatches: modus-ponens reflexivity
before release fact access; unit-resolution lexical scratch destruction;
count-two complement short-circuit; canonical TRUE/FALSE complement matching;
and canonical FALSE diagnostics. These are corrected. The strict lexical peak
regression observed RED and GREEN in release; the checked/debug fixture allows
its larger core transient allocation peak and checks complete scratch refund.
The isolated .cache/review/proof-lexical-corrected snapshot excludes parallel
hyper-resolution missing-API tests. These are development checks, not final
frozen packet gates.

The quantifier-instantiation reader now uses real nonowning output storage and
preserves conclusion assignment before the debug empty-output check, append
prefixes on failure, and unchanged outputs for unmatched proof kinds. Its native
and checked/debug regression passes; output storage does not retain AST nodes.

Hyper-resolution construction and append-only decoding are implemented with
real reference outputs, position/substitution owners and the otherwise-unused
fact and Proof-sort temporary vectors. Twelve additional tests cover actual
parameters, disabled mode, cache arity, signed position conversion, malformed
trailing pairs, source diagnostic order, ascending callbacks and every injected
builder/reader allocation prefix. Premise INC occurs before growth; conclusion
assignment precedes outer growth; position append precedes the next substitution.
The outer vector grows before allocating the Go stable reference record, an
explicit representation adapter; no source stack temporary is charged as a
fictional heap allocation. All physical owner/backing storage is charged.

The final coordinator integration snapshot is
.cache/release/proof-integration-0da9cbe, based on HEAD 0da9cbe plus only these
21 Go files. Its frozen byte inventory is .cache/proof-integration-files.json.
Final gate results are recorded below when terminal.

Still open: source TRACE/printer-backed diagnostics and compiler-unreachable
transport, the real proof_checker and its registered one-VERIFY test, solver
proof production and all interpolation. Builder tests and the independent
Boolean oracle do not satisfy the separate checker obligation. Whole-unit
method/ownership audit flags remain false.

Final integration gates passed: full native (root2.494s, mpnum22.654s),
full checked/debug (root2.556s, mpnum23.093s), all31 proof groups under
checked/debug race (1.784s), all31 explicit RUN/PASS events in actual
checked/debug JavaScript, vet, and WASI compilation. WASI execution is not
claimed. Source/no-generics evidence is recorded by the coordinator audit.
The source ast.h/.cpp units remain partial; no API or whole-unit completion
claim is added. The Boolean oracle checks1,116 cases against8 assignments.
