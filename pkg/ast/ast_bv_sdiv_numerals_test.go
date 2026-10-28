package ast

// Source: src/ast/rewriter/bv_rewriter.cpp:987-1032; src/util/rational.h:186-190.

import (
	"fmt"
	"testing"
)

func TestASTBitVectorSignedDivisionEveryActualReservation(t *testing.T) {
	for _, mode := range []string{"negative-positive", "positive-negative", "negative-negative", "minimum-negative-one", "zero", "one"} {
		complete := false
		for ordinal := int64(0); ordinal < 4096 && !complete; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", mode, ordinal), func(t *testing.T) {
				m := bvOffsetRuntimeTestManager(t)
				_, e := m.RegisterArithmeticPlugin()
				astTestMust(t, e)
				_, e = m.RegisterBitVectorPlugin()
				astTestMust(t, e)
				text, divisorText := "18446744073709551616", "3"
				if mode == "positive-negative" {
					text = "18446744073709551614"
				}
				if mode == "positive-negative" || mode == "negative-negative" {
					divisorText = "36893488147419103229"
				}
				if mode == "minimum-negative-one" {
					divisorText = "36893488147419103231"
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
				n, body := NewBitVectorSignedDivisionNumerals(m)
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
					body = n.Quotient()
				}
				var node ASTHandle
				if body == nil {
					switch {
					case one:
						node = lhs
					case zero:
						var z, yes, cond, all ASTHandle
						z, body = n.Unsigned(0, width)
						if body == nil {
							cond, body = m.MkFamilyApp(m.bitVectorPlugin.family, 27, nil, []ASTHandle{lhs, z})
						}
						if body == nil {
							yes, body = n.Unsigned(1, width)
						}
						if body == nil {
							body = n.AllOnes(width)
						}
						if body == nil {
							all, body = n.Constant(6, width)
						}
						if body == nil {
							node, body = m.MkIte(cond, yes, all)
						}
					default:
						node, body = n.Constant(3, width)
					}
				}
				m.Budget().FailAfter(-1)
				if body == nil {
					complete = true
					if n.placed[2] {
						t.Fatal("norm return survived assignment")
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
						quotient, e := m.numerals.String(&n.values[3])
						astTestMust(t, e)
						wantQ := "-6148914691236517205"
						if mode == "positive-negative" {
							wantQ = "-6148914691236517204"
						}
						if mode == "negative-negative" {
							wantQ = "6148914691236517205"
						}
						if mode == "minimum-negative-one" {
							wantQ = "18446744073709551616"
						}
						if quotient != wantQ {
							t.Fatal("truncating machine quotient", mode, quotient, wantQ)
						}
						if !n.placed[3] {
							t.Fatal("quotient ended before publication")
						}
					}
					if zero {
						for _, i := range []int{4, 5, 6} {
							if !n.placed[i] {
								t.Fatal("hardware full expression ended before publication", i)
							}
						}
						all, e := m.numerals.String(&n.values[6])
						astTestMust(t, e)
						if all != "36893488147419103231" {
							t.Fatal(all)
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
				} else if n != nil && n.stopped {
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
			t.Fatal("actual signed division campaign incomplete", mode)
		}
	}
}
