// Licensed under the MIT license.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"math"
)

func (m *UPolynomialManager) wholeView(p UPolynomial) (UPolynomialView, error) {
	d, e := m.descriptor(p)
	if e != nil {
		return UPolynomialView{}, e
	}
	return m.stateView(&d.state), nil
}
func (m *UPolynomialManager) SetRationals(p UPolynomial, a []Rational) error {
	d, e := m.descriptor(p)
	if e != nil {
		return e
	}
	if uint64(len(a)) > math.MaxUint32 {
		return arena.ErrBounds
	}
	n := uint32(len(a))
	if e = m.reserve(&d.state, n); e != nil {
		return e
	}
	dst := m.values(&d.state)
	for i := uint32(0); i < n; i++ {
		if modularSourceDebug {
			one, e := m.config.Integer.IsOne(&a[i].den)
			if e != nil {
				return e
			}
			if !one {
				return ErrUPolynomialAssertion
			}
		}
		if e = m.modular.Copy(&dst[i], &a[i].num); e != nil {
			return e
		}
	}
	return m.setSize(&d.state, n)
}
func (m *UPolynomialManager) NegInPlace(p UPolynomial) error {
	v, e := m.wholeView(p)
	if e != nil {
		return e
	}
	return m.NegViewInPlace(v)
}
func (m *UPolynomialManager) NegViewInPlace(v UPolynomialView) error {
	return m.negViewInPlaceWithSourceContext(v, integerAllocationInvocationContext{})
}
func (m *UPolynomialManager) negViewInPlaceWithSourceContext(v UPolynomialView, context integerAllocationInvocationContext) error {
	if e := m.check(); e != nil {
		return e
	}
	for i := uint32(0); i < v.count; i++ {
		a, e := m.read(v, i)
		if e != nil {
			return e
		}
		if e = m.modular.negWithSourceContext(a, context); e != nil {
			return e
		}
	}
	return nil
}
func (m *UPolynomialManager) Neg(p, out UPolynomial) error {
	v, e := m.wholeView(p)
	if e != nil {
		return e
	}
	return m.NegView(v, out)
}
func (m *UPolynomialManager) NegView(v UPolynomialView, out UPolynomial) error {
	d, e := m.descriptor(out)
	if e != nil {
		return e
	}
	s := &m.basic
	if modularSourceDebug && m.alias(v, s) {
		return ErrUPolynomialAssertion
	}
	if e = m.reserve(s, v.count); e != nil {
		return e
	}
	for i := uint32(0); i < v.count; i++ {
		a, e := m.read(v, i)
		if e != nil {
			return e
		}
		if e = m.modular.Copy(&m.values(s)[i], a); e != nil {
			return e
		}
		if e = m.modular.Neg(&m.values(s)[i]); e != nil {
			return e
		}
	}
	if e = m.setSize(s, v.count); e != nil {
		return e
	}
	d.state, *s = *s, d.state
	return nil
}
func (m *UPolynomialManager) addSubViews(a, b UPolynomialView, out UPolynomial, sub bool) error {
	d, e := m.descriptor(out)
	if e != nil {
		return e
	}
	return m.addSubState(a, b, &d.state, sub)
}
func (m *UPolynomialManager) addSubState(a, b UPolynomialView, out *upolyVectorState, sub bool) error {
	return m.addSubStateWithHensel(a, b, out, sub, nil)
}
func (m *UPolynomialManager) addSubStateWithHensel(a, b UPolynomialView, out *upolyVectorState, sub bool, receipt *upolyHenselPairInvocationReceipt) error {
	context := m.modular.henselPairAllocationContext(receipt)
	if receipt != nil && context.henselPairReceipt(m.config.Integer) == nil {
		return ErrUPolynomialOwner
	}
	return m.addSubStateWithSourceContext(a, b, out, sub, context)
}
func (m *UPolynomialManager) addSubStateWithSourceContext(a, b UPolynomialView, out *upolyVectorState, sub bool, context integerAllocationInvocationContext) error {

	var e error
	s := &m.basic
	if modularSourceDebug && (m.alias(a, s) || m.alias(b, s)) {
		return ErrUPolynomialAssertion
	}
	lo, hi := a.count, b.count
	if lo > hi {
		lo, hi = hi, lo
	}
	if e = m.reserve(s, hi); e != nil {
		return e
	}
	i := uint32(0)
	for ; i < lo; i++ {
		x, e := m.read(a, i)
		if e != nil {
			return e
		}
		y, e := m.read(b, i)
		if e != nil {
			return e
		}
		dst := &m.values(s)[i]
		if sub {
			e = m.modular.subWithSourceContext(x, y, dst, context)
		} else {
			e = m.modular.addWithSourceContext(x, y, dst, context)
		}
		if e != nil {
			return e
		}
	}
	for ; i < a.count; i++ {
		x, e := m.read(a, i)
		if e != nil {
			return e
		}
		if e = m.modular.copyWithSourceContext(&m.values(s)[i], x, context); e != nil {
			return e
		}
	}
	for ; i < b.count; i++ {
		y, e := m.read(b, i)
		if e != nil {
			return e
		}
		dst := &m.values(s)[i]
		if e = m.modular.copyWithSourceContext(dst, y, context); e != nil {
			return e
		}
		if sub {
			if e = m.modular.negWithSourceContext(dst, context); e != nil {
				return e
			}
		}
	}
	if e = m.setSize(s, hi); e != nil {
		return e
	}
	*out, *s = *s, *out
	return nil
}
func (m *UPolynomialManager) Add(p, q, out UPolynomial) error {
	a, e := m.wholeView(p)
	if e != nil {
		return e
	}
	b, e := m.wholeView(q)
	if e != nil {
		return e
	}
	return m.AddViews(a, b, out)
}
func (m *UPolynomialManager) Sub(p, q, out UPolynomial) error {
	a, e := m.wholeView(p)
	if e != nil {
		return e
	}
	b, e := m.wholeView(q)
	if e != nil {
		return e
	}
	return m.SubViews(a, b, out)
}
func (m *UPolynomialManager) AddViews(a, b UPolynomialView, out UPolynomial) error {
	return m.addSubViews(a, b, out, false)
}
func (m *UPolynomialManager) SubViews(a, b UPolynomialView, out UPolynomial) error {
	return m.addSubViews(a, b, out, true)
}
func (m *UPolynomialManager) mul(a, b UPolynomialView, out *upolyVectorState) error {
	return m.mulWithHensel(a, b, out, nil)
}
func (m *UPolynomialManager) mulWithHensel(a, b UPolynomialView, out *upolyVectorState, receipt *upolyHenselPairInvocationReceipt) error {
	context := m.modular.henselPairAllocationContext(receipt)
	if receipt != nil && context.henselPairReceipt(m.config.Integer) == nil {
		return ErrUPolynomialOwner
	}
	return m.mulWithSourceContext(a, b, out, henselAddMulChild(receipt), context)
}
func (m *UPolynomialManager) mulWithSourceContext(a, b UPolynomialView, out *upolyVectorState, child *integerAddMulInvocationReceipt, context integerAllocationInvocationContext) error {

	s := &m.basic
	if modularSourceDebug && (m.alias(a, s) || m.alias(b, s)) {
		return ErrUPolynomialAssertion
	}
	if a.count == 0 {
		if e := m.reset(s); e != nil {
			return e
		}
	} else if b.count == 0 {
		if e := m.reset(s); e != nil {
			return e
		}
	} else {
		if uint64(a.count)+uint64(b.count)-1 > math.MaxUint32 {
			return arena.ErrBounds
		}
		n := a.count + b.count - 1
		if e := m.reserve(s, n); e != nil {
			return e
		}
		for i := uint32(0); i < n; i++ {
			if e := m.modular.Reset(&m.values(s)[i]); e != nil {
				return e
			}
		}
		if a.count < b.count {
			a, b = b, a
		}
		for i := uint32(0); i < a.count; i++ {
			if e := m.checkpoint(); e != nil {
				return e
			}
			x, e := m.read(a, i)
			if e != nil {
				return e
			}
			zero, e := m.modular.IsZero(x)
			if e != nil {
				return e
			}
			if zero {
				continue
			}
			for j := uint32(0); j < b.count; j++ {
				y, e := m.read(b, j)
				if e != nil {
					return e
				}
				zero, e := m.modular.IsZero(y)
				if e != nil {
					return e
				}
				if zero {
					continue
				}
				dst := &m.values(s)[i+j]
				if e = m.modular.addMulWithSourceContext(dst, x, y, dst, child, context); e != nil {
					return e
				}
			}
		}
		if e := m.setSize(s, n); e != nil {
			return e
		}
	}
	*out, *s = *s, *out
	return nil
}
func (m *UPolynomialManager) Mul(p, q, out UPolynomial) error {
	a, e := m.wholeView(p)
	if e != nil {
		return e
	}
	b, e := m.wholeView(q)
	if e != nil {
		return e
	}
	return m.MulViews(a, b, out)
}
func (m *UPolynomialManager) MulViews(a, b UPolynomialView, out UPolynomial) error {
	d, e := m.descriptor(out)
	if e != nil {
		return e
	}
	return m.mul(a, b, &d.state)
}
func (m *UPolynomialManager) Derivative(p, out UPolynomial) error {
	v, e := m.wholeView(p)
	if e != nil {
		return e
	}
	return m.DerivativeView(v, out)
}
func (m *UPolynomialManager) DerivativeView(v UPolynomialView, out UPolynomial) error {
	d, e := m.descriptor(out)
	if e != nil {
		return e
	}
	return m.derivativeState(v, &d.state)
}
func (m *UPolynomialManager) derivativeState(v UPolynomialView, s *upolyVectorState) error {
	return m.derivativeStateWithSourceContext(v, s, integerAllocationInvocationContext{})
}

