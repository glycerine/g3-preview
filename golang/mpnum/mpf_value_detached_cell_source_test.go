// Source: src/util/mpz.h:156-164; src/util/small_object_allocator.cpp:allocate/deallocate.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestArbitraryFloatPooledDescriptorAfterSourceEnd(t *testing.T) {
	for ordinal := int64(0); ordinal < 16; ordinal++ {
		b := arena.NewBudget(1 << 26)
		m, e := NewArbitraryFloatManager(IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: memory.SmallAllocatorProfile{PointerBits: 64}})
		serviceMust(t, e)
		var source Int
		serviceMust(t, m.integer.SetLimbs(&source, []uint32{0xffffffff, 0xffffffff, 0x7fffffff}))
		var a ArbitraryFloat
		var r ArbitraryFloatValueCopyReceipt
		b.FailAfter(ordinal)
		body := m.SetComponents(&a, 8, 24, false, 0, &source)
		if m.valueCalls != nil {
			r = m.valueCalls.issued
		}
		if body != nil && r.frame != nil && r.child.frame.roots.cell.object != (memory.ObjectRef{}) && r.child.frame.roots.cell.offset != 0 {
			t.Log("actual pooled descriptor refusal", ordinal, body, r.child.frame.roots.cell)
			integerPhysicalAuditWithRaw(t, m.integer, nil, &source, &a.significand)
			var maps uint64
			for i := range m.powers {
				maps += m.powers[i].StorageBytes()
			}
			if want := m.header + m.ValueCopyHistoryBytes() + m.rational.headerBytes + integerActualOwnedBytes(m.integer) + maps; b.Used() != want {
				t.Fatal("actual cold and physical graph before fault lift", b.Used(), want)
			}
			b.FailAfter(-1)
			serviceMust(t, m.Del(&a))
			serviceMust(t, m.integer.Del(&source))
			original := r.child.frame.originalCell
			serviceMust(t, m.SourceFinalize())
			integerPhysicalAuditWithRaw(t, m.integer, nil)
			if r.child.frame.originalCell != original {
				t.Fatal("source destruction erased actual acquisition history")
			}
			e = m.CleanupValueCopy(&r, true)
			if e != nil {
				t.Fatalf("exact pooled child own cleanup after literal source DT: %v", e)
			}
			if r.child.frame.originalCell != original || r.frame.bodyCause != body || !r.child.frame.cellRetired {
				t.Fatal("own ACK erased immutable acquisition/body history")
			}
			serviceMust(t, m.Close())
			if b.Used() != 0 {
				t.Fatal(b.Used())
			}
			return
		}
		b.FailAfter(-1)
		serviceMust(t, m.Del(&a))
		serviceMust(t, m.integer.Del(&source))
		serviceMust(t, m.Close())
	}
	t.Fatal("pooled descriptor refusal not reached")
}

