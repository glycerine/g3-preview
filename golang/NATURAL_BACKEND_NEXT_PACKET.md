# Mandatory Natural backend: next implementation map

Division-storage checkpoint base: de4d6a0, following actual missing-API/behavior RED and the parent-reviewed fixture migrations. This records the implemented division checkpoint and the remaining approved full-backend work; it is not full Natural/output/source-finalization completion.

## Actual source actions

- `src/util/mpn.h:90`: four `mpn_sbuffer` members `u,v,t_ms,t_ab`, each source sbuffer inline capacity16. They are part of the manager and never allocate at construction. Destruction reverses declaration order (ab,ms,v,u).
- `src/util/mpn.cpp:232–233`: normalization computes shift then resizes u and v, in that order, before writing either normalized operand. `src/util/mpn.cpp:319` resizes ms only after those operand writes and entry into div_n. Remove the existing Go ms preflight before normalization.
- `src/util/mpn.cpp:339–344`: multiply/subtract precede publishing quotient; a borrow then decrements the published quotient before resizing ab. Remove existing unconditional ab preparation; failed ab allocation must retain the actual completed quotient/u/ms prefixes.
- `src/util/mpn.cpp:206–217`: in Z3DEBUG every ordinary division branch reaches a lexical zero-filled temp(lnum+1), then multiply, aliasing add, and equality assertion after q/r have already been written. The temp is not persistent division scratch. Its allocation or cleanup refusal cannot erase the published q/r prefix or cause a second algorithm pass.
- `src/util/mpn.h:64–84`: AMD64 buffer resize/index casts size_t to unsigned. The committed sourceLimbBuffer implements uint32 size/capacity and size_t-dependent byte requests. Preserve those casts and existing actual access guards; no new arbitrary capacity cap.

There is no standalone src/test/mpn.cpp or registered tst_mpn in this fork. Existing registered MPZ consumers plus independent bounded Algorithm D fixtures/oracles are the evidence, not invented source assertion counts.

## Concrete owner/layout/API

NaturalManager gains `self`, a real `runtime *memory.Runtime`, `ownsRuntime`, and `division [4]sourceLimbBuffer`. The inline arrays/masks and scalar raw/pending tokens reside in the actual reflected/charged Natural owner header. No per-buffer host arrays or cold closure environments are allocated. Source Runtime events occur only for actual overflow byte backing, not Budget reservations or Go metadata.

`RuntimeOwner() *memory.Runtime` returns the actual stable borrowed/default-owned owner. Keep public NewNaturalManager(arena.VectorConfig); a new private `newNaturalManagerWithMemory(c, words, ownsWords, runtime)` supplies the actual Integer runtime. The existing private constructor remains a thin same-implementation adapter for standalone/FF callers and constructs an owned Runtime when absent. The Integer child borrows its runtime, never closes it. FF Natural similarly uses an actual runtime for source overflow; no idle optional backend.

`CloseChecked() error`/owner-itself cleanup retains charged owner/roots on failure. Void Close panics the same retained owner if cleanup fails. It cleans any failed lexical debug scope first, then ab/ms/v/u, current real format roots, and only then an owned Runtime/header. Integer CloseChecked propagates this failure before closing the shared runtime. FF close must retain its parent owner on child refusal; constructor cleanup order and exact cold error fields require a narrow lifecycle integration in mpff.go, not arithmetic changes.

`divScratch [4]arena.VectorRef` may temporarily remain zero solely to compile unreviewed old diagnostics. It must not refer to actual/fake Words records or be counted as payload storage. Division accesses change to concrete indexed `sourceDivRead/sourceDivWrite/sourceDivView`, with indices0–3. The ephemeral limbView gains a source-buffer/runtime branch; no slice borrow escapes. `divCapacity` records actual source capacities, including completed growth preceding failure.

The debug lexical temp uses a real sourceLimbBuffer with source lifetime. An already charged owner field can retain the lexical buffer on cleanup refusal (no new error wrapper/header). Successful finalization clears it before return; retry is explicit cleanup, not automatic repair or C++ exception recovery. A new invocation cannot overwrite a retained scope. Its byte backing must never be reused across successful lexical invocations. Header charge records the real Go safety holder without a source malloc event.

The existing formatScratch/text still own genuine retained format allocations during this next division checkpoint. Their subsequent mandatory lexical char/three-limb replacement remains required; no claim of the full Natural/output backend until that lands.

## Exact production scope

