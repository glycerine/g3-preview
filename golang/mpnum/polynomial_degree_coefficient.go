// Source: src/math/polynomial/polynomial.cpp:1697-1818,3264-3338,3551-3553.
// Licensed under the MIT license.
package mpnum

func (m *MultivariatePolynomialCore) IsZero(p Polynomial) (bool, error) {
	if e := m.check(); e != nil {
		return false, e
	}
	n, e := m.readWord(p, 8)
	return n == 0, e
}
func (m *MultivariatePolynomialCore) IsConstant(p Polynomial) (bool, error) {
	if e := m.check(); e != nil {
		return false, e
	}
	zero, e := m.IsZero(p)
	if e != nil || zero {
		return zero, e
	}
	n, e := m.readWord(p, 8)
	if e != nil || n != 1 {
		return false, e
	}
	term, e := m.monomialCore(p, 0)
	if e != nil {
		return false, e
	}
	sz, e := m.monomials.sizeCore(term)
	return sz == 0, e
}
func (m *MultivariatePolynomialCore) MaxVariable(p Polynomial) (uint32, error) {
	if e := m.check(); e != nil {
		return 0, e
	}
	n, e := m.readWord(p, 8)
	if e != nil {
		return 0, e
	}
	if n == 0 {
		return ^uint32(0), nil
	}
	term, e := m.monomialCore(p, 0)
	if e != nil {
		return 0, e
	}
	return m.monomials.MaxVariable(term)
}
func (m *MultivariatePolynomialCore) Degree(p Polynomial, x uint32) (uint32, error) {
	if e := m.check(); e != nil {
		return 0, e
	}
	n, e := m.readWord(p, 8)
	if e != nil || n == 0 {
		return 0, e
	}
	term, e := m.monomialCore(p, 0)
	if e != nil {
		return 0, e
	}
	sz, e := m.monomials.sizeCore(term)
	if e != nil || sz == 0 {
		return 0, e
	}
	v, e := m.monomials.varCore(term, sz-1)
	if e != nil {
		return 0, e
	}
	if v == x {
		return m.monomials.degreeAtCore(term, sz-1)
	}
	var result uint32
	for i := uint32(0); i < n; i++ {
		term, e = m.monomialCore(p, i)
		if e != nil {
			return 0, e
		}
		d, e := m.monomials.DegreeOf(term, x)
		if e != nil {
			return 0, e
		}
		if d > result {
			result = d
		}
	}
	return result, nil
}
func (m *MultivariatePolynomialCore) coefficientAssertions(x uint32, reduct bool) error {
	if !modularSourceDebug {
		return nil
	}
	valid, e := m.monomials.IsValidVar(x)
	if e != nil {
		return e
	}
	if !valid {
		return ErrPolynomialAssertion
	}
	n, e := m.cheap.ms.size(m)
	if e != nil {
		return e
	}
	if n != 0 {
		return ErrPolynomialAssertion
	}
	if reduct {
		n, e = m.cheap2.ms.size(m)
		if e != nil {
			return e
		}
		if n != 0 {
			return ErrPolynomialAssertion
		}
	}
	return nil
}
func (m *MultivariatePolynomialCore) generatedRemoveVariable(term Monomial, x uint32) (Monomial, error) {
	if m.generatedActive {
		return Monomial{}, ErrPolynomialPending
	}
	outcome := monomialFactoryOutcome{}
	a, e := m.monomials.removeVariableCreated(term, x, &outcome)
	if e != nil && outcome.started {
		m.recordGeneratedCallee(outcome)
	}
	if e == nil && outcome.created {
		m.generatedMonomial = a
		if s := m.compositionOwner; s != nil && s.executing {
			s.coreGenerated = a
		}
		m.generatedActive = true
		m.generatedPSC = m.pscExecutingOwner()
		if m.translation.executing {
			m.translation.generated = a
		}
	}
	return a, e
}
func (m *MultivariatePolynomialCore) Coeff(p Polynomial, x, k uint32) (Polynomial, error) {
	return m.coeffCore(p, x, k, nil)
}
func (m *MultivariatePolynomialCore) CoeffWithReduct(p Polynomial, x, k uint32, reduct *PolynomialRef) (Polynomial, error) {
	if reduct == nil || reduct.owner != m {
		return Polynomial{}, ErrPolynomialOwner
	}
	if e := reduct.open(); e != nil {
		return Polynomial{}, e
	}
	return m.coeffCore(p, x, k, &reduct.root)
}
func (m *MultivariatePolynomialCore) coeffCore(p Polynomial, x, k uint32, reduct *polynomialReference) (Polynomial, error) {
	if e := m.check(); e != nil {
		return Polynomial{}, e
	}
	if e := m.coefficientAssertions(x, reduct != nil); e != nil {
		return Polynomial{}, e
	}
	n, e := m.readWord(p, 8)
	if e != nil {
		return Polynomial{}, e
	}
	for i := uint32(0); i < n; i++ {
		term, e := m.monomialCore(p, i)
		if e != nil {
			return Polynomial{}, e
		}
		d, e := m.monomials.DegreeOf(term, x)
		if e != nil {
			return Polynomial{}, e
		}
		if d == k {
			a, e := m.coefficientCore(p, i)
			if e != nil {
				return Polynomial{}, e
			}
			term, e = m.generatedRemoveVariable(term, x)
			if e != nil {
				return Polynomial{}, e
			}
			if e = m.cheap.add(&a, term, false); e != nil {
				return Polynomial{}, e
			}
		} else if reduct != nil {
			a, e := m.coefficientCore(p, i)
			if e != nil {
				return Polynomial{}, e
			}
			if e = m.cheap2.add(&a, term, false); e != nil {
				return Polynomial{}, e
			}
		}
	}
	if reduct != nil {
		r, e := m.cheap2.mk(false)
		if e != nil {
			return Polynomial{}, e
		}
		if e = reduct.assign(m, r); e != nil {
			return Polynomial{}, e
		}
	}
	return m.cheap.mk(false)
}
func (m *MultivariatePolynomialCore) LeadingCoefficient(p Polynomial, x uint32) (Polynomial, error) {
	d, e := m.Degree(p, x)
	if e != nil {
		return Polynomial{}, e
	}
	return m.Coeff(p, x, d)
}
func (m *MultivariatePolynomialCore) ConstantCoefficient(p Polynomial, x, k uint32, c *Int) (bool, error) {
	if e := m.check(); e != nil {
		return false, e
	}
	if modularSourceDebug {
		valid, e := m.monomials.IsValidVar(x)
		if e != nil {
			return false, e
		}
		if !valid {
			return false, ErrPolynomialAssertion
		}
	}
	if e := m.modular.Reset(c); e != nil {
		return false, e
	}
	n, e := m.readWord(p, 8)
	if e != nil {
		return false, e
	}
	for i := uint32(0); i < n; i++ {
		term, e := m.monomialCore(p, i)
		if e != nil {
			return false, e
		}
		d, e := m.monomials.DegreeOf(term, x)
		if e != nil {
			return false, e
		}
		if d == k {
			sz, e := m.monomials.sizeCore(term)
			if e != nil {
				return false, e
			}
			if (k > 0 && sz > 1) || (k == 0 && sz > 0) {
				return false, nil
			}
			a, e := m.coefficientCore(p, i)
			if e != nil {
				return false, e
			}
			if e = m.modular.Copy(c, &a); e != nil {
				return false, e
			}
		}
	}
	return true, nil
}

