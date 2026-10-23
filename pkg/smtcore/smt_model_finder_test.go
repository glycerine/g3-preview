package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/util/lbool"
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
