# AST dependency owning alias packet

Source: ast.h2232–2234, complete obj_ref.h, ref_vector.h, ref_buffer.h, their concrete pointer vector/buffer algorithms, and actual AST dependency INC/DEC callbacks. This packet closes the selected dependency owning aliases, not the generic templates or their other configurations. Source Microsoft notices will be preserved in translated files. No generic Go, unsafe, per-entry Go pointers, hidden slices, or substitute callback owners.

## Concrete ownership and storage

All public owners are distinct pointer-free scalar tokens qualified by the containing ASTManager. Explicit construction/copy/move/destruction is mandatory; Go shallow copy is prohibited. Manager methods use the existing lock and existing charged allocator services. No manager fields, existing core headers, constructor, storage files, or teardown methods need edits: actual owned records/backing live in the already budgeted allocator graphs, and force-close has the existing AST teardown/bulk-reclamation policy.

`ASTDependencyRef` owns a16-byte encoded cold record containing dependency slot/generation plus record validation fields. It borrows the ASTManager by scalar provenance, not a heap owner pointer. A default/null obj_ref is a constructed record holding null, distinct from an invalid unconstructed Go token. Constructor allocation is an explicit Go locator adaptation before the source INC; copy allocates another real record then INC, move allocates a record and transfers without INC/DEC. Get/bool/equality copy the raw pointer token and do not invent a dependency dereference. Actual dereference/query remains on the dependency manager. Steal transfers one retained reference to the returned raw dependency without DEC; caller must eventually DEC it.

`ASTDependencyRefVector` owns a24-byte encoded cold record, including actual cleanup target/cursor words, source-geometric backing0→2→3→5..., and8-byte slot/generation entries. Reuse existing fixed-width AST ref-vector geometry/byte codecs only, never its AST-node callbacks. Selected source header8 versus actual charged header12 includes initialized high-water. Growth preserves ALL old capacity bytes and initialization history; no INC/DEC during relocation. Copy constructor starts empty and appends geometrically, unlike raw vector exact-capacity copy. `set(other)` resets then appends, self no-op; public Go `Assign` denotes this source set method because ordinary copy assignment is private. Move transfers backing only. All pointer-returning Get/Back/element/iterator adapters copy retained raw dependency tokens; actual dependency dereference validates generation separately.

`ASTDependencyRefBuffer` owns an actual160-byte encoded record:32 bytes of state/initialization metadata and the actual inline16×8-byte dependency entries. This is charged physical Go storage, distinct from the selected source LP64 wrapper/header plus128-byte inline area. No phantom separately allocated inline block or per-buffer Go slice/header. Overflow backing is in the borrowed vector allocator with actual12-byte metadata plus doubled pointer capacity16→32→64.... The source capacity doubles uint32, and selected actual byte/host/allocator guards diagnose unsafe wrapped allocation before writing. Source buffer growth copies ONLY logical m_pos entries, so overflow initialized extent becomes that logical prefix; it does not preserve inactive dynamic tail. The original inline area and its separately recorded initialization extent survive growth/finalize until the owner record is destroyed. Finalize restores inline16, size0, leaving actual retained inline bytes intact. Reset keeps the current backing. The intended copy constructor starts inline16 and appends, but the preserved ref_buffer constructor is uninstantiable under conforming dependent-base lookup (see diagnostic below). Source buffer assignment resets then appends. No buffer move constructor is invented (the source user copy constructor/destructor prevents one).

Scalar element and iterator borrows bind to actual owner/physical backing and captured end; no cold pointer-return object or runtime heap allocation. Vector mutable element_ref assignment is INC-before-DEC and debug rejects null at its actual SASSERT. Buffer mutable back/c_ptr returns T*& rather than a smart proxy: expose a clearly named raw borrow-write route with NO callbacks, alongside ordinary callback-correct Set, without claiming that raw mutation retains ownership. A borrowed raw pointer write can create a source-invalid lifetime; later callbacks validate at actual dereference.

