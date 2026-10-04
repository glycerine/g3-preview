// Licensed under the MIT license.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"reflect"
)

// obj_hash_entry<expr> has no cached hash: every USED probe reads the node.
// Generation checks adapt source dangling-pointer reads, without retaining nodes.
type astExprIdentityTable struct {
	epoch                   uint64
	entries                 []astNodeRef
	capacity, size, deleted uint32
	bytes                   uint64
	pending                 uint8 // 1 reset allocation; 2 finalize allocation
}

func astExprUsed(r astNodeRef) bool { return r.generation != 0 }
func (t *astExprIdentityTable) allocate(m *ASTManager, n uint32) ([]astNodeRef, error) {
	bytes := uint64(n) * 8
	if n == 0 || n&(n-1) != 0 || bytes > uint64(^uint(0)>>1) {
		return nil, arena.ErrBounds
	}
	if e := m.budget.Reserve(bytes); e != nil {
		return nil, e
	}
	return make([]astNodeRef, int(n)), nil
}
func (t *astExprIdentityTable) init(m *ASTManager) error {
	a, e := t.allocate(m, 8)
	if e != nil {
		return e
	}
	t.epoch++
	t.entries = a
	t.capacity = 8
	t.bytes = 64
	return nil
}
func (t *astExprIdentityTable) rehash(m *ASTManager, n uint32) error {
	if t.epoch == ^uint64(0) {
		return arena.ErrBounds
	}
	if t.pending != 0 {
		return arena.ErrBounds
	}
	a, e := t.allocate(m, n)
	if e != nil {
		return e
	}
	for _, r := range t.entries {
		if !astExprUsed(r) {
			continue
		}
		if e = m.validNode(r); e != nil {
			m.budget.Release(uint64(n) * 8)
			return e
		}
		i := m.nodeAt(r.slot).hash & (n - 1)
		for astExprUsed(a[i]) {
			i = (i + 1) & (n - 1)
		}
		a[i] = r
	}
	m.budget.Release(t.bytes)
	t.epoch++
	t.entries = a
	t.capacity = n
	t.bytes = uint64(n) * 8
	t.deleted = 0
	return nil
}
func (t *astExprIdentityTable) find(m *ASTManager, r astNodeRef) (uint32, bool, error) {
	if len(t.entries) == 0 {
		return 0, false, arena.ErrBounds
	}
	if e := m.validNode(r); e != nil {
		return 0, false, e
	}
	h := m.nodeAt(r.slot).hash
	i := h & (t.capacity - 1)
	for count := uint32(0); count < t.capacity; count++ {
		v := t.entries[i]
		if v == (astNodeRef{}) {
			return i, false, nil
		}
		if astExprUsed(v) {
			if e := m.validNode(v); e != nil {
				return 0, false, e
			}
			if m.nodeAt(v.slot).hash == h && v == r {
				return i, true, nil
			}
		}
		i = (i + 1) & (t.capacity - 1)
	}
	return 0, false, arena.ErrBounds
}
func (t *astExprIdentityTable) insert(m *ASTManager, r astNodeRef) (bool, error) {
	if t.pending != 0 {
		return false, arena.ErrBounds
	}
	if ((t.size + t.deleted) << 2) > t.capacity*3 {
		if t.capacity > ^uint32(0)/2 {
			return false, arena.ErrBounds
		}
		if e := t.rehash(m, t.capacity*2); e != nil {
			return false, e
		}
	}
	if e := m.validNode(r); e != nil {
		return false, e
	}
	h := m.nodeAt(r.slot).hash
	i := h & (t.capacity - 1)
	dead := ^uint32(0)
	for count := uint32(0); count < t.capacity; count++ {
		v := t.entries[i]
		if v == (astNodeRef{}) {
			if dead != ^uint32(0) {
				i = dead
				t.deleted--
			}
			t.entries[i] = r
			t.size++
			return true, nil
		}
		if !astExprUsed(v) {
			dead = i
		} else {
			if e := m.validNode(v); e != nil {
				return false, e
			}
			if m.nodeAt(v.slot).hash == h && v == r {
				t.entries[i] = r
				return false, nil
			}
		}
		i = (i + 1) & (t.capacity - 1)
	}
	return false, arena.ErrBounds
}
func (t *astExprIdentityTable) erase(m *ASTManager, r astNodeRef) error {
	i, yes, e := t.find(m, r)
	if e != nil || !yes {
		return e
	}
	next := (i + 1) & (t.capacity - 1)
	if t.entries[next] == (astNodeRef{}) {
		t.entries[i] = astNodeRef{}
	} else {
		t.entries[i] = astNodeRef{slot: 1}
		t.deleted++
	}
	t.size--
	if t.entries[i].slot == 1 && t.entries[i].generation == 0 && t.deleted > t.size && t.deleted > 64 {
		if m.exprIDs.outOfMemory != nil && m.exprIDs.outOfMemory() {
			return nil
		}
		return t.rehash(m, t.capacity)
	}
	return nil
}
func (t *astExprIdentityTable) complete(m *ASTManager) error {
	n := t.capacity
	if t.pending == 2 {
		n = 64
	}
	a, e := t.allocate(m, n)
	if e != nil {
		return e
	}
	t.entries = a
	t.capacity = n
	t.bytes = uint64(n) * 8
	t.size = 0
	t.deleted = 0
	t.pending = 0
	return nil
}
func (t *astExprIdentityTable) reset(m *ASTManager) error {
	if t.pending != 0 {
		return t.complete(m)
	}
	if t.size == 0 && t.deleted == 0 {
		return nil
	}
	// Valid source counts describe exactly the FREE slots. Preflight only
	// a backing deletion; in-place clearing preserves entry-pointer lifetime.
	shrinks := t.capacity > 16 && (t.capacity-t.size-t.deleted)*4 > t.capacity*3
	if shrinks && t.epoch == ^uint64(0) {
		return arena.ErrBounds
	}
	free := uint32(0)
	for i, r := range t.entries {
		if r == (astNodeRef{}) {
			free++
		} else {
			t.entries[i] = astNodeRef{}
		}
	}
	if t.capacity > 16 && free*4 > t.capacity*3 {
		t.epoch++
		m.budget.Release(t.bytes)
		t.bytes = 0
		t.entries = nil
		t.capacity >>= 1
		t.pending = 1
		return t.complete(m)
	}
	t.size = 0
	t.deleted = 0
	return nil
}
func (t *astExprIdentityTable) finalize(m *ASTManager) error {
	if t.pending != 0 {
		return t.complete(m)
	}
	if t.capacity <= 64 {
		return t.reset(m)
	}
	if t.epoch == ^uint64(0) {
		return arena.ErrBounds
	}
	t.epoch++
	m.budget.Release(t.bytes)
	t.bytes = 0
	t.entries = nil
	t.pending = 2
	return t.complete(m)
}
func (t *astExprIdentityTable) close(m *ASTManager) {
	m.budget.Release(t.bytes)
	t.entries = nil
	t.bytes = 0
}

