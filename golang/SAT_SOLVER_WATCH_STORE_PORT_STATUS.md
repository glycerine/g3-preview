# Solver common watched-list storage

This independently testable prerequisite implements real source watched-list operations using one owner-local concrete payload service. It is not the SAT solver, its outer watches vector, an extension, global memory runtime, or a full vector-template port. Full solver search/propagation and its registered user-scope test remain required. Only new `sat_solver_watch_store.go`, its tests and this status are owned here; committed clause/watch/arena/model/set utilities remain unchanged.

## Source and executable map

Read the full preserved `sat_watched.h/.cpp`, `vector.h` lifetime/copy/move/growth bodies, actual solver watch consumers and model-converter header. No standalone watched test registration or source ENSURE fixture was found under src/test. The named Go groups below are independent/source-order regressions, not fabricated source test assertions. Existing concrete watched constructors/getters implement their source SASSERT sites and are reused, including the external getter defect (value3) and debug roundtrip rejection of non3 external indices.

| Preserved source | Store implementation |
| --- | --- |
| vector.h47–124, destructor/finalize/reset215–226 | NewList, ResetList, Finalize, Free, Reset, Close; logical destructors clear each slot lifetime, physical finalization frees both payload and bitmap. Reset retains erased live tails beyond logical size; Finalize keeps the descriptor object alive but ends backing. Free increments/retains generation or retires exhausted slots. |
| vector.h60–100 expansion | ReserveCapacity expands 2,3,5,8,... with source unsigned byte-overflow domain (watched size8, two uint32 header fields). It copies the FULL old payload capacity and FULL bitmap, preserving erased live tails and destroyed holes. Each completed expansion remains published when a later expansion fails. |
| vector.h102–118 copy_core,161–214 ctor/assign/move | CloneList/CopyList allocate exact source capacity and construct logical size only. AssignList destroys destination before preparation; copied size publishes before ascending scalar constructions, and unconstructed accesses guard at the actual slot. MoveList/MoveAssignList transfer within the common owner and leave source empty. SwapLists swaps payloads. Cross-owner CopyList/AssignList copy actual words, never foreign arena descriptors. |
| vector.h156–159 fill constructor,178–184 array constructor | NewListFill uses source geometric ResizeFill; NewListArray performs sequential Push. Failed unpublished construction cleans its actual roots (explicit Go ownership continuation); completed pooled physical allocations remain charged until Close. Default watched construction/Resize/Reserve without a fill value is ill-formed in the source because watched has no default constructor. |
| vector.h302–375 indexed/back/push/pop | Read/Write/GetDefault/Back/Push/Pop preserve logical source-debug assertions, release access to retained initialized slots, construction and destruction. Pointer accesses are represented by readPhysical and value iterators, which do not invent source operator[] assertions. |
| vector.h376–389 erase | EraseAt/Erase/EraseTernaryWatch shift and decrement without destroying the trailing watch. |
| vector.h390–434,287–301 | Shrink/SetEnd/ResizeFill/ReserveFill destroy or construct the source range. SetEnd does not borrow Shrink's allocated debug assertion; unsafe allocated increasing destructor loops get a genuine bounds guard. With null backing both methods preserve finite release no-op and their own source-debug assertion. Resize publishes complete logical size before constructing new slots. |
| vector.h436–493 append/reverse/fill/contains/swap | Append/AppendArray/Reverse/Fill/Contains/SwapLists. Nonempty same-list Append has the documented source growing-size nontermination guard before action. Empty self-append is valid. This is distinct from model_converter::copy's captured-end semantics. |
| vector.h496–528 | SetX preserves uint32 idx+1 wrap and completed resize/destruction before guarding the subsequent UINTMAX write. GetDefault skips accesses outside logical size. |
| vector.h252–285 raw forward/reverse iterators | Iter/ReverseIter return cold concrete VALUES; pointer-receiver Next retains captured end, reads actual lifetime, and guards actual scalar arena backing generations. Reset/erase/pop alone do not relocate pointers; pointer lifetime checks distinguish live erase tails and destroyed holes. Actual backing free/reallocation or StoreClose invalidates access; within-owner move/swap preserves old iterators following that same backing, including when the moved-from list object is freed. Destination destruction updates the captured bitmap at the real slot. |
| sat_watched.cpp27–46 | EraseClauseWatch shifts then SetEnd destroys tail BEFORE size publication, distinct from generic erase. |
| sat_watched.cpp48–81 | DisplayInto reads real watches and real borrowed clause allocator, writes exact binary/ternary/clause/external branches into borrowed NumericTextStream, no intermediate or hidden literal storage. Dead clause owners are checked at actual display dereference; scalar watch copies retain raw references without reading clauses. |
| solver.cpp1251–1255 watched_lt | SortByKind is an additional convenience semantic-order adapter using concrete allocation-free quadratic insertion sort. The real solver std::stable_sort backend/performance integration remains OPEN; comparator and stable order are tested, but this helper does not satisfy the source large-list complexity contract. A separately authorized concrete O(NlogN) stable partition/rotation or merge packet follows the model-owning freeze. |

