// Copyright (c) 2011 Microsoft Corporation. Licensed under the MIT license.
// Concrete core_manager::factors, actual scoped coefficient ownership.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"math"
	"reflect"
)

type UPolynomialFactors struct {
	self                      *UPolynomialFactors
	manager                   *UPolynomialManager
	factors                   []upolyVectorState
	count                     uint32
	degrees                   upolySequenceUnsigned
	constant                  Int
	totalFactors, totalDegree uint32
	header                    uint64
	closed                    bool
	closeCleared              bool
}
type UPolynomialFactorsStats struct {
	OwnerBytes, OuterBytes, MultiplicityBytes, TotalBytes, ManagerBackingBytes  uint64
	Count, MultiplicityCount, Capacity, MultiplicityCapacity, OwnedNumeralCells uint32
}

func (m *UPolynomialManager) NewFactors() (*UPolynomialFactors, error) {
	if e := m.check(); e != nil {
		return nil, e
	}
	if m.liveFactors == math.MaxUint32 {
		return nil, arena.ErrSlotLimit
	}
	bytes := uint64(reflect.TypeOf(UPolynomialFactors{}).Size())
	if e := m.config.Integer.budget.Reserve(bytes); e != nil {
		return nil, e
	}
	fs := &UPolynomialFactors{manager: m, header: bytes}
	fs.self = fs
	if e := m.modular.SetInt32(&fs.constant, 1); e != nil {
		m.modular.Del(&fs.constant)
		m.config.Integer.budget.Release(bytes)
		return nil, e
	}
	m.liveFactors++
	return fs, nil
}
func (fs *UPolynomialFactors) check() error {
	if fs == nil || fs.self != fs || fs.manager == nil {
		return ErrUPolynomialOwner
	}
	if fs.closed {
		return arena.ErrClosed
	}
	return fs.manager.check()
}
func upolyFactorNextCapacity(old uint32, host uint64) (uint32, uint64, error) {
	var n uint32
	if old == 0 {
		n = 2
	} else {
		n = (3*old + 1) >> 1
		if n <= old || 8*n+8 <= 8*old+8 {
			return 0, 0, arena.ErrBounds
		}
	}
	stride := uint64(reflect.TypeOf(upolyVectorState{}).Size())
	if uint64(n)*8+8 > math.MaxUint32 || uint64(n) > host/stride {
		return 0, 0, arena.ErrBounds
	}
	return n, uint64(n) * stride, nil
}
func (fs *UPolynomialFactors) pushEmpty() (*upolyVectorState, error) {
	if fs.count == math.MaxUint32 {
		return nil, arena.ErrBounds
	}
	if uint64(fs.count) == uint64(cap(fs.factors)) {
		n, bytes, e := upolyFactorNextCapacity(uint32(cap(fs.factors)), uint64(int(^uint(0)>>1)))
		if e != nil {
			return nil, e
		}
		b := fs.manager.config.Integer.budget
		if e = b.Reserve(bytes); e != nil {
			return nil, e
		}
		next := make([]upolyVectorState, n)
		for i := uint32(0); i < fs.count; i++ {
			next[i] = fs.factors[i]
			fs.factors[i] = upolyVectorState{}
		}
		b.Release(uint64(cap(fs.factors)) * uint64(reflect.TypeOf(upolyVectorState{}).Size()))
		fs.factors = next
	}
	i := fs.count
	fs.factors[i] = upolyVectorState{}
	fs.count++
	return &fs.factors[i], nil
}
func (fs *UPolynomialFactors) factor(index uint32) (*upolyVectorState, error) {
	if e := fs.check(); e != nil {
		return nil, e
	}
	if index >= fs.count {
		return nil, arena.ErrBounds
	}
	return &fs.factors[index], nil
}
func (fs *UPolynomialFactors) multiplicity(index uint32) (uint32, error) {
	if e := fs.check(); e != nil {
		return 0, e
	}
	if modularSourceDebug && index >= fs.degrees.size {
		return 0, ErrUPolynomialAssertion
	}
	if index >= fs.degrees.initialized {
		return 0, arena.ErrBounds
	}
	return fs.degrees.values[index], nil
}
func (fs *UPolynomialFactors) Manager() (*UPolynomialManager, error) {
	if e := fs.check(); e != nil {
		return nil, e
	}
	return fs.manager, nil
}
func (fs *UPolynomialFactors) NumeralManager() (*IntegerModularManager, error) {
	if e := fs.check(); e != nil {
		return nil, e
	}
	return fs.manager.modular, nil
}
func (fs *UPolynomialFactors) DistinctFactors() (uint32, error) {
	if e := fs.check(); e != nil {
		return 0, e
	}
	return fs.count, nil
}
func (fs *UPolynomialFactors) TotalFactors() (uint32, error) {
	if e := fs.check(); e != nil {
		return 0, e
	}
	return fs.totalFactors, nil
}
func (fs *UPolynomialFactors) TotalDegree() (uint32, error) {
	if e := fs.check(); e != nil {
		return 0, e
	}
	return fs.totalDegree, nil
}
func (fs *UPolynomialFactors) Multiplicity(index uint32) (uint32, error) {
	return fs.multiplicity(index)
}

