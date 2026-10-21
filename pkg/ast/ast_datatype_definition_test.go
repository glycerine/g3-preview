package ast

import "testing"

func TestASTDatatypeDefinitionBlockAndUtilities(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterDatatypePlugin()
	astTestMust(t, err)
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)

	evenName := astTestSymbol(t, m, "EvenList")
	oddName := astTestSymbol(t, m, "OddList")
	sorts, err := m.MkDatatypes([]DatatypeDefinition{
		{
			Name: evenName,
			Constructors: []DatatypeConstructorDefinition{
				{Name: astTestSymbol(t, m, "even-nil"), Recognizer: astTestSymbol(t, m, "is-even-nil")},
				{Name: astTestSymbol(t, m, "even-cons"), Recognizer: astTestSymbol(t, m, "is-even-cons"), Accessors: []DatatypeAccessorDefinition{
					{Name: astTestSymbol(t, m, "even-head"), Range: boolean},
					{Name: astTestSymbol(t, m, "even-tail"), Recursive: true, RecursiveIndex: 1},
				}},
			},
		},
		{
			Name: oddName,
			Constructors: []DatatypeConstructorDefinition{
				{Name: astTestSymbol(t, m, "odd-cons"), Recognizer: astTestSymbol(t, m, "is-odd-cons"), Accessors: []DatatypeAccessorDefinition{
					{Name: astTestSymbol(t, m, "odd-head"), Range: boolean},
					{Name: astTestSymbol(t, m, "odd-tail"), Recursive: true, RecursiveIndex: 0},
				}},
			},
		},
	})
	astTestMust(t, err)
	if len(sorts) != 2 {
		t.Fatal("datatype sort count", len(sorts))
	}
	siblings, err := m.DatatypeAreSiblings(sorts[0], sorts[1])
	astTestMust(t, err)
	if !siblings {
		t.Fatal("mutually recursive datatypes are not siblings")
	}
	count, err := m.DatatypeNumConstructors(sorts[0])
	astTestMust(t, err)
	if count != 2 {
		t.Fatal("constructor count", count)
	}
	constructors, err := m.DatatypeConstructors(sorts[0])
	astTestMust(t, err)
	if len(constructors) != 2 {
		t.Fatal("constructors", len(constructors))
	}
	index, matched, err := m.DatatypeConstructorIndex(constructors[1])
	astTestMust(t, err)
	if !matched || index != 1 {
		t.Fatal("constructor index", index, matched)
	}
	accessors, err := m.DatatypeConstructorAccessors(constructors[1])
	astTestMust(t, err)
	if len(accessors) != 2 {
		t.Fatal("accessors", len(accessors))
	}
	if fieldRange, e := m.GetRange(accessors[1]); e != nil || fieldRange != sorts[1] {
		t.Fatal("recursive accessor range", fieldRange, e)
	}
	recognizer, err := m.DatatypeConstructorRecognizer(constructors[1])
	astTestMust(t, err)
	if original, matched, e := m.DatatypeRecognizerConstructor(recognizer); e != nil || !matched || original != constructors[1] {
		t.Fatal("recognizer constructor", original, matched, e)
	}
	if got, matched, e := m.DatatypeRecognizerConstructorIndex(recognizer); e != nil || !matched || got != 1 {
		t.Fatal("recognizer constructor index", got, matched, e)
	}
	if owner, matched, e := m.DatatypeAccessorConstructor(accessors[0]); e != nil || !matched || owner != constructors[1] {
		t.Fatal("accessor constructor", owner, matched, e)
	}
	if enum, e := m.DatatypeIsEnumSort(sorts[0]); e != nil || enum {
		t.Fatal("recursive datatype classified as enum", enum, e)
	}
	if recursive, e := m.DatatypeIsRecursive(sorts[0]); e != nil || !recursive {
		t.Fatal("mutually recursive datatype", recursive, e)
	}
	nonRecursive, found, err := m.DatatypeNonRecursiveConstructor(sorts[0])
	astTestMust(t, err)
	if !found {
		t.Fatal("missing non-recursive constructor")
	}
	if got, matched, e := m.DatatypeConstructorIndex(nonRecursive); e != nil || !matched || got != 0 {
		t.Fatal("non-recursive constructor", got, matched, e)
	}
}

