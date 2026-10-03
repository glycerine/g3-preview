# Concrete infinity/epsilon rational representation

Final parent integration with the reviewed parameter-description packet passed
full native default and full checked debug module tests, all eleven targeted
checked debug race groups, checked debug vet and WASI compilation, and source/
no-generics audits. Actual checked debug JS execution passed every package with
all eleven new names explicitly RUN/PASS in
`.cache/inf-eps-descr-integrated-final-js.log`. Production/tests matched the
verified snapshot before commit. WASI runtime is not claimed.

Owned files: `mpnum/inf_eps_rational.go`, `mpnum/inf_eps_rational_test.go`, and
this status. This ports the complete active `src/util/inf_eps_rational.h` surface
for its only actual repository specialization, `inf_eps_rational<inf_rational>`.
There are no Go generics, boxed numeral interfaces, production math/big, unsafe
storage, per-record pointers, new global managers, or numeric caches.

## Actual instantiation and test audit

Repository-wide C++ header/source searches find only `Numeral=inf_rational`:
`opt/opt_solver.h:37`, `math/simplex/model_based_opt.h:38`, `smt/theory_opt.h:32`,
`smt/theory_diff_logic.h:68`, `smt/theory_arith.h:1236`, and arithmetic/dense
 difference-logic method declarations and bodies use this specialization.
No `rational` or `inf_int_rational` specialization is actually instantiated.
Header `N` template declarations are operators, not additional instantiations.

There is no registered direct inf_eps_rational source test and no direct active
numeric assertion to claim ported. `src/test/model_based_opt.cpp` and
`src/test/qe_arith.cpp` mention alias `opt::inf_eps` in maximize/print demos, but
their registration entrypoints return unconditionally after `test10()` and
`test_project()` respectively, before those paths. Those downstream optimization
integrations remain outside this representation packet. All eleven test groups
here are explicitly additional source-derived or independent coverage.

## Complete active method map

`OwnedInfEpsInfRational` stores private infinity `Rational`, nested
`OwnedInfRational`, and scalar service provenance. All methods execute under the
existing `RationalService` mutex and validate the whole owning record first.

