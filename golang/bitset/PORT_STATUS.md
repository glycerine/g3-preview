# Dense bit vector: initial utility port

Owned production file: `bit_vector.go`. The only production dependency is the independent utilhash package. Source: `src/util/bit_vector.h`, `src/util/bit_vector.cpp`, Leonardo de Moura, Microsoft 2006. Copyright/license notices are preserved.

| Source operations | Go counterpart |
| --- | --- |
| default/reserve/word constructors | zero value, `New`, `FromWords` |
| copy constructor, assignment, destructor | `Clone`, `Assign`, Go owner release |
| expand_to, expand, capacity | private `expandTo`, source growth in `Push`/`Resize`, `CapacityWords` |
| reset, swap, fill0 | `Reset`, `Swap`, `Fill0` |
| size, empty, num_words, get_word | `Size`, `Empty`, `NumWords`, `Word` |
| get, set overloads, unset | `Get`, `Set(index, bool)`, `Unset` |
| push_back, pop_back, back, shrink | `Push`, `Pop`, `Back`, `Shrink` |
| resize, reserve, shift_right | `Resize`, `Reserve`, `ShiftRight` |
| equality/inequality, OR/AND, neg | `Equal`/its negation, `Or`, `And`, `Neg` |
| display, contains, get_hash | `String`, `Contains`, `Hash` |
| fr_bit_vector reset/fill0/set/push_back | `FastReset.Reset`, `Fill0`, `Set`, `Push` |

Ownership: words and the fast-reset index vector are owner-local pointer-free uint32 backing slices. Assignment/clone explicitly copy all source capacity words; Go struct assignment is documented as aliasing, and must not substitute for these source operations. Logical Reset retains capacity and clears every capacity word. FastReset.Reset keeps logical size and clears the recorded one-bit indices in insertion order, retaining duplicate entries until reset, just like source. GC replaces physical deallocation; there is no logical reference counting in this source class. Shrink/Pop preserve stale words and resize/bitwise operations use the original masks. Source capacity schedules are retained. Go additionally diagnoses invalid bit indexes, empty pop/back, invalid shrink, and size overflow instead of allowing a corrupted slice/access; num_words rounds via uint64 to avoid source addition overflow on extremely large bit counts.

Source behavior caveats deliberately retained: Contains examines only the receiver's length and requires the other vector's capacity to cover that many words; it ignores excess other bits. Hash hashes only floor(size/8) bytes, so incomplete trailing bits do not participate. The byte representation is the utilhash package's fixed little-endian, signed-char reference baseline. WordBytes preserves source transitions with zero allocation. ShiftRight inserts low zero bits and raises original bit indices; the name comes from source despite the numeric shift direction.

Every assertion in reachable `src/test/bit_vector.cpp` methods is translated: resize crash regressions, all six OR cases, both AND cases, reset/growth for lengths 1–83, and bit-31 equality. The print-only shift sequence has explicit expectations after shifts 16,1,32,42,16,63. Formerly unreachable tst2 and tst1 are also ported; tst1 uses fixed independent PCG choices and a bool-slice oracle instead of platform libc rand. A 5,000-step bounded independent bool-slice state machine covers resize/shrink/pop/push/shift/neg/reset/fill/and/or. Additional cases cover all constructors, capacity growth, clone/assignment independence, display/words, contains, fast reset, source hash byte rounding, empty/longer AND and warmed allocation behavior. Source hash byte fixtures are embedded from `test_vectors/source_hash.json`; their maintainer-only generator is `hash_reference.cpp`. Tests never require C++ or source files. No permanent fixture is under testdata.

TDD evidence: tests written first failed to compile on missing Vector/FastReset; the source translation then passed. Native CGO-disabled/g3checked tests and vet pass; actual js/wasm tests pass with Node v22.22.2 and the toolchain's official wrapper in a minimal environment; both wasm targets build. Host toolchain is go1.26.4-pont-onethread. WASI execution hits its pre-main runtime failure documented in utilhash/PORT_STATUS.md; no WASI semantic execution claim is made. Warmed bounded Hash/Or/And/Neg/Reset/Resize collectively allocate zero Go objects (AllocsPerRun test); no unmeasured speed claim.

