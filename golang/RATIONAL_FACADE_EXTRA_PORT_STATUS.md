# Remaining owner-local rational facade groups

Source: `src/util/rational.h` and four `src/util/mpq.h` bitwise forwarders,
Copyright (c) 2006 Microsoft Corporation. New files only:
`mpnum/rational_service_extra.go`, its test, `mpnum/mpq_bitwise.go`, its test,
and this status. No existing service/manager file, central ledger or git edit.

## Source signature inventory

The following maps the entire selected rational.h facade to concrete APIs,
including previously committed groups. C++ overloads/operators are named Go
methods; raw owners and source global references do not escape.

| Source group | Owner-local route |
| --- | --- |
| initialize/finalize, global manager/constants/powers | RationalService constructor/Close, private per-owner constants/cache and both committed infinitesimal initializer/finalizer hooks |
| default, copy, move constructors/destructor | NewValue, Copy/CopyFrom, Move, Del; service Close invalidates outstanding tokens |
| int, unsigned, signed pair, text, i64/ui64 constructors | Existing SetInt64, SetUnsigned, SetFractionSigned, SetString, SetInt64/SetUint64; source signed/unsigned32 representation boundaries remain distinct |
| mpz constructor | NewFromScopedInt through checked explicit source owner and IntegerManager.CopyFrom; SetFromScopedInt also provided |
| mpq constructor | NewFromScopedRational through checked explicit source owner and RationalManager.CopyFrom; SetFromScopedRational also provided |
| double constructor and assignment | NewDouble/AssignDouble return typed RationalDoubleError, the source UNREACHABLE development-domain adaptation; no conversion is invented |
| assignment rational/int, swap | Existing Copy/CopyFrom, AssignInt and Swap |
| to_mpq const borrow | ToScopedRational independent copy into a checked concrete destination owner; no returned raw borrow |
| get_rational, to_rational, get_infinitesimal, is_rational | GetRational/ToRational independent copies, GetInfinitesimal checked zero copy, IsRational checked true |
| bitsize/storage_size/reset, small/big/int and sign/zero/one predicates, parity | Existing locked concrete inspection/reset methods |
| hash, hash_proc, eq_proc | Existing Hash and Equal callable methods; C++ stateless functors need no new owning object |
| to_string/display/operator<< | Existing String, new Display and DisplayInto/FormatInto; caller-owned string/buffer replaces native std::ostream surface |
| display_decimal | DecimalString and FormatDecimalInto, forwarding owned source arithmetic workspace; SMT2String/FormatSMT2Into additionally expose ready raw printing variant |
| is/get uint64/int64, unsigned/int32, double | Existing guarded source queries; wrapper32 inclusivity preserved, Float64 retains source conversion behavior |
| numerator/denominator value extraction | Existing Numerator/Denominator; NumeratorToScopedInt/DenominatorToScopedInt additionally copy into checked integer owners |
| RR compound +,-,*,/,%; int compound + | Existing *Assign/AddAssignInt; scalar constructs rational(int) before source route |
| RR value +,-,*,/; both int directions | Existing initial lhs Copy, compound, returned-lvalue Copy; scalar temporary constructed and destroyed within the expression |
| unary minus/abs/member neg | Existing copied named result, mutation, owning transfer; NegAssign |
| div/machine_div/mod/% values; div/mod output forms | Existing fresh-result IDiv/MachineDiv/Mod/Rem and direct identity IDivInto/ModInto |
| mod_hat and get_num_bits | Existing exact wrapper branches and repeated fresh-IDiv/Copy assignment loop; no numerical shortcut |
| prefix/postfix ++/-- | Existing local source mk_q(1) raw rational Add/Sub; postfix copied old value |
| six RR comparisons, four R-left-int comparisons | Existing structural Equal, swapped/negated Less routes; signed rational temporary for scalar comparison |
| floor/ceil/expt/free power, power_of_two and static constants | Existing fresh-result rounding/Power, source signed-to-uint32 exponent forwarding, cache/constants owning-copy adaptation |
| addmul/submul | Existing coefficient +/-1 branches, else Copy(k), Mul(kcopy,c), receiver Add/Sub; local lifetime preserved |
| is_int_perfect_square/is_perfect_square/root | New direct output-identity manager forwards, including bool and output mutation on successful false |
| gcd pair | Existing GCD fresh result |
| gcd(r1,r2,a,b) | New GCDCoefficients fresh gcd result and direct coefficient identities to manager ExtendedGCD; distinct from committed util/ext_gcd.h service ExtendedGCD |
| lcm | New LCM fresh result, source signed shortcuts preserved |
| bitwise or/and/xor/not | New fresh-result service wrappers plus complete four raw rational forwards to reviewed integer algorithm and then denominator reset |

No selected active facade arithmetic primitive remains missing after these
owner-local adaptations. This inventory does not equate Go syntax, native stream
objects or unsafe reference borrows with literal C++ ABI compatibility, and does
not declare unrelated MPZ/MPQ diagnostics or additional template instantiations
complete. Central source-unit coverage remains the coordinator's decision.

## Exact publication, ownership and domain behavior

Root/square wrappers pass actual output raw identity; they do not hide source
alias bugs by copying inputs into a new wrapper result. Root's inexact numerator
publishes while retaining the previous denominator; denominator results are not
normalized. Integer square output aliases reset to zero; fractional square input
aliases may attempt source undefined 0/0 normalization, represented by the existing
typed divide-by-zero error without publication. Existing predictable allocation
failure staging is inherited unchanged. Domain/error order comes from the reviewed
raw root methods, including negative-even and degree-zero classification.

