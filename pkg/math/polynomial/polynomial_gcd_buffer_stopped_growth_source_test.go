// Source: src/util/ref_buffer.h; src/util/buffer.h;
// src/math/polynomial/polynomial.cpp:3895-4015.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// Exact captured allocation progress permits signed-host teardown continuation.
func polynomialTLSCloseFull(t *testing.T, m *MultivariatePolynomialManager, r *util.Runtime, mm *MonomialManager) {
	t.Helper()
	for attempts := 0; ; attempts++ {
		if attempts > 64 {
			t.Fatal("bounded actual owning continuation")
		}
		var oldCells [2]*polynomialLocation
		var oldObjects [2]util.ObjectRef
		for i, p := range [2]Polynomial{m.core.zero, m.core.unit} {
			if p != (Polynomial{}) {
				l, e := m.core.polynomialLocation(p)
				serviceMust(t, e)
				if !l.released {
					oldCells[i] = l
					oldObjects[i] = l.raw
				}
			}
		}
		live := r.Stats().LiveBlocks
		e := m.CloseChecked()
		if e == nil {
			break
		}
		if !errors.Is(e, arena.ErrBounds) {
			t.Fatal("actual reached owner refusal", e)
		}
		consumed := false
		for i, l := range oldCells {
			if l != nil && l.released {
				if _, readErr := mm.allocator.ReadUint32(oldObjects[i], 0); readErr == nil {
					t.Fatal("captured cached polynomial allocation remains readable after reached physical release")
				}
				consumed = true
			}
		}
		if consumed {
			if m.core.ids.FreeCapacity() == 0 {
				t.Fatal("actual new ID backing missing")
			}
			polynomialImpPhysicalAuditNamedRaw(t, m, nil, nil, 0, nil)
		}
		if !consumed && r.Stats().LiveBlocks >= live {
			t.Fatal("no reached physical owning progress", e, m.cleanupStage, m.core.closeStage, m.core.zero, m.core.unit, m.core.ids.FreeCapacity(), live, r.Stats().LiveBlocks)
		}
	}
}
