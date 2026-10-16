package arith

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	goaltactic "github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/numeral"
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
	want := "-------- [propagate_ineqs_tactic] operator() src/tactic/arith/propagate_ineqs_tactic.cpp:492 ---------\n" +
		"(goal\n  (>= trace-x 2.0))\n" +
		"2 <= trace-x < oo\n" +
		"bound propagator:\n" +
		"2 | 2 <= x0 < oo\n" +
		"------------------------------------------------\n" +
		"-------- [propagate_ineqs_tactic] operator() src/tactic/arith/propagate_ineqs_tactic.cpp:509 ---------\n" +
		"after propagation:\n" +
		"2 <= trace-x < oo\n" +
		"2 | 2 <= x0 < oo\n" +
		"------------------------------------------------\n" +
		"-------- [propagate_ineqs_tactic] operator() src/tactic/arith/propagate_ineqs_tactic.cpp:510 ---------\n" +
		"(goal\n  (>= trace-x 2.0))\n" +
		"------------------------------------------------\n"
	if got := output.String(); got != want {
		t.Fatalf("restored-goal TRACE = %q, want %q", got, want)
	}
}

func TestInequalitySourceDisplayBoundsStrictAndUnbounded(t *testing.T) {
	m, symbols, _, _ := polynomialASTManager(t)
	owner, _ := equationManager(t)
	store, err := NewBoundEquationStore(owner)
	equationMust(t, err)
	t.Cleanup(func() { equationMust(t, store.Close()) })
	builder, err := NewPolynomialBuilder(m, store)
	equationMust(t, err)
	t.Cleanup(func() { equationMust(t, builder.Close()) })
	polynomialTraceBindParams(t, m)
	family, err := m.RegisterArithmeticPlugin()
	equationMust(t, err)
	sort, err := m.MkFamilySort(family, 0, nil)
	equationMust(t, err)
	for _, name := range []string{"bound-x", "bound-y"} {
		symbol, e := symbols.Intern(name)
		equationMust(t, e)
		term, e := m.MkConst(symbol, sort)
		equationMust(t, e)
		_, e = builder.mkVar(term)
		equationMust(t, e)
	}
	value := numeral.NewRational()
	equationMust(t, owner.SetInt32(&value, 3))
	_, err = store.AssertUpper(1, &value, true)
	equationMust(t, err)
	equationMust(t, owner.Del(&value))
	var out bytes.Buffer
	equationMust(t, builder.displayBounds(&out))
	if got, want := out.String(), "-oo < bound-x < oo\n-oo < bound-y < 3\n"; got != want {
		t.Fatalf("source bound display = %q, want %q", got, want)
	}
}
