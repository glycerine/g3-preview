// Source: src/util/rational.h:43,306-308,462-464.
package util

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestRationalServiceBorrowedOneCopyAndSubtraction(t *testing.T) {
	s, b := serviceNew(t)
	oneBefore := s.OneField
	dst := serviceValue(t, s, "18446744073709551617")
	var copyCall RationalServiceInvocation
	// Copying the small static one must not construct an owned intermediate.
	b.FailAfter(0)
	e := s.CopyOneWithInvocation(&dst, &copyCall)
	b.FailAfter(-1)
	serviceMust(t, e)
	serviceText(t, s, &dst, "1")
	serviceMust(t, copyCall.SourceReady())
	serviceMust(t, copyCall.Cleanup(false))
	serviceMust(t, copyCall.Cleanup(true))
	serviceMust(t, copyCall.EndContext())
	serviceMust(t, s.Del(&dst))
	for _, tc := range []struct{ input, want string }{
		{"0", "-1"}, {"1", "0"}, {"-1", "-2"}, {"3/2", "1/2"},
		{"18446744073709551617", "18446744073709551616"},
	} {
		a := serviceValue(t, s, tc.input)
		var call RationalServiceInvocation
		out, e := s.SubOneWithInvocation(&a, &call)
		serviceMust(t, e)
		serviceMust(t, call.SourceReady())
		serviceText(t, s, &out, tc.want)
		serviceText(t, s, &a, tc.input)
		serviceMust(t, call.Cleanup(false))
		serviceMust(t, s.Del(&out))
		serviceMust(t, call.Cleanup(true))
		serviceMust(t, call.EndContext())
		serviceMust(t, s.Del(&a))
	}
	if s.OneField != oneBefore {
		t.Fatal("borrowed static one changed")
	}
}

func TestRationalServiceBorrowedOneSubtractionEveryActualReservation(t *testing.T) {
	complete, failures := false, 0
	for ordinal := int64(0); ordinal < 256 && !complete; ordinal++ {
		b := arena.NewBudget(1 << 27)
		rt, e := NewRuntime(RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f Failure) error { return f }})
		serviceMust(t, e)
		cfg := serviceConfig(b)
		cfg.Integers.Memory = rt
		cfg.Integers.Profile = SmallAllocatorProfile{PointerBits: 64, Debug: true}
		s, e := NewRationalService(cfg)
		serviceMust(t, e)
		a := serviceValue(t, s, "18446744073709551617/3")
		oneBefore := s.OneField
		var call RationalServiceInvocation
		b.FailAfter(ordinal)
		out, body := s.SubOneWithInvocation(&a, &call)
		b.FailAfter(-1)
		if body == nil {
			complete = true
			serviceText(t, s, &out, "18446744073709551614/3")
			serviceMust(t, call.SourceReady())
		} else {
			failures++
			if !numericPhysicalReservationRefusal(body) {
				t.Fatal(ordinal, body)
			}
		}
		if s.OneField != oneBefore {
			t.Fatal("borrowed static one changed on refusal")
		}
		bodyBefore, sourceBefore, _ := call.Causes()
		serviceMust(t, call.Cleanup(false))
		if out.ownerID != 0 {
			serviceMust(t, s.CleanupValue(&out))
		}
		serviceMust(t, call.Cleanup(true))
		bodyAfter, sourceAfter, _ := call.Causes()
		if bodyBefore != bodyAfter || sourceBefore != sourceAfter {
			t.Fatal("cleanup changed stopped source history")
		}
		serviceMust(t, call.EndContext())
		serviceMust(t, s.Del(&a))
		serviceMust(t, s.CloseChecked())
		serviceMust(t, rt.Close())
		if b.Used() != 0 {
			t.Fatal("retained graph", b.Used())
		}
	}
	if !complete || failures == 0 {
		t.Fatal("reservation campaign incomplete", complete, failures)
	}
}
