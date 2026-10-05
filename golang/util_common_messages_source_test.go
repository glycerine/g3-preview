// Source: src/util/common_msgs.h:22-39; src/util/common_msgs.cpp:21-27;
// src/util/rlimit.cpp:62-68; src/sat/sat_solver.h:268-282.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package g3

import (
	"errors"
	"testing"
)

func TestCommonMessagesSourceDefaults(t *testing.T) {
	got := [7]string{Z3CanceledMessage, Z3MaxMemoryMessage, Z3MaxScopesMessage, Z3MaxStepsMessage, Z3MaxFramesMessage, Z3NoProofGenerationMessage, Z3MaxResourceMessage}
	want := [7]string{"canceled", "max. memory exceeded", "max. scopes exceeded", "max. steps exceeded", "max. frames exceeded", "component does not support proof generation", "max. resource limit exceeded"}
	if got != want {
		t.Fatal(got)
	}
}

func TestCommonMessagesAllStaticBindingsRemainMutable(t *testing.T) {
	bindings := [7]*string{&Z3CanceledMessage, &Z3MaxMemoryMessage, &Z3MaxScopesMessage, &Z3MaxStepsMessage, &Z3MaxFramesMessage, &Z3NoProofGenerationMessage, &Z3MaxResourceMessage}
	var original [7]string
	for i, p := range bindings {
		original[i] = *p
	}
	t.Cleanup(func() {
		for i, p := range bindings {
			*p = original[i]
		}
	})
	for i, p := range bindings {
		*p = "replacement " + original[i]
		if *p == original[i] {
			t.Fatal("source static pointer binding was frozen", i)
		}
		for j, other := range bindings {
			if i != j && *other != original[j] {
				t.Fatal("distinct source static bindings aliased", i, j)
			}
		}
		*p = original[i]
	}
}

func TestCommonMessagesResourceLimitReadsCurrentBindings(t *testing.T) {
	oldCancel, oldResource := Z3CanceledMessage, Z3MaxResourceMessage
	t.Cleanup(func() { Z3CanceledMessage, Z3MaxResourceMessage = oldCancel, oldResource })
	var r utilReslimit
	Z3MaxResourceMessage = "replacement resource limit"
	if r.cancelMessage() != Z3MaxResourceMessage {
		t.Fatal("resource message captured before the reached lookup")
	}
	r.cancelCount.Store(1)
	Z3CanceledMessage = "replacement cancellation"
	if r.cancelMessage() != Z3CanceledMessage {
		t.Fatal("cancellation message captured before the reached lookup")
	}
	Z3CanceledMessage = "second cancellation"
	if r.cancelMessage() != Z3CanceledMessage {
		t.Fatal("same resource owner did not reread the shared binding")
	}
}

func TestCommonMessagesSATExceptionCapturesReachedBinding(t *testing.T) {
	oldCancel, oldMemory := Z3CanceledMessage, Z3MaxMemoryMessage
	t.Cleanup(func() { Z3CanceledMessage, Z3MaxMemoryMessage = oldCancel, oldMemory })
	s, _, _ := satPropagationFixture(t, 0)
	s.limit.cancel()
	Z3CanceledMessage = "replacement cancellation"
	e := s.checkpoint()
	var ex SATSolverException
	if !errors.As(e, &ex) || ex.Message() != Z3CanceledMessage || ex.ErrorCode() != ErrorOK {
		t.Fatal("reached solver exception did not read shared cancellation binding", e)
	}
	Z3CanceledMessage = "later cancellation"
	if ex.Message() != "replacement cancellation" {
		t.Fatal("already thrown exception changed with the global binding")
	}
	s.limit.resetCancel()
	s.config.state.MaxMemory = 0
	h, e := s.memory.Allocate(8)
	satOK(t, e)
	defer func() { satOK(t, s.memory.Deallocate(&h)) }()
	Z3MaxMemoryMessage = "replacement memory limit"
	for i := 0; i < 9; i++ {
		satOK(t, s.checkpoint())
	}
	e = s.checkpoint()
	if !errors.As(e, &ex) || ex.Message() != Z3MaxMemoryMessage || ex.ErrorCode() != ErrorOK || s.numCheckpoints != 0 {
		t.Fatal("reached memory exception did not capture shared binding", e)
	}
	Z3MaxMemoryMessage = "later memory limit"
	if ex.Message() != "replacement memory limit" {
		t.Fatal("already thrown memory exception changed")
	}
}
