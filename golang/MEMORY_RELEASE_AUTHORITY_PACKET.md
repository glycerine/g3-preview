# Proposed retained release authority for the actual memory owners

Selected tracked runtime/allocator prerequisite implemented on4370e7e after genuine tests-first RED. Codec prerequisite is separately committed4370e7e. MPZ/Natural and transitive numerical failure-owner APIs below remain design-only and are not implemented by this packet. This design preserves one real deallocation implementation, existing source action positions, and pointer-free user tokens. Exported progress flags cannot authorize skipped actions.

## Persistent authority and actual charge

Extend the existing private runtime directory record with a concrete scalar release record: old generation, nonzero owner-local release serial, stored extent, and stage. Extend the existing private allocator location record with a scalar release record: original requested n, matching ObjectRef generation, actual raw Block, runtime release serial and allocator stage. These fields are physically part of the actual existing directory backing; every directory reserve/copy/old+new peak uses the increased reflected record stride. No separate per-record heap allocation, slice, interface, map or closure is added. Owner-local serial allocation is a scalar cold header field, reflected/charged once; it never wraps/reuses authority.

The returned ReleaseProgress is a pointer-free observational snapshot with private owner/slot/generation/serial identity and visible completed-action flags. CompleteRelease resolves the genuine owning location and its persistent phase, checks the private identity, and uses ONLY persistent stage to choose actions. Caller changes to visible booleans do not change authority. A copied snapshot is safe: completion is idempotent while its exact completed phase remains retained; an old generation/serial after reuse is rejected and can never free the newly allocated object. A foreign/forged/zero progress cannot bypass a source action. Snapshot equality is not the ownership model.

Existing ObjectRef and Block layouts remain unchanged. IntegerManager's descriptor/pending cell stores only the real scalar token and release snapshot; there is no global registry. All ordinary source operations retain their existing owner/lifetime checks and source counter-before-ref-check positions. Reentrancy/concurrent use of a confined owner is forbidden as before.

## One-body runtime transition

Current memory/runtime.go:282–306:

1. Validate genuine live Block and read its initialized SIZE_T header.
2. beginExtentDeallocation subtracts from local/global counter at the selected source point. ILP32 may fail at signed subtraction BEFORE mutation/free; LP64 source size_t arithmetic is modular.
3. Release actual physical Budget bytes/initialization backing, clear data/bitmap, mark not live, increment checked generation, decrement live count, and clear caller Block.
4. finishDeallocation may synchronize TLS after physical free. It can publish global size then fail at cumulative global count addition; there is no still-live block to free again.

Internal tracked phase records HeaderRead -> RawAccounted -> PhysicalReleased. A before-subtraction refusal remains HeaderRead; retry through explicit continuation may attempt that subtraction only after caller has resolved the source-undefined injected counter condition. An error after PhysicalReleased keeps that completed stage and the actual partially published synchronization state. Owning cleanup never repeats subtraction/free or invents successful synchronization/rollback. There is no ownership root left to release in that phase; cleanup acknowledges completion without repairing source counters. Ordinary getters still report the actual resulting fields.

Existing Runtime.Deallocate delegates to this single body; an internal progress-aware entry receives/resolves the same charged record. Allocation/reuse cannot erase an active before-free phase while the real block remains live. A freed slot may be reused; old completed-token continuation is rejected or recognized as a completed no-action tombstone, never resolved to the new root. The exact retained/tombstone scalar representation needs tests first before edits.

## One-body allocator transition

Current memory/small_allocator.go:367–430:

- open/n==0/size guards occur where they already do.
- Debug non-Windows bypass resolves the direct object and invokes real Runtime.Deallocate without requested counter subtraction.
- Pooled/release and debug-Windows paths perform source debug counter assertion, then requested subtraction BEFORE location/provenance validation.
- Actual direct branch performs interior-header guard then Runtime.Deallocate; the source location's raw/live fields follow the caller Block's actual cleared/noncleared result.
- Actual pooled branch writes the intrusive pointer word, publishes the free-list head, then live=false/free=true. Defined duplicate pooled release with the same generation still performs the source subtraction and self-link; a completed normal release is not treated as a pending error and cannot be used to prohibit that finite source path.

