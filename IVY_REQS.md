# GoIvy SMT Client Requirements

This is the original client survey, retained as research context. Porting ended
at the user's request on 2026-10-02; these requirements do not describe a
completed backend. See `README.md` for the final handoff.

This file records the SPI that `~/ivy/goivy` needed at the time of the survey
from a future pure-Go SMT backend. It was surveyed from `~/ivy/goivy/smt/` and the production
`~/ivy/goivy/z3bridge*.go` files.

## Client Shape Recorded During The Survey

GoIvy talks to Z3 through package `github.com/glycerine/ivy/goivy/smt` and a
higher-level `goivy.Solver`/`Translator` bridge. The replacement backend under
`golemma/smt` should eventually provide the same semantic capabilities, even if
the final API is adapted behind an adapter instead of matching type names
one-for-one.

## Low-Level SMT Surface

The low-level client surface is context-owned:

- `Z3Context` owns solver state, AST/sort/function objects, symbol caches,
  check counters, error handling, and interruption.
- `Z3Sort`, `Z3Expr`, `FuncDecl`, `Z3Solver`, and `Model` are lightweight
  values tied to a context.
- Solvers expose assertion, push/pop/reset, assumption checks, unsat cores,
  SMT-LIB dumping, model extraction, and solver parameters.

The replacement needs equivalents for:

- Context lifecycle: create, close, clear/reset caches, interrupt an in-flight
  check, maintain deterministic check counters.
- Sort construction and inspection: Bool, uninterpreted, Int, Real, String,
  bit-vector, array, enumeration, array domain/range, sort equality and sort
  IDs.
- Term construction: constants, boolean literals, integer literals, string
  literals, bit-vector literals, function declarations/applications.
- Boolean and arithmetic operators: not, and, or, implies, iff, equality, ite,
  add, sub, mul, div, comparisons.
- Bit-vector operators: width creation/inspection, value creation, bitwise ops,
  arithmetic ops, shifts, concat, extract, bv/int conversion, unsigned
  comparisons.
- Array operators: select, store, const-array.
- Quantifiers: forall, exists, bound-variable inspection, body inspection,
  substitution.
- Inspection: app/var/quantifier/numeral predicates; AST IDs; arg/decl access;
  declaration name, kind, arity, domain, range.
- Solving: assert, check, check assumptions, unsat core, reason unknown, model,
  model eval, sort universes, push, pop, reset, set parameters.
- Text IO: parse SMT-LIB with supplied sort/function declarations, render
  solver assertions in canonical SMT-LIB-like form.
- Interpolation: create interpolation context, mark interpolants, compute
  binary/tree interpolants, convert interpolants back into Ivy formulas.

## High-Level Bridge Requirements

`goivy.Solver` and `goivy.Translator` add Ivy-specific semantics on top of the
raw solver:

- Module-scoped session cache for sorts, predicates, constants, functions, and
  inverse sort lookup. Multiple solver instances for one Ivy module share a
  context/cache.
- Deterministic symbol naming with special handling for Z3 builtins,
  polymorphic operators, array helper names, bit extraction, numerals, and
  generated skolems.
- Translation of Ivy sorts: Boolean, uninterpreted sorts, enumerated sorts,
  function sorts, array sorts, bit-vectors, integer/nat/range interpreted
  sorts, strings, and solver-only encoded sorts.
- Translation of Ivy expressions: variables, constants, applications,
  equality, boolean connectives, ite, quantifiers, raw forall, and definitions.
- Quantifier constraints for interpreted sorts, especially nat non-negativity
  and range-sort lower/upper bounds.
- Enumerated-sort support in two modes: native solver enumerations and binary
  Boolean encodings.
- Numeral handling for int, nat, range sorts, bit-vectors, strings, and quoted
  literals, including range clamping where configured.
- Clause-level operations: formula-to-solver assertion, clauses-to-assertion,
  negated clauses, bound quantifiers, duplicate removal, implication over one
  or many goals, and check sequences of assume/assert steps.
- Model APIs: model values for constants, sort universes, Herbrand-style model
  mining, relation/function model-to-clauses conversion, small-model search,
  final-condition callbacks, and redundant fact filtering.
- Unsat-core APIs: assumptions over clauses, optional core minimization, stable
  activation literal naming, and soft-assumption deletion order compatible with
  the current C++/Python behavior.
- Conversion back from solver ASTs to Ivy formulas, including interpolants,
  numerals, native declarations, arrays, quantifiers, and range arithmetic.
- Canonical rendering and tracing hooks used by xtrace/conformance tests.

## Porting Implications

- The OpenSMT port must preserve stable integer handles for sorts, symbols,
  terms, SAT literals, clauses, enodes, and proof/interpolation nodes. GoIvy
  relies heavily on caching and repeated identity comparisons.
- Incremental solver scopes are mandatory: push/pop must cover SAT clauses,
  theory state, term declarations, assumptions, and model/interpolation state.
- Interpolation is not optional for the end state. It can be staged after SAT,
  EUF, arithmetic, arrays, and model extraction, but the data structures should
  keep partition/proof metadata from the beginning.
- SMT-LIB parse/render compatibility matters because GoIvy has direct SMT-LIB
  paths and canonical-output tests.
- The first Go memory layer should keep hot data pointer-free: `[]uint32`
  arenas for terms, clauses, literals, and IDs; sparse pointerful maps only for
  names, hash-cons tables, and non-hot orchestration.
