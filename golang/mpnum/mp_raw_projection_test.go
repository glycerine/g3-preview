// Licensed under the MIT license.
// Source-action/domain tests for mpq.h191-237,562,666-669 and Inf reference adapters.
// Source: src/util/mpz.h, src/util/mpz.cpp, src/util/mpq.h.
package mpnum

import (
	"errors"
	"math/big"
	"testing"

	"github.com/glycerine/g3/golang/arena"
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
			src := Rational{num: NewInt(int32(n)), den: NewInt(den)}
			dst := Rational{num: NewInt(19), den: NewInt(0)}
			serviceMust(t, s.manager.Copy(&dst, &src))
			serviceMust(t, s.manager.Copy(&dst, &dst))
			if dst.num.val != int32(n) || dst.den.val != den {
				t.Fatal(dst, src)
			}
			serviceAuditRationalRoots(t, s, &src, &dst)
			serviceMust(t, s.manager.Del(&dst))
		}
	}
	// Actual BIG copy retains the source requested capacity, not a normalized value.
	src := NewRational()
	dst := NewRational()
	n := new(big.Int).Lsh(big.NewInt(1), 289)
	setIntegerOracle(t, s.manager.z, &src.num, n)
	src.den = NewInt(0)
	serviceMust(t, s.manager.Copy(&dst, &src))
	rawProjectionEqual(t, s.manager.z, &dst.num, n)
	sc := s.manager.z.cell(src.num.cell)
	dc := s.manager.z.cell(dst.num.cell)
	if dc.capacity != sc.capacity || dc.size != sc.size || dst.num.cell == src.num.cell {
		t.Fatal(sc, dc)
	}
	serviceAuditRationalRoots(t, s, &src, &dst)
	serviceMust(t, s.manager.Del(&src))
	serviceMust(t, s.manager.Del(&dst))
}

func TestRawProjectionQueriesAndUnreadComponents(t *testing.T) {
	s, _ := serviceNew(t)
	dead := NewInt(0)
	setIntegerOracle(t, s.manager.z, &dead, new(big.Int).Lsh(big.NewInt(1), 100))
	stale := dead
	serviceMust(t, s.manager.z.Del(&dead))
	for _, n := range []int32{-7, -1, 0, 1, 9} {
		a := Rational{num: NewInt(n), den: stale}
		sign, e := s.manager.Sign(&a)
		if e != nil || sign != n {
			t.Fatal(sign, e, n)
		}
		for _, q := range []struct {
			call func(*Rational) (bool, error)
			want bool
		}{
			{s.manager.IsZero, n == 0}, {s.manager.IsPos, n > 0}, {s.manager.IsNeg, n < 0},
			{s.manager.IsNonPos, n <= 0}, {s.manager.IsNonNeg, n >= 0}, {s.manager.IsOne, false}, {s.manager.IsMinusOne, false},
		} {
			got, e := q.call(&a)
			if e != nil || got != q.want {
				t.Fatal(n, got, e)
			}
		}
		h, e := s.manager.Hash(&a)
		want := uint32(n) // source inline integer hash, independent expectation
		if e != nil || h != want {
			t.Fatal(h, want, e)
		}
		a.den = NewInt(0)
	}
	// Pure sign examines the scalar m_val, not the ended magnitude payload.
	a := Rational{num: stale, den: NewInt(0)}
	if v, e := s.manager.Sign(&a); e != nil || v != stale.val {
		t.Fatal(v, e)
	}
	if _, e := s.manager.Hash(&a); e == nil {
		t.Fatal("hash must guard an accessed ended magnitude")
	}
	a.num = NewInt(0)
	serviceAuditRationalRoots(t, s, &a)
}

