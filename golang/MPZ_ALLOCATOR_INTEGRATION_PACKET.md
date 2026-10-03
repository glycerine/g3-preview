# Actual MPZ allocator/backend integration proposal

Phase B design and new missing-API tests only, following shared memory phase A commit6ffd717. No memory/mpnum/arena production, existing tests, metadata or Git changes are authorized in this phase. The objective is to replace the MPZ cell backing actually used by the numerical algorithms, not to install an unused second allocator or reinterpret Budget reservations as source malloc calls. Full solver/runtime integration remains required.

## Audited source and current boundaries

Read allocation/layout/lifetime bodies in src/util/mpz.h:46–184, cpp:86–245,247–280,306–354,1185–1216,1555–1595; arithmetic scratch consumers and source mpn.h/.cpp/buffer.h allocation bodies; memory_manager.cpp global initialization/getters and both TLS/non-TLS allocation branches; committed standalone allocator bodies. Current IntegerManager/mpz_slab, limbView/NaturalManager/format storage, RationalManager, RationalService, and BinaryRationalManager ownership were inspected. Existing concurrent power/publication drafts remain parent-owned; integration must be rebased on their final committed results rather than silently restore an older action order.

Current Go cells are numericCellSlab records containing a Words descriptor, size, and capacity. Allocate acquires a logical slab record and a separate geometric Words payload; free releases both. This is an audited storage adaptation, but it is not the source small-object-allocator topology. NaturalManager also obtains four Words allocations for source inline buffers. SourceCounters cannot be repaired by adding events to either operation: neither existing slab/page allocation equals a source cell allocation, and a per-cell malloc counter would count pooled objects incorrectly.

The source manager owns one small_object_allocator, one mpn_manager, two temporary and two argument cell pointers, intMin and two64. MPQ inherits that same MPZ manager (mpq.h:43), so numerator and denominator must share one allocator. Binary rational managers borrow their MPZ manager (mpbq.h); RationalService owns a RationalManager and synchronizes its operations. No allocator per rational, coefficient, wrapper, or cached value is appropriate.

## Selected source profile and exact cell layout

The current arithmetic is explicitly _MP_INTERNAL with unsigned32 digits. This is not the default Linux GMP backend: mpz.h:46–53 selects GMP when neither internal nor MSBIGNUM is specified on non-Windows. GMP's mpz_t and external GMP limb allocations, and MSBIGNUM's size_t digits, are separate OPEN source profiles; no cgo, unsafe, generic, or GMP substitute is proposed.

For the selected internal profile, sizeof(mpz_cell)=8: size at byte0, capacity at byte4, and little-endian uint32 digits at byte8+4*i. Actual cell allocation is one object with those bytes. The selected pointer width changes allocator rounding and runtime SIZE_T prefix, not digit width or these header offsets. capacity6 requests32 bytes in both profiles. allocate writes capacity immediately after allocator.allocate; m_size is not initialized by that helper. Argument construction then writes size1. New/free-list storage must not be silently zeroed or presented as source-initialized digits. Freelist pointer writes really overwrite the first4/8 object bytes; LP64 therefore overwrites both source header fields. Reallocation of that free object rewrites capacity, not all digits or size.

Important width: cell_size returns unsigned, even on LP64 (mpz.h:122). Compute the source expression then narrow to uint32: uint32(8+4*uint64(capacity)), not an unrestricted uint64 cell extent. Large capacities can wrap to a finite small allocation. Preserve that allocation/capacity publication and guard only the actual endangered header/digit access. A global capacity rejection would erase source prefixes. Any source capacity assertion remains debug-only at its actual site.

Boundary fixtures: LP64 capacity59 =>244-byte pooled object, capacity60 =>248-byte direct object. ILP32 capacity60 =>248-byte pooled object, capacity61 =>252-byte direct object. Debug non-Windows directs every nonzero cell; debug Windows pools. Source initial capacity is6 for uint32 digits; the source64-digit capacity4 branch remains OPEN.

## Actual shared package boundary and phase B codecs

Phase A now provides the sole real memory.Runtime/accounting/SmallAllocator implementation, with main error and real byte-stream bridges. RuntimeConfig carries Budget, SizeTBits32/64, ThreadLocal and one static borrowed ErrorTransport. SmallAllocatorConfig carries the actual Runtime, PointerBits/Debug/Windows profile, one cold borrowed AllocatorIO receiver and Verbosity. Main exported source errors alias lower identities where applicable. Block and ObjectRef are scalar owner/slot/generation locators; ObjectRef additionally retains the deterministic source-pointer-order address adapter. RuntimeOwner() on the allocator returns the actual borrowed runtime after Close. Phase A preserved all41 original groups plus12 new groups; source process-global/TLS and generated initialization remain honest separate dependencies.

Phase B proposes exactly these missing concrete codecs, using actual byte initialization state and no slice borrows:

```go
func (*memory.Runtime) ReadUint32(ref memory.Block, offset uint64) (uint32, error)
func (*memory.Runtime) WriteUint32(ref memory.Block, offset uint64, value uint32) error
func (*memory.SmallAllocator) ReadUint32(ref memory.ObjectRef, offset uint64) (uint32, error)
func (*memory.SmallAllocator) WriteUint32(ref memory.ObjectRef, offset uint64, value uint32) error
```