All methods of the two assigned source classes have counterparts. The following fixed-bit-vector packet is mapped below; sparse bitsets and other source containers remain separate unported components. Contains with insufficient other capacity remains an invalid source precondition, not a supported mathematical subset operation.

## Fixed bit vector and manager packet

Added `fixed_bit_vector.go` and `fixed_bit_vector_test.go`, translating `src/util/fixed_bit_vector.h/.cpp`, copyright Microsoft 2014, authors Nikolaj Bjorner and Leonardo de Moura. This supersedes the earlier fixed-bit-vector unported status; sparse bitsets and other containers remain separate work.

| Assigned source method | Go counterpart and evidence |
| --- | --- |
| fixed_bit_vector constructor/destructor; manager constructor/member allocator destructor | `NewFixedManager`, pointer-free `FixedRef`, `Close`; source bit width, word count, byte count, mask and shared zero-width m_0 are represented |
| fixed get_word/get/operator[] | manager `Word`/`Get` (owner resolves compact refs) |
| fixed private set/unset/set(bool) and manager overloads | `Set(ref,index,bool)`, `Unset` |
| fixed set(other,hi,lo), manager overload | `SetRange`; aligned word path and residual bit path, unaligned bit path, source forward overlapping self-copy order |
| allocate(), allocate0(), allocate1(), allocate(existing) | `Allocate`, `Allocate0`, `Allocate1`, `Clone` |
| deallocate(), reset() allocator | `Deallocate`, manager `Reset`, `Close` |
| copy(), reset(vector), fill0/fill1 | `Copy`/`Assign`, `ResetVector`, `Fill0`/`Fill1` |
| set_and, set_or, set_neg | `And`, `Or`, `Neg` |
| num_words, num_bytes, num_bits, get_mask | `NumWords`, `NumBytes`, `NumBits`, `Mask` |
| last_word, equals, contains | `LastWord`, `Equals`, `Contains`; masked last word retains all source unused high bits in storage |
| hash, display | `Hash`, `String`; exact floor(numBits/8) bytes with init seed numBits, MSB-first rendering |
| raw words accessible to source tbv_manager friend | manager-only `WriteWord`; no arena slice borrow escapes |

`FixedRef` is an owner-local arena slot/generation token. There is no cross-owner token detection claim: each call must use the manager that owns its token. Both zero-width Allocate0 and Allocate1 return the same zero ref, representing the constructor-owned source m_0. Deallocate of that root does nothing, manager Reset leaves it available, and Close ends the owner lifetime. The embedded zeroWord field preserves the manager-owned source root budget/storage even though no valid bit addresses it. Nonempty allocation creates a logical vector; Clone copies all storage words, not just active bits. Source Allocate leaves words unspecified; arena allocation supplies zero words, while callers requiring definite source zero/one initialization still use Allocate0/Allocate1. This is an uninitialized-storage adaptation, not a replacement for a source initialization action.

The manager owns an `arena.Words` pool of pointer-free words and descriptors, with explicit source allocation/deallocation actions and generation invalidation. It allocates no per-vector object or per-bit object. Reset releases arena payload pages and invalidates live nonzero handles while preserving the arena generation ledger; Close releases pages/ledger and the manager root charge. Its own header, including the embedded zero-width root, is charged before construction and rolled back on invalid pool configuration. Arena accounting includes backing pages/metadata/directory; the arena API currently excludes its own cold Go allocator owner headers from accounting. That substrate limitation remains recorded rather than using reflection into private arena structures. No manager hash scratch buffer exists: approved `utilhash.WordReader` feeds source word bytes through a checked manager accessor. The accessor's closure and warmed Hash/Neg/Allocate1/Deallocate sequence allocate zero Go objects, verified with AllocsPerRun.

