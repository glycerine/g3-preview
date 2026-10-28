package numeral

// Source: src/util/mpff.cpp:1226-1321; independent decimal digit model.

import (
	"math/big"
	"strconv"
	"testing"
)

func fastDecimalModel(m *FastFloatManager, v *FastFloat, digits uint32) string {
	sig := m.sig(v)
	num := new(big.Int)
	for i := m.precision; i > 0; {
		i--
		num.Lsh(num, 32)
		num.Add(num, new(big.Int).SetUint64(uint64(sig.read(i))))
	}
	exp := int64(v.exponent)
	den := big.NewInt(1)
	if exp < 0 {
		den.Lsh(den, uint(-exp))
	} else {
		num.Lsh(num, uint(exp))
	}
	whole, rem := new(big.Int), new(big.Int)
	whole.QuoRem(num, den, rem)
	result := whole.String() + "."
	if v.negative() {
		result = "-" + result
	}
	ten := big.NewInt(10)
	for i := uint32(0); rem.Sign() != 0 && i < digits; i++ {
		rem.Mul(rem, ten)
		q := new(big.Int)
		q.QuoRem(rem, den, rem)
		result += strconv.FormatInt(q.Int64(), 10)
	}
	if rem.Sign() != 0 {
		result += "?"
	}
	return result
}

func TestFastFloatDecimalIndependentBinaryValueModel(t *testing.T) {
	type sampleValue struct {
		num int32
		den uint32
	}
	for _, precision := range []uint32{2, 4, 16} {
		m := fastManager(t, precision)
		samples := []sampleValue{{1, 2}, {-5, 8}, {1, 3}}
		seed := uint32(0x6d2b79f5)
		for sample := 0; sample < 96; sample++ {
			seed ^= seed << 13
			seed ^= seed >> 17
			seed ^= seed << 5
			num := int32(seed%101) - 50
			if num == 0 {
				num = 1
			}
			den := uint32(2 + (seed>>8)%97)
			samples = append(samples, sampleValue{num, den})
		}
		for sample, pair := range samples {
			var value FastFloat
			serviceMust(t, m.SetFraction32(&value, pair.num, pair.den))
			integer, e := m.IsInteger(&value)
			serviceMust(t, e)
			if !integer {
				for _, digits := range []uint32{0, 1, 3, 12} {
					var dst [96]byte
					n, e := m.FormatDecimalInto(&value, digits, 512, dst[:])
					serviceMust(t, e)
					if got, want := string(dst[:n]), fastDecimalModel(m, &value, digits); got != want {
						t.Fatalf("precision=%d sample=%d %d/%d digits=%d: got %q, want %q", precision, sample, pair.num, pair.den, digits, got, want)
					}
				}
			}
			serviceMust(t, m.Del(&value))
		}
	}
}
