package smtcore

import "testing"

func TestSMTSequenceSolutionMapDependenciesOverwriteAndScopes(t *testing.T) {
	m := astTestManager(t)
	a, err := m.MkString("a")
	astTestMust(t, err)
	b, err := m.MkString("b")
	astTestMust(t, err)
	c, err := m.MkString("c")
	astTestMust(t, err)
	d, err := m.MkString("d")
	astTestMust(t, err)
	dependencies := &smtSequenceDependencyManager{}
	first := dependencies.Leaf(smtSequenceAssumption{literal: NewSMTLiteral(3, false)})
	second := dependencies.Leaf(smtSequenceAssumption{literal: NewSMTLiteral(5, true)})
	solutions, err := newSMTSequenceSolutionMap(m, dependencies)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, solutions.Close()) })
	astTestMust(t, solutions.Update(a, b, first))
	astTestMust(t, solutions.PushScope())
	astTestMust(t, solutions.Update(b, c, second))
	value, dependency, err := solutions.Find(a)
	astTestMust(t, err)
	if value != c {
		t.Fatal("transitive sequence solution", value)
	}
	assumptions := dependencies.Linearize(dependency)
	if len(assumptions) != 2 || assumptions[0].literal != NewSMTLiteral(3, false) || assumptions[1].literal != NewSMTLiteral(5, true) {
		t.Fatal("solution dependencies", assumptions)
	}
	astTestMust(t, solutions.Update(a, d, second))
	value, _, err = solutions.Find(a)
	astTestMust(t, err)
	if value != d {
		t.Fatal("overwritten sequence solution", value)
	}
	astTestMust(t, solutions.PopScopes(1))
	value, dependency, err = solutions.Find(a)
	astTestMust(t, err)
	if value != b || len(dependencies.Linearize(dependency)) != 1 || !solutions.IsRoot(b) {
		t.Fatal("sequence solution rollback", value, dependency)
	}
}

func TestSMTSequenceExclusionTableCanonicalPairsAndScopes(t *testing.T) {
	m := astTestManager(t)
	a, err := m.MkString("left")
	astTestMust(t, err)
	b, err := m.MkString("right")
	astTestMust(t, err)
	c, err := m.MkString("scoped")
	astTestMust(t, err)
	table, err := newSMTSequenceExclusionTable(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, table.Close()) })
	astTestMust(t, table.Update(a, b))
	astTestMust(t, table.Update(b, a))
	if !table.Contains(a, b) || !table.Contains(b, a) || len(table.trail) != 1 {
		t.Fatal("canonical sequence exclusion")
	}
	astTestMust(t, table.PushScope())
	astTestMust(t, table.Update(a, c))
	if !table.Contains(c, a) {
		t.Fatal("scoped sequence exclusion")
	}
	astTestMust(t, table.PopScopes(1))
	if table.Contains(a, c) || !table.Contains(a, b) {
		t.Fatal("sequence exclusion rollback")
	}
}
