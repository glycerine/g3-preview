Selected destructive capacity and base-two Power now preserve caller publication;
see ../MPZ_SETTER_PORT_STATUS.md for current scope and remaining staging work.

<!-- Historical packet evidence follows. The MPZ source-publication follow-up
     supersedes earlier fixed scratch-count and automatic retry claims.
     Higher-level staged wrappers remain explicit fidelity work. -->

# Numeric foundation packet

This file is a chronological packet log: later entries and the current
`../port_manifest.json` supersede early missing-method lists. The selected
internal integer and rational algorithms now include parsing, powers, roots,
bitwise operations, logarithms, GCD/LCM, rounding and owned rational facades.
Current public integer routes are mapped in
[MPZ_FACADE_EXTRA_PORT_STATUS.md](../MPZ_FACADE_EXTRA_PORT_STATUS.md);
storage/formatting and number-theory packets have their own source audits.
Remaining backend alternatives, trace/debug diagnostics, binary rationals,
modular arithmetic and complete solver consumers are not inferred complete.

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
OpenMP locking. Selected allocation/publication routes now preserve the C++
action prefix: insufficient scratch and big-set destination cells are freed
before replacement allocation; result setters publish before replenishing
missing scratch. ensure_capacity separately retains old storage until replacement.
Failed destination pointers become owned, permanently dead descriptor tokens
with guarded reads and explicit cleanup. No automatic scratch repair or retry
is promised. See ../MPZ_SOURCE_PUBLICATION_PORT_STATUS.md.

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

Source publication follow-up: paired quot/rem publishes the quotient before the
remainder, without preallocating replacement scratch. Mul publishes its result
before replenishing scratch. Div/Mod mutate the caller destination and reread
aliases at the actual source adjustment action. Source undefined failure states
receive guarded Go cleanup; valid completed results remain readable. Source
allocator topology and exact malloc profiles are still open.
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
destination shapes x 16 positions. The reconciled tests audit independently
named live cells and Words before formatting or teardown, including transferred
results, freed scratch and guarded failed tokens. Successful endpoints retain
the exact arithmetic oracle; failure paths do not require implicit scratch
repair. Close releases the whole charged graph. These are finite deterministic injected
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
Unimplemented MPQ methods include parsers/scientific notation,
decimal/SMT2/display variants, getters and
numerator-only swaps, scalar mixed arithmetic overloads, rational AddMul,
GCD/LCM/divisibility, approximate conversions, and the separate rational wrapper.
No partial parser is exposed. Source scientific parsing requires Power,
completed by the later power/shift packet;
source prev_power_of_two dependencies are completed by the rounding/log packet below.

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
Parser and decimal print-only source cases remain deferred; prev_power_of_two
source cases are completed in the later packet.
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


## Integer bit positions and rational rounding packet

Status remains **partial MPZ and MPQ**. New mpz_log.go, mpq_round.go and
mpq_round_test.go only; no shared arithmetic, arena, ownership or manifest
changes. Source mpz.cpp/mpq.cpp bodies, mpz.h/mpq.h wrappers and util.cpp
unsigned log2 helper were read before translation. No production Power or
shift method was introduced as scaffolding.

| Source method | Go counterpart |
| --- | --- |
| unsigned helper log2 | private log2Uint32, exact 16/8/4/2/1 mask algorithm |
| mpz log2 / mlog2 / bitsize | Log2 / MLog2 / Bitsize, top 32-bit limb and size computation |
| mpz prev_power_of_two | IntegerManager.PrevPowerOfTwo delegates Log2 |
| mpq floor / ceil to mpz | FloorToInt / CeilToInt |
| mpq floor / ceil to mpq | Floor / Ceil, denominator reset to one |
| mpq prev_power_of_two | RationalManager.PrevPowerOfTwo, scoped integer Floor then integer wrapper |

Log2 returns zero for nonpositive input; MLog2 returns zero for nonnegative
input. These are explicit source branches, not positivity preconditions or
mathematical real logarithms. Bitsize follows the source statistics convention
and returns one for zero. Small negative magnitude uses int64 before narrowing
to uint32; this makes the intended INT_MIN magnitude defined instead of relying
on source signed unary-negation overflow. The source top-limb position and
bit-size calculations are checked before narrowing to uint32 and report bounds
errors if their source unsigned counters would overflow.

