# Shared concrete memory package: phase A API and extraction map

Implemented phase A extraction, frozen on baseline57d0c64. One real runtime/accounting/allocator implementation now lives in package memory and is used by the main g3 adapters. The original41 named tests are preserved:37 lower-package groups and four genuine main stream fixtures. Eight new API groups and four main bridge groups give53 selected groups. MPZ backend adoption remains the next required phase; this packet does not claim that mpnum already uses the allocator. Parent owns central metadata, progress and Git integration.

## Implemented public API

Package path github.com/glycerine/g3/golang/memory. No generics, unsafe, cgo, per-record interface fields, payload slices escaping to callers, or duplicate runtime/allocator bodies.

```go
type RuntimeConfig struct {
    Budget *arena.Budget
    SizeTBits uint8             // default64; supported32 or64
    ThreadLocal bool           // selected confined lane profile
    Transport ErrorTransport  // one borrowed static named function; nil uses Failure
}
type FailureKind uint8
const (
    FailureMemout FailureKind = iota + 1
    FailureAllocExceeded
    FailureVectorOverflow
)
type Failure struct { Kind FailureKind }
func (Failure) Error() string
func (Failure) Message() string
func (Failure) ErrorCode() uint32
func (Failure) HasErrorCode() bool
type ErrorTransport func(Failure) error

func NewRuntime(RuntimeConfig) (*Runtime, error)
func (*Runtime) CheckOpen() error
func (*Runtime) OwnerID() uint64
func (*Runtime) Budget() *arena.Budget // borrowed; not a source memory getter
func (*Runtime) SizeTBits() uint8
func (*Runtime) Allocate(uint64) (Block, error)
func (*Runtime) Reallocate(*Block, uint64) error
func (*Runtime) Deallocate(*Block) error
func (*Runtime) CleanupPartial(*Block) error
func (*Runtime) Size(Block) (uint64, error)
func (*Runtime) Read(Block, uint64) (byte, error)
func (*Runtime) Write(Block, uint64, byte) error
func (*Runtime) SetMaxSize(uint64) error
func (*Runtime) SetMaxCount(uint64) error
func (*Runtime) SetWatermark(uint64) error
func (*Runtime) AboveWatermark() (bool, error)
func (*Runtime) AllocationSize() (uint64, error)
func (*Runtime) MaximumUsed() (uint64, error)
func (*Runtime) AllocationCount() (uint64, error)
func (*Runtime) State() RuntimeState
func (*Runtime) Stats() RuntimeStats
func (*Runtime) Close() error

// Opaque pointer-free records, same scalar fields as existing private tokens.
type Block struct { origin uint64; slot, generation uint32 }
func (Block) OwnerID() uint64
func (Block) Slot() uint32
func (Block) Generation() uint32

type RuntimeState struct {
    GlobalSize, LocalSize, GlobalCount, LocalCount, Peak int64
    OutOfMemory bool
}
type RuntimeStats struct {
    OwnerBytes, DirectoryBytes, PhysicalBytes, InitializationBytes, TotalBytes uint64
    LiveBlocks uint32
}

type SmallAllocatorProfile struct { PointerBits uint8; Debug, Windows bool }
type AllocatorIOEvent uint8
const (
    AllocatorIOLeak AllocatorIOEvent = iota + 1
    AllocatorIOVerboseStart
    AllocatorIOVerboseEnd
)
type AllocatorIO interface {
    Emit(*SmallAllocator, AllocatorIOEvent) error
}
type SmallAllocatorConfig struct {
    Runtime *Runtime
    Profile SmallAllocatorProfile
    IO AllocatorIO             // single borrowed cold receiver; no closure
    Verbosity uint32
}
func NewSmallAllocator(SmallAllocatorConfig) (*SmallAllocator, error)
func (*SmallAllocator) RuntimeOwner() *Runtime
func (*SmallAllocator) Allocate(uint64) (ObjectRef, error)
func (*SmallAllocator) Deallocate(uint64, ObjectRef) error
func (*SmallAllocator) Read(ObjectRef, uint64) (byte, error)
func (*SmallAllocator) Write(ObjectRef, uint64, byte) error
func (*SmallAllocator) RawLocation(ObjectRef) (Block, uint64, error)
func (*SmallAllocator) AllocationSize() (uint64, error)
func (*SmallAllocator) WastedSize() (uint64, error)
func (*SmallAllocator) NumFreeObjects() (uint64, error)
func (*SmallAllocator) Consolidate() error
func (*SmallAllocator) Reset() error
func (*SmallAllocator) CleanupPending() error
func (*SmallAllocator) CleanupChunks() error
func (*SmallAllocator) Close() error
func (*SmallAllocator) Stats() AllocatorStats
func (*SmallAllocator) PlacementDelete(ObjectRef) error
func (*SmallAllocator) PlacementDeleteArray(ObjectRef) error

type ObjectRef struct { owner, address uint64; slot, generation uint32 }
func (ObjectRef) OwnerID() uint64
func (ObjectRef) Slot() uint32
func (ObjectRef) Generation() uint32
func (ObjectRef) LogicalAddress() uint64
type AllocatorStats struct {
    OwnerBytes, LocationBytes, TotalBytes uint64
    Chunks, DirectObjects, PendingBlocks uint32
}
```

