// Source: src/ast/rewriter/bv_rewriter.cpp:2190-2203.
package ast

import (
	"errors"
	"testing"
)

func TestASTBitVectorMultiplicationNumeralLocalLifetime(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	node, e := m.BitVectorNumeralUnsigned(16, 8)
	astTestMust(t, e)
	astTestPin(t, m, node)
	n, e := NewBitVectorMultiplicationNumerals(m)
	astTestMust(t, e)
	defer func() { astTestMust(t, n.Close()) }()
	if !n.placed {
		t.Fatal("default v missing")
	}
	copied := *n
	if e = copied.SourceEnd(); !errors.Is(e, ErrASTHandle) {
		t.Fatal(e)
	}
	yes, width, shift, e := n.ReadPower(node)
	astTestMust(t, e)
	if !yes || width != 8 || shift != 4 {
		t.Fatal(yes, width, shift)
	}
	zero, e := n.Zero(shift)
	astTestMust(t, e)
	astTestPin(t, m, zero)
	if !n.placed || n.ended {
		t.Fatal("zero factory ended v")
	}
	text, e := m.numerals.String(&n.value)
	astTestMust(t, e)
	if text != "16" {
		t.Fatal(text)
	}
	astTestMust(t, m.BitVectorNumeralInvocationSourceReady(&n.zero))
	if !n.zero.receipt.retired {
		t.Fatal("zero temporary survived initializer")
	}
	out, e := m.NewRef(zero)
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	astTestMust(t, n.SourceEnd())
	if n.placed || !n.ended {
		t.Fatal("v source DT missing")
	}
	if _, _, _, e = n.ReadPower(node); !errors.Is(e, ErrASTPending) {
		t.Fatal("ended reuse", e)
	}
}