These are selected little-endian internal digit/header adapters, independent of host unsafe casts. Validate genuine owner/generation, then the actual endangered four-byte range using offset<=extent and extent-offset>=4 without offset+4 overflow; check initialized bytes for reads. A word write validates its physical range at the word access before publishing its four initialized bytes. It does not reject an earlier finite wrapped cell allocation or header publication elsewhere. Misaligned byte offsets remain representable by explicit codec; MPZ offsets0/4/8+4*i are aligned within their source objects. Codec operations never reserve storage or emit source allocate/free/count events. Allocator codecs resolve actual ObjectRef provenance/lifetime and use the same Runtime backing, never a second payload.

Failure transport remains static/borrowed; direct lower default errors preserve101/113 and no-code vector overflow. The selected allocator diagnostic receiver may be caller-provided but must outlive the manager. No production main-package import is introduced into mpnum. Native process stderr/verbose/stdout binding and source SYNCH locking remain separately audited adapters, not fake sinks.

## MPZ record and physical access replacement

Retain Int's current release uint32/check-mode uint64 numericCellRef layout and owner-local implicit-owner contract. Use a charged numeric descriptor ledger mapping that ID/generation to one memory.ObjectRef. The record is pointer-free. Keep the existing permanently dead failedRef as charged auxiliary identity metadata with no source cell allocation or counter event. Source size/capacity and digits are stored in the actual allocator object. A cached scalar view may exist only if its initialization and writes are kept explicitly synchronized with those bytes; it cannot retain a second digit array or bypass the physical header. Prefer separate capacity/size accessors so reading capacity does not invent a read of an uninitialized size field.

Proposed IntegerConfig adds Memory *memory.Runtime and AllocatorProfile memory.SmallAllocatorProfile plus AllocatorIO memory.AllocatorIO/Verbosity if genuine diagnostics are enabled. Cells.Budget and Words.Descriptors.Budget must still match; when Memory is supplied its Budget must be that exact pointer, Memory must be live, and profile PointerBits defaults to Memory.SizeTBits with exact equality validation. Every constructor uses the real SmallAllocator backend: no opt-in branch continuing old Words digit cells.

When Memory is nil, existing NewIntegerManager remains the owning standalone API: it constructs one actual Runtime with the same Budget, selected default64/nonTLS profile and default lower error transport. This is an explicitly isolated runtime visible through RuntimeOwner(), not a claim of process-global memory fidelity. When Memory is supplied, it is borrowed and is never closed by the manager. Full solver/generated numerical cohorts must pass the real shared domain; unrelated polynomial coefficient owners cannot clone/finalize it. The default constructor always owns the source allocator and Natural manager regardless of runtime borrowing. Constructor unwind finalizes its completed real roots then closes only an owned isolated Runtime, retaining partial roots/charges where actual cleanup refusal prevents release.

The cold descriptor ledger is Go safety/ownership metadata, not a C++ malloc event. A descriptor-growth refusal must not preempt the source allocator's requested-size increment. Proposed preparation order: actual SmallAllocator.Allocate(cell_size), actual capacity header write, then prepare/publish the numeric descriptor. If late descriptor preparation fails, retain the completed ObjectRef in a charged manager pending-cell field; publish no numeric handle, return the error, and provide explicit cleanup through the real allocator. This is an additional Go metadata boundary inside the helper, preserving completed physical/header/counter prefixes. Source allocator pending raw roots remain owned by that allocator, distinct from a completed cell awaiting a numeric descriptor.

Replace IntegerManager.allocate/free, cell/store, digit/writeDigit and resizeCell; remove its MPZ Words payload ownership. Keep Words only for still-untranslated non-cell storage, with its incomplete source allocation mapping explicit. Modify transient limbView to have a concrete MPZ-object view alongside existing external/Words views, and route MPZ division and formatting input/output views to actual cell bytes. No persistent hot view contains a Go owner pointer. MPZ calls do not marshal through host slices or decimal strings.

Every allocator free derives cell_size from the live physical capacity before intrusive overwrite. Invalidate/recycle the numeric descriptor only after the real source release action's appropriate completion. An actual before-free/after-free runtime transport failure needs charged release-phase metadata and explicit cleanup, not ordinary retry that subtracts requested bytes again. Close cannot forget either completed unpublished cells or actual live raw roots.

## Source action map that must remain intact

