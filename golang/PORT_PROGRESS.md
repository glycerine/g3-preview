# Port progress

## Real standalone SAT storage integration

The concrete clause allocator/temporary clauses, binary/wrapper records and
vectors, borrowed clause sets, watches/watch lists, justifications and mutable
activity-backed variable queue now preserve their source methods and failure
prefixes. Tests first exposed missing APIs; independent review regressions
then corrected escaping iterator wrappers, retained scalar capacity, SetEnd
ordering and watch erase-tail lifetimes. All records remain concrete and
pointer-free; real lifetime metadata and old/new backing peaks are charged.

The sixteen-file packet passed full native and checked/debug tests on the
committed AST prerequisite, forty named race and actual JS groups, vet, WASI
compilation and no-generics/source audits. Exact aliases, invalid source
instantiations, conditional diagnostics and ownership adapters are in
SAT_STORAGE_PORT_STATUS.md. Eight SAT source units moved to partial; the
ledger now records2,102 units/677 APIs:33 ported,77 partial,1,992 unported.

Storage has no propagation/search or solver result. The full SAT solver and
its registered scope test continue as an actual algorithmic packet with their
real simplification/default-policy dependencies; no reduced solver is substituted.
Proof/checker, SMT/theories and interpolation remain within the active goal.

## Real AST runtime integration

The first runtime packet now supplies all five node kinds, six parameter kinds,
real declaration information and structural interning, source IDs and logical
reference actions, normal/format managers with five eager plugins, owning
references/vectors, and persistent expression arrays with actual AST callbacks.
Tests were written before implementation; review regressions additionally
observed RED before correcting source allocation prefixes and retained pointer
storage. Six quantifier-update overloads also began missing-API RED, then passed
with identity early returns and direct retained payload copying.

The frozen67-file packet passed full native and checked/debug tests, 52 named
checked/debug race and actual JS groups, five additional release JS branches,
vet, WASI compilation, and source/no-generics verification. The source AST test
executes all nine active ENSURE actions; the array stress test executes its
full million updates with real Var handles. AST_CORE_PACKET.md records exact
hashes, ownership adapters and limits. Source inventory remains2,102 units and
677 APIs:33 ported,69 partial,2,000 unported. Registered AST test coverage moved
to partial because its conditional sizeof TRACE output remains open.

Remaining AST copy/translation, dependencies/marks, arithmetic coercions/plugins,
trace/printers, runtime debug-refcount and global resources are explicit next
work. The independently tested proof and SAT drafts are outside this commit.
The complete solver, proof checker and interpolation goal remains active.

## Concrete internal32 fast-float integration

The selected MPFF manager and its concrete scalar/vector scopes now implement
the source internal32 arithmetic, directed rounding, conversions, predicates,
limits/power and byte-formatting families. Tests observed missing-API RED before
implementation. Review regressions separately observed RED before correcting
release integer truncation/getters and source diagnostic ordering. No Go
generics, unsafe storage or production math/big were introduced.

The final isolated prospective tree passed full native, checked/debug, race,
actual checked/debug JS, vet and WASI compilation. All 22 new named MPFF groups
were explicitly observed RUN/PASS under JS; release JS also executes the two
source conditional conversion/getter fixtures. Source tests retain all finite
loop counts and execute 908,316 ENSURE plus two VERIFY actions, complemented by
independent directed-arithmetic/alias models and immediate ownership/fault
audits. Exact adapters, frozen hashes and gate evidence are in
MPFF_PORT_STATUS.md. WASI remains compilation evidence only.

The coordinator audited the final central manifest against the isolated packet:
2,102 units and 677 APIs, with 33 ported, 68 partial and 2,001 unported. Three
MPFF units moved from unported to partial; the existing scope source rows gained
the concrete MPFF specialization. Persistent ostream state, TRACE/STRACE,
external exception transport and the real subpaving consumers remain open, so
whole MPFF units stay partial. Parallel AST, proof and SAT drafts are outside
this commit; the full solver/AST/proof/interpolation objective remains active.

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

## Scoped symbol-to-int32 bindings

SymbolInt32Table now ports all ten source assertions and every declared source
method for int32: nested trails, reverse undo, append and domain/range traversal.
Source dynamic symbol hashes, physical tombstones/inactive data, strict growth
and cleanup, scoped overwrite bypass and vector capacities are preserved.
Allocating undo/append use fully charged physical staging; source-compatible
warm paths allocate nothing. A review regression first failed for empty
high-occupancy scope undo, then passed after its unnecessary staging was removed.

All15 tests pass in isolated full native/checked/checked-JS modules, race and vet;
WASI builds and combined module gates pass. Synthesized copy operations and other
payloads remain open, so the source header stays partial. Parser/AST integration
and the full solver objective remain open.

## Concrete source integer heap

IntHeap preserves the source heap algorithm, mutable borrowed priorities, inverse
positions, tie behavior, bounds/reset, swap, traversal/display and explicit deep
copying. Both original finite fixtures and all active assertions were ported first
and went RED before implementation. Independent review caught resize capacity
history; a RED regression drove exact repeated source vector expansion. Copy APIs
and unsigned allocation-size limits also obtained RED before implementation.

All9 heap tests pass, covering independent minima, source array/output order,
budgets/failures, peaks, both swap reservations, lifetime, epoch and warm allocation
behavior. Combined native and checked races, full actual checked JS, checked vet,
WASI build and source/no-generics audit pass. Heap source/header tests are audited;
solver consumers and the full AST/solver/proof/interpolation goal remain open.

## Integer bitwise source algorithms

BitwiseOr/And/Xor/Not now preserve the selected source's repeated64-bit
arithmetic chunk algorithms and all final source actions. The active bug4
assertion/output and independent numerical, width and alias oracles were written
first, observed RED, then reached GREEN. Source domain assertions are typed
errors; budgeted local cleanup and staged publication preserve failed outputs.

All6 tests pass, including pre-Close temporary accounting, both aliased fault
sweeps, release of old big outputs and zero-allocation warm
paths. Prospective full native/checked races/actual checked JS modules, vet,
WASI build and source/no-generics audit pass. Other MPZ/MPQ methods, full numerical
wrapper and all solver/AST/proof/interpolation milestones remain open.

## Scoped binding deep copies

Explicit Clone and Assign now preserve the source implicit member-copy policies.
Clone rehashes USED entries at source capacity and copies live marked trails and
limits at exact vector capacities. Assign resets/replays the destination table
through its own shrink/growth policy, then copies source vector history; it stages
all actions in fully charged storage before publishing. The borrowed symbol
manager and destination Budget are retained, and cold mutexes are never copied.

Four new tests went RED before implementation; all19 scoped-table tests pass
with source bucket/capacity models, scope undo, exact peaks and failure prefixes.
Isolated full native/checked/actual checked JS modules, race, vet, WASI compilation
and source/no-generics audit pass. Other symbol payloads and full solver remain open.

## Actual borrowed CString-to-int32 source map

StrInt32Map now translates all14 source map.cpp assertions with actual CString
keys and signed int32 values. Default entries retain complete dead/free payloads;
source cached hashing, last tombstones, growth-before-duplicate, copy/reset/finalize
and bulk action order remain intact. Multi-origin borrows use fully budgeted cold
bindings, while values are mutable through checked entry references.