| Source surface | Concrete counterpart |
| --- | --- |
| default constructor | `InfEpsInfNewValue` |
| int constructor | `InfEpsInfSetInt32` on an explicitly constructed value |
| Numeral constructor / Numeral assignment | `InfEpsInfSetNumeral` |
| infinity+Numeral constructor | `InfEpsInfSetPair` |
| copy constructor / assignment | `InfEpsInfClone`, `InfEpsInfCopy` |
| destructor / reset / swap | `InfEpsInfDel`, `InfEpsInfReset`, `InfEpsInfSwap` |
| hash/hash_proc/eq_proc | `InfEpsInfHash`, `InfEpsInfEqual` |
| to_string / stream insertion | `InfEpsInfString` |
| is_int/is_int64/is_uint64/is_rational/is_finite | corresponding `InfEpsInfIs...` methods |
| int64/uint64 extraction | `InfEpsInfInt64`, `InfEpsInfUint64` |
| get_numeral/get_rational/get_infinitesimal/get_infinity | corresponding `InfEpsInfGet...` methods |
| zero/one/minus_one/infinity static value constructors | corresponding `InfEpsInf...` methods |
| pair +=/-= and free +/- | `InfEpsInfAddAssign`, `InfEpsInfSubAssign`, `InfEpsInfAdd`, `InfEpsInfSub` |
| inf_rational +=/-= | `InfEpsInfAddInfAssign`, `InfEpsInfSubInfAssign` |
| rational +=/-=/*=/ /= | corresponding `InfEpsInf...RationalAssign` methods |
| rational multiplication in either order, rational division | `InfEpsInfMulRational`, `InfEpsInfDivRational` |
| prefix/postfix ++/-- | `InfEpsInfInc`, `InfEpsInfDec`, `InfEpsInfPostInc`, `InfEpsInfPostDec` |
| pair six relational operators | Equal/NotEqual/Less/Greater/LessEqual/GreaterEqual with `InfEpsInf` prefix |
| rational mixed six comparisons in both orders | corresponding `InfEpsInf...Rational` / `InfEpsInfRational...` methods |
| neg/unary minus/abs | `InfEpsInfNeg`, `InfEpsInfUnaryMinus`, `InfEpsInfAbs` |
| zero/one/minus-one/negative/positive/nonnegative/nonpositive predicates | corresponding `InfEpsInfIs...` methods |
| floor/ceil | `InfEpsInfFloor`, `InfEpsInfCeil` |
| addmul/submul | `InfEpsInfAddMulAssign`, `InfEpsInfSubMulAssign` |

The C++ constructor surface becomes explicit Go construction plus typed setters;
reusing an existing setter destination destroys/replaces its previous owned
components. Integer construction invokes the nested inf_rational(int) route:
`rational(n)` initializes its first member through move/elision, with no additional
copy constructor. Signed int32 extremes remain inline source numerals.

## Preserved source behavior and actions

Infinity is an exact rational coefficient, not a saturated sign/flag. Addition,
subtraction, scaling, division and fused operations preserve its magnitude and
allow infinity cancellation. Pair ordering is lexicographic infinity then nested
first/epsilon, with separate source LT then structural EQ checks before descending.
Greater reverses Less; <=/>= negate the correct reversed/nonreversed Less.

**Mixed equality deliberately compares a rational against the infinity coefficient
and requires the nested numeral to be zero.** Mixed Less instead uses the sign of
infinity, and descends into the numeral only at zero infinity. Consequently an
infinity coefficient one with zero numeral equals rational one, while rational
one is simultaneously less than that value. Finite numeral five does not equal
rational five under this source overload. These source defects are frozen tests.
Floor/ceil ignore infinity because their source finite-domain assertions are
commented out; the nested infinitesimal rounding rules remain unchanged.

Assignment/arithmetic publish infinity before nested first before nested epsilon.
Assignment from Numeral resets infinity before a fallible nested copy; that reset
survives failure. Source compound-return expressions clone, mutate, then copy
 the returned lvalue again. Named-result returns use explicit ownership transfer,
matching the selected move/elision adaptation used elsewhere in the port. Every
failed value expression deletes its partial local graph. All multiplication,
fused accumulation and unit increment routes call existing faithful locked raw
helpers; component coefficient aliases are reread after earlier publications.

Destruction reverses declaration order: nested numeral first (epsilon then first
rational), then infinity. A production InfEpsInfDel allocator-reuse test verifies
actual release order; it does not run a mirrored test destructor. Static value
constructors borrow existing private owner constants during construction and copy
only the fields copied by the source. No new service fields/caches are needed.
Borrowed public accessors become owning checked copies; typed conversion errors
replace source assertions. Close invalidates all values; foreign owners are
rejected before reservations or mutations.

Numerical cells, word payloads, descriptor/directories and formatting scratch use
the existing owner budget. This packet adds no independent cold owner/header or
managed backing array; caller-provided hot records follow the existing value API.
String construction stages all text in the existing accounted NaturalManager
workspace and allocates exactly one final caller-owned host output. A tests-first
regression observed six warmed allocations from intermediate strings before this
correction. Absolute-value numeral temporaries survive their formatting expression
and die before its following closing-parenthesis append. Growing text preserves
completed prefixes and checked uint32 lengths; failures clean all numeric locals.

## Tests-first evidence and checks

Initial tests preceded production and observed undefined type/API RED. Additional
coverage includes 729 independent finite triple-pair arithmetic/order cases;
all header branches and mixed comparison routes; coefficient fractions/signs;
string spellings/hash wraparound; constants/conversions; self/compound/fused
component aliases; source noncanonical structural EQ; infinity cancellation;
assignment reset prefixes; and the required returned-lvalue copy failure boundary.

Eighteen operation routes sweep reservation prefixes through first success.
Explicit test-owned filler numerals occupy spare cell pages when a real new
reservation is needed; pooled copies are not assumed to allocate. Seventeen
routes expose failures in these fixtures; one existing-capacity Numeral assignment
can complete without reserving, and a separate cold-copy fixture verifies its
published reset on failure. Immediately before formatting, Del, Close or restoring
the budget, an independent known-root audit checks both live numeric cells and
Words references. Legitimate partial caller outputs and owner scratch are the
only allowed roots. Close returns the budget to zero. Recursive layout inspection
proves pointer-free hot fields, warmed arithmetic/comparison allocate zero Go
objects, and warmed String allocates only its one escaped final output.

## Verification

Final verification uses a committed `HEAD:golang` archive plus only this packet's
Go files under `.cache/release/inf-eps-inf-final`. All writes, cache/temp and WASI
output are under golang. Targeted native/checked tests passed before the final
full gates. Full native module tests passed (mpnum 9.643s), full checked module tests passed
(9.652s), checked race for all eleven InfEpsInf groups passed (2.072s), and fresh
actual checked JS -count=1 -v ran all eleven named groups with explicit RUN/PASS
(0.490s). Vet, checked WASI compilation and no-generics audit passed.
WASI runtime execution is not claimed.

There is no remaining active method gap for the actual inf_rational specialization.
Hypothetical uninstantiated Numeral types and downstream optimization/theory
algorithms are not claimed as ported by this packet.
