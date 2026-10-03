# Port progress

The full fork remains the objective. This foundation work supplies no solver API,
SAT result, theory, AST manager, proof, or interpolation implementation.

## Current baseline

`port_manifest.json` inventories 2,102 source/build/tool/example units and 677
API declarations. All production entries start unported. The 1,687 source files
with extensions `.cpp`, `.h`, `.hpp`, or `.pyg` have content fingerprint
`f8480997c14e2d6c414f40c60ce5095ef40a89ce6d1bcf1ed196c5d682d42d8f`.
This explicitly records the current checkout rather than claiming the plan's
1,619-file fingerprint matches. There are 68 untracked generated source files in
that set; no reference files were altered during this work. Restricting the set
to Git-tracked files gives
`caabc82ba8943415d98e1d87be9c219aaed988ce8afdf8b16da3be146b4f55cc`.
The plan's four individual AST/SMT/interpolation checksums match. Generated files
and changed-baseline provenance still need detailed generator audits.

Conditional directives, executable assertion locations, and registration lines
are lexical inventory, not proof that branch activation or ownership was audited.
Per-method, binding, registration, test-runner, and ownership audits remain open
until recorded by component. Header API declarations are inventoried; their
implementation dispatch and generated binding counterparts still need audits.
Source-existence coverage is distinct from semantic completion.

## Tests first

The checker tests first failed with undefined inventory/validation symbols, then
passed after implementation. Additional RED tests exposed omitted Python binding
files and missing executable-test validation before those fixes. Synthetic trees
make normal checker tests hermetic. Maintainer reference reads are opt-in:

```
go run ./cmd/portcheck -source ..
```

The checker checks hashes, newly added source/API coverage, statuses and executable
Go test declarations. Declarations are evidence that tests exist; their passing
execution and source-method coverage remain separate review requirements. It
refuses to regenerate over an existing ledger, preventing loss of progress.

Main-package handle tests likewise failed before implementation and now pass in
release and `g3checked` modes. Storage tokens always retain generation; internal
release references retain slots only. Public owner/kind validation awaits real
domain managers. The arena, numeric, hashing, and bit-vector packets document
source assertions, independent tests, adaptations and open methods in their
package `PORT_STATUS.md` files.

## Work queue and gates

Next: word/typed-vector payload pools and scoped regions; persistent numeral
cells and complete mpz/mpq methods; remaining containers/limits/parameters;
generated tables and complete registration/ownership audits. Foundation gates
must precede dependent AST/solver work. The source manifest remains open for all
other components, including all fixedpoint engines, QE, optimization, APIs and
frontends. wasm compilation is not wasm runtime acceptance.

All authored files, scratch, build outputs and configured Go cache/temp paths
stay below `golang/`. Git commits are explicitly requested by the user.

## Second foundation integration

Typed regions, fixed-width bit vectors, initial signed integer manager ownership
and Add/Sub, and SAT literal/set primitives have reached tests-first GREEN.
The ledger now records their translated methods while keeping incomplete units
partial. Scalar lbool units are fully translated; this does not complete a solver.

The full current module passed CGO-disabled release and checked tests, checked
race tests, vet, WASI cross-build, and actual js/wasm tests using the installed
matching Go wrapper with Node 22.22.2. Source-reference inventory still matches
its explicitly recorded current baseline. WASI execution remains blocked in the
installed custom Go runtime before user code, as the utility package notes
explain; 32-bit execution also remains unverified after a host syscall failure.

MPZ formatting/header/scratch budgets remain an explicit foundation gate, along
with remaining arithmetic, parameter/registration/container work. Typed regions
do not yet implement every byte-payload lifetime or solver-specific trail class.
No P1/AST/solver/full-port gate is declared complete.

## User implementation rule: no Go generics

The user requires concrete Go types and functions throughout the project,
including tests, tools and generated code. See `AGENTS.md`. This supersedes the
plan's generic allocator sketches. Existing generic slabs/vectors/regions,
handle validators and shuffle must be converted while preserving their tests,
ownership, budgets and pointer-free representation. The concrete conversion below supersedes these earlier generic APIs.


## Concrete storage integration

All authored Go code now uses concrete types: Slab32/Slab64, descriptor slabs,
Words/Vectors64/Vectors32Signed, Region32/Region64, and a numericCellSlab owned
by mpnum. Record and undo tests also use concrete implementations. Shuffle is
concrete for uint32 and int32. No per-record interface boxing or unsafe storage
was introduced. Recursive pointer-freedom audits use reflect.TypeOf.

