// Copyright (c) 2012 Microsoft Corporation. Source src/util/mpff.h/.cpp.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"reflect"
)

// FastFloat owns a fixed significand block in its implicit manager. No owning
// shallow copies. Del retains dead scalar bits; Reset is a separate operation.
type FastFloat struct {
	bits     ffRef
	exponent int32
}

func NewFastFloat() FastFloat        { return FastFloat{} }
func (a *FastFloat) Exponent() int32 { return a.exponent }
func (a *FastFloat) index() uint32   { return uint32(a.bits) & 0x7fffffff }
func (a *FastFloat) negative() bool  { return uint32(a.bits)&0x80000000 != 0 }
func (a *FastFloat) sign(b bool) {
	if b {
		a.bits |= ffRef(1 << 31)
	} else {
		a.bits &^= ffRef(1 << 31)
	}
}

type FastFloatConfig struct {
	PrecisionWords, InitialCapacity uint32
	Storage                         arena.VectorConfig
}

func DefaultFastFloatConfig(c arena.VectorConfig) FastFloatConfig { return FastFloatConfig{2, 1024, c} }

var ErrFastFloatOverflow = errors.New("multi-precision floating point (mpff) overflow")
var ErrFastFloatDivisionByZero = errors.New("multi-precision floating point (mpff) division by zero")
var ErrFastFloatDomain = errors.New("multi-precision floating point (mpff) exception")
var ErrFastFloatInvariant = errors.New("mpnum: MPFF source invariant")
var ErrFastFloatPowerDomain = errors.New("mpnum: MPFF power mask wrapped into nontermination")
var ErrFastFloatOwner = errors.New("mpnum: invalid MPFF owner or dead significand")

type ffVector struct {
	ref            arena.VectorRef
	size, capacity uint32
}

// Cold owner: owns every referenced service and buffer; caller numerical
// conversion managers are borrowed only for their individual operation.
type FastFloatManager struct {
	self                                       *FastFloatManager
	words                                      *arena.Words
	natural                                    *NaturalManager
	config                                     FastFloatConfig
	precision, bits, capacity, next, live      uint32
	toPlus, closed                             bool
	headerBytes                                uint64
	significands, metadata, freeIDs, setBuffer ffVector
	buffers                                    [4]ffVector
	formatting                                 fastFormattingScope
	closing                                    bool
	one                                        FastFloat
}

