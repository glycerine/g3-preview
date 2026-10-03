// Copyright (c) 2011 Microsoft Corporation. Licensed under the MIT license.
// Actual sat::uint_set at src/sat/sat_types.h155-226, not global bitmap uint_set.
package g3

import (
	"reflect"

	"github.com/glycerine/g3/golang/arena"
)

type satSolverUintSet struct {
	self        *satSolverUintSet
	inSet       []byte // source svector<char>, false0/true1.
	values      []uint32
	budget      *arena.Budget
	ownerBytes  uint64
	valuesEpoch uint64
	closed      bool
}
type satSolverUintSetStats struct {
	OwnerBytes, MembershipSize, MembershipCapacity, ValuesSize, ValuesCapacity uint64
}

func newSATSolverUintSet(b *arena.Budget) (*satSolverUintSet, error) {
	n := uint64(reflect.TypeOf(satSolverUintSet{}).Size())
	if e := b.Reserve(n); e != nil {
		return nil, e
	}
	s := &satSolverUintSet{budget: b, ownerBytes: n, valuesEpoch: 1}
	s.self = s
	return s, nil
}
func (s *satSolverUintSet) open() error {
	if s == nil || s.self != s {
		return ErrSATOwner
	}
	if s.closed {
		return arena.ErrClosed
	}
	return nil
}
func (s *satSolverUintSet) Size() uint32 {
	if s == nil || s.closed {
		return 0
	}
	return uint32(len(s.values))
}
func (s *satSolverUintSet) Empty() bool { return s.Size() == 0 }
func (s *satSolverUintSet) MembershipSize() uint64 {
	if s == nil || s.closed {
		return 0
	}
	return uint64(len(s.inSet))
}
func (s *satSolverUintSet) MembershipCapacity() uint64 {
	if s == nil || s.closed {
		return 0
	}
	return uint64(cap(s.inSet))
}
func (s *satSolverUintSet) ValuesCapacity() uint64 {
	if s == nil || s.closed {
		return 0
	}
	return uint64(cap(s.values))
}
func (s *satSolverUintSet) Stats() satSolverUintSetStats {
	return satSolverUintSetStats{s.ownerBytes, s.MembershipSize(), s.MembershipCapacity(), uint64(s.Size()), s.ValuesCapacity()}
}
func (s *satSolverUintSet) Contains(v uint32) bool {
	return s != nil && s.self == s && !s.closed && uint64(v) < uint64(len(s.inSet)) && s.inSet[v] != 0
}
func (s *satSolverUintSet) ReserveMembership(n uint32) error {
	if e := s.open(); e != nil {
		return e
	}
	if uint64(n) <= uint64(len(s.inSet)) {
		return nil
	}
	for uint64(n) > uint64(cap(s.inSet)) {
		c, e := satSolverMembershipNextCapacity(uint64(cap(s.inSet)))
		if e != nil {
			return e
		}
		if e = s.budget.Reserve(c); e != nil {
			return e
		}
		next := make([]byte, len(s.inSet), int(c))
		copy(next[:cap(s.inSet)], s.inSet[:cap(s.inSet)])
		s.budget.Release(uint64(cap(s.inSet)))
		s.inSet = next
	}
	old := len(s.inSet)
	s.inSet = s.inSet[:int(n)]
	clear(s.inSet[old:])
	return nil
}
func (s *satSolverUintSet) growValues() error {
	if len(s.values) < cap(s.values) {
		return nil
	}
	if s.valuesEpoch == ^uint64(0) {
		return arena.ErrSlotLimit
	}
	n, e := satVectorNextCapacity(uint64(cap(s.values)), 4)
	if e != nil {
		return e
	}
	if e = s.budget.Reserve(n * 4); e != nil {
		return e
	}
	next := make([]uint32, len(s.values), int(n))
	copy(next[:cap(s.values)], s.values[:cap(s.values)])
	s.budget.Release(uint64(cap(s.values)) * 4)
	s.values = next
	s.valuesEpoch++
	return nil
}
func (s *satSolverUintSet) Insert(v uint32) error {
	if e := s.ReserveMembership(v + 1); e != nil {
		return e
	} // source unsigned wrap.
	if satStorageSourceDebug && uint64(v) >= uint64(len(s.inSet)) {
		return ErrSATAssertion
	}
	if uint64(v) >= uint64(cap(s.inSet)) {
		return arena.ErrBounds
	}
	all := s.inSet[:cap(s.inSet)]
	if all[v] != 0 {
		return nil
	}
	all[v] = 1 // source publication BEFORE potentially failing list push.
	if e := s.growValues(); e != nil {
		return e
	}
	s.values = append(s.values, v)
	return nil
}
func (s *satSolverUintSet) Remove(v uint32) error {
	if e := s.open(); e != nil {
		return e
	}
	if !s.Contains(v) {
		return nil
	}
	s.inSet[v] = 0
	i := 0
	for i < len(s.values) && s.values[i] != v {
		i++
	}
	if satStorageSourceDebug && i >= len(s.values) {
		return ErrSATAssertion
	}
	if len(s.values) == 0 || i >= cap(s.values) {
		return arena.ErrBounds
	}
	all := s.values[:cap(s.values)]
	all[i] = s.values[len(s.values)-1]
	s.values = s.values[:len(s.values)-1]
	return nil
}
func (s *satSolverUintSet) At(i uint32) (uint32, error) {
	if e := s.open(); e != nil {
		return 0, e
	}
	if satStorageSourceDebug && uint64(i) >= uint64(len(s.values)) {
		return 0, ErrSATAssertion
	}
	if uint64(i) >= uint64(cap(s.values)) {
		return 0, arena.ErrBounds
	}
	return s.values[:cap(s.values)][i], nil
}
func (s *satSolverUintSet) Erase() (uint32, error) {
	if e := s.open(); e != nil {
		return 0, e
	}
	if len(s.values) == 0 {
		if satStorageSourceDebug {
			return 0, ErrSATAssertion
		}
		return 0, arena.ErrBounds
	}
	v := s.values[len(s.values)-1]
	s.values = s.values[:len(s.values)-1]
	if satStorageSourceDebug && uint64(v) >= uint64(len(s.inSet)) {
		return v, ErrSATAssertion
	}
	if uint64(v) >= uint64(cap(s.inSet)) {
		return v, arena.ErrBounds
	}
	s.inSet[:cap(s.inSet)][v] = 0
	return v, nil
}
func (s *satSolverUintSet) Reset() error {
	if e := s.open(); e != nil {
		return e
	}
	s.values = s.values[:0]
	s.inSet = s.inSet[:0]
	return nil
}
func (s *satSolverUintSet) clearBacking() {
	s.budget.Release(uint64(cap(s.values))*4 + uint64(cap(s.inSet)))
	s.values = nil
	s.inSet = nil
}
func (s *satSolverUintSet) Finalize() error {
	if e := s.open(); e != nil {
		return e
	}
	if s.valuesEpoch == ^uint64(0) {
		return arena.ErrSlotLimit
	}
	s.clearBacking()
	s.valuesEpoch++
	return nil
}

