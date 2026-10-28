package nlsat

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/sat"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

// The real thread-local size limit stops after raw allocation but before the
// allocator publishes an object, producing a qualified pending physical block.
func TestAssumptionAndLazyOwnersPreserveEachOthersPendingAllocation(t *testing.T) {
	for _, olderLazy := range []bool{false, true} {
		t.Run(map[bool]string{false: "assumption-older", true: "lazy-older"}[olderLazy], func(t *testing.T) {
			m := assumptionFixtureWithProfile(t, true)
			a := m.allocator
			rt := a.RuntimeOwner()
			held, err := rt.Allocate(99980)
			assignmentMust(t, err)
			assignmentMust(t, rt.SetMaxSize(1))
			var older Justification
			var outcome *umemory.SmallAllocatorAllocationOutcome
			if olderLazy {
				older, err = NewLazyJustification(a, []sat.Literal{3, 5, 7})
				outcome = &older.lazy.allocation
			} else {
				var d *AssumptionSet
				d, err = m.mkLeaf(Assumption{})
				if d != nil {
					t.Fatal("failed factory published a dependency")
				}
				outcome = &m.allocation
			}
			if err == nil || !a.MatchesPendingAllocationOutcome(outcome) {
				t.Fatal("no qualified pending allocation", err)
			}
			before, used := rt.Stats(), m.budget.Used()
			if olderLazy {
				if _, err = m.mkLeaf(Assumption{}); !errors.Is(err, umemory.ErrAllocatorPending) {
					t.Fatal("younger assumption entry", err)
				}
				assignmentMust(t, m.cleanup())
			} else {
				younger, err := NewLazyJustification(a, []sat.Literal{11})
				if !errors.Is(err, umemory.ErrAllocatorPending) {
					t.Fatal("younger lazy entry", err)
				}
				assignmentMust(t, DeleteJustification(a, younger))
			}
			if !a.MatchesPendingAllocationOutcome(outcome) || rt.Stats() != before || m.budget.Used() != used {
				t.Fatal("younger owner cleared or changed the older physical prefix")
			}
			if olderLazy {
				assignmentMust(t, DeleteJustification(a, older))
			} else {
				assignmentMust(t, m.cleanup())
			}
			if rt.Stats().LiveBlocks != 1 {
				t.Fatal("older pending root was not released", rt.Stats())
			}
			assignmentMust(t, rt.SetMaxSize(0))
			assignmentMust(t, rt.Deallocate(&held))
		})
	}
}
