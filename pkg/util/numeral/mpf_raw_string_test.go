package numeral

// Source: src/util/mpf.cpp:1590-1606; src/util/mpf.h:292.

import (
	"strconv"
	"testing"
)

func TestArbitraryFloatRawStringLiteralClassification(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	for _, tc := range []struct {
		exp    int64
		sig    string
		normal bool
	}{{0, "0", true}, {-127, "0", false}, {-127, "3", false}, {128, "0", false}, {128, "7", false}, {1, "123456789012345678901234567890", true}} {
		for _, negative := range []bool{false, true} {
			x := ArbitraryFloat{format: arbitraryFloatFormat(8, 24, negative), exponent: tc.exp}
			serviceMust(t, m.integer.SetString(&x.significand, tc.sig))
			before := x
			got, e := m.RawString(&x)
			serviceMust(t, e)
			sign, kind := "+", "D"
			if negative {
				sign = "-"
			}
			if tc.normal {
				kind = "N"
			}
			want := "[" + sign + " " + tc.sig + " " + strconv.FormatInt(tc.exp, 10) + " " + kind + "]"
			if got != want || x != before {
				t.Fatal("literal raw string", tc, negative, got, want)
			}
			serviceMust(t, m.Del(&x))
		}
	}
}

func TestArbitraryFloatNativeRawStringInterruptedNumericalReadCanRetry(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	x := ArbitraryFloat{format: arbitraryFloatFormat(8, 24, false)}
	serviceMust(t, m.integer.SetString(&x.significand, "2147483648"))
	cell := m.integer.Cell(x.significand.Cell)
	actualSize, e := cell.Backing.ReadWord(m.integer.Allocator, 0)
	serviceMust(t, e)
	serviceMust(t, cell.Backing.writeWord(m.integer.Allocator, 0, cell.Capacity+10))
	_, e = m.RawString(&x)
	if e == nil || m.integer.Natural.Formatting.Active || m.integer.Natural.CharFormatting.active {
		t.Fatal("interrupted numerical conversion retained scratch", e, m.integer.Natural.Formatting, m.integer.Natural.CharFormatting)
	}
	serviceMust(t, cell.Backing.writeWord(m.integer.Allocator, 0, actualSize))
	got, e := m.RawString(&x)
	serviceMust(t, e)
	if got != "[+ 2147483648 0 N]" {
		t.Fatal("retry after interrupted numerical read", got)
	}
	serviceMust(t, m.Del(&x))
}
