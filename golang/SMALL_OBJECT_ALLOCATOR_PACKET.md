# Standalone source small_object_allocator packet

Status: standalone concrete implementation frozen for parent review, with 22 independent test groups. Initial missing-API RED baseline fdf6f3c; final integration base e3a76a6. This phase is the real standalone allocator using the committed raw memory runtime; numeric/SAT/AST/Params typed-owner hooks and process-wide runtime integration remain subsequent coordinated work.

## Source inventory

Full files read: small_object_allocator.h/.cpp, machine.h, vector.h (pointer-vector lifetime/allocation), memory runtime API and source memory_manager counterpart. Source test is registered at src/test/main.cpp:166 and compiled at src/test/CMakeLists.txt:99. The test has ZERO ENSURE/SASSERT assertions: its allocation/free/TRACE sequence will be replayed exactly; independent checks are added separately, not invented source assertion counts.

- Constructor cpp:29–38 clears all chunks/free heads, stores borrowed debug ID, and zeros requested allocation count; it creates no source chunk.
- Destructor cpp:40–56 traverses classes ascending and each chunk chain head-first, frees chunks, then reports the exact debug leak message if m_alloc_size>0. Direct allocations are not found/traversed/freed.
- Reset cpp:58–71 traverses/frees chunks in the same order, clears heads, resets requested count. It does not free direct allocations.
- Deallocate cpp:77–97 size0 returns immediately; debug non-Windows calls raw deallocate without counter action. Other profiles assert counter>=size and nonnull (debug only), subtract size_t requested count, classify direct threshold, or write the old free head IN the object payload and publish the new free head.
- Allocate cpp:100–138 size0 returns null; debug non-Windows calls raw allocate without counter action. Other profiles add requested size BEFORE threshold/any allocation, take an intrusive free head first, otherwise round to class and test current chunk using strict new_curr<end. A new 8192-byte chunk is actually allocated, its curr field is constructed, next/head are published, then its first object/cursor returned.
- Getter h:48 returns requested count. cpp:140–162 wasted/free getters traverse actual intrusive links, not cached normalized totals.
- Consolidate cpp:172–242 uses lexical ptr_vector<chunk> then ptr_vector<char>; ascending classes; reset (capacity retained) per class; gather chunk chain, then intrusive free list; skip only AFTER both gathers if free count<floor(CHUNK_SIZE/obj_size); sort by source pointer comparator; run the exact object cursor loop (including free_obj>curr_end rather than lower-bound normalization); free a chunk only at equality with that floor; rebuild retained chunk chain and free links in sorted traversal; publish final heads only at class end. Reverse lexical vector destruction releases free_objs backing then chunks backing.
- Placement new and new[] h:54–55 both delegate allocate. Matching placement delete/delete[] h:56–57 are UNREACHABLE, not deallocation operations; explicit typed unreachable adapters.

## Concrete API proposal

Main-package private concrete types, no templates/interfaces/unsafe/cgo:

- utilSmallObjectProfile{PointerBits uint8; Debug,Windows bool}; default pointer width from borrowed runtime. Explicit source-profile selection permits LP64/ILP32 and debug Windows/non-Windows fixtures independent of Go host. Reject unsupported/mismatched pointer/runtime size_t profiles before header allocation.
- utilSmallObjectConfig includes Runtime, Profile, static ID/IDPresent/IDNull, BorrowedID *arena.BytesRegion, nullable scalar IDRef GlobalParamCString, Diagnostics/Verbose *NumericTextStream, and Verbosity. Runtime, Budget, streams, static spelling, and mutable ID region are borrowed. Constructor does not read the ID. Destructor reads its current C-string bytes only at the actual ID insertion, after the warning prefix and chunk frees. The private direct-byte emitter consumes width once for the whole C string, with no intermediate spelling allocation. Actual process stderr/global verbosity wiring remains separate.
- newUtilSmallObjectAllocator(config)(*utilSmallObjectAllocator,error), Allocate(uint64)(utilSmallObjectRef,error), Deallocate(uint64,utilSmallObjectRef)error, Reset()error, Close()error, AllocationSize()(uint64,error), WastedSize()(uint64,error), NumFreeObjects()(uint64,error), Consolidate()error.
- utilSmallObjectRef is pointer-free scalar provenance/slot/generation/logical-address. LogicalAddress() is a value accessor (source pointer value); Read/Write(ref,offset) validate live object identity. RawLocation(ref)(utilMemoryBlock,uint64,error) provides only scalar actual backing+offset for diagnostic/integration code, never an escaping slice. PlacementDelete/PlacementDeleteArray return the typed unreachable error.
- Stats() reports actual charged owner/location metadata separately from source requested count, active chunks, direct objects and pending completed raw roots. CleanupPending() explicitly cleans partial/unpublished raw allocations and failed lexical destructor roots. CleanupChunks() explicitly scans retained real live chunk locators after interrupted consolidation; both are Go continuations, not native exception-recovery claims.

