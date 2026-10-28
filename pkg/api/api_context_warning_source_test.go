package api

// Source: src/api/api_context.cpp:109-117; src/util/warning.cpp.

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestAPIContextUncollectedObjectSourceWarningGate(t *testing.T) {
	warnings := diagnostic.BorrowProcessWarningService()
	wasEnabled := warnings.Enabled()
	var output bytes.Buffer
	astTestMust(t, warnings.SetWarningStream(&output))
	astTestMust(t, warnings.Enable(true))
	defer func() {
		astTestMust(t, warnings.SetWarningStream(nil))
		astTestMust(t, warnings.Enable(wasEnabled))
	}()

	m := astTestManager(t)
	c, err := newAPIContextResults(m, false)
	astTestMust(t, err)
	goal, err := newAPIGoalObject(c)
	astTestMust(t, err)
	astTestMust(t, c.Close())
	if !goal.object.closed {
		t.Fatal("context destructor did not release uncollected goal")
	}
	want := ""
	if diagnostic.DebugBuild {
		want = fmt.Sprintf("WARNING: Uncollected memory: %d: %T\n", goal.object.id, goal)
	}
	if got := output.String(); got != want {
		t.Fatalf("source warning gate: got %q, want %q", got, want)
	}
}

func TestAPIToggleWarningMessagesControlsReachedContextWarning(t *testing.T) {
	warnings := diagnostic.BorrowProcessWarningService()
	wasEnabled := warnings.Enabled()
	var output bytes.Buffer
	astTestMust(t, warnings.SetWarningStream(&output))
	defer func() {
		astTestMust(t, warnings.SetWarningStream(nil))
		ToggleWarningMessages(wasEnabled)
	}()

	m := astTestManager(t)
	for _, enabled := range []bool{false, true} {
		ToggleWarningMessages(enabled)
		c, err := newAPIContextResults(m, false)
		astTestMust(t, err)
		_, err = newAPIGoalObject(c)
		astTestMust(t, err)
		astTestMust(t, c.Close())
		if got := output.String(); !enabled && got != "" {
			t.Fatalf("disabled source warning: %q", got)
		} else if enabled && diagnostic.DebugBuild && !strings.HasPrefix(got, "WARNING: Uncollected memory: ") {
			t.Fatalf("enabled source warning: %q", got)
		}
	}
}
