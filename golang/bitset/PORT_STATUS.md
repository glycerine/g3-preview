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

All methods of the two assigned source classes have counterparts. `fixed_bit_vector.*`, sparse bitsets and other source containers are separate unported components. Contains with insufficient other capacity remains an invalid source precondition, not a supported mathematical subset operation.
