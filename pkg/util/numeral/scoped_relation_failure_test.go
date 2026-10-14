package numeral

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestScopedNumeralRelationsDoNotPublishResultAfterOperandFailure(t *testing.T) {
	q, _ := rationalManager(t, false)
	i, e := NewScopedInt(q.Z)
	serviceMust(t, e)
	defer func() { serviceMust(t, i.Close()) }()
	r, e := NewScopedRational(q)
	serviceMust(t, e)
	defer func() { serviceMust(t, r.Close()) }()
	serviceMust(t, i.SetInt64(1))
	serviceMust(t, r.SetInt64(1))
	badInt := Int{Val: 1, Cell: q.Z.FailedRef}
	badRat := Rational{Num: badInt, Den: NewInt(1)}
	check := func(name string, got bool, e error) {
		t.Helper()
		if got || !errors.Is(e, arena.ErrInvalidSlot) {
			t.Fatalf("%s: got %t, %v", name, got, e)
		}
	}
	for name, fn := range map[string]func(*Int) (bool, error){
		"int equal":         i.Equal,
		"int not equal":     i.NotEqual,
		"int less":          i.Less,
		"int greater":       i.Greater,
		"int less equal":    i.LessEqual,
		"int greater equal": i.GreaterEqual,
	} {
		got, e := fn(&badInt)
		check(name, got, e)
	}
	for name, fn := range map[string]func(*Rational) (bool, error){
		"rational equal":         r.Equal,
		"rational not equal":     r.NotEqual,
		"rational less":          r.Less,
		"rational greater":       r.Greater,
		"rational less equal":    r.LessEqual,
		"rational greater equal": r.GreaterEqual,
	} {
		got, e := fn(&badRat)
		check(name, got, e)
	}
}
