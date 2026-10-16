// Source: src/util/inf_int_rational.cpp:25-34. For nonzero epsilon the
// opening parenthesis is written before FIRST is formatted.
package numeral

import "testing"

func TestInfIntStringWritesOpeningPrefixBeforeFirstFailure(t *testing.T) {
	s, _ := serviceNew(t)
	v, e := s.InfIntNewValue()
	serviceMust(t, e)
	v.First.Num = NewInt(5)
	v.First.Num.Cell = s.Manager.Z.FailedRef
	v.second = 3
	if _, e = s.InfIntString(&v); e == nil {
		t.Fatal("expected invalid FIRST storage to fail")
	}
	if text := s.Manager.Z.Natural.text; len(text) == 0 || text[0] != '(' {
		t.Fatalf("missing source prefix before FIRST failure: %q", text)
	}
	v.First.Num.Cell = 0
	serviceMust(t, s.InfIntDel(&v))
}
