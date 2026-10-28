package polynomial

// Source: polynomial.cpp:3675-3767, concrete CRA reconstruction.

import (
	"errors"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"reflect"
	"strconv"
	"testing"
)

func TestPolynomialCRACleanupPreservesIndependentLazard(t *testing.T) {
	polynomialCRAIndependentLazard(t, -1)
}

func TestPolynomialCRAIndependentLazardReservationPrefixes(t *testing.T) {
	success := false
	for prefix := int64(0); prefix < 128 && !success; prefix++ {
		t.Run(strconv.FormatInt(prefix, 10), func(t *testing.T) { success = polynomialCRAIndependentLazard(t, prefix) })
		if t.Failed() {
			return
		}
	}
	if !success {
		t.Fatal("content reservation horizon did not reach success")
	}
}

func polynomialCRAIndependentLazard(t *testing.T, prefix int64) bool {
	t.Helper()
	full := polynomialPSCFullFixture(t)
	m := full.core
	oldOut, e := NewPolynomialRef(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.CleanupLazard()); serviceMust(t, oldOut.Close()) })
	out, e := NewPolynomialRef(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, full.CleanupCRA()); serviceMust(t, out.Close()) })
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
	a := polynomialQueryTerms(t, m, []int32{-1, 1}, [][]MonomialPower{{{0, 1}}, nil})
	b := polynomialQueryTerms(t, m, []int32{2, 1}, [][]MonomialPower{{{0, 1}}, nil})
	first, second := unumeral.NewInt(3), unumeral.NewInt(5)
	full.originalBudget.FailAfter(prefix)
	contentBody := full.gcdCRA(a, &first, b, &second, out)
	full.originalBudget.FailAfter(-1)
	if contentBody != nil && !numericPhysicalReservationRefusal(contentBody) {
		t.Fatal("unexpected content refusal", contentBody)
	}
	if contentBody == nil && full.cra.active {
		t.Error("successful CRA adopted independent pending scope")
	}
	serviceMust(t, full.CleanupCRA())
	if !reflect.DeepEqual(before, *old.frame) {
		t.Error("CRA cleanup consumed independent Lazard")
	}
	if contentBody == nil {
		if first != unumeral.NewInt(3) || second != unumeral.NewInt(15) {
			t.Fatal("CRA bound publication", first, second)
		}
		polynomialQueryExpect(t, m, out.root.value, map[[3]uint32]int64{{1, 0, 0}: 2, {}: 1})
	}
	serviceMust(t, m.cleanupLazardInvocation(&old, true))
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{oldInput, input, oldOut.root.value, out.root.value, a, b}, []*unumeral.Int{&first, &second}, oldOut.header+out.header+handler.header)
	return contentBody == nil
}
