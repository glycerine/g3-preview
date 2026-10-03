# SAT simplifier literal-indexed use lists

Selected source: `src/sat/sat_simplifier.h` use_list, complete bodies at
`src/sat/sat_simplifier.cpp:29–55`, the selected LAZY_USE_LIST implementation
in `sat_clause_use_list.h/.cpp`, and actual `vector.h` specializations.
This packet adds only `sat_solver_clause_use_lists.go` and its test file.
Standalone cold use-list files remain unchanged.

## Executable inventory

The source outer methods map to `Init`, `Insert`, `Erase`, `EraseExcept`,
`Get` (one owning-context adapter for both mutable and const source references),
`Finalize`, and explicit idempotent Go `Close`.
The by-value `satClauseUseListView` implements source `size`, `empty`, `insert`,
`erase`, `erase_not_removed`, `reset`, `check_invariant`, and `mk_iterator` via
`Size`, `Empty`, `Insert`, `Erase`, `EraseNotRemoved`, `Reset`, `CheckInvariant`,
and `Begin`. Its by-value iterator implements constructor consume, `at_end`,
`curr`, `next`, and destructor completion via `Begin`, `AtEnd`, `Current`,
`Next`, and explicit `Close`. Shallow copying a live iterator is prohibited.

Outer Init destroys the current logical inline elements in ascending order,
sets outer size zero, computes `2*num_vars` with source uint32 wrap, and performs
individual source capacity growth steps. Only after all steps succeed does
resize publish its final size and default construct all new inline elements.
Thus a later refusal retains completed capacities and the reset prefix. No
whole-operation rollback, literal deduplication, or all-index prevalidation is
used. Insert/Erase read clause size once and execute each literal action in
source order. EraseExcept skips every literal equal to its argument. Repeated
literals preserve release duplicate insertion and unsigned subtraction; debug
assertions occur at their actual child action, after earlier completed actions.

Child push constructs the primitive reference before publishing physical size
and live count. Lazy Erase keeps the primitive vector and decrements only live
count in release. EraseNotRemoved shifts only the first matching reference, then
decrements live count even when no match exists in release. CheckInvariant
executes actual was_removed callbacks in release as well as debug; only its
final count assertion is debug-only. Child Reset destroys its payload while the
inline list object remains alive.

Iterator captures physical vector size, not live count, and immediately
compresses the first surviving current element. Reads and writes resolve the
CURRENT child payload. Inner growth does not invalidate the inline child
reference. Current copies a retained raw clause token without an added payload
callback; the actual later was_removed callback guards the clause lifetime.
Next increments i and j before consume. AtEnd is only captured i==size and
performs no owner/lifetime dereference. Destructor completes only while i<size,
then publishes j via source primitive shrink. It deliberately differs from
AtEnd when a release caller has advanced beyond the captured end. The bounded
retained-tail regression first failed against the erroneous != completion loop
and now exercises the exact less-than condition in both profiles.

## Actual storage and lifetimes

One self-checked cold owner borrows the real clause allocator and owns one
concrete arena.Words owner. Clause allocator and its referenced clauses are
not owned or deleted by lists. A selected owner-local clause allocator must
outlive actual clause callbacks; foreign allocator tokens are not a substitute
for an allocator binding. Actual access guards diagnose invalid source domains
where C++ would read freed/foreign storage.

Each hot inline record is exactly 32 bytes, with no Go pointer or interface:
8-byte arena.VectorRef; uint32 physical length, source capacity, live size,
and initialized primitive-tail length (16 bytes); uint64 alive marker (8 bytes).
The outer record slice is the single actual fixed-stride inline array. Its slice
header resides in the charged cold owner; no per-literal cold object is created.
Each clause token is five uint32 words: owner low/high, slot, generation, kind.
Words descriptors/pages/directories/free metadata and their headers are actual
owned charged storage, not invented source heap objects. The pool's rounded
physical capacity is kept distinct from each child's source requested capacity.

Selected source architecture uses eight-byte pointers: outer element stride16,
source vector header8, inner pointer stride8. Go charges its ACTUAL owner,
32-byte record arrays, encoded payload words, and all Words ownership graph
bytes. Source uint32 extent checks and actual Go host extent checks are separate;
a simulated 32-bit-host fixture verifies the actual stride bound without a huge
allocation. Ordinary growth uses source capacities 2 then (3*c+1)>>1. Old and
new physical arrays overlap in Budget before old backing is released. Each
completed step remains published on a later refusal. The source outer
nontrivial move branch copies only logical constructed elements; ordinary Init
has already destroyed all elements and has zero live elements to move.

