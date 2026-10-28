package numeral

// Source: src/util/mpf.cpp:748-949.

import (
	"math/big"
	"math/rand/v2"
	"testing"
)

func TestArbitraryFloatFMAIndependentWideDirectedOracle(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	rng := rand.New(rand.NewChaCha8([32]byte{47, 58, 120}))
	for _, sb := range []uint32{4, 24, 120} {
		for sample := 0; sample < 40; sample++ {
			var sigs [3]*big.Int
			var exps [3]int64
			var signs [3]bool
			var values [3]*big.Rat
			for i := range sigs {
				sigs[i] = new(big.Int).SetUint64(rng.Uint64())
				sigs[i].Lsh(sigs[i], uint(rng.Uint32()%80))
				sigs[i].Mod(sigs[i], new(big.Int).Lsh(big.NewInt(1), uint(sb-1)))
				exps[i] = int64(rng.IntN(401)) - 200
				signs[i] = rng.IntN(2) != 0
			}
			if sample%4 == 0 {
				// Equal magnitudes and power-of-two denominators exercise exact quotients.
				exps[1] = exps[0]
				sigs[1].Set(sigs[0])
			} else if sample%4 == 1 {
				exps[1] = exps[0] - int64(sb)
				sigs[1].SetInt64(0)
			}
			if sample >= 32 {
				exps[0] = -1023 + int64(sample%2)
				exps[1] = -1023 + int64(sample%3)
			}
			if exps[1] == -1023 && sigs[1].Sign() == 0 {
				sigs[1].SetInt64(1)
			}
			for i := range sigs {
				full := new(big.Int).Add(sigs[i], new(big.Int).Lsh(big.NewInt(1), uint(sb-1)))
				effectiveExp := exps[i]
				if exps[i] == -1023 {
					full.Set(sigs[i])
					effectiveExp++
				}
				values[i] = mpfAddSubDyadic(full, effectiveExp-int64(sb)+1)
				if signs[i] {
					values[i].Neg(values[i])
				}
			}
			for _, mode := range mpfIntegralModes {
				exact := new(big.Rat).Mul(values[0], values[1])
				exact.Add(exact, values[2])
				want := mpfAddSubOracle(exact, sb, mode)
				for alias := 0; alias < 4; alias++ {
					var x, y, z, separate ArbitraryFloat
					for i, v := range []*ArbitraryFloat{&x, &y, &z} {
						var n Int
						serviceMust(t, m.integer.SetString(&n, sigs[i].String()))
						serviceMust(t, m.SetComponents(v, 11, sb, signs[i], exps[i], &n))
						serviceMust(t, m.integer.Del(&n))
					}
					oldX, oldY, oldZ := x, y, z // Borrowed identities, never destroyed.
					out := []*ArbitraryFloat{&separate, &x, &y, &z}[alias]
					serviceMust(t, m.FMA(mode, &x, &y, &z, out))
					q := NewRational()
					serviceMust(t, m.ToRational(out, m.rational, &q))
					n, e := m.integer.String(&q.Num)
					serviceMust(t, e)
					d, e := m.integer.String(&q.Den)
					serviceMust(t, e)
					got, ok := new(big.Rat).SetString(n + "/" + d)
					if !ok || got.Cmp(want) != 0 {
						t.Fatal("exact arithmetic oracle", sb, sample, mode, alias, got, want)
					}
					if alias != 1 && x != oldX || alias != 2 && y != oldY || alias != 3 && z != oldZ {
						t.Fatal("const operand changed")
					}
					serviceMust(t, m.rational.Del(&q))
					serviceMust(t, m.Del(&separate))
					serviceMust(t, m.Del(&x))
					serviceMust(t, m.Del(&y))
					serviceMust(t, m.Del(&z))
				}
			}
		}
	}
}
