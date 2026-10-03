# Arena foundation packet

This packet implements B7's independent slot/slab/budget storage contracts, not
the complete C++ region allocator or the complete solver memory subsystem.

## Source mapping and scope

The slab representation is the intentional Go ownership adaptation specified by
GO_PORT_PLAN.md B6/B7. It is not a line-for-line translation of region.cpp.
The source `src/test/region.cpp::tst1` is empty (TODO), so it contains no assertions
to translate. Source region branches were read, and remain explicitly open:

| Source method/branch | Counterpart or remaining work |
| --- | --- |
| region constructor/default-page allocation | Slab constructor defines page policy; scoped region construction remains open |
| region destructor / debug destructor | Slab.Close drops backing pages and ledger after caller cleanup; region teardown remains open |
| allocate: in-page, new default page, large dedicated page | Slab.Allocate implements fixed-record growth only; variable payload and dedicated region pages remain open |
| recycle_curr_page | Slab.TrimFreePages drops wholly dead fixed-record pages, retains generation ledger; region page recycle remains open |
| reset, release/default-page reuse | Slab.Reset invalidates live records, retains ledger, drops record pages; region reset remains open |
| push_scope/pop_scope, counted pop_scope | Not implemented; required by arena/region.go and domain cleanup/trail integration |
| display_mem_stats | Slab.Stats exposes fixed-record capacity, metadata, directory and liveness; source textual region diagnostic remains open |
| placement new/delete and address alignment | Pointer-free Go typed records replace placement allocation; no unsafe or C layout is required |
| debug per-object chunks/scopes | Not translated; Go checked/release domain handles are outside this package |

## API and ownership

- Slots/generations are uint32; slot zero and generation zero are invalid.
- Concrete slab record audits use ValidateRecordType recursively. Pointers, strings, slices, maps, interfaces,
  funcs, chans, unsafe.Pointer, and uintptr are rejected, including nested
  fields and arrays. Arrays/structs of numeric scalars are accepted.
- Config defaults PageSize to 1024, SlotLimit to MaxUint32; pages must have a
  power-of-two size. Make-length arithmetic is guarded on 32-bit hosts.
- Allocate publishes a zero record, stable slot and increasing generation only
  after predictable budget checks. Read copies records; Write updates them.
  There are no pointer/slice borrows or callbacks across page growth.
- Recycle requires domain destruction/cleanup to have run beforehand. It
  zeros the record, invalidates liveness and links the slot for reuse.
  MaxUint32 generation permanently retires the slot; it never wraps.
- TrimFreePages preserves all metadata. Reset invalidates all live records
  after caller cleanup and drops record pages. Close releases records and
  metadata and permanently rejects operations. All three are owner operations.
- A Slab does not run constructors, destructors, source reset hooks, logical
  reference release, or vector cleanup. Constructed-object pools must preserve
  their pooled state in a separate domain policy; generic Recycle zeroes records.
- Budget is single-owner: NewBudget(limit), Reserve, Release, Used, Limit,
  FailAfter. Nil is unlimited; limit zero allows zero bytes. Injected failures
  affect reservations, never warmed allocations. Release underflow panics as
  an ownership invariant violation.
- Budget charges actual typed backing-array capacity (including unused page
  slots), scalar metadata, and directory capacity. Directory copying reserves
  peak old+new capacity before publication. Owners separately account for cold
  slab/control headers, external scratch and cleanup reserves; these are not
  slab backing storage. Host runtime allocator overhead/OOM is not recoverable
  or accounted as exact process memory by this abstraction.
- Slot-only Read/Write cannot detect a stale slot after reuse. IsLive requires
  a generation. Public owner/domain validation and checked widened handles
  remain coordinator/domain responsibilities.

## Tests and evidence

Tests were written first. `go test ./arena` initially failed to compile with
undefined NewSlab/Config/errors (RED). Implementation then passed those tests.
A second RED test exposed missing peak directory accounting; the fix passed.

Independent deterministic state-machine oracle: 4,000 operations, 17-slot
limit, 2-record pages, live/token/value maps independent of allocator metadata.
FuzzArenaOps uses at most 512 operations, 7 slots and 2-record pages with the
same independently recorded token/value expectations. No test reads C++ files,
uses a reference executable, neighboring repository, network, or Python.
Source contains no applicable region test assertions beyond its empty TODO.

Verification on go1.26.4-pont-onethread, linux/amd64:

- CGO_ENABLED=0 go test ./arena: PASS.
- CGO_ENABLED=0 go test -tags=g3checked ./arena: PASS; slab semantics are
  identical under both tags (domain checked handles remain outside this packet).
- go vet ./arena: PASS.
- FuzzArenaOps, 3 seconds, 2 workers: PASS, 3,338 executions, 27 new interesting.
- CGO_ENABLED=0 GOOS=js GOARCH=wasm go build ./arena: PASS.
- CGO_ENABLED=0 GOOS=wasip1 GOARCH=wasm go build ./arena: PASS.
- GOARCH=386 go test ./arena: runner failed with `signal: bad system call`;
  no actual 32-bit execution acceptance is claimed.
- BenchmarkSlabWarm, 200ms, Ryzen Threadripper 3960X: 25.38 ns/op,
  0 B/op, 0 allocs/op. This is a warm fixed-record cycle, not a solver benchmark
  or comparative speedup. AllocsPerRun additionally requires exactly zero
  allocations for allocate/write/read/recycle inside existing pages.

