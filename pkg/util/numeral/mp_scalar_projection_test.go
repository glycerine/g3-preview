// Source: mpz.h:231-242,736-742; mpz.cpp:1262-1382; mpq.h:772-778;
// mpq.cpp:297-303; rational.h:101-125. New bounded profile/ownership tests.
package numeral

import (
	"errors"
	"math"
	"math/big"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func projectionExpectError(t *testing.T, e error, assertion bool) {
	t.Helper()
	if modularSourceDebug && assertion {
		if !errors.Is(e, ErrIntegerAssertion) {
			t.Fatal("actual source debug assertion", e)
		}
	} else if e != nil {
		t.Fatal("defined source release projection", e)
	}
}

func projectionExpectSourceGetterError(t *testing.T, e error, assertion bool) {
	t.Helper()
	if modularSourceDebug && assertion {
		if !isMPNVerificationThrow(e) {
			t.Fatal("actual source scalar getter Throw action", e)
		}
	} else if e != nil {
		t.Fatal("defined source release projection", e)
	}
}

func projectionLowMagnitude(n *big.Int) uint64 {
	a := new(big.Int).Abs(new(big.Int).Set(n))
	return new(big.Int).And(a, new(big.Int).SetUint64(math.MaxUint64)).Uint64()
}

func TestScalarProjectionIntegerLowWordsAndDebugProfiles(t *testing.T) {
	if modularSourceDebug {
		bindMPNDebugActions(t, strings.Repeat("T", 512))
	}
	m, _ := integerManager(t)
	inputs := []*big.Int{big.NewInt(math.MinInt32), big.NewInt(-7), big.NewInt(0), big.NewInt(math.MaxInt32), new(big.Int).SetUint64(math.MaxUint32)}
	for _, bit := range []uint{32, 63, 64, 65, 95, 127, 160} {
		for _, offset := range []int64{-1, 0, 7} {
			v := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), bit), big.NewInt(offset))
			inputs = append(inputs, v, new(big.Int).Neg(new(big.Int).Set(v)))
		}
	}
	for _, n := range inputs {
		var a Int
		setIntegerOracle(t, m, &a, n)
		before := a
		u := projectionLowMagnitude(n)
		if a.IsSmall() {
			u = uint64(int64(a.Val))
		}
		s := int64(projectionLowMagnitude(n)) // selected two's-complement signed cast
		if n.Sign() < 0 {
			s = -s // the source has an explicit MIN special case, same scalar result
		}
		if a.IsSmall() {
			s = int64(a.Val)
		}
		uv, ue := m.Uint64(&a)
		iv, ie := m.Int64(&a)
		u32, u32e := m.Uint32(&a)
		i32, i32e := m.Int32(&a)
		integerPhysicalAudit(t, m, &a)
		projectionExpectSourceGetterError(t, ue, !a.IsSmall() && n.BitLen() > 64)
		projectionExpectSourceGetterError(t, ie, !n.IsInt64())
		wu := n.Sign() >= 0 && n.BitLen() <= 64 && n.Cmp(new(big.Int).SetUint64(math.MaxUint32)) < 0
		wi := n.Cmp(big.NewInt(math.MinInt32)) > 0 && n.Cmp(big.NewInt(math.MaxInt32)) < 0
		projectionExpectSourceGetterError(t, u32e, !wu)
		projectionExpectSourceGetterError(t, i32e, !wi)
		if (ue == nil && uv != u) || (ie == nil && iv != s) || (u32e == nil && u32 != uint32(u)) || (i32e == nil && i32 != int32(s)) || a != before {
			t.Fatal("low-word source projection/preserved input", n, uv, iv, u32, i32, u, s)
		}
		if got, e := m.IsUint64(&a); e != nil || got != (n.Sign() >= 0 && n.BitLen() <= 64) {
			t.Fatal("unsigned predicate remains a fit/sign query", n, got, e)
		}
		if got, e := m.IsInt64(&a); e != nil || got != n.IsInt64() {
			t.Fatal("signed predicate remains a fit query", n, got, e)
		}
		qopsMust(t, m.Del(&a))
	}
}

func TestScalarProjectionRetainedSizeHeaderReads(t *testing.T) {
	if modularSourceDebug {
		bindMPNDebugActions(t, "TTTTTTTT")
	}
	m, _ := integerManager(t)
	var a Int
	qopsMust(t, m.SetLimbs(&a, []uint32{0x1020304, 0x12345678, 9}))
	original := m.Cell(a.Cell)
	for _, size := range []uint32{0, 1, 2, 3} {
		c := original
		c.Size = size
		m.Store(a.Cell, c)
		u, e := m.Uint64(&a)
		want := uint64(0x12345678)<<32 | 0x1020304
		if size == 1 {
			want = 0x1020304
		}
		projectionExpectSourceGetterError(t, e, size == 0 || size > 2)
		if e == nil && u != want {
			t.Fatal("source reads retained physical D1/D0, not size-zero padding", size, u, want)
		}
		i, ie := m.Int64(&a)
		fits, fe := m.IsInt64(&a)
		projectionExpectSourceGetterError(t, ie, size > 2)
		if fe != nil || fits != (size <= 2) || (ie == nil && i != int64(want)) {
			t.Fatal("signed source has no separate size>0 assertion", size, i, want, fits, ie, fe)
		}
		m.Store(a.Cell, original) // restore malformed bounded fixture before ownership audit
		integerPhysicalAudit(t, m, &a)
	}
	qopsMust(t, m.Del(&a))
}

