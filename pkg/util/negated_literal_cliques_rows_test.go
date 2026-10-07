// Source: src/util/vector.h:420-433,509-511; max_cliques owning nested rows.
package util

import (
	"testing"
)

func TestNegatedLiteralCliquesNestedResizeRefusalRetainsOldLogicalSize(t *testing.T) {
	m, b := memoryRuntimeFixture(t, false, 64)
	g, e := NewNegatedLiteralCliques(m, b)
	memoryRuntimeRequire(t, e)
	t.Cleanup(func() { memoryRuntimeRequire(t, g.Close()) })
	memoryRuntimeRequire(t, g.next.ReserveEmptyRows(1))
	b.FailAfter(2)
	e = g.next.ReserveEmptyRows(9)
	cliqueTestRefusal(t, e)
	n, er := g.next.RowCount()
	memoryRuntimeRequire(t, er)
	if n != 1 {
		t.Error("source retains old SIZE until every expansion succeeds", n)
	}
	cliqueTestAudit(t, m, g, nil)
	b.FailAfter(-1)
}
