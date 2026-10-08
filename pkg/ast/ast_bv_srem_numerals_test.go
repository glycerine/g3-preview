// Source: src/ast/rewriter/bv_rewriter.cpp:1101-1144; src/ast/bv_decl_plugin.cpp:753-765.
package ast

import (
	"fmt"
	"testing"
)

func TestASTBitVectorSignedRemainderEveryActualReservation(t *testing.T) {
	for _, mode := range []string{"negative-positive", "positive-negative", "negative-negative", "zero", "one"} {
		complete := false
		for ordinal := int64(0); ordinal < 4096 && !complete; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", mode, ordinal), func(t *testing.T) {
				m := bvOffsetRuntimeTestManager(t)
				_, e := m.RegisterArithmeticPlugin()
				astTestMust(t, e)
				_, e = m.RegisterBitVectorPlugin()
				astTestMust(t, e)
				text := "18446744073709551616"
				divisorText := "3"
				if mode == "positive-negative" {
					text = "18446744073709551614"
				}
				if mode == "positive-negative" || mode == "negative-negative" {
					divisorText = "36893488147419103229"
				}
				if mode == "zero" {
					divisorText = "0"
				}
				if mode == "one" {
					divisorText = "1"
				}
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
				lhs, rhs := numeral(text), numeral(divisorText)
				m.Budget().FailAfter(ordinal)
				n, body := NewBitVectorSignedRemainderNumerals(m)
				var width uint32
				if body == nil {
					_, body = n.Read(1, rhs, &width)
				}
				if body == nil {
					body = n.Normalize(1, width)
				}
				zero, one := false, false
				if body == nil {
					zero, body = n.IsZero(1)
				}
				if body == nil && !zero {
					one, body = n.IsOne(1)
				}
				if body == nil && !zero && !one {
					_, body = n.Read(0, lhs, &width)
				}
				if body == nil && !zero && !one {
					body = n.Normalize(0, width)
				}
				if body == nil && !zero && !one {
					body = n.Remainder()
				}
				var node ASTHandle
				if body == nil {
					switch {
					case zero:
						node = lhs
					case one:
						node, body = n.Zero(width)
					default:
						node, body = n.Constant(3, width)
					}
				}
				m.Budget().FailAfter(-1)
				if body == nil {
					complete = true
					if n.placed[2] {
						t.Fatal("norm return survived assignment full expression")
					}
					actual, e := m.numerals.String(&n.values[0])
					astTestMust(t, e)
					want := "-18446744073709551616"
					if mode == "positive-negative" {
						want = text
					}
					if zero || one {
						want = "0"
					}
					if actual != want {
						t.Fatal("physical signed lhs", mode, actual, want)
					}
					if !zero && !one {
						divisor, e := m.numerals.String(&n.values[1])
						astTestMust(t, e)
						wantDivisor := "3"
						if mode != "negative-positive" {
							wantDivisor = "-3"
						}
						if divisor != wantDivisor {
							t.Fatal("physical signed divisor", divisor, wantDivisor)
						}
						rem, e := m.numerals.String(&n.values[3])
						astTestMust(t, e)
						wantRem := "-1"
						if mode == "positive-negative" {
							wantRem = "2"
						}
						if rem != wantRem {
							t.Fatal("source rem differs from mod", rem, wantRem)
						}
						if !n.placed[3] {
							t.Fatal("remainder return ended before publication")
						}
					}
					out, e := m.NewRef(node)
					astTestMust(t, e)
					defer func() { astTestMust(t, m.DelRef(&out)) }()
					astTestMust(t, n.SourceEnd())
					for _, placed := range n.placed {
						if placed {
							t.Fatal("local still placed")
						}
					}
				} else if n != nil {
					if _, e = n.IsZero(1); e == nil {
						t.Fatal("stopped replay")
					}
				}
				if n != nil {
					copy := *n
					if e = copy.Close(); e != ErrASTHandle {
						t.Fatal("copied owner", e)
					}
					b, s, _ := n.scalar.Causes()
					nb, ns := n.norm.receipt.bodyCause, n.norm.receipt.sourceCause
					astTestMust(t, n.Close())
					astTestMust(t, n.Close())
					for _, call := range []*ASTBitVectorNumeralInvocation{&n.call, &n.norm} {
						r := &call.receipt
						if r.frame != nil && !r.retired && !r.frame.ownDone {
							t.Fatal("captured numeric child not joined", mode)
						}
					}
					ab, as, _ := n.scalar.Causes()
					if ab != b || as != s || nb != n.norm.receipt.bodyCause || ns != n.norm.receipt.sourceCause {
						t.Fatal("cleanup changed history")
					}
				}
			})
		}
		if !complete {
			t.Fatal("actual signed remainder campaign incomplete", mode)
		}
	}
}