Floor/Ceil retain the source is-integer copy branch; otherwise MachineDiv
truncates toward zero, then the captured numerator sign determines subtraction
or addition of one. The initialized normalized-rational invariant ensures a
noninteger fraction has nonzero remainder; no substitute algorithm or extra
remainder check was inserted. Both integer and rational output forms preserve
successful source output aliases. Work numerals are staged for failure atomicity;
this adds temporary peak ownership and changes cleanup timing, while persistent
storage and scratch stay with existing budgeted managers. PrevPowerOfTwo cleans
its scoped integer on every return. No additional retained numerical cache,
production math/big, Go generics, unsafe or global manager was introduced.

TDD permanent tests failed with missing rounding/log APIs (RED), then the
translation passed (GREEN). All 16 active source tst_prev_power_2 assertions
are retained in both scratch modes. All 2000 active source tst_log2 bound cases
are retained: integers 0..999 and 1000 consecutive values after the actual big
fixture. Test-only mathematical powers supply the bound checker, so an
unimplemented production Power is not claimed. Source fixtures are permanently
embedded; no runtime source read or reference executable is needed.

Additional tests use 500 signed exact fractions (including multi-limb values),
independent math/big quotient/remainder rounding, both output forms, separate
and same-rational destinations and integer output/numerator alias. They cover
zero, integral/reduced cases, negative fractions, zero/negative logarithm
semantics, INT_MIN/INT64_MIN magnitudes and 200 independent signed log inputs.
Log queries retain the same budget usage and succeed with FailAfter(0), proving
that these read-only operations do not need new owned storage. Invalid raw
rationals and closed owners are diagnosed. Two modes x 5 operations x 2 output
alias forms x 48 budget reservation positions check failure publication,
retry exact results and Close budget zero. Warmed rounding and PrevPowerOfTwo
allocate zero Go objects on the bounded workload.

Parsers, scalar mixed rational methods, decimal formatting and previously
deferred higher arithmetic remain unimplemented. Power and shifts are covered
by the later packet, including source domain/defect handling. Standalone
NaturalManager and String scratch/output ownership remains the separately
approved next storage packet, not a closed gate here.

Final rounding/log validation: native release/checked race suites pass; vet
and the required portcheck no-generics/source-gate audit pass; release/checked
test binaries cross-build for js/wasm and wasip1/wasm. Combined mpnum coverage
is 88.9%, not complete path coverage. Wasm cross-builds are compilation evidence,
not runtime execution. All command cache/temp/output and project edits remain
under golang. No manifest or git changes were made by this packet.


## Selected internal integer Power/shifts and rational Power packet

Status remains **partial MPZ and MPQ**. New mpz_power_shift.go,
mpq_power.go and mpz_power_shift_test.go only; no shared helpers, allocator,
ledger or git changes. Full selected method bodies and helper allocation actions
were read before translation: mpz.cpp internal power, is_power_of_two,
ensure_capacity, normalize, machine_div2k, mul2k; mpz.h copy wrappers and
allocate_if_needed; mpq.cpp power; source tst_2k and related active test cases.

Translated: IntegerManager.Power with actual source small 2/0/1 fast paths;
IsPowerOfTwo with scalar and high-limb/lower-zero checks; MachineDiv2k and Mul2k
with same-pointer in-place and separate-copy destinations; source requested
capacity-six ensure/promotion/reallocation and normalization; RationalManager.Power.
Both generic Power implementations retain the uint32 mask, bit selection,
source temporary copy and final extra square even when unused. The base-two
fast path builds exact limb bits directly and retains source allocated big
representation at p=31. Shifts preserve source small int64 multiply, signed
small division and indexed whole-limb/bit shift loops. Source cell requested
capacity stays separate from Words pool geometry. No production math/big,
generics, unsafe, alternate exponentiation algorithm or new retained scratch
was introduced.

Source domain/defect handling is explicit:

- PowerDomainError carries the exact uint32 exponent and reason, and unwraps
  to ErrInvalidPower or ErrNonterminatingPower. Generic uint32 mask Power loops
  would never exit after mask wraps to zero when p>=2^31; these inputs fail
  before any allocation/publication. Source integer small base 0/1/2 finite
  fast paths are selected first, retaining behavior when storage is representable.