Source allocator replacement: `small_object_allocator` byte chunks, pointer-linked free lists, address-sorted consolidation, debug Valgrind bypass and raw malloc/free are represented here by the agreed typed owner-local pooled arena. The general small_object_allocator implementation, its allocation-size/debug leak diagnostics and consolidation methods are not claimed ported by this packet. One source defect to retain in the inventory: allocator reset/destruction frees small-object chunks but does not track live separately malloc-allocated large blocks; this packet supplies explicit Go owner cleanup and does not reproduce physical leaks. Manager Reset invalidation applies uniformly to owned vectors. Logical operations, masks, initialization calls, deletion calls and traversal ordering follow the source.

All active tests from `src/test/fixed_bit_vector.cpp` were written before implementation and observed RED on missing manager/ref methods: tst1, tst_or, and tst_eq widths 15/16/17/31/32/33/63/64/65. The source tst_and is empty; added independent AND expectations via a bounded 600-step bool-slice oracle at widths 0/1/17/32/33/65/101. Additional tests independently check last-word masks and all word sizes, empty shared-root identity, exact source forward self-alias range semantics, clone/assign independence, stale generations after deallocation/reset, Close, bounds, constructor/failure budget rollback, allocation failure preserving live vectors, and bounded pool reuse. C++ hash expectations in embedded `test_vectors/fixed_hash.json` cover zero width and every floor-byte boundary through width 65; the recorded source input generator is `fixed_hash_reference.cpp`, compiled using actual hash.cpp with GCC 13.3.0. Ordinary tests require neither C++ nor repository source.

Final packet validation: CGO-disabled native and g3checked package tests, vet, and actual js/wasm tests pass using the installed go1.26.4-pont-onethread toolchain and Node v22.22.2. wasip1/wasm packages compile. The previously verified custom-Go pre-main WASI seed failure remains an execution limitation; this packet does not modify or bypass the installed runtime. Source-method audit covers all operations of the two assigned fixed-bit-vector classes; general allocator internals remain inventoried separately.

Width bound: fixed-manager constructors reject numBits > UINT32_MAX−31 with arena.ErrBounds before construction/budget changes, because source `(num_bits+31)/32` otherwise wraps to a broken zero-word allocation while reporting a nonzero bit width. A finite metadata-only boundary test covers the highest non-overflowing width (4294967264 bits, 134217727 words) without allocating its payload. This is an explicit invalid-size guard recorded alongside the source overflow defect.

No-generics conversion: constructor header accounting now uses reflect.TypeOf(FixedManager{}) rather than the generic reflect.TypeFor helper. FixedRef/Words APIs retain their concrete signatures while the arena owner replaces its underlying generic storage. No Go generic declarations or instantiations remain in this packet.
Concrete conversion validation: bitset tests pass against the concrete arena Words implementation on native, checked-tag and actual js/wasm execution; vet and WASI cross-build pass. The existing source assertions, finite masked/hash fixtures, bounded bool-vector oracle and zero-allocation pool checks remain enabled.

## uint_set packet

Added `uint_set.go`/`uint_set_test.go`, translating all declared operations in `src/util/uint_set.h`, copyright Microsoft 2006, Leonardo de Moura (2006-12-07). This is the dense unsigned-integer utility set; SAT-local dense-set classes are separate components. Its storage is an owner-local slice of pointer-free uint32 words. No Go generics, interfaces for element storage, unsafe, maps, or production random generator are used.