Source solver consumers include mk_var139–140, binary/ternary/clause attachment231–336, propagation569–582 with source raw iterator compaction and SetEnd, gc_bin2646–2662, scope shrink2708 and copy/display loops. They remain solver algorithm work; this store does not claim those methods implemented.

## Concrete ownership and codec

`satSolverWatchListRef` is private scalar owner/slot/generation; list descriptors retain generation/retirement/free links and source length/capacity. Hot descriptors and refs are audited pointer-free. No per-list Go allocation/header or per-record closure/interface, generics, unsafe, or math/big is used. One cold store owns Vectors64 and Words, a scalar descriptor directory and monotonically unique cold owner ID. It rejects copied cold owners and is owner-confined, like Budget; no concurrent-sharing guarantee is invented.

A watch occupies four uint64 words: packed val1/val2, full clause ownerID, packed clause slot/generation, and storageKind. All scalar bits survive roundtrip even when fields are unused. The source two uint32 watch fields remain distinct from widened checked locators; the Go watch codec's physical32 bytes is not claimed equal to C++ watched size8. Source literal packing/debug sites are reused unchanged.

A separate uint32-word bitmap records each slot's initialized lifetime, including noncontiguous Erase→Pop holes and word boundaries. Growth copies all old bytes and bitmap bits, whereas copy construction initializes only logical slots. The four-word extent uses uint64 arithmetic and covers every source geometric watched capacity (source maximum `(UINTMAX-8)/8` ->2147483640 codec words), with checked real host allocation bounds supplied by arena; scalar tests do not allocate multigigabyte memory. The cold directory uses source outer eight-byte watch-list growth progression but charges its actual scalar record width and host byte bounds; it is storage metadata, not a completed outer vector.

Owner headers, both nested arena owner/descriptor graphs, record directory, actual payload/class blocks, page directories, bitmap backing and every old+new overlap are charged exactly once. `Stats().ChargedBytes` reports the real retained graph; immediate failure tests compare it against Budget.Used before teardown. Budget is borrowed, as are clause allocators and output streams. Input slices and final returned stream strings are caller-owned boundaries. Reserve/Release do not rewind the injection attempt controller. No Budget number is substituted for the source global memory manager.

Source one-array expansion is adapted into charged payload-then-bitmap preparation. The source capacity publishes only when both preparations succeed. Earlier actual arena page/directory growth remains owned/charged if the unpublished payload is freed. Source logical capacities are distinct from geometric arena size classes and retained free pages. Within-owner move/swap transfer actual payload ownership without copying; cross-owner owning moves are not exposed, because arena page graphs are common-owner resources. Full solver source uses within-owner watch moves and genuine cross-owner copies. Raw c_ptr/references are represented by scalar access/iterators; mutable slices never escape.

## Independent coverage and TDD

Tests were written before the missing API, and isolated HEAD compiled RED. Additional test-first rounds covered move/iterators, copy/assignment, helpers/accounting, real display, constructor extent/bounds, and default/back/ternary helpers. A behavioral RED additionally proved AssignList must publish full copy_core size before a guarded unsafe source construction access; the fix preserves header/first-copy prefix and accounts the actual root. These are actual missing API/behavior failures, not manufactured source assertion counts.

