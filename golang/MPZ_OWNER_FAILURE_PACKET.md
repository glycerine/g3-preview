# Mandatory MPZ backend: retained owning failure contract

Read-only production proposal, based on committed tracked memory core facaeb4. This document authorizes no production edit. Existing production/test files, metadata and Git remain parent-owned. The selected source profile is non-GMP, uint32 digits, LP64 or ILP32 size_t, with actual selected allocator debug/platform branches. Source files/hashes are retained in MPZ_ALLOCATOR_INTEGRATION_PACKET.md. Other digit/GMP profiles remain open.

## Actual ownership and physical layout

Default IntegerManager must use one actual memory.Runtime and memory.SmallAllocator for every numeric cell, not keep an optional idle backend beside Words. Supplied Runtime is borrowed; nil constructs an isolated owned Runtime (explicitly not a process-global runtime claim). Borrowed Runtime and Budget must outlive every related numerical/coefficient owner. Cells.Budget and Words.Descriptors.Budget remain equal for compatibility; the Words configuration no longer creates numeric payload owners. Runtime/Budget/profile compatibility is checked before source cell work.

Each source cell is one actual allocator object: uint32 size at byte0, capacity at byte4, uint32 digits beginning byte8. Source requested bytes use uint32 `(8+4*capacity)` wrapping before widening even on LP64. Actual requested access uses codec bounds; source-defined accesses into rounded pooled slack may use actual ObjectExtent plus raw Runtime codec bounds, never the surrounding chunk extent. A wrapped allocation completes before the endangered header write reports a typed guard. Header initialization comes only from actual source stores.

Numeric descriptor retains ObjectRef, request uint32, and a private scalar release phase/cursor, instead of owning a second digit array. Current Int scalar token shape and owner-specific permanently dead failedRef stay unchanged. Directory, ledger, failedRef and pending-state charges are Go safety metadata, never source malloc events. Physical source capacity is authoritative; cached safety copies are not substitute initialized bytes.

IntegerManager embeds its Natural state, including four actual [16]uint32 buffers, initialized masks and scalar overflow/pending Blocks. An ephemeral pointer to the embedded state is permitted as cold owner metadata; it creates no separate Natural allocation/header charge. The four inline arrays are charged once through the real Integer header. Standalone NaturalManager needs its own charged owner and explicit borrowed/owned Runtime contract; its callers (notably FastFloatManager) require a separately coordinated compatibility change, not a second numeric algorithm.

## Retained owner fields and public error API

Each of IntegerManager, RationalManager and RationalService embeds a concrete cold `numeralFailureState` containing `original error`, `cleanup error`, operation/stage uint8, root cursor uint32, component cursor uint8, active bool. These two interfaces are cold error transport only; no hot record contains an interface or closure. Reflected owner header charges these fields. Each owner has its own self-identity guard; shallow copied owner operations fail without freeing the true owner.

Proposed common cold interface:

```go
type ConstructionFailure interface {
    error
    Unwrap() error
    Cleanup() error
    RuntimeOwner() *memory.Runtime
    OriginalCause() error
    CleanupCause() error
}
```

The owners themselves implement it; there is no IntegerConstructionFailure/RationalConstructionFailure/Service wrapper allocation. Fixed-label Error does not format an allocated string. OriginalCause is the first operation failure and never changes on cleanup retry. CleanupCause is the most recent cleanup refusal (nil after completed cleanup). Unwrap returns original, or cleanup when there was no construction/operation cause. Neither cause may refer to that same owner: when a leaf returns self during cleanup, extract its stored underlying transport cause, never set original=self or cleanup=self. Child-owner errors may be retained by an outer owner only when that child cannot refer back to the outer owner; their ownership graph is strictly acyclic. errors.Is/As must terminate; tests inspect the bounded unwrap chain before calling them.

`Cleanup()` delegates to that owner's checked owning teardown without changing the original classification. A successful cleanup can release all owner charges while the caller retains the Go error object, as with a caller retaining any closed Go owner; it no longer contains owning roots. Status/cursor fields are not caller-authorized flags and cannot skip any source action.

