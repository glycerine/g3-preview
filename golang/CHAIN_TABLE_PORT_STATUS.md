# Concrete chaining table prerequisite

`util_chain_table.go` translates the release algorithm of
`src/util/chashtable.h` for the actual registered `int_table`, `dint_table`
and `int_map` specializations in `src/test/chashtable.cpp`. The two concrete
owners share private scalar storage; there are no Go type parameters, boxed
entries, per-entry pointers, production Go maps, or unsafe operations. The AST
structural-equality specialization is still unported and is not simulated here.

## Source method map

| Source operation | Concrete counterpart |
| --- | --- |
| constructor/init, destructor | `NewInt32ChainTable`, `NewInt32ChainMap`, `Close` |
| `get_hash`, `equals` | cold borrowed `Hash`, default uint32(int32), scalar key equality |
| `alloc_table`, `delete_table` | checked scalar cell allocation, budget reservation/refund |
| `copy_table`, `expand_table` | bucket-ascending/head-first copy, head displacement, repeated doubled-cellar retry |
| `has_free_cells`, `get_free_cell`, `recycle_cell` | source bump cellar and LIFO recycled-cell list |
| `insert` | `Insert`, existing scalar replacement |
| `insert_if_not_there` | `InsertIfNotThere`, cold checked entry reference |
| table `insert_if_not_there2` | table `InsertIfNotThere2`, true only for a new element |
| `contains`, `find`, `find_core` | `Contains`, `Find`/`FindInto`, `FindCore` |
| `erase` | `Erase`, exact head-copy or middle unlink/recycle |
| `reset`, `finalize` | `Reset`, `Finalize`; empty reset is a no-op; collision count survives both |
| `empty`, `size`, `capacity`, `used_slots`, `collisions` | named getters and `Stats` |
| iterator ctor/default/copy/deref/arrow/pre++/post++/==/!= | concrete value iterators, `Begin`/`End`, `Entry`/`Ref`, `Next`/`PostNext`, `Equal`/`AtEnd` |
| table `swap` | `Swap`, including capacities, counters and cellar state; source hash functors stay with their owners |
| cmap forwarding methods and iteration | `Int32ChainMap` methods and concrete pair iterator/reference |

`cmap` has no source `swap` or invariant method; map `Swap` and explicit
`CheckInvariant` are Go storage/audit adapters. Source implicit owning copies
would shallow-copy the table and eventually destroy the same allocation twice;
no invented safe source copy constructor or assignment is claimed.

## Ownership and failure adaptation

The owner header is charged once using actual target layout. Every table cell
is `{key int32, value int32, next uint32}`; physical bytes use actual layout on
each target. The table-only owner retains an unused scalar value field because
it shares concrete storage with the map. Link indices replace source pointers;
reserved FREE/END indices impose checked capacity bounds. FREE/recycled payloads
are retained, as in source. The Hash environment is borrowed and must outlive
the owner; callers serialize operations and use a stable, non-reentrant hash.

Growth charges both original and replacement arrays. Each insufficient-cellar
trial is refunded before the next trial. Each attempted copy resets and updates
`m_used_slots` by reference, preserving the source's observable failed-copy prefix
even if the next reservation fails. The old physical table, entries and views
remain unchanged, but UsedSlots can then differ from the old occupied-bucket
count. The regression freezes eight old occupied buckets becoming prefix count
one when an early merged chain overflows before later buckets are visited.
`Finalize` allocates its initial table before
destroying the current table, charging the overlap and preserving the current
graph if allocation fails; source deletes before allocating. These are explicit
predictable-allocation-failure adapters.

Swap allocates no arrays. Same-budget and equal-byte cross-budget swaps reserve
nothing. Across distinct budgets, the owner receiving the larger table reserves
only its positive byte delta before the other owner refunds that delta and the
fields swap. Header charges and hash functors stay with their original owners.
Quota or epoch failure publishes no field changes. This transfers ownership
charges rather than inventing an old/new physical allocation peak.

References and iterators are caller-owned cold value wrappers. Their owner/epoch
checks conservatively invalidate borrows on membership changes, expansion,
finalize and swap. Non-growing value overwrite retains existing views. Default
and cross-owner end iterators compare equal, preserving the source's null list
pointer equality. Mutable keys are supported; as in source, callers can violate
bucket placement by changing them. Close refunds storage at exhausted epoch
without panicking and is idempotent. Structural epoch exhaustion is checked
before mutation; existing non-growing overwrite and empty Reset still succeed.

## Tests and gates

Initial tests were written before production and produced missing-API RED.
All registered source tst1/tst2/tst3/tst5/tst6 assertions are retained. The three
tst4 registrations execute exactly 1,000, 10,000 and 50,000 operations, checking
every source assertion against a bounded independent bool-array set. The libc
`rand` sequence is platform-dependent; the Go fixture uses the existing source
RNG with a fixed seed and preserves the exact registered operation counts and
branch/assertion structure. The source DEBUG_CODE tst6 value assertion is run
as additional unconditional portable coverage; this does not claim automatic
DEBUG_CODE routing.

Seventeen test groups also freeze bucket/chain order and collision counts,
grow-before-duplicate, signed modulo-to-uint32 dummy hashes, incoming hash call
order, map overwrite/reference/iteration, retained payloads, repeated cellar
retry, constructor/finalize/growth/swap failure prefixes, immediate budget sums,
closed and stale borrows, epoch exhaustion, quota transfer and warmed zero Go
allocations. A behavioral RED exposed three incoming hash calls instead of one;
the correction preserves the exact release incoming hash order after rehash.
Named getter APIs also had an observed missing-API RED before implementation.

Isolated HEAD `5ed77e7` plus only the two new Go files passed full CGO-disabled
native and checked+debug tests, checked+debug targeted race tests, all seventeen named RUN/PASS
groups on actual checked+debug JS/wasm, checked+debug vet, WASI build, and the
no-generics syntax audit. No WASI runtime execution is claimed.

## Remaining conditional/template coverage

Automatic source Z3DEBUG CASSERT call sites and `check_invariant`'s temporary
pointer-address visited hashtable are not ported. The portable explicit
`CheckInvariant` scans scalar chains and verifies structural counts without
allocating, but it does not claim the source diagnostic actions. Disabled
`#if 0` tracing/destructor output, arbitrary HashProc/EqProc/value/destructor
template instantiations, `cmap` with non-int32 fields, and the AST identity erase
and structural interner specialization remain outside this packet. Release
int32 algorithm coverage is complete; whole-template/conditional source
coverage remains partial.

## Coordinator integration gate

The prospective integrated Go tree in `.cache/release/id-chain-integrated-final`
contains the frozen string-buffer, integer facade, budgeted ID and chain-table
packets together. Full fresh CGO-disabled native and g3checked+g3debug modules
passed (mpnum9.631s/9.742s). All25 named ID/chain suites passed fresh checked-debug
race and actual JS execution, recorded in `.cache/id-chain-integrated-race.log`
and `.cache/id-chain-integrated-js.log`. Full checked-debug vet and WASI package
compilation passed. WASI compilation is not runtime evidence. Coordinator
verified every frozen production/test SHA and each named RUN/PASS registration.
