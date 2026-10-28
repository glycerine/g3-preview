package ast

// Source: src/ast/rewriter/poly_rewriter_def.h:720-721,743,799,840,869.

import (
	"errors"
	"testing"
)

func TestASTBitVectorCancellationNumeralsExplicitZeroAndExactSubtraction(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	two, e := m.BitVectorNumeralUnsigned(2, 8)
	astTestMust(t, e)
	astTestPin(t, m, two)
	three, e := m.BitVectorNumeralUnsigned(3, 8)
	astTestMust(t, e)
	astTestPin(t, m, three)
	s, e := m.GetSort(two)
	astTestMust(t, e)
	n, e := NewBitVectorPolynomialAdditionNumerals(m)
	astTestMust(t, e)
	defer func() { astTestMust(t, n.Close()) }()
	astTestMust(t, n.BeginCancellation())
	for i := 0; i < 2; i++ {
		if !n.placed[i] {
			t.Fatal("source local absent", i)
		}
	}
	yes, e := n.Read(three)
	astTestMust(t, e)
	if !yes {
		t.Fatal("not numeral")
	}
	astTestMust(t, n.AddConstant())
	yes, e = n.Read(two)
	astTestMust(t, e)
	if !yes {
		t.Fatal("not numeral")
	}
	astTestMust(t, n.SubtractConstant())
	node, e := n.Constant(s)
	astTestMust(t, e)
	astTestPin(t, m, node)
	one, e := m.BitVectorNumeralUnsigned(1, 8)
	astTestMust(t, e)
	astTestPin(t, m, one)
	if node != one {
		t.Fatal("wrong constant")
	}
	astTestMust(t, n.BeginCoefficients())
	yes, e = n.Read(three)
	astTestMust(t, e)
	astTestMust(t, n.AppendCoefficient())
	yes, e = n.Read(two)
	astTestMust(t, e)
	astTestMust(t, n.SubtractCoefficient(0))
	astTestMust(t, n.LoadCoefficient(0))
	matched, e := n.CoefficientIsOne()
	astTestMust(t, e)
	if !matched {
		t.Fatal("wrong compound coefficient")
	}
	astTestMust(t, n.Negate(true))
	matched, e = n.CoefficientIsMinusOne()
	astTestMust(t, e)
	if !matched {
		t.Fatal("wrong negated coefficient")
	}
	node, e = n.CoefficientConstant(s)
	astTestMust(t, e)
	astTestPin(t, m, node)
	allOne, e := m.BitVectorNumeralUnsigned(255, 8)
	astTestMust(t, e)
	astTestPin(t, m, allOne)
	if node != allOne {
		t.Fatal("wrong coefficient numeral")
	}
	if e = n.SourceEnd(); !errors.Is(e, ErrASTPending) {
		t.Fatal("outer locals ended before coeffs", e)
	}
	astTestMust(t, n.EndCoefficients())
	astTestMust(t, n.SourceEnd())
}
