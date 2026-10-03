# Concrete modular integer manager packet

Source `src/util/mpzzp.h`, complete selected defined-body translation integrated after central review. Whole source unit remains PARTIAL: source constructor indeterminate-bool reads, nontermination, assertion/unsupported domains and TRACE output have explicit adaptations/open classifications below. Algebraic/upolynomial/arithmetic declaration plugin and their registered tests remain OPEN.

Owned files: `mpnum/mpzzp.go`, `mpnum/mpzzp_ops.go`, `mpnum/mpzzp_debug.go`, `mpnum/mpzzp_nodebug.go`, `mpnum/mpzzp_test.go`, this status. No existing source/manager/manifest/Git files were changed.

## Source method map

| Source | Actual concrete API |
| --- | --- |
| Z ctor, mpz modulus ctor, uint64 modulus ctor, destructor | `NewIntegerModularManager`, `NewIntegerModularManagerWithModulus`, `NewIntegerModularManagerWithUint64`, `Close` |
| `precise`, `field`, `finite`, `modular`, `m`, `p` | `Precise`, `Field`, `Finite`, `Modular`, `Manager`; private `borrowedModulus` returns actual numeral; public `Modulus(out)` is an explicitly owned-copy accessor adapter |
| `setup_p`, `is_p_normalized_core`, `p_normalize_core`, `is_p_normalized`, `p_normalize` | Real setup, short-circuit bounds comparison, signed Rem then upper/Sub or lower/Add; `IsPNormalized`, `PNormalize`; both source core and outer debug normalization assertions execute |
| `set_z`, two `set_zp`, `set_p_sq`, `set_zp_swap` | `SetZ`, `SetZP`, `SetZPUint64`, `SetPSquare`, `SetZPSwap` |
| `reset`, `is_small`, `del`, `neg`, `abs` | `Reset`, `IsSmall`, `Del`, `Neg`, `Abs` |
| All seven sign/zero/one predicates | `IsZero`, `IsOne`, `IsPos`, `IsNeg`, `IsNonPos`, `IsNonNeg`, `IsMinusOne` |
| `eq`, `lt`, `le`, `gt`, `ge` | `Equal`, `Less`, `LessEqual`, `Greater`, `GreaterEqual`; source normalized-input debug assertions preserved |
| `to_string`, `display` | `String` has source normalization assertion; `FormatInto` deliberately has none, uses real borrowed manager's budgeted formatter and caller-owned final output |
| `add`, `sub`, `inc`, `dec`, `mul`, `addmul`, `submul` | `Add`, `Sub`, `Inc`, `Dec`, `Mul`, `AddMul`, `SubMul`; actual raw IntegerManager action then normalization, source debug operand sites |
| `inv`, `swap`, `divides`, `div`, `hash` | `Inv`, `Swap`, `Divides`, `Div`, `Hash`; source coefficient caches, exact short circuits, source swap-before-final-gcd assertion |
| Two `gcd`, array `gcd` | `GCD`, `ExtendedGCD`, `GCDMany`; extended coefficients normalize first then second, gcd output is not silently normalized |
| mpz mutable/const `set`, int/unsigned/string/int64/uint64 `set` | `Copy`, `SetInt32`, `SetUint32`, `SetString`, `SetInt64`, `SetUint64`; both mpz overloads share the identical source body, no extra clone route |
| Mutable machine tests/getters, const machine tests/getters, mutable double getter | `IsUint64Mutable`, `IsInt64Mutable`, `Uint64Mutable`, `Int64Mutable`, `Float64Mutable` normalize input first; const `IsUint64`, `IsInt64`, `Uint64`, `Int64` do not normalize |
| `power`, `is_perfect_square` | `Power`, `IsPerfectSquare`; source mask arithmetic, final unused local square, exact delegated alias quirks and modular unsupported branch |
| Both `mul2k`, both `machine_div2k` | `Mul2k`, `Mul2kInPlace`, `MachineDiv2k`, `MachineDiv2kInPlace`; multiplication normalizes, division only asserts final normalization |
| `power_of_two_multiple`, `log2`, `mlog2`, both `root` | `PowerOfTwoMultiple`, `Log2`, `MLog2`, `Root`, `RootInPlace`; real IntegerManager algorithms, source root debug nonmodular guard |

## Storage, lifetime, and source exceptions

One actual charged cold header owns seven real pointer-free Int fields: modulus/lower/upper/three inverse outputs/division temporary. It borrows an existing unsynchronized IntegerManager and its budget. `reflect.TypeOf(IntegerModularManager{}).Size()` includes actual cold manager/self pointers and every scalar/Int field; reservation precedes escaping object publication. Numeric cells, words, cached manager scratch, pages and overlap are already charged by that real manager. No per-record Go pointer/interface/unsafe/big-number replacement or generic code. Close deletes seven fields in source order and refunds only its own header. Caller operands still obey the raw manager's documented implicit-owner contract.

Zero-value/nil/shallow-copied cold owners are rejected by the actual self identity guard with `ErrModularOwner`, before any metadata or numeral mutation. Copying an owning manager is an explicit rejected unsafe domain, not a new source manager copy algorithm. Static `Precise` remains owner-independent. Borrowed manager lifetime normally exceeds wrapper lifetime; after forced borrowed-manager Close, wrapper Close refunds only its own external header and skips callbacks against already reclaimed cells.

Centered representations preserve odd [-floor(p/2),floor(p/2)] and even [1-p/2,p/2]. Source constructor `prime` arguments are unused and `m_p_prime` uninitialized. Both modulus constructors therefore keep `primeKnown=false`; `Field` and enabled debug Div reject only the actual indeterminate bool read. Release Div executes without inventing that read. Z mode short-circuits the flag. `SetZP` sets flag true before modulus copy; `SetPSquare` sets false before multiplication; swap preserves it. Failed operations retain real cached/caller/mode publication prefixes.

