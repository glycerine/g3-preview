// Source: src/ast/rewriter/bv_rewriter.cpp:2657-2704.
package rewriter

import (
	"fmt"
	"github.com/glycerine/g3/pkg/ast"
	"testing"
)

func TestBVRewriterIteOriginalBranchOrderAndIdentities(t *testing.T) {
	c, g, _ := bvPolynomialParamsFixture(t)
	m, _ := bvAddFixture(t)
	p, e := NewBVRewriter(m, c, g, nil)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	xs := bvMergeVars(t, m, 2)
	yes, e := m.MkTrue()
	astTestMust(t, e)
	out, e := m.NewRef(ast.ASTHandle{})
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	st, e := p.RewriteIte(yes, xs[0], xs[0], out)
	astTestMust(t, e)
	got, e := m.RefGet(out)
	astTestMust(t, e)
	if st != Rewrite1 || got != xs[0] {
		t.Fatal("equal branches", st, got)
	}
	boolSort, e := m.MkBoolSort()
	astTestMust(t, e)
	condition, e := m.MkVar(10, boolSort)
	astTestMust(t, e)
	astTestPin(t, m, condition)
	neg, e := m.MkNot(condition)
	astTestMust(t, e)
	astTestPin(t, m, neg)
	want, e := m.MkIte(condition, xs[1], xs[0])
	astTestMust(t, e)
	astTestPin(t, m, want)
	st, e = p.RewriteIte(neg, xs[0], xs[1], out)
	astTestMust(t, e)
	got, e = m.RefGet(out)
	astTestMust(t, e)
	if st != Rewrite1 || got != want {
		t.Fatal("negated condition", st, got, want)
	}
	local, e := c.NewValue()
	astTestMust(t, e)
	defer func() { astTestMust(t, c.Del(&local)) }()
	key := "bv_ite2id"
	astTestMust(t, c.SetBoolName(&local, &key, true))
	astTestMust(t, p.Update(&local))
	eq, e := m.MkEq(xs[0], xs[1])
	astTestMust(t, e)
	astTestPin(t, m, eq)
	for _, branches := range [][2]ast.ASTHandle{{xs[0], xs[1]}, {xs[1], xs[0]}} {
		st, e = p.RewriteIte(eq, branches[0], branches[1], out)
		astTestMust(t, e)
		got, e = m.RefGet(out)
		astTestMust(t, e)
		if st != Rewrite1 || got != branches[1] {
			t.Fatal("identity pair returns else branch", st, got)
		}
	}
}

func TestBVRewriterIteOneBitNumeralBranches(t *testing.T) {
	c, g, _ := bvPolynomialParamsFixture(t)
	m, _ := bvAddFixture(t)
	local, e := c.NewValue()
	astTestMust(t, e)
	defer func() { astTestMust(t, c.Del(&local)) }()
	key := "bv_ite2id"
	astTestMust(t, c.SetBoolName(&local, &key, true))
	p, e := NewBVRewriter(m, c, g, &local)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	x := bvConcatVar(t, m, 0, 1)
	one, zero := bvAddNum(t, m, 1, 1), bvAddNum(t, m, 0, 1)
	notX, e := m.MkBitVectorNot(x)
	astTestMust(t, e)
	astTestPin(t, m, notX)
	out, e := m.NewRef(ast.ASTHandle{})
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	for _, reverse := range []bool{false, true} {
		lhs, rhs := x, one
		if reverse {
			lhs, rhs = rhs, lhs
		}
		eq, e := m.MkEq(lhs, rhs)
		astTestMust(t, e)
		astTestPin(t, m, eq)
		for _, pair := range [][3]ast.ASTHandle{{one, zero, x}, {zero, one, notX}} {
			st, e := p.RewriteIte(eq, pair[0], pair[1], out)
			astTestMust(t, e)
			got, e := m.RefGet(out)
			astTestMust(t, e)
			if st != Rewrite1 || got != pair[2] {
				t.Fatal(reverse, st, got, pair[2])
			}
		}
	}
}

