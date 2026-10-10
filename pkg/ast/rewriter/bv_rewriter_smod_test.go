// Source: src/ast/rewriter/bv_rewriter.cpp:1282-1341; src/ast/rewriter/bv_rewriter.h:125,130.
package rewriter

import (
	"fmt"
	"github.com/glycerine/g3/pkg/ast"
	"testing"
)

func TestBVRewriterSignedModuloOriginalBranches(t *testing.T) {
	c, g, _ := bvPolynomialParamsFixture(t)
	m, _ := bvAddFixture(t)
	local, e := c.NewValue()
	astTestMust(t, e)
	defer func() { astTestMust(t, c.Del(&local)) }()
	p, e := NewBVRewriter(m, c, g, &local)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	out, e := m.NewRef(ast.ASTHandle{})
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	xs := bvMergeVars(t, m, 2)
	x, y := xs[0], xs[1]
	for _, tc := range []struct{ a, b, want uint32 }{{7, 3, 1}, {249, 3, 2}, {7, 253, 254}, {249, 253, 255}, {6, 3, 0}, {128, 255, 0}, {7, 1, 0}, {7, 0, 7}} {
		st, e := p.RewriteSignedModulo(bvAddNum(t, m, tc.a, 8), bvAddNum(t, m, tc.b, 8), out)
		astTestMust(t, e)
		got, e := m.RefGet(out)
		astTestMust(t, e)
		if st != RewriteDone || got != bvAddNum(t, m, tc.want, 8) {
			t.Fatal(tc, st, got)
		}
	}
	for _, hi := range []bool{true, false} {
		key := "hi_div0"
		astTestMust(t, c.SetBoolName(&local, &key, hi))
		astTestMust(t, p.Update(&local))
		zero := bvAddNum(t, m, 0, 8)
		special, e := m.MkFamilyApp(p.bvFamily, 16, nil, []ast.ASTHandle{x})
		astTestMust(t, e)
		astTestPin(t, m, special)
		st, e := p.RewriteSignedModulo(x, zero, out)
		astTestMust(t, e)
		got, e := m.RefGet(out)
		astTestMust(t, e)
		want := x
		if !hi {
			want = special
		}
		if st != RewriteDone || got != want {
			t.Fatal("zero", hi, st, got)
		}
		st, e = p.RewriteSignedModulo(x, bvAddNum(t, m, 1, 8), out)
		astTestMust(t, e)
		got, e = m.RefGet(out)
		astTestMust(t, e)
		if st != Rewrite2 || got != zero {
			t.Fatal("symbolic positive-one status", st, got)
		}
		st, e = p.RewriteSignedModulo(zero, y, out)
		astTestMust(t, e)
		got, e = m.RefGet(out)
		astTestMust(t, e)
		want, e = m.MkFamilyApp(p.bvFamily, 10, nil, []ast.ASTHandle{zero, y})
		astTestMust(t, e)
		astTestPin(t, m, want)
		if st != Rewrite1 || got != want {
			t.Fatal("lhs zero delegates raw UREM", hi, st, got)
		}
		st, e = p.RewriteSignedModulo(x, y, out)
		astTestMust(t, e)
		got, e = m.RefGet(out)
		astTestMust(t, e)
		want, e = m.MkFamilyApp(p.bvFamily, 21, nil, []ast.ASTHandle{x, y})
		astTestMust(t, e)
		astTestPin(t, m, want)
		status := RewriteDone
		if !hi {
			eq, e := m.MkEq(y, zero)
			astTestMust(t, e)
			astTestPin(t, m, eq)
			want, e = m.MkIte(eq, special, want)
			astTestMust(t, e)
			astTestPin(t, m, want)
			status = Rewrite2
		}
		if st != status || got != want {
			t.Fatal("symbolic", hi, st, got)
		}
		st, e = p.RewriteSignedModuloInternal(x, zero, out)
		astTestMust(t, e)
		got, e = m.RefGet(out)
		astTestMust(t, e)
		if st != RewriteDone || got != x {
			t.Fatal("internal zero", st, got)
		}
	}
	st, e := p.RewriteSignedModulo(bvAddNum(t, m, 249, 8), bvAddNum(t, m, 13, 4), out)
	astTestMust(t, e)
	got, e := m.RefGet(out)
	astTestMust(t, e)
	if st != RewriteDone || got != bvAddNum(t, m, 15, 4) {
		t.Fatal("stored RHS width", st, got)
	}

}

func TestBVRewriterSignedModuloEveryColdReservationAndPublication(t *testing.T) {
	for _, hi := range []bool{true, false} {
		for _, mode := range []string{"positive", "negative-positive", "positive-negative", "negative-negative", "exact", "lhs-zero", "zero", "one", "symbolic"} {
			complete, published := false, false
			for ordinal := int64(0); ordinal < 1024 && !complete; ordinal++ {
				t.Run(fmt.Sprintf("%t/%s/%d", hi, mode, ordinal), func(t *testing.T) {
					c, g, _ := bvPolynomialParamsFixture(t)
					m, _ := bvAddFixture(t)
					local, e := c.NewValue()
					astTestMust(t, e)
					key := "hi_div0"
					astTestMust(t, c.SetBoolName(&local, &key, hi))
					p, e := NewBVRewriter(m, c, g, &local)
					astTestMust(t, e)
					astTestMust(t, c.Del(&local))
					xs := bvMergeVars(t, m, 2)
					lhs, rhs := bvAddNum(t, m, 249, 8), bvAddNum(t, m, 3, 8)
					switch mode {
					case "negative-positive":
						lhs = bvAddNum(t, m, 249, 8)
					case "positive-negative":
						rhs = bvAddNum(t, m, 253, 8)
					case "negative-negative":
						lhs = bvAddNum(t, m, 249, 8)
						rhs = bvAddNum(t, m, 253, 8)
					case "exact":
						lhs = bvAddNum(t, m, 6, 8)
					case "lhs-zero":
						lhs = bvAddNum(t, m, 0, 8)
						rhs = xs[0]
					case "zero":
						lhs = xs[0]
						rhs = bvAddNum(t, m, 0, 8)
					case "one":
						lhs = xs[0]
						rhs = bvAddNum(t, m, 1, 8)
					case "symbolic":
						lhs = xs[0]
						rhs = xs[1]
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
					st, body := p.RewriteSignedModulo(lhs, rhs, out)
					m.Budget().FailAfter(-1)
					if body == nil {
						complete = true
						nativeResult, nativeErr := m.RefGet(out)
						astTestMust(t, nativeErr)
						if nativeResult != old {
							published = true
						}
						want := RewriteDone
						if mode == "lhs-zero" {
							want = Rewrite1
						}
						if mode == "one" || mode == "symbolic" && !hi {
							want = Rewrite2
						}
						if st != want {
							t.Fatal("source branch status", hi, mode, st, want)
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
						if _, e = p.RewriteSignedModulo(lhs, rhs, out); e != body {
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
			if !complete || !published {
				t.Fatal("incomplete signed modulo campaign", hi, mode, complete, published)
			}
		}
	}
}
