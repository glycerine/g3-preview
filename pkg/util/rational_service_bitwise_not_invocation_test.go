// Source: src/util/rational.h:388-392; src/util/mpz.cpp:1151-1183.
package util

import (
	"errors"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestRationalServiceBitwiseNotInvocationEveryActualReservation(t *testing.T) {
	for _, tc := range []struct {
		value string
		width uint32
	}{
		{"0", 63}, {"1", 64}, {"18446744073709551617", 193}, {"18446744073709551617", 0},
	} {
		complete, sawBodyFailure := false, false
		for ordinal := int64(0); ordinal < 512 && !complete; ordinal++ {
			b := arena.NewBudget(1 << 27)
			rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f umemory.Failure) error { return f }})
			serviceMust(t, e)
			cfg := serviceConfig(b)
			cfg.Integers.Memory = rt
			cfg.Integers.Profile = umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
			s, e := NewRationalService(cfg)
			serviceMust(t, e)
			a := serviceValue(t, s, tc.value)
			var older RationalServiceInvocation
			old, e := s.BitwiseNotWithInvocation(0, &a, &older)
			serviceMust(t, e)
			var call RationalServiceInvocation
			b.FailAfter(ordinal)
			out, body := s.BitwiseNotWithInvocation(tc.width, &a, &call)
			b.FailAfter(-1)
			raw := &s.Manager.Z.BitwiseNotRaw
			if body == nil {
				complete = true
				v, _ := new(big.Int).SetString(tc.value, 10)
				mask := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), uint(tc.width)), big.NewInt(1))
				want := new(big.Int).And(new(big.Int).Not(v), mask).String()
				serviceText(t, s, &out, want)
				serviceText(t, s, &a, tc.value)
				serviceMust(t, call.SourceReady())
				if tc.width > 63 || !a.Raw.Num.IsSmall() {
					if call.bitwiseNot.Invocation == 0 || call.bitwiseNot.Invocation != raw.Invocation || raw.cursor != 4 || !raw.ended {
						t.Fatal("original RAW end absent")
					}
				} else if call.bitwiseNot.Invocation != 0 {
					t.Fatal("small path invented raw locals")
				}
			} else if !numericPhysicalReservationRefusal(body) {
				t.Fatal(ordinal, body)
			}
			if raw.BodyCause != nil {
				sawBodyFailure = true
				if raw.cursor != 0 || raw.ended || raw.BodySucceeded || !raw.Active {
					t.Fatal("body failure ran explicit source DEL", ordinal)
				}
			}
			generation, active, cursor := raw.Invocation, raw.Active, raw.cursor
			serviceMust(t, older.Cleanup(false))
			serviceMust(t, s.CleanupValue(&old))
			serviceMust(t, older.Cleanup(true))
			serviceMust(t, older.EndContext())
			if generation != raw.Invocation || active != raw.Active || cursor != raw.cursor {
				t.Fatal("old receipt adopted younger frame")
			}
			copy := call
			if e = copy.Cleanup(false); !errors.Is(e, arena.ErrInvalidSlot) {
				t.Fatal("copied invocation", e)
			}
			cause, source := raw.BodyCause, raw.sourceCause
			serviceMust(t, call.Cleanup(false))
			if out.ownerID != 0 {
				serviceMust(t, s.CleanupValue(&out))
			}
			serviceMust(t, call.Cleanup(true))
			if call.bitwiseNot.Invocation != 0 && raw.Active {
				t.Fatal("exact raw child not joined")
			}
			if cause != raw.BodyCause || source != raw.sourceCause || generation != raw.Invocation {
				t.Fatal("history changed")
			}
			serviceMust(t, call.EndContext())
			serviceMust(t, s.Del(&a))
			serviceMust(t, s.CloseChecked())
			serviceMust(t, s.Manager.Z.Memory.Close())
			if b.Used() != 0 {
				t.Fatal("retained graph", b.Used())
			}
		}
		if tc.width == 193 && !sawBodyFailure {
			t.Fatal("raw body frontier not reached")
		}
		if !complete {
			t.Fatal("campaign incomplete", tc)
		}
	}
}

func TestRationalServiceBitwiseNotInvocationExecutingRawGuard(t *testing.T) {
	b := arena.NewBudget(1 << 27)
	var s *RationalService
	var call RationalServiceInvocation
	active, calls := false, 0
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, ThreadLocal: false, SizeTBits: 64, Transport: func(f umemory.Failure) error {
		if active {
			calls++
			raw := &s.Manager.Z.BitwiseNotRaw
			if call.bitwiseNot.Frame != raw || call.bitwiseNot.Invocation != raw.Invocation || !raw.executing || raw.Count != 4 {
				t.Fatal("actual RAW invocation not captured")
			}
			if e := call.Cleanup(false); e != arena.ErrInvalidSlot {
				t.Fatal("executing caller cleanup", e)
			}
			if e := call.SourceReady(); e != ErrRationalLexicalPending {
				t.Fatal("executing source readiness", e)
			}
			if e := s.Manager.Z.CleanupBitwiseNot(); e != ErrNumericPending {
				t.Fatal("executing RAW cleanup", e)
			}
			if e := s.Manager.Z.CloseChecked(); e != ErrNumericPending {
				t.Fatal("executing integer cleanup", e)
			}
		}
		return f
	}})
	serviceMust(t, e)
	cfg := serviceConfig(b)
	cfg.Integers.Memory = rt
	cfg.Integers.Profile = umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	s, e = NewRationalService(cfg)
	serviceMust(t, e)
	a := serviceValue(t, s, "18446744073709551617")
	serviceMust(t, rt.SetMaxCount(uint64(rt.State().GlobalCount)+1))
	active = true
	out, body := s.BitwiseNotWithInvocation(4097, &a, &call)
	active = false
	if body == nil || calls == 0 {
		t.Fatal("actual RAW transport not reached", body, calls)
	}
	serviceMust(t, rt.SetMaxCount(0))
	serviceMust(t, call.Cleanup(false))
	if out.ownerID != 0 {
		serviceMust(t, s.CleanupValue(&out))
	}
	serviceMust(t, call.Cleanup(true))
	serviceMust(t, call.EndContext())
	serviceMust(t, s.Del(&a))
	serviceMust(t, s.CloseChecked())
	serviceMust(t, rt.Close())
	if b.Used() != 0 {
		t.Fatal("callback graph retained", b.Used())
	}
}
