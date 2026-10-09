// Source: src/util/mpf.cpp:1695-1711,1922-1945.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestArbitraryFloatToRationalForeignSetterOwnsDescriptorPrefix(t *testing.T) {
	for ordinal := int64(0); ordinal < 160; ordinal++ {
		b := arena.NewBudget(1 << 28)
		config := IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}}
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
		integerPhysicalAuditWithRaw(t, m.integer, nil, &a.significand, &s.a.significand, &s.n, &s.d, &m.rational.NTmp)
		integerPhysicalAuditWithRaw(t, qm.Z, nil, &out.Num, &out.Den, &qm.NTmp)
		maps := uint64(0)
		want := m.header + m.RationalConversionHistoryBytes() + m.ValueCopyHistoryBytes() + m.powerWrapperBytes + m.rational.HeaderBytes + maps + integerActualOwnedBytes(m.integer) + qm.HeaderBytes + integerActualOwnedBytes(qm.Z)
		if b.Used() != want {
			t.Fatal("immediate actual conversion/source/output cold graph", ordinal, b.Used(), want)
		}
		matched := body != nil && s.numericPhase == 6 && (qm.Z.PendingCell.Object != (ObjectRef{}) || s.outputAllocation.numeric.Cell.Object != (ObjectRef{}))
		if matched {
			if s.outputAllocation.numeric.Cell.Object == (ObjectRef{}) || s.outputAllocation.originalCell != s.outputAllocation.numeric.Cell || qm.Z.PendingCell.Object != (ObjectRef{}) {
				t.Fatalf("foreign output descriptor at actual SET lacks exact conversion child authority: prefix=%d body=%v object=%v", ordinal, body, qm.Z.PendingCell.Object)
			}
		}

		b.FailAfter(-1)
		serviceMust(t, m.CleanupRationalConversion())
		serviceMust(t, qm.Del(&out))
		serviceMust(t, qm.CloseChecked())
		serviceMust(t, m.Del(&a))
		if source := m.SourceFinalize(); source != nil {
			if !errors.Is(source, arena.ErrInvalidSlot) || !m.integer.sourceGuard || m.integer.SourceFinalizeStage < 2 || m.integer.SourceFinalizeStage > 5 {
				t.Fatal("literal source destructor guard", ordinal, source, m.integer.SourceFinalizeStage)
			}
			index := int(m.integer.SourceFinalizeStage-2) / 2
			ref := m.integer.Tmp[index]
			if m.integer.SourceFinalizeStage%2 == 1 {
				ref = m.integer.Arg[index]
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
	t.Fatal("actual reached foreign setter descriptor refusal not found")
}
