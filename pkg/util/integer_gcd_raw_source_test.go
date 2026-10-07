// Source: src/util/mpz.cpp:730-817 (LEHMER_GCD), src/util/mpz.h:393-407.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestIntegerGCDRawEveryActualPrefix(t *testing.T) {
	t.Run("unshifted", func(t *testing.T) { integerGCDRawPrefixes(t, 0) })
	t.Run("2048bit-common-factor", func(t *testing.T) { integerGCDRawPrefixes(t, 2048) })
}
func integerGCDRawPrefixes(t *testing.T, shift uint32) {
	success, retained, childRetained := false, false, false
	for prefix := int64(0); prefix < 512; prefix++ {
		m, budget := mpzPublicationOwner(t)
		var a, b, g Int
		expected := NewInt(1)
		// Consecutive Fibonacci values: independent gcd1 and BIG Lehmer inputs.
		serviceMust(t, m.SetString(&a, "354224848179261915075"))
		serviceMust(t, m.SetString(&b, "573147844013817084101"))
		if shift != 0 {
			serviceMust(t, m.Mul2kInPlace(&a, shift))
			serviceMust(t, m.Mul2kInPlace(&b, shift))
			serviceMust(t, m.Mul2kInPlace(&expected, shift))
		}
		budget.FailAfter(prefix)
		e := m.GCD(&a, &b, &g)
		s := &m.GcdRaw
		r := integerRawInvocationReceipt{m, s, s.Invocation}
		if e != nil && !numericPhysicalReservationRefusal(e) {
			t.Fatal("actual GCD body refusal", prefix, e)
		}
		named := []*Int{&a, &b, &g, &expected}
		if m.MatchesAddMulInvocation(&s.AddMul) {
			named = append(named, &m.AddMulInvocation.Tmp)
			childRetained = childRetained || m.AddMulInvocation.Tmp.Cell != 0 && !m.IsFailureRef(m.AddMulInvocation.Tmp.Cell)
		}
		for j := uint8(0); j < s.Count; j++ {
			named = append(named, &s.Values[j])
			retained = retained || (e != nil && s.Values[j].Cell != 0 && !m.IsFailureRef(s.Values[j].Cell))
		}
		integerPhysicalAudit(t, m, named...)
		if budget.Used() != integerActualOwnedBytes(m) {
			t.Fatal("immediate five RAW + child cold graph", prefix, budget.Used(), integerActualOwnedBytes(m))
		}
		before := s.Values
		serviceMust(t, m.CleanupRawInvocation(&r, false))
		if e != nil && !s.BodySucceeded && s.Values != before {
			t.Fatal("failed GCD body executed absent five end DELs")
		}
		integerPhysicalAudit(t, m, named...)
		serviceMust(t, m.CleanupRawInvocation(&r, true))
		integerPhysicalAudit(t, m, &a, &b, &g, &expected)
		budget.FailAfter(-1)
		if e == nil {
			cmp, e := m.Compare(&g, &expected)
			serviceMust(t, e)
			if cmp != 0 {
				t.Fatal("independent consecutive Fibonacci gcd", g)
			}
			success = true
			t.Log("first actual GCD success", prefix)
		}
		serviceMust(t, m.Del(&a))
		serviceMust(t, m.Del(&b))
		serviceMust(t, m.Del(&g))
		serviceMust(t, m.Del(&expected))
		if success {
			break
		}
	}
	if !success || !retained || shift != 0 && !childRetained {
		t.Fatal("actual GCD RAW source horizon absent", success, retained)
	}
}

