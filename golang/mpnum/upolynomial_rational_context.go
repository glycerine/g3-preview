// Licensed under the MIT license.
// Source sexpr2upolynomial.cpp: owning rational constant and borrowed exponent.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
)

// NewUPolynomialManager borrows the service's actual integer owner. Close the
// polynomial manager before the service; no serialized numeral bridge is used.
func (s *RationalService) NewUPolynomialManager(c UPolynomialConfig) (*UPolynomialManager, error) {
	if e := s.lock(); e != nil {
		return nil, e
	}
	defer s.mu.Unlock()
	if e := s.lexicalReady(); e != nil {
		return nil, e
	}
	if c.Integer != nil && c.Integer != s.manager.z {
		return nil, ErrUPolynomialOwner
	}
	c.Integer = s.manager.z
	return NewUPolynomialManager(c)
}

// This concrete context borrows a locked service for the callback only. It
// keeps partial constructor abort behind the caller's completed vector unwind.
// The callback must not call public methods on that same rational service.
type UPolynomialRationalContext struct {
	service    *RationalService
	polynomial *UPolynomialManager
	active     bool
}

func (s *RationalService) WithUPolynomialRationals(m *UPolynomialManager, body func(*UPolynomialRationalContext) error) (_lexErr error) {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if m == nil || m.config.Integer != s.manager.z {
		return ErrUPolynomialOwner
	}
	if e := m.check(); e != nil {
		return e
	}
	if body == nil {
		return arena.ErrConfig
	}
	if e := s.beginLexical(); e != nil {
		return e
	}
	defer func() {
		s.endLexicalError(&_lexErr)
		if s.lexical.pending() && s.lexical.cause != nil && !errors.Is(_lexErr, s.lexical.cause) {
			_lexErr = errors.Join(_lexErr, s.lexical.cause)
		}
	}()
	c := &s.polynomialContext
	*c = UPolynomialRationalContext{service: s, polynomial: m, active: true}
	defer func() { *c = UPolynomialRationalContext{} }()
	return body(c)
}
func (c *UPolynomialRationalContext) Pending() bool {
	return c.valid() && c.service.lexical.cause != nil && c.service.lexical.pending()
}
func (c *UPolynomialRationalContext) valid() bool {
	return c != nil && c.active && c.service != nil && c == &c.service.polynomialContext
}
func (c *UPolynomialRationalContext) StopVectors(e error) {
	if c.valid() && e != nil && c.service.lexical.pending() && c.service.lexical.cause == nil {
		c.service.lexical.cause = e
	}
}
func (c *UPolynomialRationalContext) SetConstant(p UPolynomial, a *OwnedRational, nonInteger func() error) (_lexErr error) {
	if !c.valid() {
		return ErrRationalServiceOwner
	}
	s := c.service
	if e := s.rationalScalarOwner(a); e != nil {
		return e
	}
	if nonInteger == nil {
		return arena.ErrConfig
	}
	if e := s.beginLexical(); e != nil {
		return e
	}
	defer s.endLexicalError(&_lexErr)
	value := NewRational()
	complete := false
	defer s.endLexicalLocal(&value, &complete)
	if e := s.manager.Copy(&value, &a.raw); e != nil {
		return e
	}
	complete = true
	integer, e := s.manager.IsInt(&value)
	if e != nil {
		return e
	}
	if !integer {
		return nonInteger()
	}
	return c.polynomial.SetRationals(p, []Rational{value})
}

func (c *UPolynomialRationalContext) Exponent(a *OwnedRational) (uint32, bool, error) {
	if !c.valid() {
		return 0, false, ErrRationalServiceOwner
	}
	s := c.service
	if e := s.rationalScalarOwner(a); e != nil {
		return 0, false, e
	}
	ok, e := s.manager.IsUnsigned(&a.raw)
	if e != nil || !ok {
		return 0, ok, e
	}
	n, e := s.manager.Unsigned(&a.raw)
	return n, true, e
}

func (c *UPolynomialRationalContext) SetIdentity(p UPolynomial) error {
	if !c.valid() {
		return ErrRationalServiceOwner
	}
	return c.polynomial.SetRationals(p, []Rational{NewRational(), {num: NewInt(1), den: NewInt(1)}})
}

// CleanupUPolynomialRationals acknowledges completed scalar destruction, then
// the actual caller's older vectors, before partial-constructor owning abort.
func (s *RationalService) CleanupUPolynomialRationals(vectors func() error) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if vectors == nil {
		return arena.ErrConfig
	}
	l := &s.lexical
	for i := uint8(0); i < l.count; i++ {
		if l.done&(1<<i) != 0 || l.abort[i] {
			continue
		}
		if e := lexicalDelete(s.manager.z, &l.roots[i], true, &l.components[i]); e != nil {
			l.cause = e
			return e
		}
		l.done |= 1 << i
		for l.cursor < l.count && l.done&(1<<l.cursor) != 0 {
			l.cursor++
		}
	}
	if e := vectors(); e != nil {
		return e
	}
	return l.cleanup(s.manager.z)
}
