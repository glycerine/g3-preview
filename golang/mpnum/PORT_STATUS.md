# Numeric foundation packet

Status: **partial source unit**, not completion of the numerical manager layer.

Source: `src/util/mpn.h`, `src/util/mpn.cpp` (Microsoft 2011, Christoph
Wintersteiger). The full arithmetic algorithms were read before translation.
`src/util/mpz.h`, the arithmetic call sites and `quot_rem_core` in `mpz.cpp`,
and source assertions in `src/test/mpz.cpp`/`mpq.cpp` were inspected for
preconditions and the next manager packet. No `mpz` or `mpq` operation is claimed
implemented by this packet.

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

All `mpz`, `mpq`, `rational`, rounding/fixed-precision and higher arithmetic
source methods remain unimplemented. No fallback arithmetic or semantic stubs.

## Tests and TDD evidence

Tests were added first. `go test ./mpnum` initially failed to build with undefined
`NaturalManager` at all test call sites (RED). After algorithm translation the
same tests passed (GREEN). Further regression fixtures exercise quotient-estimate
correction, estimate equal to the base, and Algorithm D subtract/add-back.

`TestSourceMPZMagnitudeProjections` carries `src/test/mpz.cpp`'s `tst2`
UINT_MAX carry/square values and `bug1`'s exact large self-subtraction input
through the underlying limb methods. These are **projections**, not full ports of
the signed manager assertions. Parsing fixture input uses the independent test
oracle; the source `set` operation is not implemented. Remaining active `mpz`
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

There are no persistent numeral cells yet and no imports back into `g3`.
The next prerequisite is a coordinator-approved numeric-cell and word-page
handle contract with arena budget/failure behavior, explicit deletion/copy/move,
manager close, and scalar inline int32 storage. This packet does not claim that
scratch slices implement the plan's persistent pointer-free numeral ownership.

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