Tests first reached missing-API RED, then GREEN. Central architecture review
exposed a36-versus40-byte record mismatch on386; its RED test drove explicit
scalar padding, now audited on386/ARM/AMD64/ARM64/WASM. All10 tests pass, including
bounded independent models, lifetime, partial action failures, graph peaks and
warm allocations. Isolated full native/actual checked JS modules, checked native,
vet, WASI build and no-generics audit pass. Source map test and the concrete
string-functor/table-alias header are complete; generic map/core variants and all
AST/solver/proof/interpolation milestones remain open.

## Integer and rational roots

Integer Root/RootInPlace and perfect-square queries now retain the complete
source bound setup, bisection, Power and Swap/Copy actions. Rational counterparts
preserve short-circuit output history and source aliases, including finite false
results and deliberately unreduced output. Classified assertion/degree domains
and undefined fractional square alias division are separate from ordinary false
results. All temporaries use existing budgeted numerical storage with explicit
cleanup and staged failure publication.

All10 active integer-root assertions and two output lines were ported first and
observed RED; independent finite root/square and rational cases are added tests.
All11 tests pass, including signed ceilings, aliases, both source rational modes,
small/big output fault sweeps, immediate pre-teardown live-cell/Words accounting
and warm-zero-allocation checks. Isolated full native/checked modules, checked
race/actual checked JS registrations, vet, WASI and source/no-generics audit pass.
Other numeric methods and full solver/AST/proof/interpolation remain open.

## Integer scalar inspection

Strict32-bit conversions, parity, source sorting weight, floating conversion,
hashing, trailing-zero multiplicity and caller-buffer decomposition now preserve
the selected internal32-bit source behavior. Five tests reached missing-API RED
first, then GREEN, with bounded independent magnitude/parity oracles, source
assignment-history assertions and warm allocation/budget checks. UINT_MAX floating
radix and conversion boundary exclusions remain explicit source quirks. See
MPZ_MISC_PORT_STATUS.md. Remaining numeric and complete AST/solver/proof/
interpolation work stays open; no full-port gate is declared complete.

## SMT2 and decimal numeral display

Integer/rational SMT2 printing and rational decimal precision/truncate printing
now preserve source action order, including negative scoped temporaries, early
exact termination, precision-zero punctuation and reachable raw negative-
denominator quirks. Existing budgeted formatting workspace holds all intermediate
text; only final caller output escapes. Nine tests cover source output lines,
independent finite decimals, raw outputs, exact old-plus-new growth budgets,
failure cleanup before teardown and allocation-free warm buffer formatting.
Tests first reached missing-API RED, then GREEN. See NUMERIC_DISPLAY_PORT_STATUS.md.
Other numeral/wrapper and all solver milestones remain open.

## Mixed and fused rational arithmetic

Eighteen concrete mixed/fused and integer-valued rational operations now preserve
source arithmetic, local/cached scratch branches and component alias reads.
RatDiv keeps negative denominators and wrappers keep integer division/modulus
output-divisor quirks. Eight test groups port all24 active rational tst9
assertions and add bounded independent oracles, alias fixtures, large-limb warm
paths and injected failures with immediate root-based cells/Words accounting.
Missing APIs and a component alias fixture were observed RED before translation
and correction. See mpnum/RATIONAL_OPS_PORT_STATUS.md. Remaining numerical methods,
rational wrapper and full AST/solver/proof/interpolation scope stay open.

## Scoped numeral owners and vectors

Concrete Int/Rational scoped owners and noncopyable vectors now borrow their
manager and retain explicit deep-copy/transfer/destruction semantics. Headers and
old-plus-new backing are fully charged; source ascending truncation and failed
push/resize action prefixes are preserved. Eleven test groups cover three active
source print lines (no source ENSUREs), independent Int/Rational lifecycle models,
slot reuse order, exact backing peaks, deep-clone faults, immediate Cells/Words
ownership and warm allocations. Missing declarations were observed RED first.
Other numeral-manager specializations stay partial; full numerics, rational
service/wrapper and AST/solver/proof/interpolation milestones remain open.
See SCOPED_NUMERAL_PORT_STATUS.md.

The final combined module passed full native and nativechecked tests, full
checked-race tests, full actual checked JS execution (all33 new named tests
registered/passed), checked vet and WASI cross-build. Source/no-generics audit
passed with14ported,39partial and2049unported units. WASI is build evidence only.
The next AST-numeral prerequisites are rational inspection/extraction, explicit
cross-owner limb copying and the owner-local rational service/constants/cache;
array/extended GCD, LCM and divisibility also remain open. No full foundation
or solver completion is claimed.

## Explicit cross-owner numerical copying

Integer and rational CopyFrom now transfer owned limbs between explicitly
supplied managers, preserving small/big shape, source allocation capacity and
raw unreduced/signed-denominator representation. Rational outputs are staged
with real destination-owned cells before pair publication. Five tests reached
missing-API RED then GREEN, with independent source-close lifetimes, every
reservation prefix, immediate live-cell/Words accounting and warm allocations.
Raw managers retain their caller-supplied owner contract; public provenance and
AST consumers remain separate work. See NUMERAL_COPY_PORT_STATUS.md.

## Integer and rational number theory

Array GCD, release extended GCD, signed LCM and divisibility now preserve source
action order; integer-valued rational wrappers include source denominator-reset
prefixes and the fractional singleton array branch. Nine test groups began with
missing APIs RED and pass source calls, independent exact oracles, finite alias
fixtures, canonical large coefficients and immediate failure ownership audits.
ExtendedGCD source Z3DEBUG diagnostic arithmetic remains explicitly unported;
its complete conditional coverage and the separate ext_gcd.h wrapper are open.
See NUMBER_THEORY_PORT_STATUS.md. Full numeric/AST/solver milestones stay open.

## Rational inspection and extraction

Explicit rational predicates, conversions, hash, getters, statistics, floating
conversion and thin setters now retain exact source representation and alias
semantics. Inclusive rational32-bit projections remain distinct from strict MPZ
variants. Eleven tests port all18 source conversion checks and the10002-case hash
assignment history, plus independent values, signed/unreduced/component states,
floating NaN/Inf branches and immediate ownership/value failure audits. Missing
APIs were observed RED before implementation. See RATIONAL_INSPECTION_PORT_STATUS.md.
Owner-local wrapper service, complete operator facade and AST ownership remain open.

## Owner-local rational service core

RationalService now owns the synchronized numeric manager, constants and source
powers cache; OwnedRational is a pointer-free value with checked scalar service
provenance. Explicit copy/move/swap/delete and cross-owner copying preserve
source lifetimes without a global mutable manager. Cache growth, extra k+1 entry,
placement-copy failure prefixes, retries and ascending destruction remain exact.
Fifteen tests began with missing APIs RED; signed/unsigned constructor boundary
shape tests independently reached behavioral RED before fixing distinct signed32
and typed64 setter paths. Five selected source ENSUREs and independent cache,
provenance, ownership, concurrency and exact-budget tests pass.
See RATIONAL_SERVICE_PORT_STATUS.md. Complete operator facade, infinitesimal
initialization/finalization hooks and AST parameters remain open.

