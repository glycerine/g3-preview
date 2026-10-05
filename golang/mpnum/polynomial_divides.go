// Source: src/math/polynomial/polynomial.cpp:5154-5194;
// src/util/obj_ref.h; src/util/scoped_numeral.h.
// Licensed under the MIT license. Source author: Leonardo de Moura (leonardo).
package mpnum

import "errors"

type polynomialDividesScope struct {
	monomial                polynomialDivisionReference
	numeral                 Int
	numeric                 polynomialNumericRoots
	generation              uint64
	active, topOwner        bool
	stage                   uint8
	bodyCause, cleanupCause error
}

func (m *MultivariatePolynomialCore) finishDividesSource() error {
	s := &m.dividesLocal
	if s.cleanupCause != nil {
		return s.cleanupCause
	}
	if e := m.pscSourceStopped(); e != nil {
		s.cleanupCause = e
		return e
	}
	if s.stage == 0 {
		if _, e := sourceNumeralDel(m.integer, &s.numeral); e != nil {
			s.cleanupCause = e
			return e
		}
		s.stage = 1
	}
	if s.stage == 1 {
		if e := s.monomial.drop(m, &s.monomial.value, &s.monomial.decrementStarted, false); e != nil {
			s.cleanupCause = e
			return e
		}
		s.stage = 2
	}
	pending := s.monomial.assigning || s.topOwner && m.pscPending() || s.numeric.cell.object.OwnerID() != 0
	for _, raw := range s.numeric.allocator {
		pending = pending || raw.OwnerID() != 0
	}
	if !pending {
		*s = polynomialDividesScope{}
	}
	return nil
}
func (m *MultivariatePolynomialCore) CleanupDivides() error {
	if m == nil || m.self != m {
		return ErrPolynomialOwner
	}
	s := &m.dividesLocal
	if !s.active {
		return nil
	}
	if e := m.deleteNumeral(&s.numeral); e != nil {
		return e
	}
	if e := s.monomial.drop(m, &s.monomial.value, &s.monomial.decrementStarted, true); e != nil {
		return e
	}
	if e := s.monomial.drop(m, &s.monomial.incoming, &s.monomial.incomingStarted, true); e != nil {
		return e
	}
	s.monomial.assigning = false
	if e := s.numeric.cleanup(m.integer); e != nil {
		return e
	}
	if s.topOwner && s.generation == m.pscReceipts.generation {
		if e := m.CleanupPSC(); e != nil {
			return e
		}
	}
	*s = polynomialDividesScope{}
	return nil
}
func (m *MultivariatePolynomialManager) Divides(q, p Polynomial) (bool, error) {
	if e := m.ready(); e != nil {
		return false, e
	}
	return m.core.Divides(q, p)
}
func (m *MultivariatePolynomialCore) Divides(q, p Polynomial) (answer bool, err error) {
	if e := m.check(); e != nil {
		return false, e
	}
	zero, e := m.IsZero(p)
	if e != nil {
		return false, e
	}
	if zero {
		return true, nil
	}
	if modularSourceDebug {
		zero, e = m.IsZero(q)
		if e != nil {
			return false, e
		}
		if zero {
			return false, ErrPolynomialAssertion
		}
	}
	if m.dividesLocal.active {
		return false, ErrPolynomialPending
	}
	if e = m.som.reset(); e != nil {
		return false, e
	}
	if e = m.som2.reset(); e != nil {
		return false, e
	}
	if e = m.som.addPolynomial(p); e != nil {
		return false, e
	}
	maxQ, e := m.gradedLexMaxPosition(q)
	if e != nil {
		return false, e
	}
	mq, e := m.monomialCore(q, maxQ)
	if e != nil {
		return false, e
	}
	// Capture the source coefficient reference without reading its numeral.
	// Later arithmetic reloads the actual ObjectRef bytes, preserving both a
	// changed coefficient and a stale backing at the source-reached read.
	ql, e := m.polynomialLocation(q)
	if e != nil {
		return false, e
	}
	if modularSourceDebug {
		n, e := m.readWord(q, 8)
		if e != nil {
			return false, e
		}
		if maxQ >= n {
			return false, ErrPolynomialAssertion
		}
	}
	aq := integerArrayView{allocator: m.monomials.allocator, object: ql.raw, count: 1, offset: 12 + uint64(maxQ)*12, stride: 12}
	top := m.pscReceipts.depth == 0
	if e = m.pscEnter(); e != nil {
		return false, e
	}
	s := &m.dividesLocal
	*s = polynomialDividesScope{active: true, topOwner: top, generation: m.pscExecutingOwner()}
	defer func() { s.bodyCause = err; err = errors.Join(err, m.finishDividesSource()); m.pscLeave() }()
	for {
		if e = m.checkpoint(); e != nil {
			return false, e
		}
		maxR, e := m.som.gradedLexMaxPosition()
		if e != nil {
			return false, e
		}
		if maxR == ^uint32(0) {
			return true, nil
		}
		mr, e := m.som.ms.monomial(m, maxR)
		if e != nil {
			return false, e
		}
		if modularSourceDebug {
			n, e := m.som.as.size(m)
			if e != nil {
				return false, e
			}
			if maxR >= n {
				return false, ErrPolynomialAssertion
			}
		}
		ar := integerArrayView{runtime: m.memory, raw: m.som.as.raw, count: 1, offset: 8 + uint64(maxR)*12, stride: 12}
		quotient, divides, e := m.generatedQuotient(mr, mq)
		if e != nil {
			return false, e
		}
		if e = s.monomial.assign(m, quotient); e != nil {
			return false, e
		}
		if !divides {
			return false, nil
		}
		divisor, e := aq.checkedAt(m.memory, 0)
		if e != nil {
			return false, e
		}
		remainder, e := ar.checkedAt(m.memory, 0)
		if e != nil {
			return false, e
		}
		ok, e := m.modular.Divides(&divisor, &remainder)
		if e != nil {
			return false, e
		}
		if !ok {
			return false, nil
		}
		remainder, e = ar.checkedAt(m.memory, 0)
		if e != nil {
			return false, e
		}
		divisor, e = aq.checkedAt(m.memory, 0)
		if e != nil {
			return false, e
		}
		before, e := polynomialNumericBefore(m.integer)
		if e != nil {
			return false, e
		}
		e = m.modular.Div(&remainder, &divisor, &s.numeral)
		e = errors.Join(e, s.numeric.after(m.integer, before))
		if e != nil {
			return false, e
		}
		if e = m.modular.Neg(&s.numeral); e != nil {
			return false, e
		}
		if e = m.som.addMul(&s.numeral, quotient, q); e != nil {
			return false, e
		}
	}
}
