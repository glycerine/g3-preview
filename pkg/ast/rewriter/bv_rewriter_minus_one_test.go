// Source: src/ast/rewriter/bv_rewriter.cpp:1146-1153,1172-1184.
package rewriter

import (
	"fmt"
	"github.com/glycerine/g3/pkg/ast"
	"testing"
)

func TestBVRewriterOriginalMinusOneAndXMinusOne(t *testing.T) {
	c, g, _ := bvPolynomialParamsFixture(t)
	m, _ := bvAddFixture(t)
	p, e := NewBVRewriter(m, c, g, nil)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	xs := bvMergeVars(t, m, 2)
	minus := bvAddNum(t, m, 255, 8)
	for _, tc := range []struct {
		arg  ast.ASTHandle
		want bool
	}{{minus, true}, {bvAddNum(t, m, 254, 8), false}, {xs[0], false}} {
		got, e := p.IsMinusOne(tc.arg)
		astTestMust(t, e)
		if got != tc.want {
			t.Fatal(got, tc.want)
		}
	}
	for _, tc := range []struct {
		args []ast.ASTHandle
		want bool
		x    ast.ASTHandle
	}{
		{[]ast.ASTHandle{minus, xs[0]}, true, xs[0]},
		{[]ast.ASTHandle{xs[0], minus}, true, xs[0]},
		{[]ast.ASTHandle{minus, minus}, true, minus},
		{[]ast.ASTHandle{xs[0], xs[1]}, false, ast.ASTHandle{}},
		{[]ast.ASTHandle{minus, xs[0], xs[1]}, false, ast.ASTHandle{}},
	} {
		h, e := m.MkFamilyApp(p.bvFamily, 4, nil, tc.args)
		astTestMust(t, e)
		astTestPin(t, m, h)
		x := xs[1]
		got, e := p.IsXMinusOne(h, &x)
		astTestMust(t, e)
		if got != tc.want || got && x != tc.x || !got && x != xs[1] {
			t.Fatal("source order/output", got, x, tc)
		}
	}
	x := xs[1]
	got, e := p.IsXMinusOne(minus, &x)
	astTestMust(t, e)
	if got || x != xs[1] {
		t.Fatal("nonadd output", got, x)
	}
}

func TestBVRewriterMinusOneEveryColdReservation(t *testing.T) {
	for _, mode := range []string{"first", "second", "none", "nonadd", "arity"} {
		complete := false
		for ordinal := int64(0); ordinal < 512 && !complete; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", mode, ordinal), func(t *testing.T) {
				c, g, _ := bvPolynomialParamsFixture(t)
				m, _ := bvAddFixture(t)
				p, e := NewBVRewriter(m, c, g, nil)
				astTestMust(t, e)
				xs := bvMergeVars(t, m, 2)
				minus := bvAddNum(t, m, 255, 8)
				args := []ast.ASTHandle{minus, xs[0]}
				switch mode {
				case "second":
					args = []ast.ASTHandle{xs[0], minus}
				case "none":
					args = xs
				case "arity":
					args = []ast.ASTHandle{minus, xs[0], xs[1]}
				}
				h, e := m.MkFamilyApp(p.bvFamily, 4, nil, args)
				astTestMust(t, e)
				astTestPin(t, m, h)
				if mode == "nonadd" {
					h = minus
				}
				x := xs[1]
				m.Budget().FailAfter(ordinal)
				got, body := p.IsXMinusOne(h, &x)
				m.Budget().FailAfter(-1)
				if body == nil {
					complete = true
					want := mode == "first" || mode == "second"
					if got != want || got && x != xs[0] || !got && x != xs[1] {
						t.Fatal("result", got, x)
					}
				} else {
					if !numericHoistRefusal(body) {
						t.Fatal(body)
					}
					if got || x != xs[1] {
						t.Fatal("failed helper assigned output", got, x)
					}
					if _, e = p.IsXMinusOne(h, &x); e != body {
						t.Fatal("stopped replay", e, body)
					}
				}
				cause := p.cause
				astTestMust(t, p.Close())
				astTestMust(t, p.Close())
				if p.cause != cause {
					t.Fatal("cleanup changed history")
				}
			})
		}
		if !complete {
			t.Fatal("cold campaign incomplete", mode)
		}
	}
}
