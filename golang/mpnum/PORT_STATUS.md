# Numeric foundation packet

Status: **partial source unit**, not completion of the numerical manager layer.

Source: `src/util/mpn.h`, `src/util/mpn.cpp` (Microsoft 2011, Christoph
Wintersteiger). The full arithmetic algorithms were read before translation.
`src/util/mpz.h`, the arithmetic call sites and `quot_rem_core` in `mpz.cpp`,
and source assertions in `src/test/mpz.cpp`/`mpq.cpp` were inspected for
preconditions. The later MPZ lifecycle packet below reads each translated method
and preserves the internal 32-bit backend. Basic MPQ operation coverage is recorded in the later packet below.

## Operation coverage

| Source method | Go counterpart | State |
| --- | --- | --- |
| `compare` | `NaturalManager.Compare` | translated |
| `add` | `NaturalManager.Add` | translated Algorithm A |
| `sub` | `NaturalManager.Sub` | translated Algorithm S |
| `mul` | `NaturalManager.Mul` | translated Algorithm M |
| `div` | `NaturalManager.Div` | translated valid caller path; see preconditions below |
| `div_normalize` | `normalizeDivision` | translated |
| `div_unnormalize` | `unnormalizeDivision` | translated |
| `div_1` | `divideOne` | translated |
| `div_n` | `divideMany` | translated Algorithm D |
| `to_string` | `NaturalManager.String` | translated repeated division by ten |
| constructor/destructor OpenMP lock | owner confinement | platform adaptation; no lock lifecycle implemented |
| `display_raw`, both `trace` overloads, `trace_nl` | none | unimplemented diagnostics |

MPQ remains partial, with basic operations recorded below; the rational wrapper,
rounding/fixed-precision and higher arithmetic source methods remain incomplete.
MPZ method coverage is recorded below. No fallback
arithmetic or semantic stubs.

## Tests and TDD evidence

Tests were added first. `go test ./mpnum` initially failed to build with undefined
`NaturalManager` at all test call sites (RED). After algorithm translation the
same tests passed (GREEN). Further regression fixtures exercise quotient-estimate
correction, estimate equal to the base, and Algorithm D subtract/add-back.

`TestSourceMPZMagnitudeProjections` carries `src/test/mpz.cpp`'s `tst2`
UINT_MAX carry/square values and `bug1`'s exact large self-subtraction input
through the underlying limb methods. These are **projections**, not full ports of
the signed manager assertions. Parsing fixture input uses the independent test
oracle; the source decimal-string `set` operation is not implemented. Remaining active `mpz`
and deferred `mpq` assertions are inventoried in later packets.

Independent tests use `math/big` only in test code and a separate fixed-seed
generator: 1,000 arithmetic cases with at most eight limbs; 2,000 division and
decimal cases with at most eight limbs. They compare exact results and modular
subtraction borrow, validate preserved inputs, and test exact Add/Sub aliases
against both operands. Fixed malformed-buffer tests check precondition failures.
Tests contain their own permanent fixture values and never read C++ files.

Verified: native `go test ./mpnum`, native `go test -race ./mpnum`, coverage
run (97.4% of statements), and test-binary cross-builds
for `GOOS=js GOARCH=wasm` and `GOOS=wasip1 GOARCH=wasm`. Cross-builds are
compilation evidence, not wasm execution. Uncovered arithmetic branches at that
measurement are division's source-unreachable estimate/borrow guards.

## Ownership, contracts, and prerequisites

Public operands/output slices are borrowed and never retained. Four reusable
scratch slices belong to the natural manager. Calls to Div on one manager must
be serialized by its owner, replacing the OpenMP critical section. String uses
local scratch as the C++ method does. Add/Sub permit exact output/input aliasing;
Mul does not (the C++ integer manager stages operands/output in temporary cells).
Shifted overlapping slices are outside the supported caller contract.

The initial MPN packet had no persistent numeral cells. The subsequent MPZ
packet below integrates the approved slab/word-page ownership contract. Neither
package imports back into `g3`. MPN scratch/formatting budget integration remains
unresolved; scratch slices are not persistent pointer-free numeral storage.