- mpn_storage.go: actual runtime/inline owner construction, indexed division buffer access, reverse cleanup and owner-retained failure API. Current real format helpers remain until output packet.
- mpn_indexed.go: ephemeral source-buffer view; source normalization → ms → conditional ab order; route all ordinary branch endings through actual debug verification, preserving the earlier short-numerator return. Arithmetic stays the existing single Algorithm D pass.
- NEW mpn_source_verify.go: concrete indexed source multiply/add verification and lexical temp lifecycle, no math/big or replay/division substitute.
- mpz.go: pass existing Runtime to the Natural child and propagate checked child cleanup.
- mpff.go: only necessary Natural constructor/lifecycle/error retention; no FF arithmetic rewrite.
- rational_service.go Stats declaration/method only: source Natural overflow raw/pending counts distinct from remaining real format Words. Inline source buffers have no payload allocation; no synthetic LivePayloads entry for inline storage.
- mpn_backend_test.go and NEW mpn_backend_behavior_test.go: tests FIRST missing actual fields/API and independent source action/lifetime/fault oracle coverage. Any old test/audit migration requires its named RED and parent review before editing. upolynomial_modes.go and other agents' consumer files remain coordinator-owned.

## Finite TDD checks required

1. Default and borrowed Integer/standalone/FF runtime identity; all four inline16 buffers, zero actual division backing at construction, no arena division payload refs; pointer-free buffer records and actual owner header charge.
2. 16→32→64 genuine byte allocations and source POS-only copy/old-free order; shrink/reset retain capacity, descending destruction and full physical charge refund.
3. Independent exact division with actual addback and no-addback examples; no-addback leaves ab inline/cap16 and never allocates it. Failed conditional ab allocation observes published quotient decrement and prior normalized u/ms bytes before cleanup.
4. Debug verification raw allocation AFTER actual q/r publication, including early scalar and short magnitude branches. Independent q*d+r oracle and zero extra replay. Source debug lexical backing is freed each successful call; failure roots stay charged/owned.
5. Failure at u, v, ms, ab and lexical debug temp stages: inspect actual Runtime blocks/initialized state, owner raw+pending roots/capacity/POS, source counters and exact output prefixes before formatting or teardown. Physical malloc failure is memory.Failure(Memout101); cold metadata failure remains arena.ErrInjectedFailure.
6. Release warmed division with reusable overflow must allocate zero Go bytes/objects and make no source allocation event. Debug has real lexical temp allocations for >15 limbs, so do not assert the release zero-allocation policy on that source branch.
7. Native and checked/debug relevant tests, vet and source/no-generics. Race/platform checks remain deferred by user instruction.

Full source teardown/lexical formatting/transitive owning error APIs, runtime-global/generated initialization and cross-manager physical adoption remain mandatory later milestones, not replaced by this packet.

## Implemented division checkpoint (frozen eight Go files)

The four member buffers are actual inline16 sourceLimbBuffer values; overflow
uses the same real Runtime as IntegerManager. Standalone Natural/FF callers use
an owned Runtime. No division Words allocation or fake counter exists. u/v
resize and normalized writes precede ms; conditional ab grows only after the
subtraction and published quotient decrement. The real debug temp executes
source Algorithm M, then aliasing Algorithm A, then the comparison, and is
physically destroyed on success. Its charged scalar holder retains actual failed
roots for explicit cleanup, and a new owning invocation rejects that context.
The source short-numerator branch precedes denominator validation and returns
without common debug verification. Unsigned quotient-count underflow is guarded
at each actual endangered output write, preserving any prior zero writes.

Integer CloseChecked now propagates child cleanup before releasing its shared
Runtime. Natural CloseChecked destroys retained verification then ab/ms/v/u,
remaining real format storage, and owned Runtime/header. This is owning Go cleanup;
actual SourceFinalize ordering and the complete transitive owner stage API remain
unfinished. The current FF caller obtains the actual source-backed Natural but
its dedicated retained outer error/cursor integration remains the next lifecycle
work. Remaining formatScratch/text storage is still genuine arena storage and
must be replaced by the approved lexical output packet. Source-global/TLS binding,
generated init/finalize and cross-manager adoption remain open.

The compatibility divScratch array stays zero and is not read by division.
Legacy divRead exists solely to compile held tests; no physical scratch or
payload charge is created through it. RationalService Stats now counts real
raw/pending division and retained debug-temporary objects alongside remaining
real format Words; inline buffers contribute zero.

Source proof/consumer inventory is unchanged: no standalone registered mpn test
exists. Eleven new test groups cover actual constructor/borrowed-runtime identity,
overflow/shrink, no-addback, debug output-before-allocation and lexical destruction,
short-numerator early return, unsigned guarded write prefixes, Stats roots,
conditional add-back refusal after decrement, and warmed inline allocation plateau.
The two approved existing format fixtures preserve constructor fault sweep,
double Close/refund and independent exact division while replacing only obsolete
four-Words and unconditional-ab expectations. Release warm calls make no Go or
source allocations; debug large calls really allocate and free the lexical temp.

