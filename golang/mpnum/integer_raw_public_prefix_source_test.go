// Source: src/util/mpz.h:468-485; src/util/mpz.cpp:915-985.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"reflect"
	"testing"
)

func TestIntegerPublicRawEveryActualPrefixAndExactReceipt(t *testing.T) {
	for route := 0; route < 2; route++ {
		success, retained := false, false
		for prefix := int64(0); prefix < 256 && !success; prefix++ {
			m, budget := mpzPublicationOwner(t)
			var a, b, x, y, g Int
			digits := make([]uint32, 64)
			digits[0] = 3
			digits[31] = 1
			digits[63] = 1
			serviceMust(t, m.SetLimbs(&a, digits))
			other := make([]uint32, 33)
			other[32] = 1
			serviceMust(t, m.SetLimbs(&b, other))
			if route == 0 {
				serviceMust(t, m.Neg(&a))
			}
			budget.FailAfter(prefix)
			var callErr error
			if route == 0 {
				callErr = m.Div(&a, &b, &g)
			} else {
				callErr = m.ExtendedGCD(&a, &b, &x, &y, &g)
			}
			s := &m.divisionRaw
			if route == 1 {
				s = &m.extendedRaw
			}
			r := integerRawInvocationReceipt{m, s, s.invocation}
			named := []*Int{&a, &b, &x, &y, &g}
			for i := uint8(0); i < s.count; i++ {
				named = append(named, &s.values[i])
			}
			integerPhysicalAudit(t, m, named...)
			if budget.Used() != integerActualOwnedBytes(m) {
				t.Fatal("immediate cold graph", route, prefix, budget.Used(), integerActualOwnedBytes(m))
			}
			if callErr != nil && !numericPhysicalReservationRefusal(callErr) {
				t.Fatal("literal body refusal", route, prefix, callErr)
			}
			before := s.values
			if callErr != nil && !s.bodySucceeded {
				for i := uint8(0); i < s.count; i++ {
					if s.values[i].cell != 0 && !m.isFailureRef(s.values[i].cell) {
						retained = true
					}
				}
				copied := *s
				if !errors.Is(m.finishRawSource(&copied), ErrPolynomialOwner) {
					t.Fatal("copied actual RAW source frame accepted")
				}
				forged := r
				forged.frame = &copied
				serviceMust(t, m.cleanupRawInvocation(&forged, true))
				if !reflect.DeepEqual(copied, *s) {
					t.Fatal("copied child consumed actual RAW locals")
				}
			}
			serviceMust(t, m.cleanupRawInvocation(&r, false))
			if callErr != nil && !s.bodySucceeded && s.values != before {
				t.Fatal("failed body reached absent end DEL", route, prefix)
			}
			integerPhysicalAudit(t, m, named...)
			serviceMust(t, m.cleanupRawInvocation(&r, true))
			integerPhysicalAudit(t, m, &a, &b, &x, &y, &g)
			budget.FailAfter(-1)
			if callErr == nil {
				success = true
				if route == 0 {
					expected := make([]uint32, 32)
					expected[0] = 1
					expected[31] = 1
					var want Int
					serviceMust(t, m.SetLimbs(&want, expected))
					serviceMust(t, m.Neg(&want))
					cmp, e := m.Compare(&g, &want)
					serviceMust(t, e)
					if cmp != 0 {
						t.Fatal("independent Euclidean power quotient", g, want)
					}
					serviceMust(t, m.Del(&want))
				} else {
					if !g.IsSmall() || g != NewInt(1) {
						t.Fatal("independent odd/power-two GCD", g)
					}
					var ax, by, sum Int
					serviceMust(t, m.Mul(&a, &x, &ax))
					serviceMust(t, m.Mul(&b, &y, &by))
					serviceMust(t, m.Add(&ax, &by, &sum))
					cmp, e := m.Compare(&sum, &g)
					serviceMust(t, e)
					if cmp != 0 {
						t.Fatal("independent Bezout identity")
					}
					for _, v := range []*Int{&ax, &by, &sum} {
						serviceMust(t, m.Del(v))
					}
				}
				// A new successful actual frame must preserve stale old cleanup refusal.
				if route == 0 {
					negative, divisor := NewInt(-7), NewInt(2)
					serviceMust(t, m.Div(&negative, &divisor, &g))
				} else {
					two, zero := NewInt(2), NewInt(0)
					serviceMust(t, m.ExtendedGCD(&two, &zero, &x, &y, &g))
				}
				saved := *s
				serviceMust(t, m.cleanupRawInvocation(&r, true))
				if !reflect.DeepEqual(saved, *s) {
					t.Fatal("stale receipt mutated later actual frame")
				}
				t.Log("public RAW first success", route, prefix, "actual failed-body BIG RAW", retained)
			}
			for _, v := range []*Int{&a, &b, &x, &y, &g} {
				serviceMust(t, m.Del(v))
			}
		}
		if !success || ((route == 1 || sourceExtendedGCDDebug) && !retained) {
			t.Fatal("first success/real retained RAW body missing", route, success, retained)
		}
	}
}