Private allocator stages are RequestedAdjusted (where applicable), RawReleaseStarted, RawAccounted, PhysicalReleased, or PooledLinked. A successful source action updates phase only AFTER it actually completes. Completion skips requested subtraction only because the bound real phase says it already happened. It invokes the one runtime continuation for pending direct release, or resumes the actual still-dangerous intrusive word action before head publication. After source typed failure there is no ordinary implicit retry/reset or automatic counter repair; explicit CompleteRelease is a Go ownership continuation, not C++ exception recovery.

Invalid/foreign ref failure still retains the genuine requested subtraction prefix when source reaches that before validation. Such a result has observational flags but no valid locator-bound authority and cannot authorize CompleteRelease. Do not introduce an early global ref guard to avoid that source mutation. If a physically valid pooled root remains after another invalid argument/endangered access, owner-wide CleanupChunks/Close retains its genuine existing root traversal rather than manufacturing a successful individual free.

## Proposed signatures and guards

```go
type ReleaseProgress struct {
    // private scalar owner/ref/serial binding; no pointer fields
    RequestedAdjusted, RawAccounted, PhysicalReleased, PooledLinked bool
}
func (*SmallAllocator) DeallocateTracked(uint64, ObjectRef) (ReleaseProgress, error)
func (*SmallAllocator) CompleteRelease(uint64, ObjectRef, *ReleaseProgress) error
```

An output flags struct by itself is insufficient. CompleteRelease must verify its n matches the bound source invocation and all private identities match retained owner state; visible flags are refreshed from persistent state. It rejects nil progress at its actual continuation entry, wrong n, foreign owner, stale generation, tampered binding and slot reuse before releasing a different root. Ordinary Deallocate remains source semantics, delegating without cloning/replaying the algorithm. An active error phase may require explicit cleanup before a further mutation of that SAME root; that is a documented post-error ownership continuation rule, not a restriction on normally completed duplicate pooled free.

New helper/record changes would touch ONLY memory/runtime.go, memory/small_allocator.go and new memory/release_progress.go plus exact new tests, after approval. Reflected runtime/allocator directory stride tests and header expectations must be updated intentionally; no fake source raw counters for this Go safety metadata. Source free-lists remain intrusive in the actual object payload.

## Genuine finite tests and further implementation gates

NEW memory/release_progress_test.go already supplies two missing-API compile RED fixtures against6ffd717:

- LP64 TLS direct100000 physically frees, then real count-overflow synchronization returns ErrBounds; requested/raw prefixes and physical absence remain. Completion performs no counter or physical action.
- ILP32 direct300 requested subtraction completes, but injected LL_MIN raw-size subtraction fails before free. Actual block/bitmap remain charged; after restoring only that injected undefined counter field, completion subtracts raw once, frees once and never resubtracts requested n.

Both use the real Runtime backing; only the bounded signed-overflow counter boundary is seeded, as existing source runtime tests do. They are currently compilation RED, not executed production PASS. Implementation additionally needs caller flag-tamper/foreign/reused-slot rejection, successful/idempotent completion, bypass/debug-Windows/pooled source actions, retained directory/header charges, existing duplicate-free regression preservation, and no repeated diagnostics/free during finalization.

Constructor cleanup error retains the ALREADY allocated charged owner itself, embedding cause/stage/cursor in reflected header. It must not allocate a new failure wrapper when Budget refuses. Integer/Rational/Service owners can implement the cold ConstructionFailure interface with static Error/Unwrap/Cleanup/RuntimeOwner, returning the actual partial owner as error and retaining the most-derived real roots. Void Close cannot return failure: proposed checked continuation plus visible panic uses that same owner, with no extra exception/header. These lifetime surfaces remain pending parent exact review and are not implemented in the codec chunk.

## Exact proposed fields and retry rule after codec handoff

Codec2 is frozen independently on45f5d4b. The release implementation remains tests-first; fields below are the precise proposed persistent representation:

