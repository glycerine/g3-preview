// Source: src/util/map.h:24-196; src/util/str_hashtable.h:24-31; src/util/hashtable.h:44-63,138-140,222-653.
package util

import (
	"errors"
	"reflect"
	"strings"

	"github.com/glycerine/g3/pkg/arena"
)

const strInt32EntryBytes uint64 = 40

type strInt32Entry struct {
	hash, state uint32
	key         StrKey
	value       int32
	padding     uint32
}
type StrInt32MapConfig struct {
	Runtime     *Runtime
	Assertions  StrHashtableAssertionContext
	Statistics  bool
	Budget      *arena.Budget
	OutOfMemory func() bool
	// InitialCapacity selects the concrete core adapter capacity; the source map constructor uses8.
	InitialCapacity uint32
}

// StrInt32Map borrows its primary region and every adopted secondary region.
// Close/Reset/Erase never release spelling bytes. Swap exchanges bindings with
// the table. This owning struct must not be copied. Region Reset/rollback/Close
// invalidates affected keys; callers keep all borrowed regions alive.
type StrInt32Map struct {
	raw                   strInt32MapRaw
	table                 []strInt32Entry
	region                *arena.BytesRegion
	owners                []strOwner
	config                StrInt32MapConfig
	size, deleted         uint32
	epoch                 uint64
	rootBytes, tableBytes uint64
	directoryBytes        uint64
	closed                bool
}

