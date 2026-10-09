// Source: src/ast/rewriter/bv_rewriter.cpp:1451-1510.
package rewriter

import (
	"fmt"
	"github.com/glycerine/g3/pkg/ast"
	"testing"
)

func TestBVRewriterConcatOriginalBranchesAndStatuses(t *testing.T) {
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
	concat := func(hs ...ast.ASTHandle) ast.ASTHandle {
		h, e := m.MkBitVectorConcat(hs)
		astTestMust(t, e)
		astTestPin(t, m, h)
		return h
	}
	a, b, d := bvAddNum(t, m, 1, 2), bvAddNum(t, m, 2, 2), bvAddNum(t, m, 3, 2)
	cases := []struct {
		name   string
		args   []ast.ASTHandle
		want   ast.ASTHandle
		status RewriteStatus
	}{
		{"empty", nil, ast.ASTHandle{}, RewriteFailed},
		{"numerals", []ast.ASTHandle{a, b, d}, bvAddNum(t, m, 27, 6), RewriteDone},
		{"extract-single", []ast.ASTHandle{ex(7, 4), ex(3, 0)}, ex(7, 0), Rewrite1},
		{"extract-many", []ast.ASTHandle{ex(7, 4), ex(3, 2), ex(0, 0)}, concat(ex(7, 2), ex(0, 0)), Rewrite2},
		{"flatten-one-level", []ast.ASTHandle{concat(a, concat(b, d)), x}, concat(a, concat(b, d), x), Rewrite1},
		{"flatten-then-boundary-numeral-fusion", []ast.ASTHandle{concat(a, b), d}, concat(a, bvAddNum(t, m, 11, 4)), Rewrite1},
		{"flatten-no-extra-fusion", []ast.ASTHandle{concat(a, b), x}, concat(a, b, x), Rewrite1},
		{"extract-precedes-expanded", []ast.ASTHandle{concat(ex(7, 4)), ex(3, 0)}, ex(7, 0), Rewrite1},
		{"extract-precedes-expanded-many", []ast.ASTHandle{concat(ex(7, 4)), ex(3, 0), a}, concat(ex(7, 0), a), Rewrite2},
		{"failed", []ast.ASTHandle{x, a}, ast.ASTHandle{}, RewriteFailed},
		{"nonadjacent", []ast.ASTHandle{ex(7, 5), ex(3, 0)}, ast.ASTHandle{}, RewriteFailed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before, e := m.RefGet(out)
			astTestMust(t, e)
			st, e := p.RewriteConcat(ast.SliceArguments(tc.args), out)
			astTestMust(t, e)
			got, e := m.RefGet(out)
			astTestMust(t, e)
			want := tc.want
			if st == RewriteFailed {
				want = before
			}
			if st != tc.status || got != want {
				t.Fatal(st, tc.status, got, want)
			}
		})
	}
	// A disabled flat option leaves nested concat untouched.
	local, e := c.NewValue()
	astTestMust(t, e)
	defer func() { astTestMust(t, c.Del(&local)) }()
	key := "flat"
	astTestMust(t, c.SetBoolName(&local, &key, false))
	astTestMust(t, p.Update(&local))
	st, e := p.RewriteConcat(ast.SliceArguments([]ast.ASTHandle{concat(a, b), x}), out)
	astTestMust(t, e)
	if st != RewriteFailed {
		t.Fatal("flat disabled", st)
	}
}

