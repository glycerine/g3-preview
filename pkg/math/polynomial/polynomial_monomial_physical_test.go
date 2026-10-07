// Source: src/math/polynomial/polynomial.cpp; src/math/polynomial/polynomial.h.
package polynomial

import (
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func monomialPhysicalAudit(t *testing.T, m *MonomialManager, externalRaw ...util.Block) {
	t.Helper()
	if m == nil {
		return
	}
	r := m.memory.Stats()
	var blocks uint32
	add := func(b util.Block) {
		if b != (util.Block{}) {
			blocks++
		}
	}
	for _, raw := range externalRaw {
		add(raw)
	}
	add(m.table.raw)
	add(m.table.pending)
	add(m.renameTable.raw)
	add(m.renameTable.pending)
	add(m.renameFound)
	for i := range m.temps {
		add(m.temps[i].raw)
		add(m.temps[i].pending)
	}
	add(m.powers.raw)
	add(m.powers.pending)
	if m.idsInitialized {
		for _, b := range m.ids.RawRoots() {
			add(b)
		}
	}
	bytes := m.header + uint64(cap(m.locations))*uint64(reflect.TypeOf(monomialLocation{}).Size()) + r.TotalBytes
	if m.allocator != nil {
		a := m.allocator.Stats()
		blocks += a.Chunks + a.DirectObjects + a.PendingBlocks
		bytes += a.TotalBytes
	}
	if blocks != r.LiveBlocks {
		t.Fatalf("named actual physical roots %d runtime %d", blocks, r.LiveBlocks)
	}
	if bytes != m.config.Budget.Used() {
		t.Fatalf("actual headers/backings %d budget %d", bytes, m.config.Budget.Used())
	}
}