## Real physical layout and source topology

PointerBits64: pointer8, alignment8, NUM_SLOTS32, CHUNK_SIZE8176, direct requested threshold248 (247 pooled /248 direct). PointerBits32: pointer4, alignment4, NUM_SLOTS64, CHUNK_SIZE8184, direct threshold252 (251 pooled /252 direct). Chunk runtime Allocate payload is EXACT8192, physically8192+SIZE_T including the real memory_runtime header. First source pointer-width bytes encode next location ID; second encode current byte offset in that chunk as the source curr pointer continuation; remaining actual bytes are object storage. Runtime Write marks semantic header bytes initialized. Source requested count wraps at selected size_t width. Fields are encoded little-endian as an explicit selected platform codec.

Cold charged location records contain only scalar actual runtime handles/offsets/class/request/lifetime/generation/address. The containing typed directory slice headers are cold; hot refs and directory records contain no Go pointers. Metadata capacity grows geometrically with old+new overlap fully reserved, actual reflect sizes; no per-record boxing or mapper closures. Intrusive free links encode the next location's scalar ID in the first selected pointer-width object bytes; no extra free-link array substitutes those writes. Free/reuse preserves all other physical bytes and source LIFO. Locator generations reject stale reads without changing the retained physical object address. Exhaustion is an explicit typed continuation at reuse; ledger/tombstones stay charged and safe across reset. Reset invalidates only pooled references whose chunks were physically freed; direct references and their raw allocation/generation stay live across Reset.

Source m_alloc_size increments before adapter metadata preparation; metadata refusal cannot be moved before it. On new chunk, prepare locators before calling real Runtime.Allocate, retain completed metadata capacity prefixes if later raw allocation fails, write/construct chunk curr then next then class head in actual source order. Runtime's nonzero token+error (TLS failure after physical/header publication or wrapped bad-header allocation) is retained as a pending real root with no fabricated constructed chunk/object. CleanupPending uses Deallocate for valid raw headers and CleanupPartial only for bad-header physical continuations; no counter rollback or invented callbacks.

## Consolidation scratch and ordering

Existing concrete vector policies are the algorithm reference, but unintegrated typed-owner backing hooks cannot impersonate source memory malloc calls. Therefore propose private concrete runtime-backed pointer-ID vectors for the two lexical scratch values: true payload8+capacity*pointer-width, actual vector header capacity/size, initialcap2, unsigned32 (3cap+1)>>1 growth, source overflow checks, Runtime.Reallocate for trivially copyable scalar pointer records preserving full capacity, push publication and reset retention exactly per vector.h:56–91. These are local values (no escaping heap owner); actual runtime backing and initialization metadata are charged. Cleanup reverse lexical order, including failures. No normalized one-shot sorting/gather shortcut.

std::sort's implementation is external; use concrete in-place heapsort of scalar IDs by deterministic conceptual address. Chunk creation IDs define disjoint ascending8192-byte virtual extents; object address is chunk base plus source header and offset. This specifies a reproducible pointer-order adaptation, NEVER actual host/C++ raw-address equivalence. Preserve the source scan/relink algorithms literally, including absent lower-bound check and equality-only full-chunk test. No mutable address reuse can alias a retired live chunk.

Strict endpoint example: LP64 size1 occupies8 bytes; first chunk accepts1021 objects, although consolidation divides8176/8 to1022. All1021 free does not free that chunk. With remainder classes (e.g. width24), the number actually placed equals floor and complete free can reclaim. Both outcomes require regressions; do not repair the quirk.

## Direct lifetime and diagnostic decisions requiring review

