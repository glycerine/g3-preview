package rewriter

// Source: src/ast/rewriter/bv_rewriter.cpp:349-434.

import (
	"fmt"
	"github.com/glycerine/g3/pkg/ast"
	"testing"
)

func TestBVRewriterLEConcatOriginalShapesAndStatuses(t *testing.T) {
	m, _ := bvAddFixture(t)
	p := bvNotTestOwner(t, m, false, false)
	xs := bvMergeVars(t, m, 3)
	x, y, z := xs[0], xs[1], xs[2]
	cat := func(hs ...ast.ASTHandle) ast.ASTHandle { return bvPropagationConcat(t, m, hs...) }
	cmp := func(signed bool, a, b ast.ASTHandle) ast.ASTHandle {
		var h ast.ASTHandle
		var e error
		if signed {
			h, e = m.MkBitVectorSLE(a, b)
		} else {
			h, e = m.MkBitVectorULE(a, b)
		}
		astTestMust(t, e)
		astTestPin(t, m, h)
		return h
	}
	yes, e := m.MkTrue()
	astTestMust(t, e)
	astTestPin(t, m, yes)
	no, e := m.MkFalse()
	astTestMust(t, e)
	astTestPin(t, m, no)
	n := func(v uint32, w uint32) ast.ASTHandle { return bvAddNum(t, m, v, w) }
	for _, tc := range []struct {
		name       string
		signed     bool
		a, b, want ast.ASTHandle
		st         RewriteStatus
	}{
		{"numeric-high-less", false, cat(n(1, 4), x), cat(n(2, 4), y), yes, RewriteDone},
		{"numeric-high-greater", false, cat(n(3, 4), x), cat(n(2, 4), y), no, RewriteDone},
		{"numeric-signed-high", true, cat(n(15, 4), x), cat(n(0, 4), y), yes, RewriteDone},
		{"numeric-equal-high", false, cat(n(7, 4), x), cat(n(7, 4), y), cmp(false, x, y), Rewrite2},
		{"signed-equal-high-unsigned-tail", true, cat(n(15, 4), x), cat(n(15, 4), y), cmp(false, x, y), Rewrite2},
		{"numeric-partial-a", false, cat(n(13, 4), bvPropagationVar(t, m, 9, 6)), cat(n(3, 2), x), cmp(false, cat(n(1, 2), bvPropagationVar(t, m, 9, 6)), x), Rewrite2},
		{"numeric-partial-b", false, cat(n(3, 2), x), cat(n(13, 4), bvPropagationVar(t, m, 9, 6)), cmp(false, x, cat(n(1, 2), bvPropagationVar(t, m, 9, 6))), Rewrite2},
		{"common-prefix", false, cat(x, y), cat(x, z), cmp(false, y, z), Rewrite2},
		{"signed-common-prefix-unsigned-tail", true, cat(x, y), cat(x, z), cmp(false, y, z), Rewrite2},
		{"common-postfix", false, cat(y, x), cat(z, x), cmp(false, y, z), Rewrite2},
		{"signed-common-postfix", true, cat(y, x), cat(z, x), cmp(true, y, z), Rewrite2},
		{"failed", false, cat(x, y), cat(z, x), ast.ASTHandle{}, RewriteFailed},
		{"nonconcat", false, x, cat(y, z), ast.ASTHandle{}, RewriteFailed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, e := m.NewRef(yes)
			astTestMust(t, e)
			defer func() { astTestMust(t, m.DelRef(&out)) }()
			st, e := p.RewriteLEConcat(tc.signed, tc.a, tc.b, out)
			astTestMust(t, e)
			got, e := m.RefGet(out)
			astTestMust(t, e)
			want := tc.want
			if st == RewriteFailed {
				want = yes
			}
			if st != tc.st || got != want || p.leConcat != nil {
				t.Fatal("original concat comparison", st, tc.st, got, want)
			}
		})
	}
	st, e := p.RewriteLEConcat(false, x, y, ast.ASTRef{})
	astTestMust(t, e)
	if st != RewriteFailed {
		t.Fatal("absent output guard")
	}
}

func TestBVRewriterLEConcatEveryColdReservationAndPublication(t *testing.T) {
	for _, mode := range []string{"numeric-less", "partial-a", "partial-b", "prefix", "postfix", "failed"} {
		complete, published := false, false
		for ordinal := int64(0); ordinal < 2048 && !complete; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", mode, ordinal), func(t *testing.T) {
				m, _ := bvAddFixture(t)
				p := bvNotTestOwner(t, m, false, false)
				xs := bvMergeVars(t, m, 40)
				cat := func(hs ...ast.ASTHandle) ast.ASTHandle { return bvPropagationConcat(t, m, hs...) }
				left, right := cat(xs[0], xs[1]), cat(xs[2], xs[3])
				want := Rewrite2
				switch mode {
				case "numeric-less":
					left, right = cat(bvAddNum(t, m, 1, 4), xs[0]), cat(bvAddNum(t, m, 2, 4), xs[1])
					want = RewriteDone
				case "partial-a", "partial-b":
					aa := append([]ast.ASTHandle{bvAddNum(t, m, 13, 4)}, xs...)
					bb := append([]ast.ASTHandle{bvAddNum(t, m, 3, 2), bvPropagationVar(t, m, 90, 10)}, xs[1:]...)
					left, right = cat(aa...), cat(bb...)
					if mode == "partial-b" {
						left, right = right, left
					}
				case "prefix":
					aa := append([]ast.ASTHandle{xs[0]}, xs...)
					bb := append([]ast.ASTHandle{xs[0], xs[1]}, xs[1:]...)
					left, right = cat(aa...), cat(bb...)
				case "postfix":
					aa := append(append([]ast.ASTHandle(nil), xs...), xs[0])
					bb := append(append([]ast.ASTHandle{xs[1]}, xs[1:]...), xs[0])
					left, right = cat(aa...), cat(bb...)
				case "failed":
					want = RewriteFailed
				}
				s, e := m.BitVectorSort(8)
				astTestMust(t, e)
				var oldArgs [32]ast.ASTHandle
				for i := range oldArgs {
					oldArgs[i], e = m.MkVar(uint32(i+100), s)
					astTestMust(t, e)
				}
				old, e := m.MkBitVectorOr(oldArgs[:])
				astTestMust(t, e)
				out, e := m.NewRef(old)
				astTestMust(t, e)
				defer func() { astTestMust(t, m.DelRef(&out)) }()
				m.Budget().FailAfter(ordinal)
				st, body := p.RewriteLEConcat(true, left, right, out)
				m.Budget().FailAfter(-1)
				got, e := m.RefGet(out)
				astTestMust(t, e)
				if body == nil {
					complete = true
					nativeResult, nativeErr := m.RefGet(out)
					astTestMust(t, nativeErr)
					if nativeResult != old {
						published = true
					}
					if st != want || p.leConcat != nil || want == RewriteFailed && got != old {
						t.Fatal("source return", st, want)
					}
				} else {
					if !numericHoistRefusal(body) {
						t.Fatal(body)
					}
					if got != old {
						published = true
						if st != want {
							t.Fatal("publication status", st, want)
						}
					}
					_, replay := p.RewriteLEConcat(true, left, right, out)
					if replay != body {
						t.Fatal("stopped replay", replay, body)
					}
				}
				cause := p.cause
				astTestMust(t, p.Close())
				astTestMust(t, p.Close())
				if p.cause != cause {
					t.Fatal("history changed")
				}
			})
		}
		if !complete || mode != "failed" && !published {
			t.Fatal("cold concat comparison incomplete", mode, complete, published)
		}
	}
}
