package tactic

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/sat"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

func TestSATGoalTranslatorResultStackSourceTrace(t *testing.T) {
	m := astTestManager(t)
	budget := m.BudgetField
	context, err := uparams.NewParamsContext(m.Symbols, m.NumeralService(), uparams.ParamsConfig{Budget: budget})
	astTestMust(t, err)
	catalog, err := uparams.NewGlobalParamsCatalog(m.Symbols, uparams.GlobalParamsCatalogConfig{Budget: budget})
	astTestMust(t, err)
	global, err := uparams.NewGlobalParamsService(m.Symbols, context, uparams.GlobalParamsConfig{Budget: budget, Catalog: catalog})
	astTestMust(t, err)
	astTestMust(t, global.InitializeSource())
	previous, err := uparams.BindProcessGlobalParamsService(global)
	astTestMust(t, err)
	t.Cleanup(func() {
		_, e := uparams.BindProcessGlobalParamsService(previous)
		astTestMust(t, e)
		astTestMust(t, global.Close())
		astTestMust(t, catalog.Close())
		astTestMust(t, context.Close())
	})
	sort, err := m.MkBoolSort()
	astTestMust(t, err)
	astTestPin(t, m, sort)
	p, err := m.MkConst(astTestSymbol(t, m, "p"), sort)
	astTestMust(t, err)
	astTestPin(t, m, p)
	q, err := m.MkConst(astTestSymbol(t, m, "q"), sort)
	astTestMust(t, err)
	astTestPin(t, m, q)
	formula, err := m.MkFamilyApp(0, ast.ASTOpAnd, nil, []ast.ASTHandle{p, q})
	astTestMust(t, err)
	astTestPin(t, m, formula)
	s, _, _ := satSolverCheckFixture(t)
	tr, err := NewGoalTranslator(m, s)
	satOK(t, err)
	t.Cleanup(func() { satOK(t, tr.Close()) })
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tr.traceTags = tags
	if err = tr.traceGoalResultStack(396, "result stack\n", satGoalFrame{term: formula, root: true}); err != nil || output.Len() != 0 {
		t.Fatalf("disabled goal trace: %v %q", err, output.String())
	}
	tags.Enable("goal2sat_bug")
	tags.Enable("goal2sat")
	satOK(t, tr.Compile([]ast.ASTHandle{formula}))
	if !tags.ShouldTrace("goal2sat_bug") {
		if output.Len() != 0 {
			t.Fatalf("erased goal trace: %q", output.String())
		}
		return
	}
	pvar, err := tr.Variable(p)
	satOK(t, err)
	qvar, err := tr.Variable(q)
	satOK(t, err)
	want := "-------- [goal2sat] process src/sat/tactic/goal2sat.cpp:380 ---------\n" +
		"converting: (and p q)\n------------------------------------------------\n" +
		"-------- [goal2sat_bug] process src/sat/tactic/goal2sat.cpp:396 ---------\n" +
		"result stack\n(and p q) root: 1 sign: 0\n\n------------------------------------------------\n" +
		"-------- [goal2sat_bug] process src/sat/tactic/goal2sat.cpp:415 ---------\n" +
		"converting\n(and p q) root: 1 sign: 0\n" + sat.MakeLiteral(pvar, false).String() + " " + sat.MakeLiteral(qvar, false).String() + "\n------------------------------------------------\n" +
		"-------- [goal2sat] mk_clause src/sat/tactic/goal2sat.cpp:88 ---------\nmk_clause: " + sat.MakeLiteral(pvar, false).String() + "\n------------------------------------------------\n" +
		"-------- [goal2sat] mk_clause src/sat/tactic/goal2sat.cpp:88 ---------\nmk_clause: " + sat.MakeLiteral(qvar, false).String() + "\n------------------------------------------------\n"
	if got := output.String(); got != want {
		t.Fatalf("source frame records: got %q, want %q", got, want)
	}
	output.Reset()
	or, err := m.MkFamilyApp(0, ast.ASTOpOr, nil, []ast.ASTHandle{p, q})
	astTestMust(t, err)
	astTestPin(t, m, or)
	satOK(t, tr.Compile([]ast.ASTHandle{or}))
	want = "-------- [goal2sat] process src/sat/tactic/goal2sat.cpp:380 ---------\n" +
		"converting: (or p q)\n------------------------------------------------\n" +
		"-------- [goal2sat_bug] process src/sat/tactic/goal2sat.cpp:396 ---------\n" +
		"result stack\n(or p q) root: 1 sign: 0\n\n------------------------------------------------\n" +
		"-------- [goal2sat_bug] process src/sat/tactic/goal2sat.cpp:415 ---------\n" +
		"converting\n(or p q) root: 1 sign: 0\n" + sat.MakeLiteral(pvar, false).String() + " " + sat.MakeLiteral(qvar, false).String() + "\n------------------------------------------------\n" +
		"-------- [goal2sat] mk_clause src/sat/tactic/goal2sat.cpp:103 ---------\nmk_clause: " + sat.MakeLiteral(pvar, false).String() + " " + sat.MakeLiteral(qvar, false).String() + " \n------------------------------------------------\n"
	if got := output.String(); got != want {
		t.Fatalf("source array clause record: got %q, want %q", got, want)
	}
	output.Reset()
	tr.results = append(tr.results, sat.MakeLiteral(pvar, false))
	satOK(t, tr.traceGoalRemaining())
	tr.results = tr.results[:0]
	want = "-------- [goal2sat] process src/sat/tactic/goal2sat.cpp:421 ---------\n" +
		sat.MakeLiteral(pvar, false).String() + "\n------------------------------------------------\n"
	if got := output.String(); got != want {
		t.Fatalf("conditional residual record: got %q, want %q", got, want)
	}
}
