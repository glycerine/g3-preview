// Licensed under the MIT license.
// Concrete translation of u_map<unsigned> in src/util/map.h (2006-10-14)
// and core_hashtable/default_hash_entry in src/util/hashtable.h (2006-09-11).
// Author: Leonardo de Moura.
package utilhash

import (
	"github.com/glycerine/g3/golang/arena"
	"reflect"
)

const (
	uintMapFree uint32 = iota
	uintMapDeleted
	uintMapUsed
)
const uintMapEntryBytes uint64 = 16

// UintMapConfig supplies owner-local resources. Callback storage is cold;
// scalar entries do not carry pointers, interfaces, or callbacks.
type UintMapConfig struct {
	Budget      *arena.Budget
	OutOfMemory func() bool
}
type UintMapPair struct{ Key, Value uint32 }
type uintMapEntry struct{ hash, state, key, value uint32 }

// UintMap specializes source u_map<unsigned>: key hash is key identity, with
// source linear probing and state-tagged entries. Zero keys/values are valid.
// Construct via NewUintMap; do not copy this owning Go struct.
type UintMap struct {
	table                 []uintMapEntry
	config                UintMapConfig
	size, deleted         uint32
	epoch                 uint64
	rootBytes, tableBytes uint64
	closed                bool
}

func NewUintMap(config UintMapConfig) (*UintMap, error) { return newUintMapCapacity(config, 8) }
func newUintMapCapacity(config UintMapConfig, n uint32) (*UintMap, error) {
	root := uint64(reflect.TypeOf(UintMap{}).Size())
	if e := config.Budget.Reserve(root); e != nil {
		return nil, e
	}
	m := &UintMap{rootBytes: root}
	if e := m.constructMembers(config, n); e != nil {
		config.Budget.Release(root)
		return nil, e
	}
	return m, nil
}

// InitializeInlineUintMap constructs the actual embedded u_map<unsigned> member
// in src/ast/ast.h:1480. Its containing owner charges the concrete wrapper once;
// the same src/util/hashtable.h:244-256 engine acquires only its entry array.
// The receiver must be a fresh, unconstructed value. Failed construction leaves
// no completed member or payload; Close is explicit Go owning retirement only.
func InitializeInlineUintMap(m *UintMap, config UintMapConfig) error {
	if m == nil || m.epoch != 0 || m.closed || m.table != nil || m.rootBytes != 0 || m.tableBytes != 0 || m.size != 0 || m.deleted != 0 {
		return arena.ErrConfig
	}
	return m.constructMembers(config, 8)
}

func (m *UintMap) constructMembers(config UintMapConfig, n uint32) error {
	m.config, m.epoch = config, 1
	table, amount, e := m.allocateTable(n)
	if e != nil {
		return e
	}
	m.table, m.tableBytes = table, amount
	m.size, m.deleted = 0, 0
	return nil
}
func (m *UintMap) allocateTable(n uint32) ([]uintMapEntry, uint64, error) {
	if n < 8 || n&(n-1) != 0 || n > 1<<30 || uint64(n) > uint64(^uint(0)>>1)/uintMapEntryBytes {
		return nil, 0, arena.ErrBounds
	}
	amount := uint64(n) * uintMapEntryBytes
	if e := m.config.Budget.Reserve(amount); e != nil {
		return nil, 0, e
	}
	return make([]uintMapEntry, n), amount, nil
}
func (m *UintMap) bump() {
	if m.epoch == ^uint64(0) {
		panic("utilhash: uint map epoch exhausted")
	}
	m.epoch++
}
func (m *UintMap) open() error {
	if m.closed || len(m.table) == 0 {
		return arena.ErrClosed
	}
	return nil
}
func (m *UintMap) mustOpen() {
	if e := m.open(); e != nil {
		panic(e)
	}
}
func (m *UintMap) Size() uint32     { return m.size }
func (m *UintMap) Empty() bool      { return m.size == 0 }
func (m *UintMap) Capacity() uint32 { return uint32(len(m.table)) }
func (m *UintMap) Deleted() uint32  { return m.deleted }

// StorageBytes names this concrete owner's actual header/entry backing, including
// embedded construction whose header is charged by its containing owner.
func (m *UintMap) StorageBytes() uint64 { return m.rootBytes + m.tableBytes }

