// Source: src/util/inf_eps_rational.h:257-266. Infinity-sign comparisons
// return before reading the finite rational or nested numeral operands.
package numeral

import (
	"errors"
	"testing"
)

func TestInfEpsRationalComparisonSkipsUnreadFiniteOperand(t *testing.T) {
	s, _ := serviceNew(t)
	for _, sign := range []int32{-1, 1} {
		a, e := s.InfEpsInfNewValue()
		serviceMust(t, e)
		a.infinity.Num = NewInt(sign)
		a.numeral.First.Num.Cell = s.Manager.Z.FailedRef
		var unread *OwnedRational
		checks := []struct {
			name string
			call func() (bool, error)
			want bool
		}{
			{"inf<rat", func() (bool, error) { return s.InfEpsInfLessRational(&a, unread) }, sign < 0},
			{"inf>rat", func() (bool, error) { return s.InfEpsInfGreaterRational(&a, unread) }, sign > 0},
			{"inf<=rat", func() (bool, error) { return s.InfEpsInfLessEqualRational(&a, unread) }, sign < 0},
			{"inf>=rat", func() (bool, error) { return s.InfEpsInfGreaterEqualRational(&a, unread) }, sign > 0},
			{"rat<inf", func() (bool, error) { return s.InfEpsInfRationalLess(unread, &a) }, sign > 0},
			{"rat>inf", func() (bool, error) { return s.InfEpsInfRationalGreater(unread, &a) }, sign < 0},
			{"rat<=inf", func() (bool, error) { return s.InfEpsInfRationalLessEqual(unread, &a) }, sign > 0},
			{"rat>=inf", func() (bool, error) { return s.InfEpsInfRationalGreaterEqual(unread, &a) }, sign < 0},
		}
		for _, check := range checks {
			got, e := check.call()
			if e != nil || got != check.want {
				t.Fatalf("sign %d %s: got %v, %v; want %v", sign, check.name, got, e, check.want)
			}
		}
		a.infinity.Num = NewInt(0)
		if _, e := s.InfEpsInfLessRational(&a, unread); !errors.Is(e, ErrRationalServiceOwner) {
			t.Fatal("finite branch must reach rational operand", e)
		}
		a.numeral.First.Num.Cell = 0
		serviceMust(t, s.InfEpsInfDel(&a))
	}
}
