// Source: src/util/mpf.cpp:114-148; src/util/mpz.h:156-164.
package util

import (
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestArbitraryFloatInputDescriptorSourceEndHistory(t *testing.T) {
	for _, direct := range []bool{false, true} {
		found := false
		for ordinal := int64(0); ordinal < 30; ordinal++ {
			b := arena.NewBudget(1 << 27)
			m, e := NewArbitraryFloatManager(IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: direct}})
			serviceMust(t, e)
			var x ArbitraryFloat
			var r ArbitraryFloatValueInputReceipt
			b.FailAfter(ordinal)
			body := m.SetDoubleWithReceipt(&x, 11, 53, math.Float64frombits(0xbff123456789abcd), &r)
			integerPhysicalAudit(t, m.integer, &x.significand)
			if b.Used() != m.integer.Memory.Stats().TotalBytes+m.OwnedStorageBytes() {
				t.Fatal("before lift physical and cold graph")
			}
			var q *arbitraryFloatValueInputCall
			if r.Frame != nil {
				q, e = m.valueInputCall(&r)
				serviceMust(t, e)
			}
			selected := body != nil && q != nil && q.Numeric.Cell.Object != (ObjectRef{}) && (q.Numeric.Cell.Offset == 0) == direct
			b.FailAfter(-1)
			serviceMust(t, m.CleanupValueInput(&r, false))
			serviceMust(t, m.Del(&x))
			if selected {
				original := q.originalCell
				serviceMust(t, m.SourceFinalize())
				integerPhysicalAudit(t, m.integer)
				if q.originalCell != original || q.bodyCause != body {
					t.Fatal("source end erased issued tuple")
				}
				serviceMust(t, m.CleanupValueInput(&r, true))
				if q.originalCell != original || q.bodyCause != body || !q.cellRetired {
					t.Fatal("own end erased history or replayed cell")
				}
				found = true
			} else {
				serviceMust(t, m.CleanupValueInput(&r, true))
			}
			serviceMust(t, m.Close())
			if b.Used() != 0 {
				t.Fatal(b.Used())
			}
			if found {
				break
			}
		}
		if !found {
			t.Fatal("actual descriptor/source-end prefix missing", direct)
		}
	}
}
