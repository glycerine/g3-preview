// Source: src/util/inf_rational.h:346-357. Fused accumulation mutates FIRST
// before EPS, so a late EPS failure retains the FIRST update.
package numeral

import "testing"

func TestInfFusedKeepsFirstOnLateEpsilonFailure(t *testing.T) {
	for _, tc := range []struct {
		name string
		sub  bool
		want int32
	}{
		{"addmul", false, 10},
		{"submul", true, -4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _ := serviceNew(t)
			v, e := s.InfZero()
			serviceMust(t, e)
			k, e := s.InfZero()
			serviceMust(t, e)
			c, e := s.NewValue()
			serviceMust(t, e)
			v.First.Num = NewInt(3)
			v.Second.Num = NewInt(4)
			k.First.Num = NewInt(7)
			k.Second.Num = NewInt(11)
			k.Second.Num.Cell = s.Manager.Z.FailedRef
			c.Raw.Num = NewInt(1)
			if tc.sub {
				e = s.InfSubMulAssign(&v, &c, &k)
			} else {
				e = s.InfAddMulAssign(&v, &c, &k)
			}
			if e == nil {
				t.Fatal("expected invalid epsilon storage to fail")
			}
			if v.First.Num.Val != tc.want || v.Second.Num.Val != 4 {
				t.Fatalf("source prefix lost: first=%d epsilon=%d", v.First.Num.Val, v.Second.Num.Val)
			}
			k.Second.Num.Cell = 0
			serviceMust(t, s.Del(&c))
			serviceMust(t, s.InfDel(&k))
			serviceMust(t, s.InfDel(&v))
		})
	}
}
