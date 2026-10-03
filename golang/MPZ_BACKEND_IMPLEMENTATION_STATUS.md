# Mandatory MPZ/Natural backend implementation progress

Full mandatory backend implementation is authorized and ongoing. IntegerManager now has an in-progress mandatory physical-cell implementation draft described below; NaturalManager still uses its genuine arena scratch implementation. The committed helper checkpoint alone was not completed backend integration. This is an explicitly unfinished implementation checkpoint; no source completion is claimed. Cross-manager physical adoption, process-global runtime bindings and the full solver remain required/open.

## Source-mapped helpers actually implemented

- mpn_source_buffer.go: concrete sbuffer<uint32,16>, inline initialized mask; actual Runtime overflow allocation, source uint32 capacity doubling and size_t request; POS-only copy; old overflow free before pointer publication; retained unpublished new root on failure; reset and lexical finalization.
- mpz_allocator_backend.go: source uint32 cell_size wrapping even on LP64; real SmallAllocator object allocation followed by capacity-only write; scalar physical backing locator; generation validation and actual rounded-object word bounds. Size/digits remain uninitialized until real setters write them. Returned completed root survives a failed capacity write for pending ownership.
- mp_format_output.go: concrete source sbuffer<char,1024>, actual initialized inline bytes, source POS copy/overflow/free order, constructor zero pushes, lexical finalization. This file does not yet contain the approved numeralOutputScope; that integration is next active work.

These scalar helpers contain no Go generics/unsafe/cgo/mathbig. Inline/locator state is charged by the actual eventual enclosing owner; unit fixtures own local stack state and charge every actual lower owner/backing. No helper increments counters except through the genuine memory.Runtime/SmallAllocator action.

## Actual test-first evidence

Initial design API-absence RED: .cache/mpz-owner-design-red-facaeb4.log; full default backend/owner-error tests remain compile RED until actual manager APIs exist. No execution PASS claimed for them.

SourceLimbBuffer named baseline tests then passed. A design expectation was corrected after terminal failure54689: actual physical Budget refusal becomes memory.FailureMemout101 through Runtime, not arena.ErrInjectedFailure. No transport/accounting production was changed. Added source nonTLS failed-allocation count/extent400 with retained physical136, real TLS >100000 refusal retaining actual old+new blocks, pointer-freedom and warm inline0alloc. Actual endangered inline copy guard RED80941 (.cache/mpn-source-copy-guard-red.log) reported ErrUninitialized instead of ErrBounds after allocation/copied16; fix guard at actual inline index16 and retain allocated/copied prefix.

Cell backing FIRST RED: .cache/mpz-source-cell-backing-red.log undefined allocateSourceCell; then two source profile fixtures passed. Four-byte wrapped cell request yields LP64 rounded8 permitting capacity write, versus ILP32 rounded4 capacity-write guard with root retained. Neither branch can access a digit beyond its actual object.

Char helper FIRST RED: .cache/mpz-source-char-buffer-red.log undefined sourceCharBuffer; then inline1024/no Runtime allocation, real2048 overflow/destruction and failed4096 allocation counter/old-root fixtures passed.

Nine exact current groups:

1. TestNaturalSourceInline16AndActualOverflow
2. TestNaturalSourceBufferFailedExpansionRetainsOld
3. TestNaturalSourceLateTLSRefusalRetainsBothActualRoots
4. TestNaturalSourceInlinePointerFreedomAndWarmNoAlloc
5. TestNaturalSourceCopyGuardRetainsAllocatedPrefix
6. TestIntegerSourceCellBackingInitializesCapacityOnly
7. TestIntegerSourceCellWrappedHeaderGuardUsesActualRoundedExtent
8. TestIntegerSourceCharBufferInline1024AndActualLexicalOverflow
9. TestIntegerSourceCharGrowthRefusalRetainsActualOldAndCounterPrefix

