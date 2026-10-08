// Source: src/smt/params/qi_params.cpp:43-64; src/util/params.cpp:928-930.
package util

import (
	"bytes"
	"math"
	"strings"

	"github.com/glycerine/g3/pkg/arena"
)

// OwnedString is a checked Go text adapter for existing project APIs. Its bytes
// are Go-managed; neither storage nor growth follows a C++ string implementation.
// Budget is retained only for compatibility with callers' numeric contexts.
type OwnedString struct {
	Self               *OwnedString
	Budget             *arena.Budget
	External           []byte
	Length, Generation uint64
	Ready, Closed      bool
	goBuilder          *strings.Builder
}

type OwnedStringView struct {
	Owner              *OwnedString
	Generation, Length uint64
}

func (v OwnedStringView) Len() uint64 { return v.Length }
func (s *OwnedString) valid() bool    { return s != nil && s.Self == s && s.Ready && !s.Closed }
func (v OwnedStringView) Byte(i uint64) (byte, error) {
	if !v.Owner.valid() || v.Generation != v.Owner.Generation {
		return 0, arena.ErrInvalidSlot
	}
	if i >= v.Length {
		return 0, arena.ErrBounds
	}
	if v.Owner.goBuilder != nil {
		text := v.Owner.goBuilder.String()
		if i >= uint64(len(text)) {
			return 0, arena.ErrBounds
		}
		return text[i], nil
	}
	if i >= uint64(len(v.Owner.External)) {
		return 0, arena.ErrBounds
	}
	return v.Owner.External[i], nil
}
func (v OwnedStringView) String() (string, error) {
	if !v.Owner.valid() || v.Generation != v.Owner.Generation {
		return "", arena.ErrInvalidSlot
	}
	if v.Owner.goBuilder != nil {
		text := v.Owner.goBuilder.String()
		if v.Length > uint64(len(text)) {
			return "", arena.ErrBounds
		}
		return text[:v.Length], nil
	}
	if v.Length > uint64(len(v.Owner.External)) {
		return "", arena.ErrBounds
	}
	return string(v.Owner.External[:v.Length]), nil
}

// CStringByte supplies a virtual terminator to project interfaces that consume
// NUL-terminated bytes; no terminator is stored in the Go slice.
func (s *OwnedString) CStringByte(i uint64) (byte, error) {
	if !s.valid() {
		return 0, arena.ErrInvalidSlot
	}
	if i == s.Length {
		return 0, nil
	}
	return s.View().Byte(i)
}

// StorageBytes reports numeric-arena storage. Go owns this adapter's memory.
func (s *OwnedString) StorageBytes() uint64 { return 0 }

