# Concrete expression substitution packet

Sources independently read in full: `src/ast/expr_substitution.h/.cpp`, `util/obj_hashtable.h`, selected `core_hashtable` algorithms in `util/hashtable.h`, `ref_util.h` map callback loops, and the actual pointer/unsigned vector specializations. Microsoft notices are preserved. No Go generics, unsafe, production math/big, substituted callback manager, Go map, per-entry interface, or unaccounted heap backing.

This selects all non-display methods of expr_substitution and scoped_expr_substitution. Both display methods remain OPEN: mk_pp delegates to actual mk_ismt2_pp/ast_smt2_pp, its formatting environment and plugins, none of which is replaced by a handle-number printer. The selected ref_util map key/value and map-value DEC loops are concretely translated through actual AST/dependency callbacks; other generic hashtable/ref utilities, compiler-generated unsafe copying of these owners, rewriters, tactics, SMT/proof consumers, and whole AST remain separate obligations.

## API and actual storage

All methods are on the real ASTManager under its existing mutex. Public pointer-free ASTExprSubstitution/ASTScopedExprSubstitution tokens contain manager origin and an actual arena.SmallBytesRef. Shallow copying is prohibited. No existing manager header, field, allocator, constructor or teardown file changes. Existing charged allocator services own every record/backing; source manager borrows do not create fictitious extra manager charges.

- Constructors: NewExprSubstitution(), NewExprSubstitutionCores(bool), NewExprSubstitutionPolicy(cores,proofs bool).
- Inspectors: ExprSubstitutionManager/ProofsEnabled/UnsatCoreEnabled/Empty/Contains.
- Mutations: ExprSubstitutionInsert/Erase/Reset/Cleanup and DelExprSubstitution(*token).
- Finds: ExprSubstitutionFindProof(token,key,*def,*proof) and FindDependency(token,key,*def,*proof,*dependency), returning (bool,error). Required output pointers are nonnil. Misses and disabled policies preserve existing caller values. Go output aliases retain source sequential writes. Raw returned handles are borrowed, not INCed.
- Scoped constructor/destructor: NewScopedExprSubstitution(token), DelScopedExprSubstitution(*token).
- Scoped methods: ScopedExprSubstitutionInsert/Reset/Push/Pop/ScopeLevel/Empty/FindValue/FindProof/FindDependency/Contains/Cleanup. FindValue returns the original key only on a miss; a present NULL definition returns NULL.

Primary owner charges **128 actual encoded bytes**, not a guessed C++ sizeof. Map descriptors occupy offsets0/32/64, each32 bytes: payload slot/gen0/4, capacity8, size12, deleted16, reset cursor20, reserved phase24, availability/finalize marker28. Policy96, magic100, pending mutation104, reserved cursor108 and reserved locator words112–124 finish the real allocation. Reserved words are physically initialized storage, not claimed extra callback roots. Map entries charge **24 actual bytes**: key slot/gen0/4, value slot/gen8/12, FREE/USED/DELETED state16, callback-ownership bits20. Source LP64 obj_map entries are two pointers/16 bytes; encoded generations/state/cleanup ownership are an explicit width adaptation. Table backing is exactly capacity×24, without an invented header.

Scoped owner charges **48 bytes**: borrowed substitution locator0/4; actual ASTRefVector record locator8/12; unsigned-limit payload16/20; size24, capacity28, initialized32; operation36, callback cursor40, magic44. The real existing ASTRefVector cold record/backing is allocated separately and charged by its existing services. Limit backing charges actual12-byte header plus capacity×4; source header8 versus Go initialized-highwater metadata12 is explicit. All source limits geometry2→(3n+1)/2, uint32 byte-overflow checks, full-capacity trivial relocation and old/new allocation peaks are retained. Actual byte/host/uint32 arena guards are additional representational-domain boundaries.

Constructors create primary capacity8 BEFORE the explicit proof-policy debug assertion, then optional proof map before dependency map. Policy snapshots the real manager proof mode; subsequent toggles do not change the substitution. The explicit proof=>manager-proof assertion is debug-only. Disabled optional inputs are ignored without provenance/lifetime checks; used inputs enforce manager provenance. Optional descriptors are embedded in the actual128-byte owner (physical-layout adaptation); only enabled table backings are constructed.

## Source algorithms and publication

Three maps have distinct actual physical histories. Hashing reads real AST structural hashes; pointer equality compares encoded slot/generation. Exact pre-insert threshold `(size+deleted)<<2 > capacity*3`, expansion doubling, linear probing with wrap, last encountered tombstone before a FREE slot, and ascending source rehash order are translated. Rehash allocates/moves before deleting/publishing old storage. Stale-key hash guards diagnose source-undefined reads and free only the unpublished nonowning temporary; they do not undo callbacks.