// Source: src/math/polynomial/upolynomial.cpp:351-363 (plain RAW multiplier).
func (m *UPolynomialManager) derivativeStateWithSourceContext(v UPolynomialView, s *upolyVectorState, context integerAllocationInvocationContext) (cause error) {
	var e error
	if v.count <= 1 {
		return m.reset(s)
	}
	if e = m.reserve(s, v.count-1); e != nil {
		return e
	}
	var frame *upolyFactorDerivativeScope
	if owner := context.factorFrontendFrame(m.config.Integer); owner != nil {
		frame = &owner.derivative
		if frame.executing || frame.d.cell != 0 || frame.retired != nil || frame.invocation == ^uint64(0) {
			return ErrUPolynomialCleanupPending
		}
		*frame = upolyFactorDerivativeScope{owner: owner, invocation: frame.invocation + 1, executing: true}
		defer func() {
			frame.executing = false
			frame.bodyReturned = true
			if frame.bodyCause == nil {
				frame.bodyCause = cause
			}
		}()
	}
	for i := uint32(1); i < v.count; i++ {
		var local Int
		multiplier := &local
		if frame != nil {
			if e = frame.retireMultiplier(); e != nil {
				return e
			}
			frame.d = Int{}
			frame.constructed = true
			frame.phase = 1
			multiplier = &frame.d
		}
		if e = m.modular.setUint32WithSourceContext(multiplier, i, context); e != nil {
			return e
		}
		x, e := m.read(v, i)
		if e != nil {
			return e
		}
		if frame != nil {
			frame.phase = 2
		}
		if e = m.modular.mulWithSourceContext(x, multiplier, &m.values(s)[i-1], context); e != nil {
			return e
		}
		if frame != nil {
			frame.phase = 3
		}
	}
	return m.setSize(s, v.count-1)
}
func (m *UPolynomialManager) Equal(p, q UPolynomial) (bool, error) {
	a, e := m.wholeView(p)
	if e != nil {
		return false, e
	}
	b, e := m.wholeView(q)
	if e != nil {
		return false, e
	}
	return m.EqualViews(a, b)
}
func (m *UPolynomialManager) EqualViews(a, b UPolynomialView) (bool, error) {
	if e := m.check(); e != nil {
		return false, e
	}
	if a.count != b.count {
		return false, nil
	}
	for i := uint32(0); i < a.count; i++ {
		x, e := m.read(a, i)
		if e != nil {
			return false, e
		}
		y, e := m.read(b, i)
		if e != nil {
			return false, e
		}
		same, e := m.modular.Equal(x, y)
		if e != nil || !same {
			return false, e
		}
	}
	return true, nil
}
func (m *UPolynomialManager) Power(p UPolynomial, k uint32, out UPolynomial) error {
	v, e := m.wholeView(p)
	if e != nil {
		return e
	}
	return m.PowerView(v, k, out)
}
func (m *UPolynomialManager) PowerView(v UPolynomialView, k uint32, out UPolynomial) error {
	d, e := m.descriptor(out)
	if e != nil {
		return e
	}
	return m.powerState(v, k, &d.state)
}

