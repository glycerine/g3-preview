package ast

import (
	"testing"

	usymbol "github.com/glycerine/g3/pkg/util/symbol"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func TestASTSMT2VariableAndLabelSourceFormats(t *testing.T) {
	c, g, p, budget := dynAckSourceFixture(t)
	m, err := NewASTManager(ASTManagerConfig{Symbols: c.SymbolsField, Numerals: c.NumeralsField, Budget: budget})
	paramsMust(t, err)
	defer func() { paramsMust(t, m.Close()) }()
	out, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: budget})
	paramsMust(t, err)
	defer func() { paramsMust(t, out.Close()) }()
	check := func(format ASTHandle, want string) {
		t.Helper()
		paramsMust(t, m.PrintFormat(out, format, c, p, g))
		got, e := out.String()
		paramsMust(t, e)
		if got != want {
			t.Fatalf("format %q, want %q", got, want)
		}
		paramsMust(t, out.ResetText())
	}
	outer := astTestSymbol(t, m, "outer")
	inner := astTestSymbol(t, m, "a b")
	for _, tc := range []struct {
		index uint32
		names []usymbol.Symbol
		want  string
	}{
		{0, []usymbol.Symbol{outer, inner}, "|a b|"},
		{1, []usymbol.Symbol{outer, inner}, "outer"},
		{2, []usymbol.Symbol{outer, inner}, "(:var 2)"},
		{0, nil, "(:var 0)"},
		{0, []usymbol.Symbol{usymbol.Numerical(3)}, "k!3"},
	} {
		format, e := m.SMT2PPVar(tc.index, tc.names)
		paramsMust(t, e)
		check(format, tc.want)
	}
	body, err := m.FormatString("true")
	paramsMust(t, err)
	empty, err := m.SMT2PPLabels(true, nil, body)
	paramsMust(t, err)
	if empty != body {
		t.Fatal("empty labels changed source format node")
	}
	for _, tc := range []struct {
		positive bool
		want     string
	}{
		{true, "(! true :lblpos outer :lblpos |a b|)"},
		{false, "(! true :lblneg outer :lblneg |a b|)"},
	} {
		format, e := m.SMT2PPLabels(tc.positive, []usymbol.Symbol{outer, inner}, body)
		paramsMust(t, e)
		check(format, tc.want)
	}
	intAttribute, err := m.SMT2PPSimpleAttributeInt(":weight ", -7)
	paramsMust(t, err)
	check(intAttribute, ":weight -7")
	a, err := m.FormatString("a")
	paramsMust(t, err)
	b, err := m.FormatString("b")
	paramsMust(t, err)
	line, err := m.FormatLineBreak()
	paramsMust(t, err)
	innerBody, err := m.FormatCompose([]ASTHandle{a, line, b})
	paramsMust(t, err)
	attribute, err := m.SMT2PPAttribute(":pattern ", innerBody)
	paramsMust(t, err)
	check(attribute, ":pattern a\n         b")
}
