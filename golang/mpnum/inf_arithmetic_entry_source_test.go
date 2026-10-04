// Source: src/util/inf_rational.h; src/util/inf_int_rational.h; src/util/inf_eps_rational.h; src/util/rational.h. Licensed under the MIT license.
package mpnum

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/golang/arena"
)

// Explicit unowned failed tokens model unread fields; restore only these
// diagnostic tokens before teardown, never an actual arithmetic prefix.
func TestInfArithmeticEntryFirstBeforeEPS(t *testing.T) {
	for _, sub := range []bool{false, true} {
		s, _ := serviceNew(t)
		a, b := infApproxRaw(s, 7, 1, 3, 1), infApproxRaw(s, 2, 1, 4, 1)
		b.second.num.cell = s.manager.z.failedRef
		var e error
		if sub {
			e = s.InfSubAssign(&a, &b)
		} else {
			e = s.InfAddAssign(&a, &b)
		}
		want := int32(9)
		if sub {
			want = 5
		}
		got := a.first.num
		infAuditRoots(t, s, []*OwnedInfRational{&a, &b}, nil)
		b.second.num.cell = 0
		serviceMust(t, s.InfDel(&a))
		serviceMust(t, s.InfDel(&b))
		if !errors.Is(e, arena.ErrInvalidSlot) || got != NewInt(want) {
			t.Fatal("FIRST must complete before reached EPS", sub, got, e)
		}
	}
}

func TestInfArithmeticEntryMixedSkipsEPS(t *testing.T) {
	for _, sub := range []bool{false, true} {
		s, _ := serviceNew(t)
		a := infApproxRaw(s, 7, 1, 3, 1)
		r := OwnedRational{raw: Rational{num: NewInt(2), den: NewInt(1)}, ownerID: s.ownerID}
		a.second.den.cell = s.manager.z.failedRef
		var e error
		if sub {
			e = s.InfSubRationalAssign(&a, &r)
		} else {
			e = s.InfAddRationalAssign(&a, &r)
		}
		want := int32(9)
		if sub {
			want = 5
		}
		got := a.first.num
		untouched := a.second.den.cell == s.manager.z.failedRef
		infAuditRoots(t, s, []*OwnedInfRational{&a}, []*OwnedRational{&r})
		a.second.den.cell = 0
		serviceMust(t, s.InfDel(&a))
		serviceMust(t, s.Del(&r))
		if e != nil || got != NewInt(want) || !untouched {
			t.Fatal("mixed addition reads FIRST only", sub, got, untouched, e)
		}
	}
}

func TestInfArithmeticEntryInfinityBeforeNumeral(t *testing.T) {
	for _, sub := range []bool{false, true} {
		s, _ := serviceNew(t)
		a, b := s.infEpsInfEmpty(), s.infEpsInfEmpty()
		a.infinity.num = NewInt(7)
		b.infinity.num = NewInt(2)
		b.numeral.first.num.cell = s.manager.z.failedRef
		var e error
		if sub {
			e = s.InfEpsInfSubAssign(&a, &b)
		} else {
			e = s.InfEpsInfAddAssign(&a, &b)
		}
		want := int32(9)
		if sub {
			want = 5
		}
		got := a.infinity.num
		infEpsAudit(t, s, []*OwnedInfEpsInfRational{&a, &b}, nil, nil)
		b.numeral.first.num.cell = 0
		serviceMust(t, s.InfEpsInfDel(&a))
		serviceMust(t, s.InfEpsInfDel(&b))
		infAuditRoots(t, s, nil, nil)
		if !errors.Is(e, arena.ErrInvalidSlot) || got != NewInt(want) {
			t.Fatal("infinity completes before nested FIRST", sub, got, e)
		}
	}
}

func TestInfArithmeticEntryInfIntRawAndOverflow(t *testing.T) {
	for _, sub := range []bool{false, true} {
		s, _ := serviceNew(t)
		a := OwnedInfIntRational{first: Rational{num: NewInt(2), den: NewInt(0)}, second: 7, ownerID: s.ownerID}
		b := OwnedInfIntRational{first: Rational{num: NewInt(3), den: NewInt(1)}, second: 2, ownerID: s.ownerID}
		var e error
		if sub {
			e = s.InfIntSubAssign(&a, &b)
		} else {
			e = s.InfIntAddAssign(&a, &b)
		}
		want := int32(9)
		if sub {
			want = 5
		}
		serviceAuditRationalRoots(t, s, &a.first, &b.first)
		got := a
		serviceMust(t, s.InfIntDel(&a))
		serviceMust(t, s.InfIntDel(&b))
		if e != nil || got.first.num != NewInt(1) || got.first.den != NewInt(0) || got.second != want {
			t.Fatal("raw FIRST arithmetic before scalar epsilon", sub, got, e)
		}
	}
	s, _ := serviceNew(t)
	a := OwnedInfIntRational{first: Rational{num: NewInt(7), den: NewInt(1)}, second: 2147483647, ownerID: s.ownerID}
	b := OwnedInfIntRational{first: Rational{num: NewInt(2), den: NewInt(1)}, second: 1, ownerID: s.ownerID}
	e := s.InfIntAddAssign(&a, &b)
	serviceAuditRationalRoots(t, s, &a.first, &b.first)
	if !errors.Is(e, ErrInfIntOverflow) || a.first.num != NewInt(9) || a.second != 2147483647 {
		t.Fatal("signed-overflow adapter follows completed FIRST", a, e)
	}
	serviceMust(t, s.InfIntDel(&a))
	serviceMust(t, s.InfIntDel(&b))
}