func TestArbitraryFloatDirectDescriptorAfterSourceEnd(t *testing.T) {
	for ordinal := int64(0); ordinal < 16; ordinal++ {
		b := arena.NewBudget(1 << 26)
		m, e := NewArbitraryFloatManager(IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
		serviceMust(t, e)
		var source Int
		serviceMust(t, m.integer.SetLimbs(&source, []uint32{0xffffffff, 0xffffffff, 0x7fffffff}))
		var a ArbitraryFloat
		var r ArbitraryFloatValueCopyReceipt
		b.FailAfter(ordinal)
		body := m.SetComponents(&a, 8, 24, false, 0, &source)
		if m.valueCalls != nil {
			r = m.valueCalls.issued
		}
		if body != nil && r.frame != nil && r.child.frame.roots.cell.object != (memory.ObjectRef{}) && r.child.frame.roots.cell.offset == 0 {
			t.Log("actual direct descriptor refusal", ordinal, body, r.child.frame.roots.cell)
			integerPhysicalAuditWithRaw(t, m.integer, nil, &source, &a.significand)
			var maps uint64
			for i := range m.powers {
				maps += m.powers[i].StorageBytes()
			}
			if want := m.header + m.ValueCopyHistoryBytes() + m.rational.headerBytes + integerActualOwnedBytes(m.integer) + maps; b.Used() != want {
				t.Fatal("actual cold and physical graph before fault lift", b.Used(), want)
			}
			b.FailAfter(-1)
			serviceMust(t, m.Del(&a))
			serviceMust(t, m.integer.Del(&source))
			original := r.child.frame.originalCell
			serviceMust(t, m.SourceFinalize())
			integerPhysicalAuditWithRaw(t, m.integer, nil)
			if r.child.frame.originalCell != original {
				t.Fatal("source destruction erased actual acquisition history")
			}
			e = m.CleanupValueCopy(&r, true)
			if e != nil {
				t.Fatalf("exact direct child own cleanup after literal source DT: %v", e)
			}
			if r.child.frame.originalCell != original || r.frame.bodyCause != body || !r.child.frame.cellRetired {
				t.Fatal("own ACK erased immutable acquisition/body history")
			}
			serviceMust(t, m.Close())
			if b.Used() != 0 {
				t.Fatal(b.Used())
			}
			return
		}
		b.FailAfter(-1)
		serviceMust(t, m.Del(&a))
		serviceMust(t, m.integer.Del(&source))
		serviceMust(t, m.Close())
	}
	t.Fatal("direct descriptor refusal not reached")
}

func TestArbitraryFloatPooledDescriptorAfterPartialSourceEnd(t *testing.T) {
	for ordinal := int64(0); ordinal < 16; ordinal++ {
		b := arena.NewBudget(1 << 26)
		rt, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64})
		serviceMust(t, e)
		m, e := NewArbitraryFloatManager(IntegerConfig{Memory: rt, Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: memory.SmallAllocatorProfile{PointerBits: 64}})
		serviceMust(t, e)
		var source Int
		serviceMust(t, m.integer.SetLimbs(&source, []uint32{0xffffffff, 0xffffffff, 0x7fffffff}))
		var a ArbitraryFloat
		var r ArbitraryFloatValueCopyReceipt
		b.FailAfter(ordinal)
		body := m.SetComponents(&a, 8, 24, false, 0, &source)
		if m.valueCalls != nil {
			r = m.valueCalls.issued
		}
		if body != nil && r.frame != nil && r.child.frame.roots.cell.object != (memory.ObjectRef{}) && r.child.frame.roots.cell.offset != 0 {
			t.Log("actual pooled descriptor refusal", ordinal, body, r.child.frame.roots.cell)
			integerPhysicalAuditWithRaw(t, m.integer, nil, &source, &a.significand)
			var maps uint64
			for i := range m.powers {
				maps += m.powers[i].StorageBytes()
			}
			if want := m.header + m.ValueCopyHistoryBytes() + m.rational.headerBytes + integerActualOwnedBytes(m.integer) + maps; b.Used() != want {
				t.Fatal("actual cold and physical graph before fault lift", b.Used(), want)
			}
			b.FailAfter(-1)
			serviceMust(t, m.Del(&a))
			serviceMust(t, m.integer.Del(&source))
			original := r.child.frame.originalCell
			contextTLSNearMinimum(t, rt)
			lane, e := rt.Allocate(8)
			serviceMust(t, e)
			delta := int64(-1000000) - rt.State().LocalSize
			request := uint64(16) + uint64(delta) - 8
			if e = rt.Reallocate(&lane, request); !errors.Is(e, arena.ErrBounds) {
				t.Fatal(e)
			}
			stop := m.SourceFinalize()
			if !errors.Is(stop, arena.ErrBounds) || m.integer.sourceFinalized || !r.child.frame.cellRetired || r.child.frame.roots.cell.object != (memory.ObjectRef{}) {
				t.Fatal("actual partial source chunk release", stop, m.integer.sourceFinalizeStage, r.child.frame)
			}
			integerPhysicalAuditWithRaw(t, m.integer, []memory.Block{lane})
			recovery, e := rt.Allocate(2000000)
			serviceMust(t, e)
			if r.child.frame.originalCell != original {
				t.Fatal("source destruction erased actual acquisition history")
			}
			e = m.CleanupValueCopy(&r, true)
			if e != nil {
				t.Fatalf("exact pooled child own cleanup after literal source DT: %v", e)
			}
			if r.child.frame.originalCell != original || r.frame.bodyCause != body || m.sourceCause != stop {
				t.Fatal("own ACK erased acquisition/body/source history")
			}
			contextTLSFinish(t, rt, m.Close)
			contextTLSFinish(t, rt, func() error { return rt.Deallocate(&lane) })
			if e = rt.Deallocate(&recovery); e != nil && (!errors.Is(e, arena.ErrBounds) || recovery != (memory.Block{})) {
				t.Fatal(e)
			}
			contextTLSFinish(t, rt, rt.Close)
			if b.Used() != 0 {
				t.Fatal(b.Used())
			}
			return
		}
		b.FailAfter(-1)
		serviceMust(t, m.Del(&a))
		serviceMust(t, m.integer.Del(&source))
		serviceMust(t, m.Close())
	}
	t.Fatal("pooled descriptor refusal not reached")
}
