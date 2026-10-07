// Source: selected libstdc++ basic_string.h/basic_string.tcc and
// src/smt/params/qi_params.cpp:43-64; src/util/params.cpp:928-930.
// Licensed under the MIT license; see LICENSE.txt.
package util

import (
	"math"

	"github.com/glycerine/g3/pkg/arena"
)

// Concrete selected char std::string storage policy: inline15 plus NUL,
// constructor exact capacity, replacement geometric capacity, then dispose /
// pointer-capacity-length publication. This is not a native STL ABI promise.
type OwnedString struct {
	Self                         *OwnedString
	Budget                       *arena.Budget
	Inline                       [16]byte
	External, Pending            []byte
	Length, Capacity, Generation uint64
	Ready, Closed                bool
}

type OwnedStringView struct {
	Owner              *OwnedString
	Generation, Length uint64
}

func (v OwnedStringView) Len() uint64 { return v.Length }

func (v OwnedStringView) Byte(i uint64) (byte, error) {
	if v.Owner == nil || v.Owner.Self != v.Owner || v.Owner.Closed || v.Generation != v.Owner.Generation {
		return 0, arena.ErrInvalidSlot
	}
	if i >= v.Length {
		return 0, arena.ErrBounds
	}
	if v.Owner.External != nil {
		return v.Owner.External[i], nil
	}
	return v.Owner.Inline[i], nil
}

// String is a diagnostic caller-owned Go copy; source getter/display paths
// never call it or acquire ownership through it.
func (v OwnedStringView) String() (string, error) {
	if v.Length > uint64(^uint(0)>>1) {
		return "", arena.ErrBounds
	}
	a := make([]byte, int(v.Length))
	for i := range a {
		b, e := v.Byte(uint64(i))
		if e != nil {
			return "", e
		}
		a[i] = b
	}
	return string(a), nil
}

func (s *OwnedString) Construct(b *arena.Budget, v ParamsStringView) error {
	s.Self = s
	s.Budget = b
	s.Capacity = 15
	s.Generation = 1
	n, e := v.Length()
	if e != nil {
		return e
	}
	if n > 15 {
		if n >= uint64(^uint(0)>>1) {
			return arena.ErrBounds
		}
		if e = b.Reserve(n + 1); e != nil {
			return e
		}
		s.External = make([]byte, int(n+1))
		s.Capacity = n
	}
	for i := uint64(0); i < n; i++ {
		ch, e := v.Byte(i)
		if e != nil {
			s.SourceDestroy()
			return e
		}
		s.Put(i, ch)
	}
	s.Put(n, 0)
	s.Length = n
	s.Ready = true
	return nil
}

func (s *OwnedString) Put(i uint64, ch byte) {
	if s.External != nil {
		s.External[i] = ch
	} else {
		s.Inline[i] = ch
	}
}

func (s *OwnedString) Assign(v ParamsStringView) error {
	if s.Self != s || !s.Ready || s.Closed || s.Pending != nil {
		return ErrParamsPrecondition
	}
	n, e := v.Length()
	if e != nil {
		return e
	}
	if n <= s.Capacity {
		for i := uint64(0); i < n; i++ {
			ch, e := v.Byte(i)
			if e != nil {
				return e
			}
			s.Put(i, ch)
		}
		s.Put(n, 0)
		s.Length = n
		return nil
	}
	if s.Generation == math.MaxUint64 || n >= uint64(^uint(0)>>1) {
		return arena.ErrBounds
	}
	capacity := n
	if capacity < 2*s.Capacity {
		capacity = 2 * s.Capacity
	}
	if capacity >= uint64(^uint(0)>>1) {
		return arena.ErrBounds
	}
	if e = s.Budget.Reserve(capacity + 1); e != nil {
		return e
	}
	s.Pending = make([]byte, int(capacity+1))
	for i := uint64(0); i < n; i++ {
		ch, e := v.Byte(i)
		if e != nil {
			return e
		}
		s.Pending[i] = ch
	}
	s.Pending[n] = 0
	s.Budget.Release(uint64(len(s.External)))
	s.External = s.Pending
	s.Pending = nil
	s.Capacity = capacity
	s.Generation++
	s.Length = n
	return nil
}

