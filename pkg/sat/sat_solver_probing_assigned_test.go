// Source: src/sat/sat_types.h; src/sat/sat_probing.cpp. Licensed under the MIT license.
package sat

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestSATSolverProbingAssignedSetReachesActualRuntime(t *testing.T) {
	s, _ := satProbingFixture(t, 8)
	before, e := s.propagation().memory.AllocationSize()
	satOK(t, e)
	s.cacheLimit = before
	satOK(t, s.assigned.Insert(14))
	after, e := s.propagation().memory.AllocationSize()
	satOK(t, e)
	if after <= before {
		t.Fatal("actual source membership and values allocations absent from shared Runtime", before, after)
	}
	satOK(t, s.cacheBins(0, 0))
	if s.cache.blockLive() || len(s.cache.rows) != 0 {
		t.Fatal("actual assigned allocation triggers cache cutoff")
	}
	satProbingAudit(t, s, nil, nil)

}

func TestSATSolverProbingSourceFinalizePermitsNextRound(t *testing.T) {
	s, _ := satProbingFixture(t, 1)
	done, e := s.run(true)
	satOK(t, e)
	if !done {
		t.Fatal("first normal round")
	}
	done, e = s.run(true)
	satOK(t, e)
	if !done {
		t.Fatal("second normal round")
	}
	satOK(t, s.process(0))
	satProbingAudit(t, s, nil, nil)
}

func TestSATSolverProbingAssignedMembershipBeforeListRefusal(t *testing.T) {
	s, life := satProbingFixture(t, 2)
	satOK(t, s.assigned.reserve(3))
	s.budget.FailAfter(0)
	e := s.assigned.Insert(2)
	satProbingExpectedRefusal(t, life.propagation, e)
	member, readError := s.assigned.contains(2)
	satOK(t, readError)
	size, readError := s.assigned.size()
	satOK(t, readError)
	if !member || size != 0 {
		t.Fatal("completed membership bit before failed actual values allocation", member, size)
	}
	satProbingAudit(t, s, nil, nil)
	s.budget.FailAfter(-1)
	// Explicit source reset, not automatic rollback: logical membership becomes0
	// while its actual initialized storage remains until finalize.
	old := s.assigned.membership.Block
	satOK(t, s.assigned.Reset())
	member, readError = s.assigned.contains(2)
	satOK(t, readError)
	if member || s.assigned.membership.Block != old {
		t.Fatal("source reset retains backing and clears logical size")
	}
	satProbingAudit(t, s, nil, nil)
}

func TestSATSolverProbingUnavailableLevelHeaderAtSourceRead(t *testing.T) {
	s, life := satProbingFixture(t, 1)
	v := life.propagation.variables
	// A genuine allocated but unconstructed raw header exercises the checked
	// source-read boundary. It is not a normal solver input or a repaired numeral.
	malformed, e := life.propagation.memory.Allocate(16)
	satOK(t, e)
	original := v.level.Raw.Block
	v.level.Raw.Block = malformed
	t.Cleanup(func() {
		v.level.Raw.Block = original
		if malformed != (util.Block{}) {
			satOK(t, life.propagation.memory.Deallocate(&malformed))
		}
	})
	done, e := s.run(true)
	if e == nil || done {
		t.Fatal("num_vars must not swallow its actual header read failure", done, e)
	}
	if !errors.Is(e, util.ErrRuntimeUninitialized) {
		t.Fatal(e)
	}
	satProbingAudit(t, s, nil, nil, original)
	// Caller restores its borrowed binding, then releases only its own raw block.
	v.level.Raw.Block = original
	satOK(t, life.propagation.memory.Deallocate(&malformed))
}
