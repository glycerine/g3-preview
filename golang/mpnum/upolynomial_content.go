// Source: src/math/polynomial/upolynomial.cpp:213-232; src/util/mpz.cpp:845-912.
// Source author: Leonardo de Moura.
// Licensed under the MIT license.
package mpnum

// gcdSpan translates raw IntegerManager.GCDMany's source traversal at actual
// array reads. Only the final result receives mpzzp's normalized assertion;
// per-pair modular GCD would introduce different checks/actions.
func (m *UPolynomialManager) gcdSpan(v UPolynomialView, out *Int) error {
	return m.gcdSpanWithSourceContext(v, out, integerAllocationInvocationContext{})
}
func (m *UPolynomialManager) gcdSpanWithSourceContext(v UPolynomialView, out *Int, context integerAllocationInvocationContext) error {
	if e := m.modular.check(); e != nil {
		return e
	}
	z := m.modular.manager
	if e := z.check(out); e != nil {
		return e
	}
	switch v.count {
	case 0:
		if e := z.Reset(out); e != nil {
			return e
		}
	case 1:
		first, e := m.read(v, 0)
		if e != nil {
			return e
		}
		if e = z.copyFromOwnerWithAllocationContext(out, z, first, nil, context); e != nil {
			return e
		}
		if e = z.absWithAllocationContext(out, context); e != nil {
			return e
		}
	default:
		first, e := m.read(v, 0)
		if e != nil {
			return e
		}
		second, e := m.read(v, 1)
		if e != nil {
			return e
		}
		if e = z.gcdWithAllocationContext(first, second, out, context); e != nil {
			return e
		}
		for i := uint32(2); i < v.count; i++ {
			if isOneInteger(out) {
				break
			}
			next, e := m.read(v, i)
			if e != nil {
				return e
			}
			if e = z.gcdWithAllocationContext(out, next, out, context); e != nil {
				return e
			}
		}
	}
	return m.modular.assertNormalized(out)
}
func (m *UPolynomialManager) primitiveState(v UPolynomialView, pp *upolyVectorState, cont *Int) error {
	return m.primitiveStateWithSourceContext(v, pp, cont, integerAllocationInvocationContext{})
}
func (m *UPolynomialManager) primitiveStateWithSourceContext(v UPolynomialView, pp *upolyVectorState, cont *Int, context integerAllocationInvocationContext) error {
	if modularSourceDebug && v.count == 0 {
		return ErrUPolynomialAssertion
	}
	if e := m.gcdSpanWithSourceContext(v, cont, context); e != nil {
		return e
	}
	if modularSourceDebug {
		positive, e := m.modular.IsPos(cont)
		if e != nil {
			return e
		}
		if !positive {
			return ErrUPolynomialAssertion
		}
	}
	one, e := m.modular.IsOne(cont)
	if e != nil {
		return e
	}
	if one {
		return m.setViewWithSourceContext(pp, v, context)
	}
	if e = m.reserve(pp, v.count); e != nil {
		return e
	}
	for i := uint32(0); i < v.count; i++ {
		x, e := m.read(v, i)
		if e != nil {
			return e
		}
		zero, e := m.modular.IsZero(x)
		if e != nil {
			return e
		}
		dst := &m.values(pp)[i]
		if zero {
			e = m.modular.setInt64WithSourceContext(dst, 0, context)
		} else {
			e = m.modular.divWithAllocationContext(x, cont, dst, context)
		}
		if e != nil {
			return e
		}
	}
	return m.setSize(pp, v.count)
}
func (m *UPolynomialManager) PrimitiveAndContentViews(v UPolynomialView, pp UPolynomial, cont *Int) error {
	out, e := m.descriptor(pp)
	if e != nil {
		return e
	}
	return m.primitiveState(v, &out.state, cont)
}
func (m *UPolynomialManager) PrimitiveAndContent(p, pp UPolynomial, cont *Int) error {
	v, e := m.wholeView(p)
	if e != nil {
		return e
	}
	return m.PrimitiveAndContentViews(v, pp, cont)
}
func (m *UPolynomialManager) primitive(v UPolynomialView, pp *upolyVectorState) error {
	return m.primitiveWithSourceContext(v, pp, integerAllocationInvocationContext{})
}

// Source: src/math/polynomial/upolynomial.h:212-215.
func (m *UPolynomialManager) primitiveWithSourceContext(v UPolynomialView, pp *upolyVectorState, context integerAllocationInvocationContext) (cause error) {
	owner := context.factorFrontendFrame(m.config.Integer)
	if owner == nil {
		var cont Int
		defer m.modular.Del(&cont)
		return m.primitiveStateWithSourceContext(v, pp, &cont, context)
	}
	q := &owner.primitive
	if q.constructed || q.executing || q.invocation == ^uint64(0) {
		return ErrUPolynomialCleanupPending
	}
	*q = upolyFactorPrimitiveScope{owner: owner, invocation: q.invocation + 1, constructed: true, executing: true}
	defer func() { q.executing = false; cause = q.finish(cause) }()
	return m.primitiveStateWithSourceContext(v, pp, &q.cont, context)
}
func (m *UPolynomialManager) PrimitiveView(v UPolynomialView, pp UPolynomial) error {
	out, e := m.descriptor(pp)
	if e != nil {
		return e
	}
	return m.primitive(v, &out.state)
}
func (m *UPolynomialManager) Primitive(p, pp UPolynomial) error {
	v, e := m.wholeView(p)
	if e != nil {
		return e
	}
	return m.PrimitiveView(v, pp)
}
func (m *UPolynomialManager) normalize(v UPolynomialView) error {
	return m.normalizeWithSourceContext(v, integerAllocationInvocationContext{})
}