Explicit adaptations: uint32 limbs/uint64 intermediate arithmetic are fixed
across targets; source true-returning Add/Mul/Sub functions return only meaningful
Go results (normalized length or borrow). Div retains the source's unusual bool:
false in trivial fast paths, true after normalization. Preconditions panic in
place of C++ debug assertions or unreachable failures. The source `lnum<lden`
branch underflows its unsigned quotient-loop bound; the Go primitive excludes
this invalid shape, matching `mpz::quot_rem_core`'s earlier smaller-numerator
handling. Decimal rendering accepts canonical nonempty input, grows a Go-owned
string rather than using an unchecked caller char buffer, and uses unsigned
strconv formatting only in the source's single-limb snprintf branch.

All authored files and cross-build/coverage artifacts are under `golang/`.
The initial RED command ran before the coordinator supplied the stricter cache
restriction and used Go's default cache/temp paths; cache writes from that call
cannot be ruled out. Subsequent commands set GOCACHE, GOTMPDIR, and TMPDIR to
`golang/.cache` and all explicit build outputs are there.

## MPZ lifecycle and Add/Sub prerequisite packet

Status: **partial MPZ source unit**. Source notices (Microsoft 2006, Leonardo
de Moura) are retained. Int is scalar `{val int32, cell numericCellRef}`; the
numeric-cell ref is a distinct uint32 in release and uint64 generation/slot in
`g3checked`. Persistent cell records contain only `arena.VectorRef`, size, and
requested capacity. An owner-local concrete `numericCellSlab` and `arena.Words`
share one configured budget. Payload geometric capacity (e.g. 8) never replaces
the source requested capacity (6), preserving the source scratch growth policy.

| Source operation | Go counterpart / status |
| --- | --- |
| constructor | `NewIntegerManager`: four scratch cells capacity6, argument size1; intMin via signed set; two64 via unsigned MAX set then source Add(one) |
| destructor | `Close`: payload cleanup before cell recycling; drop scratch/constants and arena owners; outstanding caller numerals invalidated |
| `allocate`, `deallocate`, `ensure_tmp_capacity` | `allocate`, `free`, `ensureTmp`; requested growth `(3*size+1)>>1` preserved with checked width |
| `mpz(int)`, `mk_z`, `is_small` | `NewInt`, `Int.IsSmall` |
| `set_i64`, `set_big_i64`, signed int64 setter | `SetInt64`, `setBig64` |
| `set_big_ui64`, unsigned int64 setter | `SetUint64`, `setBig64`; source strict `< INT_MAX` threshold preserved |
| unsigned digit-vector setter | `SetLimbs`; high zero limbs removed, one limb uses source unsigned-int setter's inclusive `<= INT_MAX` behavior |
| `big_set`, copy setter | `Copy`, including self-copy and destination capacity reuse |
| move setter | `Move`; source cell transfer and unchanged moved-from val, plus explicit self-move no-op |
| `swap` | `Swap` |
| `del`, `reset` | `Del` preserves inline val after releasing cell; `Reset` also zeroes val |
| `neg`, `abs` | `Neg`, `Abs`, including INT_MIN big promotion |
| `get_sign_cell`, `big_compare` | `signCell`, `Compare`, `compareCells`; source special INT_MIN constant and one-limb argument buffers |
| `add`, `sub`, `big_add`, `big_sub`, `big_add_sub` | `Add`, `Sub`, `addSub`; indexed source Algorithms A/S, sign selection, compare/subtract, normalized scratch publication |
| private scratch `set<IDX>` | `publishTmp` / `publishTmpAt` implement IDX0 and IDX1 result-cell normalization/transfer |
| `is_abs_uint64`, `big_abs_to_uint64` | `absUint64` with explicit oversized error |
| `is_uint64`, `is_int64`, `get_uint64`, `get_int64` | `IsUint64`, `IsInt64`, `Uint64`, `Int64` |
| `display`, `to_string` | `String` semantic decimal rendering; stream API replaced with string |