func TestIntegerPublicRawStaleForeignAndCopiedRefuseActiveReplacement(t *testing.T) {
	proved := false
	for prefix := int64(0); prefix < 80 && !proved; prefix++ {
		m, budget := mpzPublicationOwner(t)
		two, zero := NewInt(2), NewInt(0)
		var x, y, g, a, b Int
		serviceMust(t, m.ExtendedGCD(&two, &zero, &x, &y, &g))
		old := integerRawInvocationReceipt{m, &m.extendedRaw, m.extendedRaw.invocation}
		digits := make([]uint32, 64)
		digits[0] = 3
		digits[63] = 1
		serviceMust(t, m.SetLimbs(&a, digits))
		other := make([]uint32, 33)
		other[32] = 1
		serviceMust(t, m.SetLimbs(&b, other))
		budget.FailAfter(prefix)
		e := m.ExtendedGCD(&a, &b, &x, &y, &g)
		s := &m.extendedRaw
		current := integerRawInvocationReceipt{m, s, s.invocation}
		bigLocal := false
		for i := uint8(0); i < s.count; i++ {
			if s.values[i].cell != 0 && !m.isFailureRef(s.values[i].cell) {
				bigLocal = true
			}
		}
		if e != nil && !numericPhysicalReservationRefusal(e) {
			t.Fatal("actual replacement refusal", prefix, e)
		}
		if e != nil && bigLocal && !s.bodySucceeded {
			before := *s
			runtime := m.memory.Stats()
			cells := m.cells.Stats()
			copied := before
			copyReceipt := current
			copyReceipt.frame = &copied
			copiedManager := *m
			foreign := current
			foreign.manager = &copiedManager
			serviceMust(t, m.cleanupRawInvocation(&old, true))
			serviceMust(t, m.cleanupRawInvocation(&copyReceipt, true))
			serviceMust(t, m.cleanupRawInvocation(&foreign, true))
			if !reflect.DeepEqual(before, *s) || !reflect.DeepEqual(before, copied) || runtime != m.memory.Stats() || cells != m.cells.Stats() {
				t.Fatal("stale/foreign/copied receipt changed actual newer cohort")
			}
			named := []*Int{&a, &b, &x, &y, &g}
			for i := uint8(0); i < s.count; i++ {
				named = append(named, &s.values[i])
			}
			integerPhysicalAudit(t, m, named...)
			proved = true
			t.Log("actual live replacement prefix", prefix, "old/new", old.invocation, current.invocation)
		}
		serviceMust(t, m.cleanupRawInvocation(&current, false))
		serviceMust(t, m.cleanupRawInvocation(&current, true))
		budget.FailAfter(-1)
		for _, v := range []*Int{&a, &b, &x, &y, &g} {
			serviceMust(t, m.Del(v))
		}
	}
	if !proved {
		t.Fatal("actual retained new cohort not reached")
	}
}