Runtime/SmallAllocator contain private self pointers, cold configuration, actual directories and accounting; neither can be copied as a Go owning value. Constructors validate before header allocation and preserve existing reserve/unwind behavior. Token accessors are zero-allocation scalar views, not raw pointer factories. RuntimeOwner returns the actual borrowed Runtime pointer (nil for nil/foreign-copied owner), without allocating another object; identity remains available after normal Close. All ordinary source operations retain closed/self/provenance/generation guards and existing source action positions.

Default FailureKind mapping: Memout=>code101/message `out of memory`; AllocExceeded=>code113/message `number of configured allocations exceeded`; VectorOverflow=>code0/no-code/message `Overflow encountered when expanding vector`. Failures are cold caller-owned returned errors, not owned payloads. Transport is invoked at the existing actual failure site AFTER the appropriate counter/physical prefix. Domain/ownership/initialized/header guards retain existing typed errors and are not rerouted as fabricated vsnprintf/OOM. A nonnil transport must return a nonnil error; nil return is a typed invalid-transport error at that call, never successful allocation. Only static named functions are permitted; captured closure environments would need explicit independently owned/charged storage and are outside this API contract.

Exported sentinel aliases correspond one-to-one to existing errors: ErrRuntimeOwner, ErrRuntimeLive, ErrRuntimeUninitialized; ErrAllocatorOwner, ErrAllocatorAssertion, ErrAllocatorUnreachable, ErrAllocatorPending, ErrAllocatorNonterminating. Main exports can alias those exact error identities. New ErrAllocatorIO diagnoses missing cold IO only when an actual enabled emission is reached, not at constructor. It does not claim source global stderr disappearance. Process-wide std::cerr/verbose binding, badbit/flush/exits remain OPEN.

ReadUint32/WriteUint32 byte-word codecs and MPZ-specific descriptor preparation are phase B proposals, not required by this API-absence packet. Their bounds/initialization action order needs separate tests before implementation.

## Exact cold error and diagnostic bridge contracts

A lower package cannot import main NumericTextStream, GlobalParamCString, or Z3Error. The dependency is deliberately one-way:

- Main `memoryErrorTransport(f memory.Failure) error` is a static function, mapping101/113 to the existing actual Z3Error and VectorOverflow to existing NewDefaultException with its fixed source message. No fmt/serialization/closure. RuntimeConfig stores only its function value (actual reflected owner header charge includes that field). Existing main exception definitions remain in place and retain their nominal ErrorCode surface. Lower package's default Failure supports direct mpnum/library use; main source consumers use the real bridge.
- Main `smallObjectIOBridge.Emit(a *memory.SmallAllocator, event memory.AllocatorIOEvent) error` is a static method on one cold receiver containing only borrowed Diagnostics/Verbose streams, static ID flags/string, BorrowedID region/scalar IDRef, and its owned charge/lifetime bookkeeping. It reuses current real CString and NumericTextStream insertion bodies unchanged. The interface lives ONCE in the allocator's cold config, never in ObjectRef, a numeral, location, or byte record. No per-emission/per-record closure or boxing environment is created.
- The lower core dispatches Leak only after its debug requested>0 condition and actual chunk frees. Dispatch executes the existing warning prefix, then current ID read, then count/suffix. VerboseStart occurs before lexical pointer-vector construction; VerboseEnd occurs before reverse lexical destruction. The receiver queries actual source WastedSize and Runtime.AllocationSize at their original insertion positions. Reentrant mutations/Close are forbidden; read-only getters needed by the bridge remain valid at those stages.
- Native source m_id borrowing is represented by the cold bridge's retained borrowed ID fields. Constructor stores them without reading. It normalizes the default static unknown spelling as the current main constructor does. Nullable, embedded-NUL, mutable/stale region ID behavior stays in the existing main byte emitter; lower core does not duplicate it. No owned string copy is substituted.