- Constructor cpp:86–140: allocator constructed first; mpn member already constructed; tmp0, arg0(size1), tmp1, arg1(size1), intMin, two64 construction and Add in original order. There are six managed internal cells, not a seventh temporary replacement when two64 is already big. Release constructor's small cells share one real32-byte class chunk; source raw count is1 and raw extent8192+pointerWidth. Debug non-Windows produces six real32+pointerWidth blocks. After the constructor Add, two64 uses the original tmp0 object and tmp0 receives the initially allocated two64 object; physical allocation-order and final root aliases must both be tested. The requested live total is192 for pooled/debug-Windows and0 for debug-non-Windows. Additional Go metadata/inline storage is budgeted separately.
- allocate_if_needed h:132–145: clamp minimum; small output sign1 before Allocate; growing big output frees old first, then sign1, then allocates. Failure leaves the completed source sign/free prefix and the owned failed-token continuation.
- ensure_tmp_capacity h:156–163: test current capacity; free old scratch; compute unsigned32 (3*requested+1)>>1; allocate; debug capacity assertion. No preallocation, restoration, or automatic repair of missing scratch.
- set_big_i64/ui64 cpp:165–244: allocate only when small; capacity assertion; sign and digits in source order; size last. Native signed-negation UB adapters stay distinct from physical allocation behavior.
- SetLimbs cpp:306–354: strip leading zeros/read source caller buffer first; positive sign before helper; free old first only on growth; capacity/size/digits publications at the actual sites.
- Scratch setter cpp:247–280: inspect scratch digits; inline-zero/small branches release old output; otherwise set sign, exchange pointers, set result size, and allocate missing scratch only AFTER result publication. Keep both temporary indices and paired division quotient/remainder order.
- big_set cpp:1185–1216: self early return; sign first; copy preserves source capacity on allocation; old target free precedes replacement when target capacity<size(source).
- ensure_capacity cpp:1555–1595: minimum/early branches; small pointer allocation before source sign/digit conversion; big replacement allocates and copies while old remains live, THEN frees old and assigns new. It is intentionally different from big_set/ensure_tmp.
- Arithmetic scratch consumers cpp:357–559,1018–1184 and powers/shifts/root/GCD: use these helpers and actual spans. Existing source-publication fixes are prerequisites, not replaceable with staged mathematical equivalents.
- Destructor cpp:142–162: del(two64), del(intMin), tmp0,arg0,tmp1,arg1; nested lock destruction; then reverse member destruction (mpn before allocator), which frees pooled chunks. There is no implicit source allocator Consolidate call in MPZ.

## Natural scratch and selected numerical services

For complete constructor/runtime allocation fidelity, source mpn's u,v,t_ms,t_ab are real embedded sbuffer<uint32,16> storage (mpn.h:90,buffer.h:25–45), not four arena vector allocations. Proposed NaturalManager backing embeds four [16]uint32 arrays with scalar position/capacity and optional real Runtime Block for overflow. expand doubles unsigned capacity, allocates actual new overflow backing, copies only the source logical prefix, frees old overflow, then publishes pointer/capacity. Formatting's lexical buffers and debug verification temp must follow their actual construction/destruction routes. Debug-only allocation counts must not be reported as release counts.

Current Algorithm D pre-reserves optional add-back storage before output on branches that might never use it (mpn_indexed.go:89–96); source allocates at actual use in div_n. That existing storage adaptation is a concrete remaining action-order dependency. It must be removed/tested or explicitly kept out of any claim of complete numerical raw-counter fidelity. Per-cell backend work alone does not close it.

MPQ and binary rational owners should use the same integrated IntegerManager; they acquire no separate allocator. RationalService constants/cache and scoped numeral vectors contain scalar Int/Rational values whose big referents automatically use it. Their OWN vector/cache backings still need source vector/buffer runtime integration in later coordinated work. Do not count their present Go slices or Words reservations as source raw calls. Existing mathematical wrapper staging debts stay independently OPEN; changing backing is not evidence those source action orders were repaired.

## Global/TLS topology and physical charge accounting

One caller-provided Runtime domain must be shared by participating Integer/Rational managers and later SAT/AST/Params owners. Runtime global counters are actual source state; allocator requested bytes are a separate size_t quantity. Cold owner/ledger/directories and initialized-bit storage use Budget only and are excluded from source counters. A pooled MPZ cell touches an existing8192-byte root and increments requested count without a raw count increment. A new chunk is a real runtime8192 request; a direct cell is a real cell_size request. Freelist recycle performs no raw free; direct release does.

The committed runtime has one confined lane. Source TLS is per execution thread, whereas SYNCH controls per-manager nested locking; these are independent policies. Sharing one source lane with caller confinement is a selected integration profile, not OS TLS implementation or a claim that RationalService's mutex makes a shared Budget safe. Full runtime-global locked accounting plus multiple actual lanes, generated mem_initialize/mem_finalize order, and all participating owner hooks remain parent-coordinated dependencies. Default constructors must not manufacture unrelated supposedly process-global counters per manager.

TLS malloc writes its real SIZE_T header before local size/count and synchronization; non-TLS counts/checks before physical allocation. Realloc counts its unsigned delta before real storage in both branches; MPZ cell growth itself uses allocate/copy/free rather than realloc. Free reads extent, subtracts before physical free, and TLS sync is after free. Threshold comparisons are strict >100000/<-100000; getter does not flush; sync clears only local size and repeatedly contributes cumulative local count. Preserve unsigned32 realloc shrink promotion, signed overflow guards at the actual endangered operation, source OOM/count precedence, and pending headerless roots. No Budget-to-counter conversion, runtime.MemStats/RSS fallback, phantom byte backing, or synthetic getter is acceptable.

## Selected source finalization and owning cleanup contract

