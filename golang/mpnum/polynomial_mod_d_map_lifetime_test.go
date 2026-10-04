// Source: src/math/polynomial/polynomial.cpp:6898-6922; src/util/vector.h:191-203.
// Licensed under the MIT license.
package mpnum

import (
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestPolynomialModDConstantWithStoppedMapAssignmentKeepsActualRoots(t *testing.T) {
	m := polynomialCellFixture(t)
	target, e := NewPolynomialVariableDegrees(m.memory)
	serviceMust(t, e)
	source, e := NewPolynomialVariableDegrees(m.memory)
	serviceMust(t, e)
	t.Cleanup(func() {
		m.integer.budget.FailAfter(-1)
		serviceMust(t, target.CloseChecked())
		serviceMust(t, source.CloseChecked())
	})
	serviceMust(t, target.SetDegree(1, 2))
	serviceMust(t, source.SetDegree(1, 3))
	old := target.raw
	m.integer.budget.FailAfter(0)
	e = target.CopyFrom(source)
	if !numericPhysicalReservationRefusal(e) || !target.rawDestroyed || target.raw != old || target.pending != (memory.Block{}) {
		t.Fatal("actual completed old free then replacement refusal", e, target)
	}
	cause := target.bodyCause
	got, e := m.ModD(m.unit, target)
	serviceMust(t, e)
	if got != m.unit || target.bodyCause != cause || !target.rawDestroyed || target.raw != old {
		t.Fatal("constant source branch read or repaired stopped mapping", got, target)
	}
	// The stale source pointer is history, not an owning physical block. Both
	// real cold map objects and source's live raw block remain named here.
	polynomialModDNewtonPhysicalWithRaw(t, m, nil, nil, nil, nil, nil, nil, nil, []*PolynomialVariableDegrees{target, source})
	m.integer.budget.FailAfter(-1)
	serviceMust(t, target.CloseChecked())
	polynomialModDNewtonPhysicalWithRaw(t, m, nil, nil, nil, nil, nil, nil, nil, []*PolynomialVariableDegrees{target, source})
}
