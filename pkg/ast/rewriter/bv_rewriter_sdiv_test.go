package rewriter

// Source: src/ast/rewriter/bv_rewriter.cpp:987-1032; src/ast/rewriter/bv_rewriter.h:121,126.

import (
	"fmt"
	"github.com/glycerine/g3/pkg/ast"
	"testing"
)

func TestBVRewriterSignedDivisionOriginalBranches(t *testing.T) {
	c, g, _ := bvPolynomialParamsFixture(t)
	m, _ := bvAddFixture(t)
	p, e := NewBVRewriter(m, c, g, nil)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	out, e := m.NewRef(ast.ASTHandle{})
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	xs := bvMergeVars(t, m, 2)
	x, y := xs[0], xs[1]
	for _, tc := range []struct{ a, b, want uint32 }{{7, 3, 2}, {249, 3, 254}, {7, 253, 254}, {249, 253, 2}, {128, 255, 128}, {249, 1, 249}} {
		st, e := p.RewriteSignedDivision(bvAddNum(t, m, tc.a, 8), bvAddNum(t, m, tc.b, 8), out)
		astTestMust(t, e)
		got, e := m.RefGet(out)
		astTestMust(t, e)
		if st != RewriteDone || got != bvAddNum(t, m, tc.want, 8) {
			t.Fatal(tc, st, got)
		}
	}
	zero := bvAddNum(t, m, 0, 8)
	cond, e := m.MkFamilyApp(p.bvFamily, 27, nil, []ast.ASTHandle{x, zero})
	astTestMust(t, e)
	astTestPin(t, m, cond)
	hardware, e := m.MkIte(cond, bvAddNum(t, m, 1, 8), bvAddNum(t, m, 255, 8))
	astTestMust(t, e)
	astTestPin(t, m, hardware)
	for _, hi := range []bool{true, false} {
		local, e := c.NewValue()
		astTestMust(t, e)
		key := "hi_div0"
		astTestMust(t, c.SetBoolName(&local, &key, hi))
		astTestMust(t, p.Update(&local))
		astTestMust(t, c.Del(&local))
		special, e := m.MkFamilyApp(p.bvFamily, 12, nil, []ast.ASTHandle{x})
		astTestMust(t, e)
		astTestPin(t, m, special)
		st, e := p.RewriteSignedDivision(x, zero, out)
		astTestMust(t, e)
		want, status := hardware, Rewrite2
		if !hi {
			want = special
			status = Rewrite1
		}
		got, e := m.RefGet(out)
		astTestMust(t, e)
		if st != status || got != want {
			t.Fatal("zero", hi, st, got, want)
		}
		st, e = p.RewriteSignedDivision(x, y, out)
		astTestMust(t, e)
		internal, e := m.MkFamilyApp(p.bvFamily, 17, nil, []ast.ASTHandle{x, y})
		astTestMust(t, e)
		astTestPin(t, m, internal)
		want, status = internal, RewriteDone
		if !hi {
			eq, e := m.MkEq(y, zero)
			astTestMust(t, e)
			astTestPin(t, m, eq)
			want, e = m.MkIte(eq, special, internal)
			astTestMust(t, e)
			astTestPin(t, m, want)
			status = Rewrite2
		}
		got, e = m.RefGet(out)
		astTestMust(t, e)
		if st != status || got != want {
			t.Fatal("symbolic", hi, st, got, want)
		}
		st, e = p.RewriteSignedDivisionInternal(x, zero, out)
		astTestMust(t, e)
		got, e = m.RefGet(out)
		astTestMust(t, e)
		if st != Rewrite2 || got != hardware {
			t.Fatal("internal hardware zero", hi, st, got)
		}
	}
}

func TestBVRewriterSignedDivisionEveryColdReservationAndPublication(t *testing.T) {
	for _, hi := range []bool{true, false} {
		for _, mode := range []string{"numeric", "zero", "one", "negative-one", "fallback", "symbolic"} {
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
					case "zero":
						lhs = xs[0]
						rhs = bvAddNum(t, m, 0, 8)
					case "one":
						lhs = xs[0]
						rhs = bvAddNum(t, m, 1, 8)
					case "negative-one":
						lhs = xs[0]
						rhs = bvAddNum(t, m, 255, 8)
					case "fallback":
						lhs = xs[0]
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
					st, body := p.RewriteSignedDivision(lhs, rhs, out)
					m.Budget().FailAfter(-1)
					if body == nil {
						complete = true
						nativeResult, nativeErr := m.RefGet(out)
						astTestMust(t, nativeErr)
						if nativeResult != old {
							published = true
						}
						if st == RewriteFailed {
							t.Fatal("source must produce result", hi, mode)
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
						if _, e = p.RewriteSignedDivision(lhs, rhs, out); e != body {
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
				t.Fatal("incomplete signed remainder campaign", hi, mode, complete, published)
			}
		}
	}
}

func TestBVRewriterSignedDivisionWidthsAndHardwareNumeral(t *testing.T) {
	c, g, _ := bvPolynomialParamsFixture(t)
	m, _ := bvAddFixture(t)
	p, e := NewBVRewriter(m, c, g, nil)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	out, e := m.NewRef(ast.ASTHandle{})
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	st, e := p.RewriteSignedDivision(bvAddNum(t, m, 249, 8), bvAddNum(t, m, 13, 4), out)
	astTestMust(t, e)
	got, e := m.RefGet(out)
	astTestMust(t, e)
	if st != RewriteDone || got != bvAddNum(t, m, 2, 8) {
		t.Fatal("independent signed widths", st, got)
	}
	lhs, zero := bvAddNum(t, m, 249, 8), bvAddNum(t, m, 0, 8)
	st, e = p.RewriteSignedDivision(lhs, zero, out)
	astTestMust(t, e)
	cond, e := m.MkFamilyApp(p.bvFamily, 27, nil, []ast.ASTHandle{lhs, zero})
	astTestMust(t, e)
	astTestPin(t, m, cond)
	want, e := m.MkIte(cond, bvAddNum(t, m, 1, 8), bvAddNum(t, m, 255, 8))
	astTestMust(t, e)
	astTestPin(t, m, want)
	got, e = m.RefGet(out)
	astTestMust(t, e)
	if st != Rewrite2 || got != want {
		t.Fatal("hardware zero remains original raw ITE", st, got, want)
	}
}