Existing constructor return shape remains `(*Owner,error)`. Before header reserve succeeds, return nil plus static failure. Immediately after successful header reserve, allocate/initialize the actual owner (including self identity) before acquiring child roots. On an error, attempt exact cleanup. If completed, return nil plus original cause. If refused, return nil plus that same existing owner; never allocate a second error header after Budget refusal. A constructor error therefore retains every physical and cold root until caller Cleanup. Do not release a header containing still-owned roots.

Integer/Rational expose `SourceFinalize() error`, `CloseChecked() error`, and retain existing void Close. Proposed void Close invokes CloseChecked and panics that SAME retained owner on refusal, rather than silently dropping a root; this compatibility adapter requires explicit parent approval. Service's existing Close returns checked error, and SourceFinalize remains separate. Ordinary arithmetic rejects once source finalization or owning teardown begins; cleanup operations remain available. Valid output reads after an unrelated scratch publication failure remain allowed before teardown begins.

## Constructor and lexical cleanup cursors

Integer constructor: owner/descriptor/dead-token preparation (explicit Go metadata safety adapter), actual allocator header and embedded Natural state, then source tmp0,arg0,tmp1,arg1,intMin,two64 and final tmp0/two64 exchange. Cursor reflects actual completed roots, not a presumed six-cell baseline. Missing scratch remains absent. One pendingCell field captures a successfully allocated object before capacity write/descriptor preparation; another pending allocation cannot overwrite it. The allocator itself owns any partial raw/headerless or lexical scratch failure roots.

Integer source finalization order follows mpz.cpp142–161 and reverse member destruction: two64, intMin, tmp0,arg0,tmp1,arg1; source SYNCH lock destruction; Natural t_ab,t_ms,v,u; allocator chunk frees/diagnostics. Only completed release steps advance root cursor. Failed physical release retains the object and real memory.ReleaseProgress; after physical release an error is retained and retry acknowledges the same private authority rather than refreeing or repeating requested/global subtraction. Managed roots are cleared only when their cleanup step completes; separate physical-dead state rejects dangling normal reads while still allowing cleanup continuation.

SourceFinalize never frees caller direct cells. Pooled caller locators become dangling only when actual chunks are freed. Direct live objects survive allocator source destruction; descriptor retains original request, raw Block/offset and release phase before allocator context closes. Owning CloseChecked cleans caller live cells/pending cells and any remaining source members; after SourceFinalize it uses those captured direct raw tokens through the same Runtime, never fabricates a SmallAllocator event on a closed allocator. Runtime.Deallocate retries persisted before-free authority; after-free pending acknowledgement is completed through retained genuine authority. If the current API cannot acknowledge a postfree error after allocator context is closed, preserve that context until completion; do not use exported flags to authorize skipping. The descriptor directory and header release only after actual roots and allocator pending roots are empty. Internally owned Runtime closes last; borrowed Runtime never closes.

Rational constructor currently has only one child construction edge (mpq.go49): IntegerManager. Proposed owner is charged/allocated first. If child construction fails, no other Rational root exists, so refund the Rational header and forward the child owner error unchanged. Otherwise retain child pointer immediately. Rational SourceFinalize destroys nTmp,addTmp1,addTmp2 then ltTmp1,ltTmp2,addMulTmp; each rational is NUMERATOR then DENOMINATOR (mpq.h186–189). Cursor advances component by component; never clear the whole pair after only numerator cleanup. Then call child's SourceFinalize. Owning CloseChecked follows those same leaf actions, then child's owning CloseChecked, then releases Rational header. A child failure retains the Rational parent if it still owns child/other roots. `drop` cannot remain an error-ignoring helper for owning destruction.

Service constructor currently has only one child edge (rational_service.go76): RationalManager. Before constants, a failed child is forwarded unchanged after the unused Service header is refunded. Once manager exists, service owner retains it and any initialized constant prefix. Initialization order is zero,one,minusOne,inf-rational constants,inf-int constants. Per-constant initialized components must be recorded before copy calls. Partial construction must not pretend all constants exist or free uninitialized handles.

