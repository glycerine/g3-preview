// Source: src/util/mpf.cpp:2052,2120.
package numeral

import (
	"errors"
	"os"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestArbitraryFloatRoundDefaultsReachSourceUnreachable(t *testing.T) {
	var out *os.File
	if modularSourceDebug {
		d, stream := bindMPNDebugActions(t, "TT")
		out = stream
		serviceMust(t, d.EnableAssertions(false))
	}
	m, _, _ := mpfNumeralHelpersFixture(t)
	invalid := ArbitraryFloatRoundingMode(255)
	for _, tc := range []struct {
		line    string
		sig     int32
		call    func(*ArbitraryFloat) error
		cleanup func() error
	}{{"2052", 20, func(x *ArbitraryFloat) error { return m.Round(invalid, x) }, m.CleanupRound},
		{"2120", 13, func(x *ArbitraryFloat) error { return m.RoundSqrt(invalid, x) }, m.CleanupRoundSqrt}} {
		x := ArbitraryFloat{format: arbitraryFloatFormat(8, 4, false), exponent: 0, significand: NewInt(tc.sig)}
		e := tc.call(&x)
		if modularSourceDebug {
			if !isMPNVerificationThrow(e) {
				t.Fatalf("line %s debug action: %v", tc.line, e)
			}
		} else if !errors.Is(e, diagnostic.ErrDebugNativeUnreachable) {
			t.Fatalf("line %s native guard: %v", tc.line, e)
		}
		serviceMust(t, tc.cleanup())
		serviceMust(t, m.Del(&x))
	}
	if out != nil {
		got, e := os.ReadFile(out.Name())
		serviceMust(t, e)
		const prompt = "(C)ontinue, (A)bort, (S)top, (T)hrow exception, Invoke (G)DB\n"
		want := ""
		for _, line := range []string{"2052", "2120"} {
			want += "ASSERTION VIOLATION\nFile: src/util/mpf.cpp\nLine: " + line + "\nUNREACHABLE CODE WAS REACHED.\n" + prompt
		}
		if string(got) != want {
			t.Fatalf("source diagnostic: got %q, want %q", got, want)
		}
	}
}