| Source operation | Go counterpart / contract |
| --- | --- |
| default constructor / destructor | zero value `UintSet`, explicit `Close` releases backing storage |
| copy constructor | `Clone`, source push-back loop preserving logical words and deriving capacity anew |
| implicit source assignment through unsigned_vector | `Assign`, deep copy with source capacity, self-assignment guard |
| swap, reset | `Swap`, `Reset` retaining capacity and zero logical length |
| get_max_elem | `MaxElem`, **exclusive** 32*logical-word-size endpoint, not maximum member and not backing capacity |
| empty, insert, remove, contains, num_elems | `Empty`, `Insert`, `Remove`, `Contains`, `NumElems` |
| operator OR/AND | `Union`, `Intersect`; union grows to source.size()+1, intersection shrinks to source logical size |
| equality/inequality, subset_of | `Equal`/its negation, `SubsetOf`; trailing zero words do not affect equality |
| begin, end | `Begin`, `End` constructing value iterators |
| iterator construction, scan_idx, scan_word, scan | `iterator`, `scanIndex`, `scanWord`, `scan`, original word/bit traversal order |
| iterator dereference/equality/inequality | `Value`, `Equal`/its negation; source compares index only even across different sets |
| iterator prefix/postfix increment, assignment | `Next`, `PostNext`, `Assign`; postfix returns old iterator value |
| get_hash, hash/eq functor wrappers | `Hash`, concrete `UintSetHasher`/`UintSetEqual` |
| ostream display | `String`, original ascending loop and brace/comma format |
| inherited physical vector capacity inspection | additional `CapacityWords` for storage audit |

Ownership/capacity fidelity: source unsigned_vector growth starts at 2 words, then (3*old+1)>>1 and expands repeatedly until the requested logical word size fits. Go allocation follows that schedule instead of append's unspecified capacity schedule. Reset and intersection retain physical capacity; growth zero-initializes newly logical words, so stale removed/reset members do not return. The source uint_set copy constructor pushes only logical words, while its implicitly generated assignment copies the base vector's capacity; both behaviors and independence are tested separately. Close maps the source backing-store release to Go owner cleanup. Copying a Go UintSet struct is documented as aliasing and must not replace Clone/Assign. Iterators borrow the owner and require no set mutation during traversal. Invalid end increments panic rather than proceeding past the source invariant.

Numerical fidelity: every membership word/mask and hash accumulator is uint32. Source NumElems invokes GCC __builtin_popcount; Go's bits.OnesCount32 is the equivalent platform population-count primitive. Source Hash is h += (i+1)*word[i] modulo 2^32; it does **not** call the generic utilhash string/combine functions, so this packet has no endian/char hashing adaptation. Weighted hash overflow and trailing-zero invariance have exact finite tests. Source get_max_elem overflows when logical length reaches 2^27 words, breaking iterator/display bounds. Insert/Union diagnose such invalid extents before changing/growing storage instead of silently hiding members; the guard and preservation of existing membership/capacity are tested using UINT32_MAX without giant allocation. The source vector's allocation-size/capacity overflow is also diagnosed.

Source tests: `src/test/main.cpp` registers TST(uint_set), `src/test/CMakeLists.txt` includes uint_set.cpp, and **all** tst1/tst2/tst3/tst4/tst5 calls in that file are active. Tests were written first and observed RED on undefined UintSet operations; translation then reached GREEN. tst1's exact assertion categories (count, empty, every membership) use deterministic independent PCG choices with the original 100 repeated bounded domains and fewer than 1000 operations each. tst2 preserves 1000 repetitions for each domain 1/10/31/32/100 and all 100 singleton probes. tst3 translates every assertion at n=12 and n=100, including duplicate insertion, subset direction, copies and singleton-subset equivalence; tst4 covers bit 32. No source uint-set tests are disabled.

`tst5` contains print-only hash observations and a uint_map< unsigned > integration demonstration. Its hash/render observations have embedded finite expectations in `test_vectors/uint_set_prints.json`, independently calculated from source masks and the weighted formula: hash sequence 0,2,6,8198,8214. The value-key copy independence is tested, but the source map insertion, bucket traversal and key/value iteration demonstration remains explicitly **unported map integration**; a UintSet port does not claim a map implementation. No assertion was weakened or dropped to hide that prerequisite. The source key map's iteration order is not replaced by a Go map.

A separate 3,000-step independent bool-array set oracle checks membership, cardinality, empty/equality/subset relations, union/intersection/reset/assignment/swap and complete ascending iteration. Additional deterministic cases cover all iterator operations (including equality across owners with equal index), wrapper functors, source union's extra logical zero word, retained capacity, cloning vs assignment, swap/reset/Close and weighted hash wraparound. Warmed bounded insert/remove/count/hash/union/intersection/iteration allocate zero Go objects. Ordinary tests embed permanent fixtures, never compile C++ or import the repository source, and use no testdata files.

