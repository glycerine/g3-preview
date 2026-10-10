// Source: src/util/rational.h:507-511.
package util

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	"math/big"
	"testing"
)

func TestRationalServiceAbsoluteValueInvocationPreservesInputAndReturn(t *testing.T) {
	s, _ := serviceNew(t)
	for _, tc := range []struct{ a, b, want string }{{"-7", "3", "7"}, {"7", "-3", "7"}, {"-7/3", "-3", "7/3"}, {"-18446744073709551616", "3", "18446744073709551616"}} {
		a, b := serviceValue(t, s, tc.a), serviceValue(t, s, tc.b)
		var call RationalServiceInvocation
		absolute, e := s.AbsWithInvocation(&a, &call)
		serviceMust(t, e)
		serviceMust(t, call.SourceReady())
		serviceMust(t, call.Cleanup(false))
		serviceMust(t, call.Cleanup(true))
		serviceMust(t, call.EndContext())
		got, e := s.String(&absolute)
		serviceMust(t, e)
		if got != tc.want {
			t.Fatal(tc, got)
		}
		av, e := s.String(&a)
		serviceMust(t, e)
		bv, e := s.String(&b)
		serviceMust(t, e)
		if av != tc.a || bv != tc.b {
			t.Fatal("inputs modified", av, bv)
		}
		serviceMust(t, s.Del(&absolute))
		serviceMust(t, s.Del(&b))
		serviceMust(t, s.Del(&a))
	}
}

func TestRationalServiceAbsoluteValueInvocationEveryActualReservation(t *testing.T) {
	complete := false
	for ordinal := int64(0); ordinal < 256 && !complete; ordinal++ {
		b := arena.NewBudget(1 << 27)
		cfg := serviceConfig(b)
		cfg.Integers.Profile = umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
		s, e := NewRationalService(cfg)
		serviceMust(t, e)
		a := serviceValue(t, s, "-184467440737095516170000000000000000003")
		divisor := serviceValue(t, s, "34359738369")
		var call RationalServiceInvocation
		b.FailAfter(ordinal)
		out, body := s.AbsWithInvocation(&a, &call)
		b.FailAfter(-1)
		if body != nil && !numericPhysicalReservationRefusal(body) {
			t.Fatal("unexpected refusal", ordinal, body)
		}
		if call.division.Invocation != 0 {
			t.Fatal("absolute value created division frame", ordinal)
		}
		if call.invocation == 0 || call.self != &call || call.bodyCause != body {
			t.Fatal("missing exact issued invocation", ordinal)
		}
		if body == nil {
			serviceMust(t, call.SourceReady())
			parsed, _ := new(big.Int).SetString("-184467440737095516170000000000000000003", 10)
			serviceText(t, s, &out, new(big.Int).Abs(parsed).String())
		}
		copy := call
		if e = copy.Cleanup(false); !errors.Is(e, arena.ErrInvalidSlot) {
			t.Fatal("copied ownership", e)
		}
		tuple, cause := call.invocation, call.bodyCause
		serviceMust(t, call.Cleanup(false))
		if out.ownerID != 0 {
			serviceMust(t, s.CleanupValue(&out))
		}
		serviceMust(t, call.Cleanup(true))
		if call.invocation != tuple || call.bodyCause != cause {
			t.Fatal("cleanup changed history")
		}
		serviceMust(t, call.EndContext())
		serviceMust(t, s.Del(&divisor))
		serviceMust(t, s.Del(&a))
		serviceMust(t, s.CloseChecked())
		serviceMust(t, s.Manager.Z.Memory.Close())
		if b.Used() != 0 {
			t.Fatal("absolute value graph retained", b.Used())
		}
		if body == nil {
			complete = true
			if ordinal == 0 {
				t.Fatal("missing real allocation frontier")
			}
		}
	}
	if !complete {
		t.Fatal("actual absolute value campaign incomplete")
	}
}
