// Source: src/math/polynomial/polynomial.cpp:5751-5810,5873-5901.
// Licensed under the MIT license. Source author: Leonardo de Moura (leonardo).
package mpnum

import "errors"

type polynomialPSCScope struct {
	refs                    [7]PolynomialRef
	constructed, cursor     uint8
	active                  bool
	bodyCause, cleanupCause error
}

func (m *MultivariatePolynomialCore) pscConstruct(i uint8) {
	r := &m.pscLocal.refs[i]
	*r = PolynomialRef{owner: m}
	r.self = r
	m.pscLocal.constructed = i + 1
}
func (m *MultivariatePolynomialCore) finishPSC(owning bool) error {
	return m.finishPSCPhase(owning, owning)
}
func (m *MultivariatePolynomialCore) finishPSCPhase(owning, abort bool) error {
	s := &m.pscLocal
	if !s.active {
		return nil
	}
	if !owning {
		if e := m.pscSourceStopped(); e != nil {
			return e
		}
		if m.lazardLocal.cleanupCause != nil {
			return m.lazardLocal.cleanupCause
		}
		if m.pscNext.cleanupCause != nil {
			return m.pscNext.cleanupCause
		}
		if s.cleanupCause != nil {
			return s.cleanupCause
		}
	}
	for s.cursor < s.constructed {
		r := &s.refs[s.constructed-s.cursor-1].root
		if e := r.closeValue(m, owning); e != nil {
			s.cleanupCause = e
			return e
		}
		s.cursor++
	}
	if abort {
		for i := s.constructed; i > 0; i-- {
			if e := s.refs[i-1].root.abort(m); e != nil {
				s.cleanupCause = e
				return e
			}
		}
	}
	s.cleanupCause = nil
	for i := uint8(0); i < s.constructed; i++ {
		r := &s.refs[i].root
		if r.assigning || r.incoming != (Polynomial{}) {
			return nil
		}
	}
	*s = polynomialPSCScope{}
	return nil
}

func (m *MultivariatePolynomialManager) PSCChain(p, q Polynomial, x uint32, S *PolynomialRefVector) error {
	if e := m.ready(); e != nil {
		return e
	}
	return m.core.PSCChain(p, q, x, S)
}

func (m *MultivariatePolynomialCore) PSCChain(p, q Polynomial, x uint32, S *PolynomialRefVector) error {
	if e := m.check(); e != nil {
		return e
	}
	if S == nil || S.owner != m {
		return ErrPolynomialOwner
	}
	if e := S.open(); e != nil {
		return e
	}
	if modularSourceDebug {
		d, e := m.Degree(p, x)
		if e != nil {
			return e
		}
		if d == 0 {
			return ErrPolynomialAssertion
		}
		d, e = m.Degree(q, x)
		if e != nil {
			return e
		}
		if d == 0 {
			return ErrPolynomialAssertion
		}
	}
	if e := S.Reset(); e != nil {
		return e
	}
	dp, e := m.Degree(p, x)
	if e != nil {
		return e
	}
	dq, e := m.Degree(q, x)
	if e != nil {
		return e
	}
	if dp >= dq {
		e = m.pscOptimizedCore(p, q, x, S)
	} else {
		e = m.pscOptimizedCore(q, p, x, S)
	}
	if e != nil {
		return e
	}
	n, e := S.Size()
	if e != nil {
		return e
	}
	if n == 0 {
		if e = S.Push(m.zero); e != nil {
			return e
		}
		n = 1
	}
	// std::reverse adaptation: exchange actual scalar pointer handles in place.
	for i := uint32(0); i < n/2; i++ {
		a, e := S.state.get(m, i)
		if e != nil {
			return e
		}
		b, e := S.state.get(m, n-i-1)
		if e != nil {
			return e
		}
		if e = S.state.values.putMonomial(m, i, Monomial{owner: b.owner, slot: b.slot, generation: b.generation}); e != nil {
			return e
		}
		if e = S.state.values.putMonomial(m, n-i-1, Monomial{owner: a.owner, slot: a.slot, generation: a.generation}); e != nil {
			return e
		}
	}
	return nil
}

func (m *MultivariatePolynomialCore) pscOptimizedCore(p, q Polynomial, x uint32, S *PolynomialRefVector) (err error) {
	dp, e := m.Degree(p, x)
	if e != nil {
		return e
	}
	dq, e := m.Degree(q, x)
	if e != nil {
		return e
	}
	if modularSourceDebug && dp < dq {
		return ErrPolynomialAssertion
	}
	sc := &m.pscLocal
	if sc.active {
		return ErrPolynomialPending
	}
	if e := m.pscEnter(); e != nil {
		return e
	}
	*sc = polynomialPSCScope{active: true}
	for i := uint8(0); i < 6; i++ {
		m.pscConstruct(i)
	}
	defer func() {
		m.pscLeave()
		sc.bodyCause = err
		if e := m.finishPSC(false); e != nil {
			err = errors.Join(err, e)
		}
	}()
	a, b, c, minus, lc, ps := &sc.refs[0].root, &sc.refs[1].root, &sc.refs[2].root, &sc.refs[3].root, &sc.refs[4].root, &sc.refs[5].root
	t, e := m.LeadingCoefficient(q, x)
	if e != nil {
		return e
	}
	if e = lc.assign(m, t); e != nil {
		return e
	}
	m.pscConstruct(6)
	s := &sc.refs[6].root
	if e = m.powerInto(lc.value, dp-dq, s); e != nil {
		return e
	}
	t, e = m.Neg(q)
	if e != nil {
		return e
	}
	if e = minus.assign(m, t); e != nil {
		return e
	}
	if e = a.assign(m, q); e != nil {
		return e
	}
	if e = m.ExactPseudoRemainder(p, minus.value, x, &sc.refs[1]); e != nil {
		return e
	}
	for {
		d, e := m.Degree(a.value, x)
		if e != nil {
			return e
		}
		degree, e := m.Degree(b.value, x)
		if e != nil {
			return e
		}
		zero, e := m.IsZero(b.value)
		if e != nil {
			return e
		}
		if zero {
			return nil
		}
		t, e = m.Coeff(b.value, x, d-1)
		if e != nil {
			return e
		}
		if e = ps.assign(m, t); e != nil {
			return e
		}
		zero, e = m.IsZero(ps.value)
		if e != nil {
			return e
		}
		if !zero {
			if e = S.Push(ps.value); e != nil {
				return e
			}
		}
		delta := d - degree
		if delta > 1 {
			if e = m.seLazard(d, s.value, b.value, x, c); e != nil {
				return e
			}
			t, e = m.Coeff(c.value, x, degree)
			if e != nil {
				return e
			}
			if e = ps.assign(m, t); e != nil {
				return e
			}
			zero, e = m.IsZero(ps.value)
			if e != nil {
				return e
			}
			if !zero {
				if e = S.Push(ps.value); e != nil {
					return e
				}
			}
		} else {
			if modularSourceDebug && delta != 0 && delta != 1 {
				return ErrPolynomialAssertion
			}
			if e = c.assign(m, b.value); e != nil {
				return e
			}
		}
		if degree == 0 {
			return nil
		}
		if e = m.optimizedSe1(d, degree, a.value, b.value, c.value, s.value, x, b); e != nil {
			return e
		}
		if e = a.assign(m, c.value); e != nil {
			return e
		}
		t, e = m.LeadingCoefficient(a.value, x)
		if e != nil {
			return e
		}
		if e = s.assign(m, t); e != nil {
			return e
		}
	}
}
