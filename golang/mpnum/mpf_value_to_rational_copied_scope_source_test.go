// Source: src/util/mpf.cpp:1695-1711; src/util/mpz.cpp:730-817.
// Licensed under the MIT license. Source authors: Christoph Wintersteiger; Leonardo de Moura.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

// Invalid copied Go owner metadata is distinct from a numerical source prefix.
func TestArbitraryFloatToRationalCopiedParentCannotAbortChild(t *testing.T) {
	for ordinal := int64(0); ordinal < 64; ordinal++ {
		b := arena.NewBudget(1 << 28)
		cfg := IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}}
		m, e := NewArbitraryFloatManager(cfg)
		serviceMust(t, e)
		qm, e := NewRationalManager(RationalConfig{Integers: cfg})
		serviceMust(t, e)
		var x ArbitraryFloat
		one := NewInt(1)
		serviceMust(t, m.SetComponents(&x, 8, 4000, false, 0, &one))
		_, e = m.powerTwoWrapper(3999, false)
		serviceMust(t, e)
		out := NewRational()
		b.FailAfter(ordinal)
		body := m.ToRational(&x, qm, &out)
		s := &m.rationalConversion
		reached := body != nil && qm.z.matchesRawInvocation(&s.outputGCD)
		if reached {
			copied := *s
			child := *s.outputGCD.frame
			used := b.Used()
			state := qm.z.memory.State()
			if e := m.cleanupRationalConversionOutput(&copied, true); e != memory.ErrRuntimeOwner {
				t.Fatal("copied parent adopted exact child", ordinal, e)
			}
			if s.outputGCD.frame.values != child.values || s.outputGCD.frame.cursor != child.cursor || s.outputGCD.frame.active != child.active || b.Used() != used || qm.z.memory.State() != state {
				t.Fatal("copied parent mutated actual child")
			}
		}
		integerPhysicalAudit(t, m.integer, &x.significand, &s.a.significand, &s.n, &s.d)
		integerPhysicalAudit(t, qm.z, &out.num, &out.den, &qm.nTmp)
		b.FailAfter(-1)
		serviceMust(t, m.CleanupRationalConversion())
		serviceMust(t, qm.Del(&out))
		serviceMust(t, qm.CloseChecked())
		serviceMust(t, m.Del(&x))
		if e := m.SourceFinalize(); e != nil && !m.integer.sourceGuard {
			t.Fatal(e)
		}
		serviceMust(t, m.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
		if reached {
			return
		}
	}
	t.Fatal("no actual normalization GCD child")
}
