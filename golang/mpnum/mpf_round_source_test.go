// Source: src/util/mpf.cpp:1963-2098; src/util/mpf.h:31-37.
// Licensed under the MIT license. Source author: Christoph Wintersteiger.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"math"
	"testing"
)

// These are independent branch checks for the literal last/round/sticky switch,
// not additional original driver assertions (src/test/mpf.cpp ends at line89).
func TestArbitraryFloatRoundLiteralHalfwayModes(t *testing.T) {
	for _, sign := range []bool{false, true} {
		for _, mode := range []ArbitraryFloatRoundingMode{ArbitraryFloatNearestEven, ArbitraryFloatNearestAway, ArbitraryFloatTowardPositive, ArbitraryFloatTowardNegative, ArbitraryFloatTowardZero} {
			b := arena.NewBudget(1 << 27)
			m, e := NewArbitraryFloatManager(IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
			serviceMust(t, e)
			sig := NewInt((1 << 26) + 4)
			var x ArbitraryFloat
			serviceMust(t, m.SetComponents(&x, 8, 24, sign, 0, &sig))
			serviceMust(t, m.Round(mode, &x))
			got, e := m.ToDouble(&x)
			serviceMust(t, e)
			inc := mode == ArbitraryFloatNearestAway || mode == ArbitraryFloatTowardPositive && !sign || mode == ArbitraryFloatTowardNegative && sign
			want := float64(1)
			if inc {
				want += math.Ldexp(1, -23)
			}
			if sign {
				want = -want
			}
			if got != want {
				t.Fatal("literal rounding switch", mode, sign, got, want)
			}
			serviceMust(t, m.Del(&x))
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
