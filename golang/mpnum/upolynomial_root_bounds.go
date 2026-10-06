// Source: src/math/polynomial/upolynomial.cpp:1298-1321.
// Source: src/math/polynomial/upolynomial.cpp:1906-1951 (Cauchy bound),
// src/math/polynomial/upolynomial.cpp:1995-2074 (Knuth/nonzero bounds), related root-bound bodies.
// Licensed under the MIT license.
package mpnum

func (m *UPolynomialManager) HasZeroRoots(v UPolynomialView) (bool, error) {
	if e := m.check(); e != nil {
		return false, e
	}
	if modularSourceDebug && v.count == 0 {
		return false, ErrUPolynomialAssertion
	}
	a, e := m.read(v, 0)
	if e != nil {
		return false, e
	}
	return m.modular.IsZero(a)
}
func (m *UPolynomialManager) RemoveZeroRoots(v UPolynomialView, out UPolynomial) error {
	d, e := m.descriptor(out)
	if e != nil {
		return e
	}
	return m.removeZeroRootsState(v, &d.state)
}
func (m *UPolynomialManager) removeZeroRootsState(v UPolynomialView, out *upolyVectorState) error {
	if modularSourceDebug && v.count == 0 {
		return ErrUPolynomialAssertion
	}
	a, e := m.read(v, 0)
	if e != nil {
		return e
	}
	zero, e := m.modular.IsZero(a)
	if e != nil {
		return e
	}
	if !zero {
		return m.setView(out, v)
	}
	i := uint32(0)
	for {
		if modularSourceDebug && i >= v.count {
			return ErrUPolynomialAssertion
		}
		a, e = m.read(v, i)
		if e != nil {
			return e
		}
		zero, e = m.modular.IsZero(a)
		if e != nil {
			return e
		}
		if !zero {
			break
		}
		i++
	}
	n := v.count - i
	if e = m.reserve(out, n); e != nil {
		return e
	}
	for j := uint32(0); j < n; j++ {
		a, e = m.read(v, j+i)
		if e != nil {
			return e
		}
		if e = m.modular.Copy(&m.values(out)[j], a); e != nil {
			return e
		}
	}
	return m.setSize(out, n)
}
func (m *UPolynomialManager) HasOneHalfRoot(v UPolynomialView) (result bool, cause error) {
	if e := m.check(); e != nil {
		return false, e
	}
	if v.count == 0 {
		return true, nil
	}
	if v.count == 1 {
		return false, nil
	}
	k := uint32(1)
	q, e := m.beginOneHalfRoot()
	if e != nil {
		return false, e
	}
	q.target = v
	z := m.config.Integer
	old, oldN := z.oneHalfRootCaller, z.natural.oneHalfRootCaller
	z.oneHalfRootCaller = m
	z.natural.oneHalfRootCaller = m
	defer func() {
		q.executing = false
		cause = q.finish(cause)
		z.oneHalfRootCaller = old
		z.natural.oneHalfRootCaller = oldN
	}()
	r, ak := &q.values[0], &q.values[1]
	context := q.context()
	a, e := m.read(v, v.count-1)
	if e != nil {
		return false, e
	}
	if e = m.modular.copyWithOneHalfRootContext(r, a, context); e != nil {
		return false, e
	}
	for i := v.count - 1; i > 0; {
		i--
		a, e = m.read(v, i)
		if e != nil {
			return false, e
		}
		if e = m.modular.mul2kWithOneHalfRootContext(a, k, ak, context); e != nil {
			return false, e
		}
		if e = m.modular.addWithOneHalfRootContext(r, ak, r, context); e != nil {
			return false, e
		}
		k++
	}
	return m.modular.IsZero(r)
}
func (m *UPolynomialManager) RemoveOneHalfRoot(v UPolynomialView, out UPolynomial) error {
	d, e := m.descriptor(out)
	if e != nil {
		return e
	}
	if modularSourceDebug {
		yes, e := m.HasOneHalfRoot(v)
		if e != nil {
			return e
		}
		if !yes {
			return ErrUPolynomialAssertion
		}
	}
	divisor := upolyDivisorSpan{immediate: true, pair: [2]Int{NewInt(-1), NewInt(2)}}
	return m.divSpanState(v, &divisor, &d.state)
}
func (m *UPolynomialManager) SignChanges(v UPolynomialView) (uint32, error) {
	if e := m.check(); e != nil {
		return 0, e
	}
	var changes uint32
	previous := 0
	for i := uint32(0); i < v.count; i++ {
		sign, e := m.signAt(v, i)
		if e != nil {
			return 0, e
		}
		if sign == 0 {
			continue
		}
		if sign != previous && previous != 0 {
			changes++
		}
		previous = sign
	}
	return changes, nil
}
func (m *UPolynomialManager) DescartesBound(v UPolynomialView) (uint32, error) {
	return m.SignChanges(v)
}
func (m *UPolynomialManager) DescartesBoundZeroOne(v UPolynomialView) (uint32, error) {
	if e := m.check(); e != nil {
		return 0, e
	}
	if v.count <= 1 {
		return 0, nil
	}
	q := &m.db
	if e := m.setView(q, v); e != nil {
		return 0, e
	}
	var changes uint32
	previous := 0
	for i := uint32(0); i < v.count; i++ {
		if e := m.checkpoint(); e != nil {
			return 0, e
		}
		k := uint32(1)
		for ; k < v.count-i; k++ {
			a, e := m.read(m.stateView(q).WithCount(v.count), k)
			if e != nil {
				return 0, e
			}
			prev, e := m.read(m.stateView(q).WithCount(v.count), k-1)
			if e != nil {
				return 0, e
			}
			if e = m.modular.Add(a, prev, a); e != nil {
				return 0, e
			}
		}
		sign, e := m.signAt(m.stateView(q).WithCount(v.count), k-1)
		if e != nil {
			return 0, e
		}
		if sign == 0 {
			continue
		}
		if sign != previous && previous != 0 {
			changes++
			if changes > 1 {
				return changes, nil
			}
		}
		previous = sign
	}
	return changes, nil
}
func (m *UPolynomialManager) DescartesBoundInterval(v UPolynomialView, bqm *BinaryRationalManager, a, b *BinaryRational) (uint32, error) {
	if e := m.check(); e != nil {
		return 0, e
	}
	if bqm == nil || bqm.z != m.config.Integer {
		return 0, ErrUPolynomialOwner
	}
	nonnegative, e := bqm.IsNonNeg(a)
	if e != nil {
		return 0, e
	}
	if nonnegative {
		aux := &m.dbab1
		if e = m.setView(aux, v); e != nil {
			return 0, e
		}
		if e = m.TranslateBinaryViewInPlace(m.stateView(aux).WithCount(v.count), a); e != nil {
			return 0, e
		}
		var difference BinaryRational
		defer bqm.Del(&difference)
		if e = bqm.Sub(b, a, &difference); e != nil {
			return 0, e
		}
		if e = m.ComposeBinaryXViewInPlace(m.stateView(aux), &difference); e != nil {
			return 0, e
		}
		return m.DescartesBoundZeroOne(m.stateView(aux))
	}
	nonpositive, e := bqm.IsNonPos(b)
	if e != nil {
		return 0, e
	}
	if nonpositive {
		aux := &m.dbab2
		if e = m.setView(aux, v); e != nil {
			return 0, e
		}
		if e = m.PMinusXViewInPlace(m.stateView(aux)); e != nil {
			return 0, e
		}
		var mb, ma BinaryRational
		defer bqm.Del(&mb)
		defer bqm.Del(&ma)
		if e = bqm.Copy(&mb, b); e != nil {
			return 0, e
		}
		if e = bqm.Neg(&mb); e != nil {
			return 0, e
		}
		if e = bqm.Copy(&ma, a); e != nil {
			return 0, e
		}
		if e = bqm.Neg(&ma); e != nil {
			return 0, e
		}
		return m.DescartesBoundInterval(m.stateView(aux), bqm, &mb, &ma)
	}
	hasZero, e := m.HasZeroRoots(v)
	if e != nil {
		return 0, e
	}
	zero := NewBinaryInt(0)
	if !hasZero {
		r1, e := m.DescartesBoundInterval(v, bqm, a, &zero)
		if e != nil {
			return 0, e
		}
		if r1 > 1 {
			return r1, nil
		}
		r2, e := m.DescartesBoundInterval(v, bqm, &zero, b)
		if e != nil {
			return 0, e
		}
		if r1 == 0 {
			return r2, nil
		}
		if r2 == 0 {
			return r1, nil
		}
		return 2, nil
	}
	r1, e := m.DescartesBoundInterval(v, bqm, a, &zero)
	if e != nil {
		return 0, e
	}
	if r1 > 0 {
		return 2, nil
	}
	r2, e := m.DescartesBoundInterval(v, bqm, &zero, b)
	if e != nil {
		return 0, e
	}
	if r2 > 0 {
		return 2, nil
	}
	return 1, nil
}
func (m *UPolynomialManager) RootUpperBound(v UPolynomialView, out *Int) (cause error) {
	if e := m.check(); e != nil {
		return e
	}
	if modularSourceDebug && v.count == 0 {
		return ErrUPolynomialAssertion
	}
	if modularSourceDebug {
		top, e := m.read(v, v.count-1)
		if e != nil {
			return e
		}
		zero, e := m.modular.IsZero(top)
		if e != nil {
			return e
		}
		if zero {
			return ErrUPolynomialAssertion
		}
	}
	q, e := m.beginRootUpperBound()
	if e != nil {
		return e
	}
	q.target = v
	context := q.context()
	z := m.config.Integer
	prior := z.rootUpperBoundCaller
	z.rootUpperBoundCaller = m
	priorNatural := z.natural.rootUpperBoundCaller
	z.natural.rootUpperBoundCaller = m
	defer func() {
		q.executing = false
		cause = q.finish(cause)
		z.natural.rootUpperBoundCaller = priorNatural
		z.rootUpperBoundCaller = prior
	}()
	maximum, minimum, an, r2 := &q.values[0], &q.values[1], &q.values[2], &q.values[3]
	top, e := m.read(v, v.count-1)
	if e != nil {
		return e
	}
	if e = m.modular.copyWithRootUpperContext(an, top, context); e != nil {
		return e
	}
	if e = m.modular.absWithRootUpperContext(an, context); e != nil {
		return e
	}
	q.constructed = 5
	q.sourceRemaining = 5
	c := &q.values[4]
	initialized := false
	for i := uint32(0); i < v.count; i++ {
		a, e := m.read(v, i)
		if e != nil {
			return e
		}
		zero, e := m.modular.IsZero(a)
		if e != nil {
			return e
		}
		if zero {
			continue
		}
		if e = m.modular.copyWithRootUpperContext(c, a, context); e != nil {
			return e
		}
		if e = m.modular.absWithRootUpperContext(c, context); e != nil {
			return e
		}
		if !initialized {
			if e = m.modular.copyWithRootUpperContext(maximum, c, context); e != nil {
				return e
			}
			if e = m.modular.copyWithRootUpperContext(minimum, c, context); e != nil {
				return e
			}
			initialized = true
			continue
		}
		greater, e := m.modular.Greater(c, maximum)
		if e != nil {
			return e
		}
		if greater {
			if e = m.modular.copyWithRootUpperContext(maximum, c, context); e != nil {
				return e
			}
		}
		less, e := m.modular.Less(c, minimum)
		if e != nil {
			return e
		}
		if less {
			if e = m.modular.copyWithRootUpperContext(minimum, c, context); e != nil {
				return e
			}
		}
	}
	one := NewInt(1)
	if e = m.modular.addWithRootUpperContext(minimum, maximum, out, context); e != nil {
		return e
	}
	if e = m.modular.divWithRootUpperContext(out, minimum, out, context); e != nil {
		return e
	}
	if e = m.modular.addWithRootUpperContext(out, &one, out, context); e != nil {
		return e
	}
	if e = m.modular.addWithRootUpperContext(an, maximum, r2, context); e != nil {
		return e
	}
	if e = m.modular.divWithRootUpperContext(r2, an, r2, context); e != nil {
		return e
	}
	if e = m.modular.addWithRootUpperContext(r2, &one, r2, context); e != nil {
		return e
	}
	less, e := m.modular.Less(r2, out)
	if e != nil {
		return e
	}
	if less {
		*out, *r2 = *r2, *out
	}
	if modularSourceDebug {
		le, e := m.modular.LessEqual(out, r2)
		if e != nil {
			return e
		}
		if !le {
			return ErrUPolynomialAssertion
		}
	}
	return nil
}
func (m *UPolynomialManager) KnuthPositiveRootUpperBound(v UPolynomialView) (uint32, error) {
	if e := m.check(); e != nil {
		return 0, e
	}
	if v.count == 0 {
		return 0, nil
	}
	n := v.count - 1
	a, e := m.read(v, n)
	if e != nil {
		return 0, e
	}
	positive, e := m.modular.IsPos(a)
	if e != nil {
		return 0, e
	}
	var logarithm uint32
	if positive {
		logarithm, e = m.modular.Log2(a)
	} else {
		logarithm, e = m.modular.MLog2(a)
	}
	if e != nil {
		return 0, e
	}
	maximum := uint32(0)
	for k := uint32(1); k <= n; k++ {
		c, e := m.read(v, n-k)
		if e != nil {
			return 0, e
		}
		zero, e := m.modular.IsZero(c)
		if e != nil {
			return 0, e
		}
		if zero {
			continue
		}
		pos, e := m.modular.IsPos(c)
		if e != nil {
			return 0, e
		}
		if pos == positive {
			continue
		}
		var log uint32
		if pos {
			log, e = m.modular.Log2(c)
		} else {
			log, e = m.modular.MLog2(c)
		}
		if e != nil {
			return 0, e
		}
		if logarithm > log {
			continue
		}
		current := log + 1 - logarithm
		remainder := current % k
		current /= k
		if remainder != 0 {
			current++
		}
		if current > maximum {
			maximum = current
		}
	}
	return maximum + 1, nil
}
func (m *UPolynomialManager) KnuthNegativeRootUpperBound(v UPolynomialView) (bound uint32, cause error) {
	if e := m.check(); e != nil {
		return 0, e
	}
	q, e := m.beginKnuth(v, 1)
	if e != nil {
		return 0, e
	}
	z := m.config.Integer
	old, oldN := z.knuthCaller, z.natural.knuthCaller
	z.knuthCaller = m
	z.natural.knuthCaller = m
	defer func() {
		q.executing = false
		cause = q.finish(cause)
		z.knuthCaller = old
		z.natural.knuthCaller = oldN
	}()
	return m.knuthNegativeWithContext(v, q.context())
}
func (m *UPolynomialManager) knuthNegativeWithContext(v UPolynomialView, context integerAllocationInvocationContext) (uint32, error) {
	if q := context.knuthFrame(m.config.Integer); q != nil {
		q.stage = 1
	}
	if e := m.pMinusXWithSourceContext(v, context); e != nil {
		return 0, e
	}
	if q := context.knuthFrame(m.config.Integer); q != nil {
		q.stage = 2
	}
	r, e := m.KnuthPositiveRootUpperBound(v)
	if e != nil {
		return 0, e
	}
	if q := context.knuthFrame(m.config.Integer); q != nil {
		q.stage = 3
	}
	if e = m.pMinusXWithSourceContext(v, context); e != nil {
		return 0, e
	}
	return r, nil
}
func (m *UPolynomialManager) NonzeroRootLowerBound(v UPolynomialView) (bound uint32, cause error) {
	if e := m.check(); e != nil {
		return 0, e
	}
	q, e := m.beginKnuth(v, 2)
	if e != nil {
		return 0, e
	}
	z := m.config.Integer
	old, oldN := z.knuthCaller, z.natural.knuthCaller
	z.knuthCaller = m
	z.natural.knuthCaller = m
	defer func() {
		q.executing = false
		cause = q.finish(cause)
		z.knuthCaller = old
		z.natural.knuthCaller = oldN
	}()
	return m.nonzeroWithContext(v, q.context())
}
func (m *UPolynomialManager) nonzeroWithContext(v UPolynomialView, context integerAllocationInvocationContext) (uint32, error) {
	if e := m.check(); e != nil {
		return 0, e
	}
	if modularSourceDebug && v.count == 0 {
		return 0, ErrUPolynomialAssertion
	}
	i := uint32(0)
	for {
		if modularSourceDebug && i >= v.count {
			return 0, ErrUPolynomialAssertion
		}
		a, e := m.read(v, i)
		if e != nil {
			return 0, e
		}
		zero, e := m.modular.IsZero(a)
		if e != nil {
			return 0, e
		}
		if !zero {
			break
		}
		i++
	}
	if modularSourceDebug {
		if i >= v.count {
			return 0, ErrUPolynomialAssertion
		}
		a, e := m.read(v, i)
		if e != nil {
			return 0, e
		}
		zero, e := m.modular.IsZero(a)
		if e != nil {
			return 0, e
		}
		if zero {
			return 0, ErrUPolynomialAssertion
		}
	}
	// The successful indexed read establishes physical representability of start+i.
	nz := v
	nz.start += i
	nz.count -= i
	if q := context.knuthFrame(m.config.Integer); q != nil {
		q.stage = 4
	}
	if e := m.ReciprocalViewInPlace(nz); e != nil {
		return 0, e
	}
	k1, e := m.KnuthPositiveRootUpperBound(nz)
	if e != nil {
		return 0, e
	}
	k2, e := m.knuthNegativeWithContext(nz, context)
	if e != nil {
		return 0, e
	}
	if q := context.knuthFrame(m.config.Integer); q != nil {
		q.stage = 5
	}
	if e = m.ReciprocalViewInPlace(nz); e != nil {
		return 0, e
	}
	if k2 > k1 {
		return k2, nil
	}
	return k1, nil
}
