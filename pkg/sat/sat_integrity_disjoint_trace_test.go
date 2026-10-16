package sat

import (
	"bytes"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

func TestSATIntegrityRepeatedClauseTraceAndOperatorAssertion(t *testing.T) {
	g, _ := satGCFixture(t, 4)
	l := g.lifecycle()
	a := l.propagation.clauses
	original := satCleanerClause(t, l, []Literal{0, 2, 4}, false, false)
	satOK(t, l.attachClause(original))
	learned := satGCClause(t, g, []Literal{0, 2, 4, 6}, 1)
	id, err := a.ID(original)
	satOK(t, err)
	priorID, err := a.ID(learned)
	satOK(t, err)
	satOK(t, a.writeWord(learned, 0, id))
	defer func() { satOK(t, a.writeWord(learned, 0, priorID)) }()
	c, err := newSATIntegrityChecker(l, g)
	satOK(t, err)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	l.traceTags = tags
	yes, err := c.checkDisjointClauses()
	satOK(t, err)
	if yes || output.Len() != 0 {
		t.Fatalf("disabled repeated-clause trace: result=%v output=%q", yes, output.String())
	}
	tags.Enable("sat")
	yes, err = c.checkDisjointClauses()
	satOK(t, err)
	if yes {
		t.Fatal("same clause ID was accepted")
	}
	wantRecord := ""
	if tags.ShouldTrace("sat") {
		wantRecord = "-------- [sat] check_disjoint_clauses src/sat/sat_integrity_checker.cpp:221 ---------\n" +
			"Repeated clause: " + strconv.FormatUint(uint64(id), 10) + "\n" +
			"------------------------------------------------\n"
	}
	if got := output.String(); got != wantRecord {
		t.Fatalf("source repeated-clause trace: got %q, want %q", got, wantRecord)
	}
	debug, input, assertionFile := satIntegrityActionFixture(t, "CT")
	satOK(t, debug.EnableDebug("cleaner_bug"))
	satOK(t, debug.EnableAssertions(false))
	satOK(t, g.cleaner.assertInvariant())
	satOK(t, debug.EnableAssertions(true))
	output.Reset()
	if !satStorageSourceDebug {
		satOK(t, g.cleaner.assertInvariant())
		return
	}
	satOK(t, g.cleaner.assertInvariant())
	var ex z3exception.DefaultException
	if err = g.cleaner.assertInvariant(); !errors.As(err, &ex) {
		t.Fatalf("operator disjoint T action: %v", err)
	}
	if got := output.String(); got != strings.Repeat(wantRecord, 2) {
		t.Fatalf("operator trace order: got %q, want %q", got, strings.Repeat(wantRecord, 2))
	}
	state, err := input.State()
	satOK(t, err)
	if state.Position != 2 {
		t.Fatalf("operator consumed %d debugger actions", state.Position)
	}
	contents, err := os.ReadFile(assertionFile.Name())
	satOK(t, err)
	wantAssertion := "ASSERTION VIOLATION\nFile: src/sat/sat_integrity_checker.cpp\nLine: 236\ncheck_disjoint_clauses()\n" +
		"(C)ontinue, (A)bort, (S)top, (T)hrow exception, Invoke (G)DB\n"
	if string(contents) != strings.Repeat(wantAssertion, 2) {
		t.Fatalf("operator source assertions: got %q, want %q", contents, strings.Repeat(wantAssertion, 2))
	}
	satOK(t, a.writeWord(learned, 0, priorID))
	satOK(t, g.cleaner.assertInvariant())
}
