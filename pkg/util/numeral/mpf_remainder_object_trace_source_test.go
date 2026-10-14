package numeral

import (
	"bytes"
	"testing"
)

// src/util/mpf.cpp:1621-1633 reads the first eight bytes of mpf on Linux
// AMD64. The source bitfields occupy bytes 0-3 and mpz::m_val bytes 4-7.
func TestArbitraryFloatRemainderObjectTraceSourceFirstWord(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	for _, tc := range []struct {
		name  string
		value ArbitraryFloat
		want  string
	}{
		{"positive", ArbitraryFloat{format: arbitraryFloatFormat(11, 24, false), significand: NewInt(1 << 23)}, "800000000C000B"},
		{"negative", ArbitraryFloat{format: arbitraryFloatFormat(11, 24, true), significand: NewInt(1 << 23)}, "800000800C000B"},
		{"negative-significand", ArbitraryFloat{format: arbitraryFloatFormat(11, 24, true), significand: NewInt(-1)}, "FFFFFFFF800C000B"},
		{"zero", ArbitraryFloat{format: arbitraryFloatFormat(11, 53, false)}, "1A800B"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			var formatError error
			if err := m.writeFloatObjectTrace(&out, &tc.value, &formatError); err != nil || formatError != nil || out.String() != tc.want {
				t.Fatalf("source object diagnostic: %q, %v, %v; want %q", out.String(), err, formatError, tc.want)
			}
		})
	}
}
