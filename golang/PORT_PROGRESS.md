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

## Retained-key map and numeric rounding integration

UintSetMap now executes the actual tst5 map demonstration in exact bucket order,
closing the earlier omitted integration. Keys are scalar owned word refs with
logical length/source capacity; FREE/DELETED payloads, wrapper temporary copies
and synthesized-copy rehash lifetimes follow source. Atomic word-graph budget
transfer permits cross-budget Swap without copying keys.

Integer Log2/MLog2/Bitsize and rational Floor/Ceil/previous-power methods now
preserve all 16 source rational assertions and 2,000 integer log bounds, with
independent signed/alias/failure expectations. Combined release tests, checked
races, vet, actual fresh-count js/wasm tests and WASI compilation pass. Source
and no-generics audit passes. Power/shifts/parsers, numeric scratch/formatting,
real rational-cell allocator assertions and all full AST/solver gates remain open.

## Real rational coefficient allocator assertions

Pool32 now executes both source object-allocator tests with actual MPQ numerals,
including the original 100,000 operations and selected-index worker recycling.
All source coefficient-zero/equality and constructor/destructor assertions pass.
An added big-fraction fixture verifies allocation-free numerical reset/destruction,
retained noncoefficient state, owner lifetime and complete budget release.

These integration assertions were immediately GREEN against existing methods;
no production change or artificial RED was necessary. An isolated prospective
commit passes full native tests, actual g3checked races, vet, checked js/wasm
execution and WASI compilation. The public rational wrapper, production cell
links, construct=false and full solver remain open.

## Integer Power/shifts and rational Power

Source repeated squaring, integer fast paths, power-of-two detection and whole
limb/bit shifts now have concrete counterparts. All 72 source shift cases pass
with separate and aliased outputs, including the million-bit case. Independent
signed integer/rational oracles, typed source-domain errors, representation-specific
INT_MIN division, failure cleanup and warm zero-allocation checks are retained.

Tests were ported first and observed RED before implementation. Isolated full
native tests, actual g3checked races and vet pass. Checked Nodejs/wasm execution
explicitly registers and passes all eight new tests; both handle modes compile
for JS and WASI. Earlier agent commands using the incorrect checked tag are
superseded by actual g3checked gates. Parsers, remaining numerical methods,
standalone MPN/formatting scratch ownership and full AST/solver gates remain open.

## Source byte regions and page actions

BytesRegion now translates both release and debug source region policies,
including strict fit-before-alignment, eager defaults, big-page deletion,
LIFO reuse, physical scope-mark bytes, exact diagnostic formatting, stable
scalar owner IDs, generations and complete cold/payload budgets. Tests-first
RED led to GREEN with independent byte/page models and failure/lifetime checks.
Region methods/ownership are audited; general page-user bindings remain partial.
Combined full native/checked races and vet pass; all14 new region cases execute
under checked Nodejs/wasm, and WASI compilation passes. Full solver remains open.

## Borrowed string hash-table specialization

StrHashtable now preserves source CString hashes/equality, cached hashes,
linear probing, last tombstones, strict growth/cleanup, bucket order and borrowed
spelling lifetime. Foreign opaque keys carry scalar region origins. Swap exchanges
bindings with table data and transfers table charges atomically. Multi-origin
assignment/union/intersection remain explicit source gaps, not completed methods.

Tests-first RED reached GREEN; central review added RED regressions for epoch
exhaustion before mutation and before interner allocation. Complete combined
native/checked races/vet, actual checked JS execution of all12 table tests and
WASI compilation pass. Owner-local prepared lookup supports the symbol service.

## Complete selected integer/rational text setters

Both source text setters now retain the permissive byte grammar, initial-space
sign detection, NUL termination, fractions/decimals/scientific notation, prior
rational denominator history and uint64 exponent wrapping before source range
checks. Distinct parser exception categories remain separate from budget/owner
errors and nonterminating source Power domains. Staged output/failure behavior
and source temporary cleanup order are documented.

Source tests went RED before implementation; all3 active rational parser
assertions, source integer setups, 27 rational textual inputs, independent finite
oracles and fault-position checks pass. Existing72 shift/2000 log fixtures now
use the production parser for source setup. Combined full native/checked races,
vet, actual checked JS execution of all9 new parser tests and WASI builds pass.
Public rational wrapper, remaining numerical methods and scratch ownership remain
open; the full solver objective is unchanged.

## Complete selected symbol service

Concrete pointer-free symbols and an explicit owner-local interning service now
preserve source numeric/null/marked tags, identity, cached hash prefixes,
rendering, CString equality, contains/size and signed-byte/termination ordering.
All23 active source assertions and exact output were ported first and went RED,
then GREEN. Added nullable bare-spelling and marked identity-order regressions
also went RED before their changes. An independent review verified the complete
source method/ownership map and closed the prepared-miss epoch failure gap.

Independent byte/order oracles, both logical pointer geometries, complete budget
sum and constructor/failure/close checks, canonical concurrent interning and
warm zero allocation pass. Combined full native/checked races and vet pass;
all6 symbol tests execute under checked JS, and WASI compilation passes.
Raw pointers/global cache become explicit owner/token adaptations per the plan;
context integration, scoped symbol tables, other container/numeric gaps and all
AST/solver/proof/interpolation milestones remain open.

## Owned natural scratch and numeric formatting

NaturalManager now owns budgeted source division buffers and retained decimal
limb/text workspace. Integer division shares the concrete indexed Algorithm D;
integer and rational rendering avoid unbudgeted numeral/digit copies and publish
one caller-owned final Go string. FormatInto stages complete output before writing
caller bytes. Earlier add-back reservation is an explicit storage adaptation;
the arithmetic and source alias behavior remain unchanged.

Tests went RED before these APIs/storage changes and reached GREEN with full
fault-prefix, peak-budget, lifetime, independent arithmetic and warm-zero-allocation
checks. Isolated full native and g3checked module tests, both races, vet, actual
checked JS registrations and WASI builds pass. MPN diagnostics, further numeric
methods and the full AST/solver objective remain open.

## Multi-origin borrowed string table

The borrowed StrHashtable specialization now supports foreign owner queries,
insertions, entry replacement, assignment, union, intersection, cloning and swap
with scalar hot records and a budgeted cold directory. Source bucket/action order,
cached hashes, pointer-overwrite semantics and bulk failure prefixes are retained.
No spelling is copied or freed by the table. All18 tests pass, including independent
three-owner models, adoption failures, graph peaks and transfer rollback.

New APIs went RED before implementation. Isolated full native and checked JS
module tests, native/checked tests, races, vet, source audit and WASI compilation
pass. Other table specializations, string-to-int map and enabled statistics remain
open; the full solver objective is unchanged.
