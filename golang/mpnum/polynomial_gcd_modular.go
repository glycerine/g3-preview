// Source: src/math/polynomial/polynomial.cpp:4235-4350.
// Licensed under the MIT license. Source author: Leonardo de Moura (leonardo).
package mpnum

import "errors"

func (m *MultivariatePolynomialManager) modularGCDImage(s *polynomialGCDFunctionScope, prime uint32) (bad bool, err error) {
	c := m.core
	frontier := s.depth
	if e := s.zp.construct(m, &s.nums[7]); e != nil {
		return false, e
	}
	s.stack[s.depth] = gcdZPId
	s.depth++
	defer func() { err = errors.Join(err, m.finishGCDLocalsTo(s, frontier)) }()
	u, e := m.normalizeWithReceipt(s.refs[0].root.value, &s.normalizeInvocation)
	if e != nil {
		return false, e
	}
	if e = s.refs[2].root.assign(c, u); e != nil {
		return false, e
	}
	size, e := c.readWord(s.refs[2].root.value, 8)
	if e != nil {
		return false, e
	}
	old, e := c.readWord(s.refs[0].root.value, 8)
	if e != nil {
		return false, e
	}
	if size != old {
		return true, nil
	}
	v, e := m.normalizeWithReceipt(s.refs[1].root.value, &s.normalizeInvocation)
	if e != nil {
		return false, e
	}
	if e = s.refs[3].root.assign(c, v); e != nil {
		return false, e
	}
	size, e = c.readWord(s.refs[3].root.value, 8)
	if e != nil {
		return false, e
	}
	old, e = c.readWord(s.refs[1].root.value, 8)
	if e != nil {
		return false, e
	}
	if size != old {
		return true, nil
	}
	if e = m.modGCDRec(s.refs[2].root.value, s.refs[3].root.value, prime, 0, &s.vars, &s.refs[5].root); e != nil {
		return false, e
	}
	q, e := m.GlexMonic(s.refs[5].root.value)
	if e != nil {
		return false, e
	}
	if e = s.refs[5].root.assign(c, q); e != nil {
		return false, e
	}
	s.constructNumeral(8)
	if e = m.gcdCopyNumeral(s, &s.nums[5], &s.nums[8]); e != nil {
		return false, e
	}
	unit, e := c.monomials.Unit()
	if e != nil {
		return false, e
	}
	q, e = c.Scale(&s.nums[8], unit, s.refs[5].root.value)
	if e != nil {
		return false, e
	}
	return false, s.refs[5].root.assign(c, q)
}
func (m *MultivariatePolynomialManager) modGCD(u, v Polynomial, ud, vd *polynomialGCDPowerBuffer, r *polynomialReference) (err error) {
	c := m.core
	if e := m.newton.SetOwner(c); e != nil {
		return e
	}
	if modularSourceDebug {
		field, e := c.modular.Modular()
		if e != nil {
			return e
		}
		if field {
			return ErrPolynomialAssertion
		}
		if ud.size != vd.size {
			return ErrPolynomialAssertion
		}
	}
	size := ud.size
	if modularSourceDebug && size <= 1 {
		return ErrPolynomialAssertion
	}
	s, e := m.beginGCDFunction()
	if e != nil {
		return e
	}
	s.vars.construct(c.memory)
	s.stack[s.depth] = gcdVariablesId
	s.depth++
	s.constructBuffer(c, 0)
	defer func() {
		s.bodyCause = err
		m.gcdDepth--
		err = errors.Join(err, m.finishGCDFunctionSource(s))
		c.pscLeave()
	}()
	for i := uint32(0); i < size; i++ {
		if modularSourceDebug {
			ux, e := ud.word(i, 0)
			if e != nil {
				return e
			}
			vx, e := vd.word(i, 0)
			if e != nil {
				return e
			}
			if ux != vx {
				return ErrPolynomialAssertion
			}
		}
		x, e := ud.word(i, 0)
		if e != nil {
			return e
		}
		a, e := ud.word(i, 1)
		if e != nil {
			return e
		}
		b, e := vd.word(i, 1)
		if e != nil {
			return e
		}
		if b < a {
			a = b
		}
		if e = s.powers[0].push(x, a); e != nil {
			return e
		}
	}
	if e = s.powers[0].sortDegrees(); e != nil {
		return e
	}
	if e = m.skeletons.Reset(); e != nil {
		return e
	}
	for i := uint32(0); i < size; i++ {
		x, e := s.powers[0].word(i, 0)
		if e != nil {
			return e
		}
		if e = s.vars.push(x); e != nil {
			return e
		}
		if e = m.skeletons.Push(nil); e != nil {
			return e
		}
	}
	s.constructNumeral(0)
	s.constructNumeral(1)
	s.constructReference(c, 0)
	s.constructReference(c, 1)
	if e = c.integerContentWithReceipt(u, &s.nums[0], &s.refs[0].root, &s.contentInvocation); e != nil {
		return e
	}
	if e = c.integerContentWithReceipt(v, &s.nums[1], &s.refs[1].root, &s.contentInvocation); e != nil {
		return e
	}
	s.constructNumeral(2)
	before, e := polynomialNumericBefore(c.integer)
	if e != nil {
		return e
	}
	e = c.modular.GCD(&s.nums[0], &s.nums[1], &s.nums[2])
	e = errors.Join(e, s.numeric.after(c.integer, before))
	if e != nil {
		return e
	}
	pu, e := c.gradedLexMaxPosition(s.refs[0].root.value)
	if e != nil {
		return e
	}
	pv, e := c.gradedLexMaxPosition(s.refs[1].root.value)
	if e != nil {
		return e
	}
	s.constructNumeral(3)
	s.constructNumeral(4)
	a, e := c.coefficientCore(s.refs[0].root.value, pu)
	if e != nil {
		return e
	}
	if e = m.gcdCopyNumeral(s, &a, &s.nums[3]); e != nil {
		return e
	}
	a, e = c.coefficientCore(s.refs[1].root.value, pv)
	if e != nil {
		return e
	}
	if e = m.gcdCopyNumeral(s, &a, &s.nums[4]); e != nil {
		return e
	}
	s.constructNumeral(5)
	before, e = polynomialNumericBefore(c.integer)
	if e != nil {
		return e
	}
	e = c.modular.GCD(&s.nums[3], &s.nums[4], &s.nums[5])
	e = errors.Join(e, s.numeric.after(c.integer, before))
	if e != nil {
		return e
	}
	s.constructReference(c, 2)
	s.constructReference(c, 3)
	s.constructReference(c, 4)
	s.constructNumeral(6)
	s.constructReference(c, 5)
	s.constructReference(c, 6)
	s.constructNumeral(7)
	for _, prime := range upolyBigPrimes {
		before, e = polynomialNumericBefore(c.integer)
		if e != nil {
			return e
		}
		e = c.modular.SetUint64(&s.nums[7], uint64(prime))
		e = errors.Join(e, s.numeric.after(c.integer, before))
		if e != nil {
			return e
		}
		bad, e := m.modularGCDImage(s, prime)
		if e != nil {
			return e
		}
		if bad {
			continue
		}
		constant, e := c.IsConstant(s.refs[5].root.value)
		if e != nil {
			return e
		}
		if constant {
			result, e := c.MkConstantTakeCopy(&s.nums[2])
			if e != nil {
				return e
			}
			return r.assign(c, result)
		}
		if s.refs[4].root.value == (Polynomial{}) {
			if e = s.refs[4].root.assign(c, s.refs[5].root.value); e != nil {
				return e
			}
			if e = m.gcdCopyNumeral(s, &s.nums[7], &s.nums[6]); e != nil {
				return e
			}
		} else {
			pc, e := c.gradedLexMaxPosition(s.refs[4].root.value)
			if e != nil {
				return e
			}
			mc, e := c.monomialCore(s.refs[4].root.value, pc)
			if e != nil {
				return e
			}
			pq, e := c.gradedLexMaxPosition(s.refs[5].root.value)
			if e != nil {
				return e
			}
			mq, e := c.monomialCore(s.refs[5].root.value, pq)
			if e != nil {
				return e
			}
			order, e := c.monomials.GradedLex(mq, mc)
			if e != nil {
				return e
			}
			if order < 0 {
				if e = s.refs[4].root.assign(c, s.refs[5].root.value); e != nil {
					return e
				}
				if e = m.gcdCopyNumeral(s, &s.nums[7], &s.nums[6]); e != nil {
					return e
				}
			} else {
				if e = m.gcdCRA(s.refs[5].root.value, &s.nums[7], s.refs[4].root.value, &s.nums[6], &s.refs[4]); e != nil {
					return e
				}
			}
		}
		candidate, e := m.normalizeWithReceipt(s.refs[4].root.value, &s.normalizeInvocation)
		if e != nil {
			return e
		}
		if e = s.refs[6].root.assign(c, candidate); e != nil {
			return e
		}
		frontier := s.depth
		s.constructNumeral(9)
		pos, e := c.gradedLexMaxPosition(s.refs[6].root.value)
		if e != nil {
			return e
		}
		coeff, e := c.coefficientCore(s.refs[6].root.value, pos)
		if e != nil {
			return e
		}
		if e = m.gcdCopyNumeral(s, &coeff, &s.nums[9]); e != nil {
			return e
		}
		divides, e := c.modular.Divides(&s.nums[9], &s.nums[5])
		if e != nil {
			return e
		}
		if divides {
			divides, e = c.Divides(s.refs[6].root.value, s.refs[0].root.value)
			if e != nil {
				return e
			}
		}
		if divides {
			divides, e = c.Divides(s.refs[6].root.value, s.refs[1].root.value)
			if e != nil {
				return e
			}
		}
		if divides {
			unit, e := c.monomials.Unit()
			if e != nil {
				return e
			}
			result, e := c.Scale(&s.nums[2], unit, s.refs[6].root.value)
			if e != nil {
				return e
			}
			if e = r.assign(c, result); e != nil {
				return e
			}
			return m.flipLeadingSign(r)
		}
		if e = m.finishGCDLocalsTo(s, frontier); e != nil {
			return e
		}
	}
	x, e := c.MaxVariable(u)
	if e != nil {
		return e
	}
	return m.gcdPRS(u, v, x, r)
}
