// Source: src/util/common_msgs.cpp:21-27.
package messages

import "testing"

func TestSourceMessages(t *testing.T) {
	got := [...]string{Canceled, MaxMemory, MaxScopes, MaxSteps, MaxFrames, NoProofGeneration, MaxResource}
	want := [...]string{"canceled", "max. memory exceeded", "max. scopes exceeded", "max. steps exceeded", "max. frames exceeded", "component does not support proof generation", "max. resource limit exceeded"}
	if got != want {
		t.Fatal(got)
	}
}
