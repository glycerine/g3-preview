package rewriter

// Source: src/ast/rewriter/bv_rewriter.cpp:891-985.

import (
	"fmt"
	"github.com/glycerine/g3/pkg/ast"
	"testing"
)

func TestBVRewriterArithmeticShiftOriginalBranches(t *testing.T) {
	c, g, _ := bvPolynomialParamsFixture(t)
	m, _ := bvAddFixture(t)
	p, e := NewBVRewriter(m, c, g, nil)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	out, e := m.NewRef(ast.ASTHandle{})
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	x := bvMergeVars(t, m, 1)[0]
	for _, tc := range []struct{ a, b, want uint32 }{{128, 1, 192}, {255, 2, 255}, {127, 1, 63}, {128, 8, 255}, {128, 9, 255}, {127, 8, 0}, {1, 255, 0}} {
		st, e := p.RewriteArithmeticShiftRight(bvAddNum(t, m, tc.a, 8), bvAddNum(t, m, tc.b, 8), out)
		astTestMust(t, e)
		got, e := m.RefGet(out)
		astTestMust(t, e)
		if st != RewriteDone || got != bvAddNum(t, m, tc.want, 8) {
			t.Fatal(tc, st, got)
		}
	}
	st, e := p.RewriteArithmeticShiftRight(x, bvAddNum(t, m, 0, 8), out)
	astTestMust(t, e)
	got, e := m.RefGet(out)
	astTestMust(t, e)
	if st != RewriteDone || got != x {
		t.Fatal("zero", st)
	}
	st, e = p.RewriteArithmeticShiftRight(x, bvAddNum(t, m, 2, 8), out)
	astTestMust(t, e)
	now, e := m.RefGet(out)
	astTestMust(t, e)
	if st != RewriteFailed || now != got {
		t.Fatal("symbolic source #if0", st)
	}
	for _, amount := range []uint32{2, 254} {
		inner, e := m.MkBitVectorASHR(x, bvAddNum(t, m, 3, 8))
		astTestMust(t, e)
		astTestPin(t, m, inner)
		st, e = p.RewriteArithmeticShiftRight(inner, bvAddNum(t, m, amount, 8), out)
		astTestMust(t, e)
		wantAmount := amount + 3
		if wantAmount > 8 {
			wantAmount = 8
		}
		want, e := m.MkBitVectorASHR(x, bvAddNum(t, m, wantAmount, 8))
		astTestMust(t, e)
		astTestPin(t, m, want)
		got, e := m.RefGet(out)
		astTestMust(t, e)
		if st != Rewrite1 || got != want {
			t.Fatal("nested", amount, st, got, want)
		}
	}
}

func TestBVRewriterArithmeticShiftEveryColdReservationAndPublication(t *testing.T) {
	for _, mode := range []string{"negative", "positive", "oversized", "zero", "nested", "clamp", "failed"} {
		complete, published := false, false
		for ordinal := int64(0); ordinal < 1024 && !complete; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", mode, ordinal), func(t *testing.T) {
				c, g, _ := bvPolynomialParamsFixture(t)
				m, _ := bvAddFixture(t)
				p, e := NewBVRewriter(m, c, g, nil)
				astTestMust(t, e)
				x := bvMergeVars(t, m, 1)[0]
				lhs, rhs := bvAddNum(t, m, 128, 8), bvAddNum(t, m, 1, 8)
				switch mode {
				case "positive":
					lhs = bvAddNum(t, m, 127, 8)
				case "oversized":
					rhs = bvAddNum(t, m, 8, 8)
				case "zero":
					lhs = x
					rhs = bvAddNum(t, m, 0, 8)
				case "failed":
					lhs = x
				case "nested", "clamp":
					lhs, e = m.MkBitVectorASHR(x, bvAddNum(t, m, 3, 8))
					astTestMust(t, e)
					astTestPin(t, m, lhs)
					if mode == "clamp" {
						rhs = bvAddNum(t, m, 254, 8)
					}
				}
				sort, e := m.BitVectorSort(8)
				astTestMust(t, e)
				args := make([]ast.ASTHandle, 32)
				for i := range args {
					args[i], e = m.MkVar(uint32(i+100), sort)
					astTestMust(t, e)
				}
				old, e := m.MkBitVectorOr(args)
				astTestMust(t, e)
				out, e := m.NewRef(old)
				astTestMust(t, e)
				defer func() { astTestMust(t, m.DelRef(&out)) }()
				m.Budget().FailAfter(ordinal)
				st, body := p.RewriteArithmeticShiftRight(lhs, rhs, out)
				m.Budget().FailAfter(-1)
				if body == nil {
					complete = true
					nativeResult, nativeErr := m.RefGet(out)
					astTestMust(t, nativeErr)
					if nativeResult != old {
						published = true
					}
					if (st == RewriteFailed) != (mode == "failed") {
						t.Fatal(mode, st)
					}
				} else {
					if !numericHoistRefusal(body) {
						t.Fatal(body)
					}
					got, e := m.RefGet(out)
					astTestMust(t, e)
					if got != old {
						published = true
						if st == RewriteFailed {
							t.Fatal("published status", st)
						}
					}
					if _, e = p.RewriteArithmeticShiftRight(lhs, rhs, out); e != body {
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
		if !complete || (mode != "failed" && !published) {
			t.Fatal("incomplete arithmetic shift campaign", mode, complete, published)
		}
	}
}

func TestBVRewriterArithmeticShiftOriginalWidthOverwrite(t *testing.T) {
	c, g, _ := bvPolynomialParamsFixture(t)
	m, _ := bvAddFixture(t)
	p, e := NewBVRewriter(m, c, g, nil)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	out, e := m.NewRef(ast.ASTHandle{})
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	// The lhs recognition overwrites the rhs recognition's stored width.
	st, e := p.RewriteArithmeticShiftRight(bvAddNum(t, m, 128, 8), bvAddNum(t, m, 1, 4), out)
	astTestMust(t, e)
	got, e := m.RefGet(out)
	astTestMust(t, e)
	if st != RewriteDone || got != bvAddNum(t, m, 192, 8) {
		t.Fatal(st, got)
	}
	// Nested amount recognition overwrites the outer rhs stored width.
	x := bvConcatVar(t, m, 0, 4)
	inner, e := m.MkBitVectorASHR(x, bvAddNum(t, m, 3, 4))
	astTestMust(t, e)
	astTestPin(t, m, inner)
	st, e = p.RewriteArithmeticShiftRight(inner, bvAddNum(t, m, 2, 8), out)
	astTestMust(t, e)
	want, e := m.MkBitVectorASHR(x, bvAddNum(t, m, 4, 4))
	astTestMust(t, e)
	astTestPin(t, m, want)
	got, e = m.RefGet(out)
	astTestMust(t, e)
	if st != Rewrite1 || got != want {
		t.Fatal("nested last width", st, got, want)
	}
}
