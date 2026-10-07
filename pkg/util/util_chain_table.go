// Concrete int32 specializations of src/util/chashtable.h.
package util

import (
	"errors"
	"math"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
)

var ErrChainView = errors.New("g3: invalidated chaining table view")
var ErrChainConfig = errors.New("g3: chaining table source precondition")

// Hash is borrowed, must not panic/reenter, and must remain stable while entries
// are present. Nil implements source int_hash. Owners require caller serialization.
type Int32ChainConfig struct {
	Budget                      *arena.Budget
	Hash                        func(int32) uint32
	InitialSlots, InitialCellar uint32
}
type Int32ChainPair struct{ Key, Value int32 }
type Int32ChainStats struct {
	Size, Slots, Capacity, UsedSlots, Collisions uint32
	OwnerBytes, TableBytes                       uint64
}

// Link indices replace source pointers; cell data remain retained when FREE or
// recycled. Source int tables use the key only, cmap uses both scalar fields.
type chainCell struct {
	key, value int32
	next       uint32
}

const ChainFree = math.MaxUint32
const ChainEnd = math.MaxUint32 - 1

type chainStorage struct {
	table                                                            []chainCell
	slots, initSlots, initCellar, used, size, collisions, next, free uint32
	budget                                                           *arena.Budget
	hash                                                             func(int32) uint32
	ownerBytes, epoch                                                uint64
	closed                                                           bool
}
type Int32ChainTable struct{ core chainStorage }
type Int32ChainMap struct{ core chainStorage }

func chainBytes(n uint32) uint64 { return uint64(n) * uint64(reflect.TypeOf(chainCell{}).Size()) }
func chainAllocate(b *arena.Budget, slots, cellar uint32) ([]chainCell, error) {
	n := uint64(slots) + uint64(cellar)
	if slots == 0 || slots&(slots-1) != 0 || cellar == 0 {
		return nil, ErrChainConfig
	}
	// Source unsigned capacity arithmetic and Go index/sentinel bounds.
	if n >= uint64(ChainEnd) || n > uint64(^uint(0)>>1)/uint64(reflect.TypeOf(chainCell{}).Size()) {
		return nil, arena.ErrBounds
	}
	if e := b.Reserve(chainBytes(uint32(n))); e != nil {
		return nil, e
	}
	a := make([]chainCell, int(n))
	for i := range a {
		a[i].next = ChainFree
	}
	return a, nil
}
func chainInit(c Int32ChainConfig, owner uint64) (chainStorage, error) {
	if c.InitialSlots == 0 {
		c.InitialSlots = 8
	}
	if c.InitialCellar == 0 {
		c.InitialCellar = 2
	}
	if e := c.Budget.Reserve(owner); e != nil {
		return chainStorage{}, e
	}
	a, e := chainAllocate(c.Budget, c.InitialSlots, c.InitialCellar)
	if e != nil {
		c.Budget.Release(owner)
		return chainStorage{}, e
	}
	return chainStorage{table: a, slots: c.InitialSlots, initSlots: c.InitialSlots, initCellar: c.InitialCellar, next: c.InitialSlots, free: ChainEnd, budget: c.Budget, hash: c.Hash, ownerBytes: owner, epoch: 1}, nil
}
func NewInt32ChainTable(c Int32ChainConfig) (*Int32ChainTable, error) {
	s, e := chainInit(c, uint64(reflect.TypeOf(Int32ChainTable{}).Size()))
	if e != nil {
		return nil, e
	}
	return &Int32ChainTable{core: s}, nil
}
func NewInt32ChainMap(c Int32ChainConfig) (*Int32ChainMap, error) {
	s, e := chainInit(c, uint64(reflect.TypeOf(Int32ChainMap{}).Size()))
	if e != nil {
		return nil, e
	}
	return &Int32ChainMap{core: s}, nil
}
func (s *chainStorage) open() error {
	if s == nil || s.closed {
		return arena.ErrClosed
	}
	return nil
}
func (s *chainStorage) canBump() error {
	if s.epoch == math.MaxUint64 {
		return arena.ErrBounds
	}
	return nil
}
func (s *chainStorage) h(v int32) uint32 {
	if s.hash != nil {
		return s.hash(v)
	}
	return uint32(v)
}
func (s *chainStorage) stats() Int32ChainStats {
	return Int32ChainStats{s.size, s.slots, uint32(len(s.table)), s.used, s.collisions, s.ownerBytes, chainBytes(uint32(len(s.table)))}
}
func (s *chainStorage) hasFree() bool { return s.free != ChainEnd || s.next < uint32(len(s.table)) }
func (s *chainStorage) getFree() uint32 {
	if s.free != ChainEnd {
		i := s.free
		s.free = s.table[i].next
		return i
	}
	i := s.next
	s.next++
	return i
}
func (s *chainStorage) recycle(i uint32) { s.table[i].next = s.free; s.free = i }

