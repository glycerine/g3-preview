package numeral

import (
	"errors"
	"math"
	"testing"
)

// These raw scalar shapes are actual initialized numeral fields (no storage
// pointers). Source setter/inverse/error prefixes can publish DEN0; ordinary
// readers remain strict. No new registered-source ENSURE credit is claimed.
func infApproxRaw(s *RationalService, n, d, e, ed int32) OwnedInfRational {
	v := s.infEmpty()
	v.First = Rational{Num: NewInt(n), Den: NewInt(d)}
	v.Second = Rational{Num: NewInt(e), Den: NewInt(ed)}
	return v
}
func infApproxRawEqual(t *testing.T, v *OwnedInfRational, n, d, e, ed int32) {
	t.Helper()
	if v.First.Num != NewInt(n) || v.First.Den != NewInt(d) || v.Second.Num != NewInt(e) || v.Second.Den != NewInt(ed) {
		t.Fatalf("raw result %#v want %d/%d,%d/%d", v, n, d, e, ed)
	}
}
func TestInfApproxPublicRationalRawDependencies(t *testing.T) {
	for _, synch := range []bool{false, true} {
		m, _ := rationalManager(t, synch)
		a := Rational{Num: NewInt(1), Den: NewInt(0)}
		b := NewRational()
		serviceMust(t, m.SetInt64(&b, 1))
		out := NewRational()
		for _, sub := range []bool{false, true} {
			var e error
			if sub {
				e = m.Sub(&a, &b, &out)
			} else {
				e = m.Add(&a, &b, &out)
			}
			serviceMust(t, e)
			if out.Num != NewInt(1) || out.Den != NewInt(0) {
				t.Fatal(out)
			}
			qopsAuditLiveOwnership(t, m, []*Int{&a.Num, &a.Den, &b.Num, &b.Den, &out.Num, &out.Den})
		}
		serviceMust(t, m.Power(&a, 0, &out))
		if out.Num != NewInt(1) || out.Den != NewInt(1) {
			t.Fatal(out)
		}
		serviceMust(t, m.Power(&a, 3, &out))
		if out.Num != NewInt(1) || out.Den != NewInt(0) {
			t.Fatal(out)
		}
		// Raw infinity plus infinity reaches 0/0 normalization rather than a read guard.
		if e := m.Add(&a, &a, &out); !errors.Is(e, ErrDivideByZero) {
			t.Fatal(e)
		}
		if out.Num.Val != 0 || out.Den.Val != 0 {
			t.Fatal(out)
		}
		serviceMust(t, m.Del(&out))
		serviceMust(t, m.Del(&a))
		serviceMust(t, m.Del(&b))
	}
}
func TestInfApproxSignedValueConstructionAndRawAssignments(t *testing.T) {
	if SourceExtendedGCDDebug {
		bindMPNDebugActions(t, "TTTTTT")
	}
	for _, n := range []int32{-1, 0, 1} {
		s, _ := serviceNew(t)
		v := infApproxRaw(s, 7, 1, 3, 1)
		e := s.InfSetFractionSigned(&v, n, 0)
		if SourceExtendedGCDDebug {
			if !isMPNVerificationThrow(e) {
				t.Fatal(e)
			}
			infApproxRawEqual(t, &v, 7, 1, 3, 1)
		} else if n == 0 {
			if !errors.Is(e, ErrDivideByZero) {
				t.Fatal(e)
			}
			infApproxRawEqual(t, &v, 7, 1, 3, 1)
		} else {
			serviceMust(t, e)
			infApproxRawEqual(t, &v, n, 0, 0, 1)
		}
		iv, err := s.InfIntFromFraction(n, 0)
		if SourceExtendedGCDDebug {
			if !isMPNVerificationThrow(err) {
				t.Fatal(err)
			}
		} else if n == 0 {
			if !errors.Is(err, ErrDivideByZero) {
				t.Fatal(err)
			}
		} else {
			serviceMust(t, err)
			if iv.First.Num != NewInt(n) || iv.First.Den.Val != 0 || iv.second != 0 {
				t.Fatal(iv)
			}
			serviceMust(t, s.InfIntDel(&iv))
		}
		infAuditRoots(t, s, []*OwnedInfRational{&v}, nil)
		serviceMust(t, s.InfDel(&v))
	}
	s, _ := serviceNew(t)
	a, e := s.NewValue()
	serviceMust(t, e)
	a.Raw = Rational{Num: NewInt(1), Den: NewInt(0)}
	v := infApproxRaw(s, 7, 0, 2, 0)
	serviceMust(t, s.InfSetRational(&v, &a))
	infApproxRawEqual(t, &v, 1, 0, 0, 1)
	serviceMust(t, s.InfSetPair(&v, &a, &a))
	infApproxRawEqual(t, &v, 1, 0, 1, 0)
	serviceMust(t, s.InfSetOffset(&v, &a, false))
	infApproxRawEqual(t, &v, 1, 0, -1, 1)
	iv, e := s.InfIntFromRational(&a, 3)
	serviceMust(t, e)
	if iv.First.Den.Val != 0 || iv.second != 3 {
		t.Fatal(iv)
	}
	serviceMust(t, s.InfIntSetParts(&iv, &a, -2))
	if iv.second != -2 || iv.First.Den.Val != 0 {
		t.Fatal(iv)
	}
	ev, e := s.InfEpsInfNewValue()
	serviceMust(t, e)
	serviceMust(t, s.InfEpsInfSetPair(&ev, &a, &v))
	if ev.infinity.Den.Val != 0 || ev.numeral.First.Den.Val != 0 {
		t.Fatal(ev)
	}
	serviceMust(t, s.InfEpsInfSetNumeral(&ev, &v))
	if ev.infinity.Num.Val != 0 || ev.infinity.Den.Val != 1 {
		t.Fatal(ev)
	}
	serviceMust(t, s.InfEpsInfSetInt32(&ev, math.MinInt32))
	if ev.numeral.First.Num != NewInt(math.MinInt32) || ev.numeral.Second.Num.Val != 0 {
		t.Fatal(ev)
	}
	serviceMust(t, s.InfEpsInfDel(&ev))
	serviceMust(t, s.InfIntDel(&iv))
	serviceMust(t, s.InfDel(&v))
	serviceMust(t, s.Del(&a))
	infAuditRoots(t, s, nil, nil)
}
