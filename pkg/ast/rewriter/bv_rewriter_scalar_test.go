package rewriter

// Source: src/ast/rewriter/bv_rewriter.cpp:2040-2074.

import (
	"fmt"
	"github.com/glycerine/g3/pkg/ast"
	"testing"
)

func TestBVRewriterScalarOperatorsOnDerivedOwner(t *testing.T) {
	c, g, _ := bvPolynomialParamsFixture(t)
	m, _ := bvAddFixture(t)
	p, e := NewBVRewriter(m, c, g, nil)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	xs := bvMergeVars(t, m, 2)
	out, e := m.NewRef(xs[0])
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	for _, and := range []bool{false, true} {
		for _, tc := range []struct{ input, want uint32 }{{0, 0}, {1, 1}, {254, 1}, {255, 1}} {
			want := tc.want
			if and && tc.input != 255 {
				want = 0
			}
			var st RewriteStatus
			if and {
				st, e = p.RewriteReductionAND(bvAddNum(t, m, tc.input, 8), out)
			} else {
				st, e = p.RewriteReductionOR(bvAddNum(t, m, tc.input, 8), out)
			}
			astTestMust(t, e)
			got, e := m.RefGet(out)
			astTestMust(t, e)
			if st != RewriteDone || got != bvAddNum(t, m, want, 1) {
				t.Fatal(and, tc, st, got)
			}
		}
		astTestMust(t, m.SetRef(out, xs[0]))
		var st RewriteStatus
		if and {
			st, e = p.RewriteReductionAND(xs[1], out)
		} else {
			st, e = p.RewriteReductionOR(xs[1], out)
		}
		astTestMust(t, e)
		got, e := m.RefGet(out)
		astTestMust(t, e)
		if st != RewriteFailed || got != xs[0] {
			t.Fatal("FAILED output", and, st, got)
		}
	}
	for _, tc := range []struct {
		a, b ast.ASTHandle
		want uint32
	}{{xs[0], xs[0], 1}, {bvAddNum(t, m, 1, 8), bvAddNum(t, m, 2, 8), 0}} {
		st, e := p.RewriteCompare(tc.a, tc.b, out)
		astTestMust(t, e)
		got, e := m.RefGet(out)
		astTestMust(t, e)
		if st != RewriteDone || got != bvAddNum(t, m, tc.want, 1) {
			t.Fatal("compare", st, got)
		}
	}
	st, e := p.RewriteCompare(xs[0], xs[1], out)
	astTestMust(t, e)
	eq, e := m.MkEq(xs[0], xs[1])
	astTestMust(t, e)
	astTestPin(t, m, eq)
	want, e := m.MkIte(eq, bvAddNum(t, m, 1, 1), bvAddNum(t, m, 0, 1))
	astTestMust(t, e)
	astTestPin(t, m, want)
	got, e := m.RefGet(out)
	astTestMust(t, e)
	if st != Rewrite2 || got != want {
		t.Fatal("compare ITE", st, got, want)
	}
}

func TestBVRewriterScalarEveryColdReservationAndPublication(t *testing.T) {
	for _, mode := range []string{"or-zero", "or-nonzero", "or-failed", "and-all", "and-other", "and-failed", "compare-same", "compare-numerals", "compare-symbolic"} {
		complete, published := false, false
		for ordinal := int64(0); ordinal < 512 && !complete; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", mode, ordinal), func(t *testing.T) {
				c, g, _ := bvPolynomialParamsFixture(t)
				m, _ := bvAddFixture(t)
				p, e := NewBVRewriter(m, c, g, nil)
				astTestMust(t, e)
				xs := bvMergeVars(t, m, 2)
				num0, num1, num254, num255 := bvAddNum(t, m, 0, 8), bvAddNum(t, m, 1, 8), bvAddNum(t, m, 254, 8), bvAddNum(t, m, 255, 8)
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
				invoke := func() (RewriteStatus, error) {
					switch mode {
					case "or-zero":
						return p.RewriteReductionOR(num0, out)
					case "or-nonzero":
						return p.RewriteReductionOR(num1, out)
					case "or-failed":
						return p.RewriteReductionOR(xs[0], out)
					case "and-all":
						return p.RewriteReductionAND(num255, out)
					case "and-other":
						return p.RewriteReductionAND(num254, out)
					case "and-failed":
						return p.RewriteReductionAND(xs[0], out)
					case "compare-same":
						return p.RewriteCompare(xs[0], xs[0], out)
					case "compare-numerals":
						return p.RewriteCompare(num0, num1, out)
					default:
						return p.RewriteCompare(xs[0], xs[1], out)
					}
				}
				m.Budget().FailAfter(ordinal)
				st, body := invoke()
				m.Budget().FailAfter(-1)
				failed := mode == "or-failed" || mode == "and-failed"
				if body == nil {
					complete = true
					nativeResult, nativeErr := m.RefGet(out)
					astTestMust(t, nativeErr)
					if nativeResult != old {
						published = true
					}
					want := RewriteDone
					if failed {
						want = RewriteFailed
					}
					if mode == "compare-symbolic" {
						want = Rewrite2
					}
					if st != want {
						t.Fatal("status", mode, st, want)
					}
					got, e := m.RefGet(out)
					astTestMust(t, e)
					if failed && got != old {
						t.Fatal("FAILED changed output")
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
							t.Fatal("published status")
						}
					}
					if _, e = invoke(); e != body {
						t.Fatal("stopped replay", e, body)
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
		if !complete || !(mode == "or-failed" || mode == "and-failed") && !published {
			t.Fatal("cold campaign incomplete", mode, complete, published)
		}
	}
}