// Target construction walks source buckets ascending, chains head first, and
// displaces target heads into sequential cellar cells, exactly reversing collisions.
func (s *chainStorage) copyTable(a []chainCell, slots uint32) (next, used uint32, ok bool) {
	// Source passes m_used_slots by reference: every attempted copy publishes
	// its prefix count even when the cellar overflows and a later allocation fails.
	s.used = 0
	next = slots
	for b := uint32(0); b < s.slots; b++ {
		if s.table[b].next == ChainFree {
			continue
		}
		for i := b; i != ChainEnd; i = s.table[i].next {
			idx := s.h(s.table[i].key) & (slots - 1)
			if a[idx].next == ChainFree {
				a[idx] = s.table[i]
				a[idx].next = ChainEnd
				used++
				s.used = used
			} else {
				if next == uint32(len(a)) {
					return next, used, false
				}
				a[next] = a[idx]
				a[idx] = s.table[i]
				a[idx].next = next
				next++
			}
		}
	}
	return next, used, true
}
func (s *chainStorage) expand() error {
	slots64 := uint64(s.slots) * 2
	cellar64 := uint64(uint32(len(s.table))-s.slots) * 2
	for {
		if slots64 >= uint64(ChainEnd) || cellar64 >= uint64(ChainEnd) {
			return arena.ErrBounds
		}
		a, e := chainAllocate(s.budget, uint32(slots64), uint32(cellar64))
		if e != nil {
			return e
		}
		next, used, ok := s.copyTable(a, uint32(slots64))
		if ok {
			s.budget.Release(chainBytes(uint32(len(s.table))))
			s.table = a
			s.slots = uint32(slots64)
			s.next = next
			s.free = ChainEnd
			s.used = used
			return nil
		}
		s.budget.Release(chainBytes(uint32(len(a))))
		cellar64 *= 2
	}
}
func (s *chainStorage) find(key int32, count bool) (uint32, bool) {
	i := s.h(key) & (s.slots - 1)
	return s.findAt(key, i, count)
}
func (s *chainStorage) findAt(key int32, i uint32, count bool) (uint32, bool) {
	if s.table[i].next == ChainFree {
		return ChainEnd, false
	}
	for ; i != ChainEnd; i = s.table[i].next {
		if s.table[i].key == key {
			return i, true
		}
		if count {
			s.collisions++
		}
	}
	return ChainEnd, false
}
func (s *chainStorage) insert(key, value int32, overwrite bool) (uint32, bool, error) {
	if e := s.open(); e != nil {
		return ChainEnd, false, e
	}
	grow := !s.hasFree()
	if grow {
		if e := s.canBump(); e != nil {
			return ChainEnd, false, e
		}
	}
	if grow {
		if e := s.expand(); e != nil {
			return ChainEnd, false, e
		}
		s.epoch++
	}
	idx := s.h(key) & (s.slots - 1)
	if !grow {
		if _, exists := s.findAt(key, idx, false); !exists {
			if e := s.canBump(); e != nil {
				return ChainEnd, false, e
			}
		}
	}
	i, found := s.findAt(key, idx, true)
	if found {
		if overwrite {
			s.table[i].key = key
			s.table[i].value = value
		}
		return i, false, nil
	}
	if !grow {
		s.epoch++
	}
	i = idx
	if s.table[i].next == ChainFree {
		s.used++
		s.table[i] = chainCell{key, value, ChainEnd}
	} else {
		j := s.getFree()
		s.table[j] = s.table[i]
		s.table[i] = chainCell{key, value, j}
	}
	s.size++
	return i, true, nil
}
func (s *chainStorage) erase(key int32) error {
	if e := s.open(); e != nil {
		return e
	}
	i := s.h(key) & (s.slots - 1)
	_, exists := s.findAt(key, i, false)
	if exists {
		if e := s.canBump(); e != nil {
			return e
		}
	}
	if s.table[i].next == ChainFree {
		return nil
	}
	prev := uint32(ChainEnd)
	for i != ChainEnd {
		c := s.table[i]
		if c.key == key {
			s.size--
			s.epoch++
			if prev == ChainEnd {
				if c.next == ChainEnd {
					s.used--
					s.table[i].next = ChainFree
				} else {
					s.table[i] = s.table[c.next]
					s.recycle(c.next)
				}
			} else {
				s.table[prev].next = c.next
				s.recycle(i)
			}
			return nil
		}
		s.collisions++
		prev = i
		i = c.next
	}
	return nil
}
func (s *chainStorage) finalize(reset bool) error {
	if e := s.open(); e != nil {
		return e
	}
	if reset && s.size == 0 {
		return nil
	}
	if e := s.canBump(); e != nil {
		return e
	}
	// Allocate before publication: unlike source delete-before-init, predictable
	// Go allocation failure leaves the current graph intact. Charge both arrays.
	a, e := chainAllocate(s.budget, s.initSlots, s.initCellar)
	if e != nil {
		return e
	}
	s.budget.Release(chainBytes(uint32(len(s.table))))
	s.table = a
	s.slots = s.initSlots
	s.used = 0
	s.size = 0
	s.next = s.slots
	s.free = ChainEnd
	s.epoch++
	return nil
}
func (s *chainStorage) close() {
	if s == nil || s.closed {
		return
	}
	s.budget.Release(chainBytes(uint32(len(s.table))) + s.ownerBytes)
	s.table = nil
	s.size = 0
	s.used = 0
	s.closed = true
	if s.epoch != math.MaxUint64 {
		s.epoch++
	}
}
func (s *chainStorage) swap(t *chainStorage) error {
	if e := s.open(); e != nil {
		return e
	}
	if e := t.open(); e != nil {
		return e
	}
	if s == t {
		return nil
	}
	if e := s.canBump(); e != nil {
		return e
	}
	if e := t.canBump(); e != nil {
		return e
	}
	sb, tb := chainBytes(uint32(len(s.table))), chainBytes(uint32(len(t.table)))
	// Go ownership transfer reserves only positive net bytes, not an allocation.
	if s.budget != t.budget {
		if tb > sb {
			if e := s.budget.Reserve(tb - sb); e != nil {
				return e
			}
			t.budget.Release(tb - sb)
		} else if sb > tb {
			if e := t.budget.Reserve(sb - tb); e != nil {
				return e
			}
			s.budget.Release(sb - tb)
		}
	}
	s.table, t.table = t.table, s.table
	s.slots, t.slots = t.slots, s.slots
	s.initSlots, t.initSlots = t.initSlots, s.initSlots
	s.initCellar, t.initCellar = t.initCellar, s.initCellar
	s.used, t.used = t.used, s.used
	s.size, t.size = t.size, s.size
	s.collisions, t.collisions = t.collisions, s.collisions
	s.next, t.next = t.next, s.next
	s.free, t.free = t.free, s.free
	s.epoch++
	t.epoch++
	return nil
}

