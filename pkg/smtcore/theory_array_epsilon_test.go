package smtcore

import "testing"

func TestSMTFullArrayEpsilonCacheOwnsValuesAcrossTemporaryReferences(t *testing.T) {
	m := astTestManager(t)
	family, err := m.RegisterArrayPlugin()
	astTestMust(t, err)
	_, _, state := smtInternalizerFixture(t, m, nil)
	theory, err := NewSMTArrayTheory(family)
	astTestMust(t, err)
	astTestMust(t, state.RegisterTheory(theory))
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	epsilon, err := theory.arrayEpsilon(boolean)
	astTestMust(t, err)
	temporary, err := m.NewRef(epsilon)
	astTestMust(t, err)
	astTestMust(t, m.DelRef(&temporary))
	if _, err := m.GetSort(epsilon); err != nil {
		t.Fatal("epsilon cache lost its value when a temporary reference was released", err)
	}
	again, err := theory.arrayEpsilon(boolean)
	astTestMust(t, err)
	if again != epsilon {
		t.Fatal("epsilon cache replaced its live base-scope value")
	}
	basePin, err := m.NewRef(epsilon)
	astTestMust(t, err)
	defer func() { astTestMust(t, m.DelRef(&basePin)) }()
	indexSort, err := m.MkUninterpretedSort(astTestSymbol(t, m, "owned-epsilon-index"), nil)
	astTestMust(t, err)
	indexPin, err := m.NewRef(indexSort)
	astTestMust(t, err)
	defer func() { astTestMust(t, m.DelRef(&indexPin)) }()
	astTestMust(t, state.PushScope())
	child, err := theory.arrayEpsilon(indexSort)
	astTestMust(t, err)
	childPin, err := m.NewRef(child)
	astTestMust(t, err)
	defer func() { astTestMust(t, m.DelRef(&childPin)) }()
	count, err := m.GetRefCount(child)
	astTestMust(t, err)
	if count != 2 {
		t.Fatal("child epsilon must have cache ownership and a test pin", count)
	}
	astTestMust(t, state.PopScopes(1))
	if len(theory.epsilons) != 1 || theory.epsilons[boolean] != epsilon {
		t.Fatal("pop lost the base epsilon or retained the child entry")
	}
	count, err = m.GetRefCount(child)
	astTestMust(t, err)
	if count != 1 {
		t.Fatal("pop retained child epsilon ownership", count)
	}
	astTestMust(t, theory.Close())
	count, err = m.GetRefCount(epsilon)
	astTestMust(t, err)
	if count != 1 || len(theory.epsilons) != 0 || len(theory.epsilonTrail) != 0 {
		t.Fatal("teardown retained base epsilon ownership or cache entries", count)
	}
}
