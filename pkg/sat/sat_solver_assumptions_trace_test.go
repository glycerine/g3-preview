package sat

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/lbool"
)

type satInitAssumptionsTraceObserver struct {
	output bytes.Buffer
	solver *Solver
	before bool
}

func (w *satInitAssumptionsTraceObserver) Write(p []byte) (int, error) {
	if bytes.Equal(p, []byte("1\n")) {
		value, err := w.solver.Variables.valueLiteral(2)
		w.before = err == nil && value == lbool.Undef
	}
	return w.output.Write(p)
}

// Source: src/sat/sat_solver.cpp:1080-1086. The record follows initial
// propagation but precedes user and explicit assumption assignments.
func TestSATInitAssumptionsSourceTraceBeforeAssignments(t *testing.T) {
	for _, name := range []string{"plain", "user", "converter"} {
		t.Run(name, func(t *testing.T) {
			s, _, _ := satSolverCheckFixture(t)
			for i := 0; i < 2; i++ {
				_, err := s.MkVar(true, true)
				satOK(t, err)
			}
			if name == "user" {
				satOK(t, s.conflict.userScopeLiterals.Push(0))
			}
			if name == "converter" {
				r, err := s.converter.Mk(satModelElimVar, 0)
				satOK(t, err)
				satOK(t, s.converter.InsertBinary(r, 1, 3))
			}
			w := &satInitAssumptionsTraceObserver{solver: s}
			tags := diagnostic.NewTraceTagsWithWriter(w)
			tags.Enable("sat")
			s.traceTags = tags
			satOK(t, s.initAssumptions([]Literal{2}))
			value, e := s.Variables.valueLiteral(2)
			satOK(t, e)
			if value != lbool.True {
				t.Fatal("assumption not assigned", value)
			}
			want := ""
			if tags.ShouldTrace("sat") {
				body := "1\n(sat::model-converter)\n"
				if name == "user" {
					body = "1\nuser literals: 0\n(sat::model-converter)\n"
				} else if name == "converter" {
					body = "1\n(sat::model-converter\n  (elim 0\n    (-0 -1)))\n"
				}
				want = satClauseTraceRecord("sat", "init_assumptions", "1080", body)
				if !w.before {
					t.Fatal("trace followed assumption assignment")
				}
			}
			if got := w.output.String(); got != want {
				t.Fatalf("init trace = %q, want %q", got, want)
			}
		})
	}
}
