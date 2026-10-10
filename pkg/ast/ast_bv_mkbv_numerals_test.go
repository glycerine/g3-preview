// Source: src/ast/rewriter/bv_rewriter.cpp:2642-2652; src/util/rational.h:220-225.
package ast

import (
	"fmt"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestASTBitVectorMkBVEveryActualReservationAndPostfixLifetime(t *testing.T) {
	complete := false
	for ordinal := int64(0); ordinal < 8192 && !complete; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m := bvOffsetRuntimeTestManager(t)
			_, e := m.RegisterArithmeticPlugin()
			astTestMust(t, e)
			_, e = m.RegisterBitVectorPlugin()
			astTestMust(t, e)
			m.Budget().FailAfter(ordinal)
			n, body := NewBitVectorMkBVNumerals(m)
			for i := 0; i < 65 && body == nil; i++ {
				body = n.MultiplyTwo()
				if body == nil {
					body = n.Increment()
				}
				if body == nil && n.placed[2] {
					t.Fatal("discarded postfix return survived full-expression")
				}
			}
			var node ASTHandle
			if body == nil {
				node, body = n.Constant(0, 65)
			}
			m.Budget().FailAfter(-1)
			if body == nil {
				complete = true
				text, e := m.numerals.String(&n.values[0])
				astTestMust(t, e)
				if text != "36893488147419103231" {
					t.Fatal(text)
				}
				// Result takes a reference before caller val/two source destruction.
				out, e := m.NewRef(node)
				astTestMust(t, e)
				defer func() { astTestMust(t, m.DelRef(&out)) }()
				astTestMust(t, n.SourceEnd())
				for _, placed := range n.placed {
					if placed {
						t.Fatal("source locals not destroyed")
					}
				}
			} else if n != nil {
				if e := n.MultiplyTwo(); e == nil {
					t.Fatal("stopped replay")
				}
			}
			if n != nil {
				bodyCause, sourceCause, _ := n.scalar.Causes()
				astTestMust(t, n.Close())
				b, s, _ := n.scalar.Causes()
				if b != bodyCause || s != sourceCause {
					t.Fatal("cleanup changed history")
				}
			}
		})
	}
	if !complete {
		t.Fatal("actual MkBV campaign incomplete")
	}
}

func TestASTBitVectorMkBVPostfixDistinctReturn(t *testing.T) {
	m := bvOffsetRuntimeTestManager(t)
	n, e := NewBitVectorMkBVNumerals(m)
	astTestMust(t, e)
	defer func() { astTestMust(t, n.Close()) }()
	q := m.numerals
	astTestMust(t, q.SetString(&n.values[0], "4294967296"))
	var call unumeral.RationalServiceInvocation
	old, e := q.PostIncWithInvocation(&n.values[0], &call)
	astTestMust(t, e)
	astTestMust(t, call.SourceReady())
	astTestMust(t, call.Cleanup(false))
	astTestMust(t, call.Cleanup(true))
	astTestMust(t, call.EndContext())
	defer func() { astTestMust(t, q.Del(&old)) }()
	text, e := q.String(&old)
	astTestMust(t, e)
	now, e := q.String(&n.values[0])
	astTestMust(t, e)
	if text != "4294967296" || now != "4294967297" {
		t.Fatal(text, now)
	}
}
