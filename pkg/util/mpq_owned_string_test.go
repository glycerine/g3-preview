// Source: src/util/mpq.cpp:123-127; src/util/mpz.cpp:1432-1435.
package util

import (
	utext "github.com/glycerine/g3/pkg/util/text"
	"testing"
)

func TestRationalOwnedStringLiteralNumerals(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	for _, tc := range [][2]string{{"0", "1"}, {"-7", "1"}, {"7", "3"}, {"0", "0"}, {"-123456789012345678901234567890", "98765432109876543210987654321"}, {"9", "-3"}} {
		a := NewRational()
		serviceMust(t, m.integer.SetString(&a.Num, tc[0]))
		serviceMust(t, m.integer.SetString(&a.Den, tc[1]))
		before := a
		var out utext.OwnedString
		serviceMust(t, m.rational.ToOwnedString(&a, &out))
		got, e := out.View().String()
		serviceMust(t, e)
		want := tc[0]
		if tc[1] != "1" {
			want += "/" + tc[1]
		}
		if got != want || a != before {
			t.Fatal("literal rational string", tc, got, want)
		}
		out.SourceDestroy()
		serviceMust(t, m.rational.Del(&a))
	}
}
