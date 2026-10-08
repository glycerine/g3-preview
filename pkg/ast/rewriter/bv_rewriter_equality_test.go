// Source: src/ast/rewriter/bv_rewriter.cpp:2520-2635.
package rewriter

import (
	"fmt"
	"github.com/glycerine/g3/pkg/ast"
	"testing"
)

func TestBVRewriterEqualityOriginalDispatcherBranches(t *testing.T) {
	c, g, _ := bvPolynomialParamsFixture(t)
	m, _ := bvAddFixture(t)
	p, e := NewBVRewriter(m, c, g, nil)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	xs := bvMergeVars(t, m, 2)
	two := bvAddNum(t, m, 2, 8)
	three := bvAddNum(t, m, 3, 8)
	six := bvAddNum(t, m, 6, 8)
	out, e := m.NewRef(ast.ASTHandle{})
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	yes, e := m.MkTrue()
	astTestMust(t, e)
	no, e := m.MkFalse()
	astTestMust(t, e)
	orientation, e := m.MkEq(xs[0], two)
	astTestMust(t, e)
	astTestPin(t, m, orientation)
	divided, e := m.MkEq(xs[0], two)
	astTestMust(t, e)
	product := bvMergeMul(t, m, three, xs[0])
	cases := []struct {
		lhs, rhs, want ast.ASTHandle
		status         RewriteStatus
	}{
		{xs[0], xs[0], yes, RewriteDone}, {two, three, no, RewriteDone},
		{two, xs[0], orientation, RewriteDone}, {product, six, divided, Rewrite1}, {six, product, divided, Rewrite1},
	}
	for i, tc := range cases {
		st, e := p.RewriteEquality(tc.lhs, tc.rhs, out)
		astTestMust(t, e)
		got, e := m.RefGet(out)
		astTestMust(t, e)
		if st != tc.status || got != tc.want {
			t.Fatal(i, st, got, tc.status, tc.want)
		}
	}
	// FAILED with trailing disabled preserves the previous output.
	old, e := m.RefGet(out)
	astTestMust(t, e)
	st, e := p.RewriteEquality(xs[0], xs[1], out)
	astTestMust(t, e)
	got, e := m.RefGet(out)
	astTestMust(t, e)
	if st != RewriteFailed || got != old {
		t.Fatal(st, got, old)
	}
}

func TestBVRewriterEqualityRemainderAndTrailing(t *testing.T) {
	c, g, _ := bvPolynomialParamsFixture(t)
	m, fid := bvAddFixture(t)
	local, e := c.NewValue()
	astTestMust(t, e)
	defer func() { astTestMust(t, c.Del(&local)) }()
	key := "bv_urem_simpl"
	astTestMust(t, c.SetBoolName(&local, &key, true))
	p, e := NewBVRewriter(m, c, g, &local)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	xs := bvMergeVars(t, m, 2)
	out, e := m.NewRef(ast.ASTHandle{})
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	for _, op := range []int32{10, 20} {
		divisor := bvAddNum(t, m, 200, 8)
		rem, e := m.MkFamilyApp(fid, op, nil, []ast.ASTHandle{xs[0], divisor})
		astTestMust(t, e)
		astTestPin(t, m, rem)
		rhs := bvAddNum(t, m, 100, 8)
		want, e := m.MkEq(xs[0], rhs)
		astTestMust(t, e)
		astTestPin(t, m, want)
		st, e := p.RewriteEquality(rem, rhs, out)
		astTestMust(t, e)
		got, e := m.RefGet(out)
		astTestMust(t, e)
		if st != Rewrite2 || got != want {
			t.Fatal(op, st, got, want)
		}
		st, e = p.RewriteEquality(rem, bvAddNum(t, m, 200, 8), out)
		astTestMust(t, e)
		got, e = m.RefGet(out)
		astTestMust(t, e)
		no, e := m.MkFalse()
		astTestMust(t, e)
		if st != RewriteDone || got != no {
			t.Fatal("remainder outside divisor", st)
		}
	}
	key = "bv_trailing"
	astTestMust(t, c.SetBoolName(&local, &key, true))
	astTestMust(t, p.Update(&local))
	st, e := p.RewriteEquality(xs[0], xs[1], out)
	astTestMust(t, e)
	got, e := m.RefGet(out)
	astTestMust(t, e)
	want, e := m.MkEq(xs[0], xs[1])
	astTestMust(t, e)
	if st != RewriteFailed || got != want {
		t.Fatal("trailing FAILED must publish raw equality", st, got, want)
	}
}

