// Source: src/util/vector.h:420-433,509-511; max_cliques owning nested rows.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package g3

import "testing"

func TestSATMaxCliquesNestedResizeRefusalRetainsOldLogicalSize(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	g, e := newSATMaxCliques(s.runtime.memory, s.budget)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, g.Close()) })
	satOK(t, g.next.reserveEmptyRows(1))
	s.budget.FailAfter(2)
	e = g.next.reserveEmptyRows(9)
	satProbingExpectedRefusal(t, s.propagation, e)
	n, er := g.next.rowCount()
	satOK(t, er)
	if n != 1 {
		t.Error("source retains old SIZE until every expansion succeeds", n)
	}
	satMutexGraphAudit(t, s, g)
	s.budget.FailAfter(-1)
}
