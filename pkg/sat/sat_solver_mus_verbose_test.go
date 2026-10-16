// Source: src/sat/sat_mus.cpp:52-70,95-145.
package sat

import (
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSATSolverActualMUSSourceVerboseLevels(t *testing.T) {
	for _, level := range []uint32{0, 1, 3} {
		t.Run(strconv.FormatUint(uint64(level), 10), func(t *testing.T) {
			s, ctx, params := satSolverCheckFixture(t)
			out := satInspectionStream(t, arena.NewBudget(16384))
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
			for i := 0; i < 3; i++ {
				_, err = s.MkVar(true, true)
				satOK(t, err)
			}
			satOK(t, s.MkClause([]Literal{1, 3, 5}))
			key := "core.minimize"
			satOK(t, ctx.SetBoolName(&params, &key, true))
			satOK(t, s.UpdateParams(&params))
			result, err := s.Check([]Literal{0, 2, 4})
			satOK(t, err)
			if result != lbool.False {
				t.Fatal("expected three-assumption conflict", result)
			}
			got, err := out.String()
			satOK(t, err)
			if level == 0 && got != "" {
				t.Fatal("disabled MUS verbose output", got)
			}
			if level >= 1 && (!strings.Contains(got, "(sat.mus num-to-process: 3 mus: 0)\n") || !strings.Contains(got, "(sat.mus l_true)\n")) {
				t.Fatal("source MUS loop and nested-check notices", got)
			}
			if level == 1 && (strings.Contains(got, "(sat.mus size:") || strings.Contains(got, "(sat.mus.new ")) {
				t.Fatal("level-three MUS output leaked at level one", got)
			}
			if level == 3 && (!strings.Contains(got, "(sat.mus size: 3 core: [") || !strings.Contains(got, "(sat.mus.new ")) {
				t.Fatal("source MUS entry/final-core notices", got)
			}
		})
	}
}

func TestSATSolverActualMUSQXSourceFinalCoreVerbose(t *testing.T) {
	s, ctx, params := satSolverCheckFixture(t)
	out := satInspectionStream(t, arena.NewBudget(1<<20))
	oldLevel, err := diagnostic.ProcessVerbosityLevel()
	satOK(t, err)
	oldStream, err := diagnostic.BorrowProcessVerboseStream()
	satOK(t, err)
	t.Cleanup(func() {
		satOK(t, diagnostic.SetProcessVerboseStream(oldStream))
		satOK(t, diagnostic.SetProcessVerbosityLevel(oldLevel))
	})
	satOK(t, diagnostic.SetProcessVerboseStream(out))
	satOK(t, diagnostic.SetProcessVerbosityLevel(3))
	clause, assumptions := make([]Literal, 0, 65), make([]Literal, 0, 65)
	for i := 0; i < 65; i++ {
		v, err := s.MkVar(true, true)
		satOK(t, err)
		assumptions = append(assumptions, MakeLiteral(v, false))
		clause = append(clause, MakeLiteral(v, true))
	}
	satOK(t, s.MkClause(clause))
	key := "core.minimize"
	satOK(t, ctx.SetBoolName(&params, &key, true))
	satOK(t, s.UpdateParams(&params))
	result, err := s.Check(assumptions)
	satOK(t, err)
	if result != lbool.False {
		t.Fatal("expected 65-assumption conflict", result)
	}
	got, err := out.String()
	satOK(t, err)
	if !strings.Contains(got, "(sat.mus size: 65 core: [") || !strings.Contains(got, "(sat.mus.new ") || strings.Contains(got, "(sat.mus num-to-process:") {
		t.Fatal("source QX branch entry/final report", got)
	}
}
