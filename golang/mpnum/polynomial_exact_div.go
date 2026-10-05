// Source: src/math/polynomial/polynomial.cpp:5094-5156; src/util/obj_ref.h.
// Licensed under the MIT license.
package mpnum

import "errors"

var ErrPolynomialUnreachable = errors.New("mpnum: polynomial source compiler unreachable")

type polynomialDivisionReference struct {
	value, incoming                              Monomial
	decrementStarted, incomingStarted, assigning bool
	cause                                        error
}

func (r *polynomialDivisionReference) drop(m *MultivariatePolynomialCore, p *Monomial, started *bool, owning bool) error {
	if *p == (Monomial{}) {
		return nil
	}
	if *started {
		if !owning {
			return r.cause
		}
		if e := m.monomials.CleanupValue(p); e != nil {
			return e
		}
		*started = false
		return nil
	}
	e := m.monomials.DecRef(*p)
	if e != nil {
		if l, x := m.monomials.location(*p); x == nil && l.stage != 0 {
			*started = true
		}
		r.cause = e
		return e
	}
	*p = Monomial{}
	return nil
}
func (r *polynomialDivisionReference) assign(m *MultivariatePolynomialCore, p Monomial) error {
	if r.assigning {
		return ErrPolynomialPending
	}
	if p != (Monomial{}) {
		if e := m.monomials.IncRef(p); e != nil {
			return e
		}
	}
	r.incoming = p
	r.assigning = true
	if m.generatedActive && m.generatedMonomial == p {
		m.transferGeneratedMonomial()
	}
	if e := r.drop(m, &r.value, &r.decrementStarted, false); e != nil {
		return e
	}
	r.value = r.incoming
	r.incoming = Monomial{}
	r.assigning = false
	return nil
}

// Both records are actual method locals embedded in the charged core header.
// Scalar exact_div's numeral is RAW: no source Del on pre-end exceptions.
type polynomialExactDivisionScope struct {
	invocation                           uint64
	pscOwner                             uint64
	monomial                             polynomialDivisionReference
	numeral                              Int
	numeric                              polynomialNumericRoots
	returned                             Polynomial
	kind, stage                          uint8
	active, endEntered                   bool
	ownStage                             uint8
	ownStarted                           bool
	bodyCause, sourceCause, cleanupCause error
}