Views encode cold owner, index, and OUTER physical backing identity, not an
instance-generation epoch. A same-type inline object reconstructed at the same
retained address can transparently replace the earlier object; old views and
iterators follow the reconstructed list and current child vector. A destroyed
slot is rejected until reconstructed. Actual outer growth/free invalidates old
inline addresses and is diagnosed by the backing identity guard. Child payload
growth does not change that identity. No unsafe address arithmetic is used.
Primitive initialized-tail metadata rejects genuinely never-constructed pointer
slots; initialized retained tail is preserved through child growth/erase and is
available to defined release operations. Debug current-logical-size assertions
precede physical bounds guards at the actual source operator[] sites.

Pool storage is a physical adaptation: source child finalize frees its exact
allocation, whereas this owner releases its descriptor/block and retains reusable
pool pages until Close. Outer Finalize frees the actual outer backing and destroys
all logical children but retains its charged cold owner/Words infrastructure.
Close releases that complete graph and is idempotent. Tests sum every actual
owner/backing/page/directory/descriptor charge and compare Budget.Used before
teardown. Constructor failure cleanup restores the borrowed allocator baseline;
Insert faults preserve earlier literal actions and immediately expose exactly
those live Words descriptors. Warm cycles allocate zero host objects after
preparation. These APIs do not promise the C++ allocation profile.

## Tests and selected scope

No direct registered source use_list test assertions exist; source assertion
port count is **zero**. Eighteen permanent groups add source-action regressions,
including 64 finite independent removal-mask/ordered-compaction cases, duplicate
literal/repeated unsigned erasure, source reconstruction/growth reference
behavior, ended versus never-constructed slots, copied raw freed tokens,
later invalid literal and payload prefixes, constructor and insertion fault
sweeps, completed Init growth before refusal, pointer-free record/host extents,
complete physical graph accounting, and zero warm allocation.
Initial isolated missing-API RED is
`.cache/sat-clause-use-lists-red-isolated.log` (HEAD4745701 plus tests only).
The later destructor behavior RED is
`.cache/sat-clause-use-lists-close-red.log` (release captured size1, completed
extra Next to i=j=2 with initialized tail, erroneous Close attempted a third Next).

The header/source units remain PARTIAL: nonlazy selection, STRACE output,
implicit outer/list move and outer copy instantiations, true default simplifier
algorithms, actual runtime/solver construction, and registered end-to-end solver
tests remain open. No fake child solver or disabled simplifier defaults are
introduced. Ordinary outer Init does not require live-list move instantiation.
External stream/TRACE effects and unsupported source profiles are not claimed.
The actual full solver/proof/tactic goal remains open.

## Frozen verification

Final prospective tree is `.cache/release/sat-clause-use-lists-final`, committed
HEAD4c60187 plus only this packet's two Go files. Final named inventory is
`.cache/sat-clause-use-lists-final-testnames.txt`; hashes are
`.cache/sat-clause-use-lists-final-hashes.txt`. Earlier 17-group gates were
superseded by the destructor regression/correction; final evidence below refers
to the corrected 18-group bytes.

Final frozen Go SHA256, in dependency order:

- `sat_solver_clause_use_lists.go`: `7a660462a6e1bbd5f808c991e829f47e7faee22bd27a70b383e641aab46952d9`
- `sat_solver_clause_use_lists_test.go`: `196a5c0eb987a8c6c7a9ff7e0acc86751071c0c60c7fda388154587f799f1467`

Terminal corrected-byte gates:

- Full native `./...`: PASS session40059 (g3 2.886s; mpnum22.424s).
- Full actual `g3checked,g3debug ./...`: PASS session54451 (g3 3.099s; mpnum22.656s).
- Named18 checked/debug race: PASS session23615,1.062s; all18 actual RUN/PASS registrations verified against the frozen named inventory.
- Named18 actual checked/debug JS/WASM: PASS session1132,.181s; all18 actual RUN/PASS registrations verified against the frozen named inventory. Uses the existing absolute `.cache/js-exec` toolchain wrapper and actual Node runtime.
- Checked/debug vet `./...`: terminal exit0.
- Checked/debug WASI compile: PASS session80960, actual output `.cache/sat-clause-use-lists-final-wasi.test`.

Logs are `.cache/sat-clause-use-lists-final-{native,debug,race,js,vet,wasi,audit}.log`.
All artifact/cache/temp paths remain below golang. No Go generics, unsafe,
production math/big, fake callback, fake solver, or writes to central metadata/Git.