Existing IntegerManager.Close scans all live caller cells before releasing pools. Source destructor only releases its six managed roots before allocator destruction, and standalone allocator Close intentionally leaves direct blocks caller-owned. These differ observably for outstanding large user numerals. Silent automatic direct free would invent source counter events; silently dropping roots would violate Go ownership.

Parent selects an explicit split preserving existing owning Close semantics:

```go
func (*IntegerManager) RuntimeOwner() *memory.Runtime // same borrowed/owned actual domain, cold read-only identity
func (*IntegerManager) SourceFinalize() error
func (*IntegerManager) CloseChecked() error
func (*IntegerManager) Close() // existing signature; attempts CloseChecked, retains charged retry state on failure
```

SourceFinalize releases two64, intMin, tmp0,arg0,tmp1,arg1 in source order, then Natural member overflow in reverse member order, then allocator pooled chunks/actual diagnostics. It never invents direct user-cell frees. Managed release cursor and flags record completed actions, preventing a transport/physical error from repeating requested-byte subtraction or diagnostics. Ordinary arithmetic is closed after source finalization begins; cleanup-only actions retain genuine owner/provenance validation. A source-finalized manager retains its charged descriptor directory and actual direct caller raw-location roots until explicit owning cleanup. Source-pooled caller refs are invalidated only when the underlying chunks really free. Already freed direct refs and failedRef remain distinguishable.

CloseChecked is the explicit owning Go continuation: clean caller-owned live cells and completed unpublished pending cells once, perform any remaining source-member teardown, then release descriptor/dead-token/Natural/header metadata and close only an internally owned Runtime. Existing Close delegates to it; on a cleanup refusal it must fail visibly with a concrete cleanup-error panic while retaining all charged roots/cursors, since its existing void signature cannot return the error. CloseChecked returns the same error for explicit non-panicking continuation. This selected Go failure-surface adapter needs parent approval; it is not silent successful Close or C++ exception equivalence. This preserves existing caller expectation that normal Close invalidates/releases every numeral while marking additional direct free events as owning Go cleanup, not C++ destructor actions. SourceFinalize followed by CloseChecked must clean surviving direct refs without rerunning member frees or diagnostics. The direct allocator RawLocation token can outlive allocator context for real Runtime.Deallocate when source-finalized cleanup requires it; retain the actual capacity/extent and release phase before intrusive overwrite.

Raw-failure classification needs exact action-stage evidence: allocator Deallocate may subtract requested bytes, physically free, then return a TLS synchronization error. Numeric cleanup cannot infer all completed actions from the error or token alone. Proposed narrow shared-core refactor, for separate parent review, keeps ONE implementation and makes existing Deallocate delegate to a tracked path:

```go
type ReleaseProgress struct {
    RequestedAdjusted bool // allocator counter subtraction completed
    RawAccounted bool      // runtime before-free accounting completed
    PhysicalReleased bool // actual direct/raw block physically freed
    PooledLinked bool      // payload link and free-list publication completed
}
func (*memory.SmallAllocator) DeallocateTracked(n uint64, ref memory.ObjectRef) (memory.ReleaseProgress, error)
func (*memory.SmallAllocator) CompleteRelease(n uint64, ref memory.ObjectRef, progress *memory.ReleaseProgress) error
```

Runtime requires the corresponding internal single-body deallocation progress plumbing, because the begin/finish arithmetic guards have distinct source action positions. Progress fields change only after their exact completed action. CompleteRelease is expressly Go continuation: skip already completed requested/raw accounting, retry only the still-dangerous actual action, and never repeat physical free. No progress record goes in every numeral; one charged descriptor/cold pending release phase stores it only where cleanup is pending. A read-only locator query is insufficient for a before-free accounting-prefix error. This helper/API proposal is NOT approved production scope yet. Normal source deallocation remains the existing unchanged-body entry, with tracked continuation only handling the already established Go failure domain.

## Complete tests-first integration scope

1. Shared-owner extraction: existing runtime/accounting and allocator tests unchanged in behavior, concrete type/byte-layout audits, actual same Runtime identity observed from main and mpnum, no unused allocator; error and real stream bridge action-prefix tests. Missing concrete API RED first.
2. Source registered mpz tst1 (nine active ENSURE at lines36–44) and tst2 print-only arithmetic/lifetime replay against the REAL new backend, plus existing full mathematical/source tests. No invented allocator source assertion counts (its registered test has zero).
3. Constructor physical oracle: six cells, alternating tmp/arg order, only actual capacity/argument size writes, release one8192 chunk vs debug-direct six32-byte objects, correct genuine raw counters and cold physical charges. Real rational inherits the same owner; binary borrows it. Two managers sharing a runtime accumulate actual events without conflating their requested counters.
4. Capacity59/60 and60/61 branch tables, source cell_size wrapping, actual payload offsets, headers and freelist overwrite/reuse, no duplicate Words digit payload. Generation exhaustion/dead token metadata remains charged and excluded from raw counts.
5. Actual failure RED for oldfree-before-new versus retained-old ensure_capacity; result-before-missing-scratch; quotient-before-remainder; sign/header publication; allocator requested increment before real raw and late descriptor refusal. Inspect immediate live chunk/direct/pending roots and actual counters BEFORE formatting or cleanup. No retry repairs absent scratch.
6. Natural embedded16 arrays and first expansion, exact doubling/copy/free/publication, no early optional add-back allocation, branch-specific lexical formatting/debug roots; failure peaks old+new and lexical destructor order. Independent division/recomposition and finite exact oracles remain unchanged.
7. Source finalization versus owning cleanup decision fixtures: source six-root release order, direct caller lifetime, pooled chunk freeing, warning prefix and partial/pending owner charges, real runtime failure before/after free, exactly-once cleanup and no repeated subtraction.
8. Real RationalService/MPQ/binary/scoped consumers, polynomial GCD/CRA late-failure sweeps, and cross-owner CopyFrom limb bridge. Existing graph audits must count new actual cell roots and still-explicit non-cell buffers; no blanket baseline rewrites.
9. Warmed Mul plus MachineDivRem retains actual physical root plateau and zero host allocations. Benchmark measured Go behavior only, no C++ performance equivalence claim. Full native/checked-debug module, named race/actualJS, vet/WASI, source/evidence/no-generics audit on isolated committed base plus exact approved files.

