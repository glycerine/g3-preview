// Source: src/util/mpf.cpp:1519-1569; selected libstdc++ basic_string.tcc:326-344,414-429.
// Source author: Christoph Wintersteiger. Concrete library profile is a Go adapter.
// Licensed under the MIT license; see LICENSE.txt.
package util

import (
	"math"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
)

// The actual res and ss.str temporary have distinct inline/external storage.
// This used cold transport header is charged once; NUM scope binding follows
// the exact numerical invocation seam rather than owning numerals here.
type ArbitraryFloatFormatText struct {
	Numeric                  ArbitraryFloatFormatInvocation
	Self                     *ArbitraryFloatFormatText
	budget                   *arena.Budget
	Result, Temporary        OwnedString
	Stream                   mpfFormatStream
	Header                   uint64
	closed, SourceEndedField bool
}
type mpfFormatStream struct {
	*NumericTextStream
	owner       *ArbitraryFloatFormatText
	ClosedField bool
}
type mpfFormatCString struct {
	owner      *ArbitraryFloatFormatText
	generation uint64
}

func NewArbitraryFloatFormatText(b *arena.Budget) (*ArbitraryFloatFormatText, error) {
	if b == nil {
		return nil, ErrNumericStreamConfig
	}
	n := uint64(reflect.TypeOf(ArbitraryFloatFormatText{}).Size())
	if e := b.Reserve(n); e != nil {
		return nil, e
	}
	t := &ArbitraryFloatFormatText{budget: b, Header: n}
	t.Self = t
	t.Stream.owner = t
	if e := t.Result.Construct(b, LiteralParamsStringView("")); e != nil {
		b.Release(n)
		return nil, e
	}
	return t, nil
}
func (t *ArbitraryFloatFormatText) NumericInvocation() *ArbitraryFloatFormatInvocation {
	if t == nil || t.Self != t || t.closed {
		return nil
	}
	return &t.Numeric
}
func (t *ArbitraryFloatFormatText) open() error {
	if t == nil || t.Self != t {
		return ErrNumericStreamOwner
	}
	if t.closed || t.SourceEndedField {
		return arena.ErrClosed
	}
	return nil
}
func (t *ArbitraryFloatFormatText) AssignLiteral(s string) error {
	if e := t.open(); e != nil {
		return e
	}
	return t.Result.Assign(LiteralParamsStringView(s))
}
func (t *ArbitraryFloatFormatText) NewArbitraryFloatFormatStream() (ArbitraryFloatFormatStream, error) {
	if e := t.open(); e != nil {
		return nil, e
	}
	if t.Stream.NumericTextStream != nil {
		return nil, ErrNumericStreamOwner
	}
	s, e := NewNumericTextStream(NumericTextStreamConfig{Budget: t.budget})
	if e != nil {
		return nil, e
	}
	t.Stream.NumericTextStream = s
	return &t.Stream, nil
}
func (s *mpfFormatStream) SourceFinalize() error {
	if s == nil || s.owner == nil || s.owner.Self != s.owner || &s.owner.Stream != s {
		return ErrNumericStreamOwner
	}
	if s.ClosedField {
		return nil
	}
	if s.NumericTextStream == nil {
		return ErrNumericStreamOwner
	}
	if e := s.NumericTextStream.Close(); e != nil {
		return e
	}
	s.ClosedField = true
	return nil
}
func (s *mpfFormatStream) Close() error { return s.SourceFinalize() }

// Source ss.str constructs a separate length-based string, preserving embedded
// NUL bytes. It does not use CString construction or a Go string snapshot.
func (t *ArbitraryFloatFormatText) captureStreamTemporary(s *NumericTextStream) error {
	return CaptureNumericStreamOwnedString(&t.Temporary, s, t.budget)
}