Explicitly unimplemented source methods/overloads: `cell_size` (representation
replaced by arena accounting), `allocate_if_needed`, `ensure_capacity`,
`normalize` (standalone; publication normalization is translated), `size_info`,
`sz_lt`, `precise`, `field`, `inc`, `dec`,
`gcd` (array and extended overloads), `lcm`, `divides`,
`inv`, `bitwise_or`, `bitwise_and`, `bitwise_xor`, `bitwise_not`, decimal-string
`set`, the named int32/uint32 setter overloads (their behavior is available through
SetInt64/SetLimbs), `dup`, `is_pos`, `is_neg`, `is_zero`, `sign`, `is_nonpos`,
`is_nonneg`, named `eq`/`lt`/`neq`/`gt`/`ge`/`le` wrappers (ordering available
through Compare), `is_uint`, `get_uint`, `is_int`, `get_int`, `get_double`,
`display_smt2`, `hash`, `is_one`, `is_minus_one`, `power`, both `is_power_of_two`
overloads, both `machine_div2k` overloads, both `mul2k` overloads,
`power_of_two_multiple`, `log2`, `mlog2`, `bitsize`, `is_perfect_square`,
`prev_power_of_two`, both `root` overloads, `is_even`, `is_odd`, `decompose`.
GMP/MSBIGNUM-specific operations remain inventoried platform alternatives,
not implementations. Standalone `u_gcd`/`u64_gcd` are now translated as U32GCD/U64GCD.

MPZ tests were authored before implementation, producing undefined
IntegerManager/Int/function errors (RED). Implementation made them pass (GREEN).
Active `src/test/mpz.cpp` assertions ported: `bug3`'s INT_MIN negation equals
INT_MAX+1; `tst_int_min_bug`'s UINT64_MAX minus INT_MIN equals
18446744075857035263; `bug1`'s exact large input subtracts itself to zero.
The bug1 decimal fixture is parsed only by independent test setup; decimal
parsing is not claimed ported. The remaining source assertions are not covered
by this packet. Added deterministic oracle coverage: 300 signed int64 cases
with Add/Sub each using separate destination and both input aliases; 200 signed
magnitude cases up to 32 limbs with copy, ordering, and Add/Sub. Added lifecycle
tests check requested/pool capacities, pointer freedom, self-copy, copy isolation,
move/swap, repeated Del/Reset, boundary conversion, constructor allocation
failure unwind, failed destination publication, and checked stale generations
after cell reuse.

Release native tests pass (combined MPN+MPZ 89.5% statement coverage); native
checked tests and both release/checked race tests pass. js/wasm and wasip1/wasm
release/checked test binaries cross-build. These are compilation gates, not runtime
wasm execution. All MPZ packet writes/cache/output are inside `golang/`.

Ownership limits: shallow-copying a big Int or transferring it to another
manager is prohibited. Owner identity is not embedded; matching numeric bits
from different managers cannot detect foreign use. A future public wrapper must
validate owning context. Copy here traverses one owner's source cells; a future
explicit cross-owner copy must traverse the source service. Confinement replaces
OpenMP locking. Errors during allocation preserve destinations: fresh storage
is allocated before old cells are dropped, and replacement scratch is reserved
before publication. This differs from C++ failure behavior but prevents partial
publication under the plan's injected budget failures. Failed allocations may
retain reusable arena capacity; Close proves complete budget release.

MPZ persistent cells, payloads and manager scratch cells are accounted by arena
budgets. Cold owner headers, MPN reusable scratch, and String's local copied
digits/decimal formatting scratch remain unaccounted and **unresolved**, so this
packet does not satisfy the full numeric budget/lifecycle gate. Source
`get_uint64` behavior is preserved: negative small values cast modulo 2^64,
negative big values return absolute magnitude; callers use `IsUint64` for
nonnegative-fit checking. Width-safe magnitude extraction for INT64_MIN avoids
the C++ signed-negation undefined behavior while preserving mathematical value.

## Concrete storage conversion

