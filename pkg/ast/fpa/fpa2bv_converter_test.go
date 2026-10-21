package fpa_test

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/ast/fpa"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func fixture(t *testing.T) (*ast.ASTManager, *fpa.Converter) {
	t.Helper()
	b := arena.NewBudget(1 << 30)
	s, err := usymbol.NewManager(usymbol.Config{Budget: b})
	must(t, err)
	n, err := unumeral.NewRationalService(unumeral.RationalServiceConfig{Integers: unumeral.IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	must(t, err)
	m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: s, Numerals: n, Budget: b})
	must(t, err)
	c, err := fpa.NewConverter(m)
	must(t, err)
	t.Cleanup(func() {
		must(t, c.Close())
		must(t, m.Close())
		must(t, n.Close())
		must(t, s.Close())
		if b.Used() != 0 {
			t.Fatalf("retained %d bytes", b.Used())
		}
	})
	return m, c
}

func TestConverterSortAccessSpecialsSplitAndJoin(t *testing.T) {
	m, c := fixture(t)
	sort, err := m.FloatingPointSort(40, 65)
	must(t, err)
	ebits, sbits, matched, err := m.FloatingPointSortShape(sort)
	must(t, err)
	if !matched || ebits != 40 || sbits != 65 {
		t.Fatal("floating-point sort shape", matched, ebits, sbits)
	}
	rmSort, err := m.RoundingModeSort()
	must(t, err)
	if yes, e := m.IsRoundingModeSort(rmSort); e != nil || !yes {
		t.Fatal("rounding-mode sort", yes, e)
	}

	nan, err := c.MkSpecial(sort, fpa.NaN)
	must(t, err)
	sign, exponent, significand, err := c.SplitFP(nan)
	must(t, err)
	zero1, err := m.BitVectorNumeralUnsigned(0, 1)
	must(t, err)
	top40, err := m.BitVectorNumeralAllOnes(40)
	must(t, err)
	one64, err := m.BitVectorNumeralUnsigned(1, 64)
	must(t, err)
	if sign != zero1 || exponent != top40 || significand != one64 {
		t.Fatal("canonical NaN components", sign, exponent, significand)
	}
	joined, err := c.JoinFP(nan)
	must(t, err)
	want, err := m.MkBitVectorConcat([]ast.ASTHandle{zero1, top40, one64})
	must(t, err)
	if joined != want {
		t.Fatal("joined IEEE field order", joined, want)
	}
}

func TestConverterNaNEqualityDistinctAndComponentITE(t *testing.T) {
	m, c := fixture(t)
	sort, err := m.FloatingPointSort(8, 24)
	must(t, err)
	first, err := c.MkSpecial(sort, fpa.NaN)
	must(t, err)
	one, err := m.BitVectorNumeralUnsigned(1, 1)
	must(t, err)
	top, err := m.BitVectorNumeralAllOnes(8)
	must(t, err)
	payload, err := m.BitVectorNumeralUnsigned(7, 23)
	must(t, err)
	second, err := c.MkFP(one, top, payload)
	must(t, err)
	equal, err := c.MkEq(first, second)
	must(t, err)
	if !m.IsTrue(equal) {
		t.Fatal("all NaN payloads must denote the single SMT NaN", equal)
	}
	distinct, err := c.MkDistinct([]ast.ASTHandle{first, second})
	must(t, err)
	if !m.IsFalse(distinct) {
		t.Fatal("distinct NaN NaN must be false", distinct)
	}

	positiveZero, err := c.MkSpecial(sort, fpa.PositiveZero)
	must(t, err)
	negativeZero, err := c.MkSpecial(sort, fpa.NegativeZero)
	must(t, err)
	equal, err = c.MkEq(positiveZero, negativeZero)
	must(t, err)
	if !m.IsFalse(equal) {
		t.Fatal("SMT equality preserves the zero sign", equal)
	}
	truth, err := m.MkTrue()
	must(t, err)
	choice, err := c.MkITE(truth, positiveZero, negativeZero)
	must(t, err)
	if choice != positiveZero {
		t.Fatal("component ITE true branch", choice, positiveZero)
	}
}

func TestConverterRoundingModeCodesAndEquality(t *testing.T) {
	m, c := fixture(t)
	kinds := []int32{
		ast.FPAOpRMNearestEven,
		ast.FPAOpRMNearestAway,
		ast.FPAOpRMTowardPositive,
		ast.FPAOpRMTowardNegative,
		ast.FPAOpRMTowardZero,
	}
	for code, kind := range kinds {
		rm, err := c.MkRoundingMode(kind)
		must(t, err)
		bits, err := m.GetArg(rm, 0)
		must(t, err)
		want, err := m.BitVectorNumeralUnsigned(uint32(code), 3)
		must(t, err)
		if bits != want {
			t.Fatalf("rounding mode %d: got %v want %v", kind, bits, want)
		}
		equal, err := c.MkEq(rm, rm)
		must(t, err)
		if !m.IsTrue(equal) {
			t.Fatal("rounding mode reflexivity", kind, equal)
		}
	}
}
