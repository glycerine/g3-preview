// Source: src/util/mpf.cpp:1590-1606; src/util/mpf.h:292.
package util

import (
	utext "github.com/glycerine/g3/pkg/util/text"
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
			var out utext.OwnedString
			serviceMust(t, m.ToStringRaw(&x, &out))
			got, e := out.View().String()
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
			out.SourceDestroy()
			serviceMust(t, m.Del(&x))
		}
	}
}
