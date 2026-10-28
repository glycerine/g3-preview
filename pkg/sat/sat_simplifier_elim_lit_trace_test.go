package sat

// Source: src/sat/sat_simplifier.cpp:643-680.

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func satElimLitTraceRecord(line, body string) string {
	return "-------- [elim_lit] elim_lit src/sat/sat_simplifier.cpp:" + line + " ---------\n" +
		body + "------------------------------------------------\n"
}

func TestSATSimplifierEliminateLiteralSourceTraceBranches(t *testing.T) {
	for _, tc := range []struct {
		name       string
		literals   []Literal
		assign     []lbool.LBool
		wantLine   string
		wantBody   string
		isConflict bool
	}{
		{"satisfied", []Literal{0, 2, 4}, []lbool.LBool{lbool.Undef, lbool.True}, "654", "clause was satisfied\n", false},
		{"empty", []Literal{0, 2, 4}, []lbool.LBool{lbool.Undef, lbool.False, lbool.False}, "660", "clause is empty\n", true},
		{"unit", []Literal{0, 2, 4}, []lbool.LBool{lbool.Undef, lbool.False}, "664", "clause became unit: 2\n", false},
		{"binary", []Literal{0, 2, 4}, nil, "670", "clause became binary: 1 2\n", false},
		{"nary", []Literal{0, 2, 4, 6}, nil, "676", "result: (1 2 3)+\n", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, lifecycle := satSimplifierFixture(t, 4)
			p := lifecycle.propagation
			satOK(t, s.useList.Init(4))
			satOK(t, s.kernel.InitVisited(4))
			c, err := p.clauses.MkClause(tc.literals, false)
			satOK(t, err)
			satOK(t, lifecycle.pushClause(false, c))
			satOK(t, s.useList.Insert(c))
			for i, value := range tc.assign {
				if value != lbool.Undef {
					satOK(t, p.variables.setAssignment(Literal(uint32(i)*2), value))
				}
			}
			var output bytes.Buffer
			tags := diagnostic.NewTraceTagsWithWriter(&output)
			tags.Enable("elim_lit")
			lifecycle.traceTags = tags
			satOK(t, s.eliminateLiteral(c, 0))
			if p.inconsistent != tc.isConflict {
				t.Fatal("source branch conflict", p.inconsistent, tc.isConflict)
			}
			if !tags.ShouldTrace("elim_lit") {
				if output.Len() != 0 {
					t.Fatal("build-erased source trace", output.String())
				}
				return
			}
			first := "processing: (0 1 2)+\n"
			if len(tc.literals) == 4 {
				first = "processing: (0 1 2 3)+\n"
			}
			want := satElimLitTraceRecord("644", first) + satElimLitTraceRecord(tc.wantLine, tc.wantBody)
			if got := output.String(); got != want {
				t.Fatalf("source %s trace: got %q, want %q", tc.name, got, want)
			}
		})
	}
	// The C++ TRACE macro does not evaluate its body when the tag is disabled.
	s, lifecycle := satSimplifierFixture(t, 4)
	satOK(t, s.useList.Init(4))
	c, err := lifecycle.propagation.clauses.MkClause([]Literal{0, 2, 4}, false)
	satOK(t, err)
	satOK(t, lifecycle.pushClause(false, c))
	satOK(t, s.useList.Insert(c))
	var output bytes.Buffer
	lifecycle.traceTags = diagnostic.NewTraceTagsWithWriter(&output)
	satOK(t, s.eliminateLiteral(c, 0))
	if output.Len() != 0 {
		t.Fatal("disabled trace emitted", output.String())
	}
}
