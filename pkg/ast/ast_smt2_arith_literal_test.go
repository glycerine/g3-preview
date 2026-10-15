package ast

import (
	"testing"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
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

func TestASTSMT2BooleanComparisonsWithBitVectorLiterals(t *testing.T) {
	c, g, p, budget := dynAckSourceFixture(t)
	m, err := NewASTManager(ASTManagerConfig{Symbols: c.SymbolsField, Numerals: c.NumeralsField, Budget: budget})
	paramsMust(t, err)
	defer func() { paramsMust(t, m.Close()) }()
	_, err = m.RegisterArithmeticPlugin()
	paramsMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	paramsMust(t, err)
	value, err := c.NumeralsField.NewValue()
	paramsMust(t, err)
	defer func() { paramsMust(t, c.NumeralsField.CleanupValue(&value)) }()
	out, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: budget})
	paramsMust(t, err)
	defer func() { paramsMust(t, out.Close()) }()
	for _, tc := range []struct {
		value string
		width uint32
		want  string
	}{
		{"10", 4, "(= #xa x)"},
		{"10", 5, "(= #b01010 x)"},
		{"-1", 8, "(= #xff x)"},
	} {
		paramsMust(t, c.NumeralsField.SetString(&value, tc.value))
		literal, e := m.BitVectorNumeral(&value, tc.width)
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
			t.Fatalf("%s width=%d: %q, want %q", tc.value, tc.width, got, tc.want)
		}
		paramsMust(t, out.ResetText())
	}
}

func TestASTSMT2BooleanComparisonsWithFloatingPointLiterals(t *testing.T) {
	c, g, p, budget := dynAckSourceFixture(t)
	m, err := NewASTManager(ASTManagerConfig{Symbols: c.SymbolsField, Numerals: c.NumeralsField, Budget: budget})
	paramsMust(t, err)
	defer func() { paramsMust(t, m.Close()) }()
	_, err = m.RegisterArithmeticPlugin()
	paramsMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	paramsMust(t, err)
	_, err = m.RegisterFloatingPointPlugin()
	paramsMust(t, err)
	fp := m.floatingPointPlugin
	significand := unumeral.NewInt(4)
	var value unumeral.ArbitraryFloat
	paramsMust(t, fp.numeric.SetComponents(&value, 8, 4, true, 3, &significand))
	finite, err := fp.mkNumeral(&value, nil)
	paramsMust(t, err)
	paramsMust(t, fp.numeric.Del(&value))
	zero, err := fp.mkSpecialNumeral(8, 4, astFloatPZero)
	paramsMust(t, err)
	zeroClass, err := fp.isNumeralClass(zero, astFloatPZero)
	paramsMust(t, err)
	if !zeroClass {
		t.Fatal("mkSpecialNumeral positive zero lost its source class")
	}
	nan, err := fp.mkSpecialNumeral(8, 4, astFloatNaN)
	paramsMust(t, err)
	out, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: budget})
	paramsMust(t, err)
	defer func() { paramsMust(t, out.Close()) }()
	for _, tc := range []struct {
		literal astNodeRef
		want    string
	}{
		{finite, "(= (fp #b1 #x82 #b100) x)"},
		{zero, "(= (_ +zero 8 4) x)"},
		{nan, "(= (_ NaN 8 4) x)"},
	} {
		literal := m.external(tc.literal)
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
			t.Fatalf("float literal %v: %q, want %q", tc.literal, got, tc.want)
		}
		paramsMust(t, out.ResetText())
	}
}
