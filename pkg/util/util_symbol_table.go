// Concrete int32 translation of util/symbol_table.h and its core_hashtable.
package util

import (
	"errors"
	"math"
	"reflect"
	"sync"

	"github.com/glycerine/g3/pkg/arena"
)

var ErrSymbolTableScope = errors.New("g3: symbol table has no scope")
var ErrSymbolTablePrecondition = errors.New("g3: symbol table source precondition")
var symbolInt32AppendMutex sync.Mutex

type SymbolInt32TableConfig struct {
	Budget          *arena.Budget
	InitialCapacity uint32
}

// key is also the source state: null is FREE, DummySymbol is DELETED.
// Hashes are read through the borrowed SymbolManager; none are stored here.
type symbolInt32Entry struct {
	key  Symbol
	data int32
}
type symbolInt32State struct {
	table, trail  []symbolInt32Entry
	limits        []uint32
	size, deleted uint32
}
type SymbolInt32TableStats struct {
	Size, Deleted, Capacity, TrailLength, TrailCapacity, Scopes, ScopeCapacity uint32
	OwnerBytes, TableBytes, TrailBytes, ScopeBytes                             uint64
}

// The manager is borrowed and must outlive this owner. Its teardown must not
// race operations. The cold mutex serializes this owner's graph; shared Budgets
// require caller serialization across other owners. No entry holds Go pointers.
type SymbolInt32Table struct {
	mu      sync.Mutex
	manager *SymbolManager
	budget  *arena.Budget
	symbolInt32State
	closed bool
}

func SymbolInt32EntryBytes(n int) uint64 {
	return uint64(n) * uint64(reflect.TypeOf(symbolInt32Entry{}).Size())
}
func (s *symbolInt32State) bytes() uint64 {
	return SymbolInt32EntryBytes(cap(s.table)) + SymbolInt32EntryBytes(cap(s.trail)) + uint64(cap(s.limits))*4
}
func NewSymbolInt32Table(m *SymbolManager, c SymbolInt32TableConfig) (*SymbolInt32Table, error) {
	if m == nil {
		return nil, ErrSymbolTablePrecondition
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil, arena.ErrClosed
	}
	n := c.InitialCapacity
	if n == 0 {
		n = 8
	}
	if n&(n-1) != 0 {
		return nil, ErrSymbolTablePrecondition
	}
	if uint64(n) > uint64(int(^uint(0)>>1))/uint64(reflect.TypeOf(symbolInt32Entry{}).Size()) {
		return nil, arena.ErrBounds
	}
	owner := uint64(reflect.TypeOf(SymbolInt32Table{}).Size())
	if e := c.Budget.Reserve(owner); e != nil {
		return nil, e
	}
	amount := SymbolInt32EntryBytes(int(n))
	if e := c.Budget.Reserve(amount); e != nil {
		c.Budget.Release(owner)
		return nil, e
	}
	return &SymbolInt32Table{manager: m, budget: c.Budget, symbolInt32State: symbolInt32State{table: make([]symbolInt32Entry, n)}}, nil
}
func (h *SymbolInt32Table) open() error {
	if h.closed || h.manager.closed {
		return arena.ErrClosed
	}
	return nil
}
func (h *SymbolInt32Table) key(k Symbol) error {
	if e := h.manager.checkPlain(k); e != nil {
		return e
	}
	if k == (Symbol{}) {
		return ErrInvalidSymbol
	}
	return nil
}

