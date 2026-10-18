// Source: src/math/polynomial/upolynomial.cpp:663-826;
package polynomial

import (
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

// modularImage implements the lexical scoped_set_zp body. The old modulus
// copy goes through mpzzp Copy, including source normalization. In modular
// GCD the entering mode is always Z. Cleanup on Go failure is explicit safe
// ownership adaptation; a restoration failure stays visible.

func (m *UPolynomialManager) modularImage(u, v, q *upolyVectorState, p, lc *unumeral.Int, du, dv uint32) (bool, error) {
	return m.modularImageWithSourceContext(u, v, q, p, lc, du, dv, integerAllocationInvocationContext{})
}
func (m *UPolynomialManager) modularImageWithSourceContext(u, v, q *upolyVectorState, p, lc *unumeral.Int, du, dv uint32, context integerAllocationInvocationContext) (bad bool, err error) {
	oldMode, e := m.modular.Modular()
	if e != nil {
		return false, e
	}
	var localOld, localC unumeral.Int
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
	if e = m.modular.WithAllocationContext((context).utility(m.modular.Integers())).Copy(old, &(*m.modular.BorrowModulus())); e != nil {
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

	if e = m.modular.WithAllocationContext((context).utility(m.modular.Integers())).SetZP(p); e != nil {
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
	if e = m.modular.WithAllocationContext((context).utility(m.modular.Integers())).Copy(c, lc); e != nil {
		return false, e
	}
	return false, m.mulScalarViewWithSourceContext(m.stateView(q), c, context)
}
func (m *UPolynomialManager) combineImages(c1 *upolyVectorState, b1 *unumeral.Int, c2 *upolyVectorState, b2 *unumeral.Int) error {
	return m.combineImagesWithSourceContext(c1, b1, c2, b2, integerAllocationInvocationContext{})
}
func (m *UPolynomialManager) combineImagesWithSourceContext(c1 *upolyVectorState, b1 *unumeral.Int, c2 *upolyVectorState, b2 *unumeral.Int, context integerAllocationInvocationContext) (cause error) {
	for i, bound := range []*unumeral.Int{b1, b2} {
		enabled, e := refinementSourceAssertionsEnabled()
		if e != nil {
			return e
		}
		if enabled {
			even, e := m.config.Integer.IsEven(bound)
			if e != nil {
				return e
			}
			if even {
				expression := "!m().m().is_even(b1)"
				if i == 1 {
					expression = "!m().m().is_even(b2)"
				}
				if e := m.scalarDivisionAssertionFailure(int32(664+i), expression); e != nil {
					return e
				}
			}
		}
	}
	R := &m.cra
	if e := m.reset(R); e != nil {
		return e
	}
	var local [12]unumeral.Int
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

	if e := m.modular.WithAllocationContext((context).utility(m.modular.Integers())).ExtendedGCD(b1, b2, inv1, inv2, g); e != nil {
		return e
	}
	enabled, e := refinementSourceAssertionsEnabled()
	if e != nil {
		return e
	}
	if enabled {
		one, e := m.modular.IsOne(g)
		if e != nil {
			return e
		}
		if !one {
			if e := m.scalarDivisionAssertionFailure(671, "m().is_one(g)"); e != nil {
				return e
			}
		}
	}
	z := m.config.Integer
	if e := z.WithAllocationContext((context).utility(z)).Mod(inv1, b2, inv1); e != nil {
		return e
	}
	if e := z.WithAllocationContext((context).utility(z)).Mod(inv2, b1, inv2); e != nil {
		return e
	}
	construct(5)
	if e := m.modular.WithAllocationContext((context).utility(m.modular.Integers())).Mul(b2, inv2, a1); e != nil {
		return e
	}
	if e := m.modular.WithAllocationContext((context).utility(m.modular.Integers())).Mul(b1, inv1, a2); e != nil {
		return e
	}
	construct(6)
	if e := m.modular.WithAllocationContext((context).utility(m.modular.Integers())).Mul(b1, b2, bound); e != nil {
		return e
	}
	construct(12)
	two := unumeral.NewInt(2)
	if e := m.modular.WithAllocationContext((context).utility(m.modular.Integers())).Div(bound, &two, upper); e != nil {
		return e
	}
	if e := m.modular.WithAllocationContext((context).utility(m.modular.Integers())).Copy(lower, upper); e != nil {
		return e
	}
	if e := m.modular.WithAllocationContext((context).utility(m.modular.Integers())).Neg(lower); e != nil {
		return e
	}
	zero := unumeral.NewInt(0)
	n1, n2 := c1.size, c2.size
	n := n1
	if n2 < n {
		n = n2
	}
	add := func(x, y *unumeral.Int) error {
		if e := m.modular.WithAllocationContext((context).utility(m.modular.Integers())).Mul(x, a1, tmp1); e != nil {
			return e
		}
		if e := m.modular.WithAllocationContext((context).utility(m.modular.Integers())).Mul(y, a2, tmp2); e != nil {
			return e
		}
		if e := m.modular.WithAllocationContext((context).utility(m.modular.Integers())).Add(tmp1, tmp2, tmp3); e != nil {
			return e
		}
		if e := z.WithAllocationContext((context).utility(z)).Mod(tmp3, bound, newA); e != nil {
			return e
		}
		greater, e := m.modular.Greater(newA, upper)
		if e != nil {
			return e
		}
		if greater {
			if e = m.modular.WithAllocationContext((context).utility(m.modular.Integers())).Sub(newA, bound, newA); e != nil {
				return e
			}
		}
		dst, e := m.pushDefault(R)
		if e != nil {
			return e
		}
		return m.modular.WithAllocationContext((context).utility(m.modular.Integers())).Copy(dst, newA)
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
	if e := m.modular.WithAllocationContext((context).utility(m.modular.Integers())).Copy(b2, bound); e != nil {
		return e
	}
	*R, *c2 = *c2, *R
	return nil
}
func (m *UPolynomialManager) modularGCD(a, b UPolynomialView, out *upolyVectorState) error {
	return m.modularGCDWithSourceContext(a, b, out, integerAllocationInvocationContext{})
}
func (m *UPolynomialManager) modularGCDWithSourceContext(a, b UPolynomialView, out *upolyVectorState, context integerAllocationInvocationContext) (cause error) {
	enabled, e := refinementSourceAssertionsEnabled()
	if e != nil {
		return e
	}
	if enabled && (a.count == 0 || b.count == 0) {
		if e := m.scalarDivisionAssertionFailure(727, "sz_u > 0 && sz_v > 0"); e != nil {
			return e
		}
	}
	enabled, e = refinementSourceAssertionsEnabled()
	if e != nil {
		return e
	}
	if enabled {
		finite, e := m.modular.Modular()
		if e != nil {
			return e
		}
		if finite {
			if e := m.scalarDivisionAssertionFailure(728, "!m().modular()"); e != nil {
				return e
			}
		}
	}
	var local [6]unumeral.Int
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
	if e := m.modular.WithAllocationContext((context).utility(m.modular.Integers())).Gcd(cu, cv, cg); e != nil {
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
	if e = m.modular.WithAllocationContext((context).utility(m.modular.Integers())).Gcd(lu, lv, lcg); e != nil {
		return e
	}
	if frame != nil {
		frame.constructed = 6
	}
	for i, prime := range upolyBigPrimes {
		if e = m.modular.WithAllocationContext((context).utility(m.modular.Integers())).SetUint32(p, prime); e != nil {
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
			return m.modular.WithAllocationContext((context).utility(m.modular.Integers())).Copy(dst, cg)
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
			if e = m.modular.WithAllocationContext((context).utility(m.modular.Integers())).Copy(bound, p); e != nil {
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
		enabled, e := refinementSourceAssertionsEnabled()
		if e != nil {
			return e
		}
		if enabled && q.size == 0 {
			if e := m.scalarDivisionAssertionFailure(797, "candidate.size() > 0"); e != nil {
				return e
			}
		}
		last, e := m.read(m.stateView(q), q.size-1)
		if e != nil {
			return e
		}
		ok, e := m.modular.WithAllocationContext((context).utility(m.modular.Integers())).Divides(last, lcg)
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
			if e = m.mulScalarStateWithSourceContext(q, cg, context); e != nil {
				return e
			}
			if e = m.flipSign(q); e != nil {
				return e
			}
			*q, *out = *out, *q
			return nil
		}
	}
	return m.euclidGCDWithSourceContext(a, b, out, context)
}
func (m *UPolynomialManager) gcd(a, b UPolynomialView, out *upolyVectorState) error {
	return m.gcdWithSourceContext(a, b, out, integerAllocationInvocationContext{})
}
func (m *UPolynomialManager) gcdWithSourceContext(a, b UPolynomialView, out *upolyVectorState, context integerAllocationInvocationContext) error {
	done, e := m.gcdEmpty(a, b, out, 859)
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
