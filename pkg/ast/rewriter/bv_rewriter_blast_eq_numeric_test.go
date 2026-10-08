// Source: src/ast/rewriter/bv_rewriter.cpp:2279-2304.
package rewriter

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util"
)

func TestBVRewriterBlastEqualityEveryActualNumericalReservation(t *testing.T) {
	complete := false
	failures := 0
	sawNumericOwner := false
	sawLoopPrefix := false
	for ordinal := int64(0); ordinal < 8192 && !complete; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
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
			lhs, e := m.MkBitVectorOr([]ast.ASTHandle{x, y})
			astTestMust(t, e)
			astTestPin(t, m, lhs)
			value, e := q.NewValue()
			astTestMust(t, e)
			astTestMust(t, q.SetString(&value, "18446744073709551621"))
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

			ex, e := NewMkExtractProc(m)
			astTestMust(t, e)
			defer func() { astTestMust(t, ex.Close()) }()
			b.FailAfter(ordinal)
			st, body := p.RewriteBlastEqualityValue(lhs, high, out, ex)
			b.FailAfter(-1)
			if body != nil {
				failures++
				if !numericHoistRefusal(body) {
					t.Fatal(body)
				}
				if p.blastEquality != nil && p.blastEquality.numerals != nil {
					sawNumericOwner = true
				}
				if p.blastEquality != nil && len(p.blastEquality.args.Slice()) > 0 {
					sawLoopPrefix = true
				}
				cause := p.cause
				if _, e = p.RewriteBlastEqualityValue(lhs, high, out, ex); e != cause {
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
				if p.blastEquality != nil {
					t.Fatal("source scope retained")
				}
				astTestMust(t, p.Close())
				got, e := m.RefGet(out)
				astTestMust(t, e)
				terms := make([]ast.ASTHandle, 65)
				for i := uint32(0); i < 65; i++ {
					bit, e := ex.Apply(i, i, lhs)
					astTestMust(t, e)
					value := uint32(0)
					if i == 0 || i == 2 || i == 64 {
						value = 1
					}
					terms[i], e = m.MkEq(bit, bvAddNum(t, m, value, 1))
					astTestMust(t, e)
					astTestPin(t, m, terms[i])
				}
				want, e := m.MkAndArgs(terms)
				astTestMust(t, e)
				astTestPin(t, m, want)
				if got != want {
					t.Fatal("wrong width65 bitwise equality", got, want)
				}
			}
		})
	}
	if !complete || failures == 0 || !sawNumericOwner || !sawLoopPrefix {
		t.Fatal("missing success or captured numeral/remainder/division failure", complete, failures, sawNumericOwner, sawLoopPrefix)
	}
}