func TestInfArithmeticEntryMixedSkipsInfinity(t *testing.T) {
	for _, sub := range []bool{false, true} {
		s, _ := serviceNew(t)
		a := s.infEpsInfEmpty()
		a.numeral = infApproxRaw(s, 7, 1, 3, 1)
		a.infinity.num.cell = s.manager.z.failedRef
		b := infApproxRaw(s, 2, 1, 4, 1)
		var e error
		if sub {
			e = s.InfEpsInfSubInfAssign(&a, &b)
		} else {
			e = s.InfEpsInfAddInfAssign(&a, &b)
		}
		wantFirst, wantEPS := int32(9), int32(7)
		if sub {
			wantFirst, wantEPS = 5, -1
		}
		infEpsAudit(t, s, []*OwnedInfEpsInfRational{&a}, []*OwnedInfRational{&b}, nil)
		got := a
		a.infinity.num.cell = 0
		serviceMust(t, s.InfEpsInfDel(&a))
		serviceMust(t, s.InfDel(&b))
		if e != nil || got.numeral.first.num != NewInt(wantFirst) || got.numeral.second.num != NewInt(wantEPS) || got.infinity.num.cell != s.manager.z.failedRef {
			t.Fatal("mixed Inf reads nested numeral only", sub, got, e)
		}
	}
}

func TestInfArithmeticEntryFacadeRawDenominator(t *testing.T) {
	for _, sub := range []bool{false, true} {
		s, _ := serviceNew(t)
		a := OwnedRational{raw: Rational{num: NewInt(2), den: NewInt(0)}, ownerID: s.ownerID}
		b := OwnedRational{raw: Rational{num: NewInt(3), den: NewInt(1)}, ownerID: s.ownerID}
		var e error
		if sub {
			e = s.SubAssign(&a, &b)
		} else {
			e = s.AddAssign(&a, &b)
		}
		gotNum, gotDen := a.raw.num, a.raw.den
		serviceAuditRationalRoots(t, s, &a.raw, &b.raw)
		serviceMust(t, s.Del(&a))
		serviceMust(t, s.Del(&b))
		if e != nil || gotNum != NewInt(1) || gotDen != NewInt(0) {
			t.Fatal("literal finite raw denominator action", sub, gotNum, gotDen, e)
		}
	}
}

func TestInfArithmeticEntryValueCopyBeforeRightRead(t *testing.T) {
	for _, sub := range []bool{false, true} {
		s, budget := serviceNew(t)
		a := OwnedRational{raw: NewRational(), ownerID: s.ownerID}
		words := make([]uint32, 128)
		words[0], words[len(words)-1] = 17, 1
		serviceMust(t, s.manager.z.SetLimbs(&a.raw.num, words))
		b := OwnedRational{raw: Rational{num: NewInt(3), den: NewInt(1)}, ownerID: s.ownerID}
		b.raw.den.cell = s.manager.z.failedRef
		budget.FailAfter(0)
		var out OwnedRational
		var e error
		if sub {
			out, e = s.Sub(&a, &b)
		} else {
			out, e = s.Add(&a, &b)
		}
		serviceAuditRationalRoots(t, s, &a.raw, &b.raw)
		refusal := numericPhysicalReservationRefusal(e)
		budget.FailAfter(-1)
		b.raw.den.cell = 0
		serviceMust(t, s.manager.z.CleanupPendingAllocation())
		serviceMust(t, s.Del(&a))
		serviceMust(t, s.Del(&b))
		if !refusal || out.ownerID != 0 {
			t.Fatal("left COPY constructor allocation precedes right DEN", sub, out, e)
		}
	}
}