// Both source member assignments destroy their old array before allocating
// exact source capacity. A failed first copy retains the old values array;
// a failed second copy retains the completed new membership vector.
func (s *satSolverUintSet) Assign(other *satSolverUintSet) error {
	if e := s.open(); e != nil {
		return e
	}
	if e := other.open(); e != nil {
		return e
	}
	if s == other {
		return nil
	}
	s.budget.Release(uint64(cap(s.inSet)))
	s.inSet = nil
	if cap(other.inSet) != 0 {
		if e := s.budget.Reserve(uint64(cap(other.inSet))); e != nil {
			return e
		}
		s.inSet = make([]byte, len(other.inSet), cap(other.inSet))
		copy(s.inSet, other.inSet)
	}
	if s.valuesEpoch == ^uint64(0) {
		return arena.ErrSlotLimit
	}
	s.budget.Release(uint64(cap(s.values)) * 4)
	s.values = nil
	s.valuesEpoch++
	if cap(other.values) != 0 {
		if e := s.budget.Reserve(uint64(cap(other.values)) * 4); e != nil {
			return e
		}
		s.values = make([]uint32, len(other.values), cap(other.values))
		copy(s.values, other.values)
	}
	return nil
}
func (s *satSolverUintSet) Clone(b *arena.Budget) (*satSolverUintSet, error) {
	if e := s.open(); e != nil {
		return nil, e
	}
	out, e := newSATSolverUintSet(b)
	if e != nil {
		return nil, e
	}
	if e = out.Assign(s); e != nil {
		_ = out.Close()
		return nil, e
	}
	return out, nil
}
func (s *satSolverUintSet) Intersect(other *satSolverUintSet) error {
	if e := s.open(); e != nil {
		return e
	}
	if e := other.open(); e != nil {
		return e
	}
	j := 0
	for _, v := range s.values {
		if other.Contains(v) {
			s.values[j] = v
			j++
		} else {
			if satStorageSourceDebug && uint64(v) >= uint64(len(s.inSet)) {
				return ErrSATAssertion
			}
			if uint64(v) >= uint64(cap(s.inSet)) {
				return arena.ErrBounds
			}
			s.inSet[:cap(s.inSet)][v] = 0
		}
	}
	s.values = s.values[:j]
	return nil
}
func (s *satSolverUintSet) Union(other *satSolverUintSet) error {
	if e := s.open(); e != nil {
		return e
	}
	if e := other.open(); e != nil {
		return e
	}
	for i := 0; i < len(other.values); i++ {
		if e := s.Insert(other.values[i]); e != nil {
			return e
		}
	}
	return nil
}
func (s *satSolverUintSet) Close() error {
	if s == nil || s.self != s {
		return ErrSATOwner
	}
	if s.closed {
		return nil
	}
	s.clearBacking()
	s.budget.Release(s.ownerBytes)
	s.ownerBytes = 0
	s.closed = true
	return nil
}

