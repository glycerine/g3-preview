package numeral

import (
	"testing"
)

func TestRationalRootDenominatorErrorKeepsPublishedNumerator(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertion only")
	}
	_, _ = bindMPNDebugActions(t, "TTTT")
	for _, synchronized := range []bool{false, true} {
		for _, alias := range []bool{false, true} {
			m, _ := rationalManager(t, synchronized)
			a, out := NewRational(), NewRational()
			serviceMust(t, m.Z.SetInt32(&a.Num, 9))
			serviceMust(t, m.Z.SetInt32(&a.Den, -4))
			serviceMust(t, m.SetFraction(&out, 7, 13))
			dst := &out
			if alias {
				dst = &a
			}
			exact, e := m.Root(&a, 2, dst)
			if exact || !isMPNVerificationThrow(e) {
				t.Fatalf("denominator domain error: sync=%v alias=%v exact=%v error=%v", synchronized, alias, exact, e)
			}
			if got := integerString(t, m.Z, &dst.Num); got != "3" {
				t.Fatalf("numerator publication: sync=%v alias=%v value=%s", synchronized, alias, got)
			}
			if got := integerString(t, m.Z, &dst.Den); got != "-4" {
				t.Fatalf("denominator source copy: sync=%v alias=%v value=%s", synchronized, alias, got)
			}
			if !alias && integerString(t, m.Z, &a.Num) != "9" {
				t.Fatal("distinct source numerator changed")
			}
			serviceMust(t, m.Del(&out))
			serviceMust(t, m.Del(&a))
			m.Close()
		}
	}
}
