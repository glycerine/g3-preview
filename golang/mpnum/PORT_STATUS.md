# Numeric foundation packet

Status: **partial source unit**, not completion of the numerical manager layer.

Source: `src/util/mpn.h`, `src/util/mpn.cpp` (Microsoft 2011, Christoph
Wintersteiger). The full arithmetic algorithms were read before translation.
`src/util/mpz.h`, the arithmetic call sites and `quot_rem_core` in `mpz.cpp`,
and source assertions in `src/test/mpz.cpp`/`mpq.cpp` were inspected for
preconditions. The later MPZ lifecycle packet below reads each translated method
and preserves the internal 32-bit backend. No `mpq` operation is implemented.

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

All `mpq`, `rational`, rounding/fixed-precision and higher arithmetic source
methods remain unimplemented. MPZ method coverage is recorded below. No fallback
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
and all `mpq` assertions require their real managers and must be ported later.

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
| private scratch `set<IDX>` | `publishTmp` implements currently used IDX0; IDX1 path not yet used/implemented |
| `is_abs_uint64`, `big_abs_to_uint64` | `absUint64` with explicit oversized error |
| `is_uint64`, `is_int64`, `get_uint64`, `get_int64` | `IsUint64`, `IsInt64`, `Uint64`, `Int64` |
| `display`, `to_string` | `String` semantic decimal rendering; stream API replaced with string |

Explicitly unimplemented source methods/overloads: `cell_size` (representation
replaced by arena accounting), `allocate_if_needed`, `ensure_capacity`,
`normalize` (standalone; publication normalization is translated), `size_info`,
`sz_lt`, `precise`, `field`, `inc`, `dec`, `mul`, `big_mul`, `addmul`, `submul`,
`machine_div_rem`, `machine_div`, `rem`, `div`, `mod`, `quot_rem_core`,
`big_div_rem`, `big_div`, `big_rem`, `gcd` (all three overloads), `lcm`, `divides`,
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
not implementations. Standalone `u_gcd`/`u64_gcd` remain unported.

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