Current21 groups cover source capacities, high-bit codec/provenance, bitmap word boundaries, live erase tails versus Pop holes, full-capacity growth, exact-capacity deep copy, cross-owner copy and dead clause borrow, within-owner moves, source profile guard ordering/UINTMAX SetX prefix, source erase variants, stable comparator/display, generation reset/retirement/cold-copy rejection, constructor unwind and completed-expansion fault sweep. The independent 1500-step array/lifetime oracle checks every physical slot after mixed Push/Pop/Erase/Resize/Reset/Reserve. Fault tests inspect roots/bytes immediately; warm Push/Pop and genuinely escaping iterator values allocate zero under armed Budget.

## Next packet: implicit model ownership operations

Full model_converter.h has a user destructor and an implicitly generated copy constructor/assignment for m_entries; no implicit move constructor is generated, so rvalues bind copy. Entry has explicit copy constructor and no move: outer vector growth deep-copies each nested literal vector, then destroys the old entry ascending. Named Copy is a separate captured-end APPEND operation, already corrected/tested; it must not implement Assign.

Proposed Clone(config): charge a new cold owner and literal store, allocate exact source outer capacity, publish source logical size, then copy var/kind and exact-capacity nested lists ascending. Record actual constructed prefix for failure cleanup; unconstructed reads guard at actual access, not by inventing atomic rollback. Copy constructor spare capacity remains unconstructed. Proposed Assign(src): self no-op; destroy existing logical nested entries ascending, release old outer backing, then exact-capacity source copy. Destruction/allocation/construction prefixes remain observable; preserve partial roots in charged metadata and clean every owned list once. Borrowed Budget differs between owners, requiring actual nested word transfer. An explicitly documented resumable Go continuation may manage source-unsafe interrupted construction; no C++ exception recovery is claimed. Do not copy cold Go self identity or mutex/value ownership. Tests FIRST will cover empty retained capacity, independent nested roots, destination destruction-before-refusal, ascending copy/failure lifetime and exact capacities. Other class owning operations remain a separately authorized follow-up, not claims of this watch packet.

## Source hashes

- src/sat/sat_watched.h:1d2969f295a5f9ab042c5997fdc0717dc6cfe74711c6037802a780c78c72bf28
- src/sat/sat_watched.cpp:8ffc73b9e0e8cd99b2d5a663af51f3aa629f522d4bfdce6e896c066be3037d27
- src/util/vector.h:aafafc0fe6ca0afd7bafe0c781342da259f9c34c0b94b16732c415cf2733066e
- src/sat/sat_model_converter.h:760b847933f74903f81b19dcc5cd98a560f1d01407dabc8042638aaab9d01c9a
- src/sat/sat_solver.cpp:ea35f45c69be4d179aec4e1af034b2295f2a026bb01c37a2d9f8c59abf519655

## Historical 18-group validation (superseded)

Snapshot `.cache/release/sat-watch-store-final`: HEAD `a6b3cc3177babdcdb24a47b1e22194af3cc8e379` plus only the two new Go files. Exact18 inventory `.cache/watch-store-names.json`, regex `.cache/watch-store-regex`, hashes `.cache/watch-store-hashes.txt`.

All completed exit0: full native16227, full checked/debug41621, exact18 checked/debug race33338 (1.080s), actual exact18 checked/debug JavaScript46919 (0.206s), full checked/debug vet72957, WASI build62431 and source/no-generics56652. Every exact RUN/PASS name was independently verified in both actual runtime logs; no shared-cache-only execution inference. Logs `.cache/watch-store-{native,debug,race,js,vet,wasi,audit}.log`.

Historical hashes:
- `sat_solver_watch_store.go`: `707bd70562d027a4fad999de0d4edf2e5ede4aa9aac65a584cc3b5c9c9fcb6bb`
- `sat_solver_watch_store_test.go`: `dbf1c50778959cdcb468843cfb9042c2921a10c366616dbd8a1abc0006418502`

Native host and actual checked/debug JS validate the concrete architecture layouts and scalar adapters; other external stream/ABI profiles remain separately classified. No central metadata/Git/shared files changed. Full solver and global physical allocation hooks remain open, and model implicit ownership methods await the separate authorized TDD packet.

### Required fidelity corrections before final freeze

