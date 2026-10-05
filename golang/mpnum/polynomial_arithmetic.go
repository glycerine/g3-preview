// Source: src/math/polynomial/polynomial.cpp:2582-2594,2665-2778,5940-5951. Licensed under the MIT license.
// Source author: Leonardo de Moura (leonardo), 2011-11-15.
package mpnum

func (m *MultivariatePolynomialCore) checkpoint() error {
	if f := m.factorsOwner; f != nil && f.executing {
		f.executing = false
		defer func() { f.executing = true }()
	}
	if s := m.compositionOwner; s != nil && s.executing {
		s.executing = false
		defer func() { s.executing = true }()
	}

	// These hooks are external source resource/cooperation boundaries. Their
	// primitive constructions belong to the caller, not an executing PSC frame.
	m.callbackDepth++
	defer func() { m.callbackDepth-- }()

	if !m.config.Checkpoint() {
		return ErrUPolynomialCanceled
	}
	return m.config.Cooperation()
}
func (m *MultivariatePolynomialCore) AddMul(a1 *Int, t1 Monomial, p1 Polynomial, a2 *Int, t2 Monomial, p2 Polynomial) (Polynomial, error) {
	if e := m.check(); e != nil {
		return Polynomial{}, e
	}
	if e := m.som.reset(); e != nil {
		return Polynomial{}, e
	}
	if e := m.som.addMul(a1, t1, p1); e != nil {
		return Polynomial{}, e
	}
	if e := m.som.addMul(a2, t2, p2); e != nil {
		return Polynomial{}, e
	}
	return m.som.mk(false)
}
func (m *MultivariatePolynomialCore) Add(p, q Polynomial) (Polynomial, error) {
	if e := m.check(); e != nil {
		return Polynomial{}, e
	}
	one := NewInt(1)
	u, e := m.monomials.Unit()
	if e != nil {
		return Polynomial{}, e
	}
	return m.AddMul(&one, u, p, &one, u, q)
}
func (m *MultivariatePolynomialCore) Sub(p, q Polynomial) (Polynomial, error) {
	if e := m.check(); e != nil {
		return Polynomial{}, e
	}
	one := NewInt(1)
	if e := m.modular.SetInt64(&m.minusOne, -1); e != nil {
		return Polynomial{}, e
	}
	u, e := m.monomials.Unit()
	if e != nil {
		return Polynomial{}, e
	}
	return m.AddMul(&one, u, p, &m.minusOne, u, q)
}
func (m *MultivariatePolynomialCore) MulAdd(p, q Polynomial, a *Int) (Polynomial, error) {
	if e := m.check(); e != nil {
		return Polynomial{}, e
	}
	np, e := m.readWord(p, 8)
	if e != nil {
		return Polynomial{}, e
	}
	if np == 0 {
		return m.MkConstantTakeCopy(a)
	}
	nq, e := m.readWord(q, 8)
	if e != nil {
		return Polynomial{}, e
	}
	if np == 0 || nq == 0 {
		return m.MkConstantTakeCopy(a)
	}
	if e = m.som.reset(); e != nil {
		return Polynomial{}, e
	}
	for i := uint32(0); i < np; i++ {
		if e = m.checkpoint(); e != nil {
			return Polynomial{}, e
		}
		coef, e := m.coefficientCore(p, i)
		if e != nil {
			return Polynomial{}, e
		}
		term, e := m.monomialCore(p, i)
		if e != nil {
			return Polynomial{}, e
		}
		if e = m.som.addMul(&coef, term, q); e != nil {
			return Polynomial{}, e
		}
	}
	u, e := m.monomials.Unit()
	if e != nil {
		return Polynomial{}, e
	}
	if e = m.som.add(a, u, false); e != nil {
		return Polynomial{}, e
	}
	return m.som.mk(false)
}
func (m *MultivariatePolynomialCore) Mul(p, q Polynomial) (Polynomial, error) {
	zero := NewInt(0)
	return m.MulAdd(p, q, &zero)
}
func (m *MultivariatePolynomialCore) Scale(a *Int, t Monomial, p Polynomial) (Polynomial, error) {
	if e := m.check(); e != nil {
		return Polynomial{}, e
	}
	zero, e := m.modular.IsZero(a)
	if e != nil {
		return Polynomial{}, e
	}
	if zero {
		return m.zero, nil
	}
	one, e := m.modular.IsOne(a)
	if e != nil {
		return Polynomial{}, e
	}
	if one {
		u, e := m.monomials.Unit()
		if e != nil {
			return Polynomial{}, e
		}
		if t == u {
			return p, nil
		}
	}
	n, e := m.cheap.ms.size(m)
	if e != nil {
		return Polynomial{}, e
	}
	if modularSourceDebug && n != 0 {
		return Polynomial{}, ErrPolynomialAssertion
	}
	if e = m.cheap.addMul(a, t, p); e != nil {
		return Polynomial{}, e
	}
	return m.cheap.mk(false)
}
func (m *MultivariatePolynomialCore) Neg(p Polynomial) (Polynomial, error) {
	return m.negWithReceipt(p, nil)
}
func (m *MultivariatePolynomialCore) negWithReceipt(p Polynomial, receipt *polynomialNegInvocationReceipt) (result Polynomial, cause error) {
	if e := m.check(); e != nil {
		return Polynomial{}, e
	}
	n, e := m.cheap.ms.size(m)
	if e != nil {
		return Polynomial{}, e
	}
	if modularSourceDebug && n != 0 {
		return Polynomial{}, ErrPolynomialAssertion
	}
	if m.negativeActive {
		return Polynomial{}, ErrPolynomialPending
	}
	if m.invocation == ^uint64(0) {
		return Polynomial{}, ErrPolynomialPending
	}
	invocation := m.invocation + 1
	m.polynomialNegScope = polynomialNegScope{invocation: invocation, negativeActive: true}
	if receipt != nil {
		*receipt = polynomialNegInvocationReceipt{manager: m, frame: &m.polynomialNegScope, invocation: invocation}
	}
	defer func() {
		m.negativeBodyCause = cause
		m.negativeResult = result
		if e := m.finishNegSource(); e != nil {
			result = Polynomial{}
			cause = polynomialNewtonJoin(cause, e)
		}
	}()
	n, e = m.readWord(p, 8)
	if e != nil {
		return Polynomial{}, e
	}
	for i := uint32(0); i < n; i++ {
		a, e := m.coefficientCore(p, i)
		if e != nil {
			return Polynomial{}, e
		}
		before, x := polynomialNumericBefore(m.integer)
		if x != nil {
			return Polynomial{}, x
		}
		e = m.modular.Copy(&m.negativeCoefficient, &a)
		e = polynomialNewtonJoin(e, m.numeric.after(m.integer, before))
		if e != nil {
			return Polynomial{}, e
		}
		before, x = polynomialNumericBefore(m.integer)
		if x != nil {
			return Polynomial{}, x
		}
		e = m.modular.Neg(&m.negativeCoefficient)
		e = polynomialNewtonJoin(e, m.numeric.after(m.integer, before))
		if e != nil {
			return Polynomial{}, e
		}
		t, e := m.monomialCore(p, i)
		if e != nil {
			return Polynomial{}, e
		}
		before, x = polynomialNumericBefore(m.integer)
		if x != nil {
			return Polynomial{}, x
		}
		e = m.cheap.add(&m.negativeCoefficient, t, false)
		e = polynomialNewtonJoin(e, m.numeric.after(m.integer, before))
		if e != nil {
			return Polynomial{}, e
		}
	}
	before, x := polynomialNumericBefore(m.integer)
	if x != nil {
		return Polynomial{}, x
	}
	result, e = m.cheap.mk(false)
	e = polynomialNewtonJoin(e, m.numeric.after(m.integer, before))
	return result, e
}
func (m *MultivariatePolynomialCore) MkUnivariateTake(v, n uint32, as []Int) (Polynomial, error) {
	if e := m.check(); e != nil {
		return Polynomial{}, e
	}
	return m.mkUnivariateSource(v, n, polynomialUnivariateCoefficients{values: as})
}
