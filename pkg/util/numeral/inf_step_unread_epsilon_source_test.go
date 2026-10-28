package numeral

// Source: src/util/inf_rational.h:228-238. Prefix steps use m_first only.

import "testing"

func TestInfPrefixStepsSkipUnreadEpsilon(t *testing.T) {
	s, _ := serviceNew(t)
	v, e := s.InfZero()
	serviceMust(t, e)
	v.First.Num = NewInt(5)
	v.Second.Num = NewInt(3)
	v.Second.Num.Cell = s.Manager.Z.FailedRef
	v.Second.Den.Cell = s.Manager.Z.FailedRef
	serviceMust(t, s.InfInc(&v))
	if v.First.Num.Val != 6 {
		t.Fatalf("incremented finite term = %d, want 6", v.First.Num.Val)
	}
	serviceMust(t, s.InfDec(&v))
	if v.First.Num.Val != 5 || v.Second.Num.Val != 3 {
		t.Fatalf("unexpected components after decrement: finite=%d epsilon=%d", v.First.Num.Val, v.Second.Num.Val)
	}
	v.Second.Num.Cell = 0
	v.Second.Den.Cell = 0
	serviceMust(t, s.InfDel(&v))
}
