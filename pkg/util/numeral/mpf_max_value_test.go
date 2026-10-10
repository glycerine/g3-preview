// Source: src/util/mpf.cpp:1898-1904; src/util/mpf.h:201.
package numeral

import (
	"math"
	"math/big"
	"testing"
)

func TestArbitraryFloatMkMaxValueIndependentFormat(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	for _, eb := range []uint32{2, 8, 11, 32, 63} {
		for _, sb := range []uint32{2, 24, 53, 80, 120} {
			for _, sign := range []bool{false, true} {
				x := ArbitraryFloat{format: arbitraryFloatFormat(3, 4, !sign), exponent: -7, significand: NewInt(9)}
				var receipt ArbitraryFloatValueCopyReceipt
				serviceMust(t, m.MkMaxValue(eb, sb, sign, &x, &receipt))
				want := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), uint(sb-1)), big.NewInt(1))
				got, e := m.integer.String(&x.significand)
				serviceMust(t, e)
				if got != want.String() || x.Exponent() != int64((uint64(1)<<(eb-1))-1) || x.EBits() != eb || x.SBits() != sb || x.Sign() != sign {
					t.Fatal("maximum format", eb, sb, sign, x, got, want)
				}
				if eb == 11 && sb == 53 {
					v, e := m.ToDouble(&x)
					serviceMust(t, e)
					want := math.MaxFloat64
					if sign {
						want = -want
					}
					if v != want {
						t.Fatal("maximum IEEE64", v, want)
					}
				}
				serviceMust(t, m.CleanupValueCopy(&receipt, false))
				serviceMust(t, m.Del(&x))
				serviceMust(t, m.CleanupValueCopy(&receipt, true))
			}
		}
	}
}

func TestArbitraryFloatMkMaxValueWarmInline(t *testing.T) {
	m, r, b := mpfNumeralHelpersFixture(t)
	var x ArbitraryFloat
	serviceMust(t, m.MkMaxValue(8, 24, false, &x, nil))
	used, state := b.Used(), r.State()
	b.FailAfter(0)
	serviceMust(t, m.MkMaxValue(8, 24, true, &x, nil))
	if b.Used() != used || r.State() != state || !x.Sign() || x.exponent != 127 || x.significand.Val != (1<<23)-1 {
		t.Fatal("warm constructor", x, b.Used(), used)
	}
	b.FailAfter(-1)
	serviceMust(t, m.Del(&x))
}
