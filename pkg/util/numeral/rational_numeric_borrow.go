// Source: src/util/rational.cpp; src/util/scoped_numeral.h.
package numeral

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
)

// RationalNumericBorrow is valid only inside WithNumericBorrow's synchronous
// callback. The service lock and all lexical numeric ownership remain in util.
type RationalNumericBorrow struct {
	self    *RationalNumericBorrow
	service *RationalService
	active  bool
}

func (b *RationalNumericBorrow) Valid() bool {
	return b != nil && b.self == b && b.active && b.service != nil && b == &b.service.numericBorrow
}
func (s *RationalService) WithNumericBorrow(body func(*RationalNumericBorrow) error) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if body == nil {
		return arena.ErrConfig
	}
	b := &s.numericBorrow
	if b.active {
		return ErrRationalLexicalPending
	}
	*b = RationalNumericBorrow{self: b, service: s, active: true}
	defer func() { *b = RationalNumericBorrow{} }()
	return body(b)
}
func (b *RationalNumericBorrow) Ready() error {
	if !b.Valid() {
		return ErrRationalServiceOwner
	}
	return b.service.lexicalReady()
}

// BorrowInfIntFirst is the source inf_int_rational::get_first/get_rational
// const-reference read. The returned scalar borrows its owner's integer cells
// and is valid only during this numeric-borrow callback.
func (b *RationalNumericBorrow) BorrowInfIntFirst(v *OwnedInfIntRational) (Rational, error) {
	if !b.Valid() {
		return Rational{}, ErrRationalServiceOwner
	}
	if e := b.service.infIntScalarOwner(v); e != nil {
		return Rational{}, e
	}
	return v.First, nil
}
func (b *RationalNumericBorrow) Integers() *IntegerManager {
	if !b.Valid() {
		return nil
	}
	return b.service.Manager.Z
}
func (b *RationalNumericBorrow) WithLexical(body func() error) (cause error) {
	if !b.Valid() {
		return ErrRationalServiceOwner
	}
	if body == nil {
		return arena.ErrConfig
	}
	s := b.service
	if e := s.beginLexical(); e != nil {
		return e
	}
	defer func() {
		s.endLexicalError(&cause)
		if s.Lexical.Pending() && s.Lexical.Cause != nil && !errors.Is(cause, s.Lexical.Cause) {
			cause = errors.Join(cause, s.Lexical.Cause)
		}
	}()
	return body()
}
func (b *RationalNumericBorrow) Pending() bool {
	return b.Valid() && b.service.Lexical.Cause != nil && b.service.Lexical.Pending()
}
func (b *RationalNumericBorrow) StopVectors(e error) {
	if b.Valid() && e != nil && b.service.Lexical.Pending() && b.service.Lexical.Cause == nil {
		b.service.Lexical.Cause = e
	}
}
func (b *RationalNumericBorrow) WithCopiedScalar(a *OwnedRational, body func(Rational) error) (cause error) {
	if !b.Valid() {
		return ErrRationalServiceOwner
	}
	s := b.service
	if e := s.rationalScalarOwner(a); e != nil {
		return e
	}
	if body == nil {
		return arena.ErrConfig
	}
	if e := s.beginLexical(); e != nil {
		return e
	}
	defer s.endLexicalError(&cause)
	value := NewRational()
	complete := false
	defer s.endLexicalLocal(&value, &complete)
	if e := s.Manager.Copy(&value, &a.Raw); e != nil {
		return e
	}
	complete = true
	return body(value)
}
func (b *RationalNumericBorrow) IsInt(a *Rational) (bool, error) {
	if !b.Valid() {
		return false, ErrRationalServiceOwner
	}
	return b.service.Manager.IsInt(a)
}
func (b *RationalNumericBorrow) Exponent(a *OwnedRational) (uint32, bool, error) {
	if !b.Valid() {
		return 0, false, ErrRationalServiceOwner
	}
	s := b.service
	if e := s.rationalScalarOwner(a); e != nil {
		return 0, false, e
	}
	ok, e := s.Manager.IsUnsigned(&a.Raw)
	if e != nil || !ok {
		return 0, ok, e
	}
	n, e := s.Manager.Unsigned(&a.Raw)
	return n, true, e
}

// CleanupAfterVectors finishes completed lexical scalars, then the caller's
// older vectors, and only then aborts partial scalar constructors.
func (b *RationalNumericBorrow) CleanupAfterVectors(vectors func() error) error {
	if !b.Valid() {
		return ErrRationalServiceOwner
	}
	if vectors == nil {
		return arena.ErrConfig
	}
	s := b.service
	l := &s.Lexical
	for i := uint8(0); i < l.Count; i++ {
		if l.Done&(1<<i) != 0 || l.abort[i] {
			continue
		}
		if e := lexicalDelete(s.Manager.Z, &l.Roots[i], true, &l.components[i]); e != nil {
			l.Cause = e
			return e
		}
		l.Done |= 1 << i
		for l.cursor < l.Count && l.Done&(1<<l.cursor) != 0 {
			l.cursor++
		}
	}
	if e := vectors(); e != nil {
		return e
	}
	return l.cleanup(s.Manager.Z)
}
func (b *RationalNumericBorrow) Rationals() *RationalManager {
	if !b.Valid() {
		return nil
	}
	return b.service.Manager
}
