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
	var nested InfRationalBorrow
	e = s.WithNumericBorrow(func(b *RationalNumericBorrow) error {
		escaped = b
		nested, e = b.BorrowInfEpsNumeral(&v)
		if e != nil {
			return e
		}
		if nested.value != &v.numeral {
			t.Fatal("source borrowed nested numeral identity")
		}
		nestedFirst, e := nested.First()
		if e != nil {
			return e
		}
		nestedSecond, e := nested.Infinitesimal()
		if e != nil {
			return e
		}
		if nestedFirst != v.numeral.First || nestedSecond != v.numeral.Second {
			t.Fatal("source nested numeral components", nestedFirst, nestedSecond)
		}
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
	if _, e := nested.First(); !errors.Is(e, ErrRationalServiceOwner) {
		t.Fatal("nested first escaped scope", e)
	}
	if _, e := nested.Infinitesimal(); !errors.Is(e, ErrRationalServiceOwner) {
		t.Fatal("nested infinitesimal escaped scope", e)
	}
	if _, e := escaped.BorrowInfEpsNumeral(&v); !errors.Is(e, ErrRationalServiceOwner) {
		t.Fatal("nested numeral borrow escaped scope", e)
	}
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
