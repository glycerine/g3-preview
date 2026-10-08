// Source: src/ast/rewriter/poly_rewriter.h:149-156.
package rewriter

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/pkg/ast"
	"testing"
)

func TestBVPolynomialMakeAdditionOriginalFallbackAndPair(t *testing.T) {
	for _, flat := range []bool{false, true} {
		m, _ := bvAddFixture(t)
		xs := bvMergeVars(t, m, 3)
		out, e := m.NewRef(xs[2])
		astTestMust(t, e)
		defer func() { astTestMust(t, m.DelRef(&out)) }()
		p, e := NewBVPolynomialAdditionCore(m)
		astTestMust(t, e)
		defer func() { astTestMust(t, p.Close()) }()
		astTestMust(t, p.MakeAdditionPair(xs[0], xs[1], out, flat, BVPolynomialAdditionOptions{}))
		got, e := m.RefGet(out)
		astTestMust(t, e)
		want := bvMulHoistSum(t, m, xs[0], xs[1])
		if got != want {
			t.Fatal("FAILED fallback did not assign original sum")
		}
		astTestMust(t, p.SourceReady())
	}
}

func TestBVPolynomialMakeAdditionRetainsCoreResultsAndSingleton(t *testing.T) {
	m, _ := bvAddFixture(t)
	xs := bvMergeVars(t, m, 3)
	for _, row := range []struct {
		args []ast.ASTHandle
		want ast.ASTHandle
	}{
		{[]ast.ASTHandle{xs[0]}, xs[0]},
		{[]ast.ASTHandle{xs[0], xs[0]}, bvMergeMul(t, m, bvAddNum(t, m, 2, 8), xs[0])},
		{[]ast.ASTHandle{bvAddNum(t, m, 0, 8), xs[0]}, xs[0]},
		{[]ast.ASTHandle{bvAddNum(t, m, 5, 8), bvAddNum(t, m, 7, 8)}, bvAddNum(t, m, 12, 8)},
	} {
		astTestPin(t, m, row.want)
		out, e := m.NewRef(xs[2])
		astTestMust(t, e)
		p, e := NewBVPolynomialAdditionCore(m)
		astTestMust(t, e)
		astTestMust(t, p.MakeAddition(ast.SliceArguments(row.args), out, false, BVPolynomialAdditionOptions{}))
		got, e := m.RefGet(out)
		astTestMust(t, e)
		if got != row.want {
			t.Fatal("successful core result replaced", got, row.want)
		}
		astTestMust(t, p.SourceReady())
		astTestMust(t, p.Close())
		astTestMust(t, m.DelRef(&out))
	}
}

func TestBVPolynomialMakeAdditionEveryFallbackReservationAndStoppedPair(t *testing.T) {
	complete, failures, fallbackFailures := false, 0, 0
	for ordinal := int64(0); ordinal < 200 && !complete; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m, _ := bvAddFixture(t)
			xs := bvMergeVars(t, m, 24)
			out, e := m.NewRef(ast.ASTHandle{})
			astTestMust(t, e)
			defer func() { astTestMust(t, m.DelRef(&out)) }()
			p, e := NewBVPolynomialAdditionCore(m)
			astTestMust(t, e)
			defer func() { astTestMust(t, p.Close()) }()
			m.Budget().FailAfter(ordinal)
			body := p.MakeAddition(ast.SliceArguments(xs), out, true, BVPolynomialAdditionOptions{})
			m.Budget().FailAfter(-1)
			if body != nil {
				failures++
				if !numericHoistRefusal(body) {
					t.Fatal("unexpected refusal", ordinal, body)
				}
				if p.localsEnded {
					fallbackFailures++
					if p.sourceDone || p.visited != nil || p.multiple != nil {
						t.Fatal("fallback preceded core exit")
					}
				}
			} else {
				complete = true
				got, e := m.RefGet(out)
				astTestMust(t, e)
				n, e := m.GetNumArgs(got)
				astTestMust(t, e)
				if n != 24 {
					t.Fatal("fallback arity", n)
				}
				for i, x := range xs {
					arg, e := m.GetArg(got, uint32(i))
					astTestMust(t, e)
					if arg != x {
						t.Fatal("original argument order", i)
					}
				}
				astTestMust(t, p.SourceReady())
			}
			pair := p.pair
			if e = p.MakeAdditionPair(xs[1], xs[2], out, true, BVPolynomialAdditionOptions{}); !errors.Is(e, ast.ErrASTPending) || p.pair != pair {
				t.Fatal("stopped invocation initializer changed", e)
			}
			history := p.SourceReady()
			astTestMust(t, p.Close())
			if p.SourceReady() != history {
				t.Fatal("owning cleanup changed history")
			}
		})
	}
	if !complete || failures == 0 || fallbackFailures == 0 {
		t.Fatal("fallback horizon", complete, failures, fallbackFailures)
	}
}