## Exact phase B file map proposed for review

A. Shared extraction is committed6ffd717; requested/rounded uint32 codecs committed4370e7e; persistent tracked release committedfacaeb4. These sole real implementations are parent-owned. Phase B consumes those APIs without duplicate allocation/accounting bodies. Backend production remains unapproved; this packet and the two new mpnum test files are the only current design-phase edits.

B. New design-phase files currently authorized: mpnum/mpz_allocator_phase_b_test.go and mpnum/mpn_source_buffer_test.go, this existing proposal and the precise MPZ_OWNER_FAILURE_PACKET.md companion. Codec tests are committed and unchanged. These tests deliberately compile against proposed absent methods/config fields, actual real memory owners and source constructor topology. Existing production/tests are untouched. Missing API evidence must isolate against the parent phase A committed HEAD, so an absent package itself is not mistaken for absent codec/backend behavior.

C. Later production scope for approval: mpnum/mpz.go (owner/constructor/allocate/free/source finalization), mpz_slab.go (record payload becomes scalar ObjectRef, same ledger/retirement/failedRef), both mpz_ref profile files only if accessor adaptations actually require edits; NEW mpz_allocator_backend.go, mpz_allocator_integration_test.go and memory codecs. Existing Int release/checked layout remains unchanged. Concrete cold release cursor/pending ObjectRef fields are charged through the actual manager header; no uncharged heap node/list/interface per cell. Descriptor directories/records retain actual physical charges, with no separate source raw-count event.

D. Natural topology must be included in the mandatory backend packet: mpnum/mpn_storage.go, mpn_indexed.go, mpz_div.go, mp_format.go and mpz_power_shift.go are the direct view/storage sites requiring coordinated review; NEW mpn_source_buffer.go/test may supply a concrete inline16/overflow implementation. Audit remaining formatting/power/GCD callers before any existing edits. Do not edit shared consumers under blanket authorization. Numeric agent's read-only audit confirms quotRemCore already has direct paired scratch publication and single Algorithm D; these source actions must not be restaged. Optional early ab reservation remains a real acknowledged topology/action debt requiring tests-first at actual branch use.

E. Later separate service/vector/global-lane/generated-runtime packets cover RationalService/scoped vector/cache owners, MPFF/MPBQ, polynomial backing and SAT/AST/Params sites. The container cohort borrows one actual IntegerManager across related polynomial cores; coefficient-store release never closes that manager/runtime. Mathematical wrapper staging debts remain independently OPEN. Full solver is not completed by these primitives.

## Exact physical topology and pending preparation review

Source MPZ internal cells use one32-byte request at initial capacity6: eight header bytes plus24 actual limb bytes, never a Words descriptor or second digit array. Source allocator may round the request and runtime adds its genuine SIZE_T header; Budget also charges the actual runtime initialized bitmap, location directories, numeric descriptor ledger/deadslot and owner headers exactly once. Source raw allocation counters count only actual Runtime actions. Natural's four inline16 arrays are physically part of its cold owner allocation and charged once by reflected header size, with no four runtime allocations or arena payload page roots. If a separately allocated Natural owner remains the Go layout adapter, document its actual header allocation versus C++ embedded member; runtime source calls are still zero for inline members.

Constructor metadata preparation (manager/descriptor/deadslot/Natural header) precedes source cells as an explicit Go safety adapter. It must fail/unwind without source counters if no source action occurred. Once constructor source actions begin, allocate32→writecapacity6→publishnumeric descriptor is retained for tmp0,arg0,tmp1,arg1,intMin,two64. Argument size1 writes and final two64/tmp0 exchange are tested against actual raw bytes/location order. Uninitialized tmp sizes/digits cannot be manufactured as initialized by descriptor zeroing; physical initialization bitmap is authoritative.

