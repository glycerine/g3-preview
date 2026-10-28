package numeral

// Source: src/util/rational.h:131-134,302-304.

import (
	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	"testing"
)

func TestRationalServiceBorrowedZeroAssignmentWithoutTemporary(t *testing.T) {
	b := arena.NewBudget(1 << 27)
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f umemory.Failure) error { return f }})
	serviceMust(t, e)
	cfg := serviceConfig(b)
	cfg.Integers.Memory = rt
	cfg.Integers.Profile = umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	s, e := NewRationalService(cfg)
	serviceMust(t, e)
	zeroBefore := s.ZeroField
	for _, text := range []string{"0", "3/2", "18446744073709551617/18446744073709551619", "-18446744073709551617"} {
		dst := serviceValue(t, s, text)
		var call RationalServiceInvocation
		b.FailAfter(0)
		e = s.CopyZeroWithInvocation(&dst, &call)
		b.FailAfter(-1)
		serviceMust(t, e)
		serviceText(t, s, &dst, "0")
		serviceMust(t, call.SourceReady())
		serviceMust(t, call.Cleanup(false))
		serviceMust(t, call.Cleanup(true))
		serviceMust(t, call.EndContext())
		serviceMust(t, s.Del(&dst))
	}
	if s.ZeroField != zeroBefore {
		t.Fatal("borrowed static zero changed")
	}
	serviceMust(t, s.CloseChecked())
	serviceMust(t, rt.Close())
	if b.Used() != 0 {
		t.Fatal("actual zero assignment retained graph", b.Used())
	}
}
