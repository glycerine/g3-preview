// Source: src/ast/rewriter/poly_rewriter_def.h:371-400;
// src/ast/rewriter/poly_rewriter.h:137-148.
package rewriter

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/pkg/ast"
	"testing"
)

func TestBVPolynomialFlatAdditionOnlyOneLevelAndOriginalOrder(t *testing.T) {
	m, _ := bvAddFixture(t)
	xs := bvMergeVars(t, m, 5)
	s, e := m.GetSort(xs[0])
	astTestMust(t, e)
	deeper := bvMulHoistSum(t, m, xs[1], xs[2])
	astTestPin(t, m, deeper)
	nested := bvMulHoistSum(t, m, deeper, xs[3])
	astTestPin(t, m, nested)
	args := []ast.ASTHandle{xs[0], nested, xs[4]}
	out, e := m.NewRef(ast.ASTHandle{})
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	p, e := NewBVPolynomialAdditionCore(m)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	st, e := p.RewriteFlat(s, ast.SliceArguments(args), out, BVPolynomialAdditionOptions{})
	astTestMust(t, e)
	if st != RewriteDone {
		t.Fatal("flatten fallback status", st)
	}
	got, e := m.RefGet(out)
	astTestMust(t, e)
	for i, h := range []ast.ASTHandle{xs[0], deeper, xs[3], xs[4]} {
		arg, e := m.GetArg(got, uint32(i))
		astTestMust(t, e)
		if arg != h {
			t.Fatal("flatten depth/order", i)
		}
	}
}
func TestBVPolynomialAdditionApplicationSingletonErasesSortAndNumerics(t *testing.T) {
	m, _ := bvAddFixture(t)
	xs := bvMergeVars(t, m, 2)
	out, e := m.NewRef(xs[1])
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	p, e := NewBVPolynomialAdditionCore(m)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	before := m.Budget().Used()
	m.Budget().FailAfter(0)
	st, e := p.RewriteApplication(ast.SliceArguments(xs[:1]), out, true, BVPolynomialAdditionOptions{HoistCoefficients: true, HoistMultiplication: true})
	m.Budget().FailAfter(-1)
	astTestMust(t, e)
	got, e := m.RefGet(out)
	astTestMust(t, e)
	if st != RewriteDone || got != xs[0] || p.numeric != nil || m.Budget().Used() > before {
		t.Fatal("singleton consulted core", st)
	}
	astTestMust(t, p.SourceReady())
}

func TestBVPolynomialFlatAdditionNoNestedFailedAndCollectedDone(t *testing.T) {
	for _, nested := range []bool{false, true} {
		m, _ := bvAddFixture(t)
		xs := bvMergeVars(t, m, 3)
		s, e := m.GetSort(xs[0])
		astTestMust(t, e)
		args := []ast.ASTHandle{xs[0], xs[1]}
		if nested {
			sum := bvMulHoistSum(t, m, xs[0], xs[1])
			astTestPin(t, m, sum)
			args = []ast.ASTHandle{sum, xs[0]}
		}
		out, e := m.NewRef(xs[2])
		astTestMust(t, e)
		defer func() { astTestMust(t, m.DelRef(&out)) }()
		p, e := NewBVPolynomialAdditionCore(m)
		astTestMust(t, e)
		defer func() { astTestMust(t, p.Close()) }()
		st, e := p.RewriteApplication(ast.SliceArguments(args), out, true, BVPolynomialAdditionOptions{})
		astTestMust(t, e)
		got, e := m.RefGet(out)
		astTestMust(t, e)
		if p.currentSort != s {
			t.Fatal("first argument sort not published")
		}
		if nested {
			if st != RewriteDone {
				t.Fatal(st)
			}
			two := bvAddNum(t, m, 2, 8)
			want := bvMulHoistSum(t, m, bvMergeMul(t, m, two, xs[0]), xs[1])
			if got != want {
				t.Fatal("flattened terms not collected")
			}
		} else if st != RewriteFailed || got != xs[2] {
			t.Fatal("no-nested FAILED became fallback", st)
		}
	}
}

