// Source: src/ast/rewriter/bv_rewriter.cpp:2212-2236.
package ast

import (
	"errors"
	"testing"
)

func TestASTBit2BoolSubtractionReturnAndLiteralLifetimes(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	input, e := m.BitVectorNumeralUnsigned(1, 1)
	astTestMust(t, e)
	astTestPin(t, m, input)
	n, e := NewBitVectorBit2BoolNumerals(m)
	astTestMust(t, e)
	defer func() { astTestMust(t, n.Close()) }()
	yes, e := n.Read(input)
	astTestMust(t, e)
	if !yes {
		t.Fatal("read")
	}
	node, e := n.LiteralOne(true)
	astTestMust(t, e)
	for _, placed := range n.placed {
		if !placed {
			t.Fatal("full-expression local prematurely ended")
		}
	}
	value, e := m.numerals.String(&n.values[2])
	astTestMust(t, e)
	if value != "0" {
		t.Fatal(value)
	}
	copy := *n
	if e = copy.EndExpression(); !errors.Is(e, ErrASTHandle) {
		t.Fatal(e)
	}
	out, e := m.NewRef(node)
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	astTestMust(t, n.EndExpression())
	if !n.placed[0] || n.placed[1] || n.placed[2] {
		t.Fatal("temporary/v lifetime")
	}
	astTestMust(t, n.SourceEnd())
	if n.placed[0] || !n.ended {
		t.Fatal("outer v DT")
	}
}