func TestBVRewriterEqualityCancellationIsolationAndConcat(t *testing.T) {
	c, g, _ := bvPolynomialParamsFixture(t)
	m, _ := bvAddFixture(t)
	p, e := NewBVRewriter(m, c, g, nil)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	xs := bvMergeVars(t, m, 2)
	sum, e := m.MkBitVectorPolynomialAdd(ast.ASTHandle{}, ast.SliceArguments(xs))
	astTestMust(t, e)
	astTestPin(t, m, sum)
	out, e := m.NewRef(ast.ASTHandle{})
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	st, e := p.RewriteEquality(sum, xs[1], out)
	astTestMust(t, e)
	got, e := m.RefGet(out)
	astTestMust(t, e)
	want, e := m.MkEq(xs[0], bvAddNum(t, m, 0, 8))
	astTestMust(t, e)
	astTestPin(t, m, want)
	if st != RewriteDone || got != want {
		t.Fatal("cancelled common term", st, got, want)
	}
	three := bvAddNum(t, m, 3, 8)
	difference, e := m.MkBitVectorSub(three, xs[1])
	astTestMust(t, e)
	astTestPin(t, m, difference)
	want, e = m.MkEq(xs[0], difference)
	astTestMust(t, e)
	astTestPin(t, m, want)
	st, e = p.RewriteEquality(sum, three, out)
	astTestMust(t, e)
	got, e = m.RefGet(out)
	astTestMust(t, e)
	if st != Rewrite2 || got != want {
		t.Fatal("isolation after FAILED cancellation", st, got, want)
	}
	a, b, d, f := bvConcatVar(t, m, 10, 4), bvConcatVar(t, m, 11, 4), bvConcatVar(t, m, 12, 4), bvConcatVar(t, m, 13, 4)
	lhs, e := m.MkBitVectorConcatPair(a, b)
	astTestMust(t, e)
	astTestPin(t, m, lhs)
	rhs, e := m.MkBitVectorConcatPair(d, f)
	astTestMust(t, e)
	astTestPin(t, m, rhs)
	st, e = p.RewriteEquality(lhs, rhs, out)
	astTestMust(t, e)
	got, e = m.RefGet(out)
	astTestMust(t, e)
	ex := func(arg ast.ASTHandle) ast.ASTHandle {
		h, e := p.extract.Apply(3, 0, arg)
		astTestMust(t, e)
		astTestPin(t, m, h)
		return h
	}
	low, e := m.MkEq(ex(b), ex(f))
	astTestMust(t, e)
	astTestPin(t, m, low)
	high, e := m.MkEq(ex(a), ex(d))
	astTestMust(t, e)
	astTestPin(t, m, high)
	want, e = m.MkAndArgs([]ast.ASTHandle{low, high})
	astTestMust(t, e)
	astTestPin(t, m, want)
	if st != Rewrite3 || got != want {
		t.Fatal("concat fallback", st, got, want)
	}
}

