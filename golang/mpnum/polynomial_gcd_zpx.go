// Source: src/math/polynomial/polynomial.cpp:4018-4051.
// Licensed under the MIT license. Source author: Leonardo de Moura (leonardo).
package mpnum

import "errors"

func (m *MultivariatePolynomialManager) lcGlexZPX(p Polynomial, x uint32) (result Polynomial, err error) {
	c := m.core
	if e := c.som.reset(); e != nil {
		return result, e
	}
	s, e := m.beginGCDFunction()
	if e != nil {
		return result, e
	}
	s.constructMonomial(0)
	s.constructMonomial(1)
	defer func() {
		s.bodyCause = err
		m.gcdDepth--
		err = errors.Join(err, m.finishGCDFunctionSource(s))
		c.pscLeave()
	}()
	max, tmp := &s.monomials[0], &s.monomials[1]
	size, e := c.readWord(p, 8)
	if e != nil {
		return result, e
	}
	for i := uint32(0); i < size; i++ {
		term, e := c.monomialCore(p, i)
		if e != nil {
			return result, e
		}
		k, e := c.monomials.DegreeOf(term, x)
		if e != nil {
			return result, e
		}
		if k != 0 {
			a, e := c.generatedRemoveVariable(term, x)
			if e != nil {
				return result, e
			}
			if e = tmp.assign(c, a); e != nil {
				return result, e
			}
			term = tmp.value
		}
		greater := max.value == (Monomial{})
		if !greater {
			order, e := c.monomials.GradedLex(term, max.value)
			if e != nil {
				return result, e
			}
			greater = order > 0
		}
		if greater {
			if e = c.som.reset(); e != nil {
				return result, e
			}
			if e = max.assign(c, term); e != nil {
				return result, e
			}
			a, e := c.coefficientCore(p, i)
			if e != nil {
				return result, e
			}
			mono, e := c.generatedVariable(x, k)
			if e != nil {
				return result, e
			}
			if e = c.som.add(&a, mono, false); e != nil {
				return result, e
			}
		} else if max.value == term {
			a, e := c.coefficientCore(p, i)
			if e != nil {
				return result, e
			}
			mono, e := c.generatedVariable(x, k)
			if e != nil {
				return result, e
			}
			if e = c.som.add(&a, mono, false); e != nil {
				return result, e
			}
		}
	}
	if modularSourceDebug {
		n, e := c.som.ms.size(c)
		if e != nil {
			return result, e
		}
		if n == 0 {
			return result, ErrPolynomialAssertion
		}
	}
	return c.som.mk(false)
}

func (m *MultivariatePolynomialManager) primitiveZPX(p Polynomial, x uint32, out *polynomialReference) (err error) {
	s, e := m.beginGCDFunction()
	if e != nil {
		return e
	}
	s.constructNumeral(0)
	s.constructReference(m.core, 0)
	defer func() {
		s.bodyCause = err
		m.gcdDepth--
		err = errors.Join(err, m.finishGCDFunctionSource(s))
		m.core.pscLeave()
	}()
	return m.iccpZPXWithContentReceipt(p, x, &s.nums[0], &s.refs[0].root, out, &s.contentInvocation)
}
