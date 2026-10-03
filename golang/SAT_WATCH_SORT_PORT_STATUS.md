# Stable watch sorting implementation

Read solver.cpp1251–1255: sort_watch_lits calls std::stable_sort on each logical watch_list with watched_lt. Full watched_lt in sat_watched.h has THREE equivalence classes: binary < ternary < combined {clause,external}; clause/external original interleaving must survive. The former SortByKind insertion helper preserved semantic order but had quadratic moves. This packet replaces it with the concrete backend below. No other solver body or source global memory-counter equivalence is claimed.

Implemented concrete backend: recursively stable-partition the logical range for binary, then partition the remaining suffix for ternary. Each leaf invokes actual satWatchedLess with a legal ternary/clause-kind pivot, so classification uses the exact source comparator. Recursively combine true/false halves by three in-place reversals, yielding O(NlogN) record moves and O(N) comparisons over two passes; no heap array, interface/generic sort, per-record closure or hidden scratch owner. Preserve all logical watch objects/lifetime bits and source erased live tails outside size. Existing actual backing captures/iterators stay valid because sorting only assigns existing slots. Guard invalid/unconstructed accesses at actual read/write, retaining completed permutation prefixes.

Implementation ownership: new sat_solver_watch_sort.go/test plus replace only existing SortByKind's quadratic body with delegation; no other watch/core/model/shared files. Existing store/config/Budget/codec/generation contracts unchanged. A private cold satWatchSortWork{Comparisons,Moves uint64} optional caller-owned pointer instruments actual comparator calls and successful slot assignments; default SortByKind passes nil, no retained callback/state or extra owner charge. Counter arithmetic is checked by bounded source capacity and O(NlogN) work (well below uint64); caller instrumentation is diagnostic, not a source allocator/event counter.

Tests FIRST and independent coverage: large reverse/mixed-kind fixtures carrying distinct scalar val1 and full clause locator identities; independently append-classify oracle preserves THREE classes and combined clause/external order. Instrument actual moves/comparisons and assert finite O(NlogN) bounds rather than elapsed time. Show insertion's reverse-class worst-case exceeds the bound in behavioral RED before replacement, alongside missing helper API RED if instrumentation is needed. Verify already sorted/all-equivalent/empty/singleton, retained erased tail and old iterator behavior, zero warmed Go allocations under armed Budget, and read/lifetime/closed guards. Full native/checked-debug, actual named race/JS, vet/WASI/no-generics isolated latest HEAD gates after freeze.

Selected storage adaptation: standard-library stable sorting's unobservable implementation may allocate scratch or choose an in-place fallback. This concrete always-in-place backend preserves observable stable ordering and meets the source asymptotic contract, without claiming libstdc++ exact malloc counts or comparison sequence. The actual source comparator has no observable callbacks/side effects. Full solver method must later call this real backend on its real outer watch owner; this packet does not construct a fake solver facade.

Source hashes:
- src/sat/sat_solver.cpp:ea35f45c69be4d179aec4e1af034b2295f2a026bb01c37a2d9f8c59abf519655
- src/sat/sat_watched.h:1d2969f295a5f9ab042c5997fdc0717dc6cfe74711c6037802a780c78c72bf28

Tests-first evidence: the named helper was missing and compilation failed before production. The original insertion body was then extracted unchanged into the instrumented helper, and TestSATSolverWatchSortLargeStableOrderAndWorkBound failed behaviorally: actual 1,050,622 comparator calls and 1,050,623 successful assignments versus limits 4,096 and 98,304. The replacement passes these independent work bounds. No timing threshold or manufactured failure was used.

Four new groups (there are no newly registered source tests/ENSURE assertions for this external sorting call):
- TestSATSolverWatchSortLargeStableOrderAndWorkBound: 2,048 reverse-class records; full independent stable-class oracle including combined clause/external interleaving and scalar identities.
- TestSATSolverWatchSortThreeClassesAndActualLifetime: all three classes, erase-retained tail unchanged, old iterator follows moved backing and observes sorted records, later Pop ends only its actual slot lifetime.
- TestSATSolverWatchSortIndependentMixedOracleAndWarmAllocation: deterministic 513-record mixed exact oracle, measured comparator/assignment bounds, zero warmed Go allocations with Budget.FailAfter(0).
- TestSATSolverWatchSortBoundaryAndGuardPrefixes: empty/singleton/all-equivalent boundaries; equivalent classes make no assignments; actual uninitialized-slot and closed-owner guards.

