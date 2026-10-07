// Source: src/sat/sat_solver.cpp:723–802,993–1034; source repeated Check.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package sat

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestSATSolverWarmRepeatedCheckHasBoundedActualPhysicalGraph(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	for i := 0; i < 2; i++ {
		_, e := s.MkVar(true, true)
		satOK(t, e)
	}
	satOK(t, s.MkClause([]Literal{0, 2}))
	satOK(t, s.MkClause([]Literal{1, 3}))
	for i := 0; i < 8; i++ {
		v, e := s.Check(nil)
		satOK(t, e)
		if v != util.LTrue {
			t.Fatal(v)
		}
	}
	before := s.budget.Used()
	blocks := s.runtime.memory.Stats().LiveBlocks
	for i := 0; i < 64; i++ {
		v, e := s.Check(nil)
		satOK(t, e)
		if v != util.LTrue {
			t.Fatal(v)
		}
		satFullSolverRuntimeAudit(t, s)
	}
	if s.budget.Used() != before || s.runtime.memory.Stats().LiveBlocks != blocks {
		t.Fatal("source working owners did not plateau", before, s.budget.Used(), blocks, s.runtime.memory.Stats())
	}
	allocs := testing.AllocsPerRun(10, func() {
		v, e := s.Check(nil)
		if e != nil || v != util.LTrue {
			panic("real warm source Check")
		}
	})
	t.Log("actual warmed Go allocations per Check", allocs, "physical bytes", before, "Runtime blocks", blocks)
	// This is measured source lexical behavior; do not cache source temporaries
	// or force zero allocations by changing its algorithm or defaults.
}
