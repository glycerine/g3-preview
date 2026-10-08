// Source: src/ast/rewriter/bv_rewriter.cpp:792-889.
package ast

import (
	"fmt"
	"testing"
)

func TestASTBitVectorLogicalShiftsEveryActualReservation(t *testing.T) {
	for _, mode := range []string{"wide-left", "wide-right", "narrow-left", "narrow-right", "large", "zero"} {
		complete := false
		for ordinal := int64(0); ordinal < 4096 && !complete; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", mode, ordinal), func(t *testing.T) {
				m := bvOffsetRuntimeTestManager(t)
				_, e := m.RegisterArithmeticPlugin()
				astTestMust(t, e)
				_, e = m.RegisterBitVectorPlugin()
				astTestMust(t, e)
				v, e := m.numerals.NewValue()
				astTestMust(t, e)
				width := uint32(65)
				text := "36893488147419103231"
				if mode == "narrow-left" || mode == "narrow-right" {
					width = 64
					text = "18446744073709551615"
				}
				astTestMust(t, m.numerals.SetString(&v, text))
				input, e := m.BitVectorNumeral(&v, width)
				astTestMust(t, e)
				astTestPin(t, m, input)
				astTestMust(t, m.numerals.Del(&v))
				amount, e := m.BitVectorNumeralUnsigned(1, width)
				astTestMust(t, e)
				astTestPin(t, m, amount)
				if mode == "large" {
					amount = input
				}
				if mode == "zero" {
					amount, e = m.BitVectorNumeralUnsigned(0, width)
					astTestMust(t, e)
					astTestPin(t, m, amount)
				}
				m.Budget().FailAfter(ordinal)
				n, body := NewBitVectorLogicalShiftNumerals(m)
				var stored uint32
				if body == nil {
					_, body = n.Read(1, amount, &stored)
				}
				zero, large := false, false
				if body == nil {
					zero, body = n.ZeroAmount()
				}
				if body == nil && !zero {
					large, body = n.AtLeastWidth(width)
				}
				if body == nil && !zero && !large {
					_, body = n.Read(0, input, &stored)
				}
				if body == nil && !zero && !large {
					left := mode == "wide-left" || mode == "narrow-left"
					if width > 64 {
						body = n.Wide(width, left)
					} else {
						body = n.Narrow(width, left)
					}
				}
				var node ASTHandle
				if body == nil {
					switch {
					case zero:
						node = input
					case large:
						node, body = n.Zero(width)
					default:
						slot := 0
						if width <= 64 {
							slot = 3
						}
						node, body = n.Constant(slot, width)
					}
				}
				m.Budget().FailAfter(-1)
				if body == nil {
					complete = true
					if zero || large {
						val, e := m.numerals.String(&n.values[0])
						astTestMust(t, e)
						if val != "0" {
							t.Fatal("unreached lhs read", val)
						}
					} else {
						slot := 0
						if width <= 64 {
							slot = 3
						}
						val, e := m.numerals.String(&n.values[slot])
						astTestMust(t, e)
						want := "36893488147419103230"
						if mode == "wide-right" {
							want = "18446744073709551615"
						}
						if mode == "narrow-left" {
							want = "18446744073709551614"
						}
						if mode == "narrow-right" {
							want = "9223372036854775807"
						}
						if val != want {
							t.Fatal("physical result", mode, val, want)
						}
						if n.placed[4] || n.placed[5] || width > 64 && n.placed[3] {
							t.Fatal("wide/normalization returns survived full expression")
						}
						if width <= 64 && !n.placed[3] {
							t.Fatal("rn ended before publication")
						}
						if mode == "narrow-left" || mode == "narrow-right" {
							original, e := m.numerals.String(&n.values[0])
							astTestMust(t, e)
							if original != text {
								t.Fatal("narrow modified r1", original)
							}
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
					if _, e = n.ZeroAmount(); e == nil {
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
					cb, cs := n.call.receipt.bodyCause, n.call.receipt.sourceCause
					astTestMust(t, n.Close())
					for _, call := range []*ASTBitVectorNumeralInvocation{&n.call, &n.norm} {
						r := &call.receipt
						if r.frame != nil && !r.retired && !r.frame.ownDone {
							t.Fatal("captured numeric child not joined", mode)
						}
					}

					astTestMust(t, n.Close())
					afterB, afterS, _ := n.scalar.Causes()
					if b != afterB || s != afterS || nb != n.norm.receipt.bodyCause || ns != n.norm.receipt.sourceCause || cb != n.call.receipt.bodyCause || cs != n.call.receipt.sourceCause {
						t.Fatal("cleanup changed history")
					}
				}
			})
		}
		if !complete {
			t.Fatal("incomplete actual shift campaign", mode)
		}
	}
}