The main wrapper owns one bridge receiver and one thin wrapper header. Both real Go allocations are charged with reflect.TypeOf(...).Size before make/new and refunded on unwind/final Close. Runtime core header/directories, allocator core header/location directory, and bitmaps are each charged once at their actual owning layer. Borrowed interface receiver, Budget, streams, source regions and literal/static spelling are NOT double-charged by lower owners. Pointer interface conversion is cold, not a boxed per-cell record. A caller-supplied receiver is externally owned; the test recorder is explicitly caller-owned instrumentation, not an implementation of ostream or a hidden allocator allocation.

Thin wrapper Stats aggregate its actual owned header/bridge charges in OwnerBytes/TotalBytes while retaining core source counters separately. A pending lower owner retains wrapper and IO charges as well: final release waits for the same destructorDone/pending finalization contract. Completed diagnostics are never repeated. A diagnostic error without pending roots can still finalize exactly as the committed allocator does. Main Runtime aliases need no wrapper header unless an actual allocation is introduced.

## Exact existing/new file ownership map

The approved extraction owns exactly the nine existing files below, NEW util_memory_bridge_test.go, twelve new memory Go files and this packet. The frozen Go inventory and SHA256 values are listed in the final section. No additional existing files were edited.

| Existing file | Implemented disposition | New destination/adapter |
| --- | --- | --- |
| util_memory_accounting.go | Move the sole source transition implementation; retain no duplicate body in main | memory/accounting.go, private concrete accounting types |
| util_memory_accounting_test.go | Move core tests/private boundary injection preserving assertions | memory/accounting_test.go |
| util_memory_runtime.go | Move physical/runtime methods; export names and use borrowed static transport | memory/runtime.go; main file becomes constructor/type/error aliases + static error bridge only |
| util_memory_runtime_test.go | Move actual header/physical/counter tests and private failure-boundary injections | memory/runtime_test.go; retain main transport test coverage in NEW util_memory_bridge_test.go |
| util_small_object_allocator.go | Move core config/profile/locator/request/free/reset/close/pending bodies; emit cold IO events | memory/small_allocator.go; main file becomes real forwarding adapter/constructor and reflected wrapper ownership only |
| util_small_object_allocator_vectors.go | Move sole runtime-backed lexical pointer-vector implementation | memory/small_allocator_vectors.go; delete old bodies |
| util_small_object_allocator_consolidate.go | Move sole exact traversal/relink body | memory/small_allocator_consolidate.go; delete old bodies |
| util_small_object_allocator_stream.go | Keep genuine direct CString and NumericTextStream bodies in main; make cold receiver methods | same file, bridge implementation |
| util_small_object_allocator_test.go | Move core18 groups/private boundary injections; keep four real diagnostic/verbosity groups in main, preserve all22 named evidence | memory/small_allocator_test.go + existing main adapter test file |

The four stream/diagnostic groups that stay in main are TestSmallObjectAllocatorBorrowedIDMutationAndWholeCStringWidth, TestSmallObjectAllocatorNullStaleIDErrorPrefixes, TestSmallObjectAllocatorDiagnosticOutputFailureAfterFree, and TestSmallObjectAllocatorVerboseUsesRealRuntimeAndLexicalLifetime. Move the other18 core allocator groups into memory preserving their test names/assertions. Main transport bridge tests additionally preserve existing nominal Z3Error classification checks; relocating core tests to the lower default Failure class is not evidence that the main bridge was exercised.

No edits are needed to util_z3_exception.go/test, NumericTextStream production, GlobalParamCString, arena, mpnum, or command/runtime services in phase A. Current rg dependency audit finds exactly the nine existing files above referring to private runtime/accounting/allocator core types. Parent keeps port_manifest.json, status/progress integration and Git. The approved child extraction relocated main/core tests within the exact file map above.

