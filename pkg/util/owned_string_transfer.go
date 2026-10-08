// Source: selected libstdc++ basic_string.h:858-921,3570-3588;
// basic_string.tcc:305-321; src/util/mpf.cpp:1635-1693.
package util

import (
	"github.com/glycerine/g3/pkg/arena"
	"math"
)

// Reserve follows the selected nonshrinking reserve/_M_create operation.
func (s *OwnedString) Reserve(count uint64) error {
	if s == nil || s.Self != s || !s.Ready || s.Closed || s.Pending != nil {
		return ErrParamsPrecondition
	}
	if count <= s.Capacity {
		return nil
	}
	max := uint64(^uint(0) >> 1)
	if count >= max || s.Generation == math.MaxUint64 {
		return arena.ErrBounds
	}
	capacity := count
	if s.Capacity <= (max-1)/2 && capacity < 2*s.Capacity {
		capacity = 2 * s.Capacity
	}
	if e := s.Budget.Reserve(capacity + 1); e != nil {
		return e
	}
	next := make([]byte, int(capacity+1))
	if s.External != nil {
		copy(next, s.External[:s.Length+1])
	} else {
		copy(next, s.Inline[:s.Length+1])
	}
	s.Budget.Release(uint64(len(s.External)))
	s.External = next
	s.Capacity = capacity
	s.Generation++
	return nil
}

// MoveAssign uses the equal-allocator string move: a short source copies into
// existing storage; a heap source transfers its backing and takes the old
// destination backing, if any. Both objects remain valid and the source clears.
func (s *OwnedString) MoveAssign(src *OwnedString) error {
	if s == nil || src == nil || s.Self != s || src.Self != src || !s.Ready || !src.Ready || s.Closed || src.Closed || s.Pending != nil || src.Pending != nil || s.Budget != src.Budget {
		return ErrParamsPrecondition
	}
	if s.Generation == math.MaxUint64 || src.Generation == math.MaxUint64 {
		return arena.ErrBounds
	}
	if s == src {
		s.Length = 0
		s.Put(0, 0)
		s.Generation++
		return nil
	}
	if src.External == nil {
		for i := uint64(0); i < src.Length; i++ {
			s.Put(i, src.Inline[i])
		}
		s.Length = src.Length
		s.Put(s.Length, 0)
	} else {
		s.External, src.External = src.External, s.External
		oldCapacity := s.Capacity
		s.Capacity = src.Capacity
		s.Length = src.Length
		src.Capacity = 15
		if src.External != nil {
			src.Capacity = oldCapacity
		}
	}
	src.Length = 0
	src.Put(0, 0)
	s.Generation++
	src.Generation++
	return nil
}

// ConstructCStringConcat models const-char-pointer + lvalue string: default
// construction, reserve for both operands, then two counted appends. The caller
// owns any reached constructor prefix if reserve fails.
func (s *OwnedString) ConstructCStringConcat(b *arena.Budget, prefix ParamsStringView, rhs OwnedStringView) error {
	if s == nil || s.Self != nil {
		return ErrParamsPrecondition
	}
	n, e := prefix.Length()
	if e != nil {
		return e
	}
	max := uint64(^uint(0) >> 1)
	if n >= max || rhs.Length >= max-n {
		return arena.ErrBounds
	}
	if e := s.Construct(b, LiteralParamsStringView("")); e != nil {
		return e
	}
	if e := s.Reserve(n + rhs.Length); e != nil {
		return e
	}
	if e := s.AppendCString(prefix); e != nil {
		return e
	}
	return s.Append(rhs)
}
