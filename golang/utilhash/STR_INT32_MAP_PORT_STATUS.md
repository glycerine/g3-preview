# Concrete borrowed CString → int32 map

Target: `src/util/map.h` specialization
`map<char const*, int, str_hash_proc, str_eq_proc>`, using
`default_map_entry` / `default_hash_entry<_key_data<...>>` and the applicable
`core_hashtable` algorithms in `src/util/hashtable.h`. The spellings use
`str_hash_proc` / `str_eq_proc` from `src/util/str_hashtable.h`.

This specialization is implemented; the entire generic `map.h` family remains
partial. Numeric maps, pointer identity maps, other key/value domains, and
scoped symbol maps are distinct packets. No Go generics, production Go maps,
unsafe, interface boxing, or pointers inside the hot entries are introduced.

## Source API inventory

| Source API | Concrete counterpart |
| --- | --- |
| `_key_data` key / value constructors, entry hash and equality functors | scalar `strInt32Entry` with `StrKey` / `int32`; Hash / Equal and shared CString byte readers |
| `default_map_entry` / `default_hash_entry` get/set data, cached hash, state predicates and transitions | explicit `hash,state,key,value`; entry Value / SetValue, Key / SetKey / SetKeyFrom, Hash; transitions retain complete dead/free payload |
| `table2map` / `map` constructor, core constructor/destructor | NewStrInt32Map / Close; default capacity8, optional source-core power-of-two capacity |
| implicit copy constructor, core copy_table | Clone; ascending source buckets into fresh slots using cached hash; skips dead/free payloads |
| implicit assignment / core operator= | Assign; Reset then Union; ascending source bucket order and partial insertion failure prefixes |
| reset, finalize, empty, size, capacity | Reset / Finalize / Empty / Size / Capacity |
| insert | Insert / InsertFrom; duplicate replaces both borrowed pointer and scalar value |
| insert_if_not_there_core / insert_if_not_there / insert_if_not_there2 | corresponding InsertIfNotThereCore / InsertIfNotThere / InsertIfNotThere2 and explicit foreign From variants; checked entry refs replace source pointers / data borrows |
| find_core, find output overload, get fallback | FindCore / FindFrom; Find / FindFromValue / FindInto / FindIntoFrom; Get |
| find reference overloads, const/nonconst operator[] | FindCore entry Value / SetValue; caller checks presence (source requires key present) |
| find_iterator, begin/end, iterator dereference/member access, prefix/postfix increment and equality | FindIterator / FindIteratorFrom / Begin / End; iterator Ref / Key / Value / Next / PostNext / Equal / AtEnd; mutable entry through Ref |
| contains, remove, erase | Contains / ContainsFrom / Remove / RemoveFrom / Erase / EraseFrom |
| expand_table, move_table, remove_deleted_entries | growth / cached-hash ascending rehash; strict tombstone cleanup and borrowed OOM callback |
| swap | Swap; table, directory, primary binding and membership travel together; owner/header/resource configuration stay with owner |
| core operator\|=, operator&= | Union / Intersect; intersection clones destination, queries other without adopting foreign queries, removes only absent keys, preserves values |
| check_invariant, get_num_collision | CheckInvariant / CollisionCount (zero in source-default statistics-disabled build) |

Core `dump` is not a valid instantiated operation for this map: `_key_data`
has no C++ stream insertion overload, and table2map does not forward `dump`.
The optional `HASHTABLE_STATISTICS` instrumentation is not enabled by the source
configuration and is not implemented. Arbitrary raw key mutation via source
entry references can corrupt hashing; the checked key setter accepts only
hash/equality-preserving replacement. Scalar values remain fully mutable.

## Source semantics and ownership

Seed17 CString hashing uses the established little-endian / signed-char hash
baseline, stops at the first NUL, and compares unsigned bytes for equality.
Spelling bytes are immutable borrowed regions. `StrKey` carries a scalar unique
region identity and owner-local allocation/generation/offset/prefix length;
entry arrays hold no region pointers. Primary owner binding is implicit; sorted
cold `strOwner` directory binds retained secondary origins. Queries with an
explicit foreign owner adopt nothing. Duplicate insert-if-absent retains the
old key and value and adopts nothing. Equal Insert replaces the complete pair.