Snapshot .cache/release/mpz-backing-prerequisites-203486f contains six new helper/test Go files; missing default-manager design tests are deliberately excluded. Selected current nine native groups actually RUN/PASS, log .cache/mpz-source-backing-nine-native.log (61225 terminal0). Checked/debug nine, vet and source/no-generics audit are recorded separately only after terminal inspection. Earlier full native/debug/vet/source checks of the four pre-char helper/test files all passed; these are historical prerequisite evidence, not a full current/backend freeze.

## Required remaining implementation

Wire actual source cell backing into numeric descriptor/slab and mandatory default/shared Runtime allocator; charged owner-itself error fields and original/cleanup/stage cursors; source constructor initialization/exchange; SourceFinalize versus owning CloseChecked; exact physical header and ephemeral scratch spans at all mapped setter/publish callers. Preserve all committed GCD/number-theory bodies and source diagnostics.

Embed four actual Natural inline16 arrays and remove arena division payloads; actual conditional ab allocation after quotient publication; source debug verification lexical buffer after outputs. Replace retained formatScratch/text with approved per-output assembly owner and source lexical char/three-limb lifetimes; retain already charged pendingOutput on cleanup refusal. Small integer ostream route uses no char/limb scope. Exact eight-file formatting and mpq/service/inf teardown regions are approved in MPZ_OUTPUT_LIFETIME_PACKET.md; existing test/audit migrations require real named RED and parent review.

User gate policy now defers race/JS/WASM/WASI. No such check will be launched going forward. Earlier launched handles were already terminal when steering arrived; no process remained live. Current checks use relevant native/debug tests, vet and source/no-generics audit. No reference compiler/runtime dependency is introduced into ordinary Go tests.

Final inspected current nine-helper checks: selected checked/debug36002 terminal0 with all nine RUN/PASS; current vet command terminal0; source/no-generics46208 terminal0, source ledger unchanged33/148/1921. Logs .cache/mpz-source-backing-nine-debug.log, -vet.log, -source.log. Native61225 terminal0 with all nine RUN/PASS. These checks validate six prerequisite helper/test files only, not the absent mandatory manager/output ownership wiring. Full prior baseline checks89885/15963/vet60342/source45859 also terminal0 but precede char helper addition and are historical evidence only.

Parent released current GCD files after commitc7bed52; any necessary physical access edits must preserve its exact source diagnostic/zero Copy+Abs changes. Existing additive arithmetic remains unchanged. Exact owner/error/constructor/sourceFinalize/CloseChecked and output lifecycle integration has parent approval; obsolete existing tests/diagnostics require observed RED and individual permission before editing. Next active implementation step is the actual numeric descriptor/Integer/Natural owner replacement and pending output owner.

## Mandatory cell constructor draft after checkpoint 0c9d695

Actual isolated missing-API RED: HEAD 5b414f9 plus only mpz_backend_constructor_test.go, .cache/mpz-backend-constructor-isolated-red.log. The constructor test was subsequently saved by the parent with the explicit port_pending_mpz tag at 0c9d695; this draft removes that tag because the constructor APIs now compile and execute. The separate transitive phase-B suite remains tagged and genuinely unfinished.

The draft replaces every numeric cell's Words payload with one real sourceCellBacking. IntegerConfig.Memory borrows an actual same-budget Runtime; absence constructs and owns a real Runtime. IntegerConfig.Profile explicitly selects allocator branches, while its zero default follows the source build/platform. Each owner allocates a real SmallAllocator, never an unused parallel backend. Numeric descriptors contain scalar ObjectRef/raw backing/bounds and source size/capacity/span bookkeeping, physically charged through reflected slab record size. No second limb array or fake live Words reference exists. The payload field remains zero solely for existing diagnostics to compile until their reviewed migration.