Final combined verification after the signed-fraction correction: full uncached
native and checked race suites, full actual checked JS/Wasm execution, checked
vet and checked WASI cross-build all passed. Every one of the40 new tests across
copy, number theory, inspection and service packets has explicit JS RUN/PASS
evidence. WASI execution is not claimed. Source inventory/no-generics audit:
2102 units,677 APIs;14 ported,41 partial,2047 unported. The complete solver,
AST, proof and interpolation scope remains open.

## Owner-local rational operator facade

RationalService now exposes source value/compound arithmetic, signed32 scalar
overload directions, comparisons, inspection, integer-valued arithmetic, rounding,
powers, fused wrapper expressions, ModHat and NumBits. Value arithmetic retains
source copy-before-compound and returned-lvalue-copy steps; prefix/postfix uses
rational-one arithmetic. Eleven test groups began with missing-API RED and cover
all64 tst1 ENSUREs and all56 static tst2 ENSUREs plus its VERIFY, independent
exact arithmetic, aliases, immediate failure roots and warmed allocation checks.
See RATIONAL_FACADE_PORT_STATUS.md. Roots/squares, bitwise and remaining wrapper
interoperability/display operations remain open; full solver scope is unchanged.

## Concrete rational extended-GCD wrapper

The separate ext_gcd.h Euclidean wrapper is now translated through the owned
rational facade, preserving input copies, output publication prefixes, fresh
div/mod assignments and coefficient expression return copies. Five test groups
began with missing-API RED and cover all999 source inverse conditions (500 RHS),
the actual power/assignment setup, independent signed Bezout checks, aliases,
provenance and immediate failure ownership/prefixes. See
EXT_GCD_WRAPPER_PORT_STATUS.md. The manager's distinct ExtendedGCD algorithm is
not substituted; its Z3DEBUG diagnostic branch still remains open.

## Owner-local infinitesimal rationals and complete service lifecycle

Both active infinitesimal representations now use concrete pointer-free owning
records in the existing rational service. All active header methods, rational
approximation functions, signed-epsilon behavior, owned accessors and constants
are translated. First/second publication order and reverse member destruction
retain source actions; coarse roots/powers are preserved. The service now runs
both infinitesimal initialization/finalization hooks in source order and charges
all six constants in its actual header. See INF_RATIONAL_PORT_STATUS.md,
INF_INT_RATIONAL_PORT_STATUS.md and RATIONAL_INF_LIFECYCLE_PORT_STATUS.md.

Twelve rational-infinitesimal groups began RED and port all62 source ENSURE sites
executing80 checks plus finite exact bounds, expression/alias/failure tests.
Ten integer-infinitesimal groups provide independent coverage; no absent source
test registration or assertion counts are invented. Three parent lifecycle
groups began behavioral RED and verify header, unwind and production destruction
traversal. Source-route defects found in increments and signed constructors were
fixed after behavioral RED.

Final frozen combined full native, checked race and actual checked JS/Wasm suites
passed, with all41 new test groups explicitly RUN/PASS in JS. Checked vet, WASI
build and no-generics/source audit passed; WASI runtime is not claimed. Complete
solver, AST, proof and interpolation milestones remain open. Next numeric gaps
include remaining rational facade wrappers, inf_eps_rational and source Z3DEBUG
extended-GCD diagnostics; AST parameters and ownership consumers remain open.

Final source inventory:2102 units,677 APIs;21 ported,40 partial,2041 unported.
The active rational.cpp service algorithms/hooks and both infinitesimal units
are complete; rational.h and the full solver scope remain incomplete.

## Parameter-name normalization

Both source norm_param_name overloads now preserve nullable CString, single-colon,
ASCII-only rewriting and exact non-ASCII bytes. Four tests began missing-API RED;
byte fixtures caught and drove correction of a Go UTF-8 range iteration defect.
Symbol normalization reads existing bytes without interning/intermediate strings,
and produces one independent caller output with unchanged owner budget/storage.
See PARAM_NAME_PORT_STATUS.md. Parameter descriptors and params_ref are separate
active work; neither is claimed complete by normalization.

## Extended-GCD conditional diagnostic

The remaining source extended-GCD Z3DEBUG mul/mul/add/equality branch now runs
under g3debug, independently of checked handles. Tests began missing-API RED
and verify signed Bezout cases, source alias assertion/output prefixes, large
arithmetic fault prefixes, immediate roots and complete owner refund. Release
alias behavior remains unchanged. See EXTENDED_GCD_DEBUG_PORT_STATUS.md.

## Complete owner-local rational facade

The remaining rational.h root/square, manager coefficient GCD, LCM, bitwise,
display, projection and checked scoped constructor/extraction routes are now
translated. Four raw rational bitwise forwards retain numerator-before-denominator
publication. Fourteen tests began missing-API RED and include all34 source tst8
assertions, independent roots/coefficients, aliases, fault roots, scoped lifetimes
and output allocation checks. Debug alias assertions are tested alongside the
source release prefixes. See RATIONAL_FACADE_EXTRA_PORT_STATUS.md.
The rational.h active unit is complete; raw-manager conditionals and full solver
requirements are not inferred complete from this facade.

## Concrete infinity-and-epsilon numeral

The sole actual inf_eps_rational<inf_rational> specialization now has a complete
concrete triple representation and owned service API. Eleven tests began RED
and cover independent finite triple arithmetic, the mixed equality defect,
nonsaturating infinity, component publication/destruction, source returned
copies, fault roots and one final caller string allocation. No absent source
numeric assertion registration is invented. See INF_EPS_RATIONAL_PORT_STATUS.md.

## Parameter descriptions and kind labels

All param_descrs methods and insertion helpers now preserve the source dictionary,
nullable metadata, persistent name history, namespace rewriting, sorted display
and physical-bucket copy order. Thirteen tests began missing-API RED; further
behavioral RED drove dictionary-before-history publication and exact prefix-vector
growth fixes. Owned metadata and temporary storage are physically budgeted,
with immediate failed-prefix ownership audits. See PARAM_DESCRS_PORT_STATUS.md.
cmd_context_types.cpp kind labels are complete; params_ref and command runtime
methods remain open. Dictionary coverage records this concrete specialization.

## Infinitesimal formatting ownership correction

InfString and InfIntString now stage every intermediate byte in the existing
budgeted formatter workspace and allocate only one final caller output.
Four tests began behavioral RED at three warmed allocations, then passed with
one; signed/domain cases, source abs lifetime, immediate failure roots and
old/new workspace growth peaks are covered. See INF_FORMAT_PORT_STATUS.md.

## Source exception classification and command error values

All fourteen source error codes, fixed z3_error messages/code predicates,
default_exception string values, CmdException constructors/locations and the
distinct stop-parser signal now have concrete Go counterparts. Five tests began
missing-API RED and cover source labels, embedded NUL, byte/position domains,
checked symbol composition, lifetime and unchanged borrowed-owner budgets.
See Z3_EXCEPTION_PORT_STATUS.md. The printf/va_list constructor remains open;
its integer/string source formatter is a separate active packet, and command
runtime/complete solver scope remains open.

## Integer and C-string source formatting

