// Source: src/math/polynomial/algebraic_numbers.cpp:1565-1577;
// src/util/mpz.cpp:2032-2041. Allocation-only Go diagnostic history.
package mpnum

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"testing"
)

func TestAlgebraicNegActualHistoryLaterFailureAndErasedZero(t *testing.T) {
	reached := false
	for prefix := int64(0); prefix < 32 && !reached; prefix++ {
		t.Run(fmt.Sprint(prefix), func(t *testing.T) {
			c, _, _ := algebraicCompareFixtureProfile(t, true)
			z := c.config.Rationals.z
			t.Cleanup(func() { z.budget.FailAfter(-1) })
			var first, later Algebraic
			n := NewRational()
			serviceMust(t, c.config.Rationals.SetInt32(&n, math.MinInt32))
			serviceMust(t, c.SetRational(&first, &n))
			serviceMust(t, c.SetRational(&later, &n))
			serviceMust(t, c.config.Rationals.Del(&n))
			serviceMust(t, c.Neg(&first))
			old := algebraicNegInvocationReceipt{c, &c.negation, c.negation.invocation}
			serviceMust(t, c.cleanupNegation(&old, false))
			serviceMust(t, c.cleanupNegation(&old, true))
			prior, ok := c.negationHistoryFor(&old)
			if !ok || len(c.negationHistory) == 0 {
				t.Fatal("actual allocating call lost charged history")
			}
			z.budget.FailAfter(prefix)
			body := c.Neg(&later)
			current := algebraicNegInvocationReceipt{c, &c.negation, c.negation.invocation}
			if body == nil {
				z.budget.FailAfter(-1)
				serviceMust(t, c.Del(&later))
				serviceMust(t, c.Del(&first))
				return
			}
			if !numericPhysicalReservationRefusal(body) {
				t.Fatal(body)
			}
			roots := c.negation.numeric
			state := c.negation.diagnostic()
			used := z.budget.Used()
			physical := z.memory.Stats()
			serviceMust(t, c.cleanupNegation(&old, true))
			var zero Algebraic
			z.budget.FailAfter(0)
			serviceMust(t, c.Neg(&zero))
			after, ok := c.negationHistoryFor(&old)
			if !ok || !reflect.DeepEqual(prior, after) || !reflect.DeepEqual(state, c.negation.diagnostic()) || roots != c.negation.numeric || used != z.budget.Used() || physical != z.memory.Stats() {
				t.Fatal("old acknowledgment or erased zero changed actual later call")
			}
			if c.negation.historyRecorded {
				reached = true
			}
			algebraicCellSourcePhysical(t, c, nil, nil, nil, &first, &later)
			z.budget.FailAfter(-1)
			serviceMust(t, c.cleanupNegation(&current, false))
			serviceMust(t, c.Del(&later))
			serviceMust(t, c.cleanupNegation(&current, true))
			serviceMust(t, c.Del(&first))
			final, ok := c.negationHistoryFor(&current)
			if !ok || !errors.Is(body, final.bodyCause) {
				t.Fatal("final allocating call history")
			}
		})
	}
	if !reached {
		t.Fatal("no later allocating call diagnostic record reached")
	}
}