Source capacity-only allocation occurs before descriptor preparation. A completed object that cannot obtain a descriptor remains in the charged IntegerManager.pendingCell field; bounded SlotLimit exhaustion proves its capacity write, uninitialized size, and owning CloseChecked cleanup. Scratch spans do not write source m_size. Signed64/uint64 setters initialize both digits before final physical size publication. The actual six constructor objects retain source allocation order and final tmp0/two64 swap. Indexed arithmetic/division/decimal views read the physical object, with rounded-object initialized-word checks. Genuine Natural Words scratch remains live (four constructor buffers) until its mandatory inline16 replacement; no source-completion claim is made for that intermediate topology.

Owner-itself error methods preserve original and cleanup causes without allocating another exception header. CloseChecked consumes real cells/pending objects/allocator roots, and only an owned Runtime is closed. Tracked release flags distinguish completed physical/link prefixes from retryable pending release. SourceFinalize, exact source teardown order, full Natural/output/transitive teardown, shared physical adoption/global bindings are still required and not implemented by this draft.

Six new named groups execute native and checked/debug: TestIntegerMandatoryCellConstructorTopology, TestIntegerMandatoryDefaultRuntimeOwnership, TestIntegerBackendScratchHeaderAndSpareDigit, TestIntegerBackendConstructorReservationSweep, TestIntegerBackendPooledWarmReuseNoAllocations, TestIntegerBackendPhysicalPrefixBeforeDescriptorLimit. The warm pool path holds physical/source counts and Budget charges constant with FailAfter(0), and measures zero Go allocations. Terminal selected logs: .cache/mpz-backend-five-native.log (first five), .cache/mpz-backend-six-debug.log (all six). They do not establish full package GREEN.

Actual existing-test RED logs .cache/mpz-backend-math-first.log and .cache/mpz-backend-existing-full-red.log expose old cell-Words topology and reservation assumptions. Only the individually reviewed fixture regions listed below have been migrated; the remaining RED assertions are retained. Physical source malloc refusal reports memory.Failure(Memout,101), while cold metadata refusal remains arena.ErrInjectedFailure. Warm pooled allocation can make no reservations; old fixtures need real descriptor/directory/direct allocation growth rather than assuming FailAfter(0) always rejects a recycled cell. Integer CopyFrom/FastFloat conversion fresh-cell sites received coordinator approval and now explicitly write size before copying digits; old-free errors are propagated. Existing cross-owner copy and FF→Integer value fixtures execute GREEN. Per-descriptor unpublished and cleanupPending scalar flags retain real failure roots in the charged descriptor owner; they do not represent source allocation events or physical-release authority. Full-suite success and final freeze are not yet claimed.

Interim review snapshot: .cache/release/mpz-cell-checkpoint-0c9d695 (archive plus ONLY the 11 owned Go files), with exact inventory .cache/mpz-cell-checkpoint-files.txt and SHA list .cache/mpz-cell-checkpoint-sha256.txt. Selected nine new/approved groups execute native GREEN in .cache/mpz-cell-checkpoint-nine-native.log and checked/debug GREEN in .cache/mpz-backend-nine-debug.log. Full module vet session99715 terminal0, .cache/mpz-cell-checkpoint-vet.log. Existing mathematical CopyFrom/FF fixture correction log .cache/mpz-backend-caller-green.log. The 97-group broader native/debug runs are terminal RED (sessions71304/37582), .cache/mpz-cell-checkpoint-97-native.log and -debug.log; they include held fault fixtures/private-header setup assumptions, so they are not claimed as successful full gates. Only ConstructorOwnership, ExpectFreed, TmpFree error profile split and ScratchSpan tail were individually authorized to migrate. Other tests remain unchanged and their required migration remains visible.

Final interim source/no-generics audit session79079 terminal0 on the exact 11-Go snapshot, .cache/mpz-cell-checkpoint-source.log (quiet success). Final full module vet terminal0 on the same snapshot, .cache/mpz-cell-checkpoint-final-vet.log. Parent requested holding all 11 Go files while reviewing/saving this explicitly unfinished checkpoint; no additional fixture change has been applied. Broader native/debug RED evidence remains recorded and source completion/full package GREEN are not claimed.
