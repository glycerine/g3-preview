# Integer and rational root packet

Status: **partial numeric source units**. This packet completes selected internal
MPZ root/perfect-square methods and MPQ counterparts, without claiming remaining
numeric methods, alternate GMP backends, public rational wrappers or context
ownership complete. All source algorithms were read before translation, including
mpz.h wrappers, both complete mpz.cpp root bodies, mpq.cpp root, mpq.h predicates,
source normalization, and all active mpz/mpq tests. Copyright notices are retained.
No production math/big, generics, unsafe, interfaces per limb, new persistent
manager headers or external C++ fixture/compiler were introduced.

## Source methods and contracts

| Source method | Go counterpart |
| --- | --- |
| `mpz_manager::root(mpz&,unsigned)` | `IntegerManager.RootInPlace` |
| `root(mpz const&,unsigned,mpz&)` | `IntegerManager.Root(a,n,out)` |
| `mpz_manager::is_perfect_square` | `IntegerManager.IsPerfectSquare` |
| `div_l/div_u` root bound helpers | fixed-width quotient/ceiling blocks in `rootCore` |
| `mpq_manager::root(mpq const&,unsigned,mpq&)` | `RationalManager.Root` |
| `mpq_manager::is_perfect_square(mpq const&,mpq&)` | `RationalManager.IsPerfectSquare` |
| `mpq_manager::is_int_perfect_square` | `RationalManager.IsIntPerfectSquare` |
| inherited MPQ mpz root/perfect-square forwarding overloads | existing owned `Integers()` service exposes both Integer root forms and square query; no extra overloaded Go forwarding API |

Integer root keeps the exact source log2/mlog2 power-of-two initial bounds,
unsigned `k+1` arithmetic, lower-equal-upper branch, bisection, truncating
machine_div2k(mid,1), Power before midpoint termination test, comparison order,
Swap bound updates, and final upper-bound publication. False results are signed
ceiling roots: 213^(1/5) publishes3, -213^(1/5) publishes-2. The source Power
implementation, including its final extra square and classified mask domains,
is used rather than a separate root algorithm.

Integer perfect-square query preserves the source six temporaries, initial
hi=a/sq_hi=a*a, source Copy updates, early zero/one branches, bisection using
Div(mid,2), exact lower square check and false-result upper publication. Negative
inputs return false and preserve the output. The source output Reset precedes
rereading a: a nonnegative `a==out` therefore becomes0 and returns true. This
source-derived alias defect is deliberately retained; independent mathematical
square tests use a distinct output. Negative aliases remain unchanged.

Rational root keeps `root(num)&&root(den)` short circuit, including successful
false-result publication. An inexact numerator changes only the output numerator
and retains the previous output denominator; an inexact denominator publishes
both computed roots. No normalization is added. Nonalias and alias fixtures
compare these source output parts, including deliberately unreduced2/4 results.
Rational square integer branch resets output denominator first. A negative
integer input thus returns false after changing only that denominator to1;
a negative fraction leaves all output unchanged. Inexact fraction numerator
retains the old output denominator; precise numerator/inexact denominator
publishes both upper bounds, and normalization happens only when both are precise.
Integer-only square query requires source is_int and otherwise reports its
assertion domain. Raw Go zero Rational remains invalid; use NewRational.

A correction from the initial read-only alias proposal is explicit: source
normalize does **not** special-case a zero numerator. Nonnegative fractional
`IsPerfectSquare(a,a)` resets num and den to0, both integer queries return true,
then gcd(0,0)=0 leads to source Div(0,0). This is source undefined arithmetic,
not a finite0/1 result. The adapter reports existing ErrDivideByZero and preserves
the original fraction on this error. Integer-denominator1 aliases remain the
finite source0/true result. This inventory is derived from executable source;
no C++ observed output is claimed for the undefined path.

## Classified domains and storage adaptation

`RootDomainError` retains Degree and Reason and unwraps to distinct sentinels:
ErrRootNegativeEven replaces the negative/even source assertion (also negative
n==0); ErrRootDegreeZero replaces nonzero root degree0 division by zero; and
ErrRootIntegerRequired replaces the integer-only rational square assertion.
Integer zero degree0 still returns true before source bound division. Rational
zero degree0 reaches its nonzero denominator root and reports the degree error.
The source negative midpoint -1 can inherit PowerDomainError for odd degrees
>=2^31; it is not silently optimized. Source finite nonnegative zero/one/base2
fast-path outcomes at huge degrees remain representable and tested. Predictable
invalid-domain errors leave source/destination values unchanged.

