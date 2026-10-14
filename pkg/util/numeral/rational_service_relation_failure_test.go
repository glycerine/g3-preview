package numeral

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestRationalServiceRelationsStopAtFailedComponentRead(t *testing.T) {
	s, _ := serviceNew(t)
	a := serviceValue(t, s, "1")
	b := serviceValue(t, s, "2")
	a.Raw.Num.Cell = s.Manager.Z.FailedRef
	check := func(name string, got bool, e error) {
		t.Helper()
		if got || !errors.Is(e, arena.ErrInvalidSlot) {
			t.Fatalf("%s: got %t, %v", name, got, e)
		}
	}
	for name, fn := range map[string]func(*OwnedRational, *OwnedRational) (bool, error){
		"not equal":     s.NotEqual,
		"less":          s.Less,
		"greater":       s.Greater,
		"less equal":    s.LessEqual,
		"greater equal": s.GreaterEqual,
	} {
		got, e := fn(&a, &b)
		check(name, got, e)
	}
	for name, fn := range map[string]func(*OwnedRational, int32) (bool, error){
		"not equal int":     s.NotEqualInt,
		"less int":          s.LessInt,
		"greater int":       s.GreaterInt,
		"less equal int":    s.LessEqualInt,
		"greater equal int": s.GreaterEqualInt,
	} {
		got, e := fn(&a, 2)
		check(name, got, e)
	}
	a.Raw.Num.Cell = 0
	serviceMust(t, s.Del(&a))
	serviceMust(t, s.Del(&b))
}
