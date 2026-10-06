// Source: src/util/mpf.cpp:1695-1711; src/util/mpz.cpp:57-73.
// Licensed under the MIT license. Source authors: Christoph Wintersteiger; Leonardo de Moura.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestArbitraryFloatToRationalPooledSourceConsumption(t *testing.T) {
	for _, output := range []bool{true, false} {
		reached := false
		for ordinal := int64(0); ordinal < 80; ordinal++ {
			b := arena.NewBudget(1 << 27)
			cfg := IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: memory.SmallAllocatorProfile{PointerBits: 64}}
			m, e := NewArbitraryFloatManager(cfg)
			serviceMust(t, e)
			qm, e := NewRationalManager(RationalConfig{Integers: cfg})
			serviceMust(t, e)
			var x ArbitraryFloat
			one := NewInt(1)
			serviceMust(t, m.SetComponents(&x, 8, 260, false, 0, &one))
			_, e = m.powerTwoWrapper(259, false)
			serviceMust(t, e)
			for _, z := range []*IntegerManager{m.integer, qm.z} {
				serviceMust(t, z.ensureTmp(0, 64))
				serviceMust(t, z.ensureTmp(1, 64))
			}
			out := NewRational()
			serviceMust(t, m.ToRational(&x, m.rational, &out))
			serviceMust(t, m.CleanupRationalConversion())
			serviceMust(t, m.rational.Del(&out))
			out = NewRational()
			var sourceHolders, outputHolders [16]Int
			for i := range sourceHolders {
				serviceMust(t, m.integer.SetLimbs(&sourceHolders[i], []uint32{0xffffffff, 0xffffffff, 1}))
				serviceMust(t, qm.z.SetLimbs(&outputHolders[i], []uint32{0xffffffff, 0xffffffff, 1}))
			}
			b.FailAfter(ordinal)
			body := m.ToRational(&x, qm, &out)
			s := &m.rationalConversion
			cell, roots, retired, z := &s.originalCell, &s.numeric, &s.cellRetired, m.integer
			if output {
				cell, roots, retired, z = &s.outputAllocation.originalCell, &s.outputAllocation.numeric, &s.outputAllocation.cellRetired, qm.z
			}
			matched := body != nil && roots.cell.object != (memory.ObjectRef{}) && roots.cell.offset != 0
			sourceNamed := []*Int{&x.significand, &s.a.significand, &s.n, &s.d}
			outputNamed := []*Int{&out.num, &out.den, &qm.nTmp}
			for i := range sourceHolders {
				sourceNamed = append(sourceNamed, &sourceHolders[i])
				outputNamed = append(outputNamed, &outputHolders[i])
			}
			integerPhysicalAudit(t, m.integer, sourceNamed...)
			integerPhysicalAudit(t, qm.z, outputNamed...)
			if matched {
				if _, e := z.allocator.ObjectExtent(roots.cell.object); e != nil {
					t.Fatal(e)
				}
			}
			b.FailAfter(-1)
			for i := range sourceHolders {
				serviceMust(t, m.integer.Del(&sourceHolders[i]))
				serviceMust(t, qm.z.Del(&outputHolders[i]))
			}
			if matched {
				original := *cell
				if output {
					serviceMust(t, qm.SourceFinalize())
				} else {
					if e := m.SourceFinalize(); e != nil {
						if !m.integer.sourceGuard || m.integer.sourceFinalizeStage < 2 || m.integer.sourceFinalizeStage > 5 {
							t.Fatal(e)
						}
						if roots.cell != (original) || *retired {
							t.Fatal("earlier source guard changed unconsumed pool root")
						}
						if _, e := z.allocator.ObjectExtent(original.object); e != nil {
							t.Fatal(e)
						}
						reached = true
					}
				}
				if *cell != original {
					t.Fatal("source consumed issued history")
				}
				if (output || m.sourceEnded) && (roots.cell != (sourceCellBacking{}) || !*retired) {
					t.Fatal("actual pooled owner release retained live root authority", output, ordinal, roots.cell, *retired)
				}
				reached = true
			}
			if e := m.CleanupRationalConversion(); e != nil {
				t.Fatal("own conversion mode/stage", output, ordinal, s.sourceCursor, s.ownCursor, s.numeric, s.transferred.Block(), s.outputAllocation.numeric, s.outputAllocation.transferred.Block(), s.outputGCD, s.copy, e)
			}
			if !output || !matched {
				serviceMust(t, qm.Del(&out))
			}
			serviceMust(t, qm.CloseChecked())
			if !matched || output {
				serviceMust(t, m.Del(&x))
			}
			if !m.sourceEnded {
				if e := m.SourceFinalize(); e != nil && !m.integer.sourceGuard {
					t.Fatal(e)
				}
			}
			serviceMust(t, m.Close())
			if b.Used() != 0 {
				t.Fatal(output, ordinal, b.Used())
			}
			if reached {
				break
			}
		}
		if !reached {
			t.Fatal("no pooled descriptor failure", output)
		}
	}
}
