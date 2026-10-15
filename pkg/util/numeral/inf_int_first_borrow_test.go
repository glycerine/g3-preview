package numeral

import (
	"errors"
	"testing"
)

func TestInfIntFirstSourceBorrowHasNoCopyOrDenominatorRead(t *testing.T) {
	s, budget := serviceNew(t)
	v, e := s.InfIntNewValue()
	serviceMust(t, e)
	v.First = Rational{Num: NewInt(2), Den: NewInt(0)}
	baseline := budget.Used()
	var escaped *RationalNumericBorrow
	e = s.WithNumericBorrow(func(b *RationalNumericBorrow) error {
		escaped = b
		first, e := b.BorrowInfIntFirst(&v)
		if e != nil {
			return e
		}
		if first.Num != v.First.Num || first.Den != v.First.Den {
			t.Fatal("source borrowed first component", first, v.First)
		}
		if budget.Used() != baseline {
			t.Fatal("borrow allocated a copy", budget.Used(), baseline)
		}
		return nil
	})
	serviceMust(t, e)
	if _, e := escaped.BorrowInfIntFirst(&v); !errors.Is(e, ErrRationalServiceOwner) {
		t.Fatal("borrow used after scope", e)
	}
	serviceMust(t, s.InfIntDel(&v))
}
