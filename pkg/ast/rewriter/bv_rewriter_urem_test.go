package rewriter

// Source: src/ast/rewriter/bv_rewriter.cpp:1185-1280; src/ast/rewriter/bv_rewriter.h:124,129.

import (
	"fmt"
	"github.com/glycerine/g3/pkg/ast"
	"testing"
)

func TestBVRewriterUnsignedRemainderOriginalBranches(t *testing.T) {
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
	zero, one, minus := bvAddNum(t, m, 0, 8), bvAddNum(t, m, 1, 8), bvAddNum(t, m, 255, 8)
	sum, e := m.MkBitVectorAdd(minus, x)
	astTestMust(t, e)
	astTestPin(t, m, sum)
	for _, hi := range []bool{true, false} {
		key := "hi_div0"
		astTestMust(t, c.SetBoolName(&local, &key, hi))
		astTestMust(t, p.Update(&local))
		special := func(a ast.ASTHandle) ast.ASTHandle {
			h, e := m.MkFamilyApp(p.bvFamily, 15, nil, []ast.ASTHandle{a})
			astTestMust(t, e)
			astTestPin(t, m, h)
			return h
		}
		internal := func(a, b ast.ASTHandle) ast.ASTHandle {
			h, e := m.MkFamilyApp(p.bvFamily, 20, nil, []ast.ASTHandle{a, b})
			astTestMust(t, e)
			astTestPin(t, m, h)
			return h
		}
		conditional := func(a, b, yes, no ast.ASTHandle) ast.ASTHandle {
			eq, e := m.MkEq(a, b)
			astTestMust(t, e)
			astTestPin(t, m, eq)
			h, e := m.MkIte(eq, yes, no)
			astTestMust(t, e)
			astTestPin(t, m, h)
			return h
		}
		powerExtract, e := m.MkBitVectorExtract(1, 0, x)
		astTestMust(t, e)
		astTestPin(t, m, powerExtract)
		power, e := m.MkBitVectorConcat([]ast.ASTHandle{bvAddNum(t, m, 0, 6), powerExtract})
		astTestMust(t, e)
		astTestPin(t, m, power)
		cases := []struct {
			a, b, want ast.ASTHandle
			status     RewriteStatus
		}{
			{bvAddNum(t, m, 249, 8), bvAddNum(t, m, 3, 8), zero, RewriteDone},
			{bvAddNum(t, m, 255, 8), bvAddNum(t, m, 6, 8), bvAddNum(t, m, 3, 8), RewriteDone},
			{x, one, zero, RewriteDone},
			{x, bvAddNum(t, m, 4, 8), power, Rewrite2},
			{x, bvAddNum(t, m, 3, 8), internal(x, bvAddNum(t, m, 3, 8)), RewriteDone},
		}
		wz, sz := x, RewriteDone
		if !hi {
			wz = special(x)
			sz = Rewrite1
		}
		cases = append(cases, struct {
			a, b, want ast.ASTHandle
			status     RewriteStatus
		}{x, zero, wz, sz})
		w, s := zero, RewriteDone
		if !hi {
			w = conditional(x, zero, special(zero), zero)
			s = Rewrite2
		}
		cases = append(cases, struct {
			a, b, want ast.ASTHandle
			status     RewriteStatus
		}{zero, x, w, s})
		w, s = sum, RewriteDone
		if !hi {
			w = conditional(x, zero, special(minus), sum)
			s = Rewrite2
		}
		cases = append(cases, struct {
			a, b, want ast.ASTHandle
			status     RewriteStatus
		}{sum, x, w, s})
		w, s = internal(x, y), RewriteDone
		if !hi {
			w = conditional(y, zero, special(x), w)
			s = Rewrite2
		}
		cases = append(cases, struct {
			a, b, want ast.ASTHandle
			status     RewriteStatus
		}{x, y, w, s})
		for _, tc := range cases {
			st, e := p.RewriteUnsignedRemainder(tc.a, tc.b, out)
			astTestMust(t, e)
			got, e := m.RefGet(out)
			astTestMust(t, e)
			if st != tc.status || got != tc.want {
				t.Fatal("branch", hi, st, tc.status, got, tc.want)
			}
		}
		st, e := p.RewriteUnsignedRemainderInternal(x, zero, out)
		astTestMust(t, e)
		got, e := m.RefGet(out)
		astTestMust(t, e)
		if st != RewriteDone || got != x {
			t.Fatal("forced hardware", st, got)
		}
	}
	// RHS read overwrites the width before lhs normalization (unlike UDIV).
	st, e := p.RewriteUnsignedRemainder(bvAddNum(t, m, 249, 8), bvAddNum(t, m, 13, 4), out)
	astTestMust(t, e)
	got, e := m.RefGet(out)
	astTestMust(t, e)
	if st != RewriteDone || got != bvAddNum(t, m, 9, 4) {
		t.Fatal("RHS width normalization", st, got)
	}
}

func TestBVRewriterUnsignedRemainderEveryColdReservationAndPublication(t *testing.T) {
	for _, hi := range []bool{true, false} {
		for _, mode := range []string{"numeric", "zero", "one", "power", "even", "symbolic", "lhs-zero", "minus", "minus-second", "minus-other"} {
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
					case "power":
						lhs = xs[0]
						rhs = bvAddNum(t, m, 2, 8)
					case "even":
						lhs = xs[0]
						rhs = bvAddNum(t, m, 6, 8)
					case "lhs-zero":
						lhs = bvAddNum(t, m, 0, 8)
						rhs = xs[0]
					case "minus", "minus-second", "minus-other":
						lhs, e = m.MkBitVectorAdd(bvAddNum(t, m, 255, 8), xs[0])
						astTestMust(t, e)
						astTestPin(t, m, lhs)
						rhs = xs[0]
						if mode == "minus-second" {
							lhs, e = m.MkBitVectorAdd(xs[0], bvAddNum(t, m, 255, 8))
							astTestMust(t, e)
							astTestPin(t, m, lhs)
						}
						if mode == "minus-other" {
							rhs = xs[1]
						}
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
					st, body := p.RewriteUnsignedRemainder(lhs, rhs, out)
					m.Budget().FailAfter(-1)
					if body == nil {
						complete = true
						nativeResult, nativeErr := m.RefGet(out)
						astTestMust(t, nativeErr)
						if nativeResult != old {
							published = true
						}
						want := RewriteDone
						if mode == "zero" && !hi {
							want = Rewrite1
						}
						if mode == "power" || !hi && (mode == "symbolic" || mode == "lhs-zero" || mode == "minus" || mode == "minus-second" || mode == "minus-other") {
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
						if _, e = p.RewriteUnsignedRemainder(lhs, rhs, out); e != body {
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
				t.Fatal("incomplete unsigned remainder campaign", hi, mode, complete, published)
			}
		}
	}
}