A completed allocator object is captured in one charged pendingCell ObjectRef field before header/descriptor preparation can fail; capacity-header failure at a wrapped extent retains that real root and original request extent in pending metadata. Source allocator partial/headerless raw failures remain its own PendingBlocks and CleanupPending responsibility. A descriptor reservation refusal occurs after actual cell backing and capacity write and preserves both. Preparing a second pending object while the first is uncleaned must be rejected at actual metadata storage boundary, not overwrite an owned root. Constructor failure retains the already allocated charged owner itself whenever actual cleanup refuses. No separate construction-exception allocation is permitted. The precise original/cleanup cause, stage and cursor contract in MPZ_OWNER_FAILURE_PACKET.md supersedes older wrapper sketches. A completed cleanup returns the original cause with nil manager; a refused cleanup returns nil manager plus that same retained owner as error. All transitive roots remain reachable through the most-derived retained owner or an unchanged child-owner error. No C++ recoverability is claimed.

Natural expand is source uint32 capacity<<1. Memory request uses source sizeof(uint32)*newcapacity with selected size_t arithmetic; guard only actual endangered host/index/copy operations after source allocations where wrapped buffers are finite. Full old+new physical peak is real; old overflow frees after copying source pos bytes and before pointer/capacity publication. A post-free TLS error retains new pending root and old completed-free state, not a rolled-back buffer. Reverse Natural field destruction is t_ab,t_ms,v,u. Format lexical sbuffers and source verification/debug locals need their actual lifecycle order, not retained universal formatting arenas. This profile is wider than cell-only storage and cannot be marked complete while present preallocated arena formatScratch/text or early ab reservation remain.

## Source hashes

- mpz.h: 41891597e12173581e7ae3f108c048897761c48a191b06fd3ba160d1e98049b1
- mpz.cpp: 677698562746fdfa6d280f345d6efb805e810e3157c860881d22a9e8a0597aa3
- mpn.h: 78c22a9886044151fe808848e6045bfdae75ecd63ee394a8b50d222daffa7b50
- mpn.cpp: dc001ca86b19a2af70f45670fc5419d35ee105b4ed1f86cf12b684e95475fbca
- buffer.h: f5c876511a1fd5674ba5f9fa462d9206ffaad979ee90cc30e7d1b5fc9c4e07e1
- memory_manager.cpp: 3b884edd5463aa7ddfe68d6e0badc0695eff2b6b6fd90736850dd740f43e29d6
- small_object_allocator.h: 57b0a3bbc2d9fd10f008f371e3b5b9bee5535d39ab5cfbcb4afd9324018a140b
- small_object_allocator.cpp: 70cf910ed0a3871ea110ee01ff8d23a687f74af5972ab995dce32ba3e6e35994


## Concrete Natural buffer interface and test-only RED disposition

Proposed private pointer-free sourceLimbBuffer embeds [16]uint32 plus inline initialized bits, pos/cap scalars, actual overflow memory.Block and pending overflow/release phase. Runtime is passed only to ephemeral methods; no owner pointer/interface per buffer limb. Methods capacity() uint32, size() uint32, push(*memory.Runtime,uint32) error, read(*memory.Runtime,uint32)(uint32,error), reset(), finalize(*memory.Runtime)error exercise the exact source scalar sbuffer lifecycle. Four such fields live inside the actual Natural owner header and are reflected/charged once. Initialized spare data is not silently invented: source expand copies pos limbs, and initialized bits track only real assignments. reset retains overflow/capacity, finalize returns to inline16; future full source overloads remain explicit audited work, not implied by two lifecycle tests.

Eight initial design tests in three NEW files:

- memory/uint32_codec_test.go: actual initialized raw backing/LE codec/no events/endangered ranges; source intrusive pooled reuse/stale generation/actual shared capacity bytes.
- mpnum/mpz_allocator_phase_b_test.go: real default mandatory backend and borrowed Runtime identity; source six constructor physical cells/capacity/arg-size/finaltmp0-two64 swap; source direct caller survives SourceFinalize and owns CloseChecked cleanup; pure uint32 cell_size boundary including capacity0x3ffffffe=>0 andUINTMAX=>4.
- mpnum/mpn_source_buffer_test.go: inline16 creates zero raw allocations, first16→32 creates128-byte raw object/136-byte physical LP64 root; genuine expansion refusal retains old32 buffer/value/capacity and source reset retains overflow until finalize.

Observed API-absence RED on isolated .cache/release/mpz-backend-phase-b-red-6ffd717 (archive6ffd717 with only these test files): memory package terminal exit1 undefined Runtime/SmallAllocator.ReadUint32/WriteUint32; mpnum initial terminal exit1 unknown IntegerConfig.Memory and undefined RuntimeOwner/cellObject/allocator/SourceFinalize. Natural-only test isolation (temporarily holding the new MPZ test filename within .cache, then restoring it) terminal exit1 undefined sourceLimbBuffer. Logs .cache/mpz-backend-phase-b-memory-red.log, .cache/mpz-backend-phase-b-mpnum-red.log and .cache/mpz-backend-phase-b-natural-red.log. Compilation fails before execution; no named RUN/PASS or existing production behavioral failure is claimed. Expanded tests are design contracts pending exact layout/API approval, not source algorithm completion.

Existing production/tests, central metadata and Git are untouched. New files intentionally leave the live packages RED until the approved implementation packet or coordinator exclusion; other agents must isolate committedHEAD without these tests. No no-generics exception is introduced. All snapshots, probes, build cache and logs are below golang/.cache.


