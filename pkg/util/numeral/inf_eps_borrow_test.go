package numeral

import (
	"errors"
	"testing"
)

func TestInfEpsSourceBorrowedRationalComponentsSkipPeerReads(t *testing.T) {
	s, budget := serviceNew(t)
	v, e := s.InfEpsInfNewValue()
	serviceMust(t, e)
	v.infinity = Rational{Num: NewInt(7), Den: NewInt(0)}
	v.numeral.First = Rational{Num: NewInt(3), Den: NewInt(0)}
	v.numeral.Second = Rational{Num: NewInt(-4), Den: NewInt(0)}
	baseline := budget.Used()
	var escaped *RationalNumericBorrow
	e = s.WithNumericBorrow(func(b *RationalNumericBorrow) error {
		escaped = b
		first, e := b.BorrowInfEpsRational(&v)
		if e != nil {
			return e
		}
		second, e := b.BorrowInfEpsInfinitesimal(&v)
		if e != nil {
			return e
		}
		infinity, e := b.BorrowInfEpsInfinity(&v)
		if e != nil {
			return e
		}
		if first != v.numeral.First || second != v.numeral.Second || infinity != v.infinity {
			t.Fatal("source nested component borrows", first, second, infinity, v)
		}
		if budget.Used() != baseline {
			t.Fatal("borrow allocated component copies", budget.Used(), baseline)
		}
		return nil
	})
	serviceMust(t, e)
	for _, call := range []func() (Rational, error){
		func() (Rational, error) { return escaped.BorrowInfEpsRational(&v) },
		func() (Rational, error) { return escaped.BorrowInfEpsInfinitesimal(&v) },
		func() (Rational, error) { return escaped.BorrowInfEpsInfinity(&v) },
	} {
		if _, e := call(); !errors.Is(e, ErrRationalServiceOwner) {
			t.Fatal("nested borrow escaped scope", e)
		}
	}
	serviceMust(t, s.InfEpsInfDel(&v))
}
