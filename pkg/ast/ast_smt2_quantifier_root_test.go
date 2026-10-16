package ast

import (
	"testing"

	usymbol "github.com/glycerine/g3/pkg/util/symbol"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func TestASTSMT2QuantifierRootSourceFormat(t *testing.T) {
	c, g, p, budget := dynAckSourceFixture(t)
	m, err := NewASTManager(ASTManagerConfig{Symbols: c.SymbolsField, Numerals: c.NumeralsField, Budget: budget})
	paramsMust(t, err)
	defer func() { paramsMust(t, m.Close()) }()
	boolSort, err := m.MkBoolSort()
	paramsMust(t, err)
	x := astTestSymbol(t, m, "x")
	y := astTestSymbol(t, m, "y")
	xBang1 := astTestSymbol(t, m, "x!1")
	body, err := m.MkVar(0, boolSort)
	paramsMust(t, err)
	print := func(h ASTHandle) string {
		t.Helper()
		format, e := m.SMT2PPBooleanTree(h, nil)
		paramsMust(t, e)
		out, e := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: budget})
		paramsMust(t, e)
		paramsMust(t, m.PrintFormat(out, format, c, p, g))
		text, e := out.String()
		paramsMust(t, e)
		paramsMust(t, out.Close())
		return text
	}
	for _, tc := range []struct {
		forall bool
		names  []usymbol.Symbol
		want   string
	}{
		{true, []usymbol.Symbol{x}, "(forall ((x Bool)) x)"},
		{false, []usymbol.Symbol{x, y}, "(exists ((x Bool) (y Bool)) y)"},
		{true, []usymbol.Symbol{usymbol.Numerical(0)}, "(forall ((x!1 Bool)) x!1)"},
		{false, []usymbol.Symbol{usymbol.Numerical(0), usymbol.Numerical(1)}, "(exists ((x!1 Bool) (x!2 Bool)) x!2)"},
		{true, []usymbol.Symbol{x, x}, "(forall ((x Bool) (x!1 Bool)) x!1)"},
		{true, []usymbol.Symbol{xBang1, usymbol.Numerical(0)}, "(forall ((x!1 Bool) (x!2 Bool)) x!2)"},
	} {
		sorts := make([]ASTHandle, len(tc.names))
		for i := range sorts {
			sorts[i] = boolSort
		}
		q, e := m.MkQuantifier(ASTQuantifierSpec{Forall: tc.forall, Sorts: sorts, Names: tc.names, Body: body})
		paramsMust(t, e)
		if got := print(q); got != tc.want {
			t.Fatalf("quantifier format %q, want %q", got, tc.want)
		}
	}
	q, err := m.MkQuantifier(ASTQuantifierSpec{Forall: true, Sorts: []ASTHandle{boolSort}, Names: []usymbol.Symbol{x}, Body: body, Weight: 2})
	paramsMust(t, err)
	if got := print(q); got != "(forall ((x Bool)) (! x :weight 2))" {
		t.Fatalf("weighted quantifier format %q", got)
	}
	f, err := m.MkFuncDecl(astTestSymbol(t, m, "f"), []ASTHandle{boolSort}, boolSort, ASTFuncDeclInfo{})
	paramsMust(t, err)
	term, err := m.MkApp(f, []ASTHandle{body})
	paramsMust(t, err)
	pattern, err := m.MkFamilyApp(2, 0, nil, []ASTHandle{term})
	paramsMust(t, err)
	qid := astTestSymbol(t, m, "qid")
	skid := astTestSymbol(t, m, "skid")
	q, err = m.MkQuantifier(ASTQuantifierSpec{Forall: true, Sorts: []ASTHandle{boolSort}, Names: []usymbol.Symbol{x}, Body: body,
		Patterns: []ASTHandle{pattern}, Weight: 3, QID: qid, SkolemID: skid})
	paramsMust(t, err)
	if got := print(q); got != "(forall ((x Bool)) (! x :pattern ((f x)) :weight 3 :skolemid skid :qid qid))" {
		t.Fatalf("pattern annotation format %q", got)
	}
	q, err = m.MkQuantifier(ASTQuantifierSpec{Forall: false, Sorts: []ASTHandle{boolSort}, Names: []usymbol.Symbol{x}, Body: body,
		NoPatterns: []ASTHandle{term}, QID: usymbol.Numerical(7)})
	paramsMust(t, err)
	if got := print(q); got != "(exists ((x Bool)) (! x :no-pattern (f x)))" {
		t.Fatalf("no-pattern annotation format %q", got)
	}
}