Subsequent tool cache/temp/artifact locations are golang/.cache. The initial
RED command used the default Go cache/temp locations before the additional user
restriction was relayed; it may have produced external compiler cache writes.
All authored source/status files are inside golang/arena.

## Remaining requirements

words.go, vectors.go, region.go, marked rollback, payload descriptors/classes,
large dedicated pages, optional compaction, region/trail domain cleanup and
source debug/release region conformance are unimplemented. Public owner/domain
handle checks, actual wasm execution, actual 32-bit execution and integrated
source destruction/constructed-pool behavior require later packets. No complete
allocator/full-fork milestone is asserted by this packet.

# Variable payload packet

The later `words.go` and `vectors.go` packet implements B8's non-compacting typed
payload representation. The earlier statement that those files are unimplemented
describes the first packet only; marked regions and compaction remain open.

## Contracts and ownership

VectorRef contains only a descriptor Slot/Generation. Every nonempty/reserved
vector owns one Slab descriptor and one geometric-class block. Descriptor
identity remains stable through Reserve/Resize/Append. Payload locations are
private scalar page/offset/length/capacity fields; no backing slices escape.
Zero VectorRef denotes an allocation-free empty vector. Allocate(0,nonzero)
reserves a live descriptor. Free zero succeeds; IsLive zero is false.

VectorConfig.PageSize is a power-of-two element count, default 16384; this is
64 KiB for Words (uint32) and 128 KiB for uint64 checked tokens. Descriptors.Budget
unambiguously charges descriptor slab, payload data, scalar block free links,
and page-directory capacity. Large classes use a dedicated single-block page.
Class capacity is the least power of two at least the requested capacity;
capacities above 2^31 and target-specific oversized make extents fail explicitly.

Read/Write check logical length and generation. Reserve copies logical contents
then publishes the new location within one owner operation. Resize retains
capacity when shrinking and clears discarded values; subsequent growth is zero
initialized. Free always releases the original capacity extent, clears records,
and invalidates/recycles its descriptor. No element constructors, callbacks,
destructors, domain handle retains/releases, or ownership transfer are inferred
from a scalar records. Domain cleanup must precede truncation/free/reset/close.

Failed creation leaves no live descriptor or payload, but may retain newly
allocated reusable descriptor capacity and generation metadata. Those bytes
remain charged and visible in Stats, and Close releases them. Failed growth of
an existing vector preserves identity/content/length/capacity. Directory growth
charges peak old+new directory bytes; copying also retains the old payload until
publication. No source addresses, page indices, or empty vector values are
public identities. Trim drops wholly free payload/descriptor record pages only;
generation metadata and deterministic page class headers survive. Reset drops
payloads and invalidates descriptors; Close releases all arrays/metadata.

## Source mapping

| Source behavior | Storage counterpart / limitation |
| --- | --- |
| vector constructor/default empty and buffer empty | Zero VectorRef; no separate empty allocation |
| vector size/capacity/operator[] and buffer get/set | Len/Cap/Read/Write, explicit bounds errors rather than debug-only unchecked accesses |
| vector expand_vector and buffer expand | Reserve relocates scalar contents into geometric pooled classes; 1.5x vector growth is intentionally not copied into the storage-class policy |
| vector/buffer push_back | Append, zero-growth then write |
| vector resize/shrink, buffer resize/shrink/reset | Resize including zero length preserves reserved extent; owner-local allocator Reset additionally drops payload pages |
| vector/buffer pop_back/back | Read(last index) followed by Resize(length-1); no standalone source container facade claimed |
| vector destroy/free_memory/finalize, buffer free_memory/finalize | Free, allocator Trim/Close; domain destructors remain explicit caller obligations |
| vector copy/move/assignment/swap/iterators and buffer copy/assignment/append/set_end | Not implemented as source container APIs; later util/domain port must provide these behaviors |
| ptr_scoped_buffer element deletion from src/test/buffer.cpp | Not translated here; pointer ownership conflicts with scalar payload storage and belongs in a domain wrapper |

Finite assertions from `src/test/vector.cpp::tst1` are ported in
TestSourceVectorTst1: initial emptiness, 1000 ordered pushes and readbacks,
capacity >= length, forward traversal, reverse back/pop, final empty length.
Its unbounded billion-element allocation loop becomes a finite budget-error
regression. This does not claim runtime host OOM recovery. The actual C++ source
growth policy and owning element destructor behavior remain separate from B8's
intentional storage-class adaptation.

## Verification

The new vector tests first produced RED undefined NewVectors/NewWords/VectorRef
compile failures, then passed after implementation. Tests include 2000 bounded
state-machine operations with independent Go slice/value models, repeated
grow/shrink/free plateau, live block non-overlap, payload preservation across
dedicated large-page growth, reserved-empty semantics, generation rejection,
failure cleanup, original capacity preservation, peak directory budgets and
pointer-freedom rejection.

- CGO_ENABLED=0 native and g3checked arena tests: PASS, including uint64 vectors.
- go vet ./arena: PASS.
- js/wasm and wasip1/wasm package cross-builds: PASS.
- GOARCH=386 test binary compilation: PASS. Actual 386 execution remains
  unverified following the prior host bad-system-call failure.
