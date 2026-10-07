// Concrete map<uint_set, unsigned, uint_set::hash, uint_set::eq> translation
// from src/util/map.h, hashtable.h and uint_set.h; Leonardo de Moura.
package util

import (
	"reflect"
	"strconv"
	"strings"

	"github.com/glycerine/g3/pkg/arena"
)

const (
	setMapFree uint32 = iota
	setMapDeleted
	setMapUsed
)
const setMapEntryBytes uint64 = 28

// UintSetMapConfig provides cold owner-local resources. Words.Descriptors.Budget
// is set to Budget, so the table and its complete key pool share one budget.
type UintSetMapConfig struct {
	Budget      *arena.Budget
	OutOfMemory func() bool
	Words       arena.VectorConfig
}

// setMapKey keeps source logical size and vector capacity distinct. The arena
// payload length is capacity, never the source logical length; class rounding
// is a third, physical capacity. No hot key contains a Go pointer.
type setMapKey struct {
	ref              arena.VectorRef
	length, capacity uint32
}
type setMapEntry struct {
	hash, state uint32
	key         setMapKey
	value       uint32
}

// UintSetMap owns its table and all USED, DELETED and retained FREE key payloads.
// Construct with NewUintSetMap; do not copy this owning struct. Lookups can fail
// because source table2map creates a copy-constructed key temporary.
type UintSetMap struct {
	table                 []setMapEntry
	words                 *arena.Words
	config                UintSetMapConfig
	size, deleted         uint32
	epoch                 uint64
	rootBytes, tableBytes uint64
	closed                bool
}

func NewUintSetMap(c UintSetMapConfig) (*UintSetMap, error) { return newUintSetMapCapacity(c, 8) }
func newUintSetMapCapacity(c UintSetMapConfig, n uint32) (*UintSetMap, error) {
	root := uint64(reflect.TypeOf(UintSetMap{}).Size())
	if e := c.Budget.Reserve(root); e != nil {
		return nil, e
	}
	c.Words.Descriptors.Budget = c.Budget
	pool, e := arena.NewWords(c.Words)
	if e != nil {
		c.Budget.Release(root)
		return nil, e
	}
	m := &UintSetMap{words: pool, config: c, epoch: 1, rootBytes: root}
	table, amount, e := m.allocateTable(n)
	if e != nil {
		pool.Close()
		c.Budget.Release(root)
		return nil, e
	}
	m.table, m.tableBytes = table, amount
	return m, nil
}
func (m *UintSetMap) allocateTable(n uint32) ([]setMapEntry, uint64, error) {
	if n < 8 || n&(n-1) != 0 || n > 1<<30 || uint64(n) > uint64(^uint(0)>>1)/setMapEntryBytes {
		return nil, 0, arena.ErrBounds
	}
	amount := uint64(n) * setMapEntryBytes
	if e := m.config.Budget.Reserve(amount); e != nil {
		return nil, 0, e
	}
	return make([]setMapEntry, n), amount, nil
}
func (m *UintSetMap) open() error {
	if m.closed || len(m.table) == 0 {
		return arena.ErrClosed
	}
	return nil
}
func (m *UintSetMap) mustOpen() {
	if e := m.open(); e != nil {
		panic(e)
	}
}
func (m *UintSetMap) bump() {
	if m.epoch == ^uint64(0) {
		panic("bitset: set map epoch exhausted")
	}
	m.epoch++
}
func (m *UintSetMap) Size() uint32                 { return m.size }
func (m *UintSetMap) Empty() bool                  { return m.size == 0 }
func (m *UintSetMap) Capacity() uint32             { return uint32(len(m.table)) }
func (m *UintSetMap) Deleted() uint32              { return m.deleted }
func (m *UintSetMap) CollisionCount() uint64       { return 0 }
func (m *UintSetMap) WordStats() arena.VectorStats { return m.words.Stats() }
func setMapCopyCapacity(n uint32) (uint32, error) {
	if uint64(n)*32 > uint64(^uint32(0)) {
		return 0, arena.ErrBounds
	}
	var c uint32
	for c < n {
		if c == 0 {
			c = 2
		} else {
			next := (uint64(c)*3 + 1) >> 1
			if next > uint64(^uint32(0)) {
				return 0, arena.ErrBounds
			}
			c = uint32(next)
		}
	}
	return c, nil
}
func (m *UintSetMap) allocateKey(n, c uint32) (setMapKey, error) {
	if n > c || uint64(n)*32 > uint64(^uint32(0)) {
		return setMapKey{}, arena.ErrBounds
	}
	r, e := m.words.Allocate(c, c)
	if e != nil {
		return setMapKey{}, e
	}
	return setMapKey{r, n, c}, nil
}

