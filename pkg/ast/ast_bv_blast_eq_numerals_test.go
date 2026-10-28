package ast

// Source: src/ast/rewriter/bv_rewriter.cpp:2287-2302.

import (
	"errors"
	"testing"
)

func TestASTBlastEqualityPhysicalDivisionAndRemainderLifetimes(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	arg, e := m.BitVectorNumeralUnsigned(5, 8)
	astTestMust(t, e)
	astTestPin(t, m, arg)
	n, e := NewBitVectorBlastEqualityNumerals(m)
	astTestMust(t, e)
	defer func() { astTestMust(t, n.Close()) }()
	if !n.placed[0] || n.placed[1] {
		t.Fatal("default v placement")
	}
	width := uint32(99)
	yes, e := n.Read(arg, &width)
	astTestMust(t, e)
	if !yes || width != 8 {
		t.Fatal(yes, width)
	}
	astTestMust(t, n.BeginTwo())
	copy := *n
	if e = copy.Divide(); !errors.Is(e, ErrASTHandle) {
		t.Fatal(e)
	}
	original := n.values[0]
	zero, e := n.RemainderIsZero()
	astTestMust(t, e)
	if zero || n.placed[2] || !n.placed[1] || !n.placed[0] {
		t.Fatal("remainder/outer lifetime", zero, n.placed)
	}
	if n.values[0] != original {
		t.Fatal("remainder mutated v")
	}
	node, e := n.Bit(zero)
	astTestMust(t, e)
	astTestPin(t, m, node)
	astTestMust(t, m.BitVectorNumeralInvocationSourceReady(&n.call))
	if !n.call.receipt.retired {
		t.Fatal("unsigned literal survived numeral return")
	}
	astTestMust(t, n.Divide())
	text, e := m.numerals.String(&n.values[0])
	astTestMust(t, e)
	if text != "2" {
		t.Fatal(text)
	}
	zero, e = n.RemainderIsZero()
	astTestMust(t, e)
	if !zero || n.placed[2] {
		t.Fatal(zero)
	}
	astTestMust(t, n.SourceEnd())
	for _, placed := range n.placed {
		if placed {
			t.Fatal("source scope retained")
		}
	}
	if e = n.Divide(); !errors.Is(e, ErrASTPending) {
		t.Fatal("ended replay", e)
	}
}