Behavioral RED proved null-backing Shrink release must be a no-op; debug still asserts its zero-count site. Null SetEnd likewise preserves its own source assertion/release no-op. Additional actual behavioral RED proved raw iterators survive within-owner moves/swaps. Iterators now capture concrete payload/bitmap generation references, original capacity and end instead of list identity. Original source list destruction after moving does not invalidate transferred backing; real destination Pop destroys its captured slot. Old destination backing freed by move assignment and real reallocation invalidate at actual access. The obsolete per-list iterator epoch and its unnecessary operation guards were removed. Source iterator metadata remains concrete by value; armed-budget escaping forward/reverse factories plus Next allocate zero. Actual finite-budget constructor/growth refusal is independently checked.

The terminal final21 freeze below supersedes historical18 results; old gates do not validate these corrected bytes. No implicit model-copy production/test edits were started while this packet was thawed.

### Final corrected 21-group freeze

Snapshot `.cache/release/sat-watch-store-corrected-final`: HEAD `d4da3755bf33343698fc3f298876d5e1de31d3f5` plus only the two new Go files. Exact21 inventory `.cache/watch-corrected-names.json`, regex `.cache/watch-corrected-regex`, hashes `.cache/watch-corrected-hashes.txt`.

All terminal exit0: full native21413, full checked/debug30605, exact21 checked/debug race41361 (1.089s), actual exact21 checked/debug JS58718 (0.206s), full checked/debug vet71344, WASI46770 and source/no-generics6365. Independently verified every exact RUN/PASS name in both runtime logs. Logs `.cache/watch-corrected-{native,debug,race,js,vet,wasi,audit}.log`.

Final frozen hashes:
- `sat_solver_watch_store.go`: `6a5debf3085b0bcb23a24ba3baed618f2e67df5908c7497d3c9811824b78be66`
- `sat_solver_watch_store_test.go`: `0fe7997d9a9a83577494f6d0a32b8f4b90b35c735f5efe2a0cec9092ba539c09`

Only source scalar payload-backed operations are claimed, with common-owner physical pooling and checked identity adaptations above. No new source ENSURE count, full solver behavior, source-global allocator counter equivalence, complete outer owning vector, or implicit model ownership method is claimed. No central metadata/Git/shared storage changes were made.

### Sorting dependency remains open

Parent review requires replacing the quadratic convenience sorter before claiming full solver sort_watch_lits integration. Current frozen storage and observable stable-order tests remain valid. The next dedicated packet will test large mixed source-kind stability and actual scalar comparison/move bounds before a concrete O(NlogN) backend; no flaky timing threshold or per-record closure is proposed. This status-only classification does not thaw the two Go files.

## Coordinator integration

Coordinator read the full selected storage/source operations and independently
matched both frozen Go hashes and exactly21 RUN/PASS names in race and actual
JS logs. Cross-review required finite null-backing Shrink/SetEnd behavior and
actual backing-following iterators across Move/MoveAssign/Swap before integration;
those changes each had behavioral RED evidence before correction.

Final source fingerprint/declaration/no-generics audit passed in the isolated
`.cache/review/sat-watch-integration-d4da375` tree with the coordinator ledger,
at33ported/84partial/1985unported. No unit advances to complete or newly partial:
the existing watched/vector units remain partial, and sat_solver.cpp/.h remain
unported. Full native/debug gates use committed HEADd4da375 plus only these
 two Go files, and all21 actual checked/debug race/JS events are verified.
The convenience quadratic sorter does not complete the real solver's sorting
backend or its complexity requirement; that work remains explicitly scheduled.


## Subsequent efficient stable-sort backend

The historical quadratic helper above is superseded by the integrated
SAT_WATCH_SORT_PORT_STATUS.md packet. SortByKind now uses the actual source
comparator's three equivalence classes with two stable partitions and scalar
rotations. Four missing-API/work-bound-first groups join the original21;
25 named race and actual JS groups and full integration suites passed.
The new backend uses O(NlogN) slot assignments, O(N) comparisons and no heap
scratch; it preserves clause/external interleaving and existing backing lifetimes.
Old storage hashes/gates above describe their original snapshot; the new store
hash and sorting source hashes are recorded in the new packet. Actual full
solver sort_watch_lits outer iteration and solver algorithms remain open.
