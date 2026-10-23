package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func smtQuantifierManagerFixture(t *testing.T, eager, lazy float32, config SMTQuantifierManagerConfig) (*ast.ASTManager, *SMTInternalizerState, *SMTQuantifierRegistry, *SMTQuantifierManager) {
	t.Helper()
	m := astTestManager(t)
	costs, err := NewSMTQICostModel(m, "0", "0", nil)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, costs.Close()) })
	registry, err := NewSMTQuantifierRegistry(m, NewSMTQIQueueSchedule(eager, lazy, false), costs, 100)
	astTestMust(t, err)
	context, err := NewSMTCheckerContext(m)
	astTestMust(t, err)
	state, err := NewSMTInternalizerState(m, context, registry)
	astTestMust(t, err)
	manager, err := NewSMTQuantifierManager(state, smtQuickTheoryRewriter(t, m), config)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, state.Close()) })
	return m, state, registry, manager
}

func smtManagedQuantifier(t *testing.T, m *ast.ASTManager, name string) (ast.ASTHandle, ast.ASTHandle, ast.ASTHandle) {
	t.Helper()
	sort, err := m.MkSort(astTestSymbol(t, m, name+"-sort"), ast.ASTSortInfo{})
	astTestMust(t, err)
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	predicate := smtQuickPredicate(t, m, name+"-predicate", sort, boolean)
	variable, err := m.MkVar(0, sort)
	astTestMust(t, err)
	body, err := m.MkApp(predicate, []ast.ASTHandle{variable})
	astTestMust(t, err)
	quantifier, err := m.MkQuantifier(ast.ASTQuantifierSpec{
		Forall: true, Sorts: []ast.ASTHandle{sort}, Names: []symbol.Symbol{astTestSymbol(t, m, name+"-x")}, Body: body,
	})
	astTestMust(t, err)
	return quantifier, predicate, sort
}

func TestSMTQuantifierManagerPropagatesAndScopesQueuedInstances(t *testing.T) {
	m, state, registry, manager := smtQuantifierManagerFixture(t, 10, 20, SMTQuantifierManagerConfig{})
	quantifier, _, sort := smtManagedQuantifier(t, m, "managed-queue")
	astTestMust(t, state.Internalize(quantifier, true))
	owner, binding := smtEnodeConstant(t, m, 2500, sort, 3)
	astTestMust(t, state.context.RegisterEnode(binding, true))
	inserted, err := registry.AddInstance(quantifier, ast.ASTHandle{}, []*SMTEnode{binding}, 3, 0, 0)
	astTestMust(t, err)
	if !inserted || !manager.CanPropagate() {
		t.Fatal("queued instance was not exposed as propagation work")
	}
	astTestMust(t, manager.Propagate())
	if manager.CanPropagate() || len(state.Clauses()) == 0 || registry.schedule.Stats().Instances != 1 {
		t.Fatal("queued instance was not materialized", manager.CanPropagate(), len(state.Clauses()), registry.schedule.Stats())
	}
	astTestMust(t, state.PushScope())
	astTestMust(t, state.PopScopes(1))
	if registry.scopeLevel != 0 {
		t.Fatal("quantifier registry scope was not synchronized", registry.scopeLevel)
	}
	_ = owner
}

func TestSMTQuantifierManagerFinalCheckRunsActiveQuickChecker(t *testing.T) {
	m, state, registry, manager := smtQuantifierManagerFixture(t, 10, 20, SMTQuantifierManagerConfig{QuickUnsat: true})
	quantifier, predicate, sort := smtManagedQuantifier(t, m, "managed-quick")
	astTestMust(t, state.Internalize(quantifier, true))
	_, binding := smtEnodeConstant(t, m, 2510, sort, 1)
	astTestMust(t, state.context.RegisterEnode(binding, true))
	application, _ := smtQuickApplicationNode(t, m, state.context, predicate, binding, true)
	astTestMust(t, state.context.SetRelevant(quantifier, true))
	astTestMust(t, state.context.SetAssignment(quantifier, lbool.True))
	astTestMust(t, state.context.SetAssignment(application, lbool.False))
	status, err := manager.FinalCheck(true)
	astTestMust(t, err)
	if status != SMTFinalCheckContinue || registry.schedule.Stats().Instances != 1 || len(state.Clauses()) == 0 {
		t.Fatal("quick checker did not force and internalize a conflicting instance", status, registry.schedule.Stats(), len(state.Clauses()))
	}
	status, err = manager.FinalCheck(true)
	astTestMust(t, err)
	if status != SMTFinalCheckDone {
		t.Fatal("duplicate quick-check instance prevented completion", status)
	}
}

