// Source: src/util/mpn.cpp to_string and src/util/mpz.cpp/src/util/mpq.cpp display.
package util

import (
	"errors"
	"math"
	"strings"

	"github.com/glycerine/g3/pkg/arena"
)

var ErrShortBuffer = errors.New("mpnum: decimal output buffer too short")

// reserveText retains budgeted scalar-indexed bytes. Growth preserves the
// numerator prefix while preparing a rational denominator. DirectOnly is a
// cold-storage adaptation, not the source allocation profile.
func (m *NaturalManager) reserveText(needed uint32) error {
	if needed <= m.textCapacity {
		return nil
	}
	capacity := m.textCapacity
	if capacity == 0 {
		capacity = 1024
	}
	for capacity < needed {
		if capacity > math.MaxUint32/2 {
			return arena.ErrBounds
		}
		capacity *= 2
	}
	if m.Text == nil {
		owner, e := arena.NewSmallBytes(arena.SmallBytesConfig{DirectOnly: true, Descriptors: m.config.Descriptors})
		if e != nil {
			return e
		}
		m.Text = owner
	}
	ref, e := m.Text.Allocate(capacity)
	if e != nil {
		return e
	}
	if m.textRef != (arena.SmallBytesRef{}) {
		for i := uint32(0); i < m.textCapacity; i++ {
			v, e := m.Text.Read(m.textRef, i)
			if e != nil {
				panic(e)
			}
			if e = m.Text.Write(ref, i, v); e != nil {
				panic(e)
			}
		}
		if e = m.Text.Deallocate(m.textCapacity, m.textRef); e != nil {
			panic(e)
		}
	}
	m.textRef = ref
	m.textCapacity = capacity
	return nil
}
func (m *NaturalManager) textRead(i uint32) byte {
	v, e := m.Text.Read(m.textRef, i)
	if e != nil {
		panic(e)
	}
	return v
}
func (m *NaturalManager) textWrite(i uint32, v byte) {
	if e := m.Text.Write(m.textRef, i, v); e != nil {
		panic(e)
	}
}
func (m *NaturalManager) reverseText(start, end uint32) {
	for start < end {
		end--
		if start >= end {
			break
		}
		a, b := m.textRead(start), m.textRead(end)
		m.textWrite(start, b)
		m.textWrite(end, a)
		start++
	}
}
func checkedTextEnd(offset uint32, n uint64) (uint32, error) {
	if n > math.MaxUint32-uint64(offset) {
		return 0, arena.ErrBounds
	}
	return offset + uint32(n), nil
}

// prepareNatural follows source normalize / div_1 / unnormalize by ten. The
// three source local limb buffers have fresh source lifetimes in the embedded
// charged owner. Text remains an explicit output assembly adapter.
func (m *NaturalManager) prepareNatural(a limbView, offset uint32) (uint32, error) {
	if e := m.check(); e != nil {
		return 0, e
	}
	end, e := checkedTextEnd(offset, 11*uint64(a.size))
	if e != nil {
		return 0, e
	}
	if end == offset {
		end, e = checkedTextEnd(offset, 1)
		if e != nil {
			return 0, e
		}
	}
	if e = m.reserveText(end); e != nil {
		return 0, e
	}
	if e = m.sourceToCString(a, sourceDecimalDestination{assembly: m, offset: offset}, uint64(end-offset)); e != nil {
		return 0, e
	}
	for j := offset; j < end; j++ {
		if m.textRead(j) == 0 {
			return j, nil
		}
	}
	return 0, arena.ErrBounds
}
func (m *NaturalManager) publishText(dst []byte, n uint32) (int, error) {
	if uint64(n) > uint64(^uint(0)>>1) {
		return 0, arena.ErrBounds
	}
	if uint64(n) > uint64(len(dst)) {
		return int(n), ErrShortBuffer
	}
	for i := uint32(0); i < n; i++ {
		dst[i] = m.textRead(i)
	}
	return int(n), nil
}

// finalString is the host-output ownership boundary: only the escaped Go string
// backing is caller-owned. Numerical/digit scratch is budgeted; no transient
// budget charge misrepresents the escaped string lifetime.
func (m *NaturalManager) finalString(n uint32) (string, error) {
	if uint64(n) > uint64(^uint(0)>>1) {
		return "", arena.ErrBounds
	}
	var output strings.Builder
	output.Grow(int(n))
	for i := uint32(0); i < n; i++ {
		output.WriteByte(m.textRead(i))
	}
	return output.String(), nil
}

