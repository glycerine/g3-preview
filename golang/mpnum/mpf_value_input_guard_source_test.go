// Source: src/util/mpf.cpp:69-181; explicit Go borrowed-context guard domain.
// Licensed under the MIT license. Source authors: Christoph Wintersteiger; Leonardo de Moura.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestArbitraryFloatInputCapturedContextExactFrameAndNonce(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	m, e := NewArbitraryFloatManager(IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	old, e := m.beginValueInput(nil)
	serviceMust(t, e)
	m.valueExecuting = true
	serviceMust(t, old.prepareInputAllocation(m.integer))
	issued := m.inputBinding.record.issued
	m.valueExecuting = false
	m.finishValueInput(nil)
	current, e := m.beginValueInput(nil)
	serviceMust(t, e)
	m.valueExecuting = true
	copied := m.inputBinding
	invalid := current
	invalid.mpfInput = &copied
	used := b.Used()
	head := m.inputCalls
	for _, c := range []integerAllocationInvocationContext{old, invalid} {
		if c.inputRoots(m.integer) != nil || c.prepareInputAllocation(m.integer) != memory.ErrRuntimeOwner {
			t.Fatal("saved/copy context accepted")
		}
		if b.Used() != used || m.inputCalls != head {
			t.Fatal("guard mutated actual owning list")
		}
	}
	serviceMust(t, current.prepareInputAllocation(m.integer))
	if current.inputRoots(m.integer) == nil {
		t.Fatal("actual current context rejected")
	}
	m.valueExecuting = false
	m.finishValueInput(nil)
	frameCopy := *issued.frame
	copiedReceipt := issued
	copiedReceipt.frame = &frameCopy
	used = b.Used()
	if m.CleanupValueInput(&copiedReceipt, true) != memory.ErrRuntimeOwner || b.Used() != used {
		t.Fatal("copied owning descriptor mutated actual frame")
	}
	serviceMust(t, m.CleanupValueInput(&issued, true))
	serviceMust(t, m.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
