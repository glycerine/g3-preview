package ast

import (
	"testing"

	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestASTArrayOriginalNameCallbacks(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArrayPlugin()
	astTestMust(t, e)
	for _, logicText := range []string{"", "HORN", "ALL", "QF_AUFBV"} {
		logic := usymbol.Symbol{}
		if logicText != "" {
			logic = astTestSymbol(t, m, logicText)
		}
		var names ASTBuiltinNames
		astTestMust(t, m.GetArraySortNames(&names, logic))
		if len(names.Items()) != 1 {
			t.Fatal("array sort name count", logicText, len(names.Items()))
		}
		text, e := m.Symbols.String(names.Items()[0].Name)
		astTestMust(t, e)
		if text != "Array" || names.Items()[0].Kind != 0 {
			t.Fatal("array sort name", text, names.Items()[0].Kind)
		}
		names.Reset()
		astTestMust(t, m.GetArrayOpNames(&names, logic))
		wantText := []string{"store", "select"}
		wantKind := []int32{ASTArrayOpStore, ASTArrayOpSelect}
		if logicText == "" || logicText == "HORN" || logicText == "ALL" {
			wantText = append(wantText, "const", "map", "default", "union", "intersect", "difference", "complement", "subset", "as-array", "array-ext")
			wantKind = append(wantKind, ASTArrayOpConst, ASTArrayOpMap, ASTArrayOpDefault, ASTArrayOpSetUnion, ASTArrayOpSetIntersect, ASTArrayOpSetDifference, ASTArrayOpSetComplement, ASTArrayOpSetSubset, ASTArrayOpAsArray, ASTArrayOpExt)
		}
		if len(names.Items()) != len(wantText) {
			t.Fatal("array operator count", logicText, len(names.Items()))
		}
		for i, item := range names.Items() {
			text, e = m.Symbols.String(item.Name)
			astTestMust(t, e)
			if text != wantText[i] || item.Kind != wantKind[i] {
				t.Fatal("array operator", logicText, i, text, item.Kind)
			}
		}
		names.Close()
	}
}

func TestASTArrayOriginalAsArrayTree(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArrayPlugin()
	astTestMust(t, e)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	makeLeaf := func(name string) ASTHandle {
		f, x := m.MkFuncDecl(astTestSymbol(t, m, name), []ASTHandle{b}, b, ASTFuncDeclInfo{})
		astTestMust(t, x)
		astTestPin(t, m, f)
		a, x := m.MkAsArray(f)
		astTestMust(t, x)
		astTestPin(t, m, a)
		return a
	}
	left, right := makeLeaf("array-tree-left"), makeLeaf("array-tree-right")
	condition, e := m.MkTrue()
	astTestMust(t, e)
	tree, e := m.MkIte(condition, left, right)
	astTestMust(t, e)
	astTestPin(t, m, tree)
	if yes, e := m.IsAsArrayTree(left); e != nil || !yes {
		t.Fatal("as-array leaf", yes, e)
	}
	if yes, e := m.IsAsArrayTree(tree); e != nil || !yes {
		t.Fatal("as-array ite tree", yes, e)
	}
	if yes, e := m.IsAsArrayTree(condition); e != nil || yes {
		t.Fatal("non-array leaf", yes, e)
	}
}