// FormatInto excludes the C terminator. It writes exactly n bytes, leaving dst
// unchanged on a short buffer or predictable storage failure.
func (m *NaturalManager) FormatInto(a []uint32, dst []byte) (int, error) {
	if uint64(len(a)) > math.MaxUint32 {
		return 0, arena.ErrBounds
	}
	n, e := m.prepareNatural(limbView{external: a, size: uint32(len(a))}, 0)
	if e != nil {
		return 0, e
	}
	return m.publishText(dst, n)
}
func (m *NaturalManager) String(a []uint32) (string, error) {
	if uint64(len(a)) > math.MaxUint32 {
		return "", arena.ErrBounds
	}
	n, e := m.prepareNatural(limbView{external: a, size: uint32(len(a))}, 0)
	if e != nil {
		return "", e
	}
	return m.finalString(n)
}
func (m *IntegerManager) prepareInteger(a *Int, offset uint32) (uint32, error) {
	if e := m.check(a); e != nil {
		return 0, e
	}
	if a.IsSmall() {
		value := int64(a.Val)
		end, e := checkedTextEnd(offset, 12)
		if e != nil {
			return 0, e
		}
		if e = m.Natural.reserveText(end); e != nil {
			return 0, e
		}
		if value < 0 {
			m.Natural.textWrite(offset, '-')
			offset++
			value = -value
		}
		j := offset
		for {
			m.Natural.textWrite(j, '0'+byte(value%10))
			j++
			value /= 10
			if value == 0 {
				break
			}
		}
		m.Natural.reverseText(offset, j)
		return j, nil
	}
	if a.Val < 0 {
		end, e := checkedTextEnd(offset, 1)
		if e != nil {
			return 0, e
		}
		if e = m.Natural.reserveText(end); e != nil {
			return 0, e
		}
		m.Natural.textWrite(offset, '-')
		offset = end
	}
	c := m.Cell(a.Cell)
	size, e := c.Backing.ReadWord(m.Allocator, 0)
	if e != nil {
		return 0, e
	}
	// Source unsigned multiplication precedes sbuffer construction. Its
	// physical inline capacity remains independent of the wrapped POS.
	if e = m.Natural.beginIntegerCharacters(uint32(11 * size)); e != nil {
		return 0, e
	}
	natural := m.Natural
	e = natural.sourceToCString(limbView{allocator: m.Allocator, backing: c.Backing, size: size}, sourceDecimalDestination{character: natural.CharFormatting.Buffer}, uint64(len(natural.CharFormatting.Buffer)))
	var n uint32
	if e == nil {
		n, e = natural.appendIntegerCString(offset)
	}
	if e != nil {
		natural.CharFormatting.originalCause = e
	}
	// An interrupted inner destructor retains its cursor. The outer completed
	// char object cannot be destroyed ahead of that actual source prefix.
	if natural.Formatting.CompletedField != 0 {
		return 0, e
	}
	cleanup := natural.destroyIntegerCharacters()
	if e != nil {
		return 0, e
	}
	if cleanup != nil {
		return 0, cleanup
	}
	return n, nil
}
func (m *IntegerManager) FormatInto(a *Int, dst []byte) (int, error) {
	n, e := m.prepareInteger(a, 0)
	if e != nil {
		return 0, e
	}
	return m.Natural.publishText(dst, n)
}
func (m *IntegerManager) String(a *Int) (string, error) {
	n, e := m.prepareInteger(a, 0)
	if e != nil {
		return "", e
	}
	return m.Natural.finalString(n)
}
func (m *RationalManager) prepareRational(a *Rational) (uint32, error) {
	if e := m.check(a); e != nil {
		return 0, e
	}
	n, e := m.Z.prepareInteger(&a.Num, 0)
	if e != nil {
		return 0, e
	}
	if isIntegerRational(a) {
		return n, nil
	}
	end, e := checkedTextEnd(n, 1)
	if e != nil {
		return 0, e
	}
	if e = m.Z.Natural.reserveText(end); e != nil {
		return 0, e
	}
	m.Z.Natural.textWrite(n, '/')
	return m.Z.prepareInteger(&a.Den, end)
}
func (m *RationalManager) FormatInto(a *Rational, dst []byte) (int, error) {
	n, e := m.prepareRational(a)
	if e != nil {
		return 0, e
	}
	return m.Z.Natural.publishText(dst, n)
}
func (m *RationalManager) String(a *Rational) (string, error) {
	n, e := m.prepareRational(a)
	if e != nil {
		return "", e
	}
	return m.Z.Natural.finalString(n)
}
