// Copyright (c) 2006 Microsoft Corporation
// Translated from src/util/mpz.h and mpz.cpp (_MP_INTERNAL, 32-bit digits).
// Original author: Leonardo de Moura, 2010-06-17.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"math"
	"reflect"
)

var ErrConversion = errors.New("mpnum: integer conversion out of range")

// Int is a pointer-free, owner-local mutable integer. With a zero cell ref,
// val is an inline int32. Otherwise val is the sign (-1/+1). A big Int must
// NEVER be shallow-copied: use Copy, Move, or Swap through its owning manager.
// The implicit manager owner is a caller contract; refs do not identify owners.
type Int struct {
	val  int32
	cell numericCellRef
}

func NewInt(v int32) Int     { return Int{val: v} }
func (a *Int) IsSmall() bool { return a.cell == 0 }

type numericCell struct {
	payload        arena.VectorRef
	size, capacity uint32
}
type IntegerConfig struct {
	Cells arena.Config
	Words arena.VectorConfig
}

// IntegerManager owns all persistent cells, limb payloads, two reusable result
// cells, two one-limb argument cells, and the source intMin/two64 constants.
// Single-owner confinement replaces the source synchronized OpenMP variant.
// Cells and Words must share one Budget. Close invalidates every live numeral.
type IntegerManager struct {
	cells         *numericCellSlab
	words         *arena.Words
	tmp, arg      [2]numericCellRef
	natural       *NaturalManager // child borrows this manager's Words and Budget
	intMin, two64 Int
	closed        bool
	budget        *arena.Budget
	headerBytes   uint64
}

func NewIntegerManager(config IntegerConfig) (*IntegerManager, error) {
	if config.Cells.Budget != config.Words.Descriptors.Budget {
		return nil, arena.ErrConfig
	}
	header := uint64(reflect.TypeOf(IntegerManager{}).Size())
	if e := config.Cells.Budget.Reserve(header); e != nil {
		return nil, e
	}
	cells, e := newNumericCellSlab(config.Cells)
	if e != nil {
		config.Cells.Budget.Release(header)
		return nil, e
	}
	words, e := arena.NewWords(config.Words)
	if e != nil {
		cells.Close()
		config.Cells.Budget.Release(header)
		return nil, e
	}
	m := &IntegerManager{cells: cells, words: words, budget: config.Cells.Budget, headerBytes: header}
	fail := func(err error) (*IntegerManager, error) { m.Close(); return nil, err }
	m.natural, e = newNaturalManager(config.Words, words, false)
	if e != nil {
		return fail(e)
	}
	for i := 0; i < 2; i++ {
		m.tmp[i], e = m.allocate(6, 0)
		if e != nil {
			return fail(e)
		}
		m.arg[i], e = m.allocate(6, 1)
		if e != nil {
			return fail(e)
		}
	}
	if e = m.SetInt64(&m.intMin, -int64(math.MinInt32)); e != nil {
		return fail(e)
	}
	if e = m.SetUint64(&m.two64, math.MaxUint64); e != nil {
		return fail(e)
	}
	one := NewInt(1)
	if e = m.Add(&m.two64, &one, &m.two64); e != nil {
		return fail(e)
	}
	return m, nil
}

