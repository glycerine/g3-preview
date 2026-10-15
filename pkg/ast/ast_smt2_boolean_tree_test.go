package ast

import (
	"testing"

	usymbol "github.com/glycerine/g3/pkg/util/symbol"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func TestASTSMT2BooleanTreeSourcePostorderAndLabels(t *testing.T) {
	c, g, p, budget := dynAckSourceFixture(t)
	m, err := NewASTManager(ASTManagerConfig{Symbols: c.SymbolsField, Numerals: c.NumeralsField, Budget: budget})
	paramsMust(t, err)
	defer func() { paramsMust(t, m.Close()) }()
	boolSort, err := m.MkBoolSort()
	paramsMust(t, err)
	left, err := m.MkConst(astTestSymbol(t, m, "left"), boolSort)
	paramsMust(t, err)
	right, err := m.MkConst(astTestSymbol(t, m, "right"), boolSort)
	paramsMust(t, err)
	notRight, err := m.MkNot(right)
	paramsMust(t, err)
	implies, err := m.MkImplies(left, notRight)
	paramsMust(t, err)
	label, err := m.MkLabel(true, []usymbol.Symbol{astTestSymbol(t, m, "tag")}, implies)
	paramsMust(t, err)
	lit, err := m.MkLabelLit([]usymbol.Symbol{astTestSymbol(t, m, "a b")})
	paramsMust(t, err)
	variable, err := m.MkVar(4, boolSort)
	paramsMust(t, err)
	middle, err := m.MkConst(astTestSymbol(t, m, "middle"), boolSort)
	paramsMust(t, err)
	last, err := m.MkConst(astTestSymbol(t, m, "last"), boolSort)
	paramsMust(t, err)
	inner, err := m.MkAnd(middle, last)
	paramsMust(t, err)
	flat, err := m.MkAnd(left, inner)
	paramsMust(t, err)
	repeatedAtom, err := m.MkAnd(left, flat)
	paramsMust(t, err)
	repeatedVar, err := m.MkAnd(variable, variable)
	paramsMust(t, err)
	sharedSmall, err := m.MkAnd(inner, inner)
	paramsMust(t, err)
	out, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: budget})
	paramsMust(t, err)
	defer func() { paramsMust(t, out.Close()) }()
	for _, tc := range []struct {
		root ASTHandle
		want string
	}{
		{left, "left"},
		{implies, "(=> left (not right))"},
		{label, "(! (=> left (not right)) :lblpos tag)"},
		{lit, "(! true :lblpos |a b|)"},
		{variable, "(:var 4)"},
		{flat, "(and left middle last)"},
		{repeatedAtom, "(and left left middle last)"},
		{repeatedVar, "(and (:var 4) (:var 4))"},
		{sharedSmall, "(and (and middle last) (and middle last))"},
	} {
		format, e := m.SMT2PPBooleanTree(tc.root, nil)
		paramsMust(t, e)
		paramsMust(t, m.PrintFormat(out, format, c, p, g))
		got, e := out.String()
		paramsMust(t, e)
		if got != tc.want {
			t.Fatalf("root %v: %q, want %q", tc.root, got, tc.want)
		}
		paramsMust(t, out.ResetText())
	}
	otherSort, err := m.MkSort(astTestSymbol(t, m, "Other"), ASTSortInfo{})
	paramsMust(t, err)
	nonBoolean, err := m.MkConst(astTestSymbol(t, m, "n"), otherSort)
	paramsMust(t, err)
	if _, err := m.SMT2PPBooleanTree(nonBoolean, nil); err == nil {
		t.Fatal("unported numeral/non-Boolean constant path silently formatted")
	}
	var largeArgs []ASTHandle
	for i := 0; i < 9; i++ {
		arg, e := m.MkConst(astTestSymbol(t, m, string(rune('a'+i))), boolSort)
		paramsMust(t, e)
		largeArgs = append(largeArgs, arg)
	}
	large, err := m.MkAndArgs(largeArgs)
	paramsMust(t, err)
	sharedLarge, err := m.MkAnd(large, large)
	paramsMust(t, err)
	if _, err := m.SMT2PPBooleanTree(sharedLarge, nil); err == nil {
		t.Fatal("alias-required shared application silently printed without a let")
	}
}
