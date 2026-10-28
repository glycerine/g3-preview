package numeral

// Source: src/util/inf_rational.h; src/util/inf_int_rational.h; src/util/inf_eps_rational.h; src/util/mpq.h; src/util/rational.h.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

// Failed-reference fields below are explicit unread source-prefix tokens, not
// valid numerals. No owned allocation is fabricated and no source count added.
func TestInfCompareSourceUnreadMembers(t *testing.T) {
	s, _ := serviceNew(t)
	a := infApproxRaw(s, 1, 1, 0, 1)
	b := infApproxRaw(s, 2, 1, 0, 1)
	a.Second.Num.Cell = s.Manager.Z.FailedRef
	b.Second.Den.Cell = s.Manager.Z.FailedRef
	if got, e := s.InfEqual(&a, &b); e != nil || got {
		t.Fatal("FIRST mismatch must skip EPS", got, e)
	}
	if got, e := s.InfLess(&a, &b); e != nil || !got {
		t.Fatal("FIRST less must skip EPS", got, e)
	}
	if got, e := s.InfGreater(&b, &a); e != nil || !got {
		t.Fatal("source reversed less", got, e)
	}
	// Restore only deliberately unowned diagnostic tokens before owning cleanup.
	a.Second.Num.Cell = 0
	b.Second.Den.Cell = 0
	serviceMust(t, s.InfDel(&a))
	serviceMust(t, s.InfDel(&b))
	infAuditRoots(t, s, nil, nil)
}
func TestInfCompareSourceNestedUnreadNumeral(t *testing.T) {
	s, _ := serviceNew(t)
	a, b := s.infEpsInfEmpty(), s.infEpsInfEmpty()
	a.infinity.Num = NewInt(-1)
	b.infinity.Num = NewInt(1)
	a.numeral.First.Num.Cell = s.Manager.Z.FailedRef
	b.numeral.Second.Den.Cell = s.Manager.Z.FailedRef
	if got, e := s.InfEpsInfEqual(&a, &b); e != nil || got {
		t.Fatal("infinity mismatch skips nested numeral", got, e)
	}
	if got, e := s.InfEpsInfLess(&a, &b); e != nil || !got {
		t.Fatal("infinity less skips nested numeral", got, e)
	}
	a.numeral.First.Num.Cell = 0
	b.numeral.Second.Den.Cell = 0
	serviceMust(t, s.InfEpsInfDel(&a))
	serviceMust(t, s.InfEpsInfDel(&b))
	infAuditRoots(t, s, nil, nil)
}
func TestInfUnarySourceSequentialMutation(t *testing.T) {
	s, _ := serviceNew(t)
	v := infApproxRaw(s, 7, 0, 3, 1)
	v.Second.Num.Cell = s.Manager.Z.FailedRef
	e := s.InfNeg(&v)
	if !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("reached invalid EPS", e)
	}
	if v.First.Num != NewInt(-7) || v.First.Den != NewInt(0) || v.Second.Num.Val != 3 {
		t.Fatal("FIRST Neg precedes EPS read", v)
	}
	v.Second.Num.Cell = 0
	infAuditRoots(t, s, []*OwnedInfRational{&v}, nil)
	serviceMust(t, s.InfDel(&v))
}
func TestInfCompareSourceRationalReadMask(t *testing.T) {
	for _, synch := range []bool{false, true} {
		m, _ := rationalManager(t, synch)
		a := Rational{Num: NewInt(-3), Den: NewInt(0)}
		b := Rational{Num: NewInt(7), Den: NewInt(1)}
		a.Den.Cell = m.Z.FailedRef
		if got, e := m.Equal(&a, &b); e != nil || got {
			t.Fatal("NUM unequal skips DEN", synch, got, e)
		}
		if got, e := m.Compare(&a, &b); e != nil || got != -1 {
			t.Fatal("source sign shortcut", synch, got, e)
		}
		if e := m.Neg(&a); e != nil || a.Num != NewInt(3) {
			t.Fatal("NUM-only Neg", synch, a, e)
		}
		a.Den.Cell = 0
		qopsAuditLiveOwnership(t, m, []*Int{&a.Num, &a.Den, &b.Num, &b.Den})
		serviceMust(t, m.Del(&a))
		serviceMust(t, m.Del(&b))
	}
}

