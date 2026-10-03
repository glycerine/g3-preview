# Mandatory MPZ/Natural backend implementation progress

Full mandatory backend implementation is authorized and ongoing. The existing IntegerManager/NaturalManager STILL use their arena implementation; the following new concrete helpers are prerequisites, not an optional second instantiated backend or completed integration. No manifest/progress/Git changes were made. Cross-manager physical adoption, process-global runtime bindings and the full solver remain required/open.

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
