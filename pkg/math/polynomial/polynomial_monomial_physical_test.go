package polynomial

// Source: src/math/polynomial/polynomial.cpp; src/math/polynomial/polynomial.h.

import (
	umemory "github.com/glycerine/g3/pkg/util/memory"
	"testing"
)

func monomialPhysicalAudit(t *testing.T, m *MonomialManager, externalRaw ...umemory.Block) {
	t.Helper()
	if m == nil {
		return
	}
	r := m.memory.Stats()
	var blocks uint32
	add := func(b umemory.Block) {
		if b != (umemory.Block{}) {
			blocks++
		}
	}
	for _, raw := range externalRaw {
		add(raw)
	}
	bytes := m.header + r.TotalBytes
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
