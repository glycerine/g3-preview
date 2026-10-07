// Source: src/util/str_hashtable.h:24-31; src/util/hashtable.h:94-110,138-140,222-653.
package util

import (
	"errors"
	"reflect"
	"strings"

	"github.com/glycerine/g3/pkg/arena"
)

const (
	strHashFree uint32 = iota
	strHashDeleted
	strHashUsed
)
const strHashEntryBytes uint64 = 32

// StrKey borrows one NUL-terminated spelling from the table's bound byte
// region. Bytes must remain immutable/live while any table holds this key.
// No allocation ownership is transferred by copying a key or table.
type StrKey struct {
	ref            arena.BytesRegionRef
	offset, length uint32
	origin         uint64
}

func (k StrKey) Origin() uint64                   { return k.origin }
func (k StrKey) Allocation() arena.BytesRegionRef { return k.ref }
func (k StrKey) Offset() uint32                   { return k.offset }
func (k StrKey) Length() uint32                   { return k.length }

type strHashEntry struct {
	hash, state uint32
	key         StrKey
}
type StrHashtableConfig struct {
	Runtime     *Runtime
	Assertions  StrHashtableAssertionContext
	Statistics  bool
	Budget      *arena.Budget
	OutOfMemory func() bool
	// InitialCapacity is the source constructor parameter; zero selects8.
	InitialCapacity uint32
}

type strOwner struct {
	origin uint64
	region *arena.BytesRegion
}

// StrHashtable borrows its primary region and every adopted secondary region.
// Close/Reset/Erase never release spelling bytes. Swap exchanges bindings with
// the table. This owning struct must not be copied. Region Reset/rollback/Close
// invalidates affected keys; callers keep all borrowed regions alive.
type StrHashtable struct {
	raw                   strHashtableRaw
	table                 []strHashEntry
	region                *arena.BytesRegion
	owners                []strOwner
	config                StrHashtableConfig
	size, deleted         uint32
	epoch                 uint64
	rootBytes, tableBytes uint64
	directoryBytes        uint64
	closed                bool
}

func NewStrHashtable(r *arena.BytesRegion, c StrHashtableConfig) (*StrHashtable, error) {
	n := c.InitialCapacity
	if n == 0 {
		n = 8
	}
	return newStrHashtableCapacity(r, c, n)
}
func newStrHashtableCapacity(r *arena.BytesRegion, c StrHashtableConfig, n uint32) (*StrHashtable, error) {
	if r == nil || c.Runtime != nil && (c.Runtime.Budget() != c.Budget || c.OutOfMemory != nil) {
		return nil, arena.ErrConfig
	}
	if c.Assertions != nil && c.Assertions.StrHashtableAssertionBudget() != c.Budget {
		return nil, arena.ErrConfig
	}
	root := uint64(reflect.TypeOf(StrHashtable{}).Size())
	if e := c.Budget.Reserve(root); e != nil {
		return nil, e
	}
	h := &StrHashtable{region: r, config: c, epoch: 1, rootBytes: root}
	if c.Runtime != nil {
		e := h.constructRuntime(n)
		if e != nil && h.raw.object == (Block{}) && h.raw.table == (Block{}) && h.raw.acquisitions == nil && h.raw.sourceCause == nil {
			// The source new-expression has consumed every acquired byte. Only this
			// now-empty Go preparation header remains; no failed placement owner needs
			// to survive. Actual unjoined tokens/stopped destruction stay returned.
			if cleanup := h.CloseChecked(); cleanup != nil {
				return h, errors.Join(e, cleanup)
			}
			return nil, e
		}
		return h, e
	}
	if e := h.constructLegacy(n); e != nil {
		if cleanup := h.CloseChecked(); cleanup != nil {
			return h, errors.Join(e, cleanup)
		}
		return nil, e
	}
	return h, nil
}
func (h *StrHashtable) allocateTable(n uint32) ([]strHashEntry, uint64, error) {
	if n == 0 || n > 1<<30 || uint64(n) > uint64(^uint(0)>>1)/strHashEntryBytes {
		return nil, 0, arena.ErrBounds
	}
	amount := uint64(n) * strHashEntryBytes
	if e := h.config.Budget.Reserve(amount); e != nil {
		return nil, 0, e
	}
	return make([]strHashEntry, n), amount, nil
}
func (h *StrHashtable) open() error {
	if h == nil {
		return arena.ErrClosed
	}
	if h.raw.self != h {
		return arena.ErrConfig
	}
	if h.config.Runtime != nil {
		return h.openRuntime()
	}
	if h.raw.executing {
		return ErrStrHashtableExecuting
	}
	if h.raw.destructing {
		return arena.ErrClosed
	}
	if h.raw.sourceCause != nil {
		return h.raw.sourceCause
	}
	if h.closed || len(h.table) == 0 {
		return arena.ErrClosed
	}
	return nil
}
func (h *StrHashtable) mustOpen() {
	if e := h.open(); e != nil {
		panic(e)
	}
}
func (h *StrHashtable) bump() {
	if h.epoch == ^uint64(0) {
		panic("utilhash: string table epoch exhausted")
	}
	h.epoch++
}
func (h *StrHashtable) canBump() error {
	if h.epoch == ^uint64(0) {
		return arena.ErrBounds
	}
	return nil
}
func (h *StrHashtable) Size() uint32 {
	if h.config.Runtime != nil {
		return h.mustRuntimeWord(12)
	}
	return h.size
}
func (h *StrHashtable) Empty() bool { return h.Size() == 0 }
func (h *StrHashtable) Capacity() uint32 {
	if h.config.Runtime != nil {
		return h.mustRuntimeWord(8)
	}
	return h.raw.words[2]
}
func (h *StrHashtable) Deleted() uint32 {
	if h.config.Runtime != nil {
		return h.mustRuntimeWord(16)
	}
	return h.deleted
}
func (h *StrHashtable) CollisionCount() uint64 {
	if !h.config.Statistics {
		return 0
	}
	if h.raw.self != h {
		panic(arena.ErrConfig)
	}
	if h.closed || !h.raw.complete || h.raw.destructing {
		panic(arena.ErrClosed)
	}
	v, e := h.readCollision()
	if e != nil {
		panic(e)
	}
	return v
}

