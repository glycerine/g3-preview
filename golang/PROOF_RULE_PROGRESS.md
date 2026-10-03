# Proof and interpolation prerequisite inventory

`proof_rule_matrix.json` records source evidence, not implemented Go proof support.
Every rule, producer, checker branch, translator branch and interpolation method
is **unported**, with empty Go-test lists. Cases and lexical constructor references
do not establish semantic support. Entire function/helper graphs, active conditional
paths, ownership and independent proof/Craig checks still require tests-first work.

The source puts proof rules in `basic_op_kind` after Boolean/interpolation operators
and `LAST_BASIC_OP`. Consequently, its 37 proof IDs are **14 through 50**, with
`PR_UNDEF=14`, `PR_HYPER_RESOLVE=50` and `LAST_BASIC_PR=51`.
See `src/ast/ast.h:1040`. JSON includes both these actual values and a separate
zero-based proof ordinal; the ordinal must never replace a source ID.

`basic_decl_plugin` is implemented inside `src/ast/ast.cpp`, not a separate source
file. Proof sort/argument checks are at `src/ast/ast.cpp:685` and `:698`.
Parameterized declarations handle theory lemmas, quantifier instances and hyper
resolution at `:778`; ordinary declarations start at `:805`. Constructors and
their explicit rule references are inventoried separately from lexical source
call sites. Constructor-name matches can include comments, declarations and
overloads; their producer behavior remains unaudited. Source files referenced
by the ledger carry SHA-256 fingerprints, and every evidence item has a checked
repository-relative path, line and exact trimmed source line.

## Checker obligations and source limitations

`proof_checker::check` at `src/ast/proofs/proof_checker.cpp:98` traverses proof
premises and returns its structural checking result. It accumulates residual
`side_conditions`; it does **not** discharge them. A true result therefore does
not certify the whole refutation. Non-basic families return false at `:121`;
the SPC-family dispatch and seven SPC-rule cases remain under `#if 0`, with no
active basic numeric IDs. The matrix retains this conditional inventory.

| Rule / source branch | Required audit and test evidence |
| --- | --- |
| PR_TRUE | Constructor exists at `src/ast/ast.cpp:2621`, declaration at `:813`, but no check1_basic case; checker default is false at `src/ast/proofs/proof_checker.cpp:829`. Preserve and expose this source behavior during conformance. |
| PR_UNDEF / ASSERTED / GOAL | Checker accepts without fact/provenance validation at `src/ast/proofs/proof_checker.cpp:212`; external premise/root validation is a separate obligation. |
| SYMMETRY / TRANSITIVITY / TRANSITIVITY_STAR / MONOTONICITY | Explicit source TBD checks leave symmetric/transitive/monotone relation properties unresolved at `:250`, `:269`, `:279`, `:311`. Transitivity-star uses endpoint parity, not an independently established ordered path certificate. |
| DISTRIBUTIVITY / REWRITE | Emit equality fact as residual obligation at `:367`, `:408`. |
| REWRITE_STAR | Emits conclusion OR negated premise facts at `:426`. |
| PULL_QUANT / ELIM_UNUSED_VARS / DER | Shapes are checked; actual transformation/binder conditions are source TBD at `:437`, `:470`, `:487`. |
| PUSH_QUANT | The branch checks shapes then falls through to UNREACHABLE/false at `:463`, even after matching shape. Audit this source behavior explicitly. |
| HYPOTHESIS / LEMMA | Hypothesis branch closure is TBD at `:495`; lemma checks discharged complements and its special quoted-disjunction case at `:502`. |
| UNIT_RESOLUTION | Complementary-unit and clause-removal paths at `:546`; multi-literal conclusion membership/weakening requires exact behavior tests. |
| DEF_AXIOM / DEF_INTRO / APPLY_DEF | Boolean/oeq shape acceptance at `:667`, `:677`, `:692`; axiom validity, freshness and definition provenance are not certified by these checks. |
| NNF_POS / NNF_NEG / QUANT_INST | Unconditional true with TBD/TODO at `:715`, `:719`, `:766`; semantics remain unchecked by this source checker. |
| SKOLEMIZE | Polarity/quantifier shape check at `:723`; proper instantiation remains TBD. |
| TH_LEMMA | Non-arithmetic theories dump and accept at `:757`. Arithmetic with no kind parameter or a non-Farkas kind also dumps and accepts at `:1352`, `:1364`. |
| Arithmetic Farkas | `check_arith_literal` at `:1265` handles signs, inequality absolute coefficients, strict Int rounding and strict Real inequalities. `check_arith_proof` at `:1344` normalizes coefficient denominators and requires final rewriting to false. Coefficient cardinality/index safety and every producer parameter path still need audit. |
| HYPER_RESOLVE | Substitution and pivot removal at `:770`; complementarity mismatch logs at `:811` instead of rejecting. Equivalence/quantified equivalence is emitted at `:826`. Parameter counts, binder sorts, positions and residual validity require independent checks. |

