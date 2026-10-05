// Source: src/util/rational.h:54,67; src/util/mpq.h:186-189;
// src/math/polynomial/polynomial.cpp:2530-2536,2696-2698.
// Licensed under the MIT license. Author: Leonardo de Moura.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
)

// The source global rational binding is supplied as an actual borrowed owner.
// This is call-local source state embedded in the charged prerequisite header,
// not an imp-owned rational manager or an escaping per-call scope allocation.
type polynomialConstantScope struct {
	rational                                                       Rational
	numeral                                                        Int
	result                                                         Polynomial
	active, rationalCompleted, numeralCompleted, failedConstructor bool
	cause, sourceCause                                             error
	stopped                                                        bool
	rationalStage                                                  uint8
}

func (m *MultivariatePolynomialCore) beginConstantScope() error {
	if m.config.Rationals == nil {
		return arena.ErrConfig
	}
	if e := m.config.Rationals.checkOwnerReady(); e != nil {
		return e
	}
	if m.constantScope.active {
		return ErrPolynomialPending
	}
	m.constantGeneration++
	if f := m.factorsOwner; f != nil && f.executing && m.callbackDepth == 0 {
		f.constantGeneration = m.constantGeneration
	}
	m.constantScope = polynomialConstantScope{active: true, rational: NewRational()}
	if s := m.compositionOwner; s != nil && s.executing {
		s.constantGeneration = m.constantGeneration
	}
	return nil
}
func (m *MultivariatePolynomialCore) finishConstantScope(body error) error {
	s := &m.constantScope
	if s.cause == nil {
		s.cause = body
	}
	if s.stopped {
		return errors.Join(body, s.sourceCause)
	}
	if s.numeralCompleted {
		if _, e := upolynomialSourceNumeralDel(m.integer, &s.numeral); e != nil {
			s.stopped = true
			s.sourceCause = e
			return errors.Join(body, e)
		}
		s.numeralCompleted = false
	}
	if s.rationalCompleted {
		roots := [2]*Int{&s.rational.num, &s.rational.den}
		for s.rationalStage < 2 {
			if _, e := upolynomialSourceNumeralDel(m.integer, roots[s.rationalStage]); e != nil {
				s.stopped = true
				s.sourceCause = e
				return errors.Join(body, e)
			}
			s.rationalStage++
		}
		s.rationalCompleted = false
	}
	if s.failedConstructor {
		return body
	}
	if body == nil {
		s.active = false
	} else if s.result == (Polynomial{}) {
		s.active = false
	}
	if !s.active {
		*s = polynomialConstantScope{}
	}
	return body
}
func (m *MultivariatePolynomialCore) MkConstantTakeCopy(a *Int) (Polynomial, error) {
	if e := m.check(); e != nil {
		return Polynomial{}, e
	}
	if a == nil {
		return Polynomial{}, arena.ErrConfig
	}
	if e := m.beginConstantScope(); e != nil {
		return Polynomial{}, e
	}
	s := &m.constantScope
	before, e := m.compositionNumericBefore()
	if e != nil {
		return Polynomial{}, e
	}
	e = m.compositionNumericAfter(before, m.config.Rationals.SetInteger(&s.rational, a))
	if e != nil {
		s.failedConstructor = true
		s.cause = e
		return Polynomial{}, e
	}
	s.rationalCompleted = true
	return m.constantFromRationalScope()
}
func (m *MultivariatePolynomialCore) constantFromRationalScope() (Polynomial, error) {
	s := &m.constantScope
	if modularSourceDebug && !isIntegerRational(&s.rational) {
		return Polynomial{}, m.finishConstantScope(ErrPolynomialAssertion)
	}
	s.numeral = Int{}
	s.numeralCompleted = true
	if e := m.compositionNumericCall(0, &s.rational.num, nil, nil, &s.numeral); e != nil {
		return Polynomial{}, m.finishConstantScope(e)
	}
	p, e := m.MkConstantTake(&s.numeral)
	if e != nil {
		return Polynomial{}, m.finishConstantScope(e)
	}
	s.result = p
	if e = m.finishConstantScope(nil); e != nil {
		return Polynomial{}, e
	}
	return p, nil
}
func (m *MultivariatePolynomialCore) cleanupConstantScope() error {
	s := &m.constantScope
	if !s.active {
		return nil
	}
	if s.result != (Polynomial{}) && s.result != m.zero && s.result != m.unit {
		result := s.result
		e := m.CleanupValue(&s.result)
		// Placement returned one real polynomial, borrowed by the matching
		// operation receipt. Acknowledge its actual owning consumption here;
		// no directory/refcount inference or second deletion is performed.
		if s.result == (Polynomial{}) {
			if f := m.factorsOwner; f != nil && f.constantGeneration == m.constantGeneration && f.freshResult == result {
				f.freshResult = Polynomial{}
			}
			if c := m.compositionOwner; c != nil && c.constantGeneration == m.constantGeneration && c.freshResult == result {
				c.freshResult = Polynomial{}
			}
		}
		if e != nil {
			return e
		}
	}
	if s.numeralCompleted {
		if e := m.integer.Del(&s.numeral); e != nil {
			return e
		}
		s.numeralCompleted = false
	}
	// Explicit owning cleanup of the real fields of an incomplete rational
	// constructor, not execution of its nonexistent completed destructor.
	if e := m.config.Rationals.Del(&s.rational); e != nil {
		return e
	}
	cause, sourceCause, stopped := s.cause, s.sourceCause, s.stopped
	*s = polynomialConstantScope{cause: cause, sourceCause: sourceCause, stopped: stopped}
	return nil
}
