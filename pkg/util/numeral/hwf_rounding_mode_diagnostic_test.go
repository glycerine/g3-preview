// Source: src/util/hwf.cpp:586-650 (selected Linux path).
package numeral

import (
	"os"
	"testing"
)

func TestHardwareFloatUnsupportedModeSourceDiagnostic(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source debug diagnostic")
	}
	d, output := bindMPNDebugActions(t, "TTT")
	serviceMust(t, d.EnableAssertions(false))
	m := NewHardwareFloatManager()
	qm, _ := rationalManager(t, false)
	q := NewRational()
	serviceMust(t, qm.SetString(&q, "1/3"))
	t.Cleanup(func() { serviceMust(t, qm.Del(&q)) })
	mode := ArbitraryFloatNearestAway
	a, b, result := HardwareFloat{1}, HardwareFloat{3}, HardwareFloat{17}
	for _, call := range []func() error{
		func() error { return m.Add(mode, &a, &b, &result) },
		func() error { return m.RoundToIntegral(mode, &a, &result) },
		func() error { return m.SetRational(&result, mode, qm, &q) },
	} {
		if e := call(); !isMPNVerificationThrow(e) || result.value != 17 {
			t.Fatal("source rounding-mode Throw changed output", e, result)
		}
	}
	got, e := os.ReadFile(output.Name())
	serviceMust(t, e)
	const one = "ASSERTION VIOLATION\nFile: src/util/hwf.cpp\nLine: 642\nUNREACHABLE CODE WAS REACHED.\n" +
		"(C)ontinue, (A)bort, (S)top, (T)hrow exception, Invoke (G)DB\n"
	if string(got) != one+one+one {
		t.Fatalf("source diagnostic: got %q, want %q", got, one+one+one)
	}
}