Implemented additional NEW lower files: memory/failure.go (concrete cold classes/static labels), memory/tokens.go (scalar accessors), memory/allocator_io.go (event/interface declaration and source-positioned dispatch). They contain no duplicate allocation algorithms or main sink substitutes.

## Missing-API tests-first inventory

Six groups in the new memory/shared_runtime_api_test.go:

1. TestMemorySharedRuntimeAllocatorIdentity: two real allocators borrow identical Runtime; each actual pooled chunk creates one raw event. Runtime tokens originate in that same domain; allocator write is read through the actual raw Block, proving shared backing rather than counter-only decoration. Free-list recycle invents no raw free; closing one owner does not free the other's root; all actual charges close to zero.
2. TestMemorySharedSourceProfiles: LP64/ILP32 × release/debug-non-Windows/debug-Windows. Requests1,threshold-1,threshold check physical topology, requested counters, raw counts, source direct release versus retained pooled chunks, final real cleanup.
3. TestMemorySharedTLSLaneGetterAndCumulativeCount: two allocator owners use one actual confined TLS lane; below-threshold getter does not flush. A genuine large direct object crosses100000, and its free causes a negative crossing; count contributes cumulatively3 then6. Chunk teardown leaves unflushed local negative size and nonzero stale global getter while actual physical roots are zero; do not normalize counters to Budget or force a flush.
4. TestMemorySharedRawRefusalRetainsSourcePrefix: actual non-TLS max-size refusal after requested/raw accounting publication, code101, no physical backing created; cold ownership still cleaned.
5. TestMemoryBorrowedStaticErrorAndAllocatorIO: named static transport receives genuine source failures, one borrowed concrete cold recorder sees leak dispatch only after actual chunk frees, constructor/Close retry do not emit twice. This tests event timing, not actual output text; existing four main real byte-stream fixtures remain required.
6. TestMemoryScalarLocationRecords: real Block/ObjectRef pointer-freedom and zero scalar accessors. No template or generic fixture code.

Tests call the actual shared API; there is no test allocator, fake counter, placeholder backend, or invented stdout implementation. Initial RED must be Go API absence on isolated01492fe plus only this test file. Compiler/reference source dependencies are absent from ordinary tests. Production extraction was subsequently approved and implemented under that exact API/source map.

## Later MPZ decisions carried forward

Parent selected separate source-ordered finalization and explicitly documented owning Go Close cleanup. Source finalization must not silently perform direct user-cell frees absent from the C++ destructor. The owning continuation preserves current public Close ownership semantics, with additional genuine real release events distinguished in documentation and tests.

After phase A freezes the shared API, actual default MPZ cell backing and natural embedded16/source expansion topology remain required. Shared-runtime lane/global lifecycle integration and full solver remain required. A permanently optional unused allocator is not an acceptable end state.

## Observed API-absence RED

The isolated `.cache/release/shared-memory-api-red-01492fe` snapshot contains git archive01492fe:golang plus ONLY `memory/shared_runtime_api_test.go`. `go test -v ./memory` exited1 before execution with undefined NewRuntime/RuntimeConfig/NewSmallAllocator/SmallAllocatorConfig/Runtime/AllocatorIOEvent/SmallAllocator/Failure. Log `.cache/shared-memory-api-missing-red.log`. This is genuine API absence, not evidence of an existing production algorithm failing. No tests are claimed RUN/PASS and no fake implementation was added to make compilation succeed.

Test SHA256: `e3443f82d6e5c5792003afad324393f3fa3375ae18c1763e205ddae6ecaa98e6`. That initial RED snapshot leaves existing production/tests untouched. All snapshot/build cache/temp/log writes remain inside golang/.cache. The initial six tests subsequently executed against the approved extraction; the final test SHA differs because two additional meaningful cases were added.


## Production disposition and additional behavioral coverage

The extraction moves the existing accounting transitions, initialized-byte physical runtime, strict chunk cursor, intrusive free-list payload links, lexical runtime-backed pointer vectors, consolidation and pending cleanup bodies to one lower package. Main files no longer contain duplicate core algorithms. Four main byte-stream tests still exercise actual borrowed CString reads and NumericTextStream output. No new stream, arena, mpnum or exception production edits were needed.