func TestInfArithmeticEntryBoundedValuesAndAliases(t *testing.T) {
	for av := int32(-3); av <= 3; av++ {
		for bv := int32(-3); bv <= 3; bv++ {
			for _, sub := range []bool{false, true} {
				s, _ := serviceNew(t)
				a, b := infApproxRaw(s, av, 1, av+1, 1), infApproxRaw(s, bv, 1, bv-1, 1)
				var out OwnedInfRational
				var e error
				if sub {
					out, e = s.InfSub(&a, &b)
				} else {
					out, e = s.InfAdd(&a, &b)
				}
				serviceMust(t, e)
				first, eps := av+bv, av+bv
				if sub {
					first, eps = av-bv, av-bv+2
				}
				if out.first.num != NewInt(first) || out.second.num != NewInt(eps) || a.first.num != NewInt(av) || b.first.num != NewInt(bv) {
					t.Fatal("independent integer pair oracle", av, bv, sub, out)
				}
				if sub {
					e = s.InfSubAssign(&a, &a)
				} else {
					e = s.InfAddAssign(&a, &a)
				}
				serviceMust(t, e)
				want := av * 2
				wantEPS := (av + 1) * 2
				if sub {
					want, wantEPS = 0, 0
				}
				if a.first.num != NewInt(want) || a.second.num != NewInt(wantEPS) {
					t.Fatal("actual output/operand identity", av, sub, a)
				}
				infAuditRoots(t, s, []*OwnedInfRational{&a, &b, &out}, nil)
				serviceMust(t, s.InfDel(&a))
				serviceMust(t, s.InfDel(&b))
				serviceMust(t, s.InfDel(&out))
			}
		}
	}
}

func TestInfArithmeticEntryRawValueMemberRoutes(t *testing.T) {
	for _, sub := range []bool{false, true} {
		s, _ := serviceNew(t)
		a, b := infApproxRaw(s, 2, 0, 3, 1), infApproxRaw(s, 3, 1, 4, 1)
		var out OwnedInfRational
		var e error
		if sub {
			out, e = s.InfSub(&a, &b)
		} else {
			out, e = s.InfAdd(&a, &b)
		}
		serviceMust(t, e)
		wantEPS := int32(7)
		if sub {
			wantEPS = -1
		}
		if out.first.num != NewInt(1) || out.first.den != NewInt(0) || out.second.num != NewInt(wantEPS) {
			t.Fatal("copy/compound/returned-copy raw member sequence", sub, out)
		}
		ai := OwnedInfIntRational{first: a.first, second: 3, ownerID: s.ownerID}
		bi := OwnedInfIntRational{first: b.first, second: 4, ownerID: s.ownerID}
		var oi OwnedInfIntRational
		if sub {
			oi, e = s.InfIntSub(&ai, &bi)
		} else {
			oi, e = s.InfIntAdd(&ai, &bi)
		}
		serviceMust(t, e)
		if oi.first.num != NewInt(1) || oi.first.den != NewInt(0) || oi.second != wantEPS {
			t.Fatal("InfInt raw value", sub, oi)
		}
		ae, be := s.infEpsInfEmpty(), s.infEpsInfEmpty()
		ae.numeral = a
		be.numeral = b
		var oe OwnedInfEpsInfRational
		if sub {
			oe, e = s.InfEpsInfSub(&ae, &be)
		} else {
			oe, e = s.InfEpsInfAdd(&ae, &be)
		}
		serviceMust(t, e)
		if oe.numeral.first.num != NewInt(1) || oe.numeral.first.den != NewInt(0) || oe.numeral.second.num != NewInt(wantEPS) {
			t.Fatal("nested raw value", sub, oe)
		}
		// All copied input fields are inline; these aliases own no shared allocation.
		serviceAuditRationalRoots(t, s, &a.first, &a.second, &b.first, &b.second, &out.first, &out.second, &ai.first, &bi.first, &oi.first, &ae.infinity, &be.infinity, &oe.infinity, &oe.numeral.first, &oe.numeral.second)
		serviceMust(t, s.InfDel(&out))
		serviceMust(t, s.InfIntDel(&oi))
		serviceMust(t, s.InfIntDel(&ai))
		serviceMust(t, s.InfIntDel(&bi))
		serviceMust(t, s.InfEpsInfDel(&ae))
		serviceMust(t, s.InfEpsInfDel(&be))
		serviceMust(t, s.InfEpsInfDel(&oe))
		a, b = OwnedInfRational{}, OwnedInfRational{} // inline aliases transferred to ae/be
	}
}

func TestInfArithmeticEntryInlineWarmNoAllocation(t *testing.T) {
	s, budget := serviceNew(t)
	a, b := infApproxRaw(s, 7, 1, 3, 1), infApproxRaw(s, 2, 1, 4, 1)
	serviceMust(t, s.InfAddAssign(&a, &b))
	serviceMust(t, s.InfSubAssign(&a, &b))
	before := numericWarmProfileSnapshot(t, s.manager.z, budget)
	allocs := testing.AllocsPerRun(100, func() {
		if e := s.InfAddAssign(&a, &b); e != nil {
			panic(e)
		}
		if e := s.InfSubAssign(&a, &b); e != nil {
			panic(e)
		}
	})
	numericWarmProfileAudit(t, s.manager.z, budget, before, allocs, false)
	if a.first.num != NewInt(7) || a.second.num != NewInt(3) {
		t.Fatal("independent reversible pair", a)
	}
	infAuditRoots(t, s, []*OwnedInfRational{&a, &b}, nil)
	serviceMust(t, s.InfDel(&a))
	serviceMust(t, s.InfDel(&b))
}