// copySet is uint_set's custom copy constructor: push_back active words, so
// spare source capacity is discarded and geometric capacity is reconstructed.
func (m *UintSetMap) copySet(s *UintSet) (setMapKey, error) {
	c, e := setMapCopyCapacity(uint32(len(s.words)))
	if e != nil {
		return setMapKey{}, e
	}
	k, e := m.allocateKey(uint32(len(s.words)), c)
	if e != nil {
		return setMapKey{}, e
	}
	for i, w := range s.words {
		if e := m.words.Write(k.ref, uint32(i), w); e != nil {
			m.freeKey(k)
			return setMapKey{}, e
		}
	}
	return k, nil
}
func setMapWord(pool *arena.Words, k setMapKey, i uint32) uint32 {
	if i >= k.length {
		return 0
	}
	w, e := pool.Read(k.ref, i)
	if e != nil {
		panic(e)
	}
	return w
}

// copyKey uses constructor capacity for temporaries, assignment capacity for
// stored entries. std::move resolves to uint_set COPY assignment in this source.
func (m *UintSetMap) copyKey(pool *arena.Words, k setMapKey, constructor bool) (setMapKey, error) {
	c := k.capacity
	if constructor {
		var e error
		c, e = setMapCopyCapacity(k.length)
		if e != nil {
			return setMapKey{}, e
		}
	}
	r, e := m.allocateKey(k.length, c)
	if e != nil {
		return setMapKey{}, e
	}
	for i := uint32(0); i < k.length; i++ {
		if e := m.words.Write(r.ref, i, setMapWord(pool, k, i)); e != nil {
			m.freeKey(r)
			return setMapKey{}, e
		}
	}
	return r, nil
}
func (m *UintSetMap) freeKey(k setMapKey) {
	if e := m.words.Free(k.ref); e != nil {
		panic(e)
	}
}
func setMapHash(pool *arena.Words, k setMapKey) uint32 {
	var h uint32
	for i := uint32(0); i < k.length; i++ {
		h += (i + 1) * setMapWord(pool, k, i)
	}
	return h
}
func setMapEqual(a *arena.Words, x setMapKey, b *arena.Words, y setMapKey) bool {
	n := x.length
	if y.length > n {
		n = y.length
	}
	for i := uint32(0); i < n; i++ {
		if setMapWord(a, x, i) != setMapWord(b, y, i) {
			return false
		}
	}
	return true
}
func (m *UintSetMap) destroyTable(table []setMapEntry) {
	for _, e := range table {
		m.freeKey(e.key)
	}
}
func (m *UintSetMap) copyTable(sourcePool *arena.Words, source, target []setMapEntry) error {
	mask := uint32(len(target)) - 1
	for _, entry := range source {
		if entry.state != setMapUsed {
			continue
		}
		i := entry.hash & mask
		for target[i].state != setMapFree {
			i = (i + 1) & mask
		}
		key, e := m.copyKey(sourcePool, entry.key, false)
		if e != nil {
			return e
		}
		target[i] = setMapEntry{entry.hash, setMapUsed, key, entry.value}
	}
	return nil
}
func (m *UintSetMap) rehash(n uint32) error {
	table, amount, e := m.allocateTable(n)
	if e != nil {
		return e
	}
	if e := m.copyTable(m.words, m.table, table); e != nil {
		m.destroyTable(table)
		m.config.Budget.Release(amount)
		return e
	}
	// Every old slot is destructed only after all USED copies are constructed.
	m.destroyTable(m.table)
	m.config.Budget.Release(m.tableBytes)
	m.table, m.tableBytes = table, amount
	m.deleted = 0
	m.bump()
	return nil
}
func (m *UintSetMap) prepareInsert() error {
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
func (m *UintSetMap) insertCore(pool *arena.Words, key setMapKey, value uint32, overwrite bool) (bool, UintSetMapEntryRef, error) {
	if e := m.prepareInsert(); e != nil {
		return false, UintSetMapEntryRef{}, e
	}
	h := setMapHash(pool, key)
	mask := m.Capacity() - 1
	i := h & mask
	deleted := ^uint32(0)
	for step := uint32(0); step < m.Capacity(); step++ {
		entry := &m.table[i]
		switch entry.state {
		case setMapUsed:
			if entry.hash == h && setMapEqual(m.words, entry.key, pool, key) {
				if overwrite {
					replacement, e := m.copyKey(pool, key, false)
					if e != nil {
						return false, UintSetMapEntryRef{}, e
					}
					m.freeKey(entry.key)
					entry.key, entry.value = replacement, value
				}
				return false, m.ref(i), nil
			}
		case setMapFree:
			slot := i
			if deleted != ^uint32(0) {
				slot = deleted
			}
			replacement, e := m.copyKey(pool, key, false)
			if e != nil {
				return false, UintSetMapEntryRef{}, e
			}
			m.freeKey(m.table[slot].key)
			if deleted != ^uint32(0) {
				m.deleted--
			}
			m.table[slot] = setMapEntry{h, setMapUsed, replacement, value}
			m.size++
			m.bump()
			return true, m.ref(slot), nil
		case setMapDeleted:
			deleted = i
		}
		i = (i + 1) & mask
	}
	panic("bitset: source set map has no free insertion slot")
}
func (m *UintSetMap) insertWrapper(s *UintSet, value uint32, overwrite, twoCopies bool) (bool, UintSetMapEntryRef, error) {
	if e := m.open(); e != nil {
		return false, UintSetMapEntryRef{}, e
	}
	key, e := m.copySet(s)
	if e != nil {
		return false, UintSetMapEntryRef{}, e
	}
	defer m.freeKey(key)
	if twoCopies {
		second, e := m.copyKey(m.words, key, true)
		if e != nil {
			return false, UintSetMapEntryRef{}, e
		}
		defer m.freeKey(second)
		return m.insertCore(m.words, second, value, overwrite)
	}
	return m.insertCore(m.words, key, value, overwrite)
}
func (m *UintSetMap) Insert(k *UintSet, v uint32) error {
	_, _, e := m.insertWrapper(k, v, true, false)
	return e
}
func (m *UintSetMap) InsertIfNotThereCore(k *UintSet, v uint32) (bool, UintSetMapEntryRef, error) {
	return m.insertWrapper(k, v, false, false)
}
func (m *UintSetMap) InsertIfNotThere(k *UintSet, v uint32) (UintSetMapEntryRef, error) {
	_, r, e := m.insertWrapper(k, v, false, true)
	return r, e
}
func (m *UintSetMap) InsertIfNotThere2(k *UintSet, v uint32) (UintSetMapEntryRef, error) {
	return m.InsertIfNotThere(k, v)
}
func (m *UintSetMap) findSlot(pool *arena.Words, key setMapKey) (uint32, bool) {
	h := setMapHash(pool, key)
	mask := m.Capacity() - 1
	i := h & mask
	for step := uint32(0); step < m.Capacity(); step++ {
		entry := m.table[i]
		if entry.state == setMapUsed && entry.hash == h && setMapEqual(m.words, entry.key, pool, key) {
			return i, true
		}
		if entry.state == setMapFree {
			return 0, false
		}
		i = (i + 1) & mask
	}
	return 0, false
}
func (m *UintSetMap) FindCore(s *UintSet) (UintSetMapEntryRef, bool, error) {
	if e := m.open(); e != nil {
		return UintSetMapEntryRef{}, false, e
	}
	key, e := m.copySet(s)
	if e != nil {
		return UintSetMapEntryRef{}, false, e
	}
	defer m.freeKey(key)
	i, ok := m.findSlot(m.words, key)
	if !ok {
		return UintSetMapEntryRef{}, false, nil
	}
	return m.ref(i), true, nil
}
func (m *UintSetMap) Contains(s *UintSet) (bool, error) { _, ok, e := m.FindCore(s); return ok, e }
func (m *UintSetMap) Find(s *UintSet) (uint32, bool, error) {
	r, ok, e := m.FindCore(s)
	if e != nil || !ok {
		return 0, ok, e
	}
	return m.table[r.slot].value, true, nil
}
func (m *UintSetMap) FindInto(s *UintSet, out *uint32) (bool, error) {
	v, ok, e := m.Find(s)
	if e == nil && ok {
		*out = v
	}
	return ok, e
}
func (m *UintSetMap) Get(s *UintSet, defaultValue uint32) (uint32, error) {
	v, ok, e := m.Find(s)
	if e != nil {
		return 0, e
	}
	if ok {
		return v, nil
	}
	return defaultValue, nil
}
func (m *UintSetMap) MustFind(s *UintSet) (uint32, error) {
	v, ok, e := m.Find(s)
	if e != nil {
		return 0, e
	}
	if !ok {
		return 0, arena.ErrInvalidSlot
	}
	return v, nil
}
func (m *UintSetMap) removeCore(pool *arena.Words, key setMapKey) error {
	i, ok := m.findSlot(pool, key)
	if !ok {
		return nil
	}
	next := (i + 1) & (m.Capacity() - 1)
	m.size--
	m.bump()
	if m.table[next].state == setMapFree {
		m.table[i].state = setMapFree
		return nil
	}
	m.table[i].state = setMapDeleted
	m.deleted++
	if m.deleted > m.size && m.deleted > 64 && (m.config.OutOfMemory == nil || !m.config.OutOfMemory()) {
		return m.rehash(m.Capacity())
	}
	return nil
}
func (m *UintSetMap) Remove(s *UintSet) error {
	if e := m.open(); e != nil {
		return e
	}
	key, e := m.copySet(s)
	if e != nil {
		return e
	}
	defer m.freeKey(key)
	return m.removeCore(m.words, key)
}
func (m *UintSetMap) Erase(s *UintSet) error { return m.Remove(s) }
func (m *UintSetMap) replaceEmpty(n uint32) error {
	table, amount, e := m.allocateTable(n)
	if e != nil {
		return e
	}
	m.destroyTable(m.table)
	m.config.Budget.Release(m.tableBytes)
	m.table, m.tableBytes = table, amount
	m.size, m.deleted = 0, 0
	m.bump()
	return nil
}
func (m *UintSetMap) Reset() error {
	if e := m.open(); e != nil {
		return e
	}
	if m.size == 0 && m.deleted == 0 {
		return nil
	}
	var overhead uint32
	for _, entry := range m.table {
		if entry.state == setMapFree {
			overhead++
		}
	}
	if m.Capacity() > 16 && uint64(overhead)*4 > uint64(m.Capacity())*3 {
		return m.replaceEmpty(m.Capacity() >> 1)
	}
	for i := range m.table {
		m.table[i].state = setMapFree
	}
	m.size, m.deleted = 0, 0
	m.bump()
	return nil
}
func (m *UintSetMap) Finalize() error {
	if e := m.open(); e != nil {
		return e
	}
	if m.Capacity() > 64 {
		return m.replaceEmpty(64)
	}
	return m.Reset()
}
func (m *UintSetMap) Close() {
	if m.closed {
		return
	}
	m.destroyTable(m.table)
	m.words.Close()
	m.config.Budget.Release(m.tableBytes + m.rootBytes)
	m.table = nil
	m.rootBytes, m.tableBytes = 0, 0
	m.size, m.deleted = 0, 0
	m.closed = true
	m.bump()
}
func (m *UintSetMap) Clone() (*UintSetMap, error) {
	if e := m.open(); e != nil {
		return nil, e
	}
	r, e := newUintSetMapCapacity(m.config, m.Capacity())
	if e != nil {
		return nil, e
	}
	if e := r.copyTable(m.words, m.table, r.table); e != nil {
		r.Close()
		return nil, e
	}
	r.size = m.size
	return r, nil
}
func (m *UintSetMap) insertConst(pool *arena.Words, key setMapKey, value uint32) error {
	temp, e := m.copyKey(pool, key, true)
	if e != nil {
		return e
	}
	defer m.freeKey(temp)
	_, _, e = m.insertCore(m.words, temp, value, true)
	return e
}
func (m *UintSetMap) Assign(src *UintSetMap) error {
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
	return m.Union(src)
}
func (m *UintSetMap) Union(src *UintSetMap) error {
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
		if entry.state == setMapUsed {
			if e := m.insertConst(src.words, entry.key, entry.value); e != nil {
				return e
			}
		}
	}
	return nil
}
func (m *UintSetMap) Intersect(src *UintSetMap) error {
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
		if entry.state == setMapUsed {
			if _, ok := src.findSlot(copy.words, entry.key); !ok {
				if e := m.removeCore(copy.words, entry.key); e != nil {
					return e
				}
			}
		}
	}
	return nil
}