The user's explicit no-generics requirement overrides the earlier generic
storage sketches. MPZ now uses ordinary numericCellSlab, numericCellPage,
and numericCellSlotState declarations. There are no type parameters, generic
instantiations, generic templates, or reflect.TypeFor calls in mpnum. The
reviewed slab algorithm's allocation, page growth, zeroing, free-list order,
generation ledger, retirement, failure-before-publication, trim/reset/close and
old+new directory peak budget charges are preserved. Configuration, statistics,
slots, generations, budgets, and errors retain arena's concrete scalar contracts.
Words retains the concrete NewWords API supplied by arena. Runtime reflect.TypeOf
is confined to construction/accounting and pointer-free record audits; records
contain neither interfaces nor unsafe pointers.

Concrete tests were migrated first and failed with undefined
newNumericCellSlab/arena.ValidateRecordType (RED). The concrete implementation
and the arena agent's ValidateRecordType made the unchanged arithmetic/lifecycle
suite pass (GREEN). Added tests cover page/slot growth, generation reuse and
retirement, failure invariance, final-layout-versus-peak directory budgeting,
record pointer-freedom in checked/release layouts, zero allocations for warmed
cell allocate/write/recycle, and zero allocations for warmed MPZ Add/Sub.
Native release and checked race suites, vet, and release/checked js/wasm and
wasip1/wasm test-binary builds pass. Arithmetic behavior and previously documented
budget/lifecycle limitations are unchanged. All writes/cache/artifacts remain
inside golang; no later arithmetic packet is included in this conversion.

## Internal MPZ multiplication and division packet

Status remains **partial MPZ**. The earlier unimplemented-method inventory now
excludes the translated methods below; remaining items there and deferred MPQ/
rounding/higher numerical managers are still unimplemented. The source methods
and actual tests were read before porting; no C++ reference experiments ran.

| Source operation | Go counterpart |
| --- | --- |
| `mul`, `big_mul`, mpn `mul` | `Mul`, `multiplyCells`; inline int64 product and source Algorithm M |
| `machine_div_rem`, `big_div_rem` | `MachineDivRem`, `quotRemCore` quotient+remainder mode |
| `machine_div`, `big_div` | `MachineDiv`, `quotRemCore` quotient-only mode |
| `rem`, `big_rem` | `Rem`, `quotRemCore` remainder-only mode |
| `div` | `Div`; source negative-numerator Euclidean quotient adjustment |
| `mod` | `Mod`; source negative-remainder adjustment to nonnegative modulus |
| `quot_rem_core` | `quotRemCore`; source sign-cell extraction, smaller-numerator guard, both scratch capacities, sign application and source publication order |
| mpn `div`, normalization, `div_1`, `div_n`, unnormalization | `divideCells`, `divideCellOne`, `divideCellMany`; indexed source Algorithm D over owned Words |

No production arithmetic uses math/big, Go generics, raw limb slice borrowing,
or unsafe. uint32 limbs and explicit uint64 intermediates are unchanged.
Four lazy manager-owned Words refs replace mpn's u/v/ms/ab buffers. Their
initial reserved capacity is 16 words, with geometric pool growth, corresponding
to the source sbuffer's 16 inline words and doubling expansion. Logical lengths,
normalization shifts, estimate correction, Algorithm M multiplication/Algorithm S
subtraction, and add-back control flow are retained. Source scratch requested
cell growth remains `(3*size+1)>>1`. Close frees division buffers explicitly
before closing Words. Their descriptors, page payload, directory and free-list
metadata share the manager's arena budget. This resolves MPZ division scratch
budgeting; standalone NaturalManager scratch, String formatting, and cold header
budgets remain unresolved as previously recorded.

