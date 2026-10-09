// Source: src/ast/rewriter/bv_rewriter.cpp:2380-2414.
package rewriter

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestBVRewriterIsolateTermEveryActualAllOneReservation(t *testing.T) {
	complete := false
	failures := 0
	sawAllOne := false
	for ordinal := int64(0); ordinal < 2048 && !complete; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			b := arena.NewBudget(1 << 27)
			rt, e := util.NewRuntime(util.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f util.Failure) error { return f }})
			astTestMust(t, e)
			q, e := util.NewRationalService(util.RationalServiceConfig{Integers: util.IntegerConfig{Memory: rt, Profile: util.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
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
			first := bvMergeMul(t, m, high, x)
			lhs, e := m.MkBitVectorPolynomialAdd(ast.ASTHandle{}, ast.SliceArguments([]ast.ASTHandle{first, y}))
			astTestMust(t, e)
			astTestPin(t, m, lhs)
			constant := bvAddNum(t, m, 9, 65)
			c, g, _ := bvPolynomialParamsFixture(t)
			p, e := NewBVPolynomialRewriter(m, c, g, nil)
			astTestMust(t, e)
			out, e := m.NewRef(ast.ASTHandle{})
			astTestMust(t, e)
			defer func() { astTestMust(t, m.DelRef(&out)) }()

			b.FailAfter(ordinal)
			yes, body := p.IsolateBitVectorTerm(constant, lhs, out)
			b.FailAfter(-1)
			if body != nil {
				failures++
				if !numericHoistRefusal(body) {
					t.Fatal(body)
				}
				if p.isolation != nil && p.isolation.allOne != (ast.ASTBitVectorNumeralInvocation{}) {
					sawAllOne = true
				}
				cause := p.cause
				if _, e = p.IsolateBitVectorTerm(constant, lhs, out); e != cause {
					t.Fatal("stopped retry", e, cause)
				}
				astTestMust(t, p.Close())
				if p.cause != cause {
					t.Fatal("cleanup changed history")
				}
			} else {
				complete = true
				if !yes {
					t.Fatal("unmatched")
				}
				if p.isolation != nil {
					t.Fatal("source scope retained")
				}
				astTestMust(t, p.Close())
				got, e := m.RefGet(out)
				astTestMust(t, e)
				sub, e := m.MkBitVectorSub(constant, first)
				astTestMust(t, e)
				want, e := m.MkEq(y, sub)
				astTestMust(t, e)
				astTestPin(t, m, want)
				if got != want {
					t.Fatal("wrong width65 isolation", got, want)
				}
			}
		})
	}
	if !complete || failures == 0 || !sawAllOne {
		t.Fatal("missing success or captured all-one failure", complete, failures, sawAllOne)
	}
}