The main constructor physically charges three actual allocations in order: wrapper header, cold IO receiver header, then lower allocator header. Source counters exclude these Go metadata allocations. All unwind paths release completed preparations. Pending finalization retains all three owner charges and location roots until explicit cleanup plus final Close. The lower runtime header retains the borrowed transport field within its reflected charge; the allocator header includes the single borrowed IO interface. External callback receivers, Budget, source spelling regions and streams are borrowed, not double charged.

A real constructor budget regression observed old behavior before extraction: failure injection after the old single header allowed constructor success instead of refusing the newly required second/third physical headers. Log .cache/memory-bridge-header-red.log, terminal30237 exit1. The new charged wrapper constructor passes. A separate real behavioral RED found that a configured error transport returning nil suppressed an actual source Memout event and proceeded to allocate. Log .cache/shared-memory-nil-transport-red.log. The corrected transport rejects nil at the actual failure call with ErrFailureTransport, preserving the completed source counter prefix and no physical backing.

Additional lower groups TestMemoryNilTransportCannotSuppressFailure and TestMemoryVectorOverflowRetainsActualRootAndClass exercise nil transport and actual lexical-vector overflow with its retained real runtime root. Additional main groups TestMemoryBridgeDefaultExceptionAndCopiedOwner and TestMemoryBridgePendingFinalizationRetainsAllHeaders cover actual nominal DefaultException transport, copy/closed guards, invalid-profile preflight, and actual partial-root cleanup while every cold owner stays charged. The latter extra fixtures passed immediately against already correct extraction behavior; no manufactured RED is claimed.

## Frozen final inventory and validation

Final snapshot: golang/.cache/release/shared-memory-final-57d0c64, archive57d0c64 plus only the22 owned Go files. This includes committed numeric work through the archive, without an extra numeric overlay. Exact53 test names are recorded in golang/.cache/shared-memory-final-test-names.txt. All41 original names and source assertions remain present. Initial49 native groups passed in .cache/shared-memory-first-green.log (terminal77519); the complete53 groups passed in .cache/shared-memory-named-native.log (terminal99686).

The final full native/debug, exact selected checked-debug race/actual JS, vet and WASI results are all terminal PASS as recorded below. Metadata source-method evidence relocation is parent-owned; this extraction does not promote any source unit or claim global runtime hooks/generated initialization/process exits/full solver or MPZ backend integration complete.

Frozen SHA256 inventory:

| Go file | SHA256 |
| --- | --- |
| util_memory_accounting.go | 0f8b8b949ae7d3486fec70a6f630235bb2094410e7250768e7ace9d7cc496c20 |
| util_memory_accounting_test.go | 0f5ae7537878947856d929411be03ce44abd4a96f7f1e8b40b86fbaa68333499 |
| util_memory_runtime.go | a31dbb41f359a045c0ccdf085aea9cf00d46a39afc424fac48a160177b623218 |
| util_memory_runtime_test.go | 25f49efddbdad18cafa6b995279da85fe2111eba0896bc38058bf870eb6da93e |
| util_small_object_allocator.go | 14bf31034e71a026ffe3da29ae758ac20082fbd92f7c7ca253570fd608ba5080 |
| util_small_object_allocator_consolidate.go | b7a4c869d080abbfd616c67910f64d43ea0e6af48079b457848f03382337c826 |
| util_small_object_allocator_vectors.go | e33be6008aeeab390439b5de9453319d1eea95fd478bffca99509a5db9a66ca0 |
| util_small_object_allocator_stream.go | 88f125ac64cde57b096ab6fe4a1561e96dec9554a133c5ee6e7e8a76d8141327 |
| util_small_object_allocator_test.go | 2e64a2a569d814baac04b3e8352323697af29ca5b2cf7f8311d2418fd7685da0 |
| util_memory_bridge_test.go | fbe7d1e71df01b17fd5e73d2cfd82e4c61b42a288637cec2353424b406b6625c |
| memory/accounting.go | 5033cdf9b26029e5855ade33f35e2e5607beacbb9c2c3a1e61f720c190a183e5 |
| memory/accounting_test.go | a3d2fcac2dc9dc6fc2add6f4662458fa64f87b9584530e820ff643cb8c883bd6 |
| memory/allocator_io.go | 86266a169199d30a7067898d90f659d1348eebef43ea0c6e49418e8fe45c55fa |
| memory/failure.go | 0d1a4af8e9e59616800044afdaf8cb0e25841631214e4602d8a148cb8e12ef06 |
| memory/runtime.go | 6b5cb306372bbdacdd8a928be7c027bd3aeab322c827fd20abc2e1651b280854 |
| memory/runtime_test.go | 5b05fe8ee6534d3db3416f12e47c54b583565f4da477f546c94e2037b1bcf339 |
| memory/shared_runtime_api_test.go | 9a8264dc9d7b813627993cc400425ad99af017eeaa1b552fd7629d087e91386a |
| memory/small_allocator.go | 2ac8ec32e52f04af98ee1baa8a0f40b5f8fa3ac8d150efaf33051835d2275497 |
| memory/small_allocator_consolidate.go | b5df93336b40557ce9c78810a431221f1cfa2501d33edb1c861bfc9fd40a0714 |
| memory/small_allocator_test.go | 3e4827416ab94bdd6ab2c9cf67d913a284e96b3880d8c1f70ab010302c71d326 |
| memory/small_allocator_vectors.go | fde61f4b738a7a53807cc92af5b0d7a5da9eb51bd059a02df44f2dacf8f1010c |
| memory/tokens.go | d67c5af756aef520f4d793d87848df7f2ba0e3c5fa36118f512503e89d02271c |