// The source HASHTABLE_STATISTICS definition is disabled in this baseline.
func (m *UintMap) CollisionCount() uint64 { return 0 }
func copyUintMapTable(source, target []uintMapEntry) {
	mask := uint32(len(target)) - 1
	for _, entry := range source {
		if entry.state != uintMapUsed {
			continue
		}
		i := entry.hash & mask
		for target[i].state != uintMapFree {
			i = (i + 1) & mask
		}
		target[i] = entry
	}
}
func (m *UintMap) rehash(n uint32) error {
	table, amount, e := m.allocateTable(n)
	if e != nil {
		return e
	}
	copyUintMapTable(m.table, table)
	m.config.Budget.Release(m.tableBytes)
	m.table, m.tableBytes = table, amount
	m.deleted = 0
	m.bump()
	return nil
}
func (m *UintMap) prepareInsert() error {
	if e := m.open(); e != nil {
		return e
	}
	if uint64(m.size+m.deleted)*4 > uint64(m.Capacity())*3 {
		if m.Capacity() > 1<<29 {
			return arena.ErrBounds
		}
		return m.rehash(m.Capacity() << 1)
	}
	return nil
}
func (m *UintMap) insert(key, value uint32, overwrite bool) (bool, UintMapEntryRef, error) {
	if e := m.prepareInsert(); e != nil {
		return false, UintMapEntryRef{}, e
	}
	hash := key
	mask := m.Capacity() - 1
	i := hash & mask
	deleted := ^uint32(0)
	for step := uint32(0); step < m.Capacity(); step++ {
		entry := &m.table[i]
		switch entry.state {
		case uintMapUsed:
			if entry.hash == hash && entry.key == key {
				if overwrite {
					entry.value = value
				}
				return false, m.ref(i), nil
			}
		case uintMapFree:
			slot := i
			if deleted != ^uint32(0) {
				slot = deleted
				m.deleted--
			}
			m.table[slot] = uintMapEntry{hash: hash, state: uintMapUsed, key: key, value: value}
			m.size++
			m.bump()
			return true, m.ref(slot), nil
		case uintMapDeleted:
			deleted = i
		}
		i = (i + 1) & mask
	}
	panic("utilhash: source hash table has no free insertion slot")
}
func (m *UintMap) Insert(key, value uint32) error { _, _, e := m.insert(key, value, true); return e }
func (m *UintMap) InsertIfNotThereCore(key, value uint32) (bool, UintMapEntryRef, error) {
	return m.insert(key, value, false)
}
func (m *UintMap) InsertIfNotThere(key, value uint32) (UintMapEntryRef, error) {
	_, r, e := m.insert(key, value, false)
	return r, e
}
func (m *UintMap) InsertIfNotThere2(key, value uint32) (UintMapEntryRef, error) {
	return m.InsertIfNotThere(key, value)
}
func (m *UintMap) findSlot(key uint32) (uint32, bool) {
	m.mustOpen()
	mask := m.Capacity() - 1
	i := key & mask
	for step := uint32(0); step < m.Capacity(); step++ {
		entry := m.table[i]
		if entry.state == uintMapUsed && entry.hash == key && entry.key == key {
			return i, true
		}
		if entry.state == uintMapFree {
			return 0, false
		}
		i = (i + 1) & mask
	}
	return 0, false
}
func (m *UintMap) Contains(key uint32) bool { _, ok := m.findSlot(key); return ok }
func (m *UintMap) Find(key uint32) (uint32, bool) {
	i, ok := m.findSlot(key)
	if !ok {
		return 0, false
	}
	return m.table[i].value, true
}
func (m *UintMap) FindInto(key uint32, out *uint32) bool {
	v, ok := m.Find(key)
	if ok {
		*out = v
	}
	return ok
}
func (m *UintMap) Get(key, defaultValue uint32) uint32 {
	v, ok := m.Find(key)
	if ok {
		return v
	}
	return defaultValue
}
func (m *UintMap) MustFind(key uint32) uint32 {
	v, ok := m.Find(key)
	if !ok {
		panic(arena.ErrInvalidSlot)
	}
	return v
}
func (m *UintMap) FindCore(key uint32) (UintMapEntryRef, bool) {
	i, ok := m.findSlot(key)
	if !ok {
		return UintMapEntryRef{}, false
	}
	return m.ref(i), true
}
func (m *UintMap) Remove(key uint32) error {
	if e := m.open(); e != nil {
		return e
	}
	i, ok := m.findSlot(key)
	if !ok {
		return nil
	}
	next := (i + 1) & (m.Capacity() - 1)
	m.size--
	m.bump()
	if m.table[next].state == uintMapFree {
		m.table[i].state = uintMapFree
		return nil
	}
	m.table[i].state = uintMapDeleted
	m.deleted++
	if m.deleted > m.size && m.deleted > 64 {
		if m.config.OutOfMemory != nil && m.config.OutOfMemory() {
			return nil
		}
		return m.rehash(m.Capacity())
	}
	return nil
}
func (m *UintMap) Erase(key uint32) error { return m.Remove(key) }
func (m *UintMap) Reset() error {
	if e := m.open(); e != nil {
		return e
	}
	if m.size == 0 && m.deleted == 0 {
		return nil
	}
	overhead := uint32(0)
	for _, entry := range m.table {
		if entry.state == uintMapFree {
			overhead++
		}
	}
	cap := m.Capacity()
	if cap > 16 && uint64(overhead)*4 > uint64(cap)*3 {
		// Reserve/allocate before clearing: budget failures leave live source data
		// intact instead of reproducing the source's unsafe post-delete failure.
		table, amount, e := m.allocateTable(cap >> 1)
		if e != nil {
			return e
		}
		m.config.Budget.Release(m.tableBytes)
		m.table, m.tableBytes = table, amount
	} else {
		for i := range m.table {
			m.table[i].state = uintMapFree
		}
	}
	m.size, m.deleted = 0, 0
	m.bump()
	return nil
}
func (m *UintMap) Finalize() error {
	if e := m.open(); e != nil {
		return e
	}
	if m.Capacity() > 64 {
		table, amount, e := m.allocateTable(64)
		if e != nil {
			return e
		}
		m.config.Budget.Release(m.tableBytes)
		m.table, m.tableBytes = table, amount
		m.size, m.deleted = 0, 0
		m.bump()
		return nil
	}
	return m.Reset()
}
func (m *UintMap) Close() {
	if m.closed {
		return
	}
	m.config.Budget.Release(m.tableBytes + m.rootBytes)
	m.table = nil
	m.tableBytes, m.rootBytes = 0, 0
	m.size, m.deleted = 0, 0
	m.closed = true
	m.bump()
}
func (m *UintMap) Clone() (*UintMap, error) {
	if e := m.open(); e != nil {
		return nil, e
	}
	r, e := newUintMapCapacity(m.config, m.Capacity())
	if e != nil {
		return nil, e
	}
	copyUintMapTable(m.table, r.table)
	r.size = m.size
	return r, nil
}
func (m *UintMap) Assign(src *UintMap) error {
	if e := m.open(); e != nil {
		return e
	}
	if e := src.open(); e != nil {
		return e
	}
	if m == src {
		return nil
	}
	if e := m.Reset(); e != nil {
		return e
	}
	for _, entry := range src.table {
		if entry.state == uintMapUsed {
			if e := m.Insert(entry.key, entry.value); e != nil {
				return e
			}
		}
	}
	return nil
}
func (m *UintMap) Union(src *UintMap) error {
	if e := m.open(); e != nil {
		return e
	}
	if e := src.open(); e != nil {
		return e
	}
	if m == src {
		return nil
	}
	for _, entry := range src.table {
		if entry.state == uintMapUsed {
			if e := m.Insert(entry.key, entry.value); e != nil {
				return e
			}
		}
	}
	return nil
}
func (m *UintMap) Intersect(src *UintMap) error {
	if e := m.open(); e != nil {
		return e
	}
	if e := src.open(); e != nil {
		return e
	}
	if m == src {
		return nil
	}
	copy, e := m.Clone()
	if e != nil {
		return e
	}
	defer copy.Close()
	for _, entry := range copy.table {
		if entry.state == uintMapUsed && !src.Contains(entry.key) {
			if e := m.Remove(entry.key); e != nil {
				return e
			}
		}
	}
	return nil
}
func (m *UintMap) Swap(other *UintMap) error {
	if e := m.open(); e != nil {
		return e
	}
	if e := other.open(); e != nil {
		return e
	}
	if m == other {
		return nil
	}
	if m.config.Budget != other.config.Budget {
		if e := m.config.Budget.Reserve(other.tableBytes); e != nil {
			return e
		}
		if e := other.config.Budget.Reserve(m.tableBytes); e != nil {
			m.config.Budget.Release(other.tableBytes)
			return e
		}
		m.config.Budget.Release(m.tableBytes)
		other.config.Budget.Release(other.tableBytes)
	}
	m.table, other.table = other.table, m.table
	m.tableBytes, other.tableBytes = other.tableBytes, m.tableBytes
	m.size, other.size = other.size, m.size
	m.deleted, other.deleted = other.deleted, m.deleted
	m.bump()
	other.bump()
	return nil
}
func (m *UintMap) CheckInvariant() bool {
	if m.open() != nil {
		return false
	}
	var used, deleted uint32
	for _, entry := range m.table {
		if entry.state == uintMapUsed {
			used++
		} else if entry.state == uintMapDeleted {
			deleted++
		}
	}
	return used == m.size && deleted == m.deleted
}