Final packet checks: CGO-disabled native tests, g3checked tests, vet, actual js/wasm execution with Node v22.22.2, and WASI cross-build pass. The installed custom Go runtime's previously documented WASI startup failure remains separate from package execution. Every method in the assigned uint_set header has a counterpart or explicit operator-expression equivalent. Generic source map integration and other unsigned_vector/container operations remain separate production coverage work; they are not claimed complete by this packet.

## UintSet-key map integration packet

Added `uint_set_map.go`/`uint_set_map_test.go`, the concrete specialization of
`map<uint_set, unsigned, uint_set::hash, uint_set::eq>` in `src/util/map.h` and
`src/util/hashtable.h`. This is the actual local `uint_map<unsigned>` subclass
in **active** `src/test/uint_set.cpp::tst5`, also used as state-set to state-ID
storage in `src/math/automata/symbolic_automata_def.h`. It supersedes the earlier
**unported tst5 map integration** statement above. It is distinct from the
identity-key `u_map<unsigned>` in utilhash and the still-unported direct-index
`src/util/uint_map.h` pointer-value container. This specialization does not
claim the complete general template family or the symbolic automata algorithm.

| Concrete source operation | Go counterpart |
| --- | --- |
| table2map/map/core construction, copy construction, destructor | `NewUintSetMap`, `Clone`, `Close` |
| alloc_table/delete_table, copy_table/move_table | `allocateTable`, `destroyTable`, `copyTable`, `rehash`; scalar table with distinct owned key copies |
| default_hash_entry state/hash/data methods | scalar `setMapEntry` fields, key assignment helpers, ref `Key`/`Value`/`SetValue`; source FREE/DELETED retain key ownership |
| key_data default/key/key-value construction | empty scalar key, `copySet`, wrapper temporary and scalar value |
| hash/eq functors and cached hashes | `setMapHash`, `setMapEqual`; uint32 weighted source hash and trailing-zero equality |
| insert rvalue/const data | `insertCore`/`insertConst`; concrete wrapper `Insert` |
| insert_if_not_there_core / insert_if_not_there / insert_if_not_there2 | matching exported methods; one / two / two key-copy temporaries respectively |
| find_core, find output, get default, required find/operator[] | `FindCore`, `FindInto`/`Find`, `Get`, `MustFind`/entry `SetValue` |
| contains, find_iterator | `Contains`, `FindIterator` |
| remove, erase, cleanup, growth | `Remove`, `Erase`, `removeCore`, `rehash`, `prepareInsert` |
| empty, size, capacity, collision count | `Empty`, `Size`, `Capacity`, `CollisionCount`; statistics disabled in source baseline |
| reset, finalize, swap | `Reset`, `Finalize`, `Swap` |
| source copy assignment and core union/intersection | `Assign`, `Union`, `Intersect`; source self guards and ascending-bucket traversal |
| begin/end, scan, dereference, prefix/postfix increment, equality | `Begin`/`End`, iterator `scan`, `Ref` and key/value accessors, `Next`/`PostNext`, `Equal` and its negation; ordinary cold iterator value assignment |
| checked invariant | `CheckInvariant`, source state counts plus key/hash/ownership checks |

`core_hashtable::dump` is not exposed by table2map, and cannot be instantiated
for this source specialization: `_key_data<uint_set,unsigned>` has no ostream
operator. No invented formatter is claimed as that method. The real source
tst5 prints the key and value separately through iterator access, which is
translated exactly. Mutable raw source key/pointer borrows become readonly
`UintSetMapKeyView` and owner/epoch/slot refs; explicit `CopyKey` returns a
caller-owned set. Structural mutations invalidate refs/cursors, while value or
equal-key updates preserve the slot and its view resolves the current payload.
Every callable method of this assigned concrete map is represented.

