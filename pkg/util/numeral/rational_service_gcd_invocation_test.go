package numeral

// Source: src/util/rational.h:519-523; src/util/mpq.h:gcd.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestRationalServiceGCDCoefficientsInvocation(t *testing.T) {
	s, budget := serviceNew(t)
	a := serviceValue(t, s, "17")
	b := serviceValue(t, s, "256")
	x := serviceValue(t, s, "3/7")
	y := serviceValue(t, s, "5/9")
	var call RationalServiceInvocation
	budget.FailAfter(0)
	g, e := s.GCDCoefficientsWithInvocation(&a, &b, &x, &y, &call)
	serviceMust(t, e)
	budget.FailAfter(-1)
	if call.self != &call || call.invocation == 0 || call.extended.Frame != &s.Manager.Z.ExtendedRaw {
		t.Fatal("missing exact caller invocation")
	}
	serviceIndependentText(t, s, &g, "1")
	serviceIndependentText(t, s, &x, "-15")
	serviceIndependentText(t, s, &y, "1")
	serviceMust(t, call.SourceReady())
	copied := call
	if e := copied.Cleanup(false); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("copied context accepted", e)
	}
	serviceMust(t, call.Cleanup(false))
	serviceMust(t, call.Cleanup(true))
	serviceMust(t, call.EndContext())
	for _, v := range []*OwnedRational{&a, &b, &x, &y, &g} {
		serviceMust(t, s.Del(v))
	}
}

func TestRationalServiceGCDCoefficientsInvocationEveryRuntimeReservation(t *testing.T) {
	reachedTransports := 0
	for ordinal := int64(0); ordinal < 200; ordinal++ {
		budget := arena.NewBudget(1 << 27)
		var s *RationalService
		var call RationalServiceInvocation
		active, transports := false, 0
		var ready, cleanup, reentry error
		rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: budget, ThreadLocal: true, SizeTBits: 64, Transport: func(f umemory.Failure) error {
			if active {
				transports++
				ready = call.SourceReady()
				cleanup = call.Cleanup(false)
				_, reentry = s.NewValue()
			}
			return f
		}})
		serviceMust(t, e)
		cfg := serviceConfig(budget)
		cfg.Integers.Memory = rt
		cfg.Integers.Profile = umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
		s, e = NewRationalService(cfg)
		serviceMust(t, e)
		a := serviceValue(t, s, "1329227995784915872903807060280344577")
		b := serviceValue(t, s, "18446744073709551617")
		x := serviceValue(t, s, "3/7")
		y := serviceValue(t, s, "5/9")
		budget.FailAfter(ordinal)
		active = true
		g, body := s.GCDCoefficientsWithInvocation(&a, &b, &x, &y, &call)
		active = false
		if call.extended.Frame != &s.Manager.Z.ExtendedRaw || call.extended.Invocation == 0 {
			t.Fatal("extended GCD child was not captured at entry", ordinal)
		}
		if body != nil && !numericPhysicalReservationRefusal(body) {
			t.Fatal(ordinal, body)
		}
		reachedTransports += transports
		// Arena bookkeeping reservations can refuse without Runtime transport.
		if transports != 0 && (transports != 1 || !errors.Is(ready, ErrRationalLexicalPending) || !errors.Is(cleanup, arena.ErrInvalidSlot) || !errors.Is(reentry, arena.ErrInvalidSlot)) {
			t.Fatal("Runtime transport allowed service/context reentry", ordinal, transports, ready, cleanup, reentry)
		}
		if !isOneInteger(&x.Raw.Den) || !isOneInteger(&y.Raw.Den) {
			t.Fatal("coefficient denominator prefix erased", ordinal)
		}
		roots := []*Int{&a.Raw.Num, &a.Raw.Den, &b.Raw.Num, &b.Raw.Den, &x.Raw.Num, &x.Raw.Den, &y.Raw.Num, &y.Raw.Den, &g.Raw.Num, &g.Raw.Den}
		for i := uint8(0); i < call.scope.Count; i++ {
			roots = append(roots, &call.scope.Roots[i].Num, &call.scope.Roots[i].Den)
		}
		var extra []umemory.Block
		if block := call.transferred.Block(); block != (umemory.Block{}) {
			extra = append(extra, block)
		}
		integerPhysicalAuditWithRaw(t, s.Manager.Z, extra, roots...)
		budget.FailAfter(-1)
		serviceMust(t, call.Cleanup(false))
		for _, v := range []*OwnedRational{&a, &b, &x, &y, &g} {
			if v.ownerID != 0 {
				serviceMust(t, s.CleanupValue(v))
			}
		}
		serviceMust(t, call.Cleanup(true))
		if s.Manager.Z.ExtendedRaw.Active {
			t.Fatal("call cleanup left its extended GCD child active", ordinal)
		}
		serviceMust(t, call.EndContext())
		serviceMust(t, s.CloseChecked())
		serviceMust(t, rt.Close())
		if budget.Used() != 0 {
			t.Fatal("retained graph at context end", ordinal, budget.Used())
		}
		if body == nil {
			if ordinal == 0 || reachedTransports == 0 {
				t.Fatal("missing reached allocations")
			}
			t.Log("first complete reservation ordinal", ordinal)
			return
		}
	}
	t.Fatal("reservation horizon exhausted")
}