## Exact method/API inventory

All methods below are on ASTManager. Source manager accessors return the existing containing manager after token checks. `DependencyRefGet` returns `(ASTDependency,error)`; size APIs return `(uint32,error)`; predicate APIs return `(bool,error)`; mutations return error. Constructor/copy/move return `(owningToken,error)`.

- Object: `NewDependencyRef(value ASTDependency)`, `CopyDependencyRef(src)`, `MoveDependencyRef(src)`, `DependencyRefGet`, `DependencyRefPresent`, `DependencyRefEqual`, `DependencyRefManager`, `DependencyRefSet(dst,value)`, `DependencyRefAssign(dst,src)`, `DependencyRefReset`, `SwapDependencyRefs`, `DependencyRefSteal`, `DelDependencyRef(*owner)`. Null constructor takes a null dependency. No ordinary pointer dereference substitute is introduced.
- Vector: `NewDependencyRefVector`, `NewDependencyRefVectorFrom([]ASTDependency)`, `CopyDependencyRefVector`, `MoveDependencyRefVector`, `DependencyRefVectorSize/Empty/Capacity/Manager`, `Get/GetDefault/Back`, `Set`, `Push/Pop`, `Resize/ResizeFill/Reserve/Shrink`, `Reset/Finalize`, `Erase/EraseValue/Contains`, `Append/AppendValues`, `Assign` (source set), `SwapElements/Reverse`, `SwapDependencyRefVectors`, `DelDependencyRefVector`. Method names have DependencyRefVector prefix except constructor/copy/move/swap/destructor.
- Vector borrows: `DependencyRefVectorElement`, `DependencyRefElementGet/Set/Assign/Equal`; `DependencyRefVectorBegin`, `DependencyRefVectorNext`. Concrete element/iterator scalar types. Pointer-vector c_ptr iteration is represented by these checked captured-end borrows, not an escaped Go slice.
- Buffer: `NewDependencyRefBuffer`, `DependencyRefBufferSize/Empty/Capacity/Manager`, `Get/Back`, `Set/SetX`, `Push/Pop`, `Resize/Shrink`, `Reset/Finalize`, `Append/AppendValues`, `Assign`, `DelDependencyRefBuffer`. No source-absent Reserve, ResizeFill, move/swap/erase helpers will be advertised for this alias.
- Buffer borrows: `DependencyRefBufferBegin/Next`, `DependencyRefBufferElement`, `DependencyRefBufferBackBorrow`, `DependencyRefBufferBorrowGet/WriteRaw`. Source raw back maps to DependencyRefBufferBackBorrow with its actual debug nonempty assertion; Element is a physical c_ptr offset without a logical-size assertion. Raw reference-return adapters do not INC/DEC.

Vector pointer equality helper can compare sequence values. Source ref_vector_ptr_hash requires dependency::get_id, absent from dependency.h, so this specialization is uninstantiable; do not invent a hash. Other generic ref_util map/collection callback loops and alternate buffer sizes/configurations remain open unless a real consumer requires them.

## Arithmetic, aliases, failures, and publication

INC precedes old DEC in obj_ref assignments and vector/buffer Set, including self-assignment. Incoming INC stays completed if an old-pointer lifetime/debug guard refuses. No rollback of a completed source callback. Source DEC0 debug refusal reports not consumed; a consumed deletion callback worklist refusal propagates after the real finite graph cleanup. Following a consumed DEC, publish replacement/reset/empty owner before returning its error. Destructor-range safety continuation must finish or preserve all actual roots; no lost unvisited references or ignored callback error. Clearly distinguish this Go cleanup continuation from source exceptions/assertions that abandon C++ control flow.

Push does dependency INC BEFORE storage growth. Allocation refusal therefore leaves that INC completed and not stored in the container; tests expose it and recover it explicitly. Constructor unwinding cleans stored entries but cannot silently undo the last source failed-push INC. Dynamic growth allocations preserve source old+new peaks; physical retained allocator growth remains charged.