GCDCoefficients calls the **manager** coefficient algorithm; ExtendedGCD remains
the separate ext_gcd.h wrapper. Coefficient denominators reset before allocations,
and completed coefficients remain published on failure. There is no pair-output
transaction that erases those source prefixes. Source sign rereads remain visible:
positive 120 with 84 and x aliasing120 yields x=2,y=3,g=12; 120 with-84 and y aliasing
-84 yields x=-2,y=3,g=12. Shared x/y gives2 after quotient steps 1,2,3. These are
independently inferred source-action fixtures, not C++ executions or Bezout
mathematical expectations for defective alias contexts. LCM's source signed
shortcuts retain -6 for (-6,1)/(1,-6)/(-6,-6), and -12 for (-6,4)/(6,-4).

All new bitwise wrappers retain actual integer source algorithm, operand copies,
capacity choices, nonnegative-domain errors, unsigned width and inherited
width 63 signed-mask adaptation. Rational integer preconditions remain strict;
unreduced 2/2 is not silently accepted as an integer. Raw rational output numerator
is computed before denominator reset, preserving aliases and failure publication.

Every service input/scoped destination is validated before numeric action. Scoped
owner identity/self/closed validation precedes explicit CopyFrom bridges; source
and destination managers are supplied from the checked owner, not an unlabelled
raw record. Scoped owners and their managers must remain externally confined for
the whole call; the service mutex does not make foreign managers or shared Budgets
concurrently safe. Existing low-level implicit numeric ownership contracts remain
in force; deliberate misuse of a previously borrowed Scoped.Get raw record is not
magically detected by coinciding cross-manager handle bits. These APIs return no
new raw borrow. Independent copies preserve requested source capacity, sign,
small/big shape, unreduced fractions and signed denominators; no parser or
normalization stands in for a record copy. Every intermediate is lexical and
budgeted through existing concrete managers. Named value returns transfer through
Move. Self/constant projections and to_mpq use independent copies as the explicit
Go replacement for source const references. Final Go strings belong to callers;
caller buffers exclude NUL and remain untouched on short-buffer/budget failures.

## TDD and meaningful coverage

Initial new tests compiled RED because all four raw Bitwise and the source
root/square service APIs were absent. ScopedRational constructors and ToScopedRational
also each had separately observed missing-API RED tests before implementation.

`TestRationalServiceExtraSourceTST8` executes exactly **34 ENSURE** assertions:
all 32 integer fixtures and both fractional-square assertions. Source default out
construction, signed-int/fraction constructors, predicate/equality routes,
short-circuit conditions and full-expression destruction are preserved by the
scoped assertion harness; there is no parser substitution for source integer
constructors. Source MPQ tests have no rational bitwise assertion; no added oracle
is falsely counted as a translated source assertion.

Added finite independent big.Int/big.Rat tests cover roots of exact powers over
small rational bases, root self aliases, signed coefficient identities, bitwise
small/big boundaries and 129-bit complements, raw output/all operand aliases,
source negative/successful-false outputs, source coefficient/LCM defects, display
bytes, short-buffer publication, typed UNREACHABLE doubles, provenance and
lifetime errors, and warm zero allocation. Both scoped constructor directions
and all scoped extraction paths have reservation-prefix cleanup tests. Source
raw signed-denominator copy behavior is explicitly retained.

Twelve operation failure sweeps and twelve raw bitwise output-alias sweeps audit
known caller/service/manager **cells and Words immediately before String, Del or
Close**. Pure-copy fixture banks consume prepared free slots to force real
reservations; wide (>16-limb) arithmetic forces real scratch growth; decimal
precision 1024 forces formatter growth. Sweep completion is detected rather than
assumed at an undersized arbitrary prefix. Manager coefficient failure tests
assert the source denominator-reset prefix instead of demanding unchanged outputs.
Every owner Close returns its own budget to zero; projected/scoped source copies
remain independent after mutating or closing the source.

## Completed validation and handoff

The isolated prospective tree `.cache/release/rational-extra-final` contains
committed `HEAD:golang` plus only these four approved new Go files. It passed
full native `go test ./...`, full actual `-tags=g3checked` module tests, all
fourteen targeted checked race tests, checked full-module vet, checked WASI
compilation, and the no-generics/ledger syntax audit. Actual checked
JavaScript/WASM execution listed RUN/PASS for all fourteen targeted names.
No active gate remains, and normal production/test files are frozen.

The parent is separately adding source Z3DEBUG coefficient diagnostics under
`g3debug`, distinct from handle-validation tag `g3checked`. The three intentional
release-alias coefficient fixtures in
`TestRationalServiceExtraIndependentRootsAndCoefficients` require build-aware
assertions of the diagnostic error and coefficient publication prefix; parent
owns that final test-only adaptation after this packet freeze. Distinct-output
independent Bezout properties remain valid in debug. No alias fixture should be
skipped or silently replaced with mathematical expectations for defective source
contexts. This packet's reported normal gates preceded that separate debug
integration; debug gates belong to the parent's combined review.

Final parent integration added build-aware assertions for all three defective
coefficient aliases, including invalid returned token on diagnostic failure.
The verified snapshot passed full native default and full checked debug module
tests, all fourteen checked debug race groups, checked debug vet and WASI
compilation, and no-generics/source audit. Actual checked debug JS/Wasm passed
the full mpnum suite with all fourteen new names explicitly RUN/PASS in
`.cache/rational-extra-integrated-final-js.log`. The four Go files matched that
snapshot before commit. WASI runtime is not claimed. Inventory now counts
22 ported, 41 partial, 2039 unported source units and 677 API operations.
