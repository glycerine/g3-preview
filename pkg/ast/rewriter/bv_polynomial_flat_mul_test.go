// Source: src/ast/rewriter/poly_rewriter_def.h:140-194;
// src/ast/rewriter/poly_rewriter.h:125-136,157-164.
package rewriter

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestBVPolynomialFlatMultiplicationRecursiveFactorsAndCoefficientGuard(t *testing.T) {
	m, _ := bvAddFixture(t)
	xs := bvMergeVars(t, m, 5)
	s, e := m.GetSort(xs[0])
	astTestMust(t, e)
	base := bvMergeMul(t, m, xs[0], xs[1])
	astTestPin(t, m, base)
	two := bvAddNum(t, m, 2, 8)
	three := bvAddNum(t, m, 3, 8)
	coeffProduct := bvMergeMul(t, m, three, base)
	astTestPin(t, m, coeffProduct)
	nested := bvMergeMul(t, m, xs[1], bvMergeMul(t, m, xs[2], xs[3]))
	astTestPin(t, m, nested)
	for _, row := range []struct {
		args []ast.ASTHandle
		st   RewriteStatus
		want ast.ASTHandle
	}{
		{[]ast.ASTHandle{two, base}, RewriteFailed, xs[4]},
		{[]ast.ASTHandle{two, coeffProduct}, RewriteDone, bvMergeMul(t, m, bvAddNum(t, m, 6, 8), base)},
		{[]ast.ASTHandle{xs[0], nested, xs[4]}, RewriteDone, bvMergeMul(t, m, xs...)},
	} {
		astTestPin(t, m, row.want)
		out, e := m.NewRef(xs[4])
		astTestMust(t, e)
		p, e := NewBVPolynomialMultiplicationCore(m)
		astTestMust(t, e)
		st, e := p.RewriteFlat(s, ast.SliceArguments(row.args), out, BVPolynomialMultiplicationOptions{})
		astTestMust(t, e)
		got, e := m.RefGet(out)
		astTestMust(t, e)
		if st != row.st || got != row.want {
			t.Fatal("flatten/guard", st, got, row.st, row.want)
		}
		astTestMust(t, p.SourceReady())
		astTestMust(t, p.Close())
		astTestMust(t, m.DelRef(&out))
	}
}

func TestBVPolynomialMultiplicationCallerSingletonAndOriginalFallback(t *testing.T) {
	m, _ := bvAddFixture(t)
	xs := bvMergeVars(t, m, 3)
	for _, pair := range []bool{false, true} {
		out, e := m.NewRef(xs[2])
		astTestMust(t, e)
		p, e := NewBVPolynomialMultiplicationCore(m)
		astTestMust(t, e)
		if pair {
			astTestMust(t, p.MakeMultiplicationPair(xs[0], xs[1], out, false, BVPolynomialMultiplicationOptions{}))
		} else {
			m.Budget().FailAfter(0)
			st, e := p.RewriteApplication(ast.SliceArguments(xs[:1]), out, true, BVPolynomialMultiplicationOptions{SumOfMonomials: true})
			m.Budget().FailAfter(-1)
			astTestMust(t, e)
			if st != RewriteDone || p.numeric != nil || !p.currentSort.IsNull() {
				t.Fatal("singleton consulted core")
			}
		}
		got, e := m.RefGet(out)
		astTestMust(t, e)
		want := xs[0]
		if pair {
			want = bvMergeMul(t, m, xs[0], xs[1])
		}
		if got != want {
			t.Fatal("caller result")
		}
		astTestMust(t, p.SourceReady())
		astTestMust(t, p.Close())
		astTestMust(t, m.DelRef(&out))
	}
}

