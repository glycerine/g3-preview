package numeral

import (
	"math"
	"os"
	"strings"
	"testing"
)

func TestArbitraryFloatPartialRemainderEntrySourceAssertions(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertions only")
	}
	_, log := bindMPNDebugActions(t, "TTT")
	m, _, _ := mpfNumeralHelpersFixture(t)
	var x, y ArbitraryFloat
	sig := NewInt(8)
	serviceMust(t, m.SetComponents(&x, 11, 4, false, 0, &sig))
	serviceMust(t, m.SetComponents(&y, 11, 4, false, 0, &sig))
	beforeX, beforeY := x, y
	beforeInvocation := m.partialRemainder.invocation
	for _, tc := range []struct {
		diff             int64
		line, expression string
	}{
		{-2, "1276", "-1 <= exp_diff && exp_diff < INT64_MAX"},
		{math.MaxInt64, "1276", "-1 <= exp_diff && exp_diff < INT64_MAX"},
		{15, "1277", "exp_diff < ebits+sbits || partial"},
	} {
		if e := m.PartialRemainder(&x, &y, tc.diff, false); !isMPNVerificationThrow(e) || x != beforeX || y != beforeY || m.partialRemainder.active || m.partialRemainder.invocation != beforeInvocation {
			t.Fatal("entry assertion before partial-remainder scope", tc.diff, e, x, y)
		}
		contents, e := os.ReadFile(log.Name())
		serviceMust(t, e)
		if !strings.Contains(string(contents), "File: src/util/mpf.cpp\nLine: "+tc.line+"\n"+tc.expression+"\n") {
			t.Fatal("source entry assertion site", tc.diff, string(contents))
		}
	}
	serviceMust(t, m.Del(&y))
	serviceMust(t, m.Del(&x))
}

func TestArbitraryFloatPartialRemainderInitialSignificandSourceAssertions(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertions only")
	}
	for _, tc := range []struct {
		name             string
		xSig, ySig       int32
		line, expression string
	}{
		{"x above width", 16, 8, "1287", "m_mpz_manager.lt(x.significand, m_powers2(sbits))"},
		{"x below hidden bit", 7, 8, "1288", "m_mpz_manager.ge(x.significand, m_powers2(sbits - 1))"},
		{"y above width", 8, 16, "1289", "m_mpz_manager.lt(y.significand, m_powers2(sbits))"},
		{"y below hidden bit", 8, 7, "1290", "m_mpz_manager.ge(y.significand, m_powers2(sbits - 1))"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, log := bindMPNDebugActions(t, "T")
			m, _, _ := mpfNumeralHelpersFixture(t)
			var x, y ArbitraryFloat
			xSig, ySig := NewInt(tc.xSig), NewInt(tc.ySig)
			serviceMust(t, m.SetComponents(&x, 11, 4, false, 0, &xSig))
			serviceMust(t, m.SetComponents(&y, 11, 4, false, 0, &ySig))
			beforeX, beforeY := x, y
			if e := m.PartialRemainder(&x, &y, 0, false); !isMPNVerificationThrow(e) || x != beforeX || y != beforeY {
				t.Fatal("source initial significand Throw", e, x, y)
			}
			contents, e := os.ReadFile(log.Name())
			serviceMust(t, e)
			if n := strings.Count(string(contents), "File: src/util/mpf.cpp\nLine: "+tc.line+"\n"+tc.expression+"\n"); n != 1 {
				t.Fatal("source initial significand assertion site", n, string(contents))
			}
			serviceMust(t, m.CleanupPartialRemainder())
			serviceMust(t, m.Del(&y))
			serviceMust(t, m.Del(&x))
		})
	}
}

func TestArbitraryFloatPartialRemainderPartialQuotientSourceAssertion(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertion only")
	}
	_, log := bindMPNDebugActions(t, "T")
	m, _, _ := mpfNumeralHelpersFixture(t)
	var x, y ArbitraryFloat
	sig := NewInt(8)
	serviceMust(t, m.SetComponents(&x, 11, 4, false, 0, &sig))
	serviceMust(t, m.SetComponents(&y, 11, 4, false, 0, &sig))
	beforeX, beforeY := x, y
	if e := m.PartialRemainder(&x, &y, 0, true); !isMPNVerificationThrow(e) || x != beforeX || y != beforeY {
		t.Fatal("partial quotient assertion before shift", e, x, y)
	}
	contents, e := os.ReadFile(log.Name())
	serviceMust(t, e)
	if n := strings.Count(string(contents), "File: src/util/mpf.cpp\nLine: 1313\n0 < N && N < Q_exp && Q_exp < INT_MAX\n"); n != 1 {
		t.Fatal("source partial quotient assertion site", n, string(contents))
	}
	serviceMust(t, m.CleanupPartialRemainder())
	serviceMust(t, m.Del(&y))
	serviceMust(t, m.Del(&x))
}
