// Source: src/util/mpf.cpp:108-112; src/util/mpf.h:77.
package util

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"math"
	"math/big"
	"testing"
)

func TestArbitraryFloatSignedFractionIndependentModes(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	var out ArbitraryFloat
	for _, sb := range []uint32{2, 4, 24, 53, 120} {
		for _, mode := range mpfIntegralModes {
			for _, pair := range [][2]int32{{0, 1}, {1, 3}, {-1, 3}, {7, -3}, {-7, -3}, {3, 2}, {31, 16}, {math.MinInt32, 1}, {math.MinInt32, 3}, {math.MaxInt32, -7}, {1, math.MaxInt32}} {
				serviceMust(t, m.SetFractionSigned(&out, 11, sb, mode, pair[0], pair[1]))
				exact := new(big.Rat).SetFrac(big.NewInt(int64(pair[0])), big.NewInt(int64(pair[1])))
				want := mpfAddSubOracle(exact, sb, mode)
				q := NewRational()
				serviceMust(t, m.ToRational(&out, m.rational, &q))
				n, e := m.integer.String(&q.Num)
				serviceMust(t, e)
				d, e := m.integer.String(&q.Den)
				serviceMust(t, e)
				got, ok := new(big.Rat).SetString(n + "/" + d)
				if !ok || got.Cmp(want) != 0 {
					t.Fatal("independent signed fraction", sb, mode, pair, got, want)
				}
				serviceMust(t, m.rational.Del(&q))
				serviceMust(t, m.CleanupRationalConversion())
			}
		}
	}
	serviceMust(t, m.Del(&out))
}

func TestArbitraryFloatSignedFractionReachedUndefinedNegation(t *testing.T) {
	m, _, b := mpfNumeralHelpersFixture(t)
	var out ArbitraryFloat
	serviceMust(t, m.SetInt32(&out, 8, 24, 7))
	old := out
	used := b.Used()
	for _, pair := range [][2]int32{{math.MinInt32, -1}, {1, math.MinInt32}, {0, math.MinInt32}} {
		if e := m.SetFractionSigned(&out, 8, 24, ArbitraryFloatNearestEven, pair[0], pair[1]); !errors.Is(e, arena.ErrBounds) {
			t.Fatal(e)
		}
		if out != old || b.Used() != used || m.signedFraction.childInvocation != 0 {
			t.Fatal("unreached conversion changed", pair)
		}
		serviceMust(t, m.CleanupSignedFraction())
	}
	serviceMust(t, m.Del(&out))
}

func signedFractionPhysicalAudit(t *testing.T, m *ArbitraryFloatManager, out *ArbitraryFloat, blocks []Block, extra ...*Int) {
	t.Helper()
	s, c, r := &m.signedFraction, &m.rationalSet, &m.RoundScope
	nums := []*Int{&out.significand, &s.q.Num, &s.q.Den, &m.plainRationalSet.exp, &c.sig.Num, &c.sig.Den, &c.t.Num, &c.t.Den, &c.sq.Num, &c.sq.Den, &c.exp, &c.p, &c.p2, &c.prev, &r.sigma, &r.cap, &r.sticky, &m.rational.NTmp, &m.rational.AddTmp1, &m.rational.AddTmp2, &m.rational.LtTmp1.Num, &m.rational.LtTmp1.Den, &m.rational.LtTmp2.Num, &m.rational.LtTmp2.Den}
	nums = append(nums, extra...)
	integerPhysicalAuditWithRaw(t, m.integer, blocks, nums...)
	if m.budget.Used() != m.OwnedStorageBytes()+m.memory.Stats().TotalBytes {
		t.Fatal("signed fraction physical byte graph", m.budget.Used(), m.OwnedStorageBytes(), m.memory.Stats().TotalBytes)
	}
}