// ASTExprSparseMark borrows nodes, just like expr_sparse_mark.
type ASTExprSparseMark struct {
	owner astMarkOwner
	table astExprIdentityTable
}

func NewASTExprSparseMark(m *ASTManager) (*ASTExprSparseMark, error) {
	if e := m.lock(); e != nil {
		return nil, e
	}
	defer m.mu.Unlock()
	bytes := uint64(reflect.TypeOf(ASTExprSparseMark{}).Size())
	id, e := prepareASTMarkOwner(m, bytes)
	if e != nil {
		return nil, e
	}
	s := &ASTExprSparseMark{}
	s.owner.bind(m, bytes, id)
	if e = s.table.init(m); e != nil {
		s.owner.finish()
		return nil, e
	}
	return s, nil
}
func (s *ASTExprSparseMark) IsMarked(h ASTHandle) (bool, error) {
	m, e := s.owner.lock()
	if e != nil {
		return false, e
	}
	defer m.mu.Unlock()
	r, e := m.requireExpr(h)
	if e != nil {
		return false, e
	}
	_, yes, e := s.table.find(m, r)
	return yes, e
}
func (s *ASTExprSparseMark) Mark(h ASTHandle, flag bool) error {
	m, e := s.owner.lock()
	if e != nil {
		return e
	}
	defer m.mu.Unlock()
	r, e := m.requireExpr(h)
	if e != nil {
		return e
	}
	if flag {
		_, e = s.table.insert(m, r)
		return e
	}
	return s.table.erase(m, r)
}
func (s *ASTExprSparseMark) Reset() error {
	m, e := s.owner.lock()
	if e != nil {
		return e
	}
	defer m.mu.Unlock()
	return s.table.reset(m)
}
func (s *ASTExprSparseMark) Close() error {
	if e := s.owner.check(); e != nil {
		return e
	}
	m := s.owner.manager
	m.mu.Lock()
	defer m.mu.Unlock()
	s.table.close(m)
	s.owner.finish()
	return nil
}
