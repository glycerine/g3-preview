package ast

import (
	"testing"

	"github.com/glycerine/g3/pkg/math/polynomial"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	uparams "github.com/glycerine/g3/pkg/util/params"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func TestASTSMT2ArithmeticLiteralDecimalAndAlgebraicProfiles(t *testing.T) {
	c, g, p, budget := dynAckSourceFixture(t)
	previous, err := uparams.BindProcessGlobalParamsService(g)
	paramsMust(t, err)
	defer func() {
		_, e := uparams.BindProcessGlobalParamsService(previous)
		paramsMust(t, e)
	}()
	m, err := NewASTManager(ASTManagerConfig{Symbols: c.SymbolsField, Numerals: c.NumeralsField, Budget: budget})
	paramsMust(t, err)
	defer func() { paramsMust(t, m.Close()) }()
	_, err = m.RegisterArithmeticPlugin()
	paramsMust(t, err)
	truth, err := m.MkTrue()
	paramsMust(t, err)
	if _, matched, e := m.SMT2PPArithLiteral(truth, false, 10); e != nil || matched {
		t.Fatalf("non-arithmetic constant: matched=%v error=%v", matched, e)
	}
	out, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: budget})
	paramsMust(t, err)
	defer func() { paramsMust(t, out.Close()) }()
	render := func(h ASTHandle, decimal bool, precision uint32) string {
		format, matched, e := m.SMT2PPArithLiteral(h, decimal, precision)
		paramsMust(t, e)
		if !matched {
			t.Fatal("arithmetic literal was not recognized")
		}
		paramsMust(t, m.PrintFormat(out, format, c, p, g))
		text, e := out.String()
		paramsMust(t, e)
		paramsMust(t, out.ResetText())
		return text
	}

	value, err := c.NumeralsField.NewValue()
	paramsMust(t, err)
	defer func() { paramsMust(t, c.NumeralsField.CleanupValue(&value)) }()
	for _, tc := range []struct {
		value string
		want  string
	}{
		{"1/3", "0.333?"},
		{"-1/3", "(- 0.333?)"},
		{"1/2", "0.5"},
	} {
		paramsMust(t, c.NumeralsField.SetString(&value, tc.value))
		h, e := m.ArithmeticNumeral(&value, false)
		paramsMust(t, e)
		if got := render(h, true, 3); got != tc.want {
			t.Fatalf("decimal %s: %q, want %q", tc.value, got, tc.want)
		}
	}
	paramsMust(t, c.NumeralsField.SetString(&value, "-1/3"))
	literal, err := m.ArithmeticNumeral(&value, false)
	paramsMust(t, err)
	format, err := m.SMT2PPExprTreeWithConfig(literal, nil, SMT2PPExprConfig{Decimal: true, DecimalPrecision: 3})
	paramsMust(t, err)
	paramsMust(t, m.PrintFormat(out, format, c, p, g))
	printed, err := out.String()
	paramsMust(t, err)
	if printed != "(- 0.333?)" {
		t.Fatalf("configured expression decimal: %q", printed)
	}
	paramsMust(t, out.ResetText())
	realSort, err := m.GetSort(literal)
	paramsMust(t, err)
	variable, err := m.MkVar(0, realSort)
	paramsMust(t, err)
	equality, err := m.MkEq(variable, literal)
	paramsMust(t, err)
	quantifier, err := m.MkQuantifier(ASTQuantifierSpec{
		Forall: true,
		Sorts:  []ASTHandle{realSort},
		Names:  []usymbol.Symbol{astTestSymbol(t, m, "x")},
		Body:   equality,
	})
	paramsMust(t, err)
	format, err = m.SMT2PPBooleanTreeWithConfig(quantifier, nil, SMT2PPExprConfig{Decimal: true, DecimalPrecision: 3})
	paramsMust(t, err)
	paramsMust(t, m.PrintFormat(out, format, c, p, g))
	printed, err = out.String()
	paramsMust(t, err)
	if printed != "(forall ((x Real)) (= x (- 0.333?)))" {
		t.Fatalf("configured quantified decimal: %q", printed)
	}
	paramsMust(t, out.ResetText())

	am, err := m.ArithmeticAlgebraicManager()
	paramsMust(t, err)
	two := unumeral.NewInt(2)
	var basic, root polynomial.Algebraic
	paramsMust(t, am.SetInteger(&basic, &two))
	paramsMust(t, am.Root(&basic, 2, &root))
	paramsMust(t, am.Del(&basic))
	positive, err := m.ArithmeticAlgebraicNumeral(&root)
	paramsMust(t, err)
	paramsMust(t, am.Neg(&root))
	negative, err := m.ArithmeticAlgebraicNumeral(&root)
	paramsMust(t, err)
	paramsMust(t, am.Del(&root))
	for _, tc := range []struct {
		handle             ASTHandle
		exact, approximate string
	}{
		{positive, "(root-obj (+ (^ x 2) (- 2)) 2)", "1.4142?"},
		{negative, "(root-obj (+ (^ x 2) (- 2)) 1)", "(- 1.4142?)"},
	} {
		if got := render(tc.handle, false, 4); got != tc.exact {
			t.Fatalf("exact algebraic: %q, want %q", got, tc.exact)
		}
		if got := render(tc.handle, true, 4); got != tc.approximate {
			t.Fatalf("decimal algebraic: %q, want %q", got, tc.approximate)
		}
	}
}

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

func TestASTSMT2DatalogFiniteNumeral(t *testing.T) {
	c, g, p, budget := dynAckSourceFixture(t)
	m, err := NewASTManager(ASTManagerConfig{Symbols: c.SymbolsField, Numerals: c.NumeralsField, Budget: budget})
	paramsMust(t, err)
	defer func() { paramsMust(t, m.Close()) }()
	sort, err := m.MkDatalogFiniteSort(astTestSymbol(t, m, "finite"), 8)
	paramsMust(t, err)
	literal, err := m.MkDatalogNumeral(7, sort)
	paramsMust(t, err)
	format, err := m.SMT2PPExprTree(literal, nil)
	paramsMust(t, err)
	out, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: budget})
	paramsMust(t, err)
	defer func() { paramsMust(t, out.Close()) }()
	paramsMust(t, m.PrintFormat(out, format, c, p, g))
	got, err := out.String()
	paramsMust(t, err)
	if got != "7" {
		t.Fatalf("finite-domain numeral: %q, want 7", got)
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
