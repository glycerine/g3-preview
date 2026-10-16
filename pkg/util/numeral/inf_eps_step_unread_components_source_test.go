// Source: src/util/inf_eps_rational.h:225-237 and inf_rational.h:228-238.
// Prefix increment and decrement touch only the nested finite rational.
package numeral

import "testing"

func TestInfEpsPrefixStepsSkipUnreadInfinityAndEpsilon(t *testing.T) {
	s, _ := serviceNew(t)
	v, e := s.InfEpsInfNewValue()
	serviceMust(t, e)
	v.infinity.Num = NewInt(2)
	v.numeral.First.Num = NewInt(5)
	v.numeral.Second.Num = NewInt(3)
	v.infinity.Num.Cell = s.Manager.Z.FailedRef
	v.infinity.Den.Cell = s.Manager.Z.FailedRef
	v.numeral.Second.Num.Cell = s.Manager.Z.FailedRef
	v.numeral.Second.Den.Cell = s.Manager.Z.FailedRef
	serviceMust(t, s.InfEpsInfInc(&v))
	if v.numeral.First.Num.Val != 6 {
		t.Fatalf("incremented finite term = %d, want 6", v.numeral.First.Num.Val)
	}
	serviceMust(t, s.InfEpsInfDec(&v))
	if v.numeral.First.Num.Val != 5 || v.infinity.Num.Val != 2 || v.numeral.Second.Num.Val != 3 {
		t.Fatalf("unexpected components after decrement: infinity=%d finite=%d epsilon=%d", v.infinity.Num.Val, v.numeral.First.Num.Val, v.numeral.Second.Num.Val)
	}
	v.infinity.Num.Cell = 0
	v.infinity.Den.Cell = 0
	v.numeral.Second.Num.Cell = 0
	v.numeral.Second.Den.Cell = 0
	serviceMust(t, s.InfEpsInfDel(&v))
}
