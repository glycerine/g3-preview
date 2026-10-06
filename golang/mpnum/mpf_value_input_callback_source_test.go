// Source: src/util/mpf.cpp:114-148; src/util/mpz.h:156-164.
// Licensed under the MIT license. Source authors: Christoph Wintersteiger; Leonardo de Moura.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"math"
	"testing"
)

func TestArbitraryFloatInputActualAllocationTransportOwnerGuards(t *testing.T) {
	b := arena.NewBudget(1 << 27)
	var m *ArbitraryFloatManager
	var x ArbitraryFloat
	var receipt ArbitraryFloatValueInputReceipt
	armed := false
	calls := 0
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f memory.Failure) error {
		if armed {
			calls++
			used := b.Used()
			before := x
			own, source := m.ownCause, m.sourceCause
			for _, e := range []error{m.Close(), m.SourceFinalize(), m.integer.CloseChecked(), m.integer.SourceFinalize(), m.rational.CloseChecked(), m.rational.SourceFinalize(), m.CleanupValueInput(&receipt, true), m.SetDouble(&x, 11, 53, 1)} {
				if e != ErrPolynomialPending {
					t.Fatal("rejected actual input callback", e)
				}
			}
			if x != before || b.Used() != used || m.ownCause != own || m.sourceCause != source {
				t.Fatal("callback mutated actual caller/older owner")
			}
		}
		return f
	}})
	serviceMust(t, e)
	m, e = NewArbitraryFloatManager(IntegerConfig{Memory: r, Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	pad, e := r.Allocate(uint64(99990 - r.State().LocalSize - 8))
	serviceMust(t, e)
	serviceMust(t, r.SetMaxSize(1))
	armed = true
	body := m.SetDoubleWithReceipt(&x, 11, 4000, math.Float64frombits(0xbff123456789abcd), &receipt)
	armed = false
	if body == nil || calls != 1 {
		t.Fatal("no real allocating input callback", body, calls)
	}
	q, e := m.valueInputCall(&receipt)
	serviceMust(t, e)
	if q.bodyCause != body {
		t.Fatal("body cause not retained")
	}
	integerPhysicalAuditWithRaw(t, m.integer, []memory.Block{pad}, &x.significand)
	if b.Used() != r.Stats().TotalBytes+m.OwnedStorageBytes() {
		t.Fatal("before lift exact cold graph")
	}
	serviceMust(t, r.SetMaxSize(0))
	serviceMust(t, m.CleanupValueInput(&receipt, false))
	serviceMust(t, m.Del(&x))
	serviceMust(t, m.CleanupValueInput(&receipt, true))
	serviceMust(t, m.SetDouble(&x, 11, 4000, 2.5))
	if q.bodyCause != body {
		t.Fatal("later body erased original cause")
	}
	serviceMust(t, m.Del(&x))
	if e := m.SourceFinalize(); e != nil && !m.integer.sourceGuard {
		t.Fatal(e)
	}
	serviceMust(t, m.Close())
	serviceMust(t, r.Deallocate(&pad))
	serviceMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