// Explicit portable invariant audit. Automatic source Z3DEBUG diagnostic call
// sites and its temporary pointer-address hash set are not instantiated here.
func (s *chainStorage) invariant() bool {
	if s.closed {
		return false
	}
	var size, used uint32
	for b := uint32(0); b < s.slots; b++ {
		if s.table[b].next == ChainFree {
			continue
		}
		used++
		var length uint32
		for i := b; i != ChainEnd; i = s.table[i].next {
			if i >= uint32(len(s.table)) || s.table[i].next == ChainFree || length > uint32(len(s.table)) {
				return false
			}
			if i != b && i < s.slots {
				return false
			}
			size++
			length++
		}
	}
	return size == s.size && used == s.used
}
func (h *Int32ChainTable) Stats() Int32ChainStats { return h.core.stats() }
func (h *Int32ChainTable) Empty() bool            { return h.core.size == 0 }
func (h *Int32ChainTable) Insert(v int32) error   { _, _, e := h.core.insert(v, 0, true); return e }
func (h *Int32ChainTable) InsertIfNotThere(v int32) (Int32ChainEntryRef, error) {
	i, _, e := h.core.insert(v, 0, false)
	return Int32ChainEntryRef{chainRef{&h.core, h.core.epoch, i}}, e
}
func (h *Int32ChainTable) InsertIfNotThere2(v int32) (bool, error) {
	_, added, e := h.core.insert(v, 0, false)
	return added, e
}
func (h *Int32ChainTable) Contains(v int32) (bool, error) {
	if e := h.core.open(); e != nil {
		return false, e
	}
	_, ok := h.core.find(v, true)
	return ok, nil
}
func (h *Int32ChainTable) Find(v int32) (int32, bool, error) {
	if e := h.core.open(); e != nil {
		return 0, false, e
	}
	i, ok := h.core.find(v, true)
	if !ok {
		return 0, false, nil
	}
	return h.core.table[i].key, true, nil
}
func (h *Int32ChainTable) FindInto(v int32, out *int32) (bool, error) {
	r, ok, e := h.Find(v)
	if ok && out != nil {
		*out = r
	}
	return ok, e
}
func (h *Int32ChainTable) FindCore(v int32) (Int32ChainEntryRef, bool, error) {
	if e := h.core.open(); e != nil {
		return Int32ChainEntryRef{}, false, e
	}
	i, ok := h.core.find(v, true)
	return Int32ChainEntryRef{chainRef{&h.core, h.core.epoch, i}}, ok, nil
}
func (h *Int32ChainTable) Erase(v int32) error { return h.core.erase(v) }
func (h *Int32ChainTable) Reset() error        { return h.core.finalize(true) }
func (h *Int32ChainTable) Finalize() error     { return h.core.finalize(false) }
func (h *Int32ChainTable) Swap(other *Int32ChainTable) error {
	if other == nil {
		return ErrChainConfig
	}
	return h.core.swap(&other.core)
}
func (h *Int32ChainTable) Close() {
	if h != nil {
		h.core.close()
	}
}
func (h *Int32ChainTable) CheckInvariant() bool  { return h.core.invariant() }
func (h *Int32ChainMap) Stats() Int32ChainStats  { return h.core.stats() }
func (h *Int32ChainMap) Empty() bool             { return h.core.size == 0 }
func (h *Int32ChainMap) Insert(k, v int32) error { _, _, e := h.core.insert(k, v, true); return e }
func (h *Int32ChainMap) InsertIfNotThere(k, v int32) (Int32ChainMapEntryRef, error) {
	i, _, e := h.core.insert(k, v, false)
	return Int32ChainMapEntryRef{chainRef{&h.core, h.core.epoch, i}}, e
}
func (h *Int32ChainMap) Contains(k int32) (bool, error) {
	if e := h.core.open(); e != nil {
		return false, e
	}
	_, ok := h.core.find(k, true)
	return ok, nil
}
func (h *Int32ChainMap) Find(k int32) (int32, bool, error) {
	if e := h.core.open(); e != nil {
		return 0, false, e
	}
	i, ok := h.core.find(k, true)
	if !ok {
		return 0, false, nil
	}
	return h.core.table[i].value, true, nil
}
func (h *Int32ChainMap) FindInto(k int32, out *int32) (bool, error) {
	v, ok, e := h.Find(k)
	if ok && out != nil {
		*out = v
	}
	return ok, e
}
func (h *Int32ChainMap) FindCore(k int32) (Int32ChainMapEntryRef, bool, error) {
	if e := h.core.open(); e != nil {
		return Int32ChainMapEntryRef{}, false, e
	}
	i, ok := h.core.find(k, true)
	return Int32ChainMapEntryRef{chainRef{&h.core, h.core.epoch, i}}, ok, nil
}
func (h *Int32ChainMap) Erase(k int32) error { return h.core.erase(k) }
func (h *Int32ChainMap) Reset() error        { return h.core.finalize(true) }
func (h *Int32ChainMap) Finalize() error     { return h.core.finalize(false) }
func (h *Int32ChainMap) Swap(other *Int32ChainMap) error {
	if other == nil {
		return ErrChainConfig
	}
	return h.core.swap(&other.core)
}
func (h *Int32ChainMap) Close() {
	if h != nil {
		h.core.close()
	}
}
func (h *Int32ChainMap) CheckInvariant() bool { return h.core.invariant() }