func TestBVRewriterEqualityEveryColdReservation(t *testing.T) {
	for _, branch := range []string{"remainder", "cancel", "isolate", "trailing", "failed"} {
		complete := false
		sawPublished := false
		for ordinal := int64(0); ordinal < 512 && !complete; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", branch, ordinal), func(t *testing.T) {
				c, g, _ := bvPolynomialParamsFixture(t)
				m, fid := bvAddFixture(t)
				local, e := c.NewValue()
				astTestMust(t, e)
				defer func() { astTestMust(t, c.Del(&local)) }()
				key := "bv_urem_simpl"
				astTestMust(t, c.SetBoolName(&local, &key, branch == "remainder"))
				key = "bv_trailing"
				astTestMust(t, c.SetBoolName(&local, &key, branch == "trailing"))
				p, e := NewBVRewriter(m, c, g, &local)
				astTestMust(t, e)
				xs := bvMergeVars(t, m, 2)
				lhs, rhs := xs[0], xs[1]
				if branch == "remainder" {
					lhs, e = m.MkFamilyApp(fid, 10, nil, []ast.ASTHandle{xs[0], bvAddNum(t, m, 200, 8)})
					astTestMust(t, e)
					astTestPin(t, m, lhs)
					rhs = bvAddNum(t, m, 100, 8)
				}
				if branch == "cancel" || branch == "isolate" {
					lhs, e = m.MkBitVectorPolynomialAdd(ast.ASTHandle{}, ast.SliceArguments(xs))
					astTestMust(t, e)
					astTestPin(t, m, lhs)
					if branch == "isolate" {
						rhs = bvAddNum(t, m, 3, 8)
					}
				}
				sort, e := m.BitVectorSort(8)
				astTestMust(t, e)
				older := make([]ast.ASTHandle, 32)
				for i := range older {
					older[i], e = m.MkVar(uint32(i+100), sort)
					astTestMust(t, e)
				}
				old, e := m.MkBitVectorOr(older)
				astTestMust(t, e)
				out, e := m.NewRef(old)
				astTestMust(t, e)
				defer func() { astTestMust(t, m.DelRef(&out)) }()
				m.Budget().FailAfter(ordinal)
				_, body := p.RewriteEquality(lhs, rhs, out)
				m.Budget().FailAfter(-1)
				if body == nil {
					complete = true
				} else {
					if !numericHoistRefusal(body) {
						t.Fatal(body)
					}
					got, e := m.RefGet(out)
					astTestMust(t, e)
					if got != old {
						sawPublished = true
					}
					if _, e = p.RewriteEquality(lhs, rhs, out); e != body {
						t.Fatal("stopped equality replay", e, body)
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
		if !complete || branch != "failed" && !sawPublished {
			t.Fatal("equality campaign incomplete", branch)
		}
	}
}

func TestBVRewriterEqualityBitConversionOptionOrder(t *testing.T) {
	c, g, _ := bvPolynomialParamsFixture(t)
	m, _ := bvAddFixture(t)
	local, e := c.NewValue()
	astTestMust(t, e)
	defer func() { astTestMust(t, c.Del(&local)) }()
	key := "blast_eq_value"
	astTestMust(t, c.SetBoolName(&local, &key, true))
	p, e := NewBVRewriter(m, c, g, &local)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	out, e := m.NewRef(ast.ASTHandle{})
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	a, b := bvConcatVar(t, m, 20, 1), bvConcatVar(t, m, 21, 1)
	lhs, e := m.MkBitVectorOr([]ast.ASTHandle{a, b})
	astTestMust(t, e)
	astTestPin(t, m, lhs)
	one := bvAddNum(t, m, 1, 1)
	ea, e := m.MkEq(a, one)
	astTestMust(t, e)
	astTestPin(t, m, ea)
	eb, e := m.MkEq(b, one)
	astTestMust(t, e)
	astTestPin(t, m, eb)
	want, e := m.MkOrArgs([]ast.ASTHandle{ea, eb})
	astTestMust(t, e)
	astTestPin(t, m, want)
	st, e := p.RewriteEquality(lhs, one, out)
	astTestMust(t, e)
	got, e := m.RefGet(out)
	astTestMust(t, e)
	if st != Rewrite2 || got != want {
		t.Fatal("one-bit rule must precede wider blast", st, got, want)
	}
	xs := bvMergeVars(t, m, 2)
	lhs, e = m.MkBitVectorOr(xs)
	astTestMust(t, e)
	astTestPin(t, m, lhs)
	rhs := bvAddNum(t, m, 3, 8)
	equalities := make([]ast.ASTHandle, 8)
	for i := uint32(0); i < 8; i++ {
		bit, e := p.extract.Apply(i, i, lhs)
		astTestMust(t, e)
		astTestPin(t, m, bit)
		value := uint32(0)
		if i < 2 {
			value = 1
		}
		equalities[i], e = m.MkEq(bit, bvAddNum(t, m, value, 1))
		astTestMust(t, e)
		astTestPin(t, m, equalities[i])
	}
	want, e = m.MkAndArgs(equalities)
	astTestMust(t, e)
	astTestPin(t, m, want)
	st, e = p.RewriteEquality(lhs, rhs, out)
	astTestMust(t, e)
	got, e = m.RefGet(out)
	astTestMust(t, e)
	if st != Rewrite3 || got != want {
		t.Fatal("configured wider equality", st, got, want)
	}
}