- BenchmarkVectorWarm, 200ms, same host/toolchain as first packet:
  72.60 ns/op, 0 B/op, 0 allocs/op for warm words allocate/write/read/free.
  AllocsPerRun also verifies exact zero allocations for the bounded warm cycle.

No compaction method is offered. Safe-point relocation, peak old+new payload
tables and domain raw-offset registration remain deferred together. Region
marks/rollback, byte/string pools, domain cleanup, source container facades,
actual wasm execution and full solver integration remain incomplete. No shared
manifest or previously agreed slab/budget interface was modified by this packet.

# Typed marked-region packet

`region.go` now supplies independent typed marked storage. Earlier references
to its absence describe preceding packets. This packet does not implement solver
trail variants or the raw byte/address source allocator layout.

Region32/Region64 own Slab32/Slab64, a scalar RegionRef allocation log and scalar
serial/length mark records. RegionRef carries Slot/Generation; RegionMark is an
opaque cold owner identity pointer plus serial token. The mark's owner pointer
is not stored in hot record or mark backing arrays. Marks are never reused:
serial exhaustion permanently prevents new marks, even after Reset. Every
Allocate explicitly writes its initial scalar records. Reads copy values; no raw
pointer or slice borrow exists. Individual object free is deliberately absent.

Mark/Rollback are strictly nested. Rollback rejects zero, foreign, consumed and
non-top marks. PopScopes explicitly supports a validated counted pop. Reset
cleans all records, including base-level allocations, and invalidates all marks.
Close is idempotent and drops backing arrays and ledger. TrimFreePages returns
an error so callback reentrancy can be rejected; it drops only fully dead record
pages while retaining generation metadata and reusable scalar control arrays.

The transient Cleanup32/Cleanup64 parameter is never stored as a trail closure. It runs
in reverse allocation order; every target record stays live through *all*
callbacks, after which a separate pass recycles records. This is a documented
storage cleanup contract, not a claim that source trail objects individually
remain live after source undo/destruction. Reads/Stats are allowed in callbacks;
Allocate/Write/Mark/Rollback/PopScopes/Reset/Trim/Close reject reentrant mutation.
A callback panic preserves allocator records/log/marks and clears the busy flag.
External callback side effects cannot be rolled back; repeated cleanup after
panic is not promised safe. A nil cleanup means caller cleanup already ran.

Budget checks reserve peak old+new log/mark backing capacity before copying.
An injected/limit failure during those growth steps leaves that array and budget
unchanged. If subsequent slab allocation fails, already prepared reusable log
capacity may remain retained/charged; no live region record is published.
Stats separates log/mark bytes and slab capacity/metadata. Cold owner headers,
caller scratch and source cleanup reserves remain explicit owner responsibilities.

## Source mapping and inspection

Read source region.h/.cpp/trail.h plus object_allocator.h,
small_object_allocator.h and both allocator tests before implementation.
`src/test/region.cpp::tst1` is still only TODO, and no dedicated source trail
test file exists. `src/test/small_object_allocator.cpp` exercises allocation
reuse but contains no ENSURE assertions. `src/test/object_allocator.cpp`
asserts coefficient reset, worker capacity and constructed/destructed balance;
those belong to its reset-only constructed pool. This region does not substitute
zero/reconstruction for that pool's reset-only lifecycle or claim its port.

| Executable source behavior | Typed counterpart / limit |
| --- | --- |
| region allocate/default pages and debug chunks | Region.Allocate uses stable typed slab records, not source address alignment/chunks |
| region push_scope mark(page,cursor,previous) | Mark logs typed used length and serial, explicit nesting |
| region pop_scope and counted pop | Rollback/PopScopes invalidate only suffix allocations, preserving prefix and generation ledger |
| region reset/recycle/default-page reuse | Reset recycles all typed records while preserving reusable capacity; lazy typed allocation differs from source eagerly keeping one default page |
| region destructor/display_mem_stats | Close releases all backing metadata; Stats reports typed capacity, not source textual pointer/chunk diagnostics |
| region large allocation and placement operators | Variable payload extents are Words/Vectors; this Region is fixed-record typed storage, no source address API |
| undo_trail_stack reverse loop/old_size assertion | Reverse transient cleanup followed by suffix truncation, checked mark lengths; independent scalar undo-state oracle |
| trail_stack reset base-level undo | Reset includes records before the first scope |
| trail_stack push_scope/pop_scope | Mark/PopScopes coordinate typed record reclamation only; real domain undo entries and subsystem callbacks remain open |
| value/reset_flag/restore_size/vector/map/ref/new_obj trail subclasses | Not translated by generic storage; tagged domain dispatch and logical ownership remain higher-layer work |

Read-only inspection checksums:

- region.h: fab2ab6cdc6bdc067f38a8908c7591e9dc90bfea116823d2c744ead9238b32d1
- region.cpp: 23908446c761dfb4e7b1165f2545f69f3e8726b7139f07b30133488559562eb3
- trail.h: 453e4c187865dbf98893988f51152c67bf5ae98d0a69caf9c4640eea94136a6c
- test/region.cpp: 09bc4beded6a1b360db26f37f2b3df70e0c525be0d586896acd0018cdda0243c

