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
- NewSlab audits T recursively. Pointers, strings, slices, maps, interfaces,
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
from a scalar T. Domain cleanup must precede truncation/free/reset/close.

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

Region[T] owns a Slab[T], a scalar RegionRef allocation log and scalar
serial/length mark records. RegionRef carries Slot/Generation; RegionMark is an
opaque cold owner identity pointer plus serial token. The mark's owner pointer
is not stored in hot record or mark backing arrays. Marks are never reused:
serial exhaustion permanently prevents new marks, even after Reset. Every
Allocate explicitly writes its initial scalar T. Reads copy values; no raw
pointer or slice borrow exists. Individual object free is deliberately absent.

Mark/Rollback are strictly nested. Rollback rejects zero, foreign, consumed and
non-top marks. PopScopes explicitly supports a validated counted pop. Reset
cleans all records, including base-level allocations, and invalidates all marks.
Close is idempotent and drops backing arrays and ledger. TrimFreePages returns
an error so callback reentrancy can be rejected; it drops only fully dead record
pages while retaining generation metadata and reusable scalar control arrays.

The transient Cleanup[T] parameter is never stored as a trail closure. It runs
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