Method/source map: SortByKind delegates only to this backend. sortByKindWithWork validates the actual owner/list, uses the existing exact satWatchedLess (sat_watched.h117–126), partitions binary records before a legal ternary pivot, then ternary records before a legal clause pivot. stableWatchPartition recursively joins partitions with reverseWatchRange's three scalar reversals. Source solver::sort_watch_lits (sat_solver.cpp1251–1255) still needs future integration into the full solver's actual outer watch owner. No other solver method is claimed implemented. Logical records are assigned through existing Read/Write; sorting does not change capacity, descriptor identity, bitmap, generation, owner charge, or erased live tails. Actual guard failures retain already-completed assignments; no rollback is invented.

Work bounds: each partition examines every leaf once, so at most 2N comparator calls. Each level rotates disjoint ranges with at most 2N successful slot assignments; two passes cost at most 4N*(ceil(log2 N)+1). Recursion is logarithmic stack depth; no heap scratch/callback closure/interface or retained diagnostic allocation. Empty halves skip rotations. Clauses/external records never receive separate classification, preserving their mutual equivalence. This is a selected standard-library backend adaptation and makes no libstdc++ physical-allocation/comparison-order equivalence claim.

Frozen integration snapshot: golang/.cache/release/sat-watch-sort-final, HEAD d25fd729dfb63401f763515d9dd3132bce04f68b plus ONLY the three Go files below. The model-copy commit is included; root's later literal-set packet is separate and requires final parent integration. All seven gates terminal exit0:
- Full native ./...: handle34492, .cache/watch-sort-native.log.
- Full checked-debug ./...: handle14131, .cache/watch-sort-debug.log.
- Named 25 combined watch groups checked-debug race: handle71985, .cache/watch-sort-race.log, PASS 1.452s.
- Actual JS checked-debug named25: handle90420, .cache/watch-sort-js.log, PASS 0.310s.
- Full checked-debug vet: handle24200, .cache/watch-sort-vet.log.
- Full checked-debug WASI build: handle33323, .cache/watch-sort-wasi.log.
- Source/no-generics audit: handle2044, .cache/watch-sort-audit.log.
Each of all25 test names has explicit RUN/PASS in both race/JS logs; .cache/watch-sort-names.json and .cache/watch-sort-regex record exact inventory. Existing committed watch tests/status were not edited. Only SortByKind delegation changed in the committed store file; no model/shared owner/metadata/Git edits.

Frozen SHA256:
- sat_solver_watch_store.go: 4de59e8ee0409f106530f99cb96299cf0f22cb53fcb385b42b5aeef6921f7992
- sat_solver_watch_sort.go: 1f441db9c311bbe1e99a4dfaa0a77b685b862752495fc18dccd743137d7f6479
- sat_solver_watch_sort_test.go: 3320d9e3a9190b42fac3b53ebcd82293d63782961cfd7c61188e680511959dde

Full SAT search/propagation/conflict/backtrack/inprocessing/extension and real runtime physical allocation hooks remain required separately; this packet completes the efficient concrete stable-watch backend only.

## Coordinator integration

All three live/frozen hashes matched. Prospective committed tree
.cache/review/sat-watch-sort-integration-01cc137 contains HEAD01cc137 plus
only these three Go changes and final metadata. Fresh full native PASS
(root2.395s, mpnum22.847s), fresh full g3checked+g3debug PASS
(root2.686s, mpnum23.500s), verifying interaction with the new literal set.
The coordinator checked every25 exact race and actual JS RUN/PASS names
in the immutable original gate logs. Source/no-generics inventory remains
33ported/87partial/1982unported, with full solver source methods still open.
No unrelated dependency-reference or modular-numeral draft is in this tree.
