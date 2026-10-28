package polynomial

// Source: src/math/polynomial/polynomial.cpp:810-851.

import "testing"

func TestMonomialUnitNativeRecycleFinalizationIsIdempotent(t *testing.T) {
	m := monomialFixture(t)
	b := m.memory.Budget()
	b.FailAfter(0)
	serviceMust(t, m.SourceFinalize())
	monomialPhysicalAudit(t, m)
	before := m.memory.State()
	serviceMust(t, m.SourceFinalize())
	if m.memory.State() != before {
		t.Fatal("finalization repeated unit destruction")
	}
	b.FailAfter(-1)
}