- Integer small 0^0 violates the actual source assertion and returns the typed
  invalid-domain error. Rational Power has no such assertion/fast path and
  preserves its actual 0^0=1. The integer precondition is not silently added to
  the rational method.
- Source signed `1 << 31` has no portable C++ contract. The selected 32-bit
  two's-complement baseline uses an explicit uint32 limb shift for base-two
  Power and int32 narrowing for MachineDiv2k's signed divisor. Thus small inline
  INT_MIN divided by 2^31 follows the inferred source control flow and produces
  +1; a big-cell INT_MIN instead follows magnitude shifting and produces -1.
  Permanent regressions label this as source-derived, not C++-observed. This
  source defect is not silently repaired or generalized to other representations.
- New limb-size additions are checked with uint64 before narrowing; predictable
  namespace/budget failures leave output/input aliases unchanged. Huge finite
  base-two powers and left shifts are not misclassified as mask nontermination.

Storage adaptation: public Power and shift outputs are staged in explicit owned
working numerals, preserving all successful source aliases and withholding
publication until arithmetic completes. Working cells are cleaned on every
failure. This changes peak budget and source destructor timing; the source final
unused square can still fail and must not be omitted. Existing base-manager
Words/cells and multiplication scratch account all numerical work, while the
previous standalone MPN/String scratch/output ownership gaps remain open.

Tests first failed for undefined production APIs (RED), then passed (GREEN).
All 72 actual tst_2k calls are permanent fixtures, each with separate and input
output aliases. The source ENSURE coherence uses real Power(2,k) plus Mul or
MachineDiv; additional independent math/big limb expectations avoid mirrored
arithmetic or million-bit decimal rendering. The actual -7 left shift by
1,000,000 bits is finite and runs under the one-megabyte owner budget. Source
print-only tst_pw2 values p=0..127 and tst_div2k_bug receive explicitly added
mathematical expectations, not invented source assertions.

Independent bounded coverage adds 200 signed multi-limb Power/left/right-shift
triples and both destinations; 100 rational pairs with both destinations and
scratch modes; scalar width/promotion boundaries at shifts 0/1/30/31/32/33/
63/64/65; IsPowerOfTwo positive/nonpositive/multibit cases; UINT32_MAX shift
and power-budget boundaries; typed zero/nontermination errors and same-output
error aliases; physical small/big INT_MIN defect distinction. Two modes x four
operations x two destinations x 64 reservation positions check untouched
operands/output on failure, independent exact retries and Close budget zero.
Warmed integer/rational Power and shifts allocate zero Go objects on the tested
bounded workload; raw rational initialization and closed-owner errors are tested.

Method inventory still excludes power_of_two_multiple, rational IsPowerOfTwo
wrapper, parsers/scientific input, decimal rendering, scalar mixed rational
methods and the previously recorded higher arithmetic/diagnostic source methods.
Full parser remains unadvertised even though its rational Power prerequisite is
now present. Standalone MPN scratch and caller-buffer FormatInto are approved
as separate future storage work; escaped Go string output lifetime must not be
claimed accounted through temporary budget reservations.

Validation correction: earlier sub-agent commands used `-tags=checked`, which
is not the checked-handle build tag. The actual tag is `g3checked`. This packet
reruns the complete mpnum suite and all earlier arithmetic/lifetime regressions
with the actual tag, alongside release, and cross-builds the actual two modes.
Historical incorrectly tagged commands alone are not checked-mode evidence.

Final Power/shift validation: full native release race (37.6s) and actual
`-tags=g3checked` race (38.5s) suites pass, including all earlier numeric packets.
Both native modes pass vet; the portcheck source-gate/no-generics audit passes.
Test binaries cross-build for js/wasm and wasip1/wasm in release and g3checked
modes; these are compilation evidence, not wasm runtime execution. Combined
mpnum statement coverage is 89.5%, not complete path coverage. Every edit,
cache, temp directory and artifact remains within golang. Earlier incorrectly
tagged commands are supplemented by these actual checked-handle gates.

## Complete selected MPZ/MPQ text-setter packet

Status remains **partial MPZ/MPQ source units**, with the full selected source
text setters now implemented. This supersedes earlier unimplemented-parser
inventory entries; it does not complete the separate public/global rational
wrapper or numerical manager diagnostics. New files mpz_parse.go,
mpq_parse.go and mp_parse_test.go; the approved source string setups in
mpz_power_shift_test.go and mpq_round_test.go now call production SetString.
Their fixture constants and independent expected values remain intact. No shared
arithmetic/arena helper, manifest or git change was made.