// Produce an actual ended allocation token, kept only as malformed fixture
// metadata. It carries no ownership and is restored before caller teardown.
func infSourceEndedCell(t *testing.T, z *IntegerManager) NumericCellRef {
	t.Helper()
	var tmp Int
	serviceMust(t, z.SetLimbs(&tmp, []uint32{1, 2, 3, 4}))
	ref := tmp.Cell
	serviceMust(t, z.Del(&tmp))
	return ref
}
func TestInfResetSwapSourceMemberPrefixes(t *testing.T) {
	s, _ := serviceNew(t)
	v := infApproxRaw(s, 7, 1, 3, 1)
	v.Second.Num.Cell = infSourceEndedCell(t, s.Manager.Z)
	if e := s.InfReset(&v); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal(e)
	}
	if v.First != NewRational() {
		t.Fatal("FIRST reset before invalid EPS", v)
	}
	v.Second.Num.Cell = 0
	a := infApproxRaw(s, 2, 1, 3, 1)
	b := infApproxRaw(s, 5, 1, 7, 1)
	b.Second.Num.Cell = s.Manager.Z.FailedRef
	if e := s.InfSwap(&a, &b); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal(e)
	}
	if a.First.Num != NewInt(5) || b.First.Num != NewInt(2) || a.Second.Num != NewInt(3) || b.Second.Num.Val != 7 {
		t.Fatal("FIRST swap before invalid EPS", a, b)
	}
	b.Second.Num.Cell = 0
	infAuditRoots(t, s, []*OwnedInfRational{&v, &a, &b}, nil)
	serviceMust(t, s.InfDel(&v))
	serviceMust(t, s.InfDel(&a))
	serviceMust(t, s.InfDel(&b))
}
func TestInfResetSwapSourceRationalPrefixes(t *testing.T) {
	s, _ := serviceNew(t)
	m := s.Manager
	a := Rational{Num: NewInt(7), Den: NewInt(3)}
	a.Den.Cell = infSourceEndedCell(t, m.Z)
	if e := m.Reset(&a); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal(e)
	}
	if a.Num != NewInt(0) || a.Den.Val != 3 {
		t.Fatal("NUM reset before reached DEN", a)
	}
	a.Den.Cell = 0
	a = Rational{Num: NewInt(2), Den: NewInt(3)}
	b := Rational{Num: NewInt(5), Den: NewInt(7)}
	b.Den.Cell = m.Z.FailedRef
	if e := m.Swap(&a, &b); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal(e)
	}
	if a.Num != NewInt(5) || b.Num != NewInt(2) || a.Den != NewInt(3) || b.Den.Val != 7 {
		t.Fatal("NUM swap before reached DEN", a, b)
	}
	b.Den.Cell = 0
	qopsAuditLiveOwnership(t, m, []*Int{&a.Num, &a.Den, &b.Num, &b.Den})
	serviceMust(t, m.Del(&a))
	serviceMust(t, m.Del(&b))
}
func TestInfCompareSourceRawLocalException(t *testing.T) {
	m, budget := rationalManager(t, true)
	a, b := NewRational(), NewRational()
	serviceMust(t, m.Z.SetLimbs(&a.Num, []uint32{17, 1, 2, 3, 1}))
	a.Den = NewInt(3)
	b.Num = NewInt(7)
	b.Den = NewInt(2)
	a.Den.Cell = m.Z.FailedRef
	before := m.Z.Cells.Stats().Live
	_, e := m.Compare(&a, &b)
	if !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal(e)
	}
	if m.Z.Cells.Stats().Live <= before {
		t.Fatal("source first completed crossproduct lost before end label")
	}
	qopsAuditLiveOwnership(t, m, []*Int{&a.Num, &a.Den, &b.Num, &b.Den})
	a.Den.Cell = 0
	serviceMust(t, m.CleanupRawArithmetic())
	qopsAuditLiveOwnership(t, m, []*Int{&a.Num, &a.Den, &b.Num, &b.Den})
	serviceMust(t, m.Del(&a))
	serviceMust(t, m.Del(&b))
	if budget.Used() == 0 {
		t.Fatal("manager still owns real cold storage")
	}
}