func NewStrInt32Map(r *arena.BytesRegion, c StrInt32MapConfig) (*StrInt32Map, error) {
	n := c.InitialCapacity
	if n == 0 {
		n = 8
	}
	return newStrInt32MapCapacity(r, c, n)
}
func newStrInt32MapCapacity(r *arena.BytesRegion, c StrInt32MapConfig, n uint32) (*StrInt32Map, error) {
	if r == nil || c.Runtime != nil && (c.Runtime.Budget() != c.Budget || c.OutOfMemory != nil) {
		return nil, arena.ErrConfig
	}
	if c.Assertions != nil && c.Assertions.StrHashtableAssertionBudget() != c.Budget {
		return nil, arena.ErrConfig
	}
	root := uint64(reflect.TypeOf(StrInt32Map{}).Size())
	if e := c.Budget.Reserve(root); e != nil {
		return nil, e
	}
	h := &StrInt32Map{region: r, config: c, epoch: 1, rootBytes: root}
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
func (h *StrInt32Map) allocateTable(n uint32) ([]strInt32Entry, uint64, error) {
	if n == 0 || n > 1<<30 || uint64(n) > uint64(^uint(0)>>1)/strInt32EntryBytes {
		return nil, 0, arena.ErrBounds
	}
	amount := uint64(n) * strInt32EntryBytes
	if e := h.config.Budget.Reserve(amount); e != nil {
		return nil, 0, e
	}
	return make([]strInt32Entry, n), amount, nil
}
func (h *StrInt32Map) open() error {
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
		return ErrStrInt32MapExecuting
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
func (h *StrInt32Map) mustOpen() {
	if e := h.open(); e != nil {
		panic(e)
	}
}
func (h *StrInt32Map) bump() {
	if h.epoch == ^uint64(0) {
		panic("utilhash: string table epoch exhausted")
	}
	h.epoch++
}
func (h *StrInt32Map) canBump() error {
	if h.epoch == ^uint64(0) {
		return arena.ErrBounds
	}
	return nil
}
func (h *StrInt32Map) Size() uint32 {
	if h.config.Runtime != nil {
		return h.mustRuntimeWord(12)
	}
	return h.size
}
func (h *StrInt32Map) Empty() bool { return h.Size() == 0 }
func (h *StrInt32Map) Capacity() uint32 {
	if h.config.Runtime != nil {
		return h.mustRuntimeWord(8)
	}
	return h.raw.words[2]
}
func (h *StrInt32Map) Deleted() uint32 {
	if h.config.Runtime != nil {
		return h.mustRuntimeWord(16)
	}
	return h.deleted
}
func (h *StrInt32Map) CollisionCount() uint64 {
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

// StrInt32MapStats accounts this owner, scalar table and cold owner directory.
// Borrowed regions report their own graphs and are never charged by this table.
type StrInt32MapStats struct {
	OwnerBytes, TableBytes, DirectoryBytes, RetainedBytes uint64
	Size, Deleted, Capacity                               uint32
}

func (h *StrInt32Map) Stats() StrInt32MapStats {
	return StrInt32MapStats{OwnerBytes: h.rootBytes, TableBytes: h.tableBytes, DirectoryBytes: h.directoryBytes, RetainedBytes: h.legacyRetainedBytes(), Size: h.Size(), Deleted: h.Deleted(), Capacity: h.Capacity()}
}

func (h *StrInt32Map) keyRegion(k StrKey) (*arena.BytesRegion, error) {
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
func (h *StrInt32Map) adopt(r *arena.BytesRegion) error {
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
func (h *StrInt32Map) CString(ref arena.BytesRegionRef, offset uint32) (StrKey, error) {
	if e := h.open(); e != nil {
		return StrKey{}, e
	}
	return BorrowCString(h.region, ref, offset)
}

// BorrowCString constructs a checked scalar borrow without adopting or
// allocating table metadata. The raw allocation ref is owner-local to region.

func (h *StrInt32Map) validateKey(k StrKey) error {
	r, e := h.keyRegion(k)
	if e != nil {
		return e
	}
	return validateStrKey(r, k)
}

// Hash uses the source seed17, little-endian word loads and signed-char tail.
// Hash rereads the actual CString length at this call. Mutating an inserted
// key can invalidate its table cached hash; uninserted pointer inputs remain
// ordinary caller-owned CString arguments.
func (h *StrInt32Map) Hash(k StrKey) (uint32, error) {
	if e := h.open(); e != nil {
		return 0, e
	}
	r, e := h.keyRegion(k)
	if e != nil {
		return 0, e
	}
	return hashStrKey(r, k)
}

func (h *StrInt32Map) Equal(a, b StrKey) (bool, error) {
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

func (h *StrInt32Map) Text(k StrKey) (string, error) {
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

func (h *StrInt32Map) rehash(n uint32) error { return h.rehashLegacy(n) }

func (h *StrInt32Map) prepareInsert() error {
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
func (h *StrInt32Map) insert(key StrKey, value int32, overwrite bool) (bool, StrInt32MapEntryRef, error) {
	if e := h.open(); e != nil {
		return false, StrInt32MapEntryRef{}, e
	}
	region, e := h.keyRegion(key)
	if e != nil {
		return false, StrInt32MapEntryRef{}, e
	}
	return h.insertFrom(region, key, value, overwrite)
}
func (h *StrInt32Map) insertFrom(region *arena.BytesRegion, key StrKey, value int32, overwrite bool) (bool, StrInt32MapEntryRef, error) {
	if h.config.Runtime != nil {
		return h.insertRuntime(region, key, value, overwrite)
	}
	if e := h.prepareInsert(); e != nil {
		return false, StrInt32MapEntryRef{}, e
	}
	hash, e := hashStrKey(region, key)
	if e != nil {
		return false, StrInt32MapEntryRef{}, e
	}
	capacity := h.Capacity()
	table := h.table
	mask := capacity - 1
	i := hash & mask
	deleted := ^uint32(0)
	for step := uint32(0); step < capacity; step++ {
		entry, e := mapLegacyEntryAt(table, i)
		if e != nil {
			return false, StrInt32MapEntryRef{}, e
		}
		switch entry.state {
		case strHashUsed:
			if entry.hash == hash {
				storedRegion, e := h.keyRegion(entry.key)
				if e != nil {
					return false, StrInt32MapEntryRef{}, e
				}
				equal, e := equalStrKeys(storedRegion, entry.key, region, key)
				if e != nil {
					return false, StrInt32MapEntryRef{}, e
				}
				if equal {
					if overwrite {
						if e := h.adopt(region); e != nil {
							return false, StrInt32MapEntryRef{}, e
						}
						entry.key = key
						entry.value = value
					}
					return false, h.ref(i), nil
				}
			}
			if e = h.incrementCollision(); e != nil {
				return false, StrInt32MapEntryRef{}, e
			}
		case strHashFree:
			if e := h.canBump(); e != nil {
				return false, StrInt32MapEntryRef{}, e
			}
			if e := h.adopt(region); e != nil {
				return false, StrInt32MapEntryRef{}, e
			}
			slot := i
			if deleted != ^uint32(0) {
				slot = deleted
				h.deleted--
			}
			table[slot] = strInt32Entry{hash: hash, state: strHashUsed, key: key, value: value}
			h.size++
			h.bump()
			return true, h.ref(slot), nil
		default:
			on, e := h.assertionsEnabled()
			if e != nil {
				return false, StrInt32MapEntryRef{}, e
			}
			if on && entry.state != strHashDeleted {
				line := int32(371)
				if !overwrite {
					line = 422
				}
				if e = h.assertionFailure(line, "curr->is_deleted()"); e != nil {
					return false, StrInt32MapEntryRef{}, e
				}
			}
			deleted = i
			if e = h.incrementCollision(); e != nil {
				return false, StrInt32MapEntryRef{}, e
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
	return false, StrInt32MapEntryRef{}, h.unreachable(line)
}
func (h *StrInt32Map) Insert(k StrKey, value int32) error {
	_, _, e := h.insert(k, value, true)
	return e
}
func (h *StrInt32Map) InsertIfNotThereCore(k StrKey, value int32) (bool, StrInt32MapEntryRef, error) {
	return h.insert(k, value, false)
}
func (h *StrInt32Map) InsertIfNotThere(k StrKey, value int32) (StrInt32MapEntryRef, error) {
	_, r, e := h.insert(k, value, false)
	return r, e
}
func (h *StrInt32Map) InsertIfNotThere2(k StrKey, value int32) (StrInt32MapEntryRef, error) {
	return h.InsertIfNotThere(k, value)
}
func (h *StrInt32Map) InsertFrom(region *arena.BytesRegion, k StrKey, value int32) error {
	_, _, e := h.insertFrom(region, k, value, true)
	return e
}
func (h *StrInt32Map) InsertIfNotThereCoreFrom(region *arena.BytesRegion, k StrKey, value int32) (bool, StrInt32MapEntryRef, error) {
	return h.insertFrom(region, k, value, false)
}
func (h *StrInt32Map) InsertIfNotThereFrom(region *arena.BytesRegion, k StrKey, value int32) (StrInt32MapEntryRef, error) {
	_, r, e := h.insertFrom(region, k, value, false)
	return r, e
}
func (h *StrInt32Map) InsertIfNotThere2From(region *arena.BytesRegion, k StrKey, value int32) (StrInt32MapEntryRef, error) {
	return h.InsertIfNotThereFrom(region, k, value)
}
func (h *StrInt32Map) findSlot(key StrKey) (uint32, bool, error) {
	region, e := h.keyRegion(key)
	if e != nil {
		return 0, false, e
	}
	return h.findSlotFrom(region, key)
}
func (h *StrInt32Map) findSlotFrom(region *arena.BytesRegion, key StrKey) (uint32, bool, error) {
	return h.findSlotFromSource(region, key, nil)
}
func (h *StrInt32Map) findSlotFromSource(region *arena.BytesRegion, key StrKey, exhausted *bool) (uint32, bool, error) {
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
		var entry strInt32Entry
		if h.config.Runtime == nil {
			p, re := mapLegacyEntryAt(table, i)
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
func (h *StrInt32Map) FindCore(k StrKey) (StrInt32MapEntryRef, bool, error) {
	if e := h.open(); e != nil {
		return StrInt32MapEntryRef{}, false, e
	}
	i, ok, e := h.findSlot(k)
	if e != nil || !ok {
		return StrInt32MapEntryRef{}, ok, e
	}
	return h.ref(i), true, nil
}
func (h *StrInt32Map) Contains(k StrKey) (bool, error) { _, ok, e := h.FindCore(k); return ok, e }

// FindFrom reads an explicit foreign query owner without retaining it or
// allocating directory metadata. The returned ref borrows the stored entry.
func (h *StrInt32Map) FindFrom(region *arena.BytesRegion, k StrKey) (StrInt32MapEntryRef, bool, error) {
	if e := h.open(); e != nil {
		return StrInt32MapEntryRef{}, false, e
	}
	i, ok, e := h.findSlotFrom(region, k)
	if e != nil || !ok {
		return StrInt32MapEntryRef{}, ok, e
	}
	return h.ref(i), true, nil
}
func (h *StrInt32Map) ContainsFrom(region *arena.BytesRegion, k StrKey) (bool, error) {
	_, ok, e := h.FindFrom(region, k)
	return ok, e
}

// FindCString uses a transient external query; no Go string is stored in the
// table. Equality compares raw unsigned byte identities as source strcmp.
func (h *StrInt32Map) FindCString(s string) (StrInt32MapEntryRef, bool, error) {
	if e := h.open(); e != nil {
		return StrInt32MapEntryRef{}, false, e
	}
	s = strPrefix(s)
	if uint64(len(s)) > uint64(^uint32(0)) {
		return StrInt32MapEntryRef{}, false, arena.ErrBounds
	}
	hash := String(s, 17)
	capacity, e := h.capacityChecked()
	if e != nil {
		return StrInt32MapEntryRef{}, false, e
	}
	table := h.table
	mask := capacity - 1
	i := hash & mask
	for step := uint32(0); step < capacity; step++ {
		var entry strInt32Entry
		if h.config.Runtime == nil {
			p, re := mapLegacyEntryAt(table, i)
			if re != nil {
				e = re
			} else {
				entry = *p
			}
		} else {
			entry, e = h.probeEntry(i)
		}
		if e != nil {
			return StrInt32MapEntryRef{}, false, e
		}
		if entry.state == strHashUsed && entry.hash == hash {
			if h.config.Runtime != nil {
				entry.key, e = h.entryKey(i)
			}
			if e != nil {
				return StrInt32MapEntryRef{}, false, e
			}
			region, e := h.keyRegion(entry.key)
			if e != nil {
				return StrInt32MapEntryRef{}, false, e
			}
			equal, e := strKeyEqualString(region, entry.key, s)
			if e != nil {
				return StrInt32MapEntryRef{}, false, e
			}
			if equal {
				return h.ref(i), true, nil
			}
		}
		if entry.state == strHashUsed {
			if e := h.incrementCollision(); e != nil {
				return StrInt32MapEntryRef{}, false, e
			}
		}
		if entry.state == strHashFree {
			return StrInt32MapEntryRef{}, false, nil
		}
		if e := h.incrementCollision(); e != nil {
			return StrInt32MapEntryRef{}, false, e
		}
		i++
		if i == capacity {
			i = 0
		}
	}
	return StrInt32MapEntryRef{}, false, nil
}

// PrepareFindCString preserves source interner growth before duplicate lookup.
// Only a checked borrow escapes; a missing query leaves no temporary spelling
// in the table. Allocate the owner spelling, then publish with the normal insert.
func (h *StrInt32Map) PrepareFindCString(s string) (StrInt32MapEntryRef, bool, error) {
	if e := h.prepareInsert(); e != nil {
		return StrInt32MapEntryRef{}, false, e
	}
	r, found, e := h.FindCString(s)
	if e != nil || found {
		return r, found, e
	}
	// A miss authorizes the serialized caller to allocate its owned spelling.
	// Refuse exhaustion before that allocation, rather than only at publication.
	if e := h.canBump(); e != nil {
		return StrInt32MapEntryRef{}, false, e
	}
	return StrInt32MapEntryRef{}, false, nil
}
func (h *StrInt32Map) Remove(k StrKey) error {
	if e := h.open(); e != nil {
		return e
	}
	region, e := h.keyRegion(k)
	if e != nil {
		return e
	}
	return h.RemoveFrom(region, k)
}
func (h *StrInt32Map) RemoveFrom(region *arena.BytesRegion, k StrKey) error {
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
func (h *StrInt32Map) Erase(k StrKey) error { return h.Remove(k) }
func (h *StrInt32Map) EraseFrom(region *arena.BytesRegion, k StrKey) error {
	return h.RemoveFrom(region, k)
}
func (h *StrInt32Map) Reset() error {
	if h.config.Runtime != nil {
		return h.resetRuntime()
	}
	return h.resetLegacy()
}
func (h *StrInt32Map) Finalize() error {
	if h.config.Runtime != nil {
		return h.finalizeRuntime()
	}
	return h.finalizeLegacy()
}
func (h *StrInt32Map) Close() {
	if e := h.CloseChecked(); e != nil {
		panic(e)
	}
}
func (h *StrInt32Map) Clone() (*StrInt32Map, error) {
	if e := h.open(); e != nil {
		return nil, e
	}
	if h.config.Runtime != nil {
		return h.cloneRuntime(true)
	}
	return h.cloneLegacy()
}
func (h *StrInt32Map) bothOpen(other *StrInt32Map) error {
	if e := h.open(); e != nil {
		return e
	}
	if e := other.open(); e != nil {
		return e
	}
	return nil
}
func (h *StrInt32Map) Assign(src *StrInt32Map) error {
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
func (h *StrInt32Map) Union(src *StrInt32Map) error {
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
		entry := strInt32Entry{state: state}
		if state == strHashUsed {
			entry.key, e = src.entryKey(idx)
			if e != nil {
				return e
			}
			entry.value, e = src.valueAt(idx)
			if e != nil {
				return e
			}
		}
		if entry.state == strHashUsed {
			owner, e := src.keyRegion(entry.key)
			if e != nil {
				return e
			}
			if e := h.InsertFrom(owner, entry.key, entry.value); e != nil {
				return e
			}
		}
	}
	return nil
}
func (h *StrInt32Map) Intersect(src *StrInt32Map) error {
	if h.config.Runtime != nil {
		return h.intersectRuntime(src)
	}
	return h.intersectLegacy(src)
}
func (h *StrInt32Map) Swap(other *StrInt32Map) error {
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
func (h *StrInt32Map) CheckInvariant() bool {
	ok, e := h.CheckInvariantChecked()
	if e != nil {
		panic(e)
	}
	return ok
}

type StrInt32MapEntryRef struct {
	owner *StrInt32Map
	epoch uint64
	slot  uint32
}

func (h *StrInt32Map) ref(i uint32) StrInt32MapEntryRef { return StrInt32MapEntryRef{h, h.epoch, i} }
func (r StrInt32MapEntryRef) validate() error {
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
func (r StrInt32MapEntryRef) Key() (StrKey, error) {
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
func (r StrInt32MapEntryRef) Hash() (uint32, error) {
	if e := r.validate(); e != nil {
		return 0, e
	}
	entry, e := r.owner.probeEntry(r.slot)
	return entry.hash, e
}

// SetKey represents source e->set_data(owned_spelling) after interning. It
// preserves the cached hash and therefore requires identical C-string content.
func (r StrInt32MapEntryRef) SetKey(k StrKey) error {
	if e := r.validate(); e != nil {
		return e
	}
	region, e := r.owner.keyRegion(k)
	if e != nil {
		return e
	}
	return r.SetKeyFrom(region, k)
}
func (r StrInt32MapEntryRef) SetKeyFrom(region *arena.BytesRegion, k StrKey) error {
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

type StrInt32MapIterator struct {
	owner      *StrInt32Map
	epoch      uint64
	index, end uint32
}

func (h *StrInt32Map) Begin() StrInt32MapIterator {
	h.mustOpen()
	it := StrInt32MapIterator{owner: h, epoch: h.epoch, end: h.Capacity()}
	it.scan()
	return it
}
func (h *StrInt32Map) End() StrInt32MapIterator {
	h.mustOpen()
	return StrInt32MapIterator{h, h.epoch, h.Capacity(), h.Capacity()}
}
func (h *StrInt32Map) FindIterator(k StrKey) (StrInt32MapIterator, error) {
	r, ok, e := h.FindCore(k)
	if e != nil {
		return StrInt32MapIterator{}, e
	}
	if !ok {
		return h.End(), nil
	}
	return StrInt32MapIterator{h, h.epoch, r.slot, h.Capacity()}, nil
}
func (h *StrInt32Map) FindIteratorFrom(region *arena.BytesRegion, k StrKey) (StrInt32MapIterator, error) {
	r, ok, e := h.FindFrom(region, k)
	if e != nil {
		return StrInt32MapIterator{}, e
	}
	if !ok {
		return h.End(), nil
	}
	return StrInt32MapIterator{h, h.epoch, r.slot, h.Capacity()}, nil
}
func (it *StrInt32MapIterator) scan() {
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
func (it StrInt32MapIterator) validate() error {
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
func (it StrInt32MapIterator) AtEnd() bool { return it.index == it.end }
func (it StrInt32MapIterator) Equal(other StrInt32MapIterator) bool {
	return it.owner == other.owner && it.epoch == other.epoch && it.index == other.index
}
func (it StrInt32MapIterator) Ref() (StrInt32MapEntryRef, error) {
	if e := it.validate(); e != nil {
		return StrInt32MapEntryRef{}, e
	}
	if it.AtEnd() {
		return StrInt32MapEntryRef{}, arena.ErrBounds
	}
	return it.owner.ref(it.index), nil
}
func (it StrInt32MapIterator) Key() (StrKey, error) {
	r, e := it.Ref()
	if e != nil {
		return StrKey{}, e
	}
	return r.Key()
}
func (it *StrInt32MapIterator) Next() error {
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
func (it *StrInt32MapIterator) PostNext() (StrInt32MapIterator, error) {
	old := *it
	if e := it.Next(); e != nil {
		return StrInt32MapIterator{}, e
	}
	return old, nil
}

func (h *StrInt32Map) FindIntoFrom(region *arena.BytesRegion, k StrKey, out *int32) (bool, error) {
	r, found, e := h.FindFrom(region, k)
	if e != nil || !found {
		return found, e
	}
	value, e := r.Value()
	if e != nil {
		return false, e
	}
	*out = value
	return true, nil
}

func (h *StrInt32Map) Find(k StrKey) (int32, bool, error) {
	r, ok, e := h.FindCore(k)
	if e != nil || !ok {
		return 0, ok, e
	}
	value, e := r.Value()
	return value, true, e
}

func (h *StrInt32Map) FindInto(k StrKey, out *int32) (bool, error) {
	found, ok, e := h.Find(k)
	if e == nil && ok {
		*out = found
	}
	return ok, e
}

func (h *StrInt32Map) Get(k StrKey, fallback int32) (int32, error) {
	v, ok, e := h.Find(k)
	if e != nil {
		return 0, e
	}
	if !ok {
		return fallback, nil
	}
	return v, nil
}

func (h *StrInt32Map) FindFromValue(region *arena.BytesRegion, k StrKey) (int32, bool, error) {
	r, ok, e := h.FindFrom(region, k)
	if e != nil || !ok {
		return 0, ok, e
	}
	v, e := r.Value()
	return v, true, e
}

func (r StrInt32MapEntryRef) Value() (int32, error) {
	if e := r.validate(); e != nil {
		return 0, e
	}
	return r.owner.valueAt(r.slot)
}
func (r StrInt32MapEntryRef) SetValue(v int32) error {
	if e := r.validate(); e != nil {
		return e
	}
	return r.owner.putValueAt(r.slot, v)
}
func (it StrInt32MapIterator) Value() (int32, error) {
	r, e := it.Ref()
	if e != nil {
		return 0, e
	}
	return r.Value()
}

// At and AtConst port src/util/map.h:149-163 reference-returning find.
// A continued absent-entry assertion reaches a Go null-lvalue guard.
func (h *StrInt32Map) At(k StrKey) (StrInt32MapEntryRef, error) {
	return h.sourceValueReference(k, 157)
}
func (h *StrInt32Map) AtConst(k StrKey) (StrInt32MapEntryRef, error) {
	return h.sourceValueReference(k, 151)
}
func (h *StrInt32Map) sourceValueReference(k StrKey, line int32) (StrInt32MapEntryRef, error) {
	ref, found, e := h.FindCore(k)
	if e != nil {
		return StrInt32MapEntryRef{}, e
	}
	enabled, e := h.assertionsEnabled()
	if e != nil {
		return StrInt32MapEntryRef{}, e
	}
	if enabled && !found {
		if h.config.Assertions == nil {
			return StrInt32MapEntryRef{}, ErrStrHashtableAssertion
		}
		if e = h.config.Assertions.NotifyAssertionViolation("src/util/map.h", line, "e"); e != nil {
			return StrInt32MapEntryRef{}, e
		}
		if e = h.config.Assertions.InvokeAssertionDebugger(); e != nil {
			return StrInt32MapEntryRef{}, e
		}
	}
	if !found {
		return StrInt32MapEntryRef{}, arena.ErrBounds
	}
	return ref, nil
}