Vector retained Get/Set/back use source debug logical assertions and actual physical/initialized guards only. Release increasing Shrink dec_range(begin+n,end) is empty; null backing is no-op, and initialized retained capacity can publish increased size without INC. Erase shifts and decrements size before DEC; it leaves raw trailing bytes. No DEC of inactive tail on reset/destruction. Nonempty vector self-Append is source growing-size nontermination, explicitly guarded before body. Empty self-Append is valid.

Buffer Shrink calls Resize after its debug assertion: release increasing Shrink genuinely appends null values to the new size, unlike vector. SetX preserves idx+1 uint32 wrap, then actual source Set INC-before-read/write order and typed unsafe access guard. Buffer Append captures source size and c_ptr ONCE. Finite same-buffer append is valid while captured backing survives. Inline backing survives a switch to overflow storage; dynamic captured backing is freed on subsequent expansion and guarded only at the next actual captured read, retaining prior appended values/INC prefixes. No blanket self-append rejection and no rereading growing size.

## Source tests and test-first gates

No registered direct dependency ref/ref-vector/ref-buffer assertions were found in src/test. src/test/buffer.cpp uses ptr_scoped_buffer<point> and its different delete callbacks, so its assertions are NOT claimed here. src/test/vector.cpp uses svector<int>, including a final allocation-until-failure campaign; it is not a dependency-owner assertion port. Actual consumers include macro_finder/macro_manager/quasi_macros and tactic dependency vectors, expr_replacer/th_rewriter/tactic dependency refs. No ref-buffer consumer outside its typedef was found. Their full algorithms remain separate goal obligations.

Complete API route tests preceded implementation with missing-API RED in isolated committed HEAD plus only owned tests. Added finite exact reference-count models/shared dependency DAGs; actual AST callbacks, constructor/copy/move/steal/assignment, scalar null shape and raw retained reads; vector versus buffer copy/growth differences; captured self-buffer append inline and dynamic relocation prefixes; debug source assertions versus release finite retained behavior; all immediate allocator roots/counter prefixes BEFORE rendering or Close; every meaningful growth/refusal point, scalar byte/uint32 boundaries, warm no-allocation operations and escaping iterator values. Actual source callback paths only, no invented callback fixtures. Full native/debug/race/actual JS named tests/vet/WASI/no-generics and immutable hashes before handoff.

## Owned files

NEW only: `ast_dependency_ref.go`, `ast_dependency_ref_vector.go`, `ast_dependency_ref_vector_ops.go`, `ast_dependency_ref_vector_access.go`, `ast_dependency_ref_vector_borrow.go`, `ast_dependency_ref_buffer.go`, `ast_dependency_ref_buffer_ops.go`, `ast_dependency_ref_buffer_borrow.go`, `ast_dependency_ref_test.go`, `ast_dependency_ref_vector_test.go`, `ast_dependency_ref_buffer_test.go`, `ast_dependency_ref_failure_test.go`, and this document. Additional new ast_dependency_ref-prefixed scalar helper/test file only if factoring requires it. Existing core/allocator/ref/dependency/proof/SAT/metadata/Git files remain unchanged. The concrete API/layout contract and corrections are implemented tests-first.


## Preserved uninstantiable buffer copy constructor

Actual `g++ -std=c++17 -fsyntax-only -I src` with real util/debug.h and util/ref_buffer.h, a concrete INC/DEC manager, and `ref_buffer<int,manager> b(a)` rejects ref_buffer.h162: unqualified `append(other)` is not found in its dependent base; the diagnostic explicitly requires `this->append`. This is a compiler lookup defect, not a missing Go prerequisite. Fixture `.cache/dependency-ref-audit/ref_buffer_copy.cpp` stays under golang. Do not claim the preserved constructor executed or silently fix C++.