// Actual signed-host TLS guard domain; source output precedes end-only DEL.
func TestIntegerGCDRawReachedEndLifetime(t *testing.T) {
	budget := arena.NewBudget(1 << 26)
	rt, e := NewRuntime(RuntimeConfig{Budget: budget, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	cfg := serviceConfig(budget).Integers
	cfg.Memory = rt
	m, e := NewIntegerManager(cfg)
	serviceMust(t, e)
	var a, b, x, y, g Int
	digits := make([]uint32, 64)
	digits[0] = 7
	digits[63] = 1
	serviceMust(t, m.SetLimbs(&a, digits))
	serviceMust(t, m.EnsureTmp(0, 64))
	serviceMust(t, m.EnsureTmp(1, 64))
	// Execute the literal same division once to construct its actual shared buffers.
	serviceMust(t, m.GCD(&a, &a, &g))
	oldOutput := make([]uint32, 15000)
	oldOutput[14999] = 1
	serviceMust(t, m.SetLimbs(&g, oldOutput))
	contextTLSNearMinimum(t, rt)
	lane, e := rt.Allocate(8)
	serviceMust(t, e)
	delta := int64(-50000) - rt.State().LocalSize
	if e = rt.Reallocate(&lane, uint64(16)+uint64(delta)-8); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("actual signed allocation guard", e)
	}
	returned := m.GCD(&a, &a, &g)
	s := &m.GcdRaw
	r := integerRawInvocationReceipt{m, s, s.Invocation}
	if !errors.Is(returned, arena.ErrBounds) {
		t.Fatal("actual reached source error missing", returned)
	}
	captured := returned.Error()
	source := m.RawInvocationSourceCause(&r)
	cmp, e := m.Compare(&g, &a)
	serviceMust(t, e)
	if cmp != 0 || !s.BodySucceeded || source == nil {
		t.Fatal("source output before reached end DEL", s, g, a, returned)
	}
	named := []*Int{&a, &b, &x, &y, &g}
	for i := uint8(0); i < s.Count; i++ {
		named = append(named, &s.Values[i])
	}
	integerPhysicalAuditWithRaw(t, m, []Block{lane}, named...)
	if budget.Used() != integerActualOwnedBytes(m) {
		t.Fatal("immediate cold RAW graph", budget.Used(), integerActualOwnedBytes(m))
	}
	own := m.CleanupRawInvocation(&r, false)
	if own != nil && !errors.Is(own, arena.ErrBounds) {
		t.Fatal("false own phase", own)
	}
	if m.RawInvocationSourceCause(&r) != source {
		t.Fatal("false owning phase changed unexecuted RAW end DEL/history")
	}
	integerPhysicalAuditWithRaw(t, m, []Block{lane}, named...)
	recovery, e := rt.Allocate(2000000)
	serviceMust(t, e)
	serviceMust(t, m.CleanupRawInvocation(&r, true))
	if m.RawInvocationSourceCause(&r) != source || returned.Error() != captured {
		t.Fatal("owning acknowledgement changed source cause")
	}
	integerPhysicalAuditWithRaw(t, m, []Block{lane, recovery}, &a, &b, &x, &y, &g)
	serviceMust(t, m.GCD(&a, &a, &g))
	if s.Invocation == r.Invocation || s.sourceCause != nil || returned.Error() != captured || !errors.Is(returned, arena.ErrBounds) {
		t.Fatal("new actual invocation mutated old returned error")
	}
	for _, v := range []*Int{&a, &b, &x, &y, &g} {
		serviceMust(t, m.Del(v))
	}
	serviceMust(t, rt.Deallocate(&lane))
	serviceMust(t, m.CloseChecked())
	e = rt.Deallocate(&recovery)
	if e != nil && (!errors.Is(e, arena.ErrBounds) || recovery != (Block{})) {
		t.Fatal("actual recovery consumption", e)
	}
	contextTLSFinish(t, rt, rt.Close)
	if budget.Used() != 0 {
		t.Fatal("explicit complete owner graph remains", budget.Used())
	}
	if !errors.Is(returned, arena.ErrBounds) {
		t.Fatal("reached RAW end DEL source stop discarded", returned)
	}
}

func TestIntegerGCDRawActualBindingAndReusedContext(t *testing.T) {
	m, _ := mpzPublicationOwner(t)
	s := &m.GcdRaw
	first, e := m.beginRawInvocation(s, 5)
	serviceMust(t, e)
	old := (integerAllocationInvocationContext{}).withRawInvocation(first)
	foreign := *s
	foreignReceipt := integerRawInvocationReceipt{m, &foreign, s.Invocation}
	before := *s
	if _, e = m.beginRawInvocation(&foreign, 5); !errors.Is(e, ErrNumericOwner) {
		t.Fatal("copied begin", e)
	}
	if e = m.finishRawSource(&foreign); !errors.Is(e, ErrNumericOwner) {
		t.Fatal("copied source", e)
	}
	serviceMust(t, m.CleanupRawInvocation(&foreignReceipt, true))
	if *s != before || old.roots(m) == nil {
		t.Fatal("copied frame changed actual source")
	}
	s.executing = false
	s.BodySucceeded = true
	serviceMust(t, m.finishRawSource(s))
	second, e := m.beginRawInvocation(s, 5)
	serviceMust(t, e)
	current := (integerAllocationInvocationContext{}).withRawInvocation(second)
	if old.roots(m) != nil || m.Natural.DivisionBufferReceipt(old, 0) != nil {
		t.Fatal("old context retargeted reused embedded receipt")
	}
	if current.roots(m) == nil || m.Natural.DivisionBufferReceipt(current, 0) == nil {
		t.Fatal("current actual context absent")
	}
	s.executing = false
	serviceMust(t, m.CleanupRawInvocation(second, true))
	integerPhysicalAudit(t, m)
}

func TestIntegerGCDRawOwnAfterSourceFinalize(t *testing.T) {
	proved := false
	for prefix := int64(0); prefix < 64 && !proved; prefix++ {
		m, budget := mpzPublicationOwner(t)
		var a, b, g Int
		serviceMust(t, m.SetString(&a, "354224848179261915075"))
		serviceMust(t, m.SetString(&b, "573147844013817084101"))
		serviceMust(t, m.Mul2kInPlace(&a, 2048))
		serviceMust(t, m.Mul2kInPlace(&b, 2048))
		serviceMust(t, m.EnsureTmp(0, 128))
		serviceMust(t, m.EnsureTmp(1, 128))
		budget.FailAfter(prefix)
		e := m.GCD(&a, &b, &g)
		s := &m.GcdRaw
		r := integerRawInvocationReceipt{m, s, s.Invocation}
		integerPhysicalAudit(t, m, &a, &b, &g)
		budget.FailAfter(-1)
		child := m.MatchesAddMulInvocation(&s.AddMul)
		if e != nil {
			t.Log("actual prefix", prefix, e, "child", child, m.AddMulInvocation.Stage, m.AddMulInvocation.Tmp, "scratch", m.Tmp)
		}
		if e != nil && child && m.AddMulInvocation.Tmp.Cell != 0 && !m.IsFailureRef(m.AddMulInvocation.Tmp.Cell) {
			finalize := m.SourceFinalize()
			if finalize != nil && (!errors.Is(finalize, arena.ErrInvalidSlot) || !m.sourceGuard) {
				t.Fatal("literal source finalize", finalize)
			}
			if finalize != nil && m.Tmp[0] != 0 && m.Tmp[1] != 0 {
				t.Fatal("undefined shared scratch guard absent", m.Tmp)
			}
			serviceMust(t, m.CleanupRawInvocation(&r, true))
			integerPhysicalAudit(t, m, &a, &b, &g)
			serviceMust(t, m.CloseChecked())
			if budget.Used() != 0 {
				t.Fatal("source-finalized five RAW + child graph", budget.Used())
			}
			proved = true
			t.Log("actual source-finalize retained AddMul child prefix", prefix, "literal scratch guard", finalize)
		} else {
			serviceMust(t, m.CleanupRawInvocation(&r, true))
		}
	}
	if !proved {
		t.Fatal("actual retained GCD child after source finalize absent")
	}
}

// Invalid Go metadata guard domain: a saved child context cannot follow reuse
// of the embedded receipt. This grants no ordinary mathematical prefix credit.
func TestIntegerGCDRawChildContextDoesNotRetargetReceipt(t *testing.T) {
	proved := false
	for prefix := int64(0); prefix < 64 && !proved; prefix++ {
		m, budget := mpzPublicationOwner(t)
		var a, b, g Int
		serviceMust(t, m.SetString(&a, "354224848179261915075"))
		serviceMust(t, m.SetString(&b, "573147844013817084101"))
		serviceMust(t, m.Mul2kInPlace(&a, 2048))
		serviceMust(t, m.Mul2kInPlace(&b, 2048))
		budget.FailAfter(prefix)
		e := m.GCD(&a, &b, &g)
		s := &m.GcdRaw
		r := integerRawInvocationReceipt{m, s, s.Invocation}
		if e != nil && m.MatchesAddMulInvocation(&s.AddMul) {
			old := (integerAllocationInvocationContext{}).withAddMulInvocation(&s.AddMul)
			serviceMust(t, m.CleanupRawInvocation(&r, true))
			budget.FailAfter(-1)
			current, e := m.beginRawInvocation(s, 5)
			serviceMust(t, e)
			receipt := &s.AddMul
			f := &m.AddMulInvocation
			*f = integerAddMulScope{invocation: f.invocation + 1, Active: true, executing: true, parentContext: (integerAllocationInvocationContext{}).withRawInvocation(current)}
			*receipt = integerAddMulInvocationReceipt{m, f, f.invocation}
			followed := old.roots(m) != nil
			currentChild := (integerAllocationInvocationContext{}).withAddMulInvocation(receipt)
			accepted := currentChild.roots(m) != nil
			s.executing = false
			f.executing = false
			serviceMust(t, m.CleanupRawInvocation(current, true))
			if followed || !accepted {
				t.Fatal("saved child context retargeted later embedded AddMul receipt", followed, accepted)
			}
			proved = true
		} else {
			serviceMust(t, m.CleanupRawInvocation(&r, true))
			budget.FailAfter(-1)
		}
		integerPhysicalAudit(t, m, &a, &b, &g)
	}
	if !proved {
		t.Fatal("actual GCD AddMul child refusal absent")
	}
}

func TestIntegerGCDRawMinInt32ActualAbsLeaf(t *testing.T) {
	success := false
	for prefix := int64(0); prefix < 64 && !success; prefix++ {
		m, budget := mpzPublicationOwner(t)
		a, b, g := NewInt(-2147483648), NewInt(3), NewInt(17)
		budget.FailAfter(prefix)
		e := m.GCD(&a, &b, &g)
		s := &m.GcdRaw
		r := integerRawInvocationReceipt{m, s, s.Invocation}
		if e != nil && !numericPhysicalReservationRefusal(e) {
			t.Fatal("actual MinInt32 Abs leaf", prefix, e)
		}
		if s.Invocation != 1 || s.Count != 5 {
			t.Fatal("source MIN branch skipped actual RAW declarations")
		}
		integerPhysicalAudit(t, m, &a, &b, &g)
		if budget.Used() != integerActualOwnedBytes(m) {
			t.Fatal("immediate actual MIN leaf cold graph")
		}
		serviceMust(t, m.CleanupRawInvocation(&r, false))
		serviceMust(t, m.CleanupRawInvocation(&r, true))
		integerPhysicalAudit(t, m, &a, &b, &g)
		budget.FailAfter(-1)
		if e == nil {
			if g != NewInt(1) {
				t.Fatal("source MIN gcd", g)
			}
			success = true
			t.Log("actual MIN Abs horizon", prefix)
		}
	}
	if !success {
		t.Fatal("actual MinInt32 source branch success absent")
	}
	m, _ := mpzPublicationOwner(t)
	a, b, g := NewInt(6), NewInt(3), NewInt(17)
	serviceMust(t, m.GCD(&a, &b, &g))
	if m.GcdRaw.Invocation != 0 || m.GcdRaw.Active {
		t.Fatal("small shortcut acquired absent RAW frame")
	}
	a = NewInt(0)
	serviceMust(t, m.GCD(&a, &b, &g))
	if m.GcdRaw.Invocation != 0 || m.GcdRaw.Active {
		t.Fatal("zero shortcut acquired absent RAW frame")
	}
}

type integerGCDRawCallbackFixture struct {
	manager            *IntegerManager
	budget             *arena.Budget
	input, foreign     *Int
	called             bool
	reentry, copyError error
}

var integerGCDRawCallbackCurrent *integerGCDRawCallbackFixture

func integerGCDRawFailureTransport(f Failure) error {
	s := integerGCDRawCallbackCurrent
	if s != nil && !s.called {
		s.called = true
		s.budget.FailAfter(-1)
		var g Int
		s.reentry = s.manager.GCD(s.input, s.input, &g)
		s.copyError = s.manager.Copy(s.foreign, s.input)
	}
	return f
}
func TestIntegerGCDRawRuntimeCallbackHasNoAmbientAcquisition(t *testing.T) {
	proved := false
	for prefix := int64(0); prefix < 50 && !proved; prefix++ {
		budget := arena.NewBudget(1 << 24)
		rt, e := NewRuntime(RuntimeConfig{Budget: budget, Transport: integerGCDRawFailureTransport})
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
		fixture := integerGCDRawCallbackFixture{manager: m, budget: budget, input: &a, foreign: &foreign}
		integerGCDRawCallbackCurrent = &fixture
		budget.FailAfter(prefix)
		returned := m.GCD(&a, &a, &g)
		integerGCDRawCallbackCurrent = nil
		budget.FailAfter(-1)
		if fixture.called {
			proved = true
			if returned == nil || !errors.Is(fixture.reentry, ErrNumericPending) || fixture.copyError != nil || foreign.Cell == 0 {
				t.Fatal("actual callback acquisition/reentry", prefix, returned, fixture.reentry, fixture.copyError, foreign)
			}
			integerPhysicalAudit(t, m, &a, &b, &x, &y, &g, &foreign)
			serviceMust(t, m.CleanupGCDRaw())
			cmp, e := m.Compare(&foreign, &a)
			serviceMust(t, e)
			if cmp != 0 {
				t.Fatal("callback foreign numeral consumed by RAW abort")
			}
			integerPhysicalAudit(t, m, &a, &b, &x, &y, &g, &foreign)
		} else {
			serviceMust(t, m.CleanupGCDRaw())
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
