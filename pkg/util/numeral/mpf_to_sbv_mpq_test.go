package numeral

// Source: src/util/mpf.cpp:1183-1226.

import (
	"math/big"
	"math/rand/v2"
	"testing"
)

func TestArbitraryFloatToSBVMPQIndependentRoundingOracle(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	rng := rand.New(rand.NewChaCha8([32]byte{11, 83, 12, 26}))
	for _, sb := range []uint32{2, 24, 120} {
		for sample := 0; sample < 30; sample++ {
			fraction := new(big.Int).SetUint64(rng.Uint64())
			fraction.Lsh(fraction, uint(rng.IntN(100)))
			fraction.Mod(fraction, new(big.Int).Lsh(big.NewInt(1), uint(sb-1)))
			exp := int64(rng.IntN(241)) - 120
			if sample < 6 {
				exp = int64(sample%3) - 1
				fraction.SetInt64(0)
			}
			if sample >= 27 {
				exp = -1023
				fraction.SetInt64(int64(sample - 26))
				fraction.Mod(fraction, new(big.Int).Lsh(big.NewInt(1), uint(sb-1)))
			}
			full := new(big.Int).Add(fraction, new(big.Int).Lsh(big.NewInt(1), uint(sb-1)))
			effective := exp
			if exp == -1023 {
				effective++
				full.Set(fraction)
			}
			for _, sign := range []bool{false, true} {
				for _, mode := range mpfIntegralModes {
					var x ArbitraryFloat
					serviceMust(t, m.integer.SetString(&x.significand, fraction.String()))
					x.format = arbitraryFloatFormat(11, sb, sign)
					x.exponent = exp
					before := x
					exact := mpfAddSubDyadic(full, effective-int64(sb)+1)
					if sign {
						exact.Neg(exact)
					}
					want := mpfIntegralOracle(exact, mode)
					out := NewRational()
					serviceMust(t, m.ToSBVMPQ(mode, &x, &out))
					got, e := m.integer.String(&out.Num)
					serviceMust(t, e)
					if got != want.String() || out.Den != NewInt(1) || x != before {
						t.Fatal("independent integer rounding", sb, sample, sign, mode, got, want, out.Den)
					}
					serviceMust(t, m.rational.Del(&out))
					serviceMust(t, m.Del(&x))
				}
			}
		}
	}
}
