// Source: src/sat/sat_solver.h; src/sat/sat_solver.cpp; src/sat/sat_var_queue.h; src/util/vector.h; src/util/memory_manager.cpp.
package sat

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestSATSolverNativeVariableColumnsAvoidRuntimeAllocation(t *testing.T) {
	v, _, _ := satVariableStateFixture(t)
	_, e := v.mkVar(true, true)
	satOK(t, e)
	amount, e := v.memory.AllocationSize()
	satOK(t, e)
	if amount != 0 || v.memory.Stats().LiveBlocks != 0 {
		t.Fatal("all actual scalar source allocations", amount, v.memory.Stats())
	}
}

func satScalarTestUint(t *testing.T, c *RuntimeUintColumn, i uint32) uint32 {
	t.Helper()
	v, e := c.Read(i)
	satOK(t, e)
	return v
}
func satScalarTestUintSize(t *testing.T, c *RuntimeUintColumn) uint32 {
	t.Helper()
	v, e := c.Size()
	satOK(t, e)
	return v
}

func satScalarAuditRoots(t *testing.T, v *satSolverVariableState, external ...util.Block) {
	t.Helper()
	roots := append([]util.Block{}, external...)
	satRuntimeAuditBlocks(t, v.memory, roots...)
}

func satScalarAuditOwner(t *testing.T, v *satSolverVariableState, external ...util.Block) {
	t.Helper()
	satScalarAuditRoots(t, v, external...)
	todo := v.elimTodo.Stats()
	todoBytes := todo.OwnerBytes
	if v.Stats().TotalBytes+todoBytes+v.memory.Stats().TotalBytes != v.budget.Used() {
		t.Fatal("once-only scalar graph", v.Stats(), v.memory.Stats(), v.budget.Used())
	}
}
func TestSATSolverScalarUnsignedGrowthLiteralWidths(t *testing.T) {
	for _, test := range []struct{ old, n, bytes uint32 }{{0, 2, 16}, {2, 3, 20}, {3, 5, 28}, {5, 8, 40}, {8, 12, 56}, {1 << 30, 1610612736, 2147483656}} {
		n, b, e := RuntimeScalarColumnNext(test.old, 4)
		satOK(t, e)
		if n != test.n || b != test.bytes {
			t.Fatal(test, n, b)
		}
	}
}