func (m *IntegerManager) check(a *Int) error {
	if m.closed {
		return arena.ErrClosed
	}
	if a == nil {
		return arena.ErrInvalidSlot
	}
	if a.cell != 0 && !validNumericRef(m.cells, a.cell) {
		return arena.ErrInvalidSlot
	}
	return nil
}
func (m *IntegerManager) cell(ref numericCellRef) numericCell {
	c, e := m.cells.Read(numericSlot(ref))
	if e != nil {
		panic(e)
	}
	return c
}
func (m *IntegerManager) store(ref numericCellRef, c numericCell) {
	if e := m.cells.Write(numericSlot(ref), c); e != nil {
		panic(e)
	}
}
func (m *IntegerManager) digit(ref numericCellRef, i uint32) uint32 {
	c := m.cell(ref)
	if i >= c.size {
		return 0
	}
	v, e := m.words.Read(c.payload, i)
	if e != nil {
		panic(e)
	}
	return v
}
func (m *IntegerManager) writeDigit(ref numericCellRef, i, v uint32) {
	c := m.cell(ref)
	if e := m.words.Write(c.payload, i, v); e != nil {
		panic(e)
	}
}
func (m *IntegerManager) resizeCell(ref numericCellRef, size uint32) {
	c := m.cell(ref)
	if size > c.capacity {
		panic("mpnum: cell capacity exceeded")
	}
	payload, e := m.words.Resize(c.payload, size)
	if e != nil {
		panic(e)
	}
	c.payload = payload
	c.size = size
	m.store(ref, c)
}
func (m *IntegerManager) allocate(capacity, size uint32) (numericCellRef, error) {
	if capacity < 6 {
		capacity = 6
	}
	slot, gen, e := m.cells.Allocate()
	if e != nil {
		return 0, e
	}
	payload, e := m.words.Allocate(size, capacity)
	if e != nil {
		m.cells.Recycle(slot)
		return 0, e
	}
	ref := makeNumericRef(slot, gen)
	m.store(ref, numericCell{payload: payload, size: size, capacity: capacity})
	return ref, nil
}
func (m *IntegerManager) free(ref numericCellRef) {
	c := m.cell(ref)
	if e := m.words.Free(c.payload); e != nil {
		panic(e)
	}
	if e := m.cells.Recycle(numericSlot(ref)); e != nil {
		panic(e)
	}
}
func (m *IntegerManager) Close() {
	if m.closed {
		return
	}
	// Domain cleanup precedes recycling. Also release caller-owned outstanding
	// numerals, which cannot remain usable after this owner's close.
	if m.cells != nil {
		for n := uint64(1); n <= uint64(m.cells.Stats().Slots); n++ {
			slot := arena.Slot(n)
			c, e := m.cells.Read(slot)
			if e == nil {
				m.words.Free(c.payload)
				m.cells.Recycle(slot)
			}
		}
	}
	if m.natural != nil {
		m.natural.Close()
	}
	if m.words != nil {
		m.words.Close()
	}

	if m.cells != nil {
		m.cells.Close()
	}
	m.tmp = [2]numericCellRef{}
	m.arg = [2]numericCellRef{}
	m.natural = nil
	m.intMin = Int{}
	m.two64 = Int{}
	m.budget.Release(m.headerBytes)
	m.headerBytes = 0
	m.closed = true
}

// Del preserves the source's m_val after releasing a big cell. Thus deleting a
// big positive/negative value leaves inline +1/-1. Reset additionally writes 0.
func (m *IntegerManager) Del(a *Int) error {
	if e := m.check(a); e != nil {
		return e
	}
	if a.cell != 0 {
		m.free(a.cell)
		a.cell = 0
	}
	return nil
}
func (m *IntegerManager) Reset(a *Int) error {
	if e := m.Del(a); e != nil {
		return e
	}
	a.val = 0
	return nil
}

func (m *IntegerManager) setBig64(a *Int, v uint64, sign int32) error {
	ref := a.cell
	if ref == 0 {
		var e error
		ref, e = m.allocate(6, 0)
		if e != nil {
			return e
		}
	}
	size := uint32(1)
	if v>>32 != 0 {
		size = 2
	}
	m.resizeCell(ref, size)
	m.writeDigit(ref, 0, uint32(v))
	if size == 2 {
		m.writeDigit(ref, 1, uint32(v>>32))
	}
	a.cell = ref
	a.val = sign
	return nil
}
func (m *IntegerManager) SetInt64(a *Int, v int64) error {
	if e := m.check(a); e != nil {
		return e
	}
	if v >= math.MinInt32 && v <= math.MaxInt32 {
		if e := m.Del(a); e != nil {
			return e
		}
		a.val = int32(v)
		return nil
	}
	sign := int32(1)
	magnitude := uint64(v)
	if v < 0 {
		sign = -1
		magnitude = 0 - uint64(v)
	}
	return m.setBig64(a, magnitude, sign)
}

