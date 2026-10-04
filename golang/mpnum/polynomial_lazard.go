// Source: src/math/polynomial/polynomial.cpp:5654-5691; src/util/obj_ref.h.
// Licensed under the MIT license. Source author: Leonardo de Moura (leonardo).
package mpnum

import "errors"

// These are the actual two completed source locals, charged in the core's
// cold header. No source local exists on the n==0 branch.
type polynomialLazardScope struct {
	x, c                    polynomialReference
	constructed, cursor     uint8
	active                  bool
	bodyCause, cleanupCause error
}

func (m *MultivariatePolynomialCore) finishLazard(owning bool) error {
	return m.finishLazardPhase(owning, owning)
}
func (m *MultivariatePolynomialCore) finishLazardPhase(owning, abort bool) error {
	s := &m.lazardLocal
	if !s.active {
		return nil
	}
	if !owning {
		if e := m.pscSourceStopped(); e != nil {
			return e
		}
	}
	if s.cleanupCause != nil && !owning {
		return s.cleanupCause
	}
	for s.cursor < s.constructed {
		r := &s.x
		if s.constructed-s.cursor == 2 {
			r = &s.c
		}
		if e := r.closeValue(m, owning); e != nil {
			s.cleanupCause = e
			return e
		}
		s.cursor++
	}
	if abort {
		if s.constructed == 2 {
			if e := s.c.abort(m); e != nil {
				s.cleanupCause = e
				return e
			}
		}
		if s.constructed >= 1 {
			if e := s.x.abort(m); e != nil {
				s.cleanupCause = e
				return e
			}
		}
	}
	s.cleanupCause = nil
	if !s.x.assigning && !s.c.assigning && s.x.incoming == (Polynomial{}) && s.c.incoming == (Polynomial{}) {
		*s = polynomialLazardScope{}
	}
	return nil
}

// CleanupLazard acknowledges stopped physical deletion only as an explicit
// owning action, after all completed source locals have unwound.
func (m *MultivariatePolynomialCore) CleanupLazard() error {
	if m == nil || m.self != m {
		return ErrPolynomialOwner
	}
	return m.finishLazard(true)
}

func (m *MultivariatePolynomialCore) seLazard(d uint32, y, sd1 Polynomial, x uint32, out *polynomialReference) (err error) {
	degree, e := m.Degree(sd1, x)
	if e != nil {
		return e
	}
	n := d - degree - 1
	if n == 0 {
		return out.assign(m, sd1)
	}
	s := &m.lazardLocal
	if s.active {
		return ErrPolynomialPending
	}
	if e := m.pscEnter(); e != nil {
		return e
	}
	*s = polynomialLazardScope{active: true, constructed: 1}
	defer func() {
		m.pscLeave()
		s.bodyCause = err
		if e := m.finishLazard(false); e != nil {
			err = errors.Join(err, e)
		}
	}()
	p, e := m.LeadingCoefficient(sd1, x)
	if e != nil {
		return e
	}
	if e = s.x.assign(m, p); e != nil {
		return e
	}
	a := uint32(1)
	for v := n; v > 1; v >>= 1 {
		a <<= 1
	}
	if modularSourceDebug && !(a <= n && n < 2*a) {
		return ErrPolynomialAssertion
	}
	s.constructed = 2
	if e = s.c.assign(m, s.x.value); e != nil {
		return e
	}
	n -= a
	for a != 1 {
		a /= 2
		p, e = m.Mul(s.c.value, s.c.value)
		if e != nil {
			return e
		}
		if e = s.c.assign(m, p); e != nil {
			return e
		}
		p, e = m.ExactDiv(s.c.value, y)
		if e != nil {
			return e
		}
		if e = s.c.assign(m, p); e != nil {
			return e
		}
		if n >= a {
			p, e = m.Mul(s.c.value, s.x.value)
			if e != nil {
				return e
			}
			if e = s.c.assign(m, p); e != nil {
				return e
			}
			p, e = m.ExactDiv(s.c.value, y)
			if e != nil {
				return e
			}
			if e = s.c.assign(m, p); e != nil {
				return e
			}
			n -= a
		}
	}
	p, e = m.Mul(s.c.value, sd1)
	if e != nil {
		return e
	}
	if e = out.assign(m, p); e != nil {
		return e
	}
	p, e = m.ExactDiv(out.value, y)
	if e != nil {
		return e
	}
	return out.assign(m, p)
}
