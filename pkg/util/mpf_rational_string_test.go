// Source: src/util/mpf.cpp:1572-1576; src/util/mpf.h:141.
package util

import (
	"math/big"
	"math/rand/v2"
	"testing"
)

func TestArbitraryFloatRationalStringIndependentDyadic(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	rng := rand.New(rand.NewChaCha8([32]byte{15, 72, 15, 76}))
	for _, sb := range []uint32{2, 24, 53, 120} {
		for sample := 0; sample < 20; sample++ {
			fraction := new(big.Int).SetUint64(rng.Uint64())
			fraction.Lsh(fraction, uint(rng.IntN(80)))
			fraction.Mod(fraction, new(big.Int).Lsh(big.NewInt(1), uint(sb-1)))
			exp := int64(rng.IntN(41)) - 20
			full := new(big.Int).Add(fraction, new(big.Int).Lsh(big.NewInt(1), uint(sb-1)))
			if sample == 0 {
				exp = -127
				full.Set(fraction)
			}
			for _, negative := range []bool{false, true} {
				x := ArbitraryFloat{format: arbitraryFloatFormat(8, sb, negative), exponent: exp}
				serviceMust(t, m.integer.SetString(&x.significand, fraction.String()))
				before := x
				effective := exp
				if exp == -127 {
					effective++
				}
				want := mpfAddSubDyadic(full, effective-int64(sb)+1)
				if negative {
					want.Neg(want)
				}
				var out OwnedString
				serviceMust(t, m.ToRationalString(&x, &out))
				got, e := out.View().String()
				serviceMust(t, e)
				if got != want.RatString() || x != before {
					t.Fatal("independent rational string", sb, sample, negative, got, want)
				}
				out.SourceDestroy()
				serviceMust(t, m.Del(&x))
			}
		}
	}
}
