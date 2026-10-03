// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"math"
	"math/big"
	"testing"
)

// These raw scalar shapes are actual initialized numeral fields (no storage
// pointers). Source setter/inverse/error prefixes can publish DEN0; ordinary
// readers remain strict. No new registered-source ENSURE credit is claimed.
func infApproxRaw(s *RationalService, n, d, e, ed int32) OwnedInfRational {
	v := s.infEmpty()
	v.first = Rational{num: NewInt(n), den: NewInt(d)}
	v.second = Rational{num: NewInt(e), den: NewInt(ed)}
	return v
}
func infApproxRawEqual(t *testing.T, v *OwnedInfRational, n, d, e, ed int32) {
	t.Helper()
	if v.first.num != NewInt(n) || v.first.den != NewInt(d) || v.second.num != NewInt(e) || v.second.den != NewInt(ed) {
		t.Fatalf("raw result %#v want %d/%d,%d/%d", v, n, d, e, ed)
	}
}
func TestInfApproxRawPowerSelectedBranches(t *testing.T) {
	cases := []struct {
		upper           bool
		n, d, e, ed     int32
		p               uint32
		wn, wd, we, wed int32
	}{
		{false, 1, 0, -1, 0, 2, 0, 1, 0, 1}, // even DEFAULT, no arithmetic reads
		{true, 0, 0, 1, 0, 3, 1, 1, 0, 1},   // upper constant one
		{false, 0, 0, 1, 0, 3, -1, 1, 0, 1}, // lower constant minus one
		{false, 1, 0, 1, 0, 1, 1, 0, 1, 0},  // n1 assignment FIRST then EPS
		{true, -1, 0, -1, 0, 1, -1, 0, -1, 0},
		{false, 1, 0, 0, 0, 0, 1, 1, 0, 1}, // exact rational Power: Copy before out1
		{true, -1, 0, 0, 0, 2, 1, 0, 0, 1},
		{false, 1, 0, 0, 0, 3, 1, 0, 0, 1},
	}
	for _, tc := range cases {
		s, _ := serviceNew(t)
		v := infApproxRaw(s, tc.n, tc.d, tc.e, tc.ed)
		var out OwnedInfRational
		var e error
		if tc.upper {
			out, e = s.InfUpperPower(&v, tc.p)
		} else {
			out, e = s.InfLowerPower(&v, tc.p)
		}
		serviceMust(t, e)
		infApproxRawEqual(t, &out, tc.wn, tc.wd, tc.we, tc.wed)
		infAuditRoots(t, s, []*OwnedInfRational{&v, &out}, nil)
		serviceMust(t, s.InfDel(&out))
		serviceMust(t, s.InfDel(&v))
	}
}
func TestInfApproxUnreadPowerFields(t *testing.T) {
	s, _ := serviceNew(t)
	v := infApproxRaw(s, 1, 0, -1, 0)
	v.first.num.cell = s.manager.z.failedRef
	v.second.num.cell = s.manager.z.failedRef
	out, e := s.InfLowerPower(&v, 2)
	serviceMust(t, e)
	infApproxRawEqual(t, &out, 0, 1, 0, 1)
	serviceMust(t, s.InfDel(&out))
	v.first.num.val = 0
	v.second.num.val = 1
	out, e = s.InfUpperPower(&v, 3)
	serviceMust(t, e)
	infApproxRawEqual(t, &out, 1, 1, 0, 1)
	serviceMust(t, s.InfDel(&out))
	// EPS DEN is never read by exact power; only FIRST is copied/arithmetic.
	v = infApproxRaw(s, 2, 1, 0, 1)
	v.second.den.cell = s.manager.z.failedRef
	out, e = s.InfLowerPower(&v, 3)
	serviceMust(t, e)
	infApproxRawEqual(t, &out, 8, 1, 0, 1)
	serviceMust(t, s.InfDel(&out))
	// Del acknowledges only the deliberately unowned failed-reference tokens.
	serviceMust(t, s.InfDel(&v))
	infAuditRoots(t, s, nil, nil)
}
func TestInfApproxPublicRationalRawDependencies(t *testing.T) {
	for _, synch := range []bool{false, true} {
		m, _ := rationalManager(t, synch)
		a := Rational{num: NewInt(1), den: NewInt(0)}
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
			if out.num != NewInt(1) || out.den != NewInt(0) {
				t.Fatal(out)
			}
			qopsAuditLiveOwnership(t, m, []*Int{&a.num, &a.den, &b.num, &b.den, &out.num, &out.den})
		}
		serviceMust(t, m.Power(&a, 0, &out))
		if out.num != NewInt(1) || out.den != NewInt(1) {
			t.Fatal(out)
		}
		serviceMust(t, m.Power(&a, 3, &out))
		if out.num != NewInt(1) || out.den != NewInt(0) {
			t.Fatal(out)
		}
		// Raw infinity plus infinity reaches 0/0 normalization rather than a read guard.
		if e := m.Add(&a, &a, &out); !errors.Is(e, ErrDivideByZero) {
			t.Fatal(e)
		}
		if out.num.val != 0 || out.den.val != 0 {
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
		if sourceExtendedGCDDebug {
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
		if sourceExtendedGCDDebug {
			if !errors.Is(err, ErrIntegerAssertion) {
				t.Fatal(err)
			}
		} else if n == 0 {
			if !errors.Is(err, ErrDivideByZero) {
				t.Fatal(err)
			}
		} else {
			serviceMust(t, err)
			if iv.first.num != NewInt(n) || iv.first.den.val != 0 || iv.second != 0 {
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
	a.raw = Rational{num: NewInt(1), den: NewInt(0)}
	v := infApproxRaw(s, 7, 0, 2, 0)
	serviceMust(t, s.InfSetRational(&v, &a))
	infApproxRawEqual(t, &v, 1, 0, 0, 1)
	serviceMust(t, s.InfSetPair(&v, &a, &a))
	infApproxRawEqual(t, &v, 1, 0, 1, 0)
	serviceMust(t, s.InfSetOffset(&v, &a, false))
	infApproxRawEqual(t, &v, 1, 0, -1, 1)
	iv, e := s.InfIntFromRational(&a, 3)
	serviceMust(t, e)
	if iv.first.den.val != 0 || iv.second != 3 {
		t.Fatal(iv)
	}
	serviceMust(t, s.InfIntSetParts(&iv, &a, -2))
	if iv.second != -2 || iv.first.den.val != 0 {
		t.Fatal(iv)
	}
	ev, e := s.InfEpsInfNewValue()
	serviceMust(t, e)
	serviceMust(t, s.InfEpsInfSetPair(&ev, &a, &v))
	if ev.infinity.den.val != 0 || ev.numeral.first.den.val != 0 {
		t.Fatal(ev)
	}
	serviceMust(t, s.InfEpsInfSetNumeral(&ev, &v))
	if ev.infinity.num.val != 0 || ev.infinity.den.val != 1 {
		t.Fatal(ev)
	}
	serviceMust(t, s.InfEpsInfSetInt32(&ev, math.MinInt32))
	if ev.numeral.first.num != NewInt(math.MinInt32) || ev.numeral.second.num.val != 0 {
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
	cfg.Integers.Profile = memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
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
	serviceMust(t, s.manager.SetString(&a.first, "123456789012345678901234567890123456789"))
	other.second.den.cell = s.manager.z.failedRef
	b.FailAfter(0)
	out, e := s.InfLowerMult(&a, &other)
	if !numericPhysicalReservationRefusal(e) || out.ownerID != 0 {
		t.Fatalf("FIRST Copy must precede EPS read: %v %#v", e, out)
	}
	infAuditRoots(t, s, []*OwnedInfRational{&a, &other}, nil)
	b.FailAfter(-1)
	serviceMust(t, s.manager.z.CleanupPendingAllocation())
	serviceMust(t, s.InfDel(&a))
	serviceMust(t, s.InfDel(&other))
}

func TestInfApproxFiniteSignParityAndProductTable(t *testing.T) {
	s, _ := serviceNew(t)
	for n := int32(-2); n <= 2; n++ {
		for eps := int32(-1); eps <= 1; eps++ {
			for p := uint32(0); p <= 5; p++ {
				for _, upper := range []bool{false, true} {
					v := infApproxRaw(s, n, 1, eps, 1)
					var out OwnedInfRational
					var e error
					if upper {
						out, e = s.InfUpperPower(&v, p)
					} else {
						out, e = s.InfLowerPower(&v, p)
					}
					serviceMust(t, e)
					want := new(big.Rat)
					wantE := int32(0)
					even := p&1 == 0
					exact := eps == 0
					if upper {
						exact = exact || n > 0 && eps < 0 || n < 0 && eps > 0 && even
					} else {
						exact = exact || n > 0 && eps > 0 || n < 0 && eps < 0 && even
					}
					switch {
					case p == 1:
						want.SetInt64(int64(n))
						wantE = eps
					case exact:
						want.SetInt(new(big.Int).Exp(big.NewInt(int64(n)), big.NewInt(int64(p)), nil))
					case !upper && even:
					case !upper && n == 0:
						want.SetInt64(-1)
					case upper && (n == 0 || p == 0):
						want.SetInt64(1)
					default:
						factor := int64(1)
						if upper && (n > 0 || even) || !upper && n < 0 {
							factor = 3
						}
						base := new(big.Rat).SetFrac(big.NewInt(int64(n)*factor), big.NewInt(2))
						want.SetFrac(new(big.Int).Exp(base.Num(), big.NewInt(int64(p)), nil), new(big.Int).Exp(base.Denom(), big.NewInt(int64(p)), nil))
					}
					gotN := roundPublicationDecode(t, s.manager.z, &out.first.num)
					gotD := roundPublicationDecode(t, s.manager.z, &out.first.den)
					if new(big.Rat).SetFrac(gotN, gotD).Cmp(want) != 0 || out.second.num != NewInt(wantE) || out.second.den != NewInt(1) {
						t.Fatal(n, eps, p, upper, out, want)
					}
					infAuditRoots(t, s, []*OwnedInfRational{&v, &out}, nil)
					serviceMust(t, s.InfDel(&out))
					serviceMust(t, s.InfDel(&v))
				}
			}
		}
	}
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
							gotF := new(big.Rat).SetFrac(roundPublicationDecode(t, s.manager.z, &out.first.num), roundPublicationDecode(t, s.manager.z, &out.first.den))
							gotS := new(big.Rat).SetFrac(roundPublicationDecode(t, s.manager.z, &out.second.num), roundPublicationDecode(t, s.manager.z, &out.second.den))
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

func TestInfApproxPowerActualMaskAndEarlierRefusal(t *testing.T) {
	for _, upper := range []bool{false, true} {
		for _, n := range []int32{-1, 0, 1} {
			s, _ := serviceNew(t)
			v := infApproxRaw(s, n, 1, 0, 1)
			var out OwnedInfRational
			var e error
			if upper {
				out, e = s.InfUpperPower(&v, 1<<31)
			} else {
				out, e = s.InfLowerPower(&v, 1<<31)
			}
			if !errors.Is(e, ErrNonterminatingPower) || out.ownerID != 0 {
				t.Fatal(n, upper, e, out)
			}
			infAuditRoots(t, s, []*OwnedInfRational{&v}, nil)
			serviceMust(t, s.InfDel(&v))
		}
	}
	for _, upper := range []bool{false, true} {
		budget := arena.NewBudget(1 << 24)
		cfg := serviceConfig(budget)
		cfg.Integers.Profile = memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
		s, e := NewRationalService(cfg)
		serviceMust(t, e)
		// The exact branch raises FIRST2; its first actual square allocation must
		// refuse before the later mask-wrap boundary. No gigantic arithmetic oracle.
		v := infApproxRaw(s, 2, 1, 0, 1)
		budget.FailAfter(0)
		var out OwnedInfRational
		if upper {
			out, e = s.InfUpperPower(&v, 1<<31)
		} else {
			out, e = s.InfLowerPower(&v, 1<<31)
		}
		if !numericPhysicalReservationRefusal(e) || errors.Is(e, ErrNonterminatingPower) || out.ownerID != 0 {
			t.Fatal(upper, e, out)
		}
		infAuditRoots(t, s, []*OwnedInfRational{&v}, nil)
		budget.FailAfter(-1)
		serviceMust(t, s.manager.CleanupRawArithmetic())
		serviceMust(t, s.manager.z.CleanupPendingAllocation())
		serviceMust(t, s.InfDel(&v))
		serviceMust(t, s.Close())
		if budget.Used() != 0 {
			t.Fatal(budget.Used())
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
