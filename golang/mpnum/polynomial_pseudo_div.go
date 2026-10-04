// Source: src/math/polynomial/polynomial.cpp:4905-5091; src/util/obj_ref.h.
// Licensed under the MIT license.
package mpnum

import "errors"

const (
	pseudoOldP uint8 = iota
	pseudoOldQ
	pseudoQuotient
	pseudoLeading
	pseudoPowered
	pseudoLeft
	pseudoRight
	pseudoLeadB
	pseudoReductB
	pseudoAdjusted
	pseudoConstantPower
	pseudoNumeral
	pseudoMonomial
)

// These are actual possible source locals embedded in the charged core header.
// Each record is constructed only at its source site. The scalar stack records
// lexical destruction order. At most seven records in this source scope are
// simultaneously constructed: DEBUG wrapper3 + core NUM/lead/reduct + inner
// monomial, or the seven DEBUG verification references after core destruction.
// Nested Power has its own existing result record. The candidate slots are a
// charged Go continuation representation, not the C++ stack allocation geometry.
type polynomialPseudoDivisionScope struct {
	refs                                  [11]polynomialReference
	numeral                               Int
	monomial                              polynomialDivisionReference
	stack                                 [13]uint8
	depth                                 uint8
	constructed                           uint16
	active, numeralStopped                bool
	bodyCause, cleanupCause, numeralCause error
}

