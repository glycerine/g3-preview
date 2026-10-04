// Source: src/math/polynomial/polynomial.cpp:5694-5749; src/util/obj_ref.h.
// Licensed under the MIT license. Source author: Leonardo de Moura (leonardo).
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/memory"
)

type polynomialPSCNextScope struct {
	refs                                       [9]polynomialReference
	constructed, cursor                        uint8
	active, bufferConstructed, bufferDestroyed bool
	bodyCause, cleanupCause                    error
}

func (m *MultivariatePolynomialCore) finishPSCNext(owning bool) error {
	return m.finishPSCNextPhase(owning, owning)
}
func (m *MultivariatePolynomialCore) finishPSCNextPhase(owning, abort bool) error {
	s := &m.pscNext
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
		i := s.constructed - s.cursor - 1
		// H was constructed between tmp and x_pol.
		if i < 4 && s.bufferConstructed && !s.bufferDestroyed {
			if e := m.pscBuffer.finalizePhase(m, owning, false); e != nil {
				s.cleanupCause = e
				return e
			}
			s.bufferDestroyed = true
		}
		if e := s.refs[i].closeValue(m, owning); e != nil {
			s.cleanupCause = e
			return e
		}
		s.cursor++
	}
	if s.bufferConstructed && !s.bufferDestroyed {
		if e := m.pscBuffer.finalizePhase(m, owning, false); e != nil {
			s.cleanupCause = e
			return e
		}
		s.bufferDestroyed = true
	}
	if abort {
		for i := s.constructed; i > 0; i-- {
			if e := s.refs[i-1].abort(m); e != nil {
				s.cleanupCause = e
				return e
			}
		}
		if s.bufferConstructed {
			if e := m.pscBuffer.finalize(m, true); e != nil {
				s.cleanupCause = e
				return e
			}
		}
	}
	s.cleanupCause = nil
	for i := uint8(0); i < s.constructed; i++ {
		if s.refs[i].assigning || s.refs[i].incoming != (Polynomial{}) {
			return nil
		}
	}
	if s.bufferConstructed && (m.pscBuffer.incoming != (Polynomial{}) || m.pscBuffer.pending != (memory.Block{})) {
		return nil
	}
	*s = polynomialPSCNextScope{}
	return nil
}

func (m *MultivariatePolynomialCore) CleanupPSCNext() error {
	if m == nil || m.self != m {
		return ErrPolynomialOwner
	}
	return m.finishPSCNext(true)
}

