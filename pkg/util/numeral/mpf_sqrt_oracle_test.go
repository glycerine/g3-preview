package numeral

// Source: src/util/mpf.cpp:1005-1045.

import (
	"math/big"
	"math/rand/v2"
	"testing"
)

// Compare exact squares with the halfway point; no approximate square root or
// port rounding/normalization code participates in this oracle.
func mpfSqrtOracle(value *big.Rat, sb uint32, mode ArbitraryFloatRoundingMode) *big.Rat {
	if value.Sign() == 0 {
		return new(big.Rat)
	}
	exponent := value.Num().BitLen() - value.Denom().BitLen()
	if value.Cmp(mpfAddSubDyadic(big.NewInt(1), int64(exponent))) < 0 {
		exponent--
	}
	step := int64(exponent>>1) - int64(sb) + 1
	minimum := int64(-1022) - int64(sb) + 1
	if step < minimum {
		step = minimum
	}
	scaled := new(big.Rat).Quo(value, mpfAddSubDyadic(big.NewInt(1), 2*step))
	integral := new(big.Int).Quo(scaled.Num(), scaled.Denom())
	q := new(big.Int).Sqrt(integral)
	square := new(big.Int).Mul(q, q)
	square.Mul(square, scaled.Denom())
	exact := square.Cmp(scaled.Num()) == 0
	increment := false
	switch mode {
	case ArbitraryFloatTowardPositive:
		increment = !exact
	case ArbitraryFloatNearestEven, ArbitraryFloatNearestAway:
		midpoint := new(big.Int).Lsh(q, 1)
		midpoint.Add(midpoint, big.NewInt(1))
		midpoint.Mul(midpoint, midpoint)
		midpoint.Mul(midpoint, scaled.Denom())
		cmp := new(big.Int).Lsh(scaled.Num(), 2).Cmp(midpoint)
		increment = cmp > 0 || cmp == 0 && (mode == ArbitraryFloatNearestAway || q.Bit(0) != 0)
	}
	if increment {
		q.Add(q, big.NewInt(1))
	}
	return mpfAddSubDyadic(q, step)
}

func TestArbitraryFloatSqrtIndependentWideDirectedOracle(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	rng := rand.New(rand.NewChaCha8([32]byte{10, 8, 104, 5}))
	for _, sb := range []uint32{2, 4, 24, 120} {
		for sample := 0; sample < 30; sample++ {
			sig := new(big.Int).SetUint64(rng.Uint64())
			sig.Lsh(sig, uint(rng.Uint32()%80))
			sig.Mod(sig, new(big.Int).Lsh(big.NewInt(1), uint(sb-1)))
			exponent := int64(rng.IntN(401)) - 200
			full := new(big.Int).Add(sig, new(big.Int).Lsh(big.NewInt(1), uint(sb-1)))
			effective := exponent
			if sample >= 25 {
				exponent = -1023
				effective = -1022
				full.Set(sig)
			}
			value := mpfAddSubDyadic(full, effective-int64(sb)+1)
			for _, mode := range mpfIntegralModes {
				for _, alias := range []bool{false, true} {
					var n Int
					serviceMust(t, m.integer.SetString(&n, sig.String()))
					var x, separate ArbitraryFloat
					serviceMust(t, m.SetComponents(&x, 11, sb, false, exponent, &n))
					serviceMust(t, m.integer.Del(&n))
					before := x
					out := &separate
					if alias {
						out = &x
					}
					serviceMust(t, m.Sqrt(mode, &x, out))
					q := NewRational()
					serviceMust(t, m.ToRational(out, m.rational, &q))
					num, e := m.integer.String(&q.Num)
					serviceMust(t, e)
					den, e := m.integer.String(&q.Den)
					serviceMust(t, e)
					got, ok := new(big.Rat).SetString(num + "/" + den)
					want := mpfSqrtOracle(value, sb, mode)
					if !ok || got.Cmp(want) != 0 || out.Sign() {
						t.Fatal("independent exact-square oracle", sb, sample, mode, alias, got, want)
					}
					if !alias && x != before {
						t.Fatal("const input changed")
					}
					serviceMust(t, m.rational.Del(&q))
					serviceMust(t, m.Del(&separate))
					serviceMust(t, m.Del(&x))
				}
			}
		}
	}
}