// Swap exchanges key ownership without copying payloads. Cross-budget support
// is supplied by the arena's atomic graph-budget rebind hook.
func (m *UintSetMap) Swap(other *UintSetMap) error {
	if e := m.open(); e != nil {
		return e
	}
	if e := other.open(); e != nil {
		return e
	}
	if m == other {
		return nil
	}
	// After graph rebinding there must be no fallible operation, including an
	// exhausted borrow epoch. Check both owners before preparing reservations.
	if m.epoch == ^uint64(0) || other.epoch == ^uint64(0) {
		return arena.ErrBounds
	}
	if m.config.Budget != other.config.Budget {
		if e := m.config.Budget.Reserve(other.tableBytes); e != nil {
			return e
		}
		if e := other.config.Budget.Reserve(m.tableBytes); e != nil {
			m.config.Budget.Release(other.tableBytes)
			return e
		}
		if e := m.words.SwapBudgets(other.words); e != nil {
			m.config.Budget.Release(other.tableBytes)
			other.config.Budget.Release(m.tableBytes)
			return e
		}
		m.config.Budget.Release(m.tableBytes)
		other.config.Budget.Release(other.tableBytes)
	}
	m.table, other.table = other.table, m.table
	m.words, other.words = other.words, m.words
	m.tableBytes, other.tableBytes = other.tableBytes, m.tableBytes
	m.size, other.size = other.size, m.size
	m.deleted, other.deleted = other.deleted, m.deleted
	m.bump()
	other.bump()
	return nil
}
func (m *UintSetMap) CheckInvariant() bool {
	if m.open() != nil {
		return false
	}
	var used, deleted uint32
	for _, entry := range m.table {
		if entry.state == setMapUsed {
			used++
			if setMapHash(m.words, entry.key) != entry.hash {
				return false
			}
		} else if entry.state == setMapDeleted {
			deleted++
		}
		if entry.key.length > entry.key.capacity {
			return false
		}
		if entry.key.capacity != 0 && !m.words.IsLive(entry.key.ref) {
			return false
		}
	}
	return used == m.size && deleted == m.deleted
}

