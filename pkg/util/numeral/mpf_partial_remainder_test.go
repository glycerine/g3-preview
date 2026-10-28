package numeral

// Source: src/util/mpf.cpp:1272-1436.

import (
	"math/big"
	"math/rand/v2"
	"testing"
)

func TestArbitraryFloatPartialRemainderIndependentQuotientOracle(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	rng := rand.New(rand.NewChaCha8([32]byte{12, 72, 14, 36}))
	for _, sb := range []uint32{2, 7, 24, 80, 120} {
		for sample := 0; sample < 24; sample++ {
			hidden := new(big.Int).Lsh(big.NewInt(1), uint(sb-1))
			a := new(big.Int).SetUint64(rng.Uint64())
			a.Mod(a, hidden)
			a.Add(a, hidden)
			b := new(big.Int).SetUint64(rng.Uint64())
			b.Mod(b, hidden)
			b.Add(b, hidden)
			for _, partial := range []bool{false, true} {
				d := int64(sample%int(sb)) - 1
				if partial {
					d = int64(sb) + int64(sample%10)
				}
				for _, sign := range []bool{false, true} {
					var x, y ArbitraryFloat
					serviceMust(t, m.integer.SetString(&x.significand, a.String()))
					serviceMust(t, m.integer.SetString(&y.significand, b.String()))
					x.format = arbitraryFloatFormat(11, sb, sign)
					x.exponent = d - 10
					y.format = arbitraryFloatFormat(11, sb, sample%2 != 0)
					y.exponent = -10
					yBefore := y
					xv := mpfAddSubDyadic(a, x.exponent-int64(sb)+1)
					yv := mpfAddSubDyadic(b, y.exponent-int64(sb)+1)
					quotient := new(big.Rat).Quo(xv, yv)
					var q *big.Rat
					if partial {
						step := mpfAddSubDyadic(big.NewInt(1), d-int64(sb)+1)
						scaled := new(big.Rat).Quo(quotient, step)
						integral := new(big.Int).Quo(scaled.Num(), scaled.Denom())
						q = new(big.Rat).Mul(new(big.Rat).SetInt(integral), step)
					} else {
						q = new(big.Rat).SetInt(mpfIntegralOracle(quotient, ArbitraryFloatNearestEven))
					}
					want := new(big.Rat).Sub(xv, new(big.Rat).Mul(yv, q))
					if sign {
						want.Neg(want)
					}
					want = mpfAddSubOracle(want, sb, ArbitraryFloatNearestEven)
					serviceMust(t, m.PartialRemainder(&x, &y, d, partial))
					text, e := m.integer.String(&x.significand)
					serviceMust(t, e)
					numeral, ok := new(big.Int).SetString(text, 10)
					if !ok {
						t.Fatal(text)
					}
					got := mpfAddSubDyadic(numeral, x.exponent-int64(sb)+1)
					if x.Sign() {
						got.Neg(got)
					}
					if got.Cmp(want) != 0 || y != yBefore {
						t.Fatal("independent quotient/remainder oracle", sb, sample, partial, sign, d, a, b, got, want, x)
					}
					serviceMust(t, m.Del(&y))
					serviceMust(t, m.Del(&x))
				}
			}
		}
	}
}
