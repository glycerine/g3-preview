// Source: src/util/rational.h:54,67; src/math/polynomial/polynomial.cpp:2530-2536,2696-2698. Licensed under the MIT license.
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
	cause                                                          error
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
	m.constantScope = polynomialConstantScope{active: true, rational: NewRational()}
	return nil
}
func (m *MultivariatePolynomialCore) finishConstantScope(body error) error {
	s := &m.constantScope
	s.cause = body
	if s.numeralCompleted {
		if e := m.deleteNumeral(&s.numeral); e != nil {
			return errors.Join(body, e)
		}
		s.numeralCompleted = false
	}
	if s.rationalCompleted {
		if e := m.config.Rationals.Del(&s.rational); e != nil {
			return errors.Join(body, e)
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
	if e := m.config.Rationals.SetInteger(&s.rational, a); e != nil {
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
	if e := m.modular.Copy(&s.numeral, &s.rational.num); e != nil {
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
		if e := m.deleteCore(s.result); e != nil {
			return e
		}
		s.result = Polynomial{}
	}
	if s.numeralCompleted {
		if e := m.deleteNumeral(&s.numeral); e != nil {
			return e
		}
		s.numeralCompleted = false
	}
	// Explicit owning cleanup of the real fields of an incomplete rational
	// constructor, not execution of its nonexistent completed destructor.
	if e := m.config.Rationals.Del(&s.rational); e != nil {
		return e
	}
	*s = polynomialConstantScope{}
	return nil
}