type chainRef struct {
	owner *chainStorage
	epoch uint64
	slot  uint32
}

func (r chainRef) validate() error {
	if r.owner == nil || r.owner.closed || r.epoch != r.owner.epoch || r.slot >= uint32(len(r.owner.table)) || r.owner.table[r.slot].next == ChainFree {
		return ErrChainView
	}
	return nil
}

type Int32ChainEntryRef struct{ ref chainRef }

func (r Int32ChainEntryRef) Entry() (int32, error) {
	if e := r.ref.validate(); e != nil {
		return 0, e
	}
	return r.ref.owner.table[r.ref.slot].key, nil
}

// Set preserves source mutable T& behavior. Changing a key's bucket is the
// caller's responsibility and can invalidate lookup invariants, as in C++.
func (r Int32ChainEntryRef) Set(v int32) error {
	if e := r.ref.validate(); e != nil {
		return e
	}
	r.ref.owner.table[r.ref.slot].key = v
	return nil
}

type Int32ChainMapEntryRef struct{ ref chainRef }

func (r Int32ChainMapEntryRef) Entry() (Int32ChainPair, error) {
	if e := r.ref.validate(); e != nil {
		return Int32ChainPair{}, e
	}
	c := r.ref.owner.table[r.ref.slot]
	return Int32ChainPair{c.key, c.value}, nil
}
func (r Int32ChainMapEntryRef) SetValue(v int32) error {
	if e := r.ref.validate(); e != nil {
		return e
	}
	r.ref.owner.table[r.ref.slot].value = v
	return nil
}
func (r Int32ChainMapEntryRef) SetKey(v int32) error {
	if e := r.ref.validate(); e != nil {
		return e
	}
	r.ref.owner.table[r.ref.slot].key = v
	return nil
}

