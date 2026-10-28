package ast

// Source: src/ast/rewriter/poly_rewriter_def.h:530-534,570-587,606-609.

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"testing"
)

func TestASTBitVectorPolynomialAdditionNumeralsPhysicalSlotsAndSourceEnd(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	s, e := m.BitVectorSort(8)
	astTestMust(t, e)
	n, e := NewBitVectorPolynomialAdditionNumerals(m)
	astTestMust(t, e)
	defer func() { astTestMust(t, n.Close()) }()
	astTestMust(t, n.Begin())
	c, e := m.BitVectorNumeralUnsigned(257, 8)
	astTestMust(t, e)
	astTestPin(t, m, c)
	yes, e := n.Read(c)
	astTestMust(t, e)
	if !yes {
		t.Fatal("coefficient read")
	}
	astTestMust(t, n.AddConstant())
	astTestMust(t, n.Normalize(s, false))
	astTestMust(t, n.BeginCoefficients())
	astTestMust(t, n.AppendCoefficient())
	astTestMust(t, n.AccumulateCoefficient(0))
	astTestMust(t, n.LoadCoefficient(0))
	astTestMust(t, n.Normalize(s, true))
	text, e := m.numerals.String(&n.values[1])
	astTestMust(t, e)
	if text != "2" {
		t.Fatal(text)
	}
	if &n.values[0] == &n.values[1] {
		t.Fatal("aliased scalar locals")
	}
	if e = n.SourceEnd(); !errors.Is(e, ErrASTPending) {
		t.Fatal("source returned with active coefficient buffer", e)
	}
	astTestMust(t, n.EndCoefficients())
	astTestMust(t, n.SourceEnd())
	if n.placed[0] || n.placed[1] || !n.ended {
		t.Fatal("source locals retained")
	}
	if e = n.Begin(); !errors.Is(e, ErrASTPending) {
		t.Fatal("source scope reused", e)
	}
	astTestMust(t, n.Close())
	astTestMust(t, n.Close())
}
func TestASTBitVectorPolynomialAdditionNumeralsHeaderAndCopyGuards(t *testing.T) {
	m := astTestManager(t)
	before := m.Budget().Used()
	m.Budget().FailAfter(0)
	n, e := NewBitVectorPolynomialAdditionNumerals(m)
	m.Budget().FailAfter(-1)
	if n != nil || !errors.Is(e, arena.ErrInjectedFailure) || m.Budget().Used() != before {
		t.Fatal("header refusal", n, e)
	}
	n, e = NewBitVectorPolynomialAdditionNumerals(m)
	astTestMust(t, e)
	copied := *n
	if e = copied.Begin(); !errors.Is(e, ErrASTHandle) {
		t.Fatal("copied owner accepted", e)
	}
	astTestMust(t, n.Close())
	if m.Budget().Used() != before {
		t.Fatal("unentered owner retained")
	}
}
