package smtcore

import "testing"

func TestSMTSequenceLengthCoherenceRequiresSolutionRoot(t *testing.T) {
	f := newSMTSequenceInterfaceFixture(t)
	family, err := f.m.ArithmeticFamilyID()
	astTestMust(t, err)
	arithmetic, err := NewSMTLinearArithmeticTheory(family)
	astTestMust(t, err)
	astTestMust(t, f.state.RegisterTheory(arithmetic))
	length, err := f.m.MkSequenceLength(f.x)
	astTestMust(t, err)
	minimum, err := f.m.ArithmeticIntegerNumeralText("2")
	astTestMust(t, err)
	lower, err := f.m.MkArithmeticGE(length, minimum)
	astTestMust(t, err)
	astTestMust(t, f.state.Internalize(lower, true))
	variable, found := f.state.BoolVar(lower)
	if !found {
		t.Fatal("length bound was not internalized")
	}
	astTestMust(t, arithmetic.Assign(variable, true))
	astTestMust(t, arithmetic.Propagate())
	for attempt := 0; attempt < 2; attempt++ {
		astTestMust(t, f.state.PushScope())
		astTestMust(t, f.state.PushScope())
		astTestMust(t, f.theory.solutions.Update(f.x, f.y, nil))
		before := f.state.searchProgress()
		axioms := len(f.theory.axioms.values)
		progress, err := f.theory.addLengthLowerBoundDecomposition()
		astTestMust(t, err)
		if progress || f.state.searchProgress() != before || len(f.theory.axioms.values) != axioms {
			t.Fatal("already solved sequence acquired a lower-bound decomposition")
		}
		astTestMust(t, f.state.PopScopes(1))
		progress, err = f.theory.addLengthLowerBoundDecomposition()
		astTestMust(t, err)
		if !progress || len(f.theory.axioms.values) <= axioms {
			t.Fatal("restored solution root did not acquire a lower-bound decomposition")
		}
		astTestMust(t, f.state.PopScopes(1))
	}
}
