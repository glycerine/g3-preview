package ast

// Source: src/ast/rewriter/bv_rewriter.cpp:2451-2480.

import (
	"errors"
	"testing"
)

func TestASTBitVectorMulEqualityLocalAndReturnLifetimes(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	n, e := NewBitVectorMulEqualityNumerals(m)
	astTestMust(t, e)
	defer func() { astTestMust(t, n.Close()) }()
	if !n.placed[0] || !n.placed[1] || n.placed[2] || n.placed[3] {
		t.Fatal(n.placed)
	}
	c, e := m.BitVectorNumeralUnsigned(3, 8)
	astTestMust(t, e)
	astTestPin(t, m, c)
	width := uint32(0)
	yes, e := n.Read(0, c, &width)
	astTestMust(t, e)
	if !yes || width != 8 {
		t.Fatal(yes, width)
	}
	yes, e = n.Inverse(0, 1, width)
	astTestMust(t, e)
	if !yes {
		t.Fatal("inverse")
	}
	astTestMust(t, n.Place(2))
	astTestMust(t, n.Place(3))
	coefficient, e := m.BitVectorNumeralUnsigned(5, 8)
	astTestMust(t, e)
	astTestPin(t, m, coefficient)
	yes, e = n.Read(3, coefficient, &width)
	astTestMust(t, e)
	if !yes {
		t.Fatal("coefficient")
	}
	astTestMust(t, n.Product(1, 3))
	astTestMust(t, n.NormReturn(4, width))
	astTestMust(t, n.End(7))
	if !n.placed[4] || n.placed[7] {
		t.Fatal("initializer return/product lifetime", n.placed)
	}
	normalized, e := m.numerals.String(&n.values[4])
	astTestMust(t, e)
	if normalized != "87" {
		t.Fatal(normalized)
	}
	astTestMust(t, n.Product(1, 3))
	product, e := m.numerals.String(&n.values[7])
	astTestMust(t, e)
	if product != "855" {
		t.Fatal("product was normalized/reused", product)
	}
	node, e := n.Constant(7, width)
	astTestMust(t, e)
	astTestPin(t, m, node)
	if !n.placed[7] || !n.placed[4] {
		t.Fatal("factory ended caller temporaries")
	}
	copy := *n
	if e = copy.End(7); !errors.Is(e, ErrASTHandle) {
		t.Fatal(e)
	}
	out, e := m.NewRef(node)
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	astTestMust(t, n.End(7))
	if !n.placed[4] {
		t.Fatal("new_c2 ended before block exit")
	}
	astTestMust(t, n.SourceEnd())
	for _, placed := range n.placed {
		if placed {
			t.Fatal("source DT prefix")
		}
	}
	if e = n.Product(0, 1); !errors.Is(e, ErrASTPending) {
		t.Fatal("ended call", e)
	}
}
