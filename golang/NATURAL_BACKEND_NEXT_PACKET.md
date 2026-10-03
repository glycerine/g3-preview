# Mandatory Natural backend: next implementation map

Current base: 8ade92d, followed by parent Stats correction 8d9b20f. This is implementation planning for the approved full backend objective, not source completion. The separately frozen two-fixture correction remains untouched.

## Actual source actions

- `src/util/mpn.h:90`: four `mpn_sbuffer` members `u,v,t_ms,t_ab`, each source sbuffer inline capacity16. They are part of the manager and never allocate at construction. Destruction reverses declaration order (ab,ms,v,u).
- `src/util/mpn.cpp:232–233`: normalization computes shift then resizes u and v, in that order, before writing either normalized operand. `src/util/mpn.cpp:319` resizes ms only after those operand writes and entry into div_n. Remove the existing Go ms preflight before normalization.
- `src/util/mpn.cpp:339–344`: multiply/subtract precede publishing quotient; a borrow then decrements the published quotient before resizing ab. Remove existing unconditional ab preparation; failed ab allocation must retain the actual completed quotient/u/ms prefixes.
- `src/util/mpn.cpp:206–217`: in Z3DEBUG every ordinary division branch reaches a lexical zero-filled temp(lnum+1), then multiply, aliasing add, and equality assertion after q/r have already been written. The temp is not persistent division scratch. Its allocation or cleanup refusal cannot erase the published q/r prefix or cause a second algorithm pass.
- `src/util/mpn.h:64–84`: AMD64 buffer resize/index casts size_t to unsigned. The committed sourceLimbBuffer implements uint32 size/capacity and size_t-dependent byte requests. Preserve those casts and existing actual access guards; no new arbitrary capacity cap.

There is no standalone src/test/mpn.cpp or registered tst_mpn in this fork. Existing registered MPZ consumers plus independent bounded Algorithm D fixtures/oracles are the evidence, not invented source assertion counts.

## Concrete owner/layout/API

NaturalManager gains `self`, a real `memory *memory.Runtime`, `ownsMemory`, and `division [4]sourceLimbBuffer`. The inline arrays/masks and scalar raw/pending tokens reside in the actual reflected/charged Natural owner header. No per-buffer host arrays or cold closure environments are allocated. Source Runtime events occur only for actual overflow byte backing, not Budget reservations or Go metadata.

`RuntimeOwner() *memory.Runtime` returns the actual stable borrowed/default-owned owner. Keep public NewNaturalManager(arena.VectorConfig); a new private `newNaturalManagerWithMemory(c, words, ownsWords, runtime)` supplies the actual Integer runtime. The existing private constructor remains a thin same-implementation adapter for standalone/FF callers and constructs an owned Runtime when absent. The Integer child borrows its runtime, never closes it. FF Natural similarly uses an actual runtime for source overflow; no idle optional backend.

`CloseChecked() error`/owner-itself cleanup retains charged owner/roots on failure. Void Close panics the same retained owner if cleanup fails. It cleans any failed lexical debug scope first, then ab/ms/v/u, current real format roots, and only then an owned Runtime/header. Integer CloseChecked propagates this failure before closing the shared runtime. FF close must retain its parent owner on child refusal; constructor cleanup order and exact cold error fields require a narrow lifecycle integration in mpff.go, not arithmetic changes.

`divScratch [4]arena.VectorRef` may temporarily remain zero solely to compile unreviewed old diagnostics. It must not refer to actual/fake Words records or be counted as payload storage. Division accesses change to concrete indexed `sourceDivRead/sourceDivWrite/sourceDivView`, with indices0–3. The ephemeral limbView gains a source-buffer/runtime branch; no slice borrow escapes. `divCapacity` records actual source capacities, including completed growth preceding failure.

The debug lexical temp uses a real sourceLimbBuffer with source lifetime. An already charged owner field can retain the lexical buffer on cleanup refusal (no new error wrapper/header). Successful finalization clears it before return; retry is explicit cleanup, not automatic repair or C++ exception recovery. A new invocation cannot overwrite a retained scope. Its byte backing must never be reused across successful lexical invocations. Header charge records the real Go safety holder without a source malloc event.

The existing formatScratch/text still own genuine retained format allocations during this next division checkpoint. Their subsequent mandatory lexical char/three-limb replacement remains required; no claim of the full Natural/output backend until that lands.

## Exact production scope

- mpn_storage.go: actual runtime/inline owner construction, indexed division buffer access, reverse cleanup and owner-retained failure API. Current real format helpers remain until output packet.
- mpn_indexed.go: ephemeral source-buffer view; source normalization → ms → conditional ab order; route all branch endings through actual debug verification. Arithmetic stays the existing single Algorithm D pass.
- NEW mpn_source_verify.go: concrete indexed source multiply/add verification and lexical temp lifecycle, no math/big or replay/division substitute.
- mpz.go: pass existing Runtime to the Natural child and propagate checked child cleanup.
- mpff.go: only necessary Natural constructor/lifecycle/error retention; no FF arithmetic rewrite.
- rational_service.go Stats declaration/method only: source Natural overflow raw/pending counts distinct from remaining real format Words. Inline source buffers have no payload allocation; no synthetic LivePayloads entry for inline storage.
- NEW mpn_backend_test.go: tests FIRST missing actual fields/API and independent source action/lifetime/fault oracle coverage. Any old test/audit migration requires its named RED and parent review before editing. upolynomial_modes.go and other agents' consumer files remain coordinator-owned.

## Finite TDD checks required

1. Default and borrowed Integer/standalone/FF runtime identity; all four inline16 buffers, zero actual division backing at construction, no arena division payload refs; pointer-free buffer records and actual owner header charge.
2. 16→32→64 genuine byte allocations and source POS-only copy/old-free order; shrink/reset retain capacity, descending destruction and full physical charge refund.
3. Independent exact division with actual addback and no-addback examples; no-addback leaves ab inline/cap16 and never allocates it. Failed conditional ab allocation observes published quotient decrement and prior normalized u/ms bytes before cleanup.
4. Debug verification raw allocation AFTER actual q/r publication, including early scalar and short magnitude branches. Independent q*d+r oracle and zero extra replay. Source debug lexical backing is freed each successful call; failure roots stay charged/owned.
5. Failure at u, v, ms, ab and lexical debug temp stages: inspect actual Runtime blocks/initialized state, owner raw+pending roots/capacity/POS, source counters and exact output prefixes before formatting or teardown. Physical malloc failure is memory.Failure(Memout101); cold metadata failure remains arena.ErrInjectedFailure.
6. Release warmed division with reusable overflow must allocate zero Go bytes/objects and make no source allocation event. Debug has real lexical temp allocations for >15 limbs, so do not assert the release zero-allocation policy on that source branch.
7. Native and checked/debug relevant tests, vet and source/no-generics. Race/platform checks remain deferred by user instruction.

Full source teardown/lexical formatting/transitive owning error APIs, runtime-global/generated initialization and cross-manager physical adoption remain mandatory later milestones, not replaced by this packet.