// Source: src/math/polynomial/upolynomial.cpp:366-401.
func (m *UPolynomialManager) normalizeWithSourceContext(v UPolynomialView, context integerAllocationInvocationContext) (cause error) {
	if v.count == 0 {
		return nil
	}
	if v.count == 1 {
		x, e := m.read(v, 0)
		if e != nil {
			return e
		}
		positive, e := m.modular.IsPos(x)
		if e != nil {
			return e
		}
		if positive {
			return m.modular.setInt64WithSourceContext(x, 1, context)
		}
		return m.modular.setInt64WithSourceContext(x, -1, context)
	}
	var legacy Int
	g := &legacy
	var q *upolyFactorVectorNormalizeScope
	if context != (integerAllocationInvocationContext{}) {
		o := context.factorFrontendFrame(m.config.Integer)
		if o == nil || !o.executing || o.service.manager != m {
			return ErrUPolynomialOwner
		}
		q = &o.vectorNormalize
		if q.constructed != 0 || q.executing || q.invocation == ^uint64(0) {
			return ErrUPolynomialCleanupPending
		}
		*q = upolyFactorVectorNormalizeScope{owner: o, invocation: q.invocation + 1, constructed: 1, peak: 1, executing: true}
		g = &q.g
		defer func() { q.executing = false; cause = q.finish(cause) }()
	} else {
		defer m.modular.Del(g)
	}
	if e := m.gcdSpanWithSourceContext(v, g, context); e != nil {
		return e
	}
	one, e := m.modular.IsOne(g)
	if e != nil || one {
		return e
	}
	for i := uint32(0); i < v.count; i++ {
		if e = m.normalizeCoefficientWithSourceContext(v, i, g, q, context); e != nil {
			return e
		}
	}
	return nil
}
func (m *UPolynomialManager) normalizeCoefficient(v UPolynomialView, i uint32, g *Int) error {
	return m.normalizeCoefficientWithSourceContext(v, i, g, nil, integerAllocationInvocationContext{})
}
func (m *UPolynomialManager) normalizeCoefficientWithSourceContext(v UPolynomialView, i uint32, g *Int, q *upolyFactorVectorNormalizeScope, context integerAllocationInvocationContext) (cause error) {
	var legacyOld, legacyTmp Int
	old, tmp := &legacyOld, &legacyTmp
	// Source Z3DEBUG lexical temporaries are destroyed tmp then old.
	if modularSourceDebug {
		if q != nil {
			if !q.valid() || !q.executing || q.constructed != 1 {
				return ErrUPolynomialOwner
			}
			q.old = Int{}
			q.constructed = 2
			q.peak = 2
			q.coefficientExecuting = true
			old, tmp = &q.old, &q.tmp
			defer func() { q.coefficientExecuting = false; cause = q.finishCoefficient(cause) }()
		} else {
			defer m.modular.Del(old)
		}
		x, e := m.read(v, i)
		if e != nil {
			return e
		}
		if e = m.modular.copyWithSourceContext(old, x, context); e != nil {
			return e
		}
	}
	x, e := m.read(v, i)
	if e != nil {
		return e
	}
	if e = m.modular.divWithAllocationContext(x, g, x, context); e != nil {
		return e
	}
	if modularSourceDebug {
		if q != nil {
			q.tmp = Int{}
			q.constructed = 3
			q.peak = 3
		} else {
			defer m.modular.Del(tmp)
		}
		if e = m.modular.mulWithSourceContext(g, x, tmp, context); e != nil {
			return e
		}
		ok, e := m.modular.Equal(tmp, old)
		if e != nil {
			return e
		}
		if !ok {
			return ErrUPolynomialAssertion
		}
	}
	return nil
}
func (m *UPolynomialManager) NormalizeView(v UPolynomialView) error {
	if e := m.check(); e != nil {
		return e
	}
	return m.normalize(v)
}
func (m *UPolynomialManager) Normalize(p UPolynomial) error {
	v, e := m.wholeView(p)
	if e != nil {
		return e
	}
	return m.NormalizeView(v)
}

// WithCount is the explicit source (pointer,unsigned size) boundary: it
// changes only the scalar count. It neither borrows/dereferences an owner nor
// prevalidates physical extent. Algorithms guard at each actual source read;
// an early return may therefore skip an unavailable tail exactly as C++ does.
func (v UPolynomialView) WithCount(count uint32) UPolynomialView {
	v.count = count
	return v
}
