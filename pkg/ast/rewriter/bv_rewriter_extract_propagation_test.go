// Source: src/ast/rewriter/bv_rewriter.cpp:590-790.
package rewriter

import (
	"fmt"
	"github.com/glycerine/g3/pkg/ast"
	"testing"
)

func bvPropagationVar(t *testing.T, m *ast.ASTManager, id, width uint32) ast.ASTHandle {
	t.Helper()
	s, e := m.BitVectorSort(width)
	astTestMust(t, e)
	h, e := m.MkVar(id, s)
	astTestMust(t, e)
	astTestPin(t, m, h)
	return h
}
func bvPropagationConcat(t *testing.T, m *ast.ASTManager, hs ...ast.ASTHandle) ast.ASTHandle {
	t.Helper()
	h, e := m.MkBitVectorConcat(hs)
	astTestMust(t, e)
	astTestPin(t, m, h)
	return h
}
func bvPropagationOp(t *testing.T, m *ast.ASTManager, p *BVRewriter, k int32, hs ...ast.ASTHandle) ast.ASTHandle {
	t.Helper()
	h, e := m.MkFamilyApp(p.bvFamily, k, nil, hs)
	astTestMust(t, e)
	astTestPin(t, m, h)
	return h
}
func TestBVRewriterExtractPropagationOriginalOrderAndShapes(t *testing.T) {
	m, _ := bvAddFixture(t)
	p := bvNotTestOwner(t, m, false, false)
	x4, y4 := bvPropagationVar(t, m, 0, 4), bvPropagationVar(t, m, 1, 4)
	x5 := bvPropagationVar(t, m, 2, 5)
	x6, y6 := bvPropagationVar(t, m, 3, 6), bvPropagationVar(t, m, 4, 6)
	prefix := bvPropagationVar(t, m, 5, 2)
	n4, n3, n2, n1 := bvAddNum(t, m, 15, 4), bvAddNum(t, m, 7, 3), bvAddNum(t, m, 3, 2), bvAddNum(t, m, 1, 1)
	a := bvPropagationConcat(t, m, n4, x4)
	b := bvPropagationConcat(t, m, n4, y4)
	sym := bvPropagationConcat(t, m, prefix, x6)
	num3 := bvPropagationConcat(t, m, n3, x5)
	num2 := bvPropagationConcat(t, m, n2, y6)
	tail := bvPropagationConcat(t, m, x4, y4, x4, y4, x4)
	wide := bvPropagationConcat(t, m, prefix, x4, y4, x4, y4, x4)
	for _, tc := range []struct {
		name      string
		high      uint32
		arg, want ast.ASTHandle
		removed   uint32
	}{
		{"add-shortened", 5, bvPropagationOp(t, m, p, 4, a, b), bvPropagationOp(t, m, p, 4, bvPropagationConcat(t, m, n2, x4), bvPropagationConcat(t, m, n2, y4)), 2},
		{"mul-shortened", 5, bvPropagationOp(t, m, p, 6, a, b), bvPropagationOp(t, m, p, 6, bvPropagationConcat(t, m, n2, x4), bvPropagationConcat(t, m, n2, y4)), 2},
		{"symbol-then-number", 5, bvPropagationOp(t, m, p, 4, sym, num2), bvPropagationOp(t, m, p, 4, x6, y6), 2},
		{"symbol-then-wider-number-fails", 5, bvPropagationOp(t, m, p, 4, sym, num3), ast.ASTHandle{}, 0},
		{"wider-number-then-symbol", 5, bvPropagationOp(t, m, p, 4, num3, sym), bvPropagationOp(t, m, p, 4, bvPropagationConcat(t, m, n1, x5), x6), 2},
		{"borrowed-suffix", 19, bvPropagationOp(t, m, p, 4, wide, wide), bvPropagationOp(t, m, p, 4, tail, tail), 2},
		{"numeric-nonconcat", 5, bvPropagationOp(t, m, p, 4, bvAddNum(t, m, 255, 8), bvAddNum(t, m, 127, 8)), bvPropagationOp(t, m, p, 4, bvAddNum(t, m, 63, 6), bvAddNum(t, m, 63, 6)), 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, e := m.NewRef(x4)
			astTestMust(t, e)
			defer func() { astTestMust(t, m.DelRef(&out)) }()
			removed, e := p.PropagateExtract(tc.high, tc.arg, out)
			astTestMust(t, e)
			got, e := m.RefGet(out)
			astTestMust(t, e)
			want := tc.want
			if removed == 0 {
				want = x4
			}
			if removed != tc.removed || got != want || p.propagation != nil {
				t.Fatal("source propagation", removed, tc.removed, got, want)
			}
		})
	}
	// Early guards must neither construct default val nor inspect output.
	top := bvPropagationOp(t, m, p, 4, a, b)
	m.Budget().FailAfter(0)
	for _, h := range []ast.ASTHandle{x4, top} {
		removed, e := p.PropagateExtract(7, h, ast.ASTRef{})
		astTestMust(t, e)
		if removed != 0 {
			t.Fatal("early guard")
		}
	}
	m.Budget().FailAfter(-1)
}

