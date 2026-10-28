package numeral

// Source: src/util/inf_rational.h:51-66. A zero epsilon returns the finite
// numeral's string without inspecting the epsilon denominator.

import "testing"

func TestInfStringZeroEpsilonSkipsUnreadDenominator(t *testing.T) {
	s, _ := serviceNew(t)
	v, e := s.InfFromInt32(5)
	serviceMust(t, e)
	v.Second.Den.Cell = s.Manager.Z.FailedRef
	got, e := s.InfString(&v)
	serviceMust(t, e)
	if got != "5" {
		t.Fatalf("zero-epsilon string = %q, want 5", got)
	}
	v.Second.Den.Cell = 0
	serviceMust(t, s.InfDel(&v))
}
