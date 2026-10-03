// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
// Concrete map<char const*, int, str_hash_proc, str_eq_proc> from util/map.h
// and core_hashtable default entries, Leonardo de Moura (2006-10-14).
package utilhash

import (
	"github.com/glycerine/g3/golang/arena"
	"reflect"
	"strings"
)

const strInt32EntryBytes uint64 = 40

type strInt32Entry struct {
	hash, state uint32
	key         StrKey
	value       int32
	// Keep the charged physical record width40 on 32-bit targets as well.
	padding uint32
}
type StrInt32MapConfig struct {
	Budget      *arena.Budget
	OutOfMemory func() bool
	// InitialCapacity is the source constructor parameter; zero selects8.
	InitialCapacity uint32
}

// StrInt32Map borrows its primary region and every adopted secondary region.
// Close/Reset/Erase never release spelling bytes. Swap exchanges bindings with
// the table. This owning struct must not be copied. Region Reset/rollback/Close
// invalidates affected keys; callers keep all borrowed regions alive.
type StrInt32Map struct {
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
	if r == nil {
		return nil, arena.ErrConfig
	}
	root := uint64(reflect.TypeOf(StrInt32Map{}).Size())
	if e := c.Budget.Reserve(root); e != nil {
		return nil, e
	}
	h := &StrInt32Map{region: r, config: c, epoch: 1, rootBytes: root}
	table, bytes, e := h.allocateTable(n)
	if e != nil {
		c.Budget.Release(root)
		return nil, e
	}
	h.table, h.tableBytes = table, bytes
	return h, nil
}
func (h *StrInt32Map) allocateTable(n uint32) ([]strInt32Entry, uint64, error) {
	if n == 0 || n&(n-1) != 0 || n > 1<<30 || uint64(n) > uint64(^uint(0)>>1)/strInt32EntryBytes {
		return nil, 0, arena.ErrBounds
	}
	amount := uint64(n) * strInt32EntryBytes
	if e := h.config.Budget.Reserve(amount); e != nil {
		return nil, 0, e
	}
	return make([]strInt32Entry, n), amount, nil
}
func (h *StrInt32Map) open() error {
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
func (h *StrInt32Map) Size() uint32           { return h.size }
func (h *StrInt32Map) Empty() bool            { return h.size == 0 }
func (h *StrInt32Map) Capacity() uint32       { return uint32(len(h.table)) }
func (h *StrInt32Map) Deleted() uint32        { return h.deleted }
func (h *StrInt32Map) CollisionCount() uint64 { return 0 }

// StrInt32MapStats accounts this owner, scalar table and cold owner directory.
// Borrowed regions report their own graphs and are never charged by this table.
type StrInt32MapStats struct {
	OwnerBytes, TableBytes, DirectoryBytes uint64
	Size, Deleted, Capacity                uint32
}

func (h *StrInt32Map) Stats() StrInt32MapStats {
	return StrInt32MapStats{h.rootBytes, h.tableBytes, h.directoryBytes, h.size, h.deleted, h.Capacity()}
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

func (h *StrInt32Map) validateKey(k StrKey) error {
	r, e := h.keyRegion(k)
	if e != nil {
		return e
	}
	return validateStrKey(r, k)
}
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
func copyStrInt32Table(source, target []strInt32Entry) {
	mask := uint32(len(target)) - 1
	for _, entry := range source {
		if entry.state != strHashUsed {
			continue
		}
		i := entry.hash & mask
		for target[i].state != strHashFree {
			i = (i + 1) & mask
		}
		target[i] = entry
	}
}
func (h *StrInt32Map) rehash(n uint32) error {
	if e := h.canBump(); e != nil {
		return e
	}
	table, amount, e := h.allocateTable(n)
	if e != nil {
		return e
	}
	copyStrInt32Table(h.table, table)
	h.config.Budget.Release(h.tableBytes)
	h.table, h.tableBytes = table, amount
	h.deleted = 0
	h.bump()
	return nil
}
func (h *StrInt32Map) prepareInsert() error {
	if e := h.open(); e != nil {
		return e
	}
	if uint64(h.size+h.deleted)*4 > uint64(h.Capacity())*3 {
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
	if e := h.prepareInsert(); e != nil {
		return false, StrInt32MapEntryRef{}, e
	}
	hash, e := hashStrKey(region, key)
	if e != nil {
		return false, StrInt32MapEntryRef{}, e
	}
	mask := h.Capacity() - 1
	i := hash & mask
	deleted := ^uint32(0)
	for step := uint32(0); step < h.Capacity(); step++ {
		entry := &h.table[i]
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
			h.table[slot] = strInt32Entry{hash: hash, state: strHashUsed, key: key, value: value}
			h.size++
			h.bump()
			return true, h.ref(slot), nil
		case strHashDeleted:
			deleted = i
		}
		i = (i + 1) & mask
	}
	panic("utilhash: source string table has no free insertion slot")
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
	hash, e := hashStrKey(region, key)
	if e != nil {
		return 0, false, e
	}
	mask := h.Capacity() - 1
	i := hash & mask
	for step := uint32(0); step < h.Capacity(); step++ {
		entry := h.table[i]
		if entry.state == strHashUsed && entry.hash == hash {
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
		if entry.state == strHashFree {
			return 0, false, nil
		}
		i = (i + 1) & mask
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
	return h.table[r.slot].value, true, nil
}
func (h *StrInt32Map) FindInto(k StrKey, out *int32) (bool, error) {
	found, ok, e := h.Find(k)
	if e == nil && ok {
		*out = found
	}
	return ok, e
}
func (h *StrInt32Map) FindCString(s string) (StrInt32MapEntryRef, bool, error) {
	if e := h.open(); e != nil {
		return StrInt32MapEntryRef{}, false, e
	}
	s = strPrefix(s)
	if uint64(len(s)) > uint64(^uint32(0)) {
		return StrInt32MapEntryRef{}, false, arena.ErrBounds
	}
	hash := String(s, 17)
	mask := h.Capacity() - 1
	i := hash & mask
	for step := uint32(0); step < h.Capacity(); step++ {
		entry := h.table[i]
		if entry.state == strHashUsed && entry.hash == hash {
			if e := h.validateKey(entry.key); e != nil {
				return StrInt32MapEntryRef{}, false, e
			}
			region, e := h.keyRegion(entry.key)
			if e != nil {
				return StrInt32MapEntryRef{}, false, e
			}
			equal := entry.key.length == uint32(len(s))
			if equal {
				for j := uint32(0); j < entry.key.length; j++ {
					b, e := region.Read(entry.key.ref, entry.key.offset+j)
					if e != nil {
						return StrInt32MapEntryRef{}, false, e
					}
					if b != s[j] {
						equal = false
						break
					}
				}
			}
			if equal {
				return h.ref(i), true, nil
			}
		}
		if entry.state == strHashFree {
			return StrInt32MapEntryRef{}, false, nil
		}
		i = (i + 1) & mask
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
	if e := h.open(); e != nil {
		return e
	}
	i, ok, e := h.findSlotFrom(region, k)
	if e != nil || !ok {
		return e
	}
	if e := h.canBump(); e != nil {
		return e
	}
	next := (i + 1) & (h.Capacity() - 1)
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
func (h *StrInt32Map) replaceEmpty(n uint32) error {
	if e := h.canBump(); e != nil {
		return e
	}
	table, amount, e := h.allocateTable(n)
	if e != nil {
		return e
	}
	h.config.Budget.Release(h.tableBytes)
	h.table, h.tableBytes = table, amount
	h.size, h.deleted = 0, 0
	h.bump()
	return nil
}
func (h *StrInt32Map) Reset() error {
	if e := h.open(); e != nil {
		return e
	}
	if h.size == 0 && h.deleted == 0 {
		return nil
	}
	if e := h.canBump(); e != nil {
		return e
	}
	var overhead uint32
	for _, entry := range h.table {
		if entry.state == strHashFree {
			overhead++
		}
	}
	if h.Capacity() > 16 && uint64(overhead)*4 > uint64(h.Capacity())*3 {
		return h.replaceEmpty(h.Capacity() >> 1)
	}
	for i := range h.table {
		h.table[i].state = strHashFree
	}
	h.size, h.deleted = 0, 0
	h.bump()
	return nil
}
func (h *StrInt32Map) Finalize() error {
	if e := h.open(); e != nil {
		return e
	}
	if h.Capacity() > 64 {
		return h.replaceEmpty(64)
	}
	return h.Reset()
}
func (h *StrInt32Map) Close() {
	if h.closed {
		return
	}
	h.config.Budget.Release(h.rootBytes + h.tableBytes + h.directoryBytes)
	h.table = nil
	h.owners = nil
	h.region = nil
	h.rootBytes, h.tableBytes = 0, 0
	h.directoryBytes = 0
	h.size, h.deleted = 0, 0
	h.closed = true
	if h.epoch != ^uint64(0) {
		h.bump()
	}
}
func (h *StrInt32Map) Clone() (*StrInt32Map, error) {
	if e := h.open(); e != nil {
		return nil, e
	}
	r, e := newStrInt32MapCapacity(h.region, h.config, h.Capacity())
	if e != nil {
		return nil, e
	}
	mask := r.Capacity() - 1
	for _, entry := range h.table {
		if entry.state != strHashUsed {
			continue
		}
		owner, e := h.keyRegion(entry.key)
		if e != nil {
			r.Close()
			return nil, e
		}
		if e := r.adopt(owner); e != nil {
			r.Close()
			return nil, e
		}
		i := entry.hash & mask
		for r.table[i].state != strHashFree {
			i = (i + 1) & mask
		}
		r.table[i] = entry
	}
	r.size = h.size
	return r, nil
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
	for _, entry := range src.table {
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
	if e := h.bothOpen(src); e != nil {
		return e
	}
	if h == src {
		return nil
	}
	copy, e := h.Clone()
	if e != nil {
		return e
	}
	defer copy.Close()
	for _, entry := range copy.table {
		if entry.state == strHashUsed {
			owner, e := copy.keyRegion(entry.key)
			if e != nil {
				return e
			}
			ok, e := src.ContainsFrom(owner, entry.key)
			if e != nil {
				return e
			}
			if !ok {
				if e := h.Remove(entry.key); e != nil {
					return e
				}
			}
		}
	}
	return nil
}
func (h *StrInt32Map) Swap(other *StrInt32Map) error {
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
	h.region, other.region = other.region, h.region
	h.owners, other.owners = other.owners, h.owners
	h.directoryBytes, other.directoryBytes = other.directoryBytes, h.directoryBytes
	h.tableBytes, other.tableBytes = other.tableBytes, h.tableBytes
	h.size, other.size = other.size, h.size
	h.deleted, other.deleted = other.deleted, h.deleted
	h.bump()
	other.bump()
	return nil
}
func (h *StrInt32Map) CheckInvariant() bool {
	if h.open() != nil {
		return false
	}
	var used, deleted uint32
	for i, owner := range h.owners {
		if owner.region == nil || owner.origin != owner.region.OwnerID() || owner.origin == h.region.OwnerID() || i != 0 && h.owners[i-1].origin >= owner.origin {
			return false
		}
	}
	for _, e := range h.table {
		if e.state == strHashUsed {
			used++
			if _, err := h.keyRegion(e.key); err != nil {
				return false
			}
		} else if e.state == strHashDeleted {
			deleted++
		}
	}
	return used == h.size && deleted == h.deleted
}

// EntryRef is cold owner/epoch metadata replacing source borrowed entry pointers.
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
	if r.epoch != r.owner.epoch || r.slot >= r.owner.Capacity() || r.owner.table[r.slot].state != strHashUsed {
		return arena.ErrInvalidSlot
	}
	return nil
}
func (r StrInt32MapEntryRef) Key() (StrKey, error) {
	if e := r.validate(); e != nil {
		return StrKey{}, e
	}
	k := r.owner.table[r.slot].key
	if e := r.owner.validateKey(k); e != nil {
		return StrKey{}, e
	}
	return k, nil
}
func (r StrInt32MapEntryRef) Hash() (uint32, error) {
	if e := r.validate(); e != nil {
		return 0, e
	}
	return r.owner.table[r.slot].hash, nil
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
	entry := &r.owner.table[r.slot]
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
	entry.key = k
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
	for it.index < it.end && it.owner.table[it.index].state != strHashUsed {
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

// Get implements the source default-value find overload, copying the scalar.
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

// Value/SetValue replace the source borrowed int reference and operator[].
func (r StrInt32MapEntryRef) Value() (int32, error) {
	if e := r.validate(); e != nil {
		return 0, e
	}
	return r.owner.table[r.slot].value, nil
}
func (r StrInt32MapEntryRef) SetValue(v int32) error {
	if e := r.validate(); e != nil {
		return e
	}
	r.owner.table[r.slot].value = v
	return nil
}
func (it StrInt32MapIterator) Value() (int32, error) {
	r, e := it.Ref()
	if e != nil {
		return 0, e
	}
	return r.Value()
}
