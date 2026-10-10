// Source: src/util/rational.h:503-505; src/util/mpq.cpp:282-295.
package numeral

import (
	"errors"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestRationalServicePowerInvocationOriginalSequenceAndReturn(t *testing.T) {
	s, _ := serviceNew(t)
	for _, tc := range []struct {
		base string
		p    uint32
		want string
	}{{"0", 0, "1"}, {"-3", 0, "1"}, {"-3", 1, "-3"}, {"-3", 5, "-243"}, {"-2/3", 3, "-8/27"}, {"2", 65, "36893488147419103232"}} {
		a := serviceValue(t, s, tc.base)
		var call RationalServiceInvocation
		out, e := s.PowerWithInvocation(&a, tc.p, &call)
		serviceMust(t, e)
		if call.powerGeneration == 0 || call.powerGeneration != s.Manager.RawArithmetic.PowerGeneration {
			t.Fatal("uncaptured original power frame")
		}
		serviceMust(t, call.SourceReady())
		serviceMust(t, call.Cleanup(false))
		serviceMust(t, call.Cleanup(true))
		serviceMust(t, call.EndContext())
		serviceText(t, s, &out, tc.want)
		serviceText(t, s, &a, tc.base)
		serviceMust(t, s.Del(&out))
		serviceMust(t, s.Del(&a))
	}
}

func TestRationalServicePowerInvocationEveryActualReservation(t *testing.T) {
	for _, base := range []string{"2", "2/3"} {
		complete := false
		for ordinal := int64(0); ordinal < 1024 && !complete; ordinal++ {
			b := arena.NewBudget(1 << 27)
			cfg := serviceConfig(b)
			cfg.Integers.Profile = umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
			s, e := NewRationalService(cfg)
			serviceMust(t, e)
			a := serviceValue(t, s, base)
			var older RationalServiceInvocation
			oldValue, e := s.PowerWithInvocation(&a, 0, &older)
			serviceMust(t, e)
			var call RationalServiceInvocation
			b.FailAfter(ordinal)
			out, body := s.PowerWithInvocation(&a, 65, &call)
			b.FailAfter(-1)
			if body != nil && !numericPhysicalReservationRefusal(body) {
				t.Fatal(ordinal, body)
			}
			if call.powerGeneration == 0 || call.powerGeneration != s.Manager.RawArithmetic.PowerGeneration {
				t.Fatal("missing exact power generation", ordinal)
			}
			if call.invocation == 0 || call.self != &call || call.bodyCause != body {
				t.Fatal("missing issued invocation", ordinal)
			}
			if body == nil {
				complete = true
				serviceMust(t, call.SourceReady())
				parsed, _ := new(big.Rat).SetString(base)
				numerator := new(big.Int).Exp(parsed.Num(), big.NewInt(65), nil)
				denominator := new(big.Int).Exp(parsed.Denom(), big.NewInt(65), nil)
				serviceText(t, s, &out, new(big.Rat).SetFrac(numerator, denominator).RatString())
				serviceText(t, s, &a, base)
				if ordinal == 0 {
					t.Fatal("missing actual allocation frontier")
				}
			}
			active, cursor, generationBefore := s.Manager.RawArithmetic.PowerActive, s.Manager.RawArithmetic.powerCursor, s.Manager.RawArithmetic.PowerGeneration
			serviceMust(t, older.Cleanup(false))
			serviceMust(t, s.CleanupValue(&oldValue))
			serviceMust(t, older.Cleanup(true))
			serviceMust(t, older.EndContext())
			if active != s.Manager.RawArithmetic.PowerActive || cursor != s.Manager.RawArithmetic.powerCursor || generationBefore != s.Manager.RawArithmetic.PowerGeneration {
				t.Fatal("older context joined younger power", ordinal)
			}
			copy := call
			if e = copy.Cleanup(false); !errors.Is(e, arena.ErrInvalidSlot) {
				t.Fatal("copied context", e)
			}
			generation, cause := call.powerGeneration, call.bodyCause
			rawCause := s.Manager.RawArithmetic.powerCause
			serviceMust(t, call.Cleanup(false))
			if out.ownerID != 0 {
				serviceMust(t, s.CleanupValue(&out))
			}
			serviceMust(t, call.Cleanup(true))
			if s.Manager.RawArithmetic.PowerActive {
				t.Fatal("captured power child not joined", ordinal)
			}
			if generation != call.powerGeneration || cause != call.bodyCause || rawCause != s.Manager.RawArithmetic.powerCause {
				t.Fatal("cleanup changed power history")
			}
			serviceMust(t, call.EndContext())
			serviceMust(t, s.Del(&a))
			serviceMust(t, s.CloseChecked())
			serviceMust(t, s.Manager.Z.Memory.Close())
			if b.Used() != 0 {
				t.Fatal("retained power graph", b.Used())
			}
		}
		if !complete {
			t.Fatal("actual power campaign incomplete")
		}
	}
}

func TestRationalServicePowerInvocationFinalUnusedSquareAndMaskWrap(t *testing.T) {
	s, b := serviceNew(t)
	a := serviceValue(t, s, "65536")
	var call RationalServiceInvocation
	// 65536^1 itself fits inline. The source's final unused 65536^2 needs a cell.
	b.FailAfter(0)
	out, body := s.PowerWithInvocation(&a, 1, &call)
	b.FailAfter(-1)
	if body == nil || !numericPhysicalReservationRefusal(body) {
		t.Fatal("final unused square erased", body)
	}
	serviceMust(t, call.Cleanup(false))
	if out.ownerID != 0 {
		serviceMust(t, s.CleanupValue(&out))
	}
	serviceMust(t, call.Cleanup(true))
	serviceMust(t, call.EndContext())
	serviceMust(t, s.Del(&a))
	a = serviceValue(t, s, "1")
	call = RationalServiceInvocation{}
	_, body = s.PowerWithInvocation(&a, 1<<31, &call)
	var domain *PowerDomainError
	if !errors.As(body, &domain) || domain.Reason != PowerMaskOverflow {
		t.Fatal("original mask wrap", body)
	}
	serviceMust(t, call.Cleanup(false))
	serviceMust(t, call.Cleanup(true))
	serviceMust(t, call.EndContext())
	serviceMust(t, s.Del(&a))
}