```go
type runtimeReleaseState struct {
    serial, extent uint64
    generation uint32 // generation of the original live raw token
    stage uint8       // empty, headerRead, accounted, physicalReleased
}
// memoryRuntimeBlock adds release runtimeReleaseState.
// Runtime adds nextReleaseSerial uint64 (checked never-wrap owner namespace).
type allocatorReleaseState struct {
    serial, requested uint64
    raw Block
    runtimeSerial uint64
    generation uint32 // generation of the original ObjectRef
    stage uint8       // empty, requestedAdjusted, rawStarted, pooledLinked, done
    active bool
}
// smallObjectLocation adds release allocatorReleaseState.
// SmallAllocator adds nextReleaseSerial uint64.
// ReleaseProgress has private owner/slot/generation/serial and visible flags.
```

Actual reflected directory strides and cold owner sizes automatically charge these records/scalars, including old+new peaks. Stages are not source fields or raw events. Serial exhaustion is an explicit finite metadata-identity adapter checked before starting a NEW release invocation; resuming a bound existing invocation requires no serial reservation or allocation. Completed source pooled duplicate-free still starts a NEW invocation and preserves requested wrap/self-link. No changes to public Block/ObjectRef fields.

ONE rule for ordinary Runtime.Deallocate: if its genuine live Block already has an active before-physical release phase, resume the SAME actual phase rather than begin a new subtraction. There is no dead-end ErrPending and no second accounting attempt after accounted stage. For physicalReleased, the caller Block has already cleared, so original ordinary nil/zero token behavior remains invalid; the owning allocator's bound continuation acknowledges completed release without calling Runtime again. Before-free HeaderRead failure can retry actual raw subtraction; counter overflow had not published that subtraction. Existing source-undefined error continuation is expressly a Go ownership adapter. Postphysical cleanup performs no TLS retry/repair.

Ordinary SmallAllocator.Deallocate for the SAME matching active error invocation likewise resumes without repeating requested adjustment; wrong n or generation is a typed continuation mismatch, without changing the already stored phase. An ordinary call after a fully completed pooled release is different: it retains the source's defined duplicate-free counter/self-link behavior. CompleteRelease checks private owner/ref/serial and n against real location state, ignores caller visible flags, and refreshes the observational snapshot after action. Completed matching snapshots are idempotent until source reallocation/new release invalidates their authority.

### Owning callsite inventory to preserve

- Runtime.Deallocate public path: one tracked body, scalar live token retry.
- SmallAllocator.Deallocate bypass/direct paths: source requested rules first; internal runtime progress is attached to the genuine location. Intrusive pool path has no raw free.
- SmallAllocator.freeChunks used by Reset/Close: existing cold cleanup cursor updates and invalidateChunk when raw token clears; active beforefree runtime phase resumes transparently. Afterfree error still advances actual cursor before returning. No head normalization or counter replay.
- CleanupChunks explicit consolidation continuation: scans actual retained live chunk locators, frees via same Runtime.Deallocate rule, and invalidates only physically freed chunks; no repeated oldfree.
- CleanupPending raw/partial allocation and two pending lexical roots: real Runtime.Deallocate beforefree phases resume transparently, or CleanupPartial retains its own headerless-root path. Successful cleanup clears only that real token. Partial-header cleanup has no stored SIZE_T source subtraction and must not adopt ordinary deallocation authority.
- Runtime-backed lexical pointer-vector reallocate/dispose: retained raw token points to the same active beforefree phase; retry disposal never subtracts twice. Reallocation's separate allocate/oldfree/publication ordering is not silently replaced with this deallocation change.
- Runtime.Close remains source-owner lifetime guard: it rejects genuinely live raw blocks. It does not invent freeing arbitrary caller roots; after actual owning allocator cleanup, it releases only directory/header. Physical-completed phase metadata has no live root and can be discarded with the charged directory.

