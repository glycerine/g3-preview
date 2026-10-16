package arith

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	goaltactic "github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

func polynomialTraceBindParams(t *testing.T, m *ast.ASTManager) {
	t.Helper()
	b := m.BudgetField
	c, err := uparams.NewParamsContext(m.Symbols, m.NumeralService(), uparams.ParamsConfig{Budget: b})
	equationMust(t, err)
	catalog, err := uparams.NewGlobalParamsCatalog(m.Symbols, uparams.GlobalParamsCatalogConfig{Budget: b})
	equationMust(t, err)
	g, err := uparams.NewGlobalParamsService(m.Symbols, c, uparams.GlobalParamsConfig{Budget: b, Catalog: catalog})
	equationMust(t, err)
	equationMust(t, g.InitializeSource())
	previous, err := uparams.BindProcessGlobalParamsService(g)
	equationMust(t, err)
	t.Cleanup(func() {
		_, e := uparams.BindProcessGlobalParamsService(previous)
		equationMust(t, e)
		equationMust(t, g.Close())
		equationMust(t, catalog.Close())
		equationMust(t, c.Close())
	})
}

func TestInequalityRestoredGoalSourceTrace(t *testing.T) {
	m, symbols, numbers, _ := polynomialASTManager(t)
	owner, _ := equationManager(t)
	store, err := NewBoundEquationStore(owner)
	equationMust(t, err)
	t.Cleanup(func() { equationMust(t, store.Close()) })
	builder, err := NewPolynomialBuilder(m, store)
	equationMust(t, err)
	t.Cleanup(func() { equationMust(t, builder.Close()) })
	polynomialTraceBindParams(t, m)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("propagate_ineqs_tactic")
	equationMust(t, store.BindTrace(tags))
	family, err := m.RegisterArithmeticPlugin()
	equationMust(t, err)
	sort, err := m.MkFamilySort(family, 0, nil)
	equationMust(t, err)
	name, err := symbols.Intern("trace-x")
	equationMust(t, err)
	x, err := m.MkConst(name, sort)
	equationMust(t, err)
	v, err := numbers.NewValue()
	equationMust(t, err)
	equationMust(t, numbers.SetString(&v, "2"))
	two, err := m.ArithmeticNumeral(&v, false)
	equationMust(t, err)
	equationMust(t, numbers.Del(&v))
	lower, err := m.MkFamilyApp(family, 3, nil, []ast.ASTHandle{x, two})
	equationMust(t, err)
	input, err := goaltactic.NewGoal(m, false, false, false)
	equationMust(t, err)
	t.Cleanup(func() { equationMust(t, input.Close()) })
	equationMust(t, input.Assert(lower))
	result, err := builder.Apply(input)
	equationMust(t, err)
	if result == nil || result == input {
		t.Fatal("bound tactic did not return a restored goal")
	}
	t.Cleanup(func() { equationMust(t, result.Close()) })
	if !tags.ShouldTrace("propagate_ineqs_tactic") {
		if output.Len() != 0 {
			t.Fatalf("erased tactic TRACE = %q", output.String())
		}
		return
	}
	want := "-------- [propagate_ineqs_tactic] operator() src/tactic/arith/propagate_ineqs_tactic.cpp:510 ---------\n" +
		"(goal\n  (>= trace-x 2.0))\n" +
		"------------------------------------------------\n"
	if got := output.String(); got != want {
		t.Fatalf("restored-goal TRACE = %q, want %q", got, want)
	}
}
