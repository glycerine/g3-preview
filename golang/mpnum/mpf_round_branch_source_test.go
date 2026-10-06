// Source: src/util/mpf.cpp:1948-2098.
// Licensed under the MIT license. Source author: Christoph Wintersteiger.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"math"
	"testing"
)

// Independent represented IEEE checks validate literal overflow, denormal and
// carry branches; these are not additional original registered ENSUREs.
func TestArbitraryFloatRoundLiteralOverflowDenormalCarry(t *testing.T) {
	for _, sign := range []bool{false, true} {
		for _, mode := range []ArbitraryFloatRoundingMode{ArbitraryFloatNearestEven, ArbitraryFloatNearestAway, ArbitraryFloatTowardPositive, ArbitraryFloatTowardNegative, ArbitraryFloatTowardZero} {
			b := arena.NewBudget(1 << 27)
			m, e := NewArbitraryFloatManager(IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
			serviceMust(t, e)
			for _, tc := range []struct {
				sig  int32
				exp  int64
				kind uint8
			}{{1 << 26, 2000, 0}, {1 << 26, -149, 1}, {(1 << 27) - 4, 0, 2}} {
				sig := NewInt(tc.sig)
				var x ArbitraryFloat
				serviceMust(t, m.SetComponents(&x, 8, 24, sign, tc.exp, &sig))
				serviceMust(t, m.Round(mode, &x))
				got32, e := m.ToFloat(&x)
				got := float64(got32)
				serviceMust(t, e)
				want := float64(0)
				switch tc.kind {
				case 0:
					max := mode == ArbitraryFloatTowardZero || !sign && mode == ArbitraryFloatTowardNegative || sign && mode == ArbitraryFloatTowardPositive
					if max {
						want = float64(math.MaxFloat32)
					} else {
						want = math.Inf(1)
					}
				case 1:
					want = math.Ldexp(1, -149)
				case 2:
					inc := mode == ArbitraryFloatNearestEven || mode == ArbitraryFloatNearestAway || !sign && mode == ArbitraryFloatTowardPositive || sign && mode == ArbitraryFloatTowardNegative
					if inc {
						want = 2
					} else {
						want = 2 - math.Ldexp(1, -23)
					}
				}
				if sign {
					want = -want
				}
				if got != want {
					t.Fatal("literal branch", tc.kind, mode, sign, got, want)
				}
				serviceMust(t, m.CleanupRound())
				serviceMust(t, m.Del(&x))
			}
			if e := m.SourceFinalize(); e != nil && !m.integer.sourceGuard {
				t.Fatal(e)
			}
			serviceMust(t, m.Close())
			if b.Used() != 0 {
				t.Fatal(b.Used())
			}
		}
	}
}

// The explicit Go guard is at the reached undefined signed source expression;
// it does not claim a C++ compiled-assertion prefix or scoped NUM construction.
func TestArbitraryFloatRoundReachedSignedExpressionGuard(t *testing.T) {
	for _, tc := range []struct {
		sig int32
		exp int64
	}{{1, math.MinInt64}, {1 << 27, math.MaxInt64}} {
		b := arena.NewBudget(1 << 27)
		m, e := NewArbitraryFloatManager(IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
		serviceMust(t, e)
		sig := NewInt(tc.sig)
		var x ArbitraryFloat
		serviceMust(t, m.SetComponents(&x, 8, 24, false, tc.exp, &sig))
		before := x
		if e = m.Round(ArbitraryFloatNearestEven, &x); !errors.Is(e, arena.ErrBounds) {
			t.Fatal("reached signed guard", e)
		}
		if x != before || m.roundScope.sigmaConstructed || m.roundScope.capConstructed || m.roundScope.stickyConstructed {
			t.Fatal("unreached NUM or changed borrowed fields")
		}
		integerPhysicalAudit(t, m.integer, &x.significand)
		if b.Used() != m.memory.Stats().TotalBytes+m.OwnedStorageBytes() {
			t.Fatal("named graph")
		}
		serviceMust(t, m.Del(&x))
		serviceMust(t, m.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	}
}
