package ast

import (
	"testing"

	usymbol "github.com/glycerine/g3/pkg/util/symbol"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func TestASTSMT2UnannotatedQuantifierRootSourceFormat(t *testing.T) {
	c, g, p, budget := dynAckSourceFixture(t)
	m, err := NewASTManager(ASTManagerConfig{Symbols: c.SymbolsField, Numerals: c.NumeralsField, Budget: budget})
	paramsMust(t, err)
	defer func() { paramsMust(t, m.Close()) }()
	boolSort, err := m.MkBoolSort()
	paramsMust(t, err)
	x := astTestSymbol(t, m, "x")
	y := astTestSymbol(t, m, "y")
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
	if _, err = m.SMT2PPBooleanTree(q, nil); err != ErrASTDomain {
		t.Fatalf("unported annotated quantifier accepted: %v", err)
	}
}
