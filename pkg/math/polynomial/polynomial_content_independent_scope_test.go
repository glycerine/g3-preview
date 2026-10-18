// Source: polynomial.cpp:3344-3359. Integer content has no PSC/Lazard child.
package polynomial

import (
	"errors"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"reflect"
	"strconv"
	"testing"
)

func TestPolynomialIntegerContentCleanupPreservesIndependentLazard(t *testing.T) {
	polynomialIntegerContentIndependentLazard(t, -1)
}

func TestPolynomialIntegerContentIndependentLazardReservationPrefixes(t *testing.T) {
	success := false
	for prefix := int64(0); prefix < 128 && !success; prefix++ {
		t.Run(strconv.FormatInt(prefix, 10), func(t *testing.T) { success = polynomialIntegerContentIndependentLazard(t, prefix) })
		if t.Failed() {
			return
		}
	}
	if !success {
		t.Fatal("content reservation horizon did not reach success")
	}
}

func polynomialIntegerContentIndependentLazard(t *testing.T, prefix int64) bool {
	t.Helper()
	full := polynomialPSCFullFixture(t)
	m := full.core
	oldOut, e := NewPolynomialRef(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.CleanupLazard()); serviceMust(t, oldOut.Close()) })
	out, e := NewPolynomialRef(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.CleanupIntegerContent()); serviceMust(t, out.Close()) })
	oldInput := polynomialQueryTerms(t, m, []int32{2, 4}, [][]MonomialPower{{{0, 1}}, nil})
	input := polynomialQueryTerms(t, m, []int32{6, 9}, [][]MonomialPower{{{0, 1}}, nil})
	stop := errors.New("independent Lazard source deletion")
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
	var scalar unumeral.Int
	t.Cleanup(func() { serviceMust(t, m.integer.Del(&scalar)) })
	full.originalBudget.FailAfter(prefix)
	contentBody := m.IntegerContent(input, &scalar, out)
	full.originalBudget.FailAfter(-1)
	if contentBody != nil && !numericPhysicalReservationRefusal(contentBody) {
		t.Fatal("unexpected content refusal", contentBody)
	}
	if contentBody == nil && m.contentLocal.active {
		t.Error("successful content adopted independent pending scope")
	}
	serviceMust(t, m.CleanupIntegerContent())
	if !reflect.DeepEqual(before, *old.frame) {
		t.Error("integer content cleanup consumed independent Lazard")
	}
	if contentBody == nil {
		if scalar != unumeral.NewInt(3) {
			t.Fatal("integer content scalar", scalar)
		}
		polynomialQueryExpect(t, m, out.root.value, map[[3]uint32]int64{{1, 0, 0}: 2, {}: 3})
	}
	serviceMust(t, m.cleanupLazardInvocation(&old, true))
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{oldInput, input, oldOut.root.value, out.root.value}, []*unumeral.Int{&scalar}, oldOut.header+out.header+handler.header)
	return contentBody == nil
}
