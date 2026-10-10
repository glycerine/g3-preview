// Source: src/util/rational.h:171-174.
package util

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	"math/big"
	"testing"
)

func TestRationalServiceRemAssignInvocationPhysicalReceiver(t *testing.T) {
	s, b := serviceNew(t)
	for _, tc := range []struct{ a, b, want string }{{"-7", "3", "-1"}, {"7", "-3", "1"}, {"-7", "-3", "-1"}, {"7", "7", "0"}, {"18446744073709551617", "3", "2"}} {
		for _, alias := range []bool{false, true} {
			a, divisor := serviceValue(t, s, tc.a), serviceValue(t, s, tc.b)
			rhs := &divisor
			want := tc.want
			if alias {
				rhs = &a
				want = "0"
			}
			var call RationalServiceInvocation
			b.FailAfter(0)
			e := s.RemAssignWithInvocation(&a, rhs, &call)
			b.FailAfter(-1)
			serviceMust(t, e)
			serviceText(t, s, &a, want)
			serviceText(t, s, &divisor, tc.b)
			serviceMust(t, call.SourceReady())
			serviceMust(t, call.Cleanup(false))
			serviceMust(t, call.Cleanup(true))
			serviceMust(t, call.EndContext())
			serviceMust(t, s.Del(&divisor))
			serviceMust(t, s.Del(&a))
		}
	}
}

func TestRationalServiceRemAssignInvocationEveryActualReservation(t *testing.T) {
	complete := false
	aText := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 1024), big.NewInt(123456)).String()
	dText := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 257), big.NewInt(17)).String()
	for ordinal := int64(0); ordinal < 256 && !complete; ordinal++ {
		b := arena.NewBudget(1 << 27)
		cfg := serviceConfig(b)
		cfg.Integers.Profile = umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
		s, e := NewRationalService(cfg)
		serviceMust(t, e)
		a := serviceValue(t, s, aText)
		divisor := serviceValue(t, s, dText)
		original := a
		var call RationalServiceInvocation
		b.FailAfter(ordinal)
		body := s.RemAssignWithInvocation(&a, &divisor, &call)
		b.FailAfter(-1)
		if body != nil && !numericPhysicalReservationRefusal(body) {
			t.Fatal(ordinal, body)
		}
		if call.invocation == 0 || call.self != &call || call.bodyCause != body {
			t.Fatal("missing exact invocation", ordinal)
		}
		if a.ownerID != original.ownerID {
			t.Fatal("returned owner substituted for receiver")
		}
		if body == nil {
			complete = true
			serviceMust(t, call.SourceReady())
			av, _ := new(big.Int).SetString(aText, 10)
			dv, _ := new(big.Int).SetString(dText, 10)
			serviceText(t, s, &a, new(big.Int).Rem(av, dv).String())
			serviceText(t, s, &divisor, dText)
			if ordinal == 0 {
				t.Fatal("missing actual allocation frontier")
			}
		}
		copy := call
		if e = copy.Cleanup(false); !errors.Is(e, arena.ErrInvalidSlot) {
			t.Fatal("copied context", e)
		}
		tuple, cause := call.invocation, call.bodyCause
		serviceMust(t, call.Cleanup(false))
		serviceMust(t, s.CleanupValue(&a))
		serviceMust(t, call.Cleanup(true))
		if tuple != call.invocation || cause != call.bodyCause {
			t.Fatal("cleanup changed history")
		}
		serviceMust(t, call.EndContext())
		serviceMust(t, s.Del(&divisor))
		serviceMust(t, s.CloseChecked())
		serviceMust(t, s.Manager.Z.Memory.Close())
		if b.Used() != 0 {
			t.Fatal("retained remainder graph", b.Used())
		}
	}
	if !complete {
		t.Fatal("actual remainder campaign incomplete")
	}
}
