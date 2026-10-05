// Source: src/math/polynomial/polynomial.cpp:5356-5480.
// Licensed under the MIT license. Source author: Leonardo de Moura (leonardo).
package mpnum

import "errors"

func (m *MultivariatePolynomialManager) Resultant(p, q Polynomial, x uint32, result *PolynomialRef) error {
	if e := m.ready(); e != nil {
		return e
	}
	if result == nil || result.owner != m.core {
		return ErrPolynomialOwner
	}
	if e := result.open(); e != nil {
		return e
	}
	return m.resultant(p, q, x, &result.root)
}
func (m *MultivariatePolynomialManager) resultant(p, q Polynomial, x uint32, result *polynomialReference) (err error) {
	c := m.core
	s, e := m.beginGCDFunction()
	if e != nil {
		return e
	}
	defer func() {
		s.bodyCause = err
		m.gcdDepth--
		err = errors.Join(err, m.finishGCDFunctionSource(s))
		c.pscLeave()
	}()
	s.constructReference(c, 0)
	s.constructReference(c, 1)
	A, B := &s.refs[0].root, &s.refs[1].root
	if e = A.assign(c, p); e != nil {
		return e
	}
	if e = B.assign(c, q); e != nil {
		return e
	}
	zero, e := c.IsZero(A.value)
	if e != nil {
		return e
	}
	if !zero {
		zero, e = c.IsZero(B.value)
		if e != nil {
			return e
		}
	}
	if zero {
		return result.assign(c, c.zero)
	}
	constant, e := c.IsConstant(A.value)
	if e != nil {
		return e
	}
	if constant {
		b, e := c.IsConstant(B.value)
		if e != nil {
			return e
		}
		if b {
			return result.assign(c, c.unit)
		}
		d, e := c.Degree(B.value, x)
		if e != nil {
			return e
		}
		return c.powerInto(A.value, d, result)
	}
	constant, e = c.IsConstant(B.value)
	if e != nil {
		return e
	}
	if constant {
		d, e := c.Degree(A.value, x)
		if e != nil {
			return e
		}
		return c.powerInto(B.value, d, result)
	}
	s.constructNumeral(0)
	s.constructNumeral(1)
	for i := uint8(2); i < 6; i++ {
		s.constructReference(c, i)
	}
	cA, cB, ppA, ppB := &s.refs[2].root, &s.refs[3].root, &s.refs[4].root, &s.refs[5].root
	if e = m.iccpPartsWithReceipt(A.value, x, &s.nums[0], cA, ppA, &s.iccpInvocation); e != nil {
		return e
	}
	if e = m.iccpPartsWithReceipt(B.value, x, &s.nums[1], cB, ppB, &s.iccpInvocation); e != nil {
		return e
	}
	unit, e := c.monomials.Unit()
	if e != nil {
		return e
	}
	v, e := c.Scale(&s.nums[0], unit, cA.value)
	if e != nil {
		return e
	}
	if e = cA.assign(c, v); e != nil {
		return e
	}
	unit, e = c.monomials.Unit()
	if e != nil {
		return e
	}
	v, e = c.Scale(&s.nums[1], unit, cB.value)
	if e != nil {
		return e
	}
	if e = cB.assign(c, v); e != nil {
		return e
	}
	s.constructReference(c, 6)
	t := &s.refs[6].root
	d, e := c.Degree(B.value, x)
	if e != nil {
		return e
	}
	if e = c.powerInto(cA.value, d, cA); e != nil {
		return e
	}
	d, e = c.Degree(A.value, x)
	if e != nil {
		return e
	}
	if e = c.powerInto(cB.value, d, cB); e != nil {
		return e
	}
	v, e = c.Mul(cA.value, cB.value)
	if e != nil {
		return e
	}
	if e = t.assign(c, v); e != nil {
		return e
	}
	if e = A.assign(c, ppA.value); e != nil {
		return e
	}
	if e = B.assign(c, ppB.value); e != nil {
		return e
	}
	sign := int32(1)
	degA, e := c.Degree(A.value, x)
	if e != nil {
		return e
	}
	degB, e := c.Degree(B.value, x)
	if e != nil {
		return e
	}
	if degA < degB {
		A.value, B.value = B.value, A.value
		if degA%2 == 1 && degB%2 == 1 {
			sign = -1
		}
	}
	for i := uint8(7); i < 11; i++ {
		s.constructReference(c, i)
	}
	R, g, h, newH := &s.refs[7].root, &s.refs[8].root, &s.refs[9].root, &s.refs[10].root
	if e = g.assign(c, c.unit); e != nil {
		return e
	}
	if e = h.assign(c, c.unit); e != nil {
		return e
	}
	for {
		degA, e = c.Degree(A.value, x)
		if e != nil {
			return e
		}
		degB, e = c.Degree(B.value, x)
		if e != nil {
			return e
		}
		if modularSourceDebug && degA < degB {
			return ErrPolynomialAssertion
		}
		delta := degA - degB
		if degA%2 == 1 && degB%2 == 1 {
			sign = -sign
		}
		if e = c.ExactPseudoRemainder(A.value, B.value, x, &s.refs[7]); e != nil {
			return e
		}
		if e = A.assign(c, B.value); e != nil {
			return e
		}
		v, e = c.ExactDiv(R.value, g.value)
		if e != nil {
			return e
		}
		if e = B.assign(c, v); e != nil {
			return e
		}
		for i := uint32(0); i < delta; i++ {
			v, e = c.ExactDiv(B.value, h.value)
			if e != nil {
				return e
			}
			if e = B.assign(c, v); e != nil {
				return e
			}
		}
		v, e = c.LeadingCoefficient(A.value, x)
		if e != nil {
			return e
		}
		if e = g.assign(c, v); e != nil {
			return e
		}
		if e = newH.assign(c, c.unit); e != nil {
			return e
		}
		if e = c.powerInto(g.value, delta, newH); e != nil {
			return e
		}
		if delta > 1 {
			for i := uint32(0); i < delta-1; i++ {
				v, e = c.ExactDiv(newH.value, h.value)
				if e != nil {
					return e
				}
				if e = newH.assign(c, v); e != nil {
					return e
				}
			}
		}
		if e = h.assign(c, newH.value); e != nil {
			return e
		}
		d, e = c.Degree(B.value, x)
		if e != nil {
			return e
		}
		if d == 0 {
			degA, e = c.Degree(A.value, x)
			if e != nil {
				return e
			}
			v, e = c.LeadingCoefficient(B.value, x)
			if e != nil {
				return e
			}
			if e = newH.assign(c, v); e != nil {
				return e
			}
			if e = c.powerInto(newH.value, degA, newH); e != nil {
				return e
			}
			if degA > 1 {
				for i := uint32(0); i < degA-1; i++ {
					v, e = c.ExactDiv(newH.value, h.value)
					if e != nil {
						return e
					}
					if e = newH.assign(c, v); e != nil {
						return e
					}
				}
			}
			if e = h.assign(c, newH.value); e != nil {
				return e
			}
			v, e = c.Mul(t.value, h.value)
			if e != nil {
				return e
			}
			if e = result.assign(c, v); e != nil {
				return e
			}
			if sign < 0 {
				v, e = c.Neg(result.value)
				if e != nil {
					return e
				}
				return result.assign(c, v)
			}
			return nil
		}
	}
}
