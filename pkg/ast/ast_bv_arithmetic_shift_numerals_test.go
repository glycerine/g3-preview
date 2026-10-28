package ast

// Source: src/ast/rewriter/bv_rewriter.cpp:891-985; src/ast/bv_decl_plugin.cpp:767-773.

import (
	"fmt"
	"testing"
)

func TestASTBitVectorArithmeticShiftEveryActualReservation(t *testing.T) {
	for _, mode := range []string{"wide-negative", "wide-positive", "wide-allone", "wide-zero", "narrow-negative", "narrow-positive", "narrow-allone", "zero", "clamp"} {
		complete := false
		for ordinal := int64(0); ordinal < 4096 && !complete; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", mode, ordinal), func(t *testing.T) {
				m := bvOffsetRuntimeTestManager(t)
				_, e := m.RegisterArithmeticPlugin()
				astTestMust(t, e)
				_, e = m.RegisterBitVectorPlugin()
				astTestMust(t, e)
				width := uint32(65)
				text := "18446744073709551616"
				shift := uint32(1)
				switch mode {
				case "wide-positive", "wide-zero":
					text = "18446744073709551615"
				case "narrow-negative", "narrow-allone":
					width = 64
					text = "9223372036854775808"
				case "narrow-positive":
					width = 64
					text = "9223372036854775807"
				case "clamp":
					text = "3"
					shift = 254
				case "zero":
					shift = 0
				}
				if mode == "wide-allone" || mode == "wide-zero" {
					shift = 65
				}
				if mode == "narrow-allone" {
					shift = 64
				}
				v, e := m.numerals.NewValue()
				astTestMust(t, e)
				astTestMust(t, m.numerals.SetString(&v, text))
				lhs, e := m.BitVectorNumeral(&v, width)
				astTestMust(t, e)
				astTestPin(t, m, lhs)
				astTestMust(t, m.numerals.Del(&v))
				rhs, e := m.BitVectorNumeralUnsigned(shift, width)
				astTestMust(t, e)
				astTestPin(t, m, rhs)
				m.Budget().FailAfter(ordinal)
				n, body := NewBitVectorArithmeticShiftNumerals(m)
				var stored uint32
				if body == nil {
					_, body = n.Read(1, rhs, &stored)
				}
				zero := false
				if body == nil {
					zero, body = n.ZeroAmount()
				}
				if body == nil && !zero {
					_, body = n.Read(0, lhs, &stored)
				}
				slot := 0
				zeroFactory := false
				if body == nil && !zero {
					if mode == "clamp" {
						body = n.CombineAndClamp(8)
					} else if width <= 64 {
						body = n.Narrow(width)
						slot = 3
					} else {
						large, e := n.AtLeastWidth(width)
						body = e
						if body == nil && large {
							sign, e := n.HasSign(width)
							body = e
							if body == nil {
								if sign {
									body = n.AllOnes(width)
									slot = 5
								} else {
									zeroFactory = true
								}
							}
						} else if body == nil {
							body = n.Wide(width)
						}
					}
				}
				var node ASTHandle
				if body == nil {
					switch {
					case zero:
						node = lhs
					case zeroFactory:
						node, body = n.Zero(width)
					default:
						outputWidth := width
						if mode == "clamp" {
							outputWidth = 8
						}
						node, body = n.Constant(slot, outputWidth)
					}
				}
				m.Budget().FailAfter(-1)
				if body == nil {
					complete = true
					want := "27670116110564327424"
					switch mode {
					case "wide-positive":
						want = "9223372036854775807"
					case "wide-allone":
						want = "36893488147419103231"
					case "wide-zero":
						want = text
						isZero, e := m.IsBitVectorZero(node)
						astTestMust(t, e)
						if !isZero {
							t.Fatal("oversized positive result", node)
						}
					case "zero":
						want = "0"
					case "narrow-negative":
						want = "13835058055282163712"
					case "narrow-positive":
						want = "4611686018427387903"
					case "narrow-allone":
						want = "18446744073709551615"
					case "clamp":
						want = "8"
					}
					actual, e := m.numerals.String(&n.values[slot])
					astTestMust(t, e)
					if actual != want {
						t.Fatal("source arithmetic shift", mode, actual, want)
					}
					if mode == "wide-negative" && (n.placed[3] || n.placed[4] || n.placed[5]) {
						t.Fatal("padding block or power survived before factory")
					}
					if mode == "wide-allone" && (!n.placed[3] || !n.placed[4] || !n.placed[5]) {
						t.Fatal("allone full expression ended before publication")
					}
					if mode == "narrow-negative" {
						original, e := m.numerals.String(&n.values[0])
						astTestMust(t, e)
						if original != text {
							t.Fatal("narrow modified r1", original)
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
					sb, ss := n.sign.receipt.bodyCause, n.sign.receipt.sourceCause
					astTestMust(t, n.Close())
					astTestMust(t, n.Close())
					for _, call := range []*ASTBitVectorNumeralInvocation{&n.call, &n.sign} {
						r := &call.receipt
						if r.frame != nil && !r.retired && !r.frame.ownDone {
							t.Fatal("captured numeric child not joined", mode)
						}
					}
					ab, as, _ := n.scalar.Causes()
					if ab != b || as != s || sb != n.sign.receipt.bodyCause || ss != n.sign.receipt.sourceCause {
						t.Fatal("cleanup changed history")
					}
				}
			})
		}
		if !complete {
			t.Fatal("actual arithmetic shift campaign incomplete", mode)
		}
	}
}
