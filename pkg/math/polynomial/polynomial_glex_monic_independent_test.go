package polynomial

import (
	"errors"
	"reflect"
	"strconv"
	"testing"
)

func TestPolynomialGlexMonicCleanupPreservesIndependentLazard(t *testing.T) {
	polynomialGlexMonicIndependentLazard(t, -1)
}
func TestPolynomialGlexMonicIndependentLazardReservationPrefixes(t *testing.T) {
	success := false
	for prefix := int64(0); prefix < 128 && !success; prefix++ {
		t.Run(strconv.FormatInt(prefix, 10), func(t *testing.T) { success = polynomialGlexMonicIndependentLazard(t, prefix) })
		if t.Failed() {
			return
		}
	}
	if !success {
		t.Fatal("monic reservation horizon did not reach success")
	}
}
func polynomialGlexMonicIndependentLazard(t *testing.T, prefix int64) bool {
	t.Helper()
	full := polynomialPSCFullFixture(t)
	m := full.core
	oldOut, e := NewPolynomialRef(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.CleanupLazard()); serviceMust(t, oldOut.Close()) })
	oldInput := polynomialQueryTerms(t, m, []int32{2, 4}, [][]MonomialPower{{{0, 1}}, nil})
	stop := errors.New("independent Lazard deletion")
	reached := false
	handler, e := NewPolynomialDeletionHandler(full.originalBudget, func(_ *PolynomialDeletionHandler, c *MultivariatePolynomialCore, _ Polynomial) error {
		if c.lazardLocal.finishing && !reached {
			reached = true
			return stop
		}
		return nil
	})
	serviceMust(t, e)
	serviceMust(t, m.AddDeletionHandler(handler))
	t.Cleanup(func() { serviceMust(t, m.RemoveDeletionHandler(handler)); serviceMust(t, handler.CloseChecked()) })
	var old polynomialLazardInvocationReceipt
	body := m.seLazardWithReceipt(3, m.unit, oldInput, 0, &oldOut.root, &old)
	if !reached || !errors.Is(body, stop) || !old.frame.active {
		t.Fatal("actual independent source stop missing", body)
	}
	before := *old.frame
	serviceMust(t, m.modular.SetZPUint64(7))
	input := polynomialQueryTerms(t, m, []int32{2, -3}, [][]MonomialPower{{{0, 1}}, nil})
	full.originalBudget.FailAfter(prefix)
	result, e := full.GlexMonic(input)
	full.originalBudget.FailAfter(-1)
	if e != nil && !numericPhysicalReservationRefusal(e) {
		t.Fatal("unexpected monic refusal", e)
	}
	success := e == nil
	if success {
		result = polynomialCaller(t, m, result)
	}
	if success && full.monic.active {
		t.Error("successful monic adopted independent pending scope")
	}
	serviceMust(t, full.CleanupGlexMonic())
	if !reflect.DeepEqual(before, *old.frame) {
		t.Error("monic cleanup consumed independent Lazard")
	}
	if success {
		polynomialQueryExpect(t, m, result, map[[3]uint32]int64{{1, 0, 0}: 1, {}: 2})
	} else {
		result = Polynomial{}
	}
	serviceMust(t, m.cleanupLazardInvocation(&old, true))
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{oldInput, input, oldOut.root.value, result}, nil, oldOut.header+handler.header)
	return success
}