// Same actual ss.str() owned-string constructor, used by numerical callers.
func CaptureNumericStreamOwnedString(q *OwnedString, s *NumericTextStream, b *arena.Budget) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if s.fileMode || s.Buffer == nil {
		return ErrNumericStreamFile
	}
	if q.Self != nil && !q.Closed && (q.Ready || len(q.External) != 0 || len(q.Pending) != 0) {
		return ErrParamsPrecondition
	}
	*q = OwnedString{Budget: b, Capacity: 15, Generation: 1}
	q.Self = q
	n := s.Buffer.State.Pos
	if n > 15 {
		if n >= uint64(^uint(0)>>1) {
			return arena.ErrBounds
		}
		if e := b.Reserve(n + 1); e != nil {
			return e
		}
		q.External = make([]byte, int(n+1))
		q.Capacity = n
	}
	data := s.Buffer.State.Data(s.Buffer.Inline[:])
	for i := uint64(0); i < n; i++ {
		q.Put(i, data[i])
	}
	q.Put(n, 0)
	q.Length = n
	q.Ready = true
	return nil
}
func (t *ArbitraryFloatFormatText) appendTemporary() error {
	dst, src := &t.Result, &t.Temporary
	if !src.Ready || src.Closed || dst.Pending != nil {
		return ErrParamsPrecondition
	}
	if src.Length > math.MaxUint64-dst.Length {
		return arena.ErrBounds
	}
	n := dst.Length + src.Length
	if n <= dst.Capacity {
		for i := uint64(0); i < src.Length; i++ {
			b, e := src.View().Byte(i)
			if e != nil {
				return e
			}
			dst.Put(dst.Length+i, b)
		}
		dst.Put(n, 0)
		dst.Length = n
		return nil
	}
	if n >= uint64(^uint(0)>>1) || dst.Generation == math.MaxUint64 {
		return arena.ErrBounds
	}
	capacity := n
	if capacity < 2*dst.Capacity {
		capacity = 2 * dst.Capacity
	}
	if capacity >= uint64(^uint(0)>>1) {
		return arena.ErrBounds
	}
	if e := t.budget.Reserve(capacity + 1); e != nil {
		return e
	}
	dst.Pending = make([]byte, int(capacity+1))
	for i := uint64(0); i < dst.Length; i++ {
		b, e := dst.View().Byte(i)
		if e != nil {
			return e
		}
		dst.Pending[i] = b
	}
	for i := uint64(0); i < src.Length; i++ {
		b, e := src.View().Byte(i)
		if e != nil {
			return e
		}
		dst.Pending[dst.Length+i] = b
	}
	t.budget.Release(uint64(len(dst.External)))
	dst.External = dst.Pending
	dst.Pending = nil
	dst.Capacity = capacity
	dst.Generation++
	dst.Put(n, 0)
	dst.Length = n
	return nil
}
func (t *ArbitraryFloatFormatText) AppendStreamString(stream ArbitraryFloatFormatStream) error {
	if e := t.open(); e != nil {
		return e
	}
	if stream != &t.Stream || t.Stream.ClosedField || t.Stream.NumericTextStream == nil {
		return ErrNumericStreamOwner
	}
	if e := t.captureStreamTemporary(t.Stream.NumericTextStream); e != nil {
		return e
	}
	e := t.appendTemporary()
	// Completed lexical temporary is destroyed even when append throws.
	t.Temporary.SourceDestroy()
	return e
}
func (t *ArbitraryFloatFormatText) CString() (ArbitraryFloatFormattedCString, error) {
	if e := t.open(); e != nil {
		return ArbitraryFloatFormattedCString{}, e
	}
	return BorrowArbitraryFloatFormattedCString(mpfFormatCString{t, t.Result.Generation}), nil
}
func (v mpfFormatCString) Byte(i uint64) (byte, error) {
	if e := v.owner.open(); e != nil {
		return 0, e
	}
	s := &v.owner.Result
	if v.generation != s.Generation {
		return 0, arena.ErrInvalidSlot
	}
	if i > s.Length {
		return 0, arena.ErrBounds
	}
	if i == s.Length {
		return 0, nil
	}
	return s.View().Byte(i)
}
func (v mpfFormatCString) Length() (uint64, error) {
	for i := uint64(0); ; i++ {
		b, e := v.Byte(i)
		if e != nil {
			return 0, e
		}
		if b == 0 {
			return i, nil
		}
		if i == math.MaxUint64 {
			return 0, arena.ErrBounds
		}
	}
}
func (t *ArbitraryFloatFormatText) SourceFinalize() error {
	if t == nil || t.Self != t {
		return ErrNumericStreamOwner
	}
	if t.SourceEndedField {
		return nil
	}
	if e := t.Numeric.SourceReady(); e != nil {
		return e
	}
	if t.Stream.NumericTextStream != nil && !t.Stream.ClosedField {
		return ErrParamsPrecondition
	}
	t.Result.SourceDestroy()
	t.SourceEndedField = true
	return nil
}
func (t *ArbitraryFloatFormatText) Close() error {
	if t == nil || t.Self != t {
		return ErrNumericStreamOwner
	}
	if t.closed {
		return nil
	}
	if e := t.Numeric.Cleanup(false); e != nil {
		return e
	}
	if e := t.Numeric.Cleanup(true); e != nil {
		return e
	}
	if t.Stream.NumericTextStream != nil {
		if e := t.Stream.Close(); e != nil {
			return e
		}
	}
	t.Temporary.OwnAbort()
	t.Result.OwnAbort()
	if e := t.Numeric.EndContext(); e != nil {
		return e
	}
	t.budget.Release(t.Header)
	t.Header = 0
	t.closed = true
	t.SourceEndedField = true
	return nil
}

// StorageBytes accounts for this concrete transport and its actual storage.
func (t *ArbitraryFloatFormatText) StorageBytes() uint64 {
	n := t.Header + uint64(len(t.Result.External)+len(t.Result.Pending)+len(t.Temporary.External)+len(t.Temporary.Pending))
	if s := t.Stream.NumericTextStream; s != nil {
		v := s.Stats()
		n += v.OwnerBytes + v.ScratchBytes + v.OutputOwnerBytes + v.OutputOverflowBytes
	}
	return n
}
func (t *ArbitraryFloatFormatText) SourceEnded() bool { return t.SourceEndedField }
