// Source: src/util/mpz.cpp:1432-1435; selected libstdc++
// bits/sstream.tcc:80-153 and sstream:249-256,440-450.
package util

import (
	"math"
	"strconv"

	"github.com/glycerine/g3/pkg/arena"
)

// OwnedStringBuffer is the selected append-only char basic_stringbuf used by
// mpz's local ostringstream. Position is the put pointer's offset; String.Length
// is the distinct underlying basic_string length. Caller storage holds control
// metadata; only actual string backing is reserved here. Native locale and
// stream object ABI are not claimed by this concrete decimal adapter.
type OwnedStringBuffer struct {
	Self     *OwnedStringBuffer
	String   OwnedString
	Position uint64
}

func (b *OwnedStringBuffer) Construct(budget *arena.Budget) error {
	if b == nil || b.Self != nil {
		return ErrParamsPrecondition
	}
	b.Self = b
	return b.String.Construct(budget, LiteralParamsStringView(""))
}

func (b *OwnedStringBuffer) put(ch byte) error {
	if b == nil || b.Self != b || !b.String.Ready || b.String.Closed || b.String.Pending != nil {
		return ErrParamsPrecondition
	}
	s := &b.String
	if b.Position < s.Capacity {
		s.Put(b.Position, ch)
		b.Position++
		return nil
	}
	max := uint64(^uint(0)>>1) - 1
	if s.Capacity >= max || s.Generation == math.MaxUint64 {
		return arena.ErrBounds
	}
	capacity := max
	if s.Capacity <= max/2 {
		capacity = 2 * s.Capacity
	}
	if capacity < 512 {
		capacity = 512
	}
	if e := s.Budget.Reserve(capacity + 1); e != nil {
		return e
	}
	s.Pending = make([]byte, int(capacity+1))
	old := OwnedStringView{s, s.Generation, b.Position}
	for i := uint64(0); i < b.Position; i++ {
		v, e := old.Byte(i)
		if e != nil {
			return e
		}
		s.Pending[i] = v
	}
	s.Pending[b.Position] = ch
	s.Pending[b.Position+1] = 0
	s.Budget.Release(uint64(len(s.External)))
	s.External, s.Pending = s.Pending, nil
	s.Capacity = capacity
	s.Generation++
	b.Position++
	s.Length = b.Position // overflow's temporary.assign/push_back before swap.
	return nil
}

// Str copies the high-water range into a fresh returned string. libstdc++'s
// str() uses assign on an empty string, so a 16..29 byte result has capacity 30.
func (b *OwnedStringBuffer) Str(dst *OwnedString) error {
	if b == nil || b.Self != b || b.String.Closed || !b.String.Ready || b.String.Pending != nil {
		return ErrParamsPrecondition
	}
	if dst == nil || dst.Self != nil {
		return ErrParamsPrecondition
	}
	if e := dst.Construct(b.String.Budget, LiteralParamsStringView("")); e != nil {
		return e
	}
	return dst.Append(OwnedStringView{&b.String, b.String.Generation, b.Position})
}

func (b *OwnedStringBuffer) OwnAbort() {
	if b != nil && b.Self == b {
		b.String.OwnAbort()
	}
}

// OwnedStringStream supplies only default decimal/C-string insertions required
// by mpz::display. It does not model seeking, locales or arbitrary iostream
// flags. A transported allocator/checked-borrow failure is retained, and no
// later insertion replays a stopped source operation.
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
	if e := s.check(); e != nil {
		return e
	}
	var digits [12]byte
	text := strconv.AppendInt(digits[:0], int64(n), 10)
	for _, ch := range text {
		if e := s.emit(ch); e != nil {
			return e
		}
	}
	return nil
}
func (s *OwnedStringStream) AppendCString(text string) error {
	if e := s.check(); e != nil {
		return e
	}
	for i := 0; i < len(text) && text[i] != 0; i++ {
		if e := s.emit(text[i]); e != nil {
			return e
		}
	}
	return nil
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