// UintSetMapEntryRef is a cold borrow invalidated by structural membership or
// table changes. Complete-key overwrite preserves the slot and updates its view.
type UintSetMapEntryRef struct {
	owner *UintSetMap
	epoch uint64
	slot  uint32
}

func (m *UintSetMap) ref(i uint32) UintSetMapEntryRef { return UintSetMapEntryRef{m, m.epoch, i} }
func (r UintSetMapEntryRef) validate() error {
	if r.owner == nil {
		return arena.ErrInvalidSlot
	}
	if e := r.owner.open(); e != nil {
		return e
	}
	if r.epoch != r.owner.epoch || r.slot >= r.owner.Capacity() || r.owner.table[r.slot].state != setMapUsed {
		return arena.ErrInvalidSlot
	}
	return nil
}
func (r UintSetMapEntryRef) Value() (uint32, error) {
	if e := r.validate(); e != nil {
		return 0, e
	}
	return r.owner.table[r.slot].value, nil
}
func (r UintSetMapEntryRef) SetValue(v uint32) error {
	if e := r.validate(); e != nil {
		return e
	}
	r.owner.table[r.slot].value = v
	return nil
}
func (r UintSetMapEntryRef) Key() (UintSetMapKeyView, error) {
	if e := r.validate(); e != nil {
		return UintSetMapKeyView{}, e
	}
	return UintSetMapKeyView{r}, nil
}

