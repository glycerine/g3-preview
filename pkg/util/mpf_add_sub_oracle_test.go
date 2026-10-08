// Source: src/util/mpf.cpp:470-588.
package util

import (
	"math/big"
	"math/rand/v2"
	"testing"
)

func mpfAddSubDyadic(n *big.Int, exponent int64) *big.Rat {
	if exponent >= 0 {
		return new(big.Rat).SetInt(new(big.Int).Lsh(n, uint(exponent)))
	}
	return new(big.Rat).SetFrac(n, new(big.Int).Lsh(big.NewInt(1), uint(-exponent)))
}

// Quantize an exact rational using integer quotient/remainder arithmetic. This
// oracle neither calls the port's rounder nor reproduces its alignment algorithm.
func mpfAddSubOracle(value *big.Rat, sb uint32, mode ArbitraryFloatRoundingMode) *big.Rat {
	if value.Sign() == 0 {
		return new(big.Rat)
	}
	abs := new(big.Rat).Abs(value)
	exponent := abs.Num().BitLen() - abs.Denom().BitLen()
	if abs.Cmp(mpfAddSubDyadic(big.NewInt(1), int64(exponent))) < 0 {
		exponent--
	}
	step := int64(exponent) - int64(sb) + 1
	if minimum := int64(-1022) - int64(sb) + 1; step < minimum {
		step = minimum
	}
	scaled := new(big.Rat).Quo(value, mpfAddSubDyadic(big.NewInt(1), step))
	return mpfAddSubDyadic(mpfIntegralOracle(scaled, mode), step)
}

func TestArbitraryFloatAddSubIndependentWideDirectedOracle(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	rng := rand.New(rand.NewChaCha8([32]byte{47, 58, 120}))
	for _, sb := range []uint32{2, 24, 120} {
		for sample := 0; sample < 40; sample++ {
			var sigs [2]*big.Int
			var exps [2]int64
			var signs [2]bool
			var values [2]*big.Rat
			for i := range sigs {
				sigs[i] = new(big.Int).SetUint64(rng.Uint64())
				sigs[i].Lsh(sigs[i], uint(rng.Uint32()%80))
				sigs[i].Mod(sigs[i], new(big.Int).Lsh(big.NewInt(1), uint(sb-1)))
				exps[i] = int64(rng.IntN(401)) - 200
				signs[i] = rng.IntN(2) != 0
			}
			if sample%4 == 0 {
				// Exact cancellation, near cancellation, and halfway cases.
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
				for _, sub := range []bool{false, true} {
					exact := new(big.Rat).Add(values[0], values[1])
					if sub {
						exact.Sub(values[0], values[1])
					}
					want := mpfAddSubOracle(exact, sb, mode)
					for alias := 0; alias < 3; alias++ {
						var x, y, separate ArbitraryFloat
						for i, v := range []*ArbitraryFloat{&x, &y} {
							var n Int
							serviceMust(t, m.integer.SetString(&n, sigs[i].String()))
							serviceMust(t, m.SetComponents(v, 11, sb, signs[i], exps[i], &n))
							serviceMust(t, m.integer.Del(&n))
						}
						oldX, oldY := x, y // Borrowed identities, never destroyed.
						out := []*ArbitraryFloat{&separate, &x, &y}[alias]
						if sub {
							serviceMust(t, m.Sub(mode, &x, &y, out))
						} else {
							serviceMust(t, m.Add(mode, &x, &y, out))
						}
						q := NewRational()
						serviceMust(t, m.ToRational(out, m.rational, &q))
						n, e := m.integer.String(&q.Num)
						serviceMust(t, e)
						d, e := m.integer.String(&q.Den)
						serviceMust(t, e)
						got, ok := new(big.Rat).SetString(n + "/" + d)
						if !ok || got.Cmp(want) != 0 {
							t.Fatal("exact arithmetic oracle", sb, sample, mode, sub, alias, got, want)
						}
						if alias != 1 && x != oldX || alias != 2 && y != oldY {
							t.Fatal("const operand changed")
						}
						serviceMust(t, m.rational.Del(&q))
						serviceMust(t, m.Del(&separate))
						serviceMust(t, m.Del(&x))
						serviceMust(t, m.Del(&y))
					}
				}
			}
		}
	}
}
