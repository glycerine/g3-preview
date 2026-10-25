package ast

import "testing"

func TestDatatypeUtilResetReleasesPinsAndAdvancesPreference(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterDatatypePlugin()
	astTestMust(t, err)
	sorts, err := m.MkDatatypes([]DatatypeDefinition{{
		Name: astTestSymbol(t, m, "reset-utility-sort"),
		Constructors: []DatatypeConstructorDefinition{
			{Name: astTestSymbol(t, m, "reset-utility-a"), Recognizer: astTestSymbol(t, m, "reset-utility-is-a")},
			{Name: astTestSymbol(t, m, "reset-utility-b"), Recognizer: astTestSymbol(t, m, "reset-utility-is-b")},
			{Name: astTestSymbol(t, m, "reset-utility-c"), Recognizer: astTestSymbol(t, m, "reset-utility-is-c")},
		},
	}})
	astTestMust(t, err)
	ctors, err := m.DatatypeConstructors(sorts[0])
	astTestMust(t, err)
	astTestPin(t, m, sorts[0])
	for _, c := range ctors {
		astTestPin(t, m, c)
	}
	before := make([]uint32, len(ctors)+1)
	handles := append([]ASTHandle{sorts[0]}, ctors...)
	for i, h := range handles {
		before[i], err = m.GetRefCount(h)
		astTestMust(t, err)
	}
	u, err := NewDatatypeUtil(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, u.Close()) })
	for _, expected := range []int{1, 0, 2} {
		chosen, found, err := u.NonRecursiveConstructor(sorts[0])
		astTestMust(t, err)
		if !found || chosen != ctors[expected] {
			t.Fatal("wrong reset constructor preference", expected)
		}
		copied := *u
		if copied.Reset() == nil {
			t.Fatal("copied utility accepted reset ownership")
		}
		start := u.start
		astTestMust(t, u.Reset())
		if len(u.refs) != 0 || len(u.constructors) != 0 || u.start != start+1 {
			t.Fatal("reset retained cache or lost preference advance")
		}
		for i, h := range handles {
			count, err := m.GetRefCount(h)
			astTestMust(t, err)
			if count != before[i] {
				t.Fatal("reset retained an AST pin")
			}
		}
	}
	astTestMust(t, u.Close())
	if u.Reset() == nil {
		t.Fatal("closed utility accepted reset")
	}
}