// StrHashtableStats accounts this owner, scalar table and cold owner directory.
// Borrowed regions report their own graphs and are never charged by this table.
type StrHashtableStats struct {
	OwnerBytes, TableBytes, DirectoryBytes, RetainedBytes uint64
	Size, Deleted, Capacity                               uint32
}

func (h *StrHashtable) Stats() StrHashtableStats {
	return StrHashtableStats{OwnerBytes: h.rootBytes, TableBytes: h.tableBytes, DirectoryBytes: h.directoryBytes, RetainedBytes: h.legacyRetainedBytes(), Size: h.Size(), Deleted: h.Deleted(), Capacity: h.Capacity()}
}

func (h *StrHashtable) keyRegion(k StrKey) (*arena.BytesRegion, error) {
	if k.origin == h.region.OwnerID() {
		return h.region, nil
	}
	lo, hi := 0, len(h.owners)
	for lo < hi {
		mid := lo + (hi-lo)/2
		if h.owners[mid].origin < k.origin {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	if lo < len(h.owners) && h.owners[lo].origin == k.origin {
		return h.owners[lo].region, nil
	}
	return nil, arena.ErrInvalidSlot
}

// adopt records a cold borrowed owner only when its spelling is retained.
// Sorted scalar origins make lookup deterministic. Directory movement does not
// invalidate entry borrows because no pointer into this directory escapes.
func (h *StrHashtable) adopt(r *arena.BytesRegion) error {
	if r == nil {
		return arena.ErrInvalidSlot
	}
	id := r.OwnerID()
	if id == h.region.OwnerID() {
		return nil
	}
	lo, hi := 0, len(h.owners)
	for lo < hi {
		mid := lo + (hi-lo)/2
		if h.owners[mid].origin < id {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	if lo < len(h.owners) && h.owners[lo].origin == id {
		if h.owners[lo].region != r {
			return arena.ErrInvalidSlot
		}
		return nil
	}
	if len(h.owners) == cap(h.owners) {
		capacity := cap(h.owners)
		if capacity == 0 {
			capacity = 1
		} else {
			if capacity > int(^uint(0)>>1)/2 {
				return arena.ErrBounds
			}
			capacity *= 2
		}
		entryBytes := uint64(reflect.TypeOf(strOwner{}).Size())
		if uint64(capacity) > uint64(^uint32(0)) || uint64(capacity) > uint64(^uint(0)>>1)/entryBytes {
			return arena.ErrBounds
		}
		amount := uint64(capacity) * entryBytes
		if e := h.config.Budget.Reserve(amount); e != nil {
			return e
		}
		owners := make([]strOwner, len(h.owners), capacity)
		copy(owners, h.owners)
		h.config.Budget.Release(h.directoryBytes)
		h.owners, h.directoryBytes = owners, amount
	}
	h.owners = h.owners[:len(h.owners)+1]
	copy(h.owners[lo+1:], h.owners[lo:len(h.owners)-1])
	h.owners[lo] = strOwner{id, r}
	return nil
}

// CString finds source strlen within the allocation bound. An empty spelling
// still requires a live allocation containing its terminating NUL.
func (h *StrHashtable) CString(ref arena.BytesRegionRef, offset uint32) (StrKey, error) {
	if e := h.open(); e != nil {
		return StrKey{}, e
	}
	return BorrowCString(h.region, ref, offset)
}

// BorrowCString constructs a checked scalar borrow without adopting or
// allocating table metadata. The raw allocation ref is owner-local to region.
func BorrowCString(region *arena.BytesRegion, ref arena.BytesRegionRef, offset uint32) (StrKey, error) {
	if region == nil {
		return StrKey{}, arena.ErrInvalidSlot
	}
	size, e := region.Size(ref)
	if e != nil {
		return StrKey{}, e
	}
	if !region.IsLive(ref) {
		return StrKey{}, arena.ErrInvalidSlot
	}
	if offset >= size {
		return StrKey{}, arena.ErrBounds
	}
	for i := offset; i < size; i++ {
		b, e := region.Read(ref, i)
		if e != nil {
			return StrKey{}, e
		}
		if b == 0 {
			return StrKey{ref, offset, i - offset, region.OwnerID()}, nil
		}
	}
	return StrKey{}, arena.ErrBounds
}
func (h *StrHashtable) validateKey(k StrKey) error {
	r, e := h.keyRegion(k)
	if e != nil {
		return e
	}
	return validateStrKey(r, k)
}
func validateStrKey(region *arena.BytesRegion, k StrKey) error {
	if region == nil || k.origin != region.OwnerID() {
		return arena.ErrInvalidSlot
	}
	size, e := region.Size(k.ref)
	if e != nil {
		return e
	}
	if !region.IsLive(k.ref) {
		return arena.ErrInvalidSlot
	}
	if uint64(k.offset)+uint64(k.length) >= uint64(size) {
		return arena.ErrBounds
	}
	b, e := region.Read(k.ref, k.offset+k.length)
	if e != nil {
		return e
	}
	if b != 0 {
		return arena.ErrBounds
	}
	return nil
}

// Hash uses the source seed17, little-endian word loads and signed-char tail.
// Hash rereads the actual CString length at this call. Mutating an inserted
// key can invalidate its table cached hash; uninserted pointer inputs remain
// ordinary caller-owned CString arguments.
func (h *StrHashtable) Hash(k StrKey) (uint32, error) {
	if e := h.open(); e != nil {
		return 0, e
	}
	r, e := h.keyRegion(k)
	if e != nil {
		return 0, e
	}
	return hashStrKey(r, k)
}
func hashStrKey(region *arena.BytesRegion, k StrKey) (uint32, error) {
	n, e := strKeyLength(region, k)
	if e != nil {
		return 0, e
	}
	return RegionBytes(region, k.ref, uint64(k.offset), n, 17)
}
func (h *StrHashtable) Equal(a, b StrKey) (bool, error) {
	if e := h.open(); e != nil {
		return false, e
	}
	ra, e := h.keyRegion(a)
	if e != nil {
		return false, e
	}
	rb, e := h.keyRegion(b)
	if e != nil {
		return false, e
	}
	return equalStrKeys(ra, a, rb, b)
}
func equalStrKeys(ra *arena.BytesRegion, a StrKey, rb *arena.BytesRegion, b StrKey) (bool, error) {
	for i := uint64(0); ; i++ {
		x, e := strKeyByte(ra, a, i)
		if e != nil {
			return false, e
		}
		y, e := strKeyByte(rb, b, i)
		if e != nil {
			return false, e
		}
		if x != y {
			return false, nil
		}
		if x == 0 {
			return true, nil
		}
	}
}
func (h *StrHashtable) Text(k StrKey) (string, error) {
	if e := h.open(); e != nil {
		return "", e
	}
	if e := h.validateKey(k); e != nil {
		return "", e
	}
	region, e := h.keyRegion(k)
	if e != nil {
		return "", e
	}
	var out strings.Builder
	out.Grow(int(k.length))
	for i := uint32(0); i < k.length; i++ {
		b, e := region.Read(k.ref, k.offset+i)
		if e != nil {
			return "", e
		}
		out.WriteByte(b)
	}
	return out.String(), nil
}

func (h *StrHashtable) rehash(n uint32) error { return h.rehashLegacy(n) }

func (h *StrHashtable) prepareInsert() error {
	if h.config.Runtime != nil {
		return h.prepareInsertRuntime()
	}
	if e := h.open(); e != nil {
		return e
	}
	if (h.size+h.deleted)<<2 > h.Capacity()*3 {
		if h.Capacity() > 1<<29 {
			return arena.ErrBounds
		}
		return h.rehash(h.Capacity() << 1)
	}
	return nil
}
func (h *StrHashtable) insert(key StrKey, overwrite bool) (bool, StrHashtableEntryRef, error) {
	if e := h.open(); e != nil {
		return false, StrHashtableEntryRef{}, e
	}
	region, e := h.keyRegion(key)
	if e != nil {
		return false, StrHashtableEntryRef{}, e
	}
	return h.insertFrom(region, key, overwrite)
}
func (h *StrHashtable) insertFrom(region *arena.BytesRegion, key StrKey, overwrite bool) (bool, StrHashtableEntryRef, error) {
	if h.config.Runtime != nil {
		return h.insertRuntime(region, key, overwrite)
	}
	if e := h.prepareInsert(); e != nil {
		return false, StrHashtableEntryRef{}, e
	}
	hash, e := hashStrKey(region, key)
	if e != nil {
		return false, StrHashtableEntryRef{}, e
	}
	capacity := h.Capacity()
	table := h.table
	mask := capacity - 1
	i := hash & mask
	deleted := ^uint32(0)
	for step := uint32(0); step < capacity; step++ {
		entry, e := legacyEntryAt(table, i)
		if e != nil {
			return false, StrHashtableEntryRef{}, e
		}
		switch entry.state {
		case strHashUsed:
			if entry.hash == hash {
				storedRegion, e := h.keyRegion(entry.key)
				if e != nil {
					return false, StrHashtableEntryRef{}, e
				}
				equal, e := equalStrKeys(storedRegion, entry.key, region, key)
				if e != nil {
					return false, StrHashtableEntryRef{}, e
				}
				if equal {
					if overwrite {
						if e := h.adopt(region); e != nil {
							return false, StrHashtableEntryRef{}, e
						}
						entry.key = key
					}
					return false, h.ref(i), nil
				}
			}
			if e = h.incrementCollision(); e != nil {
				return false, StrHashtableEntryRef{}, e
			}
		case strHashFree:
			if e := h.canBump(); e != nil {
				return false, StrHashtableEntryRef{}, e
			}
			if e := h.adopt(region); e != nil {
				return false, StrHashtableEntryRef{}, e
			}
			slot := i
			if deleted != ^uint32(0) {
				slot = deleted
				h.deleted--
			}
			table[slot] = strHashEntry{hash, strHashUsed, key}
			h.size++
			h.bump()
			return true, h.ref(slot), nil
		default:
			on, e := h.assertionsEnabled()
			if e != nil {
				return false, StrHashtableEntryRef{}, e
			}
			if on && entry.state != strHashDeleted {
				line := int32(371)
				if !overwrite {
					line = 422
				}
				if e = h.assertionFailure(line, "curr->is_deleted()"); e != nil {
					return false, StrHashtableEntryRef{}, e
				}
			}
			deleted = i
			if e = h.incrementCollision(); e != nil {
				return false, StrHashtableEntryRef{}, e
			}
		}
		i++
		if i == capacity {
			i = 0
		}
	}
	line := int32(395)
	if !overwrite {
		line = 451
	}
	return false, StrHashtableEntryRef{}, h.unreachable(line)
}
func (h *StrHashtable) Insert(k StrKey) error { _, _, e := h.insert(k, true); return e }
func (h *StrHashtable) InsertIfNotThereCore(k StrKey) (bool, StrHashtableEntryRef, error) {
	return h.insert(k, false)
}
func (h *StrHashtable) InsertIfNotThere(k StrKey) (StrHashtableEntryRef, error) {
	_, r, e := h.insert(k, false)
	return r, e
}
func (h *StrHashtable) InsertIfNotThere2(k StrKey) (StrHashtableEntryRef, error) {
	return h.InsertIfNotThere(k)
}
func (h *StrHashtable) InsertFrom(region *arena.BytesRegion, k StrKey) error {
	_, _, e := h.insertFrom(region, k, true)
	return e
}
func (h *StrHashtable) InsertIfNotThereCoreFrom(region *arena.BytesRegion, k StrKey) (bool, StrHashtableEntryRef, error) {
	return h.insertFrom(region, k, false)
}
func (h *StrHashtable) InsertIfNotThereFrom(region *arena.BytesRegion, k StrKey) (StrHashtableEntryRef, error) {
	_, r, e := h.insertFrom(region, k, false)
	return r, e
}
func (h *StrHashtable) InsertIfNotThere2From(region *arena.BytesRegion, k StrKey) (StrHashtableEntryRef, error) {
	return h.InsertIfNotThereFrom(region, k)
}
func (h *StrHashtable) findSlot(key StrKey) (uint32, bool, error) {
	region, e := h.keyRegion(key)
	if e != nil {
		return 0, false, e
	}
	return h.findSlotFrom(region, key)
}
func (h *StrHashtable) findSlotFrom(region *arena.BytesRegion, key StrKey) (uint32, bool, error) {
	return h.findSlotFromSource(region, key, nil)
}
func (h *StrHashtable) findSlotFromSource(region *arena.BytesRegion, key StrKey, exhausted *bool) (uint32, bool, error) {
	hash, e := hashStrKey(region, key)
	if e != nil {
		return 0, false, e
	}
	capacity, e := h.capacityChecked()
	if e != nil {
		return 0, false, e
	}
	table := h.table
	mask := capacity - 1
	i := hash & mask
	for step := uint32(0); step < capacity; step++ {
		var entry strHashEntry
		if h.config.Runtime == nil {
			p, re := legacyEntryAt(table, i)
			if re != nil {
				e = re
			} else {
				entry = *p
			}
		} else {
			entry, e = h.probeEntry(i)
		}
		if e != nil {
			return 0, false, e
		}
		if entry.state == strHashUsed && entry.hash == hash {
			if h.config.Runtime != nil {
				entry.key, e = h.entryKey(i)
			}
			if e != nil {
				return 0, false, e
			}
			storedRegion, e := h.keyRegion(entry.key)
			if e != nil {
				return 0, false, e
			}
			equal, e := equalStrKeys(storedRegion, entry.key, region, key)
			if e != nil {
				return 0, false, e
			}
			if equal {
				return i, true, nil
			}
		}
		if entry.state == strHashUsed {
			if e := h.incrementCollision(); e != nil {
				return 0, false, e
			}
		}
		if entry.state == strHashFree {
			return 0, false, nil
		}
		if e := h.incrementCollision(); e != nil {
			return 0, false, e
		}
		i++
		if i == capacity {
			i = 0
		}
	}
	if exhausted != nil {
		*exhausted = true
	}
	return 0, false, nil
}
func (h *StrHashtable) FindCore(k StrKey) (StrHashtableEntryRef, bool, error) {
	if e := h.open(); e != nil {
		return StrHashtableEntryRef{}, false, e
	}
	i, ok, e := h.findSlot(k)
	if e != nil || !ok {
		return StrHashtableEntryRef{}, ok, e
	}
	return h.ref(i), true, nil
}
func (h *StrHashtable) Contains(k StrKey) (bool, error) { _, ok, e := h.FindCore(k); return ok, e }

// FindFrom reads an explicit foreign query owner without retaining it or
// allocating directory metadata. The returned ref borrows the stored entry.
func (h *StrHashtable) FindFrom(region *arena.BytesRegion, k StrKey) (StrHashtableEntryRef, bool, error) {
	if e := h.open(); e != nil {
		return StrHashtableEntryRef{}, false, e
	}
	i, ok, e := h.findSlotFrom(region, k)
	if e != nil || !ok {
		return StrHashtableEntryRef{}, ok, e
	}
	return h.ref(i), true, nil
}
func (h *StrHashtable) ContainsFrom(region *arena.BytesRegion, k StrKey) (bool, error) {
	_, ok, e := h.FindFrom(region, k)
	return ok, e
}
func (h *StrHashtable) FindIntoFrom(region *arena.BytesRegion, k StrKey, out *StrKey) (bool, error) {
	r, found, e := h.FindFrom(region, k)
	if e != nil || !found {
		return found, e
	}
	key, e := r.Key()
	if e != nil {
		return false, e
	}
	*out = key
	return true, nil
}
func (h *StrHashtable) Find(k StrKey) (StrKey, bool, error) {
	r, ok, e := h.FindCore(k)
	if e != nil || !ok {
		return StrKey{}, ok, e
	}
	key, e := h.entryKey(r.slot)
	return key, e == nil, e
}
func (h *StrHashtable) FindInto(k StrKey, out *StrKey) (bool, error) {
	found, ok, e := h.Find(k)
	if e == nil && ok {
		*out = found
	}
	return ok, e
}
func strPrefix(s string) string {
	if i := strings.IndexByte(s, 0); i >= 0 {
		return s[:i]
	}
	return s
}

// FindCString uses a transient external query; no Go string is stored in the
// table. Equality compares raw unsigned byte identities as source strcmp.
func (h *StrHashtable) FindCString(s string) (StrHashtableEntryRef, bool, error) {
	if e := h.open(); e != nil {
		return StrHashtableEntryRef{}, false, e
	}
	s = strPrefix(s)
	if uint64(len(s)) > uint64(^uint32(0)) {
		return StrHashtableEntryRef{}, false, arena.ErrBounds
	}
	hash := String(s, 17)
	capacity, e := h.capacityChecked()
	if e != nil {
		return StrHashtableEntryRef{}, false, e
	}
	table := h.table
	mask := capacity - 1
	i := hash & mask
	for step := uint32(0); step < capacity; step++ {
		var entry strHashEntry
		if h.config.Runtime == nil {
			p, re := legacyEntryAt(table, i)
			if re != nil {
				e = re
			} else {
				entry = *p
			}
		} else {
			entry, e = h.probeEntry(i)
		}
		if e != nil {
			return StrHashtableEntryRef{}, false, e
		}
		if entry.state == strHashUsed && entry.hash == hash {
			if h.config.Runtime != nil {
				entry.key, e = h.entryKey(i)
			}
			if e != nil {
				return StrHashtableEntryRef{}, false, e
			}
			region, e := h.keyRegion(entry.key)
			if e != nil {
				return StrHashtableEntryRef{}, false, e
			}
			equal, e := strKeyEqualString(region, entry.key, s)
			if e != nil {
				return StrHashtableEntryRef{}, false, e
			}
			if equal {
				return h.ref(i), true, nil
			}
		}
		if entry.state == strHashUsed {
			if e := h.incrementCollision(); e != nil {
				return StrHashtableEntryRef{}, false, e
			}
		}
		if entry.state == strHashFree {
			return StrHashtableEntryRef{}, false, nil
		}
		if e := h.incrementCollision(); e != nil {
			return StrHashtableEntryRef{}, false, e
		}
		i++
		if i == capacity {
			i = 0
		}
	}
	return StrHashtableEntryRef{}, false, nil
}

// PrepareFindCString preserves source interner growth before duplicate lookup.
// Only a checked borrow escapes; a missing query leaves no temporary spelling
// in the table. Allocate the owner spelling, then publish with the normal insert.
func (h *StrHashtable) PrepareFindCString(s string) (StrHashtableEntryRef, bool, error) {
	if e := h.prepareInsert(); e != nil {
		return StrHashtableEntryRef{}, false, e
	}
	r, found, e := h.FindCString(s)
	if e != nil || found {
		return r, found, e
	}
	// A miss authorizes the serialized caller to allocate its owned spelling.
	// Refuse exhaustion before that allocation, rather than only at publication.
	if e := h.canBump(); e != nil {
		return StrHashtableEntryRef{}, false, e
	}
	return StrHashtableEntryRef{}, false, nil
}
func (h *StrHashtable) Remove(k StrKey) error {
	if e := h.open(); e != nil {
		return e
	}
	region, e := h.keyRegion(k)
	if e != nil {
		return e
	}
	return h.RemoveFrom(region, k)
}
func (h *StrHashtable) RemoveFrom(region *arena.BytesRegion, k StrKey) error {
	if h.config.Runtime != nil {
		return h.removeRuntime(region, k)
	}
	if e := h.open(); e != nil {
		return e
	}
	var exhausted bool
	i, ok, e := h.findSlotFromSource(region, k, &exhausted)
	if e != nil {
		return e
	}
	if !ok {
		if exhausted {
			on, e := h.assertionsEnabled()
			if e != nil {
				return e
			}
			if on {
				contains, e := h.ContainsFrom(region, k)
				if e != nil {
					return e
				}
				if contains {
					return h.assertionFailure(559, "!contains(e)")
				}
			}
		}
		return nil
	}
	if e := h.canBump(); e != nil {
		return e
	}
	next := i + 1
	if next == h.Capacity() {
		next = 0
	}
	h.size--
	h.bump()
	h.table[i].key = StrKey{}
	if h.table[next].state == strHashFree {
		h.table[i].state = strHashFree
		return nil
	}
	h.table[i].state = strHashDeleted
	h.deleted++
	if h.deleted > h.size && h.deleted > 64 && (h.config.OutOfMemory == nil || !h.config.OutOfMemory()) {
		return h.rehash(h.Capacity())
	}
	return nil
}
func (h *StrHashtable) Erase(k StrKey) error { return h.Remove(k) }
func (h *StrHashtable) EraseFrom(region *arena.BytesRegion, k StrKey) error {
	return h.RemoveFrom(region, k)
}
func (h *StrHashtable) Reset() error {
	if h.config.Runtime != nil {
		return h.resetRuntime()
	}
	return h.resetLegacy()
}
func (h *StrHashtable) Finalize() error {
	if h.config.Runtime != nil {
		return h.finalizeRuntime()
	}
	return h.finalizeLegacy()
}
func (h *StrHashtable) Close() {
	if e := h.CloseChecked(); e != nil {
		panic(e)
	}
}
func (h *StrHashtable) Clone() (*StrHashtable, error) {
	if e := h.open(); e != nil {
		return nil, e
	}
	if h.config.Runtime != nil {
		return h.cloneRuntime(true)
	}
	return h.cloneLegacy()
}
func (h *StrHashtable) bothOpen(other *StrHashtable) error {
	if e := h.open(); e != nil {
		return e
	}
	if e := other.open(); e != nil {
		return e
	}
	return nil
}
func (h *StrHashtable) Assign(src *StrHashtable) error {
	if e := h.bothOpen(src); e != nil {
		return e
	}
	if h == src {
		return nil
	}
	if e := h.Reset(); e != nil {
		return e
	}
	return h.Union(src)
}
func (h *StrHashtable) Union(src *StrHashtable) error {
	if e := h.bothOpen(src); e != nil {
		return e
	}
	if h == src {
		return nil
	}
	capacity, e := src.capacityChecked()
	if e != nil {
		return e
	}
	for idx := uint32(0); idx < capacity; idx++ {
		state, e := src.entryState(idx)
		if e != nil {
			return e
		}
		entry := strHashEntry{state: state}
		if state == strHashUsed {
			entry.key, e = src.entryKey(idx)
			if e != nil {
				return e
			}
		}
		if entry.state == strHashUsed {
			owner, e := src.keyRegion(entry.key)
			if e != nil {
				return e
			}
			if e := h.InsertFrom(owner, entry.key); e != nil {
				return e
			}
		}
	}
	return nil
}
func (h *StrHashtable) Intersect(src *StrHashtable) error {
	if h.config.Runtime != nil {
		return h.intersectRuntime(src)
	}
	return h.intersectLegacy(src)
}
func (h *StrHashtable) Swap(other *StrHashtable) error {
	if h.config.Runtime != nil || other.config.Runtime != nil {
		return h.swapRuntime(other)
	}
	if h.config.Statistics != other.config.Statistics {
		return arena.ErrConfig
	}
	if e := h.open(); e != nil {
		return e
	}
	if e := other.open(); e != nil {
		return e
	}
	if h == other {
		return nil
	}
	if e := h.canBump(); e != nil {
		return e
	}
	if e := other.canBump(); e != nil {
		return e
	}
	if h.config.Budget != other.config.Budget {
		incoming := other.tableBytes + other.directoryBytes
		outgoing := h.tableBytes + h.directoryBytes
		if e := h.config.Budget.Reserve(incoming); e != nil {
			return e
		}
		if e := other.config.Budget.Reserve(outgoing); e != nil {
			h.config.Budget.Release(incoming)
			return e
		}
		h.config.Budget.Release(outgoing)
		other.config.Budget.Release(incoming)
	}
	h.table, other.table = other.table, h.table
	h.raw.words[2], other.raw.words[2] = other.raw.words[2], h.raw.words[2]
	h.region, other.region = other.region, h.region
	h.owners, other.owners = other.owners, h.owners
	h.directoryBytes, other.directoryBytes = other.directoryBytes, h.directoryBytes
	h.tableBytes, other.tableBytes = other.tableBytes, h.tableBytes
	h.size, other.size = other.size, h.size
	h.deleted, other.deleted = other.deleted, h.deleted
	if h.config.Statistics {
		x, e := h.readCollision()
		if e != nil {
			return e
		}
		y, e := other.readCollision()
		if e != nil {
			return e
		}
		if e = h.writeCollision(y); e != nil {
			return e
		}
		if e = other.writeCollision(x); e != nil {
			return e
		}
	}
	h.bump()
	other.bump()
	return nil
}
func (h *StrHashtable) CheckInvariant() bool {
	ok, e := h.CheckInvariantChecked()
	if e != nil {
		panic(e)
	}
	return ok
}

func (h *StrHashtable) Dump() (string, error) {
	if e := h.open(); e != nil {
		return "", e
	}
	var out strings.Builder
	out.WriteByte('[')
	first := true
	capacity, e := h.capacityChecked()
	if e != nil {
		return "", e
	}
	for idx := uint32(0); idx < capacity; idx++ {
		state, e := h.entryState(idx)
		if e != nil {
			return "", e
		}
		entry := strHashEntry{state: state}
		if state == strHashUsed {
			entry.key, e = h.entryKey(idx)
			if e != nil {
				return "", e
			}
		}
		if entry.state != strHashUsed {
			continue
		}
		s, e := h.Text(entry.key)
		if e != nil {
			return "", e
		}
		if !first {
			out.WriteByte(' ')
		}
		first = false
		out.WriteString(s)
	}
	out.WriteByte(']')
	return out.String(), nil
}

type StrHashtableEntryRef struct {
	owner *StrHashtable
	epoch uint64
	slot  uint32
}

func (h *StrHashtable) ref(i uint32) StrHashtableEntryRef { return StrHashtableEntryRef{h, h.epoch, i} }
func (r StrHashtableEntryRef) validate() error {
	if r.owner == nil {
		return arena.ErrInvalidSlot
	}
	if e := r.owner.open(); e != nil {
		return e
	}
	capacity, e := r.owner.capacityChecked()
	if e != nil {
		return e
	}
	if r.epoch != r.owner.epoch || r.slot >= capacity {
		return arena.ErrInvalidSlot
	}
	state, e := r.owner.entryState(r.slot)
	if e != nil {
		return e
	}
	if state != strHashUsed {
		return arena.ErrInvalidSlot
	}
	return nil
}
func (r StrHashtableEntryRef) Key() (StrKey, error) {
	if e := r.validate(); e != nil {
		return StrKey{}, e
	}
	k, e := r.owner.entryKey(r.slot)
	if e != nil {
		return StrKey{}, e
	}
	if e := r.owner.validateKey(k); e != nil {
		return StrKey{}, e
	}
	return k, nil
}
func (r StrHashtableEntryRef) Hash() (uint32, error) {
	if e := r.validate(); e != nil {
		return 0, e
	}
	entry, e := r.owner.probeEntry(r.slot)
	return entry.hash, e
}

// SetKey represents source e->set_data(owned_spelling) after interning. It
// preserves the cached hash and therefore requires identical C-string content.
func (r StrHashtableEntryRef) SetKey(k StrKey) error {
	if e := r.validate(); e != nil {
		return e
	}
	region, e := r.owner.keyRegion(k)
	if e != nil {
		return e
	}
	return r.SetKeyFrom(region, k)
}
func (r StrHashtableEntryRef) SetKeyFrom(region *arena.BytesRegion, k StrKey) error {
	if e := r.validate(); e != nil {
		return e
	}
	hash, e := hashStrKey(region, k)
	if e != nil {
		return e
	}
	entry, e := r.owner.readEntry(r.slot)
	if e != nil {
		return e
	}
	if hash != entry.hash {
		return arena.ErrBounds
	}
	storedRegion, e := r.owner.keyRegion(entry.key)
	if e != nil {
		return e
	}
	equal, e := equalStrKeys(storedRegion, entry.key, region, k)
	if e != nil {
		return e
	}
	if !equal {
		return arena.ErrBounds
	}
	if e := r.owner.adopt(region); e != nil {
		return e
	}
	if r.owner.config.Runtime != nil {
		b, e := r.owner.runtimeTable()
		if e != nil {
			return e
		}
		return r.owner.putKeyAt(b, r.slot, k)
	}
	r.owner.table[r.slot].key = k
	return nil
}

type StrHashtableIterator struct {
	owner      *StrHashtable
	epoch      uint64
	index, end uint32
}

func (h *StrHashtable) Begin() StrHashtableIterator {
	h.mustOpen()
	it := StrHashtableIterator{owner: h, epoch: h.epoch, end: h.Capacity()}
	it.scan()
	return it
}
func (h *StrHashtable) End() StrHashtableIterator {
	h.mustOpen()
	return StrHashtableIterator{h, h.epoch, h.Capacity(), h.Capacity()}
}
func (h *StrHashtable) FindIterator(k StrKey) (StrHashtableIterator, error) {
	r, ok, e := h.FindCore(k)
	if e != nil {
		return StrHashtableIterator{}, e
	}
	if !ok {
		return h.End(), nil
	}
	return StrHashtableIterator{h, h.epoch, r.slot, h.Capacity()}, nil
}
func (h *StrHashtable) FindIteratorFrom(region *arena.BytesRegion, k StrKey) (StrHashtableIterator, error) {
	r, ok, e := h.FindFrom(region, k)
	if e != nil {
		return StrHashtableIterator{}, e
	}
	if !ok {
		return h.End(), nil
	}
	return StrHashtableIterator{h, h.epoch, r.slot, h.Capacity()}, nil
}
func (it *StrHashtableIterator) scan() {
	for it.index < it.end {
		state, e := it.owner.entryState(it.index)
		if e != nil {
			panic(e)
		}
		if state == strHashUsed {
			break
		}
		it.index++
	}
}
func (it StrHashtableIterator) validate() error {
	if it.owner == nil {
		return arena.ErrInvalidSlot
	}
	if e := it.owner.open(); e != nil {
		return e
	}
	if it.epoch != it.owner.epoch {
		return arena.ErrInvalidSlot
	}
	return nil
}
func (it StrHashtableIterator) AtEnd() bool { return it.index == it.end }
func (it StrHashtableIterator) Equal(other StrHashtableIterator) bool {
	return it.owner == other.owner && it.epoch == other.epoch && it.index == other.index
}
func (it StrHashtableIterator) Ref() (StrHashtableEntryRef, error) {
	if e := it.validate(); e != nil {
		return StrHashtableEntryRef{}, e
	}
	if it.AtEnd() {
		return StrHashtableEntryRef{}, arena.ErrBounds
	}
	return it.owner.ref(it.index), nil
}
func (it StrHashtableIterator) Key() (StrKey, error) {
	r, e := it.Ref()
	if e != nil {
		return StrKey{}, e
	}
	return r.Key()
}
func (it *StrHashtableIterator) Next() error {
	if e := it.validate(); e != nil {
		return e
	}
	if it.AtEnd() {
		return arena.ErrBounds
	}
	it.index++
	it.scan()
	return nil
}
func (it *StrHashtableIterator) PostNext() (StrHashtableIterator, error) {
	old := *it
	if e := it.Next(); e != nil {
		return StrHashtableIterator{}, e
	}
	return old, nil
}
