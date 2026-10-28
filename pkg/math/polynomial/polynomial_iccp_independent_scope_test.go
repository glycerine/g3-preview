package polynomial

// Source: polynomial.cpp:3432-3534. ICCP owns only its reached concrete children.

import (
	"errors"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"reflect"
	"strconv"
	"testing"
)

func TestPolynomialICCPCleanupPreservesIndependentLazard(t *testing.T) {
	polynomialICCPIndependentLazard(t, -1, false)
}

func TestPolynomialICCPIndependentLazardReservationPrefixes(t *testing.T) {
	success := false
	for prefix := int64(0); prefix < 128 && !success; prefix++ {
		t.Run(strconv.FormatInt(prefix, 10), func(t *testing.T) { success = polynomialICCPIndependentLazard(t, prefix, true) })
		if t.Failed() {
			return
		}
	}
	if !success {
		t.Fatal("ICCP reservation horizon did not reach success")
	}
}

func polynomialICCPIndependentLazard(t *testing.T, prefix int64, general bool) bool {
	t.Helper()
	full := polynomialPSCFullFixture(t)
	m := full.core
	oldOut, e := NewPolynomialRef(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.CleanupLazard()); serviceMust(t, oldOut.Close()) })
	out, e := NewPolynomialRef(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, full.CleanupContentPrimitiveParts()); serviceMust(t, out.Close()) })
	oldInput := polynomialQueryTerms(t, m, []int32{2, 4}, [][]MonomialPower{{{0, 1}}, nil})
	powers := [][]MonomialPower{{{0, 1}}, nil}
	if general {
		powers = [][]MonomialPower{{{0, 1}, {1, 1}}, {{1, 1}}}
	}
	input := polynomialQueryTerms(t, m, []int32{6, 9}, powers)
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
	content, e := NewPolynomialRef(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, content.Close()) })
	var scalar unumeral.Int
	t.Cleanup(func() { serviceMust(t, m.integer.Del(&scalar)) })
	full.originalBudget.FailAfter(prefix)
	contentBody := full.IntegerContentPrimitiveParts(input, 0, &scalar, content, out)
	full.originalBudget.FailAfter(-1)
	if contentBody != nil && !numericPhysicalReservationRefusal(contentBody) {
		t.Fatal("unexpected content refusal", contentBody)
	}
	if contentBody == nil && full.iccpPending() {
		t.Error("successful ICCP adopted independent pending scope")
	}
	serviceMust(t, full.CleanupContentPrimitiveParts())
	if !reflect.DeepEqual(before, *old.frame) {
		t.Error("ICCP cleanup consumed independent Lazard")
	}
	if contentBody == nil {
		if scalar != unumeral.NewInt(3) {
			t.Fatal("integer content scalar", scalar)
		}
		if general {
			polynomialQueryExpect(t, m, content.root.value, map[[3]uint32]int64{{0, 1, 0}: 1})
		} else if content.root.value != m.unit {
			t.Fatal("ICCP polynomial content is not one")
		}
		polynomialQueryExpect(t, m, out.root.value, map[[3]uint32]int64{{1, 0, 0}: 2, {}: 3})
	}
	serviceMust(t, m.cleanupLazardInvocation(&old, true))
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{oldInput, input, oldOut.root.value, out.root.value, content.root.value}, []*unumeral.Int{&scalar}, oldOut.header+out.header+content.header+handler.header)
	return contentBody == nil
}
