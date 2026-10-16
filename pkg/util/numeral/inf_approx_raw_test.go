package numeral

import (
	"errors"
	"math"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
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
	for _, n := range []int32{-1, 0, 1} {
		s, _ := serviceNew(t)
		v := infApproxRaw(s, 7, 1, 3, 1)
		e := s.InfSetFractionSigned(&v, n, 0)
		if SourceExtendedGCDDebug {
			if !errors.Is(e, ErrIntegerAssertion) {
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
			if !errors.Is(err, ErrIntegerAssertion) {
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
func TestInfApproxFirstCopyFailurePrecedesUnreadEPS(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	cfg := serviceConfig(b)
	cfg.Integers.Profile = umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	s, e := NewRationalService(cfg)
	serviceMust(t, e)
	defer func() {
		b.FailAfter(-1)
		serviceMust(t, s.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	}()
	a := infApproxRaw(s, 1, 1, 1, 1)
	other := infApproxRaw(s, 2, 1, 1, 1)
	serviceMust(t, s.Manager.SetString(&a.First, "123456789012345678901234567890123456789"))
	other.Second.Den.Cell = s.Manager.Z.FailedRef
	b.FailAfter(0)
	out, e := s.InfLowerMult(&a, &other)
	if !numericPhysicalReservationRefusal(e) || out.ownerID != 0 {
		t.Fatalf("FIRST Copy must precede EPS read: %v %#v", e, out)
	}
	infAuditRoots(t, s, []*OwnedInfRational{&a, &other}, nil)
	b.FailAfter(-1)
	serviceMust(t, s.Manager.Z.CleanupPendingAllocation())
	serviceMust(t, s.InfDel(&a))
	serviceMust(t, s.InfDel(&other))
}

func TestInfApproxRawProductSignedDenominators(t *testing.T) {
	s, _ := serviceNew(t)
	// Raw signed denominators remain finite; correction predicates use the source
	// numerator signs, independently of the mathematical fraction signs.
	for a := int32(-2); a <= 2; a++ {
		for b := int32(-2); b <= 2; b++ {
			for x := int32(-1); x <= 1; x++ {
				for y := int32(-1); y <= 1; y++ {
					for _, den := range []int32{-1, 1} {
						for _, upper := range []bool{false, true} {
							v := infApproxRaw(s, a, den, x, den)
							w := infApproxRaw(s, b, 1, y, 1)
							var out OwnedInfRational
							var e error
							if upper {
								out, e = s.InfUpperMult(&v, &w)
							} else {
								out, e = s.InfLowerMult(&v, &w)
							}
							serviceMust(t, e)
							first := new(big.Rat).SetFrac(big.NewInt(int64(a*b)), big.NewInt(int64(den)))
							second := new(big.Rat).SetFrac(big.NewInt(int64(a*y+x*b)), big.NewInt(int64(den)))
							if upper && (x > 0 && y > 0 || x < 0 && y < 0) {
								second.Add(second, big.NewRat(1, 1))
							}
							if !upper && (x > 0 && y < 0 || x < 0 && y > 0) {
								second.Sub(second, big.NewRat(1, 1))
							}
							gotF := new(big.Rat).SetFrac(roundPublicationDecode(t, s.Manager.Z, &out.First.Num), roundPublicationDecode(t, s.Manager.Z, &out.First.Den))
							gotS := new(big.Rat).SetFrac(roundPublicationDecode(t, s.Manager.Z, &out.Second.Num), roundPublicationDecode(t, s.Manager.Z, &out.Second.Den))
							if gotF.Cmp(first) != 0 || gotS.Cmp(second) != 0 {
								t.Fatal(a, b, x, y, den, upper, out, first, second)
							}
							infAuditRoots(t, s, []*OwnedInfRational{&v, &w, &out}, nil)
							serviceMust(t, s.InfDel(&out))
							serviceMust(t, s.InfDel(&v))
							serviceMust(t, s.InfDel(&w))
						}
					}
				}
			}
		}
	}
}

func TestInfApproxRawProductReachedZeroDenominators(t *testing.T) {
	for _, upper := range []bool{false, true} {
		for _, sign := range []int32{-1, 1} {
			s, _ := serviceNew(t)
			v := infApproxRaw(s, 1, 1, sign, 0)
			w := infApproxRaw(s, 2, 1, 0, 1)
			var out OwnedInfRational
			var e error
			if upper {
				out, e = s.InfUpperMult(&v, &w)
			} else {
				out, e = s.InfLowerMult(&v, &w)
			}
			serviceMust(t, e)
			infApproxRawEqual(t, &out, 2, 1, sign, 0)
			infAuditRoots(t, s, []*OwnedInfRational{&v, &w, &out}, nil)
			serviceMust(t, s.InfDel(&out))
			serviceMust(t, s.InfDel(&v))
			serviceMust(t, s.InfDel(&w))
			// FIRST infinity is finite through the first product and both cross-products
			// when the RHS EPS is nonzero; the final epsilon correction retains DEN0.
			v = infApproxRaw(s, sign, 0, 1, 1)
			w = infApproxRaw(s, 2, 1, 1, 1)
			if upper {
				out, e = s.InfUpperMult(&v, &w)
			} else {
				out, e = s.InfLowerMult(&v, &w)
			}
			serviceMust(t, e)
			infApproxRawEqual(t, &out, sign, 0, sign, 0)
			infAuditRoots(t, s, []*OwnedInfRational{&v, &w, &out}, nil)
			serviceMust(t, s.InfDel(&out))
			serviceMust(t, s.InfDel(&v))
			serviceMust(t, s.InfDel(&w))
			// With RHS EPS0, the first cross-product reaches source 0/0 normalization.
			v = infApproxRaw(s, sign, 0, 1, 1)
			w = infApproxRaw(s, 2, 1, 0, 1)
			if upper {
				out, e = s.InfUpperMult(&v, &w)
			} else {
				out, e = s.InfLowerMult(&v, &w)
			}
			if !errors.Is(e, ErrDivideByZero) || out.ownerID != 0 {
				t.Fatal(upper, sign, e, out)
			}
			infAuditRoots(t, s, []*OwnedInfRational{&v, &w}, nil)
			serviceMust(t, s.InfDel(&v))
			serviceMust(t, s.InfDel(&w))
		}
	}
}
