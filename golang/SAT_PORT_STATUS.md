# SAT prerequisite translation

`util_lbool.go` translates all scalar/printing operations in `src/util/lbool.h`
and `.cpp`. Undefined stays distinct, including under negation. Returned rendering
strings are Go-owned or immutable literals. There is no substantive source lbool
unit-test file; the finite semantic table supplements source definitions.

`sat_types.go` translates the literal constructors (including actual SASSERT
variable/sign expectations), extraction/sign/unsigned/hash/unsign/negation,
comparison through scalar operators, literal/DIMACS/vector printing, and partial
model lookup/printing. The source default literal is an explicit constructor;
a Go zero literal is positive variable zero. Literals remain uint32 under
`g3checked` and are distinct from arena refs. Invalid constructor widths panic
as internal precondition failures instead of silently truncating variables.

`sat_sets.go` translates SAT's insertion-list uint_set and literal_set:
insert/remove/contains, copy assignment, erase/pop, ordered traversal/vector
conversion, union/intersection, reset and finalize. Remove swaps in the last
member exactly as source. This is distinct from util/uint_set.h's dense bit set.
Backing slices contain scalars; copy is explicit, reset retains storage, finalize
drops owner references, returned vectors are independent copies. Slice growth
uses Go backing arrays; solver-owner budget plumbing is pending. No Go map
iteration chooses order. The 3,000-step bool-array oracle is independent of the
production membership/list representation and RNG. Additional finite cases lock
down swap-remove order, aliases, reset/reuse and literal polarity.

`sat_approx_set.go` translates the uint32 mask specialization of approx_set_tpl
used by SAT literals/variables. Other template specializations and the 64-bit
approx_set iterator/diagnostics are unported. Low-bit collisions are deliberately
preserved; may-contain is not exact set membership.

Both primitive suites were authored first and failed with undefined types and
functions before implementation. Main-package release/checked tests and actual
js/wasm execution pass. These types do not implement propagation, search,
clauses, standalone SAT, legacy SMT, model production or a solver API. Remaining
sat_types source coverage: error classes/parameter errors, source memory
statistics/platform adaptation, phase/external-index aliases and solver-backed
integration. SAT source scope/assumption tests are currently independent oracle
fixtures only; their real producer path awaits the translated solver.