func (m *MultivariatePolynomialCore) generatedQuotient(a, b Monomial) (Monomial, bool, error) {
	if m.generatedActive {
		return Monomial{}, false, ErrPolynomialPending
	}
	outcome := monomialFactoryOutcome{}
	r, ok, e := m.monomials.divideCreated(a, b, &outcome)
	if e != nil && outcome.started {
		m.recordGeneratedCallee(outcome)
	}
	if e == nil && outcome.created {
		m.generatedMonomial = r
		m.generatedActive = true
		m.generatedPSC = m.pscExecutingOwner()
	}
	return r, ok, e
}
func (m *MultivariatePolynomialCore) exactDivisionCleanup(owning bool) error {
	return m.exactDivisionCleanupPhase(owning, owning)
}
func (m *MultivariatePolynomialCore) exactDivisionCleanupPhase(owning, abort bool) error {
	s := &m.exactDivision
	if !s.active {
		return nil
	}
	if s.kind == 2 && !abort && !s.endEntered {
		return nil
	}
	if !owning && s.sourceCause != nil {
		return s.sourceCause
	}
	cursor := &s.stage
	if owning {
		if !s.ownStarted {
			s.ownStage = s.stage
			s.ownStarted = true
		}
		cursor = &s.ownStage
	}
	if *cursor == 0 {
		var e error
		if owning {
			e = m.deleteNumeral(&s.numeral)
		} else {
			_, e = upolynomialSourceNumeralDel(m.integer, &s.numeral)
		}
		if e != nil {
			if !owning {
				s.sourceCause = e
			}
			return e
		}
		*cursor = 1
	}
	if s.kind == 1 && *cursor == 1 {
		if e := s.monomial.drop(m, &s.monomial.value, &s.monomial.decrementStarted, owning); e != nil {
			if !owning {
				s.sourceCause = e
			}
			return e
		}
		*cursor = 2
	}
	pendingNumeric := s.numeric.cell.object.OwnerID() != 0
	for _, raw := range s.numeric.allocator {
		pendingNumeric = pendingNumeric || raw.OwnerID() != 0
	}
	if !abort && (s.monomial.assigning || pendingNumeric) {
		return nil
	}
	if abort {
		if e := s.numeric.cleanup(m.integer); e != nil {
			return e
		}
		if e := s.monomial.drop(m, &s.monomial.incoming, &s.monomial.incomingStarted, true); e != nil {
			return e
		}
		s.monomial.assigning = false
		if s.returned != (Polynomial{}) {
			if e := m.CleanupValue(&s.returned); e != nil {
				return e
			}
		}
	}
	if s.returned != (Polynomial{}) && !abort {
		return nil
	}
	if owning {
		s.active = false
		// The matched cohort borrow ends with the actual owning continuation;
		// retain source cause/cursor independently for diagnostics.
		s.pscOwner = 0
		s.cleanupCause = nil
		return nil
	}
	*s = polynomialExactDivisionScope{invocation: s.invocation}
	return nil
}
func (m *MultivariatePolynomialCore) CleanupExactDivision() error {
	if m == nil || m.self != m {
		return ErrPolynomialOwner
	}
	e := m.exactDivisionCleanup(true)
	if e != nil {
		m.exactDivision.cleanupCause = e
	}
	return e
}
func (m *MultivariatePolynomialCore) ExactDivScalar(p Polynomial, c *Int) (result Polynomial, err error) {
	if e := m.check(); e != nil {
		return Polynomial{}, e
	}
	if modularSourceDebug {
		zero, e := m.modular.IsZero(c)
		if e != nil {
			return Polynomial{}, e
		}
		if zero {
			return Polynomial{}, ErrPolynomialAssertion
		}
	}
	if m.exactDivision.active {
		return Polynomial{}, ErrPolynomialPending
	}
	if e := m.som.reset(); e != nil {
		return Polynomial{}, e
	}
	s := &m.exactDivision
	if s.invocation == ^uint64(0) {
		return Polynomial{}, ErrPolynomialPending
	}
	*s = polynomialExactDivisionScope{active: true, kind: 2, pscOwner: m.pscExecutingOwner(), invocation: s.invocation + 1}
	defer func() {
		if !s.active {
			return
		}
		s.bodyCause = err
		// No heap-owned numeral remains when only an inline scalar was published.
		if err != nil && !s.endEntered && s.numeral.cell == 0 {
			*s = polynomialExactDivisionScope{invocation: s.invocation}
		}
	}()
	n, e := m.readWord(p, 8)
	if e != nil {
		return Polynomial{}, e
	}
	for i := uint32(0); i < n; i++ {
		if modularSourceDebug {
			a, e := m.coefficientCore(p, i)
			if e != nil {
				return Polynomial{}, e
			}
			divisible, e := m.modular.Divides(c, &a)
			if e != nil {
				return Polynomial{}, e
			}
			if !divisible {
				return Polynomial{}, ErrPolynomialAssertion
			}
		}
		a, e := m.coefficientCore(p, i)
		if e != nil {
			return Polynomial{}, e
		}
		if e = m.modular.Div(&a, c, &s.numeral); e != nil {
			return Polynomial{}, e
		}
		zero, e := m.modular.IsZero(&s.numeral)
		if e != nil {
			return Polynomial{}, e
		}
		if !zero {
			term, e := m.monomialCore(p, i)
			if e != nil {
				return Polynomial{}, e
			}
			if e = m.som.add(&s.numeral, term, false); e != nil {
				return Polynomial{}, e
			}
		}
	}
	s.endEntered = true
	if e = m.exactDivisionCleanup(false); e != nil {
		s.cleanupCause = e
		return Polynomial{}, e
	}
	return m.som.mk(false)
}
func (m *MultivariatePolynomialCore) ExactDiv(p, q Polynomial) (Polynomial, error) {
	return m.exactDivWithReceipt(p, q, nil)
}
func (m *MultivariatePolynomialCore) exactDivWithReceipt(p, q Polynomial, receipt *polynomialExactDivInvocationReceipt) (result Polynomial, err error) {
	if e := m.check(); e != nil {
		return Polynomial{}, e
	}
	zero, e := m.IsZero(p)
	if e != nil {
		return Polynomial{}, e
	}
	if zero {
		return p, nil
	}
	if modularSourceDebug {
		zero, e = m.IsZero(q)
		if e != nil {
			return Polynomial{}, e
		}
		if zero {
			return Polynomial{}, ErrPolynomialAssertion
		}
	}
	if m.exactDivision.active {
		return Polynomial{}, ErrPolynomialPending
	}
	if e = m.som.reset(); e != nil {
		return Polynomial{}, e
	}
	if e = m.som2.reset(); e != nil {
		return Polynomial{}, e
	}
	if e = m.som.addPolynomial(p); e != nil {
		return Polynomial{}, e
	}
	maxQ, e := m.gradedLexMaxPosition(q)
	if e != nil {
		return Polynomial{}, e
	}
	monomialQ, e := m.monomialCore(q, maxQ)
	if e != nil {
		return Polynomial{}, e
	}
	coefficientQ, e := m.coefficientCore(q, maxQ)
	if e != nil {
		return Polynomial{}, e
	}
	s := &m.exactDivision
	if s.invocation == ^uint64(0) {
		return Polynomial{}, ErrPolynomialPending
	}
	*s = polynomialExactDivisionScope{active: true, kind: 1, pscOwner: m.pscExecutingOwner(), invocation: s.invocation + 1}
	if receipt != nil {
		*receipt = polynomialExactDivInvocationReceipt{m, s, s.invocation}
	}
	defer func() {
		s.bodyCause = err
		cleanup := m.exactDivisionCleanup(false)
		if cleanup != nil {
			s.cleanupCause = cleanup
			// The source evaluates C.mk before destroying locals. Its actual new
			// returned object remains named if a later local destruction refuses.
			if result != (Polynomial{}) {
				s.returned = result
				result = Polynomial{}
			}
			err = errors.Join(err, cleanup)
		}
	}()
	for {
		if e = m.checkpoint(); e != nil {
			return Polynomial{}, e
		}
		maxR, e := m.som.gradedLexMaxPosition()
		if e != nil {
			return Polynomial{}, e
		}
		if maxR == ^uint32(0) {
			if e = m.som.reset(); e != nil {
				return Polynomial{}, e
			}
			return m.som2.mk(false)
		}
		monomialR, e := m.som.ms.monomial(m, maxR)
		if e != nil {
			return Polynomial{}, e
		}
		coefficientR, e := m.som.as.coefficient(m, maxR)
		if e != nil {
			return Polynomial{}, e
		}
		quotient, divides, e := m.generatedQuotient(monomialR, monomialQ)
		if e != nil {
			return Polynomial{}, e
		}
		// VERIFY evaluates in every profile. The selected GNU release profile
		// invokes __builtin_unreachable on false; this is a typed UB guard.
		if !divides {
			if modularSourceDebug {
				return Polynomial{}, ErrPolynomialAssertion
			}
			return Polynomial{}, ErrPolynomialUnreachable
		}
		if e = s.monomial.assign(m, quotient); e != nil {
			return Polynomial{}, e
		}
		if e = m.modular.Div(&coefficientR, &coefficientQ, &s.numeral); e != nil {
			return Polynomial{}, e
		}
		if e = m.som2.addConsumedNumeric(&s.numeral, quotient, false, nil, &s.numeric); e != nil {
			return Polynomial{}, e
		}
		if e = m.modular.Neg(&s.numeral); e != nil {
			return Polynomial{}, e
		}
		if e = m.som.addMul(&s.numeral, quotient, q); e != nil {
			return Polynomial{}, e
		}
	}
}