The format2ostream integer/C-string routes now preserve source byte-vector
growth, AMD64 one-call truncation/fallback and the other-profile retry sequence.
Eleven groups began missing-API RED; count overflow also produced behavioral
RED before its fix. Exact flags, byte/NUL domains, Linux I64 interpretation,
actual budget peaks and immediate cleanup are covered. The formatted exception
printed case is translated (zero source assertions). Broader C printf domains,
Windows CRT state and warning I/O remain open. See PRINTF_PORT_STATUS.md.

## Shared parameters references and values

The remaining params_ref/private params methods now preserve shallow sharing,
ordered detach/copy, shared reset and validation, typed getter/fallback scans,
source signed casts and display routes. Eighteen tests began missing-API RED;
self-copy rational identity also produced behavioral RED before correction.
Failure sweeps audit live roots before display/teardown, and actual context
arrays/text/scratch are budgeted. The complete params.h/.cpp utility surface
is mapped; solver parameter consumers remain open. See PARAMS_REF_PORT_STATUS.md.

## Concrete S-expressions and reference wrappers

All sexpr.h/.cpp methods now have concrete pointer-free storage and owned
reference/vector counterparts. Sixteen tests began RED and preserve deletion
order, RC wrap, source rendering quirks, full byte strings and checked views.
Review-driven RED corrected both the source BV debug expression and its local
inline16/doubling digit buffer. Allocation sweeps audit roots before cleanup,
and whole owner graphs and overlap peaks are charged. Five relevant escaped
print inputs are covered without inventing source assertions; other escaped
branches and non-sexpr wrapper specializations remain open. See SEXPR_PORT_STATUS.md.


## Concrete string buffers

All actual fork string_buffer sizes32/64/128 now have concrete owners and complete
selected methods, including mutating CStr, nine cross-buffer routes and direct
symbol insertion. Ten test groups began missing-API RED and port the three active
source ENSURE assertions, plus independent byte models, lifetime guards, immediate
failure-prefix roots and exact inline/overflow overlap budgets. Full native and
checked/debug suites, targeted checked/debug race and actual JS, vet, WASI and
no-generics/source audits passed. See STRING_BUFFER_PORT_STATUS.md. Arbitrary
INITIAL_SIZE instantiations and Windows secure-CRT runtime profile remain open;
string_buffer.h and the generic symbol insertion header remain partial.

## Integer manager public routes

Twenty-one concrete MPZ facade methods now expose source signed predicates,
comparison forwarding, scalar setters, Inc/Dec, owning Dup and unsupported Inv.
Seven groups began with six missing-API RED tests and retain eighteen exact
source MPZ/MPQ comparison assertions. Independent bounded arithmetic, immediate
failure roots, representation quirks and zero warmed allocations are covered.
Full native/checked-debug, targeted race/actual JS, vet and WASI build passed.
Backend/diagnostic alternatives and separate MPBQ/interval/modular managers
remain open. See MPZ_FACADE_EXTRA_PORT_STATUS.md.

## Budgeted recycled source IDs

The source ID helper now supports charged free-ID backing and explicit preflight
for future AST reclamation. Four added tests began missing-API RED, then verify
source capacities, exact old/new overlap, immediate failure prefixes, OOM skip,
reset retention and cleanup refunds. Eight total ID suites passed integrated
native/checked-debug, race and actual JS gates; vet and WASI build passed. Full
resource-service and AST registration/deletion remain open. See ID_PORT_STATUS.md.

## Source chained integer tables and maps

Concrete int32 table/map release algorithms now preserve bucket traversal,
head displacement, cellar retry/free-list order, grow-before-duplicate and
collision counters. Seventeen groups began missing-API RED. Review-driven RED
also corrected incoming hash call order and the source by-reference UsedSlots
update retained on failed copy. Registered source assertion loops keep their
exact1000/10000/50000 bounds with an independent finite set oracle. Integrated
full native/checked-debug, race, actual JS, vet and WASI build passed. Automatic
debug diagnostic actions and AST specialization remain open; AST_CORE_PORT_STATUS.md
records the real manager prerequisites. See CHAIN_TABLE_PORT_STATUS.md.

## SMT2 symbol utilities and complete escaping bytes

All four real SMT2 utilities now preserve the ASCII whitelist, nullable C-string
predicate, special outer-pipe scanning, symbol overload and bytewise quote
constructor. Eight test groups began missing-API RED and cover the one source
implementation precondition, exhaustive byte models, borrowed-owner lifetimes,
exact old/new peaks and immediate failure cleanup. Escaped now covers all thirteen
registered print-only inputs (zero ENSURE), plus null/trim/indent domains and
published quote/space failure prefixes. Behavioral RED corrected an invented
uint32 output bound; valid UINTMAX indent now stops at finite-budget growth
failure after the source prefix. Full native/checked-debug, named race/actual JS,
vet/WASI/no-generics gates passed. See SMT2_UTIL_PORT_STATUS.md. Full AST printers
and external custom ostream state remain open; util.h/.cpp stay partial.

## Real AST family registration

The source family_manager now has concrete registration, lookup, both membership
routes, name lookup and domain/range enumeration. Six added tests began missing-
API RED; seven final groups cover real init-family order, source failure counter/
map/name gaps, immediate nested budgets, exact old/new peaks and warm zero host
allocations. Integrated full native/checked-debug, named race/actual JS, vet/WASI
and source/no-generics gates passed. These are added fixtures (no source family
ENSURE suite). ast.h/.cpp are partial; real nodes/plugins/AST manager and solver
remain open. See FAMILY_PORT_STATUS.md.

## Standalone sort cardinality scalar

All standalone sort_size constructors, scalar copy, factories, predicates,
inclusive64 threshold, finite-size guard and exact display now have a concrete
pointer-free value. Four test groups began missing-API RED, exercise real rational
IsUint64 then Uint64 classification (including UINT64_MAX and2^64), independent
finite byte output, owner provenance/lifetime and unchanged borrowed budgets.
No source ENSURE assertions are invented. Full native/checked-debug, named race
and actual JS, vet/WASI/source-no-generics gates passed. See SORT_SIZE_PORT_STATUS.md.
ast.h/.cpp remain partial for all other AST nodes, managers and plugins.

## Concrete persistent integer arrays

Both source int32 root policies now use real ROOT/SET/PUSH/POP trails, reversals,
materialization and RC30 ownership. Tests began missing-API RED and retain all712
registered int32 ENSURE executions, plus independent version and graph oracles.
Fifteen groups preserve defined failure prefixes and the proven source PUSH
inverse index defect; review-driven RED added checked physical host bounds.
Integrated full native/checked-debug, race/actual JS, vet/WASI and no-generics
gates passed. Actual AST/dependency/bound callbacks, the real AST million-update
test and automatic source diagnostics remain open. See PARRAY_PORT_STATUS.md.

## Binary rational arithmetic and owning scopes

The selected internal32 MPBQ contract now includes arithmetic, roots, directed
approximation, all four interval-selection paths, rational conversions, and
plain/pretty/SMT2/decimal byte output. Fourteen test groups began missing-API RED
and retain both active print-only source bodies (zero ENSURE, six independent
expected outputs). Alias publication, raw-shape rounding/refinement quirks,
source debug short-circuit, immediate numeral ownership, charged scope/vector
backing and failure prefixes have permanent regressions. Frozen integrated full
native/checked-debug, named race/actual JS, vet and WASI compilation are checked
independently. Whole MPBQ units remain partial for source trace and full ostream
integration; MPBQI, polynomial/algebraic managers
and the full solver remain open. See MPBQ_PORT_STATUS.md.

