// Source: src/math/polynomial/polynomial.cpp:2582-2594,2665-2778,5940-5951. Licensed under the MIT license.
// Source author: Leonardo de Moura (leonardo), 2011-11-15.
package mpnum

func (m *MultivariatePolynomialCore) checkpoint() error {
	if s := m.compositionOwner; s != nil && s.executing {
		s.executing = false
		defer func() { s.executing = true }()
	}

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
func (m *MultivariatePolynomialCore) Neg(p Polynomial) (result Polynomial, cause error) {
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
	m.negativeCoefficient = Int{}
	m.negativeActive = true
	m.negativeBodyCause = nil
	m.negativeCleanupCause = nil
	defer func() {
		m.negativeBodyCause = cause
		m.negativeResult = result
		cleanup := m.deleteNumeral(&m.negativeCoefficient)
		m.negativeCleanupCause = cleanup
		if cleanup == nil {
			m.negativeActive = false
			m.negativeResult = Polynomial{}
		} else {
			result = Polynomial{}
			if cause == nil {
				cause = cleanup
			}
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
		if e = m.modular.Copy(&m.negativeCoefficient, &a); e != nil {
			return Polynomial{}, e
		}
		if e = m.modular.Neg(&m.negativeCoefficient); e != nil {
			return Polynomial{}, e
		}
		t, e := m.monomialCore(p, i)
		if e != nil {
			return Polynomial{}, e
		}
		if e = m.cheap.add(&m.negativeCoefficient, t, false); e != nil {
			return Polynomial{}, e
		}
	}
	return m.cheap.mk(false)
}
func (m *MultivariatePolynomialCore) MkUnivariateTake(v, n uint32, as []Int) (Polynomial, error) {
	if e := m.check(); e != nil {
		return Polynomial{}, e
	}
	return m.mkUnivariateSource(v, n, polynomialUnivariateCoefficients{values: as})
}
