package ast

// Source: src/ast/rewriter/poly_rewriter_def.h:667-701.

import (
	"errors"
	"testing"
)

func TestASTBitVectorUnaryMinusPhysicalLocalAndLiteralLifetimes(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	s, e := m.BitVectorSort(8)
	astTestMust(t, e)
	astTestPin(t, m, s)
	n, e := NewBitVectorPolynomialNegationNumerals(m)
	astTestMust(t, e)
	defer func() { astTestMust(t, n.Close()) }()
	astTestMust(t, n.Begin())
	if !n.placed[0] || n.placed[1] {
		t.Fatal("default a not placed alone")
	}
	copy := *n
	if e := copy.SourceEnd(); !errors.Is(e, ErrASTHandle) {
		t.Fatal("copied owner", e)
	}
	x, e := m.MkVar(0, s)
	astTestMust(t, e)
	astTestPin(t, m, x)
	yes, e := n.Read(x)
	astTestMust(t, e)
	if yes {
		t.Fatal("symbolic read matched")
	}
	node, e := n.Product(s, x)
	astTestMust(t, e)
	if !n.placed[0] || !n.placed[1] {
		t.Fatal("factory ended caller locals")
	}
	text, e := m.numerals.String(&n.values[1])
	astTestMust(t, e)
	if text != "-1" {
		t.Fatal("factory mutated caller literal", text)
	}
	out, e := m.NewRef(node)
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	astTestMust(t, n.SourceEnd())
	if n.placed[0] || n.placed[1] || !n.ended {
		t.Fatal("caller source DT incomplete")
	}
}

func TestASTBitVectorSubtractionLiteralRefAcquiredBeforeTemporaryRetires(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	s, e := m.BitVectorSort(8)
	astTestMust(t, e)
	astTestPin(t, m, s)
	n, e := NewBitVectorPolynomialSubtractionLiteral(m)
	astTestMust(t, e)
	defer func() { astTestMust(t, n.Close()) }()
	var out ASTRef
	astTestMust(t, n.ConstructReference(s, &out))
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	if n.placed || !n.ended || out == (ASTRef{}) {
		t.Fatal("construction/literal DT incomplete")
	}
	node, e := m.RefGet(out)
	astTestMust(t, e)
	width, e := m.BitVectorSize(node)
	astTestMust(t, e)
	if width != 8 {
		t.Fatal(width)
	}
	refCount, e := m.GetRefCount(node)
	astTestMust(t, e)
	if refCount != 1 {
		t.Fatal("caller ref missing or duplicated", refCount)
	}
	copy := *n
	if e := copy.Close(); !errors.Is(e, ErrASTHandle) {
		t.Fatal("copied owner", e)
	}
	astTestMust(t, n.SourceReady())
	astTestMust(t, n.Close())
	nodeAgain, e := m.RefGet(out)
	astTestMust(t, e)
	if nodeAgain != node {
		t.Fatal("literal owner cleanup consumed caller ref")
	}
}