func TestInfUnarySourceRawOwningCopies(t *testing.T) {
	s, _ := serviceNew(t)
	v := infApproxRaw(s, -7, 0, 3, 0)
	neg, e := s.InfUnaryMinus(&v)
	serviceMust(t, e)
	infApproxRawEqual(t, &neg, 7, 0, -3, 0)
	abs, e := s.InfAbs(&v)
	serviceMust(t, e)
	infApproxRawEqual(t, &abs, 7, 0, -3, 0)
	infAuditRoots(t, s, []*OwnedInfRational{&v, &neg, &abs}, nil)
	serviceMust(t, s.InfDel(&neg))
	serviceMust(t, s.InfDel(&abs))
	serviceMust(t, s.InfDel(&v))
	x := OwnedInfIntRational{First: Rational{Num: NewInt(-7), Den: NewInt(0)}, second: 3, ownerID: s.ownerID}
	xn, e := s.InfIntNegCopy(&x)
	serviceMust(t, e)
	xa, e := s.InfIntAbs(&x)
	serviceMust(t, e)
	if xn.First.Num != NewInt(7) || xa.First.Num != NewInt(7) || xn.First.Den != NewInt(0) || xa.second != -3 {
		t.Fatal(xn, xa)
	}
	serviceMust(t, s.InfIntDel(&xn))
	serviceMust(t, s.InfIntDel(&xa))
	serviceMust(t, s.InfIntDel(&x))
	y := s.infEpsInfEmpty()
	y.infinity = Rational{Num: NewInt(-2), Den: NewInt(0)}
	y.numeral = infApproxRaw(s, 3, 0, -5, 0)
	yn, e := s.InfEpsInfUnaryMinus(&y)
	serviceMust(t, e)
	ya, e := s.InfEpsInfAbs(&y)
	serviceMust(t, e)
	if yn.infinity.Num != NewInt(2) || ya.infinity.Num != NewInt(2) || yn.numeral.First.Num != NewInt(-3) || yn.numeral.Second.Num != NewInt(5) {
		t.Fatal(yn, ya)
	}
	serviceMust(t, s.InfEpsInfDel(&yn))
	serviceMust(t, s.InfEpsInfDel(&ya))
	serviceMust(t, s.InfEpsInfDel(&y))
	infAuditRoots(t, s, nil, nil)
}
func TestInfCompareSourceServiceOverloads(t *testing.T) {
	s, _ := serviceNew(t)
	a := OwnedRational{Raw: Rational{Num: NewInt(-7), Den: NewInt(0)}, ownerID: s.ownerID}
	b := OwnedRational{Raw: Rational{Num: NewInt(3), Den: NewInt(0)}, ownerID: s.ownerID}
	if got, e := s.Less(&a, &b); e != nil || !got {
		t.Fatal(got, e)
	}
	if got, e := s.Equal(&a, &b); e != nil || got {
		t.Fatal(got, e)
	}
	serviceMust(t, s.NegAssign(&a))
	if a.Raw.Num != NewInt(7) || a.Raw.Den != NewInt(0) {
		t.Fatal(a)
	}
	neg, e := s.Neg(&a)
	serviceMust(t, e)
	abs, e := s.Abs(&b)
	serviceMust(t, e)
	if neg.Raw.Num != NewInt(-7) || neg.Raw.Den != NewInt(0) || abs.Raw != b.Raw {
		t.Fatal(neg, abs)
	}
	serviceAuditRationalRoots(t, s, &a.Raw, &b.Raw, &neg.Raw, &abs.Raw)
	serviceMust(t, s.Del(&neg))
	serviceMust(t, s.Del(&abs))
	serviceMust(t, s.Reset(&a))
	serviceMust(t, s.Del(&a))
	serviceMust(t, s.Del(&b))
}

