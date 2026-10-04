// Licensed under the MIT license.
// Source MPFF conversion fixtures.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"math/big"
	"testing"
)

func TestFastFloatIntegerConversionSource(t *testing.T) {
	m := fastManager(t, 2)
	s, e := NewRationalService(serviceConfig(m.config.Storage.Descriptors.Budget))
	serviceMust(t, e)
	defer s.Close()
	z := s.manager.z
	var n, out Int
	defer z.Del(&n)
	defer z.Del(&out)
	var a FastFloat
	defer m.Del(&a)
	two := NewInt(2)
	serviceMust(t, z.Power(&two, 512, &n))
	serviceMust(t, m.SetInteger(&a, z, &n))
	fastText(t, m, &a, "9223372036854775808*2^449")
	nText, e := z.String(&n)
	serviceMust(t, e)
	fText, e := m.String(&a)
	serviceMust(t, e)
	wantN := new(big.Int).Lsh(big.NewInt(1), 512).String()
	if "[mpz->mpff] a: "+nText+", b: "+fText != "[mpz->mpff] a: "+wantN+", b: 9223372036854775808*2^449" {
		t.Fatal("source full printed conversion", nText, fText)
	}
	serviceMust(t, m.ToInteger(&a, z, &out))
	ok, e := z.Equal(&n, &out)
	fastEnsure(t, ok, e)
	q := NewRational()
	defer s.manager.Del(&q)
	serviceMust(t, m.ToRational(&a, s.manager, &q))
	ok, e = s.manager.z.Equal(&n, &q.num)
	fastEnsure(t, ok, e)
	// Full odd-width finite domain, signed bounds and source selected fast paths.
	serviceMust(t, m.Del(&a))
	a = FastFloat{}
	for _, p := range []uint32{2, 3, 5} {
		f := fastManager(t, p)
		for _, v := range []int64{-9223372036854775808, -1, 0, 1, 9223372036854775807} {
			serviceMust(t, z.SetInt64(&n, v))
			serviceMust(t, f.SetInteger(&a, z, &n))
			serviceMust(t, f.ToInteger(&a, z, &out))
			ok, e = z.Equal(&n, &out)
			fastEnsure(t, ok, e)
			serviceMust(t, f.Del(&a))
			a = FastFloat{}
		}
	}
}

