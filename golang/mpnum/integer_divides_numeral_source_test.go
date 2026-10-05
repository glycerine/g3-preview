// Source: src/util/mpz.cpp:2054-2069; src/util/scoped_numeral.h:31-33.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

type integerDividesReentryFixture struct {
	manager  *IntegerManager
	finalize bool
	called   bool
	changed  bool
	result   error
	context  integerAllocationInvocationContext
}

var integerDividesReentryCurrent *integerDividesReentryFixture

func integerDividesReentryTransport(f memory.Failure) error {
	s := integerDividesReentryCurrent
	if s != nil && !s.called {
		s.called = true
		m := s.manager
		s.context = integerAllocationInvocationContext{boolDivides: &m.dividesNumeral.receipt, boolDividesInvocation: m.dividesNumeral.invocation}
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
func TestIntegerDividesScopedNUMExecutingOwnerRefusal(t *testing.T) {
	for _, finalize := range []bool{true, false} {
		budget := arena.NewBudget(1 << 24)
		rt, e := memory.NewRuntime(memory.RuntimeConfig{Budget: budget, Transport: integerDividesReentryTransport})
		serviceMust(t, e)
		cfg := serviceConfig(budget).Integers
		cfg.Memory = rt
		cfg.Profile = memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
		m, e := NewIntegerManager(cfg)
		serviceMust(t, e)
		var a, b Int
		digits := make([]uint32, 64)
		digits[0] = 3
		digits[63] = 1
		serviceMust(t, m.SetLimbs(&b, digits))
		a = NewInt(7)
		fixture := integerDividesReentryFixture{manager: m, finalize: finalize}
		integerDividesReentryCurrent = &fixture
		budget.FailAfter(0)
		_, body := m.Divides(&a, &b)
		integerDividesReentryCurrent = nil
		if !fixture.called || body == nil || !errors.Is(fixture.result, ErrPolynomialPending) || fixture.changed {
			t.Fatalf("executing scoped NUM owner refusal: %+v body=%v", fixture, body)
		}
		integerPhysicalAudit(t, m, &a, &b, &m.dividesNumeral.numeral)
		budget.FailAfter(-1)
		serviceMust(t, m.cleanupDividesNumeralInvocation(&m.dividesNumeral.receipt, false))
		serviceMust(t, m.cleanupDividesNumeralInvocation(&m.dividesNumeral.receipt, true))
		serviceMust(t, m.Del(&b))
		serviceMust(t, m.CloseChecked())
		serviceMust(t, rt.Close())
	}
}

func TestIntegerDividesScopedNUMBeforeZeroBranch(t *testing.T) {
	m, _ := integerManager(t)
	zero, one := NewInt(0), NewInt(1)
	result, e := m.dividesWithAllocationContext(&zero, &one, integerAllocationInvocationContext{})
	serviceMust(t, e)
	s := &m.dividesNumeral
	if result || s.invocation == 0 || s.receipt.manager != m || s.receipt.frame != s || s.receipt.invocation != s.invocation || !s.constructed || !s.ended || s.active || s.numeral != (Int{}) {
		t.Fatal("literal scoped tmp must construct before zero branch and DT on exit", result, s)
	}
	previous := s.invocation
	result, e = m.dividesWithAllocationContext(&zero, &zero, integerAllocationInvocationContext{})
	serviceMust(t, e)
	if !result || s.invocation <= previous || !s.ended {
		t.Fatal("literal zero divides zero/new actual scoped invocation", result, s.invocation)
	}
}

// Invalid Go descriptor domain: authority still names only the actual guarded
// primitive frame, even when a saved pointer addresses the reused receipt.
func TestIntegerDividesScopedNUMCopiedAndRetargetedContextsRefuse(t *testing.T) {
	m, _ := integerManager(t)
	r, e := m.beginDividesNumeralInvocation(integerAllocationInvocationContext{})
	serviceMust(t, e)
	old := integerAllocationInvocationContext{boolDivides: r, boolDividesInvocation: r.invocation}
	if old.roots(m) != &r.frame.numeric {
		t.Fatal("current actual scoped invocation")
	}
	buffer := m.natural.divisionBufferReceipt(old, 0)
	if buffer == nil || !buffer.bound(true) {
		t.Fatal("current actual Natural member binding")
	}
	copyBuffer := *buffer
	if copyBuffer.bound(true) || copyBuffer.bound(false) {
		t.Fatal("copied buffer has no authority")
	}
	copyFrame := *r.frame
	copied := integerDividesNumeralInvocationReceipt{m, &copyFrame, r.invocation}
	before := r.frame.numeral
	if m.dividesNumeralContextRoots(integerAllocationInvocationContext{boolDivides: &copied, boolDividesInvocation: copied.invocation}) != nil {
		t.Fatal("copied child accepted")
	}
	if !errors.Is(m.finishDividesNumeralSource(&copyFrame), ErrPolynomialOwner) {
		t.Fatal("copied source frame accepted")
	}
	serviceMust(t, m.cleanupDividesNumeralInvocation(&copied, true))
	if r.frame.numeral != before || !r.frame.active {
		t.Fatal("copied child mutated real scope")
	}
	r.frame.executing = false
	serviceMust(t, m.finishDividesNumeralSource(r.frame))
	next, e := m.beginDividesNumeralInvocation(integerAllocationInvocationContext{})
	serviceMust(t, e)
	current := integerAllocationInvocationContext{boolDivides: next, boolDividesInvocation: next.invocation}
	if old.boolDivides != current.boolDivides || old.boolDividesInvocation == current.boolDividesInvocation || old.roots(m) != nil || m.natural.divisionBufferReceipt(old, 0) != nil || current.roots(m) != &next.frame.numeric {
		t.Fatal("saved embedded pointer retargeted invocation")
	}
	next.frame.executing = false
	serviceMust(t, m.finishDividesNumeralSource(next.frame))
}

// Actual signed-host TLS guard domain, not ordinary mathematical exhaustion.
func TestIntegerDividesScopedNUMSourceDTAndImmutableHistory(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	rt, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	cfg := serviceConfig(b).Integers
	cfg.Memory = rt
	m, e := NewIntegerManager(cfg)
	serviceMust(t, e)
	var a, d Int
	words := make([]uint32, 64)
	words[63] = 2
	serviceMust(t, m.SetLimbs(&a, words))
	words[0] = 7
	words[63] = 3
	serviceMust(t, m.SetLimbs(&d, words))
	_, e = m.Divides(&a, &d)
	serviceMust(t, e)
	serviceMust(t, m.ensureTmp(0, 64))
	serviceMust(t, m.ensureTmp(1, 64))
	contextTLSNearMinimum(t, rt)
	lane, e := rt.Allocate(8)
	serviceMust(t, e)
	delta := int64(-99900) - rt.State().LocalSize
	if e = rt.Reallocate(&lane, uint64(16)+uint64(delta)-8); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("actual signed allocation guard", e)
	}
	result, returned := m.Divides(&a, &d)
	s := &m.dividesNumeral
	r := s.receipt
	if returned == nil || !errors.Is(returned, arena.ErrBounds) || result || s.sourceCause == nil || s.bodyCause != nil {
		t.Fatal("literal completed Rem then scoped NUM DT stop", result, returned, s.bodyCause, s.sourceCause, "verification", m.natural.verificationInvocation.constructed, m.natural.verificationInvocation.sourceCause, m.natural.verificationInvocation.active, "num", s.ended, s.numeral)
	}
	captured, source := returned.Error(), s.sourceCause
	integerPhysicalAuditWithRaw(t, m, []memory.Block{lane}, &a, &d, &s.numeral)
	if b.Used() != integerActualOwnedBytes(m) {
		t.Fatal("immediate named cold graph", b.Used(), integerActualOwnedBytes(m))
	}
	own := m.cleanupDividesNumeralInvocation(&r, false)
	if own != nil && !errors.Is(own, arena.ErrBounds) {
		t.Fatal("false own suffix", own)
	}
	if m.dividesNumeralInvocationSourceCause(&r) != source || m.finishDividesNumeralSource(s) != source {
		t.Fatal("first actual source DT cause lost after own suffix")
	}
	recovery, e := rt.Allocate(2000000)
	serviceMust(t, e)
	serviceMust(t, m.cleanupDividesNumeralInvocation(&r, true))
	if m.dividesNumeralInvocationSourceCause(&r) != source || returned.Error() != captured {
		t.Fatal("own ack changed source history")
	}
	zero := NewInt(0)
	_, e = m.Divides(&zero, &zero)
	serviceMust(t, e)
	if s.invocation == r.invocation || returned.Error() != captured || !errors.Is(returned, arena.ErrBounds) {
		t.Fatal("later actual invocation retargeted returned cause")
	}
	serviceMust(t, m.Del(&a))
	serviceMust(t, m.Del(&d))
	serviceMust(t, rt.Deallocate(&lane))
	serviceMust(t, m.CloseChecked())
	e = rt.Deallocate(&recovery)
	if e != nil && (!errors.Is(e, arena.ErrBounds) || recovery != (memory.Block{})) {
		t.Fatal("recovery consumed outcome", e)
	}
	contextTLSFinish(t, rt, rt.Close)
	if b.Used() != 0 {
		t.Fatal("explicit complete owner roots", b.Used())
	}
}

func TestIntegerDividesScopedNUMEveryActualRemPrefix(t *testing.T) {
	success, acquired := false, false
	for prefix := int64(0); prefix < 256 && !success; prefix++ {
		m, budget := mpzPublicationOwner(t)
		a, b := Int{}, Int{}
		modulus := make([]uint32, 513)
		modulus[512] = 1
		dividend := make([]uint32, 640)
		dividend[0] = 1
		dividend[500] = 1
		dividend[639] = 1
		serviceMust(t, m.SetLimbs(&a, modulus))
		serviceMust(t, m.SetLimbs(&b, dividend))
		budget.FailAfter(prefix)
		result, body := m.dividesWithAllocationContext(&a, &b, integerAllocationInvocationContext{})
		s := &m.dividesNumeral
		integerPhysicalAudit(t, m, &a, &b, &s.numeral)
		if body != nil && !numericPhysicalReservationRefusal(body) {
			t.Fatal("actual Rem/source NUM DT", prefix, body)
		}
		if polynomialNegNumericPending(&s.numeric) {
			acquired = true
			if s.receipt.frame != s || s.receipt.invocation != s.invocation || !s.active {
				t.Fatal("actual pending leaf missing scoped NUM tuple", prefix)
			}
		}
		for i := range s.buffers {
			if s.buffers[i].pending != m.natural.division[i].pending && m.natural.division[i].pending != (naturalPNBufferReceipt{}).pending {
				t.Fatal("actual Rem shared-buffer pending missing child", prefix, i)
			}
		}
		budget.FailAfter(-1)
		serviceMust(t, m.cleanupDividesNumeralInvocation(&s.receipt, false))
		serviceMust(t, m.cleanupDividesNumeralInvocation(&s.receipt, true))
		integerPhysicalAudit(t, m, &a, &b)
		if body == nil {
			if result {
				t.Fatal("independent nonzero remainder", prefix)
			}
			success = true
			t.Log("actual Rem first success", prefix)
		}
		serviceMust(t, m.Del(&a))
		serviceMust(t, m.Del(&b))
	}
	if !success || !acquired {
		t.Fatal("actual pending/first success not reached", success, acquired)
	}
}