// UintMapEntryRef is a cold nonowning snapshot. Structural membership changes
// or reallocation invalidate it; value updates keep it live. No array pointer
// escapes. This explicit lifetime is stricter than source raw pointer borrows.
type UintMapEntryRef struct {
	owner *UintMap
	epoch uint64
	slot  uint32
}

func (m *UintMap) ref(i uint32) UintMapEntryRef { return UintMapEntryRef{m, m.epoch, i} }
func (r UintMapEntryRef) validate() error {
	if r.owner == nil {
		return arena.ErrInvalidSlot
	}
	if e := r.owner.open(); e != nil {
		return e
	}
	if r.epoch != r.owner.epoch || r.slot >= r.owner.Capacity() || r.owner.table[r.slot].state != uintMapUsed {
		return arena.ErrInvalidSlot
	}
	return nil
}
func (r UintMapEntryRef) Entry() (UintMapPair, error) {
	if e := r.validate(); e != nil {
		return UintMapPair{}, e
	}
	entry := r.owner.table[r.slot]
	return UintMapPair{entry.key, entry.value}, nil
}
func (r UintMapEntryRef) SetValue(v uint32) error {
	if e := r.validate(); e != nil {
		return e
	}
	r.owner.table[r.slot].value = v
	return nil
}

// UintMapIterator is a cold, nonowning bucket cursor. Mutating membership while
// traversing invalidates the cursor; source value mutation remains supported.
type UintMapIterator struct {
	owner      *UintMap
	epoch      uint64
	index, end uint32
}