## TDD and checks

Observed initial RED: undefined NewRegion/RegionRef/ErrMark/ErrReentrant before
implementation. GREEN tests cover nested prefix/suffix liveness, foreign and
consumed marks, counted scopes, zero-level reset cleanup, cleanup before recycle,
reverse dispatch, all mutation reentrancy, callback-panic storage preservation,
generation persistence through reset/trim, mark serial exhaustion, injected mark
and log growth failures, peak old+new mark/log budgets and Close budget balance.
An independent scalar undo/checkpoint oracle runs 3000 finite operations. These
tests do not require reading source or invoking a reference executable.

- Native CGO-disabled arena tests: PASS.
- g3checked arena tests, go vet, js/wasm and wasip1/wasm package builds: PASS.
- GOARCH=386 test binary compilation: PASS; actual 32-bit execution still awaits
  a compatible runner following the earlier bad-system-call failure.
- FuzzRegionTrailOps: PASS, 3 seconds, 2 workers, 140338 executions,
  33 new interesting cases; each input is capped at 512 operations.
- BenchmarkRegionWarm: 37.02 ns/op, 0 B/op, 0 allocs/op for warm
  mark/allocate/rollback on the previously recorded host/toolchain.
- AllocsPerRun independently requires exact zero allocations for that warm cycle.

Remaining: source typed/domain trail variants, constructed-object pools and
their destructor/reset behavior, SMT pop ordering, region-owned variable payload
marks, cleanup-list integration, promotions into persistent owners, optional
compaction, source diagnostic formatting and actual wasm execution. This packet
does not close full memory-subsystem or full-fork acceptance.

# Concrete arena conversion (user requirement: no Go generics)

The current arena uses only concrete Go types. Earlier RED/GREEN constructor
names in this report describe historical packets, superseded by the concrete
API below. No type-parameter declaration, instantiated generic type/function,
reflect.TypeFor, per-record interface storage, unsafe code, or reusable generic
template remains in arena production or test source.

| Concrete storage | Constructor / use |
| --- | --- |
| Slab32 | NewSlab32(Config), uint32 scalar records |
| Slab64 | NewSlab64(Config), uint64 scalar records |
| descriptorSlab | Private newDescriptorSlab(Config), vectorDescriptor records |
| Words | NewWords(VectorConfig), uint32 payloads, previous public Words API preserved |
| Vectors64 | NewVectors64(VectorConfig), widened scalar handles |
| Vectors32Signed | NewVectors32Signed(VectorConfig), exact int32 source-vector fixture values |
| Region32 / Region64 | NewRegion32/64(Config), Cleanup32/64 transient caller dispatch |
| recordSlab | Test-only concrete rich record allocator, preserving fixed-array/uint64 record regression coverage |

Config, Stats, VectorConfig, VectorStats, RegionStats, Slot, Generation,
VectorRef, RegionRef, RegionMark and Budget contracts are unchanged.
ValidateRecordType(reflect.Type) provides a nongeneric recursive audit; nil
types and all previous pointerful/uintptr categories are rejected. Constructors
use reflect.TypeOf only. Fixed pointer-free record pages remain one backing
allocation per page; concrete duplication does not introduce per-record objects.

Tests migrated first to missing concrete API names and produced RED compile
failures, then GREEN after concrete implementations. All earlier lifetime,
state-machine, failure injection, non-overlap, capacity plateau, generation
retirement, budget peak, cleanup/reentrancy, source vector assertions, fuzz seed,
and zero-allocation checks remain. The undo oracle stores its former single-field
uint32 old-value payload directly in Region32; its independent checkpoint model
and operation count are unchanged. Extra direct concrete slab tests verify
32/64-bit maximum values, slot limits, failure-atomic page recovery, generation
retirement, reset/trim and budget balance. Signed vector and Region64 checks
verify negative/wide values survive relocation and callback dispatch.

Conversion verification on the previously recorded host/toolchain:

- CGO_ENABLED=0 native arena and g3checked arena tests: PASS.
- go vet ./arena: PASS.
- js/wasm and wasip1/wasm package builds: PASS.
- GOARCH=386 test binary compilation: PASS; prior actual native 386 runner
  limitation still applies.
- Actual js/wasm arena tests under Node v22.22.2 with the toolchain-matched
  go_js_wasm_exec: PASS. Initial inherited environment exceeded wasm argv/env
  limits; retry with a minimal explicit environment passed.
- FuzzArenaOps, 3 seconds/2 workers: PASS, 43693 executions.
- FuzzRegionTrailOps, 3 seconds/2 workers: PASS, 162583 executions.
- Warm benchmark remeasurement (200ms): record slab 25.00 ns/op, words
  62.23 ns/op, region 42.71 ns/op; all 0 B/op and 0 allocs/op. These short
  measurements establish no comparative performance claim.

Temporary original generic snapshots and one-shot conversion scripts were
removed after verification. Only concrete reviewed source is retained. No
domain algorithms, source traversal policy, allocation class policy, cleanup
ordering or previously listed incomplete subsystem features changed.

# Concrete constructed scalar pool packet

`pool32.go` supplies the approved **constructed=true scalar policy subset** of
`src/util/object_allocator.h`. It is not a whole source-template port and does
not complete `small_object_allocator` or the real rational-cell source tests.
It introduces no generics, per-record interface/closure, intrusive free-list
overwrite, unsafe operation, or edits to existing slabs/vectors/regions.

