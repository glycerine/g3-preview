package tactic

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

type expr2varTraceObserver struct {
	bytes.Buffer
	translator *GoalTranslator
	key        ast.ASTHandle
	beforePin  bool
}

func (w *expr2varTraceObserver) Write(p []byte) (int, error) {
	if strings.Contains(string(p), "interpreted:\n") {
		_, found := w.translator.Atoms.Get2(w.key)
		w.beforePin = !found
	}
	return w.Buffer.Write(p)
}

func (w *expr2varTraceObserver) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

// Source: expr2var.cpp:27-34 and goal2sat.cpp:124-136. An interpreted
// equality reaches the trace before expr2var publishes its retained key.
func TestSATGoalInterpretedAtomExpr2VarTrace(t *testing.T) {
	m := astTestManager(t)
	goalTraceBindParams(t, m)
	sort, err := m.MkUninterpretedSort(astTestSymbol(t, m, "U"), nil)
	astTestMust(t, err)
	astTestPin(t, m, sort)
	x, err := m.MkConst(astTestSymbol(t, m, "x"), sort)
	astTestMust(t, err)
	astTestPin(t, m, x)
	y, err := m.MkConst(astTestSymbol(t, m, "y"), sort)
	astTestMust(t, err)
	astTestPin(t, m, y)
	eq, err := m.MkEq(x, y)
	astTestMust(t, err)
	astTestPin(t, m, eq)
	s, _, _ := satSolverCheckFixture(t)
	tr, err := NewGoalTranslator(m, s)
	satOK(t, err)
	t.Cleanup(func() { satOK(t, tr.Close()) })
	output := &expr2varTraceObserver{translator: tr, key: eq}
	tags := diagnostic.NewTraceTagsWithWriter(output)
	tags.Enable("expr2var")
	tr.traceTags = tags
	satOK(t, tr.Compile([]ast.ASTHandle{eq}))
	interpreted, err := tr.HasInterpretedAtoms()
	satOK(t, err)
	if !interpreted {
		t.Fatal("equality was not retained as an interpreted atom")
	}
	if !tags.ShouldTrace("expr2var") {
		if output.Len() != 0 {
			t.Fatal("erased expr2var trace", output.String())
		}
		return
	}
	want := "-------- [expr2var] insert src/ast/expr2var.cpp:29 ---------\n" +
		"interpreted:\n(= x y)\n------------------------------------------------\n"
	if got := output.String(); got != want || !output.beforePin {
		t.Fatalf("interpreted atom trace: got %q, want %q", got, want)
	}
}
