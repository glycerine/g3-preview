package numeral

// Source: src/util/inf_rational.h:313-333. Noninteger FIRST skips epsilon;
// integer FIRST needs only epsilon's sign.

import "testing"

func TestInfRoundSkipsUnreadEpsilonStorage(t *testing.T) {
	for _, tc := range []struct {
		name, floor, ceil           string
		firstNum, firstDen, epsilon int32
		badEpsilonNum               bool
	}{
		{"fractional-first", "3", "4", 7, 2, 1, true},
		{"integer-first", "4", "5", 5, 1, -1, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _ := serviceNew(t)
			v, e := s.InfZero()
			serviceMust(t, e)
			v.First.Num = NewInt(tc.firstNum)
			v.First.Den = NewInt(tc.firstDen)
			v.Second.Num = NewInt(tc.epsilon)
			if tc.badEpsilonNum {
				v.Second.Num.Cell = s.Manager.Z.FailedRef
			}
			v.Second.Den.Cell = s.Manager.Z.FailedRef
			floor, e := s.InfFloor(&v)
			serviceMust(t, e)
			ceil, e := s.InfCeil(&v)
			serviceMust(t, e)
			serviceText(t, s, &floor, tc.floor)
			serviceText(t, s, &ceil, tc.ceil)
			serviceMust(t, s.Del(&floor))
			serviceMust(t, s.Del(&ceil))
			v.Second.Num.Cell = 0
			v.Second.Den.Cell = 0
			serviceMust(t, s.InfDel(&v))
		})
	}
}
