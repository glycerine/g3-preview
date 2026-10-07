// Source: src/util/mpf.cpp:335-374; src/util/mpz.h:156-164.
// Licensed under the MIT license. Source author: Christoph Wintersteiger.
package util

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestArbitraryFloatFormattedSetUnpackCallbackOwnerRefusal(t *testing.T) {
	b := arena.NewBudget(1 << 27)
	var m *ArbitraryFloatManager
	var x, out ArbitraryFloat
	armed := false
	calls := 0
	rt, e := NewRuntime(RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f Failure) error {
		if armed {
			calls++
			t.Logf("actual callback flags value=%v copy=%v raw=%v", m.valueExecuting, m.integer.CopyAllocation.Executing, m.integer.rawInvocationExecuting())
			beforeX, beforeOut := x, out
			used := b.Used()
			scope := m.formattedSet
			iz := m.integer.SourceFinalizeStage
			qz := m.rational.cleanupStage
			for index, call := range []func() error{m.SourceFinalize, m.Close, m.integer.SourceFinalize, m.integer.CloseChecked, m.rational.SourceFinalize, m.rational.CloseChecked, func() error { return m.SetFormat(&out, 8, 24, ArbitraryFloatNearestEven, &x) }, func() error { return m.CleanupFormattedSet() }} {
				if err := call(); err != ErrNumericPending {
					t.Fatalf("actual callback owner refusal index%d: %v", index, err)
				}
			}
			if x != beforeX || out != beforeOut || b.Used() != used || m.formattedSet != scope || m.integer.SourceFinalizeStage != iz || m.rational.cleanupStage != qz {
				t.Fatal("callback mutated actual parent/older owner")
			}
		}
		return f
	}})
	serviceMust(t, e)
	m, e = NewArbitraryFloatManager(IntegerConfig{Memory: rt, Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	one := NewInt(1)
	serviceMust(t, m.SetComponents(&x, 8, 4000, false, 0, &one))
	_, e = m.powerTwoWrapper(3999, false)
	serviceMust(t, e)
	_, e = m.IsNaN(&x)
	serviceMust(t, e)
	_, e = m.IsInf(&x)
	serviceMust(t, e)
	_, e = m.IsZero(&x)
	serviceMust(t, e)
	_, e = m.IsNormal(&x)
	serviceMust(t, e)
	pad, e := rt.Allocate(uint64(99990 - rt.State().LocalSize - 8))
	serviceMust(t, e)
	serviceMust(t, rt.SetMaxSize(1))
	armed = true
	body := m.SetFormat(&out, 8, 24, ArbitraryFloatNearestEven, &x)
	armed = false
	if body == nil || calls != 1 {
		t.Fatal("reached real unpack allocation", body, calls)
	}
	integerPhysicalAuditWithRaw(t, m.integer, []Block{pad}, &x.significand, &out.significand)
	if b.Used() != rt.Stats().TotalBytes+m.OwnedStorageBytes() {
		t.Fatal("before-lift exact cold graph")
	}
	serviceMust(t, rt.SetMaxSize(0))
	serviceMust(t, rt.Deallocate(&pad))
	serviceMust(t, m.CleanupFormattedSetWithReceipt(nil, false))
	serviceMust(t, m.CleanupValue(&out))
	serviceMust(t, m.CleanupFormattedSetWithReceipt(nil, true))
	serviceMust(t, m.Del(&x))
	if e = m.SourceFinalize(); e != nil && !m.integer.sourceGuard {
		t.Fatal(e)
	}
	serviceMust(t, m.Close())
	serviceMust(t, rt.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