// SetUint64 preserves the source's strict < INT_MAX inline threshold.
func (m *IntegerManager) SetUint64(a *Int, v uint64) error {
	if e := m.check(a); e != nil {
		return e
	}
	if v < math.MaxInt32 {
		return m.SetInt64(a, int64(v))
	}
	return m.setBig64(a, v, 1)
}

// SetLimbs is the source unsigned digit-vector setter, with trailing zeros
// removed. The caller's slice is borrowed and never retained.
func (m *IntegerManager) SetLimbs(a *Int, digits []uint32) error {
	if e := m.check(a); e != nil {
		return e
	}
	for len(digits) > 0 && digits[len(digits)-1] == 0 {
		digits = digits[:len(digits)-1]
	}
	if len(digits) == 0 {
		return m.Reset(a)
	}
	if len(digits) == 1 {
		return m.SetInt64(a, int64(digits[0]))
	}
	if uint64(len(digits)) > math.MaxUint32 {
		return arena.ErrBounds
	}
	size := uint32(len(digits))
	ref := a.cell
	if ref == 0 || m.cell(ref).capacity < size {
		var e error
		ref, e = m.allocate(max(size, 6), size)
		if e != nil {
			return e
		}
	} else {
		m.resizeCell(ref, size)
	}
	for i, v := range digits {
		m.writeDigit(ref, uint32(i), v)
	}
	old := a.cell
	a.cell = ref
	a.val = 1
	if old != 0 && old != ref {
		m.free(old)
	}
	return nil
}
func (m *IntegerManager) Copy(dst, src *Int) error {
	if e := m.check(dst); e != nil {
		return e
	}
	if e := m.check(src); e != nil {
		return e
	}
	if dst == src {
		return nil
	}
	if src.IsSmall() {
		v := src.val
		if e := m.Del(dst); e != nil {
			return e
		}
		dst.val = v
		return nil
	}
	source := m.cell(src.cell)
	ref := dst.cell
	if ref == 0 || m.cell(ref).capacity < source.size {
		var e error
		ref, e = m.allocate(source.capacity, source.size)
		if e != nil {
			return e
		}
	} else {
		m.resizeCell(ref, source.size)
	}
	for i := uint32(0); i < source.size; i++ {
		m.writeDigit(ref, i, m.digit(src.cell, i))
	}
	old := dst.cell
	dst.cell = ref
	dst.val = src.val
	if old != 0 && old != ref {
		m.free(old)
	}
	return nil
}

// Move transfers the cell and preserves the source moved-from inline m_val.
// Self move is explicitly a no-op, an added Go contract.
func (m *IntegerManager) Move(dst, src *Int) error {
	if e := m.check(dst); e != nil {
		return e
	}
	if e := m.check(src); e != nil {
		return e
	}
	if dst == src {
		return nil
	}
	if e := m.Del(dst); e != nil {
		return e
	}
	dst.val = src.val
	dst.cell = src.cell
	src.cell = 0
	return nil
}
func (m *IntegerManager) Swap(a, b *Int) error {
	if e := m.check(a); e != nil {
		return e
	}
	if e := m.check(b); e != nil {
		return e
	}
	*a, *b = *b, *a
	return nil
}
func (m *IntegerManager) Neg(a *Int) error {
	if e := m.check(a); e != nil {
		return e
	}
	if a.IsSmall() && a.val == math.MinInt32 {
		return m.SetInt64(a, -int64(math.MinInt32))
	}
	a.val = -a.val
	return nil
}
func (m *IntegerManager) Abs(a *Int) error {
	if e := m.check(a); e != nil {
		return e
	}
	if a.IsSmall() {
		if a.val < 0 {
			return m.Neg(a)
		}
	} else {
		a.val = 1
	}
	return nil
}

