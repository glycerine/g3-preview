// Source: src/util/common_msgs.h:22-39; src/util/common_msgs.cpp:21-27;
// src/util/rlimit.cpp:62-68; src/sat/sat_solver.h:268-282.
package util

import (
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
	var r Reslimit
	Z3MaxResourceMessage = "replacement resource limit"
	if r.CancelMessage() != Z3MaxResourceMessage {
		t.Fatal("resource message captured before the reached lookup")
	}
	r.cancelCount.Store(1)
	Z3CanceledMessage = "replacement cancellation"
	if r.CancelMessage() != Z3CanceledMessage {
		t.Fatal("cancellation message captured before the reached lookup")
	}
	Z3CanceledMessage = "second cancellation"
	if r.CancelMessage() != Z3CanceledMessage {
		t.Fatal("same resource owner did not reread the shared binding")
	}
}