## Concrete external numeric text stream

A concrete cold text stream now models the selected C-locale/libstdc++ AMD64
precision ABI and nearest/even IEEE64 default/fixed/scientific output. Eight
test groups cover independent exact binary-rational rounding, signs/specials,
integer type widths/base, persistent flags and one-shot width, real old/new
scratch peaks and allocation-failure byte prefixes. Initial missing-API RED
preceded implementation; behavioral RED corrected a virtual width host cap and
embedded-NUL String truncation. Independent frozen full native/checked-debug,
named race/actual JS, vet/WASI/source-no-generics gates passed. This is an
external num_put platform adaptation, not completed statistics production.
The Statistics audit now records the real nonadvancing internal key loop;
locales, process rounding, other stream runtimes, collectors and solver
consumers remain open. See FLOAT_STREAM_PORT_STATUS.md.

## Concrete open binary rational intervals

The selected MPBQI manager now implements its executable open finite interval
methods and owning scopes with three retained multiplication scratch bounds.
Ten test groups began missing-API RED, adapt four real shared-template source
assertions, and check independent dyadic arithmetic, exact source alias/Disjoint
quirks, immediate physical ownership and completed failure prefixes. Independent
frozen full native/checked-debug, ten named race/actual JS, vet/WASI compilation
checks passed. Generic closed/other numerical configurations, field-only inverse
and division, persistent ostream and real polynomial/algebraic consumers remain
open. Three source units are partial; full solver scope is unchanged. See
MPBQI_PORT_STATUS.md.

## Concrete statistics and resource-limit collection

Statistics now retains raw uint/double contributions, borrowed key identities,
source vector/copy/move actions, real aggregation maps and source display quirks.
Nineteen groups began missing-API RED; review-driven RED corrected host-byte
binding limits and raw copying of closed borrowed key owners. Independent frozen
full native/checked-debug, nineteen named race/actual JS, vet/WASI checks passed.
The internal source loop never advances a nonempty key: its explicit typed guard
preserves prior empty-key output. Real reslimit thresholds are implemented;
process memory services and solver/API/timer producers remain open. Statistics
source units remain partial. See STATISTICS_PORT_STATUS.md.

## Borrowed byte queries for AST name interning

A concrete byte-input boundary now queries the existing CString table without
a temporary Go string. Four tests began missing-API RED and preserve first-NUL,
signed-tail hashes, probing, growth-before-duplicate and exhausted-epoch behavior;
long warm queries allocate no Go objects. Independent frozen full native/checked
debug, named race/actual JS, vet/WASI/source-no-generics gates passed. Source/API
coverage counts are unchanged; real AST name construction and the full manager
remain in progress. See utilhash/PORT_STATUS.md.

## Heap insertion failure prefixes

Source review for the SAT activity queue found an existing heap ordering mismatch:
insert publishes an inverse index before its value-vector push can fail. Three
new regressions first failed, then passed after preserving that prefix and the
source's forward-only invariant traversal. Empty reset keeps the inverse entry;
nonempty reset clears it. Defined in-range continuations remain executable, while
actual out-of-bounds continuations return a typed error before mutation. Frozen
full native/checked-debug, twelve named race/actual JS, vet/WASI and source/no-
generics checks passed. Coverage counts are unchanged. AST, MPFF and complete SAT
storage packets continue in parallel; the full solver port remains incomplete.

## Next proof-construction source contract

The coordinator read all out-of-line AST proof builder/reader bodies and their
inline inspection contracts, recording all 44 distinct method names and every
overload as required next work in AST_PROOF_PACKET.md. The audit distinguishes
per-method proof-disabled behavior, the legacy two-argument three-input helper,
ignored NNF validation results, both actual unit-resolution bodies and append-only
hyper-resolution output prefixes. The registered proof-checker test still needs
the real checker and its one VERIFY; proof construction cannot stand in for it.
This is a source contract artifact, with no source/API completion relabeling.

## Fast-float retained vector storage correction

Four new regressions first failed, then passed after preserving initialized
capacity tails through trivial-copy growth and release Shrink/At access. Debug
assertions retain their source order. The source scoped resize overload cannot
instantiate for mpff; Go Resize explicitly uses the executable base-vector
default-construction adapter. Full isolated native, checked/debug, race, actual
JS, vet, WASI compilation and source/no-generics gates passed on frozen bytes.
MPFF_PORT_STATUS.md records precise limits and hashes; source coverage counts
remain unchanged and the full solver port remains active.

## AST proof builder and reader integration

All44 distinct out-of-line proof builder/reader names, their overloads and
inline mode/fact/parent inspection now use the real AST manager. Tests-first
behavioral corrections preserve source canonical constants, release short
circuits, lexical destruction and allocation failure prefixes. Hyper-resolution
uses real owning outputs and temporary vectors; quantifier bindings remain
nonowning. Frozen21-file/31-group native, checked/debug, race, actual JS, vet,
WASI compile and source/no-generics gates passed. An independent Boolean oracle
checks1,116 cases across8 assignments; this does not replace the real checker.
AST_PROOF_PORT_STATUS.md records scope and adapters. ast.h/.cpp remain partial;
proof checking, complete solver, diagnostics and interpolation remain required.

## Expression-array retained storage fidelity

Three regressions first failed, then passed after matching source debug-only
logical bounds for Get/Set/SetFrom. Release preserves retained root pointers and
nonroot arbitrary-index SET cells. Charged12-byte capacity/initialized headers
guard never-written storage without discarding source retained tails.
Full frozen native/debug, six named race/actual JS groups, vet/WASI compile and
source/no-generics checks passed; see AST_EXPR_ARRAY_RETAINED_STATUS.md.
Coverage counts are unchanged; the full port remains active.

## Concrete AST marking integration

Concrete expression/declaration ID marks, both fast planes, both owning ref-fast
planes, scoped marks and real debug mark-owner checks are implemented. Tests-first
failures caught unsigned-wrap physical reads and allocation-before-reservation;
all seven constructors now reject a refused header without allocating.
The frozen six-file/sixteen-group packet passed native/debug, race, actual JS,
vet/WASI and source/no-generics gates. Scoped marks use actual owning ref vectors;
failed fast mark/INC prefixes remain source-visible. obj_mark.h becomes partial;
other mark specializations and whole AST/plugin/solver work remain open.
See AST_MARK_PORT_STATUS.md for exact source surface and coordinator integration.

## Memory accounting transition component

Tests-first scalar global/lane accounting preserves source strict thresholds,
cumulative TLS count contributions, limit-failure counter prefixes and stored
peak/watermark predicates. Deallocation separates subtraction before actual
free from synchronization afterward. Eight frozen native/debug/race/actual JS
groups, full native/debug, vet/WASI and source/no-generics checks passed.
MEMORY_ACCOUNTING_PORT_STATUS.md distinguishes this component from the required
physical allocator/global runtime, generated callbacks and failure transport.
memory_manager.h/.cpp become partial; the full solver remains incomplete.