Storage failure adaptation: paired quot/rem preallocates replacement scratch
before either result publication, and Mul publishes only after capacity is ready.
Div/Mod stage owner-local numerals until all adjustment allocations succeed,
then Move commits the result. Every staged numeral is explicitly deleted on
failure. Staging increases peak owned cells/capacity and changes physical
allocation/deletion timing from C++; no allocation-profile equivalence is claimed.
The source divisor/output reread is emulated using staged work when c==b, so
normal arithmetic and the source alias defects below remain exactly observable.
Division by zero returns classified ErrDivideByZero and leaves destinations unchanged;
source division by zero has no valid mathematical contract (native undefined
arithmetic/unreachable paths). This explicit development error is not UNKNOWN.

TDD: tests first failed for undefined Mul/MachineDivRem/MachineDiv/Rem/Div/Mod
(RED); the exact algorithms made them pass (GREEN). Source tst1's active large
integer arithmetic/comparison assertions are checked through Mul/Add/Neg/Compare;
its decimal parsing is independent fixture setup, not a claimed setter port.
Source tst2/tst2b only print: added exact expected 2^64 square and 2^256 after
three input/output-alias squares. Added finite independent cases include 2352
small signed nonzero-divisor pairs, 500 signed operands of 1..12 limbs with
all single-output input aliases and swapped two-output input aliases,
INT32/INT64 boundaries, Algorithm D quotient correction/base-estimate/add-back,
and the source smaller-numerator length guard. Independent math/big uses distinct
oracle values for truncating QuoRem versus Euclidean DivMod. Multiplication and
paired division are checked for zero warmed Go allocations. Reserved division
scratch capacity and complete Close budget release are explicitly asserted.

Predictable failure sweeps: 32 paired-division reservation positions; 2 Euclidean
operations x 3 destination shapes x 40 reservation positions; 3 multiplication
destination shapes x 16 positions. Every failure preserves destinations and alias
inputs, retains no extra live numeric cell, retries to the exact expected result,
and releases its whole budget on Close. These are finite deterministic injected
budget checks, not claims about arbitrary Go runtime OOM/cancellation recovery.

Known preserved source alias defects (independently derived from executable
source control flow, not C++ observed goldens):

- `Div(-26,7,b)` with c==b yields -2; `Div(-26,-7,b)` yields 2. The source checks
  the sign of the overwritten quotient as if it were the original divisor.
- `Mod(-26,+/-7,b)` with c==b yields 0 because subtraction sees the remainder
  as both operands. Mathematical Euclidean tests use unaffected destinations;
  source-defect tests separately freeze these values, including large operands.
- `MachineDivRem` with q==r ends in remainder on the small/general path, but
  ends in zero when denominator limb length exceeds numerator length because
  source writes remainder then resets quotient. No new alias restriction is
  introduced. Mathematically independent paired tests use distinct outputs.

Source numeric parser, named comparison convenience wrappers, power/two-shift,
array/extended GCD/bitwise/root/log and other inventory entries remain unported. This packet
does not complete the manager, full arithmetic gate, or full-fork port.

Verified final packet gates: native release and checked race tests pass; vet
passes; release/checked js/wasm and wasip1/wasm test binaries cross-build.
Combined mpnum statement coverage is 90.7%; this includes unreachable guard and
unexercised error branches and is not claimed complete path coverage. Cross-build
results are compilation evidence, not wasm runtime execution. Files written:
mpz_mul.go, mpz_div.go, mpz_mul_div_test.go, the approved mpz.go shared helper/
scratch changes, and this status document. All artifacts/cache remain in golang.

## Selected internal Lehmer GCD and fused arithmetic packet

Status remains **partial MPZ**. No shared mpz.go, manifest, or arena changes
were needed. New files: mpz_gcd.go and mpz_gcd_test.go. Source methods and
conditional selection were read before translation: the internal backend
defines LEHMER_GCD, while EUCLID_GCD/BINARY_GCD/LS_BINARY_GCD are inactive.
The port preserves that choice; inactive alternatives are not substituted.

| Source operation | Go counterpart |
| --- | --- |
| scalar `gcd_core`, `u_gcd` | `U32GCD`; explicit uint32 binary algorithm specialization |
| scalar `gcd_core`, `u64_gcd` | `U64GCD`; explicit uint64 binary algorithm specialization |
| `addmul`, `submul` | `AddMul`, `SubMul`; source inline +/-1 shortcuts and temporary product/cleanup |
| `gcd(a,b,c)` | `GCD`; source small-int/INT_MIN exclusions, zero fast paths, exact Lehmer A/B/C/D matrix update, B==0 remainder fallback, small-divisor finishing loop |

