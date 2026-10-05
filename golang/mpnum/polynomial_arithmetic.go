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
	return m.addMulWithReceipt(a1, t1, p1, a2, t2, p2, nil)
}
func (m *MultivariatePolynomialCore) addMulWithReceipt(a1 *Int, t1 Monomial, p1 Polynomial, a2 *Int, t2 Monomial, p2 Polynomial, receipt *polynomialArithmeticInvocationReceipt) (Polynomial, error) {
	if e := m.check(); e != nil {
		return Polynomial{}, e
	}
	if e := m.som.reset(); e != nil {
		return Polynomial{}, e
	}
	if receipt != nil {
		if e := m.som.acquireArithmeticInvocation(receipt); e != nil {
			return Polynomial{}, e
		}
	}
	if e := m.som.addMulWithArithmeticReceipt(a1, t1, p1, receipt); e != nil {
		return Polynomial{}, e
	}
	if e := m.som.addMulWithArithmeticReceipt(a2, t2, p2, receipt); e != nil {
		return Polynomial{}, e
	}
	return m.som.mkWithArithmeticReceipt(false, receipt)
}
func (m *MultivariatePolynomialCore) Add(p, q Polynomial) (Polynomial, error) {
	return m.addWithReceipt(p, q, nil)
}
func (m *MultivariatePolynomialCore) addWithReceipt(p, q Polynomial, receipt *polynomialArithmeticInvocationReceipt) (Polynomial, error) {
	if e := m.check(); e != nil {
		return Polynomial{}, e
	}
	one := NewInt(1)
	u, e := m.monomials.Unit()
	if e != nil {
		return Polynomial{}, e
	}
	return m.addMulWithReceipt(&one, u, p, &one, u, q, receipt)
}
func (m *MultivariatePolynomialCore) Sub(p, q Polynomial) (Polynomial, error) {
	return m.subWithReceipt(p, q, nil)
}
func (m *MultivariatePolynomialCore) subWithReceipt(p, q Polynomial, receipt *polynomialArithmeticInvocationReceipt) (Polynomial, error) {
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
	return m.addMulWithReceipt(&one, u, p, &m.minusOne, u, q, receipt)
}
func (m *MultivariatePolynomialCore) MulAdd(p, q Polynomial, a *Int) (Polynomial, error) {
	return m.mulAddWithReceipt(p, q, a, nil)
}
func (m *MultivariatePolynomialCore) mulAddWithReceipt(p, q Polynomial, a *Int, receipt *polynomialArithmeticInvocationReceipt) (Polynomial, error) {
	if e := m.check(); e != nil {
		return Polynomial{}, e
	}
	np, e := m.readWord(p, 8)
	if e != nil {
		return Polynomial{}, e
	}
	if np == 0 {
		return m.mkConstantTakeCopyWithArithmeticReceipt(a, receipt)
	}
	nq, e := m.readWord(q, 8)
	if e != nil {
		return Polynomial{}, e
	}
	if np == 0 || nq == 0 {
		return m.mkConstantTakeCopyWithArithmeticReceipt(a, receipt)
	}
	if e = m.som.reset(); e != nil {
		return Polynomial{}, e
	}
	if receipt != nil {
		if e = m.som.acquireArithmeticInvocation(receipt); e != nil {
			return Polynomial{}, e
		}
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
		if e = m.som.addMulWithArithmeticReceipt(&coef, term, q, receipt); e != nil {
			return Polynomial{}, e
		}
	}
	u, e := m.monomials.Unit()
	if e != nil {
		return Polynomial{}, e
	}
	if e = m.som.addWithArithmeticReceipt(a, u, receipt); e != nil {
		return Polynomial{}, e
	}
	return m.som.mkWithArithmeticReceipt(false, receipt)
}
func (m *MultivariatePolynomialCore) Mul(p, q Polynomial) (Polynomial, error) {
	return m.mulWithReceipt(p, q, nil)
}
func (m *MultivariatePolynomialCore) mulWithReceipt(p, q Polynomial, receipt *polynomialArithmeticInvocationReceipt) (Polynomial, error) {
	zero := NewInt(0)
	return m.mulAddWithReceipt(p, q, &zero, receipt)
}
func (m *MultivariatePolynomialCore) Scale(a *Int, t Monomial, p Polynomial) (Polynomial, error) {
	return m.scaleWithReceipt(a, t, p, nil)
}
func (m *MultivariatePolynomialCore) scaleWithReceipt(a *Int, t Monomial, p Polynomial, receipt *polynomialArithmeticInvocationReceipt) (Polynomial, error) {
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
	if receipt != nil {
		if e = m.cheap.acquireArithmeticInvocation(receipt); e != nil {
			return Polynomial{}, e
		}
	}
	if e = m.cheap.addMulWithArithmeticReceipt(a, t, p, receipt); e != nil {
		return Polynomial{}, e
	}
	return m.cheap.mkWithArithmeticReceipt(false, receipt)
}
func (m *MultivariatePolynomialCore) Neg(p Polynomial) (Polynomial, error) {
	return m.negWithReceipt(p, nil)
}
func (m *MultivariatePolynomialCore) negWithReceipt(p Polynomial, receipt *polynomialNegInvocationReceipt) (result Polynomial, cause error) {
	return m.negWithParentReceipts(p, receipt, nil, nil)
}
func (m *MultivariatePolynomialCore) negWithParentReceipts(p Polynomial, receipt *polynomialNegInvocationReceipt, appendReceipt *polynomialCheapAppendInvocationReceipt, factoryReceipt *polynomialFactoryInvocationReceipt) (result Polynomial, cause error) {
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
		if m.matchesFactoryInvocation(factoryReceipt) && m.factoryInvocation.result == result {
			// The guarded constructor transferred its raw pointer to the
			// parent factory frame before this NUM's source destructor.
			m.negativeResult = Polynomial{}
		}
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
		e = m.cheap.addWithAppendReceipt(&m.negativeCoefficient, t, false, appendReceipt)
		e = polynomialNewtonJoin(e, m.numeric.after(m.integer, before))
		if e != nil {
			return Polynomial{}, e
		}
	}
	before, x := polynomialNumericBefore(m.integer)
	if x != nil {
		return Polynomial{}, x
	}
	result, e = m.cheap.mkWithAppendReceipt(false, appendReceipt, factoryReceipt)
	e = polynomialNewtonJoin(e, m.numeric.after(m.integer, before))
	return result, e
}
func (m *MultivariatePolynomialCore) MkUnivariateTake(v, n uint32, as []Int) (Polynomial, error) {
	if e := m.check(); e != nil {
		return Polynomial{}, e
	}
	return m.mkUnivariateSource(v, n, polynomialUnivariateCoefficients{values: as})
}