IntegerManager.SetString(out,text) translates all of mpz.cpp's actual
set(mpz&,char const*) body. RationalManager.SetString(out,text) translates all
of mpq.cpp's corresponding integer/fraction/decimal/scientific body, including
source temporary scopes, final sign adjustment and normalization. There is no
text buffer/length overload in these managers: the source length/payload setter
is the limb setter, already covered by IntegerManager.SetLimbs. No invented
text overload or partial parser is advertised. Complete arithmetic dependency
closure uses existing Mul/Add/Neg/Reset/Copy/SetParts/Rational Add/Mul/Div/Power
and normalize; no new getter or global-manager dependency was introduced.

Input is borrowed string bytes, never retained/copied into numeral storage.
Finite Go length supplies the safe boundary of the source C pointer; embedded
NUL stops all phases and hides later characters. Only leading ASCII space is
skipped for sign detection. The first remaining '-' determines the whole-value
sign; all other nondigits are ignored where the source ignores them. The parser
uses bytes, not Unicode digits/whitespace or a stricter replacement grammar.
MPZ ignores punctuation throughout. MPQ stops its initial digit scan at the
first slash/dot/e/E, then follows the exact suffix branches:

- Fraction denominator digit scanning skips other punctuation/signs and stops
  at e/E. Thus 1/-2 is 1/2 and 1/2/3 is 1/23. An e/E after a fraction marker
  triggers the source mixed-notation exception, including exponent zero.
- Decimal scanning similarly skips punctuation, so 1.2/3 is 123/100; each ASCII
  fractional digit multiplies the denominator-of-the-fraction by ten. Whole
  numerator plus that fraction is formed through the actual rational operations.
- Scientific sign is recognized only when '-' immediately follows e/E.
  Nondigits except slash are skipped during exponent accumulation. A slash in
  the exponent triggers the source mixed-notation exception; a slash in the
  decimal fractional scan is instead ignored, as above.
- The exponent accumulator retains unsigned uint64 wrap before its final
  UINT32_MAX comparison. Exponent 2^64 wraps to zero; 2^64+1 wraps to one.
  Generic source mask nontermination remains a distinct PowerDomainError.

The source history defect is preserved explicitly: entry resets numerator but
retains prior denominator. Scientific input without a decimal/slash does not
reset it. Parsing 1e2 into prior 1/3 produces 100/3; 1e0 produces 1/3. Staged
work copies the original denominator, rather than silently substituting a fresh
0/1 value. Plain integer and decimal forms still reset/rebuild denominator at
the actual source points. Decimal digit arithmetic also retains the source big
representation of parsed -2147483648; its subsequent right-shift31 uses the big
magnitude path (-1), distinct from the source inline INT_MIN defect (+1).

ParseError{Reason} distinguishes the three source exceptions: mixed notation,
exponent too big, and zero fraction denominator. errors.Is unwraps to
ErrMixedNotation, ErrExponentTooLarge or existing ErrDivideByZero; errors.As
retains the structured category. Allocation/budget/closed-owner failures and
PowerDomainError remain separately classified. Empty/junk input and incomplete
scientific exponents do not gain invented syntax errors. Source original
exceptions leave a partially changed target; the authorized storage adaptation
stages all work and publishes only after success, so every error preserves the
old destination. This adds temporary peak ownership/cleanup timing, explicitly
not the source allocation profile. Source suffix temporaries leave scope before
the final sign/normalize; MPZ temporary deletion precedes final sign as in source.
All numeric work uses existing budgeted cells/Words/reusable scratch. Escaped
String output and standalone MPN scratch gaps remain open, not covered through
transient accounting or the parser's borrowed input.

TDD tests first failed with undefined SetString APIs (RED), then passed (GREEN).
All three active set_str_bug assertions are retained. Source parsed-big tst1
arithmetic/order, bug1 self-subtraction and int_min_bug parsed expected value
are exercised. All 72 source shifts and 2000 source log bounds rerun with
production parser setup and unchanged independent expectations. Source bug2/
scoped print-only literals receive labeled independent numerical expectations;
27 embedded active rational.cpp text literals and parser identities are projected
through concrete managers without claiming the global wrapper/hash/root methods.