func TestArbitraryFloatSignedFractionEveryColdReservation(t *testing.T) {
	for _, tc := range []struct {
		n, d int32
		sb   uint32
	}{{3, 2, 4000}, {math.MinInt32, 3, 120}, {math.MinInt32, 8, 120}, {7, -3, 120}, {0, 1, 120}} {
		success := false
		for ordinal := int64(0); ordinal < 500; ordinal++ {
			b := arena.NewBudget(1 << 28)
			m, e := NewArbitraryFloatManager(IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}})
			serviceMust(t, e)
			var out ArbitraryFloat
			b.FailAfter(ordinal)
			body := m.SetFractionSigned(&out, 11, tc.sb, ArbitraryFloatNearestEven, tc.n, tc.d)
			signedFractionPhysicalAudit(t, m, &out, nil)
			if body != nil && !numericPhysicalReservationRefusal(body) {
				t.Fatal("nonreservation cold failure", tc, ordinal, body)
			}
			s := &m.signedFraction
			if body != nil && s.childInvocation == 0 && s.normalizationOwned {
				t.Fatal("conversion missing after completed normalization", tc, ordinal, body)
			}
			b.FailAfter(-1)
			if body == nil {
				q := NewRational()
				serviceMust(t, m.ToRational(&out, m.rational, &q))
				ns, e := m.integer.String(&q.Num)
				serviceMust(t, e)
				ds, e := m.integer.String(&q.Den)
				serviceMust(t, e)
				got, ok := new(big.Rat).SetString(ns + "/" + ds)
				want := mpfAddSubOracle(new(big.Rat).SetFrac(big.NewInt(int64(tc.n)), big.NewInt(int64(tc.d))), tc.sb, ArbitraryFloatNearestEven)
				if !ok || got.Cmp(want) != 0 {
					t.Fatal("cold result", tc, ordinal, got, want)
				}
				serviceMust(t, m.rational.Del(&q))
				serviceMust(t, m.CleanupRationalConversion())
				success = true
			}
			serviceMust(t, m.CleanupSignedFraction())
			if s.exponentInvocation != 0 && s.exponentInvocation == m.rationalSet.invocation {
				c := &m.rationalSet
				if c.active || c.sigLive || c.expLive || c.pLive || c.tLive || c.sqLive || m.integer.MatchesRawInvocation(&c.gcd) || m.integer.MatchesRawInvocation(&c.division) || m.integer.MatchesRawInvocation(&c.extended) {
					t.Fatal("wrapper failed to join actual conversion child", tc, ordinal)
				}
			}
			serviceMust(t, m.CleanupValue(&out))
			if e = m.Close(); e != nil {
				if !m.integer.sourceGuard || m.integer.SourceFinalizeStage < 2 || m.integer.SourceFinalizeStage > 5 {
					t.Fatal("unclassified owner stop", tc, ordinal, e)
				}
				signedFractionScratchHole(t, m)
				serviceMust(t, m.Close())
			}
			if b.Used() != 0 {
				t.Fatal("cold refunds", tc, ordinal, b.Used())
			}
			if success {
				break
			}
		}
		if !success {
			t.Fatal("no complete cold call", tc)
		}
	}
}

func TestArbitraryFloatSignedFractionWarmAndLaterChild(t *testing.T) {
	m, r, b := mpfNumeralHelpersFixture(t)
	var out ArbitraryFloat
	serviceMust(t, m.SetFractionSigned(&out, 8, 4, ArbitraryFloatNearestEven, 3, 2))
	state, used := r.State(), b.Used()
	b.FailAfter(0)
	serviceMust(t, m.SetFractionSigned(&out, 8, 4, ArbitraryFloatNearestEven, -3, 2))
	if out.exponent != 0 || out.significand != NewInt(4) || !out.Sign() || r.State() != state || b.Used() != used {
		t.Fatal("warm source allocated", out)
	}
	b.FailAfter(-1)
	q := NewRational()
	serviceMust(t, m.rational.SetFractionSignedSource(&q, 1, 3))
	serviceMust(t, m.SetRational(&out, 8, 4, ArbitraryFloatTowardZero, &q))
	later, child, used := m.plainRationalSet, m.rationalSet, b.Used()
	serviceMust(t, m.CleanupSignedFraction())
	if m.plainRationalSet != later || m.rationalSet != child || b.Used() != used {
		t.Fatal("ended parent cleanup touched later child")
	}
	serviceMust(t, m.CleanupPlainRationalSet())
	serviceMust(t, m.rational.Del(&q))
	serviceMust(t, m.Del(&out))
}

