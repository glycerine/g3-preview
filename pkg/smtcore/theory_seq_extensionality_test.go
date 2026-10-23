package smtcore

import "testing"

func TestSMTSequenceExtensionalityHonorsDisabledRelevancy(t *testing.T) {
	m := astTestManager(t)
	value, err := m.MkString("a")
	astTestMust(t, err)
	sort, err := m.GetSort(value)
	astTestMust(t, err)
	x, err := m.MkConst(astTestSymbol(t, m, "unmarked-sequence-x"), sort)
	astTestMust(t, err)
	y, err := m.MkConst(astTestSymbol(t, m, "unmarked-sequence-y"), sort)
	astTestMust(t, err)
	family, err := m.SequenceFamilyID()
	astTestMust(t, err)
	theory, err := NewSMTSequenceTheory(family)
	astTestMust(t, err)
	context, relevancy, state := smtInternalizerFixture(t, m, nil)
	astTestMust(t, state.RegisterTheory(theory))
	astTestMust(t, state.Internalize(x, false))
	astTestMust(t, state.Internalize(y, false))
	equality, err := m.MkEq(x, y)
	astTestMust(t, err)
	astTestPin(t, m, equality)
	for _, enabled := range []bool{true, false, true} {
		relevancy.enabled = enabled
		astTestMust(t, state.PushScope())
		progress, err := theory.checkSequenceExtensionality()
		astTestMust(t, err)
		if progress == enabled {
			t.Fatal("extensionality ignored the current relevance policy", enabled, progress)
		}
		_, found := state.BoolVar(equality)
		if found == enabled {
			t.Fatal("interface equality was not scoped to the disabled-filter pass", enabled, found)
		}
		if context.relevant[x] || context.relevant[y] {
			t.Fatal("querying the relevance policy changed its marks")
		}
		astTestMust(t, state.PopScopes(1))
		if _, found := state.BoolVar(equality); found {
			t.Fatal("extensionality equality survived scope rollback")
		}
	}
}