CopyDependencyRefBuffer is omitted: no intended-source repair adapter is authorized or required by an actual consumer. Ordinary inherited assignment remains executable and differs from this constructor lookup. Buffer owning move remains absent. Source specialization/template status stays partial for this constructor and nonselected configs. The real same-header buffer assignment instantiation `.cache/dependency-ref-audit/ref_buffer_assign.cpp` compiles successfully; this route is selected.


## Interrupted callback cleanup and actual publication

A vector charges 24 cold bytes (six uint32 words), not the earlier 16-byte sketch. Actual backing still charges its 12-byte header and eight bytes per entry. Buffer uses its already charged metadata words20/24, retaining160 total bytes and inline offset32. Target plus an active flag and next-unconsumed index form a pointer-free continuation. Actual maximum capacity floor((UINT32_MAX−12)/8) is less than2^29; a scalar boundary test proves the active bit31 cannot overlap any permitted target.

Range destruction retains source size and raw slots while recording only DEC callbacks actually consumed. A refused callback preserves current and suffix roots; retry resumes at the first unconsumed callback. Consumed-error cleanup continues over the finite range, and a later requested shrinking phase completes before the first error is returned. Move and whole-owner swap transfer progress. Structural mutation while cleanup is pending returns ErrASTDependencyCleanupPending; Reset/Resize/Del can resume. This is a Go continuation for source-undefined interrupted destruction, not C++ normal-path semantics or rollback. Existing manager force-Close bulk reclaims actual pending records/backings; tests prove live records and charged bytes refund without callback replay.

Constructor cleanup refusal also preserves a visible owner: NewDependencyRefVectorFrom and CopyDependencyRefVector return the original construction/input error together with a nonzero charged token only if cleanup cannot consume all owned roots. Caller must recover the offending source counter and retry Del. Fully consumed constructor cleanup returns the zero token. This is an explicit exception-publication adapter to C++ failed construction, without undoing source INC prefixes or hiding orphaned local roots.

Vector Capacity and buffer Capacity/Manager are added inspection APIs, not claimed source accessors. Buffer growth copies only logical entries; a private forced-growth test observes this policy, while ordinary public growth occurs at full capacity. Mutable buffer physical Element creation lacks a logical assertion, whereas BackBorrow preserves source !empty. Equal-size vector ResizeFill performs no action and never validates its unused value. Reference getters copy retained raw pointers; a subsequent dependency query performs lifetime validation.

## Test-first correction evidence

Initial missing-API isolated RED92032 preceded translation. Actual range-refusal and unused ResizeFill behavioral RED8198 preceded the continuation and no-action fixes; resumed consumed-error RED65575 preceded complete lexical cleanup. Constructor refusal RED53373 and copy-constructor refusal RED61371 preceded partial-owner publication. Targeted suites use actual dependency callbacks and counters, not synthetic callback hooks. There are zero registered direct dependency-alias source assertions claimed; tests cover concrete source action routes and independently finite count/ownership models. Full generic helper configurations, uninstantiable buffer copy/hash, and actual consumer algorithms remain open.


## Final frozen verification (2026-10-03)

Fresh committed HEAD aadb037:golang plus ONLY these twelve new Go files, isolated at `.cache/release/ast-dependency-ref-corrected`. Unrelated active drafts are excluded. No existing Go files, manager headers, central metadata, or Git files were edited by this packet.

- Full native `go test -count=1 ./...`: terminal80340 PASS; g3 2.501s, mpnum23.240s.
- Full actual `g3checked,g3debug` suite: terminal44715 PASS; g3 2.830s, mpnum23.851s.
- Exact30 named checkeddebug race tests: terminal43795 PASS,1.357s.
- Exact30 named actual checkeddebug JS tests through `.cache/js-exec`: terminal68087 PASS,.297s. Both race and JS logs were checked for exact equality of expected names, RUN names and PASS names.
- Full checkeddebug vet terminal6256 PASS.
- Checkeddebug WASI test binary and full package build terminal57189 PASS.
- Source-ledger/no-generics audit terminal50630 PASS. This checks the committed ledger baseline, not an invented completed-template claim.

