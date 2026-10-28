package numeral

// Source: src/util/mpf.cpp:748-949.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestArbitraryFloatFMAEveryColdGeneralReservation(t *testing.T) {
	sawIncompleteRes := false
	for _, sb := range []uint32{4, 24, 120, 4000} {
		for alias := 0; alias < 4; alias++ {
			for _, mode := range []ArbitraryFloatRoundingMode{ArbitraryFloatNearestEven, ArbitraryFloatTowardZero} {
				complete := false
				for ordinal := int64(0); ordinal < 256; ordinal++ {
					m, r, b := mpfNumeralHelpersFixture(t)
					n := NewInt(1)
					serviceMust(t, m.integer.Mul2k(&n, sb-1, &n))
					serviceMust(t, m.integer.Dec(&n))
					var x, y, z, separate ArbitraryFloat
					serviceMust(t, m.SetComponents(&x, 17, sb, true, 1, &n))
					serviceMust(t, m.SetComponents(&y, 17, sb, false, -int64(sb), &n))
					serviceMust(t, m.SetComponents(&z, 17, sb, false, -2, &n))
					serviceMust(t, m.integer.Del(&n))
					old := x // borrowed identity only
					out := &separate
					if alias == 1 {
						out = &x
					}
					if alias == 2 {
						out = &y
					}
					if alias == 3 {
						out = &z
					}
					b.FailAfter(ordinal)
					operation := m.FMA
					body := operation(mode, &x, &y, &z, out)
					s := &m.fma
					if s.resLive && !s.resSourceLive {
						sawIncompleteRes = true
					}
					integerPhysicalAudit(t, m.integer, &x.significand, &separate.significand, &s.a.significand, &s.b.significand, &s.sticky, &m.RoundScope.sigma, &m.RoundScope.cap, &m.RoundScope.sticky, &y.significand, &z.significand, &s.mr.significand, &s.c.significand, &s.res.significand)
					if b.Used() != m.OwnedStorageBytes()+r.Stats().TotalBytes {
						t.Fatal("immediate FMA scope physical graph", sb, alias, mode, ordinal, body, b.Used(), m.OwnedStorageBytes(), r.Stats())
					}
					if alias != 1 && x != old {
						t.Fatal("stopped const input changed", ordinal)
					}
					if body != nil && !numericPhysicalReservationRefusal(body) {
						t.Fatal("unexpected cold failure", ordinal, body)
					}
					if s.active {
						if e := operation(mode, &x, &y, &z, out); !errors.Is(e, ErrNumericPending) {
							t.Fatal("stopped scope replay", e)
						}
					}
					b.FailAfter(-1)
					if e := m.CleanupFMA(); e != nil {
						t.Fatalf("cleanup sb=%d alias=%d mode=%d ordinal=%d body=%v scope=%+v: %v", sb, alias, mode, ordinal, body, *s, e)
					}
					serviceMust(t, m.CleanupValue(&separate))
					serviceMust(t, m.CleanupValue(&x))
					serviceMust(t, m.CleanupValue(&y))
					serviceMust(t, m.CleanupValue(&z))
					if source := m.SourceFinalize(); source != nil {
						z := m.integer
						if !errors.Is(source, arena.ErrInvalidSlot) || !z.sourceGuard || z.SourceFinalizeStage < 2 || z.SourceFinalizeStage > 5 {
							t.Fatal("unexpected source destructor stop", sb, alias, mode, ordinal, source, z.SourceFinalizeStage)
						}
						index := int(z.SourceFinalizeStage-2) / 2
						ref := z.Tmp[index]
						if z.SourceFinalizeStage%2 == 1 {
							ref = z.Arg[index]
						}
						if ref != 0 {
							t.Fatal("unproved actual source scratch hole", ordinal, ref)
						}
					}
					serviceMust(t, m.Close())
					serviceMust(t, r.Close())
					if b.Used() != 0 {
						t.Fatal("FMA scope refund", ordinal, b.Used())
					}
					if body == nil {
						complete = true
						break
					}
				}
				if !complete {
					t.Fatal("FMA rounding reservation horizon")
				}
			}
		}
	}
	if !sawIncompleteRes {
		t.Fatal("cold sweep did not reach failed result copy construction")
	}
}
