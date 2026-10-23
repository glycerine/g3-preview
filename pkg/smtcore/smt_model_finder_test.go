package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestSMTModelFinderRegistrationRelevanceAndScopeLifecycle(t *testing.T) {
	m, state, _, manager := smtQuantifierManagerFixture(t, 10, 20, SMTQuantifierManagerConfig{})
	finder := manager.modelFinder
	if finder == nil {
		t.Fatal("quantifier manager omitted model finder")
	}
	first, _, _ := smtManagedQuantifier(t, m, "model-finder-first")
	second, _, _ := smtManagedQuantifier(t, m, "model-finder-second")
	astTestMust(t, state.Internalize(first, true))
	if len(finder.quantifiers) != 1 || finder.quantifiers[0] != first {
		t.Fatal("base quantifier was not registered", finder.quantifiers)
	}
	astTestMust(t, state.context.SetRelevant(first, true))
	astTestMust(t, state.context.SetAssignment(first, lbool.True))
	relevant, err := finder.relevantQuantifiers()
	astTestMust(t, err)
	if len(relevant) != 1 || relevant[0] != first {
		t.Fatal("active quantifier collection", relevant)
	}
	astTestMust(t, state.PushScope())
	astTestMust(t, state.Internalize(second, true))
	if len(finder.quantifiers) != 2 {
		t.Fatal("scoped quantifier was not registered", finder.quantifiers)
	}
	astTestMust(t, state.PopScopes(1))
	if len(finder.quantifiers) != 1 || finder.quantifiers[0] != first {
		t.Fatal("scoped quantifier was not restored", finder.quantifiers)
	}
	if _, found := finder.registered[second]; found {
		t.Fatal("popped quantifier remained registered")
	}
	astTestMust(t, manager.InitSearch())
}

func TestSMTModelFinderCollectsDirectPatternGroundArguments(t *testing.T) {
	m, state, _, manager := smtQuantifierManagerFixture(t, 10, 20, SMTQuantifierManagerConfig{})
	sort, err := m.MkSort(astTestSymbol(t, m, "model-finder-pattern-sort"), ast.ASTSortInfo{})
	astTestMust(t, err)
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	predicate := smtQuickPredicate(t, m, "model-finder-pattern-p", sort, boolean)
	variable, err := m.MkVar(0, sort)
	astTestMust(t, err)
	body, err := m.MkApp(predicate, []ast.ASTHandle{variable})
	astTestMust(t, err)
	pattern, err := m.MkPattern([]ast.ASTHandle{body})
	astTestMust(t, err)
	quantifier, err := m.MkQuantifier(ast.ASTQuantifierSpec{
		Forall: true, Sorts: []ast.ASTHandle{sort}, Names: []symbol.Symbol{astTestSymbol(t, m, "model-finder-pattern-x")},
		Body: body, Patterns: []ast.ASTHandle{pattern},
	})
	astTestMust(t, err)
	astTestMust(t, state.Internalize(quantifier, true))
	owner, binding := smtEnodeConstant(t, m, 9100, sort, 7)
	astTestMust(t, state.context.RegisterEnode(binding, true))
	_, _ = smtQuickApplicationNode(t, m, state.context, predicate, binding, true)
	astTestMust(t, manager.modelFinder.collectPatternInstantiationTerms(quantifier))
	set, err := manager.modelFinder.instantiationSet(quantifier, 0, false)
	astTestMust(t, err)
	if set == nil || set.elements[owner] != 7 || len(set.elements) != 1 {
		t.Fatal("direct trigger argument was not collected", set)
	}
}
