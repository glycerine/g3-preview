// Source: src/sat/sat_solver.cpp:1289-1346,2852-3017; src/sat/sat_types.h.
package sat

import (
	"github.com/glycerine/g3/pkg/util/lbool"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func satInspectionStream(t *testing.T, b *arena.Budget) *utext.NumericTextStream {
	t.Helper()
	o, e := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: b})
	satOK(t, e)
	t.Cleanup(func() { b.FailAfter(-1); satOK(t, o.Close()) })
	return o
}
func satInspectionText(t *testing.T, o *utext.NumericTextStream, want string) {
	t.Helper()
	got, e := o.String()
	satOK(t, e)
	if got != want {
		t.Fatalf("source output\ngot %q\nwant %q", got, want)
	}
}
func satInspectionFixture(t *testing.T) *Solver {
	s, _, _ := satSolverCheckFixture(t)
	for i := 0; i < 4; i++ {
		_, e := s.MkVar(false, true)
		satOK(t, e)
	}
	_, e := s.Lifecycle.MkClauseCore(2, []Literal{0, 2}, false)
	satOK(t, e)
	_, e = s.Lifecycle.MkClauseCore(3, []Literal{1, 4, 6}, false)
	satOK(t, e)
	satOK(t, s.propagation.assign(3, satJustification{}))
	return s
}
func TestSATSolverInspectionExactSourceDisplays(t *testing.T) {
	s := satInspectionFixture(t)
	o := satInspectionStream(t, arena.NewBudget(math.MaxUint64))
	n, e := s.numClauses()
	satOK(t, e)
	if n != 3 {
		t.Fatal(n)
	}
	satOK(t, s.displayDIMACS(o))
	satInspectionText(t, o, "p cnf 4 3\n-2 0\n1 2 0\n-1 3 4 0\n")
	satOK(t, o.ResetText())
	satOK(t, s.displayInto(o))
	satInspectionText(t, o, "(sat\n-1 \n(0 1)\n(-0 2 3)+\n)\n")
	satOK(t, o.ResetText())
	satOK(t, s.displayWCNF(o, 2, []Literal{1, 4}, []uint32{2, 7}))
	satInspectionText(t, o, "p wcnf 4 5 8\nc soft 2\n8 -2 0\n8 1 2 0\n8 -1 3 4 0\n2 -0 0\n7 2 0\n")
	satOK(t, o.ResetText())
	satOK(t, s.displayAssignment(o))
	satInspectionText(t, o, "-1\n")
	satOK(t, o.ResetText())
	satOK(t, s.displayJustification(o, newSATBinaryJustification(3)))
	satInspectionText(t, o, "binary -1")
	satFullSolverRuntimeAudit(t, s)
}
func TestSATSolverInspectionExternalModelIndependentTruthTable(t *testing.T) {
	s := satInspectionFixture(t)
	for bits := uint32(0); bits < 16; bits++ {
		m := make(Model, 4)
		for v := range m {
			m[v] = lbool.False
			if bits&(1<<uint(v)) != 0 {
				m[v] = lbool.True
			}
		}
		want := (bits&1 != 0 || bits&2 != 0) && (bits&1 == 0 || bits&4 != 0 || bits&8 != 0)
		got, e := s.checkModelSpan(satSolverModelSpan{slice: m})
		satOK(t, e)
		if got != want {
			t.Fatal(bits, got, want)
		}
	}
}
func TestSATSolverInspectionWCNFUnsignedWrapAndPartialSoftSpan(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	_, e := s.MkVar(false, true)
	satOK(t, e)
	o := satInspectionStream(t, arena.NewBudget(math.MaxUint64))
	satOK(t, s.displayWCNF(o, 1, []Literal{1}, []uint32{math.MaxUint32}))
	satInspectionText(t, o, "p wcnf 1 1 0\nc soft 1\n4294967295 -0 0\n")
}