func TestIntegerPublicRawOwnAfterSourceFinalize(t *testing.T) {
	proved := false
	for prefix := int64(0); prefix < 32 && !proved; prefix++ {
		m, budget := mpzPublicationOwner(t)
		var a, b, x, y, g Int
		digits := make([]uint32, 64)
		digits[0] = 3
		digits[63] = 1
		serviceMust(t, m.SetLimbs(&a, digits))
		digits[0] = 7
		serviceMust(t, m.SetLimbs(&b, digits))
		budget.FailAfter(prefix)
		e := m.ExtendedGCD(&a, &b, &x, &y, &g)
		s := &m.extendedRaw
		r := integerRawInvocationReceipt{m, s, s.invocation}
		bigLocal := false
		for i := uint8(0); i < s.count; i++ {
			if s.values[i].cell != 0 && !m.isFailureRef(s.values[i].cell) {
				bigLocal = true
			}
		}
		named := []*Int{&a, &b, &x, &y, &g}
		for i := uint8(0); i < s.count; i++ {
			named = append(named, &s.values[i])
		}
		integerPhysicalAudit(t, m, named...)
		budget.FailAfter(-1)
		if e != nil && bigLocal && !s.bodySucceeded && !polynomialNegNumericPending(&s.numeric) && m.tmp[0] != 0 && m.tmp[1] != 0 {
			serviceMust(t, m.SourceFinalize())
			serviceMust(t, m.cleanupRawInvocation(&r, true))
			integerPhysicalAudit(t, m, &a, &b, &x, &y, &g)
			serviceMust(t, m.CloseChecked())
			if budget.Used() != 0 {
				t.Fatal("source-finalized RAW owner graph remains", budget.Used())
			}
			proved = true
		} else {
			serviceMust(t, m.cleanupRawInvocation(&r, true))
		}
	}
	if !proved {
		t.Fatal("real failed RAW body source-finalize path absent")
	}
}

func TestIntegerPublicRawUnpublishedCellOwnAfterSourceFinalize(t *testing.T) {
	proved := false
	for prefix := int64(0); prefix < 64 && !proved; prefix++ {
		m, budget := mpzPublicationOwner(t)
		var a, b, x, y, g Int
		digits := make([]uint32, 8)
		digits[0] = 3
		digits[7] = 1
		serviceMust(t, m.SetLimbs(&a, digits))
		digits[0] = 7
		serviceMust(t, m.SetLimbs(&b, digits))
		budget.FailAfter(prefix)
		e := m.ExtendedGCD(&a, &b, &x, &y, &g)
		s := &m.extendedRaw
		r := integerRawInvocationReceipt{m, s, s.invocation}
		named := []*Int{&a, &b, &x, &y, &g}
		for i := uint8(0); i < s.count; i++ {
			named = append(named, &s.values[i])
		}
		integerPhysicalAudit(t, m, named...)
		budget.FailAfter(-1)
		if e != nil && s.numeric.cell.object != (memory.ObjectRef{}) && m.tmp[0] != 0 && m.tmp[1] != 0 {
			serviceMust(t, m.SourceFinalize())
			serviceMust(t, m.cleanupRawInvocation(&r, true))
			integerPhysicalAudit(t, m, &a, &b, &x, &y, &g)
			serviceMust(t, m.CloseChecked())
			if budget.Used() != 0 {
				t.Fatal("source-finalized unpublished RAW owner graph remains", budget.Used())
			}
			proved = true
		} else {
			serviceMust(t, m.cleanupRawInvocation(&r, true))
		}
	}
	if !proved {
		t.Fatal("actual unpublished raw cell path absent")
	}
}

// Runtime failure transport can reenter public methods, but it receives no
// explicit RAW context. This proves exclusion at real public acquisition.
type integerRawPublicCallbackFixture struct {
	manager            *IntegerManager
	budget             *arena.Budget
	input, foreign     *Int
	called             bool
	reentry, copyError error
}

var integerRawPublicCallbackCurrent *integerRawPublicCallbackFixture

