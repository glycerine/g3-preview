// Source: src/ast/rewriter/bv_rewriter.cpp:590-790.
package rewriter

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestBVRewriterExtractOriginalBranchesAndStatuses(t *testing.T) {
	m, _ := bvAddFixture(t)
	p := bvNotTestOwner(t, m, false, false)
	xs := bvMergeVars(t, m, 3)
	x, y, z := xs[0], xs[1], xs[2]
	ex := func(hi, lo uint32, h ast.ASTHandle) ast.ASTHandle {
		v, e := m.MkBitVectorExtract(hi, lo, h)
		astTestMust(t, e)
		astTestPin(t, m, v)
		return v
	}
	cat := func(hs ...ast.ASTHandle) ast.ASTHandle {
		v, e := m.MkBitVectorConcat(hs)
		astTestMust(t, e)
		astTestPin(t, m, v)
		return v
	}
	op := func(k int32, hs ...ast.ASTHandle) ast.ASTHandle {
		v, e := m.MkFamilyApp(p.bvFamily, k, nil, hs)
		astTestMust(t, e)
		astTestPin(t, m, v)
		return v
	}
	condition, e := m.MkEq(x, y)
	astTestMust(t, e)
	astTestPin(t, m, condition)
	ite, e := m.MkIte(condition, x, y)
	astTestMust(t, e)
	astTestPin(t, m, ite)
	iteResult, e := m.MkIte(condition, ex(6, 3, x), ex(6, 3, y))
	astTestMust(t, e)
	astTestPin(t, m, iteResult)
	concat := cat(x, y, z)
	for _, tc := range []struct {
		name      string
		hi, lo    uint32
		arg, want ast.ASTHandle
		st        RewriteStatus
	}{
		{"identity", 7, 0, x, x, RewriteDone},
		{"numeric", 6, 2, bvAddNum(t, m, 255, 8), bvAddNum(t, m, 31, 5), RewriteDone},
		{"nested", 3, 2, ex(6, 1, x), ex(4, 3, x), RewriteDone},
		{"concat-child", 15, 8, concat, y, RewriteDone},
		{"concat-fragment", 13, 10, concat, ex(5, 2, y), Rewrite1},
		{"concat-whole-run", 15, 0, concat, cat(y, z), RewriteDone},
		{"concat-prefix-fragment", 13, 0, concat, cat(ex(5, 0, y), z), Rewrite2},
		{"concat-suffix-fragment", 15, 2, concat, cat(y, ex(7, 2, z)), Rewrite2},
		{"concat-two-fragments", 13, 2, concat, cat(ex(5, 0, y), ex(7, 2, z)), Rewrite2},
		{"not", 6, 3, op(32, x), op(32, ex(6, 3, x)), Rewrite2},
		{"or", 6, 3, op(31, x, y), op(31, ex(6, 3, x), ex(6, 3, y)), Rewrite2},
		{"xor", 6, 3, op(33, x, y), op(33, ex(6, 3, x), ex(6, 3, y)), Rewrite2},
		{"add-low-zero", 3, 0, op(4, x, y), op(4, ex(3, 0, x), ex(3, 0, y)), Rewrite2},
		{"mul-low-zero", 3, 0, op(6, x, y), op(6, ex(3, 0, x), ex(3, 0, y)), Rewrite2},
		{"ite", 6, 3, ite, iteResult, Rewrite2},
		{"failed", 6, 3, x, ast.ASTHandle{}, RewriteFailed},
		{"add-low-nonzero-failed", 6, 3, op(4, x, y), ast.ASTHandle{}, RewriteFailed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, e := m.NewRef(z)
			astTestMust(t, e)
			defer func() { astTestMust(t, m.DelRef(&out)) }()
			st, e := p.RewriteExtract(tc.hi, tc.lo, tc.arg, out)
			astTestMust(t, e)
			got, e := m.RefGet(out)
			astTestMust(t, e)
			want := tc.want
			if tc.st == RewriteFailed {
				want = z
			}
			if st != tc.st || got != want || p.extraction != nil || p.propagation != nil {
				t.Fatal("original extraction", st, tc.st, got, want)
			}
		})
	}
	st, e := p.RewriteExtract(6, 3, x, ast.ASTRef{})
	astTestMust(t, e)
	if st != RewriteFailed {
		t.Fatal("FAILED read absent output")
	}
}

func TestBVRewriterExtractEveryColdReservationAndPublication(t *testing.T) {
	for _, mode := range []string{"numeric", "nested", "concat-fragment", "concat-span", "distribute", "ite", "failed"} {
		complete, published := false, false
		for ordinal := int64(0); ordinal < 1024 && !complete; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", mode, ordinal), func(t *testing.T) {
				m, _ := bvAddFixture(t)
				xs := bvMergeVars(t, m, 40)
				x, y := xs[0], xs[1]
				arg, hi, lo, wantStatus := x, uint32(6), uint32(3), Rewrite2
				var e error
				switch mode {
				case "numeric":
					arg = bvAddNum(t, m, 255, 8)
					wantStatus = RewriteDone
				case "nested":
					arg, e = m.MkBitVectorExtract(7, 1, x)
					wantStatus = RewriteDone
				case "concat-fragment":
					arg, e = m.MkBitVectorConcat([]ast.ASTHandle{x, y})
					wantStatus = Rewrite1
				case "concat-span":
					arg, e = m.MkBitVectorConcat(xs)
					hi, lo = 318, 1
				case "distribute":
					arg, e = m.MkBitVectorOr(xs)
				case "ite":
					condition, er := m.MkEq(x, y)
					astTestMust(t, er)
					astTestPin(t, m, condition)
					arg, e = m.MkIte(condition, x, y)
				case "failed":
					wantStatus = RewriteFailed
				}
				astTestMust(t, e)
				astTestPin(t, m, arg)
				p := bvNotTestOwner(t, m, false, false)
				sort, e := m.BitVectorSort(8)
				astTestMust(t, e)
				var oldArgs [32]ast.ASTHandle
				for i := range oldArgs {
					oldArgs[i], e = m.MkVar(uint32(i+100), sort)
					astTestMust(t, e)
				}
				old, e := m.MkBitVectorOr(oldArgs[:])
				astTestMust(t, e)
				out, e := m.NewRef(old)
				astTestMust(t, e)
				defer func() { astTestMust(t, m.DelRef(&out)) }()
				m.Budget().FailAfter(ordinal)
				st, body := p.RewriteExtract(hi, lo, arg, out)
				m.Budget().FailAfter(-1)
				got, e := m.RefGet(out)
				astTestMust(t, e)
				if body == nil {
					complete = true
					if st != wantStatus || p.extraction != nil || mode == "failed" && got != old {
						t.Fatal("source status/end", st, wantStatus)
					}
				} else {
					if !numericHoistRefusal(body) {
						t.Fatal(body)
					}
					if got != old {
						published = true
						if st != wantStatus {
							t.Fatal("publication status", st, wantStatus)
						}
					}
					_, replay := p.RewriteExtract(hi, lo, arg, out)
					if replay != body {
						t.Fatal("stopped replay", replay, body)
					}
				}
				cause := p.cause
				astTestMust(t, p.Close())
				astTestMust(t, p.Close())
				if cause != p.cause {
					t.Fatal("cleanup changed history")
				}
			})
		}
		if !complete || mode != "failed" && !published {
			t.Fatal("cold extraction campaign incomplete", mode, complete, published)
		}
	}
}
