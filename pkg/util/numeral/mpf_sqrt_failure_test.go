package numeral

// Source: src/util/mpf.cpp:1005-1045.

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"testing"
)

func TestArbitraryFloatSqrtEveryColdGeneralReservation(t *testing.T) {
	for _, sb := range []uint32{2, 24, 120} {
		for _, alias := range []bool{false, true} {
			for _, mode := range []ArbitraryFloatRoundingMode{ArbitraryFloatNearestEven, ArbitraryFloatTowardZero} {
				complete := false
				for ordinal := int64(0); ordinal < 2048; ordinal++ {
					m, r, b := mpfNumeralHelpersFixture(t)
					n := NewInt(1)
					serviceMust(t, m.integer.Mul2k(&n, sb-1, &n))
					serviceMust(t, m.integer.Dec(&n))
					var x, separate ArbitraryFloat
					serviceMust(t, m.SetComponents(&x, 17, sb, false, 1, &n))
					serviceMust(t, m.integer.Del(&n))
					old := x // borrowed identity only
					out := &separate
					if alias {
						out = &x
					}
					b.FailAfter(ordinal)
					body := m.Sqrt(mode, &x, out)
					s := &m.sqrt
					integerPhysicalAudit(t, m.integer, &x.significand, &separate.significand, &s.a.significand, &m.RoundScope.sigma, &m.RoundScope.cap, &m.RoundScope.sticky)
					if b.Used() != m.OwnedStorageBytes()+r.Stats().TotalBytes {
						t.Fatal("immediate sqrt scope physical graph", sb, alias, mode, ordinal, body, b.Used(), m.OwnedStorageBytes(), r.Stats())
					}
					if !alias && x != old {
						t.Fatal("stopped const input changed", ordinal)
					}
					if body != nil && !numericPhysicalReservationRefusal(body) {
						t.Fatal("unexpected cold failure", ordinal, body)
					}
					if s.active {
						if e := m.Sqrt(mode, &x, out); !errors.Is(e, ErrNumericPending) {
							t.Fatal("stopped scope replay", e)
						}
					}
					b.FailAfter(-1)
					if e := m.CleanupSqrt(); e != nil {
						t.Fatalf("cleanup sb=%d alias=%v mode=%d ordinal=%d body=%v scope=%+v: %v", sb, alias, mode, ordinal, body, *s, e)
					}
					serviceMust(t, m.CleanupValue(&separate))
					serviceMust(t, m.CleanupValue(&x))
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
						t.Fatal("sqrt scope refund", ordinal, b.Used())
					}
					if body == nil {
						complete = true
						break
					}
				}
				if !complete {
					t.Fatal("sqrt rounding reservation horizon", sb, alias, mode)
				}
			}
		}
	}
}