func (m *MultivariatePolynomialCore) IsUnivariate(p Polynomial) (bool, error) {
	if e := m.check(); e != nil {
		return false, e
	}
	n, e := m.readWord(p, 8)
	if e != nil {
		return false, e
	}
	constant, e := m.IsConstant(p)
	if e != nil || constant {
		return constant, e
	}
	// The source obtains m(0) before max_var, even though the loop overwrites it.
	if _, e = m.monomialCore(p, 0); e != nil {
		return false, e
	}
	x, e := m.MaxVariable(p)
	if e != nil {
		return false, e
	}
	for i := uint32(0); i < n; i++ {
		term, e := m.monomialCore(p, i)
		if e != nil {
			return false, e
		}
		size, e := m.monomials.sizeCore(term)
		if e != nil {
			return false, e
		}
		if size == 1 {
			v, e := m.monomials.varCore(term, 0)
			if e != nil {
				return false, e
			}
			if v == x {
				continue
			}
		}
		// This is a separate source read after the first short-circuit condition.
		size, e = m.monomials.sizeCore(term)
		if e != nil {
			return false, e
		}
		if size == 0 {
			continue
		}
		return false, nil
	}
	return true, nil
}
func (m *MultivariatePolynomialCore) IsLinear(p Polynomial) (bool, error) {
	if e := m.check(); e != nil {
		return false, e
	}
	n, e := m.readWord(p, 8)
	if e != nil {
		return false, e
	}
	for i := uint32(0); i < n; i++ {
		term, e := m.monomialCore(p, i)
		if e != nil {
			return false, e
		}
		size, e := m.monomials.sizeCore(term)
		if e != nil {
			return false, e
		}
		if size == 0 {
			continue
		}
		size, e = m.monomials.sizeCore(term)
		if e != nil {
			return false, e
		}
		if size == 1 {
			degree, e := m.monomials.degreeAtCore(term, 0)
			if e != nil {
				return false, e
			}
			if degree == 1 {
				continue
			}
		}
		return false, nil
	}
	return true, nil
}
func (m *MultivariatePolynomialCore) TotalDegree(p Polynomial) (uint32, error) {
	if e := m.check(); e != nil {
		return 0, e
	}
	n, e := m.readWord(p, 8)
	if e != nil {
		return 0, e
	}
	result := uint32(0)
	for i := uint32(0); i < n; i++ {
		term, e := m.monomialCore(p, i)
		if e != nil {
			return 0, e
		}
		d, e := m.monomials.degreeCore(term)
		if e != nil {
			return 0, e
		}
		if d > result {
			result = d
		}
	}
	return result, nil
}

// UnivariateCoefficient returns the borrowed scalar numeral, not an owning copy.
func (m *MultivariatePolynomialCore) UnivariateCoefficient(p Polynomial, k uint32) (Int, error) {
	if e := m.check(); e != nil {
		return Int{}, e
	}
	if modularSourceDebug {
		univariate, e := m.IsUnivariate(p)
		if e != nil {
			return Int{}, e
		}
		if !univariate {
			return Int{}, ErrPolynomialAssertion
		}
	}
	n, e := m.readWord(p, 8)
	if e != nil {
		return Int{}, e
	}
	for i := uint32(0); i < n; i++ {
		term, e := m.monomialCore(p, i)
		if e != nil {
			return Int{}, e
		}
		d, e := m.monomials.degreeCore(term)
		if e != nil {
			return Int{}, e
		}
		if d == k {
			return m.coefficientCore(p, i)
		}
	}
	return Int{}, nil
}
