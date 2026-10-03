// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
// Concrete str_hashtable/ptr_hashtable<char> from util/str_hashtable.h and
// util/hashtable.h, Leonardo de Moura (2006-09-13 / 2006-09-11).
package utilhash

import (
	"github.com/glycerine/g3/golang/arena"
	"reflect"
	"strings"
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

func (k StrKey) Allocation() arena.BytesRegionRef { return k.ref }
func (k StrKey) Offset() uint32                   { return k.offset }
func (k StrKey) Length() uint32                   { return k.length }

type strHashEntry struct {
	hash, state uint32
	key         StrKey
}
type StrHashtableConfig struct {
	Budget      *arena.Budget
	OutOfMemory func() bool
	// InitialCapacity is the source constructor parameter; zero selects8.
	InitialCapacity uint32
}

// StrHashtable binds one borrowed region. Close/Reset/Erase never release
// spelling bytes. Cross-region Swap exchanges the binding with its table;
// Assign/Union/Intersect require a shared binding. This owning struct must not
// be copied. Region Reset/rollback/Close invalidates all affected borrowed keys.
type StrHashtable struct {
	table                 []strHashEntry
	region                *arena.BytesRegion
	config                StrHashtableConfig
	size, deleted         uint32
	epoch                 uint64
	rootBytes, tableBytes uint64
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
	if r == nil {
		return nil, arena.ErrConfig
	}
	root := uint64(reflect.TypeOf(StrHashtable{}).Size())
	if e := c.Budget.Reserve(root); e != nil {
		return nil, e
	}
	h := &StrHashtable{region: r, config: c, epoch: 1, rootBytes: root}
	table, bytes, e := h.allocateTable(n)
	if e != nil {
		c.Budget.Release(root)
		return nil, e
	}
	h.table, h.tableBytes = table, bytes
	return h, nil
}
func (h *StrHashtable) allocateTable(n uint32) ([]strHashEntry, uint64, error) {
	if n == 0 || n&(n-1) != 0 || n > 1<<30 || uint64(n) > uint64(^uint(0)>>1)/strHashEntryBytes {
		return nil, 0, arena.ErrBounds
	}
	amount := uint64(n) * strHashEntryBytes
	if e := h.config.Budget.Reserve(amount); e != nil {
		return nil, 0, e
	}
	return make([]strHashEntry, n), amount, nil
}
func (h *StrHashtable) open() error {
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
func (h *StrHashtable) Size() uint32           { return h.size }
func (h *StrHashtable) Empty() bool            { return h.size == 0 }
func (h *StrHashtable) Capacity() uint32       { return uint32(len(h.table)) }
func (h *StrHashtable) Deleted() uint32        { return h.deleted }
func (h *StrHashtable) CollisionCount() uint64 { return 0 }

// StrHashtableStats accounts only this owner and scalar table; the borrowed
// region reports its own graph separately and is never charged by this table.
type StrHashtableStats struct {
	OwnerBytes, TableBytes  uint64
	Size, Deleted, Capacity uint32
}

func (h *StrHashtable) Stats() StrHashtableStats {
	return StrHashtableStats{h.rootBytes, h.tableBytes, h.size, h.deleted, h.Capacity()}
}

// CString finds source strlen within the allocation bound. An empty spelling
// still requires a live allocation containing its terminating NUL.
func (h *StrHashtable) CString(ref arena.BytesRegionRef, offset uint32) (StrKey, error) {
	if e := h.open(); e != nil {
		return StrKey{}, e
	}
	size, e := h.region.Size(ref)
	if e != nil {
		return StrKey{}, e
	}
	if !h.region.IsLive(ref) {
		return StrKey{}, arena.ErrInvalidSlot
	}
	if offset >= size {
		return StrKey{}, arena.ErrBounds
	}
	for i := offset; i < size; i++ {
		b, e := h.region.Read(ref, i)
		if e != nil {
			return StrKey{}, e
		}
		if b == 0 {
			return StrKey{ref, offset, i - offset, h.region.OwnerID()}, nil
		}
	}
	return StrKey{}, arena.ErrBounds
}
func (h *StrHashtable) validateKey(k StrKey) error {
	if k.origin != h.region.OwnerID() {
		return arena.ErrInvalidSlot
	}
	size, e := h.region.Size(k.ref)
	if e != nil {
		return e
	}
	if !h.region.IsLive(k.ref) {
		return arena.ErrInvalidSlot
	}
	if uint64(k.offset)+uint64(k.length) >= uint64(size) {
		return arena.ErrBounds
	}
	b, e := h.region.Read(k.ref, k.offset+k.length)
	if e != nil {
		return e
	}
	if b != 0 {
		return arena.ErrBounds
	}
	return nil
}

// Hash uses the source seed17, little-endian word loads and signed-char tail.
// Caller-controlled mutation of a stored spelling violates the source borrow.
func (h *StrHashtable) Hash(k StrKey) (uint32, error) {
	if e := h.open(); e != nil {
		return 0, e
	}
	if e := h.validateKey(k); e != nil {
		return 0, e
	}
	var readError error
	hash := WordReader(k.length, 17, func(word uint32) uint32 {
		var v uint32
		for j := uint32(0); j < 4; j++ {
			i := word*4 + j
			if i >= k.length {
				break
			}
			b, e := h.region.Read(k.ref, k.offset+i)
			if e != nil {
				readError = e
				return 0
			}
			v |= uint32(b) << (8 * j)
		}
		return v
	})
	return hash, readError
}
func (h *StrHashtable) Equal(a, b StrKey) (bool, error) {
	if e := h.open(); e != nil {
		return false, e
	}
	if e := h.validateKey(a); e != nil {
		return false, e
	}
	if e := h.validateKey(b); e != nil {
		return false, e
	}
	if a.length != b.length {
		return false, nil
	}
	for i := uint32(0); i < a.length; i++ {
		x, e := h.region.Read(a.ref, a.offset+i)
		if e != nil {
			return false, e
		}
		y, e := h.region.Read(b.ref, b.offset+i)
		if e != nil {
			return false, e
		}
		if x != y {
			return false, nil
		}
	}
	return true, nil
}
func (h *StrHashtable) Text(k StrKey) (string, error) {
	if e := h.open(); e != nil {
		return "", e
	}
	if e := h.validateKey(k); e != nil {
		return "", e
	}
	var out strings.Builder
	out.Grow(int(k.length))
	for i := uint32(0); i < k.length; i++ {
		b, e := h.region.Read(k.ref, k.offset+i)
		if e != nil {
			return "", e
		}
		out.WriteByte(b)
	}
	return out.String(), nil
}
func copyStrTable(source, target []strHashEntry) {
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
func (h *StrHashtable) rehash(n uint32) error {
	if e := h.canBump(); e != nil {
		return e
	}
	table, amount, e := h.allocateTable(n)
	if e != nil {
		return e
	}
	copyStrTable(h.table, table)
	h.config.Budget.Release(h.tableBytes)
	h.table, h.tableBytes = table, amount
	h.deleted = 0
	h.bump()
	return nil
}
func (h *StrHashtable) prepareInsert() error {
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
func (h *StrHashtable) insert(key StrKey, overwrite bool) (bool, StrHashtableEntryRef, error) {
	if e := h.prepareInsert(); e != nil {
		return false, StrHashtableEntryRef{}, e
	}
	hash, e := h.Hash(key)
	if e != nil {
		return false, StrHashtableEntryRef{}, e
	}
	mask := h.Capacity() - 1
	i := hash & mask
	deleted := ^uint32(0)
	for step := uint32(0); step < h.Capacity(); step++ {
		entry := &h.table[i]
		switch entry.state {
		case strHashUsed:
			if entry.hash == hash {
				equal, e := h.Equal(entry.key, key)
				if e != nil {
					return false, StrHashtableEntryRef{}, e
				}
				if equal {
					if overwrite {
						entry.key = key
					}
					return false, h.ref(i), nil
				}
			}
		case strHashFree:
			if e := h.canBump(); e != nil {
				return false, StrHashtableEntryRef{}, e
			}
			slot := i
			if deleted != ^uint32(0) {
				slot = deleted
				h.deleted--
			}
			h.table[slot] = strHashEntry{hash, strHashUsed, key}
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
func (h *StrHashtable) findSlot(key StrKey) (uint32, bool, error) {
	hash, e := h.Hash(key)
	if e != nil {
		return 0, false, e
	}
	mask := h.Capacity() - 1
	i := hash & mask
	for step := uint32(0); step < h.Capacity(); step++ {
		entry := h.table[i]
		if entry.state == strHashUsed && entry.hash == hash {
			equal, e := h.Equal(entry.key, key)
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
func (h *StrHashtable) Find(k StrKey) (StrKey, bool, error) {
	r, ok, e := h.FindCore(k)
	if e != nil || !ok {
		return StrKey{}, ok, e
	}
	return h.table[r.slot].key, true, nil
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
	mask := h.Capacity() - 1
	i := hash & mask
	for step := uint32(0); step < h.Capacity(); step++ {
		entry := h.table[i]
		if entry.state == strHashUsed && entry.hash == hash {
			if e := h.validateKey(entry.key); e != nil {
				return StrHashtableEntryRef{}, false, e
			}
			equal := entry.key.length == uint32(len(s))
			if equal {
				for j := uint32(0); j < entry.key.length; j++ {
					b, e := h.region.Read(entry.key.ref, entry.key.offset+j)
					if e != nil {
						return StrHashtableEntryRef{}, false, e
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
			return StrHashtableEntryRef{}, false, nil
		}
		i = (i + 1) & mask
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
	i, ok, e := h.findSlot(k)
	if e != nil || !ok {
		return e
	}
	if e := h.canBump(); e != nil {
		return e
	}
	next := (i + 1) & (h.Capacity() - 1)
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
func (h *StrHashtable) replaceEmpty(n uint32) error {
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
func (h *StrHashtable) Reset() error {
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
		h.table[i].key = StrKey{}
	}
	h.size, h.deleted = 0, 0
	h.bump()
	return nil
}
func (h *StrHashtable) Finalize() error {
	if e := h.open(); e != nil {
		return e
	}
	if h.Capacity() > 64 {
		return h.replaceEmpty(64)
	}
	return h.Reset()
}
func (h *StrHashtable) Close() {
	if h.closed {
		return
	}
	h.config.Budget.Release(h.rootBytes + h.tableBytes)
	h.table = nil
	h.rootBytes, h.tableBytes = 0, 0
	h.size, h.deleted = 0, 0
	h.closed = true
	if h.epoch != ^uint64(0) {
		h.bump()
	}
}
func (h *StrHashtable) Clone() (*StrHashtable, error) {
	if e := h.open(); e != nil {
		return nil, e
	}
	r, e := newStrHashtableCapacity(h.region, h.config, h.Capacity())
	if e != nil {
		return nil, e
	}
	copyStrTable(h.table, r.table)
	r.size = h.size
	return r, nil
}
func (h *StrHashtable) sameRegion(other *StrHashtable) error {
	if e := h.open(); e != nil {
		return e
	}
	if e := other.open(); e != nil {
		return e
	}
	if h.region != other.region {
		return arena.ErrConfig
	}
	return nil
}
func (h *StrHashtable) Assign(src *StrHashtable) error {
	if e := h.sameRegion(src); e != nil {
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
	if e := h.sameRegion(src); e != nil {
		return e
	}
	if h == src {
		return nil
	}
	for _, entry := range src.table {
		if entry.state == strHashUsed {
			if e := h.Insert(entry.key); e != nil {
				return e
			}
		}
	}
	return nil
}
func (h *StrHashtable) Intersect(src *StrHashtable) error {
	if e := h.sameRegion(src); e != nil {
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
			ok, e := src.Contains(entry.key)
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
func (h *StrHashtable) Swap(other *StrHashtable) error {
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
		if e := h.config.Budget.Reserve(other.tableBytes); e != nil {
			return e
		}
		if e := other.config.Budget.Reserve(h.tableBytes); e != nil {
			h.config.Budget.Release(other.tableBytes)
			return e
		}
		h.config.Budget.Release(h.tableBytes)
		other.config.Budget.Release(other.tableBytes)
	}
	h.table, other.table = other.table, h.table
	h.region, other.region = other.region, h.region
	h.tableBytes, other.tableBytes = other.tableBytes, h.tableBytes
	h.size, other.size = other.size, h.size
	h.deleted, other.deleted = other.deleted, h.deleted
	h.bump()
	other.bump()
	return nil
}
func (h *StrHashtable) CheckInvariant() bool {
	if h.open() != nil {
		return false
	}
	var used, deleted uint32
	for _, e := range h.table {
		if e.state == strHashUsed {
			used++
		} else if e.state == strHashDeleted {
			deleted++
		}
	}
	return used == h.size && deleted == h.deleted
}

// Dump is the source [spellings separated by spaces] byte-oriented format.
func (h *StrHashtable) Dump() (string, error) {
	if e := h.open(); e != nil {
		return "", e
	}
	var out strings.Builder
	out.WriteByte('[')
	first := true
	for _, entry := range h.table {
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
	if r.epoch != r.owner.epoch || r.slot >= r.owner.Capacity() || r.owner.table[r.slot].state != strHashUsed {
		return arena.ErrInvalidSlot
	}
	return nil
}
func (r StrHashtableEntryRef) Key() (StrKey, error) {
	if e := r.validate(); e != nil {
		return StrKey{}, e
	}
	k := r.owner.table[r.slot].key
	if e := r.owner.validateKey(k); e != nil {
		return StrKey{}, e
	}
	return k, nil
}
func (r StrHashtableEntryRef) Hash() (uint32, error) {
	if e := r.validate(); e != nil {
		return 0, e
	}
	return r.owner.table[r.slot].hash, nil
}

// SetKey represents source e->set_data(owned_spelling) after interning. It
// preserves the cached hash and therefore requires identical C-string content.
func (r StrHashtableEntryRef) SetKey(k StrKey) error {
	if e := r.validate(); e != nil {
		return e
	}
	hash, e := r.owner.Hash(k)
	if e != nil {
		return e
	}
	entry := &r.owner.table[r.slot]
	if hash != entry.hash {
		return arena.ErrBounds
	}
	equal, e := r.owner.Equal(entry.key, k)
	if e != nil {
		return e
	}
	if !equal {
		return arena.ErrBounds
	}
	entry.key = k
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
func (it *StrHashtableIterator) scan() {
	for it.index < it.end && it.owner.table[it.index].state != strHashUsed {
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