func integerRawPublicFailureTransport(f memory.Failure) error {
	s := integerRawPublicCallbackCurrent
	if s != nil && !s.called {
		s.called = true
		s.budget.FailAfter(-1)
		two, zero := NewInt(2), NewInt(0)
		var x, y, g Int
		s.reentry = s.manager.ExtendedGCD(&two, &zero, &x, &y, &g)
		s.copyError = s.manager.Copy(s.foreign, s.input)
	}
	return f
}
func TestIntegerPublicRawRuntimeCallbackHasNoAmbientAcquisition(t *testing.T) {
	proved := false
	for prefix := int64(0); prefix < 50 && !proved; prefix++ {
		budget := arena.NewBudget(1 << 24)
		rt, e := memory.NewRuntime(memory.RuntimeConfig{Budget: budget, Transport: integerRawPublicFailureTransport})
		serviceMust(t, e)
		cfg := serviceConfig(budget).Integers
		cfg.Memory = rt
		cfg.Profile = memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
		m, e := NewIntegerManager(cfg)
		serviceMust(t, e)
		var a, b, x, y, g, foreign Int
		digits := make([]uint32, 64)
		digits[0] = 3
		digits[63] = 1
		serviceMust(t, m.SetLimbs(&a, digits))
		serviceMust(t, m.SetInt64(&b, 0))
		fixture := integerRawPublicCallbackFixture{manager: m, budget: budget, input: &a, foreign: &foreign}
		integerRawPublicCallbackCurrent = &fixture
		budget.FailAfter(prefix)
		returned := m.ExtendedGCD(&a, &b, &x, &y, &g)
		integerRawPublicCallbackCurrent = nil
		budget.FailAfter(-1)
		if fixture.called {
			proved = true
			if returned == nil || !errors.Is(fixture.reentry, ErrPolynomialPending) || fixture.copyError != nil || foreign.cell == 0 {
				t.Fatal("actual callback acquisition/reentry", prefix, returned, fixture.reentry, fixture.copyError, foreign)
			}
			integerPhysicalAudit(t, m, &a, &b, &x, &y, &g, &foreign)
			serviceMust(t, m.CleanupExtendedGCD())
			cmp, e := m.Compare(&foreign, &a)
			serviceMust(t, e)
			if cmp != 0 {
				t.Fatal("callback foreign numeral consumed by RAW abort")
			}
			integerPhysicalAudit(t, m, &a, &b, &x, &y, &g, &foreign)
		} else {
			serviceMust(t, m.CleanupExtendedGCD())
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

func TestIntegerPublicRawContextNonceDoesNotFollowEmbeddedReceiptReuse(t *testing.T) {
	m, _ := mpzPublicationOwner(t)
	s := &m.extendedRaw
	first, e := m.beginRawInvocation(s, 4)
	serviceMust(t, e)
	old := (integerAllocationInvocationContext{}).withRawInvocation(first)
	if old.roots(m) == nil {
		t.Fatal("actual executing context absent")
	}
	s.executing = false
	s.bodySucceeded = true
	serviceMust(t, m.finishRawSource(s))
	second, e := m.beginRawInvocation(s, 4)
	serviceMust(t, e)
	current := (integerAllocationInvocationContext{}).withRawInvocation(second)
	defer func() { s.executing = false; serviceMust(t, m.cleanupRawInvocation(second, true)) }()
	if old.roots(m) != nil {
		t.Fatal("old context silently followed reused embedded receipt")
	}
	if m.natural.divisionBufferReceipt(old, 0) != nil {
		t.Fatal("old context acquired Natural buffer binding")
	}
	if current.roots(m) == nil {
		t.Fatal("new actual context absent")
	}
	if b := m.natural.divisionBufferReceipt(current, 0); b == nil || !b.bound(true) {
		t.Fatal("actual new Natural member binding absent")
	}
	if sourceExtendedGCDDebug {
		view := func(n uint32) limbView { return limbView{external: []uint32{n}, size: 1} }
		before := m.natural.verificationInvocation
		serviceMust(t, m.natural.sourceVerifyDivisionWithAllocationContext(view(3), view(2), view(1), view(1), old))
		if !reflect.DeepEqual(before, m.natural.verificationInvocation) || s.verification.invocation != 0 {
			t.Fatal("old context acquired verification authority")
		}
		serviceMust(t, m.natural.sourceVerifyDivisionWithAllocationContext(view(3), view(2), view(1), view(1), current))
		if s.verification.invocation == 0 || m.natural.verificationInvocation.raw.invocation != current.rawInvocation {
			t.Fatal("actual new verification binding absent")
		}
	}
	integerPhysicalAudit(t, m)
	s.executing = false
	serviceMust(t, m.cleanupRawInvocation(second, true))
}
