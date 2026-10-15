package ast

import (
	"testing"

	utext "github.com/glycerine/g3/pkg/util/text"
)

func TestASTSMT2BooleanComparisonsWithArithmeticLiterals(t *testing.T) {
	c, g, p, budget := dynAckSourceFixture(t)
	m, err := NewASTManager(ASTManagerConfig{Symbols: c.SymbolsField, Numerals: c.NumeralsField, Budget: budget})
	paramsMust(t, err)
	defer func() { paramsMust(t, m.Close()) }()
	_, err = m.RegisterArithmeticPlugin()
	paramsMust(t, err)
	value, err := c.NumeralsField.NewValue()
	paramsMust(t, err)
	defer func() { paramsMust(t, c.NumeralsField.CleanupValue(&value)) }()
	out, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: budget})
	paramsMust(t, err)
	defer func() { paramsMust(t, out.Close()) }()
	for _, tc := range []struct {
		value   string
		integer bool
		want    string
	}{
		{"23", true, "(= 23 x)"},
		{"-23", true, "(= (- 23) x)"},
		{"2", false, "(= 2.0 x)"},
		{"1/3", false, "(= (/ 1.0 3.0) x)"},
		{"-1/3", false, "(= (- (/ 1.0 3.0)) x)"},
	} {
		paramsMust(t, c.NumeralsField.SetString(&value, tc.value))
		literal, e := m.ArithmeticNumeral(&value, tc.integer)
		paramsMust(t, e)
		sort, e := m.GetSort(literal)
		paramsMust(t, e)
		x, e := m.MkConst(astTestSymbol(t, m, "x"), sort)
		paramsMust(t, e)
		root, e := m.MkEq(literal, x)
		paramsMust(t, e)
		format, e := m.SMT2PPBooleanTree(root, nil)
		paramsMust(t, e)
		paramsMust(t, m.PrintFormat(out, format, c, p, g))
		got, e := out.String()
		paramsMust(t, e)
		if got != tc.want {
			t.Fatalf("%s integer=%v: %q, want %q", tc.value, tc.integer, got, tc.want)
		}
		paramsMust(t, out.ResetText())
	}
}
