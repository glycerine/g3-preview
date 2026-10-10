package numeral

import (
	"errors"
	"testing"
)

func TestRationalNumericBorrowCopiesLexicalScalarAndExpires(t *testing.T) {
	s, _ := serviceNew(t)
	a, e := s.NewValue()
	serviceMust(t, e)
	serviceMust(t, s.SetInt64(&a, 17))
	var retained *RationalNumericBorrow
	serviceMust(t, s.WithNumericBorrow(func(b *RationalNumericBorrow) error {
		retained = b
		copy := *b
		if copy.Valid() {
			t.Fatal("copied borrower accepted")
		}
		if !b.Valid() || b.Integers() != s.Manager.Z {
			t.Fatal("missing actual utility owner")
		}
		return b.WithLexical(func() error {
			return b.WithCopiedScalar(&a, func(value Rational) error {
				got, e := s.Manager.Z.Int64(value.BorrowNumerator())
				if e != nil || got != 17 {
					t.Fatal("lexical scalar copy", got, e)
				}
				if value.Num.Cell != 0 || value.Den != NewInt(1) {
					t.Fatal("source scalar shape changed")
				}
				return nil
			})
		})
	}))
	if retained.Valid() {
		t.Fatal("borrow survived callback")
	}
	if _, _, e := retained.Exponent(&a); !errors.Is(e, ErrRationalServiceOwner) {
		t.Fatal("expired access accepted", e)
	}
	got, e := s.String(&a)
	serviceMust(t, e)
	if got != "17" {
		t.Fatal("const input consumed", got)
	}
	serviceMust(t, s.Del(&a))
}
