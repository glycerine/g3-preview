package rewriter

// Source: src/ast/rewriter/bv_rewriter.cpp:792-889.

import (
	"fmt"
	"github.com/glycerine/g3/pkg/ast"
	"testing"
)

func TestBVRewriterLogicalShiftsOriginalBranches(t *testing.T) {
	c, g, _ := bvPolynomialParamsFixture(t)
	m, _ := bvAddFixture(t)
	p, e := NewBVRewriter(m, c, g, nil)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	out, e := m.NewRef(ast.ASTHandle{})
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	x := bvMergeVars(t, m, 1)[0]
	ex := func(hi, lo uint32) ast.ASTHandle {
		h, e := m.MkBitVectorExtract(hi, lo, x)
		astTestMust(t, e)
		astTestPin(t, m, h)
		return h
	}
	concat := func(a, b ast.ASTHandle) ast.ASTHandle {
		h, e := m.MkBitVectorConcatPair(a, b)
		astTestMust(t, e)
		astTestPin(t, m, h)
		return h
	}
	for _, left := range []bool{true, false} {
		call := p.RewriteShiftLeft
		if !left {
			call = p.RewriteLogicalShiftRight
		}
		cases := []struct {
			lhs, rhs, want ast.ASTHandle
			status         RewriteStatus
		}{
			{x, bvAddNum(t, m, 0, 8), x, RewriteDone},
			{x, bvAddNum(t, m, 8, 8), bvAddNum(t, m, 0, 8), RewriteDone},
			{x, bvAddNum(t, m, 255, 8), bvAddNum(t, m, 0, 8), RewriteDone},
			{x, x, ast.ASTHandle{}, RewriteFailed},
		}
		numeric := uint32(28)
		symbolic := concat(ex(5, 0), bvAddNum(t, m, 0, 2))
		if !left {
			numeric = 1
			symbolic = concat(bvAddNum(t, m, 0, 2), ex(7, 2))
		}
		cases = append(cases, struct {
			lhs, rhs, want ast.ASTHandle
			status         RewriteStatus
		}{bvAddNum(t, m, 7, 8), bvAddNum(t, m, 2, 8), bvAddNum(t, m, numeric, 8), RewriteDone})
		cases = append(cases, struct {
			lhs, rhs, want ast.ASTHandle
			status         RewriteStatus
		}{x, bvAddNum(t, m, 2, 8), symbolic, Rewrite2})
		// The lhs sort, not the shift's stored width, selects the bound/result width.
		cases = append(cases, struct {
			lhs, rhs, want ast.ASTHandle
			status         RewriteStatus
		}{bvAddNum(t, m, 7, 8), bvAddNum(t, m, 2, 4), bvAddNum(t, m, numeric, 8), RewriteDone})
		for _, tc := range cases {
			before, e := m.RefGet(out)
			astTestMust(t, e)
			st, e := call(tc.lhs, tc.rhs, out)
			astTestMust(t, e)
			got, e := m.RefGet(out)
			astTestMust(t, e)
			want := tc.want
			if st == RewriteFailed {
				want = before
			}
			if st != tc.status || got != want {
				t.Fatal(left, st, tc.status, got, want)
			}
		}
	}
}

func TestBVRewriterLogicalShiftsEveryColdReservationAndPublication(t *testing.T) {
	for _, left := range []bool{true, false} {
		for _, mode := range []string{"numeric", "symbolic", "zero", "large", "failed"} {
			complete, published := false, false
			for ordinal := int64(0); ordinal < 1024 && !complete; ordinal++ {
				t.Run(fmt.Sprintf("%t/%s/%d", left, mode, ordinal), func(t *testing.T) {
					c, g, _ := bvPolynomialParamsFixture(t)
					m, _ := bvAddFixture(t)
					p, e := NewBVRewriter(m, c, g, nil)
					astTestMust(t, e)
					x := bvMergeVars(t, m, 1)[0]
					lhs, rhs := bvAddNum(t, m, 127, 8), bvAddNum(t, m, 2, 8)
					switch mode {
					case "symbolic":
						lhs = x
					case "zero":
						lhs = x
						rhs = bvAddNum(t, m, 0, 8)
					case "large":
						lhs = x
						rhs = bvAddNum(t, m, 8, 8)
					case "failed":
						rhs = x
					}
					sort, e := m.BitVectorSort(8)
					astTestMust(t, e)
					oldArgs := make([]ast.ASTHandle, 32)
					for i := range oldArgs {
						oldArgs[i], e = m.MkVar(uint32(i+100), sort)
						astTestMust(t, e)
					}
					old, e := m.MkBitVectorOr(oldArgs)
					astTestMust(t, e)
					out, e := m.NewRef(old)
					astTestMust(t, e)
					defer func() { astTestMust(t, m.DelRef(&out)) }()
					call := p.RewriteShiftLeft
					if !left {
						call = p.RewriteLogicalShiftRight
					}
					m.Budget().FailAfter(ordinal)
					st, body := call(lhs, rhs, out)
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
						if _, e = call(lhs, rhs, out); e != body {
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
				t.Fatal("incomplete shift campaign", left, mode, complete, published)
			}
		}
	}
}

func TestBVRewriterLogicalShiftsWideNumericAndUnreachedLeftCopy(t *testing.T) {
	c, g, _ := bvPolynomialParamsFixture(t)
	m, _ := bvAddFixture(t)
	p, e := NewBVRewriter(m, c, g, nil)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	out, e := m.NewRef(ast.ASTHandle{})
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	for _, left := range []bool{true, false} {
		call := p.RewriteShiftLeft
		want := uint32(12)
		if !left {
			call = p.RewriteLogicalShiftRight
			want = 0
		}
		st, e := call(bvAddNum(t, m, 3, 65), bvAddNum(t, m, 2, 65), out)
		astTestMust(t, e)
		got, e := m.RefGet(out)
		astTestMust(t, e)
		if st != RewriteDone || got != bvAddNum(t, m, want, 65) {
			t.Fatal("wide numeric", left, st, got)
		}
	}
	// The threshold uses the wide lhs sort before attempting its numeral read.
	lhs := bvAddNum(t, m, 3, 65)
	st, e := p.RewriteShiftLeft(lhs, bvAddNum(t, m, 65, 65), out)
	astTestMust(t, e)
	got, e := m.RefGet(out)
	astTestMust(t, e)
	if st != RewriteDone || got != bvAddNum(t, m, 0, 65) {
		t.Fatal("wide threshold", st)
	}
}
