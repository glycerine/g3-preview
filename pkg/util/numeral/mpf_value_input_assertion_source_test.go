package numeral

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestArbitraryFloatUint64ComponentsSourceAssertionActions(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertions only")
	}
	for _, tc := range []struct {
		name, line, expression string
		sig                    uint64
		exp                    int64
	}{
		{"significand", "312", "m_mpz_manager.lt(o.significand, m_powers2(sbits-1))", 8, 0},
		{"top exponent", "313", "o.exponent <= mk_top_exp(ebits)", 1, 129},
		{"bottom exponent", "314", "o.exponent >= mk_bot_exp(ebits)", 1, -128},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, log := bindMPNDebugActions(t, "TC")
			b := arena.NewBudget(1 << 25)
			m, e := NewArbitraryFloatManager(serviceConfig(b).Integers)
			serviceMust(t, e)
			var out ArbitraryFloat
			call := func() error { return m.SetComponentsUint64(&out, 8, 4, true, tc.exp, tc.sig) }
			if e := call(); !isMPNVerificationThrow(e) {
				t.Fatal("Throw after source scalar publication", e)
			}
			if out.EBits() != 8 || out.SBits() != 4 || !out.Sign() || out.Exponent() != tc.exp {
				t.Fatal("published source prefix", out)
			}
			if e := call(); e != nil {
				t.Fatal("Continue retains published source fields", e)
			}
			contents, e := os.ReadFile(log.Name())
			serviceMust(t, e)
			if n := strings.Count(string(contents), "File: src/util/mpf.cpp\nLine: "+tc.line+"\n"+tc.expression+"\n"); n != 2 {
				t.Fatal("source assertion site and action count", n, string(contents))
			}
			serviceMust(t, d.EnableAssertions(false))
			if e := call(); e != nil {
				t.Fatal("disabled assertions retain source fields", e)
			}
			serviceMust(t, m.Del(&out))
			serviceMust(t, m.Close())
			if b.Used() != 0 {
				t.Fatal("unreleased component owner", b.Used())
			}
		})
	}
}

func TestArbitraryFloatNativeInputSourceRangeAssertionSites(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertions only")
	}
	for _, tc := range []struct {
		name string
		call func(*ArbitraryFloatManager, *ArbitraryFloat) error
	}{
		{"double", func(m *ArbitraryFloatManager, out *ArbitraryFloat) error { return m.SetDouble(out, 11, 53, 1.25) }},
		{"float", func(m *ArbitraryFloatManager, out *ArbitraryFloat) error { return m.SetFloat(out, 8, 24, 1.25) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, _ := bindMPNDebugActions(t, "")
			b := arena.NewBudget(1 << 25)
			m, e := NewArbitraryFloatManager(serviceConfig(b).Integers)
			serviceMust(t, e)
			var out ArbitraryFloat
			serviceMust(t, d.Close())
			if e := tc.call(m, &out); !errors.Is(e, arena.ErrClosed) {
				t.Fatal("valid source predicate reads closed owner", e)
			}
			if out != (ArbitraryFloat{}) {
				t.Fatal("range assertion precedes output publication", out)
			}
			serviceMust(t, m.Close())
			if b.Used() != 0 {
				t.Fatal("unreleased native-input owner", b.Used())
			}
		})
	}
}
