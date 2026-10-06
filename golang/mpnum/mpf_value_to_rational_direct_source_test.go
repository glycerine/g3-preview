// Source: src/util/mpf.cpp:1695-1711; src/util/mpz.cpp:57-73.
// Licensed under the MIT license. Source authors: Christoph Wintersteiger; Leonardo de Moura.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestArbitraryFloatToRationalDirectOutputAfterSourceEnd(t *testing.T) {
	for ordinal := int64(0); ordinal < 80; ordinal++ {
		b := arena.NewBudget(1 << 28)
		cfg := IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}}
		m, e := NewArbitraryFloatManager(cfg)
		serviceMust(t, e)
		qm, e := NewRationalManager(RationalConfig{Integers: cfg})
		serviceMust(t, e)
		var x ArbitraryFloat
		one := NewInt(1)
		serviceMust(t, m.SetComponents(&x, 8, 4000, false, 0, &one))
		warm := NewRational()
		serviceMust(t, m.ToRational(&x, m.rational, &warm))
		serviceMust(t, m.CleanupRationalConversion())
		serviceMust(t, m.rational.Del(&warm))
		serviceMust(t, qm.z.ensureTmp(0, 256))
		serviceMust(t, qm.z.ensureTmp(1, 256))
		out := NewRational()
		b.FailAfter(ordinal)
		body := m.ToRational(&x, qm, &out)
		s := &m.rationalConversion
		integerPhysicalAudit(t, m.integer, &x.significand, &s.a.significand, &s.n, &s.d)
		integerPhysicalAudit(t, qm.z, &out.num, &out.den, &qm.nTmp)
		var maps uint64
		for i := range m.powers {
			maps += m.powers[i].StorageBytes()
		}
		want := m.header + m.RationalConversionHistoryBytes() + m.ValueCopyHistoryBytes() + m.powerWrapperBytes + m.rational.headerBytes + maps + integerActualOwnedBytes(m.integer) + qm.headerBytes + integerActualOwnedBytes(qm.z)
		if b.Used() != want {
			t.Fatal("immediate direct conversion graph", ordinal, b.Used(), want)
		}
		matched := body != nil && s.outputAllocation.numeric.cell.object != (memory.ObjectRef{}) && s.outputAllocation.numeric.cell.offset == 0
		b.FailAfter(-1)
		if matched {
			original := s.outputAllocation.originalCell
			serviceMust(t, qm.SourceFinalize())
			if s.outputAllocation.numeric.cell != original || s.outputAllocation.cellRetired || original != (s.outputAllocation.originalCell) {
				t.Fatal("direct raw must survive allocator source directory end")
			}
			serviceMust(t, m.CleanupRationalConversion())
			if !s.outputAllocation.cellRetired || s.outputAllocation.numeric.cell != (sourceCellBacking{}) || s.outputAllocation.originalCell != original {
				t.Fatal("own direct release lost original token")
			}
		} else {
			serviceMust(t, m.CleanupRationalConversion())
			serviceMust(t, qm.Del(&out))
		}
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
	t.Fatal("no reached direct output descriptor failure")
}