func TestBVPolynomialAdditionApplicationFlatSelection(t *testing.T) {
	for _, flat := range []bool{false, true} {
		m, _ := bvAddFixture(t)
		xs := bvMergeVars(t, m, 4)
		nested := bvMulHoistSum(t, m, xs[0], xs[1])
		astTestPin(t, m, nested)
		out, e := m.NewRef(xs[3])
		astTestMust(t, e)
		defer func() { astTestMust(t, m.DelRef(&out)) }()
		p, e := NewBVPolynomialAdditionCore(m)
		astTestMust(t, e)
		defer func() { astTestMust(t, p.Close()) }()
		st, e := p.RewriteApplication(ast.SliceArguments([]ast.ASTHandle{nested, xs[2]}), out, flat, BVPolynomialAdditionOptions{})
		astTestMust(t, e)
		got, e := m.RefGet(out)
		astTestMust(t, e)
		if flat {
			if st != RewriteDone {
				t.Fatal(st)
			}
			n, e := m.GetNumArgs(got)
			astTestMust(t, e)
			if n != 3 {
				t.Fatal("flat selected wrong core")
			}
		}
		if !flat && (st != RewriteFailed || got != xs[3]) {
			t.Fatal("nonflat selected wrong core", st)
		}
	}
}

func TestBVPolynomialFlatAdditionEverySpillAndFallbackReservation(t *testing.T) {
	complete, failures, fallbackFailures := false, 0, 0
	for ordinal := int64(0); ordinal < 200 && !complete; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m, _ := bvAddFixture(t)
			xs := bvMergeVars(t, m, 48)
			args := make([]ast.ASTHandle, 24)
			for i := range args {
				args[i] = bvMulHoistSum(t, m, xs[2*i], xs[2*i+1])
				astTestPin(t, m, args[i])
			}
			out, e := m.NewRef(ast.ASTHandle{})
			astTestMust(t, e)
			defer func() { astTestMust(t, m.DelRef(&out)) }()
			p, e := NewBVPolynomialAdditionCore(m)
			astTestMust(t, e)
			defer func() { astTestMust(t, p.Close()) }()
			m.Budget().FailAfter(ordinal)
			st, body := p.RewriteApplication(ast.SliceArguments(args), out, true, BVPolynomialAdditionOptions{})
			m.Budget().FailAfter(-1)
			if body != nil {
				failures++
				if !numericHoistRefusal(body) {
					t.Fatal("unexpected refusal", ordinal, body)
				}
				if p.localsEnded {
					fallbackFailures++
					if p.numeric == nil || p.multiple != nil || p.visited != nil || p.sourceDone {
						t.Fatal("callee locals did not end before fallback")
					}
				}
				if _, e = p.RewriteApplication(ast.SliceArguments(args), out, true, BVPolynomialAdditionOptions{}); !errors.Is(e, ast.ErrASTPending) {
					t.Fatal("stopped call reused", e)
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
					t.Fatal("flattened arity", n)
				}
				for i, x := range xs {
					h, e := m.GetArg(got, uint32(i))
					astTestMust(t, e)
					if h != x {
						t.Fatal("source flatten order", i)
					}
				}
				astTestMust(t, p.SourceReady())
			}
			history := p.SourceReady()
			astTestMust(t, p.Close())
			if p.SourceReady() != history {
				t.Fatal("source history changed")
			}
		})
	}
	if !complete || failures == 0 || fallbackFailures == 0 {
		t.Fatal("spill/fallback horizon", complete, failures, fallbackFailures)
	}
}

func TestBVPolynomialFlatAdditionPropagatesFullHoistStatus(t *testing.T) {
	m, _ := bvAddFixture(t)
	xs := bvMergeVars(t, m, 3)
	a := bvMergeMul(t, m, xs[0], xs[1])
	b := bvMergeMul(t, m, xs[0], xs[2])
	nested := bvMulHoistSum(t, m, a, b)
	astTestPin(t, m, nested)
	zero := bvAddNum(t, m, 0, 8)
	out, e := m.NewRef(ast.ASTHandle{})
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	p, e := NewBVPolynomialAdditionCore(m)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	st, e := p.RewriteApplication(ast.SliceArguments([]ast.ASTHandle{nested, zero}), out, true, BVPolynomialAdditionOptions{HoistMultiplication: true})
	astTestMust(t, e)
	if st != RewriteFull {
		t.Fatal("flat caller lost hoist status", st)
	}
	astTestMust(t, p.SourceReady())
}