func TestBVRewriterExtractPropagationEveryColdReservation(t *testing.T) {
	for _, mode := range []string{"shorten-wide", "drop-wide", "order-failed", "parent", "ite-guard"} {
		complete, published := false, false
		for ordinal := int64(0); ordinal < 2048 && !complete; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", mode, ordinal), func(t *testing.T) {
				m, _ := bvAddFixture(t)
				p := bvNotTestOwner(t, m, false, false)
				p.polynomial.config.bitVector.ExtractPropagation = true
				tail := bvMergeVars(t, m, 40)
				prefix := bvAddNum(t, m, 15, 4)
				if mode == "drop-wide" {
					prefix = bvPropagationVar(t, m, 90, 2)
				}
				parts := append([]ast.ASTHandle{prefix}, tail...)
				cat := bvPropagationConcat(t, m, parts...)
				args := make([]ast.ASTHandle, 40)
				for i := range args {
					args[i] = cat
				}
				arg := bvPropagationOp(t, m, p, 4, args...)
				high := uint32(321)
				wantRemoved := uint32(2)
				wantStatus := Rewrite2
				if mode == "drop-wide" {
					high = 319
				}
				if mode == "order-failed" {
					sym := bvPropagationVar(t, m, 91, 2)
					x6 := bvPropagationVar(t, m, 92, 6)
					x5 := bvPropagationVar(t, m, 93, 5)
					arg = bvPropagationOp(t, m, p, 4, bvPropagationConcat(t, m, sym, x6), bvPropagationConcat(t, m, bvAddNum(t, m, 7, 3), x5))
					high = 5
					wantRemoved = 0
				}
				if mode == "ite-guard" {
					c, e := m.MkEq(tail[0], tail[1])
					astTestMust(t, e)
					astTestPin(t, m, c)
					arg, e = m.MkIte(c, tail[0], tail[1])
					astTestMust(t, e)
					astTestPin(t, m, arg)
					high = 6
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
				var removed uint32
				var body error
				st := RewriteFailed
				parent := mode == "parent" || mode == "ite-guard"
				if parent {
					st, body = p.RewriteExtract(high, 1, arg, out)
				} else {
					removed, body = p.PropagateExtract(high, arg, out)
				}
				m.Budget().FailAfter(-1)
				got, e := m.RefGet(out)
				astTestMust(t, e)
				if body == nil {
					complete = true
					if p.propagation != nil || p.extraction != nil || parent && st != wantStatus || !parent && removed != wantRemoved || mode == "order-failed" && got != old {
						t.Fatal("source return", removed, st)
					}
				} else {
					if !numericHoistRefusal(body) {
						t.Fatal(body)
					}
					if got != old {
						published = true
					}
					var replay error
					if parent {
						_, replay = p.RewriteExtract(high, 1, arg, out)
					} else {
						_, replay = p.PropagateExtract(high, arg, out)
					}
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
		if !complete || mode != "order-failed" && !published {
			t.Fatal("campaign incomplete", mode, complete, published)
		}
	}
}

func TestBVRewriterExtractPropagationConfigurationAndIdentity(t *testing.T) {
	m, _ := bvAddFixture(t)
	p := bvNotTestOwner(t, m, false, false)
	x, y := bvPropagationVar(t, m, 0, 4), bvPropagationVar(t, m, 1, 4)
	n := bvAddNum(t, m, 15, 4)
	arg := bvPropagationOp(t, m, p, 4, bvPropagationConcat(t, m, n, x), bvPropagationConcat(t, m, n, y))
	out, e := m.NewRef(x)
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	st, e := p.RewriteExtract(5, 1, arg, out)
	astTestMust(t, e)
	got, e := m.RefGet(out)
	astTestMust(t, e)
	if st != RewriteFailed || got != x {
		t.Fatal("disabled propagation changed output", st)
	}
	p.polynomial.config.bitVector.ExtractPropagation = true
	st, e = p.RewriteExtract(5, 1, arg, out)
	astTestMust(t, e)
	got, e = m.RefGet(out)
	astTestMust(t, e)
	hi, e := m.BitVectorExtractHigh(got)
	astTestMust(t, e)
	lo, e := m.BitVectorExtractLow(got)
	astTestMust(t, e)
	sum, e := m.GetArg(got, 0)
	astTestMust(t, e)
	width, e := m.BitVectorSize(sum)
	astTestMust(t, e)
	if st != Rewrite2 || hi != 5 || lo != 1 || width != 6 {
		t.Fatal("enabled propagation", st, hi, lo, width)
	}
	// Whole-width assignment returns before the default numeral/owner allocation.
	astTestMust(t, m.SetRef(out, x))
	m.Budget().FailAfter(0)
	st, e = p.RewriteExtract(3, 0, x, out)
	m.Budget().FailAfter(-1)
	astTestMust(t, e)
	if st != RewriteDone || p.extraction != nil {
		t.Fatal("identity constructed extraction scope")
	}
}
