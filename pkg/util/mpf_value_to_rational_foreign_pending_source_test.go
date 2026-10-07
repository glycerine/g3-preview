// Source: src/util/mpf.cpp:1695-1711,1922-1945; src/util/mpz.h:133-146.
package util

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestArbitraryFloatToRationalOwnACKPreservesLaterForeignDescriptor(t *testing.T) {
	for ordinal := int64(0); ordinal < 60; ordinal++ {
		b := arena.NewBudget(1 << 28)
		cfg := IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}}
		m, e := NewArbitraryFloatManager(cfg)
		serviceMust(t, e)
		qm, e := NewRationalManager(RationalConfig{Integers: cfg})
		serviceMust(t, e)
		var x ArbitraryFloat
		one := NewInt(1)
		serviceMust(t, m.SetComponents(&x, 8, 4000, false, 0, &one))
		_, e = m.powerTwoWrapper(3999, false)
		serviceMust(t, e)
		var foreignSource Int
		limbs := make([]uint32, 200)
		limbs[199] = 1
		serviceMust(t, m.integer.SetLimbs(&foreignSource, limbs))
		out := NewRational()
		b.FailAfter(ordinal)
		body := m.ToRational(&x, qm, &out)
		s := &m.rationalConversion
		matched := body != nil && s.numericPhase == 2 && s.numeric.Cell.Object != (ObjectRef{})
		b.FailAfter(-1)
		var foreign Int
		foreignPending := false
		var holders [16]Int
		if matched {
			for i := range holders {
				serviceMust(t, m.integer.Copy(&holders[i], &foreignSource))
			}
			for f := int64(0); f < 12; f++ {
				b.FailAfter(f)
				e = m.integer.Copy(&foreign, &foreignSource)
				if e != nil && m.integer.PendingCell.Object != (ObjectRef{}) {
					foreignPending = true
					break
				}
				b.FailAfter(-1)
				serviceMust(t, m.integer.Del(&foreign))
				serviceMust(t, m.integer.cleanupPendingAllocation())
			}
			if !foreignPending {
				t.Fatal("no reached later descriptor refusal")
			}
			captured := m.integer.PendingCell
			named := []*Int{&x.significand, &foreignSource, &foreign, &s.a.significand, &s.n, &s.d}
			for i := range holders {
				named = append(named, &holders[i])
			}
			integerPhysicalAudit(t, m.integer, named...)
			integerPhysicalAudit(t, qm.Z, &out.Num, &out.Den, &qm.NTmp)
			var maps uint64
			for i := range m.powers {
				maps += m.powers[i].StorageBytes()
			}
			want := m.header + m.ValueCopyHistoryBytes() + m.RationalConversionHistoryBytes() + m.powerWrapperBytes + m.rational.HeaderBytes + maps + integerActualOwnedBytes(m.integer) + qm.HeaderBytes + integerActualOwnedBytes(qm.Z)
			if b.Used() != want {
				t.Fatal("before-lift later foreign graph", b.Used(), want)
			}
			b.FailAfter(-1)

			serviceMust(t, m.CleanupRationalConversion())
			if m.integer.PendingCell != captured {
				t.Fatal("old exact ACK drained later foreign descriptor")
			}
			if _, e = m.integer.Allocator.ObjectExtent(captured.Object); e != nil {
				t.Fatal("foreign backing physically consumed", e)
			}
			serviceMust(t, m.integer.cleanupPendingAllocation())
		} else {
			serviceMust(t, m.CleanupRationalConversion())
		}
		for i := range holders {
			serviceMust(t, m.integer.Del(&holders[i]))
		}
		serviceMust(t, m.integer.Del(&foreign))
		serviceMust(t, m.integer.Del(&foreignSource))
		serviceMust(t, qm.Del(&out))
		serviceMust(t, qm.CloseChecked())
		serviceMust(t, m.Del(&x))
		if e := m.SourceFinalize(); e != nil && !m.integer.sourceGuard {
			t.Fatal(e)
		}
		serviceMust(t, m.Close())
		if b.Used() != 0 {
			t.Fatal(ordinal, b.Used())
		}
		if matched {
			return
		}
	}
	t.Fatal("no actual parent descriptor refusal")
}