func TestRawProjectionSelectedCopiesAndResetOrder(t *testing.T) {
	s, _ := serviceNew(t)
	dead := NewInt(0)
	setIntegerOracle(t, s.manager.z, &dead, new(big.Int).Lsh(big.NewInt(1), 110))
	stale := dead
	serviceMust(t, s.manager.z.Del(&dead))
	a := Rational{num: NewInt(7), den: stale}
	out := NewRational()
	i := NewInt(0)
	serviceMust(t, s.manager.Numerator(&a, &i))
	serviceMust(t, s.manager.NumeratorRational(&a, &out))
	if i.val != 7 || out.num.val != 7 || out.den.val != 1 {
		t.Fatal(i, out)
	}
	a.num = stale
	a.den = NewInt(-3)
	serviceMust(t, s.manager.Denominator(&a, &i))
	serviceMust(t, s.manager.DenominatorRational(&a, &out))
	if i.val != -3 || out.num.val != -3 || out.den.val != 1 {
		t.Fatal(i, out)
	}
	a.num = NewInt(2)
	a.den = NewInt(0)
	serviceMust(t, s.manager.DenominatorRational(&a, &a))
	if a.num.val != 0 || a.den.val != 1 {
		t.Fatal("source same-output denominator copy precedes reset", a)
	}
	a.num = NewInt(8)
	a.den = NewInt(-5)
	serviceMust(t, s.manager.Numerator(&a, &a.den))
	if a.den.val != 8 {
		t.Fatal(a)
	}
	a.num = stale
	a.den = NewInt(0)
	out = Rational{num: NewInt(13), den: NewInt(-9)}
	if e := s.manager.NumeratorRational(&a, &out); e == nil {
		t.Fatal("accessed stale NUM")
	}
	if out.num.val != 13 || out.den.val != -9 {
		t.Fatal("DEN reset must follow successful NUM copy", out)
	}
	a.num = NewInt(0)
	serviceAuditRationalRoots(t, s, &a, &out)
}