// manager.mu is held by every public storage operation.
func (h *SymbolInt32Table) hash(k Symbol) uint32 {
	if k.IsNumerical() {
		return k.Value
	}
	var v uint32
	ref := arena.BytesRegionRef{Slot: arena.Slot(k.Value), Generation: k.Generation}
	for i := uint32(0); i < 4; i++ {
		b, e := h.manager.region.Read(ref, i)
		if e != nil {
			panic("g3: borrowed symbol lifetime violated")
		}
		v |= uint32(b) << (8 * i)
	}
	return v
}
func used(k Symbol) bool { return k != (Symbol{}) && k != DummySymbol() }
func (h *SymbolInt32Table) find(s *symbolInt32State, k Symbol) (int, bool) {
	hash := h.hash(k)
	mask := uint32(len(s.table)) - 1
	for step := uint32(0); step < uint32(len(s.table)); step++ {
		i := int((hash + step) & mask)
		e := s.table[i]
		if e.key == (Symbol{}) {
			return i, false
		}
		if used(e.key) && h.hash(e.key) == hash && e.key == k {
			return i, true
		}
	}
	return -1, false
}
func (h *SymbolInt32Table) put(s *symbolInt32State, e symbolInt32Entry) {
	hash := h.hash(e.key)
	mask := uint32(len(s.table)) - 1
	del := -1
	for step := uint32(0); step < uint32(len(s.table)); step++ {
		i := int((hash + step) & mask)
		k := s.table[i].key
		if used(k) {
			if h.hash(k) == hash && k == e.key {
				s.table[i] = e
				return
			}
		} else if k == DummySymbol() {
			del = i
		} else {
			if del >= 0 {
				i = del
				s.deleted--
			}
			s.table[i] = e
			s.size++
			return
		}
	}
	panic("g3: symbol table full")
}
func tableNeedsGrowth(s *symbolInt32State) bool {
	return (uint64(s.size)+uint64(s.deleted))*4 > uint64(len(s.table))*3
}
func (h *SymbolInt32Table) newTable(s *symbolInt32State, n uint32) ([]symbolInt32Entry, error) {
	if n == 0 || uint64(n) > uint64(int(^uint(0)>>1))/uint64(reflect.TypeOf(symbolInt32Entry{}).Size()) {
		return nil, arena.ErrBounds
	}
	if e := h.budget.Reserve(SymbolInt32EntryBytes(int(n))); e != nil {
		return nil, e
	}
	next := symbolInt32State{table: make([]symbolInt32Entry, n)}
	for _, entry := range s.table {
		if used(entry.key) {
			h.put(&next, entry)
		}
	}
	return next.table, nil
}
func NextSymbolVectorCapacity(n int) (int, error) {
	if n == 0 {
		return 2, nil
	}
	v := (uint64(n)*3 + 1) / 2
	if v > math.MaxUint32 || v <= uint64(n) || v > uint64(int(^uint(0)>>1)) {
		return 0, arena.ErrBounds
	}
	return int(v), nil
}
func (h *SymbolInt32Table) prepareTrail(s *symbolInt32State) ([]symbolInt32Entry, error) {
	if len(s.trail) < cap(s.trail) {
		return s.trail, nil
	}
	n, e := NextSymbolVectorCapacity(cap(s.trail))
	if e != nil {
		return nil, e
	}
	if uint64(n) > uint64(int(^uint(0)>>1))/uint64(reflect.TypeOf(symbolInt32Entry{}).Size()) {
		return nil, arena.ErrBounds
	}
	if e = h.budget.Reserve(SymbolInt32EntryBytes(n)); e != nil {
		return nil, e
	}
	p := make([]symbolInt32Entry, len(s.trail), n)
	copy(p, s.trail)
	return p, nil
}