Memory size_t-width review preserves LP64 unsigned modular updates and ILP32
unsigned realloc subtraction before LL64 promotion, confirmed by a compiled
source-promotion probe and an additional tests-first width fixture.

## Expression-array source failure-prefix follow-up

Three tests-first regressions preserve finite release empty-nonroot POP
unsigned size wrap and incoming INC before invalid/uninitialized old root SET
access. Debug empty assertions precede output copying; typed guards preserve
completed source effects. The existing AST/parray units remain partial.
Full native/debug, nine named checked/debug race/actual JS groups, the three
new actual release JS groups, vet/WASI compile and source/no-generics gates
passed. See AST_EXPR_ARRAY_RETAINED_STATUS.md for frozen evidence.

## Real standalone SAT model conversion prerequisites

Concrete nested literal storage, model-converter builders/reconstruction and
actual namespace sat::uint_set are integrated. Twenty-four tests-first finite
groups cover real clause/wrapper insertion, reverse model patching, source
CollectVars, exact capacity/deep-copy ownership and failure prefixes. Review
caught and corrected finite captured-end self-copy and immediate debug clause
assertion order, each with an observed behavioral RED before its correction.

The final HEAD0ec768f plus six-file snapshot passed full native/debug, all24
named checked/debug race and actual JavaScript groups, vet and WASI production
build. Final coordinator source/declaration/no-generics audit passed at
33 ported/82 partial/1987 unported; only model_converter.cpp/.h become partial.
See SAT_SOLVER_PACKET.md for exact source maps, hashes and terminal evidence.
Implicit converter owning Clone/Assign and TRACE diagnostics remain open.
The real full SAT solver/default inprocessing/runtime source test remains RED
and is excluded from this separately verified prerequisite commit. No complete
SAT/SMT/interpolation solver claim is made; the original goal remains active.

## Actual AST dependency and dependency-array integration

Real AST-valued leaves/joins, RC30, FIFO identity traversal, stable in-place
linearization and LIFO destruction are implemented with actual AST callbacks.
The persistent dependency-array specialization preserves source retained raw
tokens, inverse patches, shared roots and allocation/reference prefixes.
Twenty-five new groups plus eager binding passed;356 source parray conditions
are explicitly adapted to this reference-enabled specialization, not claimed
as original direct dependency-specialization assertions.

Review regressions corrected source copy-before-INC, debug RC0 refusal
consumption, copied-prefix cleanup and raw retained Get before actual lifetime
dereference. Full native/debug,26 named race/actual JavaScript, vet/WASI compile
and final source/no-generics checks passed. The full manager/solver/interpolation
objective remains active. Source ownership aliases, alternate scoped configs,
TRACE and specific private parray/Reroot diagnostics remain open, as enumerated
in AST_DEPENDENCY_PACKET.md. Only dependency.h/ast_util.h become partial;
final ledger counts33ported/84partial/1985unported.

## Common solver watch payload storage

The concrete common-owner watched-list service encodes scalar clause provenance
and tracks actual per-slot lifetime in charged arena payloads/bitmaps. Source
copy/move/erase/resize/publication order and full retained capacity are preserved.
Review regressions corrected null-backing release no-ops and pointer iterators
following physically transferred backing through move/swap until real lifetime
ends. Twenty-one named groups and full native/debug, race/actual JS, vet/WASI,
source/no-generics checks passed. Existing watched/vector units remain partial;
full solver propagation, outer storage and std sorting backend remain open.
See SAT_SOLVER_WATCH_STORE_PORT_STATUS.md for immutable evidence and adapters.


## Actual sparse marks and shared occurrences

Concrete expression identity tables and shared-occurrence collection now preserve
source defaults, all eight traversal policies, current-node hashing, strict load
thresholds, last tombstones and physical bucket order. Real shared-set publication
precedes AST INC; reset releases references in ascending buckets. Separate node
shared flags, inline visitation, scalar DFS frames and source failure prefixes
use charged concrete storage throughout. Tests-first review corrected iterator
lifetimes across in-place table changes, source frame byte bounds and expression
domain guards.

Six frozen Go files passed fresh full native and checked/debug tests alongside
committed watch storage. Twenty-one exact race and actual JS groups, whole-module
vet, WASI compilation and source/no-generics audits passed. See
AST_SPARSE_SHARED_PORT_STATUS.md for exact source mappings, hashes and evidence.
Shared-occurrence source units and obj_hashtable.h become partial; ledger counts
are33ported/87partial/1982unported. Real SMT2 display, general table templates,
implicit owning copies, full solver, proof checker and interpolation remain open.


## Real model converter implicit copies

Explicit owning Clone/Assign now preserve the source class's actual implicit
copy constructor and assignment: exact outer capacity, ascending deep nested
copies and destination destruction before allocation. Source size/capacity
publish before constructors; incomplete prefixes retain their real physical
ownership and have safe Go cleanup/restart. No implicit move is invented.
Eight missing-API-first ownership groups cover copy independence, exact spare
capacity, constructor/assignment refusals, actual destruction order and late
unconstructed/header guards.

The frozen32-group packet passed native/debug, race, actual JS, vet, WASI
compilation and source/no-generics gates; full integration suites also passed
against HEAD81f1059. See SAT_MODEL_COPY_PORT_STATUS.md. Source inventory remains
33ported/87partial/1982unported; the full solver and runtime dependencies remain
open under the original complete-port goal.


## Actual charged SAT literal sets

The real literal-set specialization now uses the charged SAT uint set,
with geometric returned vectors, exact deep owning copies and explicit
source iterator operators. Tests first observed missing APIs before code;
seven finite groups check independent ordered membership, source publication
faults, immediate graph accounting and retained scalar iterator lifetimes.
Full native/debug, race, actual JS, vet, WASI compilation and source/no-generics
gates passed on the committed sparse/shared and model-copy base. See
SAT_LITERAL_SET_PORT_STATUS.md. Full solver and runtime integration remain open.


## Efficient concrete stable watch sorting

The former insertion helper is replaced by two stable partitions over the exact
source comparator classes, with scalar rotations and logarithmic recursion.
Independent work-bound tests first observed the quadratic helper fail, then
verified at most2N comparisons and O(NlogN) assignments. Clause/external
interleaving, retained erase tails, transferred backing iterators and zero warm
allocations remain verified. Four new groups join21 watch-storage groups.
Full native/debug integration,25 named race/actual JS groups, vet, WASI
compilation and source/no-generics gates passed. SAT_WATCH_SORT_PORT_STATUS.md
records the selected standard-library backend adaptation and exact hashes.
The real solver's outer watch sorting call and full algorithms remain open.


## Real dependency owning aliases

Actual dependency obj_ref, ref_vector and inline16 ref_buffer specializations
now use source callbacks and scalar owned tokens in existing AST allocators.
Tests first exposed missing APIs; independent review additionally corrected
raw scalar comparisons, equal-size no-action resize, raw buffer back access,
consumed cleanup progress and constructor cleanup refusal. An interrupted
cleanup tracks its first unconsumed callback without clearing retained raw
slots. A constructor that cannot release all roots returns a charged partial
owner with its original error for explicit caller cleanup. These safe Go
continuations are documented separately from source undefined failure domains.

