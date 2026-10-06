// Source: src/util/mpf.cpp:114-148; src/util/mpz.h:156-164.
// Licensed under the MIT license. Source authors: Christoph Wintersteiger; Leonardo de Moura.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"math"
	"testing"
)

func TestArbitraryFloatInputReachedSETAndShiftAllocationReceipt(t *testing.T) {
	captured := false
	completed := false
	for ordinal := int64(0); ordinal < 60; ordinal++ {
		b := arena.NewBudget(1 << 28)
		m, e := NewArbitraryFloatManager(IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
		serviceMust(t, e)
		var x ArbitraryFloat
		var r ArbitraryFloatValueInputReceipt
		b.FailAfter(ordinal)
		body := m.SetDoubleWithReceipt(&x, 11, 4000, math.Float64frombits(0xbff123456789abcd), &r)
		if body != nil && m.integer.pendingCell.object != (memory.ObjectRef{}) {
			t.Fatal("actual supplied input allocation failed without exact child transfer", ordinal, body)
		}
		integerPhysicalAudit(t, m.integer, &x.significand)
		if b.Used() != m.integer.memory.Stats().TotalBytes+m.OwnedStorageBytes() {
			t.Fatal("before-lift input cold graph", ordinal, b.Used())
		}
		if r != (ArbitraryFloatValueInputReceipt{}) {
			q, e := m.valueInputCall(&r)
			serviceMust(t, e)
			if q.bodyCause != body {
				t.Fatal("original body cause lost")
			}
			if body != nil && (q.numeric.cell.object != (memory.ObjectRef{}) || q.transferred.Block() != (memory.Block{})) {
				captured = true
			}
		}
		b.FailAfter(-1)
		serviceMust(t, m.CleanupValueInput(&r, false))
		serviceMust(t, m.Del(&x))
		serviceMust(t, m.CleanupValueInput(&r, true))
		if e := m.SourceFinalize(); e != nil && !m.integer.sourceGuard {
			t.Fatal(e)
		}
		serviceMust(t, m.Close())
		if b.Used() != 0 {
			t.Fatal(ordinal, b.Used())
		}
		if body == nil {
			completed = true
			break
		}
	}
	if !captured || !completed {
		t.Fatal("missing actual allocating input refusal/success", captured, completed)
	}
}