// Independent lexicographic expectations exercise every same-type relation and
// operand reversal. Canonical small inputs avoid source representation-equality
// quirks; those are separately covered by the existing raw-shape tests.
func TestInfCompareSourceCompleteRelationalMatrix(t *testing.T) {
	s, _ := serviceNew(t)
	for x := int32(-1); x <= 1; x++ {
		for ex := int32(-1); ex <= 1; ex++ {
			for y := int32(-1); y <= 1; y++ {
				for ey := int32(-1); ey <= 1; ey++ {
					a := infApproxRaw(s, x, 1, ex, 1)
					b := infApproxRaw(s, y, 1, ey, 1)
					lt := x < y || (x == y && ex < ey)
					eq := x == y && ex == ey
					gt := x > y || (x == y && ex > ey)
					check := func(got bool, e error, want bool) {
						t.Helper()
						if e != nil || got != want {
							t.Fatal(x, ex, y, ey, got, want, e)
						}
					}
					g, e := s.InfLess(&a, &b)
					check(g, e, lt)
					g, e = s.InfGreater(&a, &b)
					check(g, e, gt)
					g, e = s.InfEqual(&a, &b)
					check(g, e, eq)
					g, e = s.InfNotEqual(&a, &b)
					check(g, e, !eq)
					g, e = s.InfLessEqual(&a, &b)
					check(g, e, !gt)
					g, e = s.InfGreaterEqual(&a, &b)
					check(g, e, !lt)
					ai := OwnedInfIntRational{First: a.First, second: ex, ownerID: s.ownerID}
					bi := OwnedInfIntRational{First: b.First, second: ey, ownerID: s.ownerID}
					g, e = s.InfIntLess(&ai, &bi)
					check(g, e, lt)
					g, e = s.InfIntGreater(&ai, &bi)
					check(g, e, gt)
					g, e = s.InfIntEqual(&ai, &bi)
					check(g, e, eq)
					g, e = s.InfIntNotEqual(&ai, &bi)
					check(g, e, !eq)
					g, e = s.InfIntLessEqual(&ai, &bi)
					check(g, e, !gt)
					g, e = s.InfIntGreaterEqual(&ai, &bi)
					check(g, e, !lt)
					ae, be := s.infEpsInfEmpty(), s.infEpsInfEmpty()
					ae.infinity.Num = NewInt(x)
					be.infinity.Num = NewInt(y)
					ae.numeral.First.Num = NewInt(ex)
					be.numeral.First.Num = NewInt(ey)
					g, e = s.InfEpsInfLess(&ae, &be)
					check(g, e, lt)
					g, e = s.InfEpsInfGreater(&ae, &be)
					check(g, e, gt)
					g, e = s.InfEpsInfEqual(&ae, &be)
					check(g, e, eq)
					g, e = s.InfEpsInfNotEqual(&ae, &be)
					check(g, e, !eq)
					g, e = s.InfEpsInfLessEqual(&ae, &be)
					check(g, e, !gt)
					g, e = s.InfEpsInfGreaterEqual(&ae, &be)
					check(g, e, !lt)
					// All values contain only inline numerals; scalar aliases do not duplicate
					// storage ownership. Each logical owning token is nevertheless closed.
					serviceMust(t, s.InfDel(&a))
					serviceMust(t, s.InfDel(&b))
					serviceMust(t, s.InfIntDel(&ai))
					serviceMust(t, s.InfIntDel(&bi))
					serviceMust(t, s.InfEpsInfDel(&ae))
					serviceMust(t, s.InfEpsInfDel(&be))
				}
			}
		}
	}
	infAuditRoots(t, s, nil, nil)
}

func TestInfUnarySourceMovePublication(t *testing.T) {
	m, _ := rationalManager(t, false)
	src := Rational{Num: NewInt(7), Den: NewInt(3)}
	dst := NewRational()
	serviceMust(t, m.Z.SetLimbs(&src.Num, []uint32{17, 3, 1}))
	cell := src.Num.Cell
	src.Den.Cell = infSourceEndedCell(t, m.Z)
	if e := m.Move(&dst, &src); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal(e)
	}
	if dst.Num.Cell != cell || src.Num.Cell != 0 || dst.Den != NewInt(1) || src.Den.Val != 3 {
		t.Fatal("NUM transfer before reached DEN", src, dst)
	}
	src.Den.Cell = 0
	qopsAuditLiveOwnership(t, m, []*Int{&src.Num, &src.Den, &dst.Num, &dst.Den})
	serviceMust(t, m.Del(&src))
	serviceMust(t, m.Del(&dst))
}

