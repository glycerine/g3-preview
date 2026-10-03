# Independent finite SAT oracle

This package is test infrastructure, not a translation or replacement of the
production SAT/SMT solver. It imports no g3 algorithms or representations.

`Literal` is a separate signed one-based representation; `CNF` owns no manager
state. `Solve` checks all literals and variable bounds before enumerating the
entire Boolean domain (at most 16 variables, 65,536 assignments). It returns
exact SAT with a total copied model or exact UNSAT. Malformed/oversized inputs
return `InputError`; they never produce unknown or a semantic solver result.
`ValidModel` checks all clauses and assumptions against a total Boolean model.
Inputs are borrowed without modification/retention; returned models belong to
the caller. The implementation has no propagation, learning, random generator,
solver dependency, or hidden mutable global state.

Worst-case work is 2^Variables times input literal occurrences plus assumptions.
There is no claim of an unbounded SAT oracle, EUF/quantifier/SMT interpretation,
production assumption handling, production user scopes, unsat cores, proof
objects, or solver model reconstruction.

## Source expectations and additions

Read `GO_PORT_PLAN.md` C1/C4, `src/test/check_assumptions.cpp`, and
`src/test/sat_user_scope.cpp`. Source scenario fixtures are permanently embedded
under `test_vectors/`; ordinary tests require no source files, compiler, process,
network, Python, or neighboring checkout.

- `check_assumptions.cpp` actively asserts SAT for `p or q or r` with assumption
  `not p`. The fixture preserves that expectation; choosing q=true proves SAT.
- Its final query asserts `not p` and assumes `not q,not r`, but has no result
  assertion in the source: it is a crash regression. The added explicit UNSAT
  expectation is independently derived because all disjuncts become false.
- `sat_user_scope.cpp` compares an incremental solver with a rebuilt solver in
  an infinite loop. The finite embedded script supplies independent active-CNF
  expectations for nested pushes/pops, empty frames, and temporary assumptions.
  Frame management here is a test harness only. No production scope behavior is
  implemented or validated yet; the oracle is ready for future solver comparison.
- Added permanent mathematical fixtures cover empty conjunction/disjunction,
  opposing units, duplicate/complementary literals, binary implications,
  six-literal clauses, and contradictory assumptions.
- Added tests exercise all sixteen two-variable truth tables. Clauses are
  constructed by excluding false assignments; returned witnesses are checked
  directly against the independent truth-table bits. Tests at the 16-variable
  bound select the final all-true assignment, then independently contradict it.
  Invalid input, int32 minimum literal, full-input checking after an empty clause,
  malformed models, input isolation, and model isolation are checked separately.

## TDD and validation

Tests and permanent fixtures were authored first. The initial test run failed
with undefined `Literal`, `CNF`, `Solve`, and `ValidModel` (RED). The subsequent
enumeration implementation made the tests pass (GREEN). Native package tests
pass at 100% statement coverage. Race tests pass. Test-binary cross-builds for
js/wasm and wasip1/wasm pass; this is compilation evidence, not wasm execution.
All writes, temp files, cache files, and build outputs from this packet are
under `golang/`; commands use `golang/.cache` for GOCACHE/GOTMPDIR/TMPDIR.