func NewFastFloatManager(c FastFloatConfig) (*FastFloatManager, error) {
	if c.PrecisionWords < 2 || c.PrecisionWords > uint32(0x7fffffff)/32 || c.InitialCapacity == 0 || uint64(c.InitialCapacity)*uint64(c.PrecisionWords) > uint64(^uint32(0))/4 || c.InitialCapacity > 0x7fffffff || c.Storage.Descriptors.Budget == nil {
		return nil, ErrFastFloatDomain
	}
	h := uint64(reflect.TypeOf(FastFloatManager{}).Size())
	if e := c.Storage.Descriptors.Budget.Reserve(h); e != nil {
		return nil, e
	}
	m := &FastFloatManager{config: c, precision: c.PrecisionWords, bits: c.PrecisionWords * 32, capacity: c.InitialCapacity, headerBytes: h}
	m.self = m
	var e error
	m.words, e = arena.NewWords(c.Storage)
	if e != nil {
		m.Close()
		return nil, e
	}
	m.natural, e = newNaturalManager(c.Storage, m.words, false)
	if e != nil {
		m.Close()
		return nil, e
	}
	if e = m.resize(&m.significands, m.capacity*m.precision); e == nil {
		e = m.resize(&m.metadata, m.capacity*2)
	}
	for i := 0; i < 4 && e == nil; i++ {
		e = m.resize(&m.buffers[i], 2*m.precision)
	}
	if e == nil {
		if m.mkID() != 0 {
			e = ErrFastFloatInvariant
		}
	}
	if e == nil {
		e = m.SetInt32(&m.one, 1)
	}
	if e != nil {
		m.Close()
		return nil, e
	}
	return m, nil
}
func (m *FastFloatManager) open() error {
	if m == nil || m.self != m {
		return ErrFastFloatOwner
	}
	if m.closed || m.closing {
		return arena.ErrClosed
	}
	return nil
}
func (m *FastFloatManager) check(a *FastFloat) error {
	if e := m.open(); e != nil {
		return e
	}
	if a == nil {
		return arena.ErrInvalidSlot
	}
	i := a.index()
	if i == 0 {
		return nil
	}
	if i >= m.capacity || m.read(&m.metadata, 2*i) != 1 {
		return ErrFastFloatOwner
	}
	g := ffGeneration(a.bits)
	if g != 0 && g != m.read(&m.metadata, 2*i+1) {
		return ErrFastFloatOwner
	}
	return nil
}
func (m *FastFloatManager) pair(a, b *FastFloat) error {
	if e := m.check(a); e != nil {
		return e
	}
	return m.check(b)
}
func (m *FastFloatManager) triple(a, b, c *FastFloat) error {
	if e := m.pair(a, b); e != nil {
		return e
	}
	return m.check(c)
}
func (m *FastFloatManager) read(v *ffVector, i uint32) uint32 {
	n, e := m.words.Read(v.ref, i)
	if e != nil {
		panic(e)
	}
	return n
}
func (m *FastFloatManager) write(v *ffVector, i, n uint32) {
	if e := m.words.Write(v.ref, i, n); e != nil {
		panic(e)
	}
}
func (m *FastFloatManager) reserve(v *ffVector, n uint32) error {
	for v.capacity < n {
		c := uint64(2)
		if v.capacity != 0 {
			c = (3*uint64(v.capacity) + 1) / 2
		}
		if c > uint64(^uint32(0))/4 || c <= uint64(v.capacity) {
			return arena.ErrBounds
		}
		var r arena.VectorRef
		var e error
		if v.ref == (arena.VectorRef{}) {
			r, e = m.words.Allocate(v.size, uint32(c))
		} else {
			r, e = m.words.Reserve(v.ref, uint32(c))
		}
		if e != nil {
			return e
		}
		v.ref = r
		v.capacity = uint32(c)
	}
	return nil
}
func (m *FastFloatManager) resize(v *ffVector, n uint32) error {
	if e := m.reserve(v, n); e != nil {
		return e
	}
	if n == 0 && v.ref == (arena.VectorRef{}) {
		v.size = 0
		return nil
	}
	r, e := m.words.Resize(v.ref, n)
	if e != nil {
		return e
	}
	v.ref = r
	v.size = n
	return nil
}
func (m *FastFloatManager) mkID() uint32 {
	if m.freeIDs.size != 0 {
		i := m.read(&m.freeIDs, m.freeIDs.size-1)
		m.resize(&m.freeIDs, m.freeIDs.size-1)
		return i
	}
	i := m.next
	m.next++
	return i
}
func (m *FastFloatManager) allocate(a *FastFloat) error {
	if a.index() != 0 {
		return nil
	}
	i := m.mkID()
	if i == 0 || i > 0x7fffffff {
		return ErrFastFloatDomain
	}
	for i >= m.capacity {
		c := uint64(m.capacity) * 2
		if c > 0x7fffffff || c*uint64(m.precision) > uint64(^uint32(0))/4 {
			return arena.ErrBounds
		}
		if e := m.resize(&m.significands, uint32(c)*m.precision); e != nil {
			return e
		}
		if e := m.resize(&m.metadata, uint32(c)*2); e != nil {
			return e
		}
		m.capacity = uint32(c)
	}
	// Real capacity preflight ensures every live block can be lexically reclaimed
	// without Reserve even under the failure that caused its caller to unwind.
	needed := uint64(m.freeIDs.size) + uint64(m.live) + 1
	if needed > uint64(^uint32(0)) {
		return arena.ErrBounds
	}
	if e := m.reserve(&m.freeIDs, uint32(needed)); e != nil {
		return e
	}
	g := m.read(&m.metadata, 2*i+1)
	if g == ^uint32(0) {
		return ErrFastFloatOwner
	}
	g++
	m.write(&m.metadata, 2*i, 1)
	m.write(&m.metadata, 2*i+1, g)
	m.live++
	a.bits = ffPack(i, g, a.negative())
	if sourceExtendedGCDDebug && !ffIsZero(m.sig(a)) {
		return ErrFastFloatInvariant
	}
	return nil
}
func (m *FastFloatManager) sig(a *FastFloat) ffLimbView {
	return ffLimbView{m: m, v: &m.significands, offset: a.index() * m.precision, size: m.precision}
}
func (m *FastFloatManager) buffer(i int, size uint32) ffLimbView {
	return ffLimbView{m: m, v: &m.buffers[i], size: size}
}
func (m *FastFloatManager) Del(a *FastFloat) error {
	if e := m.check(a); e != nil {
		return e
	}
	i := a.index()
	if i == 0 {
		return nil
	}
	g := m.read(&m.metadata, 2*i+1)
	if g != ^uint32(0) {
		n := m.freeIDs.size
		if e := m.resize(&m.freeIDs, n+1); e != nil {
			return e
		}
		m.write(&m.freeIDs, n, i)
		m.write(&m.metadata, 2*i, 0)
	} else {
		m.write(&m.metadata, 2*i, 2)
	}
	ffZero(m.sig(a))
	m.live--
	return nil
}
func (m *FastFloatManager) Reset(a *FastFloat) error {
	if e := m.Del(a); e != nil {
		return e
	}
	*a = FastFloat{}
	return m.debugCheck(a)
}
func (m *FastFloatManager) CloseChecked() error {
	if m == nil || m.self != m {
		return ErrFastFloatOwner
	}
	if m.closed {
		return nil
	}
	m.closing = true
	if m.natural != nil {
		if e := m.CleanupFormatting(); e != nil {
			return e
		}
	}
	if m.words != nil {
		if m.one.index() != 0 { // Del needs the intact owner during this final action.
			m.closing = false
			e := m.Del(&m.one)
			m.closing = true
			if e != nil {
				return e
			}
			m.one = FastFloat{}
		}
		if m.natural != nil {
			if e := m.natural.CloseChecked(); e != nil {
				return e
			}
		}
		m.words.Close()
	}
	m.config.Storage.Descriptors.Budget.Release(m.headerBytes)
	m.headerBytes = 0
	m.closed = true
	return nil
}
func (m *FastFloatManager) Close() {
	if m == nil || m.self != m {
		return
	}
	if e := m.CloseChecked(); e != nil {
		panic(m)
	}
}

