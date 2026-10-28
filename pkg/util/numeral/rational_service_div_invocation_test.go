package numeral

// Source: src/util/rational.h operator/, reached by MPBQ debug snapshots.

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestRationalServiceDivisionInvocationEveryActualReservation(t *testing.T) {
	complete, issued := false, false
	failures := 0
	for ordinal := int64(0); ordinal < 256 && !complete; ordinal++ {
		budget := arena.NewBudget(1 << 27)
		runtime, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: budget, ThreadLocal: true})
		serviceMust(t, e)
		cfg := serviceConfig(budget)
		cfg.Integers.Memory = runtime
		cfg.Integers.Profile = umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
		s, e := NewRationalService(cfg)
		serviceMust(t, e)
		a := serviceValue(t, s, "18446744073709551617")
		b := serviceValue(t, s, "18446744073709551619")
		var call RationalServiceInvocation
		budget.FailAfter(ordinal)
		out, body := s.DivWithInvocation(&a, &b, &call)
		budget.FailAfter(-1)
		issued = issued || s.Manager.Z.MatchesRawInvocation(&call.gcd)
		if body != nil {
			failures++
			if !numericPhysicalReservationRefusal(body) {
				t.Fatal(ordinal, body)
			}
		} else {
			complete = true
			// The odd numerator and denominator differ by two, so this is
			// independently the reduced quotient of the original operands.
			serviceText(t, s, &out, "18446744073709551617/18446744073709551619")
		}
		serviceText(t, s, &a, "18446744073709551617")
		serviceText(t, s, &b, "18446744073709551619")
		cause, source, own := call.Causes()
		serviceMust(t, call.Cleanup(false))
		if out != (OwnedRational{}) {
			serviceMust(t, s.CleanupValue(&out))
		}
		serviceMust(t, call.Cleanup(true))
		bc, sc, oc := call.Causes()
		if bc != cause || sc != source || oc != own {
			t.Fatal("owning acknowledgement changed source history")
		}
		serviceMust(t, call.EndContext())
		serviceMust(t, s.Del(&b))
		serviceMust(t, s.Del(&a))
		serviceMust(t, s.Close())
		serviceMust(t, runtime.Close())
		if budget.Used() != 0 {
			t.Fatal("division invocation retained charged storage", ordinal, budget.Used())
		}
	}
	if !complete || failures == 0 || !issued {
		t.Fatal("division did not reach failure, success and an actual GCD child", complete, failures, issued)
	}
}
