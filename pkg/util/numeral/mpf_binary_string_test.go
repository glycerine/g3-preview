// Source: src/util/mpf.cpp:1635-1693; src/util/mpf.h:295.
package numeral

import (
	"fmt"
	utext "github.com/glycerine/g3/pkg/util/text"
	"math/big"
	"math/rand/v2"
	"strconv"
	"strings"
	"testing"
)

func TestArbitraryFloatBinaryStringLiteralBranches(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	for _, tc := range []struct {
		exp          int64
		sig          int32
		negative     bool
		upper, lower uint32
		want         string
	}{
		{128, 1, false, 0, 0, "#b0 #b11111111 #b01 (NaN)"},
		{128, 1, true, 0, 0, "#b0 #b11111111 #b01 (NaN)"},
		{128, 0, false, 0, 0, "#b0 #b11111111 #b001 (+oo)"},
		{128, 0, true, 0, 0, "#b1 #b11111111 #b001 (-oo)"},
		{-127, 0, false, 0, 0, "#b0 #b00000000 #b00 (+zero)"},
		{-127, 0, true, 0, 0, "#b1 #b00000000 #b00 (-zero)"},
		{0, 3, false, 0, 0, "#b0 #b01111111 #b0.11"},
		{0, 3, true, 0, 0, "#b1 #b01111111 #b0.11"},
		{-160, 3, false, 0, 0, "#b0 #b11011111 #b0.11"},
		{-127, 1, false, 0, 0, "#b0 #b00000001 #b0.01"},
		{0, 1, false, 2, 1, "#b0 #b01111111 #b000.00,1"},
		{0, 3, false, ^uint32(0), 0, "#b0 #b01111111 #b .11"},
		{0, 3, false, ^uint32(0) - 1, 0, "#b0 #b01111111 #b.11"},
	} {
		x := ArbitraryFloat{format: arbitraryFloatFormat(8, 3, tc.negative), exponent: tc.exp, significand: NewInt(tc.sig)}
		before := x
		var out utext.OwnedString
		serviceMust(t, m.ToStringBinary(&x, tc.upper, tc.lower, &out))
		actual, e := out.View().String()
		serviceMust(t, e)
		if actual != tc.want || x != before {
			t.Fatal("literal binary", tc, actual)
		}
		out.SourceDestroy()
	}
}

func binaryStringOracle(sig *big.Int, eb, sb, upper, lower uint32, exp int64, negative, denormal bool) string {
	sign := "0"
	if negative {
		sign = "1"
	}
	biased := exp + (int64(1) << (eb - 1)) - 1
	if denormal {
		biased = 1
	}
	mask := uint64(1)<<eb - 1
	exponent := strconv.FormatUint(uint64(biased)&mask, 2)
	exponent = strings.Repeat("0", int(eb)-len(exponent)) + exponent
	n := upper + sb + lower
	size := int(n)
	if sig.BitLen() > size {
		size = sig.BitLen()
	}
	digits := fmt.Sprintf("%0*s", size, sig.Text(2))
	// Text(2) gives "0" for zero; source emits no character when n==0.
	if size == 0 {
		digits = ""
	}
	if lower != 0 && uint64(lower) <= uint64(size) {
		i := size - int(lower)
		digits = digits[:i] + "," + digits[i:]
	}
	cut := sb + lower - 1
	if cut != 0 && uint64(cut) <= uint64(size) {
		i := size - int(cut)
		separator := "."
		if cut == n {
			separator = " ."
		}
		digits = digits[:i] + separator + digits[i:]
	}
	return "#b" + sign + " #b" + exponent + " #b" + digits
}

func TestArbitraryFloatBinaryStringIndependentBits(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	rng := rand.New(rand.NewChaCha8([32]byte{49, 23, 5}))
	for _, sb := range []uint32{2, 3, 24, 53, 120} {
		for sample := 0; sample < 25; sample++ {
			words := make([]byte, (sb+7)/8)
			for i := range words {
				words[i] = byte(rng.Uint32())
			}
			sig := new(big.Int).SetBytes(words)
			// Include significands that exceed sbits: the source continues until zero.
			if sample%3 != 0 {
				sig.And(sig, new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), uint(sb-1)), big.NewInt(1)))
			}
			exp := int64(int(rng.Uint32()%241) - 120)
			denormal := sample == 0
			if denormal {
				exp = -127
				sig.SetBit(sig, 0, 1)
			}
			x := ArbitraryFloat{format: arbitraryFloatFormat(8, sb, sample%2 != 0), exponent: exp}
			serviceMust(t, m.integer.SetString(&x.significand, sig.String()))
			before := x
			for _, extra := range [][2]uint32{{0, 0}, {1, 0}, {0, 1}, {2, 3}, {30, 7}} {
				var out utext.OwnedString
				serviceMust(t, m.ToStringBinary(&x, extra[0], extra[1], &out))
				actual, e := out.View().String()
				serviceMust(t, e)
				expected := binaryStringOracle(sig, 8, sb, extra[0], extra[1], exp, x.Sign(), denormal)
				if actual != expected || x != before {
					t.Fatal("independent binary", sb, sample, extra, actual, expected)
				}
				out.SourceDestroy()
			}
			serviceMust(t, m.Del(&x))
		}
	}
}
