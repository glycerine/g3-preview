// Source: src/util/vector.h:59-100; src/util/mpf.h:65-73;
// src/util/memory_manager.cpp:274-289.
// Source author: Leonardo de Moura; Christoph Wintersteiger.
// Licensed under the MIT license; see LICENSE.txt.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestArbitraryFloatVectorGrowthActualTransportProtectsOwnedMembers(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	var m *ArbitraryFloatManager
	active, calls := false, 0
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f memory.Failure) error {
		if active {
			calls++
			oldBody, oldSource, oldOwn := m.bodyCause, m.sourceCause, m.ownCause
			for _, call := range []func() error{m.Close, m.SourceFinalize, m.rational.CloseChecked, m.rational.SourceFinalize, m.integer.CloseChecked, m.integer.SourceFinalize} {
				if e := call(); !errors.Is(e, ErrPolynomialPending) {
					t.Fatal("actual parent/member lifecycle callback", e)
				}
			}
			if m.bodyCause != oldBody || m.sourceCause != oldSource || m.ownCause != oldOwn || m.rational.closing || m.rational.sourceFinalizing || m.integer.sourceFinalizing || m.integer.closed {
				t.Fatal("callback mutated exact owner history or member cursor")
			}
		}
		return f
	}})
	serviceMust(t, e)
	m, e = NewArbitraryFloatManager(IntegerConfig{Memory: r, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}})
	serviceMust(t, e)
	padding, e := r.Allocate(uint64(99990 - r.State().LocalSize - 8))
	serviceMust(t, e)
	serviceMust(t, r.SetMaxSize(1))
	active = true
	raw, call := m.GrowRuntimeValueStorage(memory.Block{}, 72)
	active = false
	if call == nil || calls == 0 || raw == (memory.Block{}) || m.valueExecuting || r.FailureTransportActive() {
		t.Fatal("actual returned unjoined value backing", call, calls, raw)
	}
	integerPhysicalAuditWithRaw(t, m.integer, []memory.Block{padding, raw})
	if b.Used() != m.header+m.rational.headerBytes+integerActualOwnedBytes(m.integer)+m.powers[0].StorageBytes()+m.powers[1].StorageBytes()+m.powers[2].StorageBytes()+m.powers[3].StorageBytes() {
		t.Fatal("immediate exact cold/raw graph", b.Used())
	}
	serviceMust(t, r.SetMaxSize(0))
	serviceMust(t, r.Deallocate(&raw))
	serviceMust(t, r.Deallocate(&padding))
	serviceMust(t, m.Close())
	serviceMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("exact acquired/caller member ends", b.Used())
	}
}
