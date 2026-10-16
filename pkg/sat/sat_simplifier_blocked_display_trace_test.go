// Source: src/sat/sat_simplifier.cpp:1079.
package sat

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	utext "github.com/glycerine/g3/pkg/util/text"
)

type satBlockedDisplayObserver struct {
	output bytes.Buffer
	solver *Solver
	match  []bool
}

func (o *satBlockedDisplayObserver) Write(p []byte) (int, error) {
	if bytes.HasPrefix(p, []byte("trail: ")) {
		s := o.solver
		n, err := s.Variables.trail.Size(s.Variables.trailRef)
		if err != nil {
			return 0, err
		}
		out, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: arena.NewBudget(^uint64(0))})
		if err != nil {
			return 0, err
		}
		defer out.Close()
		if err = s.displayWatches(out); err != nil {
			return 0, err
		}
		if err = s.displayInto(out); err != nil {
			return 0, err
		}
		body, err := out.String()
		if err != nil {
			return 0, err
		}
		want := "trail: " + strconv.FormatUint(uint64(n), 10) + "\n" + body
		o.match = append(o.match, string(p) == want)
	}
	return o.output.Write(p)
}

func TestSATActualBlockedEliminationSolverDisplaySourceTrace(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	for i := 0; i < 3; i++ {
		_, err := s.MkVar(true, true)
		satOK(t, err)
	}
	satOK(t, s.MkClause([]Literal{0, 2, 4}))
	observer := &satBlockedDisplayObserver{solver: s}
	tags := diagnostic.NewTraceTagsWithWriter(observer)
	tags.Enable("blocked_clause_bug")
	s.Lifecycle.traceTags = tags
	for i := 0; i < 3; i++ {
		satOK(t, s.simplifier.Run(false))
	}
	if !tags.ShouldTrace("blocked_clause_bug") {
		if observer.output.Len() != 0 {
			t.Fatal("build-erased blocked solver display", observer.output.String())
		}
		return
	}
	if len(observer.match) != 1 || !observer.match[0] {
		t.Fatal("source blocked display must use the live solver at emission", observer.match)
	}
	got := observer.output.String()
	if !strings.HasPrefix(got, "-------- [blocked_clause_bug] elim_blocked_clauses src/sat/sat_simplifier.cpp:1079 ---------\ntrail: ") ||
		strings.Count(got, "src/sat/sat_simplifier.cpp:1079") != 1 ||
		strings.Count(got, "------------------------------------------------\n") != 1 {
		t.Fatal("one complete third-call blocked display record", got)
	}
}
