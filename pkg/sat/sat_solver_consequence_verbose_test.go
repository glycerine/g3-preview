package sat

// Source: src/sat/sat_solver.cpp:3435-3465.

import (
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSATSolverActualConsequenceProgressSourceVerbose(t *testing.T) {
	for _, level := range []uint32{0, 1} {
		t.Run(strconv.FormatUint(uint64(level), 10), func(t *testing.T) {
			s, _, _ := satSolverCheckFixture(t)
			out := satInspectionStream(t, arena.NewBudget(8192))
			oldLevel, err := diagnostic.ProcessVerbosityLevel()
			satOK(t, err)
			oldStream, err := diagnostic.BorrowProcessVerboseStream()
			satOK(t, err)
			t.Cleanup(func() {
				satOK(t, diagnostic.SetProcessVerboseStream(oldStream))
				satOK(t, diagnostic.SetProcessVerbosityLevel(oldLevel))
			})
			satOK(t, diagnostic.SetProcessVerboseStream(out))
			satOK(t, diagnostic.SetProcessVerbosityLevel(level))
			for i := 0; i < 4; i++ {
				_, err = s.MkVar(true, true)
				satOK(t, err)
			}
			satOK(t, s.MkClause([]Literal{1, 2}))
			satOK(t, s.MkClause([]Literal{3, 4}))
			rows, err := newSATSolverLiteralRows(s.runtime.memory, s.budget)
			satOK(t, err)
			t.Cleanup(func() { satOK(t, rows.CloseChecked()) })
			result, err := s.getConsequences([]Literal{0}, []BoolVar{1, 2, 3}, rows)
			satOK(t, err)
			if result != lbool.True {
				t.Fatal("source consequence result", result)
			}
			got, err := out.String()
			satOK(t, err)
			if level == 0 && got != "" {
				t.Fatal("disabled progress output", got)
			}
			if level == 1 && !strings.Contains(got, "(sat.get-consequences iterations: 1 variables: 0 fixed: 2 status: l_true pre-assigned: 0 unfixed: 1)\n") {
				t.Fatal("source consequence progress fields", got)
			}
		})
	}
}