// powerState preserves core_manager::pw on an actual lexical numeral vector.
func (m *UPolynomialManager) powerState(v UPolynomialView, k uint32, out *upolyVectorState) error {
	var e error
	if k == 0 {
		if modularSourceDebug && v.count == 0 {
			return ErrUPolynomialAssertion
		}
		if e = m.reserve(out, 1); e != nil {
			return e
		}
		if e = m.modular.SetInt32(&m.values(out)[0], 1); e != nil {
			return e
		}
		return m.setSize(out, 1)
	}
	shortcut := k == 1 || v.count == 0
	if !shortcut && v.count == 1 {
		x, e := m.read(v, 0)
		if e != nil {
			return e
		}
		shortcut, e = m.modular.IsOne(x)
		if e != nil {
			return e
		}
	}
	if shortcut {
		return m.setView(out, v)
	}
	if e = m.setView(&m.pw, v); e != nil {
		return e
	}
	for i := uint32(1); i < k; i++ {
		if e = m.mul(m.stateView(&m.pw), v, &m.pw); e != nil {
			return e
		}
	}
	*out, m.pw = m.pw, *out
	return nil
}
func (m *UPolynomialManager) MulScalar(p UPolynomial, b *Int) error {
	d, e := m.descriptor(p)
	if e != nil {
		return e
	}
	zero, e := m.modular.IsZero(b)
	if e != nil {
		return e
	}
	if zero {
		return m.reset(&d.state)
	}
	return m.MulScalarView(m.stateView(&d.state), b)
}
func (m *UPolynomialManager) MulScalarView(v UPolynomialView, b *Int) error {
	return m.MulScalarViewWithHensel(v, b, nil)
}
func (m *UPolynomialManager) MulScalarViewWithHensel(v UPolynomialView, b *Int, receipt *upolyHenselPairInvocationReceipt) error {
	context := m.modular.henselPairAllocationContext(receipt)
	if receipt != nil && context.henselPairReceipt(m.config.Integer) == nil {
		return ErrUPolynomialOwner
	}
	return m.mulScalarViewWithSourceContext(v, b, context)
}
func (m *UPolynomialManager) mulScalarViewWithSourceContext(v UPolynomialView, b *Int, context integerAllocationInvocationContext) error {

	if e := m.check(); e != nil {
		return e
	}
	if modularSourceDebug {
		zero, e := m.modular.IsZero(b)
		if e != nil {
			return e
		}
		if zero {
			return ErrUPolynomialAssertion
		}
	}
	one, e := m.modular.IsOne(b)
	if e != nil {
		return e
	}
	if one {
		return nil
	}
	for i := uint32(0); i < v.count; i++ {
		a, e := m.read(v, i)
		if e != nil {
			return e
		}
		if e = m.modular.mulWithSourceContext(a, b, a, context); e != nil {
			return e
		}
	}
	return nil
}
