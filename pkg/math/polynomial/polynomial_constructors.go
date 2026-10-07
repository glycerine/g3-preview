// Source: src/math/polynomial/polynomial.cpp:2558-2633. Licensed under the MIT license.
// Source author: Leonardo de Moura (leonardo), 2011-11-15.
package polynomial

import (
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func (m *MultivariatePolynomialCore) rational2Numeral(as []util.Rational, n uint32) error {
	size, e := m.rationalTerms.size(m)
	if e != nil {
		return e
	}
	if polynomialSourceDebug && size != 0 {
		return ErrPolynomialAssertion
	}
	for i := uint32(0); i < n; i++ {
		if uint64(i) >= uint64(len(as)) {
			return arena.ErrBounds
		}
		if polynomialSourceDebug && !util.IsIntegerRational(&as[i]) {
			return ErrPolynomialAssertion
		}
		if e = m.rationalTerms.appendCoefficient(m, util.Int{}); e != nil {
			return e
		}
		pos, e := m.rationalTerms.size(m)
		if e != nil {
			return e
		}
		a, e := m.rationalTerms.coefficient(m, pos-1)
		if e != nil {
			return e
		}
		e = m.modular.Copy(&a, &(*as[i].BorrowNumerator()))
		if w := m.rationalTerms.putCoefficient(m, pos-1, a); w != nil {
			return w
		}
		if e != nil {
			return e
		}
	}
	return nil
}
func (m *MultivariatePolynomialCore) resetRationalTerms() error {
	if polynomialSourceDebug {
		n, e := m.rationalTerms.size(m)
		if e != nil {
			return e
		}
		for i := uint32(0); i < n; i++ {
			a, e := m.rationalTerms.coefficient(m, i)
			if e != nil {
				return e
			}
			zero, e := m.modular.IsZero(&a)
			if e != nil {
				return e
			}
			if !zero {
				return ErrPolynomialAssertion
			}
		}
	}
	return m.rationalTerms.reset(m)
}
func (m *MultivariatePolynomialCore) mkTermsVector(as *polynomialVector, ms []Monomial, mv *polynomialVector) (Polynomial, error) {
	return m.mkTermsIndexed(as, monomialArrayView{values: ms, count: uint32(len(ms)), vector: mv})
}
func (m *MultivariatePolynomialCore) mkTermsIndexed(as *polynomialVector, ms monomialArrayView) (Polynomial, error) {
	if e := m.som.reset(); e != nil {
		return Polynomial{}, e
	}
	n, e := as.size(m)
	if e != nil {
		return Polynomial{}, e
	}
	return m.mkTermsSource(n, polynomialUnivariateCoefficients{vector: as}, ms)
}
func (m *MultivariatePolynomialCore) mkTermsSource(n uint32, as polynomialUnivariateCoefficients, ms monomialArrayView) (Polynomial, error) {
	for i := uint32(0); i < n; i++ {
		var local util.Int
		a, e := as.reference(m, i, &local)
		if e != nil {
			return Polynomial{}, e
		}
		term, e := ms.at(m, i)
		if e != nil {
			return Polynomial{}, e
		}
		consumed := false
		e = m.som.addConsumed(a, term, false, &consumed)
		if ms.vector == &m.linearMonomials && consumed && uint64(i) < uint64(len(m.linearGenerated)) {
			m.linearGenerated[i] = false
		}
		if e != nil {
			return Polynomial{}, e
		}
	}
	return m.som.mk(false)
}
func (m *MultivariatePolynomialCore) MkRationalTerms(as []util.Rational, ms []Monomial) (Polynomial, error) {
	if e := m.check(); e != nil {
		return Polynomial{}, e
	}
	if m.overloadResult != (Polynomial{}) {
		return Polynomial{}, ErrPolynomialPending
	}
	if uint64(len(as)) > uint64(^uint32(0)) {
		return Polynomial{}, arena.ErrBounds
	}
	if e := m.rational2Numeral(as, uint32(len(as))); e != nil {
		return Polynomial{}, e
	}
	p, e := m.mkTermsVector(&m.rationalTerms, ms, nil)
	if e != nil {
		return Polynomial{}, e
	}
	m.overloadResult = p
	if e = m.resetRationalTerms(); e != nil {
		return Polynomial{}, e
	}
	m.overloadResult = Polynomial{}
	return p, nil
}
func (m *MultivariatePolynomialCore) mkUnivariateVector(v, n uint32, as *polynomialVector) (Polynomial, error) {
	return m.mkUnivariateSource(v, n, polynomialUnivariateCoefficients{vector: as})
}
func (m *MultivariatePolynomialCore) MkRationalUnivariate(v, n uint32, as []util.Rational) (Polynomial, error) {
	if e := m.check(); e != nil {
		return Polynomial{}, e
	}
	if m.overloadResult != (Polynomial{}) {
		return Polynomial{}, ErrPolynomialPending
	}
	if polynomialSourceDebug && v >= m.monomials.nextVar {
		return Polynomial{}, ErrPolynomialAssertion
	}
	if e := m.rational2Numeral(as, n+1); e != nil {
		return Polynomial{}, e
	}
	p, e := m.mkUnivariateVector(v, n, &m.rationalTerms)
	if e != nil {
		return Polynomial{}, e
	}
	m.overloadResult = p
	if e = m.resetRationalTerms(); e != nil {
		return Polynomial{}, e
	}
	m.overloadResult = Polynomial{}
	return p, nil
}
func (m *MultivariatePolynomialCore) MkLinearTake(as []util.Int, xs []uint32, c *util.Int) (Polynomial, error) {
	if e := m.check(); e != nil {
		return Polynomial{}, e
	}
	if m.overloadResult != (Polynomial{}) {
		return Polynomial{}, ErrPolynomialPending
	}
	an, e := m.linearTerms.size(m)
	if e != nil {
		return Polynomial{}, e
	}
	mn, e := m.linearMonomials.size(m)
	if e != nil {
		return Polynomial{}, e
	}
	if polynomialSourceDebug && (an != 0 || mn != 0) {
		return Polynomial{}, ErrPolynomialAssertion
	}
	return m.mkLinearInput(as, nil, uint32(len(as)), xs, c)
}
func (m *MultivariatePolynomialCore) mkLinearInput(as []util.Int, source *polynomialVector, count uint32, xs []uint32, c *util.Int) (Polynomial, error) {
	var e error
	for i := uint32(0); i < count; i++ {
		var original *util.Int
		var borrowed util.Int
		if source != nil {
			var e error
			borrowed, e = source.coefficient(m, i)
			if e != nil {
				return Polynomial{}, e
			}
			original = &borrowed
		} else {
			if uint64(i) >= uint64(len(as)) {
				return Polynomial{}, arena.ErrBounds
			}
			original = &as[i]
		}
		zero, e := m.modular.IsZero(original)
		if e != nil {
			return Polynomial{}, e
		}
		if zero {
			continue
		}
		if e = m.linearTerms.appendCoefficient(m, util.Int{}); e != nil {
			return Polynomial{}, e
		}
		n, e := m.linearTerms.size(m)
		if e != nil {
			return Polynomial{}, e
		}
		a, e := m.linearTerms.coefficient(m, n-1)
		if e != nil {
			return Polynomial{}, e
		}
		if e = m.modular.Swap(&a, original); e != nil {
			return Polynomial{}, e
		}
		if e = m.linearTerms.putCoefficient(m, n-1, a); e != nil {
			return Polynomial{}, e
		}
		if source != nil {
			if e = source.putCoefficient(m, i, *original); e != nil {
				return Polynomial{}, e
			}
		}
		if uint64(i) >= uint64(len(xs)) {
			return Polynomial{}, arena.ErrBounds
		}
		term, e := m.generatedVariable(xs[i], 1)
		if e != nil {
			return Polynomial{}, e
		}
		index, e := m.linearMonomials.size(m)
		if e != nil {
			return Polynomial{}, e
		}
		if e = m.prepareLinearGenerated(index); e != nil {
			return Polynomial{}, e
		}
		if e = m.linearMonomials.appendMonomial(m, term); e != nil {
			return Polynomial{}, e
		}
		m.linearGenerated[index] = m.generatedActive
		m.transferGeneratedMonomial()
	}
	zero, e := m.modular.IsZero(c)
	if e != nil {
		return Polynomial{}, e
	}
	if !zero {
		if e = m.linearTerms.appendCoefficient(m, util.Int{}); e != nil {
			return Polynomial{}, e
		}
		n, e := m.linearTerms.size(m)
		if e != nil {
			return Polynomial{}, e
		}
		a, e := m.linearTerms.coefficient(m, n-1)
		if e != nil {
			return Polynomial{}, e
		}
		if e = m.modular.Swap(&a, c); e != nil {
			return Polynomial{}, e
		}
		if e = m.linearTerms.putCoefficient(m, n-1, a); e != nil {
			return Polynomial{}, e
		}
		u, e := m.monomials.Unit()
		if e != nil {
			return Polynomial{}, e
		}
		if e = m.linearMonomials.appendMonomial(m, u); e != nil {
			return Polynomial{}, e
		}
	}
	p, e := m.mkTermsVector(&m.linearTerms, nil, &m.linearMonomials)
	if e != nil {
		return Polynomial{}, e
	}
	m.overloadResult = p
	if e = m.linearTerms.reset(m); e != nil {
		return Polynomial{}, e
	}
	if e = m.linearMonomials.reset(m); e != nil {
		return Polynomial{}, e
	}
	m.overloadResult = Polynomial{}
	return p, nil
}

func (m *MultivariatePolynomialCore) MkRationalLinear(as []util.Rational, xs []uint32, c *util.Rational) (Polynomial, error) {
	if e := m.check(); e != nil {
		return Polynomial{}, e
	}
	if c == nil {
		return Polynomial{}, arena.ErrBounds
	}
	if polynomialSourceDebug && !util.IsIntegerRational(c) {
		return Polynomial{}, ErrPolynomialAssertion
	}
	if m.overloadResult != (Polynomial{}) || m.linearConstantActive {
		return Polynomial{}, ErrPolynomialPending
	}
	if uint64(len(as)) > uint64(^uint32(0)) {
		return Polynomial{}, arena.ErrBounds
	}
	if e := m.rational2Numeral(as, uint32(len(as))); e != nil {
		return Polynomial{}, e
	}
	m.linearConstant = util.Int{}
	m.linearConstantActive = true
	if e := m.modular.Copy(&m.linearConstant, &(*c.BorrowNumerator())); e != nil {
		return Polynomial{}, e
	}
	an, e := m.linearTerms.size(m)
	if e != nil {
		return Polynomial{}, e
	}
	mn, e := m.linearMonomials.size(m)
	if e != nil {
		return Polynomial{}, e
	}
	if polynomialSourceDebug && (an != 0 || mn != 0) {
		return Polynomial{}, ErrPolynomialAssertion
	}
	p, e := m.mkLinearInput(nil, &m.rationalTerms, uint32(len(as)), xs, &m.linearConstant)
	if e != nil {
		return Polynomial{}, e
	}
	m.overloadResult = p
	if polynomialSourceDebug {
		zero, e := m.modular.IsZero(&m.linearConstant)
		if e != nil {
			return Polynomial{}, e
		}
		if !zero {
			return Polynomial{}, ErrPolynomialAssertion
		}
	}
	if e = m.resetRationalTerms(); e != nil {
		return Polynomial{}, e
	}
	// Bare source numeral has no manager destructor. Completed source swaps
	// leave zero here; a failed body retains actual fields for owning abort.
	m.linearConstantActive = false
	m.overloadResult = Polynomial{}
	return p, nil
}
