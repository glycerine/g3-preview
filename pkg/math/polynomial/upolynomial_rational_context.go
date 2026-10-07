// Source: math/polynomial/sexpr2upolynomial.cpp; util scoped-numeral lifetimes.
package polynomial

import (
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func NewUPolynomialManagerFromRationalService(s *util.RationalService, c UPolynomialConfig) (out *UPolynomialManager, cause error) {
	cause = s.WithNumericBorrow(func(b *util.RationalNumericBorrow) error {
		if e := b.Ready(); e != nil {
			return e
		}
		if c.Integer != nil && c.Integer != b.Integers() {
			return ErrUPolynomialOwner
		}
		c.Integer = b.Integers()
		var e error
		out, e = NewUPolynomialManager(c)
		return e
	})
	return out, cause
}

// The polynomial context belongs to its actual caller, borrowing primitive
// service access only during the locked callback.
type UPolynomialRationalContext struct {
	self       *UPolynomialRationalContext
	borrow     *util.RationalNumericBorrow
	polynomial *UPolynomialManager
	active     bool
}

func numericRationalService_WithUPolynomialRationals(s *util.RationalService, m *UPolynomialManager, body func(*UPolynomialRationalContext) error) error {
	return s.WithNumericBorrow(func(b *util.RationalNumericBorrow) error {
		if m == nil || m.config.Integer != b.Integers() {
			return ErrUPolynomialOwner
		}
		if e := m.check(); e != nil {
			return e
		}
		if body == nil {
			return arena.ErrConfig
		}
		return b.WithLexical(func() error {
			c := &m.rationalContext
			if c.active {
				return ErrUPolynomialCleanupPending
			}
			*c = UPolynomialRationalContext{self: c, borrow: b, polynomial: m, active: true}
			defer func() { *c = UPolynomialRationalContext{} }()
			return body(c)
		})
	})
}
func (c *UPolynomialRationalContext) valid() bool {
	return c != nil && c.self == c && c.active && c.polynomial != nil && c == &c.polynomial.rationalContext && c.borrow.Valid()
}
func (c *UPolynomialRationalContext) Pending() bool { return c.valid() && c.borrow.Pending() }
func (c *UPolynomialRationalContext) StopVectors(e error) {
	if c.valid() {
		c.borrow.StopVectors(e)
	}
}
func (c *UPolynomialRationalContext) SetConstant(p UPolynomial, a *util.OwnedRational, nonInteger func() error) error {
	if !c.valid() {
		return util.ErrRationalServiceOwner
	}
	// WithCopiedScalar preserves source owner validation before the callback
	// precondition and the reached lexical constructor.
	if nonInteger == nil {
		return c.borrow.WithCopiedScalar(a, nil)
	}
	return c.borrow.WithCopiedScalar(a, func(value util.Rational) error {
		integer, e := c.borrow.IsInt(&value)
		if e != nil {
			return e
		}
		if !integer {
			return nonInteger()
		}
		return c.polynomial.SetRationals(p, []util.Rational{value})
	})
}
func (c *UPolynomialRationalContext) Exponent(a *util.OwnedRational) (uint32, bool, error) {
	if !c.valid() {
		return 0, false, util.ErrRationalServiceOwner
	}
	return c.borrow.Exponent(a)
}
func (c *UPolynomialRationalContext) SetIdentity(p UPolynomial) error {
	if !c.valid() {
		return util.ErrRationalServiceOwner
	}
	return c.polynomial.SetRationals(p, []util.Rational{util.NewRational(), util.RationalFromBorrowedNumerals(util.NewInt(1), util.NewInt(1))})
}
func numericRationalService_CleanupUPolynomialRationals(s *util.RationalService, vectors func() error) error {
	return s.WithNumericBorrow(func(b *util.RationalNumericBorrow) error { return b.CleanupAfterVectors(vectors) })
}
