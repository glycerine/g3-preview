// Source-action/domain tests for mpq.h:191-237,562,666-669 and Inf reference adapters.
// Source: src/util/mpz.h, src/util/mpz.cpp, src/util/mpq.h.
package numeral

import (
	"errors"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func rawProjectionEqual(t *testing.T, z *IntegerManager, a *Int, want *big.Int) {
	t.Helper()
	got := roundPublicationDecode(t, z, a)
	if got.Cmp(want) != 0 {
		t.Fatalf("raw component %s want %s", got, want)
	}
}

func TestRawProjectionCopyShapesAndAliases(t *testing.T) {
	s, _ := serviceNew(t)
	for _, den := range []int32{0, 1, -3, 4} {
		for _, n := range []int64{-7, 0, 2} {
			src := Rational{Num: NewInt(int32(n)), Den: NewInt(den)}
			dst := Rational{Num: NewInt(19), Den: NewInt(0)}
			serviceMust(t, s.Manager.Copy(&dst, &src))
			serviceMust(t, s.Manager.Copy(&dst, &dst))
			if dst.Num.Val != int32(n) || dst.Den.Val != den {
				t.Fatal(dst, src)
			}
			serviceAuditRationalRoots(t, s, &src, &dst)
			serviceMust(t, s.Manager.Del(&dst))
		}
	}
	// Actual BIG copy retains the source requested capacity, not a normalized value.
	src := NewRational()
	dst := NewRational()
	n := new(big.Int).Lsh(big.NewInt(1), 289)
	setIntegerOracle(t, s.Manager.Z, &src.Num, n)
	src.Den = NewInt(0)
	serviceMust(t, s.Manager.Copy(&dst, &src))
	rawProjectionEqual(t, s.Manager.Z, &dst.Num, n)
	sc := s.Manager.Z.Cell(src.Num.Cell)
	dc := s.Manager.Z.Cell(dst.Num.Cell)
	if dc.Capacity != sc.Capacity || dc.Size != sc.Size || dst.Num.Cell == src.Num.Cell {
		t.Fatal(sc, dc)
	}
	serviceAuditRationalRoots(t, s, &src, &dst)
	serviceMust(t, s.Manager.Del(&src))
	serviceMust(t, s.Manager.Del(&dst))
}

func TestRawProjectionQueriesAndUnreadComponents(t *testing.T) {
	s, _ := serviceNew(t)
	dead := NewInt(0)
	setIntegerOracle(t, s.Manager.Z, &dead, new(big.Int).Lsh(big.NewInt(1), 100))
	stale := dead
	serviceMust(t, s.Manager.Z.Del(&dead))
	for _, n := range []int32{-7, -1, 0, 1, 9} {
		a := Rational{Num: NewInt(n), Den: stale}
		sign, e := s.Manager.Sign(&a)
		if e != nil || sign != n {
			t.Fatal(sign, e, n)
		}
		for _, q := range []struct {
			call func(*Rational) (bool, error)
			want bool
		}{
			{s.Manager.IsZero, n == 0}, {s.Manager.IsPos, n > 0}, {s.Manager.IsNeg, n < 0},
			{s.Manager.IsNonPos, n <= 0}, {s.Manager.IsNonNeg, n >= 0}, {s.Manager.IsOne, false}, {s.Manager.IsMinusOne, false},
		} {
			got, e := q.call(&a)
			if e != nil || got != q.want {
				t.Fatal(n, got, e)
			}
		}
		h, e := s.Manager.Hash(&a)
		want := uint32(n) // source inline integer hash, independent expectation
		if e != nil || h != want {
			t.Fatal(h, want, e)
		}
		a.Den = NewInt(0)
	}
	// Pure sign examines the scalar m_val, not the ended magnitude payload.
	a := Rational{Num: stale, Den: NewInt(0)}
	if v, e := s.Manager.Sign(&a); e != nil || v != stale.Val {
		t.Fatal(v, e)
	}
	if _, e := s.Manager.Hash(&a); e == nil {
		t.Fatal("hash must guard an accessed ended magnitude")
	}
	a.Num = NewInt(0)
	serviceAuditRationalRoots(t, s, &a)
}

func TestRawProjectionSelectedCopiesAndResetOrder(t *testing.T) {
	s, _ := serviceNew(t)
	dead := NewInt(0)
	setIntegerOracle(t, s.Manager.Z, &dead, new(big.Int).Lsh(big.NewInt(1), 110))
	stale := dead
	serviceMust(t, s.Manager.Z.Del(&dead))
	a := Rational{Num: NewInt(7), Den: stale}
	out := NewRational()
	i := NewInt(0)
	serviceMust(t, s.Manager.Numerator(&a, &i))
	serviceMust(t, s.Manager.NumeratorRational(&a, &out))
	if i.Val != 7 || out.Num.Val != 7 || out.Den.Val != 1 {
		t.Fatal(i, out)
	}
	a.Num = stale
	a.Den = NewInt(-3)
	serviceMust(t, s.Manager.Denominator(&a, &i))
	serviceMust(t, s.Manager.DenominatorRational(&a, &out))
	if i.Val != -3 || out.Num.Val != -3 || out.Den.Val != 1 {
		t.Fatal(i, out)
	}
	a.Num = NewInt(2)
	a.Den = NewInt(0)
	serviceMust(t, s.Manager.DenominatorRational(&a, &a))
	if a.Num.Val != 0 || a.Den.Val != 1 {
		t.Fatal("source same-output denominator copy precedes reset", a)
	}
	a.Num = NewInt(8)
	a.Den = NewInt(-5)
	serviceMust(t, s.Manager.Numerator(&a, &a.Den))
	if a.Den.Val != 8 {
		t.Fatal(a)
	}
	a.Num = stale
	a.Den = NewInt(0)
	out = Rational{Num: NewInt(13), Den: NewInt(-9)}
	if e := s.Manager.NumeratorRational(&a, &out); e == nil {
		t.Fatal("accessed stale NUM")
	}
	if out.Num.Val != 13 || out.Den.Val != -9 {
		t.Fatal("DEN reset must follow successful NUM copy", out)
	}
	a.Num = NewInt(0)
	serviceAuditRationalRoots(t, s, &a, &out)
}

func TestRawProjectionCopyLateReadAndFailurePrefixes(t *testing.T) {
	s, _ := serviceNew(t)
	dead := NewInt(0)
	setIntegerOracle(t, s.Manager.Z, &dead, new(big.Int).Lsh(big.NewInt(1), 110))
	stale := dead
	serviceMust(t, s.Manager.Z.Del(&dead))
	src := Rational{Num: NewInt(7), Den: stale}
	dst := Rational{Num: NewInt(-3), Den: NewInt(19)}
	if e := s.Manager.Copy(&dst, &src); e == nil {
		t.Fatal("late stale DEN")
	}
	if dst.Num.Val != 7 || dst.Den.Val != 19 {
		t.Fatal("NUM copy precedes DEN read", dst)
	}
	src.Den = NewInt(0)
	serviceAuditRationalRoots(t, s, &src, &dst)
	// Each real reservation prefix gets a fresh owner. NUM is infallible inline;
	// replacement of BIG DEN publishes its sign before physical allocation.
	for fail := int64(0); fail < 256; fail++ {
		ss, bb := serviceNew(t)
		a := Rational{Num: NewInt(17), Den: NewInt(0)}
		d := Rational{Num: NewInt(-9), Den: NewInt(0)}
		magnitude := new(big.Int).Lsh(big.NewInt(1), 300)
		setIntegerOracle(t, ss.Manager.Z, &a.Den, magnitude)
		bb.FailAfter(fail)
		err := ss.Manager.Copy(&d, &a)
		serviceAuditRationalRoots(t, ss, &a, &d)
		if d.Num.Val != 17 {
			t.Fatal("completed NUM", fail, d.Num)
		}
		if err != nil {
			if !numericPhysicalReservationRefusal(err) {
				t.Fatal(fail, err)
			}
			if !errors.Is(err, arena.ErrInjectedFailure) && !ss.Manager.Z.Memory.State().OutOfMemory {
				t.Fatal("physical Memout requires real OOM state", fail, err)
			}
			if d.Den.Val != 1 || d.Den.Cell != ss.Manager.Z.FailedNullRef {
				t.Fatal("source sign/failed allocation prefix", fail, d.Den)
			}
		} else {
			rawProjectionEqual(t, ss.Manager.Z, &d.Den, magnitude)
		}
		bb.FailAfter(-1)
		serviceMust(t, ss.Manager.Z.CleanupPendingAllocation())
		serviceMust(t, ss.Manager.Del(&a))
		serviceMust(t, ss.Manager.Del(&d))
		serviceMust(t, ss.Close())
		if bb.Used() != 0 {
			t.Fatal("cleanup", fail, bb.Used())
		}
		if err == nil {
			break
		}
		if fail == 255 {
			t.Fatal("sweep never completed")
		}
	}
}

func TestRawProjectionServiceAndInfOwningReferences(t *testing.T) {
	s, _ := serviceNew(t)
	a, e := s.NewValue()
	serviceMust(t, e)
	a.Raw = Rational{Num: NewInt(-7), Den: NewInt(0)}
	d, e := s.NewValue()
	serviceMust(t, e)
	serviceMust(t, s.Copy(&d, &a))
	for _, call := range []func(*OwnedRational) (OwnedRational, error){s.ToRational, s.GetRational, s.Numerator, s.Denominator, s.GetInfinitesimal} {
		v, e := call(&a)
		serviceMust(t, e)
		serviceAuditRationalRoots(t, s, &a.Raw, &d.Raw, &v.Raw)
		serviceMust(t, s.Del(&v))
	}
	if got, e := s.IsRational(&a); e != nil || !got {
		t.Fatal(got, e)
	}
	x, e := s.InfIntNewValue()
	serviceMust(t, e)
	x.First = Rational{Num: NewInt(2), Den: NewInt(0)}
	x.second = 5
	y, e := s.NewInfValue()
	serviceMust(t, e)
	y.First = x.First
	y.Second = Rational{Num: NewInt(-3), Den: NewInt(0)}
	z, e := s.InfEpsInfNewValue()
	serviceMust(t, e)
	z.infinity = Rational{Num: NewInt(1), Den: NewInt(0)}
	z.numeral.First = y.First
	z.numeral.Second = y.Second
	xx, e := s.InfIntClone(&x)
	serviceMust(t, e)
	yy, e := s.InfClone(&y)
	serviceMust(t, e)
	zz, e := s.InfEpsInfClone(&z)
	serviceMust(t, e)
	for _, call := range []func() (OwnedRational, error){func() (OwnedRational, error) { return s.InfIntFirst(&x) }, func() (OwnedRational, error) { return s.InfIntInfinitesimal(&x) }, func() (OwnedRational, error) { return s.InfGetFirst(&y) }, func() (OwnedRational, error) { return s.InfGetInfinitesimal(&y) }, func() (OwnedRational, error) { return s.InfEpsInfGetInfinity(&z) }, func() (OwnedRational, error) { return s.InfEpsInfGetRational(&z) }, func() (OwnedRational, error) { return s.InfEpsInfGetInfinitesimal(&z) }} {
		v, e := call()
		serviceMust(t, e)
		serviceMust(t, s.Del(&v))
	}
	n, e := s.InfEpsInfGetNumeral(&z)
	serviceMust(t, e)
	serviceMust(t, s.InfDel(&n))
	for _, q := range []struct {
		call func() (uint32, error)
		want uint32
	}{
		{func() (uint32, error) { return s.Hash(&a) }, uint32(0xfffffff9)},
		{func() (uint32, error) { return s.InfIntHash(&x) }, 2 ^ (5 + 1)},
		{func() (uint32, error) { return s.InfHash(&y) }, 2 ^ (uint32(0xfffffffd) + 1)},
		{func() (uint32, error) { return s.InfEpsInfHash(&z) }, 1 ^ (2 ^ (uint32(0xfffffffd) + 1))},
	} {
		h, e := q.call()
		if e != nil || h != q.want {
			t.Fatal(h, q.want, e)
		}
	}
	serviceAuditRationalRoots(t, s, &a.Raw, &d.Raw, &x.First, &xx.First, &y.First, &y.Second, &yy.First, &yy.Second, &z.infinity, &z.numeral.First, &z.numeral.Second, &zz.infinity, &zz.numeral.First, &zz.numeral.Second)
	serviceMust(t, s.InfIntDel(&x))
	serviceMust(t, s.InfIntDel(&xx))
	serviceMust(t, s.InfDel(&y))
	serviceMust(t, s.InfDel(&yy))
	serviceMust(t, s.InfEpsInfDel(&z))
	serviceMust(t, s.InfEpsInfDel(&zz))
	serviceMust(t, s.Del(&a))
	serviceMust(t, s.Del(&d))
}

func TestRawProjectionInfExtractionSourceDebugBoundary(t *testing.T) {
	disableInfProjectionAssertions(t)
	s, _ := serviceNew(t)
	x, e := s.InfIntNewValue()
	serviceMust(t, e)
	x.First = Rational{Num: NewInt(7), Den: NewInt(0)}
	x.second = 4
	y, e := s.NewInfValue()
	serviceMust(t, e)
	y.First = x.First
	y.Second = Rational{Num: NewInt(3), Den: NewInt(0)}
	for _, den := range []bool{false, true} {
		var out OwnedInfIntRational
		if den {
			out, e = s.InfIntDenominator(&x)
		} else {
			out, e = s.InfIntNumerator(&x)
		}
		serviceMust(t, e)
		want := int32(7)
		if den {
			want = 0
		}
		if out.First.Num.Val != want || out.First.Den.Val != 1 || out.second != 0 {
			t.Fatal(out)
		}
		serviceMust(t, s.InfIntDel(&out))
		var r OwnedInfRational
		if den {
			r, e = s.InfDenominator(&y)
		} else {
			r, e = s.InfNumerator(&y)
		}
		serviceMust(t, e)
		want = int32(7)
		if den {
			want = 0
		}
		if r.First.Num.Val != want || r.First.Den.Val != 1 || r.Second.Num.Val != 0 {
			t.Fatal(r)
		}
		serviceMust(t, s.InfDel(&r))
	}
	serviceAuditRationalRoots(t, s, &x.First, &y.First, &y.Second)
	serviceMust(t, s.InfIntDel(&x))
	serviceMust(t, s.InfDel(&y))
}

func TestRawProjectionInfQueriesAndShortCircuits(t *testing.T) {
	s, _ := serviceNew(t)
	a, e := s.InfIntNewValue()
	serviceMust(t, e)
	b, e := s.NewInfValue()
	serviceMust(t, e)
	c, e := s.InfEpsInfNewValue()
	serviceMust(t, e)
	for _, first := range []int32{-3, -1, 0, 1, 7} {
		for _, eps := range []int32{-4, 0, 5} {
			for _, infty := range []int32{-1, 0, 1} {
				a.First = Rational{Num: NewInt(first), Den: NewInt(1)}
				a.second = eps
				b.First = a.First
				b.Second = Rational{Num: NewInt(eps), Den: NewInt(0)}
				c.infinity = Rational{Num: NewInt(infty), Den: NewInt(0)}
				c.numeral.First = b.First
				c.numeral.Second = b.Second
				sign := first
				if sign == 0 {
					sign = eps
				}
				fullsign := infty
				if fullsign == 0 {
					fullsign = sign
				}
				for _, q := range []struct {
					call func() (bool, error)
					want bool
				}{
					{func() (bool, error) { return s.InfIntIsZero(&a) }, first == 0 && eps == 0},
					{func() (bool, error) { return s.InfIntIsOne(&a) }, first == 1 && eps == 0},
					{func() (bool, error) { return s.InfIntIsMinusOne(&a) }, first == -1 && eps == 0},
					{func() (bool, error) { return s.InfIntIsPos(&a) }, sign > 0}, {func() (bool, error) { return s.InfIntIsNeg(&a) }, sign < 0},
					{func() (bool, error) { return s.InfIntIsNonPos(&a) }, sign <= 0}, {func() (bool, error) { return s.InfIntIsNonNeg(&a) }, sign >= 0},
					{func() (bool, error) { return s.InfIsZero(&b) }, first == 0 && eps == 0},
					{func() (bool, error) { return s.InfIsOne(&b) }, first == 1 && eps == 0},
					{func() (bool, error) { return s.InfIsMinusOne(&b) }, first == -1 && eps == 0},
					{func() (bool, error) { return s.InfIsPos(&b) }, sign > 0}, {func() (bool, error) { return s.InfIsNeg(&b) }, sign < 0},
					{func() (bool, error) { return s.InfIsNonPos(&b) }, sign <= 0}, {func() (bool, error) { return s.InfIsNonNeg(&b) }, sign >= 0},
					{func() (bool, error) { return s.InfEpsInfIsZero(&c) }, infty == 0 && first == 0 && eps == 0},
					{func() (bool, error) { return s.InfEpsInfIsOne(&c) }, infty == 0 && first == 1 && eps == 0},
					{func() (bool, error) { return s.InfEpsInfIsMinusOne(&c) }, infty == 0 && first == -1 && eps == 0},
					{func() (bool, error) { return s.InfEpsInfIsPos(&c) }, fullsign > 0}, {func() (bool, error) { return s.InfEpsInfIsNeg(&c) }, fullsign < 0},
					{func() (bool, error) { return s.InfEpsInfIsNonPos(&c) }, fullsign <= 0}, {func() (bool, error) { return s.InfEpsInfIsNonNeg(&c) }, fullsign >= 0},
					{func() (bool, error) { return s.InfEpsInfIsFinite(&c) }, infty == 0},
				} {
					got, e := q.call()
					if e != nil || got != q.want {
						t.Fatal(first, eps, infty, got, q.want, e)
					}
				}
			}
		}
	}
	dead := NewInt(0)
	setIntegerOracle(t, s.Manager.Z, &dead, new(big.Int).Lsh(big.NewInt(1), 95))
	stale := dead
	serviceMust(t, s.Manager.Z.Del(&dead))
	b.First = Rational{Num: stale, Den: stale}
	b.Second = Rational{Num: NewInt(0), Den: stale}
	if yes, e := s.InfIsPos(&b); e != nil || !yes {
		t.Fatal("sign uses scalar m_val", yes, e)
	}
	c.infinity = Rational{Num: NewInt(1), Den: stale}
	c.numeral.First = b.First
	c.numeral.Second = b.Second
	if yes, e := s.InfEpsInfIsPos(&c); e != nil || !yes {
		t.Fatal(yes, e)
	}
	v, e := s.InfEpsInfGetInfinity(&c)
	if e == nil {
		s.Del(&v)
		t.Fatal("selected DEN must be read")
	}
	c.infinity.Den = NewInt(0)
	v, e = s.InfEpsInfGetInfinity(&c)
	serviceMust(t, e)
	serviceMust(t, s.Del(&v))
	// Source reference projections select a subobject without inspecting FIRST.
	b.Second.Den = NewInt(0)
	selected, e := s.InfGetInfinitesimal(&b)
	serviceMust(t, e)
	serviceMust(t, s.Del(&selected))
	a.First = Rational{Num: stale, Den: stale}
	a.second = 4
	selected, e = s.InfIntInfinitesimal(&a)
	serviceMust(t, e)
	if selected.Raw.Num.Val != 4 || selected.Raw.Den.Val != 1 {
		t.Fatal(selected)
	}
	serviceMust(t, s.Del(&selected))
	a.First = NewRational()
	raw, e := s.NewValue()
	serviceMust(t, e)
	raw.Raw = Rational{Num: stale, Den: stale}
	if yes, e := s.IsRational(&raw); e != nil || !yes {
		t.Fatal(yes, e)
	}
	selected, e = s.GetInfinitesimal(&raw)
	serviceMust(t, e)
	serviceMust(t, s.Del(&selected))
	raw.Raw = NewRational()
	serviceMust(t, s.Del(&raw))
	b.First = NewRational()
	b.Second = NewRational()
	c.numeral.First = NewRational()
	c.numeral.Second = NewRational()
	serviceAuditRationalRoots(t, s, &a.First, &b.First, &b.Second, &c.infinity, &c.numeral.First, &c.numeral.Second)
	serviceMust(t, s.InfIntDel(&a))
	serviceMust(t, s.InfDel(&b))
	serviceMust(t, s.InfEpsInfDel(&c))
}

func TestRawProjectionOwningBigCopiesAndMemberFailureOrder(t *testing.T) {
	// Reference adapters return independent storage; source ref getters themselves
	// do not allocate. This owning Go boundary deliberately executes real Copy.
	s, _ := serviceNew(t)
	x, e := s.InfEpsInfNewValue()
	serviceMust(t, e)
	n := new(big.Int).Lsh(big.NewInt(1), 207)
	for _, r := range []*Rational{&x.infinity, &x.numeral.First, &x.numeral.Second} {
		setIntegerOracle(t, s.Manager.Z, &r.Num, n)
		r.Den = NewInt(0)
	}
	out, e := s.InfEpsInfClone(&x)
	serviceMust(t, e)
	for _, pair := range []struct{ a, b *Rational }{{&x.infinity, &out.infinity}, {&x.numeral.First, &out.numeral.First}, {&x.numeral.Second, &out.numeral.Second}} {
		if pair.a.Num.Cell == pair.b.Num.Cell {
			t.Fatal("shallow owning copy")
		}
		rawProjectionEqual(t, s.Manager.Z, &pair.b.Num, n)
	}
	serviceMust(t, s.InfEpsInfDel(&x))
	serviceAuditRationalRoots(t, s, &out.infinity, &out.numeral.First, &out.numeral.Second)
	serviceMust(t, s.InfEpsInfDel(&out))
	// Late nested source invalidity is guarded where the component is copied,
	// after infinity and nested FIRST have actually published.
	a, e := s.InfEpsInfNewValue()
	serviceMust(t, e)
	d, e := s.InfEpsInfNewValue()
	serviceMust(t, e)
	a.infinity = Rational{Num: NewInt(2), Den: NewInt(0)}
	a.numeral.First = Rational{Num: NewInt(3), Den: NewInt(0)}
	dead := NewInt(0)
	setIntegerOracle(t, s.Manager.Z, &dead, n)
	stale := dead
	serviceMust(t, s.Manager.Z.Del(&dead))
	a.numeral.Second.Num = stale
	if e = s.InfEpsInfCopy(&d, &a); e == nil {
		t.Fatal("actual late component read")
	}
	if d.infinity.Num.Val != 2 || d.infinity.Den.Val != 0 || d.numeral.First.Num.Val != 3 || d.numeral.First.Den.Val != 0 || d.numeral.Second.Num.Val != 0 {
		t.Fatal(d)
	}
	a.numeral.Second.Num = NewInt(0)
	serviceAuditRationalRoots(t, s, &a.infinity, &a.numeral.First, &a.numeral.Second, &d.infinity, &d.numeral.First, &d.numeral.Second)
	serviceMust(t, s.InfEpsInfDel(&a))
	serviceMust(t, s.InfEpsInfDel(&d))
	for fail := int64(0); fail < 256; fail++ {
		ss, bb := serviceNew(t)
		a, e := ss.InfEpsInfNewValue()
		serviceMust(t, e)
		for _, r := range []*Rational{&a.infinity, &a.numeral.First, &a.numeral.Second} {
			setIntegerOracle(t, ss.Manager.Z, &r.Num, n)
			r.Den = NewInt(0)
		}
		bb.FailAfter(fail)
		clone, err := ss.InfEpsInfClone(&a)
		roots := []*Rational{&a.infinity, &a.numeral.First, &a.numeral.Second}
		if err == nil {
			roots = append(roots, &clone.infinity, &clone.numeral.First, &clone.numeral.Second)
		} else if clone.ownerID != 0 || !numericPhysicalReservationRefusal(err) {
			t.Fatal(fail, clone, err)
		}
		serviceAuditRationalRoots(t, ss, roots...)
		if err != nil && !errors.Is(err, arena.ErrInjectedFailure) && !ss.Manager.Z.Memory.State().OutOfMemory {
			t.Fatal("physical Memout requires real OOM state", fail, err)
		}
		bb.FailAfter(-1)
		serviceMust(t, ss.Manager.Z.CleanupPendingAllocation())
		serviceMust(t, ss.InfEpsInfDel(&a))
		if err == nil {
			serviceMust(t, ss.InfEpsInfDel(&clone))
		}
		serviceMust(t, ss.Close())
		if bb.Used() != 0 {
			t.Fatal(fail, bb.Used())
		}
		if err == nil {
			break
		}
		if fail == 255 {
			t.Fatal("no successful clone")
		}
	}
}

func TestRawProjectionWarmPhysicalGraphAndIdentity(t *testing.T) {
	s, b := serviceNew(t)
	r, e := s.NewValue()
	serviceMust(t, e)
	r.Raw = Rational{Num: NewInt(1), Den: NewInt(0)}
	x, e := s.InfIntNewValue()
	serviceMust(t, e)
	x.First = r.Raw
	y, e := s.NewInfValue()
	serviceMust(t, e)
	y.First = r.Raw
	y.Second.Den = NewInt(0)
	z, e := s.InfEpsInfNewValue()
	serviceMust(t, e)
	z.infinity.Den = NewInt(0)
	z.numeral.First = y.First
	z.numeral.Second = y.Second
	before := numericWarmProfileSnapshot(t, s.Manager.Z, b)
	var err error
	allocs := testing.AllocsPerRun(100, func() {
		if _, e := s.IsPos(&r); e != nil {
			err = e
		}
		if _, e := s.IsNeg(&r); e != nil {
			err = e
		}
		if _, e := s.IsZero(&r); e != nil {
			err = e
		}
		if _, e := s.IsOne(&r); e != nil {
			err = e
		}
		if _, e := s.IsMinusOne(&r); e != nil {
			err = e
		}
		if _, e := s.IsNonPos(&r); e != nil {
			err = e
		}
		if _, e := s.IsNonNeg(&r); e != nil {
			err = e
		}
		if _, e := s.Hash(&r); e != nil {
			err = e
		}
		if _, e := s.InfIntIsPos(&x); e != nil {
			err = e
		}
		if _, e := s.InfIntIsNeg(&x); e != nil {
			err = e
		}
		if _, e := s.InfIntIsZero(&x); e != nil {
			err = e
		}
		if _, e := s.InfIntIsOne(&x); e != nil {
			err = e
		}
		if _, e := s.InfIntIsMinusOne(&x); e != nil {
			err = e
		}
		if _, e := s.InfIntIsNonPos(&x); e != nil {
			err = e
		}
		if _, e := s.InfIntIsNonNeg(&x); e != nil {
			err = e
		}
		if _, e := s.InfIntHash(&x); e != nil {
			err = e
		}
		if _, e := s.InfIsPos(&y); e != nil {
			err = e
		}
		if _, e := s.InfIsNeg(&y); e != nil {
			err = e
		}
		if _, e := s.InfIsZero(&y); e != nil {
			err = e
		}
		if _, e := s.InfIsOne(&y); e != nil {
			err = e
		}
		if _, e := s.InfIsMinusOne(&y); e != nil {
			err = e
		}
		if _, e := s.InfIsNonPos(&y); e != nil {
			err = e
		}
		if _, e := s.InfIsNonNeg(&y); e != nil {
			err = e
		}
		if _, e := s.InfHash(&y); e != nil {
			err = e
		}
		if _, e := s.InfEpsInfIsPos(&z); e != nil {
			err = e
		}
		if _, e := s.InfEpsInfIsNeg(&z); e != nil {
			err = e
		}
		if _, e := s.InfEpsInfIsZero(&z); e != nil {
			err = e
		}
		if _, e := s.InfEpsInfIsOne(&z); e != nil {
			err = e
		}
		if _, e := s.InfEpsInfIsMinusOne(&z); e != nil {
			err = e
		}
		if _, e := s.InfEpsInfIsNonPos(&z); e != nil {
			err = e
		}
		if _, e := s.InfEpsInfIsNonNeg(&z); e != nil {
			err = e
		}
		if _, e := s.InfEpsInfIsFinite(&z); e != nil {
			err = e
		}
		if _, e := s.InfEpsInfHash(&z); e != nil {
			err = e
		}
		v, e := s.GetRational(&r)
		if e != nil {
			err = e
			return
		}
		err = s.Del(&v)
	})
	serviceMust(t, err)
	serviceAuditRationalRoots(t, s, &r.Raw, &x.First, &y.First, &y.Second, &z.infinity, &z.numeral.First, &z.numeral.Second)
	numericWarmProfileAudit(t, s.Manager.Z, b, before, allocs, false)
	bad := r
	bad.ownerID++
	if _, e := s.Hash(&bad); !errors.Is(e, ErrRationalServiceOwner) {
		t.Fatal(e)
	}
	serviceMust(t, s.Del(&r))
	serviceMust(t, s.InfIntDel(&x))
	serviceMust(t, s.InfDel(&y))
	serviceMust(t, s.InfEpsInfDel(&z))
	if _, e := s.Hash(&r); !errors.Is(e, ErrRationalServiceOwner) {
		t.Fatal(e)
	}
}

func TestRawProjectionNestedOwnerAtActualBranch(t *testing.T) {
	s, _ := serviceNew(t)
	v, e := s.InfEpsInfNewValue()
	serviceMust(t, e)
	v.infinity = Rational{Num: NewInt(1), Den: NewInt(0)}
	saved := v.numeral.ownerID
	v.numeral.ownerID++ // borrowed malformed provenance fixture
	if got, e := s.InfEpsInfIsPos(&v); e != nil || !got {
		t.Fatal("unread nested owner", got, e)
	}
	out, e := s.InfEpsInfGetInfinity(&v)
	serviceMust(t, e)
	serviceMust(t, s.Del(&out))
	if _, e := s.InfEpsInfGetRational(&v); !errors.Is(e, ErrRationalServiceOwner) {
		t.Fatal("accessed nested owner", e)
	}
	v.infinity.Num = NewInt(0)
	if _, e := s.InfEpsInfIsPos(&v); !errors.Is(e, ErrRationalServiceOwner) {
		t.Fatal("reached nested sign", e)
	}
	v.numeral.ownerID = saved
	serviceAuditRationalRoots(t, s, &v.infinity, &v.numeral.First, &v.numeral.Second)
	serviceMust(t, s.InfEpsInfDel(&v))
}

func TestRawProjectionBothPoliciesAllComponentAliases(t *testing.T) {
	for _, synch := range []bool{false, true} {
		m, _ := rationalManager(t, synch)
		source := Rational{Num: NewInt(31), Den: NewInt(0)}
		destination := NewRational()
		serviceMust(t, m.Copy(&destination, &source))
		if destination.Num.Val != 31 || destination.Den.Val != 0 {
			t.Fatal(synch, destination)
		}
		qopsAuditLiveOwnership(t, m, []*Int{&source.Num, &source.Den, &destination.Num, &destination.Den})
		for _, den := range []int32{-3, 0, 1, 4} {
			for _, numerator := range []bool{false, true} {
				for _, outNum := range []bool{false, true} {
					a := Rational{Num: NewInt(-7), Den: NewInt(den)}
					out := &a.Den
					if outNum {
						out = &a.Num
					}
					expected := den
					if numerator {
						expected = -7
					}
					var err error
					if numerator {
						err = m.Numerator(&a, out)
					} else {
						err = m.Denominator(&a, out)
					}
					serviceMust(t, err)
					if out.Val != expected {
						t.Fatal(synch, den, numerator, outNum, a)
					}
					qopsAuditLiveOwnership(t, m, []*Int{&a.Num, &a.Den})
				}
				a := Rational{Num: NewInt(-7), Den: NewInt(den)}
				expected := den
				if numerator {
					expected = -7
				}
				var err error
				if numerator {
					err = m.NumeratorRational(&a, &a)
				} else {
					err = m.DenominatorRational(&a, &a)
				}
				serviceMust(t, err)
				if a.Num.Val != expected || a.Den.Val != 1 {
					t.Fatal(synch, den, numerator, a)
				}
				qopsAuditLiveOwnership(t, m, []*Int{&a.Num, &a.Den})
			}
		}
		dead := NewInt(0)
		setIntegerOracle(t, m.Z, &dead, new(big.Int).Lsh(big.NewInt(1), 99))
		stale := dead
		serviceMust(t, m.Z.Del(&dead))
		a := Rational{Num: NewInt(7), Den: NewInt(0)}
		out := Rational{Num: NewInt(-9), Den: stale}
		if e := m.NumeratorRational(&a, &out); e == nil {
			t.Fatal("actual late DEN reset guard")
		}
		if out.Num.Val != 7 {
			t.Fatal("selected copy precedes old DEN destruction", out)
		}
		out.Den = NewInt(0)
		qopsAuditLiveOwnership(t, m, []*Int{&a.Num, &a.Den, &out.Num, &out.Den})
	}
}
