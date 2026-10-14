package numeral

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestBinaryRelationsDoNotInvertFailedComponentRead(t *testing.T) {
	m, _ := binaryManager(t)
	a, b := NewBinaryIntPower(1, 0), NewBinaryIntPower(2, 0)
	i := NewInt(2)
	q := NewRational()
	q.Num = NewInt(2)
	a.Num.Cell = m.Z.FailedRef
	check := func(name string, got bool, e error) {
		t.Helper()
		if got || !errors.Is(e, arena.ErrInvalidSlot) {
			t.Fatalf("%s: got %t, %v", name, got, e)
		}
	}
	for name, fn := range map[string]func(*BinaryRational, *BinaryRational) (bool, error){
		"not equal":     m.NotEqual,
		"less equal":    m.LessEqual,
		"greater equal": m.GreaterEqual,
	} {
		got, e := fn(&a, &b)
		check(name, got, e)
	}
	for name, fn := range map[string]func(*BinaryRational, *Int) (bool, error){
		"integer not equal":     m.NotEqualInteger,
		"integer greater":       m.GreaterInteger,
		"integer greater equal": m.GreaterEqualInteger,
	} {
		got, e := fn(&a, &i)
		check(name, got, e)
	}
	for name, fn := range map[string]func(*BinaryRational, *RationalManager, *Rational) (bool, error){
		"rational not equal":     m.NotEqualRational,
		"rational greater":       m.GreaterRational,
		"rational greater equal": m.GreaterEqualRational,
	} {
		got, e := fn(&a, m.Diagnostics.Manager, &q)
		check(name, got, e)
	}
	got, e := m.IsNonzero(&a)
	check("nonzero", got, e)
	a.Num.Cell = 0
	serviceMust(t, m.Del(&a))
	serviceMust(t, m.Del(&b))
}