func TestInfCompareSourceWarmPhysicalGraph(t *testing.T) {
	for _, synch := range []bool{false, true} {
		m, budget := rationalManager(t, synch)
		a, b := NewRational(), NewRational()
		serviceMust(t, m.SetString(&a, "18446744073709551617/3"))
		serviceMust(t, m.SetString(&b, "18446744073709551618/5"))
		for i := 0; i < 3; i++ {
			got, e := m.Compare(&a, &b)
			serviceMust(t, e)
			if got != 1 {
				t.Fatal(got)
			}
		}
		before := numericWarmProfileSnapshot(t, m.Z, budget)
		var e error
		var got int
		allocs := testing.AllocsPerRun(20, func() { got, e = m.Compare(&a, &b) })
		if e != nil || got != 1 {
			t.Fatal(synch, got, e)
		}
		numericWarmProfileAudit(t, m.Z, budget, before, allocs, modularSourceDebug && synch)
		qopsAuditLiveOwnership(t, m, []*Int{&a.Num, &a.Den, &b.Num, &b.Den})
		serviceMust(t, m.Del(&a))
		serviceMust(t, m.Del(&b))
	}
}
func TestInfCompareSourceEndGuardRetainsCursor(t *testing.T) {
	m, _, _ := rawArithmeticGuardOwner(t)
	a, b := NewRational(), NewRational()
	serviceMust(t, m.SetString(&a, "18446744073709551617/3"))
	serviceMust(t, m.SetString(&b, "18446744073709551618/5"))
	r := &m.RawArithmetic
	// Explicitly execute the source crossproduct body up to its end label.
	// There is no numeric callback between the products and source del; this
	// bounded malformed-layout fixture tests that concrete end continuation.
	r.CompareOne, r.CompareTwo = NewRational(), NewRational()
	r.CompareActive = true
	serviceMust(t, m.Z.Mul(&a.Num, &b.Den, &r.CompareOne.Num))
	serviceMust(t, m.Z.Mul(&b.Num, &a.Den, &r.CompareTwo.Num))
	ref := r.CompareTwo.Num.Cell
	original := m.Z.Cell(ref)
	rawArithmeticCorrupt(t, m.Z, &r.CompareTwo.Num)
	e := m.rawCompareEnd()
	if !errors.Is(e, arena.ErrBounds) || !r.compareEnd || r.compareGuard || r.compareCursor != 2 || r.CompareOne.Num.Cell != 0 || r.CompareTwo.Num.Cell != ref {
		t.Fatal(r, e)
	}
	qopsAuditLiveOwnership(t, m, []*Int{&a.Num, &a.Den, &b.Num, &b.Den})
	if _, e := m.Compare(&a, &b); !errors.Is(e, ErrRationalRawArithmeticPending) {
		t.Fatal(e)
	}
	if e := m.CleanupRawArithmetic(); !errors.Is(e, arena.ErrBounds) || r.compareCursor != 2 {
		t.Fatal("owning cleanup stopped at actual object", e, r)
	}
	qopsAuditLiveOwnership(t, m, []*Int{&a.Num, &a.Den, &b.Num, &b.Den})
	c := m.Z.Cell(ref)
	c.Backing.Requested = original.Backing.Requested
	m.Z.Store(ref, c)
	serviceMust(t, m.CleanupRawArithmetic())
	if r.Pending() {
		t.Fatal("retained comparison not consumed")
	}
	qopsAuditLiveOwnership(t, m, []*Int{&a.Num, &a.Den, &b.Num, &b.Den})
	serviceMust(t, m.Del(&a))
	serviceMust(t, m.Del(&b))
}