Key storage contains no per-key objects, Go pointers, interfaces, unsafe, or
Go maps: each 28-byte scalar entry holds state/hash/value plus arena slot,
generation, **logical word length**, and **source capacity**. Arena payload
length is the source capacity, so copy assignment at capacity 7/logical length 1
allocates all seven source words even if the arena capacity class rounds to 8.
Only active words are copied; source spare words are unspecified and Go supplies
zero storage. Empty copy-constructed keys use the zero ref and are distinguished
from FREE entries by the state tag. Weighted hash is h += (i+1)*word[i] modulo
2^32; no endian or signed-char adaptation applies to this specialization.

The source uint_set custom copy constructor pushes logical words and derives
capacity; its user-defined copy constructor/destructor suppress move operations.
Consequently synthesized `_key_data`/default_hash_entry move assignment calls
uint_set **copy assignment**, preserving source capacity. An assembly-only audit
using the original headers verified that call (artifacts under golang/.cache).
Rehash/Clone allocate distinct USED key payloads while every old USED/FREE/
DELETED payload remains owned until table destruction. Removal and nonshrinking
Reset only mark states; even an empty-clean Reset retains FREE payloads.
Overwriting a slot destroys its former key. Insert/lookup/remove wrappers create
and destroy all source temporaries; insert-if-absent wrappers preserve both
copies. Internal intersection uses cloned key_data directly rather than adding
table2map temporary copies that the source does not make. Swap exchanges table
and pool ownership without key copies or destruction.

Budget accounting includes map root, scalar arrays, Words/descriptor cold
headers, payload and all pool metadata/directories, with uint64 charges. This
depends on the arena owner-header and atomic `Words.SwapBudgets` packets; their
current implementations also supersede the older fixed-manager paragraph's
substrate header-accounting limitation. Cross-budget Swap reserves both incoming
table charges first, then uses the atomic complete-pool graph hook, refunds
table reservations on pool failure, and exchanges pointers only after success.
Owner root charges and cold callbacks stay with their owners. Tests exercise
failure in each destination's table/pool preparation and nil/non-nil budgets.
Physical arena free pages may cache storage, but descriptor liveness follows
every source logical deletion. Explicit Close refunds all graph charges.

Typed arena errors replace unchecked source allocation/size failures. Constructor,
key-copy assignment, table rehash/Clone, and table replacement allocate before
destroying old storage to preserve existing contents on failure; this adds a
documented old+new replacement peak where source vector assignment/reset would
destroy old data first and could leave unsafe state after allocation failure.
Successful pre-insert rehash remains applied if a later stored-key copy fails;
multi-entry Assign/Union and Remove with failing cleanup are source-ordered
partial operations, not transactions. Pool growth may retain reusable physical
pages after failed operations; tests require no surviving temporary descriptors
or final budget leaks. uint32 iteration extents and table sizing diagnose overflow.

Tests first obtained RED on the missing concrete API, then GREEN after source
translation. Additional iterator postfix and cross-budget swap cases were RED
before their implementations. tst5 has the complete independently computed
finite stdout expectation: hashes 0/2/6/8198 and bucket-order lines
`{1, 2, 44} : 1` then `{1, 2, 4, 44} : 3`. No source assertions are disabled in
that test. A 700-operation independent eight-member mask oracle checks mappings
and complete iteration; test-only Go maps serve as the mathematical oracle.
Additional coverage checks strict 3/4 growth before lookup, last tombstone reuse,
FREE/DELETED retained owners, strict cleanup thresholds and OOM callback, one-step
reset shrinking and 64-entry finalize, source capacity vs logical/physical size,
clone/assignment lifetimes, exact wrapper temporary counts, injected constructor/
insert/lookup/rehash/swap failures, alias guards, stale borrows and caller copies.
Warmed Find/Insert allocates zero Go objects in AllocsPerRun. Ordinary tests need
no C++ tools, source headers, testdata, or platform addresses.

Native, g3checked, vet, actual fresh-count js/wasm tests under Node v22.22.2,
and wasip1/wasm cross-build pass. WASI execution retains the documented installed
custom-Go runtime pre-main seed limitation; there is no WASI execution claim.
