# Concrete int32 persistent-array prerequisite

`util_parray.go` translates `src/util/parray.h` for both registered int32
configurations in `src/test/parray.cpp`. It uses the real ROOT/SET/PUSH_BACK/
POP_BACK cells, root-preserving inverse trails, materialization and reroot
algorithms. It does not replace persistence with copied Go slices. There are
no Go generics, production maps, unsafe casts, boxed values or per-cell Go
objects. Cells and value blocks use the borrowed `arena.SmallBytes` allocator.

## Exact source assertion and specialization inventory

| Source test, each preserve-roots policy | ENSURE sites | Executions |
| --- | --- | --- |
| tst1 | 15 | 15 |
| tst2 | 4 | 213: two standalone assertions plus loops100 and111 |
| tst3 | 14 | 128 |
| both true and false policies | 66 instantiated sites | 712 |

The source fixture ports every active ENSURE and all operations before/between/
after them. Tst3's guarded Get calls preserve the actual short-circuit routes.
The initial read-only arithmetic count710 was incorrect: tst2 is1+1+100+111=
213. The executable fixture independently counts712 assertions and is named
`TestPArraySource712Assertions`; no assertion is removed to meet the earlier
mistaken total.

Both int32 policies suppress value callbacks, preserve roots true/false,
max-trail8 and factor2. CELL reference counting still occurs in both policies.
Actual AST expression and dependency arrays use real retain/release callbacks,
preserve-roots true, max-trail16 and factor2. Subpaving bound arrays suppress
value callbacks but use preserve-roots true/max-trail16/factor2. All these
non-int32 specializations remain unported; no fake callback service stands in
for them.

Registered parray tst5 has zero ENSURE sites but genuine AST initialization,
100 distinct variables, Copy, exactly1,000,000 Set updates, two process memory
high-water prints and explicit Del. It remains OPEN until the actual AST manager,
expression callbacks and resource/output adaptation exist. Integer plateau
checks do not claim that source integration route. Disabled `#if0` tst4,
commented tst4 registration, trace enabling, allocation diagnostics and its old
AST max-trail8 configuration remain distinguished from current AST code.
Registered ast.cpp tst5's seven persistent-expression assertions likewise remain
pending the real AST owner.

## Source method map

| Source method/action | Concrete counterpart |
| --- | --- |
| constructor; borrowed value manager/allocator | `NewInt32PArrayManager`, no int32 value callbacks; explicit allocator borrow |
| public `manager()` returning dummy_value_manager<int>& | explicit Go configuration adapter: int32 callback suppression is fixed by the concrete type; no shared mutable dummy-manager object/accessor is exported |
| private cell ctor/mk/del/inc/dec | encoded RC/kind cells, `mkCell`/`freeCell`/`inc`/`dec`; iterative source chain deletion |
| capacity/allocate_values/deallocate_values/expand | source-width capacity header; full encoded int32 capacity; exact0→2→3→5→8... growth |
| root rset/rpush_back/rpop_back | Set/Push/Pop root branches; retained popped slots; value callbacks suppressed |
| copy_values/get_values | `getValues` root prefix copy at same capacity, source forward replay, retained path scratch |
| ref default/root/unshared | zero `Int32PArrayRef`; checked private cell kind/RC |
| mk/del/copy | `Mk`/`Del`/`Copy`; explicit manual token ownership |
| size/empty/get | `Size`/`Empty`/mutable-pointer `Get`, strict trail threshold |
| set and source-output overload | `Set`/`SetFrom` |
| push_back and source-output overload | `PushBack`/`PushBackFrom` |
| pop_back and source-output overload | `PopBack`/`PopBackFrom` |
| unshare/unfold/reroot | `Unshare`/`Unfold`/`Reroot`, including finite source PUSH inverse defect |
| display_info | `DisplayInfo`, scalar identity display adapter and charged lexical formatter |
| manager destructor implicit scratch cleanup | `Close` owner/scratch refund with explicit no-live-cell contract |