Removal marks FREE when the next bucket is FREE, otherwise DELETED. Rebuild occurs only when deleted>size and deleted>64, and is skipped on the actual bound source-memory OOM predicate `m.exprIDs.outOfMemory`. Reset counts preexisting FREE slots, marks entries FREE, and may delete/halve/reallocate a sparse table; that source half-capacity publication precedes allocation. Finalize instead deletes then allocates64 THEN publishes64: refusal preserves old capacity with nil backing and an explicit pending-finalize marker. Recovery allocation is a Go undefined-continuation adapter. Prior bindings remain retryable after primary expansion refusal before callbacks.

Insert uses **definition==NULL**, not “new key”, to select its source new-entry branch. Source key INC precedes definition INC/publication. Optional new-map insertion publishes its raw pointer before INC. Replacement INCs new before DEC old, then publishes value; no staging/whole-substitution rollback. Disabled maps do nothing. Repeated NULL definitions retain the source repeated key INC, optional old-value overwrite without DEC, and exact debug contains assertion after primary publication. Extra unlisted INCs are not silently reclaimed as invented map multiplicity. Tests explicitly recover these source prefixes.

Erase processes proof, dependency, then primary key and definition, with source erase/hash afterward. An unpinned primary key can be freed before that final source hash read; the typed lifetime guard retains completed DEC/entry prefixes instead of dereferencing ended storage. A separately pinned key (including the scoped trail) makes ordinary erase valid. Reset is primary key+value per ascending bucket, then proof values, then dependency values. Actual destructor executes source reset including sparse reallocation before reverse dependency/proof/primary member storage destruction.

## Callback errors and Go lifetime continuation

Real AST INC/DEC and dependency INC/DEC are used. Per-entry ownership bits advance only when callbacks consume their references. A refused callback preserves the current and later roots; successful consumed callbacks are never replayed. Consumed deletion-worklist errors finish the **same entry**, then Reset stops before later entries/tables, preserving their roots. Retry starts at the next unconsumed action. Destructor can complete finite remaining lexical cleanup and refund the record before returning the first consumed error; on refusal it preserves the owning token. These are explicit Go continuations for source exceptions/undefined interrupted destruction, not C++ ordinary semantics or callback rollback.

Partially published inconsistent mutations guard further mutation with ErrASTPending until Reset/Cleanup/Del recovers the real stored roots. Completed new INCs that the source has not stored remain observable caller-recoverable prefixes. A pure primary expansion refusal before callbacks does not set that guard and allows ordinary source retry.

Scoped Insert first Contains, then base Insert, then actual trail Push. Duplicate keys ignore all unused incoming values. Failed trail growth leaves the installed binding and source unstored trail INC. Push saves current trail size. Pop computes unsigned size−n and reads the actual limit with source debug logical assertions plus real physical/initialized guards; it erases ascending keys, resizes the real trail, then resizes limits. Release retained-limit underflow can therefore grow the trail with NULLs and construct zero uint limits; it is not universally rejected. Debug rejects at the actual logical index assertion.

Scoped Reset resets the borrowed substitution, then trail, then limits. Scoped Cleanup forwards only underlying Cleanup and leaves trail/limits unchanged. Destructor destroys limits then trail and **never resets the borrowed map**. Pending pop/deletion cursor is charged scalar metadata. After a refused tail DEC, destruction drains the still-unconsumed tail and then the still-owned prefix, without replay. Stage3 payload requires prefix<2^29; the actual pointer-vector maximum capacity floor((UINT32_MAX−12)/8) proves this, and simulated bounds plus actual refused Pop→refused Del→repaired Del test it. Pending growth owns only the real old logical prefix. No callback preflight or synthetic callback hook is used.

## Test evidence and source registration boundary

`src/test/main.cpp` registers tst_expr_substitution. Its active body constructs the object but has **zero ENSURE/VERIFY assertions**; insertion and substitution-rewriter use are commented divergent code. The Go test ports the constructor/destructor action route and labels this boundary; the full BV rewrite/printed sequence remains OPEN. `src/test/substitution.cpp` exercises the different substitution/unifier classes and is not counted. Actual consumers include th_rewriter, SMT asserted_formulas, dom_simplify, solve_eqs and other tactics; their algorithms remain OPEN.

Initial full API missing-implementation RED71072 preceded translation. Behavioral RED77946 covered consumed same-entry erase and the missing destructor sparse-reset allocation. RED85540 covered real-hash collisions/wrap/last-tombstone insertion, finalize capacity publication order, and preserving later entries on consumed Reset error. Disabled unused-argument RED45084 preceded correction. Retained-limit and caught-pop destructor RED6329, and primary growth-retry RED20211, preceded correction. Later tests use actual source callbacks/counters with explicit typed-domain counter seeds, never fabricated callback implementations.