func TestRawProjectionCopyLateReadAndFailurePrefixes(t *testing.T) {
	s, _ := serviceNew(t)
	dead := NewInt(0)
	setIntegerOracle(t, s.manager.z, &dead, new(big.Int).Lsh(big.NewInt(1), 110))
	stale := dead
	serviceMust(t, s.manager.z.Del(&dead))
	src := Rational{num: NewInt(7), den: stale}
	dst := Rational{num: NewInt(-3), den: NewInt(19)}
	if e := s.manager.Copy(&dst, &src); e == nil {
		t.Fatal("late stale DEN")
	}
	if dst.num.val != 7 || dst.den.val != 19 {
		t.Fatal("NUM copy precedes DEN read", dst)
	}
	src.den = NewInt(0)
	serviceAuditRationalRoots(t, s, &src, &dst)
	// Each real reservation prefix gets a fresh owner. NUM is infallible inline;
	// replacement of BIG DEN publishes its sign before physical allocation.
	for fail := int64(0); fail < 256; fail++ {
		ss, bb := serviceNew(t)
		a := Rational{num: NewInt(17), den: NewInt(0)}
		d := Rational{num: NewInt(-9), den: NewInt(0)}
		magnitude := new(big.Int).Lsh(big.NewInt(1), 300)
		setIntegerOracle(t, ss.manager.z, &a.den, magnitude)
		bb.FailAfter(fail)
		err := ss.manager.Copy(&d, &a)
		serviceAuditRationalRoots(t, ss, &a, &d)
		if d.num.val != 17 {
			t.Fatal("completed NUM", fail, d.num)
		}
		if err != nil {
			if !numericPhysicalReservationRefusal(err) {
				t.Fatal(fail, err)
			}
			if !errors.Is(err, arena.ErrInjectedFailure) && !ss.manager.z.memory.State().OutOfMemory {
				t.Fatal("physical Memout requires real OOM state", fail, err)
			}
			if d.den.val != 1 || d.den.cell != ss.manager.z.failedNullRef {
				t.Fatal("source sign/failed allocation prefix", fail, d.den)
			}
		} else {
			rawProjectionEqual(t, ss.manager.z, &d.den, magnitude)
		}
		bb.FailAfter(-1)
		serviceMust(t, ss.manager.z.CleanupPendingAllocation())
		serviceMust(t, ss.manager.Del(&a))
		serviceMust(t, ss.manager.Del(&d))
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
	a.raw = Rational{num: NewInt(-7), den: NewInt(0)}
	d, e := s.NewValue()
	serviceMust(t, e)
	serviceMust(t, s.Copy(&d, &a))
	for _, call := range []func(*OwnedRational) (OwnedRational, error){s.ToRational, s.GetRational, s.Numerator, s.Denominator, s.GetInfinitesimal} {
		v, e := call(&a)
		serviceMust(t, e)
		serviceAuditRationalRoots(t, s, &a.raw, &d.raw, &v.raw)
		serviceMust(t, s.Del(&v))
	}
	if got, e := s.IsRational(&a); e != nil || !got {
		t.Fatal(got, e)
	}
	x, e := s.InfIntNewValue()
	serviceMust(t, e)
	x.first = Rational{num: NewInt(2), den: NewInt(0)}
	x.second = 5
	y, e := s.NewInfValue()
	serviceMust(t, e)
	y.first = x.first
	y.second = Rational{num: NewInt(-3), den: NewInt(0)}
	z, e := s.InfEpsInfNewValue()
	serviceMust(t, e)
	z.infinity = Rational{num: NewInt(1), den: NewInt(0)}
	z.numeral.first = y.first
	z.numeral.second = y.second
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
	serviceAuditRationalRoots(t, s, &a.raw, &d.raw, &x.first, &xx.first, &y.first, &y.second, &yy.first, &yy.second, &z.infinity, &z.numeral.first, &z.numeral.second, &zz.infinity, &zz.numeral.first, &zz.numeral.second)
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
	s, _ := serviceNew(t)
	x, e := s.InfIntNewValue()
	serviceMust(t, e)
	x.first = Rational{num: NewInt(7), den: NewInt(0)}
	x.second = 4
	y, e := s.NewInfValue()
	serviceMust(t, e)
	y.first = x.first
	y.second = Rational{num: NewInt(3), den: NewInt(0)}
	for _, den := range []bool{false, true} {
		var out OwnedInfIntRational
		if den {
			out, e = s.InfIntDenominator(&x)
		} else {
			out, e = s.InfIntNumerator(&x)
		}
		if modularSourceDebug {
			if !errors.Is(e, ErrInfIntNotRational) || out.ownerID != 0 {
				t.Fatal(out, e)
			}
		} else {
			serviceMust(t, e)
			want := int32(7)
			if den {
				want = 0
			}
			if out.first.num.val != want || out.first.den.val != 1 || out.second != 0 {
				t.Fatal(out)
			}
			serviceMust(t, s.InfIntDel(&out))
		}
		var r OwnedInfRational
		if den {
			r, e = s.InfDenominator(&y)
		} else {
			r, e = s.InfNumerator(&y)
		}
		if modularSourceDebug {
			if !errors.Is(e, ErrInfRationalDomain) || r.ownerID != 0 {
				t.Fatal(r, e)
			}
		} else {
			serviceMust(t, e)
			want := int32(7)
			if den {
				want = 0
			}
			if r.first.num.val != want || r.first.den.val != 1 || r.second.num.val != 0 {
				t.Fatal(r)
			}
			serviceMust(t, s.InfDel(&r))
		}
	}
	serviceAuditRationalRoots(t, s, &x.first, &y.first, &y.second)
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
				a.first = Rational{num: NewInt(first), den: NewInt(1)}
				a.second = eps
				b.first = a.first
				b.second = Rational{num: NewInt(eps), den: NewInt(0)}
				c.infinity = Rational{num: NewInt(infty), den: NewInt(0)}
				c.numeral.first = b.first
				c.numeral.second = b.second
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
	setIntegerOracle(t, s.manager.z, &dead, new(big.Int).Lsh(big.NewInt(1), 95))
	stale := dead
	serviceMust(t, s.manager.z.Del(&dead))
	b.first = Rational{num: stale, den: stale}
	b.second = Rational{num: NewInt(0), den: stale}
	if yes, e := s.InfIsPos(&b); e != nil || !yes {
		t.Fatal("sign uses scalar m_val", yes, e)
	}
	c.infinity = Rational{num: NewInt(1), den: stale}
	c.numeral.first = b.first
	c.numeral.second = b.second
	if yes, e := s.InfEpsInfIsPos(&c); e != nil || !yes {
		t.Fatal(yes, e)
	}
	v, e := s.InfEpsInfGetInfinity(&c)
	if e == nil {
		s.Del(&v)
		t.Fatal("selected DEN must be read")
	}
	c.infinity.den = NewInt(0)
	v, e = s.InfEpsInfGetInfinity(&c)
	serviceMust(t, e)
	serviceMust(t, s.Del(&v))
	// Source reference projections select a subobject without inspecting FIRST.
	b.second.den = NewInt(0)
	selected, e := s.InfGetInfinitesimal(&b)
	serviceMust(t, e)
	serviceMust(t, s.Del(&selected))
	a.first = Rational{num: stale, den: stale}
	a.second = 4
	selected, e = s.InfIntInfinitesimal(&a)
	serviceMust(t, e)
	if selected.raw.num.val != 4 || selected.raw.den.val != 1 {
		t.Fatal(selected)
	}
	serviceMust(t, s.Del(&selected))
	a.first = NewRational()
	raw, e := s.NewValue()
	serviceMust(t, e)
	raw.raw = Rational{num: stale, den: stale}
	if yes, e := s.IsRational(&raw); e != nil || !yes {
		t.Fatal(yes, e)
	}
	selected, e = s.GetInfinitesimal(&raw)
	serviceMust(t, e)
	serviceMust(t, s.Del(&selected))
	raw.raw = NewRational()
	serviceMust(t, s.Del(&raw))
	b.first = NewRational()
	b.second = NewRational()
	c.numeral.first = NewRational()
	c.numeral.second = NewRational()
	serviceAuditRationalRoots(t, s, &a.first, &b.first, &b.second, &c.infinity, &c.numeral.first, &c.numeral.second)
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
	for _, r := range []*Rational{&x.infinity, &x.numeral.first, &x.numeral.second} {
		setIntegerOracle(t, s.manager.z, &r.num, n)
		r.den = NewInt(0)
	}
	out, e := s.InfEpsInfClone(&x)
	serviceMust(t, e)
	for _, pair := range []struct{ a, b *Rational }{{&x.infinity, &out.infinity}, {&x.numeral.first, &out.numeral.first}, {&x.numeral.second, &out.numeral.second}} {
		if pair.a.num.cell == pair.b.num.cell {
			t.Fatal("shallow owning copy")
		}
		rawProjectionEqual(t, s.manager.z, &pair.b.num, n)
	}
	serviceMust(t, s.InfEpsInfDel(&x))
	serviceAuditRationalRoots(t, s, &out.infinity, &out.numeral.first, &out.numeral.second)
	serviceMust(t, s.InfEpsInfDel(&out))
	// Late nested source invalidity is guarded where the component is copied,
	// after infinity and nested FIRST have actually published.
	a, e := s.InfEpsInfNewValue()
	serviceMust(t, e)
	d, e := s.InfEpsInfNewValue()
	serviceMust(t, e)
	a.infinity = Rational{num: NewInt(2), den: NewInt(0)}
	a.numeral.first = Rational{num: NewInt(3), den: NewInt(0)}
	dead := NewInt(0)
	setIntegerOracle(t, s.manager.z, &dead, n)
	stale := dead
	serviceMust(t, s.manager.z.Del(&dead))
	a.numeral.second.num = stale
	if e = s.InfEpsInfCopy(&d, &a); e == nil {
		t.Fatal("actual late component read")
	}
	if d.infinity.num.val != 2 || d.infinity.den.val != 0 || d.numeral.first.num.val != 3 || d.numeral.first.den.val != 0 || d.numeral.second.num.val != 0 {
		t.Fatal(d)
	}
	a.numeral.second.num = NewInt(0)
	serviceAuditRationalRoots(t, s, &a.infinity, &a.numeral.first, &a.numeral.second, &d.infinity, &d.numeral.first, &d.numeral.second)
	serviceMust(t, s.InfEpsInfDel(&a))
	serviceMust(t, s.InfEpsInfDel(&d))
	for fail := int64(0); fail < 256; fail++ {
		ss, bb := serviceNew(t)
		a, e := ss.InfEpsInfNewValue()
		serviceMust(t, e)
		for _, r := range []*Rational{&a.infinity, &a.numeral.first, &a.numeral.second} {
			setIntegerOracle(t, ss.manager.z, &r.num, n)
			r.den = NewInt(0)
		}
		bb.FailAfter(fail)
		clone, err := ss.InfEpsInfClone(&a)
		roots := []*Rational{&a.infinity, &a.numeral.first, &a.numeral.second}
		if err == nil {
			roots = append(roots, &clone.infinity, &clone.numeral.first, &clone.numeral.second)
		} else if clone.ownerID != 0 || !numericPhysicalReservationRefusal(err) {
			t.Fatal(fail, clone, err)
		}
		serviceAuditRationalRoots(t, ss, roots...)
		if err != nil && !errors.Is(err, arena.ErrInjectedFailure) && !ss.manager.z.memory.State().OutOfMemory {
			t.Fatal("physical Memout requires real OOM state", fail, err)
		}
		bb.FailAfter(-1)
		serviceMust(t, ss.manager.z.CleanupPendingAllocation())
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
	r.raw = Rational{num: NewInt(1), den: NewInt(0)}
	x, e := s.InfIntNewValue()
	serviceMust(t, e)
	x.first = r.raw
	y, e := s.NewInfValue()
	serviceMust(t, e)
	y.first = r.raw
	y.second.den = NewInt(0)
	z, e := s.InfEpsInfNewValue()
	serviceMust(t, e)
	z.infinity.den = NewInt(0)
	z.numeral.first = y.first
	z.numeral.second = y.second
	before := numericWarmProfileSnapshot(t, s.manager.z, b)
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
	serviceAuditRationalRoots(t, s, &r.raw, &x.first, &y.first, &y.second, &z.infinity, &z.numeral.first, &z.numeral.second)
	numericWarmProfileAudit(t, s.manager.z, b, before, allocs, false)
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
	v.infinity = Rational{num: NewInt(1), den: NewInt(0)}
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
	v.infinity.num = NewInt(0)
	if _, e := s.InfEpsInfIsPos(&v); !errors.Is(e, ErrRationalServiceOwner) {
		t.Fatal("reached nested sign", e)
	}
	v.numeral.ownerID = saved
	serviceAuditRationalRoots(t, s, &v.infinity, &v.numeral.first, &v.numeral.second)
	serviceMust(t, s.InfEpsInfDel(&v))
}

func TestRawProjectionBothPoliciesAllComponentAliases(t *testing.T) {
	for _, synch := range []bool{false, true} {
		m, _ := rationalManager(t, synch)
		source := Rational{num: NewInt(31), den: NewInt(0)}
		destination := NewRational()
		serviceMust(t, m.Copy(&destination, &source))
		if destination.num.val != 31 || destination.den.val != 0 {
			t.Fatal(synch, destination)
		}
		qopsAuditLiveOwnership(t, m, []*Int{&source.num, &source.den, &destination.num, &destination.den})
		for _, den := range []int32{-3, 0, 1, 4} {
			for _, numerator := range []bool{false, true} {
				for _, outNum := range []bool{false, true} {
					a := Rational{num: NewInt(-7), den: NewInt(den)}
					out := &a.den
					if outNum {
						out = &a.num
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
					if out.val != expected {
						t.Fatal(synch, den, numerator, outNum, a)
					}
					qopsAuditLiveOwnership(t, m, []*Int{&a.num, &a.den})
				}
				a := Rational{num: NewInt(-7), den: NewInt(den)}
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
				if a.num.val != expected || a.den.val != 1 {
					t.Fatal(synch, den, numerator, a)
				}
				qopsAuditLiveOwnership(t, m, []*Int{&a.num, &a.den})
			}
		}
		dead := NewInt(0)
		setIntegerOracle(t, m.z, &dead, new(big.Int).Lsh(big.NewInt(1), 99))
		stale := dead
		serviceMust(t, m.z.Del(&dead))
		a := Rational{num: NewInt(7), den: NewInt(0)}
		out := Rational{num: NewInt(-9), den: stale}
		if e := m.NumeratorRational(&a, &out); e == nil {
			t.Fatal("actual late DEN reset guard")
		}
		if out.num.val != 7 {
			t.Fatal("selected copy precedes old DEN destruction", out)
		}
		out.den = NewInt(0)
		qopsAuditLiveOwnership(t, m, []*Int{&a.num, &a.den, &out.num, &out.den})
	}
}
