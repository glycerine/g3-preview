// Source: src/util/mpz.cpp:730-817; Go returning-transport owner boundary.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

type integerRawOwnerReentryFixture struct {
	manager  *IntegerManager
	called   bool
	finalize bool
	result   error
	changed  bool
}

var integerRawOwnerReentryCurrent *integerRawOwnerReentryFixture

func integerRawOwnerReentryTransport(f Failure) error {
	s := integerRawOwnerReentryCurrent
	if s != nil && !s.called {
		s.called = true
		m := s.manager
		stage, finalizing, finalized, cause := m.SourceFinalizeStage, m.SourceFinalizing, m.SourceFinalized, m.CleanupCauseField
		if s.finalize {
			s.result = m.SourceFinalize()
		} else {
			s.result = m.CloseChecked()
		}
		s.changed = stage != m.SourceFinalizeStage || finalizing != m.SourceFinalizing || finalized != m.SourceFinalized || cause != m.CleanupCauseField
	}
	return f
}
func TestIntegerRawExecutingOwnerRefusesFinalizeAndClose(t *testing.T) {
	for _, operation := range []string{"GCD", "AddMul", "SubMul"} {
		for _, finalize := range []bool{true, false} {
			t.Run(operation+"/"+map[bool]string{true: "SourceFinalize", false: "CloseChecked"}[finalize], func(t *testing.T) {
				budget := arena.NewBudget(1 << 24)
				rt, e := NewRuntime(RuntimeConfig{Budget: budget, Transport: integerRawOwnerReentryTransport})
				serviceMust(t, e)
				cfg := serviceConfig(budget).Integers
				cfg.Memory = rt
				cfg.Profile = SmallAllocatorProfile{PointerBits: 64, Debug: true}
				m, e := NewIntegerManager(cfg)
				serviceMust(t, e)
				var a, g Int
				digits := make([]uint32, 64)
				digits[0] = 3
				digits[63] = 1
				serviceMust(t, m.SetLimbs(&a, digits))
				factor := NewInt(2)
				fixture := integerRawOwnerReentryFixture{manager: m, finalize: finalize}
				integerRawOwnerReentryCurrent = &fixture
				budget.FailAfter(0)
				var body error
				switch operation {
				case "GCD":
					body = m.GCD(&a, &a, &g)
				case "AddMul":
					body = m.AddMul(&a, &factor, &a, &g)
				case "SubMul":
					body = m.SubMul(&a, &factor, &a, &g)
				}
				integerRawOwnerReentryCurrent = nil
				if !fixture.called || body == nil || !errors.Is(fixture.result, ErrNumericPending) || fixture.changed {
					t.Fatalf("executing owner refusal: called=%v body=%v result=%v changed=%v", fixture.called, body, fixture.result, fixture.changed)
				}
				integerPhysicalAudit(t, m, &a, &g)
				if budget.Used() != integerActualOwnedBytes(m) {
					t.Fatal("immediate named cold graph")
				}
				if operation == "GCD" {
					serviceMust(t, m.CleanupGCDRaw())
				} else {
					serviceMust(t, m.CleanupAddMulRaw())
				}
				budget.FailAfter(-1)
				if operation == "GCD" {
					serviceMust(t, m.GCD(&a, &a, &g))
					cmp, e := m.Compare(&a, &g)
					serviceMust(t, e)
					if cmp != 0 {
						t.Fatal("same-owner retry gcd")
					}
				} else {
					// Actual cold Add/SubMul consumes old scratch before this refusal.
					// Own acknowledgment cannot repair the source-invalid pointer.
					var next error
					if operation == "AddMul" {
						next = m.AddMul(&a, &factor, &a, &g)
					} else {
						next = m.SubMul(&a, &factor, &a, &g)
					}
					if !errors.Is(next, arena.ErrInvalidSlot) || m.Tmp[0] != 0 {
						t.Fatal("source scratch guard after actual callback", next, m.Tmp[0])
					}
					serviceMust(t, m.CleanupAddMulRaw())
				}
				if !errors.Is(body, Failure{Kind: FailureMemout}) {
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
}