Source Reset/Close leave direct blocks untouched. Proposed precise contract: returned direct object is caller-owned; RawLocation exposes its actual runtime token before allocator teardown. Reset frees pooled chunks and invalidates only their locators, clears counter/heads, but retains live direct locators and all necessary charged directory storage. Direct bytes remain readable and Deallocate(originaldirect) still executes the source profile actions: debug non-Windows raw-free with counter0; release size_t subtraction wraps from0 before raw-free; debug Windows asserts count>=size before subtraction/free. Close invalidates allocator context and releases its location graph/pooled chunks, but DOES NOT call Runtime.Deallocate on outstanding direct blocks. The borrowed runtime still owns/tracks those physical roots; callers retain scalar raw tokens and explicitly deallocate them later. Tests prove original direct tokens survive Reset, all three subsequent free branches and their exact counters/raw roots, plus live raw storage after Close and explicit caller cleanup. No leak reproduction disguised as complete ownership: lifetime obligation is documented and observable; no automatic free events invented.

Debug non-Windows bypass uses direct raw storage for every nonzero size, with requested count ALWAYS0 and no source leak message. Debug Windows uses pooled policies/counter/assertions. Destructor diagnostics (after chunk frees) use actual borrowed NumericTextStream state and C-string ID, preserving exact source text; nil/error sink guards occur only at actual output action. Actual global stderr and verbosity stream binding are OPEN. Consolidate optional verbosity20 messages require a supplied real output stream and runtime AllocationSize getter; disabled verbosity executes no fake collector or formatting. Actual global stdout/stderr/verbosity binding and persistent ostream badbit/flush behavior remain OPEN; the supplied byte streams preserve selected finite output and failure prefixes only.

Wrong-size deallocation, duplicate free, and invalid pointer paths need source-domain classification at their actual endangered action, not broad preemptive guards: debug source counter/nonnull assertions precede subtraction; release size_t underflow itself is defined and must wrap. Size0 ignores the pointer. Valid same-size ownership is the primary tested domain. Intrusive wrong-class relink can be finite before later overlap/OOB; do not claim a universal source matching-size assertion which is absent. Direct deallocation of an interior pooled pointer is guarded at the raw header action after the counter prefix.

## Tests-first scope and gates

New tests first: exact registered print-only replay; complete branch/profile boundary table; intrusive bytes/LIFO/data retention; strict cursor and consolidation quirk vs reclaimable remainder class; m_alloc_size before raw source allocation refusal with immediate real runtime counter/header/physical checks; lexical scratch allocation order/reuse/exception cleanup; direct Reset/Close lifetime; pointer-free/copy guards and generation retention; independent bounded size/free model and warmed physical plateau/zero host allocation. Constructor/header/first locator refusal and pending TLS/wrapped raw root fixtures are now included. A bounded 12-prefix cold reservation sweep, reset/generation retirement fixtures, real late-consolidation free failure, and lexical before-free failure now inspect immediate roots and exact cleanup.

Actual missing-API RED was observed in isolated git archive fdf6f3c plus ONLY the new test. Ordinary tests require no reference compiler, network, or source tree. Final native/checked-debug/race/actualJS/vet/WASI/no-generics gates use e3a76a6 plus only the five frozen Go files. Full numeric allocator topology, typed-owner hooks, generated memory lifecycle and full solver remain OPEN.

## Source SHA256

- `src/util/small_object_allocator.h`: `57b0a3bbc2d9fd10f008f371e3b5b9bee5535d39ab5cfbcb4afd9324018a140b`
- `src/util/small_object_allocator.cpp`: `70cf910ed0a3871ea110ee01ff8d23a687f74af5972ab995dce32ba3e6e35994`
- `src/test/small_object_allocator.cpp`: `98d72b253f1b8bda2ef5e6c9ad460064ceaee571ae15a7016b37cbf55ea79618`
- `src/util/machine.h`: `dba7365724c1167ee4bced664de38169c02624eedd6059ebb0d3d66213e64abc`
- `src/util/vector.h`: `aafafc0fe6ca0afd7bafe0c781342da259f9c34c0b94b16732c415cf2733066e`
- `src/util/memory_manager.cpp`: `3b884edd5463aa7ddfe68d6e0badc0695eff2b6b6fd90736850dd740f43e29d6`

## Initial test inventory and observed RED

The initial thirteen test groups existed in `util_small_object_allocator_test.go`. They use the actual committed raw runtime, before allocator implementation, with no replacement counters. Registered print-only replay includes all source actions; source assertion count remains zero. Actual missing-API RED was observed in the isolated `.cache/release/small-object-allocator-red-fdf6f3c` tree on fdf6f3c plus only this new test. Log `.cache/small-object-allocator-missing-api-red.log`. Later production regressions are recorded below; initial RED is API absence, not evidence that any existing production algorithm failed.