// All reservations precede table/trail publication. On failure only successful
// fault-injection attempt prefixes are consumed, matching Budget's contract.
func (h *SymbolInt32Table) insert(s *symbolInt32State, k Symbol, v int32) error {
	i, present := h.find(s, k)
	scoped := len(s.limits) > 0
	var trail = []symbolInt32Entry(nil)
	var e error
	if scoped {
		trail, e = h.prepareTrail(s)
		if e != nil {
			return e
		}
	}
	next := s.table
	if !(scoped && present) && tableNeedsGrowth(s) {
		if len(s.table) > math.MaxUint32/2 {
			e = arena.ErrBounds
		} else {
			next, e = h.newTable(s, uint32(len(s.table))*2)
		}
		if e != nil {
			if scoped && cap(trail) != cap(s.trail) {
				h.budget.Release(SymbolInt32EntryBytes(cap(trail)))
			}
			return e
		}
	}
	if scoped {
		if cap(trail) != cap(s.trail) {
			h.budget.Release(SymbolInt32EntryBytes(cap(s.trail)))
		}
		s.trail = trail
		if present {
			s.trail = append(s.trail, s.table[i])
		} else {
			marked, _ := k.Mark()
			s.trail = append(s.trail, symbolInt32Entry{key: marked})
		}
	}
	if len(next) != len(s.table) {
		h.budget.Release(SymbolInt32EntryBytes(len(s.table)))
		s.table = next
		s.deleted = 0
	}
	if scoped && present {
		s.table[i].data = v
	} else {
		h.put(s, symbolInt32Entry{key: k, data: v})
	}
	return nil
}
func (h *SymbolInt32Table) lock() error {
	h.mu.Lock()
	h.manager.mu.Lock()
	if e := h.open(); e != nil {
		h.unlock()
		return e
	}
	return nil
}
func (h *SymbolInt32Table) unlock() { h.manager.mu.Unlock(); h.mu.Unlock() }
func (h *SymbolInt32Table) Insert(k Symbol, v int32) error {
	if e := h.lock(); e != nil {
		return e
	}
	defer h.unlock()
	if e := h.key(k); e != nil {
		return e
	}
	return h.insert(&h.symbolInt32State, k, v)
}
func (h *SymbolInt32Table) Find(k Symbol) (int32, bool, error) {
	if e := h.lock(); e != nil {
		return 0, false, e
	}
	defer h.unlock()
	if e := h.key(k); e != nil {
		return 0, false, e
	}
	i, ok := h.find(&h.symbolInt32State, k)
	if !ok {
		return 0, false, nil
	}
	return h.table[i].data, true, nil
}
func (h *SymbolInt32Table) FindInto(k Symbol, out *int32) (bool, error) {
	if out == nil {
		return false, ErrSymbolTablePrecondition
	}
	v, ok, e := h.Find(k)
	if ok && e == nil {
		*out = v
	}
	return ok, e
}
func (h *SymbolInt32Table) Contains(k Symbol) (bool, error) { _, ok, e := h.Find(k); return ok, e }
func (h *SymbolInt32Table) ScopeLevel() uint32 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return uint32(len(h.limits))
}
func (h *SymbolInt32Table) BeginScope() error {
	if e := h.lock(); e != nil {
		return e
	}
	defer h.unlock()
	if len(h.limits) == cap(h.limits) {
		n, e := NextSymbolVectorCapacity(cap(h.limits))
		if e != nil {
			return e
		}
		if uint64(n) > uint64(int(^uint(0)>>1))/4 {
			return arena.ErrBounds
		}
		if e = h.budget.Reserve(uint64(n) * 4); e != nil {
			return e
		}
		p := make([]uint32, len(h.limits), n)
		copy(p, h.limits)
		h.budget.Release(uint64(cap(h.limits)) * 4)
		h.limits = p
	}
	h.limits = append(h.limits, uint32(len(h.trail)))
	return nil
}
func (h *SymbolInt32Table) erase(s *symbolInt32State, k Symbol) error {
	i, ok := h.find(s, k)
	if !ok {
		return nil
	}
	next := (i + 1) % len(s.table)
	s.size--
	if s.table[next].key == (Symbol{}) {
		s.table[i].key = Symbol{}
		return nil
	}
	s.table[i].key = DummySymbol()
	s.deleted++
	if s.deleted > s.size && s.deleted > 64 {
		p, e := h.newTable(s, uint32(len(s.table)))
		if e != nil {
			return e
		}
		h.budget.Release(SymbolInt32EntryBytes(len(s.table)))
		s.table = p
		s.deleted = 0
	}
	return nil
}

