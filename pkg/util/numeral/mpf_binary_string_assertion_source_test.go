package numeral

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func TestArbitraryFloatBinaryStringSignificandSourceAssertion(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertion only")
	}
	d, log := bindMPNDebugActions(t, "T")
	m, _, _ := mpfNumeralHelpersFixture(t)
	var x ArbitraryFloat
	var sig Int
	serviceMust(t, m.integer.SetString(&sig, "-18446744073709551617"))
	serviceMust(t, m.SetComponents(&x, 11, 120, false, 0, &sig))
	serviceMust(t, m.integer.Del(&sig))
	before := x
	var out utext.OwnedString
	if e := m.ToStringBinary(&x, 0, 0, &out); !isMPNVerificationThrow(e) || x != before || !out.Closed {
		t.Fatal("negative heap significand source Throw", e, x, out)
	}
	contents, e := os.ReadFile(log.Name())
	serviceMust(t, e)
	if n := strings.Count(string(contents), "File: src/util/mpf.cpp\nLine: 1654\nm_mpz_manager.is_nonneg(sig(x))\n"); n != 1 {
		t.Fatal("source binary-string assertion site", n, string(contents))
	}
	serviceMust(t, m.CleanupBinaryString())
	serviceMust(t, m.Del(&x))
	serviceMust(t, m.SetDouble(&x, 11, 53, 1.5))
	serviceMust(t, d.Close())
	var later utext.OwnedString
	if e := m.ToStringBinary(&x, 0, 0, &later); !errors.Is(e, arena.ErrClosed) || !later.Closed {
		t.Fatal("valid general input reads closed source debug owner", e, later)
	}
	serviceMust(t, m.CleanupBinaryString())
	serviceMust(t, m.Del(&x))
}
