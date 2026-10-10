// Source: src/util/rational.h:462-464.
package util

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestRationalServiceSubReturnEveryActualReservation(t *testing.T) {
	complete := false
	failures := 0
	for ordinal := int64(0); ordinal < 256 && !complete; ordinal++ {
		b := arena.NewBudget(1 << 27)
		rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f umemory.Failure) error { return f }})
		serviceMust(t, e)
		cfg := serviceConfig(b)
		cfg.Integers.Memory = rt
		cfg.Integers.Profile = umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
		s, e := NewRationalService(cfg)
		serviceMust(t, e)
		a := serviceValue(t, s, "1")
		other := serviceValue(t, s, "18446744073709551617")
		var c RationalServiceInvocation
		b.FailAfter(ordinal)
		out, body := s.SubWithInvocation(&a, &other, &c)
		b.FailAfter(-1)
		if body != nil {
			failures++
			if !numericPhysicalReservationRefusal(body) {
				t.Fatal(ordinal, body)
			}
		} else {
			complete = true
			serviceText(t, s, &out, "-18446744073709551616")
		}
		serviceText(t, s, &a, "1")
		serviceText(t, s, &other, "18446744073709551617")
		cause, source, own := c.Causes()
		serviceMust(t, c.Cleanup(false))
		if out != (OwnedRational{}) {
			serviceMust(t, s.CleanupValue(&out))
		}
		serviceMust(t, c.Cleanup(true))
		bc, sc, oc := c.Causes()
		if bc != cause || sc != source || oc != own {
			t.Fatal("changed source history")
		}
		serviceMust(t, c.EndContext())
		serviceMust(t, s.Del(&other))
		serviceMust(t, s.Del(&a))
		serviceMust(t, s.CloseChecked())
		serviceMust(t, rt.Close())
		if b.Used() != 0 {
			t.Fatal(ordinal, b.Used())
		}
	}
	if !complete || failures == 0 {
		t.Fatal(complete, failures)
	}
}
