// Source: src/util/mpz.cpp:1432-1435.
package util

import (
	"github.com/glycerine/g3/pkg/arena"
	"math"
	"strconv"
	"strings"
)

// OwnedStringBuffer adapts Go's strings.Builder to the numeric output APIs.
// String retains the existing checked-view interface during migration of those
// APIs; its data comes from builder, not from a C++ string storage policy.
// Builder allocations belong to Go and are not charged to the numeric arena.
type OwnedStringBuffer struct {
	Self     *OwnedStringBuffer
	String   OwnedString
	Position uint64
	builder  strings.Builder
}

func (b *OwnedStringBuffer) Construct(budget *arena.Budget) error {
	if b == nil || b.Self != nil {
		return ErrParamsPrecondition
	}
	b.Self = b
	if e := b.String.Construct(budget, LiteralParamsStringView("")); e != nil {
		return e
	}
	b.String.goBuilder = &b.builder
	return nil
}

func (b *OwnedStringBuffer) put(ch byte) error {
	if b == nil || b.Self != b || !b.String.Ready || b.String.Closed {
		return ErrParamsPrecondition
	}
	if b.Position >= uint64(^uint(0)>>1) {
		return arena.ErrBounds
	}
	if e := b.builder.WriteByte(ch); e != nil {
		return e
	}
	b.Position++
	b.String.Length = b.Position
	return nil
}

func (b *OwnedStringBuffer) writeString(text string) error {
	if b == nil || b.Self != b || !b.String.Ready || b.String.Closed {
		return ErrParamsPrecondition
	}
	if uint64(len(text)) > uint64(^uint(0)>>1)-b.Position {
		return arena.ErrBounds
	}
	if _, e := b.builder.WriteString(text); e != nil {
		return e
	}
	b.Position += uint64(len(text))
	b.String.Length = b.Position
	return nil
}

// Str returns text independent of subsequent writes and buffer destruction.
func (b *OwnedStringBuffer) Str(dst *OwnedString) error {
	if b == nil || b.Self != b || b.String.Closed || !b.String.Ready {
		return ErrParamsPrecondition
	}
	if dst == nil || dst.Self != nil {
		return ErrParamsPrecondition
	}
	// CopyConstruct preserves counted bytes, including embedded NULs.
	return dst.CopyConstruct(b.String.Budget, &b.String)
}

func (b *OwnedStringBuffer) OwnAbort() {
	if b != nil && b.Self == b {
		b.String.OwnAbort()
		b.builder.Reset()
	}
}

// OwnedStringStream uses Go decimal formatting and strings.Builder for the
// project’s numeric text output. Checked numeral errors remain sticky.
type OwnedStringStream struct {
	Self   *OwnedStringStream
	Buffer OwnedStringBuffer
	Cause  error
}

func (s *OwnedStringStream) Construct(b *arena.Budget) error {
	if s == nil || s.Self != nil {
		return ErrParamsPrecondition
	}
	s.Self = s
	return s.Buffer.Construct(b)
}
func (s *OwnedStringStream) check() error {
	if s == nil || s.Self != s || s.Buffer.String.Closed {
		return ErrParamsPrecondition
	}
	return s.Cause
}
func (s *OwnedStringStream) emit(ch byte) error {
	e := s.Buffer.put(ch)
	if e != nil {
		s.Cause = e
	}
	return e
}
func (s *OwnedStringStream) AppendInt32(n int32) error {
	return s.AppendInt64(int64(n))
}

// AppendInt64 supplies the signed exponent insertion used by mpf.cpp:1598.
func (s *OwnedStringStream) appendText(text string) error {
	if e := s.check(); e != nil {
		return e
	}
	if e := s.Buffer.writeString(text); e != nil {
		s.Cause = e
		return e
	}
	return nil
}

func (s *OwnedStringStream) AppendInt64(n int64) error {
	if e := s.check(); e != nil {
		return e
	}
	return s.appendText(strconv.FormatInt(n, 10))
}

func (s *OwnedStringStream) AppendCString(text string) error {
	if e := s.check(); e != nil {
		return e
	}
	if i := strings.IndexByte(text, 0); i >= 0 {
		text = text[:i]
	}
	return s.appendText(text)
}
func (s *OwnedStringStream) AppendNumeralCString(v NumeralCStringView) error {
	if e := s.check(); e != nil {
		return e
	}
	for i := uint64(0); ; i++ {
		ch, e := v.Byte(i)
		if e != nil {
			s.Cause = e
			return e
		}
		if ch == 0 {
			return nil
		}
		if e = s.emit(ch); e != nil {
			return e
		}
		if i == math.MaxUint64 {
			s.Cause = arena.ErrBounds
			return s.Cause
		}
	}
}
func (s *OwnedStringStream) Str(dst *OwnedString) error {
	if e := s.check(); e != nil {
		return e
	}
	e := s.Buffer.Str(dst)
	if e != nil {
		s.Cause = e
	}
	return e
}
func (s *OwnedStringStream) OwnAbort() {
	if s != nil && s.Self == s {
		s.Buffer.OwnAbort()
	}
}
