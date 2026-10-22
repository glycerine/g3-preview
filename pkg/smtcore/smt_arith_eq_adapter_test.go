package smtcore

import "testing"

func TestSMTArithmeticEqualityAdapterEventsAndScopes(t *testing.T) {
	f := newSMTIntegerDiffFixture(t)
	astTestMust(t, f.state.Internalize(f.x, false))
	astTestMust(t, f.state.Internalize(f.y, false))
	left := f.state.context.enodes[f.x].TheoryVar(f.family)
	right := f.state.context.enodes[f.y].TheoryVar(f.family)
	astTestMust(t, f.state.PushScope())
	astTestMust(t, f.state.enqueueTheoryEquality(f.family, left, right))
	astTestMust(t, f.state.PropagateTheoryEvents())
	if f.theory.equalityAdapter.NumAxioms() != 1 || len(f.theory.atoms) != 2 || len(f.state.Clauses()) != 4 {
		t.Fatal("integer equality event adapter", f.theory.equalityAdapter.NumAxioms(), len(f.theory.atoms), len(f.state.Clauses()))
	}
	astTestMust(t, f.theory.NewDisequality(left, right))
	if f.theory.equalityAdapter.NumAxioms() != 1 {
		t.Fatal("duplicate pair was processed twice")
	}
	astTestMust(t, f.state.PopScopes(1))
	if len(f.theory.atoms) != 0 || len(f.state.Clauses()) != 0 {
		t.Fatal("adapter products survived scope rollback", len(f.theory.atoms), len(f.state.Clauses()))
	}
	astTestMust(t, f.state.PushScope())
	astTestMust(t, f.theory.NewDisequality(left, right))
	if f.theory.equalityAdapter.NumAxioms() != 2 || len(f.theory.atoms) != 2 {
		t.Fatal("rolled-back pair was not recreated", f.theory.equalityAdapter.NumAxioms(), len(f.theory.atoms))
	}
}

func TestSMTRealDifferenceEqualityEventAdapter(t *testing.T) {
	f := newSMTRealDiffFixture(t)
	astTestMust(t, f.state.Internalize(f.x, false))
	astTestMust(t, f.state.Internalize(f.y, false))
	left := f.state.context.enodes[f.x].TheoryVar(f.family)
	right := f.state.context.enodes[f.y].TheoryVar(f.family)
	astTestMust(t, f.state.enqueueTheoryDisequality(f.family, left, right))
	astTestMust(t, f.state.PropagateTheoryEvents())
	if f.theory.equalityAdapter.NumAxioms() != 1 || len(f.theory.realAtoms) != 2 || len(f.state.Clauses()) != 4 {
		t.Fatal("real disequality event adapter", f.theory.equalityAdapter.NumAxioms(), len(f.theory.realAtoms), len(f.state.Clauses()))
	}
}

func TestSMTArithmeticEqualityAdapterRecreatesSearchAxiomsAtRestart(t *testing.T) {
	f := newSMTIntegerDiffFixture(t)
	astTestMust(t, f.state.Internalize(f.x, false))
	astTestMust(t, f.state.Internalize(f.y, false))
	left := f.state.context.enodes[f.x].TheoryVar(f.family)
	right := f.state.context.enodes[f.y].TheoryVar(f.family)
	params := SMTParamsState{AgilityFactor: 0.9999, RestartAdaptive: true}
	astTestMust(t, f.state.InitializeTheorySearch())
	astTestMust(t, f.state.beginSearchPersistence(0, params))
	defer f.state.endSearchPersistence()
	astTestMust(t, f.state.PushScope())
	astTestMust(t, f.state.enqueueTheoryEquality(f.family, left, right))
	astTestMust(t, f.state.PropagateTheoryEvents())
	if len(f.theory.equalityAdapter.restart) != 1 || len(f.state.Clauses()) != 4 {
		t.Fatal("search equality was not queued for restart", len(f.theory.equalityAdapter.restart), len(f.state.Clauses()))
	}
	astTestMust(t, f.state.PopScopes(1))
	if len(f.state.Clauses()) != 0 {
		t.Fatal("search axioms survived decision backtracking", len(f.state.Clauses()))
	}
	astTestMust(t, f.state.RestartTheories())
	if len(f.theory.equalityAdapter.restart) != 0 || f.theory.equalityAdapter.NumAxioms() != 2 || len(f.state.Clauses()) != 4 {
		t.Fatal("restart did not recreate equality axioms", len(f.theory.equalityAdapter.restart), f.theory.equalityAdapter.NumAxioms(), len(f.state.Clauses()))
	}
}
