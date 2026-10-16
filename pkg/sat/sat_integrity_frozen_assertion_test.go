package sat

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/z3exception"
)

func TestSATCleanerFrozenCountAssertionUsesSourceActions(t *testing.T) {
	g, _ := satGCFixture(t, 2)
	g.numFrozen = 1
	defer func() { g.numFrozen = 0 }()
	debug, input, outputFile := satIntegrityActionFixture(t, "CT")
	satOK(t, debug.EnableDebug("cleaner_bug"))
	satOK(t, debug.EnableAssertions(false))
	before := g.lifecycle().propagation.limit.Steps
	satOK(t, g.cleaner.assertInvariant())
	if got := g.lifecycle().propagation.limit.Steps; got != before {
		t.Fatalf("disabled checker reached resource limit: %d", got)
	}
	satOK(t, debug.EnableAssertions(true))
	if !satStorageSourceDebug {
		satOK(t, g.cleaner.assertInvariant())
		return
	}
	satOK(t, g.cleaner.assertInvariant())
	var ex z3exception.DefaultException
	if err := g.cleaner.assertInvariant(); !errors.As(err, &ex) {
		t.Fatalf("frozen count T action: %v", err)
	}
	state, err := input.State()
	satOK(t, err)
	if state.Position != 2 {
		t.Fatalf("frozen count consumed %d actions", state.Position)
	}
	contents, err := os.ReadFile(outputFile.Name())
	satOK(t, err)
	want := "ASSERTION VIOLATION\nFile: src/sat/sat_integrity_checker.cpp\nLine: 124\nnum_frozen == s.m_num_frozen\n" +
		"(C)ontinue, (A)bort, (S)top, (T)hrow exception, Invoke (G)DB\n"
	if string(contents) != strings.Repeat(want, 2) {
		t.Fatalf("frozen count source records: got %q, want %q", contents, strings.Repeat(want, 2))
	}
	g.numFrozen = 0
	satOK(t, g.cleaner.assertInvariant())
}