type chainIterator struct {
	owner        *chainStorage
	epoch        uint64
	bucket, slot uint32
}

func (s *chainStorage) begin() chainIterator {
	it := chainIterator{s, s.epoch, 0, ChainEnd}
	if s.closed {
		return it
	}
	it.scan()
	return it
}
func (s *chainStorage) end() chainIterator { return chainIterator{s, s.epoch, s.slots, ChainEnd} }
func (it *chainIterator) scan() {
	for it.bucket < it.owner.slots {
		if it.owner.table[it.bucket].next != ChainFree {
			it.slot = it.bucket
			return
		}
		it.bucket++
	}
	it.slot = ChainEnd
}
func (it chainIterator) validate() error {
	if it.owner == nil || it.owner.closed || it.epoch != it.owner.epoch {
		return ErrChainView
	}
	return nil
}
func (it chainIterator) ref() (chainRef, error) {
	if e := it.validate(); e != nil {
		return chainRef{}, e
	}
	if it.slot == ChainEnd {
		return chainRef{}, ErrChainView
	}
	return chainRef{it.owner, it.epoch, it.slot}, nil
}
func (it *chainIterator) nextEntry() error {
	if e := it.validate(); e != nil {
		return e
	}
	if it.slot == ChainEnd {
		return ErrChainView
	}
	it.slot = it.owner.table[it.slot].next
	if it.slot == ChainEnd {
		it.bucket++
		it.scan()
	}
	return nil
}
func (it chainIterator) equal(other chainIterator) bool {
	if (it.owner == nil || it.slot == ChainEnd) && (other.owner == nil || other.slot == ChainEnd) {
		return true
	}
	return it.owner == other.owner && it.epoch == other.epoch && it.slot == other.slot
}

