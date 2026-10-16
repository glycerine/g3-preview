// Source: src/sat/sat_solver.cpp:2071-2084.
package sat

import (
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSATSolverActualMUSCoreSourceVerboseReport(t *testing.T) {
	for _, level := range []uint32{1, 2} {
		t.Run(strconv.FormatUint(uint64(level), 10), func(t *testing.T) {
			s, ctx, params := satSolverCheckFixture(t)
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
				t.Fatal("expected assumption conflict", result)
			}
			coreSize, err := s.conflict.core.Size()
			satOK(t, err)
			if coreSize != 3 {
				t.Fatal("expected three necessary assumptions", coreSize)
			}
			core := make([]Literal, coreSize)
			for i := range core {
				value, readErr := s.conflict.core.Read(uint32(i))
				satOK(t, readErr)
				core[i] = Literal(value)
			}
			got, err := out.String()
			satOK(t, err)
			want := "(sat.core: " + satLiteralsString(core) + ")\n"
			if (level == 2) != strings.Contains(got, want) || strings.Count(got, "(sat.core: ") > 1 {
				t.Fatal("source final core report gate or value", level, got, want)
			}
		})
	}
}
