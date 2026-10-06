// Source: src/util/mpf.cpp:1984-2027; Runtime signed-host guard helper profile.
// Licensed under the MIT license. Source author: Christoph Wintersteiger.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestArbitraryFloatRoundActualTransportOwnerGuards(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	var m *ArbitraryFloatManager
	var x ArbitraryFloat
	armed := false
	calls := 0
	var oldContext integerAllocationInvocationContext
	rt, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f memory.Failure) error {
		if armed {
			calls++
			before := x
			used := b.Used()
			stage := m.completedMaps
			source, own := m.sourceCause, m.ownCause
			scope := m.roundScope
			oldContext = integerAllocationInvocationContext{mpfRound: &m.roundScope, mpfRoundInvocation: m.roundScope.invocation}
			if oldContext.roots(m.integer) != nil {
				t.Fatal("callback acquired Round roots")
			}
			copied := m.roundScope
			if (integerAllocationInvocationContext{mpfRound: &copied, mpfRoundInvocation: copied.invocation}).roundFrame(m.integer, false) != nil {
				t.Fatal("copied scope acquired authority")
			}
			for _, err := range []error{m.SourceFinalize(), m.Close(), m.integer.SourceFinalize(), m.integer.CloseChecked(), m.rational.SourceFinalize(), m.rational.CloseChecked(), m.CleanupRound(), m.Round(ArbitraryFloatNearestEven, &x)} {
				if err != ErrPolynomialPending {
					t.Fatal("callback owner guard", err)
				}
			}
			if x != before || b.Used() != used || m.completedMaps != stage || m.sourceCause != source || m.ownCause != own || m.roundScope != scope {
				t.Fatal("callback changed older state")
			}
		}
		return f
	}})
	serviceMust(t, e)
	m, e = NewArbitraryFloatManager(IntegerConfig{Memory: rt, Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	for _, n := range []uint32{2004, 2002, 2000, 1999} {
		_, e = m.roundPower(n, false)
		serviceMust(t, e)
	}
	_, e = m.roundBound(8, false)
	serviceMust(t, e)
	_, e = m.roundBound(8, true)
	serviceMust(t, e)
	digits := make([]uint32, 63)
	digits[0] = 1
	digits[56] = 1 << 8
	digits[62] = 1 << 18
	var sig Int
	serviceMust(t, m.integer.SetLimbs(&sig, digits))
	serviceMust(t, m.SetComponents(&x, 8, 2000, false, -5000, &sig))
	pad, e := rt.Allocate(uint64(99990 - rt.State().LocalSize - 8))
	serviceMust(t, e)
	serviceMust(t, rt.SetMaxSize(1))
	armed = true
	body := m.Round(ArbitraryFloatNearestEven, &x)
	armed = false
	if calls != 1 {
		t.Fatal("actual returning transport", calls, body)
	}
	if body == nil {
		t.Fatal("actual reached allocation did not stop")
	}
	s := &m.roundScope
	integerPhysicalAuditWithRaw(t, m.integer, []memory.Block{pad}, &sig, &x.significand, &s.sigma, &s.cap, &s.sticky)
	if b.Used() != rt.Stats().TotalBytes+m.OwnedStorageBytes() {
		t.Fatal("before lift exact cold graph")
	}
	pending, e := m.integer.allocator.PendingRoots()
	serviceMust(t, e)
	retained := false
	if block := s.transferred.Block(); block != (memory.Block{}) {
		retained = true
		if s.history == nil || s.history.originalPending != block {
			t.Fatal("transferred allocation lost issued leaf token")
		}
	}
	if s.numeric.cell.object != (memory.ObjectRef{}) {
		retained = true
		if s.history == nil || s.history.originalCell != s.numeric.cell {
			t.Fatal("detached cell lost issued backing token")
		}
	}
	for i, p := range pending {
		if p != (memory.Block{}) {
			retained = true
			if s.numeric.allocator[i] != p {
				t.Fatal("actual reached raw allocation prefix lacks round child", body, p)
			}
		}
	}
	for i := range m.integer.natural.division {
		p := m.integer.natural.division[i].pending
		if p != (memory.Block{}) {
			retained = true
			if s.buffers[i].pending != p {
				t.Fatal("actual reached Natural buffer lacks round child", body, p)
			}
		}
	}
	if !retained {
		t.Fatal("recipe did not reach actual pending prefix", body)
	}
	serviceMust(t, rt.SetMaxSize(0))

	if s.history == nil {
		t.Fatal("actual leaf did not issue history")
	}
	oldReceipt := s.history.issued
	copiedHistory := *s.history
	invalidReceipt := oldReceipt
	invalidReceipt.frame = &copiedHistory
	used, state := b.Used(), rt.State()
	if err := m.CleanupRoundWithReceipt(&invalidReceipt, true); err != memory.ErrRuntimeOwner {
		t.Fatal("copied history guard", err)
	}
	if used != b.Used() || state != rt.State() {
		t.Fatal("copied history mutated owner")
	}
	serviceMust(t, m.CleanupRound())
	laterSig := NewInt((1 << 26) + 4)
	serviceMust(t, m.SetComponents(&x, 8, 24, false, 0, &laterSig))
	serviceMust(t, m.Round(ArbitraryFloatNearestEven, &x))
	if oldContext.roundFrame(m.integer, false) != nil || oldContext.roots(m.integer) != nil {
		t.Fatal("saved context retargeted later embedded frame")
	}
	used, state = b.Used(), rt.State()
	serviceMust(t, m.CleanupRoundWithReceipt(&oldReceipt, true))
	if used != b.Used() || state != rt.State() {
		t.Fatal("old receipt swept later child")
	}
	serviceMust(t, m.CleanupRound())
	serviceMust(t, m.Del(&x))
	serviceMust(t, m.integer.Del(&sig))
	if e := m.SourceFinalize(); e != nil && !m.integer.sourceGuard {
		t.Fatal(e)
	}
	serviceMust(t, m.Close())
	contextTLSFinish(t, rt, func() error {
		if pad == (memory.Block{}) {
			return nil
		}
		return rt.Deallocate(&pad)
	})
	contextTLSFinish(t, rt, rt.Close)
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
