// Source: src/ast/rewriter/poly_rewriter_def.h:708-883.
package rewriter

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestBVPolynomialCancellationEveryActualNumericReservation(t *testing.T) {
	complete := false
	failures := 0
	for ordinal := int64(0); ordinal < 256 && !complete; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			b := arena.NewBudget(1 << 27)
			rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f umemory.Failure) error { return f }})
			astTestMust(t, e)
			q, e := util.NewRationalService(util.RationalServiceConfig{Integers: util.IntegerConfig{Memory: rt, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
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
			astTestMust(t, q.SetString(&value, "18446744073709551617"))
			num, e := m.BitVectorNumeral(&value, 65)
			astTestMust(t, e)
			astTestPin(t, m, num)
			astTestMust(t, q.Del(&value))
			lhs := bvMulHoistSum(t, m, bvMergeMul(t, m, num, x), num)
			astTestPin(t, m, lhs)
			rhs := bvMulHoistSum(t, m, x, bvAddNum(t, m, 1, 65), y)
			astTestPin(t, m, rhs)
			c, g, _ := bvPolynomialParamsFixture(t)
			p, e := NewBVPolynomialRewriter(m, c, g, nil)
			astTestMust(t, e)
			lo, e := m.NewRef(ast.ASTHandle{})
			astTestMust(t, e)
			defer func() { astTestMust(t, m.DelRef(&lo)) }()
			ro, e := m.NewRef(ast.ASTHandle{})
			astTestMust(t, e)
			defer func() { astTestMust(t, m.DelRef(&ro)) }()
			b.FailAfter(ordinal)
			st, body := p.CancelMonomials(lhs, rhs, true, lo, ro)
			b.FailAfter(-1)
			if body != nil {
				failures++
				if !numericHoistRefusal(body) {
					t.Fatal(body)
				}
				cause := p.cause
				if _, e = p.CancelMonomials(lhs, rhs, false, lo, ro); e != cause {
					t.Fatal("stopped retry", e, cause)
				}
				if n := p.cancellation; n != nil {
					bodyHistory, sourceHistory := n.bodyCause, n.sourceCause
					astTestMust(t, p.Close())
					if n.bodyCause != bodyHistory || n.sourceCause != sourceHistory {
						t.Fatal("cleanup changed child history")
					}
				} else {
					astTestMust(t, p.Close())
				}
				if p.cause != cause {
					t.Fatal("cleanup changed parent history")
				}
			} else {
				complete = true
				if st != RewriteDone {
					t.Fatal(st)
				}
				astTestMust(t, p.Close())
				value, e = q.NewValue()
				astTestMust(t, e)
				astTestMust(t, q.SetString(&value, "18446744073709551616"))
				big, e := m.BitVectorNumeral(&value, 65)
				astTestMust(t, e)
				astTestPin(t, m, big)
				astTestMust(t, q.SetString(&value, "36893488147419103231"))
				minus, e := m.BitVectorNumeral(&value, 65)
				astTestMust(t, e)
				astTestPin(t, m, minus)
				astTestMust(t, q.Del(&value))
				want := bvMulHoistSum(t, m, bvMergeMul(t, m, big, x), bvMergeMul(t, m, minus, y))
				l, e := m.RefGet(lo)
				astTestMust(t, e)
				r, e := m.RefGet(ro)
				astTestMust(t, e)
				if l != want || r != big {
					t.Fatal("wrong large cancellation", l, r, want, big)
				}
			}
		})
	}
	if !complete || failures == 0 {
		t.Fatal(complete, failures)
	}
}
