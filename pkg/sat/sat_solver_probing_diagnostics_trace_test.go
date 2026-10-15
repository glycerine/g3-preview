package sat

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/lbool"
)

// Source: src/sat/sat_solver.cpp:3053-3065. Both missed-unit and missed-empty
// paths print before the source unreachable assertion; frozen clauses do not.
func TestSATMissedPropagationSourceTraceBeforeAssertion(t *testing.T) {
	for _, mode := range []string{"unit", "empty", "frozen"} {
		t.Run(mode, func(t *testing.T) {
			_, life := satProbingFixture(t, 3)
			c, e := life.propagation.clauses.MkClause([]Literal{0, 2, 4}, true)
			satOK(t, e)
			satOK(t, life.pushClause(true, c))
			var output bytes.Buffer
			tags := diagnostic.NewTraceTagsWithWriter(&output)
			tags.Enable("sat_missed_prop")
			life.traceTags = tags
			satOK(t, life.propagation.variables.setAssignment(0, lbool.False))
			satOK(t, life.propagation.variables.setAssignment(2, lbool.False))
			if mode != "unit" {
				satOK(t, life.propagation.variables.setAssignment(4, lbool.False))
			}
			if mode == "frozen" {
				satOK(t, life.propagation.clauses.Freeze(c))
			}
			e = life.checkMissedPropagation()
			if mode == "frozen" {
				satOK(t, e)
			} else if !errors.Is(e, ErrSATAssertion) {
				t.Fatal(e)
			}
			want := ""
			if mode != "frozen" && tags.ShouldTrace("sat_missed_prop") {
				value := "l_undef"
				if mode == "empty" {
					value = "l_false"
				}
				want = satClauseTraceRecord("sat_missed_prop", "check_missed_propagation", "3061",
					"missed_propagation: (0 1 2)+*\n0: l_false\n1: l_false\n2: "+value+"\n")
			}
			if got := output.String(); got != want {
				t.Fatalf("trace = %q, want %q", got, want)
			}
		})
	}
}
