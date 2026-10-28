package rewriter

// Source: src/ast/rewriter/poly_rewriter_def.h:667-681.

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
)

func TestBVPolynomialUnaryMinusNumeralsAndSymbolicProduct(t *testing.T) {
	c, g, _ := bvPolynomialParamsFixture(t)
	m, _ := bvAddFixture(t)
	p, e := NewBVPolynomialRewriter(m, c, g, nil)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	xs := bvMergeVars(t, m, 1)
	out, e := m.NewRef(ast.ASTHandle{})
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	for _, v := range []uint32{0, 1, 255, 128} {
		arg := bvAddNum(t, m, v, 8)
		st, e := p.RewriteUnaryMinus(arg, out)
		astTestMust(t, e)
		got, e := m.RefGet(out)
		astTestMust(t, e)
		if st != RewriteDone || got != bvAddNum(t, m, (256-v)%256, 8) {
			t.Fatal(st, got, v)
		}
	}
	st, e := p.RewriteUnaryMinus(xs[0], out)
	astTestMust(t, e)
	got, e := m.RefGet(out)
	astTestMust(t, e)
	if st != Rewrite1 || got != bvMergeMul(t, m, bvAddNum(t, m, 255, 8), xs[0]) {
		t.Fatal(st, got)
	}
}

func TestBVPolynomialUnaryMinusEveryColdReservationAndPlacedBeforeSort(t *testing.T) {
	for _, symbolic := range []bool{false, true} {
		complete := false
		for ordinal := int64(0); ordinal < 128 && !complete; ordinal++ {
			t.Run(fmt.Sprintf("%t/%d", symbolic, ordinal), func(t *testing.T) {
				c, g, _ := bvPolynomialParamsFixture(t)
				m, _ := bvAddFixture(t)
				p, e := NewBVPolynomialRewriter(m, c, g, nil)
				astTestMust(t, e)
				x := bvMergeVars(t, m, 1)[0]
				arg := bvAddNum(t, m, 128, 8)
				if symbolic {
					arg = x
				}
				out, e := m.NewRef(x)
				astTestMust(t, e)
				defer func() { astTestMust(t, m.DelRef(&out)) }()
				m.Budget().FailAfter(ordinal)
				st, body := p.RewriteUnaryMinus(arg, out)
				m.Budget().FailAfter(-1)
				if body == nil {
					complete = true
					if (!symbolic && st != RewriteDone) || (symbolic && st != Rewrite1) || p.negation != nil {
						t.Fatal(st)
					}
				} else {
					if !errors.Is(body, arena.ErrInjectedFailure) {
						t.Fatal(body)
					}
					if _, e = p.RewriteUnaryMinus(arg, out); e != body {
						t.Fatal("stopped retry", e)
					}
				}
				cause := p.cause
				astTestMust(t, p.Close())
				if p.cause != cause {
					t.Fatal("history changed")
				}
			})
		}
		if !complete {
			t.Fatal("never completed", symbolic)
		}
	}
	c, g, _ := bvPolynomialParamsFixture(t)
	m, _ := bvAddFixture(t)
	p, e := NewBVPolynomialRewriter(m, c, g, nil)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	out, e := m.NewRef(ast.ASTHandle{})
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	_, e = p.RewriteUnaryMinus(ast.ASTHandle{}, out)
	if e == nil || p.negation == nil || !p.currentSort.IsNull() {
		t.Fatal("missing pre-sort source prefix", e)
	}
}