// clonePhysical deliberately copies sentinel state AND retained inactive data.
// It is transactional scratch, not the source copy constructor's rehash.
func (h *SymbolInt32Table) clonePhysical(s *symbolInt32State) (symbolInt32State, error) {
	if e := h.budget.Reserve(s.bytes()); e != nil {
		return symbolInt32State{}, e
	}
	p := *s
	p.table = make([]symbolInt32Entry, len(s.table))
	copy(p.table, s.table)
	if cap(s.trail) > 0 {
		p.trail = make([]symbolInt32Entry, len(s.trail), cap(s.trail))
		copy(p.trail[:cap(p.trail)], s.trail[:cap(s.trail)])
	}
	if cap(s.limits) > 0 {
		p.limits = make([]uint32, len(s.limits), cap(s.limits))
		copy(p.limits[:cap(p.limits)], s.limits[:cap(s.limits)])
	}
	return p, nil
}
func (h *SymbolInt32Table) undo(s *symbolInt32State) error {
	old := s.limits[len(s.limits)-1]
	s.limits = s.limits[:len(s.limits)-1] // source order
	for len(s.trail) > int(old) {
		e := s.trail[len(s.trail)-1]
		if e.key.IsMarked() {
			k, _ := e.key.unmark()
			s.trail[len(s.trail)-1].key = k
			if x := h.erase(s, k); x != nil {
				return x
			}
		} else {
			if tableNeedsGrowth(s) {
				p, x := h.newTable(s, uint32(len(s.table))*2)
				if x != nil {
					return x
				}
				h.budget.Release(SymbolInt32EntryBytes(len(s.table)))
				s.table = p
				s.deleted = 0
			}
			h.put(s, e)
		}
		s.trail = s.trail[:len(s.trail)-1]
	}
	return nil
}

// During undo size+deleted never increases: restores overwrite extant bindings,
// while new bindings erase. If initially <=75% and deleted cannot reach65,
// neither insert growth nor deletion cleanup can occur. Thus direct replay is
// failure-free and does not allocate. Without restore records there is no insert
// growth trigger regardless of initial occupancy. Other cases stage the graph.
func (h *SymbolInt32Table) EndScope() error {
	if e := h.lock(); e != nil {
		return e
	}
	defer h.unlock()
	if len(h.limits) == 0 {
		return ErrSymbolTableScope
	}
	old := h.limits[len(h.limits)-1]
	if uint32(len(h.trail)) == old {
		h.limits = h.limits[:len(h.limits)-1]
		return nil
	}
	newKeys := uint64(0)
	restores := false
	for _, e := range h.trail[old:] {
		if e.key.IsMarked() {
			newKeys++
		} else {
			restores = true
		}
	}
	if (!restores || !tableNeedsGrowth(&h.symbolInt32State)) && uint64(h.deleted)+newKeys <= 64 {
		return h.undo(&h.symbolInt32State)
	}
	p, e := h.clonePhysical(&h.symbolInt32State)
	if e != nil {
		return e
	}
	if e = h.undo(&p); e != nil {
		h.budget.Release(p.bytes())
		return e
	}
	h.budget.Release(h.symbolInt32State.bytes())
	h.symbolInt32State = p
	return nil
}
func (h *SymbolInt32Table) Reset() error {
	if e := h.lock(); e != nil {
		return e
	}
	defer h.unlock()
	if e := h.resetTable(&h.symbolInt32State); e != nil {
		return e
	}
	h.trail = h.trail[:0]
	h.limits = h.limits[:0]
	return nil
}
func (h *SymbolInt32Table) resetTable(s *symbolInt32State) error {
	if s.size == 0 && s.deleted == 0 {
		return nil
	}
	free := uint64(len(s.table)) - uint64(s.size) - uint64(s.deleted)
	if len(s.table) > 16 && free*4 > uint64(len(s.table))*3 {
		n := len(s.table) / 2
		if e := h.budget.Reserve(SymbolInt32EntryBytes(n)); e != nil {
			return e
		}
		p := make([]symbolInt32Entry, n)
		h.budget.Release(SymbolInt32EntryBytes(len(s.table)))
		s.table = p
	} else {
		for i := range s.table {
			s.table[i].key = Symbol{}
		}
	}
	s.size = 0
	s.deleted = 0
	return nil
}

