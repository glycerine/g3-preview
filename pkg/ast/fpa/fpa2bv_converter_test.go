package fpa_test

import (
	"errors"
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

func TestConverterVariablesAndIEEEBitVectorConversions(t *testing.T) {
	m, c := fixture(t)
	sort, err := m.FloatingPointSort(8, 24)
	must(t, err)
	variable, err := c.MkVar(7, sort)
	must(t, err)
	sign, exponent, significand, err := c.SplitFP(variable)
	must(t, err)
	if index, e := m.GetVarIndex(sign); e != nil || index != 7 {
		t.Fatal("sign variable", index, e)
	}
	if index, e := m.GetVarIndex(significand); e != nil || index != 8 {
		t.Fatal("significand variable", index, e)
	}
	if index, e := m.GetVarIndex(exponent); e != nil || index != 9 {
		t.Fatal("exponent variable", index, e)
	}

	packedSort, err := m.BitVectorSort(32)
	must(t, err)
	packed, err := m.MkVar(10, packedSort)
	must(t, err)
	converted, err := c.FromIEEEBV(sort, packed)
	must(t, err)
	sign, exponent, significand, err = c.SplitFP(converted)
	must(t, err)
	wantSign, err := m.MkBitVectorExtract(31, 31, packed)
	must(t, err)
	wantExponent, err := m.MkBitVectorExtract(30, 23, packed)
	must(t, err)
	wantSignificand, err := m.MkBitVectorExtract(22, 0, packed)
	must(t, err)
	if sign != wantSign || exponent != wantExponent || significand != wantSignificand {
		t.Fatal("IEEE split ranges", sign, exponent, significand)
	}

	canonicalNaN, err := c.MkSpecial(sort, fpa.NaN)
	must(t, err)
	one, err := m.BitVectorNumeralUnsigned(1, 1)
	must(t, err)
	top, err := m.BitVectorNumeralAllOnes(8)
	must(t, err)
	payload, err := m.BitVectorNumeralUnsigned(17, 23)
	must(t, err)
	alternateNaN, err := c.MkFP(one, top, payload)
	must(t, err)
	got, err := c.ToIEEEBV(alternateNaN)
	must(t, err)
	want, err := c.JoinFP(canonicalNaN)
	must(t, err)
	if got != want {
		t.Fatal("high-unspecified NaN canonicalization", got, want)
	}
	must(t, c.SetHighUnspecified(false))
	if _, err = c.ToIEEEBV(alternateNaN); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("low-unspecified UF branch must remain explicit", err)
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

func TestConverterClassificationsUnarySignsAndComparisons(t *testing.T) {
	m, c := fixture(t)
	sort, err := m.FloatingPointSort(8, 24)
	must(t, err)
	values := make(map[fpa.SpecialValue]ast.ASTHandle)
	for _, value := range []fpa.SpecialValue{fpa.PositiveInfinity, fpa.NegativeInfinity, fpa.NaN, fpa.PositiveZero, fpa.NegativeZero} {
		values[value], err = c.MkSpecial(sort, value)
		must(t, err)
	}
	zeroSign, err := m.BitVectorNumeralUnsigned(0, 1)
	must(t, err)
	zeroExp, err := m.BitVectorNumeralUnsigned(0, 8)
	must(t, err)
	oneExp, err := m.BitVectorNumeralUnsigned(1, 8)
	must(t, err)
	zeroSig, err := m.BitVectorNumeralUnsigned(0, 23)
	must(t, err)
	oneSig, err := m.BitVectorNumeralUnsigned(1, 23)
	must(t, err)
	subnormal, err := c.MkFP(zeroSign, zeroExp, oneSig)
	must(t, err)
	normal, err := c.MkFP(zeroSign, oneExp, zeroSig)
	must(t, err)

	classify := func(value ast.ASTHandle, kind ast.FPAClassificationKind, want bool) {
		t.Helper()
		got, e := c.Classify(value, kind)
		must(t, e)
		if m.IsTrue(got) != want || m.IsFalse(got) == want {
			t.Fatalf("classification %d: got %v want %v", kind, got, want)
		}
	}
	classify(values[fpa.NaN], ast.FPAIsNaN, true)
	classify(values[fpa.NaN], ast.FPAIsPositive, false)
	classify(values[fpa.PositiveInfinity], ast.FPAIsInfinite, true)
	classify(values[fpa.NegativeInfinity], ast.FPAIsNegative, true)
	classify(values[fpa.PositiveZero], ast.FPAIsZero, true)
	classify(values[fpa.NegativeZero], ast.FPAIsNegative, true)
	classify(subnormal, ast.FPAIsSubnormal, true)
	classify(subnormal, ast.FPAIsNormal, false)
	classify(normal, ast.FPAIsNormal, true)

	abs, err := c.Abs(values[fpa.NegativeZero])
	must(t, err)
	if abs != values[fpa.PositiveZero] {
		t.Fatal("absolute value clears sign", abs, values[fpa.PositiveZero])
	}
	negatedNaN, err := c.Negate(values[fpa.NaN])
	must(t, err)
	if negatedNaN != values[fpa.NaN] {
		t.Fatal("negation preserves NaN sign", negatedNaN, values[fpa.NaN])
	}

	compare := func(kind int32, left, right ast.ASTHandle, want bool) {
		t.Helper()
		got, e := c.Compare(kind, left, right)
		must(t, e)
		if m.IsTrue(got) != want || m.IsFalse(got) == want {
			t.Fatalf("comparison %d: got %v want %v", kind, got, want)
		}
	}
	compare(ast.FPAOpEq, values[fpa.PositiveZero], values[fpa.NegativeZero], true)
	compare(ast.FPAOpEq, values[fpa.NaN], values[fpa.NaN], false)
	compare(ast.FPAOpLt, values[fpa.NegativeInfinity], values[fpa.PositiveInfinity], true)
	compare(ast.FPAOpLt, values[fpa.PositiveInfinity], values[fpa.NegativeInfinity], false)
	compare(ast.FPAOpLt, values[fpa.NegativeZero], values[fpa.PositiveZero], false)
	compare(ast.FPAOpLt, values[fpa.NaN], values[fpa.PositiveInfinity], false)
	compare(ast.FPAOpLe, values[fpa.NegativeZero], values[fpa.PositiveZero], true)
	compare(ast.FPAOpGt, values[fpa.PositiveInfinity], values[fpa.NegativeInfinity], true)
	compare(ast.FPAOpGe, values[fpa.NegativeZero], values[fpa.PositiveZero], true)
}
