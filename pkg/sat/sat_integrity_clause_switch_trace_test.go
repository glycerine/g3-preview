package sat

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

type satCloseDebugOnTrace struct {
	debug  *diagnostic.DebugContext
	closed bool
}

func (w *satCloseDebugOnTrace) Write(p []byte) (int, error) {
	if !w.closed {
		w.closed = true
		if err := w.debug.Close(); err != nil {
			return 0, err
		}
	}
	return len(p), nil
}

// Source: src/sat/sat_integrity_checker.cpp:55-60. The tagged CTRACE runs
// between two independent SASSERT expansions for the same clause literal.
func TestSATIntegrityClauseRereadsAssertionSwitchAfterTrace(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	for i := 0; i < 3; i++ {
		_, err := s.MkVar(false, true)
		satOK(t, err)
	}
	cl := satCleanerClause(t, s.Lifecycle, []Literal{0, 2, 4}, true, true)
	s.Variables.eliminated[0] = true
	defer func() { s.Variables.eliminated[0] = false }()
	checker, err := newSATIntegrityChecker(s.Lifecycle, s.gc)
	satOK(t, err)
	debug, _, _ := satIntegrityActionFixture(t, "")
	satOK(t, debug.EnableAssertions(true))
	writer := &satCloseDebugOnTrace{debug: debug}
	tags := diagnostic.NewTraceTagsWithWriter(writer)
	tags.Enable("sat_bug")
	s.Lifecycle.traceTags = tags
	if !tags.ShouldTrace("sat_bug") {
		return
	}
	yes, err := checker.checkClause(cl)
	if !writer.closed {
		t.Fatal("eliminated-clause trace did not run")
	}
	if satStorageSourceDebug {
		if yes || !errors.Is(err, arena.ErrClosed) {
			t.Fatal("line 60 did not re-read the closed debug owner", yes, err)
		}
	} else if err != nil || !yes {
		t.Fatal("release build evaluated an erased assertion", yes, err)
	}
}