Actual RED evidence:

- `.cache/natural-missing-api-red.log`: missing RuntimeOwner/division/CloseChecked,
  isolated HEAD95a8236; pending_mpn tag removed now these three tests execute.
- `.cache/natural-action-red.log`: unused ab incorrectly grew to32.
- `.cache/natural-debug-local-red.log`: two debug lexical source actions absent.
- `.cache/natural-early-return-red.log` and
  `.cache/natural-unsigned-prefix-red.log`: both prior implementations reached
  zero-denominator normalization and did not terminate; sessions12680/39031
  were deliberately interrupted (exit130), not reported as ordinary test failures.
- `.cache/natural-stats-red.log`: real u/v/ms allocations absent from Stats.
- `.cache/natural-obsolete-two-red.log` and `.cache/natural-old-addback-red.log`:
  the two individually reviewed obsolete fixtures.

Final isolated snapshot: `.cache/release/natural-division-final-de4d6a0` contains
committed HEAD plus exactly the eight changed/new Go files. Frozen SHA inventory:
`.cache/natural-division-final-sha256.txt`. Gate results are appended only after
terminal completion; broad preexisting backend topology/fault fixture RED is not
claimed resolved by selected source division coverage.

### Terminal checkpoint campaign before the reported zero-value guard correction

Selected exact frozen tree: native82228 exit0 (31 RUN,29 PASS,2 intentional
Z3DEBUG-only skips), checked/debug14845 exit0 (31 RUN/PASS), full vet31954 exit0,
source/no-generics6436 exit0 (2102 units/677 APIs,33 ported/148 partial/1921 unported).
Logs: `.cache/natural-division-final-native.log`, `-debug.log`, `-vet.log`,
`-source.log`. Full native20633 and checked/debug83818 exit1 are recorded in
`.cache/natural-division-full-native.log`/`-debug.log`; known broader cell/backend
root and allocation-point assumptions remain RED. Those campaigns are not repeated
or reported GREEN. The full native log additionally identified the narrow new
zero-value Natural FormatInto guard classification discrepancy; parent review is
requested before touching the frozen production byte. No unrelated failure
assertions are weakened.

### Approved zero-value guard correction and final freeze

Named actual RED `TestOwnedDecimalFormatAndPublication:79` led to the narrowly
approved guard correction: nil/uninitialized wordsnil Natural returns ErrClosed;
constructed copied self mismatch still returns memory.ErrRuntimeOwner. No math or
other failure fixture changes accompanied this correction. The eight-file SHA
inventory below supersedes the earlier storage hash; the same snapshot was
updated only with this approved guard.

```text
e933c8c2a1a7b6104f0a00f8470747cfb7d94f7ad62933ae1d3a224a02af7943  mpnum/mpn_storage.go
e38482d3fb1820f707d32f46402a797c01f755e9f144665af87f0a04422a083b  mpnum/mpn_indexed.go
e22e780fb3ded24ee4cb18ce8ad9489a43e99b8eaefc5f306283ff872b9fad30  mpnum/mpn_source_verify.go
d862e76fb8496847a80f41f2c3e2ef3feff6b07fcdd4b0fd54b304c5d87b1926  mpnum/mpn_backend_behavior_test.go
a4eb7cdb08ab7f469010362867bef1619783ea26e74f4613db282a66556b29c7  mpnum/mpn_backend_test.go
ce1f2b00a618a2c73b25d4efcdd96699d2ed06eb393b35942f5ece9ce1df502f  mpnum/mpz.go
2660cbc8d64a2df9f03add44765112de5197fb65902f38353ac7d57f473734cf  mpnum/rational_service.go
3570b67756fb5cf6c7f7025361ea248e17289fc7647703b5fadc28c11ad51eca  mpnum/mp_format_test.go
```

Final affected/selected native32753 exit0:32 RUN,30 PASS and2 Z3DEBUG-only SKIP,
0.601s. Checked/debug45520 exit0:32 RUN/PASS,0.477s. Full vet52063 exit0.
Final logs `.cache/natural-division-corrected-native.log`, `-debug.log`, `-vet.log`.
No full-suite rerun was made after the guard: the prior full RED campaigns remain
honest baseline evidence, including still-unclassified broader root/fixture issues.
No source completion or whole-package GREEN claim accompanies this checkpoint.

Final source/no-generics26746 terminal exit0 on the corrected frozen tree:
2102 units/677 APIs,33 ported/148 partial/1921 unported. Log
`.cache/natural-division-corrected-source.log`. All selected final gate handles
are terminal; eight Go files and this packet are released to parent for checkpoint.
User-deferred race/platform checks were neither launched nor claimed.