Tests were migrated to the new API before implementations, producing RED builds.
The no-generics checker likewise started RED, rejected the earlier infrastructure,
and now passes. It scans project Go declarations, explicit type arguments and
known generic standard-library calls, excluding build scratch/cache; this syntax
guard complements implementation review. Source-generation and future package
work must maintain the same rule.

Full module release/checked tests, checked race tests, vet and actual js/wasm
execution pass after integration. Warmed allocator and MPZ Add/Sub operations
still assert zero Go allocations. The source coverage, budget and platform gaps
recorded above remain open; no full foundation/solver milestone is asserted.

## Concrete pool, dense set and multiplication/division integration

Constructed scalar pools now preserve reset-only recycling, worker destinations,
eager pages and destructor order. Dense UintSet preserves source word capacities,
weighted hashing and ascending iteration. Signed MPZ multiplication and division
use owned budgeted limb scratch and source Algorithms M/D; source output/divisor
alias defects have explicit tests. The ID generator preserves LIFO reuse and
source wrap/debug/hash behavior through concrete scalar storage.

These packets were driven from RED source/assertion tests to GREEN, with bounded
independent oracles and failure/alias checks. Full release/checked tests, checked
race tests, vet, actual js/wasm execution and WASI compilation pass. The source
manifest/no-generics audit passes with 2 ported, 24 partial and 2,076 unported units.
Real rational allocator instances, unconstructed pools, full numeric methods,
formatting/header budgets and UintSet-key map integration remain open. Compilation
is still the only WASI evidence; no full foundation or solver gate is complete.

## Resource-limit service

Source reslimit and its three scope wrappers now have concrete Go counterparts,
including source nesting/cancellation quirks. Header/vector budgets, growth
failure invariance, scope cleanup and asynchronous cancellation are tested.
Full release/checked tests, checked race, vet, actual js/wasm tests and WASI
compilation pass at integration. The source/no-generics audit records 4 ported,
24 partial and 2,074 unported units; this still leaves the full solver open.
A Go-only archive of commit 5d01444 also passes its full hermetic test suite
without reference C++ sources. All extraction/cache/temp paths stay in golang.

## Lehmer GCD, scalar hash map and byte allocator integration

Scalar GCD uses the selected source Lehmer algorithm, with concrete binary GCD
helpers and AddMul/SubMul. The scalar uint32 map preserves source probing,
tombstones, strict thresholds, bucket order and copy/swap behavior. SmallBytes
implements pooled, direct and diagnostic direct-only byte policies, with stable
generations and the source consolidation retention quirk. Source cleanup and
conceptual-address adaptations are documented beside the method coverage.

Integration release/checked tests, checked races, vet, actual release/checked
js/wasm execution and WASI compilation pass. Verbose wasm runs explicitly
confirmed new test registrations; the compiler uses the workspace environment,
and the minimal environment is confined to the Node runtime wrapper. A prior
minimal-environment compiler invocation omitted new registrations; no cache
fault is established. The no-generics/source audit records 4 ported, 31 partial
and 2,067 unported units. MPQ, remaining MPZ, other map variants, domain allocator
ownership, cold substrate headers and the full solver remain incomplete.

## Cancellation scope, complete cold-owner charges and basic MPQ

Concrete resource cancellation scopes preserve first-caller identity and nested
destructor counter behavior. Arena/integer owners now charge their cold headers
and nested services exactly once, retain them across Reset and release them on
Close; all constructor reservation prefixes and nested failure rollback are tested.
Basic MPQ preserves source normalization, both scratch modes, arithmetic/order
and reciprocal/divisor destination aliases. Explicit rational construction yields
source 0/1; raw Go zero denominators are diagnosed.

Source assertions were ported before implementation and observed RED, followed
by GREEN with independent signed rational oracles, storage/failure/alias tests.
Integrated release tests, checked races, vet and WASI compilation pass. Actual
js/wasm runs explicitly list and pass all new MPQ and owner-budget tests. The
manifest/no-generics audit records 4 ported, 36 partial, 2,062 unported units.
MPQ parsing/Power/rounding/getters, standalone MPN scratch and formatting budgets,
set-key containers and all AST/solver milestones remain open.