func (m *MultivariatePolynomialCore) optimizedSe1(d, e uint32, a, sd1, se, den Polynomial, x uint32, out *polynomialReference) (err error) {
	if modularSourceDebug {
		v, z := m.Degree(a, x)
		if z != nil {
			return z
		}
		if v != d {
			return ErrPolynomialAssertion
		}
		v, z = m.Degree(sd1, x)
		if z != nil {
			return z
		}
		if v != e {
			return ErrPolynomialAssertion
		}
	}
	s := &m.pscNext
	if s.active {
		return ErrPolynomialPending
	}
	if e := m.pscEnter(); e != nil {
		return e
	}
	*s = polynomialPSCNextScope{active: true, constructed: 4}
	defer func() {
		m.pscLeave()
		s.bodyCause = err
		if z := m.finishPSCNext(false); z != nil {
			err = errors.Join(err, z)
		}
	}()
	cd1, lead, xj, tmp := &s.refs[0], &s.refs[1], &s.refs[2], &s.refs[3]
	p, z := m.LeadingCoefficient(sd1, x)
	if z != nil {
		return z
	}
	if z = cd1.assign(m, p); z != nil {
		return z
	}
	p, z = m.LeadingCoefficient(se, x)
	if z != nil {
		return z
	}
	if z = lead.assign(m, p); z != nil {
		return z
	}
	if z = m.pscBuffer.construct(m); z != nil {
		return z
	}
	s.bufferConstructed = true
	h := &m.pscBuffer
	if z = xj.assign(m, m.unit); z != nil {
		return z
	}
	for j := uint32(0); j <= e-1; j++ {
		p, z = m.MkVariable(x, j)
		if z != nil {
			return z
		}
		if z = xj.assign(m, p); z != nil {
			return z
		}
		p, z = m.Mul(lead.value, xj.value)
		if z != nil {
			return z
		}
		if z = h.push(m, p); z != nil {
			return z
		}
	}
	if modularSourceDebug && h.count != e {
		return ErrPolynomialAssertion
	}
	p, z = m.MkVariable(x, e)
	if z != nil {
		return z
	}
	if z = xj.assign(m, p); z != nil {
		return z
	}
	p, z = m.Mul(lead.value, xj.value)
	if z != nil {
		return z
	}
	if z = xj.assign(m, p); z != nil {
		return z
	}
	p, z = m.Sub(xj.value, se)
	if z != nil {
		return z
	}
	if z = h.push(m, p); z != nil {
		return z
	}
	if modularSourceDebug && h.count != e+1 {
		return ErrPolynomialAssertion
	}
	s.constructed = 7
	xpol, xh, xhe := &s.refs[4], &s.refs[5], &s.refs[6]
	p, z = m.MkVariable(x, 1)
	if z != nil {
		return z
	}
	if z = xpol.assign(m, p); z != nil {
		return z
	}
	for j := e + 1; j <= d-1; j++ {
		previous, z := h.get(m, j-1)
		if z != nil {
			return z
		}
		p, z = m.Mul(xpol.value, previous)
		if z != nil {
			return z
		}
		if z = xh.assign(m, p); z != nil {
			return z
		}
		p, z = m.Coeff(xh.value, x, e)
		if z != nil {
			return z
		}
		if z = xhe.assign(m, p); z != nil {
			return z
		}
		p, z = m.Mul(xhe.value, sd1)
		if z != nil {
			return z
		}
		if z = tmp.assign(m, p); z != nil {
			return z
		}
		p, z = m.ExactDiv(tmp.value, cd1.value)
		if z != nil {
			return z
		}
		if z = tmp.assign(m, p); z != nil {
			return z
		}
		p, z = m.Sub(xh.value, tmp.value)
		if z != nil {
			return z
		}
		if z = h.push(m, p); z != nil {
			return z
		}
	}
	if modularSourceDebug && h.count != d {
		return ErrPolynomialAssertion
	}
	s.constructed = 8
	D := &s.refs[7]
	if z = D.assign(m, m.zero); z != nil {
		return z
	}
	for j := uint32(0); j < d; j++ {
		p, z = m.Coeff(a, x, j)
		if z != nil {
			return z
		}
		if z = tmp.assign(m, p); z != nil {
			return z
		}
		v, z := h.get(m, j)
		if z != nil {
			return z
		}
		p, z = m.Mul(tmp.value, v)
		if z != nil {
			return z
		}
		if z = tmp.assign(m, p); z != nil {
			return z
		}
		p, z = m.Add(D.value, tmp.value)
		if z != nil {
			return z
		}
		if z = D.assign(m, p); z != nil {
			return z
		}
	}
	s.constructed = 9
	lca := &s.refs[8]
	p, z = m.LeadingCoefficient(a, x)
	if z != nil {
		return z
	}
	if z = lca.assign(m, p); z != nil {
		return z
	}
	p, z = m.ExactDiv(D.value, lca.value)
	if z != nil {
		return z
	}
	if z = D.assign(m, p); z != nil {
		return z
	}
	v, z := h.get(m, d-1)
	if z != nil {
		return z
	}
	p, z = m.Mul(xpol.value, v)
	if z != nil {
		return z
	}
	if z = xh.assign(m, p); z != nil {
		return z
	}
	p, z = m.Coeff(xh.value, x, e)
	if z != nil {
		return z
	}
	if z = xhe.assign(m, p); z != nil {
		return z
	}
	p, z = m.Mul(xhe.value, sd1)
	if z != nil {
		return z
	}
	if z = xhe.assign(m, p); z != nil {
		return z
	}
	p, z = m.Add(xh.value, D.value)
	if z != nil {
		return z
	}
	if z = out.assign(m, p); z != nil {
		return z
	}
	p, z = m.Mul(cd1.value, out.value)
	if z != nil {
		return z
	}
	if z = out.assign(m, p); z != nil {
		return z
	}
	p, z = m.Sub(out.value, xhe.value)
	if z != nil {
		return z
	}
	if z = out.assign(m, p); z != nil {
		return z
	}
	p, z = m.ExactDiv(out.value, den)
	if z != nil {
		return z
	}
	if z = out.assign(m, p); z != nil {
		return z
	}
	if (d-e+1)%2 == 1 {
		p, z = m.Neg(out.value)
		if z != nil {
			return z
		}
		return out.assign(m, p)
	}
	return nil
}