## Transitive constructor and close ownership audit

Actual production constructor edges are mpnum/mpq.go:49 NewIntegerManager and rational_service.go:76 NewRationalManager; no other production NewIntegerManager call was found. On failed integer construction, NewRationalManager currently releases only its unpublished own header and returns the original error unchanged. A retained IntegerConstructionFailure therefore stays reachable through that error without allocating a global registry. NewRationalService does the same for a failed RationalManager constructor. If a later RationalService constant init fails after the manager exists, current rational_service.go:97 calls s.Close() and discards its result; this path must instead preserve both the original failure and an error-retained cleanup owner if close refuses. Existing mpq.Close ignores leaf Del errors and releases its own header after z.Close, while RationalService.Close currently calls void manager.Close and then releases its service header. These are actual required transitive production sites, not solved by IntegerManager alone.

Later exact coordinated scope therefore adds mpnum/mpq.go and rational_service.go constructor/close paths and their dedicated tests, plus referenced destroyCache/infinitesimal helpers only where actual ignored Del errors require cleanup-cursor changes. BinaryRationalManager in mpbq.go borrows z but ignores eleven scratch Del errors before header release; its own CloseChecked/cursor must not close the shared z. The coefficient cohort similarly borrows z and must release only its own scratch/store roots. Existing root polynomial owners and other NumericalService cold adapters require a callsite inventory before edits, not blanket authorization.

Proposed retained construction-failure error owns a real charged partial owner pointer and exact original/cleanup error; Cleanup resumes its actual cursor. It is caller-owned cold exception boundary storage and never a manager-wide orphan registry. If a higher owner has already allocated its own header and live values, it must retain THAT owner until its real child cleanup completes, rather than forward a child-only cleanup handle while freeing the outer owner containing remaining roots. Existing constructors return nil,error on ordinary errors; retained cleanup is explicitly available via errors.As on the exceptional post-failure-cleanup case. Required tests include constructor reservation sweeps, actual late raw error prefix, outer-header retention until Cleanup, no header double refund, and no foreign owner cleanup.

NEW memory/release_progress_test.go adds two finite real-boundary contracts: LP64 TLS direct100000 free physically completes before genuine count-overflow synchronization error; ILP32 direct300 free adjusts requested counter before signed raw-counter subtraction refusal while physical root remains live. The tests seed only the already established internal counter undefined boundaries and use real allocated backing, never fake allocation events. Completion must skip exactly the published actions. No production has been edited to implement these proposed tracked methods.


### Exact cold failure-owner surface (proposed)

```go
type ConstructionFailure interface {
    error
    Unwrap() error
    Cleanup() error
    RuntimeOwner() *memory.Runtime
}
// IntegerManager, RationalManager and RationalService themselves implement this.
```

These are concrete cold errors, with one common cold interface for errors.As. No numeric cell/hot numeral contains an interface or cleanup closure. The most-derived allocated owner whose roots survive is retained, with its own charged header and lexical cleanup cursor. If an unpublished outer header has already been fully refunded, the original child error is forwarded unchanged. Cleanup never finalizes a borrowed runtime/cohort manager. Unwrap preserves the actual original refusal class; Error may use fixed labels rather than unbudgeted dynamic formatting. Existing void Integer/Rational Close must use checked cleanup then panic with a concrete owner-retaining cleanup error when a real error cannot be returned. RationalService.Close already returns error and must propagate checked manager/leaf cleanup instead of reporting nil and freeing its header. This is a Go explicit owning-failure adapter pending approval, not claimed C++ exception behavior.

TestIntegerConstructorRefusalRetainsTransitiveOwnership adds a bounded reservation sweep across NewIntegerManager, NewRationalManager and NewRationalService. It checks immediate runtime live roots/cold charges before runtime close, borrowed-domain lifetime, successful sweep endpoint, and explicit ConstructionFailure reachability/Cleanup whenever cleanup itself refuses. Budget failure alone does not force a post-free raw transport error; those genuine finite boundaries are separately covered by the two tracked-release tests. No claim is made that all late constructor cleanup errors were executed before the missing APIs compile.

ReleaseProgress needs scalar owner/ref/request identity bound to the actual retained release phase; CompleteRelease must reject foreign/tampered/replayed phase state before skipping actions. Exported flag snapshots are observational only and cannot authorize skipping real frees without matching actual owner state. This safety metadata is charged through real descriptor/cold header layout; no global orphan registry or hidden heap callback environment is permitted.


### Rounded physical object access is distinct from requested-byte API

Source cell_size(UINTMAX)==4. Actual LP64 release pool rounds this to8 bytes, so the capacity header word at offset4 is finite within that allocated class block. ILP32 pool rounds4 to4 and debug-nonWindows directly allocates4, where the same write is genuinely endangered. Existing SmallAllocator.Read/Write intentionally bounds ordinary byte access to requested bytes; reusing that boundary for the source physical header would incorrectly reject the defined LP64 action. Add proposed read-only ObjectExtent(ObjectRef)(uint64,error) returning loc.extent for pooled objects and actual Runtime.Size for direct objects, with genuine owner/generation/lifetime guards and no allocation/counter actions. MPZ RawLocation+Runtime word views must additionally validate every word range against THIS physical object extent, never whole chunk extent or sourceclaimed capacity. SmallAllocator byte/word requested-view APIs keep their ordinary requested bounds. The new finite TestMemoryMPZRoundedPhysicalObjectExtent covers64 release=>8,32 release=>4,64 debug direct=>4, existing requested guard unchanged and actual LP64 capacity publication. Header size/capacity/digit guard positions remain exact, with no blanket capacity rejection.