func TestBVRewriterConcatEveryColdReservationAndPublication(t *testing.T) {
	for _, mode := range []string{"numeral", "extract", "flat", "failed"} {
		complete, published := false, false
		for ordinal := int64(0); ordinal < 1024 && !complete; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", mode, ordinal), func(t *testing.T) {
				c, g, _ := bvPolynomialParamsFixture(t)
				m, _ := bvAddFixture(t)
				p, e := NewBVRewriter(m, c, g, nil)
				astTestMust(t, e)
				x := bvMergeVars(t, m, 1)[0]
				var args []ast.ASTHandle
				switch mode {
				case "numeral":
					args = []ast.ASTHandle{bvAddNum(t, m, 3, 2), bvAddNum(t, m, 2, 2), bvAddNum(t, m, 1, 2)}
				case "extract":
					hi, e := m.MkBitVectorExtract(7, 4, x)
					astTestMust(t, e)
					astTestPin(t, m, hi)
					lo, e := m.MkBitVectorExtract(3, 0, x)
					astTestMust(t, e)
					astTestPin(t, m, lo)
					args = []ast.ASTHandle{hi, lo}
				case "flat":
					inner := make([]ast.ASTHandle, 40)
					for i := range inner {
						inner[i] = x
					}
					h, e := m.MkBitVectorConcat(inner)
					astTestMust(t, e)
					astTestPin(t, m, h)
					args = []ast.ASTHandle{h, x}
				case "failed":
					args = make([]ast.ASTHandle, 40)
					for i := range args {
						args[i] = x
					}
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
				m.Budget().FailAfter(ordinal)
				st, body := p.RewriteConcat(ast.SliceArguments(args), out)
				m.Budget().FailAfter(-1)
				if body == nil {
					complete = true
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
					if _, e = p.RewriteConcat(ast.AppArguments{}, out); e != body {
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
			t.Fatal("incomplete campaign", mode, complete, published)
		}
	}
}

func TestBVConcatRefBufferPushPopAndForwardDestruction(t *testing.T) {
	m, _ := bvAddFixture(t)
	x := bvMergeVars(t, m, 1)[0]
	before, e := m.GetRefCount(x)
	astTestMust(t, e)
	var b bvConcatRefBuffer
	for i := 0; i < 16; i++ {
		astTestMust(t, b.push(m, x))
	}
	m.Budget().FailAfter(0)
	body := b.push(m, x)
	m.Budget().FailAfter(-1)
	rc, e := m.GetRefCount(x)
	astTestMust(t, e)
	if body != nil || len(b.values.Slice()) != 17 || !b.pending.IsNull() || rc != before+17 {
		t.Fatal("native buffer push", body, rc)
	}
	astTestMust(t, b.close(m))
	rc, e = m.GetRefCount(x)
	astTestMust(t, e)
	if rc != before {
		t.Fatal("pending/spilled ownership", rc)
	}
	// An unpinned 32-child node forces a real consumed decrement refusal.
	makeOwned := func() ast.ASTHandle {
		sort, e := m.BitVectorSort(8)
		astTestMust(t, e)
		hs := make([]ast.ASTHandle, 32)
		for i := range hs {
			hs[i], e = m.MkVar(uint32(100+i), sort)
			astTestMust(t, e)
		}
		h, e := m.MkBitVectorOr(hs)
		astTestMust(t, e)
		return h
	}
	h := makeOwned()
	astTestMust(t, b.push(m, h))
	m.Budget().FailAfter(0)
	body = b.pop(m)
	m.Budget().FailAfter(-1)
	if body == nil || len(b.values.Slice()) != 0 || !b.pending.IsNull() {
		t.Fatal("pop before consumed DEC", body)
	}
	astTestMust(t, b.close(m))
	h = makeOwned()
	astTestMust(t, b.push(m, h))
	astTestMust(t, b.push(m, x))
	m.Budget().FailAfter(0)
	body = b.close(m)
	m.Budget().FailAfter(-1)
	rc, e = m.GetRefCount(x)
	astTestMust(t, e)
	hs := b.values.Slice()
	if body == nil || len(hs) != 2 || !hs[0].IsNull() || hs[1] != x || rc != before+1 {
		t.Fatal("forward DT consumed prefix", body, rc)
	}
	astTestMust(t, b.close(m))
	astTestMust(t, b.close(m))
	rc, e = m.GetRefCount(x)
	astTestMust(t, e)
	if rc != before {
		t.Fatal("DEC replay", rc)
	}
}