Unlike ptr-based StrHashtable, default hash entries preserve **both key and
value** when marked FREE or DELETED. Reset without replacement preserves those
payloads; rehash, shrinking reset, finalize replacement and clone copy only
USED entries. Regions and their bytes are never copied, freed or closed by map
operations. Bound owner metadata remains until Close, including after removal
and reset; this is checked-borrow bookkeeping, not source spelling ownership.
Callers serialize owners and ensure retained spellings remain live/immutable.
Region rollback/reset/close invalidates affected checked key reads. A scalar
value read does not dereference the borrowed key, matching source behavior.

Linear probing wraps bucket order, remembers the **last** tombstone before
FREE, and grows strictly above75% before duplicate detection. Removal marks
FREE iff the following wrapped slot is FREE; otherwise DELETED. Cleanup occurs
only with deleted>size and deleted>64, unless the OOM callback suppresses it.
Reset counts preexisting FREE slots and shrinks once only above75% overhead
with capacity>16; finalize caps large tables to64. Rehash/clone use cached hash,
not fresh spelling hash. Assignment/union/intersection preserve source ordering
and already-published prefixes on failure.

Root object, full physical entry arrays and full physical cold directory arrays
are budgeted. Replacement arrays coexist with the old arrays while copying.
Directory growth reserves the incoming complete backing array before releasing
old bytes. Swap pre-reserves both complete incoming table+directory graphs and
rolls back the first reservation if the second fails. Clone unwinds every
failed published prefix. Predictable reset/finalize allocation failure is
staged before clearing entries, strengthening the source allocation exception
contract; cleanup failure retains the preceding source removal. Epoch exhaustion
is preflighted before structural mutation/adoption. Existing-key overwrite and
scalar mutation need no structural epoch. Close always releases at epoch max.
Checked owner/epoch refs replace raw pointer/reference lifetimes.

FindCString / PrepareFindCString are transient string boundary adapters, not
stored string keys. The latter follows source growth-before-duplicate and
preflights a missing insertion epoch before a caller allocates spelling bytes.
Raw arena allocation refs are owner-local; callers use BorrowCString with the
matching owner to form a key. `InitialCapacity` exposes the underlying source
core constructor; source `map` itself always starts at8.

## Tests and gates

Tests were written before production declarations; native compilation was RED
with missing StrInt32Map/Config/entry APIs. GREEN implements all14 active ENSURE
statements in `src/test/map.cpp` tst1, in source order. No source cases are
inactive in that file. Ten permanent tests cover these assertions plus:

- 700 operations checked against independent bounded bool/value/key arrays;
  two origins, signed int32 extremes and exact ascending bucket iteration.
- Independent permanent hash.cpp fixtures, CString NUL suffix/high bytes,
  complete pointer+value replacement, absent-insert preservation and default
  output preservation; cached hash and FREE/DELETED payload shape.
- Wrapped last-tombstone slot placement, strict growth-before-duplicate,
  duplicate failure before overwrite, strict cleanup / OOM suppression and
  removal-before-cleanup failure; strict reset overhead / finalize limits.
- Deep entry array copies with shallow spellings; union/intersection/assignment
  value semantics, source ascending multi-owner union failure prefix, self guards.
- Constructor/clone injection prefixes, directory old+new peak refusal,
  second cross-budget swap reserve rollback, full graph charges, stale refs,
  scoped-region lifetime, epoch exhaustion and interner preparation.
- Warm foreign query and mutable scalar value path measures zero Go allocations.

Gates completed: native `CGO_ENABLED=0 go test -count=1 ./utilhash`, native
`-tags=g3checked`, `go vet ./utilhash`; actual JS/WASM
`-tags=g3checked -count=1 -v -run StrInt32Map` executed all ten named tests
successfully; WASI test executable build succeeded. WASI semantic execution
remains blocked by the already-recorded custom Go runtime pre-main seed/env
issue; this packet does not claim WASI execution.

All source files, tests, status, build outputs, Go caches and compiler temporary
paths were kept below golang for this packet. Existing StrHashtable files were
not changed. Commit and global manifest updates belong to the coordinator.

Central review correction: an explicit trailing uint32 padding field makes the
physical hot entry exactly40 bytes on both 32-bit and64-bit Go targets. The
architecture audit derives the scalar go/types shape from actual reflected
fields (without manually repeating the record), and initially failed RED with
386 size36 versus charged40. After padding, 386/amd64/arm/arm64/wasm all report40;
native Stats/budget tests additionally check the exact physical array charge.
Padding is representation metadata, not additional source key/value data.
