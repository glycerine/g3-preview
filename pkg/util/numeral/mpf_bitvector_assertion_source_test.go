package numeral

import (
	"math"
	"os"
	"strings"
	"testing"
)

func TestArbitraryFloatToMPZSourceAssertionAction(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertion only")
	}
	_, log := bindMPNDebugActions(t, "TC")
	m, _, _ := mpfNumeralHelpersFixture(t)
	x := ArbitraryFloat{format: arbitraryFloatFormat(11, 2, false), exponent: 4294967297, significand: NewInt(6)}
	out := NewInt(7)
	if e := m.ToMPZ(&x, m.integer, &out); !isMPNVerificationThrow(e) || out != NewInt(7) || m.toMPZ.active {
		t.Fatal("source Throw before integer COPY", e, out)
	}
	serviceMust(t, m.ToMPZ(&x, m.integer, &out))
	if out != NewInt(6) {
		t.Fatal("Continue retains source signed32 exponent narrowing", out)
	}
	contents, e := os.ReadFile(log.Name())
	serviceMust(t, e)
	if n := strings.Count(string(contents), "File: src/util/mpf.cpp\nLine: 1172\nx.exponent < INT_MAX\n"); n != 2 {
		t.Fatal("source to_mpz assertion site", n, string(contents))
	}
}

func TestArbitraryFloatToSBVMPQSourceAssertionSites(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertions only")
	}
	for _, tc := range []struct {
		name, line, expression string
		make                   func(*testing.T, *ArbitraryFloatManager, *ArbitraryFloat)
	}{
		{"nan before local construction", "1184", "!is_nan(x) && !is_inf(x)", func(t *testing.T, m *ArbitraryFloatManager, x *ArbitraryFloat) {
			serviceMust(t, m.MkNaN(8, 24, x, nil))
		}},
		{"unpacked exponent", "1192", "t.exponent() < INT_MAX", func(t *testing.T, m *ArbitraryFloatManager, x *ArbitraryFloat) {
			sig := NewInt(1)
			serviceMust(t, m.SetComponents(x, 33, 4, false, math.MaxInt32, &sig))
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, log := bindMPNDebugActions(t, "T")
			m, _, _ := mpfNumeralHelpersFixture(t)
			var x ArbitraryFloat
			tc.make(t, m, &x)
			out := NewRational()
			serviceMust(t, m.rational.SetInt32(&out, 7))
			before := out
			if e := m.ToSBVMPQ(ArbitraryFloatNearestEven, &x, &out); !isMPNVerificationThrow(e) || out != before {
				t.Fatal("source bit-vector rational Throw", e, out)
			}
			contents, e := os.ReadFile(log.Name())
			serviceMust(t, e)
			if n := strings.Count(string(contents), "File: src/util/mpf.cpp\nLine: "+tc.line+"\n"+tc.expression+"\n"); n != 1 {
				t.Fatal("source bit-vector rational assertion site", n, string(contents))
			}
			serviceMust(t, m.CleanupToSBVMPQ())
			serviceMust(t, m.rational.Del(&out))
			serviceMust(t, m.Del(&x))
		})
	}
}