// UintSetMapKeyView resolves its slot on every access and never exposes a mutable
// key or arena borrow. CopyKey explicitly transfers a new caller-owned copy.
type UintSetMapKeyView struct{ entry UintSetMapEntryRef }

func (v UintSetMapKeyView) key() (setMapKey, error) {
	if e := v.entry.validate(); e != nil {
		return setMapKey{}, e
	}
	return v.entry.owner.table[v.entry.slot].key, nil
}
func (v UintSetMapKeyView) NumWords() (uint32, error)      { k, e := v.key(); return k.length, e }
func (v UintSetMapKeyView) CapacityWords() (uint32, error) { k, e := v.key(); return k.capacity, e }
func (v UintSetMapKeyView) MaxElem() (uint32, error)       { k, e := v.key(); return k.length * 32, e }
func (v UintSetMapKeyView) Contains(n uint32) (bool, error) {
	k, e := v.key()
	if e != nil {
		return false, e
	}
	return setMapWord(v.entry.owner.words, k, n>>5)&(1<<(n&31)) != 0, nil
}
func (v UintSetMapKeyView) Hash() (uint32, error) {
	k, e := v.key()
	if e != nil {
		return 0, e
	}
	return setMapHash(v.entry.owner.words, k), nil
}
func (v UintSetMapKeyView) CopyKey() (*UintSet, error) {
	k, e := v.key()
	if e != nil {
		return nil, e
	}
	r := new(UintSet)
	for i := uint32(0); i < k.length; i++ {
		r.pushWord(setMapWord(v.entry.owner.words, k, i))
	}
	return r, nil
}
func (v UintSetMapKeyView) String() (string, error) {
	k, e := v.key()
	if e != nil {
		return "", e
	}
	var b strings.Builder
	b.WriteByte('{')
	first := true
	for i := uint32(0); i < k.length*32; i++ {
		if setMapWord(v.entry.owner.words, k, i>>5)&(1<<(i&31)) != 0 {
			if !first {
				b.WriteString(", ")
			}
			b.WriteString(strconv.FormatUint(uint64(i), 10))
			first = false
		}
	}
	b.WriteByte('}')
	return b.String(), nil
}