func (m *IntegerManager) signCell(a *Int, idx int) (int32, numericCellRef) {
	if !a.IsSmall() {
		return a.val, a.cell
	}
	if a.val == math.MinInt32 {
		return -1, m.intMin.cell
	}
	sign := int32(1)
	v := a.val
	if v < 0 {
		sign = -1
		v = -v
	}
	m.writeDigit(m.arg[idx], 0, uint32(v))
	return sign, m.arg[idx]
}
func (m *IntegerManager) compareCells(a, b numericCellRef) int {
	for i := max(m.cell(a).size, m.cell(b).size); i > 0; i-- {
		u, v := m.digit(a, i-1), m.digit(b, i-1)
		if u > v {
			return 1
		}
		if u < v {
			return -1
		}
	}
	return 0
}
func (m *IntegerManager) Compare(a, b *Int) (int, error) {
	if e := m.check(a); e != nil {
		return 0, e
	}
	if e := m.check(b); e != nil {
		return 0, e
	}
	if a.IsSmall() && b.IsSmall() {
		if a.val < b.val {
			return -1, nil
		}
		if a.val > b.val {
			return 1, nil
		}
		return 0, nil
	}
	sa, ca := m.signCell(a, 0)
	sb, cb := m.signCell(b, 1)
	if sa > 0 {
		if sb > 0 {
			return m.compareCells(ca, cb), nil
		}
		return 1, nil
	}
	if sb > 0 {
		return -1, nil
	}
	return m.compareCells(cb, ca), nil
}

func (m *IntegerManager) ensureTmp(idx int, size uint32) error {
	if m.cell(m.tmp[idx]).capacity >= size {
		return nil
	}
	capacity := (3*uint64(size) + 1) >> 1
	if capacity > math.MaxUint32 {
		return arena.ErrBounds
	}
	ref, e := m.allocate(uint32(capacity), 0)
	if e != nil {
		return e
	}
	m.free(m.tmp[idx])
	m.tmp[idx] = ref
	return nil
}
func (m *IntegerManager) publishTmp(dst *Int, sign int32, size uint32) error {
	return m.publishTmpAt(0, dst, sign, size, 0)
}

