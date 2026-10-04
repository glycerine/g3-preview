// Licensed under the MIT license.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"math"
)

type upolyDRSFrame struct{ parent, sizeFlags uint32 }

func newUPolyDRSFrame(parent, size uint32, left bool) upolyDRSFrame {
	f := upolyDRSFrame{parent: parent, sizeFlags: (size & ((1 << 30) - 1)) | (1 << 30)}
	if left {
		f.sizeFlags |= 1 << 31
	}
	return f
}
func (f *upolyDRSFrame) polynomialSize() uint32 { return f.sizeFlags & ((1 << 30) - 1) }
func (f *upolyDRSFrame) firstVisit() bool       { return f.sizeFlags&(1<<30) != 0 }
func (f *upolyDRSFrame) leftChild() bool        { return f.sizeFlags&(1<<31) != 0 }

type upolyDRSStack struct {
	values []upolyDRSFrame
	size   uint32
}

func (m *UPolynomialManager) pushDRS(s *upolyDRSStack, f upolyDRSFrame) error {
	if s.size == math.MaxUint32 {
		return arena.ErrBounds
	}
	if s.size == uint32(cap(s.values)) {
		n, bytes, e := upolyIsolationNextCapacity(uint32(cap(s.values)), 8, 8, uint64(int(^uint(0)>>1)))
		if e != nil {
			return e
		}
		if e = m.config.Integer.budget.Reserve(bytes); e != nil {
			return e
		}
		next := make([]upolyDRSFrame, n)
		copy(next, s.values[:s.size])
		m.config.Integer.budget.Release(uint64(cap(s.values)) * 8)
		s.values = next
	}
	s.values[s.size] = f
	s.size++
	return nil
}
func (m *UPolynomialManager) dropDRS(s *upolyDRSStack) {
	m.config.Integer.budget.Release(uint64(cap(s.values)) * 8)
	*s = upolyDRSStack{}
}
func (m *UPolynomialManager) popDRS(p *upolyVectorState, s *upolyDRSStack) error {
	if s.size == 0 {
		return ErrUPolynomialAssertion
	}
	n := s.values[s.size-1].polynomialSize()
	if modularSourceDebug && n > p.size {
		return ErrUPolynomialAssertion
	}
	for i := uint32(0); i < n; i++ {
		if p.size == 0 {
			return arena.ErrBounds
		}
		if e := m.modular.Del(&m.values(p)[p.size-1]); e != nil {
			return e
		}
		p.size--
	}
	s.size--
	return nil
}
func (m *UPolynomialManager) pushDRSChildren(sz uint32, v UPolynomialView, p *upolyVectorState, s *upolyDRSStack) error {
	parent := uint32(math.MaxUint32)
	if s.size != 0 {
		parent = s.size - 1
	}
	aux := &m.push
	if e := m.setView(aux, v.WithCount(sz)); e != nil {
		return e
	}
	if e := m.ComposeTwoDegreeDivideXViewInPlace(m.stateView(aux)); e != nil {
		return e
	}
	if e := m.normalize(m.stateView(aux)); e != nil {
		return e
	}
	for i := uint32(0); i < sz; i++ {
		dst, e := m.pushDefault(p)
		if e != nil {
			return e
		}
		src, e := m.read(m.stateView(aux), i)
		if e != nil {
			return e
		}
		if e = m.modular.Copy(dst, src); e != nil {
			return e
		}
	}
	if e := m.pushDRS(s, newUPolyDRSFrame(parent, sz, true)); e != nil {
		return e
	}
	if p.size < sz {
		return arena.ErrBounds
	}
	left := m.stateView(p)
	left.start = p.size - sz
	left.count = sz
	if e := m.setView(aux, left); e != nil {
		return e
	}
	if e := m.TranslateOneViewInPlace(m.stateView(aux).WithCount(sz)); e != nil {
		return e
	}
	if e := m.normalize(m.stateView(aux)); e != nil {
		return e
	}
	for i := uint32(0); i < sz; i++ {
		dst, e := m.pushDefault(p)
		if e != nil {
			return e
		}
		src, e := m.read(m.stateView(aux), i)
		if e != nil {
			return e
		}
		*dst, *src = *src, *dst
	}
	return m.pushDRS(s, newUPolyDRSFrame(parent, sz, false))
}
func (b *BinaryRationalManager) binaryLiteral(s *upolyBinaryVectorState, value BinaryRational) error {
	dst, e := b.binaryPushDefault(s)
	if e != nil {
		return e
	}
	*dst = value
	return nil
}
func (m *UPolynomialManager) addDRSInterval(s *upolyDRSStack, b *BinaryRationalManager, l, u *BinaryRationalVector) error {
	lower, upper := NewBinaryInt(0), NewBinaryInt(1)
	defer b.Del(&lower)
	defer b.Del(&upper)
	i := s.size - 1
	one := NewInt(1)
	for i != math.MaxUint32 {
		if i >= s.size {
			return arena.ErrBounds
		}
		f := &s.values[i]
		if !f.leftChild() {
			if e := b.AddInteger(&lower, &one, &lower); e != nil {
				return e
			}
			if e := b.AddInteger(&upper, &one, &upper); e != nil {
				return e
			}
		}
		if e := b.Div2(&lower); e != nil {
			return e
		}
		if e := b.Div2(&upper); e != nil {
			return e
		}
		i = f.parent
	}
	if _, e := b.binaryPushDefault(&l.state); e != nil {
		return e
	}
	if _, e := b.binaryPushDefault(&u.state); e != nil {
		return e
	}
	// Reborrow back after both pushes; aliases may relocate shared backing.
	if e := b.Swap(&l.state.values[l.state.size-1], &lower); e != nil {
		return e
	}
	return b.Swap(&u.state.values[u.state.size-1], &upper)
}
func (m *UPolynomialManager) addDRSRoot(s *upolyDRSStack, b *BinaryRationalManager, r *BinaryRationalVector) error {
	value := NewBinaryIntPower(1, 1)
	defer b.Del(&value)
	i := s.size - 1
	one := NewInt(1)
	for i != math.MaxUint32 {
		if i >= s.size {
			return arena.ErrBounds
		}
		f := &s.values[i]
		if !f.leftChild() {
			if e := b.AddInteger(&value, &one, &value); e != nil {
				return e
			}
		}
		if e := b.Div2(&value); e != nil {
			return e
		}
		i = f.parent
	}
	if _, e := b.binaryPushDefault(&r.state); e != nil {
		return e
	}
	return b.Swap(&r.state.values[r.state.size-1], &value)
}
func (m *UPolynomialManager) drsZeroOne(v UPolynomialView, b *BinaryRationalManager, r, l, u *BinaryRationalVector) error {
	k, e := m.DescartesBoundZeroOne(v)
	if e != nil {
		return e
	}
	if k == 0 {
		return nil
	}
	if k == 1 {
		if e = b.binaryLiteral(&l.state, NewBinaryInt(0)); e != nil {
			return e
		}
		return b.binaryLiteral(&u.state, NewBinaryInt(1))
	}
	var q, p upolyVectorState
	defer m.dropLocalVector(&q)
	defer m.dropLocalVector(&p)
	var s upolyDRSStack
	defer m.dropDRS(&s)
	half, e := m.HasOneHalfRoot(v)
	if e != nil {
		return e
	}
	if half {
		if e = b.binaryLiteral(&r.state, NewBinaryIntPower(1, 1)); e != nil {
			return e
		}
		divisor := upolyDivisorSpan{immediate: true, pair: [2]Int{NewInt(-1), NewInt(2)}}
		if modularSourceDebug {
			yes, e := m.HasOneHalfRoot(v)
			if e != nil {
				return e
			}
			if !yes {
				return ErrUPolynomialAssertion
			}
		}
		if e = m.divSpanState(v, &divisor, &q); e != nil {
			return e
		}
		if e = m.pushDRSChildren(q.size, m.stateView(&q), &p, &s); e != nil {
			return e
		}
	} else {
		if e = m.pushDRSChildren(v.count, v, &p, &s); e != nil {
			return e
		}
	}
	if modularSourceDebug && s.size == 0 {
		return ErrUPolynomialAssertion
	}
	for s.size != 0 {
		if e = m.checkpoint(); e != nil {
			return e
		}
		f := &s.values[s.size-1]
		sz := f.polynomialSize()
		if modularSourceDebug && sz > p.size {
			return ErrUPolynomialAssertion
		}
		if sz > p.size {
			return arena.ErrBounds
		}
		current := m.stateView(&p)
		current.start = p.size - sz
		current.count = sz
		if !f.firstVisit() {
			if e = m.popDRS(&p, &s); e != nil {
				return e
			}
			continue
		}
		f.sizeFlags &^= 1 << 30
		k, e = m.DescartesBoundZeroOne(current)
		if e != nil {
			return e
		}
		if k == 0 {
			if e = m.popDRS(&p, &s); e != nil {
				return e
			}
			continue
		}
		if k == 1 {
			if e = m.addDRSInterval(&s, b, l, u); e != nil {
				return e
			}
			if e = m.popDRS(&p, &s); e != nil {
				return e
			}
			continue
		}
		half, e = m.HasOneHalfRoot(current)
		if e != nil {
			return e
		}
		if half {
			if e = m.addDRSRoot(&s, b, r); e != nil {
				return e
			}
			if modularSourceDebug {
				yes, e := m.HasOneHalfRoot(current)
				if e != nil {
					return e
				}
				if !yes {
					return ErrUPolynomialAssertion
				}
			}
			divisor := upolyDivisorSpan{immediate: true, pair: [2]Int{NewInt(-1), NewInt(2)}}
			if e = m.divSpanState(current, &divisor, &q); e != nil {
				return e
			}
			if e = m.pushDRSChildren(q.size, m.stateView(&q), &p, &s); e != nil {
				return e
			}
		} else {
			if e = m.pushDRSChildren(sz, current, &p, &s); e != nil {
				return e
			}
		}
	}
	return nil
}
func (b *BinaryRationalManager) adjustBinary(v *BinaryRationalVector, start, k uint32, negative bool) error {
	n := v.state.size
	for i := start; i < n; i++ {
		slot, e := b.binarySlot(&v.state, i)
		if e != nil {
			return e
		}
		if e = b.Mul2k(slot, k); e != nil {
			return e
		}
		if negative {
			if e = b.Neg(slot); e != nil {
				return e
			}
		}
	}
	return nil
}
func (m *UPolynomialManager) DRSIsolateRootsWithBounds(v UPolynomialView, neg, pos uint32, b *BinaryRationalManager, r, l, u *BinaryRationalVector) error {
	if e := m.isolationOwners(b, r, l, u); e != nil {
		return e
	}
	var aux upolyVectorState
	defer m.dropLocalVector(&aux)
	if e := m.setView(&aux, v); e != nil {
		return e
	}
	if neg > pos {
		pos = neg
	}
	if e := m.ComposePowerTwoXViewInPlace(m.stateView(&aux).WithCount(v.count), pos); e != nil {
		return e
	}
	oldR, oldL := r.state.size, l.state.size
	if e := m.drsZeroOne(m.stateView(&aux).WithCount(v.count), b, r, l, u); e != nil {
		return e
	}
	if modularSourceDebug && l.state.size != u.state.size {
		return ErrUPolynomialAssertion
	}
	for _, x := range [3]struct {
		v     *BinaryRationalVector
		start uint32
	}{{r, oldR}, {l, oldL}, {u, oldL}} {
		if e := b.adjustBinary(x.v, x.start, pos, false); e != nil {
			return e
		}
	}
	if e := m.PMinusXViewInPlace(v); e != nil {
		return e
	}
	if e := m.ComposePowerTwoXViewInPlace(v, neg); e != nil {
		return e
	}
	oldR, oldL = r.state.size, l.state.size
	if e := m.drsZeroOne(v, b, r, l, u); e != nil {
		return e
	}
	if modularSourceDebug && l.state.size != u.state.size {
		return ErrUPolynomialAssertion
	}
	for _, x := range [3]struct {
		v     *BinaryRationalVector
		start uint32
	}{{r, oldR}, {l, oldL}, {u, oldL}} {
		if e := b.adjustBinary(x.v, x.start, neg, true); e != nil {
			return e
		}
	}
	if modularSourceDebug && l.state.size != u.state.size {
		return ErrUPolynomialAssertion
	}
	n := l.state.size
	for i := oldL; i < n; i++ {
		lower, e := b.binarySlot(&l.state, i)
		if e != nil {
			return e
		}
		upper, e := b.binarySlot(&u.state, i)
		if e != nil {
			return e
		}
		if e = b.Swap(lower, upper); e != nil {
			return e
		}
	}
	return nil
}
func (m *UPolynomialManager) DRSIsolateRoots(v UPolynomialView, b *BinaryRationalManager, r, l, u *BinaryRationalVector) error {
	if e := m.isolationOwners(b, r, l, u); e != nil {
		return e
	}
	if modularSourceDebug {
		if l.state.size != u.state.size {
			return ErrUPolynomialAssertion
		}
		zero, e := m.HasZeroRoots(v)
		if e != nil {
			return e
		}
		if zero {
			return ErrUPolynomialAssertion
		}
	}
	var p upolyVectorState
	defer m.dropLocalVector(&p)
	if e := m.setView(&p, v); e != nil {
		return e
	}
	if e := m.normalize(m.stateView(&p)); e != nil {
		return e
	}
	pos, e := m.KnuthPositiveRootUpperBound(v)
	if e != nil {
		return e
	}
	neg, e := m.KnuthNegativeRootUpperBound(v)
	if e != nil {
		return e
	}
	return m.DRSIsolateRootsWithBounds(m.stateView(&p), neg, pos, b, r, l, u)
}
func (m *UPolynomialManager) squareFreeNonzeroIsolate(v UPolynomialView, b *BinaryRationalManager, r, l, u *BinaryRationalVector) error {
	if modularSourceDebug {
		zero, e := m.HasZeroRoots(v)
		if e != nil {
			return e
		}
		if zero {
			return ErrUPolynomialAssertion
		}
	}
	return m.DRSIsolateRoots(v, b, r, l, u)
}
func (m *UPolynomialManager) SquareFreeIsolateRoots(v UPolynomialView, b *BinaryRationalManager, r, l, u *BinaryRationalVector) error {
	if e := m.isolationOwners(b, r, l, u); e != nil {
		return e
	}
	if e := r.Reset(); e != nil {
		return e
	}
	if e := l.Reset(); e != nil {
		return e
	}
	if e := u.Reset(); e != nil {
		return e
	}
	zero, e := m.HasZeroRoots(v)
	if e != nil {
		return e
	}
	if !zero {
		return m.squareFreeNonzeroIsolate(v, b, r, l, u)
	}
	if e = b.binaryLiteral(&r.state, NewBinaryInt(0)); e != nil {
		return e
	}
	var nz upolyVectorState
	defer m.dropLocalVector(&nz)
	if e = m.removeZeroRootsState(v, &nz); e != nil {
		return e
	}
	if modularSourceDebug {
		zero, e = m.HasZeroRoots(m.stateView(&nz))
		if e != nil {
			return e
		}
		if zero {
			return ErrUPolynomialAssertion
		}
	}
	return m.squareFreeNonzeroIsolate(m.stateView(&nz), b, r, l, u)
}
func (m *UPolynomialManager) IsolateRoots(v UPolynomialView, b *BinaryRationalManager, r, l, u *BinaryRationalVector) error {
	if e := m.isolationOwners(b, r, l, u); e != nil {
		return e
	}
	if modularSourceDebug && v.count == 0 {
		return ErrUPolynomialAssertion
	}
	var sqf upolyVectorState
	defer m.dropLocalVector(&sqf)
	if e := m.squareFree(v, &sqf); e != nil {
		return e
	}
	return m.SquareFreeIsolateRoots(m.stateView(&sqf), b, r, l, u)
}