// ConstantCopy is an owned-copy adaptation of the borrowed source accessor.
func (fs *UPolynomialFactors) ConstantCopy(out *Int) error {
	if e := fs.check(); e != nil {
		return e
	}
	return fs.manager.modular.Copy(out, &fs.constant)
}
func (fs *UPolynomialFactors) SetConstant(value *Int) error {
	if e := fs.check(); e != nil {
		return e
	}
	return fs.manager.modular.Copy(&fs.constant, value)
}
func (fs *UPolynomialFactors) FactorView(index uint32) (UPolynomialView, error) {
	s, e := fs.factor(index)
	if e != nil {
		return UPolynomialView{}, e
	}
	return fs.manager.stateView(s), nil
}
func upolyStateDegree(s *upolyVectorState) uint32 {
	if s.size == 0 {
		return 0
	}
	return s.size - 1
}
func upolyViewDegree(v UPolynomialView) uint32 {
	if v.count == 0 {
		return 0
	}
	return v.count - 1
}
func (fs *UPolynomialFactors) SetMultiplicity(index, degree uint32) error {
	s, e := fs.factor(index)
	if e != nil {
		return e
	}
	old, e := fs.multiplicity(index)
	if e != nil {
		return e
	}
	fs.totalDegree -= upolyStateDegree(s) * old
	fs.totalFactors -= old
	fs.degrees.values[index] = degree
	fs.totalFactors += degree
	fs.totalDegree += upolyStateDegree(s) * degree
	return nil
}
func (fs *UPolynomialFactors) PushCopy(v UPolynomialView, degree uint32) error {
	if e := fs.check(); e != nil {
		return e
	}
	if modularSourceDebug && degree == 0 {
		return ErrUPolynomialAssertion
	}
	out, e := fs.pushEmpty()
	if e != nil {
		return e
	}
	if e = fs.manager.sequenceUnsignedPush(&fs.degrees, degree); e != nil {
		return e
	}
	if e = fs.manager.setView(out, v); e != nil {
		return e
	}
	fs.totalFactors += degree
	fs.totalDegree += upolyViewDegree(v) * degree
	return nil
}
func (fs *UPolynomialFactors) pushSwap(source *UPolynomialManager, p *upolyVectorState, degree uint32) error {
	if modularSourceDebug && degree == 0 {
		return ErrUPolynomialAssertion
	}
	out, e := fs.pushEmpty()
	if e != nil {
		return e
	}
	if e = fs.manager.sequenceUnsignedPush(&fs.degrees, degree); e != nil {
		return e
	}
	if e = fs.manager.adoptState(source, p); e != nil {
		return e
	}
	*p, *out = *out, *p
	fs.totalFactors += degree
	// Literal source computes degree(p) AFTER the swap with the empty element.
	fs.totalDegree += upolyStateDegree(p) * degree
	return nil
}
func (fs *UPolynomialFactors) PushMove(value UPolynomial, degree uint32) error {
	if e := fs.check(); e != nil {
		return e
	}
	return fs.PushMoveFrom(fs.manager, value, degree)
}
func (fs *UPolynomialFactors) PushMoveFrom(source *UPolynomialManager, value UPolynomial, degree uint32) error {
	if e := fs.check(); e != nil {
		return e
	}
	d, e := source.descriptor(value)
	if e != nil {
		return e
	}
	return fs.pushSwap(source, &d.state, degree)
}
func (fs *UPolynomialFactors) SwapFactor(index uint32, value UPolynomial) error {
	if e := fs.check(); e != nil {
		return e
	}
	return fs.SwapFactorFrom(index, fs.manager, value)
}
func (fs *UPolynomialFactors) SwapFactorFrom(index uint32, source *UPolynomialManager, value UPolynomial) error {
	s, e := fs.factor(index)
	if e != nil {
		return e
	}
	d, e := source.descriptor(value)
	if e != nil {
		return e
	}
	degree, e := fs.multiplicity(index)
	if e != nil {
		return e
	}
	// Physical preparation is a checked Go adapter before the source writes.
	if e = fs.manager.adoptState(source, &d.state); e != nil {
		return e
	}
	if e = source.adoptState(fs.manager, s); e != nil {
		return e
	}
	fs.totalDegree -= upolyStateDegree(s) * degree
	fs.totalDegree += upolyStateDegree(&d.state) * degree
	*s, d.state = d.state, *s
	return nil
}
func (fs *UPolynomialFactors) Swap(other *UPolynomialFactors) error {
	if e := fs.check(); e != nil {
		return e
	}
	if e := other.check(); e != nil {
		return e
	}
	if fs.manager.config.Integer != other.manager.config.Integer {
		return ErrUPolynomialOwner
	}
	for i := uint32(0); i < other.count; i++ {
		if e := fs.manager.adoptState(other.manager, &other.factors[i]); e != nil {
			return e
		}
	}
	for i := uint32(0); i < fs.count; i++ {
		if e := other.manager.adoptState(fs.manager, &fs.factors[i]); e != nil {
			return e
		}
	}
	fs.factors, other.factors = other.factors, fs.factors
	fs.count, other.count = other.count, fs.count
	fs.degrees, other.degrees = other.degrees, fs.degrees
	if e := fs.manager.modular.Swap(&fs.constant, &other.constant); e != nil {
		return e
	}
	fs.totalFactors, other.totalFactors = other.totalFactors, fs.totalFactors
	fs.totalDegree, other.totalDegree = other.totalDegree, fs.totalDegree
	return nil
}
func (fs *UPolynomialFactors) Clear() error {
	if e := fs.check(); e != nil {
		return e
	}
	m := fs.manager
	for i := uint32(0); i < fs.count; i++ {
		if e := m.reset(&fs.factors[i]); e != nil {
			return e
		}
	}
	// Outer vector reset destroys each actual inner vector in ascending order.
	for i := uint32(0); i < fs.count; i++ {
		m.freeBacking(&fs.factors[i])
	}
	fs.count = 0
	fs.degrees.size = 0
	if e := m.modular.SetInt32(&fs.constant, 1); e != nil {
		return e
	}
	fs.totalFactors = 0
	fs.totalDegree = 0
	return nil
}
func (fs *UPolynomialFactors) Reset() error { return fs.Clear() }
func (fs *UPolynomialFactors) Close() error {
	if e := fs.check(); e != nil {
		return e
	}
	m := fs.manager
	// Clear must finish before raw member destruction. A leaf cleanup refusal
	// leaves this actual owner, backing and liveFactors retain available to retry.
	if !fs.closeCleared {
		if e := fs.Clear(); e != nil {
			return e
		}
		fs.closeCleared = true
	}
	if e := m.modular.Del(&fs.constant); e != nil {
		return e
	}
	// Reverse source member destruction after destructor body: degrees, factors.
	b := m.config.Integer.budget
	b.Release(uint64(cap(fs.degrees.values)) * 4)
	fs.degrees = upolySequenceUnsigned{}
	b.Release(uint64(cap(fs.factors)) * uint64(reflect.TypeOf(upolyVectorState{}).Size()))
	fs.factors = nil
	b.Release(fs.header)
	m.liveFactors--
	fs.closed = true
	return nil
}
func (fs *UPolynomialFactors) multiply(out *upolyVectorState) error {
	m := fs.manager
	if e := m.reset(out); e != nil {
		return e
	}
	zero, e := m.modular.IsZero(&fs.constant)
	if e != nil {
		return e
	}
	if zero {
		if modularSourceDebug && fs.count != 0 {
			return ErrUPolynomialAssertion
		}
		return nil
	}
	value, e := m.pushDefault(out)
	if e != nil {
		return e
	}
	if e = m.modular.Copy(value, &fs.constant); e != nil {
		return e
	}
	for i := uint32(0); i < fs.count; i++ {
		degree, e := fs.multiplicity(i)
		if e != nil {
			return e
		}
		v := m.stateView(&fs.factors[i])
		if degree > 1 {
			var power upolyVectorState
			// Source lexical power vector is freed before the next loop iteration.
			e = fs.multiplyPower(out, v, degree, &power)
			if e != nil {
				return e
			}
		} else {
			if e = m.mul(m.stateView(out), v, out); e != nil {
				return e
			}
		}
	}
	return nil
}
func (fs *UPolynomialFactors) multiplyPower(out *upolyVectorState, v UPolynomialView, degree uint32, power *upolyVectorState) error {
	m := fs.manager
	defer m.dropLocalVector(power)
	if e := m.powerState(v, degree, power); e != nil {
		return e
	}
	if e := m.mul(m.stateView(out), m.stateView(power), out); e != nil {
		return e
	}
	// Source explicit numeric reset precedes lexical raw-vector destruction.
	return m.reset(power)
}
func (fs *UPolynomialFactors) Multiply(out UPolynomial) error {
	if e := fs.check(); e != nil {
		return e
	}
	d, e := fs.manager.descriptor(out)
	if e != nil {
		return e
	}
	return fs.multiply(&d.state)
}
func (fs *UPolynomialFactors) Stats() UPolynomialFactorsStats {
	if fs == nil || fs.self != fs || fs.closed || fs.manager == nil || fs.manager.closed {
		return UPolynomialFactorsStats{}
	}
	st := UPolynomialFactorsStats{OwnerBytes: fs.header, Count: fs.count, MultiplicityCount: fs.degrees.size, Capacity: uint32(cap(fs.factors)), MultiplicityCapacity: uint32(cap(fs.degrees.values)), OuterBytes: uint64(cap(fs.factors)) * uint64(reflect.TypeOf(upolyVectorState{}).Size()), MultiplicityBytes: uint64(cap(fs.degrees.values)) * 4}
	z := fs.manager.config.Integer
	if validNumericRef(z.cells, fs.constant.cell) {
		st.OwnedNumeralCells++
	}
	for i := uint32(0); i < fs.count; i++ {
		s := &fs.factors[i]
		st.ManagerBackingBytes += uint64(s.capacity) * uint64(reflect.TypeOf(Int{}).Size())
		for j := uint32(0); j < s.size; j++ {
			if validNumericRef(z.cells, fs.manager.values(s)[j].cell) {
				st.OwnedNumeralCells++
			}
		}
	}
	st.TotalBytes = st.OwnerBytes + st.OuterBytes + st.MultiplicityBytes
	return st
}
