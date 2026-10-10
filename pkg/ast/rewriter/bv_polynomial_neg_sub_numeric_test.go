// Source: src/ast/rewriter/poly_rewriter_def.h:667-701.
package rewriter

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestBVPolynomialNegationSubtractionEveryActualNumericReservation(t *testing.T) {
	for _, op := range []string{"unary", "symbolic", "sub"} {
		complete := false
		failures := 0
		for ordinal := int64(0); ordinal < 256 && !complete; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", op, ordinal), func(t *testing.T) {
				b := arena.NewBudget(1 << 27)
				rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f umemory.Failure) error { return f }})
				astTestMust(t, e)
				q, e := unumeral.NewRationalService(unumeral.RationalServiceConfig{Integers: unumeral.IntegerConfig{Memory: rt, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
				astTestMust(t, e)
				sy, e := usymbol.NewManager(usymbol.Config{Budget: b})
				astTestMust(t, e)
				m, e := ast.NewASTManager(ast.ASTManagerConfig{Symbols: sy, Numerals: q, Budget: b})
				astTestMust(t, e)
				t.Cleanup(func() {
					astTestMust(t, m.Close())
					astTestMust(t, sy.Close())
					astTestMust(t, q.CloseChecked())
					astTestMust(t, rt.Close())
					if b.Used() != 0 {
						t.Fatal("retained actual graph", b.Used())
					}
				})
				_, e = m.RegisterArithmeticPlugin()
				astTestMust(t, e)
				_, e = m.RegisterBitVectorPlugin()
				astTestMust(t, e)
				s, e := m.BitVectorSort(65)
				astTestMust(t, e)
				x, e := m.MkVar(0, s)
				astTestMust(t, e)
				astTestPin(t, m, x)
				v, e := q.NewValue()
				astTestMust(t, e)
				astTestMust(t, q.SetString(&v, "18446744073709551617"))
				num, e := m.BitVectorNumeral(&v, 65)
				astTestMust(t, e)
				astTestPin(t, m, num)
				astTestMust(t, q.Del(&v))
				zero := bvAddNum(t, m, 0, 65)
				c, g, _ := bvPolynomialParamsFixture(t)
				p, e := NewBVPolynomialRewriter(m, c, g, nil)
				astTestMust(t, e)
				out, e := m.NewRef(x)
				astTestMust(t, e)
				defer func() { astTestMust(t, m.DelRef(&out)) }()
				b.FailAfter(ordinal)
				var st RewriteStatus
				var body error
				switch op {
				case "unary":
					st, body = p.RewriteUnaryMinus(num, out)
				case "symbolic":
					st, body = p.RewriteUnaryMinus(x, out)
				case "sub":
					st, body = p.RewriteSubtraction(ast.SliceArguments([]ast.ASTHandle{x, num, zero}), out)
				}
				b.FailAfter(-1)
				if body != nil {
					failures++
					if !numericHoistRefusal(body) {
						t.Fatal(body)
					}
					cause := p.cause
					if _, e = p.RewriteUnaryMinus(x, out); e != cause {
						t.Fatal("stopped retry", e, cause)
					}
					astTestMust(t, p.Close())
					if p.cause != cause {
						t.Fatal("cleanup changed history")
					}
				} else {
					complete = true
					astTestMust(t, p.Close())
					got, e := m.RefGet(out)
					astTestMust(t, e)
					if op == "unary" {
						if st != RewriteDone {
							t.Fatal(st)
						}
						r, e := q.NewValue()
						astTestMust(t, e)
						var width uint32
						yes, e := m.BitVectorNumeralValue(got, &r, &width)
						astTestMust(t, e)
						text, e := q.String(&r)
						astTestMust(t, e)
						astTestMust(t, q.Del(&r))
						if !yes || width != 65 || text != "18446744073709551615" {
							t.Fatal(yes, width, text)
						}
					} else {
						v, e = q.NewValue()
						astTestMust(t, e)
						astTestMust(t, q.SetString(&v, "36893488147419103231"))
						minus, e := m.BitVectorNumeral(&v, 65)
						astTestMust(t, e)
						astTestPin(t, m, minus)
						astTestMust(t, q.Del(&v))
						var want ast.ASTHandle
						if op == "symbolic" {
							want = bvMergeMul(t, m, minus, x)
							if st != Rewrite1 {
								t.Fatal(st)
							}
						} else {
							want = bvMulHoistSum(t, m, x, bvMergeMul(t, m, minus, num))
							if st != Rewrite2 {
								t.Fatal(st)
							}
						}
						if got != want {
							t.Fatal("incorrect source expression", got, want)
						}
					}
				}
			})
		}
		if !complete || failures == 0 {
			t.Fatal(op, complete, failures)
		}
	}
}
