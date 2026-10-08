// Source: src/util/mpf.cpp:470-588; src/util/mpf.h:124-125.
package util

import (
	"math"
	"math/rand/v2"
	"testing"
)

func TestArbitraryFloatAddSubIndependentIEEEPairsAndAliases(t *testing.T) {
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
			for _, sub := range []bool{false, true} {
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
					if sub {
						serviceMust(t, m.Sub(ArbitraryFloatNearestEven, &x, &y, out))
					} else {
						serviceMust(t, m.Add(ArbitraryFloatNearestEven, &x, &y, out))
					}
					got, e := m.ToDouble(out)
					serviceMust(t, e)
					want := xValue + yValue
					if sub {
						want = xValue - yValue
					}
					if math.IsNaN(want) {
						if !math.IsNaN(got) {
							t.Fatal("NaN", xValue, yValue, sub, alias, got)
						}
					} else if math.Float64bits(got) != math.Float64bits(want) {
						t.Fatal("independent IEEE arithmetic", xValue, yValue, sub, alias, got, want)
					}
					serviceMust(t, m.Del(&separate))
					serviceMust(t, m.Del(&x))
					serviceMust(t, m.Del(&y))
				}
			}
		}
	}
}

func TestArbitraryFloatAddSubSignedZeroModes(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	for _, mode := range mpfIntegralModes {
		for _, sx := range []bool{false, true} {
			for _, sy := range []bool{false, true} {
				for _, sub := range []bool{false, true} {
					var x, y, out ArbitraryFloat
					serviceMust(t, m.MkZero(11, 53, sx, &x))
					serviceMust(t, m.MkZero(11, 53, sy, &y))
					if sub {
						serviceMust(t, m.Sub(mode, &x, &y, &out))
					} else {
						serviceMust(t, m.Add(mode, &x, &y, &out))
					}
					zero, e := m.IsZero(&out)
					serviceMust(t, e)
					negative := sx && (sy != sub) || mode == ArbitraryFloatTowardNegative && (sx != (sy != sub))
					if !zero || out.Sign() != negative {
						t.Fatal("source signed-zero switch", mode, sx, sy, sub, out)
					}
					serviceMust(t, m.Del(&out))
					serviceMust(t, m.Del(&x))
					serviceMust(t, m.Del(&y))
				}
			}
		}
	}
}
