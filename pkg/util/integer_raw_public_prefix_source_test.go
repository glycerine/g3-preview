// Source: src/util/mpz.h:468-485; src/util/mpz.cpp:915-985.
package util

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
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
			s := &m.DivisionRaw
			if route == 1 {
				s = &m.ExtendedRaw
			}
			r := integerRawInvocationReceipt{m, s, s.Invocation}
			named := []*Int{&a, &b, &x, &y, &g}
			for i := uint8(0); i < s.Count; i++ {
				named = append(named, &s.Values[i])
			}
			integerPhysicalAudit(t, m, named...)
			if budget.Used() != integerActualOwnedBytes(m) {
				t.Fatal("immediate cold graph", route, prefix, budget.Used(), integerActualOwnedBytes(m))
			}
			if callErr != nil && !numericPhysicalReservationRefusal(callErr) {
				t.Fatal("literal body refusal", route, prefix, callErr)
			}
			before := s.Values
			if callErr != nil && !s.BodySucceeded {
				for i := uint8(0); i < s.Count; i++ {
					if s.Values[i].Cell != 0 && !m.IsFailureRef(s.Values[i].Cell) {
						retained = true
					}
				}
				copied := *s
				if !errors.Is(m.finishRawSource(&copied), ErrNumericOwner) {
					t.Fatal("copied actual RAW source frame accepted")
				}
				forged := r
				forged.Frame = &copied
				serviceMust(t, m.CleanupRawInvocation(&forged, true))
				if !reflect.DeepEqual(copied, *s) {
					t.Fatal("copied child consumed actual RAW locals")
				}
			}
			serviceMust(t, m.CleanupRawInvocation(&r, false))
			if callErr != nil && !s.BodySucceeded && s.Values != before {
				t.Fatal("failed body reached absent end DEL", route, prefix)
			}
			integerPhysicalAudit(t, m, named...)
			serviceMust(t, m.CleanupRawInvocation(&r, true))
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
				serviceMust(t, m.CleanupRawInvocation(&r, true))
				if !reflect.DeepEqual(saved, *s) {
					t.Fatal("stale receipt mutated later actual frame")
				}
				t.Log("public RAW first success", route, prefix, "actual failed-body BIG RAW", retained)
			}
			for _, v := range []*Int{&a, &b, &x, &y, &g} {
				serviceMust(t, m.Del(v))
			}
		}
		if !success || ((route == 1 || SourceExtendedGCDDebug) && !retained) {
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
		old := integerRawInvocationReceipt{m, &m.ExtendedRaw, m.ExtendedRaw.Invocation}
		digits := make([]uint32, 64)
		digits[0] = 3
		digits[63] = 1
		serviceMust(t, m.SetLimbs(&a, digits))
		other := make([]uint32, 33)
		other[32] = 1
		serviceMust(t, m.SetLimbs(&b, other))
		budget.FailAfter(prefix)
		e := m.ExtendedGCD(&a, &b, &x, &y, &g)
		s := &m.ExtendedRaw
		current := integerRawInvocationReceipt{m, s, s.Invocation}
		bigLocal := false
		for i := uint8(0); i < s.Count; i++ {
			if s.Values[i].Cell != 0 && !m.IsFailureRef(s.Values[i].Cell) {
				bigLocal = true
			}
		}
		if e != nil && !numericPhysicalReservationRefusal(e) {
			t.Fatal("actual replacement refusal", prefix, e)
		}
		if e != nil && bigLocal && !s.BodySucceeded {
			before := *s
			runtime := m.Memory.Stats()
			cells := m.Cells.Stats()
			copied := before
			copyReceipt := current
			copyReceipt.Frame = &copied
			copiedManager := *m
			foreign := current
			foreign.Manager = &copiedManager
			serviceMust(t, m.CleanupRawInvocation(&old, true))
			serviceMust(t, m.CleanupRawInvocation(&copyReceipt, true))
			serviceMust(t, m.CleanupRawInvocation(&foreign, true))
			if !reflect.DeepEqual(before, *s) || !reflect.DeepEqual(before, copied) || runtime != m.Memory.Stats() || cells != m.Cells.Stats() {
				t.Fatal("stale/foreign/copied receipt changed actual newer cohort")
			}
			named := []*Int{&a, &b, &x, &y, &g}
			for i := uint8(0); i < s.Count; i++ {
				named = append(named, &s.Values[i])
			}
			integerPhysicalAudit(t, m, named...)
			proved = true
			t.Log("actual live replacement prefix", prefix, "old/new", old.Invocation, current.Invocation)
		}
		serviceMust(t, m.CleanupRawInvocation(&current, false))
		serviceMust(t, m.CleanupRawInvocation(&current, true))
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
		s := &m.ExtendedRaw
		r := integerRawInvocationReceipt{m, s, s.Invocation}
		bigLocal := false
		for i := uint8(0); i < s.Count; i++ {
			if s.Values[i].Cell != 0 && !m.IsFailureRef(s.Values[i].Cell) {
				bigLocal = true
			}
		}
		named := []*Int{&a, &b, &x, &y, &g}
		for i := uint8(0); i < s.Count; i++ {
			named = append(named, &s.Values[i])
		}
		integerPhysicalAudit(t, m, named...)
		budget.FailAfter(-1)
		if e != nil && bigLocal && !s.BodySucceeded && !NumericAllocationPending(&s.Numeric) && m.Tmp[0] != 0 && m.Tmp[1] != 0 {
			serviceMust(t, m.SourceFinalize())
			serviceMust(t, m.CleanupRawInvocation(&r, true))
			integerPhysicalAudit(t, m, &a, &b, &x, &y, &g)
			serviceMust(t, m.CloseChecked())
			if budget.Used() != 0 {
				t.Fatal("source-finalized RAW owner graph remains", budget.Used())
			}
			proved = true
		} else {
			serviceMust(t, m.CleanupRawInvocation(&r, true))
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
		s := &m.ExtendedRaw
		r := integerRawInvocationReceipt{m, s, s.Invocation}
		named := []*Int{&a, &b, &x, &y, &g}
		for i := uint8(0); i < s.Count; i++ {
			named = append(named, &s.Values[i])
		}
		integerPhysicalAudit(t, m, named...)
		budget.FailAfter(-1)
		if e != nil && s.Numeric.Cell.Object != (ObjectRef{}) && m.Tmp[0] != 0 && m.Tmp[1] != 0 {
			serviceMust(t, m.SourceFinalize())
			serviceMust(t, m.CleanupRawInvocation(&r, true))
			integerPhysicalAudit(t, m, &a, &b, &x, &y, &g)
			serviceMust(t, m.CloseChecked())
			if budget.Used() != 0 {
				t.Fatal("source-finalized unpublished RAW owner graph remains", budget.Used())
			}
			proved = true
		} else {
			serviceMust(t, m.CleanupRawInvocation(&r, true))
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

func integerRawPublicFailureTransport(f Failure) error {
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
		rt, e := NewRuntime(RuntimeConfig{Budget: budget, Transport: integerRawPublicFailureTransport})
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
		fixture := integerRawPublicCallbackFixture{manager: m, budget: budget, input: &a, foreign: &foreign}
		integerRawPublicCallbackCurrent = &fixture
		budget.FailAfter(prefix)
		returned := m.ExtendedGCD(&a, &b, &x, &y, &g)
		integerRawPublicCallbackCurrent = nil
		budget.FailAfter(-1)
		if fixture.called {
			proved = true
			if returned == nil || !errors.Is(fixture.reentry, ErrNumericPending) || fixture.copyError != nil || foreign.Cell == 0 {
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
	s := &m.ExtendedRaw
	first, e := m.beginRawInvocation(s, 4)
	serviceMust(t, e)
	old := (integerAllocationInvocationContext{}).withRawInvocation(first)
	if old.roots(m) == nil {
		t.Fatal("actual executing context absent")
	}
	s.executing = false
	s.BodySucceeded = true
	serviceMust(t, m.finishRawSource(s))
	second, e := m.beginRawInvocation(s, 4)
	serviceMust(t, e)
	current := (integerAllocationInvocationContext{}).withRawInvocation(second)
	defer func() { s.executing = false; serviceMust(t, m.CleanupRawInvocation(second, true)) }()
	if old.roots(m) != nil {
		t.Fatal("old context silently followed reused embedded receipt")
	}
	if m.Natural.DivisionBufferReceipt(old, 0) != nil {
		t.Fatal("old context acquired Natural buffer binding")
	}
	if current.roots(m) == nil {
		t.Fatal("new actual context absent")
	}
	if b := m.Natural.DivisionBufferReceipt(current, 0); b == nil || !b.Bound(true) {
		t.Fatal("actual new Natural member binding absent")
	}

	integerPhysicalAudit(t, m)
	s.executing = false
	serviceMust(t, m.CleanupRawInvocation(second, true))
}
