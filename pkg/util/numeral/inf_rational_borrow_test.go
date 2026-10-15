package numeral

import (
	"errors"
	"testing"
)

func TestInfRationalSourceBorrowedComponentsDoNotCopyOrReadPeer(t *testing.T) {
	s, budget := serviceNew(t)
	v, e := s.NewInfValue()
	serviceMust(t, e)
	v.First = Rational{Num: NewInt(3), Den: NewInt(0)}
	v.Second = Rational{Num: NewInt(-4), Den: NewInt(0)}
	baseline := budget.Used()
	var escaped *RationalNumericBorrow
	e = s.WithNumericBorrow(func(b *RationalNumericBorrow) error {
		escaped = b
		first, e := b.BorrowInfFirst(&v)
		if e != nil {
			return e
		}
		second, e := b.BorrowInfInfinitesimal(&v)
		if e != nil {
			return e
		}
		if first != v.First || second != v.Second {
			t.Fatal("source component references", first, second, v)
		}
		if budget.Used() != baseline {
			t.Fatal("borrow allocated component copies", budget.Used(), baseline)
		}
		return nil
	})
	serviceMust(t, e)
	if _, e := escaped.BorrowInfFirst(&v); !errors.Is(e, ErrRationalServiceOwner) {
		t.Fatal("first borrow escaped scope", e)
	}
	if _, e := escaped.BorrowInfInfinitesimal(&v); !errors.Is(e, ErrRationalServiceOwner) {
		t.Fatal("infinitesimal borrow escaped scope", e)
	}
	serviceMust(t, s.InfDel(&v))
}
