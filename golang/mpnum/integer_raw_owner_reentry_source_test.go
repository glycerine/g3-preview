// Source: src/util/mpz.cpp:730-817; Go returning-transport owner boundary.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

type integerRawOwnerReentryFixture struct {
	manager  *IntegerManager
	called   bool
	finalize bool
	result   error
	changed  bool
}

var integerRawOwnerReentryCurrent *integerRawOwnerReentryFixture

func integerRawOwnerReentryTransport(f memory.Failure) error {
	s := integerRawOwnerReentryCurrent
	if s != nil && !s.called {
		s.called = true
		m := s.manager
		stage, finalizing, finalized, cause := m.sourceFinalizeStage, m.sourceFinalizing, m.sourceFinalized, m.cleanupCause
		if s.finalize {
			s.result = m.SourceFinalize()
		} else {
			s.result = m.CloseChecked()
		}
		s.changed = stage != m.sourceFinalizeStage || finalizing != m.sourceFinalizing || finalized != m.sourceFinalized || cause != m.cleanupCause
	}
	return f
}
func TestIntegerRawExecutingOwnerRefusesFinalizeAndClose(t *testing.T) {
	for _, finalize := range []bool{true, false} {
		t.Run(map[bool]string{true: "SourceFinalize", false: "CloseChecked"}[finalize], func(t *testing.T) {
			budget := arena.NewBudget(1 << 24)
			rt, e := memory.NewRuntime(memory.RuntimeConfig{Budget: budget, Transport: integerRawOwnerReentryTransport})
			serviceMust(t, e)
			cfg := serviceConfig(budget).Integers
			cfg.Memory = rt
			cfg.Profile = memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
			m, e := NewIntegerManager(cfg)
			serviceMust(t, e)
			var a, g Int
			digits := make([]uint32, 64)
			digits[0] = 3
			digits[63] = 1
			serviceMust(t, m.SetLimbs(&a, digits))
			fixture := integerRawOwnerReentryFixture{manager: m, finalize: finalize}
			integerRawOwnerReentryCurrent = &fixture
			budget.FailAfter(0)
			body := m.GCD(&a, &a, &g)
			integerRawOwnerReentryCurrent = nil
			if !fixture.called || body == nil || !errors.Is(fixture.result, ErrPolynomialPending) || fixture.changed {
				t.Fatalf("executing owner refusal: called=%v body=%v result=%v changed=%v", fixture.called, body, fixture.result, fixture.changed)
			}
			integerPhysicalAudit(t, m, &a, &g)
			if budget.Used() != integerActualOwnedBytes(m) {
				t.Fatal("immediate named cold graph")
			}
			serviceMust(t, m.CleanupGCDRaw())
			budget.FailAfter(-1)
			serviceMust(t, m.GCD(&a, &a, &g))
			cmp, e := m.Compare(&a, &g)
			serviceMust(t, e)
			if cmp != 0 {
				t.Fatal("same-owner retry gcd")
			}
			if !errors.Is(body, memory.Failure{Kind: memory.FailureMemout}) {
				t.Fatal("earlier failure changed", body)
			}
			serviceMust(t, m.Del(&a))
			serviceMust(t, m.Del(&g))
			serviceMust(t, m.CloseChecked())
			serviceMust(t, rt.Close())
			if budget.Used() != 0 {
				t.Fatal("actual owner leak", budget.Used())
			}
		})
	}
}
