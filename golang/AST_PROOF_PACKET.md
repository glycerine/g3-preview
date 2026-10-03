# AST proof construction and inspection: source contract packet

Coordinator-owned audit for the next implementation packet. No proof builder,
proof checker, solver proof production, or interpolation completion is claimed.
Current AST core drafts are reviewed separately and are not a frozen prerequisite.

Read source: ast.h:2080–2212 and ast.cpp:2590–3224, including every executable
proof construction/inspection body. Registered test/proof_checker.cpp has one
VERIFY through the actual proof_checker; merely constructing its proof graph
cannot satisfy that test. The complete checker and interpolation remain required.

## Complete builder and reader scope

- mk_proof
- mk_true_proof
- mk_asserted
- mk_goal
- mk_modus_ponens
- mk_reflexivity
- mk_oeq_reflexivity
- mk_commutativity
- mk_iff_true
- mk_iff_false
- mk_symmetry
- mk_transitivity
- mk_monotonicity
- mk_congruence
- mk_oeq_congruence
- mk_quant_intro
- mk_oeq_quant_intro
- mk_distributivity
- mk_rewrite
- mk_oeq_rewrite
- mk_rewrite_star
- mk_pull_quant
- mk_push_quant
- mk_elim_unused_vars
- mk_der
- mk_quant_inst
- is_quant_inst
- is_rewrite
- mk_def_axiom
- mk_unit_resolution
- mk_hypothesis
- mk_lemma
- mk_def_intro
- mk_apply_defs
- mk_iff_oeq
- check_nnf_proof_parents
- mk_nnf_pos
- mk_nnf_neg
- mk_skolemization
- mk_and_elim
- mk_not_or_elim
- mk_th_lemma
- mk_hyper_resolve
- is_hyper_resolve

Also include proof mode readers/toggle, is_proof (App and exact Proof sort),
undef identity, all inline kind predicates and unary matchers, has_fact,
get_fact, get_num_parents, and get_parent. All overloads of the listed methods
remain in scope, including the two distinct unit-resolution algorithms.

## Source distinctions to test before translating

- mk_proof returns null when disabled. Its three-expression overload creates
  three local entries but passes count TWO. Preserve this finite legacy route.
  It does not license changing the ordinary three-argument mk_app overload.
- Disabled mode is checked per body. Reflexivity, hypothesis, symmetry, binary
  transitivity, monotonicity, congruence, unit resolution and hyper-resolution
  directly call mk_app; do not add one universal proof-disabled shortcut.
  mk_true_proof builds true before calling the guarded helper. Assertions in
  mk_asserted/mk_goal/def_intro precede that helper even when mode is disabled.
- Null-parent early returns, reflexivity elimination, double-symmetry elimination
  and OEQ/EQ selection in transitivity preserve exact identity and branch order.
  Array transitivity folds left. The endpoint overload returns null at zero and
  its sole input at one without constructing an endpoint fact.
- PR_GOAL declaration is cached with TWO parents, while mk_goal supplies one
  fact. This source inconsistency must follow real application validation rather
  than an invented successful goal node. Fixed declaration caches and hyper-resolve
  first-count caching are source contracts, not corrected dynamic signatures.
- has_fact examines only the last argument sort; parent count follows that test.
  A generic proof app can have no fact. get_parent converts the requested arg
  to App; genuine out-of-bounds/type-undefined continuations get typed guards at
  their actual access, separate from source debug assertions.
- In inferred unit resolution, two complementary facts produce false directly.
  Otherwise scan original OR literals in order against parents1..N-1. Retain
  duplicate unmatched literals and choose false/singleton/or by remaining size.
  Debug found.setx and continue run inside DEBUG_CODE, so all complementary
  parents are marked in debug; release breaks on the first complement. Its
  later debug parent-coverage loop is conditioned on proofs_enabled.
- Explicit-fact unit resolution appends parents and fact before its debug-only
  validation. It counts matches independently and accepts the source duplicate
  literal cases; do not replace it with the inferred-fact algorithm.
- check_nnf_proof_parents returns a bool that mk_nnf_pos/neg IGNORE. It still
  executes. Preserve its short-circuit reads and any earlier invalid access,
  but do not reject a well-formed non-OEQ parent solely for this ignored false.
- Quant-inst parameters are actual owning AST parameters in binding order. Its
  reader requires an empty output only in debug and appends in source order.
  Rewrite inspection executes VERIFY of eq-or-iff extraction in all profiles;
  OEQ is not added to this disjunction.
- Theory lemma prepends the actual family-name Symbol parameter before copied
  caller parameters, then appends parents and fact. Hyper-resolve flattens each
  substitution then two integer position entries between adjacent substitutions.
  It also builds the real temporary fact/Proof-sort vectors even where these
  local arrays are not consumed by the final mk_app. No dead-operation elision
  where allocations, budgets, assertions, references or failures are observable.
- Hyper-resolve decoding appends premises, assigns conclusion, pushes the first
  substitution vector, then reads parameter pairs/substitutions in order. An
  exception keeps completed output/refcount prefixes. It does not reset caller
  outputs or infer missing positions. Negative int parameters convert uint32.

## Ownership and test requirements

Use actual ASTManager, eager/lazy basic plugin declarations, pointer-free nodes,
source ID/hash interning, exact sorts and real RC; no fake proof graph. Temporary
argument/sort ptr_buffer arrays have real inline16 and doubling overflow charges.
Parameter vector copy/destruction and output ref-vectors use real source actions,
including inc_ref-before-growth failure prefixes. All cold owner headers and
retained/temporary backings have complete old+new budget accounting.

Tests first: observe missing-API or behavioral RED in an isolated committed HEAD
plus approved AST prerequisite snapshot. Cover all source assertion sites, both
release and g3debug branches, each overload, null/identity simplifications, real
proof kinds/facts/parents/parameters and source application-error routes. Add an
independent tiny Boolean clause-resolution oracle and directed malformed-input
guards; do not use valid statuses or uninterpreted dummy callbacks as a checker.
Fault every reservation and inspect actual live nodes/IDs/table/caller outputs
immediately after failure, before cleanup. Frozen full native/checked-debug,
named race/actual JS, vet/WASI and source/no-generics gates precede a commit.

The registered checker fixture must later run the real checker on hypothesis a,
hypothesis(not a), unit-resolution false, and lemma(or a (not a)), retaining
its actual one VERIFY and real side-condition vector. Its source prints require
real AST printers before any complete diagnostic-output claim.

## Read-only baseline fingerprints

- src/ast/ast.h 5d006f0ccd5f44f25bc8e0de9f629d49cd1df8fa1275bb121696cd922d0069fd
- src/ast/ast.cpp ac1de339a71c12fabaf951db2b0936bb1a84990b337ce268490633cb7bb5ff52
- src/test/proof_checker.cpp 2ae06c3cecd842fe5d3034cdda5125dc345ab012efe99941ffcc8ec41fa1e450
- src/util/ref_vector.h 60bdb516c3df2130840407979d05022ed119a7dc17e93493298075a68df8c1a7
- src/util/ref_buffer.h c617e23849390bd3d20d98e538bc5e4f53e62b2577dc97bbe10678590e33d4eb
