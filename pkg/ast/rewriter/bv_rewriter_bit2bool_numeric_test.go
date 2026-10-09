// Source: src/ast/rewriter/bv_rewriter.cpp:2205-2277.
package rewriter

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestBVRewriterBit2BoolEveryActualNumeralCopyReservation(t *testing.T) {
	complete := false
	failures := 0
	sawAfterCore := false
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
			s, e := m.BitVectorSort(1)
			astTestMust(t, e)
			x, e := m.MkVar(0, s)
			astTestMust(t, e)
			astTestPin(t, m, x)
			y, e := m.MkVar(1, s)
			astTestMust(t, e)
			astTestPin(t, m, y)
			lhs, e := m.MkBitVectorOr([]ast.ASTHandle{x, y})
			astTestMust(t, e)
			astTestPin(t, m, lhs)
			value, e := q.NewValue()
			astTestMust(t, e)
			astTestMust(t, q.SetString(&value, "18446744073709551616"))
			high, e := m.BitVectorNumeral(&value, 65)
			astTestMust(t, e)
			astTestPin(t, m, high)
			astTestMust(t, q.Del(&value))
			c, g, _ := bvPolynomialParamsFixture(t)
			p, e := NewBVPolynomialRewriter(m, c, g, nil)
			astTestMust(t, e)
			out, e := m.NewRef(ast.ASTHandle{})
			astTestMust(t, e)
			defer func() { astTestMust(t, m.DelRef(&out)) }()
			// Source checks only lhs width; rhs copied width does not add a guard.
			b.FailAfter(ordinal)
			st, body := p.RewriteBit2Bool(lhs, high, out)
			b.FailAfter(-1)
			if body != nil {
				failures++
				if !numericHoistRefusal(body) {
					t.Fatal(body)
				}
				if p.bitBool != nil && p.bitBool.numerals != nil {
					sawAfterCore = true
				}
				cause := p.cause
				if _, e = p.RewriteBit2Bool(lhs, high, out); e != cause {
					t.Fatal("stopped retry", e, cause)
				}
				astTestMust(t, p.Close())
				if p.cause != cause {
					t.Fatal("cleanup changed history")
				}
			} else {
				complete = true
				if st != Rewrite3 {
					t.Fatal(st)
				}
				if p.bitBool != nil {
					t.Fatal("source scope retained")
				}
				astTestMust(t, p.Close())
				got, e := m.RefGet(out)
				astTestMust(t, e)
				one := bvAddNum(t, m, 1, 1)
				a, e := m.MkEq(x, one)
				astTestMust(t, e)
				b, e := m.MkEq(y, one)
				astTestMust(t, e)
				base, e := m.MkOrArgs([]ast.ASTHandle{a, b})
				astTestMust(t, e)
				want, e := m.MkNot(base)
				astTestMust(t, e)
				astTestPin(t, m, want)
				if got != want {
					t.Fatal("wrong original mixed-width bit2bool result", got, want)
				}
			}
		})
	}
	if !complete || failures == 0 || !sawAfterCore {
		t.Fatal("missing success or captured numeral-copy failure", complete, failures, sawAfterCore)
	}
}
