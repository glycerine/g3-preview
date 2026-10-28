package numeral

// Source: src/util/mpf.cpp:1005-1045; src/util/mpf.h:131.

import (
	"math"
	"math/rand/v2"
	"testing"
)

func TestArbitraryFloatSqrtIndependentIEEEAndAlias(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	values := []float64{0, math.Copysign(0, -1), math.Inf(1), math.Inf(-1), math.NaN(), -1, 1, 2, 3, 4, 9, math.SmallestNonzeroFloat64, math.MaxFloat64}
	rng := rand.New(rand.NewChaCha8([32]byte{10, 8, 10, 45}))
	for i := 0; i < 150; i++ {
		values = append(values, math.Float64frombits(rng.Uint64()&math.MaxInt64))
	}
	for _, v := range values {
		for _, alias := range []bool{false, true} {
			var x, separate ArbitraryFloat
			serviceMust(t, m.SetDouble(&x, 11, 53, v))
			out := &separate
			if alias {
				out = &x
			}
			before := x
			serviceMust(t, m.Sqrt(ArbitraryFloatNearestEven, &x, out))
			got, e := m.ToDouble(out)
			serviceMust(t, e)
			want := math.Sqrt(v)
			if math.IsNaN(want) {
				if !math.IsNaN(got) {
					t.Fatal("sqrt NaN", v, got)
				}
			} else if math.Float64bits(got) != math.Float64bits(want) {
				t.Fatal("independent sqrt", v, alias, got, want)
			}
			if !alias && x != before {
				t.Fatal("const source changed")
			}
			serviceMust(t, m.Del(&separate))
			serviceMust(t, m.Del(&x))
		}
	}
}