Current tests-first scope now consists of twelve groups in four new files, not the earlier initial eight. The larger semantic/model/fault suites listed above remain required during the approved implementation packet; compilation-only RED is not evidence those paths executed.


### Revised failure ownership: no extra header after reservation refusal

Parent review correctly rejects relying on an uncharged newly allocated exception header when Budget already refuses. Prefer the already allocated, reflected/charged partial owner itself as the concrete ConstructionFailure: IntegerManager, RationalManager and RationalService embed failure cause/stage and cleanup cursor in their actual owner header and implement fixed-label Error/Unwrap/Cleanup/RuntimeOwner only for this cold boundary. Returning the existing owner pointer as error introduces no second error-owned heap/header or closure. Normal pre-owner allocation refusal returns the existing static cause, with no partial owner. Failed child constructors forward that child owner error unchanged after refunding unpublished outer header; later outer constant failure retains the existing outer owner itself. Fields must avoid self-Unwrap cycles and retain every already constructed child root. This replaces the earlier separate IntegerConstructionFailure/RationalConstructionFailure/RationalServiceConstructionFailure wrapper-struct sketch. CloseChecked similarly returns the existing retained owner as a cleanup error when a real failure occurs; void Close visibly panics that same retained error rather than silently succeeding. Exact API/field edits remain pending review.

True constructor initialization map (_MP_INTERNAL uint32 digits): allocate writes capacity ONLY. Constructor writes arg0/arg1 size1; big intMin setter writes digit0=0x80000000 and spare digit1=0 before size1; original two64 setter writes size2/digits0,1=UINTMAX. Add's get_sign_cell<1>(one) writes arg1 digit0=1; arg0 digit0 is still uninitialized. Natural add writes originaltmp0 digits0..2=0,0,1, and result transfer writes its size3 after swap; tmp0 then owns the oldtwo64 object with size2 and two initialized UINTMAX digits. Untouched tmp1 size/digits remain uninitialized. The expanded constructor test now asserts these actual initialized/uninitialized byte domains, never treating zeroed safety metadata as initialized source memory.

Existing resizeCell/prepareTmpSpan are not automatically physical m_size writers: source scratch capacity/span preparation passes a digit buffer length to mpn while source set<IDX> writes result m_size only after transfer. Later integration must distinguish transient scratch-span bookkeeping from source header publication at every actual caller. Copying current cached Go size blindly into physical scratch headers would invent defined writes/failure prefixes. Separate physical size/capacity getters and ephemeral span views are required; future indexed access guards check actual object extent/initialization without globally reading an uninitialized scratch header.


## Phase B tests/design handoff (no production approval)

Final design-test inventory has12 groups/four new files. Snapshot .cache/release/mpz-backend-phase-b-red-6ffd717 contains only the committed shared package plus these new tests; API-absence compile RED is recorded, and full frozen-tree CheckNoGenerics completed exit0 in .cache/mpz-backend-phase-b-nogen.log. The new tracked-release-only isolation separately failed compile on undefined DeallocateTracked/CompleteRelease, log .cache/mpz-backend-phase-b-release-progress-red.log. No execution PASS or corrected production behavior is claimed. Final test hashes:

| New test file | SHA256 |
| --- | --- |
| memory/uint32_codec_test.go | cc6146cf1dbc4059c709bea0f174367cc0b946d81b43f5dd7f0b47ee66ce3a5e |
| memory/release_progress_test.go | bf682276c73cae9d7aff843b56938f783af25761d84b239ed929b0b05f6aff9f |
| mpnum/mpz_allocator_phase_b_test.go | 9fa9804908acfeaa43ed9babf82eff62c27cb0eca39281509523bed7747a0d5d |
| mpnum/mpn_source_buffer_test.go | 8f149f480933792fe6d0bce8cd94a4bcae0a1b572e0fee84e6f101a12dad773a |

The review gate must settle tracked release progress authority, error-retained OWNER ITSELF versus void Close failure surface, transitive rational/service cleanup cursors, actual physical rounded object access, uninitialized temporary field handling and Natural/full formatting topology before any production edits. These are concrete required source/runtime dependencies; they do not authorize narrowing to cell-only optional backing. Numeric Div/Inv and container coefficient-cohort proposals remain separately owned and must be coordinated without copying their production or inventing independent runtimes. Full solver/runtime integration remains required.


## Current design authority

The exact phaseB ownership/cause/cursor/initialized-header and file contract is now MPZ_OWNER_FAILURE_PACKET.md. Backend default replacement and complete Natural topology are still NOT implemented. The eight design test groups (six Integer, two Natural) compile against absent backend APIs; codec/release implementation evidence does not establish these tests as runtime passes. Production interfaces and final file ownership require parent approval.