func TestBVPolynomialFlatMultiplicationEveryStackSpillAndFallbackReservation(t *testing.T) {
	complete, failures, completeFlattenFailures, fallbackFailures := false, 0, 0, 0
	for ordinal := int64(0); ordinal < 200 && !complete; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m, _ := bvAddFixture(t)
			xs := bvMergeVars(t, m, 48)
			inner := bvMergeMul(t, m, xs[2], xs[3])
			nestedArgs := []ast.ASTHandle{xs[1], inner}
			nestedArgs = append(nestedArgs, xs[4:]...)
			nested := bvMergeMul(t, m, nestedArgs...)
			astTestPin(t, m, nested)
			args := ast.SliceArguments([]ast.ASTHandle{xs[0], nested})
			out, e := m.NewRef(ast.ASTHandle{})
			astTestMust(t, e)
			defer func() { astTestMust(t, m.DelRef(&out)) }()
			p, e := NewBVPolynomialMultiplicationCore(m)
			astTestMust(t, e)
			defer func() { astTestMust(t, p.Close()) }()
			m.Budget().FailAfter(ordinal)
			st, body := p.RewriteApplication(args, out, true, BVPolynomialMultiplicationOptions{})
			m.Budget().FailAfter(-1)
			if body != nil {
				failures++
				if !numericHoistRefusal(body) {
					t.Fatal("unexpected refusal", ordinal, body)
				}
				if p.numeric == nil && len(p.flatArgs.Slice()) == 48 {
					completeFlattenFailures++
					for i, h := range p.flatArgs.Slice() {
						if h != xs[i] {
							t.Fatal("recursive stack order before core", i)
						}
					}
				}
				if p.localsEnded {
					fallbackFailures++
					if p.sourceDone {
						t.Fatal("fallback claimed source completion")
					}
				}
			} else {
				complete = true
				if st != RewriteDone {
					t.Fatal(st)
				}
				got, e := m.RefGet(out)
				astTestMust(t, e)
				n, e := m.GetNumArgs(got)
				astTestMust(t, e)
				if n != 48 {
					t.Fatal("flatten arity", n)
				}
				for i, x := range xs {
					h, e := m.GetArg(got, uint32(i))
					astTestMust(t, e)
					if h != x {
						t.Fatal("flatten order", i)
					}
				}
				astTestMust(t, p.SourceReady())
			}
			saved := p.pair
			if e = p.MakeMultiplicationPair(xs[2], xs[3], out, true, BVPolynomialMultiplicationOptions{}); !errors.Is(e, ast.ErrASTPending) || p.pair != saved {
				t.Fatal("stopped initializer overwritten", e)
			}
			ready := p.SourceReady()
			astTestMust(t, p.Close())
			if p.SourceReady() != ready {
				t.Fatal("owning cleanup changed source history")
			}
		})
	}
	if !complete || failures == 0 || completeFlattenFailures == 0 || fallbackFailures == 0 {
		t.Fatal("stack/fallback horizon", complete, failures, completeFlattenFailures, fallbackFailures)
	}
}

func TestBVPolynomialMultiplicationCallerSelectionAndConvenience(t *testing.T) {
	for _, flat := range []bool{false, true} {
		m, _ := bvAddFixture(t)
		xs := bvMergeVars(t, m, 4)
		s, e := m.GetSort(xs[0])
		astTestMust(t, e)
		nested := bvMergeMul(t, m, xs[1], xs[2])
		astTestPin(t, m, nested)
		args := ast.SliceArguments([]ast.ASTHandle{xs[0], nested})
		out, e := m.NewRef(xs[3])
		astTestMust(t, e)
		p, e := NewBVPolynomialMultiplicationCore(m)
		astTestMust(t, e)
		st, e := p.RewriteApplication(args, out, flat, BVPolynomialMultiplicationOptions{})
		astTestMust(t, e)
		if p.currentSort != s {
			t.Fatal("first argument current sort")
		}
		if flat && st != RewriteDone || !flat && st != RewriteFailed {
			t.Fatal("wrong core selected", flat, st)
		}
		astTestMust(t, p.Close())
		astTestMust(t, m.DelRef(&out))
		out, e = m.NewRef(ast.ASTHandle{})
		astTestMust(t, e)
		p, e = NewBVPolynomialMultiplicationCore(m)
		astTestMust(t, e)
		astTestMust(t, p.MakeMultiplication(args, out, flat, BVPolynomialMultiplicationOptions{}))
		got, e := m.RefGet(out)
		astTestMust(t, e)
		want := bvMergeMul(t, m, xs[0], nested)
		if flat {
			want = bvMergeMul(t, m, xs[0], xs[1], xs[2])
		}
		if got != want {
			t.Fatal("convenience fallback operands")
		}
		astTestMust(t, p.SourceReady())
		astTestMust(t, p.Close())
		astTestMust(t, m.DelRef(&out))
	}
}

func TestBVPolynomialFlatMultiplicationPropagatesCartesianRewriteDepth(t *testing.T) {
	m, _ := bvAddFixture(t)
	xs := bvMergeVars(t, m, 5)
	left := bvMulHoistSum(t, m, xs[0], xs[1])
	right := bvMulHoistSum(t, m, xs[2], xs[3])
	nested := bvMergeMul(t, m, left, right)
	astTestPin(t, m, nested)
	out, e := m.NewRef(ast.ASTHandle{})
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	p, e := NewBVPolynomialMultiplicationCore(m)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	st, e := p.RewriteApplication(ast.SliceArguments([]ast.ASTHandle{xs[4], nested}), out, true, BVPolynomialMultiplicationOptions{SumOfMonomials: true, MonomialLimit: 3})
	astTestMust(t, e)
	if st != Rewrite2 {
		t.Fatal("flattening lost Cartesian revisit depth", st)
	}
	got, e := m.RefGet(out)
	astTestMust(t, e)
	for i, pair := range [][2]int{{0, 2}, {1, 2}, {0, 3}, {1, 3}} {
		h, e := m.GetArg(got, uint32(i))
		astTestMust(t, e)
		want := bvMergeMul(t, m, xs[4], xs[pair[0]], xs[pair[1]])
		if h != want {
			t.Fatal("flattened Cartesian factor order", i)
		}
	}
	astTestMust(t, p.SourceReady())
}
