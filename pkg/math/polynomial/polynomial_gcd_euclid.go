// Source: src/math/polynomial/polynomial.cpp:3639-3672.
package polynomial

import (
	"errors"
)

func (m *MultivariatePolynomialManager) gcdEuclid(u, v Polynomial, r *polynomialReference) (err error) {
	c := m.core
	if polynomialSourceDebug {
		field, e := c.modular.Modular()
		if e != nil {
			return e
		}
		if !field {
			return ErrPolynomialAssertion
		}
		one, e := c.IsUnivariate(u)
		if e != nil {
			return e
		}
		if !one {
			return ErrPolynomialAssertion
		}
		one, e = c.IsUnivariate(v)
		if e != nil {
			return e
		}
		if !one {
			return ErrPolynomialAssertion
		}
	}
	zero, e := c.IsZero(u)
	if e != nil {
		return e
	}
	if zero {
		if e = r.assign(c, v); e != nil {
			return e
		}
		return m.flipLeadingSign(r)
	}
	zero, e = c.IsZero(v)
	if e != nil {
		return e
	}
	if zero {
		if e = r.assign(c, u); e != nil {
			return e
		}
		return m.flipLeadingSign(r)
	}
	if u == v {
		if e = r.assign(c, u); e != nil {
			return e
		}
		return m.flipLeadingSign(r)
	}
	constant, e := c.IsConstant(u)
	if e != nil {
		return e
	}
	if !constant {
		constant, e = c.IsConstant(v)
		if e != nil {
			return e
		}
	}
	if constant {
		s, e := m.beginGCDFunction()
		if e != nil {
			return e
		}
		s.constructNumeral(0)
		s.constructNumeral(1)
		defer func() {
			s.bodyCause = err
			m.gcdDepth--
			err = errors.Join(err, m.finishGCDFunctionSource(s))
			c.pscLeave()
		}()
		if e = m.gcdScalarContent(s, v, &s.nums[1]); e != nil {
			return e
		}
		if e = m.gcdScalarContent(s, u, &s.nums[0]); e != nil {
			return e
		}
		s.constructNumeral(2)
		before, e := polynomialNumericBefore(c.integer)
		if e != nil {
			return e
		}
		e = c.modular.GCD(&s.nums[1], &s.nums[0], &s.nums[2])
		e = errors.Join(e, s.numeric.after(c.integer, before))
		if e != nil {
			return e
		}
		result, e := c.MkConstantTakeCopy(&s.nums[2])
		if e != nil {
			return e
		}
		return r.assign(c, result)
	}
	x, e := c.MaxVariable(u)
	if e != nil {
		return e
	}
	return m.gcdPRS(u, v, x, r)
}