func TestFastFloatRationalRawShapeAndConversionFailures(t *testing.T) {
	for op := 0; op < 4; op++ {
		complete := false
		for fail := int64(0); fail < 512; fail++ {
			budget := arena.NewBudget(1 << 28)
			c := DefaultFastFloatConfig(serviceConfig(budget).Integers.Words)
			c.InitialCapacity = 2
			c.PrecisionWords = 5
			m, e := NewFastFloatManager(c)
			serviceMust(t, e)
			s, e := NewRationalService(serviceConfig(budget))
			serviceMust(t, e)
			qm := s.manager
			z := qm.z
			q, out := NewRational(), NewRational()
			var n, ni Int
			var a FastFloat
			serviceMust(t, z.SetString(&n, "123456789123456789123456789123456789123456789123456789"))
			serviceMust(t, z.Copy(&q.num, &n))
			serviceMust(t, z.SetInt32(&q.den, -7))
			serviceMust(t, m.SetFraction32(&a, 1, 3))
			serviceMust(t, m.RoundToPlusInf())
			budget.FailAfter(fail)
			switch op {
			case 0:
				e = m.SetInteger(&a, z, &n)
			case 1:
				e = m.SetRational(&a, qm, &q)
			case 2:
				e = m.ToRational(&a, qm, &out)
			case 3:
				e = m.Significand(&a, z, &ni)
			}
			fastAudit(t, m, &a)
			qopsAuditLiveOwnership(t, qm, []*Int{&n, &ni, &q.num, &q.den, &out.num, &out.den})
			if !m.toPlus {
				t.Fatal("rounding not restored", op, fail)
			}
			if e != nil && !numericPhysicalReservationRefusal(e) {
				t.Fatal(op, fail, e)
			}
			budget.FailAfter(-1) // only after immediate actual physical/named root audits
			serviceMust(t, m.Del(&a))
			serviceMust(t, qm.Del(&out))
			serviceMust(t, qm.Del(&q))
			serviceMust(t, z.Del(&ni))
			serviceMust(t, z.Del(&n))
			fastAudit(t, m)
			qopsAuditLiveOwnership(t, qm, nil)
			s.Close()
			m.Close()
			if budget.Used() != 0 {
				t.Fatal("conversion leak", op, fail, budget.Used())
			}
			if e == nil {
				complete = true
				break
			}
		}
		if !complete {
			t.Fatal("incomplete conversion sweep", op)
		}
	}
	m := fastManager(t, 3)
	s, e := NewRationalService(serviceConfig(m.config.Storage.Descriptors.Budget))
	serviceMust(t, e)
	defer s.Close()
	qm := s.manager
	for _, tc := range []struct{ n, d int32 }{{2, 4}, {2, -4}, {-2, -4}, {-2, 4}} {
		for _, up := range []bool{false, true} {
			q := NewRational()
			var a FastFloat
			serviceMust(t, qm.z.SetInt32(&q.num, tc.n))
			serviceMust(t, qm.z.SetInt32(&q.den, tc.d))
			serviceMust(t, m.SetRounding(up))
			serviceMust(t, m.SetRational(&a, qm, &q))
			if fastExact(m, &a).Cmp(big.NewRat(int64(tc.n), int64(tc.d))) != 0 {
				t.Fatal("raw rational shape", tc, up)
			}
			fastAudit(t, m, &a)
			qopsAuditLiveOwnership(t, qm, []*Int{&q.num, &q.den})
			serviceMust(t, m.Del(&a))
			serviceMust(t, qm.Del(&q))
		}
	}
}
func TestFastFloatIntegerSourceDiscardedLimbDefects(t *testing.T) {
	m := fastManager(t, 2)
	s, e := NewRationalService(serviceConfig(m.config.Storage.Descriptors.Budget))
	serviceMust(t, e)
	defer s.Close()
	z := s.manager.z
	var n Int
	var a FastFloat
	defer z.Del(&n)
	defer m.Del(&a)
	// Four words, leading lz=1. The two discarded nonzero words cause two
	// source increments, rather than the single mathematical directed increment.
	serviceMust(t, z.SetString(&n, "85070591730234615884290395935830507521"))
	serviceMust(t, m.RoundToPlusInf())
	serviceMust(t, m.SetInteger(&a, z, &n))
	exact := new(big.Int)
	exact.SetString("85070591730234615884290395935830507521", 10)
	shifted := new(big.Int).Lsh(new(big.Int).Set(exact), 1)
	hi := new(big.Int).Rsh(shifted, 64)
	hi.Add(hi, big.NewInt(2))
	want := new(big.Rat).SetInt(new(big.Int).Lsh(hi, 63))
	if got := fastExact(m, &a); got.Cmp(want) != 0 {
		t.Fatal("source increment fixture", got, want)
	}
	// A full-width leading word drives the literal source SASSERT(k!=0).
	serviceMust(t, z.SetString(&n, "170141183460469231731687303715884105729"))
	e = m.SetInteger(&a, z, &n)
	if sourceExtendedGCDDebug {
		if !errors.Is(e, ErrFastFloatInvariant) {
			t.Fatal(e)
		}
	} else {
		serviceMust(t, e)
	}
	fastAudit(t, m, &a)
	qopsAuditLiveOwnership(t, s.manager, []*Int{&n})
}

func TestFastFloatToIntegerSourceReleaseTruncation(t *testing.T) {
	m := fastManager(t, 2)
	s, e := NewRationalService(serviceConfig(m.config.Storage.Descriptors.Budget))
	serviceMust(t, e)
	defer s.Close()
	z := s.manager.z
	var a FastFloat
	out := NewInt(7)
	defer m.Del(&a)
	defer z.Del(&out)
	for _, n := range []int32{3, -3} {
		serviceMust(t, m.SetFraction32(&a, n, 2))
		m.write(&m.buffers[0], 0, 123)
		e = m.ToInteger(&a, z, &out)
		if sourceExtendedGCDDebug {
			if !errors.Is(e, ErrFastFloatInvariant) || out.val != 7 || m.read(&m.buffers[0], 0) != 123 {
				t.Fatal("source assertion before truncation", e, out)
			}
		} else {
			serviceMust(t, e)
			want := int32(1)
			if n < 0 {
				want = -1
			}
			if out.val != want {
				t.Fatal("release defined truncation", out, want)
			}
		}
		serviceMust(t, z.SetInt32(&out, 7))
		fastAudit(t, m, &a)
		qopsAuditLiveOwnership(t, s.manager, []*Int{&out})
	}
	serviceMust(t, m.SetInt32(&a, 1))
	serviceMust(t, m.SetExponent(&a, -100))
	e = m.ToInteger(&a, z, &out)
	if sourceExtendedGCDDebug {
		if !errors.Is(e, ErrFastFloatInvariant) || out.val != 7 {
			t.Fatal(e, out)
		}
	} else {
		serviceMust(t, e)
		if out.val != 0 {
			t.Fatal("source wide shift", out)
		}
	}
	fastAudit(t, m, &a)
	qopsAuditLiveOwnership(t, s.manager, []*Int{&out})
	serviceMust(t, z.SetInt32(&out, 7))
	serviceMust(t, m.SetExponent(&a, -2147483648))
	m.write(&m.buffers[0], 0, 123)
	e = m.ToInteger(&a, z, &out)
	if sourceExtendedGCDDebug {
		if !errors.Is(e, ErrFastFloatInvariant) || m.read(&m.buffers[0], 0) != 123 {
			t.Fatal("debug assertion precedes undefined negation", e)
		}
	} else if !errors.Is(e, ErrFastFloatDomain) || m.read(&m.buffers[0], 0) != m.sig(&a).read(0) {
		t.Fatal("source INT_MIN unary negation adapter after copy", e)
	}
	if out.val != 7 {
		t.Fatal("undefined negation published integer", out)
	}
	fastAudit(t, m, &a)
	qopsAuditLiveOwnership(t, s.manager, []*Int{&out})
}