Source shallow ref copies do not retain; ordinary Go record copies are borrowed
aliases. Use Copy/Del to acquire/release ownership. Source has no whole-manager
Clone/Swap/Reset/Finalize API. `Stats` and typed errors are Go ownership adapters.
The source public dummy-manager accessor is adapted away, rather than claimed
as an implemented mutable callback object. Its int32 inc/dec methods do nothing
and C::ref_count=false suppresses their use; the concrete configuration retains
that behavior. No real AST array/value-manager behavior is claimed by the int32
implementation.

## Storage, lifetime and physical budget

The cold manager owns its actual reflected header and two retained scalar
scratch vectors; scratch grows by source vector2→3→5... with old/new overlap
charged. Its allocator is borrowed, must outlive the manager, and must use the
same Budget. This is an explicit caller contract because the existing allocator
has no public Budget identity accessor; no shared API was altered to add one.

Each cell has a fixed explicit24-byte encoding: packed lower30-bit RC/upper2-bit
kind at0, idx/size at4, int32 element at8, padding at12, scalar link descriptor
slot/generation at16/20. The descriptor represents a cell link or ROOT value
block according to kind. No hot pointer is stored. Public references also carry
scalar manager identity and a per-reference update counter. The counter is not
shared in the cell. Reshaping roots/trails retains their descriptor generations;
there is no global epoch invalidation of persistent versions.

The value block begins with target-pointer-width size_t capacity metadata,
followed by the WHOLE int32 capacity. Logical ROOT size is separate. This
capacity header is genuinely allocated even when capacity0 during copying an
empty shared root. Source expand copies every old capacity element and performs
no value callbacks; root copy allocates the same capacity but copies only the
logical prefix. Unused newly allocated storage is not exposed as initialized
source data. Numeric byte encoding is explicit little-endian; these blocks are
not a reinterpretation of C++ raw memory.

Actual cell bytes, descriptor records/generations/free metadata, value header+
payload, pool chunks/direct blocks/directories, manager header and scratch
capacities are charged by their real owners. Source C++ cell pointer layout
is not claimed equal to the fixed checked encoding. Source large size_t arrays
beyond the allocator's uint32 byte domain return typed ErrBounds before
allocation. Tests inspect complete Budget sums using allocator owner, payload,
metadata and directory bytes, avoiding double-counting DirectBytes (already
included in PayloadBytes).

Close requires zero live cells and returns ErrPArrayLive without changing owner
state otherwise. It never closes the borrowed allocator. After explicit Del,
Close refunds only the manager header/scratch. Pool cleanup remains with the
allocator owner. Null cells are source-safe no-ops; null refs have size0 and are
empty; Push initializes them. Invalid null Get/Set/Pop/Unshare/Unfold/Reroot and
foreign/stale references receive typed adapters for source invalid accesses.

## Source actions and known finite defect

All cells start RC1. Increment wraps to30bits without destroying on zero;
dec-zero returns the source-assertion error adapter. Normal deletion processes
SET/PUSH's stored value (suppressed for int32), ROOT logical values ascending
(suppressed) and its block, then deallocates the cell, decrements next RC and
continues only on zero. Future real callback specializations remain required.

Copy retains source before releasing target and resets that caller's update
counter, including self-Copy. From overloads detect src==out and directly
mutate, skipping Copy. Get takes a pointer because it can reroot and reset that
particular reference. Both trail and root update thresholds use strict greater
than, not greater-or-equal.

Shared root mutation increments its caller update counter before allocating
its new root. The old block transfers without copying into a new root RC2;
the older root becomes the inverse operation; its former caller reference is
released; the caller is published to the new root before the final requested
Set/Push/Pop action. Nonroot and non-preserving shared-root updates create
forward trail cells and transfer the existing reference into next without an
extra retain/release. Materialization replays oldest-to-newest; Unfold releases
old next before becoming ROOT; Unshare creates a new root and releases the
old reference. Reroot reverses the collected trail with actual source RC
publication/destruction order, including reading updated predecessor RC after
former-root destruction.

**Source finite defect preserved:** Reroot's PUSH_BACK reversal sets old-root
kind=POP_BACK and idx=sz BEFORE incrementing sz, although Size(POP) returns
idx-1. An older mathematical size1 becomes source size0; an older size2 becomes
source size1. Exact fixtures cover forward PUSH with preserve-roots false and
reverse PUSH produced by Pop with preserve-roots true. These regressions
explicitly state mathematical versus actual source sizes. The implementation
does not silently repair this source defect.

