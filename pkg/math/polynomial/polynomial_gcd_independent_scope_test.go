package polynomial

import (
	"errors"
	"reflect"
	"strconv"
	"testing"
)

func TestPolynomialGCDCleanupPreservesIndependentLazard(t *testing.T) {
	polynomialGCDIndependentLazard(t, -1, false)
}
func TestPolynomialGCDIndependentLazardReservationPrefixes(t *testing.T) {
	success := false
	for prefix := int64(0); prefix < 256 && !success; prefix++ {
		t.Run(strconv.FormatInt(prefix, 10), func(t *testing.T) { success = polynomialGCDIndependentLazard(t, prefix, true) })
		if t.Failed() {
			return
		}
	}
	if !success {
		t.Fatal("GCD reservation horizon did not reach success")
	}
}
func polynomialGCDIndependentLazard(t *testing.T, prefix int64, general bool) bool {
	t.Helper()
	full := polynomialPSCFullFixture(t)
	c := full.core
	oldOut, e := NewPolynomialRef(c)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, c.CleanupLazard()); serviceMust(t, oldOut.Close()) })
	oldInput := polynomialQueryTerms(t, c, []int32{2, 4}, [][]MonomialPower{{{0, 1}}, nil})
	p := polynomialQueryTerms(t, c, []int32{6, 9}, [][]MonomialPower{{{0, 1}}, nil})
	q := p
	if general {
		q = polynomialQueryTerms(t, c, []int32{4, 6}, [][]MonomialPower{{{0, 1}}, nil})
	}
	out, e := NewPolynomialRef(c)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, full.CleanupGCD()); serviceMust(t, out.Close()) })
	stop, reached := errors.New("independent Lazard source deletion"), false
	h, e := NewPolynomialDeletionHandler(full.originalBudget, func(_ *PolynomialDeletionHandler, m *MultivariatePolynomialCore, _ Polynomial) error {
		if m.lazardLocal.finishing && !reached {
			reached = true
			return stop
		}
		return nil
	})
	serviceMust(t, e)
	serviceMust(t, c.AddDeletionHandler(h))
	t.Cleanup(func() { serviceMust(t, c.RemoveDeletionHandler(h)); serviceMust(t, h.CloseChecked()) })
	var old polynomialLazardInvocationReceipt
	body := c.seLazardWithReceipt(3, c.unit, oldInput, 0, &oldOut.root, &old)
	if !reached || !errors.Is(body, stop) || !old.frame.active {
		t.Fatal("actual independent source stop missing", body)
	}
	before := *old.frame
	full.originalBudget.FailAfter(prefix)
	body = full.GCD(p, q, out)
	full.originalBudget.FailAfter(-1)
	if body != nil && !numericPhysicalReservationRefusal(body) {
		t.Fatal("unexpected GCD refusal", body)
	}
	if body == nil && full.gcdPending() {
		t.Error("successful GCD adopted independent pending scope")
	}
	if full.monic.active {
		captured := false
		for _, g := range full.gcdFrames {
			captured = captured || full.matchesMonicInvocation(&g.directMonic)
		}
		if !captured {
			t.Fatal("GCD lost its accepted pending monic child")
		}
	}
	serviceMust(t, full.CleanupGCD())
	if full.monic.active {
		t.Fatal("GCD cleanup left its actual monic child")
	}
	if !reflect.DeepEqual(before, *old.frame) {
		t.Error("GCD cleanup consumed independent Lazard")
	}
	if c.pscReceipts.generation != old.frame.sourceGeneration {
		serviceMust(t, c.CleanupPSC())
	}
	if !reflect.DeepEqual(before, *old.frame) {
		t.Error("current PSC cohort cleanup consumed independent Lazard")
	}
	if body == nil {
		if general {
			polynomialQueryExpect(t, c, out.root.value, map[[3]uint32]int64{{1, 0, 0}: 2, {}: 3})
		} else if out.root.value != p {
			t.Fatal("source identity shortcut changed pointer")
		}
	}
	serviceMust(t, c.cleanupLazardInvocation(&old, true))

	polys := []Polynomial{oldInput, p, oldOut.root.value, out.root.value}
	if general {
		polys = append(polys, q)
	}
	polynomialImpPhysicalAuditNamed(t, full, polys, nil, oldOut.header+out.header+h.header)
	return body == nil
}
