// Source: src/math/polynomial/upolynomial.cpp:663-826;
// Licensed under the MIT license.
package mpnum

// modularImage implements the lexical scoped_set_zp body. The old modulus
// copy goes through mpzzp Copy, including source normalization. In modular
// GCD the entering mode is always Z. Cleanup on Go failure is explicit safe
// ownership adaptation; a restoration failure stays visible.
func (m *UPolynomialManager) modularImage(u, v, q *upolyVectorState, p, lc *Int, du, dv uint32) (bool, error) {
	return m.modularImageWithSourceContext(u, v, q, p, lc, du, dv, integerAllocationInvocationContext{})
}
func (m *UPolynomialManager) modularImageWithSourceContext(u, v, q *upolyVectorState, p, lc *Int, du, dv uint32, context integerAllocationInvocationContext) (bad bool, err error) {
	oldMode, e := m.modular.Modular()
	if e != nil {
		return false, e
	}
	var localOld, localC Int
	old, c := &localOld, &localC
	var frame *upolyFactorImageScope
	if owner := context.factorFrontendFrame(m.config.Integer); owner != nil {
		frame = &owner.image
		if frame.savedConstructed || frame.classConstructed || frame.cConstructed || frame.invocation == ^uint64(0) {
			return false, ErrUPolynomialCleanupPending
		}
		*frame = upolyFactorImageScope{owner: owner, invocation: frame.invocation + 1, oldMode: oldMode, savedConstructed: true, executing: true}
		old, c = &frame.old, &frame.c
		defer func() { frame.executing = false; err = frame.finish(err) }()
	} else {
		defer m.modular.Del(old)
	}
	if e = m.modular.copyWithSourceContext(old, &m.modular.p, context); e != nil {
		return false, e
	}
	if frame == nil {
		defer func() {
			var restore error
			if oldMode {
				restore = m.modular.SetZP(old)
			} else {
				restore = m.modular.SetZ()
			}
			if restore != nil {
				err = restore
			}
		}()
	}

	if e = m.modular.setZPWithAllocationContext(p, context); e != nil {
		return false, e
	}
	if frame != nil {
		frame.classConstructed = true
	}
	U, V := &m.mgcd[2], &m.mgcd[3]
	if e = m.setViewWithSourceContext(U, m.stateView(u), context); e != nil {
		return false, e
	}
	if e = m.setViewWithSourceContext(V, m.stateView(v), context); e != nil {
		return false, e
	}
	ud, vd := uint32(0), uint32(0)
	if U.size != 0 {
		ud = U.size - 1
	}
	if V.size != 0 {
		vd = V.size - 1
	}
	if ud < du {
		return true, nil
	}
	if vd < dv {
		return true, nil
	}
	if e = m.euclidGCDWithSourceContext(m.stateView(U), m.stateView(V), q, context); e != nil {
		return false, e
	}
	if e = m.monicLocalWithSourceContext(m.stateView(q), context); e != nil {
		return false, e
	}
	if frame != nil {
		frame.cConstructed = true
	} else {
		defer m.modular.Del(c)
	}
	if e = m.modular.copyWithSourceContext(c, lc, context); e != nil {
		return false, e
	}
	return false, m.mulScalarViewWithSourceContext(m.stateView(q), c, context)
}
func (m *UPolynomialManager) combineImages(c1 *upolyVectorState, b1 *Int, c2 *upolyVectorState, b2 *Int) error {
	return m.combineImagesWithSourceContext(c1, b1, c2, b2, integerAllocationInvocationContext{})
}
func (m *UPolynomialManager) combineImagesWithSourceContext(c1 *upolyVectorState, b1 *Int, c2 *upolyVectorState, b2 *Int, context integerAllocationInvocationContext) (cause error) {
	if modularSourceDebug {
		even, e := m.config.Integer.IsEven(b1)
		if e != nil {
			return e
		}
		if even {
			return ErrUPolynomialAssertion
		}
		even, e = m.config.Integer.IsEven(b2)
		if e != nil {
			return e
		}
		if even {
			return ErrUPolynomialAssertion
		}
	}
	R := &m.cra
	if e := m.reset(R); e != nil {
		return e
	}
	var local [12]Int
	values := &local
	var frame *upolyFactorCRAScope
	if owner := context.factorFrontendFrame(m.config.Integer); owner != nil {
		frame = &owner.cra
		if frame.constructed != 0 || frame.executing || frame.invocation == ^uint64(0) {
			return ErrUPolynomialCleanupPending
		}
		*frame = upolyFactorCRAScope{owner: owner, invocation: frame.invocation + 1, constructed: 3, peak: 3, executing: true}
		values = &frame.values
		defer func() { frame.executing = false; cause = frame.finish(cause) }()
	} else {
		defer func() {
			for i := 11; i >= 0; i-- {
				m.modular.Del(&local[i])
			}
		}()
	}
	inv1, inv2, g, a1, a2, bound, lower, upper, newA, tmp1, tmp2, tmp3 := &values[0], &values[1], &values[2], &values[3], &values[4], &values[5], &values[6], &values[7], &values[8], &values[9], &values[10], &values[11]
	construct := func(n uint8) {
		if frame != nil {
			frame.constructed = n
			frame.peak = n
		}
	}

	if e := m.modular.extendedGCDWithSourceContext(b1, b2, inv1, inv2, g, context); e != nil {
		return e
	}
	if modularSourceDebug {
		one, e := m.modular.IsOne(g)
		if e != nil {
			return e
		}
		if !one {
			return ErrUPolynomialAssertion
		}
	}
	z := m.config.Integer
	if e := z.modWithAllocationContext(inv1, b2, inv1, context); e != nil {
		return e
	}
	if e := z.modWithAllocationContext(inv2, b1, inv2, context); e != nil {
		return e
	}
	construct(5)
	if e := m.modular.mulWithSourceContext(b2, inv2, a1, context); e != nil {
		return e
	}
	if e := m.modular.mulWithSourceContext(b1, inv1, a2, context); e != nil {
		return e
	}
	construct(6)
	if e := m.modular.mulWithSourceContext(b1, b2, bound, context); e != nil {
		return e
	}
	construct(12)
	two := NewInt(2)
	if e := m.modular.divWithAllocationContext(bound, &two, upper, context); e != nil {
		return e
	}
	if e := m.modular.copyWithSourceContext(lower, upper, context); e != nil {
		return e
	}
	if e := m.modular.negWithSourceContext(lower, context); e != nil {
		return e
	}
	zero := NewInt(0)
	n1, n2 := c1.size, c2.size
	n := n1
	if n2 < n {
		n = n2
	}
	add := func(x, y *Int) error {
		if e := m.modular.mulWithSourceContext(x, a1, tmp1, context); e != nil {
			return e
		}
		if e := m.modular.mulWithSourceContext(y, a2, tmp2, context); e != nil {
			return e
		}
		if e := m.modular.addWithSourceContext(tmp1, tmp2, tmp3, context); e != nil {
			return e
		}
		if e := z.modWithAllocationContext(tmp3, bound, newA, context); e != nil {
			return e
		}
		greater, e := m.modular.Greater(newA, upper)
		if e != nil {
			return e
		}
		if greater {
			if e = m.modular.subWithSourceContext(newA, bound, newA, context); e != nil {
				return e
			}
		}
		dst, e := m.pushDefault(R)
		if e != nil {
			return e
		}
		return m.modular.copyWithSourceContext(dst, newA, context)
	}
	i := uint32(0)
	for ; i < n; i++ {
		if e := add(&m.values(c1)[i], &m.values(c2)[i]); e != nil {
			return e
		}
	}
	for ; i < n1; i++ {
		if e := add(&m.values(c1)[i], &zero); e != nil {
			return e
		}
	}
	for ; i < n2; i++ {
		if e := add(&zero, &m.values(c2)[i]); e != nil {
			return e
		}
	}
	if e := m.modular.copyWithSourceContext(b2, bound, context); e != nil {
		return e
	}
	*R, *c2 = *c2, *R
	return nil
}
func (m *UPolynomialManager) modularGCD(a, b UPolynomialView, out *upolyVectorState) error {
	return m.modularGCDWithSourceContext(a, b, out, integerAllocationInvocationContext{})
}
func (m *UPolynomialManager) modularGCDWithSourceContext(a, b UPolynomialView, out *upolyVectorState, context integerAllocationInvocationContext) (cause error) {
	if modularSourceDebug {
		if a.count == 0 || b.count == 0 {
			return ErrUPolynomialAssertion
		}
		finite, e := m.modular.Modular()
		if e != nil {
			return e
		}
		if finite {
			return ErrUPolynomialAssertion
		}
	}
	var local [6]Int
	values := &local
	var frame *upolyFactorModularGCDScope
	if owner := context.factorFrontendFrame(m.config.Integer); owner != nil {
		frame = &owner.modularGCD
		if frame.constructed != 0 || frame.invocation == ^uint64(0) {
			return ErrUPolynomialCleanupPending
		}
		*frame = upolyFactorModularGCDScope{owner: owner, invocation: frame.invocation + 1, constructed: 2, executing: true}
		values = &frame.values
		defer func() { frame.executing = false; cause = frame.finish(cause) }()
	} else {
		defer func() {
			for i := 5; i >= 0; i-- {
				m.modular.Del(&local[i])
			}
		}()
	}
	cu, cv, cg, lcg, p, bound := &values[0], &values[1], &values[2], &values[3], &values[4], &values[5]
	u, v, q, C := &m.mgcd[0], &m.mgcd[1], &m.mgcd[4], &m.mgcd[5]
	if e := m.primitiveStateWithSourceContext(a, u, cu, context); e != nil {
		return e
	}
	if e := m.primitiveStateWithSourceContext(b, v, cv, context); e != nil {
		return e
	}
	if frame != nil {
		frame.constructed = 3
	}
	if e := m.modular.gcdWithSourceContext(cu, cv, cg, context); e != nil {
		return e
	}
	du, dv := a.count-1, b.count-1
	lu, e := m.read(m.stateView(u), du)
	if e != nil {
		return e
	}
	lv, e := m.read(m.stateView(v), dv)
	if e != nil {
		return e
	}
	if frame != nil {
		frame.constructed = 4
	}
	if e = m.modular.gcdWithSourceContext(lu, lv, lcg, context); e != nil {
		return e
	}
	if frame != nil {
		frame.constructed = 6
	}
	for i, prime := range upolyBigPrimes {
		if e = m.modular.setUint32WithSourceContext(p, prime, context); e != nil {
			return e
		}
		bad, e := m.modularImageWithSourceContext(u, v, q, p, lcg, du, dv, context)
		if e != nil {
			return e
		}
		if bad {
			continue
		}
		if e = m.trim(q); e != nil {
			return e
		}
		if q.size <= 1 {
			if e = m.reset(out); e != nil {
				return e
			}
			dst, e := m.pushDefault(out)
			if e != nil {
				return e
			}
			return m.modular.copyWithSourceContext(dst, cg, context)
		}
		discard := i == 0
		if !discard {
			discard = q.size < C.size
			if !discard {
				discard, e = m.config.Integer.IsEven(p)
				if e != nil {
					return e
				}
			}
			if !discard {
				discard, e = m.config.Integer.IsEven(bound)
				if e != nil {
					return e
				}
			}
		}
		if discard {
			if e = m.setViewWithSourceContext(C, m.stateView(q), context); e != nil {
				return e
			}
			if e = m.modular.copyWithSourceContext(bound, p, context); e != nil {
				return e
			}
		} else {
			if e = m.combineImagesWithSourceContext(q, p, C, bound, context); e != nil {
				return e
			}
		}
		if e = m.primitiveWithSourceContext(m.stateView(C), q, context); e != nil {
			return e
		}
		if modularSourceDebug && q.size == 0 {
			return ErrUPolynomialAssertion
		}
		last, e := m.read(m.stateView(q), q.size-1)
		if e != nil {
			return e
		}
		ok, e := m.modular.dividesWithAllocationContext(last, lcg, context)
		if e != nil {
			return e
		}
		if ok {
			ok, e = m.dividesViewsWithSourceContext(m.stateView(u), m.stateView(q), context)
			if e != nil {
				return e
			}
		}
		if ok {
			ok, e = m.dividesViewsWithSourceContext(m.stateView(v), m.stateView(q), context)
			if e != nil {
				return e
			}
		}
		if ok {
			if e = m.mulScalarState(q, cg); e != nil {
				return e
			}
			if e = m.flipSign(q); e != nil {
				return e
			}
			*q, *out = *out, *q
			return nil
		}
	}
	return m.euclidGCD(a, b, out)
}
func (m *UPolynomialManager) gcd(a, b UPolynomialView, out *upolyVectorState) error {
	return m.gcdWithSourceContext(a, b, out, integerAllocationInvocationContext{})
}
func (m *UPolynomialManager) gcdWithSourceContext(a, b UPolynomialView, out *upolyVectorState, context integerAllocationInvocationContext) error {
	done, e := m.gcdEmpty(a, b, out)
	if done {
		return e
	}
	finite, e := m.modular.Modular()
	if e != nil {
		return e
	}
	if !finite {
		return m.modularGCDWithSourceContext(a, b, out, context)
	}
	return m.euclidGCDWithSourceContext(a, b, out, context)
}
func (m *UPolynomialManager) GCDViews(a, b UPolynomialView, g UPolynomial) error {
	out, e := m.descriptor(g)
	if e != nil {
		return e
	}
	return m.gcd(a, b, &out.state)
}
func (m *UPolynomialManager) GCD(a, b, g UPolynomial) error {
	av, e := m.wholeView(a)
	if e != nil {
		return e
	}
	bv, e := m.wholeView(b)
	if e != nil {
		return e
	}
	return m.GCDViews(av, bv, g)
}