Other ordinary equality/propositional branches have their observed matching
conditions summarized per rule in JSON. Their presence is not a completeness
claim; each still needs positive, malformed-premise, wrong-result, ownership and
producer-path tests. Do not strengthen the source checker silently or weaken
the independent oracle to match a discovered source defect.

The existing `src/test/proof_checker.cpp:10` fixture constructs hypotheses `a`
and `not a`, resolves false, then checks a lemma concluding `a OR not a` at
`:28`. Its earlier assigned lemma `a` is replaced before checking. This is a
mandatory source fixture to port once AST/proof construction exists; it is not
currently a Go test or an independent validation of all residual obligations.

## Translator and interpolation obligations

`src/interp/iz3translate.h:58` defines `IZ3_TRANSLATE_FULL`. The direct translator
is an alternative implementation in `iz3translate_direct.cpp`; no separate
`iz3translate_direct.h` exists. Keep the full translator as default.

Both translators first use a successful locality result as an assumption, so
absence of a main-switch rule does not prove a rule always fails. Full locality
and special unit-resolution/lemma dispatch appear at
`src/interp/iz3translate.cpp:1704` and `:1725`. Its nonlocal switch at `:1848`
explicitly translates transitivity, transitivity-star, quantifier-intro/
monotonicity, reflexivity, symmetry, modus-ponens, theory lemmas, hypotheses,
quantifier instances, def axioms, iff-true/false, commutativity and elimination.
Skolem/modus-ponens-oeq/commuted and quantified idioms run outside that switch.
All precise conditions remain pending tests, including local-only iff-false and
unsupported commute consumers.

The theory dispatch at `src/interp/iz3translate.cpp:1906` handles arithmetic and
arrays explicitly; other theory values throw unsupported. Arithmetic branches
include Farkas, triangle equality/bounds, GCD-test conversion, assign-bounds,
Gomory-cut, equality propagation and mystery-rule guessing. Constructor helpers
and coefficient extraction are not validated just because those branches exist.
JSON inventories source `lemma_theory` and `lemma_kind` ordering from
`src/interp/iz3mgr.h:384` and `:399`, including unknown sentinels.

Direct translation at `src/interp/iz3translate_direct.cpp:1530` dispatches unit
resolution/lemma specially, then tries theory-lemma extraction, resolvent
pushing and secondary-prover lemma repair at `:1559`. Its small simple-proof
switch at `:1166` handles only transitivity, monotonicity, reflexivity and symmetry.
Neither lexical PR mentions nor that switch justify a broad direct-support claim.
The JSON includes conditional definitions and lexical throw/catch sites so
unsupported, proof_error, invalid_lemma and helper exceptions stay visible.

`iz3proof_itp.h:41` declares twelve interpolation-rule enum values, while its
interface provides fifteen named method families, with overloaded resolution,
axiom and congruence methods. These are inventoried separately: an enum label is
not proof of an implemented algorithm. The implementation represents nodes as
interpolants, composes mixed/local proof placeholders, and has numerous helper
failure paths. Its constructor forces `weak=false` at
`src/interp/iz3proof_itp.cpp:3045` despite the passed argument; preserve the actual
behavior until a documented source issue is resolved.

Localization and binding require separate branch coverage. The active
`add_quants` at `src/interp/iz3proof_itp.cpp:2946` walks variables in reverse
creation order; an origin inside the A range gets Exists, otherwise Forall.
`interpolate` at `:2968` simplifies, adds quantifiers, and closes remaining free
variables universally. The inherited `iz3mgr::apply_quant` at
`src/interp/iz3mgr.cpp:914` distributes forall over conjunction/exists over
disjunction, drops absent variables, substitutes constrained equalities, and
otherwise creates a binder. The full translator also defines a distinct local
helper at `src/interp/iz3translate.cpp:594`. Audit actual dispatch, fresh names,
skolem provenance, de Bruijn binding, simplification, polarity and result order.

The full translator's outer loop at `src/interp/iz3translate.cpp:2071` scans
skolems, computes each cut, translates proof_error/unsupported exceptions, and
publishes a lemma with interpolants. Solver SAT/UNSAT/UNKNOWN/error statuses are
separate higher-level contracts; this inventory supplies none of those APIs.

## Validation and remaining work

A maintainer-only builder/validator lives under `golang/.cache` and reads source;
ordinary Go tests do not depend on it. Validation checks all 37 unique IDs, exact
14..50 ordering, sentinels, and every recorded evidence path/line/text. JSON
output is deterministic and contains source SHA-256 hashes without timestamps.
This is inventory validation, not semantic tests or a completed proof gate.
The inventory includes 105 source fingerprints and 490 lexical producer evidence
items, plus twelve interpolation enum labels and fifteen method families.

Next implementation work must port source assertions first and add independent
rule/certificate checks, malformed-proof regressions, producer-to-checker tests,
residual-condition discharge, checked-handle lifetime tests and the C6 Craig/
quantifier matrix. Full/direct translator helper graphs and branch conditions
remain unaudited beyond the specific observations above. Nothing in the matrix
is marked implemented, tested, semantically supported or complete.