func TestInfCompareSourceReservationPrefixes(t *testing.T) {
	for _, synch := range []bool{false, true} {
		failed, complete := 0, false
		for prefix := int64(0); prefix < 512; prefix++ {
			m, budget := rationalManager(t, synch)
			a, b := NewRational(), NewRational()
			// SetLimbs constructs the operands without prewarming arithmetic
			// scratch; the first real product must grow beyond its inline buffers.
			aw, bw := make([]uint32, 64), make([]uint32, 33)
			aw[0], aw[63] = 17, 1
			bw[0], bw[32] = 19, 1
			serviceMust(t, m.Z.SetLimbs(&a.Num, aw))
			a.Den = NewInt(3)
			serviceMust(t, m.Z.SetLimbs(&b.Num, bw))
			b.Den = NewInt(5)
			beforeA, beforeB := a, b // read-only scalar snapshot, no owning destructor
			budget.FailAfter(prefix)
			got, e := m.Compare(&a, &b)
			qopsAuditLiveOwnership(t, m, []*Int{&a.Num, &a.Den, &b.Num, &b.Den})
			if a != beforeA || b != beforeB {
				t.Fatal("source comparison mutated operands", synch, prefix)
			}
			if e != nil {
				if !numericPhysicalReservationRefusal(e) {
					t.Fatal(synch, prefix, e)
				}
				failed++
				// With SYNCH the two real RAW locals remain at exact entered stage;
				// persistent mode already owns its member caches. Neither is repaired.
				if synch && m.RawArithmetic.compareEnd {
					t.Fatal("allocation refusal reached source end label", prefix)
				}
			} else if got != 1 {
				t.Fatal(synch, prefix, got)
			}
			budget.FailAfter(-1)
			serviceMust(t, m.CleanupRawArithmetic())
			serviceMust(t, m.Z.CleanupPendingAllocation())
			qopsAuditLiveOwnership(t, m, []*Int{&a.Num, &a.Den, &b.Num, &b.Den})
			serviceMust(t, m.Del(&a))
			serviceMust(t, m.Del(&b))
			serviceMust(t, m.CloseChecked())
			if e == nil {
				complete = true
				t.Logf("synch=%t refused prefixes 0..%d, first success=%d", synch, prefix-1, prefix)
				break
			}
		}
		if failed == 0 || !complete {
			t.Fatal("finite comparison refusal horizon", synch, failed, complete)
		}
	}
}

func TestInfCompareSourceServiceSwap(t *testing.T) {
	s, _ := serviceNew(t)
	a := OwnedRational{Raw: Rational{Num: NewInt(1), Den: NewInt(0)}, ownerID: s.ownerID}
	b := OwnedRational{Raw: Rational{Num: NewInt(2), Den: NewInt(0)}, ownerID: s.ownerID}
	serviceMust(t, s.Swap(&a, &b))
	if a.Raw.Num != NewInt(2) || b.Raw.Num != NewInt(1) || a.Raw.Den != NewInt(0) || b.Raw.Den != NewInt(0) {
		t.Fatal(a, b)
	}
	serviceAuditRationalRoots(t, s, &a.Raw, &b.Raw)
	serviceMust(t, s.Del(&a))
	serviceMust(t, s.Del(&b))
}
func TestInfCompareSourceMixedUnreadMasks(t *testing.T) {
	s, _ := serviceNew(t)
	a := infApproxRaw(s, 1, 1, 0, 1)
	a.Second.Den.Cell = s.Manager.Z.FailedRef
	r := OwnedRational{Raw: Rational{Num: NewInt(1), Den: NewInt(1)}, ownerID: s.ownerID}
	if got, e := s.InfEqualRational(&a, &r); e != nil || !got {
		t.Fatal("EPS.is_zero reads NUMval only", got, e)
	}
	if got, e := s.InfLessRational(&a, &r); e != nil || got {
		t.Fatal("EPS.is_neg reads NUMval only", got, e)
	}
	if got, e := s.InfRationalLess(&r, &a); e != nil || got {
		t.Fatal("EPS.is_pos reads NUMval only", got, e)
	}
	v := s.infEpsInfEmpty()
	v.infinity.Num = NewInt(1)
	v.numeral.ownerID = 0 // explicitly unread nested owner on decisive infinity
	if got, e := s.InfEpsInfLessRational(&v, &r); e != nil || got {
		t.Fatal(got, e)
	}
	if got, e := s.InfEpsInfRationalLess(&r, &v); e != nil || !got {
		t.Fatal(got, e)
	}
	v.numeral.ownerID = s.ownerID
	// Literal source mixed equality compares infinity against rational, then
	// tests finite NUM fields, even though its less relation orders infinity.
	if got, e := s.InfEpsInfEqualRational(&v, &r); e != nil || !got {
		t.Fatal("source mixed equality", got, e)
	}
	a.Second.Den.Cell = 0
	infAuditRoots(t, s, []*OwnedInfRational{&a}, []*OwnedRational{&r})
	serviceMust(t, s.InfDel(&a))
	serviceMust(t, s.InfEpsInfDel(&v))
	serviceMust(t, s.Del(&r))
}
