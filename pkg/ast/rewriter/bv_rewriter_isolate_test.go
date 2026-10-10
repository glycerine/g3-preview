// Source: src/ast/rewriter/bv_rewriter.cpp:2380-2431.
package rewriter

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestBVRewriterIsolateTermOriginalOrientation(t *testing.T) {
	c, g, _ := bvPolynomialParamsFixture(t)
	m, _ := bvAddFixture(t)
	p, e := NewBVPolynomialRewriter(m, c, g, nil)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	xs := bvMergeVars(t, m, 3)
	constant := bvAddNum(t, m, 9, 8)
	out, e := m.NewRef(ast.ASTHandle{})
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	for _, minus := range []bool{false, true} {
		first := xs[0]
		if minus {
			first = bvMergeMul(t, m, bvAddNum(t, m, 255, 8), xs[0])
		}
		sum, e := m.MkBitVectorPolynomialAdd(ast.ASTHandle{}, ast.SliceArguments([]ast.ASTHandle{first, xs[1], xs[2]}))
		astTestMust(t, e)
		astTestPin(t, m, sum)
		tail, e := m.MkBitVectorPolynomialAdd(ast.ASTHandle{}, ast.SliceArguments(xs[1:]))
		astTestMust(t, e)
		astTestPin(t, m, tail)
		target, removed := first, tail
		if minus {
			target, removed = tail, first
		}
		sub, e := m.MkBitVectorSub(constant, removed)
		astTestMust(t, e)
		want, e := m.MkEq(target, sub)
		astTestMust(t, e)
		astTestPin(t, m, want)
		for _, reverse := range []bool{false, true} {
			lhs, rhs := constant, sum
			if reverse {
				lhs, rhs = rhs, lhs
			}
			yes, e := p.IsolateBitVectorTerm(lhs, rhs, out)
			astTestMust(t, e)
			got, e := m.RefGet(out)
			astTestMust(t, e)
			if !yes || got != want {
				t.Fatal(minus, reverse, yes, got, want)
			}
		}
	}
}

func TestBVRewriterIsolateTermGuardsBinaryAndConstantPredicate(t *testing.T) {
	c, g, _ := bvPolynomialParamsFixture(t)
	m, _ := bvAddFixture(t)
	p, e := NewBVPolynomialRewriter(m, c, g, nil)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	xs := bvMergeVars(t, m, 3)
	constant := bvAddNum(t, m, 9, 8)
	old := bvAddNum(t, m, 0, 8)
	out, e := m.NewRef(old)
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	yes, e := p.IsolateBitVectorTerm(xs[0], xs[1], out)
	astTestMust(t, e)
	got, e := m.RefGet(out)
	astTestMust(t, e)
	if yes || got != old || p.isolation != nil {
		t.Fatal(yes, got)
	}
	for _, first := range []ast.ASTHandle{xs[0], bvMergeMul(t, m, xs[0], bvAddNum(t, m, 255, 8)), bvMergeMul(t, m, bvAddNum(t, m, 255, 8), xs[0], xs[1])} {
		sum, e := m.MkBitVectorPolynomialAdd(ast.ASTHandle{}, ast.SliceArguments([]ast.ASTHandle{first, xs[2]}))
		astTestMust(t, e)
		astTestPin(t, m, sum)
		yes, e = p.IsolateBitVectorTerm(constant, sum, out)
		astTestMust(t, e)
		sub, e := m.MkBitVectorSub(constant, xs[2])
		astTestMust(t, e)
		want, e := m.MkEq(first, sub)
		astTestMust(t, e)
		astTestPin(t, m, want)
		got, e = m.RefGet(out)
		astTestMust(t, e)
		if !yes || got != want {
			t.Fatal("must keep first term", yes, got, want)
		}
	}
	product := bvMergeMul(t, m, constant, xs[0])
	reversed := bvMergeMul(t, m, xs[0], constant)
	for _, tc := range []struct {
		args []ast.ASTHandle
		want bool
	}{{[]ast.ASTHandle{constant, product}, true}, {[]ast.ASTHandle{constant, reversed}, false}, {[]ast.ASTHandle{product, xs[1]}, false}} {
		sum, e := m.MkBitVectorPolynomialAdd(ast.ASTHandle{}, ast.SliceArguments(tc.args))
		astTestMust(t, e)
		astTestPin(t, m, sum)
		yes, e := IsBitVectorAddMulConstant(m, sum)
		astTestMust(t, e)
		if yes != tc.want {
			t.Fatal(yes, tc.want)
		}
	}
}

func TestBVRewriterIsolateTermEveryColdReservation(t *testing.T) {
	complete := false
	sawOwnedTerms := false
	sawPublished := false
	for ordinal := int64(0); ordinal < 256 && !complete; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			c, g, _ := bvPolynomialParamsFixture(t)
			m, _ := bvAddFixture(t)
			p, e := NewBVPolynomialRewriter(m, c, g, nil)
			astTestMust(t, e)
			xs := bvMergeVars(t, m, 3)
			constant := bvAddNum(t, m, 9, 8)
			first := bvMergeMul(t, m, bvAddNum(t, m, 255, 8), xs[0])
			sum, e := m.MkBitVectorPolynomialAdd(ast.ASTHandle{}, ast.SliceArguments([]ast.ASTHandle{first, xs[1], xs[2]}))
			astTestMust(t, e)
			astTestPin(t, m, sum)
			s, e := m.GetSort(first)
			astTestMust(t, e)
			older := make([]ast.ASTHandle, 32)
			for i := range older {
				older[i], e = m.MkVar(uint32(i+100), s)
				astTestMust(t, e)
			}
			old, e := m.MkBitVectorOr(older)
			astTestMust(t, e)
			out, e := m.NewRef(old)
			astTestMust(t, e)
			defer func() { astTestMust(t, m.DelRef(&out)) }()
			m.Budget().FailAfter(ordinal)
			yes, body := p.IsolateBitVectorTerm(constant, sum, out)
			m.Budget().FailAfter(-1)
			if body == nil {
				complete = true
				nativeResult, nativeErr := m.RefGet(out)
				astTestMust(t, nativeErr)
				if nativeResult != old {
					sawPublished = true
				}
				if !yes || p.isolation != nil {
					t.Fatal(yes)
				}
			} else {
				if !numericHoistRefusal(body) {
					t.Fatal(body)
				}
				if p.isolation != nil && p.isolation.terms[0] != (ast.ASTRef{}) && p.isolation.terms[1] != (ast.ASTRef{}) {
					sawOwnedTerms = true
				}
				got, e := m.RefGet(out)
				astTestMust(t, e)
				if got != old {
					sawPublished = true
				}
				cause := p.cause
				if _, e = p.IsolateBitVectorTerm(constant, sum, out); e != cause {
					t.Fatal("replay", e, cause)
				}
				astTestMust(t, p.Close())
				if p.cause != cause {
					t.Fatal("history")
				}
			}
			astTestMust(t, p.Close())
		})
	}
	if !complete || !sawOwnedTerms || !sawPublished {
		t.Fatal(complete, sawOwnedTerms, sawPublished)
	}
}
