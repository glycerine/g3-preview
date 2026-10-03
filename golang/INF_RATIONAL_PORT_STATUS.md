# Owner-local infinitesimal rational port

Owned packet: `mpnum/inf_rational.go`, `mpnum/inf_rational_test.go`, and this
status. Source is the complete active `src/util/inf_rational.h/.cpp` representation
and eight approximation functions. Parent separately owns service fields/hooks,
lifecycle integration, the integer-epsilon specialization and rational facade.

## Representation and method inventory

`OwnedInfRational` contains two private raw `Rational` values and scalar service
provenance. Hot values contain no pointers/interfaces. The existing service
mutex and synchronized manager own all arithmetic and budgeted numeric backing
storage. No additional manager, numeric cache, Go generics, unsafe storage or
production math/big is introduced.

| Source surface | Concrete counterpart |
| --- | --- |
| default/int/int-fraction/rational/pair/signed-offset constructors | `NewInfValue`, `InfSetInt32`, `InfSetFractionSigned`, `InfSetFraction`, `InfSetRational`, `InfSetPair`, `InfSetOffset` |
| copy constructor/assignment, rational assignment | `InfClone`, `InfCopy`, `InfSetRational` |
| destructor/reset/swap | `InfDel`, `InfReset`, `InfSwap` |
| `init`/`finalize`, free initialize/finalize wrappers | private `initInfRationalConstants`/`destroyInfRationalConstants`, called by parent service lifecycle |
| `zero`/`one`/`minus_one` | `InfZero`, `InfOne`, `InfMinusOne` |
| `hash`, hash_proc, eq_proc | `InfHash`, `InfEqual` |
| `to_string`, stream insertion | `InfString` |
| `is_int`, `is_int64`, `is_uint64`, `is_rational`, int64/uint64 extraction | `InfIsInt`, `InfIsInt64`, `InfIsUint64`, `InfIsRational`, `InfInt64`, `InfUint64` |
| rational/first/infinitesimal accessors | `InfGetRational`, `InfGetFirst`, `InfGetInfinitesimal` |
| numerator/denominator | `InfNumerator`, `InfDenominator` |
| infinitesimal +=/-= and free +/- | `InfAddAssign`, `InfSubAssign`, `InfAdd`, `InfSub` |
| rational +=/-=/*=/ /= | `InfAddRationalAssign`, `InfSubRationalAssign`, `InfMulRationalAssign`, `InfDivRationalAssign` |
| rational-left multiplication, rational-right multiplication/division | `InfMulRational`, `InfDivRational` (multiplication operand-order wrappers share the same scalar method) |
| prefix/postfix ++/-- | `InfInc`, `InfDec`, `InfPostInc`, `InfPostDec` |
| unary minus, `neg`, `abs` | `InfUnaryMinus`, `InfNeg`, `InfAbs` |
| all six pair comparison operators | `InfEqual`, `InfNotEqual`, `InfLess`, `InfGreater`, `InfLessEqual`, `InfGreaterEqual` |
| all six mixed comparisons in both operand orders | corresponding `Inf...Rational` and `InfRational...` methods |
| zero/one/minus-one/negative/positive/nonnegative/nonpositive predicates | corresponding `InfIs...` methods |
| floor/ceil | `InfFloor`, `InfCeil` |
| addmul/submul | `InfAddMulAssign`, `InfSubMulAssign` |
| inf_mult/sup_mult | `InfLowerMult`, `InfUpperMult` |
| inf_div/sup_div | `InfLowerDiv`, `InfUpperDiv` |
| inf_power/sup_power | `InfLowerPower`, `InfUpperPower` |
| inf_root/sup_root | `InfLowerRoot`, `InfUpperRoot` |

The additional `InfMove` delegates existing raw-manager move semantics (including
preserved moved-from inline values); source inf_rational has no move constructor.
`InfCompare`/`InfCompareRational` are three-way convenience adapters; source
relational operators use the explicit Less/reversed-Less routes instead.

## Source action fidelity and ownership adaptations

Arithmetic and assignment publish first component before second; failures retain
already-published prefixes. Member destruction is the reverse: **second then
first**, because the empty C++ destructor implicitly destroys members in reverse
declaration order. Constants traverse zero, one, minus-one. Parent integration
initializes these after rational constants and before integer-epsilon constants,
and destroys them before integer-epsilon constants, the powers cache, rational
constants and manager. Constructor failure unwinds the same owned graph.

Source free compound-expression results explicitly copy the input, mutate the
copy, then copy the returned lvalue again. Named-result returns transfer ownership
without inventing a further copy. Approximation rational binary expressions use
that same clone/compound/return-copy path. Product temporaries coexist until the
sum/assignment full expression completes; denominator/half/power expression
objects are retained through their full expressions and deleted on all exits.
Fused wrapper operations follow rational.h's one/minus-one shortcuts and scoped
copy(k), multiply, accumulate; they do not substitute manager fused arithmetic.

Lexicographic Less first executes rational LT, then separate structural EQ before
considering epsilon. Greater reverses Less operands; <= and >= negate the correct
reversed/nonreversed Less. Unnormalized raw numerals therefore retain source
structural-EQ behavior. Increment/decrement and epsilon corrections use rational
Add/Sub with mk_q(1), rather than the manager mixed AddInt route. A tests-first
fault regression observed the wrong mixed route succeeding at injected prefix 8
where the required rational route failed; it now passes. A separate INT_MIN
constructor fault regression caught fractional normalization replacing the
source rational(int) temporary/copy route; InfSetInt32 now follows that route.

All eight approximation functions preserve source branches, including coarse
zero lower roots, unchanged upper roots, exponent-zero behavior and dropped
epsilon coefficients outside exponent one. These routines are not replaced by
more precise mathematics. Hash retains numerator-only component hash behavior
and uint32 addition/XOR wraparound.

C++ borrowed const accessors/constants become owned copies, so callers cannot
modify internal components or service constants through exported borrows. Every
public operation validates service provenance, raw handles and closed lifetime.
Copying across distinct service owners is rejected; numeric cross-owner copies
remain a separate explicit service facility. Public numerator/denominator require
zero epsilon; roots require nonnegative values; division approximation requires
nonzero denominator first component. Source assertions become typed errors.
The signed fraction constructor follows existing service widening for signed
INT_MIN-negation undefined domains. `InfSetFraction` additionally exposes the
existing int64/uint64 constructor adapter; source-int arguments use
`InfSetFractionSigned`.

## Tests-first evidence and coverage

The initial RED was undefined representation/APIs after tests were written.
`TestInfRationalSource80Assertions` ports every active ENSURE from
`src/test/inf_rational.cpp`: **62 sites, 80 executions** (18 helper sites called
twice, 42 direct main sites, two tst0 sites). It also executes numerator and
denominator constructors that the source does not assert. Source test registration
is `src/test/main.cpp:152` and `src/test/CMakeLists.txt:54`.

Twelve named groups cover those source assertions; fixed approximation branches;
225 independent finite multiplication bounds, 180 division bounds and 90 power
bounds evaluated with math/big rationals at epsilon=1/10000; coarse root fixtures;
hash/query/fused/source aliases; structural equality versus numeric ordering;
first-component scalar rereads; all four increment/postfix fault routes; and an
explicit failure boundary introduced by the required compound return copy.

Fifteen operations sweep injected allocation prefixes through first success.
Immediately after each operation, before formatting/deletion/Close or restoring
the budget, an independent caller/service-root graph audits live numeric cells
and Words references. Published partial output roots are allowed; unnamed local
roots are not. Service Close refunds the complete budget. Lifetime tests reject
foreign owners and use-after-Close.

## Verification and scope

An isolated `HEAD:golang` archive under `.cache/release/inf-rational-final` overlays
only this packet's Go files plus required parent service hooks/lifecycle tests
and integer-epsilon production dependency. All cache/temp/output writes remain
below golang. Full native and full g3checked module suites passed. Checked race
and fresh actual checked JS (`-count=1 -v`) passed all twelve InfRational groups
and the three parent lifecycle groups, with explicit RUN/PASS output. Vet and
no-generics syntax audit passed. Checked WASI compilation passed; runtime WASI
execution is not claimed. Parent separately verifies combined integration.

There is no remaining active inf_rational method/function gap in this packet.
Commented-out general pair multiplication/division/remainder, general div/power,
gcd/lcm declarations and the duplicate #if0 declarations are not active source
implementations. The separate inf_int_rational and inf_eps_rational types are not
claimed by this packet. Source tracing and assertion-message formatting are not
ported; source domain assertions have typed Go errors as described above.

Parent final combined frozen native30734, full checked race63802, and full actual
checked JS67573 passed across every module package. All41 new packet tests have
explicit JS RUN/PASS evidence in
`.cache/rational-facade-inf-ext-gcd-frozen-checked-js.log`; source/test checksums
remained unchanged. Checked full vet, WASI build and no-generics/source audit
also passed. These combined results include the final signed-constructor and
increment-route corrections; WASI runtime execution is not claimed.
