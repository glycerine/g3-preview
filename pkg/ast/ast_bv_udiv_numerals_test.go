// Source: src/ast/rewriter/bv_rewriter.cpp:1034-1099.
package ast

import (
	"fmt"
	"testing"
)

func TestASTBitVectorUnsignedDivisionEveryActualReservation(t *testing.T) {
	for _, mode := range []string{"numeric", "zero", "one", "power", "inverse", "noninvertible"} {
		complete := false
		for ordinal := int64(0); ordinal < 4096 && !complete; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", mode, ordinal), func(t *testing.T) {
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
				lhs := numeral("36893488147419103231")
				divisor := "3"
				switch mode {
				case "zero":
					divisor = "0"
				case "one":
					divisor = "1"
				case "power":
					divisor = "18446744073709551616"
				case "noninvertible":
					divisor = "6"
				}
				rhs := numeral(divisor)
				symbolic := mode == "power" || mode == "inverse" || mode == "noninvertible"
				if symbolic {
					sort, e := m.BitVectorSort(65)
					astTestMust(t, e)
					lhs, e = m.MkVar(0, sort)
					astTestMust(t, e)
					astTestPin(t, m, lhs)
				}
				m.Budget().FailAfter(ordinal)
				n, body := NewBitVectorUnsignedDivisionNumerals(m)
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
				if body == nil && !zero && !one && !symbolic {
					body = n.Normalize(0, width)
				}
				if body == nil && !zero && !one && !symbolic {
					body = n.Quotient()
				}
				var node ASTHandle
				invertible := false
				if body == nil {
					switch {
					case zero:
						body = n.AllOnes(width)
						if body == nil {
							node, body = n.Constant(6, width)
						}
					case one:
						node = lhs
					case !symbolic:
						node, body = n.Constant(3, width)
					default:
						var power bool
						var shift uint32
						power, shift, body = n.PowerOfTwo()
						if body == nil && power {
							var amount ASTHandle
							amount, body = n.Unsigned(shift, width)
							if body == nil {
								node, body = m.MkBitVectorLSHR(lhs, amount)
							}
						} else if body == nil {
							invertible, body = n.Inverse(width)
							if body == nil && invertible {
								var coefficient ASTHandle
								coefficient, body = n.Constant(7, width)
								if body == nil {
									node, body = m.MkBitVectorMul(coefficient, lhs)
								}
							} else if body == nil {
								body = n.End(7)
								if body == nil {
									node, body = m.MkFamilyApp(m.bitVectorPlugin.family, 18, nil, []ASTHandle{lhs, rhs})
								}
							}
						}
					}
				}
				m.Budget().FailAfter(-1)
				if body == nil {
					complete = true
					if n.placed[2] {
						t.Fatal("norm return survived assignment")
					}
					if mode == "numeric" {
						value, e := m.numerals.String(&n.values[3])
						astTestMust(t, e)
						if value != "12297829382473034410" {
							t.Fatal("unsigned quotient", value)
						}
						original, e := m.numerals.String(&n.values[0])
						astTestMust(t, e)
						if original != "36893488147419103231" {
							t.Fatal("unsigned lhs normalization", original)
						}
					}
					if mode == "inverse" {
						value, e := m.numerals.String(&n.values[7])
						astTestMust(t, e)
						if !invertible || value != "12297829382473034411" || !n.placed[7] {
							t.Fatal("inverse through publication", invertible, value)
						}
					}
					if mode == "power" && n.placed[7] {
						t.Fatal("power rule placed optional inverse")
					}
					if mode == "noninvertible" && (invertible || n.placed[7]) {
						t.Fatal("inverse block survived fallback")
					}
					if zero || one {
						value, e := m.numerals.String(&n.values[0])
						astTestMust(t, e)
						if value != "0" {
							t.Fatal("shortcut read lhs", value)
						}
					}
					if zero && (!n.placed[4] || !n.placed[5] || !n.placed[6]) {
						t.Fatal("allone full expression ended before assignment")
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
						t.Fatal("stopped numeric replay")
					}
				}
				if n != nil {
					copy := *n
					if e = copy.Close(); e != ErrASTHandle {
						t.Fatal("copied owner", e)
					}
					b, s, _ := n.scalar.Causes()
					ib, is := n.inverse.receipt.bodyCause, n.inverse.receipt.sourceCause
					astTestMust(t, n.Close())
					astTestMust(t, n.Close())
					for _, call := range []*ASTBitVectorNumeralInvocation{&n.call, &n.norm, &n.inverse} {
						r := &call.receipt
						if r.frame != nil && !r.retired && !r.frame.ownDone {
							t.Fatal("captured numeric child not joined", mode)
						}
					}
					ab, as, _ := n.scalar.Causes()
					if ab != b || as != s || ib != n.inverse.receipt.bodyCause || is != n.inverse.receipt.sourceCause {
						t.Fatal("cleanup changed history")
					}
				}
			})
		}
		if !complete {
			t.Fatal("actual unsigned division campaign incomplete", mode)
		}
	}
}
