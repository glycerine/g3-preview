package sat

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

func TestSATIntegrityNaryResidualSourceTraceAndAssertion(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	for i := 0; i < 4; i++ {
		_, err := s.MkVar(false, true)
		satOK(t, err)
	}
	cl := satCleanerClause(t, s.Lifecycle, []Literal{0, 2, 4, 6}, true, false)
	satOK(t, s.Lifecycle.attachClause(cl))
	p := s.Lifecycle.propagation
	satOK(t, p.assign(1, satJustification{}))
	defer func() { satOK(t, s.Lifecycle.unassignVars(0)) }()
	p.variables.qhead = 1
	c, err := newSATIntegrityChecker(s.Lifecycle, s.gc)
	satOK(t, err)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	s.Lifecycle.traceTags = tags
	_, err = c.checkClause(cl)
	if satStorageSourceDebug {
		if !errors.Is(err, ErrSATAssertion) {
			t.Fatalf("residual assertion: %v", err)
		}
	} else {
		satOK(t, err)
	}
	if output.Len() != 0 {
		t.Fatalf("disabled residual trace emitted %q", output.String())
	}
	tags.Enable("sat_bug")
	_, err = c.checkClause(cl)
	if satStorageSourceDebug {
		if !errors.Is(err, ErrSATAssertion) {
			t.Fatalf("traced residual assertion: %v", err)
		}
	} else {
		satOK(t, err)
	}
	record := ""
	if tags.ShouldTrace("sat_bug") {
		record = "-------- [sat_bug] check_clause src/sat/sat_integrity_checker.cpp:90 ---------\n" +
			"(0 1 2 3)+* status: l_undef\n" +
			"val(0): l_false\nval(1): l_undef\nval(2): l_undef\nval(3): l_undef\n" +
			"------------------------------------------------\n"
	}
	wantCount := 2
	if satStorageSourceDebug {
		wantCount = 1
	}
	if got := output.String(); got != strings.Repeat(record, wantCount) {
		t.Fatalf("residual source trace: got %q, want %q", got, strings.Repeat(record, wantCount))
	}
	debug, input, assertionFile := satIntegrityActionFixture(t, "CT")
	satOK(t, debug.EnableAssertions(false))
	output.Reset()
	yes, err := c.checkClause(cl)
	satOK(t, err)
	if !yes || output.String() != strings.Repeat(record, 2) {
		t.Fatalf("disabled assertions: result=%v, trace=%q", yes, output.String())
	}
	satOK(t, debug.EnableAssertions(true))
	satOK(t, debug.EnableDebug("cleaner_bug"))
	output.Reset()
	err = s.cleaner.assertInvariant()
	if satStorageSourceDebug {
		var ex z3exception.DefaultException
		if !errors.As(err, &ex) {
			t.Fatalf("real cleaner residual T action: %v", err)
		}
		if got := output.String(); got != strings.Repeat(record, 2) {
			t.Fatalf("cleaner residual trace: got %q, want %q", got, strings.Repeat(record, 2))
		}
		state, e := input.State()
		satOK(t, e)
		if state.Position != 2 {
			t.Fatalf("residual assertion consumed %d actions", state.Position)
		}
		contents, e := os.ReadFile(assertionFile.Name())
		satOK(t, e)
		wantAssertion := "ASSERTION VIOLATION\nFile: src/sat/sat_integrity_checker.cpp\nLine: 93\ns.value(c[i]) == l_false\n" +
			"(C)ontinue, (A)bort, (S)top, (T)hrow exception, Invoke (G)DB\n"
		if string(contents) != strings.Repeat(wantAssertion, 2) {
			t.Fatalf("residual source assertions: got %q", contents)
		}
	} else {
		satOK(t, err)
		if output.Len() != 0 {
			t.Fatalf("release cleaner reached checked trace: %q", output.String())
		}
	}
}
