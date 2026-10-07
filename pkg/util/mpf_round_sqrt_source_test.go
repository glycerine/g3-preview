// Source: src/util/mpf.cpp:2100-2131; src/util/mpf.h:229.
// Licensed under the MIT license. Source author: Christoph Wintersteiger.
package util

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestArbitraryFloatRoundSqrtLiteralBitModes(t *testing.T) {
	for _, tail := range []int32{0, 1, 2, 3, 4, 5, 6, 7} {
		for _, mode := range []ArbitraryFloatRoundingMode{ArbitraryFloatNearestEven, ArbitraryFloatNearestAway, ArbitraryFloatTowardPositive, ArbitraryFloatTowardNegative, ArbitraryFloatTowardZero} {
			b := arena.NewBudget(1 << 26)
			m, e := NewArbitraryFloatManager(IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}})
			serviceMust(t, e)
			var x ArbitraryFloat
			sig := NewInt((1 << 26) + tail)
			serviceMust(t, m.SetComponents(&x, 8, 24, false, 0, &sig))
			serviceMust(t, m.RoundSqrt(mode, &x))
			got, e := m.integer.Int64(&x.significand)
			serviceMust(t, e)
			sticky := tail&3 != 0
			round := tail&4 != 0
			inc := (mode == ArbitraryFloatNearestEven || mode == ArbitraryFloatNearestAway) && round && sticky || mode == ArbitraryFloatTowardPositive && (round || sticky)
			want := int64(0)
			if inc {
				want = 1
			}
			if got != want || x.exponent != 0 || x.Sign() {
				t.Fatal("literal sqrt bit switch", mode, tail, got, want)
			}
			serviceMust(t, m.Del(&x))
			serviceMust(t, m.Close())
			if b.Used() != 0 {
				t.Fatal(b.Used())
			}
		}
	}
}