Logs: `.cache/dependency-ref-corrected-{native,debug,race,js,vet,wasi,audit}.log`; named inventory `.cache/dependency-ref-corrected-names.txt`. Final Go hashes are recorded below and in `.cache/dependency-ref-corrected-sha256.txt`. Documentation is finalized after terminal gates; Go bytes are unchanged from their gated snapshot.

```
f6add0f97d695632cc3908971d402ea2c2cde74f94ee05614c24365b37afc47c  golang/ast_dependency_ref_buffer_borrow.go
10c93111e75a067bea1904d112c2975c926c69a9f11fa9a2d2caf207134fc9f1  golang/ast_dependency_ref_buffer.go
74ff4479f27b0c3afa87808ea4eb5d1631e22988f41a9a246cd485580e1c6b15  golang/ast_dependency_ref_buffer_ops.go
78643819e9b028257ba22b33414d5beecfc3d3a53df9ccd2c341fe74bc563be9  golang/ast_dependency_ref_buffer_test.go
3619103889e192547bef4a9df104ca7537c69f26f54952b2a15abfb58d63981e  golang/ast_dependency_ref_failure_test.go
d2e914cac9a5b3dc13c1dbc1b830f6e2d5d32e772cdf8527f16e5448e451e2e8  golang/ast_dependency_ref.go
d564837878e9bf60ad647005fbe3582a779a8fc640afbf3e43396bb36b867131  golang/ast_dependency_ref_test.go
241ec7a4d00cebe9080ffd19d9ed9920c17a6c66e68e24e8984859c122d58494  golang/ast_dependency_ref_vector_access.go
4a6091d8e7a5e1ec1e02a5e4f609aa965371c91b14fe519eec95e7af58df01c3  golang/ast_dependency_ref_vector_borrow.go
5f01298d9682b71adcb1e964a31200a9fabf7dd8c3fd34ead83abef843951a26  golang/ast_dependency_ref_vector.go
ef8eac5d2d3f1b81a998fa6bfe60d3e61255108446ec94469bbe82717e478f5a  golang/ast_dependency_ref_vector_ops.go
6ef2fab9825e753ed3395bb43e16ed3c232dedec07da8d42ea99a0bb7ebc477a  golang/ast_dependency_ref_vector_test.go
```

Exact named test inventory:

```
golang/ast_dependency_ref_vector_test.go:func TestASTDependencyRefVectorSourceCallbacksAndFullRoutes(t *testing.T) {
golang/ast_dependency_ref_failure_test.go:func TestASTDependencyRefOwnersPointerFreeAndDistinctStorage(t *testing.T) {
golang/ast_dependency_ref_failure_test.go:func TestASTDependencyRefVectorAndBufferGrowthCopyDifferentTails(t *testing.T) {
golang/ast_dependency_ref_failure_test.go:func TestASTDependencyRefBufferCapturedDynamicSelfAppendPrefix(t *testing.T) {
golang/ast_dependency_ref_failure_test.go:func TestASTDependencyRefVectorVsBufferIncreasingShrinkSourceProfiles(t *testing.T) {
golang/ast_dependency_ref_failure_test.go:func TestASTDependencyRefSourcePushRefusalRetainsIncrement(t *testing.T) {
golang/ast_dependency_ref_failure_test.go:func TestASTDependencyRefRawBufferBorrowDoesNotInventCallbacks(t *testing.T) {
golang/ast_dependency_ref_failure_test.go:func TestASTDependencyRefBufferInlineBorrowSurvivesFinalizeAndGrowth(t *testing.T) {
golang/ast_dependency_ref_failure_test.go:func TestASTDependencyRefVectorDebugCounterRefusalKeepsOwnedBacking(t *testing.T) {
golang/ast_dependency_ref_failure_test.go:func TestASTDependencyRefVectorConsumedCallbackErrorReclaimsLexicalOwner(t *testing.T) {
golang/ast_dependency_ref_failure_test.go:func TestASTDependencyRefWarmCallbacksAndEscapingBorrowValuesAllocateZero(t *testing.T) {
golang/ast_dependency_ref_failure_test.go:func TestASTDependencyRefFiniteIndependentContainerCounterOracle(t *testing.T) {
golang/ast_dependency_ref_failure_test.go:func TestASTDependencyRefVectorConstructorFaultSweepImmediateOwnership(t *testing.T) {
golang/ast_dependency_ref_failure_test.go:func TestASTDependencyRefBufferScalarCapacityBoundaries(t *testing.T) {
golang/ast_dependency_ref_failure_test.go:func TestASTDependencyRefVectorPointerComparisonsDoNotDereferenceAbsentDeadValue(t *testing.T) {
golang/ast_dependency_ref_failure_test.go:func TestASTDependencyRefAssignmentDebugRefusalRetainsSourceIncomingIncrement(t *testing.T) {
golang/ast_dependency_ref_failure_test.go:func TestASTDependencyRefRetainedCopyOnlyGetAndPhysicalBorrowOrdering(t *testing.T) {
golang/ast_dependency_ref_failure_test.go:func TestASTDependencyRefBufferSetXWrappedResizeBeforeIncrement(t *testing.T) {
golang/ast_dependency_ref_failure_test.go:func TestASTDependencyRefVectorGrowingSelfAppendGuardAndEmptyFastPath(t *testing.T) {
golang/ast_dependency_ref_failure_test.go:func TestASTDependencyRefBufferRawPointerCreationHasNoLogicalAssertion(t *testing.T) {
golang/ast_dependency_ref_failure_test.go:func TestASTDependencyRefRangeRefusalResumesAfterConsumedPrefix(t *testing.T) {
golang/ast_dependency_ref_failure_test.go:func TestASTDependencyRefVectorResizeFillUnusedDeadValueIsNoAction(t *testing.T) {
golang/ast_dependency_ref_failure_test.go:func TestASTDependencyRefCleanupMetadataBoundAndProgressTransfer(t *testing.T) {
golang/ast_dependency_ref_failure_test.go:func TestASTDependencyRefPendingForceManagerCloseReclaimsPhysicalGraph(t *testing.T) {
golang/ast_dependency_ref_failure_test.go:func TestASTDependencyRefResumedConsumedErrorStillFinishesRequestedDestruction(t *testing.T) {
golang/ast_dependency_ref_failure_test.go:func TestASTDependencyRefConstructorRefusedCleanupPublishesOwner(t *testing.T) {
golang/ast_dependency_ref_failure_test.go:func TestASTDependencyRefCopyConstructorRefusedCleanupPublishesOwner(t *testing.T) {
golang/ast_dependency_ref_test.go:func TestASTDependencyRefActualOwnerCopyMoveStealAndAssignment(t *testing.T) {
golang/ast_dependency_ref_buffer_test.go:func TestASTDependencyRefBufferInlineSourceRoutesAndCapturedAppend(t *testing.T) {
golang/ast_dependency_ref_buffer_test.go:func TestASTDependencyRefBufferMutableBackUsesSourceAssertion(t *testing.T) {
```

## Coordinator integration

All twelve live Go files matched both the immutable SHA inventory and the final
corrected snapshot. Prospective tree .cache/review/ast-dependency-ref-integration-aadb037
contains the SAME committed HEADaadb037 and only those twelve files, plus final
metadata. The full/native/debug gates above therefore cover the exact prospective
Go tree. The coordinator independently verified names==RUN==PASS for every one
of30 race and actual JS groups, and rechecked all twelve hashes before commit.
Final source/no-generics audit PASS; ledger33ported/88partial/1981unported.
Only ref_buffer.h becomes newly partial; obj_ref.h/ref_vector.h/ast.h gain
selected alias inventories, with whole-source completion flags still false.
Unrelated modular arithmetic, global registry/configuration proposals and the
full solver's intentionally missing-API fixture are excluded from this committed
packet. No claim of full live-draft workspace GREEN is made.
