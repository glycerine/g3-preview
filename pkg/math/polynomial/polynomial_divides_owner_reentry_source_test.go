// Source: src/math/polynomial/polynomial.cpp:5154-5194,2346-2370.
package polynomial

import (
	"errors"
	"testing"
)

func TestPolynomialDividesWholeOwnerCheckpointRefusalBeforeMutation(t *testing.T) {
	full := polynomialArithmeticDirectNumericFixture(t)
	c := full.core
	calls := 0
	c.config.Checkpoint = func() bool {
		calls++
		closing, stage, used, stats := full.closing, full.cleanupStage, c.integer.Budget().Used(), c.memory.Stats()
		e := full.CloseChecked()
		if !errors.Is(e, ErrPolynomialPending) {
			t.Errorf("executing source checkpoint close = %v", e)
		}
		if full.closing != closing || full.cleanupStage != stage || c.integer.Budget().Used() != used || c.memory.Stats() != stats {
			t.Errorf("rejected source close mutated real owner: closing %v/%v stage %d/%d", closing, full.closing, stage, full.cleanupStage)
		}
		return false
	}
	_, cause := c.Divides(c.unit, c.unit)
	if !errors.Is(cause, ErrUPolynomialCanceled) || calls != 1 {
		t.Fatalf("actual source checkpoint %d %v", calls, cause)
	}
	c.config.Checkpoint = func() bool { return true }
	serviceMust(t, c.CleanupDivides())
	polynomialImpPhysicalAudit(t, full)
}
