// Source: src/math/polynomial/polynomial.cpp:2757-2768.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

type polynomialScalarDivScope struct {
	a                      Int
	returned               Polynomial
	active, ended          bool
	bodyCause, sourceCause error
	invocation             uint64
	numeric                polynomialNumericRoots
}
type polynomialScalarDivInvocationReceipt struct {
	manager    *MultivariatePolynomialCore
	frame      *polynomialScalarDivScope
	invocation uint64
}

func (m *MultivariatePolynomialCore) matchesScalarDivInvocation(r *polynomialScalarDivInvocationReceipt) bool {
	return r != nil && r.manager == m && r.frame == &m.scalarDivision && r.invocation != 0 && r.invocation == r.frame.invocation && r.frame.active
}
func (m *MultivariatePolynomialCore) scalarDivInvocationSourceCause(r *polynomialScalarDivInvocationReceipt) error {
	if m.matchesScalarDivInvocation(r) {
		return r.frame.sourceCause
	}
	return nil
}
func (m *MultivariatePolynomialCore) finishScalarDivSource() error {
	s := &m.scalarDivision
	if !s.active || s.ended {
		return nil
	}
	if s.sourceCause != nil {
		return s.sourceCause
	}
	_, e := sourceNumeralDel(m.integer, &s.a)
	if e != nil {
		s.sourceCause = e
		return e
	}
	s.ended = true
	s.returned = Polynomial{}
	if !polynomialNegNumericPending(&s.numeric) {
		s.active = false
	}
	return nil
}
func (m *MultivariatePolynomialCore) cleanupScalarDivInvocation(r *polynomialScalarDivInvocationReceipt, abort bool) error {
	if !m.matchesScalarDivInvocation(r) {
		return nil
	}
	return m.cleanupScalarDivPhase(abort)
}
func (m *MultivariatePolynomialCore) cleanupScalarDivPhase(abort bool) error {
	s := &m.scalarDivision
	if !s.active {
		return nil
	}
	if !s.ended {
		if e := m.deleteNumeral(&s.a); e != nil {
			return e
		}
		s.ended = true
		s.sourceCause = nil
	}
	if abort {
		if s.returned != (Polynomial{}) {
			if e := m.CleanupValue(&s.returned); e != nil {
				return e
			}
		}
		if e := s.numeric.cleanup(m.integer); e != nil {
			return e
		}
	}
	if s.returned == (Polynomial{}) && !polynomialNegNumericPending(&s.numeric) {
		invocation := s.invocation
		*s = polynomialScalarDivScope{invocation: invocation}
	}
	return nil
}
func (m *MultivariatePolynomialCore) scalarDiv(p Polynomial, d *Int) (Polynomial, error) {
	return m.scalarDivWithReceipt(p, d, nil)
}
func (m *MultivariatePolynomialCore) scalarDivWithReceipt(p Polynomial, d *Int, receipt *polynomialScalarDivInvocationReceipt) (result Polynomial, cause error) {
	if e := m.check(); e != nil {
		return Polynomial{}, e
	}
	size, e := m.cheap.ms.size(m)
	if e != nil {
		return Polynomial{}, e
	}
	if modularSourceDebug && size != 0 {
		return Polynomial{}, ErrPolynomialAssertion
	}
	// Source reads p->size before constructing scoped NUM a. The divisor is
	// read only by an actual loop DIV, including when p has no coefficients.
	n, e := m.readWord(p, 8)
	if e != nil {
		return Polynomial{}, e
	}
	s := &m.scalarDivision
	if s.active || s.invocation == ^uint64(0) {
		return Polynomial{}, ErrPolynomialPending
	}
	*s = polynomialScalarDivScope{active: true, invocation: s.invocation + 1}
	if receipt != nil {
		*receipt = polynomialScalarDivInvocationReceipt{manager: m, frame: s, invocation: s.invocation}
	}
	defer func() {
		s.bodyCause = cause
		s.returned = result
		if e := m.finishScalarDivSource(); e != nil {
			result = Polynomial{}
			cause = polynomialNewtonJoin(cause, e)
		}
	}()
	for i := uint32(0); i < n; i++ {
		a, e := m.coefficientCore(p, i)
		if e != nil {
			return Polynomial{}, e
		}
		before, x := polynomialNumericBefore(m.integer)
		if x != nil {
			return Polynomial{}, x
		}
		e = m.modular.Div(&a, d, &s.a)
		e = polynomialNewtonJoin(e, s.numeric.after(m.integer, before))
		if e != nil {
			return Polynomial{}, e
		}
		term, e := m.monomialCore(p, i)
		if e != nil {
			return Polynomial{}, e
		}
		before, x = polynomialNumericBefore(m.integer)
		if x != nil {
			return Polynomial{}, x
		}
		e = m.cheap.add(&s.a, term, false)
		e = polynomialNewtonJoin(e, s.numeric.after(m.integer, before))
		if e != nil {
			return Polynomial{}, e
		}
	}
	before, x := polynomialNumericBefore(m.integer)
	if x != nil {
		return Polynomial{}, x
	}
	result, e = m.cheap.mk(false)
	return result, polynomialNewtonJoin(e, s.numeric.after(m.integer, before))
}