func TestArbitraryFloatSignedFractionEntryAuthority(t *testing.T) {
	m, _, b := mpfNumeralHelpersFixture(t)
	var out ArbitraryFloat
	old, used := out, b.Used()
	if e := m.SetFractionSigned(nil, 8, 24, ArbitraryFloatNearestEven, 1, 2); !errors.Is(e, ErrRuntimeOwner) {
		t.Fatal("nil output", e)
	}
	copied := *m
	if e := copied.SetFractionSigned(&out, 8, 24, ArbitraryFloatNearestEven, 1, 2); !errors.Is(e, ErrRuntimeOwner) {
		t.Fatal("copied manager", e)
	}
	m.signedFraction.invocation = ^uint64(0)
	if e := m.SetFractionSigned(&out, 8, 24, ArbitraryFloatNearestEven, 1, 2); !errors.Is(e, ErrNumericPending) || out != old || b.Used() != used || m.signedFraction.qLive {
		t.Fatal("counter wrapped", e)
	}
	m.signedFraction.invocation = 0
	serviceMust(t, m.SetFractionSigned(&out, 8, 24, ArbitraryFloatNearestEven, 0, 1))
	if m.signedFraction.childInvocation == 0 || m.signedFraction.exponentInvocation != 0 || m.signedFraction.child.frame != nil || m.signedFraction.qLive || m.signedFraction.active {
		t.Fatal("zero path invented nonzero child")
	}
	serviceMust(t, m.Del(&out))
}

func TestArbitraryFloatSignedFractionZeroDenominator(t *testing.T) {
	for _, n := range []int32{0, 1, -1} {
		m, _, _ := mpfNumeralHelpersFixture(t)
		var out ArbitraryFloat
		serviceMust(t, m.SetInt32(&out, 8, 24, 7))
		before := out
		body := m.SetFractionSigned(&out, 8, 24, ArbitraryFloatNearestEven, n, 0)
		if modularSourceDebug {
			if !errors.Is(body, ErrIntegerAssertion) || out != before || m.signedFraction.childInvocation != 0 {
				t.Fatal("debug denominator reached conversion", n, body)
			}
		} else {
			want := ErrDivideByZero
			if n != 0 {
				want = ErrUninitializedRational
			}
			if !errors.Is(body, want) {
				t.Fatal("release reached guarded invalid denominator", n, body)
			}
			if n == 0 && m.signedFraction.childInvocation != 0 {
				t.Fatal("0/0 reached conversion")
			}
			if n != 0 && m.signedFraction.exponentInvocation == 0 {
				t.Fatal("nonzero/0 guard preceded original conversion")
			}
		}
		serviceMust(t, m.CleanupSignedFraction())
		serviceMust(t, m.CleanupValue(&out))
	}
}

func signedFractionScratchHole(t *testing.T, m *ArbitraryFloatManager) {
	t.Helper()
	z := m.integer
	if !z.sourceGuard || z.SourceFinalizeStage < 2 || z.SourceFinalizeStage > 5 {
		t.Fatal("unclassified scratch source guard", z.SourceFinalizeStage)
	}
	i := int(z.SourceFinalizeStage-2) / 2
	ref := z.Tmp[i]
	if z.SourceFinalizeStage%2 == 1 {
		ref = z.Arg[i]
	}
	if ref != 0 {
		t.Fatal("unproved scratch hole", ref)
	}
}
