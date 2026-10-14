package diagnostic

import "testing"

func TestProcessDebugDefaultOwnsSourceAssertionIO(t *testing.T) {
	d := BorrowProcessDebugContext()
	if d == nil || d != defaultProcessDebugContext {
		t.Fatal("default process debug owner")
	}
	on, err := d.AssertionsEnabled()
	if err != nil || !on {
		t.Fatal("source assertion switch default", on, err)
	}
	if d.assertionStreams == nil || d.assertionActions == nil || d.assertionActions.profile != DebugInteractive {
		t.Fatal("default source notification and debugger action")
	}
	stdout, err := d.assertionStreams.Stdout()
	if err != nil {
		t.Fatal(err)
	}
	stderr, err := d.assertionStreams.Stderr()
	if err != nil || stdout.File == nil || stderr.File == nil {
		t.Fatal("default borrowed standard streams", err)
	}
	tied, err := d.assertionActions.input.Tie()
	if err != nil || tied != stdout {
		t.Fatal("debugger input must share default stdout", err)
	}
}
