// Source: src/util/mpf.cpp:1519-1569; selected libstdc++ basic_string.tcc:326-344,414-429.
// Source author: Christoph Wintersteiger. Concrete library profile is a Go adapter.
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/mpnum"
	"math"
	"reflect"
)

// The actual res and ss.str temporary have distinct inline/external storage.
// This used cold transport header is charged once; NUM scope binding follows
// the exact numerical invocation seam rather than owning numerals here.
type ArbitraryFloatFormatText struct {
	numeric             mpnum.ArbitraryFloatFormatInvocation
	self                *ArbitraryFloatFormatText
	budget              *arena.Budget
	result, temporary   qiOwnedString
	stream              mpfFormatStream
	header              uint64
	closed, sourceEnded bool
}
type mpfFormatStream struct {
	*NumericTextStream
	owner  *ArbitraryFloatFormatText
	closed bool
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
	t := &ArbitraryFloatFormatText{budget: b, header: n}
	t.self = t
	t.stream.owner = t
	if e := t.result.construct(b, literalParamsStringView("")); e != nil {
		b.Release(n)
		return nil, e
	}
	return t, nil
}
func (t *ArbitraryFloatFormatText) NumericInvocation() *mpnum.ArbitraryFloatFormatInvocation {
	if t == nil || t.self != t || t.closed {
		return nil
	}
	return &t.numeric
}
func (t *ArbitraryFloatFormatText) open() error {
	if t == nil || t.self != t {
		return ErrNumericStreamOwner
	}
	if t.closed || t.sourceEnded {
		return arena.ErrClosed
	}
	return nil
}
func (t *ArbitraryFloatFormatText) AssignLiteral(s string) error {
	if e := t.open(); e != nil {
		return e
	}
	return t.result.assign(literalParamsStringView(s))
}
func (t *ArbitraryFloatFormatText) NewArbitraryFloatFormatStream() (mpnum.ArbitraryFloatFormatStream, error) {
	if e := t.open(); e != nil {
		return nil, e
	}
	if t.stream.NumericTextStream != nil {
		return nil, ErrNumericStreamOwner
	}
	s, e := NewNumericTextStream(NumericTextStreamConfig{Budget: t.budget})
	if e != nil {
		return nil, e
	}
	t.stream.NumericTextStream = s
	return &t.stream, nil
}
func (s *mpfFormatStream) SourceFinalize() error {
	if s == nil || s.owner == nil || s.owner.self != s.owner || &s.owner.stream != s {
		return ErrNumericStreamOwner
	}
	if s.closed {
		return nil
	}
	if s.NumericTextStream == nil {
		return ErrNumericStreamOwner
	}
	if e := s.NumericTextStream.Close(); e != nil {
		return e
	}
	s.closed = true
	return nil
}
func (s *mpfFormatStream) Close() error { return s.SourceFinalize() }