Service teardown follows existing audited rational service ordering: inf-rational zero/one/minusOne, each SECOND then FIRST (wrapper lexical member destruction); inf-int zero/one/minusOne FIRST; ascending power-cache values, each NUM then DEN; free cache backing only after all entries clean; zero,one,minusOne NUM then DEN; manager checked teardown; service header last. Separate stage/cache-index/component cursors retain the exact current root. Parent/child locks follow existing service->manager order; Cleanup must not call a public service method recursively while holding its mutex. A late initialization error never calls Close and discards its result. Store both original and cleanup cause in the existing service, retaining its manager and partially initialized components.

## Physical initialized-header and span map

| Source operation | Actual write/action order |
| --- | --- |
| mpz.h124 allocate | actual object allocation, capacity store ONLY; no size/digit initialization |
| h132 allocate_if_needed | small sign1 before allocation; growing old free before sign1/new allocation; capacity repeated only at its source statement |
| h156 ensure_tmp | old free; then uint32 `(3*needed+1)>>1`; allocate/capacity only; actual endangered span guard after allocation |
| cpp165/207 set_big_i64/ui64 | allocate/pointer publication if needed; sign; low AND high32 digit stores even when high0; size last |
| cpp247 set scratch | inspect actual requested scratch digits; sign/ref swap; result size after swap; missing scratch allocation after result publication |
| cpp306 limb set | trim input; sign; optional oldfree/allocate; size and capacity source stores before logical digit copy |
| cpp1185 big_set | self early return; sign; optional oldfree/allocate; size/capacity stores, then logical digits copy |
| cpp1555 ensure_capacity | new allocation while old live for big; new size before digit copy; oldfree then pointer publication; distinct from big_set |
| cpp1596 normalize | scans existing size/digits; reset-small cleanup or final new size after scan |
| cpp1630 div2k | digit shift first; new size; normalize |
| cpp1700 mul2k | ensure_capacity; zero-fill new digit range BEFORE publishing new size; shift/copy; normalize |
| cpp1472 power-of-two fast path | allocate_if_needed; size BEFORE zero-fill/final digit; sign last |
| add/sub/mul/div scratch | capacity preparation/ephemeral span only; no physical scratch size store until source result-transfer statement |

`prepareTmpSpan` and current resizeCell cannot simply become physical size writers. Separate ephemeral digit span extent from actual published header. Scratch access checks ObjectExtent and actual initialized words at that action, not an uninitialized size header. Every current store/resize/publishTmpAt caller must be mapped before backend changes; do not globally preflight or repair missing scratch. No setter synthesis of zeros to make undefined source scratch fields readable.

## Full Natural topology included, not deferred behind optional cells

Four embedded source sbuffer<uint32,16> fields expand with source uint32 doubled capacity. Each actual raw allocation succeeds before copy of POS initialized words, old overflow deallocation, and new pointer/capacity publication. Old/new raw backing and bitmap/cold metadata peaks are charged honestly. Retained pending new and completed oldfree phases require scalar per-buffer root state on a failure. Reset retains capacity; finalization frees overflow in reverse field order and returns to inline16.

mpn.cpp308 div_n resizes ms(n+1) first; writes quotient q_hat, then if borrow decrements quotient and resizes ab(n+2) at THAT branch. Remove current universal preprepared ab: no-addback paths do not allocate it. Late ab failure preserves prior numerator subtraction/quotient decrement. Existing single AlgorithmD and direct paired quotient/remainder publication are retained, not restaged.

mpn.cpp206 Z3DEBUG verification is a real lexical temp buffer after output writes, even in early division branches. It must not be an always-preflight scratch allocation. Formatting (cpp360) with >1 limb owns three lexical inline16 sbuffers temp,t_numer,t_denom, constructed in that order and destroyed reverse; one-limb route uses the actual bounded unsigned stack conversion. mpz display emits sign before its inline1024 char buffer (uint32 `11*size` requested construction), then converts, then C-string insertion. Existing retained formatScratch/text owners must be removed in the faithful backend, not counted as equivalent topology. External ostream/string allocation and TRACE channel bindings remain explicit separately audited runtime prerequisites.

## Exact proposed production/test ownership map (not yet approved)

Core: mpnum/mpz.go, mpz_slab.go; NEW mpz_allocator_backend.go and numeral_failure.go; reference profile files only if unchanged scalar accessor integration requires it. Direct data/span sites: mpz_mul.go,mpz_div.go,mpz_power_shift.go,mpn_storage.go,mpn_indexed.go,mp_format.go; physical-cell read sites mpz_gcd.go,mpz_misc.go,mpz_log.go also require signature/guard review before deciding whether existing arithmetic-body edits are necessary; NEW mpn_source_buffer.go (and concrete char-buffer helper in a coordinated file). No parallel second Natural algorithm.

