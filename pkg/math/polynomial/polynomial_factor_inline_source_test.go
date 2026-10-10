// Source: src/math/polynomial/polynomial.cpp:1613-1642,6822-6830.
package polynomial

import (
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestPolynomialFactorInlineConstantAndResetAcquireNoHistoryReservation(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	f, e := NewPolynomialFactors(full)
	serviceMust(t, e)
	constant := polynomialQueryTerms(t, full.core, []int32{3}, [][]MonomialPower{{}})
	zero := unumeral.NewInt(0)
	for i := 0; i < 4; i++ {
		full.originalBudget.FailAfter(0)
		if e = full.Factor(constant, f, DefaultUPolynomialFactorParams()); e != nil {
			t.Fatalf("inline Factor constant acquired a Core frame at %d: %v", i, e)
		}
		if e = f.SetConstant(&zero); e != nil {
			t.Fatalf("source inline SET0 acquired archive reservation at %d: %v", i, e)
		}
		if e = f.Reset(); e != nil {
			t.Fatalf("source inline reset SET1 acquired archive reservation at %d: %v", i, e)
		}
		if e = full.Factor(full.core.zero, f, DefaultUPolynomialFactorParams()); e != nil {
			t.Fatalf("zero wrapper acquired metadata reservation at %d: %v", i, e)
		}
		full.originalBudget.FailAfter(-1)
	}
	serviceMust(t, f.CloseChecked())
}

func TestPolynomialFactorZeroErasesCoreWithUnrelatedRetainedCaller(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	p := polynomialQueryTerms(t, full.core, []int32{1, 2, 1}, [][]MonomialPower{{}, {{0, 1}}, {{0, 2}}})
	oldOutput, e := NewPolynomialFactors(full)
	serviceMust(t, e)
	f, e := NewPolynomialFactors(full)
	serviceMust(t, e)
	full.core.config.Checkpoint = func() bool { return false }
	if full.Factor(p, oldOutput, DefaultUPolynomialFactorParams()) == nil {
		t.Fatal("actual older caller refusal absent")
	}
	old := full.factorRoot
	body, source, own := old.frame.bodyCause, old.frame.sourceCause, old.frame.cleanupCause
	full.originalBudget.FailAfter(0)
	e = full.Factor(full.core.zero, f, DefaultUPolynomialFactorParams())
	full.originalBudget.FailAfter(-1)
	if e != nil || f.constant != unumeral.NewInt(0) || full.factorRoot != old || old.frame.bodyCause != body || old.frame.sourceCause != source || old.frame.cleanupCause != own {
		t.Fatal("erased zero branch touched unrelated caller/allocated metadata", e)
	}
	serviceMust(t, full.CleanupFactor())
	serviceMust(t, f.CloseChecked())
	serviceMust(t, oldOutput.CloseChecked())
}
