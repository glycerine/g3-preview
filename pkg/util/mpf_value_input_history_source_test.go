// Source: src/util/mpf.cpp:69-106,114-181; src/util/mpz.h:156-164.
package util

import (
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestArbitraryFloatInputMultiplePendingAndInlineHistory(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	m, e := NewArbitraryFloatManager(IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	var values [2]ArbitraryFloat
	var receipts [2]ArbitraryFloatValueInputReceipt
	var causes [2]error
	for i := range values {
		found := false
		for ordinal := int64(0); ordinal < 40; ordinal++ {
			b.FailAfter(ordinal)
			var r ArbitraryFloatValueInputReceipt
			body := m.SetDoubleWithReceipt(&values[i], 11, 4000, math.Float64frombits(0xbff123456789abcd), &r)
			integerPhysicalAudit(t, m.integer, &values[0].significand, &values[1].significand)
			if b.Used() != m.integer.Memory.Stats().TotalBytes+m.OwnedStorageBytes() {
				t.Fatal("before lift named cold graph", i, ordinal)
			}
			if r != (ArbitraryFloatValueInputReceipt{}) {
				q, e := m.valueInputCall(&r)
				serviceMust(t, e)
				if body != nil && (q.Numeric.Cell.Object != (ObjectRef{}) || q.Transferred.Block() != (Block{})) {
					receipts[i], causes[i], found = r, body, true
					b.FailAfter(-1)
					break
				}
			}
			b.FailAfter(-1)
			serviceMust(t, m.CleanupValueInput(&r, false))
			serviceMust(t, m.Del(&values[i]))
			serviceMust(t, m.CleanupValueInput(&r, true))
		}
		if !found {
			t.Fatal("no actual retained allocating prefix", i)
		}
	}
	if receipts[0].Frame == receipts[1].Frame {
		t.Fatal("later allocation reused owning frame")
	}
	// Repeated inline SET has no reservation/archive and does not overwrite the
	// optional receipt or either older failed allocation tuple.
	var inline ArbitraryFloat
	old := receipts[0]
	b.FailAfter(0)
	for i := 0; i < 3; i++ {
		serviceMust(t, m.SetFloatWithReceipt(&inline, 8, 24, 1, &old))
	}
	if old != receipts[0] {
		t.Fatal("inline receipt overwritten")
	}
	b.FailAfter(-1)
	for i := range receipts {
		q, e := m.valueInputCall(&receipts[i])
		serviceMust(t, e)
		serviceMust(t, m.CleanupValueInput(&receipts[i], false))
		serviceMust(t, m.Del(&values[i]))
		serviceMust(t, m.CleanupValueInput(&receipts[i], true))
		if q.bodyCause != causes[i] || q.Issued != receipts[i] {
			t.Fatal("retirement erased issued cause")
		}
	}
	serviceMust(t, m.Del(&inline))
	if e := m.SourceFinalize(); e != nil && !m.integer.sourceGuard {
		t.Fatal(e)
	}
	serviceMust(t, m.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