func TestFastFloatGetterSourceReleaseActions(t *testing.T) {
	m := fastManager(t, 2)
	var a FastFloat
	defer m.Del(&a)
	for _, n := range []int32{3, -3} {
		serviceMust(t, m.SetFraction32(&a, n, 2))
		u, ue := m.Uint64(&a)
		i, ie := m.Int64(&a)
		if sourceExtendedGCDDebug {
			if !errors.Is(ue, ErrFastFloatInvariant) || !errors.Is(ie, ErrFastFloatInvariant) {
				t.Fatal("debug predicate assertion", ue, ie)
			}
		} else {
			serviceMust(t, ue)
			serviceMust(t, ie)
			want := int64(1)
			if n < 0 {
				want = -1
			}
			if u != 1 || i != want {
				t.Fatal("release top64 truncation", n, u, i)
			}
		}
		fastAudit(t, m, &a)
	}
	serviceMust(t, m.SetUint64(&a, 1<<63))
	i, e := m.Int64(&a)
	if sourceExtendedGCDDebug {
		if !errors.Is(e, ErrFastFloatInvariant) {
			t.Fatal(e)
		}
	} else if e != nil || i != -9223372036854775808 {
		t.Fatal("source signed-conversion adapter", i, e)
	}
	for _, exp := range []int32{1, -64, -2147483648} {
		serviceMust(t, m.SetExponent(&a, exp))
		_, ue := m.Uint64(&a)
		_, ie := m.Int64(&a)
		if sourceExtendedGCDDebug {
			if !errors.Is(ue, ErrFastFloatInvariant) || !errors.Is(ie, ErrFastFloatInvariant) {
				t.Fatal(exp, ue, ie)
			}
		} else if !errors.Is(ue, ErrFastFloatDomain) || !errors.Is(ie, ErrFastFloatDomain) {
			t.Fatal("undefined source shift/negation", exp, ue, ie)
		}
		fastAudit(t, m, &a)
	}
	// Raw nonnormalized negative half has a defined shift63, unlike canonical
	// half's undefined shift64. Getters do not invoke source check(n).
	serviceMust(t, m.SetExponent(&a, -63))
	m.sig(&a).write(1, 0x40000000)
	serviceMust(t, m.Neg(&a))
	u, ue := m.Uint64(&a)
	i, ie := m.Int64(&a)
	if sourceExtendedGCDDebug {
		if !errors.Is(ue, ErrFastFloatInvariant) || !errors.Is(ie, ErrFastFloatInvariant) {
			t.Fatal(ue, ie)
		}
	} else if ue != nil || ie != nil || u != 0 || i != 0 {
		t.Fatal("defined raw negative half", u, i, ue, ie)
	}
	fastAudit(t, m, &a)
	odd := fastManager(t, 3)
	var b FastFloat
	defer odd.Del(&b)
	serviceMust(t, odd.SetUint64(&b, 1<<63))
	serviceMust(t, odd.Neg(&b))
	odd.sig(&b).write(0, 1)
	u, ue = odd.Uint64(&b)
	i, ie = odd.Int64(&b)
	if sourceExtendedGCDDebug {
		if !errors.Is(ue, ErrFastFloatInvariant) || !errors.Is(ie, ErrFastFloatInvariant) {
			t.Fatal(ue, ie)
		}
	} else if ue != nil || ie != nil || u != 1<<63 || i != -9223372036854775808 {
		t.Fatal("source top64 INT_MIN branch ignores low limb", u, i, ue, ie)
	}
	fastAudit(t, odd, &b)
}
