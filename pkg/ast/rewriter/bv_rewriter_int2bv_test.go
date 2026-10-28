package rewriter

// Source: src/ast/rewriter/bv_rewriter.cpp:1343-1361.

import (
	"fmt"
	"github.com/glycerine/g3/pkg/ast"
	"testing"
)

func TestBVRewriterInt2BVOriginalBranches(t *testing.T) {
	m, qc, _, _ := boolRewriterConfigFixture(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	c, g, _ := bvPolynomialParamsFixture(t)
	p, e := NewBVRewriter(m, c, g, nil)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	xs := bvMergeVars(t, m, 2)
	out, e := m.NewRef(xs[0])
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	for _, integer := range []bool{true, false} {
		for _, tc := range []struct {
			text        string
			width, want uint32
		}{{"-1", 8, 255}, {"257", 8, 1}, {"-257", 8, 255}, {"65539", 4, 3}} {
			v, e := qc.NumeralsField.NewValue()
			astTestMust(t, e)
			astTestMust(t, qc.NumeralsField.SetString(&v, tc.text))
			h, e := m.ArithmeticNumeral(&v, integer)
			astTestMust(t, e)
			astTestPin(t, m, h)
			astTestMust(t, qc.NumeralsField.Del(&v))
			st, e := p.RewriteInt2BV(tc.width, h, out)
			astTestMust(t, e)
			got, e := m.RefGet(out)
			astTestMust(t, e)
			if st != RewriteDone || got != bvAddNum(t, m, tc.want, tc.width) {
				t.Fatal(tc, integer, st, got)
			}
		}
	}
	inner, e := m.MkFamilyApp(p.bvFamily, 58, nil, []ast.ASTHandle{xs[0]})
	astTestMust(t, e)
	astTestPin(t, m, inner)
	for _, width := range []uint32{8, 7} {
		astTestMust(t, m.SetRef(out, xs[1]))
		st, e := p.RewriteInt2BV(width, inner, out)
		astTestMust(t, e)
		got, e := m.RefGet(out)
		astTestMust(t, e)
		if width == 8 {
			if st != RewriteDone || got != xs[0] {
				t.Fatal("reverse conversion", st, got)
			}
		} else if st != RewriteFailed || got != xs[1] {
			t.Fatal("width guard", st, got)
		}
	}
	astTestMust(t, m.SetRef(out, xs[1]))
	st, e := p.RewriteInt2BV(8, xs[0], out)
	astTestMust(t, e)
	got, e := m.RefGet(out)
	astTestMust(t, e)
	if st != RewriteFailed || got != xs[1] {
		t.Fatal("FAILED output", st, got)
	}
}

func TestBVRewriterInt2BVEveryColdReservationAndPublication(t *testing.T) {
	for _, mode := range []string{"numeric", "reverse", "width", "failed"} {
		complete, published := false, false
		for ordinal := int64(0); ordinal < 512 && !complete; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", mode, ordinal), func(t *testing.T) {
				m, qc, _, _ := boolRewriterConfigFixture(t)
				_, e := m.RegisterArithmeticPlugin()
				astTestMust(t, e)
				_, e = m.RegisterBitVectorPlugin()
				astTestMust(t, e)
				c, g, _ := bvPolynomialParamsFixture(t)
				p, e := NewBVRewriter(m, c, g, nil)
				astTestMust(t, e)
				xs := bvMergeVars(t, m, 2)
				arg, width := xs[0], uint32(8)
				if mode == "numeric" {
					v, e := qc.NumeralsField.NewValue()
					astTestMust(t, e)
					astTestMust(t, qc.NumeralsField.SetInt64(&v, -1))
					arg, e = m.ArithmeticNumeral(&v, true)
					astTestMust(t, e)
					astTestPin(t, m, arg)
					astTestMust(t, qc.NumeralsField.Del(&v))
					width = 65
				} else if mode == "reverse" || mode == "width" {
					arg, e = m.MkFamilyApp(p.bvFamily, 58, nil, []ast.ASTHandle{xs[0]})
					astTestMust(t, e)
					astTestPin(t, m, arg)
					if mode == "width" {
						width = 7
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
				st, body := p.RewriteInt2BV(width, arg, out)
				m.Budget().FailAfter(-1)
				if body == nil {
					complete = true
					nativeResult, nativeErr := m.RefGet(out)
					astTestMust(t, nativeErr)
					if nativeResult != old {
						published = true
					}
					want := RewriteDone
					if mode == "failed" || mode == "width" {
						want = RewriteFailed
					}
					if st != want {
						t.Fatal("status", mode, st, want)
					}
					got, e := m.RefGet(out)
					astTestMust(t, e)
					if want == RewriteFailed && got != old {
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
					if _, e = p.RewriteInt2BV(width, arg, out); e != body {
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
		if !complete || (mode == "numeric" || mode == "reverse") && !published {
			t.Fatal("cold campaign incomplete", mode, complete, published)
		}
	}
}