func TestASTDatatypeDefinitionEnumAndWellFoundedRefusal(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterDatatypePlugin()
	astTestMust(t, err)

	color, err := m.MkDatatypes([]DatatypeDefinition{{
		Name: astTestSymbol(t, m, "Color"),
		Constructors: []DatatypeConstructorDefinition{
			{Name: astTestSymbol(t, m, "red"), Recognizer: astTestSymbol(t, m, "is-red")},
			{Name: astTestSymbol(t, m, "blue"), Recognizer: astTestSymbol(t, m, "is-blue")},
		},
	}})
	astTestMust(t, err)
	if enum, e := m.DatatypeIsEnumSort(color[0]); e != nil || !enum {
		t.Fatal("enum datatype", enum, e)
	}
	if recursive, e := m.DatatypeIsRecursive(color[0]); e != nil || recursive {
		t.Fatal("enum recursion", recursive, e)
	}

	_, err = m.MkDatatypes([]DatatypeDefinition{{
		Name: astTestSymbol(t, m, "Bottom"),
		Constructors: []DatatypeConstructorDefinition{{
			Name: astTestSymbol(t, m, "bottom"), Recognizer: astTestSymbol(t, m, "is-bottom"),
			Accessors: []DatatypeAccessorDefinition{{Name: astTestSymbol(t, m, "next-bottom"), Recursive: true}},
		}},
	}})
	if err == nil {
		t.Fatal("accepted datatype without a finite constructor")
	}
}

func TestASTParametricDatatypeDefinitionInstantiation(t *testing.T) {
	m := astTestManager(t)
	family, err := m.RegisterDatatypePlugin()
	astTestMust(t, err)
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	formal, err := m.MkSort(astTestSymbol(t, m, "T"), ASTSortInfo{})
	astTestMust(t, err)
	astTestPin(t, m, formal)

	name := astTestSymbol(t, m, "BoxOf")
	_, err = m.MkDatatypes([]DatatypeDefinition{{
		Name:       name,
		Parameters: []ASTHandle{formal},
		Constructors: []DatatypeConstructorDefinition{{
			Name:       astTestSymbol(t, m, "box-of"),
			Recognizer: astTestSymbol(t, m, "is-box-of"),
			Accessors:  []DatatypeAccessorDefinition{{Name: astTestSymbol(t, m, "unbox-of"), Range: formal}},
		}},
	}})
	astTestMust(t, err)

	p0, err := m.NewParameterSymbol(name)
	astTestMust(t, err)
	p1, err := m.NewParameterAST(boolean)
	astTestMust(t, err)
	boxBool, err := m.MkFamilySort(family, ASTDatatypeSort, []ASTParameter{p0, p1})
	astTestMust(t, err)
	astTestMust(t, m.DelParameter(&p1))
	astTestMust(t, m.DelParameter(&p0))
	constructors, err := m.DatatypeConstructors(boxBool)
	astTestMust(t, err)
	if len(constructors) != 1 {
		t.Fatal("parametric constructors", len(constructors))
	}
	if domain, e := m.GetDomain(constructors[0], 0); e != nil || domain != boolean {
		t.Fatal("substituted constructor domain", domain, e)
	}
	accessors, err := m.DatatypeConstructorAccessors(constructors[0])
	astTestMust(t, err)
	if len(accessors) != 1 {
		t.Fatal("parametric accessors", len(accessors))
	}
	if fieldRange, e := m.GetRange(accessors[0]); e != nil || fieldRange != boolean {
		t.Fatal("substituted accessor range", fieldRange, e)
	}
	if count, e := m.DatatypeNumParameters(boxBool); e != nil || count != 1 {
		t.Fatal("datatype parameter count", count, e)
	}
	if parameter, e := m.DatatypeParameter(boxBool, 0); e != nil || parameter != boolean {
		t.Fatal("datatype parameter", parameter, e)
	}
}