## Terminal final gates and scope handoff

All gates use the frozen57d0c64 archive plus exactly22 Go files, normal workspace compiler environment, and caches/temp/logs only under golang/.cache:

| Gate | Terminal handle | Result | Log |
| --- | --- | --- | --- |
| Full native go test -count=1 ./... | 44867 | PASS, exit0 | .cache/shared-memory-final-native.log |
| Full g3checked+g3debug go test -count=1 ./... | 47215 | PASS, exit0 | .cache/shared-memory-final-debug.log |
| Selected g3checked+g3debug race . ./memory | 85524 | all53 named RUN/PASS, exit0 | .cache/shared-memory-final-race.log |
| Selected actual JS g3checked+g3debug . ./memory | 65799 | all53 named RUN/PASS, exit0 | .cache/shared-memory-final-js.log |
| Full g3checked+g3debug vet ./... | 18702 | PASS, exit0 | .cache/shared-memory-final-vet.log |
| Full g3checked+g3debug WASI build ./... | 55930 | PASS, exit0 | .cache/shared-memory-final-wasi.log |
| Complete frozen tree CheckNoGenerics AST audit | terminal command | PASS, exit0 | .cache/shared-memory-final-nogen.log |

The race and JS logs contain exactly53 top-level names, each present in both RUN and PASS sets. Independent SHA verification confirms all22 workspace files and snapshot files still match the frozen inventory. Original41 named bodies retain their original fatal/error assertion counts; nominal lower error classification changes use actual Failure while the main bridge retains real Z3Error/DefaultException checks. No source method is newly promoted by moving it. Parent owns source manifest/evidence relocation and final source-hash audit; no central ledger or Git changes were made by this packet.

Final selected name inventory:

```text
TestMemoryAccountingBoundedIndependentLaneModel
TestMemoryAccountingDeallocationClampAndWatermark
TestMemoryAccountingDeallocationPhysicalBoundary
TestMemoryAccountingGlobalCounterPrefixes
TestMemoryAccountingSignedBoundaryPrefixes
TestMemoryAccountingSourceSizeTPromotions
TestMemoryAccountingTLSRepeatedCountAndLimitOrder
TestMemoryAccountingTLSStrictThresholds
TestMemoryBorrowedStaticErrorAndAllocatorIO
TestMemoryBridgeColdHeadersAndConstructorUnwind
TestMemoryBridgeDefaultExceptionAndCopiedOwner
TestMemoryBridgeNominalOOMAndCountErrors
TestMemoryBridgePendingFinalizationRetainsAllHeaders
TestMemoryNilTransportCannotSuppressFailure
TestMemoryRuntimeActualBlocksHeadersAndReallocate
TestMemoryRuntimeActualGettersAndUninitializedBytes
TestMemoryRuntimeFailedReallocateKeepsActualOldBlock
TestMemoryRuntimeImmediateFaultGraphAndOwnerDomains
TestMemoryRuntimeIndependentPhysicalAndCounterModel
TestMemoryRuntimeLimitBeforePhysicalAndTLSOwnedFailure
TestMemoryRuntimeReallocateGenerationRefusalPrefix
TestMemoryRuntimeShrinkRegrowByteLifetime
TestMemoryRuntimeSourceUnsignedDeltaWatermarkAndGuards
TestMemoryRuntimeTLSStrictThresholdAndRepeatedCounts
TestMemoryRuntimeWrappedHeaderActualPhysicalPrefix
TestMemoryScalarLocationRecords
TestMemorySharedRawRefusalRetainsSourcePrefix
TestMemorySharedRuntimeAllocatorIdentity
TestMemorySharedSourceProfiles
TestMemorySharedTLSLaneGetterAndCumulativeCount
TestMemoryVectorOverflowRetainsActualRootAndClass
TestSmallObjectAllocatorBorrowedIDMutationAndWholeCStringWidth
TestSmallObjectAllocatorColdFailureSweepAndResetLedger
TestSmallObjectAllocatorConsolidateLateFreeRetainedRoots
TestSmallObjectAllocatorConsolidateLexicalScratchAndFailure
TestSmallObjectAllocatorConsolidateReclaimsRemainderClass
TestSmallObjectAllocatorConstructorAndMetadataFailurePrefixes
TestSmallObjectAllocatorDiagnosticOutputFailureAfterFree
TestSmallObjectAllocatorDirectCallerLifetimeClose
TestSmallObjectAllocatorGenerationExhaustionLedger
TestSmallObjectAllocatorIndependentModelWarmPlateau
TestSmallObjectAllocatorIntrusiveLinksAndPreservedBytes
TestSmallObjectAllocatorLexicalFailedFreeRoots
TestSmallObjectAllocatorNullStaleIDErrorPrefixes
TestSmallObjectAllocatorPendingCompletedRawRoots
TestSmallObjectAllocatorPooledDuplicateFreePrefix
TestSmallObjectAllocatorProfilesThresholdsAndHeaders
TestSmallObjectAllocatorRegisteredPrintOnlyReplay
TestSmallObjectAllocatorRequestedPrefixBeforeRealRawRefusal
TestSmallObjectAllocatorResetRetainsDirectThreeSourceProfiles
TestSmallObjectAllocatorScalarRefAuditAndPlacementDeletes
TestSmallObjectAllocatorStrictChunkEndpointConsolidateQuirk
TestSmallObjectAllocatorVerboseUsesRealRuntimeAndLexicalLifetime
```

READY: Go production/tests are frozen for parent review/integration. This phase establishes the genuine reusable shared runtime and allocator, plus charged nominal-main bridges. Source numerical cell backend replacement, natural embedded16 layout, generated initialization/finalization, full participating typed-owner hooks, native process IO/exits and full solver remain required follow-on work. Source counter algorithms are actual existing runtime bodies; Budget reservations are never substituted for malloc/free events.


## Parent integration verification

Parent snapshot `.cache/review/shared-memory-final-57d0c64` independently
archives committed57d0c64 and overlays the exact22 frozen Go files, plus the
reviewed ledger/progress. All24 live/frozen hashes agree. Normalized body review
finds identical accounting algorithms and tests, runtime changes limited to
static error transport, and unchanged allocator traversal/cleanup algorithms.
All41 original test names and their fatal/error assertion counts are preserved.

Full native14577 and checked/debug27555 exited0. Exact53 selected groups each
RUN/PASS once under checked/debug race18206 and actual JS39397, with the eight
main and45 lower groups both executed. Vet39848, full WASI build23412, and
separate root48859, memory and mpnum69266 test compilations exited0. WASI is
compilation evidence; actual WASI runtime execution remains open. Parent logs
are `.cache/shared-memory-parent-{native,debug,race,js,vet,wasi-*}.log`.

Integrated source/evidence/no-generics audit95986 exited0, using the exact
relocated method targets and both real main bridge and lower-core test packages:
2102 units,677 APIs,33ported/145partial/1924unported. Source fingerprint remains
`f8480997c14e2d6c414f40c60ce5095ef40a89ce6d1bcf1ed196c5d682d42d8f`.
No whole source unit is promoted. Parent owns the final exact-index commit;
MPZ/natural default backend and the full solver/proof/interpolation objective
remain active.