// Source ss.str constructs a separate length-based string, preserving embedded
// NUL bytes. It does not use CString construction or a Go string snapshot.
func (t *ArbitraryFloatFormatText) captureStreamTemporary(s *NumericTextStream) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if s.fileMode || s.buffer == nil {
		return ErrNumericStreamFile
	}
	q := &t.temporary
	if q.self != nil && !q.closed && (q.ready || len(q.external) != 0 || len(q.pending) != 0) {
		return ErrParamsPrecondition
	}
	*q = qiOwnedString{budget: t.budget, capacity: 15, generation: 1}
	q.self = q
	n := s.buffer.state.pos
	if n > 15 {
		if n >= uint64(^uint(0)>>1) {
			return arena.ErrBounds
		}
		if e := t.budget.Reserve(n + 1); e != nil {
			return e
		}
		q.external = make([]byte, int(n+1))
		q.capacity = n
	}
	data := s.buffer.state.data(s.buffer.inline[:])
	for i := uint64(0); i < n; i++ {
		q.put(i, data[i])
	}
	q.put(n, 0)
	q.length = n
	q.ready = true
	return nil
}
func (t *ArbitraryFloatFormatText) appendTemporary() error {
	dst, src := &t.result, &t.temporary
	if !src.ready || src.closed || dst.pending != nil {
		return ErrParamsPrecondition
	}
	if src.length > math.MaxUint64-dst.length {
		return arena.ErrBounds
	}
	n := dst.length + src.length
	if n <= dst.capacity {
		for i := uint64(0); i < src.length; i++ {
			b, e := src.view().Byte(i)
			if e != nil {
				return e
			}
			dst.put(dst.length+i, b)
		}
		dst.put(n, 0)
		dst.length = n
		return nil
	}
	if n >= uint64(^uint(0)>>1) || dst.generation == math.MaxUint64 {
		return arena.ErrBounds
	}
	capacity := n
	if capacity < 2*dst.capacity {
		capacity = 2 * dst.capacity
	}
	if capacity >= uint64(^uint(0)>>1) {
		return arena.ErrBounds
	}
	if e := t.budget.Reserve(capacity + 1); e != nil {
		return e
	}
	dst.pending = make([]byte, int(capacity+1))
	for i := uint64(0); i < dst.length; i++ {
		b, e := dst.view().Byte(i)
		if e != nil {
			return e
		}
		dst.pending[i] = b
	}
	for i := uint64(0); i < src.length; i++ {
		b, e := src.view().Byte(i)
		if e != nil {
			return e
		}
		dst.pending[dst.length+i] = b
	}
	t.budget.Release(uint64(len(dst.external)))
	dst.external = dst.pending
	dst.pending = nil
	dst.capacity = capacity
	dst.generation++
	dst.put(n, 0)
	dst.length = n
	return nil
}
func (t *ArbitraryFloatFormatText) AppendStreamString(stream mpnum.ArbitraryFloatFormatStream) error {
	if e := t.open(); e != nil {
		return e
	}
	if stream != &t.stream || t.stream.closed || t.stream.NumericTextStream == nil {
		return ErrNumericStreamOwner
	}
	if e := t.captureStreamTemporary(t.stream.NumericTextStream); e != nil {
		return e
	}
	e := t.appendTemporary()
	// Completed lexical temporary is destroyed even when append throws.
	t.temporary.sourceDestroy()
	return e
}
func (t *ArbitraryFloatFormatText) CString() (mpnum.ArbitraryFloatFormattedCString, error) {
	if e := t.open(); e != nil {
		return mpnum.ArbitraryFloatFormattedCString{}, e
	}
	return mpnum.BorrowArbitraryFloatFormattedCString(mpfFormatCString{t, t.result.generation}), nil
}
func (v mpfFormatCString) Byte(i uint64) (byte, error) {
	if e := v.owner.open(); e != nil {
		return 0, e
	}
	s := &v.owner.result
	if v.generation != s.generation {
		return 0, arena.ErrInvalidSlot
	}
	if i > s.length {
		return 0, arena.ErrBounds
	}
	if i == s.length {
		return 0, nil
	}
	return s.view().Byte(i)
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
	if t == nil || t.self != t {
		return ErrNumericStreamOwner
	}
	if t.sourceEnded {
		return nil
	}
	if e := t.numeric.SourceReady(); e != nil {
		return e
	}
	if t.stream.NumericTextStream != nil && !t.stream.closed {
		return ErrParamsPrecondition
	}
	t.result.sourceDestroy()
	t.sourceEnded = true
	return nil
}
func (t *ArbitraryFloatFormatText) Close() error {
	if t == nil || t.self != t {
		return ErrNumericStreamOwner
	}
	if t.closed {
		return nil
	}
	if e := t.numeric.Cleanup(false); e != nil {
		return e
	}
	if e := t.numeric.Cleanup(true); e != nil {
		return e
	}
	if t.stream.NumericTextStream != nil {
		if e := t.stream.Close(); e != nil {
			return e
		}
	}
	t.temporary.ownAbort()
	t.result.ownAbort()
	if e := t.numeric.EndContext(); e != nil {
		return e
	}
	t.budget.Release(t.header)
	t.header = 0
	t.closed = true
	t.sourceEnded = true
	return nil
}