Transitive required owning paths: mpq.go and rational_service.go; inf_rational.go and inf_int_rational.go destructor helpers; their current arithmetic regions remain numeric-agent owned and must be coordinated before any edit. No existing file is authorized by this proposal itself.

Other borrowing/cleanup consumers requiring review, not blanket edits: mpbq.go,mpbqi.go,mpzzp.go,scoped_numeral.go,scoped_binary.go,scoped_binary_interval.go,inf_eps_rational.go; FastFloatManager mpff.go uses standalone Natural and needs actual Runtime/Natural lifetime compatibility. UPolynomialManager/coefficient cohort borrows Integer; no clone/implicit Runtime finalization. Polynomial/scoped public roots remain live until their own cleanup; backend owner Close does not constitute source destruction of borrowed container owners. Existing source mathematics tests remain unchanged except genuinely obsolete physical topology/failure atomicity assertions with linked replacement tests approved separately.

Production constructor search (excluding tests/cache): only mpq.go→NewIntegerManager and rational_service.go→NewRationalManager; other owners receive borrowed managers. No production NewRationalService caller currently appears. At owner boundaries, every ignored Del/Close in the selected call graph must either propagate checked completion or retain an explicit root cursor. Tests will compare immediate named owned roots BEFORE formatting/cleanup, never infer six scratch roots universally.

## Test-first acceptance and current honest status

Two NEW test files currently contain six Integer groups plus two Natural groups: mandatory default/borrowed Runtime; exact source six constructor objects/capacity and final exchange; initialized/uninitialized physical fields including intMin spare high0; direct caller survival through SourceFinalize; uint32 size wrap; transitive bounded constructor reservation sweep; scratch span does not publish size and typed64 setter high0; actual inline16 expansion/reset/failure oldroot retention. They remain missing-API compile RED, not behavioral execution passes.

Next genuine behavioral tests after API implementation: every header/store ordering at finite allocation refusal; constructor failure original/cleanup causes and acyclic errors.As/Is; retained Integer->Rational->Service charge/root graphs; beforefree/postfree retry without duplicate counters; borrowed domain untouched; pending wrapped-header object; partial pair/cache destruction cursors; source direct finalization versus owning cleanup; no-addback/addback/debug lexical/format lexical allocation prefixes; warmed nontrivial arithmetic true source physical plateau/no Go heap allocations. Genuine postfree fault transport must use real Runtime transitions, not fake Budget accounting. Existing lower release tests establish those lower boundaries; no claim yet that transitive numerical constructor late-cleanup branches have run. If a bounded transitive failure cannot be reached through current public core profile, report that test gap rather than invent a synthetic event or uncharged test hook.

## Refreshed API-absence evidence

Isolated archive facaeb4:golang with ONLY the two new mpnum test files: .cache/release/mpz-owner-design-red-facaeb4. Command `go test ./mpnum -run 'TestInteger(Default|Source|Constructor|Scratch)|TestNaturalSource' -count=1` terminated exit1 (tool terminal result, no running handle). Log .cache/mpz-owner-design-red-facaeb4.log reports undefined ConstructionFailure, absent IntegerConfig.Memory/RuntimeOwner/cellObject/allocator and undefined sourceLimbBuffer. Compilation ends before tests run; zero RUN/PASS claims. Compile-time method-set assertions require the three actual owner pointers themselves to implement the cold failure interface; the bounded constructor fixture checks an acyclic unwrap chain before errors.Is, unchanged original cause after Cleanup, separately cleared cleanup cause and borrowed Runtime identity. Current code still lacks these methods.

New test hashes: mpz_allocator_phase_b_test.go 276157f6221e857ccd66219e95d2aefd44bc5485d3a499c32a444a1107291227; mpn_source_buffer_test.go b1d38a65f73fe7f668146bd36277ce887cb43827e71489b2ff1059100d41cca8. No production was changed; no full native/checked/race/JS/vet/WASI PASS is claimed for the missing backend.