func TestBVRewriterIteUnreachedConditionAndWideBranches(t *testing.T) {
	c, g, _ := bvPolynomialParamsFixture(t)
	m, _ := bvAddFixture(t)
	local, e := c.NewValue()
	astTestMust(t, e)
	defer func() { astTestMust(t, c.Del(&local)) }()
	key := "bv_ite2id"
	astTestMust(t, c.SetBoolName(&local, &key, true))
	p, e := NewBVRewriter(m, c, g, &local)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	x := bvConcatVar(t, m, 20, 1)
	one := bvAddNum(t, m, 1, 1)
	out, e := m.NewRef(x)
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	// Equal branches bypass even a null condition and every numeric placement.
	m.Budget().FailAfter(0)
	st, e := p.RewriteIte(ast.ASTHandle{}, x, x, out)
	m.Budget().FailAfter(-1)
	astTestMust(t, e)
	if st != Rewrite1 {
		t.Fatal(st)
	}
	eq, e := m.MkEq(x, one)
	astTestMust(t, e)
	astTestPin(t, m, eq)
	wideT, wideE := bvAddNum(t, m, 1, 65), bvAddNum(t, m, 0, 65)
	st, e = p.RewriteIte(eq, wideT, wideE, out)
	astTestMust(t, e)
	got, e := m.RefGet(out)
	astTestMust(t, e)
	if st != RewriteFailed || got != x || p.ite != nil {
		t.Fatal("wide branch guard", st, got)
	}
	// RHS nonnumeral ends the copy chain without reading the two branch numerals.
	y := bvConcatVar(t, m, 21, 1)
	eq, e = m.MkEq(x, y)
	astTestMust(t, e)
	astTestPin(t, m, eq)
	st, e = p.RewriteIte(eq, one, bvAddNum(t, m, 0, 1), out)
	astTestMust(t, e)
	if st != RewriteFailed || p.ite != nil {
		t.Fatal("nonnumeral condition guard", st)
	}
}

func TestBVRewriterIteEveryColdReservationAndPublication(t *testing.T) {
	for _, branch := range []string{"not_condition", "numeral_not", "numeral_identity", "failed"} {
		complete, published := false, false
		for ordinal := int64(0); ordinal < 512 && !complete; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", branch, ordinal), func(t *testing.T) {
				c, g, _ := bvPolynomialParamsFixture(t)
				m, _ := bvAddFixture(t)
				local, e := c.NewValue()
				astTestMust(t, e)
				defer func() { astTestMust(t, c.Del(&local)) }()
				key := "bv_ite2id"
				astTestMust(t, c.SetBoolName(&local, &key, true))
				p, e := NewBVRewriter(m, c, g, &local)
				astTestMust(t, e)
				x := bvConcatVar(t, m, 0, 1)
				one, zero := bvAddNum(t, m, 1, 1), bvAddNum(t, m, 0, 1)
				condition, e := m.MkEq(x, one)
				astTestMust(t, e)
				astTestPin(t, m, condition)
				tval, eval := zero, one
				if branch == "numeral_identity" {
					tval, eval = one, zero
				}
				if branch == "failed" {
					tval, eval = bvAddNum(t, m, 1, 65), bvAddNum(t, m, 0, 65)
				}
				if branch == "not_condition" {
					condition, e = m.MkNot(condition)
					astTestMust(t, e)
					astTestPin(t, m, condition)
				}
				sort, e := m.BitVectorSort(8)
				astTestMust(t, e)
				older := make([]ast.ASTHandle, 32)
				for i := range older {
					older[i], e = m.MkVar(uint32(100+i), sort)
					astTestMust(t, e)
				}
				old, e := m.MkBitVectorOr(older)
				astTestMust(t, e)
				out, e := m.NewRef(old)
				astTestMust(t, e)
				defer func() { astTestMust(t, m.DelRef(&out)) }()
				m.Budget().FailAfter(ordinal)
				_, body := p.RewriteIte(condition, tval, eval, out)
				m.Budget().FailAfter(-1)
				if body == nil {
					complete = true
					nativeResult, nativeErr := m.RefGet(out)
					astTestMust(t, nativeErr)
					if nativeResult != old {
						published = true
					}
				} else {
					if !numericHoistRefusal(body) {
						t.Fatal(body)
					}
					got, e := m.RefGet(out)
					astTestMust(t, e)
					if got != old {
						published = true
					}
					if _, e = p.RewriteIte(ast.ASTHandle{}, tval, eval, out); e != body {
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
		if !complete || branch != "failed" && !published {
			t.Fatal("ITE cold campaign incomplete", branch, complete, published)
		}
	}
}