func TestScalarProjectionRationalRawNumeratorsAndPredicates(t *testing.T) {
	for _, sync := range []bool{false, true} {
		m, _ := rationalManager(t, sync)
		for _, n := range []int64{math.MinInt64, math.MinInt32, -7, 0, 1, math.MaxInt32, math.MaxUint32, math.MaxInt64} {
			for _, den := range []int32{-3, 0, 1, 3} {
				a := Rational{Den: NewInt(den)}
				qopsMust(t, m.Z.SetInt64(&a.Num, n))
				u, ue := m.Uint64(&a)
				s, se := m.Int64(&a)
				u32, u32e := m.Unsigned(&a)
				i32, i32e := m.Int32(&a)
				qopsAuditLiveOwnership(t, m, []*Int{&a.Num, &a.Den})
				projectionExpectError(t, ue, den != 1 || n < 0)
				projectionExpectError(t, se, den != 1)
				projectionExpectError(t, u32e, den != 1 || n < 0 || uint64(n) > math.MaxUint32)
				projectionExpectError(t, i32e, den != 1 || n < math.MinInt32 || n > math.MaxInt32)
				wantU := uint64(n)
				if !a.Num.IsSmall() && n < 0 {
					wantU = uint64(-n)
				}
				if (ue == nil && u != wantU) || (se == nil && s != n) || (u32e == nil && u32 != uint32(wantU)) || (i32e == nil && i32 != int32(n)) {
					t.Fatal("source wrapper forwards actual numerator", sync, n, den, u, s, u32, i32)
				}
				up, upe := m.IsUint64(&a)
				ip, ipe := m.IsInt64(&a)
				unp, unpe := m.IsUnsigned(&a)
				inp, inpe := m.IsInt32(&a)
				if upe != nil || ipe != nil || unpe != nil || inpe != nil || up != (den == 1 && n >= 0) || ip != (den == 1) || unp != (den == 1 && n >= 0 && uint64(n) <= math.MaxUint32) || inp != (den == 1 && n >= math.MinInt32 && n <= math.MaxInt32) {
					t.Fatal("raw denominator predicate short circuit and inclusive wrapper boundaries", n, den, up, ip, unp, inp, upe, ipe, unpe, inpe)
				}
				qopsMust(t, m.Del(&a))
			}
		}
	}
}

func TestScalarProjectionFloatRawIEEEAndDebugAssertion(t *testing.T) {
	for _, sync := range []bool{false, true} {
		m, _ := rationalManager(t, sync)
		for _, tc := range []struct{ n, d int32 }{{2, -3}, {1, 0}, {-1, 0}, {0, 0}, {0, -1}, {2, 3}} {
			a := Rational{Num: NewInt(tc.n), Den: NewInt(tc.d)}
			v, e := m.Float64(&a)
			qopsAuditLiveOwnership(t, m, []*Int{&a.Num, &a.Den})
			if modularSourceDebug && tc.d <= 0 {
				if !errors.Is(e, ErrRationalFloatDomain) {
					t.Fatal("DEBUG after converted denominator", tc, e)
				}
			} else {
				want := float64(tc.n) / float64(tc.d)
				if e != nil || (math.IsNaN(want) && !math.IsNaN(v)) || (!math.IsNaN(want) && (v != want || math.Signbit(v) != math.Signbit(want))) {
					t.Fatal("source IEEE quotient/sign", tc, v, want, e)
				}
			}
		}
		var a Rational
		a.Num = NewInt(1)
		limbs := make([]uint32, 35)
		limbs[0], limbs[34] = 1, 1
		qopsMust(t, m.Z.SetLimbs(&a.Den, limbs))
		v, e := m.Float64(&a)
		qopsAuditLiveOwnership(t, m, []*Int{&a.Num, &a.Den})
		if modularSourceDebug {
			if !errors.Is(e, ErrRationalFloatDomain) {
				t.Fatal("NaN fails actual DEBUG d>0", e)
			}
		} else if e != nil || !math.IsNaN(v) {
			t.Fatal("release NaN denominator quotient", v, e)
		}
		qopsMust(t, m.Del(&a))
	}
}

func TestScalarProjectionInvalidOwnershipAndWarmGraph(t *testing.T) {
	if modularSourceDebug {
		d, _ := bindMPNDebugActions(t, "")
		serviceMust(t, d.EnableAssertions(false))
	}
	m, budget := integerManager(t)
	var a Int
	qopsMust(t, m.SetLimbs(&a, []uint32{7, 3}))
	before := numericWarmProfileSnapshot(t, m, budget)
	allocs := testing.AllocsPerRun(50, func() {
		m.IsUint64(&a)
		m.IsInt64(&a)
		m.Uint64(&a)
		m.Int64(&a)
		m.Uint32(&a)
		m.Int32(&a)
	})
	integerPhysicalAudit(t, m, &a)
	numericWarmProfileAudit(t, m, budget, before, allocs, false)
	stale := a
	qopsMust(t, m.Del(&a))
	for _, call := range []func() error{func() error { _, e := m.Uint64(&stale); return e }, func() error { _, e := m.Int64(&stale); return e }, func() error { _, e := m.Uint32(&stale); return e }, func() error { _, e := m.Int32(&stale); return e }} {
		if e := call(); !errors.Is(e, arena.ErrInvalidSlot) {
			t.Fatal("ownership validation precedes source assertion/read", e)
		}
	}
	integerPhysicalAudit(t, m, &a)
}
