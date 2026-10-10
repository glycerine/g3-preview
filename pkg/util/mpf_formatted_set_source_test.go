// Source: src/util/mpf.cpp:335-374; src/util/mpf.h:86.
package util

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestArbitraryFloatFormattedSetLiteralPrecisionRoutes(t *testing.T) {
	for _, tc := range []struct {
		src, dst uint32
		frac     int32
		want     int64
		exp      int64
	}{{24, 30, 0, 0, 0}, {30, 24, 0, 0, 0}, {24, 21, 0, 1 << 23, 0}, {24, 24, 17, 17, 0}} {
		b := arena.NewBudget(1 << 27)
		m, e := NewArbitraryFloatManager(IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
		serviceMust(t, e)
		var x, y ArbitraryFloat
		sig := NewInt(tc.frac)
		serviceMust(t, m.SetComponents(&x, 8, tc.src, false, 0, &sig))
		serviceMust(t, m.SetFormat(&y, 8, tc.dst, ArbitraryFloatNearestEven, &x))
		got, e := m.integer.Int64(&y.significand)
		serviceMust(t, e)
		if y.SBits() != tc.dst || y.EBits() != 8 || got != tc.want || y.exponent != tc.exp {
			t.Fatal("literal ds branch", tc, got, y.exponent)
		}
		serviceMust(t, m.Del(&x))
		serviceMust(t, m.Del(&y))
		if e = m.SourceFinalize(); e != nil && !m.integer.sourceGuard {
			t.Fatal(e)
		}
		serviceMust(t, m.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	}
}
