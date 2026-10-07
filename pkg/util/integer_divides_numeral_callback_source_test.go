// Source: src/util/mpz.cpp:2054-2069; explicit returning-transport helper profile.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

type integerDividesScopedCallbackFixture struct {
	manager            *IntegerManager
	budget             *arena.Budget
	input, foreign     *Int
	called             bool
	reentry, copyError error
}

var integerDividesScopedCallbackCurrent *integerDividesScopedCallbackFixture

func integerDividesScopedFailureTransport(f Failure) error {
	s := integerDividesScopedCallbackCurrent
	if s != nil && !s.called {
		s.called = true
		s.budget.FailAfter(-1)
		_, s.reentry = s.manager.Divides(s.input, s.input)
		s.copyError = s.manager.Copy(s.foreign, s.input)
	}
	return f
}
func TestIntegerDividesScopedNUMRuntimeCallbackHasNoAmbientAcquisition(t *testing.T) {
	proved := false
	for prefix := int64(0); prefix < 50 && !proved; prefix++ {
		budget := arena.NewBudget(1 << 24)
		rt, e := NewRuntime(RuntimeConfig{Budget: budget, Transport: integerDividesScopedFailureTransport})
		serviceMust(t, e)
		cfg := serviceConfig(budget).Integers
		cfg.Memory = rt
		cfg.Profile = SmallAllocatorProfile{PointerBits: 64, Debug: true}
		m, e := NewIntegerManager(cfg)
		serviceMust(t, e)
		var a, b, x, y, g, foreign Int
		digits := make([]uint32, 64)
		digits[0] = 3
		digits[63] = 1
		serviceMust(t, m.SetLimbs(&a, digits))
		serviceMust(t, m.SetInt64(&b, 0))
		fixture := integerDividesScopedCallbackFixture{manager: m, budget: budget, input: &a, foreign: &foreign}
		integerDividesScopedCallbackCurrent = &fixture
		budget.FailAfter(prefix)
		_, returned := m.Divides(&a, &a)
		integerDividesScopedCallbackCurrent = nil
		budget.FailAfter(-1)
		if fixture.called {
			proved = true
			if returned == nil || !errors.Is(fixture.reentry, ErrNumericPending) || fixture.copyError != nil || foreign.Cell == 0 {
				t.Fatal("actual callback acquisition/reentry", prefix, returned, fixture.reentry, fixture.copyError, foreign)
			}
			integerPhysicalAudit(t, m, &a, &b, &x, &y, &g, &foreign)
			serviceMust(t, m.CleanupDividesNumeral())
			cmp, e := m.Compare(&foreign, &a)
			serviceMust(t, e)
			if cmp != 0 {
				t.Fatal("callback foreign numeral consumed by RAW abort")
			}
			integerPhysicalAudit(t, m, &a, &b, &x, &y, &g, &foreign)
		} else {
			serviceMust(t, m.CleanupDividesNumeral())
		}
		for _, v := range []*Int{&a, &b, &x, &y, &g, &foreign} {
			serviceMust(t, m.Del(v))
		}
		serviceMust(t, m.CloseChecked())
		serviceMust(t, rt.Close())
		if budget.Used() != 0 {
			t.Fatal("callback physical/cold owner", budget.Used())
		}
	}
	if !proved {
		t.Fatal("real Runtime transport frontier not reached")
	}
}
