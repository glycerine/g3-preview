// Source: src/math/polynomial/polynomial.cpp:4076-4230.
package polynomial

import (
	"errors"

	"github.com/glycerine/g3/pkg/arena"
)

func (m *MultivariatePolynomialManager) modGCDRec(u, v Polynomial, p, idx uint32, vars []uint32, r *polynomialReference) (err error) {
	c := m.core
	if err = m.traceModGCDRecPolynomials(4078, "mod_gcd_rec\nu: ", true, u, v); err != nil {
		return err
	}
	numVars := uint32(len(vars))
	if polynomialSourceDebug && numVars == 0 {
		return ErrPolynomialAssertion
	}
	if idx == numVars-1 {
		if polynomialSourceDebug {
			a, e := c.IsUnivariate(u)
			if e != nil {
				return e
			}
			if !a {
				return ErrPolynomialAssertion
			}
			a, e = c.IsUnivariate(v)
			if e != nil {
				return e
			}
			if !a {
				return ErrPolynomialAssertion
			}
		}
		if err = m.gcdEuclid(u, v, r); err != nil {
			return err
		}
		return m.traceModGCDRecPolynomials(4085, "mod_gcd_rec result: ", true, r.value)
	}
	if uint64(idx) >= uint64(len(vars)) {
		if polynomialSourceDebug {
			return ErrPolynomialAssertion
		}
		return arena.ErrBounds
	}
	x := vars[idx]
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
	s.constructNumeral(0)
	s.constructNumeral(1)
	// c_u,pp_u,lc_u,c_v,pp_v,lc_v are actual completed DEFAULT refs.
	for i := uint8(0); i < 6; i++ {
		s.constructReference(c, i)
	}
	cu, pu, lu := &s.refs[0].root, &s.refs[1].root, &s.refs[2].root
	cv, pv, lv := &s.refs[3].root, &s.refs[4].root, &s.refs[5].root
	if e = m.iccpZPXWithContentReceipt(u, x, &s.nums[0], cu, pu, &s.contentInvocation); e != nil {
		return e
	}
	if e = m.iccpZPXWithContentReceipt(v, x, &s.nums[1], cv, pv, &s.contentInvocation); e != nil {
		return e
	}
	a, e := m.lcGlexZPX(pu.value, x)
	if e != nil {
		return e
	}
	if e = lu.assign(c, a); e != nil {
		return e
	}
	a, e = m.lcGlexZPX(pv.value, x)
	if e != nil {
		return e
	}
	if e = lv.assign(c, a); e != nil {
		return e
	}
	s.constructNumeral(2)
	s.constructReference(c, 6)
	s.constructReference(c, 7)
	cg, lg := &s.refs[6].root, &s.refs[7].root
	if e = m.traceModGCDRecInitial(idx, x, pu.value, pv.value, cu.value, cv.value, lu.value, lv.value, &s.nums[0], &s.nums[1]); e != nil {
		return e
	}
	before, e := polynomialNumericBefore(c.integer)
	if e != nil {
		return e
	}
	e = c.modular.GCD(&s.nums[0], &s.nums[1], &s.nums[2])
	e = errors.Join(e, s.numeric.after(c.integer, before))
	if e != nil {
		return e
	}
	if e = m.gcdEuclid(cu.value, cv.value, cg); e != nil {
		return e
	}
	if e = m.gcdEuclid(lu.value, lv.value, lg); e != nil {
		return e
	}
	if e = m.traceModGCDRecContents(cg.value, lg.value, &s.nums[2]); e != nil {
		return e
	}
	sk, e := m.skeletons.Get(idx)
	if e != nil {
		return e
	}
	interpolator, e := m.newton.At(idx)
	if e != nil {
		return e
	}
	// Even dense interpolation constructs the actual nil-skeleton sparse local.
	if e = constructPolynomialSparseInterpolator(m, sk, &s.sparse); e != nil {
		return e
	}
	s.stack[s.depth] = gcdSparseId
	s.depth++
	for i := uint8(8); i < 11; i++ {
		s.constructReference(c, i)
	}
	u1, v1, q := &s.refs[8].root, &s.refs[9].root, &s.refs[10].root
	s.constructNumeral(3)
	s.constructNumeral(4)
	for i := uint8(11); i < 14; i++ {
		s.constructReference(c, i)
	}
	h, candidate, lh := &s.refs[11].root, &s.refs[12].root, &s.refs[13].root
	minDegree := ^uint32(0)
	// uint32 counter deliberately wraps as in the source unsigned loop.
	for counter := uint32(0); ; counter++ {
		for {
			if e = m.peekFresh(interpolator.Inputs(), p, &s.nums[3]); e != nil {
				return e
			}
			if e = m.UnivariateEvaluate(lg.value, x, &s.nums[3], &s.nums[4]); e != nil {
				return e
			}
			zero, e := c.modular.IsZero(&s.nums[4])
			if e != nil {
				return e
			}
			if !zero {
				break
			}
		}
		if e = m.traceModGCDRecPoint(x, &s.nums[3]); e != nil {
			return e
		}
		a, e = c.SubstituteInteger(pu.value, x, &s.nums[3])
		if e != nil {
			return e
		}
		if e = u1.assign(c, a); e != nil {
			return e
		}
		a, e = c.SubstituteInteger(pv.value, x, &s.nums[3])
		if e != nil {
			return e
		}
		if e = v1.assign(c, a); e != nil {
			return e
		}
		if e = m.modGCDRec(u1.value, v1.value, p, idx+1, vars, q); e != nil {
			return e
		}
		a, e = m.glexMonicWithReceipt(q.value, &s.directMonic)
		if e != nil {
			return e
		}
		if e = q.assign(c, a); e != nil {
			return e
		}
		unit, e := c.monomials.Unit()
		if e != nil {
			return e
		}
		a, e = c.Scale(&s.nums[4], unit, q.value)
		if e != nil {
			return e
		}
		if e = q.assign(c, a); e != nil {
			return e
		}
		qvar, e := c.MaxVariable(q.value)
		if e != nil {
			return e
		}
		degree := uint32(0)
		if qvar != ^uint32(0) {
			degree, e = c.Degree(q.value, qvar)
			if e != nil {
				return e
			}
		}
		if e = m.traceModGCDRecIteration(counter, idx, q.value, degree, minDegree, vars[idx+1], qvar); e != nil {
			return e
		}
		if degree < minDegree {
			m.traceModGCDRecDetailLiteral(4152, "reseting...\n")
			counter = 0
			minDegree = degree
			if sk == nil {
				if e = interpolator.Reset(); e != nil {
					return e
				}
				if e = interpolator.Add(&s.nums[3], q.value); e != nil {
					return e
				}
			} else {
				if e = s.sparse.Reset(); e != nil {
					return e
				}
				ok, e := s.sparse.Add(&s.nums[3], q.value)
				if e != nil {
					return e
				}
				if !ok {
					return &polynomialSparseMGCDException{}
				}
			}
		} else if degree == minDegree {
			m.traceModGCDRecDetailLiteral(4167, "adding sample point...\n")
			if sk == nil {
				if e = interpolator.Add(&s.nums[3], q.value); e != nil {
					return e
				}
			} else {
				ok, e := s.sparse.Add(&s.nums[3], q.value)
				if e != nil {
					return e
				}
				if !ok {
					return &polynomialSparseMGCDException{}
				}
			}
		} else {
			m.traceModGCDRecSkipped()
			continue
		}
		found := false
		if sk == nil {
			if polynomialSourceDebug {
				n, e := interpolator.NumSamplePoints()
				if e != nil {
					return e
				}
				if n == 0 {
					return ErrPolynomialAssertion
				}
			}
			if e = interpolator.Mk(x, &s.refs[11]); e != nil {
				return e
			}
			if e = m.traceModGCDRecCandidate(4184, idx, "candidate H: ", h.value, "", Polynomial{}); e != nil {
				return e
			}
			a, e = m.lcGlexZPX(h.value, x)
			if e != nil {
				return e
			}
			if e = lh.assign(c, a); e != nil {
				return e
			}
			if e = m.traceModGCDRecCandidate(4186, idx, "lc_H: ", lh.value, "lc_g: ", lg.value); e != nil {
				return e
			}
			found, e = c.Equal(lh.value, lg.value)
			if e != nil {
				return e
			}
		} else {
			ready, e := s.sparse.Ready()
			if e != nil {
				return e
			}
			if ready {
				ok, e := s.sparse.Interpolate(&s.refs[11])
				if e != nil {
					return e
				}
				if !ok {
					return &polynomialSparseMGCDException{}
				}
				found = true
			}
		}
		done := false
		if found {
			d, e := c.Degree(h.value, x)
			if e != nil {
				return e
			}
			if d > 0 {
				if e = m.primitiveZPX(h.value, x, candidate); e != nil {
					return e
				}
			} else {
				a, e = m.normalizeWithReceipt(h.value, &s.normalizeInvocation)
				if e != nil {
					return e
				}
				if e = candidate.assign(c, a); e != nil {
					return e
				}
			}
			if e = m.traceModGCDRecDivides(candidate.value, pu.value, pv.value, &s.dividesInvocation); e != nil {
				return e
			}
			first, e := c.dividesWithReceipt(candidate.value, pu.value, &s.dividesInvocation)
			if e != nil {
				return e
			}
			second := false
			if first {
				second, e = c.dividesWithReceipt(candidate.value, pv.value, &s.dividesInvocation)
				if e != nil {
					return e
				}
			}
			if first && second {
				a, e = c.mulWithReceipt(cg.value, candidate.value, &s.directArithmetic)
				if e != nil {
					return e
				}
				if e = r.assign(c, a); e != nil {
					return e
				}
				unit, e := c.monomials.Unit()
				if e != nil {
					return e
				}
				a, e = c.Scale(&s.nums[2], unit, r.value)
				if e != nil {
					return e
				}
				if e = r.assign(c, a); e != nil {
					return e
				}
				done = true
			} else if minDegree == 0 {
				if e = r.assign(c, cg.value); e != nil {
					return e
				}
				unit, e := c.monomials.Unit()
				if e != nil {
					return e
				}
				a, e = c.Scale(&s.nums[2], unit, r.value)
				if e != nil {
					return e
				}
				if e = r.assign(c, a); e != nil {
					return e
				}
				done = true
			} else if sk != nil {
				return &polynomialSparseMGCDException{}
			}
		}
		if done {
			if e = m.traceModGCDRecResult(idx, r.value); e != nil {
				return e
			}
			if sk == nil && m.useSparseGCD {
				// Source raw new_sk becomes owned by the persistent vector only after
				// its Set publication succeeds. Retain the actual returned pointer.
				s.newSkeleton, e = newGCDPolynomialSkeleton(c, h.value, x)
				if e != nil {
					return e
				}
				if e = m.skeletons.Set(idx, s.newSkeleton); e != nil {
					return e
				}
				s.newSkeleton.memoryTraceLive = true
				s.newSkeleton = nil
			}
			return nil
		}
	}
}