Independent tests add 200 signed multi-limb integers, fractions and four-place
scientific decimals per scratch mode using math/big solely in tests, plus exact
source-derived byte interruptions, signed/non-ASCII bytes, embedded NUL,
stateful prior-denominator sequences, uint64-wrap, exception priority and typed
mask-domain cases. A finite valid exponent near 2^31 is refused with only 4KB
budget growth available, avoiding unnecessary enormous source squaring while
retaining real source arithmetic and budget failure behavior. Two modes x four
text forms x integer/rational setters x 80 reservation positions verify untouched
old values, independent mathematical retry answers and Close budget zero.
Warmed setters allocate zero Go objects on the tested bounded workload;
initialized/reused small/big numerals, invalid raw rational and closed owners
are checked. Fixtures are permanent and never read source/reference binaries
at runtime. No production math/big, Go generics, unsafe or borrowed input
persistence was introduced.

Final text-setter validation: full native release race (39.7s) and actual
`-tags=g3checked` race (39.4s) suites pass, including the production-parser
source shift/log setup changes. Native release/g3checked vet and portcheck
source-gate/no-generics audit pass. Test binaries cross-build for js/wasm and
wasip1/wasm in both actual handle modes; this is compilation evidence, not wasm
runtime execution. Combined mpnum coverage is 89.8%, not complete path coverage.
All edits/cache/temp/output remain under golang. Source unit status remains
partial; formatting ownership, getters/scalar wrappers, higher arithmetic and
public rational-wrapper integration are not silently marked implemented.

## Standalone MPN and numeric formatting ownership closure

This packet supersedes the earlier MPN slice-scratch and cold-formatting ownership
notes. It remains a **partial numerical source unit**: MPN diagnostics/trace,
remaining MPZ/MPQ operations and public/context wrapper integration are not
claimed complete. Production arithmetic still uses source 32-bit limb Algorithms
A/S/M/D and repeated decimal division; no production math/big, generics, unsafe,
per-limb interfaces, or exposed arena slices were introduced.

`NewNaturalManager(arena.VectorConfig)` / `Close` replace raw Go construction for
stateful MPN operations. The standalone owner charges its own scalar/pointer
header, Words and descriptor headers/pages, and all four source division buffers
with requested initial capacity16. Requested capacities double separately from
arena physical classes. An IntegerManager child borrows its parent's existing
Words and the same Budget: it owns only its header, scratch refs and formatting
byte service, and never duplicates the Words owner charge. Integer construction
creates this source MPN member before initializing four numeric scratch cells and
constants. Constructor reservation-prefix tests and idempotent Close release
complete owner graphs; child Close leaves borrowed Words usable by the parent.

One concrete ephemeral `limbView` chooses borrowed caller buffers or scalar
Words indexing, and is never retained or returned. Public Natural Div and MPZ
`divideCells` now share one Algorithm D implementation, including source quotient
estimate correction, source zero-limb multiplication, subtraction and add-back.
Natural `Div` now returns `(bool,error)` and requires initialized/open ownership.
Its original valid-buffer assertions and non-overlap contracts remain explicit.
MPZ shorter-numerator handling, output publication order, q==r behavior, signed
quotient/remainder conventions, and documented divisor-output alias defects are
unchanged. Existing independent sign/limb/alias/failure suites still run.

Predictable-failure preparation reserves potential add-back storage before any
borrowed quotient/remainder publication. **This is a narrow storage-profile
adaptation**: ab can grow earlier, and on non-add-back paths where the source
would retain capacity16. Its retained requested capacity is bounded by source
16/doubling, and the logical add-back branch still executes only when the source
borrow test requires it. Arithmetic remains a single source Algorithm D pass;
no replay or substitute algorithm is used. A deterministic wide power-of-two
fixture requires no add-back, verifies this earlier ab growth separately from
its independent quotient/remainder oracle, and verifies warm zero allocations.
Other source division normalization/growth policies remain intact.