Lehmer coefficients and top-word approximations retain explicit int64 widths;
32-bit cells and existing exact multiplication/remainder managers supply all
full-value arithmetic. No production math/big, Go generics, unsafe, raw slice
borrows, new numerical caches, or approximate arithmetic were introduced.
All owned working numerals are explicitly deleted in source cleanup order,
including on allocation failures. Source zero-case Copy+Abs is staged before
Move so INT_MIN promotion failure cannot publish a partial destination; this
changes temporary storage/cleanup timing, not arithmetic. Main source GCD
publishes only at its final swap/set, so destination aliases preserve inputs
through the work phase. Logical temporary cells and all arithmetic scratch
retain existing arena budget/growth policies. The previously documented MPN
formatting/header budget limitations are still unresolved.

TDD tests first failed with missing GCD/U32GCD/U64GCD/AddMul/SubMul (RED), then
the algorithms passed (GREEN). Active rand_tst_gcd(50,3,2) coherence is retained
as 50 finite exact GCD checks with independent math/big expectations. Its
rand/168-prime/Power generator is adapted to an independent fixed-seed generator
over the first 16 source primes, exponents 1..2, inclusion ratio 1/3 and both
signs. This is labeled a bounded fixture-generation adaptation: production Power
is neither used as an oracle nor claimed implemented. The source inactive
tst_gcd stress loop remains inventoried, not silently activated. Active
tst_gcd_bug prints GCD(INT_MIN,INT_MIN); added independent expectation is 2^31.

Additional independent checks: every signed pair in [-20,20]^2; scalar unsigned
width boundaries; 250 signed operands up to 16 limbs with both destination
aliases; zero/equal/one cases and 400-step consecutive Fibonacci inputs for
multi-update Lehmer batches. AddMul/SubMul cover 8^3 signed boundary triples,
both operations, and separate/all three operand destinations. Failure sweeps
check 3 GCD destinations x 40 reservation positions and 2 fused operations x
4 destinations x 24 positions, plus zero-case INT_MIN promotion failure. Failed
operations preserve values and live-cell counts, retry to independently exact
answers, and Close returns the budget to zero. Warmed GCD/AddMul/SubMul are
verified to allocate zero Go objects on the tested bounded workload.

Array GCD, extended GCD, LCM, divisibility, decimal parsing, power/shifts,
bitwise/root/log, deferred MPQ and other prior inventory entries remain unimplemented.
No new source defect was inferred for these selected methods; bounded passing
tests do not establish correctness for all unbounded inputs or close the port.

Final packet validation: native release/checked race tests pass; vet passes;
release/checked test binaries cross-build for js/wasm and wasip1/wasm. Combined
mpnum statement coverage is 90.4%, not complete path coverage. No wasm runtime
execution is claimed from these builds. Every write/temp/cache/output is within
golang, including command TMPDIR/GOTMPDIR/GOCACHE settings.


## Basic MPQ lifecycle, normalization and arithmetic packet

Status remains **partial MPQ and MPZ**. New files are mpq.go and mpq_test.go;
no shared integer arithmetic helper was changed. Source mpq.h/mpq.cpp methods
were read before translation. Rational is the concrete pointer-free pair of
Int numerals; NewRational constructs the source 0/1. The raw Go zero value is
diagnosed with ErrUninitializedRational, rather than silently changing its
invalid denominator into source default storage. Owned big values prohibit
shallow copies and cross-manager transfer; Copy, Move and Swap are explicit.