// copyVectors follows vector::copy_core: copy live elements, retaining the
// source capacity. Spare source elements are not constructed/copied in C++.
func (h *SymbolInt32Table) copyVectors(dst, src *symbolInt32State) error {
	trailBytes := SymbolInt32EntryBytes(cap(src.trail))
	scopeBytes := uint64(cap(src.limits)) * 4
	var trail []symbolInt32Entry
	var limits []uint32
	if trailBytes > 0 {
		if e := h.budget.Reserve(trailBytes); e != nil {
			return e
		}
		trail = make([]symbolInt32Entry, len(src.trail), cap(src.trail))
		copy(trail, src.trail)
	}
	if scopeBytes > 0 {
		if e := h.budget.Reserve(scopeBytes); e != nil {
			h.budget.Release(trailBytes)
			return e
		}
		limits = make([]uint32, len(src.limits), cap(src.limits))
		copy(limits, src.limits)
	}
	h.budget.Release(SymbolInt32EntryBytes(cap(dst.trail)) + uint64(cap(dst.limits))*4)
	dst.trail = trail
	dst.limits = limits
	return nil
}

// Clone implements the source implicit copy constructor: table entries rehash
// USED buckets at exactly source capacity, while vectors copy their capacities
// and live marked history. The new owner borrows the same manager and charges
// all of its storage to destinationBudget; mutexes are freshly initialized.
func (h *SymbolInt32Table) Clone(destinationBudget *arena.Budget) (*SymbolInt32Table, error) {
	if e := h.lock(); e != nil {
		return nil, e
	}
	defer h.unlock()
	header := uint64(reflect.TypeOf(SymbolInt32Table{}).Size())
	if e := destinationBudget.Reserve(header); e != nil {
		return nil, e
	}
	copy := &SymbolInt32Table{manager: h.manager, budget: destinationBudget}
	table, e := copy.newTable(&h.symbolInt32State, uint32(len(h.table)))
	if e != nil {
		destinationBudget.Release(header)
		return nil, e
	}
	copy.table = table
	copy.size = h.size
	if e = copy.copyVectors(&copy.symbolInt32State, &h.symbolInt32State); e != nil {
		destinationBudget.Release(header + copy.symbolInt32State.bytes())
		return nil, e
	}
	return copy, nil
}

