package ast

// Source: src/ast/rewriter/bv_rewriter.cpp:1185-1280.

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	"testing"
)

func TestASTBitVectorUnsignedRemainderEveryActualReservation(t *testing.T) {
	for _, mode := range []string{"numeric", "zero", "one", "power", "allones", "symbolic"} {
		complete := false
		for ordinal := int64(0); ordinal < 1024 && !complete; ordinal++ {
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
				text := "3"
				switch mode {
				case "zero":
					text = "0"
				case "one":
					text = "1"
				case "power":
					text = "18446744073709551616"
				}
				rhs := numeral(text)
				sort, e := m.BitVectorSort(65)
				astTestMust(t, e)
				x, e := m.MkVar(0, sort)
				astTestMust(t, e)
				astTestPin(t, m, x)
				y, e := m.MkVar(1, sort)
				astTestMust(t, e)
				astTestPin(t, m, y)
				if mode == "power" || mode == "allones" {
					lhs = x
				}
				if mode == "allones" || mode == "symbolic" {
					rhs = y
				}
				m.Budget().FailAfter(ordinal)
				n, body := NewBitVectorUnsignedRemainderNumerals(m)
				var width uint32
				first, second := false, false
				if body == nil {
					first, body = n.Read(0, lhs, &width)
				}
				if body == nil {
					second, body = n.Read(1, rhs, &width)
				}
				if body == nil && second {
					body = n.Normalize(1, width)
				}
				zero, one := false, false
				if body == nil && second {
					zero, body = n.IsZero(1)
				}
				if body == nil && second && !zero {
					one, body = n.IsOne(1)
				}
				var node ASTHandle
				if body == nil {
					switch {
					case zero:
						node = lhs
					case one:
						node, body = n.Unsigned(0, width)
					case first && second:
						body = n.Normalize(0, width)
						if body == nil {
							body = n.Remainder()
						}
						if body == nil {
							node, body = n.Constant(0, width)
						}
					case mode == "power":
						var power bool
						var shift uint32
						power, shift, body = n.PowerOfTwo()
						if body == nil && (!power || shift != 64) {
							t.Fatal("large power predicate", power, shift)
						}
						var z, part ASTHandle
						if body == nil {
							z, body = n.Unsigned(0, width-shift)
						}
						if body == nil {
							part, body = m.MkBitVectorExtract(shift-1, 0, lhs)
						}
						if body == nil {
							node, body = m.MkBitVectorConcat([]ASTHandle{z, part})
						}
					case mode == "allones":
						// The x-minus-one low branch's declaration ends this full expression
						// after NUM construction and before the later conditional factories.
						body = n.AllOnes(65)
						if body == nil {
							node, body = n.Constant(5, 65)
						}
						if body == nil {
							body = n.End(5, 4, 3)
						}
					default:
						node, body = m.MkFamilyApp(m.bitVectorPlugin.family, 20, nil, []ASTHandle{lhs, rhs})
					}
				}
				m.Budget().FailAfter(-1)
				if body == nil {
					complete = true
					if n.placed[2] {
						t.Fatal("norm return survived assignment")
					}
					if mode == "numeric" {
						got, e := m.numerals.String(&n.values[0])
						astTestMust(t, e)
						if got != "1" {
							t.Fatal("physical remainder", got)
						}
					}
					if mode == "zero" || mode == "one" || mode == "symbolic" {
						got, e := m.numerals.String(&n.values[0])
						astTestMust(t, e)
						if got != "36893488147419103231" {
							t.Fatal("initial lhs copy skipped", mode, got)
						}
					}
					if mode == "allones" {
						for i := 3; i < 6; i++ {
							if n.placed[i] {
								t.Fatal("declaration full expression survived", i)
							}
						}
					}
					out, e := m.NewRef(node)
					astTestMust(t, e)
					defer func() { astTestMust(t, m.DelRef(&out)) }()
					astTestMust(t, n.SourceEnd())
					for _, placed := range n.placed {
						if placed {
							t.Fatal("local survived source return")
						}
					}
				} else {
					var f umemory.Failure
					if !errors.Is(body, arena.ErrInjectedFailure) && (!errors.As(body, &f) || f.Kind != umemory.FailureMemout) {
						t.Fatal(body)
					}
					if n != nil && n.stopped {
						if _, e = n.IsZero(0); e != ErrASTPending {
							t.Fatal("stopped replay", e)
						}
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
					astTestMust(t, n.Close())
					ab, as, _ := n.scalar.Causes()
					if b != ab || s != as || nb != n.norm.receipt.bodyCause || ns != n.norm.receipt.sourceCause || cb != n.call.receipt.bodyCause || cs != n.call.receipt.sourceCause {
						t.Fatal("cleanup changed child history")
					}
					for _, call := range []*ASTBitVectorNumeralInvocation{&n.norm, &n.call} {
						r := &call.receipt
						if r.frame != nil && !r.retired && !r.frame.ownDone {
							t.Fatal("captured child not joined")
						}
					}
				}
			})
		}
		if !complete {
			t.Fatal("actual unsigned remainder campaign incomplete", mode)
		}
	}
}