## Source lifecycle and API

Pool32Config defaults to 8 workers and 1024 scalar records/page, exactly the
source constants (`object_allocator.h:26`). SlotLimit is a testable physical
slot limit **per worker region**. Each worker eagerly owns an empty first page;
filling a page immediately prepares another empty page, including at a configured
slot limit. A fresh initialized slot invokes the one owner-level Construct
dispatcher once. A recycled slot invokes neither Construct nor Destroy.

Pool32Policy.Reset is called on Recycle before publishing the ref onto the
chosen worker's separate LIFO free array. It returns the retained scalar state.
Nil Reset is the source do-nothing reset policy, so recycled state survives
unchanged. Nil Construct gives the concrete scalar zero initializer.
Pool32Policy.Destroy runs once per initialized record on Reset/Close, including
records on free lists and records retired after generation exhaustion.
DisableDestructors suppresses Destroy only. No destructor runs on Recycle.
Reset destroys old objects and creates fresh construction opportunities while
preserving generations; Close is idempotent and releases all backing storage.

Source destructor traversal is preserved: workers in order, last partially used
page first, then earlier full pages in allocation order; an eager empty tail page
has no initialized values to destroy (`object_allocator.h:78`). The source's
header prose says destructors run only on allocator deletion, but its executable
region::reset calls free_memory and therefore runs them during Reset too
(`:118`). The implementation follows the executable path.

Pool32Ref stores original RegionWorker, Slot and Generation. Cross-worker recycle
puts that ref on the destination worker's free list without changing physical
ownership. Reallocation increments its generation; old refs never revive after
reuse or Reset. Exhaustion retires the slot permanently. Its constructed value
survives retirement until the single applicable Reset/Close destructor.

Allocate/Recycle use worker zero in nonconcurrent mode. AllocateWorker/
RecycleWorker and the cold Pool32Worker wrapper require concurrent mode.
ReserveWorkers and Reset reject concurrent mode, matching source assertions.
Worker wrappers survive reserve and mode toggles. Close works in either mode,
as the source destructor does. Read/Write/IsLive validate live generation;
public context-owner validation remains the domain wrapper's responsibility.

A cold owner mutex serializes metadata, dispatch and budget mutation. Worker
partitions and destination semantics are preserved, but source parallel throughput
is not claimed. Policy callbacks **must not call methods on the same pool**,
because that mutex is held; doing so would deadlock. Callbacks must not panic.
External callback effects are not transactional, and reuse/retry after a panic
is not supported. This is an explicit caller invariant for the finite dispatcher
policy. Shared budgets across other owners require external synchronization.

Budget accounts record/ledger pages, worker/page directories, separate free
arrays and Reset scratch. Page/directory and free-array growth charges old+new
capacity. Reset preallocates replacement first pages and scratch while old pages
still exist, failing before destructor effects. Page-boundary growth is checked
before Construct, and free-array growth before Reset policy effects. This is a
predictable-limit adaptation: the C++ source runs construction/reset before a
subsequent vector/page allocation can fail. No exact recovery of host runtime OOM
is claimed. Failed multi-step preparation may retain reusable charged capacity,
but never publishes a new constructed record or invokes its Construct callback.
Cold owner/policy headers and callback-owned allocations are separately budgeted
by the caller. Generation/tombstone ledgers remain measurable after Reset.

## Source method coverage and pending source assertions

| Source | Concrete counterpart / pending behavior |
| --- | --- |
| object_allocator constructor/reserve/capacity (`:150`, `:172`, `:187`) | NewPool32/ReserveWorkers/Stats.Workers, eager first pages |
| region allocate (`:108`) | Allocate fresh scalar construction, eager boundary page; construct=false remains pending |
| allocate_core (`:136`) | Destination worker LIFO first, no reconstruction on reuse |
| recycle_core (`:146`) | Reset-only policy, separate scalar free refs, no destructor |
| worker access/mode/wrapper (`:164`, `:229`, `:252`) | Explicit mode errors, indexed operations and persistent cold wrapper |
| region destructor/reset (`:103`, `:118`) | Pool Reset/Close perform exact initialized-value destructor traversal |
| get_objects_count (`:276`) | Stats.Live tracks allocated-minus-free logical objects; generation retirement is an additional storage category |
| do_nothing_reset_proc/simple_reset_proc (`:30`, `:36`) | Nil Reset / supplied scalar reset dispatcher; real cell reset implementation is not replaced with fake arithmetic |
| source tst1 rational coefficient zero (`test/object_allocator.cpp:65`) | **Pending** actual rational-cell manager test; no integer stand-in claimed |
| source tst2 default worker count (`:70`) | Finite default-capacity assertion ported |
| source tst2 rational value zero/preservation (`:86`, `:96`) | **Pending** actual rational manager/cell integration |
| source constructed/deallocated balance (`:121`) | Finite scalar-policy lifetime/counter projection, explicitly partial; real rational-cell source instance remains pending |
| small_object_allocator source/tests | Read and inventoried only; byte alignment, chunk lists, debug Valgrind path, exact size accounting and consolidate algorithm remain unported |

Read-only source fingerprints:

