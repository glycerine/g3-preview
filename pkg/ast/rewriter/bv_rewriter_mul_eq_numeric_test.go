// Source: src/ast/rewriter/bv_rewriter.cpp:2448-2508.
package rewriter

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util"
)

func TestBVRewriterMultiplicationEqualityEveryActualNumericReservation(t *testing.T) {
	for _, branch := range []string{"constant", "coefficient", "unit", "sum", "search"} {
		complete := false
		failures := 0
		sawNumericOwner := false
		for ordinal := int64(0); ordinal < 8192 && !complete; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", branch, ordinal), func(t *testing.T) {
				b := arena.NewBudget(1 << 27)
				rt, e := util.NewRuntime(util.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f util.Failure) error { return f }})
				astTestMust(t, e)
				q, e := util.NewRationalService(util.RationalServiceConfig{Integers: util.IntegerConfig{Memory: rt, Profile: util.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
				astTestMust(t, e)
				sy, e := util.NewSymbolManager(util.SymbolConfig{Budget: b})
				astTestMust(t, e)
				m, e := ast.NewASTManager(ast.ASTManagerConfig{Symbols: sy, Numerals: q, Budget: b})
				astTestMust(t, e)
				t.Cleanup(func() {
					astTestMust(t, m.Close())
					astTestMust(t, sy.Close())
					astTestMust(t, q.CloseChecked())
					astTestMust(t, rt.Close())
					if b.Used() != 0 {
						t.Fatal("actual graph retained", b.Used())
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
				y, e := m.MkVar(1, s)
				astTestMust(t, e)
				astTestPin(t, m, y)
				value, e := q.NewValue()
				astTestMust(t, e)
				astTestMust(t, q.SetString(&value, "36893488147419103231"))
				high, e := m.BitVectorNumeral(&value, 65)
				astTestMust(t, e)
				astTestPin(t, m, high)
				astTestMust(t, q.Del(&value))
				lhs := bvMergeMul(t, m, high, x)
				rhs := bvAddNum(t, m, 9, 65)
				status := Rewrite1
				if branch == "unit" {
					rhs = bvMergeMul(t, m, high, y)
				}
				if branch == "coefficient" {
					rhs = bvMergeMul(t, m, bvAddNum(t, m, 5, 65), y)
				}
				if branch == "sum" || branch == "search" {
					rhs, e = m.MkBitVectorPolynomialAdd(ast.ASTHandle{}, ast.SliceArguments([]ast.ASTHandle{bvMergeMul(t, m, bvAddNum(t, m, 2, 65), x), bvMergeMul(t, m, high, y)}))
					astTestMust(t, e)
					astTestPin(t, m, rhs)
					status = Rewrite2
					if branch == "search" {
						lhs = bvAddNum(t, m, 9, 65)
						status = Rewrite3
					}
				}
				c, g, _ := bvPolynomialParamsFixture(t)
				p, e := NewBVPolynomialRewriter(m, c, g, nil)
				astTestMust(t, e)
				out, e := m.NewRef(ast.ASTHandle{})
				astTestMust(t, e)
				defer func() { astTestMust(t, m.DelRef(&out)) }()

				b.FailAfter(ordinal)
				st, body := p.RewriteMultiplicationEquality(lhs, rhs, out)
				b.FailAfter(-1)
				if body != nil {
					failures++
					if !numericHoistRefusal(body) {
						t.Fatal(body)
					}
					if p.mulEquality != nil {
						sawNumericOwner = true
					}
					cause := p.cause
					if _, e = p.RewriteMultiplicationEquality(lhs, rhs, out); e != cause {
						t.Fatal("stopped retry", e, cause)
					}
					astTestMust(t, p.Close())
					if p.cause != cause {
						t.Fatal("cleanup changed history")
					}
				} else {
					complete = true
					if st != status {
						t.Fatal(st, status)
					}
					if p.mulEquality != nil {
						t.Fatal("source scope retained")
					}
					astTestMust(t, p.Close())
					got, e := m.RefGet(out)
					astTestMust(t, e)
					// Reuse the actual all-one input for its modular inverse.
					inv := high
					var want ast.ASTHandle
					if branch == "constant" || branch == "coefficient" {
						expected := "36893488147419103223"
						if branch == "coefficient" {
							expected = "36893488147419103227"
						}
						v, e := q.NewValue()
						astTestMust(t, e)
						astTestMust(t, q.SetString(&v, expected))
						num, e := m.BitVectorNumeral(&v, 65)
						astTestMust(t, e)
						astTestPin(t, m, num)
						astTestMust(t, q.Del(&v))
						target := num
						if branch == "coefficient" {
							target = bvMergeMul(t, m, num, y)
						}
						want, e = m.MkEq(x, target)
						astTestMust(t, e)
					} else if branch == "unit" {
						want, e = m.MkEq(x, y)
					} else {
						right := bvMergeMul(t, m, inv, rhs)
						left := x
						if branch == "search" {
							v, e := q.NewValue()
							astTestMust(t, e)
							astTestMust(t, q.SetString(&v, "36893488147419103223"))
							left, e = m.BitVectorNumeral(&v, 65)
							astTestMust(t, e)
							astTestPin(t, m, left)
							astTestMust(t, q.Del(&v))
						}
						want, e = m.MkEq(left, right)
					}
					astTestMust(t, e)
					astTestPin(t, m, want)
					if got != want {
						t.Fatal("wrong width65 multiplication equality", got, want)
					}
				}
			})
		}
		if !complete || failures == 0 || !sawNumericOwner {
			t.Fatal("missing success or captured inverse/product/norm failure", complete, failures, sawNumericOwner)
		}
	}
}