Before production add genuine RED for forged visible flags, private foreign/slot-generation/serial authority, slot reuse, successful duplicate pooled release, exact increased charge/cleanup baseline, and SmallAllocator Close/CleanupPending retry after a REAL beforefree runtime refusal. This latter fixture must use allocated roots and bounded source arithmetic failure, not a pretend callback budget event. All these fit approved runtime/small_allocator/new release helper/test scope; consolidation/vector files need no changed algorithms if ordinary Runtime retry works as specified. Any actual caller gap requiring them must be reported before edits.


## Implemented selected prerequisite and frozen disposition

The sole runtime Deallocate now delegates to deallocateTracked, whose actual private memoryRuntimeBlock.release records serial/extent/originalgeneration/header→accounted→physical stage. Allocate and successful Reallocate clear a prior generation's release record at actual new-binding publication. Header/initialized-byte/source size_t and counter algorithms are unchanged; the new scalar record only remembers completed actions. Runtime.nextReleaseSerial never wraps. The real directory record stride/header reflection charge includes it.

The sole allocator Deallocate delegates to DeallocateTracked. actual smallObjectLocation.release contains serial/requested/originalraw/runtimeSerial/originalgeneration/runtimeStage/active/requestedAdjusted/pooledLinked scalars. SmallAllocator.nextReleaseSerial is charged within its reflected owner header. An active genuine same-n/ref retry resumes; fully completed normal pooled free starts a new serial and preserves source duplicate self-link/requested wrap. Pool object reuse clears its old release authority when generation actually advances; direct locator installation clears via actual replacement record. Invalid pointer observations have no bound completion authority while preserving requested-before-ref-check prefix.

CompleteRelease validates owner/slot/generation/serial/request against the retained real record and ignores all caller visible flags. Complete acknowledgments perform no physical/counter/TLS repair actions. copied/foreign/forged/zero/stale/reused binding is rejected. Namespace exhaustion stops before a new release begins, preserving existing root; already-active continuation needs no new serial. Source object/Block layouts remain unchanged, no generic/unsafe/header-owning exception wrapper or per-record heap/closure was introduced.

Eight new release groups:

1. TestMemoryTrackedReleaseLatePhysicalPrefix: actual LP64 TLS free then genuine count-overflow; completed physical/counter prefixes preserved, acknowledgment no replay.
2. TestMemoryTrackedReleaseBeforePhysicalRefusal: actual ILP32 raw signed-boundary refusal retains root after requested adjustment, completion frees once.
3. TestMemoryOrdinaryReleaseResumesRequestedPrefix: actual behavioral RED before implementation (4294966996 versus0 after retry), now corrected single-body continuation.
4. TestMemoryReleaseAuthorityIgnoresCallerFlagsAndRejectsReuse: tampered visible flags cannot skip free, completed binding idempotent, reused/empty/wrong-size authority cannot free a new root.
5. TestMemoryReleaseAuthorityPreservesDuplicatePoolAndCharges: genuine same-generation source duplicate counter wrap/intrusive cycle still occurs, newer serial invalidates older authority, real cleanup refunds all charges.
6. TestMemoryReleaseAuthorityRejectsForeignAndTamperedBinding: private serial/generation/foreign/copied/nil rejection leaves source fields and actual root unchanged.
7. TestMemoryOwningCleanupResumesBeforePhysicalFree: genuine chunk Close and retained lexical CleanupPending retries; these already passed before tracked implementation, so no fabricated new RED or caller edit is claimed.
8. TestMemoryReleaseAuthorityPhysicalChargesWarmAndNamespace: pointer-free snapshot, reflected owner/directory graph agreement, namespace no-wrap, armed Budget physical plateau and0warmhostallocations.

Real API-absence RED .cache/memory-release-authority-api-red.log (45f5d4b plus newtest, -gcflags=-e) includes missing methods/type/private serial metadata. Actual behavioral RED .cache/memory-release-resume-actual-red.log uses only the extracted ordinary retry fixture against45f5d4b and fails on completed requested subtraction repeated. Existing owning-beforefree fixtures PASS .cache/memory-release-owner-cleanup-existing.log, honestly distinguished from new missing API/behavior. First lower native/debug GREEN logs .cache/memory-release-first-green.log and .cache/memory-release-first-debug-green.log.