type UintSetMapIterator struct {
	owner      *UintSetMap
	epoch      uint64
	index, end uint32
}

func (m *UintSetMap) Begin() UintSetMapIterator {
	m.mustOpen()
	it := UintSetMapIterator{owner: m, epoch: m.epoch, end: m.Capacity()}
	it.scan()
	return it
}
func (m *UintSetMap) End() UintSetMapIterator {
	m.mustOpen()
	return UintSetMapIterator{m, m.epoch, m.Capacity(), m.Capacity()}
}
func (m *UintSetMap) FindIterator(s *UintSet) (UintSetMapIterator, error) {
	r, ok, e := m.FindCore(s)
	if e != nil {
		return UintSetMapIterator{}, e
	}
	if !ok {
		return m.End(), nil
	}
	return UintSetMapIterator{m, m.epoch, r.slot, m.Capacity()}, nil
}
func (it *UintSetMapIterator) scan() {
	for it.index < it.end && it.owner.table[it.index].state != setMapUsed {
		it.index++
	}
}
func (it UintSetMapIterator) validate() error {
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
func (it UintSetMapIterator) AtEnd() bool { return it.index == it.end }
func (it UintSetMapIterator) Equal(other UintSetMapIterator) bool {
	return it.owner == other.owner && it.epoch == other.epoch && it.index == other.index
}
func (it UintSetMapIterator) Ref() (UintSetMapEntryRef, error) {
	if e := it.validate(); e != nil {
		return UintSetMapEntryRef{}, e
	}
	if it.AtEnd() {
		return UintSetMapEntryRef{}, arena.ErrBounds
	}
	return it.owner.ref(it.index), nil
}
func (it *UintSetMapIterator) Next() error {
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

func (it *UintSetMapIterator) PostNext() (UintSetMapIterator, error) {
	old := *it
	if e := it.Next(); e != nil {
		return UintSetMapIterator{}, e
	}
	return old, nil
}
