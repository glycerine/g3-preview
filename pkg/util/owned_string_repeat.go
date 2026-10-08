// Source: selected libstdc++ bits/basic_string.tcc:258-272;
// src/util/mpf.cpp:1639-1654,1673,1686.
package util

import "github.com/glycerine/g3/pkg/arena"

// ConstructRepeated follows the selected char string(count,ch) constructor:
// inline15 or exact count capacity, followed by character fill and length/NUL.
// Failure leaves only the actual constructor prefix for OwnAbort.
func (s *OwnedString) ConstructRepeated(b *arena.Budget, count uint64, ch byte) error {
	if s == nil || s.Self != nil {
		return ErrParamsPrecondition
	}
	s.Self = s
	s.Budget = b
	s.Capacity = 15
	s.Generation = 1
	if count > 15 {
		if count >= uint64(^uint(0)>>1) {
			return arena.ErrBounds
		}
		if e := b.Reserve(count + 1); e != nil {
			return e
		}
		s.External = make([]byte, int(count+1))
		s.Capacity = count
	}
	for i := uint64(0); i < count; i++ {
		s.Put(i, ch)
	}
	s.Put(count, 0)
	s.Length = count
	s.Ready = true
	return nil
}

// Reverse swaps the actual counted bytes in place, retaining all borrowed-view
// identities and the trailing NUL. It acquires no new storage.
func (s *OwnedString) Reverse() error {
	if s == nil || s.Self != s || !s.Ready || s.Closed || s.Pending != nil {
		return ErrParamsPrecondition
	}
	if s.Length < 2 {
		return nil
	}
	view := s.View()
	for left, right := uint64(0), s.Length-1; left < right; left, right = left+1, right-1 {
		a, e := view.Byte(left)
		if e != nil {
			return e
		}
		b, e := view.Byte(right)
		if e != nil {
			return e
		}
		s.Put(left, b)
		s.Put(right, a)
	}
	return nil
}
