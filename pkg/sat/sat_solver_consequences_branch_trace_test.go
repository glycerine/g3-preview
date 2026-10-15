package sat

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/lbool"
)

// Source: src/sat/sat_solver.cpp:3420,3450. Call the retained literal
// consequence search itself: the public variable route first runs a complete
// SAT check and returns before either conditional source branch on these CNFs.
func TestSATConsequenceConditionalTraceReachedBySearch(t *testing.T) {
	for _, tc := range []struct {
		name    string
		clauses [][]Literal
		line    string
		body    string
	}{
		{
			name:    "failed speculative literal",
			clauses: [][]Literal{{0, 2}, {1, 2}, {0, 3}, {1, 3}},
			line:    "3420",
		},
		{
			name:    "bounded search unsat",
			clauses: [][]Literal{{0, 2, 4}, {1, 2, 4}, {0, 3, 4}, {1, 3, 4}, {0, 2, 5}, {1, 2, 5}, {0, 3, 5}, {1, 3, 5}},
			line:    "3450",
			body:    "unsat\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _, _ := satSolverCheckFixture(t)
			for i := 0; i < 3; i++ {
				_, e := s.MkVar(true, true)
				satOK(t, e)
			}
			for _, clause := range tc.clauses {
				satOK(t, s.MkClause(clause))
			}
			out, e := newSATSolverLiteralRows(s.runtime.memory, s.budget)
			satOK(t, e)
			t.Cleanup(func() { satOK(t, out.CloseChecked()) })
			body := tc.body
			if tc.line == "3420" {
				var display bytes.Buffer
				satOK(t, s.traceUserDisplayInto(&display))
				body = "inconsistent\n" + display.String()
			}
			var output bytes.Buffer
			tags := diagnostic.NewTraceTagsWithWriter(&output)
			tags.Enable("sat")
			s.traceTags = tags
			result, e := s.getLiteralConsequences(satConsequenceLiterals(nil), satConsequenceLiterals([]Literal{0}), out)
			satOK(t, e)
			if result != lbool.True || s.propagation.inconsistent {
				t.Fatalf("search result %v, inconsistent %v", result, s.propagation.inconsistent)
			}
			if !tags.ShouldTrace("sat") {
				if output.Len() != 0 {
					t.Fatalf("erased TRACE emitted %q", output.String())
				}
				return
			}
			want := satClauseTraceRecord("sat", "get_consequences", tc.line, body)
			if n := strings.Count(output.String(), want); n != 1 {
				t.Fatalf("source record count %d, want 1 in %q", n, output.String())
			}
			other := "3450"
			if tc.line == "3450" {
				other = "3420"
			}
			if strings.Contains(output.String(), "get_consequences src/sat/sat_solver.cpp:"+other) {
				t.Fatalf("wrong conditional branch in %q", output.String())
			}
		})
	}
}