Source setup positivity/not-one, SetPSquare/Swap nonmodular preconditions, normalized operand/result assertions, inverse zero and final gcd-one checks and root nonmodular assertions occur under `g3debug` only. Finite release modulus1 and negative-modulus actions are preserved, including negative-modulus defective bounds/normalization. Composite/nonunit inverse release returns its source coefficient; debug returns assertion error AFTER caller swap. Source traces/printer side effects remain OPEN; `g3checked` alone does not simulate them.

Source integer-mode inverse UNREACHABLE and modular perfect square NOT_IMPLEMENTED are typed `ErrModularUnsupported`. Source integer manager retains its own defined/undefined arithmetic domain errors (e.g. remainder by zero). No modular square-root replacement is invented.

Power copies its real local then publishes output1, executes every selected multiplication and every local square (including final unused square), and advances the uint32 mask. On the actual 2^31→0 mask wrap, after all completed iteration actions, it returns the existing typed `PowerDomainError`/`ErrNonterminatingPower`; no earlier output rollback or alternate power algorithm. **Raw C++ mpz has no destructor:** source `del(power)` runs only at normal end. Go deferred Del additionally cleans the local on allocation/assertion/nontermination error as a safe ownership adaptation, while retaining all caller/cached source prefixes. Constructor failure similarly cleans the unpublished partial wrapper; it does not claim raw source MPZ member exception unwinding freed those payloads. Immediate root accounting tests distinguish these adapters from cached/source-owned live numerals.

## Tests and observed RED

No registered direct mpzzp assertions exist; actual registered polynomial/upolynomial/algebraic consumers remain unported. Thirteen added finite groups cover centered bounded arithmetic, all mode/constructor routes, inverse/division/fused aliases, every predicate and scalar method route, const/mutable conversion/display differences, source noncanonical alias quirks and debug/release preconditions, high-bit Power prefix and unused-square allocation failure, cold identity/header refusal, 112 operation-failure prefixes and16 constructor failure prefixes. Each failure audits live cells AND Word refs before formatting/Del/Close, with only actual caller/wrapper/seven source fields/permanent manager caches allowed. Warm arithmetic and rejected-header construction are measured zero Go allocations.

- Missing complete API RED: `.cache/mpzzp-initial-red.log`, exit1.
- Explicit in-place overload missing-API RED: `.cache/mpzzp-inplace-red.log`, exit1.
- Defined release setup precondition regression RED: `.cache/mpzzp-setup-red.log`, exit1; source debug-only guards corrected.
- Zero/copy-owner behavioral RED: `.cache/mpzzp-owner-red.log`, exit1 (two actual panics, copied Close accepted); charged selfguard GREEN.
- Final targeted checked+debug thirteen groups GREEN: `.cache/mpzzp-refined-target.log`, mpnum0.025s.

## Immutable final gates

Snapshot `.cache/release/mpzzp-final2`, committed base aadb037b4aa28e4c3c7d84d86c6a200141ce8d99 plus only the five owned Go files. Earlier eleven-group snapshot on01cc137 is superseded. Logs `.cache/release/mpzzp-final2-logs/`; all tool cache/temp/environment writes below golang.

All gates terminal PASS, exit0:

- Fresh full native `go test -count=1 ./...`: mpnum21.985s (`native.log`).
- Fresh full checked+debug `go test -count=1 -tags "g3checked g3debug" ./...`: mpnum22.615s (`debug.log`).
- Named thirteen groups checked+debug race: thirteen RUN/thirteen PASS, mpnum1.223s (`race.log`).
- Actual named checked+debug JavaScript wasm: thirteen RUN/thirteen PASS, mpnum0.382s (`js.log`); original golang-local js-exec/TMPDIR preserved.
- Full checked+debug vet (`vet.log`), full-module WASI build (`wasi.log`), source/no-generics audit (`source-no-generics.log`).
- Audit reports2102units/677APIs:33ported87partial1982unported on committed-base metadata; this packet does not alter that metadata.

All five worktree Go hashes rechecked against frozen snapshot; no edits after final2. Coordinator owns metadata/integration/commit.

Frozen Go hashes:

```
77b6a527fc84f0c5355cf68a14a6a8757aaa843a2e83fa59260af1ac6b1241ef  mpnum/mpzzp.go
e279ccaa3f7ba19c5b150a129669074c9dee288c2838cdc996a0342ad76f2575  mpnum/mpzzp_debug.go
ab968575be758cd244b9c2a3ce5d0dc12ece655133bc4391d853c9304277c4ee  mpnum/mpzzp_nodebug.go
51c14a40b86e5711e3dbeae5a9ddc54cd872c7d33bf0eafb3181e7dfefb290f8  mpnum/mpzzp_ops.go
39a2407e664bbeeca0e7dd6c994f1513351f5a3612522a480fb7518b491575b9  mpnum/mpzzp_test.go
```

## Coordinator integration

All five live Go files match the final2 snapshot/hash inventory. Every one
of13 exact race/actual JS RUN/PASS names was independently checked.
Prospective committed tree .cache/review/mpzzp-integration-1c2bb33 contains
HEAD1c2bb33 (including actual dependency owning aliases) plus only these
five Go files and final metadata. Fresh full native PASS (root2.435s,
mpnum22.579s), fresh full g3checked+g3debug PASS (root2.800s, mpnum22.951s);
final source/no-generics PASS33ported/89partial/1980unported. Code hashes
were rechecked before commit; no unrelated draft appears in this tree.
Full registered upolynomial/algebraic consumers remain unimplemented.
