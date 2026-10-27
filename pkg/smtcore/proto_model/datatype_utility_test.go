package protomodel

import (
	"github.com/glycerine/g3/pkg/ast"
	"testing"
)

func TestDatatypeFactoryUsesIndependentCachedConstructorUtility(t *testing.T) {
	m, p := protoFixture(t)
	_, err := m.RegisterDatatypePlugin()
	protoMust(t, err)
	makeSort := func(name string) ast.ASTHandle {
		var constructors []ast.DatatypeConstructorDefinition
		for _, tag := range []string{"a", "b", "c"} {
			constructors = append(constructors, ast.DatatypeConstructorDefinition{Name: protoSymbol(t, m, name+tag), Recognizer: protoSymbol(t, m, "is-"+name+tag)})
		}
		sorts, err := m.MkDatatypes([]ast.DatatypeDefinition{{Name: protoSymbol(t, m, name), Constructors: constructors}})
		protoMust(t, err)
		protoPin(t, m, sorts[0])
		return sorts[0]
	}
	warm, choice := makeSort("UtilityWarm"), makeSort("UtilityChoice")
	_, _, err = m.DatatypeNonRecursiveConstructor(warm)
	protoMust(t, err)
	constructors, err := m.DatatypeConstructors(choice)
	protoMust(t, err)
	for i := 0; i < 3; i++ {
		value, err := m.GetSomeValue(choice)
		protoMust(t, err)
		protoPin(t, m, value)
		constructor, found, err := m.DatatypeConstructorApp(value)
		protoMust(t, err)
		if !found || constructor != constructors[2] {
			t.Fatal("plugin utility did not cache its own second-sort choice")
		}
	}
	f, err := NewDatatypeFactory(p)
	protoMust(t, err)
	protoMust(t, p.RegisterFactory(f))
	for i := 0; i < 3; i++ {
		value, err := p.SomeValue(choice)
		protoMust(t, err)
		constructor, found, err := m.DatatypeConstructorApp(value)
		protoMust(t, err)
		if !found || constructor != constructors[1] {
			t.Fatal("model factory borrowed the plugin's selection counter")
		}
	}
}

func TestDatatypeFactoryUsesOwnedRecursionCache(t *testing.T) {
	m, f, _, sort, _ := datatypeFactoryBVFixture(t, 0)
	_, err := f.FreshValue(sort)
	protoMust(t, err)
	before, err := m.GetRefCount(sort)
	protoMust(t, err)
	recursive, err := f.utility.IsRecursive(sort)
	protoMust(t, err)
	after, err := m.GetRefCount(sort)
	protoMust(t, err)
	if recursive || before != after {
		t.Fatal("factory bypassed its independent recursive-result owner", recursive, before, after)
	}
}
