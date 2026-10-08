// Source: src/util/mpf.cpp:667-746; src/util/mpf.h:127.
package util

import (
	"math"
	"math/rand/v2"
	"testing"
)

func TestArbitraryFloatDivSharedInputAndOutput(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	for _, input := range []float64{0, math.Copysign(0, -1), math.SmallestNonzeroFloat64, -1.5, math.MaxFloat64, math.Inf(-1), math.NaN()} {
		var x ArbitraryFloat
		serviceMust(t, m.SetDouble(&x, 11, 53, input))
		serviceMust(t, m.Div(ArbitraryFloatNearestEven, &x, &x, &x))
		got, e := m.ToDouble(&x)
		serviceMust(t, e)
		want := input / input
		if math.IsNaN(want) {
			if !math.IsNaN(got) {
				t.Fatal("shared NaN input", got)
			}
		} else if math.Float64bits(got) != math.Float64bits(want) {
			t.Fatal("shared input/output quotient", input, got, want)
		}
		serviceMust(t, m.Del(&x))
	}
}

func TestArbitraryFloatDivIndependentIEEEPairsAndAliases(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	values := []float64{0, math.Copysign(0, -1), 1, -1, 2.5, -2.5, math.SmallestNonzeroFloat64, -math.SmallestNonzeroFloat64, math.Float64frombits(1 << 52), -math.Float64frombits(1 << 52), math.MaxFloat64, -math.MaxFloat64, math.Inf(1), math.Inf(-1), math.NaN(), math.Ldexp(1, -53), math.Ldexp(1, -54)}
	rng := rand.New(rand.NewChaCha8([32]byte{47, 0, 58, 8}))
	for i := 0; i < 300; i++ {
		values = append(values, math.Float64frombits(rng.Uint64()))
	}
	for i, xValue := range values {
		partners := values
		if i >= 17 {
			partners = []float64{values[rng.IntN(len(values))], -xValue, xValue}
		}
		for _, yValue := range partners {
			for alias := 0; alias < 3; alias++ {
				var x, y, separate ArbitraryFloat
				serviceMust(t, m.SetDouble(&x, 11, 53, xValue))
				serviceMust(t, m.SetDouble(&y, 11, 53, yValue))
				out := &separate
				if alias == 1 {
					out = &x
				}
				if alias == 2 {
					out = &y
				}
				serviceMust(t, m.Div(ArbitraryFloatNearestEven, &x, &y, out))
				got, e := m.ToDouble(out)
				serviceMust(t, e)
				want := xValue / yValue
				if math.IsNaN(want) {
					if !math.IsNaN(got) {
						t.Fatal("NaN", xValue, yValue, alias, got)
					}
				} else if math.Float64bits(got) != math.Float64bits(want) {
					t.Fatal("independent IEEE arithmetic", xValue, yValue, alias, got, want)
				}
				serviceMust(t, m.Del(&separate))
				serviceMust(t, m.Del(&x))
				serviceMust(t, m.Del(&y))
			}
		}
	}
}