- object_allocator.h: e2e1d6936945317d9e7af20cb1f564fb0ef384deb90360157b262ad50a7a291a
- test/object_allocator.cpp: 03098e9577fb03cd306eb6302dc70570d6e8efafc9b6f59aab77e300bdff88d3
- small_object_allocator.h: 57b0a3bbc2d9fd10f008f371e3b5b9bee5535d39ab5cfbcb4afd9324018a140b
- small_object_allocator.cpp: 70cf910ed0a3871ea110ee01ff8d23a687f74af5972ab995dce32ba3e6e35994
- test/small_object_allocator.cpp: 98d72b253f1b8bda2ef5e6c9ad460064ceaee571ae15a7016b37cbf55ea79618

## TDD and verification

Tests first produced RED undefined NewPool32/config/policy/mode errors, then
GREEN after implementation. A subsequent RED regression detected generation
wrap when Reset discarded a still-live MaxUint32-generation record; Reset now
retires that slot without extra Reset-policy calls or duplicate destruction.

Tests verify source defaults and eager pages, do-nothing retained state,
constructor/reset/destructor timing, cross-worker destination, wrapper validity,
source destructor order, generation reset/retirement, disabled destructors,
failure-before-callback behavior, peak Reset budgets, budget balance, pointer-free
record metadata and actual concurrent workers. The independent deterministic
model executes 2000 bounded operations with retained scalar values and exact
constructor/destructor counts; model selection does not depend on map order.
All ordinary fixtures remain hermetic and require no source/reference process.

- Native and g3checked CGO-disabled arena tests: PASS.
- go vet ./arena: PASS.
- Native race check of all TestPool32 cases: PASS, including concurrent workers.
- js/wasm and wasip1/wasm package builds: PASS.
- Actual js/wasm arena tests under Node v22.22.2, matched wrapper/minimal env: PASS.
- GOARCH=386 test compilation: PASS; prior native 386 runner limitation persists.
- BenchmarkPool32Warm, 200ms on the recorded host/toolchain: 30.06 ns/op,
  0 B/op, 0 allocs/op for warm allocate/recycle. AllocsPerRun independently
  requires exactly zero allocations. No source throughput comparison is claimed.

Whole-template ownership, construct=false, real rational-cell assertions and
small-object allocator behavior remain incomplete. This partial packet must not
be used as evidence that those source components or the full memory gate are done.

# Concrete SmallBytes release pooled + direct storage packet

Owned files: `small_bytes.go`, `small_bytes_test.go`, this status append. No
shared slab/vector contract, other package, manifest or git changes. No generics,
unsafe, per-record interface or escaped byte slices.

`SmallBytesConfig` selects source pointer-width size policy (Alignment4/8,
explicit default8), source default ChunkSize8192-2*Alignment, and Descriptors
Config for slot/page/budget controls. Alignment is a size-class policy, not a
physical Go pointer-alignment promise. Tiny chunks exercise exact boundaries.
`SmallBytesRef` is a checked pointer-free Slot/Generation descriptor identity;
internal existing concrete descriptorSlab stores chunk/offset/size/capacity.

| Method/behavior | Source mapping and adaptation |
| --- | --- |
| construction | util/small_object_allocator.h:27-38 constants/lists/counters; cpp:28-36 empty initialization |
| Allocate(0) | cpp:97-98 null; distinguished zero ref, no backing allocation |
| pooled Allocate | cpp:111-137 ceil(size/Alignment) class, LIFO free list, strict new_curr<end, fresh chunk; fixed scalar links replace intrusive pointer writes |
| direct Allocate | cpp:104-106 size>=256-Alignment; dedicated pointer-free byte block, no pooled free-list entry |
| Deallocate(0) | cpp:72-73 no-op; closed Go owner still rejects operations |
| nonzero Deallocate | cpp:80-93 requested-byte decrement, immediate direct release or class free-list prepend; Go additionally validates exact original requested size and generation |
| Stats.RequestedBytes | header:48 m_alloc_size / cpp:82,104 requested logical bytes, distinct from aligned free waste and owned backing |
| Stats.FreeObjects/WastedBytes | cpp:140-166 free list count and aligned class-byte sum; dedicated blocks never counted as pooled waste |
| Consolidate | cpp:174-245 floor(CHUNK_SIZE/classSize) count gate and per-chunk reclaim/relink; does not move live allocations |
| Reset/Close pooled | cpp:39-67 pooled chunk traversal and counter/list clearing; generation ledger survives reset |
| Reset/Close direct | explicit tracked-owner cleanup extension: source direct blocks are separately caller-owned and not traversed by reset/destructor; Go releases all owned direct backing and invalidates refs, after caller logical-object cleanup |
| Size/Read/Write/IsLive | Go storage access/checked lifetime adaptation; no source public borrow or C++ raw-address alias claim |

Consolidation preserves an awkward exact source behavior: strict endpoint
allocation usually provides one fewer object for exact-dividing classes, but
consolidate still requires floor(chunk/class) free objects. Those wholly dead
chunks remain retained. Likewise incompletely filled free chunks remain retained.
The new tests deliberately verify this rather than silently improve reclamation.
Raw-address sorting adapts to deterministic chunk/block conceptual addresses;
reclaimed chunk indices can be reused, bounding repeated reclaim/grow metadata.
Reconstructed free-list order follows descending conceptual addresses. Original
machine address order is not reproducible or asserted. Free byte values are
unspecified; separate scalar metadata does not promise retained contents.

