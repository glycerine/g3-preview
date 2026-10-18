// Source: polynomial.cpp:6237-6279, retained scalar integer substitution.
package polynomial

import (
	"errors"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestPolynomialSubstitutionInvocationReservationPrefixes(t *testing.T) {
	success, retained := false, false
	for prefix := int64(0); prefix < 128 && !success; prefix++ {
		full := polynomialSparseDirectFactoryFixture(t)
		c := full.core
		p := polynomialQueryTerms(t, c, []int32{3, 5, 7}, [][]MonomialPower{{{0, 3}, {1, 1}}, {{0, 2}}, nil})
		value := unumeral.NewInt(2)
		var receipt polynomialSubstitutionInvocationReceipt
		full.originalBudget.FailAfter(prefix)
		result, body := c.substituteIntegerWithReceipt(p, 0, &value, &receipt)
		full.originalBudget.FailAfter(-1)
		if body != nil && !numericPhysicalReservationRefusal(body) {
			t.Fatal("unexpected substitution refusal", prefix, body)
		}
		if !c.matchesSubstitutionInvocation(&receipt) {
			t.Fatal("accepted entry lost actual frame identity", prefix)
		}
		copyFrame := *receipt.frame
		wrong := receipt
		wrong.frame = &copyFrame
		state, used := c.memory.State(), full.originalBudget.Used()
		serviceMust(t, c.cleanupSubstitutionInvocation(&wrong, true))
		if state != c.memory.State() || used != full.originalBudget.Used() {
			t.Fatal("copied frame acquired actual work")
		}
		if body != nil && c.substitutionInvocationPending(&receipt) {
			retained = true
			var refused polynomialSubstitutionInvocationReceipt
			_, e := c.substituteIntegerWithReceipt(p, 0, &value, &refused)
			if !errors.Is(e, ErrPolynomialPending) || refused != (polynomialSubstitutionInvocationReceipt{}) {
				t.Fatal("refused entry acquired older invocation", e)
			}
			state, used = c.memory.State(), full.originalBudget.Used()
			serviceMust(t, c.cleanupSubstitutionInvocation(&refused, true))
			if state != c.memory.State() || used != full.originalBudget.Used() {
				t.Fatal("refused entry cleaned older work")
			}
		}
		serviceMust(t, c.cleanupSubstitutionInvocation(&receipt, false))
		serviceMust(t, c.cleanupSubstitutionInvocation(&receipt, true))
		if c.substitutionInvocationPending(&receipt) {
			t.Fatal("exact owning acknowledgement left pending work", prefix)
		}
		if body == nil {
			success = true
			result = polynomialCaller(t, c, result)
			polynomialQueryExpect(t, c, result, map[[3]uint32]int64{{0, 1, 0}: 24, {}: 27})
		}
		polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, result}, nil, 0)
		if body != nil {
			old := receipt
			retry, e := c.substituteIntegerWithReceipt(p, 0, &value, &receipt)
			serviceMust(t, e)
			retry = polynomialCaller(t, c, retry)
			if old.invocation == receipt.invocation {
				t.Fatal("retry reused invocation")
			}
			state, used = c.memory.State(), full.originalBudget.Used()
			serviceMust(t, c.cleanupSubstitutionInvocation(&old, true))
			if state != c.memory.State() || used != full.originalBudget.Used() {
				t.Fatal("stale receipt acquired replacement")
			}
		}
	}
	if !success || !retained {
		t.Fatal("missed completed result or actual retained invocation", success, retained)
	}
}