// Source begin/end are const unsigned pointer VALUES. Only actual values
// backing relocation/free invalidates this adapter. Logical Reset/pop leaves
// scalar retained data alive and the captured endpoint unchanged.
type satSolverUintSetIterator struct {
	owner      *satSolverUintSet
	epoch      uint64
	index, end uint32
}

func (s *satSolverUintSet) Iter() (satSolverUintSetIterator, error) {
	if e := s.open(); e != nil {
		return satSolverUintSetIterator{}, e
	}
	return satSolverUintSetIterator{s, s.valuesEpoch, 0, uint32(len(s.values))}, nil
}
func (i *satSolverUintSetIterator) Next() (uint32, bool, error) {
	if i == nil || i.owner == nil {
		return 0, false, ErrSATOwner
	}
	if e := i.owner.open(); e != nil {
		return 0, false, e
	}
	if i.epoch != i.owner.valuesEpoch {
		return 0, false, arena.ErrInvalidSlot
	}
	if i.index >= i.end {
		return 0, false, nil
	}
	if uint64(i.index) >= uint64(cap(i.owner.values)) {
		return 0, false, arena.ErrBounds
	}
	v := i.owner.values[:cap(i.owner.values)][i.index]
	i.index++
	return v, true, nil
}

// Source SZ arithmetic is uint32 even on LP64. Char backing permits
// capacities where 3*old wraps; preserve that overflow before host allocation.
func satSolverMembershipNextCapacity(old uint64) (uint64, error) {
	if old > uint64(^uint32(0)) {
		return 0, arena.ErrBounds
	}
	n := uint32(2)
	if old != 0 {
		n = (3*uint32(old) + 1) >> 1
		if uint64(n) <= old || n+8 <= uint32(old)+8 {
			return 0, arena.ErrBounds
		}
	}
	if uint64(n) > uint64(^uint(0)>>1) {
		return 0, arena.ErrBounds
	}
	return uint64(n), nil
}
