// Source: src/ast/rewriter/poly_rewriter_def.h:683-701.
package rewriter

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
)

func TestBVPolynomialSubtractionRetainsSourceProductsAndSkipsZero(t *testing.T) {
	c, g, _ := bvPolynomialParamsFixture(t)
	m, _ := bvAddFixture(t)
	p, e := NewBVPolynomialRewriter(m, c, g, nil)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	xs := bvMergeVars(t, m, 3)
	zero := bvAddNum(t, m, 0, 8)
	two := bvAddNum(t, m, 2, 8)
	out, e := m.NewRef(ast.ASTHandle{})
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	args := []ast.ASTHandle{xs[0], zero, xs[1], two, xs[2]}
	st, e := p.RewriteSubtraction(ast.SliceArguments(args), out)
	astTestMust(t, e)
	got, e := m.RefGet(out)
	astTestMust(t, e)
	minus := bvAddNum(t, m, 255, 8)
	want := bvMulHoistSum(t, m, xs[0], bvMergeMul(t, m, minus, xs[1]), bvMergeMul(t, m, minus, two), bvMergeMul(t, m, minus, xs[2]))
	if st != Rewrite2 || got != want {
		t.Fatal("subtraction folded or reordered source products", st, got, want)
	}
	st, e = p.RewriteSubtraction(ast.SliceArguments([]ast.ASTHandle{xs[0], zero, zero}), out)
	astTestMust(t, e)
	got, e = m.RefGet(out)
	astTestMust(t, e)
	if st != Rewrite2 || got != xs[0] {
		t.Fatal("all zero status", st, got)
	}
	astTestMust(t, p.SetFlat(false))
	astTestMust(t, p.SetSortSums(true))
	astTestMust(t, p.MakeSubtractionPair(xs[0], xs[1], out))
	got, e = m.RefGet(out)
	astTestMust(t, e)
	if got != bvMulHoistSum(t, m, xs[0], bvMergeMul(t, m, minus, xs[1])) {
		t.Fatal("pair overload used addition core or changed order")
	}

}

func TestBVPolynomialSubtractionEveryVectorAndFactoryReservation(t *testing.T) {
	complete := false
	sawUnpublished := false
	for ordinal := int64(0); ordinal < 512 && !complete; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			c, g, _ := bvPolynomialParamsFixture(t)
			m, _ := bvAddFixture(t)
			p, e := NewBVPolynomialRewriter(m, c, g, nil)
			astTestMust(t, e)
			xs := bvMergeVars(t, m, 40)
			zero := bvAddNum(t, m, 0, 8)
			args := append(xs[:1:1], zero)
			args = append(args, xs[1:]...)
			out, e := m.NewRef(xs[0])
			astTestMust(t, e)
			defer func() { astTestMust(t, m.DelRef(&out)) }()
			m.Budget().FailAfter(ordinal)
			st, body := p.RewriteSubtraction(ast.SliceArguments(args), out)
			m.Budget().FailAfter(-1)
			if body == nil {
				complete = true
				if st != Rewrite2 || p.subtraction != nil {
					t.Fatal(st)
				}
			} else {
				if !errors.Is(body, arena.ErrInjectedFailure) {
					t.Fatal(body)
				}
				if n := p.subtraction; n != nil {
					pending, e := n.push.Unpublished()
					astTestMust(t, e)
					sawUnpublished = sawUnpublished || pending
					if n.minusPlaced {
						minus, e := m.RefGet(n.minusOne)
						astTestMust(t, e)
						width, e := m.BitVectorSize(minus)
						astTestMust(t, e)
						if width != 8 {
							t.Fatal("wrong retained minus-one reference")
						}
					}
				}
				old := p.currentSort
				oldPair := p.pair
				if e = p.MakeSubtractionPair(xs[1], xs[2], out); e != body {
					t.Fatal("stopped pair retry", e)
				}
				if p.pair != oldPair {
					t.Fatal("stopped pair changed initializer")
				}
				if _, e = p.RewriteSubtraction(ast.SliceArguments(xs[:1]), out); e != body {
					t.Fatal("stopped retry", e)
				}
				if old != p.currentSort {
					t.Fatal("retry changed current sort")
				}
			}
			cause := p.cause
			astTestMust(t, p.Close())
			if p.cause != cause {
				t.Fatal("history changed")
			}
		})
	}
	if !complete || !sawUnpublished {
		t.Fatal("missing success or retained INC", complete, sawUnpublished)
	}
}

func TestBVPolynomialSubtractionSingletonBeforeSortAndAllocation(t *testing.T) {
	c, g, _ := bvPolynomialParamsFixture(t)
	m, _ := bvAddFixture(t)
	p, e := NewBVPolynomialRewriter(m, c, g, nil)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	x := bvMergeVars(t, m, 1)[0]
	out, e := m.NewRef(x)
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	m.Budget().FailAfter(0)
	st, e := p.RewriteSubtraction(ast.SliceArguments([]ast.ASTHandle{x}), out)
	m.Budget().FailAfter(-1)
	astTestMust(t, e)
	if st != RewriteDone || !p.currentSort.IsNull() || p.subtraction != nil {
		t.Fatal(st)
	}
}