Twelve frozen Go files passed full native/debug,30 exact named race/actual JS
groups, vet, WASI test/full compilation and source/no-generics verification.
Actual24-byte vector metadata and160-byte inline buffer storage are charged.
See AST_DEPENDENCY_REF_PACKET.md for source methods, hashes, prefixes and the
preserved uninstantiable C++ buffer-copy constructor. ref_buffer.h becomes
partial; ledger33ported/88partial/1981unported. Full consumers and the complete
solver/proof/interpolation scope remain open.


## Actual centered integer/modular arithmetic

The concrete mpzzp manager now borrows the real IntegerManager and owns its
seven source numerals, preserving centered Z/GF(p)/Zp operations and mode
publication. Actual inverse swaps before its final assertion; finite powers
perform their final unused square. Mutable conversions normalize while const
conversions do not. Tests-first review corrected universal debug guards and
copied/zero cold-owner lifetime checks. Source constructor indeterminate prime
reads and high-bit power nontermination receive typed guards only at the actual
read/wrap; safe Go local cleanup is distinguished from raw MPZ source failures.

Five immutable Go files passed full native/debug on the actual dependency
owning-alias base,13 exact race/actual JS groups, vet, WASI compilation and
source/no-generics gates. MPZZP_PORT_STATUS.md records full selected methods,
hashes and immediate allocation-prefix root audits. mpzzp.h becomes partial;
ledger33ported/89partial/1980unported. Dense polynomial/GCD/Sturm/algebraic and
actual arithmetic plugin algorithms remain within the full active goal.


## Borrowed parameter string spelling

Actual interned set_str spelling can now be retained as a canonical scalar Symbol
without a second Words allocation. Copy preserves identity; replacement and deletion
release only owned text. Four tests-first groups cover copy, empty spelling, lifetime
guards, zero warm allocations and twelve immediate COW failure prefixes. The
registry remains under implementation; see GPARAMS_PORT_STATUS.md.

Full native/debug integration on HEAD1ccea0b, exact22 named race/actual JS
results, vet, WASI compilation and source/no-generics audit passed. Ledger
counts remain33ported/89partial/1980unported.


## Actual single-thread source cooperation

The real cooperate.cpp selected _NO_OMP_ profile now preserves runtime enable,
owner0 checkpoint handoff and source section/task assertion and destruction
sites. A charged concrete cold owner replaces process globals; unused task names
are not retained. Tests first failed on missing APIs, then five action/ownership
groups and a sixth actual C++ thirteen-transition release fixture passed.
Full native/debug, release and debug race/actual JS, vet and WASI compilation
passed. See UTIL_COOPERATE_PORT_STATUS.md. OpenMP multilane and TRACE profiles
remain open; this provides the actual single-thread polynomial checkpoint service.

Source/no-generics audit passed; cooperate.cpp/h become partial with
ledger33ported/91partial/1978unported.


## Descriptor formatting into charged output

Actual parameter descriptor display now writes directly to a borrowed
StringBuffer64, reading canonical symbols and packed metadata without an
intermediate output string. Source sort/name transforms, description assertion
positions and output failure prefixes are preserved. Five tests-first groups
include the source uint32 scratch-byte overflow regression, huge indentation
stopped by actual growth refusal, and immediate temporary/destination accounting.
Four new Go files on HEADfb08673 passed full native/debug, eighteen exact named
race/actual JS groups, vet, WASI compilation and source/no-generics checks.
See PARAM_DESCRS_STREAM_PORT_STATUS.md. Registry composition and the complete
solver/proof/interpolation objective remain open.

Root final-source/no-generics audit passed; ledger counts remain
33ported/91partial/1978unported.


## Actual generated SAT configuration descriptors

The actual static SAT collector now publishes all24 source descriptors in
declaration order with exact name/kind/description/default/module bytes. Five
tests-first groups cover complete metadata, existing duplicate preservation,
180 immediate fault graphs, lifetime guards and zero warmed duplicate allocations.
Two frozen Go files passed full native/debug, exact five named race/actual JS
groups, vet, WASI build and source/no-generics gates. See
SAT_CONFIG_DESCRS_PORT_STATUS.md. config.cpp/h and sat_params.pyg become partial;
ledger33ported/94partial/1975unported. Actual config updates, registry catalog and
full solver algorithms remain required within the active objective.


## Actual expression substitution and scoped ownership

Concrete expression substitution now runs actual AST/proof/dependency callbacks
through three independent scalar tables. Scoped substitution uses real owning
AST trail and retained unsigned limits. Tests-first review caught live-bucket
overwrite, finalize publication order, later-root Reset prefixes, unused-input
validation, retained-limit release Pop growth and interrupted destructor replay.
Source NULL-definition quirks and late freed-key hash guards retain actual
publication prefixes. Same-entry consumed callback completion and finite
cleanup continuation are explicit Go adapters, not source rollback.

Seven frozen Go files passed full native/debug on current HEAD81529c8, exact31
checked-debug race/actual JS groups, vet, WASI test/full compilation and
source/no-generics verification. See AST_EXPR_SUBSTITUTION_PACKET.md. Actual
display requires the real SMT pretty-printer and remains open. Source
expr_substitution.cpp/h and selected ref_util.h map callback loops become
partial; ledger33ported/97partial/1972unported. Full consumers and the complete
solver/proof/interpolation objective remain required.


## Actual global parameter registry

Concrete registry operations now use the real Symbol/Params/descriptor owners,
three independent scalar dictionaries and current shared module snapshots.
Tests-first fixes preserve TRUE-before-registration failure, empty-cell set
publication, borrowed descriptions, reset metadata plateau and nil-stream action
prefixes. Fourteen frozen Go files passed full native/debug, twenty-two exact
race/actual JS groups, vet, WASI build and source/no-generics verification.
See GPARAMS_REGISTRY_PORT_STATUS.md. Full generated catalogs and process binding
remain open; gparams.cpp/h become partial, ledger33ported/99partial/1970unported.
SAT search, SMT/theories, proof/checker and interpolation remain required.


## Actual SAT main configuration updates

The complete source configuration constructor/update now uses current real
registry module snapshots and all twenty-six source scalar assignments.
Tests-first coverage preserves all enum branches, local/module typed fallback,
hidden local-only options, uninitialized dyn-only fields, unrestricted IEEE
values and sequential invalid-policy prefixes. Four frozen Go files passed
full native/debug against committed registrye59f032, twenty-one exact race/
actual JS groups, vet, WASI and source/no-generics checks. See
SAT_MAIN_CONFIG_PORT_STATUS.md. Ledger remains33ported/99partial/1970unported;
complete SAT search and the full solver/proof/interpolation goal remain open.


## Actual dense univariate polynomial core

Concrete dense storage and core arithmetic now use the real modular integer
manager, all twenty-one source scratch vectors and actual resource/cooperation
callbacks. Tests first caught physical-view read timing, trim-mutating predicates
and wrapped source extents; twenty-one named groups cover nine pretranslation
assertions, arithmetic/evaluation oracles, aliases and immediate fault graphs.
Eight frozen Go files passed full native/debug, exact race/actual JS, vet, WASI
and source/no-generics gates. See UPOLYNOMIAL_PORT_STATUS.md. Full content,
division, GCD, Sturm, isolation/refinement and algebraic consumers remain required.
Ledger33ported/102partial/1967unported; the full solver/proof/interpolation objective remains open.


