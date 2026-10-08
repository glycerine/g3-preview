// Source: selected libstdc++ bits/basic_string.h:586-594,3208-3211;
// bits/basic_string.tcc:222-255,664-677; src/util/mpf.cpp:207-218.
package util

import "github.com/glycerine/g3/pkg/arena"

// OwnedStringNPos is the selected uint64 size-type string's absent-position
// sentinel. FindByte and ConstructSubstring operate on counted bytes, including NUL.
const OwnedStringNPos = ^uint64(0)

func (s *OwnedString) countedLength() (uint64, error) {
	if s == nil || s.Self != s || !s.Ready || s.Closed {
		return 0, ErrParamsOwner
	}
	// The checked adapter guards actual backing spans before any byte access.
	if s.Length > s.Capacity {
		return 0, arena.ErrBounds
	}
	if s.External != nil {
		if s.Length >= uint64(len(s.External)) {
			return 0, arena.ErrBounds
		}
	} else if s.Length >= uint64(len(s.Inline)) {
		return 0, arena.ErrBounds
	}
	return s.Length, nil
}

// FindByte follows the selected string::find(char,pos) and bounded traits::find.
// A start at or beyond the counted end returns npos without reading the NUL.
func (s *OwnedString) FindByte(ch byte, pos uint64) (uint64, error) {
	n, e := s.countedLength()
	if e != nil {
		return OwnedStringNPos, e
	}
	if pos >= n {
		return OwnedStringNPos, nil
	}
	view := s.View()
	for i := pos; i < n; i++ {
		got, e := view.Byte(i)
		if e != nil {
			return OwnedStringNPos, e
		}
		if got == ch {
			return i, nil
		}
	}
	return OwnedStringNPos, nil
}

// ConstructSubstring follows string::substr's position check and returned
// counted constructor. It copies into independent inline15 or exact-capacity
// storage. The caller owns the returned string and any failed constructor prefix.
func (s *OwnedString) ConstructSubstring(b *arena.Budget, src *OwnedString, pos, count uint64) error {
	if s == nil || s.Self != nil {
		return ErrParamsPrecondition
	}
	length, e := src.countedLength()
	if e != nil {
		return e
	}
	// substr checks the position before the returned constructor begins.
	if pos > length {
		return arena.ErrBounds
	}
	n := length - pos
	if n > count {
		n = count
	}
	s.Self = s
	s.Budget = b
	s.Capacity = 15
	s.Generation = 1
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
		ch, e := view.Byte(pos + i)
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