Current31 groups additionally cover a500-action independent bounded mapping/reference-count model, all four policies, raw output preservation, NULL quirks in both profiles, every constructor reservation prefix to observed completion, distinct map growth-refusal prefixes, immediate physical roots before Close, actual128/24/48 widths, simulated host bounds, scoped nesting/cleanup/NULL lookup, refused/consumed continuation and zero warm allocations under budget refusal. Final frozen full native/checkeddebug/race/actual JS/vet/WASI/no-generics evidence is recorded below.

## Owned files

NEW only: ast_expr_substitution.go, ast_expr_substitution_table.go, ast_expr_substitution_ops.go, ast_expr_substitution_scope.go; ast_expr_substitution_test.go, ast_expr_substitution_failure_test.go, ast_expr_substitution_scope_test.go; this document. Existing core/dependency/reference owners, central metadata, proof/SAT files and Git are untouched.


## Final immutable verification (2026-10-03)

Isolated committed HEAD b4c79dd:golang plus ONLY these seven new Go files and this document at `.cache/release/ast-expr-substitution-final`. Root renewed full source review accepted the recorded corrections. Newer base81529c8 combines separately under coordinator verification; these seven Go files remain identical.

- Full native `go test -count=1 ./...`: terminal84300 PASS; g3 2.470s, mpnum22.303s.
- Full actual `g3checked,g3debug` suite: terminal83091 PASS; g3 2.967s, mpnum22.666s.
- Exact31 named checkeddebug race: terminal42257 PASS,1.445s.
- Exact31 named actual checkeddebug JS through `.cache/js-exec`: terminal58452 PASS,.314s. Independently checked exact equality of expected test names, RUN names and PASS names for both execution profiles.
- Full checkeddebug vet terminal42064 PASS.
- Checkeddebug WASI test binary plus full-package build terminal56949 PASS.
- Source-ledger/no-generics terminal30688 PASS: committed baseline2102 units/677 APIs,33 ported94partial1975unported. Central new source status updates are coordinator-owned; no false completed-template claim is inferred from this audit.

Logs `.cache/expr-substitution-final-{native,debug,race,js,vet,wasi,audit}.log`; named inventory `.cache/expr-substitution-final-names.txt`; Go SHA256 inventory `.cache/expr-substitution-final-sha256.txt`. All seven hashes reverified after terminal gates. Documentation finalized afterward; Go bytes unchanged.

```
a1bbe10659e64f24473eeff38fd0bb1bf5c24c73a7c7fb89435f58e60c8358dc  golang/ast_expr_substitution_failure_test.go
c3365a0be3ad70e38f21e3283c46c3ab645f4d597dc476d828ee78e35dd60937  golang/ast_expr_substitution.go
1ac9f2cbbac3790de3a4c4b2aea7bce7b9d5764c692a4bd20d3bcc8e5f513273  golang/ast_expr_substitution_ops.go
6203439c680071d45bad19a09da624d2a599a9da3ab8e7ec6ecd52deb902f260  golang/ast_expr_substitution_scope.go
046fdbcf16f4f42f46194559633fb7680d2c4db24d832ce5daab2e9a02ec35db  golang/ast_expr_substitution_scope_test.go
382a5fa6a788778c3223a42e803220327f9b7277c77d8d5aa6a5e162ebd48baa  golang/ast_expr_substitution_table.go
a9c0ab298f56e75cdb1ec9333d7b6ef7f4872a134e69bbe5fc4e9ac9f7ee200c  golang/ast_expr_substitution_test.go
```

Exact named inventory:

```
golang/ast_expr_substitution_test.go:func TestASTExprSubstitutionRegisteredConstructorAndPolicies(t *testing.T) {
golang/ast_expr_substitution_test.go:func TestASTExprSubstitutionActualMapOutputsAndReferenceOrder(t *testing.T) {
golang/ast_expr_substitution_test.go:func TestASTExprSubstitutionBoundedIndependentReferenceModel(t *testing.T) {
golang/ast_expr_substitution_test.go:func TestASTExprSubstitutionAllPolicyMapsAndAliasedFindOutputs(t *testing.T) {
golang/ast_expr_substitution_test.go:func TestASTExprSubstitutionNullDefinitionRepeatedNewBranch(t *testing.T) {
golang/ast_expr_substitution_test.go:func TestASTExprSubstitutionNullOptionalOverwriteSourceProfiles(t *testing.T) {
golang/ast_expr_substitution_test.go:func TestASTExprSubstitutionWarmNoAllocationsUnderRefusal(t *testing.T) {
golang/ast_expr_substitution_scope_test.go:func TestASTScopedExprSubstitutionActualTrailAndBorrowedDestructor(t *testing.T) {
golang/ast_expr_substitution_scope_test.go:func TestASTScopedExprSubstitutionCleanupKeepsTrailAndNestedScopes(t *testing.T) {
golang/ast_expr_substitution_scope_test.go:func TestASTScopedExprSubstitutionNullValueIsPresentAndReturned(t *testing.T) {
golang/ast_expr_substitution_scope_test.go:func TestASTScopedExprSubstitutionRefusedTrailCleanupResumesOnlyUnconsumed(t *testing.T) {
golang/ast_expr_substitution_scope_test.go:func TestASTScopedExprSubstitutionRetainedLimitsReleaseResizePaths(t *testing.T) {
golang/ast_expr_substitution_scope_test.go:func TestASTScopedExprSubstitutionDestructorAfterConsumedPopError(t *testing.T) {
golang/ast_expr_substitution_scope_test.go:func TestASTScopedExprSubstitutionDestructorInterruptedTailThenPrefix(t *testing.T) {
golang/ast_expr_substitution_failure_test.go:func TestASTExprSubstitutionConsumedEraseFinishesSameEntry(t *testing.T) {
golang/ast_expr_substitution_failure_test.go:func TestASTExprSubstitutionDestructorExecutesSourceSparseReset(t *testing.T) {
golang/ast_expr_substitution_failure_test.go:func TestASTExprSubstitutionResetRefusalKeepsConsumedKeyProgress(t *testing.T) {
golang/ast_expr_substitution_failure_test.go:func TestASTExprSubstitutionConstructorEveryReservationPrefix(t *testing.T) {
golang/ast_expr_substitution_failure_test.go:func TestASTExprSubstitutionPhysicalLayoutBoundsAndBorrowedProvenance(t *testing.T) {
golang/ast_expr_substitution_failure_test.go:func TestASTExprSubstitutionExplicitProofGuardRunsAfterPrimaryConstruction(t *testing.T) {
golang/ast_expr_substitution_failure_test.go:func TestASTExprSubstitutionResetFailureKeepsLaterDependencyTableRoots(t *testing.T) {
golang/ast_expr_substitution_failure_test.go:func TestASTScopedExprSubstitutionFailedTrailPushRetainsInstalledBinding(t *testing.T) {
golang/ast_expr_substitution_failure_test.go:func TestASTScopedExprSubstitutionConsumedPopErrorResumesSuffix(t *testing.T) {
golang/ast_expr_substitution_failure_test.go:func TestASTExprSubstitutionLateEraseHashGuardReturnsRawConsumedPrefix(t *testing.T) {
golang/ast_expr_substitution_failure_test.go:func TestASTExprSubstitutionFinalizeRefusalPreservesOldCapacity(t *testing.T) {
golang/ast_expr_substitution_failure_test.go:func TestASTExprSubstitutionCollisionsWrapAndLastTombstone(t *testing.T) {
golang/ast_expr_substitution_failure_test.go:func TestASTExprSubstitutionResetConsumedErrorPreservesLaterEntries(t *testing.T) {
golang/ast_expr_substitution_failure_test.go:func TestASTExprSubstitutionDisabledPolicyDoesNotInspectUnusedForeignArguments(t *testing.T) {
golang/ast_expr_substitution_failure_test.go:func TestASTScopedExprSubstitutionConstructorEveryReservationPrefix(t *testing.T) {
golang/ast_expr_substitution_failure_test.go:func TestASTExprSubstitutionGrowthFailurePreservesEachMapPrefix(t *testing.T) {
golang/ast_expr_substitution_failure_test.go:func TestASTExprSubstitutionPrimaryGrowthRefusalAllowsSourceRetry(t *testing.T) {
```


## Terminal evidence and root integration

Agent immutable seven-Go snapshot .cache/release/ast-expr-substitution-final
on HEADb4c79dd passed full native84300 and checked-debug83091, exact31 named
race42257 PASS1.445s and actualJS58452 PASS0.314s, vet42064, WASI test/full
compilation56949 and source/no-generics30688. Ordered hashes and all31 exact
RUN/PASS entries were reverified by the root. Logs
.cache/expr-substitution-final-{native,debug,race,js,vet,wasi,audit}.log.

Root .cache/review/expr-substitution-integration-81529c8 combines the identical
seven Go files with committed actual24 SAT descriptors and current central
method metadata. Full native14058 and checked-debug47146 terminal exit0;
source/no-generics82080 terminal exit0,33ported/97partial/1972unported.
expr_substitution.cpp/h and ref_util.h selected callback loops become partial;
existing hashtable/obj_hashtable units remain partial with selected specialization
annotations. Exact seven Go hashes and every31 named race/JS RUN/PASS were
checked again before commit. Registry, polynomial, config and full-solver drafts
were excluded; this evidence does not claim the entire live draft worktree green.

The full source solver, SMT theories, proof checker, interpolation and all other
required plan components remain within the active objective and incomplete.