Frozen snapshot .cache/release/memory-release-green-4370e7e includes committed4370e7e plus ONLY memory/runtime.go, small_allocator.go, release_progress.go and release_progress_test.go. Exact57 lower test names (committed49+new8) are .cache/memory-release-final-names.txt. All other new backend/Natural API drafts are excluded. No main, consolidation/vector, codec, arena, mpnum, metadata or Git changes in this chunk. Final hashes:

| File | SHA256 |
| --- | --- |
| memory/runtime.go | fa3a0c19c19005515c814d5217d4d707f801a5a8e5c44e0cf01524136b89bee7 |
| memory/small_allocator.go | cf0fa58e94f51579c653274448e7920b70d143b0b7f9c519090584712f9856b9 |
| memory/release_progress.go | b3da8fe2530e34444cf2c4cd217e0a716ae455018e8fa30ca379e9f4e214f40e |
| memory/release_progress_test.go | 7c0e02f75cb5991d8d1051e0c4a7cce02fe40e41aa4f52df98fb01af5b5ef1f4 |


## Final authority accuracy and terminal gate handoff

Private runtimeSerial in the allocator record is an observed receipt from its real Runtime deallocation; it is NOT a separate caller-supplied validator and never authorizes skipped actions. Runtime resumes from its OWN actual memoryRuntimeBlock.release after block owner/slot/live-generation validation. Allocator CompleteRelease validates its own private progress owner/slot/generation/serial/request against its real location release state. On a pending direct call it passes loc.raw (the actual still-live Block) through the Runtime body's genuine validation; after physical release, its retained runtimeStage acknowledges completion without dereferencing/revalidating a possibly reused raw slot. Thus a completed old free cannot target new backing. Public flags are ignored entirely.

All following gates use4370e7e plus exact frozen4; parent will independently integrate with later965949a prime commit. The phaseB numeric/Natural and additive/factor drafts are excluded, and no C++ source/compiler dependency is required by ordinary tests.

| Gate | Terminal handle | Result | Log |
| --- | --- | --- | --- |
| Full native ./... | 1352 | PASS exit0 | .cache/memory-release-final-native.log |
| Full checked-debug ./... | 2840 | PASS exit0 | .cache/memory-release-final-debug.log |
| Complete lower memory checked-debug race | 24885 | exact57 RUN/PASS exit0 | .cache/memory-release-final-race.log |
| Complete lower memory actual checked-debug JS | 64848 | exact57 RUN/PASS exit0 | .cache/memory-release-final-js.log |
| Full checked-debug vet | 33469 | PASS exit0 | .cache/memory-release-final-vet.log |
| Full checked-debug WASI build | 64611 | PASS exit0 | .cache/memory-release-final-wasi.log |
| Complete frozen CheckNoGenerics AST audit | terminal command | PASS exit0 | .cache/memory-release-final-nogen.log |

Both RUN/PASS sets were checked exactly against all57 selected names. All four workspace/snapshot hashes still match the frozen table. Production/test/doc are released to parent. Metadata/source evidence/Git remain parent-owned; this prerequisite adds no solver, global memory-hook, MPZ/Natural or numerical transitive cleanup implementation claim. Those required integration contracts stay open until separately approved/translated/tested.


## Parent terminal integration on prime base

Independent snapshot `.cache/review/memory-release-parent-965949a` is965949a plus exactly four Go files and three documentation/ledger files. Full native72101 and checked/debug55470 passed. Checked/debug race50756 and actual JS62888 each executed the identical57 lower-memory RUN/PASS groups. Vet82921, full WASI build90753 and checked/debug test compilation (root93815, mpnum31297, memory immediate) passed. Integrated source/evidence/no-generics audit64978 exited0:2102units,677APIs,33ported/148partial/1921unported. Four reviewed Go hashes remained unchanged through verification. Parent read both deallocation bodies, actual accounting begin/finish action sites, all eight new tests and the borrowed receipt/authority distinction. No whole unit promoted. Concurrent additive, factor/adoption, full MPZ/Natural and SAT drafts were excluded; the complete live worktree is not claimed green. Numeric construction/cleanup owners remain separate required work.