// publishTmpAt consumes an optional preallocated replacement scratch cell.
// This lets paired division prepare allocations before publishing either value.
func (m *IntegerManager) publishTmpAt(idx int, dst *Int, sign int32, size uint32, replacement numericCellRef) error {
	ref := m.tmp[idx]
	for size > 0 && m.digit(ref, size-1) == 0 {
		size--
	}
	if size == 0 {
		if replacement != 0 {
			m.free(replacement)
		}
		return m.Reset(dst)
	}
	if size == 1 && m.digit(ref, 0) <= math.MaxInt32 {
		if replacement != 0 {
			m.free(replacement)
		}
		v := int32(m.digit(ref, 0))
		if sign < 0 {
			v = -v
		}
		return m.SetInt64(dst, int64(v))
	}
	old := dst.cell
	if old == 0 {
		if replacement != 0 {
			old = replacement
		} else {
			var e error
			old, e = m.allocate(6, 0)
			if e != nil {
				return e
			}
		}
	} else if replacement != 0 {
		m.free(replacement)
	}
	m.resizeCell(ref, size)
	dst.cell = ref
	dst.val = sign
	m.tmp[idx] = old
	return nil
}
func (m *IntegerManager) Add(a, b, c *Int) error { return m.addSub(a, b, c, false) }
func (m *IntegerManager) Sub(a, b, c *Int) error { return m.addSub(a, b, c, true) }
func (m *IntegerManager) addSub(a, b, c *Int, subtract bool) error {
	for _, n := range []*Int{a, b, c} {
		if e := m.check(n); e != nil {
			return e
		}
	}
	if a.IsSmall() && b.IsSmall() {
		v := int64(b.val)
		if subtract {
			v = -v
		}
		return m.SetInt64(c, int64(a.val)+v)
	}
	sa, ca := m.signCell(a, 0)
	sb, cb := m.signCell(b, 1)
	if subtract {
		sb = -sb
	}
	na, nb := m.cell(ca).size, m.cell(cb).size
	n := max(na, nb)
	sign := sa
	if sa == sb {
		if n == math.MaxUint32 {
			return arena.ErrBounds
		}
		if e := m.ensureTmp(0, n+1); e != nil {
			return e
		}
		m.resizeCell(m.tmp[0], n+1)
		var k uint32
		for j := uint32(0); j < n; j++ {
			u, v := m.digit(ca, j), m.digit(cb, j)
			r := u + v
			c1 := r < u
			out := r + k
			c2 := out < r
			m.writeDigit(m.tmp[0], j, out)
			k = 0
			if c1 || c2 {
				k = 1
			}
		}
		m.writeDigit(m.tmp[0], n, k)
		return m.publishTmp(c, sign, n+1)
	}
	cmp := m.compareCells(ca, cb)
	if cmp == 0 {
		return m.Reset(c)
	}
	if cmp < 0 {
		ca, cb = cb, ca
		sign = sb
		n = nb
	} else {
		n = na
	}
	if e := m.ensureTmp(0, n); e != nil {
		return e
	}
	m.resizeCell(m.tmp[0], n)
	var k uint32
	for j := uint32(0); j < n; j++ {
		u, v := m.digit(ca, j), m.digit(cb, j)
		r := u - v
		c1 := r > u
		out := r - k
		c2 := out > r
		m.writeDigit(m.tmp[0], j, out)
		k = 0
		if c1 || c2 {
			k = 1
		}
	}
	if k != 0 {
		panic("mpnum: unexpected subtraction borrow")
	}
	return m.publishTmp(c, sign, n)
}

func (m *IntegerManager) absUint64(a *Int) (uint64, error) {
	c := m.cell(a.cell)
	if c.size > 2 {
		return 0, ErrConversion
	}
	v := uint64(m.digit(a.cell, 0))
	if c.size == 2 {
		v |= uint64(m.digit(a.cell, 1)) << 32
	}
	return v, nil
}
func (m *IntegerManager) IsUint64(a *Int) (bool, error) {
	if e := m.check(a); e != nil {
		return false, e
	}
	if a.val < 0 {
		return false, nil
	}
	return a.IsSmall() || m.cell(a.cell).size <= 2, nil
}
func (m *IntegerManager) IsInt64(a *Int) (bool, error) {
	if e := m.check(a); e != nil {
		return false, e
	}
	if a.IsSmall() {
		return true, nil
	}
	v, e := m.absUint64(a)
	if e != nil {
		return false, nil
	}
	msb := v & (uint64(1) << 63)
	if a.val >= 0 {
		return msb == 0, nil
	}
	return msb == 0 || msb == v, nil
}

// Uint64 preserves the source accessor: inline negatives cast to uint64,
// big negatives return their absolute magnitude. Use IsUint64 for fit/sign.
func (m *IntegerManager) Uint64(a *Int) (uint64, error) {
	if e := m.check(a); e != nil {
		return 0, e
	}
	if a.IsSmall() {
		return uint64(int64(a.val)), nil
	}
	return m.absUint64(a)
}
func (m *IntegerManager) Int64(a *Int) (int64, error) {
	ok, e := m.IsInt64(a)
	if e != nil {
		return 0, e
	}
	if !ok {
		return 0, ErrConversion
	}
	if a.IsSmall() {
		return int64(a.val), nil
	}
	v, _ := m.absUint64(a)
	if a.val < 0 {
		if v != 0 && v<<1 == 0 {
			return math.MinInt64, nil
		}
		return -int64(v), nil
	}
	return int64(v), nil
}
