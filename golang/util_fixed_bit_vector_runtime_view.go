// Source: src/util/fixed_bit_vector.h; src/util/fixed_bit_vector.cpp.
// Licensed under the MIT license. Source authors: Nikolaj Bjorner, 2014-9-15;
// Leonardo de Moura, 2006-10-03.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"github.com/glycerine/g3/golang/utilhash"
	"io"
)

// View is concrete call-local provenance. It owns no allocation and does not
// replace a raw source pointer's lifetime with an inferred registry lookup.
type RuntimeFixedBitVectorView struct {
	manager   *RuntimeFixedBitVectorManager
	value     RuntimeFixedBitVector
	raw       RuntimeFixedBitVectorSpan
	rawDomain bool
}

func (m *RuntimeFixedBitVectorManager) Borrow(v RuntimeFixedBitVector) (RuntimeFixedBitVectorView, error) {
	if e := m.open(); e != nil {
		return RuntimeFixedBitVectorView{}, e
	}
	if v.origin != m.origin {
		return RuntimeFixedBitVectorView{}, ErrSATOwner
	}
	return RuntimeFixedBitVectorView{manager: m, value: v}, nil
}
func (v RuntimeFixedBitVectorView) byteAt(i uint64) (byte, error) {
	if v.rawDomain {
		if i >= v.raw.Extent {
			return 0, arena.ErrBounds
		}
		if v.raw.Runtime == nil {
			return 0, ErrSATOwner
		}
		return v.raw.Runtime.Read(v.raw.Block, v.raw.Offset+i)
	}
	return v.manager.readByte(v.value, i)
}
func (v RuntimeFixedBitVectorView) putByte(i uint64, b byte) error {
	if v.rawDomain {
		if i >= v.raw.Extent {
			return arena.ErrBounds
		}
		if v.raw.Runtime == nil {
			return ErrSATOwner
		}
		return v.raw.Runtime.Write(v.raw.Block, v.raw.Offset+i, b)
	}
	return v.manager.writeByte(v.value, i, b)
}
func (v RuntimeFixedBitVectorView) wordAt(i uint32) (uint32, error) {
	if v.rawDomain {
		return v.raw.Word(i)
	}
	return v.manager.Word(v.value, i)
}
func (v RuntimeFixedBitVectorView) putWord(i, w uint32) error {
	if v.rawDomain {
		off := uint64(i) * 4
		if off > v.raw.Extent || v.raw.Extent-off < 4 {
			return arena.ErrBounds
		}
		if v.raw.Runtime == nil {
			return ErrSATOwner
		}
		return v.raw.Runtime.WriteUint32(v.raw.Block, v.raw.Offset+off, w)
	}
	return v.manager.putWord(v.value, i, w)
}
func (v RuntimeFixedBitVectorView) get(i uint32) (bool, error) {
	w, e := v.wordAt(i / 32)
	return w&(uint32(1)<<(i%32)) != 0, e
}
func (v RuntimeFixedBitVectorView) setBool(i uint32, b bool) error {
	w, e := v.wordAt(i / 32)
	if e != nil {
		return e
	}
	var n uint32
	if b {
		n = ^uint32(0)
	}
	w ^= (n ^ w) & (uint32(1) << (i % 32))
	return v.putWord(i/32, w)
}
func (m *RuntimeFixedBitVectorManager) SetView(v RuntimeFixedBitVectorView, i uint32) error {
	if e := m.open(); e != nil {
		return e
	}
	if satStorageSourceDebug && i >= m.numBits {
		return ErrSATAssertion
	}
	w, e := v.wordAt(i / 32)
	if e != nil {
		return e
	}
	return v.putWord(i/32, w|(uint32(1)<<(i%32)))
}
func (m *RuntimeFixedBitVectorManager) UnsetView(v RuntimeFixedBitVectorView, i uint32) error {
	if e := m.open(); e != nil {
		return e
	}
	if satStorageSourceDebug && i >= m.numBits {
		return ErrSATAssertion
	}
	w, e := v.wordAt(i / 32)
	if e != nil {
		return e
	}
	return v.putWord(i/32, w & ^(uint32(1)<<(i%32)))
}
func (m *RuntimeFixedBitVectorManager) SetBoolView(v RuntimeFixedBitVectorView, i uint32, b bool) error {
	if e := m.open(); e != nil {
		return e
	}
	if satStorageSourceDebug && i >= m.numBits {
		return ErrSATAssertion
	}
	return v.setBool(i, b)
}
func (m *RuntimeFixedBitVectorManager) CopyViews(dst, src RuntimeFixedBitVectorView) error {
	if e := m.open(); e != nil {
		return e
	}
	for i := uint64(0); i < uint64(m.numBytes); i++ {
		b, e := src.byteAt(i)
		if e != nil {
			return e
		}
		if e = dst.putByte(i, b); e != nil {
			return e
		}
	}
	return nil
}
func (m *RuntimeFixedBitVectorManager) Fill0View(v RuntimeFixedBitVectorView) error {
	return m.fillView(v, 0)
}
func (m *RuntimeFixedBitVectorManager) Fill1View(v RuntimeFixedBitVectorView) error {
	return m.fillView(v, 255)
}
func (m *RuntimeFixedBitVectorManager) ResetView(v RuntimeFixedBitVectorView) error {
	return m.Fill0View(v)
}
func (m *RuntimeFixedBitVectorManager) fillView(v RuntimeFixedBitVectorView, b byte) error {
	if e := m.open(); e != nil {
		return e
	}
	for i := uint64(0); i < uint64(m.numBytes); i++ {
		if e := v.putByte(i, b); e != nil {
			return e
		}
	}
	return nil
}
func (m *RuntimeFixedBitVectorManager) UnionViews(dst, src RuntimeFixedBitVectorView) error {
	return m.combineViews(dst, src, true)
}
func (m *RuntimeFixedBitVectorManager) IntersectViews(dst, src RuntimeFixedBitVectorView) error {
	return m.combineViews(dst, src, false)
}
func (m *RuntimeFixedBitVectorManager) combineViews(dst, src RuntimeFixedBitVectorView, or bool) error {
	if e := m.open(); e != nil {
		return e
	}
	for i := uint32(0); i < m.numWords; i++ {
		a, e := dst.wordAt(i)
		if e != nil {
			return e
		}
		b, e := src.wordAt(i)
		if e != nil {
			return e
		}
		if or {
			a |= b
		} else {
			a &= b
		}
		if e = dst.putWord(i, a); e != nil {
			return e
		}
	}
	return nil
}
func (m *RuntimeFixedBitVectorManager) NegView(v RuntimeFixedBitVectorView) error {
	if e := m.open(); e != nil {
		return e
	}
	for i := uint32(0); i < m.numWords; i++ {
		w, e := v.wordAt(i)
		if e != nil {
			return e
		}
		if e = v.putWord(i, ^w); e != nil {
			return e
		}
	}
	return nil
}
func (m *RuntimeFixedBitVectorManager) LastWordView(v RuntimeFixedBitVectorView) (uint32, error) {
	if e := m.open(); e != nil {
		return 0, e
	}
	if m.numWords == 0 {
		return 0, nil
	}
	w, e := v.wordAt(m.numWords - 1)
	return w & m.mask, e
}
func (m *RuntimeFixedBitVectorManager) EqualViews(a, b RuntimeFixedBitVectorView) (bool, error) {
	if e := m.open(); e != nil {
		return false, e
	}
	if a.address() == b.address() {
		return true, nil
	}
	if m.numWords == 0 {
		return true, nil
	}
	for i := uint32(0); i < m.numWords-1; i++ {
		x, e := a.wordAt(i)
		if e != nil {
			return false, e
		}
		y, e := b.wordAt(i)
		if e != nil {
			return false, e
		}
		if x != y {
			return false, nil
		}
	}
	x, e := m.LastWordView(a)
	if e != nil {
		return false, e
	}
	y, e := m.LastWordView(b)
	return x == y, e
}
func (m *RuntimeFixedBitVectorManager) ContainsViews(a, b RuntimeFixedBitVectorView) (bool, error) {
	if e := m.open(); e != nil {
		return false, e
	}
	if m.numWords == 0 {
		return true, nil
	}
	for i := uint32(0); i < m.numWords-1; i++ {
		x, e := a.wordAt(i)
		if e != nil {
			return false, e
		}
		y, e := b.wordAt(i)
		if e != nil {
			return false, e
		}
		z, e := b.wordAt(i)
		if e != nil {
			return false, e
		}
		if x&y != z {
			return false, nil
		}
	}
	y, e := m.LastWordView(b)
	if e != nil {
		return false, e
	}
	x, e := m.LastWordView(a)
	return x&y == y, e
}
func (m *RuntimeFixedBitVectorManager) SetRangeViews(dst, src RuntimeFixedBitVectorView, hi, lo uint32) error {
	if e := m.open(); e != nil {
		return e
	}
	if satStorageSourceDebug && (lo > hi || hi >= m.numBits) {
		return ErrSATAssertion
	}
	count := hi - lo + 1
	var start uint32
	if lo%32 == 0 {
		n := count / 32
		for i := uint32(0); i < n; i++ {
			w, e := src.wordAt(i)
			if e != nil {
				return e
			}
			if e = dst.putWord(lo/32+i, w); e != nil {
				return e
			}
		}
		start = n * 32
	}
	for i := start; i < count; i++ {
		b, e := src.get(i)
		if e != nil {
			return e
		}
		if e = dst.setBool(lo+i, b); e != nil {
			return e
		}
	}
	return nil
}
func (m *RuntimeFixedBitVectorManager) HashView(v RuntimeFixedBitVectorView) (uint32, error) {
	if e := m.open(); e != nil {
		return 0, e
	}
	n := m.numBits / 8
	if n == 0 {
		return utilhash.RuntimeBytesBounded(nil, memory.Block{}, 0, 0, 0, m.numBits)
	}
	if v.rawDomain {
		return utilhash.RuntimeBytesBounded(v.raw.Runtime, v.raw.Block, v.raw.Offset, v.raw.Extent, n, m.numBits)
	}
	if e := v.manager.open(); e != nil {
		return 0, e
	}
	if v.value.origin != v.manager.origin {
		return 0, ErrSATOwner
	}
	if v.value.inline {
		return utilhash.InlineWordBytes(&v.manager.zero, &v.manager.zeroInitialized, 4, n, m.numBits)
	}
	s, e := v.manager.Capture(v.value)
	if e != nil {
		return 0, e
	}
	return utilhash.RuntimeBytesBounded(s.Runtime, s.Block, s.Offset, s.Extent, n, m.numBits)
}
func (m *RuntimeFixedBitVectorManager) DisplayView(w io.Writer, v RuntimeFixedBitVectorView) error {
	if e := m.open(); e != nil {
		return e
	}
	for i := m.numBits; i > 0; {
		i--
		b, e := v.get(i)
		if e != nil {
			return e
		}
		c := byte('0')
		if b {
			c = '1'
		}
		n, e := w.Write([]byte{c})
		if e != nil {
			return e
		}
		if n != 1 {
			return io.ErrShortWrite
		}
	}
	return nil
}

// View captures the raw source pointer fields by value. It is non-owning and
// bypasses only the dead allocator locator; actual provider/extent guards remain.
func (s RuntimeFixedBitVectorSpan) View() RuntimeFixedBitVectorView {
	return RuntimeFixedBitVectorView{raw: s, rawDomain: true}
}

// Address is the selected physical provider/token/offset binding acquired at
// allocation or explicit raw capture. No extent, byte load, ObjectRef lookup,
// or generation validation is performed by source address equality.
type fixedRuntimeAddress struct {
	runtime              *memory.Runtime
	block                memory.Block
	offset, inlineOrigin uint64
	inline               bool
}

func (v RuntimeFixedBitVectorView) address() fixedRuntimeAddress {
	if v.rawDomain {
		return fixedRuntimeAddress{runtime: v.raw.Runtime, block: v.raw.Block, offset: v.raw.Offset}
	}
	if v.value.inline {
		return fixedRuntimeAddress{inline: true, inlineOrigin: v.value.origin}
	}
	var r *memory.Runtime
	if v.manager != nil {
		r = v.manager.runtime
	}
	return fixedRuntimeAddress{runtime: r, block: v.value.rawBlock, offset: v.value.rawOffset}
}
