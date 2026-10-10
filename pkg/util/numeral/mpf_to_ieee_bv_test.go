// Source: src/util/mpf.cpp:1228-1253; src/util/mpf.h:218.
package numeral

import (
	"math"
	"math/big"
	"math/rand/v2"
	"testing"
)

func TestArbitraryFloatToIEEEBVIndependentEncoding(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	rng := rand.New(rand.NewChaCha8([32]byte{12, 28, 12, 53}))
	for _, width := range [][2]uint32{{3, 2}, {8, 24}, {11, 53}, {17, 120}, {32, 80}} {
		eb, sb := width[0], width[1]
		bias := int64(uint64(1)<<(eb-1)) - 1
		for sample := 0; sample < 25; sample++ {
			fraction := new(big.Int).SetUint64(rng.Uint64())
			fraction.Lsh(fraction, uint(rng.IntN(80)))
			fraction.Mod(fraction, new(big.Int).Lsh(big.NewInt(1), uint(sb-1)))
			exp := int64(rng.Uint64()%uint64(2*bias)) - bias
			if sample == 0 {
				exp = -bias
			}
			for _, sign := range []bool{false, true} {
				x := ArbitraryFloat{format: arbitraryFloatFormat(eb, sb, sign), exponent: exp}
				serviceMust(t, m.integer.SetString(&x.significand, fraction.String()))
				before := x
				out := NewInt(7)
				serviceMust(t, m.ToIEEEBVMPZ(&x, &out))
				want := new(big.Int).SetInt64(exp + bias)
				if sign {
					want.SetBit(want, int(eb), 1)
				}
				want.Lsh(want, uint(sb-1)).Add(want, fraction)
				got, e := m.integer.String(&out)
				serviceMust(t, e)
				if got != want.String() || x != before {
					t.Fatal("independent encoding", width, sample, sign, got, want)
				}
				serviceMust(t, m.integer.Del(&out))
				serviceMust(t, m.Del(&x))
			}
		}
	}
	for _, bits := range []uint64{0, 1, 0x8000000000000000, 0x3ff0000000000000, 0xbff8000000000000, 0x0010000000000000, 0x7fefffffffffffff, 0x7ff0000000000000, 0xfff0000000000000} {
		var x ArbitraryFloat
		serviceMust(t, m.SetDouble(&x, 11, 53, math.Float64frombits(bits)))
		out := NewInt(0)
		serviceMust(t, m.ToIEEEBVMPZ(&x, &out))
		want := new(big.Int).SetUint64(bits)
		got, e := m.integer.String(&out)
		serviceMust(t, e)
		if got != want.String() {
			t.Fatal("IEEE64 encoding", bits, got, want)
		}
		serviceMust(t, m.integer.Del(&out))
		serviceMust(t, m.Del(&x))
	}
}
