package numeral

// Source: src/util/mpf.cpp:1438-1485.

import (
	"math/big"
	"math/rand/v2"
	"testing"
)

func TestArbitraryFloatRemIndependentWideRationalOracle(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	rng := rand.New(rand.NewChaCha8([32]byte{14, 38, 120}))
	for _, sb := range []uint32{2, 7, 24, 120} {
		for sample := 0; sample < 30; sample++ {
			var sigs [2]*big.Int
			var exps [2]int64
			var signs [2]bool
			var values [2]*big.Rat
			for i := range sigs {
				sigs[i] = new(big.Int).SetUint64(rng.Uint64())
				sigs[i].Lsh(sigs[i], uint(rng.IntN(100)))
				sigs[i].Mod(sigs[i], new(big.Int).Lsh(big.NewInt(1), uint(sb-1)))
				exps[i] = int64(rng.IntN(101)) - 50
				signs[i] = rng.IntN(2) != 0
			}
			if sample%3 == 0 {
				exps[0] = exps[1] + 3*int64(sb) + 5
			}
			if sample >= 26 {
				exps[0] = -1023
				sigs[0].SetInt64(int64(sample - 25))
				sigs[0].Mod(sigs[0], new(big.Int).Lsh(big.NewInt(1), uint(sb-1)))
				exps[1] = -1022
			}
			for i := range sigs {
				full := new(big.Int).Add(sigs[i], new(big.Int).Lsh(big.NewInt(1), uint(sb-1)))
				effective := exps[i]
				if exps[i] == -1023 {
					full.Set(sigs[i])
					effective++
				}
				values[i] = mpfAddSubDyadic(full, effective-int64(sb)+1)
				if signs[i] {
					values[i].Neg(values[i])
				}
			}
			quotient := new(big.Rat).Quo(values[0], values[1])
			integer := mpfIntegralOracle(quotient, ArbitraryFloatNearestEven)
			exact := new(big.Rat).Sub(values[0], new(big.Rat).Mul(values[1], new(big.Rat).SetInt(integer)))
			want := mpfAddSubOracle(exact, sb, ArbitraryFloatNearestEven)
			for alias := 0; alias < 3; alias++ {
				var x, y, separate ArbitraryFloat
				for i, v := range []*ArbitraryFloat{&x, &y} {
					var n Int
					serviceMust(t, m.integer.SetString(&n, sigs[i].String()))
					serviceMust(t, m.SetComponents(v, 11, sb, signs[i], exps[i], &n))
					serviceMust(t, m.integer.Del(&n))
				}
				beforeX, beforeY := x, y
				out := []*ArbitraryFloat{&separate, &x, &y}[alias]
				serviceMust(t, m.Rem(&x, &y, out))
				q := NewRational()
				serviceMust(t, m.ToRational(out, m.rational, &q))
				n, e := m.integer.String(&q.Num)
				serviceMust(t, e)
				d, e := m.integer.String(&q.Den)
				serviceMust(t, e)
				got, ok := new(big.Rat).SetString(n + "/" + d)
				if !ok || got.Cmp(want) != 0 {
					t.Fatal("independent wide remainder", sb, sample, alias, got, want)
				}
				if alias != 1 && x != beforeX || alias != 2 && y != beforeY {
					t.Fatal("const input changed")
				}
				serviceMust(t, m.rational.Del(&q))
				serviceMust(t, m.Del(&separate))
				serviceMust(t, m.Del(&x))
				serviceMust(t, m.Del(&y))
			}
		}
	}
}
