package tactic

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestSATGoalDependencyFormulaSourceTrace(t *testing.T) {
	m := astTestManager(t)
	goalTraceBindParams(t, m)
	s, _, _ := satSolverCheckFixture(t)
	tr, err := NewGoalTranslator(m, s)
	satOK(t, err)
	t.Cleanup(func() { satOK(t, tr.Close()) })
	p := goalTestAtom(t, m, "self-label")
	q := goalTestAtom(t, m, "retained")
	leaf, err := m.DependencyLeaf(p)
	satOK(t, err)
	g := goalTestNew(t, m, false, true)
	satOK(t, g.AssertWithProof(p, ast.ASTHandle{}, leaf))
	satOK(t, g.Assert(q))
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tr.traceTags = tags
	satOK(t, tr.traceGoalCompiledFormula(q))
	if output.Len() != 0 {
		t.Fatalf("disabled goal formula trace: %q", output.String())
	}
	tags.Enable("sat")
	satOK(t, tr.CompileGoal(g))
	if !tags.ShouldTrace("sat") {
		if output.Len() != 0 {
			t.Fatalf("erased goal formula trace: %q", output.String())
		}
		return
	}
	want := "-------- [sat] operator() src/sat/tactic/goal2sat.cpp:480 ---------\nretained\n------------------------------------------------\n"
	if got := output.String(); got != want {
		t.Fatalf("source goal formula after self-label skip: got %q, want %q", got, want)
	}
}
