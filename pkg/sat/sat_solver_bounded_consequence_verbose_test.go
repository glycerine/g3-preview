package sat

// Source: src/sat/sat_solver.cpp:3301-3362; src/util/util.h:176-186.

import (
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSATSolverBoundedConsequencesSourceAbortVerbose(t *testing.T) {
	s, c, p := satSolverCheckFixture(t)
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
	satOK(t, diagnostic.SetProcessVerbosityLevel(10))
	for _, setting := range []struct {
		name  string
		value uint32
	}{
		{"restart.initial", 0},
		{"restart.max", 0},
		{"max_conflicts", 100},
	} {
		satOK(t, c.SetUIntName(&p, &setting.name, setting.value))
	}
	satOK(t, s.UpdateParams(&p))
	s.nextSimplify = ^uint32(0)
	for i := 0; i < 2; i++ {
		_, err = s.MkVar(false, true)
		satOK(t, err)
	}
	for i := 0; i < 4; i++ {
		satOK(t, s.MkClause([]Literal{MakeLiteral(0, i&1 != 0), MakeLiteral(1, i&2 != 0)}))
	}
	rows, err := newSATSolverLiteralRows(s.runtime.memory, s.budget)
	satOK(t, err)
	t.Cleanup(func() { satOK(t, rows.CloseChecked()) })
	result, err := s.getConsequences(nil, []BoolVar{0, 1}, rows)
	satOK(t, err)
	got, err := out.String()
	satOK(t, err)
	if result != lbool.Undef || !strings.Contains(got, "(sat \"abort: max-restarts\")\n") {
		t.Fatal("source bounded-consequences restart exit", result, got)
	}
}
