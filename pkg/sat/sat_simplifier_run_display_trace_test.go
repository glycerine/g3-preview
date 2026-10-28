package sat

// Source: src/sat/sat_simplifier.cpp:143-169.

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

type satSimplifierRunDisplayObserver struct {
	output bytes.Buffer
	solver *Solver
	match  []bool
}

func (w *satSimplifierRunDisplayObserver) Write(p []byte) (int, error) {
	if bytes.HasPrefix(p, []byte("(sat\n")) {
		var current bytes.Buffer
		err := w.solver.traceUserDisplayInto(&current)
		w.match = append(w.match, err == nil && bytes.Equal(p, current.Bytes()))
	}
	return w.output.Write(p)
}

func TestSATActualSimplifierInitializeAndRunDisplaySourceTraces(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	for i := 0; i < 3; i++ {
		_, err := s.MkVar(true, true)
		satOK(t, err)
	}
	satOK(t, s.MkClause([]Literal{0, 2, 4}))
	observer := &satSimplifierRunDisplayObserver{solver: s}
	tags := diagnostic.NewTraceTagsWithWriter(observer)
	tags.Enable("after_cleanup")
	tags.Enable("before_simplifier")
	s.Lifecycle.traceTags = tags
	satOK(t, s.simplifier.Run(false))
	if !tags.ShouldTrace("after_cleanup") {
		if observer.output.Len() != 0 {
			t.Fatal("build-erased source displays", observer.output.String())
		}
		return
	}
	if len(observer.match) != 3 || !observer.match[0] || !observer.match[1] || !observer.match[2] {
		t.Fatal("each trace body must match the live solver display at emission", observer.match)
	}
	got := observer.output.String()
	header := []string{
		"-------- [after_cleanup] initialize src/sat/sat_simplifier.cpp:147 ---------\n",
		"-------- [before_simplifier] operator() src/sat/sat_simplifier.cpp:162 ---------\n",
		"-------- [after_cleanup] operator() src/sat/sat_simplifier.cpp:168 ---------\n",
	}
	position := 0
	for _, h := range header {
		i := strings.Index(got[position:], h)
		if i < 0 {
			t.Fatalf("source display header %q missing or out of order in %q", h, got)
		}
		position += i + len(h)
	}
	if strings.Count(got, "src/sat/sat_simplifier.cpp:") != 3 || strings.Count(got, "------------------------------------------------\n") != 3 {
		t.Fatal("expected exactly three complete source records", got)
	}
}
