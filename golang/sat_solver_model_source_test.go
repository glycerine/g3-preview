// Source: src/sat/sat_solver.cpp:1257–1346; src/sat/sat_clause.cpp satisfied_by.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"testing"
)

func TestSATSolverSourceModelUsesRealAssignmentAndConverter(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	for i := 0; i < 2; i++ {
		_, e := s.variables.mkVar(false, true)
		satOK(t, e)
	}
	_, e := s.lifecycle.mkClauseCore(2, []satLiteral{0, 2}, false)
	satOK(t, e)
	satOK(t, s.propagation.assign(0, satJustification{}))
	satOK(t, s.propagation.assign(3, satJustification{}))
	satOK(t, s.mkModel())
	m, e := s.AppendModel(nil)
	satOK(t, e)
	if len(m) != 2 || m[0] != lTrue || m[1] != lFalse {
		t.Fatal(m)
	}
	good, e := s.checkModel()
	satOK(t, e)
	if !good {
		t.Fatal("valid model rejected")
	}
	satOK(t, (satSolverModelSpan{column: &s.model}).write(0, lFalse))
	good, e = s.checkModel()
	satOK(t, e)
	if good {
		t.Fatal("unsatisfied binary model accepted")
	}
}
func TestSATSolverSourceModelPublishesCurrentBeforeNumVarsRead(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	_, e := s.variables.mkVar(false, true)
	satOK(t, e)
	// Actual initialized-byte guard at source num_vars()==m_level.size().
	raw := s.variables.level.raw.block
	satOK(t, s.runtime.memory.Deallocate(&raw))
	s.variables.modelIsCurrent = false
	e = s.mkModel()
	if e == nil || !s.variables.modelIsCurrent {
		t.Fatal("source reset/current/read ordering", e)
	}
	if n, e := s.model.size(); e != nil || n != 0 {
		t.Fatal(n, e)
	}
	// Explicit caller acknowledgment of the already freed malformed member;
	// no production source header repair or bypass is performed.
	s.variables.level.raw.block = raw
	if satStorageSourceDebug {
		// This malformed member also violates the actual destructor invariant.
		// Observe that source guard before explicitly acknowledging only this
		// diagnostic fixture's stopped assertion for owning member teardown.
		if e = s.Close(); !errors.Is(e, ErrSATAssertion) || s.closeStage != 1 {
			t.Fatal("source destructor guard", e, s.closeStage)
		}
		satOK(t, s.CleanupStopped())
	}
	satOK(t, s.Close())
	if e = s.mkModel(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}