func (s *OwnedString) SourceDestroy() {
	if s.Self != s || s.Closed {
		return
	}
	s.Budget.Release(uint64(len(s.External)))
	s.External = nil
	s.Closed = true
}

func (s *OwnedString) OwnAbort() {
	s.SourceDestroy()
	if s.Pending != nil {
		s.Budget.Release(uint64(len(s.Pending)))
		s.Pending = nil
	}
}

func (s *OwnedString) View() OwnedStringView { return OwnedStringView{s, s.Generation, s.Length} }

func (out *NumericTextStream) AppendOwnedString(s *OwnedString) error {
	if e := out.lock(); e != nil {
		return e
	}
	defer out.mu.Unlock()
	if s.Self != s || s.Closed || !s.Ready {
		return ErrParamsOwner
	}
	view := s.View()
	n := s.Length // actual data/size fields acquired before insertion.
	if out.FileBad || out.fileFail {
		return nil
	}
	pad, e := out.fieldPadding(n)
	if e != nil {
		return e
	}
	if out.state.Alignment != NumericLeft {
		if e = out.padding(pad); e != nil {
			return e
		}
	}
	for i := uint64(0); i < n; i++ {
		ch, e := view.Byte(i)
		if e != nil {
			return e
		}
		if e = out.emitByte(ch); e != nil {
			return e
		}
	}
	if out.state.Alignment == NumericLeft {
		return out.padding(pad)
	}
	return nil
}

// Selected libstdc++ basic_string.h:547-552/basic_string.tcc:222-255:
// copy uses the source length directly, exact constructor capacity and the
// actual construction guard dispose if a reached checked byte read fails.
func (s *OwnedString) CopyConstruct(b *arena.Budget, src *OwnedString) error {
	if src == nil || src.Self != src || src.Closed || !src.Ready {
		return ErrParamsOwner
	}
	s.Self = s
	s.Budget = b
	s.Capacity = 15
	s.Generation = 1
	n := src.Length
	if n > 15 {
		if n >= uint64(^uint(0)>>1) {
			return arena.ErrBounds
		}
		if e := b.Reserve(n + 1); e != nil {
			return e
		}
		s.External = make([]byte, int(n+1))
		s.Capacity = n
	}
	view := src.View()
	for i := uint64(0); i < n; i++ {
		ch, e := view.Byte(i)
		if e != nil {
			s.SourceDestroy()
			return e
		}
		s.Put(i, ch)
	}
	s.Put(n, 0)
	s.Length = n
	s.Ready = true
	return nil
}

// Selected libstdc++ basic_string.tcc:278-300 _M_assign allocates, disposes
// old storage, publishes pointer/capacity, then copies bytes and sets length.
// This differs from the existing const-char-pointer replacement source leaf.
func (s *OwnedString) CopyAssign(src *OwnedString) error {
	if s.Self != s || !s.Ready || s.Closed || s.Pending != nil {
		return ErrParamsPrecondition
	}
	if src == nil || src.Self != src || src.Closed || !src.Ready {
		return ErrParamsOwner
	}
	if s == src {
		return nil
	}
	n := src.Length
	if n > s.Capacity {
		if s.Generation == math.MaxUint64 || n >= uint64(^uint(0)>>1) {
			return arena.ErrBounds
		}
		cap := n
		if cap < 2*s.Capacity {
			cap = 2 * s.Capacity
		}
		if cap >= uint64(^uint(0)>>1) {
			return arena.ErrBounds
		}
		if e := s.Budget.Reserve(cap + 1); e != nil {
			return e
		}
		next := make([]byte, int(cap+1))
		s.Budget.Release(uint64(len(s.External)))
		s.External = next
		s.Capacity = cap
		s.Generation++
	}
	view := src.View()
	for i := uint64(0); i < n; i++ {
		ch, e := view.Byte(i)
		if e != nil {
			return e
		}
		s.Put(i, ch)
	}
	s.Put(n, 0)
	s.Length = n
	return nil
}

func (k *OwnedString) Matches(v string) bool {
	if k.Length != uint64(len(v)) {
		return false
	}
	for i := 0; i < len(v); i++ {
		b := k.Inline[0]
		if k.External != nil {
			b = k.External[i]
		} else {
			b = k.Inline[i]
		}
		if b != v[i] {
			return false
		}
	}
	return true
}