Budget includes descriptor records/generation metadata/directories, byte payload,
scalar block metadata, and full actual-capacity chunk directory. Growth reserves
old+new directory backing and new chunk arrays simultaneously before publication.
Consolidation uses no allocated scratch. Failed allocate can retain reusable
charged descriptor capacity/generation history but publishes no live allocation
and leaves existing content intact. Runtime OOM is not recoverable. Caller
separately budgets the fixed cold owner header and performs domain destruction.

Tests-first RED: missing SmallBytes/config/ref APIs. Pooled GREEN then direct
RED missing direct stats; pooled+direct GREEN. Source test/small_object_allocator
has no ENSURE assertions, only traces: the finite13/14/1-byte source script is
preserved, with independently added counts/LIFO checks. An independent3000-step
copied-byte model checks values, live/dead identities, non-overlap, reset and
consolidation; other tests cover tiny exact/nondividing chunks, class thresholds
for both pointer-width policies, release of live direct blocks at owner teardown,
failure injection, budget peak growth, generation retirement, budget balance,
reclaim/grow plateau, and0 warmed Go allocations.

Validation at pooled+direct GREEN:

- CGO-disabled native and g3checked full arena tests, go vet: PASS.
- js/wasm and wasip1/wasm package builds: PASS.
- Actual Nodejs wasm execution of compiled current test binary printed all10
  TestSmallBytes case names and PASS. Compilation uses workspace Go wrapper;
  only Node execution has minimal env, preserving TMPDIR under golang/.cache.
- Warm benchmark100ms:55.11ns/op,0B/op,0allocs/op on recorded host/toolchain.

Diagnostic Z3DEBUG&&!WINDOWS direct-only bypass remains inventoried pending next
packet; its source m_alloc_size stays0 because its allocate/free branches return
before counter updates. Whole AST allocator/domain ownership and real rational
constructed-pool tests remain pending. This storage packet does not establish
source raw-pointer ABI behavior or finish the full memory/solver gate.

# SmallBytes diagnostic direct-only follow-up

`SmallBytesConfig.DirectOnly` now translates the diagnostic allocation/free
bypass at util/small_object_allocator.cpp:75-78 and100-102 (Z3DEBUG&&!WINDOWS).
Every nonzero request gets a dedicated byte block regardless of small class;
Deallocate releases its payload immediately. No pooled chunks/free objects/waste
are created. Consolidate is consequently a storage no-op for direct blocks.
Zero allocate/free semantics and generational liveness checks remain unchanged.

`SmallBytesStats.SourceAllocationBytes` exposes the source m_alloc_size counter:
release mode matches requested live bytes; diagnostic mode stays0 because source
returns before the normal updates. `RequestedBytes`, `DirectBytes` and budget
storage measurements still track actual owned diagnostic bytes, avoiding a false
memory-use claim from the zero source diagnostic counter. Storage does not emit
source debug leak diagnostics; specifically the diagnostic zero counter produces
no leak message. Logical-object destruction remains the caller's responsibility,
and tracked-owner Reset/Close cleanup also applies to diagnostic direct blocks.
Cold owner header budget accounting still belongs to the caller and is an
explicit pending integration responsibility, not included in backing-array stats.

Tests first failed RED missing DirectOnly and SourceAllocationBytes fields, then
GREEN. Three new tests cover mixed small/boundary direct payloads and contents,
source zero counter versus measurable ownership, immediate release, no pooled
waste, nonmoving consolidate, reset/stale generation, close budget balance,
failure-before-publication with existing content preserved, and the unchanged
release counter. Ordinary fixtures require no reference source/toolchain runtime.

- Full native and g3checked CGO-disabled arena tests: PASS.
- go vet ./arena, js/wasm and wasip1/wasm builds: PASS.
- Actual Nodejs execution of current wasm binary printed all13 TestSmallBytes
  names including DirectOnlyDiagnostics, ReleaseSourceCounter and
  DirectOnlyFailureAtomic, each PASS. Compilation used normal workspace Go;
  minimal environment applied only to runtime wrapper, with TMPDIR in .cache.

This completes the finite concrete pooled/direct/diagnostic storage policy
translation with the documented owner cleanup and conceptual-address adaptations.
It does not establish raw-pointer ABI fidelity, AST-domain cleanup/integration,
source reference execution, real rational-cell constructed pools or the complete
arena/memory/solver gate.

# Arena cold-owner header budget integration

The pending arena header responsibility recorded above is now implemented inside
constructors. Source storage/cleanup algorithms and existing backing directory
charges are unchanged. `OwnerBytes` separately exposes cold owned bytes rather
than hiding them in record metadata or counting directory slice headers twice.

| Owner | Newly charged fixed allocation and unwind |
| --- | --- |
| Slab32/Slab64/private descriptorSlab | reflect.TypeOf(concreteOwner{}).Size after config validation, before owner allocation; Close refunds once |
| test recordSlab | same concrete rule, preserving rich-record state-machine/fuzz/zero-allocation test coverage |
| Words/Vectors64/Vectors32Signed | own header plus child descriptorSlab header; parent reserve failure closes child to refund constructor prefix |
| Region32/Region64 | own header plus separately allocated nonzero regionIdentity, in addition to nested slab header; reserve failure closes child |
| Pool32 | own header includes embedded mutex/config/policy; failed eager-worker construction refunds owner header; backing worker/page/free arrays remain separately charged |
| SmallBytes | own header plus child descriptorSlab; parent reserve failure closes child; flattened Stats.OwnerBytes includes both |

