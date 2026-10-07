package sat

import (
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
)

type satClauseSet struct {
	self              *satClauseSet
	owner             *ClauseAllocator
	ids               []uint32
	refs              []StoredClauseRef
	budget            *arena.Budget
	ownerBytes, epoch uint64
	closed            bool
}

func NewSATClauseSet(a *ClauseAllocator, b *arena.Budget) (*satClauseSet, error) {
	if e := a.open(); e != nil {
		return nil, e
	}
	return newSATClauseSet(a, b)
}
func newSATClauseSet(a *ClauseAllocator, b *arena.Budget) (*satClauseSet, error) {
	n := uint64(reflect.TypeOf(satClauseSet{}).Size())
	if e := b.Reserve(n); e != nil {
		return nil, e
	}
	s := &satClauseSet{owner: a, budget: b, ownerBytes: n, epoch: 1}
	s.self = s
	return s, nil
}
func (s *satClauseSet) open() error {
	if s == nil || s.self != s {
		return ErrSATOwner
	}
	if s.closed {
		return arena.ErrClosed
	}
	return nil
}
func (s *satClauseSet) change() error {
	if e := s.open(); e != nil {
		return e
	}
	if s.epoch == ^uint64(0) {
		return arena.ErrSlotLimit
	}
	return nil
}
func (s *satClauseSet) Size() uint32 {
	if s == nil || s.closed {
		return 0
	}
	return uint32(len(s.refs))
}
func (s *satClauseSet) Empty() bool { return s.Size() == 0 }
func (s *satClauseSet) growIDs(need uint64) error {
	for need > uint64(cap(s.ids)) {
		n, e := satVectorNextCapacity(uint64(cap(s.ids)), 4)
		if e != nil {
			return e
		}
		if e = s.budget.Reserve(n * 4); e != nil {
			return e
		}
		next := make([]uint32, len(s.ids), int(n))
		copy(next, s.ids)
		s.budget.Release(uint64(cap(s.ids)) * 4)
		s.ids = next
		if s.epoch != ^uint64(0) {
			s.epoch++
		}
	}
	return nil
}
func (s *satClauseSet) reserveIDs(n uint32) error {
	if e := s.growIDs(uint64(n)); e != nil {
		return e
	}
	old := len(s.ids)
	if int(n) > old {
		s.ids = s.ids[:int(n)]
		for i := old; i < int(n); i++ {
			s.ids[i] = ^uint32(0)
		}
	}
	return nil
}
func (s *satClauseSet) growRefs(need uint64) error {
	stride := uint64(reflect.TypeOf(StoredClauseRef{}).Size())
	for need > uint64(cap(s.refs)) {
		n, e := satVectorNextCapacity(uint64(cap(s.refs)), stride)
		if e != nil {
			return e
		}
		if e = s.budget.Reserve(n * stride); e != nil {
			return e
		}
		next := make([]StoredClauseRef, len(s.refs), int(n))
		copy(next, s.refs)
		s.budget.Release(uint64(cap(s.refs)) * stride)
		s.refs = next
		if s.epoch != ^uint64(0) {
			s.epoch++
		}
	}
	return nil
}
func (s *satClauseSet) Contains(r StoredClauseRef) (bool, error) {
	if e := s.open(); e != nil {
		return false, e
	}
	id, e := s.owner.ID(r)
	if e != nil {
		return false, e
	}
	return uint64(id) < uint64(len(s.ids)) && s.ids[id] != ^uint32(0), nil
}
func (s *satClauseSet) Insert(r StoredClauseRef) error {
	if e := s.change(); e != nil {
		return e
	}
	id, e := s.owner.ID(r)
	if e != nil {
		return e
	}
	if e = s.reserveIDs(id + 1); e != nil {
		return e
	}
	if uint64(id) >= uint64(len(s.ids)) {
		return arena.ErrBounds
	}
	if s.ids[id] != ^uint32(0) {
		return nil
	}
	s.ids[id] = uint32(len(s.refs))
	if e = s.growRefs(uint64(len(s.refs)) + 1); e != nil {
		return e
	}
	s.refs = append(s.refs, r)
	if s.epoch != ^uint64(0) {
		s.epoch++
	}
	return nil
}
func (s *satClauseSet) Erase(r StoredClauseRef) error {
	if e := s.change(); e != nil {
		return e
	}
	id, e := s.owner.ID(r)
	if e != nil {
		return e
	}
	if uint64(id) >= uint64(len(s.ids)) || len(s.refs) == 0 {
		return nil
	}
	pos := s.ids[id]
	if pos == ^uint32(0) {
		return nil
	}
	s.ids[id] = ^uint32(0)
	last := len(s.refs) - 1
	if uint64(pos) != uint64(last) {
		lastRef := s.refs[last]
		if uint64(pos) >= uint64(len(s.refs)) {
			return arena.ErrBounds
		}
		s.refs[pos] = lastRef
		lid, e := s.owner.ID(lastRef)
		if e != nil {
			return e
		}
		if uint64(lid) >= uint64(len(s.ids)) {
			return arena.ErrBounds
		}
		s.ids[lid] = pos
	}
	s.refs = s.refs[:last]
	if s.epoch != ^uint64(0) {
		s.epoch++
	}
	return nil
}
func (s *satClauseSet) Pop() (StoredClauseRef, error) {
	if e := s.change(); e != nil {
		return StoredClauseRef{}, e
	}
	if len(s.refs) == 0 {
		if satStorageSourceDebug {
			return StoredClauseRef{}, ErrSATAssertion
		}
		return StoredClauseRef{}, arena.ErrBounds
	}
	r := s.refs[len(s.refs)-1]
	id, e := s.owner.ID(r)
	if e != nil {
		return r, e
	}
	if satStorageSourceDebug && (uint64(id) >= uint64(len(s.ids)) || s.ids[id] != uint32(len(s.refs)-1)) {
		return r, ErrSATAssertion
	}
	if uint64(id) < uint64(len(s.ids)) {
		s.ids[id] = ^uint32(0)
	}
	s.refs = s.refs[:len(s.refs)-1]
	if s.epoch != ^uint64(0) {
		s.epoch++
	}
	return r, nil
}
func (s *satClauseSet) Get(i uint32) (StoredClauseRef, error) {
	if e := s.open(); e != nil {
		return StoredClauseRef{}, e
	}
	if uint64(i) >= uint64(len(s.refs)) {
		return StoredClauseRef{}, arena.ErrBounds
	}
	return s.refs[i], nil
}
func (s *satClauseSet) CheckInvariant() (bool, error) {
	if e := s.open(); e != nil {
		return false, e
	}
	if !satStorageSourceDebug {
		return true, nil
	}
	for pos, r := range s.refs {
		id, e := s.owner.ID(r)
		if e != nil {
			return false, e
		}
		if uint64(id) >= uint64(len(s.ids)) || s.ids[id] != uint32(pos) {
			return false, ErrSATAssertion
		}
	}
	for id, pos := range s.ids {
		if pos == ^uint32(0) {
			continue
		}
		if uint64(pos) >= uint64(len(s.refs)) {
			return false, ErrSATAssertion
		}
		got, e := s.owner.ID(s.refs[pos])
		if e != nil {
			return false, e
		}
		if got != uint32(id) {
			return false, ErrSATAssertion
		}
	}
	return true, nil
}
func (s *satClauseSet) Reset() error {
	if e := s.change(); e != nil {
		return e
	}
	s.ids = s.ids[:0]
	s.refs = s.refs[:0]
	if s.epoch != ^uint64(0) {
		s.epoch++
	}
	return nil
}
func (s *satClauseSet) Finalize() error {
	if e := s.change(); e != nil {
		return e
	}
	s.budget.Release(uint64(cap(s.ids)) * 4)
	s.ids = nil
	s.budget.Release(uint64(cap(s.refs)) * uint64(reflect.TypeOf(StoredClauseRef{}).Size()))
	s.refs = nil
	if s.epoch != ^uint64(0) {
		s.epoch++
	}
	return nil
}
func (s *satClauseSet) Close() error {
	if s == nil || s.self != s {
		return ErrSATOwner
	}
	if s.closed {
		return nil
	}
	s.budget.Release(s.ownerBytes + uint64(cap(s.ids))*4 + uint64(cap(s.refs))*uint64(reflect.TypeOf(StoredClauseRef{}).Size()))
	s.ids = nil
	s.refs = nil
	s.ownerBytes = 0
	s.closed = true
	return nil
}
func (s *satClauseSet) AppendTo(dst []StoredClauseRef) ([]StoredClauseRef, error) {
	if e := s.open(); e != nil {
		return dst, e
	}
	return append(dst, s.refs...), nil
}
func (s *satClauseSet) copyIDs(src *satClauseSet) error {
	bytes := uint64(cap(src.ids)) * 4
	if bytes == 0 {
		return nil
	}
	if e := s.budget.Reserve(bytes); e != nil {
		return e
	}
	s.ids = make([]uint32, len(src.ids), cap(src.ids))
	copy(s.ids, src.ids)
	return nil
}
func (s *satClauseSet) copyRefs(src *satClauseSet) error {
	bytes := uint64(cap(src.refs)) * uint64(reflect.TypeOf(StoredClauseRef{}).Size())
	if bytes == 0 {
		return nil
	}
	if e := s.budget.Reserve(bytes); e != nil {
		return e
	}
	s.refs = make([]StoredClauseRef, len(src.refs), cap(src.refs))
	copy(s.refs, src.refs)
	return nil
}
func (s *satClauseSet) Clone(b *arena.Budget) (*satClauseSet, error) {
	if e := s.open(); e != nil {
		return nil, e
	}
	v, e := newSATClauseSet(s.owner, b)
	if e != nil {
		return nil, e
	}
	if e = v.copyIDs(s); e == nil {
		e = v.copyRefs(s)
	}
	if e != nil {
		v.Close()
		return nil, e
	}
	return v, nil
}
func (s *satClauseSet) Assign(src *satClauseSet) error {
	if e := s.change(); e != nil {
		return e
	}
	if e := src.open(); e != nil {
		return e
	}
	if s == src {
		return nil
	}
	if s.owner != src.owner {
		return ErrSATOwner
	}
	s.budget.Release(uint64(cap(s.ids)) * 4)
	s.ids = nil
	if e := s.copyIDs(src); e != nil {
		return e
	}
	s.budget.Release(uint64(cap(s.refs)) * uint64(reflect.TypeOf(StoredClauseRef{}).Size()))
	s.refs = nil
	if s.epoch != ^uint64(0) {
		s.epoch++
	}
	return s.copyRefs(src)
}
func (s *satClauseSet) MoveAssign(src *satClauseSet) error { return s.Assign(src) }
func (s *satClauseSet) MoveClone(b *arena.Budget) (*satClauseSet, error) {
	if e := s.change(); e != nil {
		return nil, e
	}
	v, e := newSATClauseSet(s.owner, b)
	if e != nil {
		return nil, e
	}
	n := uint64(cap(s.ids))*4 + uint64(cap(s.refs))*uint64(reflect.TypeOf(StoredClauseRef{}).Size())
	if b != s.budget {
		if e = b.Reserve(n); e != nil {
			v.Close()
			return nil, e
		}
		s.budget.Release(n)
	}
	v.ids = s.ids
	v.refs = s.refs
	s.ids = nil
	s.refs = nil
	if s.epoch != ^uint64(0) {
		s.epoch++
	}
	return v, nil
}

type satClauseSetIterator struct {
	owner *satClauseSet
	epoch uint64
	index uint32
}

func (s *satClauseSet) Iter() satClauseSetIterator {
	return satClauseSetIterator{owner: s, epoch: s.epoch}
}
func (i *satClauseSetIterator) Next() (StoredClauseRef, bool, error) {
	if e := i.owner.open(); e != nil {
		return StoredClauseRef{}, false, e
	}
	if i.epoch != i.owner.epoch {
		return StoredClauseRef{}, false, arena.ErrInvalidSlot
	}
	if i.index >= i.owner.Size() {
		return StoredClauseRef{}, false, nil
	}
	r := i.owner.refs[i.index]
	i.index++
	return r, true, nil
}
