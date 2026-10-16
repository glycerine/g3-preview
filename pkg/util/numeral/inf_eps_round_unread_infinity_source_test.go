// Source: src/util/inf_eps_rational.h:310-317. The finite-domain assertions
// are comments, and both roundings delegate directly to the nested numeral.
package numeral

import "testing"

func TestInfEpsRoundSkipsUnreadInfinityStorage(t *testing.T) {
	s, _ := serviceNew(t)
	v, e := s.InfEpsInfNewValue()
	serviceMust(t, e)
	v.infinity.Num.Cell = s.Manager.Z.FailedRef
	v.infinity.Den.Cell = s.Manager.Z.FailedRef
	v.numeral.First.Num = NewInt(5)
	v.numeral.Second.Num = NewInt(-1)
	floor, e := s.InfEpsInfFloor(&v)
	serviceMust(t, e)
	ceil, e := s.InfEpsInfCeil(&v)
	serviceMust(t, e)
	serviceText(t, s, &floor, "4")
	serviceText(t, s, &ceil, "5")
	serviceMust(t, s.Del(&floor))
	serviceMust(t, s.Del(&ceil))
	v.infinity.Num.Cell = 0
	v.infinity.Den.Cell = 0
	serviceMust(t, s.InfEpsInfDel(&v))
}