Parent review correction before production: the source Reset never frees direct pointers, so the initial proposal to invalidate them was rejected. The updated design retains their locator/generation and tests the defined release counter underflow, non-Windows debug raw-free, and Windows debug assertion at its actual position. No allocator production existed when this correction was made.

The initial pending-root fixtures include both bounded SIZE_T overflow allocations (requestedUINTMAX → real3-byte ILP32 /7-byte LP64 backing, guarded at actual raw header write) and TLS postphysical limit failure (100000+8 bytes, cumulative source count remains1 through initial sync and2 after explicit free sync). These assert immediate actual runtime counters/physical ownership, never Budget as a substitute. Constructor and first locator refusal distinguish actual cold charges from source raw calls.


## Final executable scope and continuation boundaries

All declared source allocator methods and placement allocation/deletion helpers have corresponding concrete bodies: constructor, Allocate/Deallocate, Reset/Close, requested/free/waste getters, Consolidate, placement Allocate routes and typed unreachable delete routes. This does not claim integrated numeral storage, generated process lifecycle, global stream binding, or the full solver. Physical roots use the genuine runtime allocator and its SIZE_T headers/counters; cold owner/location metadata is separately physically budgeted, never relabeled source malloc calls.

The free-list link is physically written into the pooled payload, not a side-array link. LP64/ILP32 release and debug-Windows use actual source chunk policy, while debug-non-Windows takes the direct-only source branch. Same-generation duplicate pooled free preserves subtraction and intrusive self-link; debug-Windows permits it when another outstanding allocation leaves enough counter for the second source assertion. Ordinary readers remain live-only; a reused stale generation or physically freed direct token is rejected. The later cycle traversal returns a typed nontermination continuation, after its finite traversal prefix.

A real late TLS raw-free error can occur after physical free but before class-head publication. The charged cleanupChunksNeeded flag is set before the dangerous free. On error, original source heads stay unchanged and actual remaining chunk roots stay in the location directory. Explicit CleanupChunks performs actual runtime frees once per still-live chunk, without stale source-head traversal or normalized head publication at failure. Reset/Close reject that interrupted traversal until explicit cleanup. Direct objects remain caller-owned.

Two charged scalar pendingScratch handles preserve the reverse-lexical free_objs/chunks roots when ILP32 signed counter subtraction fails before physical free. CleanupPending performs real per-root frees; it does not fabricate counter rollback. A headerless wrapped block uses runtime CleanupPartial, which releases actual physical storage but intentionally retains its source accounting prefix because no readable source SIZE_T header exists. A source-unpublished TLS allocation with a valid header uses actual Deallocate instead.

Close separates completed source destructor actions from Go owner finalization through a charged destructorDone flag. After chunk frees and one diagnostic attempt, pending roots retain the complete charged owner/location graph until CleanupPending and final Close. A retry does not repeat diagnostics or completed frees. No new operations are allowed after destructor completion. This is explicit Go cleanup of partial source states, not a claim that C++ resumes after invalid memory or exceptions.

## Observed test-first evidence

- Initial API absence: `.cache/small-object-allocator-missing-api-red.log`, isolated fdf6f3c plus test only.
- Initial diagnostic API absence: `.cache/small-object-allocator-stream-missing-api-red.log`.
- First implementation gate 9317 found a real directory-reservation underflow: growth had used len+missing instead of cap+missing; corrected before GREEN. The tool output records the budget-release panic. That early shared log was superseded by final native output, so it is not claimed as a preserved standalone log.
- Late consolidation cleanup RED 60288: `.cache/small-object-allocator-late-free-red.log`, actual postphysical TLS signed-count synchronization failure, surviving roots checked before Close hit invalid old traversal.
- Duplicate pooled free RED 11263: `.cache/small-object-allocator-duplicate-red.log`, source-defined second relink incorrectly rejected by live-only free guard.
- Lexical before-free RED 68195: `.cache/small-object-allocator-lexical-free-red.log`, real ILP32 signed extent transport refusal, two live scratch roots absent from allocator ownership graph.
- Pending owner finalization RED 39571: `.cache/small-object-allocator-close-pending-red.log`, both partial and scratch Close prematurely released charged owner metadata.