Stats.OwnerBytes is per owner; nested VectorStats.Descriptors.OwnerBytes and
RegionStats.Storage.OwnerBytes are added separately. RegionStats.OwnerBytes
includes its identity. Flattened SmallBytesStats includes private descriptor
header exactly once. Close sets owner stats to0 and refunds fixed ownership once;
Reset/Trim/Consolidate retain cold owner charges while still releasing eligible
backing arrays. This uses predictable typed logical byte accounting consistent
with existing FixedManager precedent, not a claim to measure Go runtime heap
rounding or caller-retained closed handles/tokens.

No owner separately charges embedded configs or per-page slice headers already
included in charged directory entries. Returned Pool32Worker/RegionMark values,
borrowed shared Budget, caller callback closure environments, domain cleanup and
scratch remain external ownership rather than per-owner duplicate charges.
FixedManager already charges only its own root (a Words pointer, no embedded
Words object), so that root charge remains correct. IntegerManager and its private
numericCellSlab headers are charged by mpnum after coordinator integration;
arena does not precharge them.
RationalManager's own new header charge likewise excludes referenced child owners.

Finite tests first failed RED for missing OwnerBytes, then GREEN. Across11
concrete owner variants, exact constructor charges (including eagerly owned pool
backing) are compared with independent reflect sizes; Reset retains fixed owner
ownership; repeated Close restores a preexisting external7-byte reservation.
Zero budgets, one-byte-short nested budget limits, every constructor reservation
fault prefix and invalid child configurations preserve external charges and
fully unwind failed construction. Existing backing-only peak tests now include
owner baselines intentionally; payload/free-list state-machine coverage remains.

Validation:

- Native and g3checked full arena tests, go vet: PASS.
- js/wasm and wasip1/wasm package builds: PASS.
- Actual Nodejs wasm execution printed all3 OwnerBudget test names and all11
  constructor/lifetime/fault subcases (invalid-config test9 subcases), each PASS.
- Native bitset dependency tests: PASS.
- Native mpnum dependency initially exposed new rational fault-sweep
  expectations requiring a failure even when warmed operations reserve nothing.
  The numeric owner corrected that distinction and added cold setter/inverse
  failure regressions. Full integrated release/checked races pass. No mpnum
  edits were made by the arena agent.

This closes the concrete arena cold-header accounting gap. It does not complete
all domain-owned roots/callback resources, the real rational constructed-pool
fixture, source raw ABI fidelity or the whole memory/solver gate.

# Words cross-budget ownership transfer prerequisite

Owned files: Words method/helper, `words_budget_swap_test.go`, this status append.
`Words.SwapBudgets(other *Words) error` exchanges the two complete owner graphs'
accounting destinations without moving records, changing refs/free lists or
running domain cleanup. This is a Go typed-owner budget adaptation needed by
UIntSetMap cross-budget source swap, not a new source allocator algorithm.

Closed owners reject before actions. Equal Budget pointers (includingnil and
self-swap) return without reservations. For distinct budgets, both graph sums are
checked for uint64 overflow before any Reserve; overflow is ErrBounds. Each sum
includes Words.OwnerBytes, payload, block metadata, payload directory and private
descriptor OwnerBytes/record bytes/generation metadata/directory. The borrowed
Budget object and caller/domain roots are not duplicated in the graph charge.

The original outgoing graph stays charged while its destination reserves the
entire incoming graph. First failure changes nothing. Second failure refunds the
first prepared Used charge and leaves both owner/config/payload states unchanged.
A successful Reserve attempt still consumes its ordinary FailAfter prefix even
when later refunded; no Budget transaction bypass or injector restoration exists.
After both reservations succeed, original graph charges are released and both
Words.config.Descriptors.Budget and descriptorSlab.config.Budget are rebound.
The caller must serialize all affected owners/budgets; an outer table transaction
can prepare its own distinct charges and refund them if this method fails.

Tests-first missing SwapBudgets RED then GREEN. A further explicit overflow
regression produced RED ErrInjectedFailure instead of ErrBounds; checked graph
addition now yields GREEN before reservations. Five tests cover first/second
failures, external baseline charges, nested owner totals, original values/ref
identity, growth routed to the new budget, Reset/Close after transfer, nil and
same-budget behavior, closed rejection, destination old+incoming peak limits,
overflow on either graph and exactly0 warmed Go allocations over paired swaps.

- Native and g3checked full arena tests, go vet: PASS.
- js/wasm and wasip1/wasm package builds: PASS.
- Actual Nodejs execution printed all5 TestWordsSwapBudgets names and both
  first/second failure subcases, each PASS. Compiler environment is the normal
  workspace environment; only Node runtime is minimal, with TMPDIR under .cache.

No map/table implementation, manifest or other agent files were modified. This
prerequisite alone does not complete UIntSetMap's table/epoch transaction or the
full memory/source ownership gate.