Translated source methods: manager constructor/destructor; numeral default
construction; is_small; reset/del; copy/set integer, unsigned-denominator
fraction and integer parts; move/swap; neg/abs; normalize; rat_add/rat_sub,
integer fast paths and add/sub wrappers; rat_mul/mul; rat_lt and ordering through
Compare; equality; div with its divisor/output alias branch; both inv forms
through Inv(a,out); integer/fraction display through String. Integers exposes
the owned base manager for already translated inherited integer operations.
Unimplemented MPQ methods include parsers/scientific notation, Power,
floor/ceil, prev_power_of_two/log, decimal/SMT2/display variants, getters and
numerator-only swaps, scalar mixed arithmetic overloads, rational AddMul,
GCD/LCM/divisibility, approximate conversions, and the separate rational wrapper.
No partial parser is exposed. Source scientific parsing requires Power;
source prev_power_of_two tests also require Floor and integer Log2.

RationalConfig.Synchronized selects the source SYNCH local-temporary branch;
false uses retained nTmp/addTmp1/addTmp2/ltTmp1/ltTmp2. These retain the source
GCD reduction, cross-products and positive-denominator normalization. Both modes
are owner-confined; the option does not provide concurrent arithmetic. The
RationalManager header, including scalar scratch, is charged by reflect.Size
before base-manager construction and released after source destructor order.
Persistent big scratch uses the existing budgeted concrete cells/Words. Referenced
base/arena headers are charged by their own owners, never again by MPQ. Standalone
MPN slice scratch and String formatting ownership remain separate open gaps.

Storage adaptation: multi-component writes are staged in explicit owner-local
working numerals and published only after normalization succeeds. This adds
peak-budget working cells and changes allocation/destruction timing, while
retaining source arithmetic branches and successful alias results. Undefined
zero divisor/inverse/denominator use returns ErrDivideByZero. No new source
arithmetic defect was inferred in these selected normalized-rational methods.

TDD RED was the undefined Rational/RationalManager API in permanent tests;
implementation then reached GREEN. Active source tst0 divides (2/3)/(4/3)
into its divisor and asserts 1/2; bug1/bug2 assert in-place inverses of +/-2.
Active tst1 inherited big-integer arithmetic and comparisons are retained with
independent fixture setup instead of advertising the missing source parser.
Parser, decimal print-only and prev_power_of_two source cases remain deferred.
Independent math/big.Rat tests add 250 signed multi-limb fraction pairs, each
Add/Sub/Mul/Div with separate/left/right destination, reciprocal, comparison and
equality; zero/negative denominators, INT_MIN, reduction and all-self aliases.
Lifecycle tests cover explicit default, raw-zero rejection, Copy isolation,
Move/Swap, repeated Del/Reset, closed owners and pointer-free record shape.

Failure sweeps test constructor rollback, 5 arithmetic operations x 3 aliases
x 48 reservation positions, 3 copy/set paths x 40 positions, and separate
cold small-fraction setter and INT_MIN inverse-promotion sweeps in both scratch modes. Inv and a small setter
can reuse all storage warmed by fixture normalization, so those sweeps do not
incorrectly require a new reservation failure. Other paths test both injected
failures and success. Failures preserve public operands/output, retry to exact
independent answers, and Close restores budget zero; unsynchronized scratch may
legitimately retain owner-local cells after failure. Warmed Add/Sub/Mul/Div and
Compare allocate zero Go objects on the tested bounded workload; tests separately
verify retained cache use versus local-temp cleanup. Bounded evidence does not
establish unbounded correctness or complete the numeric source unit.

SetParts follows the exact mpq.h integer-parts overload: a negative source
denominator is copied and both signs are flipped before normalize. This is
not an extension beyond its source sign handling. The unsigned-denominator
SetFraction overload retains its source nonzero-denominator precondition,
classified as ErrDivideByZero instead of an unchecked assertion/undefined path.

Final packet validation with integrated integer/arena owner headers: native
release/checked race suites pass, as do the added cold inverse-promotion race
regressions in both builds; vet passes; release/checked binaries cross-build
for js/wasm and wasip1/wasm. Cross-builds are compilation evidence, not wasm
runtime execution. All edits, build caches, temp directories and output artifacts
remain under golang. No Go generics, unsafe, production math/big, source-file
runtime fixtures or global manager were introduced.
