package numeral

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestRationalParserDigitSourceAssertionStops(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertions only")
	}
	for _, tc := range []struct {
		name, input string
	}{
		{"numerator", "2"},
		{"exponent", "e2"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, _ := bindMPNDebugActions(t, "")
			m, _, _ := mpfNumeralHelpersFixture(t)
			out := Rational{Num: NewInt(71), Den: NewInt(1)}
			serviceMust(t, d.Close())
			err := m.rational.SetString(&out, tc.input)
			if !errors.Is(err, arena.ErrClosed) || out.Num != NewInt(0) || out.Den != NewInt(1) {
				t.Fatal("assertion must stop before digit arithmetic", tc.name, err, out)
			}
		})
	}
}
