package numeral

import (
	"os"
	"strings"
	"testing"
)

func TestArbitraryFloatRemFormatSourceActions(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertions only")
	}
	d, log := bindMPNDebugActions(t, "TC")
	m, _, _ := mpfNumeralHelpersFixture(t)
	var x, y, out ArbitraryFloat
	serviceMust(t, m.SetDouble(&x, 8, 3, 1.5))
	serviceMust(t, m.MkInf(11, 53, false, &y))
	serviceMust(t, m.SetDouble(&out, 8, 3, 9))
	before := out
	if e := m.Rem(&x, &y, &out); !isMPNVerificationThrow(e) || out != before || m.rem.active {
		t.Fatal("format assertion before trace and output", e, out)
	}
	for _, label := range []string{"Continue", "disabled"} {
		if label == "disabled" {
			serviceMust(t, d.EnableAssertions(false))
		}
		serviceMust(t, m.Rem(&x, &y, &out))
		if out != x {
			t.Fatal(label, "infinite divisor copies x", out, x)
		}
	}
	contents, e := os.ReadFile(log.Name())
	serviceMust(t, e)
	if n := strings.Count(string(contents), "File: src/util/mpf.cpp\nLine: 1439\nx.sbits == y.sbits && x.ebits == y.ebits\n"); n != 2 {
		t.Fatal("source rem format assertion site and action count", n, string(contents))
	}
	serviceMust(t, m.Del(&out))
	serviceMust(t, m.Del(&y))
	serviceMust(t, m.Del(&x))
}