## Complete preserved global parameter catalog

The complete test-runtime dispatcher now collects all458 ordered insertion
attempts into450 distinct descriptors, with two globals, thirty-two module
calls and sixteen exact borrowed-description assignments. Five new tests-first
groups plus twenty-two registry groups verify exact metadata, publication order,
shared description lifetime and immediate pending-owner fault graphs. Seven
frozen Go files passed full native/debug, exact27 race/actual JS groups, vet,
WASI and source/no-generics gates. Coordinator independently checked every
ordered metadata row against actual C++ collector bodies. See
GPARAMS_CATALOG_PORT_STATUS.md. Generated value accessors and full runtime
initializers/alternative catalog selections remain open. Ledger33ported/140partial/1929unported; full solver/proof/interpolation remains required.


## Actual lazy clause-use-list compression

Source lazy list methods and lexical iterator compression now use actual
clause vectors and removed-flag reads. Tests-first coverage preserves captured
counts with current backing, Begin writes, partial iterator cleanup, current
assignment bindings and release unsigned counter bodies. Two frozen Go files
passed full native/debug and twenty exact race/actual JS groups, vet, WASI
and source/no-generics checks. See SAT_SOLVER_CLAUSE_USE_LIST_PORT_STATUS.md.
Implicit moves/nonlazy/TRACE and complete solver remain open. Sourcecpp/h
become partial; ledger33ported/142partial/1927unported.


## Actual raw memory allocations and counters

The selected raw allocator now owns real prefix-bearing byte backings and
invokes actual source counters at malloc/realloc/free action positions in
both profiles. Eleven tests-first/regression/model groups cover thresholds,
actual failure roots, generation exhaustion, wrapped-header partial ownership
and initialized-byte lifetimes. Source getters never substitute Budget.Used.
See RAW_MEMORY_PORT_STATUS.md. Full runtime initialization, typed-owner hooks
and process transports remain open; ledger remains33ported/142partial/1927unported.
The complete solver/proof/interpolation objective remains required.


## Literal-indexed SAT simplifier use lists

The actual use_list wrapper and inline lazy children now preserve ordered literal
actions, same-address reconstruction, outer backing invalidation and source
iterator destruction. Eighteen tests-first/source-action groups include finite
compaction, exact owned graph and injected failure prefixes; root review caught
and regression-tested the destructor's i<size condition after an extra release
Next. See SAT_SOLVER_CLAUSE_USE_LISTS_PORT_STATUS.md. Default simplification,
implicit moves/copies and the complete solver remain required. Both simplifier
units become partial; ledger33ported/144partial/1925unported.


## Polynomial content, division and complete selected GCD chain

Actual primitive/content normalization, pseudo/exact division, signed remainder,
Euclidean/subresultant/modular GCD, centered prime-image reconstruction, extended
GCD and square-free predicates now use the committed concrete numeric owners.
Tests-first coverage ports all thirty exact-division ENSURE routes and preserves
source aliases, nonmonic extended-GCD quirks, raw array read timing and immediate
failure roots. The exact active231-prime table is checked directly against source;
late reconstruction tests reach actual post-append reservation failures. See
UPOLYNOMIAL_GCD_PORT_STATUS.md. Sturm, isolation/refinement, factorization,
multivariate conversion and algebraic consumers remain required. The active
prime-table unit becomes partial; ledger33ported/145partial/1924unported.
The complete solver/proof/interpolation objective remains open.


## Internal integer allocation and publication reconciliation

Selected MPZ scratch replacement, big/digit setters, result transfer, paired
division and Div/Mod now preserve actual source action order rather than
staging replacement allocations. Tests-first failures expose published results,
freed old cells and exact owned dead-token continuations; composed rational,
binary and infinitesimal cleanup releases live companions without weakening
read/provenance checks. Retired descriptors are metadata, not live numerals;
polynomial diagnostics and dependent tests use immediate named ownership graphs.
See MPZ_SOURCE_PUBLICATION_PORT_STATUS.md. Frozen full native and checked/debug,
234 named race and actual JS groups, vet, WASI compile and source/no-generics
gates pass. Source allocator topology, bitwise/GCD/Power/root/rational staging
and the complete solver/proof/interpolation goal remain open. Ledger remains
33ported/145partial/1924unported; selected allocation method evidence is partial.


## Destructive numeric capacity and base-two Power

The actual allocate_if_needed helper and base-two Power branch now mutate the
caller in source order, preserving old-free/sign publication on replacement
refusal and capacity reuse on success. Scalar setters execute their source
capacity diagnostic before writes; existing normalize and ordinary scalar OOM
routes have independent confirmation. Nine new tests plus eight existing
power/shift groups pass named native/debug/race/actual JS; full isolated native
and checked/debug pass. Vet, WASI compilation and no-generics/source gates pass.
See MPZ_SETTER_PORT_STATUS.md. Scalar infallible write layout, generic Power,
0^0 release/debug policy, shifts, other staged wrappers and true source allocator
integration remain required. Neither whole MPZ unit becomes complete.


## Dense sequences, Sturm transforms and root-count prerequisites

The actual flat three-vector sequence, Sturm/Tarski/Fourier chains, four sign
variation locations, RootID, all selected transforms and Descartes/root bounds
now use the concrete committed numeral owners. Tests first port sixteen remaining
translation/reciprocal ENSURE routes and five half-root routes, with an independent
rational Euclidean oracle, actual allocation refusals and cancellation prefixes.
The source lexical [-1,2] divisor uses a concrete stack span over the existing
division core. See UPOLYNOMIAL_STURM_PACKET.md for final gates, lifetime and
physical accounting evidence. Root isolation/refinement, multivariate conversion,
factorization, algebraic consumers and printers remain required. No whole source
unit is promoted; ledger remains33ported/145partial/1924unported.


## Standalone source small-object allocator

The standalone allocator now executes the source profile branches, requested
counter publication, intrusive chunk/free links, strict cursor boundary and
full consolidation scan using actual memory-runtime allocations. Tests first
replay the registered print-only actions and exercise independent storage,
lifetime, allocation-refusal and diagnostic-prefix checks. Explicit Go cleanup
retains live roots after interrupted frees and keeps their owner charged until
finalization. See SMALL_OBJECT_ALLOCATOR_PACKET.md for source boundaries and
final verification. Numerical and solver allocator wiring, global diagnostics
and the full solver/proof/interpolation objective remain required. All three
allocator source units remain partial; no whole-unit status is promoted.


## Direct Power, shift and rational component publication

Integer and rational repeated squaring now publish directly to the caller,
including the final unused square and the actual unsigned mask-wrap boundary.
Copy-output shifts execute the source Copy before the in-place core. Public
rational Copy and Mul publish components in source order using the existing
arithmetic and normalization algorithms. Tests first reproduce allocation
refusals and source prefixes, then verify independent finite arithmetic, aliases
and immediate owned roots. Dependent parameter, interval, vector and service
tests retain the completed source components on failures. See
MPZ_POWER_PUBLICATION_PACKET.md for frozen gates and remaining boundaries.
Source allocator/backend integration, other staged arithmetic and the complete
solver/proof/interpolation goal remain open; no whole source unit is promoted.
