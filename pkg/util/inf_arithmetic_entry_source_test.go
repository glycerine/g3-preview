// Source: src/util/inf_rational.h; src/util/inf_int_rational.h; src/util/inf_eps_rational.h; src/util/rational.h. Licensed under the MIT license.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

// Explicit unowned failed tokens model unread fields; restore only these
// diagnostic tokens before teardown, never an actual arithmetic prefix.
func TestInfArithmeticEntryFirstBeforeEPS(t *testing.T) {
	for _, sub := range []bool{false, true} {
		s, _ := serviceNew(t)
		a, b := infApproxRaw(s, 7, 1, 3, 1), infApproxRaw(s, 2, 1, 4, 1)
		b.Second.Num.Cell = s.Manager.Z.FailedRef
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
		got := a.First.Num
		infAuditRoots(t, s, []*OwnedInfRational{&a, &b}, nil)
		b.Second.Num.Cell = 0
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
		r := OwnedRational{Raw: Rational{Num: NewInt(2), Den: NewInt(1)}, ownerID: s.ownerID}
		a.Second.Den.Cell = s.Manager.Z.FailedRef
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
		got := a.First.Num
		untouched := a.Second.Den.Cell == s.Manager.Z.FailedRef
		infAuditRoots(t, s, []*OwnedInfRational{&a}, []*OwnedRational{&r})
		a.Second.Den.Cell = 0
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
		a.infinity.Num = NewInt(7)
		b.infinity.Num = NewInt(2)
		b.numeral.First.Num.Cell = s.Manager.Z.FailedRef
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
		got := a.infinity.Num
		infEpsAudit(t, s, []*OwnedInfEpsInfRational{&a, &b}, nil, nil)
		b.numeral.First.Num.Cell = 0
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
		a := OwnedInfIntRational{First: Rational{Num: NewInt(2), Den: NewInt(0)}, second: 7, ownerID: s.ownerID}
		b := OwnedInfIntRational{First: Rational{Num: NewInt(3), Den: NewInt(1)}, second: 2, ownerID: s.ownerID}
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
		serviceAuditRationalRoots(t, s, &a.First, &b.First)
		got := a
		serviceMust(t, s.InfIntDel(&a))
		serviceMust(t, s.InfIntDel(&b))
		if e != nil || got.First.Num != NewInt(1) || got.First.Den != NewInt(0) || got.second != want {
			t.Fatal("raw FIRST arithmetic before scalar epsilon", sub, got, e)
		}
	}
	s, _ := serviceNew(t)
	a := OwnedInfIntRational{First: Rational{Num: NewInt(7), Den: NewInt(1)}, second: 2147483647, ownerID: s.ownerID}
	b := OwnedInfIntRational{First: Rational{Num: NewInt(2), Den: NewInt(1)}, second: 1, ownerID: s.ownerID}
	e := s.InfIntAddAssign(&a, &b)
	serviceAuditRationalRoots(t, s, &a.First, &b.First)
	if !errors.Is(e, ErrInfIntOverflow) || a.First.Num != NewInt(9) || a.second != 2147483647 {
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
		a.infinity.Num.Cell = s.Manager.Z.FailedRef
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
		a.infinity.Num.Cell = 0
		serviceMust(t, s.InfEpsInfDel(&a))
		serviceMust(t, s.InfDel(&b))
		if e != nil || got.numeral.First.Num != NewInt(wantFirst) || got.numeral.Second.Num != NewInt(wantEPS) || got.infinity.Num.Cell != s.Manager.Z.FailedRef {
			t.Fatal("mixed Inf reads nested numeral only", sub, got, e)
		}
	}
}

func TestInfArithmeticEntryFacadeRawDenominator(t *testing.T) {
	for _, sub := range []bool{false, true} {
		s, _ := serviceNew(t)
		a := OwnedRational{Raw: Rational{Num: NewInt(2), Den: NewInt(0)}, ownerID: s.ownerID}
		b := OwnedRational{Raw: Rational{Num: NewInt(3), Den: NewInt(1)}, ownerID: s.ownerID}
		var e error
		if sub {
			e = s.SubAssign(&a, &b)
		} else {
			e = s.AddAssign(&a, &b)
		}
		gotNum, gotDen := a.Raw.Num, a.Raw.Den
		serviceAuditRationalRoots(t, s, &a.Raw, &b.Raw)
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
		a := OwnedRational{Raw: NewRational(), ownerID: s.ownerID}
		words := make([]uint32, 128)
		words[0], words[len(words)-1] = 17, 1
		serviceMust(t, s.Manager.Z.SetLimbs(&a.Raw.Num, words))
		b := OwnedRational{Raw: Rational{Num: NewInt(3), Den: NewInt(1)}, ownerID: s.ownerID}
		b.Raw.Den.Cell = s.Manager.Z.FailedRef
		budget.FailAfter(0)
		var out OwnedRational
		var e error
		if sub {
			out, e = s.Sub(&a, &b)
		} else {
			out, e = s.Add(&a, &b)
		}
		serviceAuditRationalRoots(t, s, &a.Raw, &b.Raw)
		refusal := numericPhysicalReservationRefusal(e)
		budget.FailAfter(-1)
		b.Raw.Den.Cell = 0
		serviceMust(t, s.Manager.Z.CleanupPendingAllocation())
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
				if out.First.Num != NewInt(first) || out.Second.Num != NewInt(eps) || a.First.Num != NewInt(av) || b.First.Num != NewInt(bv) {
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
				if a.First.Num != NewInt(want) || a.Second.Num != NewInt(wantEPS) {
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
		if out.First.Num != NewInt(1) || out.First.Den != NewInt(0) || out.Second.Num != NewInt(wantEPS) {
			t.Fatal("copy/compound/returned-copy raw member sequence", sub, out)
		}
		ai := OwnedInfIntRational{First: a.First, second: 3, ownerID: s.ownerID}
		bi := OwnedInfIntRational{First: b.First, second: 4, ownerID: s.ownerID}
		var oi OwnedInfIntRational
		if sub {
			oi, e = s.InfIntSub(&ai, &bi)
		} else {
			oi, e = s.InfIntAdd(&ai, &bi)
		}
		serviceMust(t, e)
		if oi.First.Num != NewInt(1) || oi.First.Den != NewInt(0) || oi.second != wantEPS {
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
		if oe.numeral.First.Num != NewInt(1) || oe.numeral.First.Den != NewInt(0) || oe.numeral.Second.Num != NewInt(wantEPS) {
			t.Fatal("nested raw value", sub, oe)
		}
		// All copied input fields are inline; these aliases own no shared allocation.
		serviceAuditRationalRoots(t, s, &a.First, &a.Second, &b.First, &b.Second, &out.First, &out.Second, &ai.First, &bi.First, &oi.First, &ae.infinity, &be.infinity, &oe.infinity, &oe.numeral.First, &oe.numeral.Second)
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
	before := numericWarmProfileSnapshot(t, s.Manager.Z, budget)
	allocs := testing.AllocsPerRun(100, func() {
		if e := s.InfAddAssign(&a, &b); e != nil {
			panic(e)
		}
		if e := s.InfSubAssign(&a, &b); e != nil {
			panic(e)
		}
	})
	numericWarmProfileAudit(t, s.Manager.Z, budget, before, allocs, false)
	if a.First.Num != NewInt(7) || a.Second.Num != NewInt(3) {
		t.Fatal("independent reversible pair", a)
	}
	infAuditRoots(t, s, []*OwnedInfRational{&a, &b}, nil)
	serviceMust(t, s.InfDel(&a))
	serviceMust(t, s.InfDel(&b))
}
