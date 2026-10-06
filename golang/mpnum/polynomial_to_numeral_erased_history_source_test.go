// Source: src/math/polynomial/upolynomial.h:401-450.
package mpnum

import "testing"

func TestPolynomialToNumeralZeroReuseErasesHistoryReservation(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	t.Cleanup(func() { full.originalBudget.FailAfter(-1) })
	var output upolyVectorState
	var prior polynomialFactorNumeralVectorReceipt
	serviceMust(t, full.toNumeralVectorVariableWithReceipt(full.core.zero, 0, &output, &prior))
	if output.backing == 0 || output.initialized == 0 {
		t.Fatal("actual retained initialized output", output)
	}
	serviceMust(t, full.cleanupFactorNumeralVector(&prior, false))
	serviceMust(t, full.cleanupFactorNumeralVector(&prior, true))
	history, ok := full.factorNumeralVectorDiagnostic(&prior)
	if !ok {
		t.Fatal("real prior diagnostic")
	}
	raw, size, initialized, used := output.backing, output.size, output.initialized, full.originalBudget.Used()
	full.originalBudget.FailAfter(0)
	var later polynomialFactorNumeralVectorReceipt
	cause := full.toNumeralVectorVariableWithReceipt(full.core.zero, 0, &output, &later)
	if cause != nil {
		t.Fatal("literal zero/reused output has no source allocation", cause)
	}
	if output.backing != raw || output.size != size || output.initialized != initialized || full.originalBudget.Used() != used {
		t.Fatal("erased source path changed owned storage")
	}
	old, ok := full.factorNumeralVectorDiagnostic(&prior)
	if !ok || old != history {
		t.Fatal("prior real history changed", old, history)
	}
	polynomialImpPhysicalAuditNamed(t, full, nil, nil, 0)
	serviceMust(t, full.cleanupFactorNumeralVector(&prior, true))
	serviceMust(t, full.cleanupFactorNumeralVector(&later, false))
	serviceMust(t, full.cleanupFactorNumeralVector(&later, true))
	full.originalBudget.FailAfter(-1)
	serviceMust(t, full.upm.reset(&output))
	full.upm.freeBacking(&output)
}
