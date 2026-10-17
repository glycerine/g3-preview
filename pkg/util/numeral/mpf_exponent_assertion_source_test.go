package numeral

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestArbitraryFloatExponentBoundSourceAssertionSites(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertions only")
	}
	d, log := bindMPNDebugActions(t, "TTTT")
	m, _, _ := mpfNumeralHelpersFixture(t)
	for _, tc := range []struct {
		name             string
		get              func(uint32) (int64, error)
		arg              uint32
		line, expression string
	}{
		{"bottom", m.MkBotExp, 1, "1830", "ebits >= 2"},
		{"top", m.MkTopExp, 1, "1835", "ebits >= 2"},
		{"minimum", m.MkMinExp, 0, "1840", "ebits > 0"},
		{"maximum", m.MkMaxExp, 0, "1846", "ebits > 0"},
	} {
		if _, e := tc.get(tc.arg); !isMPNVerificationThrow(e) {
			t.Fatal(tc.name, "source Throw before power lookup", e)
		}
		contents, e := os.ReadFile(log.Name())
		serviceMust(t, e)
		if n := strings.Count(string(contents), "File: src/util/mpf.cpp\nLine: "+tc.line+"\n"+tc.expression+"\n"); n != 1 {
			t.Fatal(tc.name, "source exponent assertion site", n, string(contents))
		}
	}
	serviceMust(t, d.EnableAssertions(false))
	top, e := m.MkTopExp(1)
	serviceMust(t, e)
	bottom, e := m.MkBotExp(1)
	serviceMust(t, e)
	if top != 1 || bottom != 0 {
		t.Fatal("disabled source assertions retain width-one exponent arithmetic", top, bottom)
	}
	serviceMust(t, d.Close())
	for _, get := range []func(uint32) (int64, error){m.MkBotExp, m.MkTopExp, m.MkMinExp, m.MkMaxExp} {
		if _, e := get(2); !errors.Is(e, arena.ErrClosed) {
			t.Fatal("valid width reads closed source assertion owner", e)
		}
	}
}
