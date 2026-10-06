// Source: src/util/mpf.cpp:1695-1711,1922-1945.
// Licensed under the MIT license. Source author: Christoph Wintersteiger.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestArbitraryFloatToRationalReachedUnpackOwnsDescriptorPrefix(t *testing.T) {
	for ordinal := int64(0); ordinal < 32; ordinal++ {
		b := arena.NewBudget(1 << 28)
		config := IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}}
		m, e := NewArbitraryFloatManager(config)
		serviceMust(t, e)
		qm, e := NewRationalManager(RationalConfig{Integers: config})
		serviceMust(t, e)
		var a ArbitraryFloat
		n := NewInt(1)
		serviceMust(t, m.SetComponents(&a, 8, 4000, false, 0, &n))
		// This is the real lazy source member, not a completed conversion warmup.
		_, e = m.powerTwoWrapper(a.SBits()-1, false)
		serviceMust(t, e)
		out := NewRational()
		b.FailAfter(ordinal)
		body := m.ToRational(&a, qm, &out)
		s := &m.rationalConversion
		integerPhysicalAuditWithRaw(t, m.integer, nil, &a.significand, &s.a.significand, &s.n, &s.d, &m.rational.nTmp)
		integerPhysicalAuditWithRaw(t, qm.z, nil, &out.num, &out.den, &qm.nTmp)
		maps := uint64(0)
		for i := range m.powers {
			maps += m.powers[i].StorageBytes()
		}
		want := m.header + m.RationalConversionHistoryBytes() + m.ValueCopyHistoryBytes() + m.powerWrapperBytes + m.rational.headerBytes + maps + integerActualOwnedBytes(m.integer) + qm.headerBytes + integerActualOwnedBytes(qm.z)
		if b.Used() != want {
			t.Fatal("immediate actual conversion/source/output cold graph", ordinal, b.Used(), want)
		}
		matched := body != nil && s.numericPhase == 2 && (m.integer.pendingCell.object != (memory.ObjectRef{}) || s.numeric.cell.object != (memory.ObjectRef{}))
		if matched {
			if s.numeric.cell.object == (memory.ObjectRef{}) || s.originalCell != s.numeric.cell || m.integer.pendingCell.object != (memory.ObjectRef{}) {
				t.Fatalf("actual unpack descriptor prefix=%d missing exact conversion authority: body=%v", ordinal, body)
			}
		}
		b.FailAfter(-1)
		serviceMust(t, m.CleanupRationalConversion())
		serviceMust(t, qm.Del(&out))
		serviceMust(t, qm.CloseChecked())
		serviceMust(t, m.Del(&a))
		if source := m.SourceFinalize(); source != nil {
			if !errors.Is(source, arena.ErrInvalidSlot) || !m.integer.sourceGuard || m.integer.sourceFinalizeStage < 2 || m.integer.sourceFinalizeStage > 5 {
				t.Fatal("literal source destructor guard", ordinal, source, m.integer.sourceFinalizeStage)
			}
			index := int(m.integer.sourceFinalizeStage-2) / 2
			ref := m.integer.tmp[index]
			if m.integer.sourceFinalizeStage%2 == 1 {
				ref = m.integer.arg[index]
			}
			if ref != 0 {
				t.Fatal("unproved source scratch hole", ordinal, ref)
			}
		}
		serviceMust(t, m.Close())
		if b.Used() != 0 {
			t.Fatal(ordinal, b.Used())
		}
		if matched {
			t.Log("actual descriptor captured and fully owned", ordinal)
			return
		}
	}
	t.Fatal("actual reached unpack descriptor refusal not found")
}