// Assign implements implicit member assignment. Unlike Clone, source
// core_hashtable::operator= resets the destination then replays source buckets
// through normal table insertion; destination capacity may shrink or grow.
// A charged physical destination copy stages the transaction so predictable
// failure leaves its graph unchanged. Manager and Budget are never reassigned.
func (h *SymbolInt32Table) Assign(other *SymbolInt32Table) error {
	if other == nil {
		return ErrSymbolTablePrecondition
	}
	symbolInt32AppendMutex.Lock()
	defer symbolInt32AppendMutex.Unlock()
	h.mu.Lock()
	defer h.mu.Unlock()
	if other != h {
		other.mu.Lock()
		defer other.mu.Unlock()
	}
	h.manager.mu.Lock()
	defer h.manager.mu.Unlock()
	if e := h.open(); e != nil {
		return e
	}
	if other.closed {
		return arena.ErrClosed
	}
	if h.manager != other.manager {
		return ErrSymbolOwner
	}
	if h == other {
		return nil
	}
	p, e := h.clonePhysical(&h.symbolInt32State)
	if e != nil {
		return e
	}
	if e = h.resetTable(&p); e != nil {
		h.budget.Release(p.bytes())
		return e
	}
	for _, entry := range other.table {
		if !used(entry.key) {
			continue
		}
		if tableNeedsGrowth(&p) {
			var table []symbolInt32Entry
			table, e = h.newTable(&p, uint32(len(p.table))*2)
			if e != nil {
				h.budget.Release(p.bytes())
				return e
			}
			h.budget.Release(SymbolInt32EntryBytes(len(p.table)))
			p.table = table
			p.deleted = 0
		}
		h.put(&p, entry)
	}
	if e = h.copyVectors(&p, &other.symbolInt32State); e != nil {
		h.budget.Release(p.bytes())
		return e
	}
	h.budget.Release(h.symbolInt32State.bytes())
	h.symbolInt32State = p
	return nil
}
func (h *SymbolInt32Table) Append(other *SymbolInt32Table) error {
	if other == nil {
		return ErrSymbolTablePrecondition
	}
	symbolInt32AppendMutex.Lock()
	defer symbolInt32AppendMutex.Unlock()
	h.mu.Lock()
	defer h.mu.Unlock()
	if other != h {
		other.mu.Lock()
		defer other.mu.Unlock()
	}
	h.manager.mu.Lock()
	defer h.manager.mu.Unlock()
	if e := h.open(); e != nil {
		return e
	}
	if other.closed {
		return arena.ErrClosed
	}
	if h.manager != other.manager {
		return ErrSymbolOwner
	}
	if other == h {
		if len(h.limits) == 0 && tableNeedsGrowth(&h.symbolInt32State) && h.size > 0 {
			return ErrSymbolTablePrecondition
		}
		if len(h.limits) == 0 {
			return nil
		}
	}
	if other.size == 0 {
		return nil
	}
	// Conservatively bound every source insertion's occupancy and trail push.
	// Scoped existing-only inserts bypass the table growth check entirely.
	// If neither array can grow, replay cannot fail and needs no workspace.
	missing := uint64(0)
	for _, entry := range other.table {
		if used(entry.key) {
			if _, ok := h.find(&h.symbolInt32State, entry.key); !ok {
				missing++
			}
		}
	}
	if (len(h.limits) > 0 && missing == 0 || (uint64(h.size)+uint64(h.deleted)+missing)*4 <= uint64(len(h.table))*3) &&
		(len(h.limits) == 0 || uint64(len(h.trail))+uint64(other.size) <= uint64(cap(h.trail))) {
		for _, entry := range other.table {
			if used(entry.key) {
				if e := h.insert(&h.symbolInt32State, entry.key, entry.data); e != nil {
					panic("g3: nonallocating append preflight violated")
				}
			}
		}
		return nil
	}
	p, e := h.clonePhysical(&h.symbolInt32State)
	if e != nil {
		return e
	}
	for _, v := range other.table {
		if used(v.key) {
			if e = h.insert(&p, v.key, v.data); e != nil {
				h.budget.Release(p.bytes())
				return e
			}
		}
	}
	h.budget.Release(h.symbolInt32State.bytes())
	h.symbolInt32State = p
	return nil
}

// Outputs are caller-owned buffers and preserve their existing prefixes.
func (h *SymbolInt32Table) AppendDom(dst []Symbol) ([]Symbol, error) {
	if e := h.lock(); e != nil {
		return dst, e
	}
	defer h.unlock()
	for _, v := range h.table {
		if used(v.key) {
			dst = append(dst, v.key)
		}
	}
	return dst, nil
}
func (h *SymbolInt32Table) AppendRange(dst []int32) ([]int32, error) {
	if e := h.lock(); e != nil {
		return dst, e
	}
	defer h.unlock()
	for _, v := range h.table {
		if used(v.key) {
			dst = append(dst, v.data)
		}
	}
	return dst, nil
}
func (h *SymbolInt32Table) Stats() SymbolInt32TableStats {
	h.mu.Lock()
	defer h.mu.Unlock()
	s := SymbolInt32TableStats{Size: h.size, Deleted: h.deleted, Capacity: uint32(len(h.table)), TrailLength: uint32(len(h.trail)), TrailCapacity: uint32(cap(h.trail)), Scopes: uint32(len(h.limits)), ScopeCapacity: uint32(cap(h.limits)), TableBytes: SymbolInt32EntryBytes(cap(h.table)), TrailBytes: SymbolInt32EntryBytes(cap(h.trail)), ScopeBytes: uint64(cap(h.limits)) * 4}
	if !h.closed {
		s.OwnerBytes = uint64(reflect.TypeOf(SymbolInt32Table{}).Size())
	}
	return s
}
func (h *SymbolInt32Table) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return nil
	}
	h.budget.Release(uint64(reflect.TypeOf(SymbolInt32Table{}).Size()) + h.symbolInt32State.bytes())
	h.symbolInt32State = symbolInt32State{}
	h.closed = true
	return nil
}
