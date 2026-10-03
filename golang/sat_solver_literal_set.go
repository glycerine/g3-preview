// Copyright (c) 2011 Microsoft Corporation. Licensed under the MIT license.
// Concrete sat::literal_set specialization of src/sat/sat_types.h.
package g3

import (
	"reflect"

	"github.com/glycerine/g3/golang/arena"
)

// The two actual cold headers are charged. Membership and ordered entries
// remain the source concrete uint-set representation, with scalar payloads.
type satSolverLiteralSet struct {
	self       *satSolverLiteralSet
	set        *satSolverUintSet
	budget     *arena.Budget
	ownerBytes uint64
	closed     bool
}

func newSATSolverLiteralSet(b *arena.Budget) (*satSolverLiteralSet, error) {
	n := uint64(reflect.TypeOf(satSolverLiteralSet{}).Size())
	if e := b.Reserve(n); e != nil {
		return nil, e
	}
	u, e := newSATSolverUintSet(b)
	if e != nil {
		b.Release(n)
		return nil, e
	}
	s := &satSolverLiteralSet{set: u, budget: b, ownerBytes: n}
	s.self = s
	return s, nil
}
func newSATSolverLiteralSetFromVector(b *arena.Budget, src *satSolverLiteralStore, r satSolverLiteralListRef) (*satSolverLiteralSet, error) {
	s, e := newSATSolverLiteralSet(b)
	if e != nil {
		return nil, e
	}
	if e = s.insertVector(src, r); e != nil {
		_ = s.Close()
		return nil, e
	}
	return s, nil
}
func (s *satSolverLiteralSet) open() error {
	if s == nil || s.self != s {
		return ErrSATOwner
	}
	if s.closed {
		return arena.ErrClosed
	}
	return nil
}
func (s *satSolverLiteralSet) insertVector(src *satSolverLiteralStore, r satSolverLiteralListRef) error {
	n, e := src.Size(r)
	if e != nil {
		return e
	}
	for i := uint32(0); i < n; i++ {
		v, e := src.Read(r, i)
		if e != nil {
			return e
		}
		if e = s.Insert(v); e != nil {
			return e
		}
	}
	return nil
}
func (s *satSolverLiteralSet) AssignVector(src *satSolverLiteralStore, r satSolverLiteralListRef) error {
	if e := s.Reset(); e != nil {
		return e
	}
	return s.insertVector(src, r)
}
func (s *satSolverLiteralSet) Assign(other *satSolverLiteralSet) error {
	if e := s.open(); e != nil {
		return e
	}
	if e := other.open(); e != nil {
		return e
	}
	if s == other {
		return nil
	}
	return s.set.Assign(other.set)
}
func (s *satSolverLiteralSet) Clone(b *arena.Budget) (*satSolverLiteralSet, error) {
	if e := s.open(); e != nil {
		return nil, e
	}
	out, e := newSATSolverLiteralSet(b)
	if e != nil {
		return nil, e
	}
	if e = out.Assign(s); e != nil {
		_ = out.Close()
		return nil, e
	}
	return out, nil
}
func (s *satSolverLiteralSet) Insert(v satLiteral) error {
	if e := s.open(); e != nil {
		return e
	}
	return s.set.Insert(v.index())
}
func (s *satSolverLiteralSet) Remove(v satLiteral) error {
	if e := s.open(); e != nil {
		return e
	}
	return s.set.Remove(v.index())
}
func (s *satSolverLiteralSet) Pop() (satLiteral, error) {
	if e := s.open(); e != nil {
		return 0, e
	}
	v, e := s.set.Erase()
	return satLiteralFromUint(v), e
}
func (s *satSolverLiteralSet) Contains(v satLiteral) bool {
	return s != nil && s.self == s && !s.closed && s.set.Contains(v.index())
}
func (s *satSolverLiteralSet) Size() uint32 {
	if s == nil || s.self != s || s.closed {
		return 0
	}
	return s.set.Size()
}
func (s *satSolverLiteralSet) Empty() bool { return s.Size() == 0 }
func (s *satSolverLiteralSet) Reset() error {
	if e := s.open(); e != nil {
		return e
	}
	return s.set.Reset()
}
func (s *satSolverLiteralSet) Finalize() error {
	if e := s.open(); e != nil {
		return e
	}
	return s.set.Finalize()
}
func (s *satSolverLiteralSet) Intersect(other *satSolverLiteralSet) error {
	if e := s.open(); e != nil {
		return e
	}
	if e := other.open(); e != nil {
		return e
	}
	return s.set.Intersect(other.set)
}
func (s *satSolverLiteralSet) Union(other *satSolverLiteralSet) error {
	if e := s.open(); e != nil {
		return e
	}
	if e := other.open(); e != nil {
		return e
	}
	return s.set.Union(other.set)
}
func (s *satSolverLiteralSet) Close() error {
	if s == nil || s.self != s {
		return ErrSATOwner
	}
	if s.closed {
		return nil
	}
	if e := s.set.Close(); e != nil {
		return e
	}
	s.budget.Release(s.ownerBytes)
	s.ownerBytes = 0
	s.closed = true
	return nil
}

