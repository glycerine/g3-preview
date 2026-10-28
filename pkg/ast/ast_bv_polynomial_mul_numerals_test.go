package ast

// Source: src/ast/rewriter/poly_rewriter_def.h:202-209,236,259-263.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestASTBitVectorPolynomialMultiplicationNumeralsPlacementAndShadowLifetime(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	c, e := m.BitVectorNumeralUnsigned(2, 8)
	astTestMust(t, e)
	astTestPin(t, m, c)
	n, e := NewBitVectorPolynomialMultiplicationNumerals(m)
	astTestMust(t, e)
	defer func() { astTestMust(t, n.Close()) }()
	astTestMust(t, n.Place(0))
	yes, e := n.Read(c, 0)
	astTestMust(t, e)
	if !yes {
		t.Fatal("first coefficient read")
	}
	if n.placed[1] || n.placed[2] {
		t.Fatal("cheap read constructed later locals")
	}
	astTestMust(t, n.SetOne())
	astTestMust(t, n.Multiply(0))
	astTestMust(t, n.Place(2))
	astTestMust(t, n.DebugAlreadySimplified(c))
	if n.placed[3] || !n.placed[2] {
		t.Fatal("shadow ended outer c_prime")
	}
	one, e := n.One(1)
	astTestMust(t, e)
	if one {
		t.Fatal("physical coefficient receiver not multiplied")
	}
	astTestMust(t, n.SourceEnd())
	for _, placed := range n.placed {
		if placed {
			t.Fatal("source local retained")
		}
	}
	astTestMust(t, n.Close())
	astTestMust(t, n.Close())
}
func TestASTBitVectorPolynomialMultiplicationNumeralsHeaderAndCopyGuards(t *testing.T) {
	m := astTestManager(t)
	before := m.Budget().Used()
	m.Budget().FailAfter(0)
	n, e := NewBitVectorPolynomialMultiplicationNumerals(m)
	m.Budget().FailAfter(-1)
	if n != nil || !errors.Is(e, arena.ErrInjectedFailure) || m.Budget().Used() != before {
		t.Fatal("header refusal", e)
	}
	n, e = NewBitVectorPolynomialMultiplicationNumerals(m)
	astTestMust(t, e)
	copied := *n
	if e = copied.Place(0); !errors.Is(e, ErrASTHandle) {
		t.Fatal("copied owner accepted", e)
	}
	astTestMust(t, n.Close())
	if m.Budget().Used() != before {
		t.Fatal("unentered owner retained")
	}
}
