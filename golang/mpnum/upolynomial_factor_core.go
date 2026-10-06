// Source: src/math/polynomial/upolynomial.cpp:2960-3118;
// src/math/polynomial/upolynomial_factorization.cpp:1036-1050.
// Licensed under the MIT license.
package mpnum

func (s *UPolynomialFactorizationService) Factor(f UPolynomialView, out *UPolynomialFactors, params UPolynomialFactorParams) (bool, error) {
	if e := s.check(); e != nil {
		return false, e
	}
	if e := s.manager.checkFactors(out); e != nil {
		return false, e
	}
	return s.factorCore(f, out, params)
}
func (s *UPolynomialFactorizationService) flipFactorConstant(fs *UPolynomialFactors) (cause error) {
	m := s.manager
	o := m.factorFrontendCaller
	if o == nil || !o.actual() || !o.executing || o.result != fs {
		return ErrUPolynomialOwner
	}
	q := &o.flip
	if q.constructed || q.executing || q.invocation == ^uint64(0) {
		return ErrUPolynomialCleanupPending
	}
	*q = upolyFactorFlipScope{owner: o, invocation: q.invocation + 1, constructed: true, executing: true}
	defer func() { q.executing = false; cause = q.finish(cause) }()
	context := m.modular.factorFrontendAllocationContext(o)
	if e := m.modular.copyWithSourceContext(&q.c, &fs.constant, context); e != nil {
		return e
	}
	if e := m.modular.negWithSourceContext(&q.c, context); e != nil {
		return e
	}
	return fs.setConstantWithSourceContext(&q.c, context)
}
func (s *UPolynomialFactorizationService) flipLeadingIfNegative(p *upolyVectorState, fs *UPolynomialFactors, k uint32) error {
	m := s.manager
	if p.size == 0 {
		return nil
	}
	top, e := m.read(m.stateView(p), p.size-1)
	if e != nil {
		return e
	}
	negative, e := m.modular.IsNeg(top)
	if e != nil || !negative {
		return e
	}
	if e = m.negViewInPlaceWithSourceContext(m.stateView(p), m.modular.factorFrontendAllocationContext(m.factorFrontendCaller)); e != nil {
		return e
	}
	if k%2 == 1 {
		return s.flipFactorConstant(fs)
	}
	return nil
}
func (s *UPolynomialFactorizationService) factorQuadratic(p *upolyVectorState, fs *UPolynomialFactors, k uint32) (cause error) {
	m := s.manager
	if modularSourceDebug && p.size != 3 {
		return ErrUPolynomialAssertion
	}
	a, e := m.read(m.stateView(p), 2)
	if e != nil {
		return e
	}
	b, e := m.read(m.stateView(p), 1)
	if e != nil {
		return e
	}
	c, e := m.read(m.stateView(p), 0)
	if e != nil {
		return e
	}
	o := m.factorFrontendCaller
	if o == nil || !o.actual() || !o.executing || o.result != fs {
		return ErrUPolynomialOwner
	}
	q := &o.quadratic
	if q.constructed != 0 || q.executing || q.invocation == ^uint64(0) {
		return ErrUPolynomialCleanupPending
	}
	*q = upolyFactorQuadraticScope{owner: o, invocation: q.invocation + 1, constructed: 3, peak: 3, executing: true}
	defer func() { q.executing = false; cause = q.finish(cause) }()
	b2, ac, disc, root := &q.values[0], &q.values[1], &q.values[2], &q.values[3]
	context := m.modular.factorFrontendAllocationContext(o)
	if e = m.modular.powerWithSourceContext(b, 2, b2, context); e != nil {
		return e
	}
	if e = m.modular.mulWithSourceContext(a, c, ac, context); e != nil {
		return e
	}
	minus4 := NewInt(-4)
	if e = m.modular.addMulWithSourceContext(b2, &minus4, ac, disc, &o.addMul, context); e != nil {
		return e
	}
	if modularSourceDebug {
		zero, e := m.modular.IsZero(disc)
		if e != nil {
			return e
		}
		if zero {
			return ErrUPolynomialAssertion
		}
	}
	q.constructed = 4
	q.peak = 4
	square, e := m.modular.isPerfectSquareWithSourceContext(disc, root, context)
	if e != nil {
		return e
	}
	if !square {
		return fs.pushCopyWithSourceContext(m.stateView(p), k, m.modular.factorFrontendAllocationContext(m.factorFrontendCaller))
	}
	f1, f2 := &q.f1, &q.f2
	q.constructed = 6
	q.peak = 6
	if e = m.reserve(f1, 2); e != nil {
		return e
	}
	if e = m.reserve(f2, 2); e != nil {
		return e
	}
	if e = m.modular.subWithSourceContext(b, root, &m.values(f1)[0], context); e != nil {
		return e
	}
	if e = m.modular.addWithSourceContext(b, root, &m.values(f2)[0], context); e != nil {
		return e
	}
	two := NewInt(2)
	if e = m.modular.mulWithSourceContext(a, &two, &m.values(f1)[1], context); e != nil {
		return e
	}
	if e = m.modular.mulWithSourceContext(a, &two, &m.values(f2)[1], context); e != nil {
		return e
	}
	if e = m.setSize(f1, 2); e != nil {
		return e
	}
	if e = m.setSize(f2, 2); e != nil {
		return e
	}
	if e = m.normalizeWithSourceContext(m.stateView(f1), context); e != nil {
		return e
	}
	if e = m.normalizeWithSourceContext(m.stateView(f2), context); e != nil {
		return e
	}
	if modularSourceDebug {
		if e = s.quadraticDiagnosticWithFrontend(p, f1, f2, q, context); e != nil {
			return e
		}
	}
	if e = fs.pushCopyWithSourceContext(m.stateView(f1), k, context); e != nil {
		return e
	}
	return fs.pushCopyWithSourceContext(m.stateView(f2), k, context)
}
func (s *UPolynomialFactorizationService) quadraticDiagnosticWithFrontend(p, f1, f2 *upolyVectorState, q *upolyFactorQuadraticScope, context integerAllocationInvocationContext) (cause error) {
	m := s.manager
	q.constructed = 7
	q.peak = 7
	defer func() { cause = q.finishProduct(cause) }()
	if e := m.mulWithSourceContext(m.stateView(f1), m.stateView(f2), &q.product, &q.owner.addMul, context); e != nil {
		return e
	}
	same, e := m.EqualViews(m.stateView(&q.product), m.stateView(p))
	if e != nil {
		return e
	}
	if !same {
		return ErrUPolynomialAssertion
	}
	return nil
}
func (s *UPolynomialFactorizationService) factorSqfPrimitive(p *upolyVectorState, fs *UPolynomialFactors, k uint32, params UPolynomialFactorParams) (bool, error) {
	return s.factorSqfPrimitiveWithReceipt(p, fs, k, params, nil)
}
func (s *UPolynomialFactorizationService) factorSqfPrimitiveWithReceipt(p *upolyVectorState, fs *UPolynomialFactors, k uint32, params UPolynomialFactorParams, r *upolyFactorSquareFreeInvocationReceipt) (bool, error) {
	m := s.manager
	if modularSourceDebug && p.size != 0 {
		top, e := m.read(m.stateView(p), p.size-1)
		if e != nil {
			return false, e
		}
		positive, e := m.modular.IsPos(top)
		if e != nil {
			return false, e
		}
		if !positive {
			return false, ErrUPolynomialAssertion
		}
	}
	if p.size <= 2 {
		return true, fs.pushCopyWithSourceContext(m.stateView(p), k, m.modular.factorFrontendAllocationContext(m.factorFrontendCaller))
	}
	if p.size == 3 {
		return true, s.factorQuadratic(p, fs, k)
	}
	return s.factorSquareFreeWithReceipt(m.stateView(p), fs, k, params, r)
}
func (s *UPolynomialFactorizationService) exactDivVerify(a, b UPolynomialView, out *upolyVectorState) error {
	return s.exactDivVerifyWithSourceContext(a, b, out, nil, integerAllocationInvocationContext{})
}
func (s *UPolynomialFactorizationService) exactDivVerifyWithSourceContext(a, b UPolynomialView, out *upolyVectorState, child *integerAddMulInvocationReceipt, context integerAllocationInvocationContext) error {
	yes, e := s.manager.exactDivStateWithSourceContext(a, b, out, child, context)
	if e != nil {
		return e
	}
	if modularSourceDebug && !yes {
		return ErrUPolynomialAssertion
	}
	return nil
}
func (s *UPolynomialFactorizationService) factorCore(f UPolynomialView, fs *UPolynomialFactors, params UPolynomialFactorParams) (completed bool, failure error) {
	m := s.manager
	if f.count == 0 {
		zero := NewInt(0)
		return true, s.setFrontendConstant(fs, &zero)
	}
	if f.count == 1 {
		a, e := m.read(f, 0)
		if e != nil {
			return false, e
		}
		return true, s.setFrontendConstant(fs, a)
	}
	q, e := s.beginFactorFrontend()
	if e != nil {
		return false, e
	}
	q.result = fs
	defer func() { q.bodyReturned(); failure = q.finish(failure) }()
	content := &q.content
	pp, prime, A, B, D := &q.pp, &q.prime, &q.A, &q.B, &q.D
	q.constructed = 2
	if e := m.primitiveStateWithSourceContext(f, pp, content, m.modular.factorFrontendAllocationContext(q)); e != nil {
		return false, e
	}
	if e := fs.setConstantWithSourceContext(content, m.modular.factorFrontendAllocationContext(q)); e != nil {
		return false, e
	}
	q.constructed = 3
	if e := m.derivativeStateWithSourceContext(m.stateView(pp), prime, m.modular.factorFrontendAllocationContext(q)); e != nil {
		return false, e
	}
	q.constructed = 6
	if e := m.gcdWithSourceContext(m.stateView(pp), m.stateView(prime), B, m.modular.factorFrontendAllocationContext(q)); e != nil {
		return false, e
	}
	result := true
	if B.size <= 1 {
		if modularSourceDebug && pp.size <= 1 {
			return false, ErrUPolynomialAssertion
		}
		if e := s.flipLeadingIfNegative(pp, fs, 1); e != nil {
			return false, e
		}
		yes, e := s.factorSqfPrimitiveWithReceipt(pp, fs, 1, params, &q.squareFree)
		if e != nil {
			return false, e
		}
		if !yes {
			result = false
		}
	} else {
		if e := s.exactDivVerifyWithSourceContext(m.stateView(pp), m.stateView(B), A, &q.addMul, m.modular.factorFrontendAllocationContext(q)); e != nil {
			return false, e
		}
		for j := uint32(1); A.size > 1; j++ {
			if e := m.checkpoint(); e != nil {
				return false, e
			}
			if e := m.gcdWithSourceContext(m.stateView(A), m.stateView(B), D, m.modular.factorFrontendAllocationContext(q)); e != nil {
				return false, e
			}
			if e := s.exactDivVerifyWithSourceContext(m.stateView(A), m.stateView(D), pp, &q.addMul, m.modular.factorFrontendAllocationContext(q)); e != nil {
				return false, e
			}
			if pp.size > 1 {
				if e := s.flipLeadingIfNegative(pp, fs, j); e != nil {
					return false, e
				}
				yes, e := s.factorSqfPrimitiveWithReceipt(pp, fs, j, params, &q.squareFree)
				if e != nil {
					return false, e
				}
				if !yes {
					result = false
				}
			} else {
				if modularSourceDebug && pp.size != 1 {
					return false, ErrUPolynomialAssertion
				}
				a, e := m.read(m.stateView(pp), 0)
				if e != nil {
					return false, e
				}
				if modularSourceDebug {
					one, e := m.modular.IsOne(a)
					if e != nil {
						return false, e
					}
					if !one {
						minus, e := m.modular.IsMinusOne(a)
						if e != nil {
							return false, e
						}
						if !minus {
							return false, ErrUPolynomialAssertion
						}
					}
				}
				minus, e := m.modular.IsMinusOne(a)
				if e != nil {
					return false, e
				}
				if minus && j%2 == 1 {
					if e = s.flipFactorConstant(fs); e != nil {
						return false, e
					}
				}
			}
			if e := s.exactDivVerifyWithSourceContext(m.stateView(B), m.stateView(D), B, &q.addMul, m.modular.factorFrontendAllocationContext(q)); e != nil {
				return false, e
			}
			*A, *D = *D, *A
		}
		if modularSourceDebug {
			if A.size != 1 {
				return false, ErrUPolynomialAssertion
			}
			one, e := m.modular.IsOne(&m.values(A)[0])
			if e != nil {
				return false, e
			}
			if !one {
				return false, ErrUPolynomialAssertion
			}
		}
	}
	return result, nil
}
