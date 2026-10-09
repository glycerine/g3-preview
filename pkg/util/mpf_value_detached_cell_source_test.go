// Source: src/util/mpz.h:156-164; src/util/small_object_allocator.cpp:allocate/deallocate.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestArbitraryFloatPooledDescriptorAfterSourceEnd(t *testing.T) {
	for ordinal := int64(0); ordinal < 16; ordinal++ {
		b := arena.NewBudget(1 << 26)
		m, e := NewArbitraryFloatManager(IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: SmallAllocatorProfile{PointerBits: 64}})
		serviceMust(t, e)
		var source Int
		serviceMust(t, m.integer.SetLimbs(&source, []uint32{0xffffffff, 0xffffffff, 0x7fffffff}))
		var a ArbitraryFloat
		var r ArbitraryFloatValueCopyReceipt
		b.FailAfter(ordinal)
		body := m.SetComponents(&a, 8, 24, false, 0, &source)
		if m.ValueCalls != nil {
			r = m.ValueCalls.Issued
		}
		if body != nil && r.frame != nil && r.Child.Frame.Roots.Cell.Object != (ObjectRef{}) && r.Child.Frame.Roots.Cell.Offset != 0 {
			t.Log("actual pooled descriptor refusal", ordinal, body, r.Child.Frame.Roots.Cell)
			integerPhysicalAuditWithRaw(t, m.integer, nil, &source, &a.significand)
			var maps uint64
			if want := m.header + m.ValueCopyHistoryBytes() + m.rational.HeaderBytes + integerActualOwnedBytes(m.integer) + maps; b.Used() != want {
				t.Fatal("actual cold and physical graph before fault lift", b.Used(), want)
			}
			b.FailAfter(-1)
			serviceMust(t, m.Del(&a))
			serviceMust(t, m.integer.Del(&source))
			original := r.Child.Frame.originalCell
			serviceMust(t, m.SourceFinalize())
			integerPhysicalAuditWithRaw(t, m.integer, nil)
			if r.Child.Frame.originalCell != original {
				t.Fatal("source destruction erased actual acquisition history")
			}
			e = m.CleanupValueCopy(&r, true)
			if e != nil {
				t.Fatalf("exact pooled child own cleanup after literal source DT: %v", e)
			}
			if r.Child.Frame.originalCell != original || r.frame.bodyCause != body || !r.Child.Frame.cellRetired {
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
		m, e := NewArbitraryFloatManager(IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}})
		serviceMust(t, e)
		var source Int
		serviceMust(t, m.integer.SetLimbs(&source, []uint32{0xffffffff, 0xffffffff, 0x7fffffff}))
		var a ArbitraryFloat
		var r ArbitraryFloatValueCopyReceipt
		b.FailAfter(ordinal)
		body := m.SetComponents(&a, 8, 24, false, 0, &source)
		if m.ValueCalls != nil {
			r = m.ValueCalls.Issued
		}
		if body != nil && r.frame != nil && r.Child.Frame.Roots.Cell.Object != (ObjectRef{}) && r.Child.Frame.Roots.Cell.Offset == 0 {
			t.Log("actual direct descriptor refusal", ordinal, body, r.Child.Frame.Roots.Cell)
			integerPhysicalAuditWithRaw(t, m.integer, nil, &source, &a.significand)
			var maps uint64
			if want := m.header + m.ValueCopyHistoryBytes() + m.rational.HeaderBytes + integerActualOwnedBytes(m.integer) + maps; b.Used() != want {
				t.Fatal("actual cold and physical graph before fault lift", b.Used(), want)
			}
			b.FailAfter(-1)
			serviceMust(t, m.Del(&a))
			serviceMust(t, m.integer.Del(&source))
			original := r.Child.Frame.originalCell
			serviceMust(t, m.SourceFinalize())
			integerPhysicalAuditWithRaw(t, m.integer, nil)
			if r.Child.Frame.originalCell != original {
				t.Fatal("source destruction erased actual acquisition history")
			}
			e = m.CleanupValueCopy(&r, true)
			if e != nil {
				t.Fatalf("exact direct child own cleanup after literal source DT: %v", e)
			}
			if r.Child.Frame.originalCell != original || r.frame.bodyCause != body || !r.Child.Frame.cellRetired {
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
		rt, e := NewRuntime(RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64})
		serviceMust(t, e)
		m, e := NewArbitraryFloatManager(IntegerConfig{Memory: rt, Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: SmallAllocatorProfile{PointerBits: 64}})
		serviceMust(t, e)
		var source Int
		serviceMust(t, m.integer.SetLimbs(&source, []uint32{0xffffffff, 0xffffffff, 0x7fffffff}))
		var a ArbitraryFloat
		var r ArbitraryFloatValueCopyReceipt
		b.FailAfter(ordinal)
		body := m.SetComponents(&a, 8, 24, false, 0, &source)
		if m.ValueCalls != nil {
			r = m.ValueCalls.Issued
		}
		if body != nil && r.frame != nil && r.Child.Frame.Roots.Cell.Object != (ObjectRef{}) && r.Child.Frame.Roots.Cell.Offset != 0 {
			t.Log("actual pooled descriptor refusal", ordinal, body, r.Child.Frame.Roots.Cell)
			integerPhysicalAuditWithRaw(t, m.integer, nil, &source, &a.significand)
			var maps uint64
			if want := m.header + m.ValueCopyHistoryBytes() + m.rational.HeaderBytes + integerActualOwnedBytes(m.integer) + maps; b.Used() != want {
				t.Fatal("actual cold and physical graph before fault lift", b.Used(), want)
			}
			b.FailAfter(-1)
			serviceMust(t, m.Del(&a))
			serviceMust(t, m.integer.Del(&source))
			original := r.Child.Frame.originalCell
			contextTLSNearMinimum(t, rt)
			lane, e := rt.Allocate(8)
			serviceMust(t, e)
			delta := int64(-1000000) - rt.State().LocalSize
			request := uint64(16) + uint64(delta) - 8
			if e = rt.Reallocate(&lane, request); !errors.Is(e, arena.ErrBounds) {
				t.Fatal(e)
			}
			stop := m.SourceFinalize()
			if !errors.Is(stop, arena.ErrBounds) || m.integer.SourceFinalized || !r.Child.Frame.cellRetired || r.Child.Frame.Roots.Cell.Object != (ObjectRef{}) {
				t.Fatal("actual partial source chunk release", stop, m.integer.SourceFinalizeStage, r.Child.Frame)
			}
			integerPhysicalAuditWithRaw(t, m.integer, []Block{lane})
			recovery, e := rt.Allocate(2000000)
			serviceMust(t, e)
			if r.Child.Frame.originalCell != original {
				t.Fatal("source destruction erased actual acquisition history")
			}
			e = m.CleanupValueCopy(&r, true)
			if e != nil {
				t.Fatalf("exact pooled child own cleanup after literal source DT: %v", e)
			}
			if r.Child.Frame.originalCell != original || r.frame.bodyCause != body || m.sourceCause != stop {
				t.Fatal("own ACK erased acquisition/body/source history")
			}
			contextTLSFinish(t, rt, m.Close)
			contextTLSFinish(t, rt, func() error { return rt.Deallocate(&lane) })
			if e = rt.Deallocate(&recovery); e != nil && (!errors.Is(e, arena.ErrBounds) || recovery != (Block{})) {
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
