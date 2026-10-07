// Source: src/sat/sat_solver.cpp (shared message integration).
package sat

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestCommonMessagesSATExceptionCapturesReachedBinding(t *testing.T) {
	oldCancel, oldMemory := util.Z3CanceledMessage, util.Z3MaxMemoryMessage
	t.Cleanup(func() { util.Z3CanceledMessage, util.Z3MaxMemoryMessage = oldCancel, oldMemory })
	s, _, _ := satPropagationFixture(t, 0)
	s.limit.Cancel()
	util.Z3CanceledMessage = "replacement cancellation"
	e := s.checkpoint()
	var ex SATSolverException
	if !errors.As(e, &ex) || ex.Message() != util.Z3CanceledMessage || ex.ErrorCode() != util.ErrorOK {
		t.Fatal("reached solver exception did not read shared cancellation binding", e)
	}
	util.Z3CanceledMessage = "later cancellation"
	if ex.Message() != "replacement cancellation" {
		t.Fatal("already thrown exception changed with the global binding")
	}
	s.limit.ResetCancel()
	s.config.state.MaxMemory = 0
	h, e := s.memory.Allocate(8)
	satOK(t, e)
	defer func() { satOK(t, s.memory.Deallocate(&h)) }()
	util.Z3MaxMemoryMessage = "replacement memory limit"
	for i := 0; i < 9; i++ {
		satOK(t, s.checkpoint())
	}
	e = s.checkpoint()
	if !errors.As(e, &ex) || ex.Message() != util.Z3MaxMemoryMessage || ex.ErrorCode() != util.ErrorOK || s.numCheckpoints != 0 {
		t.Fatal("reached memory exception did not capture shared binding", e)
	}
	util.Z3MaxMemoryMessage = "later memory limit"
	if ex.Message() != "replacement memory limit" {
		t.Fatal("already thrown memory exception changed")
	}
}