func (m *FastFloatManager) Precise() bool { return false }
func (m *FastFloatManager) Field() bool   { return true }
func (m *FastFloatManager) SetRounding(up bool) error {
	if e := m.open(); e != nil {
		return e
	}
	m.toPlus = up
	return nil
}
func (m *FastFloatManager) RoundToPlusInf() error  { return m.SetRounding(true) }
func (m *FastFloatManager) RoundToMinusInf() error { return m.SetRounding(false) }
func (m *FastFloatManager) RoundingToPlusInf() (bool, error) {
	if e := m.open(); e != nil {
		return false, e
	}
	return m.toPlus, nil
}
func (m *FastFloatManager) invariant(ok bool) error {
	if sourceExtendedGCDDebug && !ok {
		return ErrFastFloatInvariant
	}
	return nil
}
func (m *FastFloatManager) debugCheck(a *FastFloat) error {
	if !sourceExtendedGCDDebug {
		return nil
	}
	if a.index() == 0 {
		return m.invariant(!a.negative() && a.exponent == 0 && ffIsZero(m.sig(a)))
	}
	return m.invariant(m.sig(a).read(m.precision-1)&0x80000000 != 0)
}
func (m *FastFloatManager) Check(a *FastFloat) (bool, error) {
	if e := m.check(a); e != nil {
		return false, e
	}
	if e := m.debugCheck(a); e != nil {
		return false, e
	}
	return true, nil
}
func (m *FastFloatManager) SetExponent(a *FastFloat, e int32) error {
	if x := m.check(a); x != nil {
		return x
	}
	if a.index() == 0 {
		return nil
	}
	a.exponent = e
	return m.debugCheck(a)
}
func (m *FastFloatManager) Copy(dst, src *FastFloat) error {
	if e := m.pair(dst, src); e != nil {
		return e
	}
	if src.index() == 0 {
		return m.Reset(dst)
	}
	if dst == src {
		return nil
	}
	if e := m.allocate(dst); e != nil {
		return e
	}
	dst.sign(src.negative())
	dst.exponent = src.exponent
	ffCopy(m.sig(src), m.sig(dst))
	return m.debugCheck(dst)
}
func (m *FastFloatManager) Swap(a, b *FastFloat) error {
	if e := m.pair(a, b); e != nil {
		return e
	}
	*a, *b = *b, *a
	return nil
}
func (m *FastFloatManager) Neg(a *FastFloat) error {
	if e := m.check(a); e != nil {
		return e
	}
	if a.index() != 0 {
		a.sign(!a.negative())
	}
	return nil
}
func (m *FastFloatManager) Abs(a *FastFloat) error {
	if e := m.check(a); e != nil {
		return e
	}
	a.sign(false)
	return nil
}
func (m *FastFloatManager) SetInt32(a *FastFloat, v int32) error {
	if e := m.check(a); e != nil {
		return e
	}
	n := uint32(v)
	if v < 0 {
		n = 0 - n
	}
	if e := m.SetUint32(a, n); e != nil {
		return e
	}
	if v < 0 {
		a.sign(true)
	}
	return m.debugCheck(a)
}
func (m *FastFloatManager) SetUint32(a *FastFloat, v uint32) error {
	if e := m.check(a); e != nil {
		return e
	}
	if v == 0 {
		return m.Reset(a)
	}
	if e := m.allocate(a); e != nil {
		return e
	}
	k := ffLeading(v)
	a.sign(false)
	a.exponent = 32 - int32(k) - int32(m.bits)
	s := m.sig(a)
	ffZero(s)
	s.write(m.precision-1, v<<k)
	return m.debugCheck(a)
}
func (m *FastFloatManager) SetInt64(a *FastFloat, v int64) error {
	if v == 0 {
		if e := m.Reset(a); e != nil {
			return e
		}
	} else {
		n := uint64(v)
		if v < 0 {
			n = 0 - n
		}
		if e := m.SetUint64(a, n); e != nil {
			return e
		}
		if v < 0 {
			a.sign(true)
		}
	}
	if e := m.debugCheck(a); e != nil {
		return e
	}
	if sourceExtendedGCDDebug {
		got, e := m.Int64(a)
		if e != nil {
			return e
		}
		return m.invariant(got == v)
	}
	return nil
}
func (m *FastFloatManager) SetUint64(a *FastFloat, v uint64) error {
	original := v
	if e := m.check(a); e != nil {
		return e
	}
	if v == 0 {
		if e := m.Reset(a); e != nil {
			return e
		}
	} else {
		if e := m.allocate(a); e != nil {
			return e
		}
		k := uint32(0)
		for v&(uint64(1)<<63) == 0 {
			v <<= 1
			k++
		}
		a.sign(false)
		a.exponent = 64 - int32(k) - int32(m.bits)
		s := m.sig(a)
		ffZero(s)
		s.write(m.precision-1, uint32(v>>32))
		s.write(m.precision-2, uint32(v))
	}
	if e := m.debugCheck(a); e != nil {
		return e
	}
	if sourceExtendedGCDDebug {
		got, e := m.Uint64(a)
		if e != nil {
			return e
		}
		return m.invariant(got == original)
	}
	return nil
}
func (m *FastFloatManager) SetFraction32(a *FastFloat, n int32, d uint32) error {
	if e := m.check(a); e != nil {
		return e
	}
	var x, y FastFloat
	defer func() { m.Del(&y); m.Del(&x) }()
	if e := m.SetInt32(&x, n); e != nil {
		return e
	}
	if e := m.SetUint32(&y, d); e != nil {
		return e
	}
	return m.Div(&x, &y, a)
}
func (m *FastFloatManager) SetFraction64(a *FastFloat, n int64, d uint64) error {
	if e := m.check(a); e != nil {
		return e
	}
	var x, y FastFloat
	defer func() { m.Del(&y); m.Del(&x) }()
	if e := m.SetInt64(&x, n); e != nil {
		return e
	}
	if e := m.SetUint64(&y, d); e != nil {
		return e
	}
	return m.Div(&x, &y, a)
}