Dedicated retained formatting Words refs replace the source three local
`temp`, `t_numer`, `t_denom` sbuffers. Multi-limb conversion copies into owned
scratch, normalizes divisor10 by28 bits, executes source div_1, unnormalizes the
remainder, removes high zero quotient limbs and reverses the decimal segment.
Small signed/unsigned formatting uses finite scalar decimal emission equivalent
to source stream/snprintf output. There are no unbudgeted numeral copies, digit
arrays, per-call Natural owners, or intermediate numerator/denominator strings.
A lazy budgeted `arena.SmallBytes` DirectOnly owner retains the char workspace
with initial requested capacity1024 and doubling. Cold storage headers/pages and
old-plus-new growth peaks are charged; this allocator/timing profile deliberately
adapts source stack/stream storage. Byte growth copies the entire existing prefix
before freeing the old workspace, so rational denominator preparation cannot
overwrite the already formatted numerator or slash.

Natural/Integer/Rational `FormatInto` stage complete owned text then publish to a
borrowed `[]byte`. Returned length **excludes the C terminator**; exactly n bytes
are written, with no extra NUL required or written. `ErrShortBuffer` reports the
needed length and preserves destination bytes. Budget/fault errors also preserve
caller output; successfully retained scratch after failure remains charged until
owner Close. All source inputs are unchanged. Closed managers and a raw-zero
NaturalManager are diagnosed; raw-zero Rational retains its invalid-denominator
diagnostic. Natural String now returns `(string,error)`; existing Integer and
Rational String signatures are unchanged. Each wrapper builds exactly one final
escaped Go string from indexed owned bytes. That string is caller-owned host
output, explicitly outside the numeric owner's budget/lifetime: there is no
transient reservation pretending to account its escaped lifetime. Full public
context/output-service ownership remains a separate integration gate.

TDD: the migrated existing MPN assertions/oracles and initial new format tests
were written before implementation; the observed package build was RED with
undefined constructor/FormatInto/ErrShortBuffer APIs. The implementation passed
those tests and all prior source fixtures (including production parser, powers,
rounding, signed aliases and known source defects). The first package invocation
used the repository root and failed package resolution; it wrote only configured
`golang/.cache` cache/temp paths. The intended RED was rerun from golang with
GO111MODULE=on. Existing storage assertions now count four additional cold
Natural refs, force real wide-buffer growth for allocation-failure fixtures, and
warm formatting before a Power-domain budget-invariance baseline.

New permanent tests include complete constructor fault prefixes, exact nested
owner/header/payload/metadata accounting, child/parent Close isolation, finite
independent math/big division/decimal expectations, >1024 rational-prefix growth,
cold/warm three-service formatting failure sweeps, borrowed-output division
failure sweeps, real budget-limit refusal, short buffers/exact trailing-byte
preservation, small INT_MIN text, raw/closed diagnostics, returned string lifetime
past workspace replacement and Close, warm FormatInto/Div zero allocations and
exactly one warm String allocation for the final caller-owned result. Tests are
hermetic and finite; original 1000 arithmetic/2000 division-and-decimal independent
MPN cases and all exact Add/Sub aliases remain. Source arithmetic assertions are
MPZ magnitude projections and existing signed/rational source tests; additional
buffer/budget/ownership expectations are explicitly Go adaptations, not claimed
new C++ observations.

Current method mapping: `mpn::compare/add/sub/mul` keep their existing public
methods; `div` is `(bool,error)` `NaturalManager.Div`; `div_normalize` and
`div_unnormalize` are the indexed normalization/remainder blocks of `divide`
(and decimal preparation); `div_1` is `divideViewOne`; `div_n` is `divideMany`;
`to_string` is `prepareNatural` + `FormatInto`/`String`. MPZ numeric `display` /
`to_string` are `prepareInteger` + `FormatInto`/`String`; MPQ numeric rendering
uses `prepareRational` + its two public rendering methods. Ostream interfaces,
source trace/diagnostic methods and unrelated remaining numeric methods are
unimplemented; no source-unit completion is implied.

Verification for this packet: native/actual `g3checked` package suites GREEN;
native race74.308s and actual checked race73.986s GREEN; native/checked vet and
no-generics `portcheck` GREEN; full native numerical coverage89.4%; release and
actual checked JS/WASI test-binary builds GREEN. All12 new storage/format test
registrations ran and passed under actual `g3checked` JS with the existing
TMPDIR-preserving toolchain-matched wrapper. Cross-builds alone are compilation
evidence; only those stated JS registrations are execution evidence. Every
write/cache/temp/output in this packet stayed under golang/.cache or the listed
mpnum source/test/status files; no source C++ compiler/external fixture was used.
