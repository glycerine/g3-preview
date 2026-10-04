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
func (s *UPolynomialFactorizationService) flipFactorConstant(fs *UPolynomialFactors) error {
	z := s.manager.config.Integer
	var c Int
	defer z.Del(&c)
	if e := z.Copy(&c, &fs.constant); e != nil {
		return e
	}
	if e := z.Neg(&c); e != nil {
		return e
	}
	return fs.SetConstant(&c)
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
	if e = m.flipSign(p); e != nil {
		return e
	}
	if k%2 == 1 {
		return s.flipFactorConstant(fs)
	}
	return nil
}
func (s *UPolynomialFactorizationService) factorQuadratic(p *upolyVectorState, fs *UPolynomialFactors, k uint32) error {
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
	var b2, ac, disc, root Int
	defer m.modular.Del(&b2)
	defer m.modular.Del(&ac)
	defer m.modular.Del(&disc)
	if e = m.modular.Power(b, 2, &b2); e != nil {
		return e
	}
	if e = m.modular.Mul(a, c, &ac); e != nil {
		return e
	}
	minus4 := NewInt(-4)
	if e = m.modular.AddMul(&b2, &minus4, &ac, &disc); e != nil {
		return e
	}
	if modularSourceDebug {
		zero, e := m.modular.IsZero(&disc)
		if e != nil {
			return e
		}
		if zero {
			return ErrUPolynomialAssertion
		}
	}
	defer m.modular.Del(&root)
	square, e := m.modular.IsPerfectSquare(&disc, &root)
	if e != nil {
		return e
	}
	if !square {
		return fs.PushCopy(m.stateView(p), k)
	}
	var f1, f2 upolyVectorState
	defer m.dropLocalVector(&f1)
	defer m.dropLocalVector(&f2)
	if e = m.reserve(&f1, 2); e != nil {
		return e
	}
	if e = m.reserve(&f2, 2); e != nil {
		return e
	}
	if e = m.modular.Sub(b, &root, &m.values(&f1)[0]); e != nil {
		return e
	}
	if e = m.modular.Add(b, &root, &m.values(&f2)[0]); e != nil {
		return e
	}
	two := NewInt(2)
	if e = m.modular.Mul(a, &two, &m.values(&f1)[1]); e != nil {
		return e
	}
	if e = m.modular.Mul(a, &two, &m.values(&f2)[1]); e != nil {
		return e
	}
	if e = m.setSize(&f1, 2); e != nil {
		return e
	}
	if e = m.setSize(&f2, 2); e != nil {
		return e
	}
	if e = m.normalize(m.stateView(&f1)); e != nil {
		return e
	}
	if e = m.normalize(m.stateView(&f2)); e != nil {
		return e
	}
	if modularSourceDebug {
		if e = s.quadraticDiagnostic(p, &f1, &f2); e != nil {
			return e
		}
	}
	if e = fs.PushCopy(m.stateView(&f1), k); e != nil {
		return e
	}
	return fs.PushCopy(m.stateView(&f2), k)
}
func (s *UPolynomialFactorizationService) quadraticDiagnostic(p, f1, f2 *upolyVectorState) error {
	m := s.manager
	var product upolyVectorState
	defer m.dropLocalVector(&product)
	if e := m.mul(m.stateView(f1), m.stateView(f2), &product); e != nil {
		return e
	}
	same, e := m.EqualViews(m.stateView(&product), m.stateView(p))
	if e != nil {
		return e
	}
	if !same {
		return ErrUPolynomialAssertion
	}
	return nil
}
func (s *UPolynomialFactorizationService) factorSqfPrimitive(p *upolyVectorState, fs *UPolynomialFactors, k uint32, params UPolynomialFactorParams) (bool, error) {
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
		return true, fs.PushCopy(m.stateView(p), k)
	}
	if p.size == 3 {
		return true, s.factorQuadratic(p, fs, k)
	}
	return s.factorSquareFree(m.stateView(p), fs, k, params)
}
func (s *UPolynomialFactorizationService) exactDivVerify(a, b UPolynomialView, out *upolyVectorState) error {
	yes, e := s.manager.exactDivState(a, b, out)
	if e != nil {
		return e
	}
	if modularSourceDebug && !yes {
		return ErrUPolynomialAssertion
	}
	return nil
}
func (s *UPolynomialFactorizationService) factorCore(f UPolynomialView, fs *UPolynomialFactors, params UPolynomialFactorParams) (bool, error) {
	m := s.manager
	if f.count == 0 {
		zero := NewInt(0)
		return true, fs.SetConstant(&zero)
	}
	if f.count == 1 {
		a, e := m.read(f, 0)
		if e != nil {
			return false, e
		}
		return true, fs.SetConstant(a)
	}
	var content Int
	defer m.modular.Del(&content)
	var pp, prime, A, B, D upolyVectorState
	defer m.dropLocalVector(&pp)
	if e := m.primitiveState(f, &pp, &content); e != nil {
		return false, e
	}
	if e := fs.SetConstant(&content); e != nil {
		return false, e
	}
	defer m.dropLocalVector(&prime)
	if e := m.derivativeState(m.stateView(&pp), &prime); e != nil {
		return false, e
	}
	defer m.dropLocalVector(&A)
	defer m.dropLocalVector(&B)
	defer m.dropLocalVector(&D)
	if e := m.gcd(m.stateView(&pp), m.stateView(&prime), &B); e != nil {
		return false, e
	}
	result := true
	if B.size <= 1 {
		if modularSourceDebug && pp.size <= 1 {
			return false, ErrUPolynomialAssertion
		}
		if e := s.flipLeadingIfNegative(&pp, fs, 1); e != nil {
			return false, e
		}
		yes, e := s.factorSqfPrimitive(&pp, fs, 1, params)
		if e != nil {
			return false, e
		}
		if !yes {
			result = false
		}
	} else {
		if e := s.exactDivVerify(m.stateView(&pp), m.stateView(&B), &A); e != nil {
			return false, e
		}
		for j := uint32(1); A.size > 1; j++ {
			if e := m.checkpoint(); e != nil {
				return false, e
			}
			if e := m.gcd(m.stateView(&A), m.stateView(&B), &D); e != nil {
				return false, e
			}
			if e := s.exactDivVerify(m.stateView(&A), m.stateView(&D), &pp); e != nil {
				return false, e
			}
			if pp.size > 1 {
				if e := s.flipLeadingIfNegative(&pp, fs, j); e != nil {
					return false, e
				}
				yes, e := s.factorSqfPrimitive(&pp, fs, j, params)
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
				a, e := m.read(m.stateView(&pp), 0)
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
			if e := s.exactDivVerify(m.stateView(&B), m.stateView(&D), &B); e != nil {
				return false, e
			}
			A, D = D, A
		}
		if modularSourceDebug {
			if A.size != 1 {
				return false, ErrUPolynomialAssertion
			}
			one, e := m.modular.IsOne(&m.values(&A)[0])
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