Words' physical capacity class limit (2^31 words), five-word token indexing,
and host extent checks are explicit concrete-storage bounds distinct from
source pointer capacity. They do not claim that all hypothetical source giant
allocations can be represented by this backend. Invalid/destroyed references
receive typed errors at actual accesses. Interrupted invalid-source iteration
may retain already completed scalar/compaction prefixes; it is not rolled back
or claimed to have a generally safe C++ retry contract.
- Final source/no-generics audit: PASS session10335 on corrected frozen bytes;
  `2102 units, 677 APIs: 33 ported, 142 partial, 1927 unported` in that
  prospective committed-base ledger. This packet adds no central rows itself.


## Cache-only warmed workload measurement

The isolated performance tree `.cache/perf/sat-clause-use-lists-final` copies
exactly the frozen two Go files from the verified final tree. Its only added
benchmark file is cache-only `sat_solver_clause_use_lists_perf_test.go`.
Production and permanent tests were not changed. The actual sequential command
for each mode is `go test -run '^$' -bench
'^BenchmarkSATClauseUseListsWarmedRealistic$' -benchtime=1s -count=5 ./`, with
`-tags=g3checked,g3debug` for debug. The final release campaign completed before
the debug campaign began; preliminary concurrent samples were discarded.
Host: linux/amd64, AMD Ryzen Threadripper3960X24-core, benchmark suffix48;
Go `go1.26.4-pont-onethread linux/amd64`. Other machine activity is uncontrolled.
These are local component timings, not a C++ comparison or whole-SAT forecast.

Each timed operation does source Init(2), inserts the SAME still-live real
three-literal clause [0,1,2], then for each literal obtains its current inline
list, constructs the compressing iterator, visits Current, reads Size through
the actual clause allocator, advances Next, and closes the iterator. Thus three
real current clause visits accumulate checksum9 per operation. No clause/model
or solver callback is fabricated. Clause construction, cold owner creation,
and one warm cycle occur outside timing. Timing includes subsequent destruction
of all four logical lists, recreation, three pooled reference insertions, and
all three lexical iterator completions. ReportAllocs is enabled. The benchmark
asserts the retained Budget baseline stays constant and full Close refunds to
borrowed allocator baseline, then allocator Close refunds to zero.

- native: 2346, 2062, 2083, 2098, 2055 ns/op; median 2083 ns/op; 0 B/op; 0 allocs/op
- debug: 2530, 2506, 2457, 2540, 2497 ns/op; median 2506 ns/op; 0 B/op; 0 allocs/op

Actual prepared outer graph in both profiles: owner96B; five-record backing160B
(logical length4, source capacity5); Words owner80B; descriptor owner88B;
descriptor records16384B, metadata12288B, directory56B; word page payload256B,
block metadata16B, word directory64B. Three live child payloads each have source
capacity2 and ten encoded words, using rounded pooled capacity16words. Total
outer graph29488B. Words page size64 and default descriptor page size1024 are
actual constructor settings. The borrowed allocator plus one real clause costs
42392B in release (total Budget71880B),30168B in checked/debug (total59656B,
source debug DirectOnly allocator profile). All cold headers/directories/pages
are counted once; no per-literal heap object/header is charged or allocated.

The 0B/op result describes host allocation after warming, not zero retained
memory: the29488B outer graph is retained and accounted. The descriptor page
is the largest cold component for this tiny workload; changing its geometry
would require a separate measured proposal, not a claim of source allocation
fidelity. Logs `.cache/sat-clause-use-lists-bench-{native,debug}.log` contain all
five samples, exact prepared graph, CPU header and terminal PASS. Cache-only
benchmark checksum and source clause size prevent empty-loop substitution.


## Coordinator integration review

Frozen integration `.cache/review/sat-clause-use-lists-integration-4c60187`
contains the committed base plus only the two reviewed Go files and selected
metadata. Full native session5518 and checked/debug session81475 both terminated
exit0; root-package times2.678s/2.994s and mpnum21.205s/21.955s. Source/no-generics
audit session43200 terminated exit0 with2102units677APIs33ported/144partial/1925unported.
Coordinator independently verified both exact eighteen RUN/PASS inventories
in terminal race/actualJS logs against unchanged production/test SHA256 values
in `.cache/sat-clause-use-lists-integration-hashes.json`. Reviewed the wrapper
and inline/iterator bodies against the preserved source, including the
destructor less-than correction established through an actual RED regression.
Vet/WASI and local storage benchmark evidence above apply to the same Go bytes.
Remaining untracked polynomial/solver/numeral TDD drafts are outside this frozen
chunk; this is not a claim that the entire live worktree or full port is green.
