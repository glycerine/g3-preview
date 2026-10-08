// Source: src/ast/rewriter/bv_rewriter.cpp:2574-2589.
package ast

import (
	"fmt"
	"testing"
)

func TestASTBitVectorRemainderEqualityEveryActualReservation(t *testing.T) {
	complete := false
	for ordinal := int64(0); ordinal < 4096 && !complete; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m := bvOffsetRuntimeTestManager(t)
			_, e := m.RegisterArithmeticPlugin()
			astTestMust(t, e)
			_, e = m.RegisterBitVectorPlugin()
			astTestMust(t, e)
			numeral := func(text string) ASTHandle {
				v, e := m.numerals.NewValue()
				astTestMust(t, e)
				astTestMust(t, m.numerals.SetString(&v, text))
				h, e := m.BitVectorNumeral(&v, 65)
				astTestMust(t, e)
				astTestPin(t, m, h)
				astTestMust(t, m.numerals.Del(&v))
				return h
			}
			divisor := numeral("36893488147419103231")
			rhs := numeral("18446744073709551616")
			m.Budget().FailAfter(ordinal)
			n, body := NewBitVectorRemainderEqualityNumerals(m)
			var width uint32
			var yes bool
			if body == nil {
				yes, body = n.Read(1, rhs, &width)
				if body == nil && !yes {
					t.Fatal("rhs")
				}
			}
			if body == nil {
				yes, body = n.Read(0, divisor, &width)
				if body == nil && !yes {
					t.Fatal("divisor")
				}
			}
			if body == nil {
				yes, body = n.Outside()
				if body == nil && yes {
					t.Fatal("rhs outside divisor")
				}
			}
			if body == nil {
				yes, body = n.ReachesModulus(width)
				if body == nil && !yes {
					t.Fatal("sum below modulus")
				}
			}
			m.Budget().FailAfter(-1)
			if body == nil {
				complete = true
				if n.placed[2] || n.placed[3] || !n.placed[0] || !n.placed[1] {
					t.Fatal("condition/local lifetime", n.placed)
				}
				text, e := m.numerals.String(&n.values[0])
				astTestMust(t, e)
				if text != "36893488147419103231" {
					t.Fatal("divisor mutated", text)
				}
				astTestMust(t, n.SourceEnd())
			} else if n != nil {
				if _, e = n.Read(1, rhs, &width); e == nil {
					t.Fatal("failed numerical call replayed")
				}
			}
			if n != nil {
				historyBody, historySource, _ := n.scalar.Causes()
				astTestMust(t, n.Close())
				afterBody, afterSource, _ := n.scalar.Causes()
				if afterBody != historyBody || afterSource != historySource {
					t.Fatal("cleanup changed scalar source history")
				}
			}
		})
	}
	if !complete {
		t.Fatal("actual numerical campaign incomplete")
	}
}