The independent mathematical version-array oracle uses materialization before
inspection to avoid this defective PUSH reroot route. Separate unaffected SET
reroot fixtures run both policies, POP reroot is checked independently, and
all source registered Get/Reroot routes remain present. This does not exclude
false-policy coverage or treat all reroot behavior as correct by assumption.

## Predictable failure prefixes and explicit unsafe-continuation adapters

Defined prefixes are retained: update counter increments before cell allocation;
From performs Copy before its operation; first Push can publish an empty ROOT
before value allocation fails; shared Push can publish inverse trail/new ROOT
before payload growth fails; scratch preserves successful physical growth
prefixes. There is no fake whole-operation rollback guarantee.

Two unsafe source exception continuations are explicitly adapted. Mk releases
its prior reference before replacement allocation; Go clears the destroyed
output, rather than leaving a dangling descriptor if allocation fails. PUSH
reroot changes the former ROOT tag before expanding its full payload; source
failure can leave its values pointer interpreted as a cell link. Go preflights
that expansion and publishes the valid replacement block before destructive
retagging. Successful logical source action order and finite inverse-index
defect are retained. Unpublished int32 materialization cells/blocks receive
explicit lexical failure cleanup; no source exception-safety guarantee is
invented for these raw source temporaries.

DisplayInfo uses slot/generation identity spelling instead of machine addresses.
Its lexical bytes are budgeted with overlap on growth; only final caller output
is an uncharged string. Predictable format failure returns the already emitted
prefix and refunds the lexical buffer. Nil Go reference parameters are typed
errors; source C++ reference parameters cannot represent that call shape.

## Tests and validation

Fifteen groups cover all712 source assertions, independent bounded version
oracles, both policies, source RC30 wrap, forward/inverse trail metadata,
strict trail/update thresholds, retained popped slots, zero-capacity header,
self-Copy versus self-From, foreign/stale/null refs, borrowed Close lifetime,
known PUSH defect, unaffected SET/POP reroot, plateau/zero warm Go allocations,
source publication prefixes, Mk clearing, formatter prefix and bounds. The
formatter's checked helper tests exact MaxHostInt, addition and doubling overflow
for32/64-bit hosts without huge allocations and imposes no uint32 output limit.

Failure matrices independently compute reachable unique cells/ROOT blocks and
expected cell RC from caller references plus graph edges. They compare these
with live allocation counts and complete physical Budget sums BEFORE any
Get/String/Del/Close cleanup can hide a temporary leak. Unshare/Unfold/Reroot
and shared Push exercise all bounded allocation prefixes; both successful and
failed shapes are checked. Initial isolated tests produced missing-API RED;
source assertions and independent fixtures drove the implementation. Format
prefix and nil-reference regressions separately produced behavioral RED then
GREEN. The earlier arithmetic count correction is not an implementation RED.

Final isolated HEAD `6c30500` plus only the two new Go files passed full
CGO-disabled native and checked+debug suites, checked+debug targeted race,
all fifteen named RUN/PASS groups on actual checked+debug JS/wasm,
checked+debug vet, WASI build and the no-generics audit after the host-bounds
correction. No WASI runtime
execution is claimed. Automatic source SASSERT/TRACE diagnostic execution and
machine-address rendering are distinguished from typed validation/output
adapters; arbitrary template configurations and non-int32 value callbacks remain
OPEN. Release concrete int32 source algorithms are implemented with the public
dummy-manager accessor explicitly adapted above, not the whole
parray template or its AST/subpaving integrations.

## Coordinator integration

Frozen `.cache/release/parray-sort-size-integrated` contains the final two
persistent-array files and two SortSize files atop HEAD6c30500. Full fresh
CGO-disabled native and checked-debug modules passed (mpnum9.803s/9.833s).
All19 named checked-debug race and actual JS RUN/PASS groups were independently
verified; checked-debug vet and WASI package builds passed. Frozen Go hashes
were matched before commit. Logs: `.cache/parray-sort-size-integrated-{race,js}.log`.
WASI build is not runtime evidence.