func TestSMTQuantifierManagerFiltersMBQIByQuantifierIDPrefix(t *testing.T) {
	m, _, _, manager := smtQuantifierManagerFixture(t, 10, 20, SMTQuantifierManagerConfig{
		MBQI: true, MaxCexs: 1, MaxIterations: 10, MBQIID: "selected.",
	})
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	variable, err := m.MkVar(0, boolean)
	astTestMust(t, err)
	makeQuantifier := func(name string, id symbol.Symbol) ast.ASTHandle {
		predicate := smtQuickPredicate(t, m, "mbqi-prefix-"+name, boolean, boolean)
		body, makeErr := m.MkApp(predicate, []ast.ASTHandle{variable})
		astTestMust(t, makeErr)
		value, makeErr := m.MkQuantifier(ast.ASTQuantifierSpec{
			Forall: true, Sorts: []ast.ASTHandle{boolean}, Names: []symbol.Symbol{astTestSymbol(t, m, "mbqi-prefix-x")},
			Body: body, QID: id,
		})
		astTestMust(t, makeErr)
		return value
	}
	selected := makeQuantifier("selected", astTestSymbol(t, m, "selected.quantifier"))
	rejected := makeQuantifier("rejected", astTestSymbol(t, m, "other.quantifier"))
	numerical := makeQuantifier("numerical", symbol.Numerical(7))
	unnamed := makeQuantifier("unnamed", symbol.Symbol{})
	for _, test := range []struct {
		name       string
		quantifier ast.ASTHandle
		want       bool
	}{
		{"selected", selected, true}, {"rejected", rejected, false}, {"numerical", numerical, false}, {"unnamed", unnamed, false},
	} {
		got, err := manager.mbqiEnabled(test.quantifier)
		astTestMust(t, err)
		if got != test.want {
			t.Fatal(test.name, "MBQI id-prefix classification", got, "want", test.want)
		}
	}
}

func TestSMTQuantifierManagerEMatchingFeedsInstanceQueue(t *testing.T) {
	m, state, registry, manager := smtQuantifierManagerFixture(t, 10, 20, SMTQuantifierManagerConfig{Ematching: true})
	sort, err := m.MkSort(astTestSymbol(t, m, "managed-ematch-sort"), ast.ASTSortInfo{})
	astTestMust(t, err)
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	predicate := smtQuickPredicate(t, m, "managed-ematch-p", sort, boolean)
	variable, err := m.MkVar(0, sort)
	astTestMust(t, err)
	body, err := m.MkApp(predicate, []ast.ASTHandle{variable})
	astTestMust(t, err)
	pattern, err := m.MkPattern([]ast.ASTHandle{body})
	astTestMust(t, err)
	quantifier, err := m.MkQuantifier(ast.ASTQuantifierSpec{
		Forall: true, Sorts: []ast.ASTHandle{sort}, Names: []symbol.Symbol{astTestSymbol(t, m, "managed-ematch-x")}, Body: body, Patterns: []ast.ASTHandle{pattern},
	})
	astTestMust(t, err)
	astTestMust(t, state.Internalize(quantifier, true))
	_, binding := smtEnodeConstant(t, m, 2520, sort, 2)
	astTestMust(t, state.context.RegisterEnode(binding, true))
	application, _ := smtQuickApplicationNode(t, m, state.context, predicate, binding, true)
	astTestMust(t, state.context.SetRelevant(quantifier, true))
	astTestMust(t, state.context.SetAssignment(quantifier, lbool.True))
	astTestMust(t, state.context.SetAssignment(application, lbool.False))
	if !manager.CanPropagate() {
		t.Fatal("active patterned quantifier did not expose matching work")
	}
	astTestMust(t, manager.Propagate())
	if registry.schedule.Stats().Instances != 1 || len(state.Clauses()) == 0 {
		t.Fatal("E-matching did not materialize its instance", registry.schedule.Stats(), len(state.Clauses()))
	}
}
