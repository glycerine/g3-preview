// Source: src/util/rational.h; src/util/inf_rational.h; src/util/inf_int_rational.h; src/util/inf_eps_rational.h.
package numeral

import (
	"errors"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

// Failed tokens here are explicit unowned diagnostic fields. Actual completed
// source fields and named roots are inspected before restoring those tokens.
func TestInfMulDivEntryFacadeNUMBeforeDEN(t *testing.T) {
	for _, div := range []bool{false, true} {
		s, _ := serviceNew(t)
		a := OwnedRational{Raw: Rational{Num: NewInt(7), Den: NewInt(1)}, ownerID: s.ownerID}
		b := OwnedRational{Raw: Rational{Num: NewInt(3), Den: NewInt(2)}, ownerID: s.ownerID}
		a.Raw.Den.Cell = s.Manager.Z.FailedRef
		var e error
		if div {
			e = s.DivAssign(&a, &b)
		} else {
			e = s.MulAssign(&a, &b)
		}
		serviceAuditRationalRoots(t, s, &a.Raw, &b.Raw)
		want := int32(21)
		if div {
			want = 14
		}
		got := a.Raw.Num
		untouched := a.Raw.Den.Cell == s.Manager.Z.FailedRef
		a.Raw.Den.Cell = 0
		serviceMust(t, s.Del(&a))
		serviceMust(t, s.Del(&b))
		if !errors.Is(e, arena.ErrInvalidSlot) || got != NewInt(want) || !untouched {
			t.Error("NUM action before reached DEN", div, got, untouched, e)
		}
	}
}

func TestInfMulDivEntryFIRSTBeforeEPS(t *testing.T) {
	for _, div := range []bool{false, true} {
		s, _ := serviceNew(t)
		a := infApproxRaw(s, 6, 1, 3, 1)
		r := OwnedRational{Raw: Rational{Num: NewInt(2), Den: NewInt(1)}, ownerID: s.ownerID}
		a.Second.Num.Cell = s.Manager.Z.FailedRef
		var e error
		if div {
			e = s.InfDivRationalAssign(&a, &r)
		} else {
			e = s.InfMulRationalAssign(&a, &r)
		}
		infAuditRoots(t, s, []*OwnedInfRational{&a}, []*OwnedRational{&r})
		want := int32(12)
		if div {
			want = 3
		}
		got := a.First.Num
		untouched := a.Second.Num.Cell == s.Manager.Z.FailedRef
		a.Second.Num.Cell = 0
		serviceMust(t, s.InfDel(&a))
		serviceMust(t, s.Del(&r))
		if !errors.Is(e, arena.ErrInvalidSlot) || got != NewInt(want) || !untouched {
			t.Error("FIRST source action precedes EPS", div, got, e)
		}
	}
}

func TestInfMulDivEntryInfinityBeforeNestedFIRST(t *testing.T) {
	for _, div := range []bool{false, true} {
		s, _ := serviceNew(t)
		a := s.infEpsInfEmpty()
		a.infinity.Num = NewInt(6)
		r := OwnedRational{Raw: Rational{Num: NewInt(2), Den: NewInt(1)}, ownerID: s.ownerID}
		a.numeral.First.Num.Cell = s.Manager.Z.FailedRef
		var e error
		if div {
			e = s.InfEpsInfDivRationalAssign(&a, &r)
		} else {
			e = s.InfEpsInfMulRationalAssign(&a, &r)
		}
		infEpsAudit(t, s, []*OwnedInfEpsInfRational{&a}, nil, []*OwnedRational{&r})
		want := int32(12)
		if div {
			want = 3
		}
		got := a.infinity.Num
		a.numeral.First.Num.Cell = 0
		serviceMust(t, s.InfEpsInfDel(&a))
		serviceMust(t, s.Del(&r))
		if !errors.Is(e, arena.ErrInvalidSlot) || got != NewInt(want) {
			t.Error("infinity source action precedes nested FIRST", div, got, e)
		}
	}
}

func TestInfMulDivEntryInfIntCoefficientBeforeFIRST(t *testing.T) {
	for _, den := range []int32{0, 2} {
		s, _ := serviceNew(t)
		a := OwnedInfIntRational{First: Rational{Num: NewInt(7), Den: NewInt(1)}, second: 3, ownerID: s.ownerID}
		r := OwnedRational{Raw: Rational{Num: NewInt(1), Den: NewInt(den)}, ownerID: s.ownerID}
		a.First.Num.Cell = s.Manager.Z.FailedRef
		r.Raw.Num.Cell = s.Manager.Z.FailedRef
		e := s.InfIntMulAssign(&a, &r)
		serviceAuditRationalRoots(t, s, &a.First, &r.Raw)
		untouched := a.First.Num.Cell == s.Manager.Z.FailedRef && a.First.Num.Val == 7 && a.second == 3
		a.First.Num.Cell = 0
		r.Raw.Num.Cell = 0
		serviceMust(t, s.InfIntDel(&a))
		serviceMust(t, s.Del(&r))
		if !errors.Is(e, ErrInfIntScalar) || !untouched {
			t.Error("structural coefficient fit rejects before either NUM", den, untouched, e)
		}
	}
}

func TestInfMulDivEntryValueCOPYRefusalBeforeRight(t *testing.T) {
	for kind := 0; kind < 4; kind++ {
		for _, div := range []bool{false, true} {
			if kind == 3 && div {
				continue
			} // source InfInt has multiplication only
			s, budget := serviceNew(t)
			base := NewRational()
			words := make([]uint32, 128)
			words[0], words[127] = 17, 1
			serviceMust(t, s.Manager.Z.SetLimbs(&base.Num, words))
			r := OwnedRational{Raw: Rational{Num: NewInt(3), Den: NewInt(2)}, ownerID: s.ownerID}
			if kind != 3 {
				r.Raw.Den.Cell = s.Manager.Z.FailedRef
			} else {
				r.Raw.Num.Cell = s.Manager.Z.FailedRef
			}
			budget.FailAfter(0)
			var e error
			owner := uint64(0)
			switch kind {
			case 0:
				a := OwnedRational{Raw: base, ownerID: s.ownerID}
				var out OwnedRational
				if div {
					out, e = s.Div(&a, &r)
				} else {
					out, e = s.Mul(&a, &r)
				}
				owner = uint64(out.ownerID)
			case 1:
				a := s.infEmpty()
				a.First = base
				var out OwnedInfRational
				if div {
					out, e = s.InfDivRational(&a, &r)
				} else {
					out, e = s.InfMulRational(&a, &r)
				}
				owner = uint64(out.ownerID)
			case 2:
				a := s.infEpsInfEmpty()
				a.infinity = base
				var out OwnedInfEpsInfRational
				if div {
					out, e = s.InfEpsInfDivRational(&a, &r)
				} else {
					out, e = s.InfEpsInfMulRational(&a, &r)
				}
				owner = uint64(out.ownerID)
			case 3:
				a := OwnedInfIntRational{First: base, second: 3, ownerID: s.ownerID}
				var out OwnedInfIntRational
				out, e = s.InfIntMul(&r, &a)
				owner = uint64(out.ownerID)
			}
			serviceAuditRationalRoots(t, s, &base, &r.Raw)
			refusal := numericPhysicalReservationRefusal(e)
			budget.FailAfter(-1)
			if kind != 3 {
				r.Raw.Den.Cell = 0
			} else {
				r.Raw.Num.Cell = 0
			}
			serviceMust(t, s.Manager.Z.CleanupPendingAllocation())
			serviceMust(t, s.Manager.Del(&base))
			serviceMust(t, s.Del(&r))
			if !refusal || owner != 0 {
				t.Error("COPY constructor allocation precedes right fit/read", kind, div, owner, e)
			}
		}
	}
}

func TestInfMulDivEntryCoefficientAliasReread(t *testing.T) {
	s, _ := serviceNew(t)
	a := OwnedInfIntRational{First: Rational{Num: NewInt(3), Den: NewInt(1)}, second: 2, ownerID: s.ownerID}
	// Real same-numeral source reference, no production converter/model call.
	serviceMust(t, s.mulInfInt(&a, &a.First))
	if a.First.Num != NewInt(9) || a.second != 18 {
		t.Fatal("get_int32 rereads coefficient AFTER FIRST product", a)
	}
	serviceAuditRationalRoots(t, s, &a.First)
	serviceMust(t, s.InfIntDel(&a))
}

func infMulDivExpect(t *testing.T, s *RationalService, a *Rational, want *big.Rat) {
	t.Helper()
	n := roundPublicationDecode(t, s.Manager.Z, &a.Num)
	d := roundPublicationDecode(t, s.Manager.Z, &a.Den)
	if d.Sign() == 0 {
		t.Fatal("finite oracle encountered zero denominator", n, d)
	}
	got := new(big.Rat).SetFrac(n, d)
	if got.Cmp(want) != 0 {
		t.Fatal("independent finite rational oracle", got, want)
	}
}

func TestInfMulDivEntryBoundedNumericValueAndCompound(t *testing.T) {
	for av := int32(-3); av <= 3; av++ {
		for cv := int32(-2); cv <= 2; cv++ {
			for _, div := range []bool{false, true} {
				if div && cv == 0 {
					continue
				} // zero division has a separate literal-source fixture
				s, _ := serviceNew(t)
				a := infApproxRaw(s, av, 2, av+1, 3)
				r := OwnedRational{Raw: Rational{Num: NewInt(cv), Den: NewInt(2)}, ownerID: s.ownerID}
				first := big.NewRat(int64(av), 2)
				eps := big.NewRat(int64(av+1), 3)
				coef := big.NewRat(int64(cv), 2)
				if div {
					first.Quo(first, coef)
					eps.Quo(eps, coef)
				} else {
					first.Mul(first, coef)
					eps.Mul(eps, coef)
				}
				var out OwnedInfRational
				var e error
				if div {
					out, e = s.InfDivRational(&a, &r)
				} else {
					out, e = s.InfMulRational(&a, &r)
				}
				serviceMust(t, e)
				infMulDivExpect(t, s, &out.First, first)
				infMulDivExpect(t, s, &out.Second, eps)
				if a.First.Num != NewInt(av) || a.First.Den != NewInt(2) || a.Second.Num != NewInt(av+1) || r.Raw.Num != NewInt(cv) {
					t.Fatal("value-return input mutated", a, r)
				}
				if div {
					e = s.InfDivRationalAssign(&a, &r)
				} else {
					e = s.InfMulRationalAssign(&a, &r)
				}
				serviceMust(t, e)
				infMulDivExpect(t, s, &a.First, first)
				infMulDivExpect(t, s, &a.Second, eps)
				ae := s.infEpsInfEmpty()
				ae.infinity = Rational{Num: NewInt(2), Den: NewInt(5)}
				ae.numeral = infApproxRaw(s, av, 2, av+1, 3)
				var oe OwnedInfEpsInfRational
				if div {
					oe, e = s.InfEpsInfDivRational(&ae, &r)
				} else {
					oe, e = s.InfEpsInfMulRational(&ae, &r)
				}
				serviceMust(t, e)
				infinity := big.NewRat(2, 5)
				if div {
					infinity.Quo(infinity, coef)
				} else {
					infinity.Mul(infinity, coef)
				}
				infMulDivExpect(t, s, &oe.infinity, infinity)
				infMulDivExpect(t, s, &oe.numeral.First, first)
				infMulDivExpect(t, s, &oe.numeral.Second, eps)
				if div {
					e = s.InfEpsInfDivRationalAssign(&ae, &r)
				} else {
					e = s.InfEpsInfMulRationalAssign(&ae, &r)
				}
				serviceMust(t, e)
				infMulDivExpect(t, s, &ae.infinity, infinity)
				infMulDivExpect(t, s, &ae.numeral.First, first)
				infMulDivExpect(t, s, &ae.numeral.Second, eps)
				infEpsAudit(t, s, []*OwnedInfEpsInfRational{&ae, &oe}, []*OwnedInfRational{&a, &out}, []*OwnedRational{&r})
				serviceMust(t, s.InfDel(&a))
				serviceMust(t, s.InfDel(&out))
				serviceMust(t, s.InfEpsInfDel(&ae))
				serviceMust(t, s.InfEpsInfDel(&oe))
				serviceMust(t, s.Del(&r))
			}
		}
	}
}

func TestInfMulDivEntryScalarMemberAliasRereads(t *testing.T) {
	for _, div := range []bool{false, true} {
		s, _ := serviceNew(t)
		a := infApproxRaw(s, 3, 1, 2, 1)
		op := byte('*')
		if div {
			op = '/'
		}
		serviceMust(t, s.infScalar(&a, &a.First, op))
		first, eps := int32(9), int32(18)
		if div {
			first, eps = 1, 2
		}
		if a.First.Num != NewInt(first) || a.Second.Num != NewInt(eps) {
			t.Fatal("EPS rereads reached scalar FIRST", div, a)
		}
		b := s.infEpsInfEmpty()
		b.infinity.Num = NewInt(3)
		b.numeral = infApproxRaw(s, 4, 1, 2, 1)
		serviceMust(t, s.infEpsInfScalar(&b, &b.infinity, op))
		infinity, numeral, epsilon := int32(9), int32(36), int32(18)
		if div {
			infinity, numeral, epsilon = 1, 4, 2
		}
		if b.infinity.Num != NewInt(infinity) || b.numeral.First.Num != NewInt(numeral) || b.numeral.Second.Num != NewInt(epsilon) {
			t.Fatal("nested numeral rereads scalar infinity", div, b)
		}
		infEpsAudit(t, s, []*OwnedInfEpsInfRational{&b}, []*OwnedInfRational{&a}, nil)
		serviceMust(t, s.InfDel(&a))
		serviceMust(t, s.InfEpsInfDel(&b))
	}
}

func TestInfMulDivEntryRawDenominatorAndZero(t *testing.T) {
	for _, div := range []bool{false, true} {
		s, _ := serviceNew(t)
		a := infApproxRaw(s, 2, 0, 3, 0)
		r := OwnedRational{Raw: Rational{Num: NewInt(2), Den: NewInt(1)}, ownerID: s.ownerID}
		var e error
		if div {
			e = s.InfDivRationalAssign(&a, &r)
		} else {
			e = s.InfMulRationalAssign(&a, &r)
		}
		serviceMust(t, e)
		if a.First.Num != NewInt(1) || a.First.Den != NewInt(0) || a.Second.Num != NewInt(1) || a.Second.Den != NewInt(0) {
			t.Fatal("source finite raw 1/0 normalization", div, a)
		}
		if _, e = s.InfString(&a); !errors.Is(e, ErrUninitializedRational) {
			t.Fatal("ordinary readers remain strict", e)
		}
		infAuditRoots(t, s, []*OwnedInfRational{&a}, []*OwnedRational{&r})
		serviceMust(t, s.InfDel(&a))
		serviceMust(t, s.Del(&r))
	}
	s, _ := serviceNew(t)
	a := infApproxRaw(s, 0, 1, 3, 1)
	r := OwnedRational{Raw: Rational{Num: NewInt(0), Den: NewInt(1)}, ownerID: s.ownerID}
	e := s.InfDivRationalAssign(&a, &r)
	infAuditRoots(t, s, []*OwnedInfRational{&a}, []*OwnedRational{&r})
	if !errors.Is(e, ErrDivideByZero) || a.First.Num != NewInt(0) || a.First.Den != NewInt(0) || a.Second.Num != NewInt(3) {
		t.Fatal("reached unsafe normalize after actual FIRST0/0; EPS unread", a, e)
	}
	serviceMust(t, s.InfDel(&a))
	serviceMust(t, s.Del(&r))
}

func TestInfMulDivEntryInlineWarmZero(t *testing.T) {
	s, budget := serviceNew(t)
	a := infApproxRaw(s, 3, 1, 2, 1)
	r := OwnedRational{Raw: Rational{Num: NewInt(2), Den: NewInt(1)}, ownerID: s.ownerID}
	serviceMust(t, s.InfMulRationalAssign(&a, &r))
	serviceMust(t, s.InfDivRationalAssign(&a, &r))
	before := numericWarmProfileSnapshot(t, s.Manager.Z, budget)
	allocs := testing.AllocsPerRun(100, func() {
		if e := s.InfMulRationalAssign(&a, &r); e != nil {
			panic(e)
		}
		if e := s.InfDivRationalAssign(&a, &r); e != nil {
			panic(e)
		}
	})
	numericWarmProfileAudit(t, s.Manager.Z, budget, before, allocs, false)
	if a.First.Num != NewInt(3) || a.Second.Num != NewInt(2) {
		t.Fatal("reversible inline oracle", a)
	}
	infAuditRoots(t, s, []*OwnedInfRational{&a}, []*OwnedRational{&r})
	serviceMust(t, s.InfDel(&a))
	serviceMust(t, s.Del(&r))
}

func TestInfMulDivEntryForeignAndClosedBeforeActions(t *testing.T) {
	s, _ := serviceNew(t)
	foreign, _ := serviceNew(t)
	r := OwnedRational{Raw: NewRational(), ownerID: foreign.ownerID}
	a := infApproxRaw(s, 7, 1, 3, 1)
	ai := OwnedInfIntRational{First: a.First, second: 3, ownerID: s.ownerID}
	ae := s.infEpsInfEmpty()
	ae.numeral = a
	ar := OwnedRational{Raw: a.First, ownerID: s.ownerID}
	for _, div := range []bool{false, true} {
		var e error
		if div {
			e = s.DivAssign(&ar, &r)
		} else {
			e = s.MulAssign(&ar, &r)
		}
		if !errors.Is(e, ErrRationalServiceOwner) || ar.Raw.Num != NewInt(7) {
			t.Fatal("foreign facade precedes numeric actions", div, e)
		}
		if div {
			e = s.InfDivRationalAssign(&a, &r)
		} else {
			e = s.InfMulRationalAssign(&a, &r)
		}
		if !errors.Is(e, ErrRationalServiceOwner) || a.First.Num != NewInt(7) {
			t.Fatal("foreign Inf coefficient", div, e)
		}
		if div {
			e = s.InfEpsInfDivRationalAssign(&ae, &r)
		} else {
			e = s.InfEpsInfMulRationalAssign(&ae, &r)
		}
		if !errors.Is(e, ErrRationalServiceOwner) || ae.numeral.First.Num != NewInt(7) {
			t.Fatal("foreign nested coefficient", div, e)
		}
	}
	if e := s.InfIntMulAssign(&ai, &r); !errors.Is(e, ErrRationalServiceOwner) || ai.First.Num != NewInt(7) {
		t.Fatal("foreign InfInt coefficient", e)
	}
	serviceAuditRationalRoots(t, s, &a.First, &a.Second, &ai.First, &ae.infinity, &ar.Raw)
	// Entirely inline source records; separate constructor-owned identities.
	serviceMust(t, s.InfDel(&a))
	serviceMust(t, s.InfIntDel(&ai))
	serviceMust(t, s.InfEpsInfDel(&ae))
	serviceMust(t, s.Del(&ar))
	serviceMust(t, foreign.Del(&r))
	serviceMust(t, s.Close())
	if e := s.InfMulRationalAssign(&a, &r); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("closed owner guard precedes nulled caller fields", e)
	}
	if e := s.InfIntMulAssign(&ai, &r); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	if e := s.InfEpsInfDivRationalAssign(&ae, &r); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	if e := s.DivAssign(&ar, &r); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}
