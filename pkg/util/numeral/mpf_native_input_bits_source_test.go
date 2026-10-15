// Source: src/util/mpf.cpp:114-183.
package numeral

import (
	"math"
	"testing"
)

func TestArbitraryFloatNativeInputIndependentIEEEFieldOracle(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	var out ArbitraryFloat
	defer func() { serviceMust(t, m.Del(&out)) }()
	formats := [...]struct{ ebits, sbits uint32 }{{8, 8}, {8, 24}, {11, 53}, {15, 60}}
	clamp := func(exp int64, ebits uint32) int64 {
		top := int64(1) << (ebits - 1)
		if exp <= -(top - 1) {
			return -(top - 1)
		}
		if exp >= top {
			return top
		}
		return exp
	}
	check := func(name string, raw uint64, inputBits, sourceBits uint32, sign bool, exp int64, sig uint64) {
		t.Helper()
		for _, f := range formats {
			var e error
			if inputBits == 64 {
				e = m.SetDouble(&out, f.ebits, f.sbits, math.Float64frombits(raw))
			} else {
				e = m.SetFloat(&out, f.ebits, f.sbits, math.Float32frombits(uint32(raw)))
			}
			serviceMust(t, e)
			wantSig := sig
			if f.sbits < sourceBits {
				wantSig >>= sourceBits - f.sbits
			} else if f.sbits > sourceBits {
				wantSig <<= f.sbits - sourceBits
			}
			gotSig, e := m.integer.Uint64(&out.significand)
			serviceMust(t, e)
			if out.EBits() != f.ebits || out.SBits() != f.sbits || out.Sign() != sign || out.Exponent() != clamp(exp, f.ebits) || gotSig != wantSig {
				t.Fatalf("%s raw=%#x format=%d/%d: got sign=%v exp=%d sig=%d, want sign=%v exp=%d sig=%d", name, raw, f.ebits, f.sbits, out.Sign(), out.Exponent(), gotSig, sign, clamp(exp, f.ebits), wantSig)
			}
			if inputBits == 64 && f.ebits == 11 && f.sbits == 53 {
				back, e := m.ToDouble(&out)
				serviceMust(t, e)
				if math.Float64bits(back) != raw {
					t.Fatalf("%s double round trip: input=%#x output=%#x", name, raw, math.Float64bits(back))
				}
			}
			if inputBits == 32 && f.ebits == 8 && f.sbits == 24 {
				back, e := m.ToFloat(&out)
				serviceMust(t, e)
				if math.Float32bits(back) != uint32(raw) {
					t.Fatalf("%s float round trip: input=%#x output=%#x", name, raw, math.Float32bits(back))
				}
			}
		}
	}
	for _, raw := range []uint64{0, 1, 0x000fffffffffffff, 0x0010000000000000, 0x7fefffffffffffff, 0x7ff0000000000000, 0x7ff0000000000001, 0x7ff8000000000000, 0x8000000000000000, 0xbff8000000000000, 0xfff0000000000000} {
		sign := raw>>63 != 0
		exp := int64((raw>>52)&0x7ff) - 1023
		sig := raw & 0x000fffffffffffff
		check("double edge", raw, 64, 53, sign, exp, sig)
	}
	for _, raw := range []uint32{0, 1, 0x007fffff, 0x00800000, 0x7f7fffff, 0x7f800000, 0x7f800001, 0x7fc00000, 0x80000000, 0xbfc00000, 0xff800000} {
		sign := raw>>31 != 0
		exp := int64((raw>>23)&0xff) - 127
		sig := uint64(raw & 0x007fffff)
		check("float edge", uint64(raw), 32, 24, sign, exp, sig)
	}
	// A fixed full-width sequence checks exponent/sign/fraction combinations
	// beyond the named boundaries without depending on the implementation.
	seed := uint64(0x9e3779b97f4a7c15)
	for i := 0; i < 256; i++ {
		seed ^= seed << 13
		seed ^= seed >> 7
		seed ^= seed << 17
		raw := seed
		check("double sequence", raw, 64, 53, raw>>63 != 0, int64((raw>>52)&0x7ff)-1023, raw&0x000fffffffffffff)
		bits := uint32(raw >> 32)
		check("float sequence", uint64(bits), 32, 24, bits>>31 != 0, int64((bits>>23)&0xff)-127, uint64(bits&0x007fffff))
	}
}
