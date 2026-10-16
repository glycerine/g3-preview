// Source: src/util/inf_eps_rational.h:322-333. The outer infinity term is
// updated before the nested finite and epsilon terms.
package numeral

import "testing"

func TestInfEpsFusedKeepsEarlierTermsOnLateFailure(t *testing.T) {
	for _, tc := range []struct {
		name         string
		sub          bool
		wantInfinity int32
		wantFirst    int32
	}{
		{"addmul", false, 7, 10},
		{"submul", true, -3, -4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _ := serviceNew(t)
			v, e := s.InfEpsInfNewValue()
			serviceMust(t, e)
			k, e := s.InfEpsInfNewValue()
			serviceMust(t, e)
			c, e := s.NewValue()
			serviceMust(t, e)
			v.infinity.Num = NewInt(2)
			v.numeral.First.Num = NewInt(3)
			v.numeral.Second.Num = NewInt(4)
			k.infinity.Num = NewInt(5)
			k.numeral.First.Num = NewInt(7)
			k.numeral.Second.Num = NewInt(11)
			k.numeral.Second.Num.Cell = s.Manager.Z.FailedRef
			c.Raw.Num = NewInt(1)
			if tc.sub {
				e = s.InfEpsInfSubMulAssign(&v, &c, &k)
			} else {
				e = s.InfEpsInfAddMulAssign(&v, &c, &k)
			}
			if e == nil {
				t.Fatal("expected invalid epsilon storage to fail")
			}
			if v.infinity.Num.Val != tc.wantInfinity || v.numeral.First.Num.Val != tc.wantFirst || v.numeral.Second.Num.Val != 4 {
				t.Fatalf("source prefix lost: infinity=%d first=%d epsilon=%d", v.infinity.Num.Val, v.numeral.First.Num.Val, v.numeral.Second.Num.Val)
			}
			k.numeral.Second.Num.Cell = 0
			serviceMust(t, s.Del(&c))
			serviceMust(t, s.InfEpsInfDel(&k))
			serviceMust(t, s.InfEpsInfDel(&v))
		})
	}
}