// Source returned literal_vector owns a newly constructed geometric list.
// The caller owns its cold store; failure destroys the unpublished list only.
func (s *satSolverLiteralSet) ToVector(dst *satSolverLiteralStore) (satSolverLiteralListRef, error) {
	if e := s.open(); e != nil {
		return satSolverLiteralListRef{}, e
	}
	r, e := dst.NewList()
	if e != nil {
		return satSolverLiteralListRef{}, e
	}
	it, e := s.Iter()
	if e != nil {
		_ = dst.Free(r)
		return satSolverLiteralListRef{}, e
	}
	for {
		v, ok, e := it.Next()
		if e == nil && !ok {
			return r, nil
		}
		if e == nil {
			e = dst.Push(r, v)
		}
		if e != nil {
			_ = dst.Free(r)
			return satSolverLiteralListRef{}, e
		}
	}
}

// Source iterator wraps a const unsigned pointer value and captures its end.
// Logical changes retain backing; relocation/free invalidates actual access.
type satSolverLiteralSetIterator struct {
	owner       *satSolverLiteralSet
	inner       satSolverUintSetIterator
	nullBacking bool
}

func (s *satSolverLiteralSet) Iter() (satSolverLiteralSetIterator, error) {
	if e := s.open(); e != nil {
		return satSolverLiteralSetIterator{}, e
	}
	it, e := s.set.Iter()
	return satSolverLiteralSetIterator{s, it, cap(s.set.values) == 0}, e
}
func (s *satSolverLiteralSet) Begin() (satSolverLiteralSetIterator, error) { return s.Iter() }
func (s *satSolverLiteralSet) End() (satSolverLiteralSetIterator, error) {
	it, e := s.Iter()
	it.inner.index = it.inner.end
	return it, e
}
func (it *satSolverLiteralSetIterator) Value() (satLiteral, error) {
	if it == nil {
		return 0, ErrSATOwner
	}
	if e := it.owner.open(); e != nil {
		return 0, e
	}
	u := it.inner.owner
	if it.inner.epoch != u.valuesEpoch {
		return 0, arena.ErrInvalidSlot
	}
	if uint64(it.inner.index) >= uint64(cap(u.values)) {
		return 0, arena.ErrBounds
	}
	return satLiteralFromUint(u.values[:cap(u.values)][it.inner.index]), nil
}
func (it *satSolverLiteralSetIterator) Advance() error {
	if it == nil {
		return ErrSATOwner
	}
	if e := it.owner.open(); e != nil {
		return e
	}
	u := it.inner.owner
	if it.inner.epoch != u.valuesEpoch {
		return arena.ErrInvalidSlot
	}
	if uint64(it.inner.index) >= uint64(cap(u.values)) {
		return arena.ErrBounds
	}
	it.inner.index++
	return nil
}
func (it *satSolverLiteralSetIterator) PostAdvance() (satSolverLiteralSetIterator, error) {
	if it == nil {
		return satSolverLiteralSetIterator{}, ErrSATOwner
	}
	previous := *it
	e := it.Advance()
	return previous, e
}

// Comparison has no referent access. These cold scalar captures represent
// backing identities even after release; two source null pointers are equal.
func (it satSolverLiteralSetIterator) Equal(other satSolverLiteralSetIterator) bool {
	if it.nullBacking && other.nullBacking {
		return it.inner.index == other.inner.index
	}
	return !it.nullBacking && !other.nullBacking && it.inner.owner == other.inner.owner && it.inner.epoch == other.inner.epoch && it.inner.index == other.inner.index
}
func (it *satSolverLiteralSetIterator) Next() (satLiteral, bool, error) {
	if it == nil {
		return 0, false, ErrSATOwner
	}
	if e := it.owner.open(); e != nil {
		return 0, false, e
	}
	v, ok, e := it.inner.Next()
	return satLiteralFromUint(v), ok, e
}