func (s *polynomialPseudoDivisionScope) construct(which uint8) {
	s.constructed |= uint16(1) << which
	s.stack[s.depth] = which
	s.depth++
}
func (m *MultivariatePolynomialCore) pseudoPop(owning bool) error {
	s := &m.pseudoDivision
	if s.depth == 0 {
		return nil
	}
	id := s.stack[s.depth-1]
	var e error
	switch id {
	case pseudoNumeral:
		if s.numeralStopped && !owning {
			return s.numeralCause
		}
		e = m.deleteNumeral(&s.numeral)
		if e != nil {
			s.numeralStopped = true
			s.numeralCause = e
		}
	case pseudoMonomial:
		e = s.monomial.drop(m, &s.monomial.value, &s.monomial.decrementStarted, owning)
	default:
		e = s.refs[id].closeValue(m, owning)
	}
	if e != nil {
		return e
	}
	s.depth--
	return nil
}
func (m *MultivariatePolynomialCore) pseudoCleanup(owning bool) error {
	s := &m.pseudoDivision
	if !s.active {
		return nil
	}
	if m.powerLocal.active && m.powerLocal.cleanupCause != nil {
		if !owning {
			return m.powerLocal.cleanupCause
		}
		if e := m.CleanupPolynomialPower(); e != nil {
			return e
		}
	}
	for s.depth != 0 {
		if e := m.pseudoPop(owning); e != nil {
			return e
		}
	}
	// Unjoined INCs are absent from ordinary source destructors.
	if owning {
		for i := len(s.refs) - 1; i >= 0; i-- {
			if e := s.refs[i].abort(m); e != nil {
				return e
			}
		}
		if e := s.monomial.drop(m, &s.monomial.incoming, &s.monomial.incomingStarted, true); e != nil {
			return e
		}
		s.monomial.assigning = false
	}
	for i := range s.refs {
		if s.refs[i].assigning || s.refs[i].incoming != (Polynomial{}) {
			return nil
		}
	}
	if s.monomial.assigning || s.monomial.incoming != (Monomial{}) {
		return nil
	}
	*s = polynomialPseudoDivisionScope{}
	return nil
}
func (m *MultivariatePolynomialCore) CleanupPseudoDivision() error {
	if m == nil || m.self != m {
		return ErrPolynomialOwner
	}
	e := m.pseudoCleanup(true)
	if e != nil {
		m.pseudoDivision.cleanupCause = e
	}
	return e
}
func (m *MultivariatePolynomialCore) generatedDivideVariablePower(a Monomial, x, k uint32) (Monomial, error) {
	if m.generatedActive {
		return Monomial{}, ErrPolynomialPending
	}
	outcome := monomialFactoryOutcome{}
	p, e := m.monomials.variableTransformCreated(a, x, k, 0, &outcome)
	if e != nil && outcome.started {
		m.recordGeneratedCallee(outcome)
	}
	if e == nil && outcome.created {
		m.generatedMonomial = p
		m.generatedActive = true
	}
	return p, e
}
func (m *MultivariatePolynomialCore) pseudoEntry(q, r *PolynomialRef) error {
	if e := m.check(); e != nil {
		return e
	}
	if r == nil || r.owner != m {
		return ErrPolynomialOwner
	}
	if e := r.open(); e != nil {
		return e
	}
	if q != nil {
		if q.owner != m {
			return ErrPolynomialOwner
		}
		if e := q.open(); e != nil {
			return e
		}
	}
	if m.pseudoDivision.active || m.powerLocal.active {
		return ErrPolynomialPending
	}
	m.pseudoDivision = polynomialPseudoDivisionScope{active: true}
	return nil
}
func (m *MultivariatePolynomialCore) pseudoFinish(err *error) {
	s := &m.pseudoDivision
	s.bodyCause = *err
	if s.cleanupCause != nil {
		*err = errors.Join(*err, s.cleanupCause)
		return
	}
	e := m.pseudoCleanup(false)
	if e != nil {
		s.cleanupCause = e
		*err = errors.Join(*err, e)
	}
}
func (m *MultivariatePolynomialCore) PseudoDivide(p, q Polynomial, x uint32, d *uint32, Q, R *PolynomialRef) (err error) {
	if d == nil || Q == nil {
		return ErrPolynomialOwner
	}
	if e := m.pseudoEntry(Q, R); e != nil {
		return e
	}
	defer m.pseudoFinish(&err)
	return m.pseudoCore(p, q, x, d, &Q.root, &R.root, false, true, false, nil)
}
func (m *MultivariatePolynomialCore) ExactPseudoDivide(p, q Polynomial, x uint32, Q, R *PolynomialRef) (err error) {
	if Q == nil {
		return ErrPolynomialOwner
	}
	if e := m.pseudoEntry(Q, R); e != nil {
		return e
	}
	defer m.pseudoFinish(&err)
	var d uint32
	return m.pseudoCore(p, q, x, &d, &Q.root, &R.root, true, true, false, nil)
}
func (m *MultivariatePolynomialCore) ExactPseudoRemainder(p, q Polynomial, x uint32, R *PolynomialRef) (err error) {
	if e := m.pseudoEntry(nil, R); e != nil {
		return e
	}
	defer m.pseudoFinish(&err)
	s := &m.pseudoDivision
	s.construct(pseudoQuotient)
	var d uint32
	return m.pseudoCore(p, q, x, &d, &s.refs[pseudoQuotient], &R.root, true, false, false, nil)
}
func (m *MultivariatePolynomialCore) PseudoRemainder(p, q Polynomial, x uint32, d *uint32, R *PolynomialRef) (err error) {
	if d == nil {
		return ErrPolynomialOwner
	}
	if e := m.pseudoEntry(nil, R); e != nil {
		return e
	}
	defer m.pseudoFinish(&err)
	s := &m.pseudoDivision
	if modularSourceDebug {
		s.construct(pseudoOldP)
		s.construct(pseudoOldQ)
		if e := s.refs[pseudoOldP].assign(m, p); e != nil {
			return e
		}
		if e := s.refs[pseudoOldQ].assign(m, q); e != nil {
			return e
		}
	}
	s.construct(pseudoQuotient)
	if e := m.pseudoCore(p, q, x, d, &s.refs[pseudoQuotient], &R.root, false, modularSourceDebug, false, nil); e != nil {
		return e
	}
	if !modularSourceDebug {
		return nil
	}
	s.construct(pseudoLeading)
	degree, e := m.Degree(q, x)
	if e != nil {
		return e
	}
	l, e := m.Coeff(s.refs[pseudoOldQ].value, x, degree)
	if e != nil {
		return e
	}
	if e = s.refs[pseudoLeading].assign(m, l); e != nil {
		return e
	}
	s.construct(pseudoPowered)
	if e = m.powerInto(s.refs[pseudoLeading].value, *d, &s.refs[pseudoPowered]); e != nil {
		return e
	}
	s.construct(pseudoLeft)
	lhs, e := m.Mul(s.refs[pseudoPowered].value, s.refs[pseudoOldP].value)
	if e != nil {
		return e
	}
	if e = s.refs[pseudoLeft].assign(m, lhs); e != nil {
		return e
	}
	s.construct(pseudoRight)
	rhs, e := m.Mul(s.refs[pseudoQuotient].value, s.refs[pseudoOldQ].value)
	if e != nil {
		return e
	}
	if e = s.refs[pseudoRight].assign(m, rhs); e != nil {
		return e
	}
	rhs, e = m.Add(s.refs[pseudoRight].value, R.root.value)
	if e != nil {
		return e
	}
	if e = s.refs[pseudoRight].assign(m, rhs); e != nil {
		return e
	}
	equal, e := m.Equal(s.refs[pseudoLeft].value, s.refs[pseudoRight].value)
	if e != nil {
		return e
	}
	if !equal {
		return ErrPolynomialAssertion
	}
	return nil
}
func (m *MultivariatePolynomialCore) pseudoCore(p, q Polynomial, x uint32, d *uint32, Q, R *polynomialReference, exact, quotient, modulo bool, bounds *PolynomialVariableDegrees) (err error) {
	s := &m.pseudoDivision
	baseDepth := s.depth
	defer func() {
		if s.cleanupCause != nil {
			return
		}
		if m.powerLocal.active && m.powerLocal.cleanupCause != nil {
			err = errors.Join(err, m.powerLocal.cleanupCause)
			return
		}
		for s.depth > baseDepth {
			if e := m.pseudoPop(false); e != nil {
				s.cleanupCause = e
				err = errors.Join(err, e)
				return
			}
		}
	}()
	if modularSourceDebug {
		valid, e := m.monomials.IsValidVar(x)
		if e != nil {
			return e
		}
		if !valid {
			return ErrPolynomialAssertion
		}
	}
	if modularSourceDebug && modulo && bounds == nil {
		return ErrPolynomialAssertion
	}
	degA, e := m.Degree(p, x)
	if e != nil {
		return e
	}
	degB, e := m.Degree(q, x)
	if e != nil {
		return e
	}
	if degB == 0 {
		if e = R.assign(m, m.zero); e != nil {
			return e
		}
		if quotient {
			if exact {
				*d = degA + 1
				if *d == 1 {
					return Q.assign(m, p)
				}
				s.construct(pseudoConstantPower)
				if e = m.powerInto(q, *d-1, &s.refs[pseudoConstantPower]); e != nil {
					return e
				}
				product, e := m.Mul(p, s.refs[pseudoConstantPower].value)
				if e != nil {
					return e
				}
				if e = Q.assign(m, product); e != nil {
					return e
				}
				if modulo {
					product, e = m.ModD(Q.value, bounds)
					if e != nil {
						return e
					}
					return Q.assign(m, product)
				}
				return nil
			}
			*d = 1
			return Q.assign(m, p)
		}
		return nil
	}
	if degB > degA {
		if e = Q.assign(m, m.zero); e != nil {
			return e
		}
		if e = R.assign(m, p); e != nil {
			return e
		}
		*d = 0
	}
	s.construct(pseudoNumeral)
	s.construct(pseudoLeadB)
	s.construct(pseudoReductB)
	lead, e := m.coeffCore(q, x, degB, &s.refs[pseudoReductB])
	if e != nil {
		return e
	}
	if e = s.refs[pseudoLeadB].assign(m, lead); e != nil {
		return e
	}
	*d = 0
	if e = R.assign(m, p); e != nil {
		return e
	}
	if e = Q.assign(m, m.zero); e != nil {
		return e
	}
	for {
		if e = m.checkpoint(); e != nil {
			return e
		}
		degR, e := m.Degree(R.value, x)
		if e != nil {
			return e
		}
		if degB > degR {
			if exact {
				exactD := degA - degB + 1
				if modularSourceDebug && *d > exactD {
					return ErrPolynomialAssertion
				}
				if *d < exactD {
					s.construct(pseudoAdjusted)
					if e = m.powerInto(s.refs[pseudoLeadB].value, exactD-*d, &s.refs[pseudoAdjusted]); e != nil {
						return e
					}
					if quotient {
						product, e := m.Mul(s.refs[pseudoAdjusted].value, Q.value)
						if e != nil {
							return e
						}
						if e = Q.assign(m, product); e != nil {
							return e
						}
						if modulo {
							product, e = m.ModD(Q.value, bounds)
							if e != nil {
								return e
							}
							if e = Q.assign(m, product); e != nil {
								return e
							}
						}
					}
					product, e := m.Mul(s.refs[pseudoAdjusted].value, R.value)
					if e != nil {
						return e
					}
					if e = R.assign(m, product); e != nil {
						return e
					}
					if modulo {
						product, e = m.ModD(R.value, bounds)
						if e != nil {
							return e
						}
						if e = R.assign(m, product); e != nil {
							return e
						}
					}
				}
			}
			return nil
		}
		if e = m.som.reset(); e != nil {
			return e
		}
		if e = m.som2.reset(); e != nil {
			return e
		}
		size, e := m.readWord(R.value, 8)
		if e != nil {
			return e
		}
		for i := uint32(0); i < size; i++ {
			term, e := m.monomialCore(R.value, i)
			if e != nil {
				return e
			}
			a, e := m.coefficientCore(R.value, i)
			if e != nil {
				return e
			}
			degree, e := m.monomials.DegreeOf(term, x)
			if e != nil {
				return e
			}
			if degree == degR {
				s.construct(pseudoMonomial)
				prime, e := m.generatedDivideVariablePower(term, x, degB)
				if e != nil {
					return e
				}
				if e = s.monomial.assign(m, prime); e != nil {
					return e
				}
				if modularSourceDebug {
					before, e := m.monomials.DegreeOf(term, x)
					if e != nil {
						return e
					}
					if before != degR {
						return ErrPolynomialAssertion
					}
					after, e := m.monomials.DegreeOf(s.monomial.value, x)
					if e != nil {
						return e
					}
					if after != degR-degB {
						return ErrPolynomialAssertion
					}
				}
				if quotient {
					if e = m.som2.add(&a, s.monomial.value, false); e != nil {
						return e
					}
				}
				if e = m.modular.Copy(&s.numeral, &a); e != nil {
					return e
				}
				if e = m.modular.Neg(&s.numeral); e != nil {
					return e
				}
				if e = m.som.addMul(&s.numeral, s.monomial.value, s.refs[pseudoReductB].value); e != nil {
					return e
				}
				if e = m.pseudoPop(false); e != nil {
					s.cleanupCause = e
					return e
				}
			} else {
				if e = m.som.addMul(&a, term, s.refs[pseudoLeadB].value); e != nil {
					return e
				}
			}
		}
		if modulo {
			if e = m.som.modD(bounds); e != nil {
				return e
			}
		}
		remainder, e := m.som.mk(false)
		if e != nil {
			return e
		}
		if e = R.assign(m, remainder); e != nil {
			return e
		}
		if quotient {
			size, e := m.readWord(Q.value, 8)
			if e != nil {
				return e
			}
			for i := uint32(0); i < size; i++ {
				a, e := m.coefficientCore(Q.value, i)
				if e != nil {
					return e
				}
				term, e := m.monomialCore(Q.value, i)
				if e != nil {
					return e
				}
				if e = m.som2.addMul(&a, term, s.refs[pseudoLeadB].value); e != nil {
					return e
				}
			}
			if modulo {
				if e = m.som2.modD(bounds); e != nil {
					return e
				}
			}
			result, e := m.som2.mk(false)
			if e != nil {
				return e
			}
			if e = Q.assign(m, result); e != nil {
				return e
			}
		}
		*d++
	}
}