func paramsStringText(v ParamsStringView) (string, error) {
	if v.Mode == 1 && v.PresentField {
		text := v.literal
		if i := strings.IndexByte(text, 0); i >= 0 {
			text = text[:i]
		}
		return text, nil
	}
	var text strings.Builder
	for i := uint64(0); ; i++ {
		ch, e := v.Byte(i)
		if e != nil {
			return "", e
		}
		if ch == 0 {
			return text.String(), nil
		}
		if i >= uint64(^uint(0)>>1)-1 {
			return "", arena.ErrBounds
		}
		text.WriteByte(ch)
	}
}
func (s *OwnedString) initialize(b *arena.Budget, data []byte) error {
	if s == nil || (s.Self != nil && !s.Closed) {
		return ErrParamsPrecondition
	}
	generation := uint64(1)
	if s.Self != nil {
		if s.Generation == math.MaxUint64 {
			return arena.ErrBounds
		}
		generation = s.Generation + 1
	}
	*s = OwnedString{Budget: b, External: data, Length: uint64(len(data)), Generation: generation, Ready: true}
	s.Self = s
	return nil
}
func (s *OwnedString) Construct(b *arena.Budget, v ParamsStringView) error {
	if s == nil || (s.Self != nil && !s.Closed) {
		return ErrParamsPrecondition
	}
	text, e := paramsStringText(v)
	if e != nil {
		return e
	}
	return s.initialize(b, []byte(text))
}
func (s *OwnedString) replace(data []byte) error {
	if !s.valid() || s.goBuilder != nil {
		return ErrParamsPrecondition
	}
	if s.Generation == math.MaxUint64 {
		return arena.ErrBounds
	}
	s.External = data
	s.Length = uint64(len(data))
	s.Generation++
	return nil
}
func (s *OwnedString) Assign(v ParamsStringView) error {
	if !s.valid() || s.goBuilder != nil {
		return ErrParamsPrecondition
	}
	text, e := paramsStringText(v)
	if e != nil {
		return e
	}
	return s.replace([]byte(text))
}
func (s *OwnedString) Put(i uint64, ch byte) {
	if !s.valid() || s.goBuilder != nil {
		return
	}
	if i < uint64(len(s.External)) {
		s.External[i] = ch
	}
}
func (s *OwnedString) SourceDestroy() {
	if s == nil || s.Self != s || s.Closed {
		return
	}
	s.External = nil
	s.goBuilder = nil
	s.Closed = true
}
func (s *OwnedString) OwnAbort()             { s.SourceDestroy() }
func (s *OwnedString) View() OwnedStringView { return OwnedStringView{s, s.Generation, s.Length} }
func (s *OwnedString) Append(v OwnedStringView) error {
	if !s.valid() || s.goBuilder != nil {
		return ErrParamsPrecondition
	}
	if v.Length >= uint64(^uint(0)>>1)-s.Length {
		return arena.ErrBounds
	}
	if v.Length == 0 {
		return nil
	}
	text, e := v.String()
	if e != nil {
		return e
	}
	if s.Generation == math.MaxUint64 {
		return arena.ErrBounds
	}
	s.External = append(s.External, text...)
	s.Length = uint64(len(s.External))
	s.Generation++
	return nil
}
func (s *OwnedString) AppendCString(v ParamsStringView) error {
	if !s.valid() || s.goBuilder != nil {
		return ErrParamsPrecondition
	}
	text, e := paramsStringText(v)
	if e != nil {
		return e
	}
	if uint64(len(text)) >= uint64(^uint(0)>>1)-s.Length || s.Generation == math.MaxUint64 {
		return arena.ErrBounds
	}
	s.External = append(s.External, text...)
	s.Length = uint64(len(s.External))
	s.Generation++
	return nil
}
func (s *OwnedString) CopyConstruct(b *arena.Budget, src *OwnedString) error {
	if !src.valid() {
		return ErrParamsOwner
	}
	text, e := src.View().String()
	if e != nil {
		return e
	}
	return s.initialize(b, []byte(text))
}
func (s *OwnedString) CopyAssign(src *OwnedString) error {
	if !s.valid() {
		return ErrParamsPrecondition
	}
	if !src.valid() {
		return ErrParamsOwner
	}
	if s == src {
		return nil
	}
	text, e := src.View().String()
	if e != nil {
		return e
	}
	return s.replace([]byte(text))
}
func (s *OwnedString) Matches(text string) bool {
	if !s.valid() {
		return false
	}
	actual, e := s.View().String()
	return e == nil && actual == text
}
func (s *OwnedString) Reserve(count uint64) error {
	if !s.valid() || s.goBuilder != nil {
		return ErrParamsPrecondition
	}
	if count >= uint64(^uint(0)>>1) {
		return arena.ErrBounds
	}
	if count > uint64(cap(s.External)) {
		buffer := bytes.NewBuffer(s.External)
		buffer.Grow(int(count) - buffer.Len())
		s.External = buffer.Bytes()
	}
	return nil
}
func (s *OwnedString) MoveAssign(src *OwnedString) error {
	if !s.valid() || !src.valid() || s.goBuilder != nil || src.goBuilder != nil {
		return ErrParamsPrecondition
	}
	if s == src {
		return nil
	}
	if s.Generation == math.MaxUint64 || src.Generation == math.MaxUint64 {
		return arena.ErrBounds
	}
	s.External, src.External = src.External, nil
	s.Length, src.Length = src.Length, 0
	s.Generation++
	src.Generation++
	return nil
}
func (s *OwnedString) ConstructCStringConcat(b *arena.Budget, prefix ParamsStringView, rhs OwnedStringView) error {
	first, e := paramsStringText(prefix)
	if e != nil {
		return e
	}
	last, e := rhs.String()
	if e != nil {
		return e
	}
	return s.initialize(b, []byte(first+last))
}
func (s *OwnedString) ConstructRepeated(b *arena.Budget, count uint64, ch byte) error {
	if s == nil || (s.Self != nil && !s.Closed) {
		return ErrParamsPrecondition
	}
	if count >= uint64(^uint(0)>>1) {
		return arena.ErrBounds
	}
	return s.initialize(b, bytes.Repeat([]byte{ch}, int(count)))
}
func (s *OwnedString) Reverse() error {
	if !s.valid() || s.goBuilder != nil {
		return ErrParamsPrecondition
	}
	for left, right := 0, len(s.External)-1; left < right; left, right = left+1, right-1 {
		s.External[left], s.External[right] = s.External[right], s.External[left]
	}
	return nil
}

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
