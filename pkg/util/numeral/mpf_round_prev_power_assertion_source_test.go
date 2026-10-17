package numeral

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestArbitraryFloatRoundWidthSourceAssertion(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertion only")
	}
	d, log := bindMPNDebugActions(t, "T")
	m, _, _ := mpfNumeralHelpersFixture(t)
	x := ArbitraryFloat{format: arbitraryFloatFormat(8, 3, false), exponent: 0, significand: NewInt(128)}
	before := x
	if e := m.Round(ArbitraryFloatNearestEven, &x); !isMPNVerificationThrow(e) || x != before {
		t.Fatal("Round width assertion before exponent bounds", e, x)
	}
	contents, e := os.ReadFile(log.Name())
	serviceMust(t, e)
	if n := strings.Count(string(contents), "File: src/util/mpf.cpp\nLine: 1972\nm_mpz_manager.lt(o.significand, p_m3)\n"); n != 1 {
		t.Fatal("source Round width assertion site", n, string(contents))
	}
	serviceMust(t, m.CleanupRound())
	x.significand = NewInt(0)
	serviceMust(t, d.Close())
	if e := m.Round(ArbitraryFloatNearestEven, &x); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("valid Round operand reads closed assertion owner", e)
	}
	serviceMust(t, m.CleanupRound())
}

func TestArbitraryFloatPrevPowerSpecialSourceActions(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertion only")
	}
	d, log := bindMPNDebugActions(t, "TC")
	m, _, _ := mpfNumeralHelpersFixture(t)
	var x ArbitraryFloat
	serviceMust(t, m.MkNaN(8, 24, &x, nil))
	if _, e := m.PrevPowerOfTwo(&x); !isMPNVerificationThrow(e) {
		t.Fatal("NaN source Throw", e)
	}
	serviceMust(t, m.MkInf(8, 24, false, &x))
	if got, e := m.PrevPowerOfTwo(&x); e != nil || got != 151 {
		t.Fatal("Continue follows positive-infinity source formula", got, e)
	}
	contents, e := os.ReadFile(log.Name())
	serviceMust(t, e)
	if n := strings.Count(string(contents), "File: src/util/mpf.cpp\nLine: 2134\n!is_nan(a) && !is_pinf(a) && !is_ninf(a)\n"); n != 2 {
		t.Fatal("source previous-power assertion site and actions", n, string(contents))
	}
	serviceMust(t, d.EnableAssertions(false))
	if got, e := m.PrevPowerOfTwo(&x); e != nil || got != 151 {
		t.Fatal("disabled assertion retains source formula", got, e)
	}
	serviceMust(t, m.Del(&x))
}