The final 22 named groups cover the exact registered print-only actions (zero source ENSURE), six source profiles and thresholds, actual headers, intrusive LIFO/preserved bytes, strict cursor and exact-divisor consolidation quirk, reclaimable remainder classes, source requested increment before raw refusal, lexical scratch growth/reuse, direct Reset/Close lifetime, independent 4000-operation oracle and warmed zero-allocation plateau, scalar type audit, placement deletes, constructor/cold refusal, wrapped/TLS completed roots, mutable/NUL/null/stale ID reads, diagnostic failure prefix, real runtime verbosity/state/lexical lifetime, late free roots, duplicate free, cold reservation sweep/reset ledger, lexical failed frees, and generation exhaustion. No invented C++ assertion counts.

## Frozen Go SHA256

- `util_small_object_allocator.go`: `0e120e405b0f70612100ebee66f3c68d7bd80e6deaddbe0928706b71889b6de9`
- `util_small_object_allocator_consolidate.go`: `7fd77cf2082f4d202722c2994ed170d578b917dd55dbbc4a569663c21fd1548e`
- `util_small_object_allocator_stream.go`: `1433d5f10a5cb634fcc148e3b91b55a9eae97dccfd831f70660f6ba5ef786156`
- `util_small_object_allocator_test.go`: `55e968645d5024ab2ab939eaf5ef604200abb09604c0d7364a8295a9cb5510e0`
- `util_small_object_allocator_vectors.go`: `c48e90e69f9bd9a46ccfda9692b495ac025f4db6e86ced108d339cf129fbd3a2`

The snapshot is `.cache/release/small-object-allocator-green-e3a76a6`, committed e3a76a6 plus only these five files. Production and tests are frozen during final gates; terminal evidence will be appended after completion.

Additional parent-reviewed post-destructor assertions were added to the existing lexical cleanup group. They pass on unchanged production: ordinary Allocate/getter calls are rejected both before and after pending cleanup until final Close, without physical or budget mutations. This is immediate GREEN coverage, not fabricated RED. The final test-only hash supersedes the earlier 7af3ccf hash; all four production hashes remain unchanged.


## Final terminal execution evidence

All gates use the isolated e3a76a6 plus five Go files snapshot; no unrelated drafts are included. The final test-only post-destructor addition changes no production. Parent will repeat final full module integration on that final test hash.

- Full native module gate48927 and full checked-debug module66473 exited0 on the identical four production files and preceding test hash. mpnum completed20.885s/22.194s. These are historical full-module evidence because the final additional test assertions were added afterwards.
- Final22 named native32857 and checked-debug87647 exited0,0.031s/0.035s.
- Final22 checked-debug race79262 exited0,1.426s.
- Final22 actual checked-debug JavaScript14521 exited0,0.269s using the minimal Node wrapper. RUN and PASS were verified for every22 declaration in both race and JavaScript logs, not merely package summaries.
- Vet72445 and WASI compile73533 exited0 with final test hash.
- Logs: `.cache/small-object-allocator-final-{native,debug,race,js}.log`, `small-object-allocator-full-{native,debug}.log`, `small-object-allocator-vet.log`, `small-object-allocator-wasi.log`.
- Exact test-name inventory: `.cache/small-object-allocator-test-inventory.json`. The snapshot and live five Go hashes match byte-for-byte.

All build caches/temp/output remain under golang/.cache. No source files, existing production files, central metadata, or Git state were edited by this packet.

Source/evidence/no-generics audit55849 exited0 using the relative reference root ../../../..: 2102 units,677 APIs,33 ported/145 partial/1924 unported. This audits the committed metadata inventory and frozen tree; it does not promote allocator source rows before parent integration. Log `.cache/small-object-allocator-audit.log`.


## Parent final integration verification

Parent independently froze e3a76a6 plus the final five Go files and three
metadata/document files in .cache/review/small-object-final2-e3a76a6. Full native
67452 and checked/debug81684 exited0 on the final test hash (mpnum21.231s and
22.040s). Named checked/debug race28978 and actual JavaScript81622 exited0;
all22 RUN and PASS names exactly match the declaration inventory. Vet96573,
full WASI build60396 and root WASI test compilation38371 exited0. Final
source-evidence/no-generics audit17827 exited0 with the integrated manifest:
2102units/677APIs,33ported/145partial/1924unported. The preceding relative-source
audit26012 exited0 and an independent check matched all2102 recorded source/tool
SHA256 values. Final production bytes were unchanged through these gates.
Parent logs and inventories are .cache/small-object-parent-final-*.log and
.cache/small-object-parent-final-inventory.json. No full solver completion is
claimed; standalone source units retain partial status and integration remains
required.
