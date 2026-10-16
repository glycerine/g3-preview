// Source: src/util/inf_eps_rational.h:47-70, src/util/inf_rational.h:51-66.
package numeral

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestInfEpsStringSkipsUnreadNestedDenominators(t *testing.T) {
	s, _ := serviceNew(t)
	v, e := s.InfEpsInfNewValue()
	serviceMust(t, e)
	bad := s.Manager.Z.FailedRef

	v.infinity.Num = NewInt(1)
	v.numeral.First.Den.Cell = bad
	v.numeral.Second.Den.Cell = bad
	if got, e := s.InfEpsInfString(&v); e != nil || got != "oo" {
		t.Fatal("zero nested numeral must skip both denominators", got, e)
	}
	v.numeral.First.Den.Cell = 0
	v.infinity.Num = NewInt(0)
	if got, e := s.InfEpsInfString(&v); e != nil || got != "0" {
		t.Fatal("zero epsilon must skip its denominator", got, e)
	}
	v.infinity.Num = NewInt(2)
	v.numeral.First.Num = NewInt(3)
	if got, e := s.InfEpsInfString(&v); e != nil || got != "(2*oo + 3)" {
		t.Fatal("nonzero first must still skip zero epsilon denominator", got, e)
	}
	v.numeral.First.Den.Cell = bad
	if _, e := s.InfEpsInfString(&v); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("nonzero first display must reach its denominator", e)
	}
	v.numeral.First.Den.Cell = 0
	v.numeral.Second.Den.Cell = 0
	serviceMust(t, s.InfEpsInfDel(&v))
}
