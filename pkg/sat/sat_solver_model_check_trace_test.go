package sat

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/lbool"
)

// Source: src/sat/sat_solver.cpp:1299-1342. All four failed validation
// branches expose their own source TRACE record and leave the model invalid.
func TestSATModelValidationSourceTraceFailures(t *testing.T) {
	for _, name := range []string{"clause", "binary", "assumption", "converter"} {
		t.Run(name, func(t *testing.T) {
			s, _, _ := satSolverCheckFixture(t)
			n := 1
			if name == "binary" {
				n = 2
			}
			if name == "clause" {
				n = 3
			}
			for i := 0; i < n; i++ {
				_, err := s.MkVar(false, true)
				satOK(t, err)
			}
			model := make(Model, n)
			for i := range model {
				model[i] = lbool.False
			}
			switch name {
			case "clause":
				_, err := s.Lifecycle.MkClauseCore(3, []Literal{0, 2, 4}, true)
				satOK(t, err)
			case "binary":
				_, err := s.Lifecycle.MkClauseCore(2, []Literal{0, 2}, true)
				satOK(t, err)
			case "assumption":
				satOK(t, s.conflict.assumptions.Push(0))
			case "converter":
				r, err := s.converter.Mk(satModelElimVar, 0)
				satOK(t, err)
				satOK(t, s.converter.InsertLiterals(r, []Literal{0}))
			}
			var output bytes.Buffer
			tags := diagnostic.NewTraceTagsWithWriter(&output)
			tags.Enable("sat")
			s.traceTags = tags
			good, err := s.checkModelSpan(satSolverModelSpan{slice: model})
			satOK(t, err)
			if good {
				t.Fatal("invalid model accepted")
			}
			if !tags.ShouldTrace("sat") {
				if output.Len() != 0 {
					t.Fatalf("erased trace = %q", output.String())
				}
				return
			}
			record := func(line, body string) string {
				return satClauseTraceRecord("sat", "check_model", line, body)
			}
			var want string
			switch name {
			case "clause":
				want = record("1299", "failed: (0 1 2)+*\nassumptions: \ntrail: \nmodel: -0 -1 -2\n(sat::model-converter)\n")
			case "binary":
				want = record("1322", "failed binary: 0 1 learned: 1\n(sat::model-converter)\n") +
					record("1322", "failed binary: 1 0 learned: 1\n(sat::model-converter)\n")
			case "assumption":
				want = record("1331", "0 does not model check\ntrail: \nmodel: -0\n(sat::model-converter)\n")
			case "converter":
				want = record("1342", "model: -0\n(sat::model-converter\n  (elim 0\n    (0)))\n")
			}
			if got := output.String(); got != want {
				t.Fatalf("%s source records = %q, want %q", name, got, want)
			}
		})
	}
}
