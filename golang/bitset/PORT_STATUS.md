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