type Int32ChainIterator struct{ it chainIterator }

func (h *Int32ChainTable) Begin() Int32ChainIterator              { return Int32ChainIterator{h.core.begin()} }
func (h *Int32ChainTable) End() Int32ChainIterator                { return Int32ChainIterator{h.core.end()} }
func (it Int32ChainIterator) AtEnd() bool                         { return it.it.owner == nil || it.it.slot == ChainEnd }
func (it Int32ChainIterator) Equal(other Int32ChainIterator) bool { return it.it.equal(other.it) }
func (it Int32ChainIterator) Ref() (Int32ChainEntryRef, error) {
	r, e := it.it.ref()
	return Int32ChainEntryRef{r}, e
}
func (it Int32ChainIterator) Entry() (int32, error) {
	r, e := it.Ref()
	if e != nil {
		return 0, e
	}
	return r.Entry()
}
func (it *Int32ChainIterator) Next() error { return it.it.nextEntry() }
func (it *Int32ChainIterator) PostNext() (Int32ChainIterator, error) {
	old := *it
	return old, it.Next()
}

type Int32ChainMapIterator struct{ it chainIterator }

func (h *Int32ChainTable) Size() uint32       { return h.core.size }
func (h *Int32ChainTable) Capacity() uint32   { return uint32(len(h.core.table)) }
func (h *Int32ChainTable) UsedSlots() uint32  { return h.core.used }
func (h *Int32ChainTable) Collisions() uint32 { return h.core.collisions }
func (h *Int32ChainMap) Size() uint32         { return h.core.size }
func (h *Int32ChainMap) Capacity() uint32     { return uint32(len(h.core.table)) }
func (h *Int32ChainMap) UsedSlots() uint32    { return h.core.used }
func (h *Int32ChainMap) Collisions() uint32   { return h.core.collisions }

func (h *Int32ChainMap) Begin() Int32ChainMapIterator                   { return Int32ChainMapIterator{h.core.begin()} }
func (h *Int32ChainMap) End() Int32ChainMapIterator                     { return Int32ChainMapIterator{h.core.end()} }
func (it Int32ChainMapIterator) AtEnd() bool                            { return it.it.owner == nil || it.it.slot == ChainEnd }
func (it Int32ChainMapIterator) Equal(other Int32ChainMapIterator) bool { return it.it.equal(other.it) }
func (it Int32ChainMapIterator) Ref() (Int32ChainMapEntryRef, error) {
	r, e := it.it.ref()
	return Int32ChainMapEntryRef{r}, e
}
func (it Int32ChainMapIterator) Entry() (Int32ChainPair, error) {
	r, e := it.Ref()
	if e != nil {
		return Int32ChainPair{}, e
	}
	return r.Entry()
}
func (it *Int32ChainMapIterator) Next() error { return it.it.nextEntry() }
func (it *Int32ChainMapIterator) PostNext() (Int32ChainMapIterator, error) {
	old := *it
	return old, it.Next()
}