func (m *UintMap) Begin() UintMapIterator {
	m.mustOpen()
	it := UintMapIterator{owner: m, epoch: m.epoch, end: m.Capacity()}
	it.scan()
	return it
}
func (m *UintMap) End() UintMapIterator {
	m.mustOpen()
	return UintMapIterator{m, m.epoch, m.Capacity(), m.Capacity()}
}
func (m *UintMap) FindIterator(key uint32) UintMapIterator {
	i, ok := m.findSlot(key)
	if !ok {
		return m.End()
	}
	return UintMapIterator{m, m.epoch, i, m.Capacity()}
}
func (it *UintMapIterator) scan() {
	for it.index < it.end && it.owner.table[it.index].state != uintMapUsed {
		it.index++
	}
}
func (it UintMapIterator) validate() error {
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
func (it UintMapIterator) AtEnd() bool { return it.index == it.end }
func (it UintMapIterator) Equal(other UintMapIterator) bool {
	return it.owner == other.owner && it.epoch == other.epoch && it.index == other.index
}
func (it UintMapIterator) Ref() (UintMapEntryRef, error) {
	if e := it.validate(); e != nil {
		return UintMapEntryRef{}, e
	}
	if it.AtEnd() {
		return UintMapEntryRef{}, arena.ErrBounds
	}
	return it.owner.ref(it.index), nil
}
func (it UintMapIterator) Entry() (UintMapPair, error) {
	r, e := it.Ref()
	if e != nil {
		return UintMapPair{}, e
	}
	return r.Entry()
}
func (it *UintMapIterator) Next() error {
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
func (it *UintMapIterator) PostNext() (UintMapIterator, error) { old := *it; return old, it.Next() }
