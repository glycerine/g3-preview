package sat

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/lbool"
)

// Source: src/sat/sat_solver.cpp:1272-1274. The converter display precedes
// application, and the model listing observes its completed assignment.
func TestSATModelConstructionSourceTraceConverterThenValues(t *testing.T) {
	for _, converted := range []bool{false, true} {
		name := "assigned"
		if converted {
			name = "converted"
		}
		t.Run(name, func(t *testing.T) {
			s, _, _ := satSolverCheckFixture(t)
			_, e := s.MkVar(false, true)
			satOK(t, e)
			if converted {
				satOK(t, s.Variables.setEliminated(0, true))
				r, err := s.converter.Mk(satModelElimVar, 0)
				satOK(t, err)
				satOK(t, s.converter.InsertLiterals(r, []Literal{0}))
			} else {
				satOK(t, s.propagation.assign(0, satJustification{}))
			}
			var output bytes.Buffer
			tags := diagnostic.NewTraceTagsWithWriter(&output)
			tags.Enable("sat_mc_bug")
			tags.Enable("sat")
			s.traceTags = tags
			satOK(t, s.mkModel())
			value, e := s.model.Read(0)
			satOK(t, e)
			if lbool.LBool(int32(value)) != lbool.True {
				t.Fatal(value)
			}
			want := ""
			if tags.ShouldTrace("sat_mc_bug") {
				converter := "(sat::model-converter)\n"
				if converted {
					converter = "(sat::model-converter\n  (elim 0\n    (0)))\n"
				}
				want += satClauseTraceRecord("sat_mc_bug", "mk_model", "1272", converter)
			}
			if tags.ShouldTrace("sat") {
				want += satClauseTraceRecord("sat", "mk_model", "1274", "0: l_true\n")
			}
			if got := output.String(); got != want {
				t.Fatalf("model trace = %q, want %q", got, want)
			}
		})
	}
}