All temporaries are ordinary scalar Int/Rational values whose big payloads live
in existing IntegerManager cells/Words with the same owner Budget. Deferred
cleanup follows source local order and also runs on error. There are no per-call
unbudgeted numeral/digit arrays or new numeric owner objects. Rational methods
use existing synchronized/local versus unsynchronized/cached normalization
paths, remaining owner-confined in both modes.

Root output is staged in owned numerals before final Move, including in-place
calls. Square queries stage output while retaining source negative/alias branches.
Rational work starts as an explicit Copy of the previous destination and shadows
input when aliased, preserving exact source rereads and short-circuit values.
This is an intentional allocation/timing adaptation: old output storage survives
until publication, copied/staged owners can increase the peak, and source partial
mutation on an allocation/Power error becomes unchanged caller output. Source
finite false results are still published. Retained manager scratch may grow on
failure and remains charged until Close; no claim of identical source physical
allocation/destructor timing is made. Final Close releases complete owner graphs.

## TDD and permanent tests

Initial tests were written before implementation and the observed package build
was RED with missing Root, RootInPlace, IsPerfectSquare and domain error APIs.
The implemented source algorithms then reached GREEN. The first alias fixture's
assumption of fractional0/1 was corrected by rereading normalize: the code already
reported the correct undefined division category, so no arithmetic was changed
to make that mistaken expected value pass.

`TestIntegerSourceRoots` ports all ten active assertions in the five source
`tst_root` cases plus the two printed result lines. `src/test/mpq.cpp` contains
no root/perfect-square assertions: rational cases are clearly added expectations
from source control flow, not invented source assertions. Independent math/big
is test-only, using whole-integer binary search with different bounds rather than
source log-based iteration. Tests cover signed finite roots in[-64,64], degrees
1..7, eighty fixed-seed up-to128-bit inputs, both output aliases, square values
0..512 and negative preservation, large perfect-square boundaries at shifts
31/32/63/96/127 with±1, and independent canonical rational numerator/denominator
roots in both source modes for degrees1/2/3/5. Source short-circuit and alias
fixtures are separate from independent mathematical expectations.

Three integer methods x two alias forms x small/big prior-output shapes use bounded fault-prefix sweeps and
require finite success plus unchanged inputs/outputs/live cells on failure;
allocation-free aliased square Reset does not require an artificial failure.
Both rational modes x three methods x two aliases x small/big prior-output
shapes sweep prefixes. Immediately after every injected error, before String or
Close, live numeric cells and Words refs must equal the caller+manager baseline.
The exercised root/integer-square branches do not normalize, so a spare cached
GCD cell is not permitted to conceal a lost temporary. Values are then checked
unchanged and final Close must return the budget to zero.
Successful aliased integer square branches may consume no new reservations with
prepared pools. Degree0/negative-even/large-mask/zero/one domains, integer-only
assertion, raw-zero initialization and closed owners are tested. Warm integer
and genuinely big rational root/square operations allocate zero Go objects and
succeed under FailAfter(0) after bounded preparation. All tests are hermetic and
finite; writes/cache/temp/artifacts remain under golang.

## Verification

Native and actual g3checked targeted root suites GREEN. Prospective isolated
HEAD:golang with only these four approved new files, under
`.cache/release/numeric-roots-final`, passed full native `go test -count=1 ./...`
(mpnum5.706s) and actual checked full module tests (mpnum5.700s). All11 new root
registrations ran and passed actual g3checked JS with the existing toolchain
wrapper preserving TMPDIR (0.570s). Actual checked race root tests passed2.390s;
checked vet and no-generics portcheck passed. Release/checked WASI test binaries
cross-built; this is compilation evidence, not WASI execution. Remaining full
combined packet review/commit belongs to the coordinator. No shared manifest,
existing numeric header/files, git metadata or unrelated package was edited.

Central review strengthened the existing failure sweeps with immediate pre-String
cell/Words live baselines and both small/big prior-output shapes. These additions
were immediately GREEN without any source algorithm change.
Final strengthened-sweep rerun: all11 native root tests passed0.168s, actual
checked race passed2.742s, and all11 actual checked JS registrations passed0.692s.
The prospective isolated test/status copies match the reviewed workspace files.
