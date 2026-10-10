// Source: src/ast/rewriter/bv_rewriter.cpp:1567-1734.
package rewriter

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestBVRewriterOROriginalBranchesAndStatuses(t *testing.T) {
	m, _ := bvAddFixture(t)
	p := bvNotTestOwner(t, m, false, false)
	xs := bvMergeVars(t, m, 3)
	x, y, z := xs[0], xs[1], xs[2]
	makeOR := func(hs ...ast.ASTHandle) ast.ASTHandle {
		h, e := m.MkBitVectorOr(hs)
		astTestMust(t, e)
		astTestPin(t, m, h)
		return h
	}
	makeConcat := func(hs ...ast.ASTHandle) ast.ASTHandle {
		h, e := m.MkBitVectorConcat(hs)
		astTestMust(t, e)
		astTestPin(t, m, h)
		return h
	}
	extract := func(hi, lo uint32, h ast.ASTHandle) ast.ASTHandle {
		x, e := m.MkBitVectorExtract(hi, lo, h)
		astTestMust(t, e)
		astTestPin(t, m, x)
		return x
	}
	notX, e := m.MkBitVectorNot(x)
	astTestMust(t, e)
	astTestPin(t, m, notX)
	a := makeConcat(x, bvAddNum(t, m, 0, 8))
	b := makeConcat(bvAddNum(t, m, 0, 8), y)
	zeroConcat := makeConcat(bvAddNum(t, m, 0, 8), bvAddNum(t, m, 0, 8))
	otherZeroConcat := makeConcat(bvAddNum(t, m, 0, 16))
	oneBit := bvAddNum(t, m, 1, 1)
	for _, tc := range []struct {
		name string
		args []ast.ASTHandle
		want ast.ASTHandle
		st   RewriteStatus
	}{
		{"singleton", []ast.ASTHandle{x}, x, RewriteDone},
		{"duplicate", []ast.ASTHandle{x, x}, x, RewriteDone},
		{"negative-duplicate", []ast.ASTHandle{notX, notX}, notX, RewriteDone},
		{"opposite", []ast.ASTHandle{x, notX}, bvAddNum(t, m, 255, 8), RewriteDone},
		{"opposite-reverse", []ast.ASTHandle{notX, x}, bvAddNum(t, m, 255, 8), RewriteDone},
		{"numerals", []ast.ASTHandle{bvAddNum(t, m, 1, 8), bvAddNum(t, m, 2, 8)}, bvAddNum(t, m, 3, 8), RewriteDone},
		{"all-one", []ast.ASTHandle{x, bvAddNum(t, m, 255, 8)}, bvAddNum(t, m, 255, 8), RewriteDone},
		{"zero", []ast.ASTHandle{bvAddNum(t, m, 0, 8), bvAddNum(t, m, 0, 8)}, bvAddNum(t, m, 0, 8), RewriteDone},
		{"erase-zero", []ast.ASTHandle{x, bvAddNum(t, m, 0, 8)}, x, RewriteDone},
		{"failed", []ast.ASTHandle{x, y}, ast.ASTHandle{}, RewriteFailed},
		{"single-coefficient-failed", []ast.ASTHandle{x, y, bvAddNum(t, m, 1, 8)}, ast.ASTHandle{}, RewriteFailed},
		{"coefficients", []ast.ASTHandle{x, y, bvAddNum(t, m, 1, 8), bvAddNum(t, m, 2, 8)}, makeOR(x, y, bvAddNum(t, m, 3, 8)), RewriteDone},
		{"flatten-merge", []ast.ASTHandle{makeOR(x, y), x}, makeOR(x, y), RewriteDone},
		{"flatten-one-level", []ast.ASTHandle{makeOR(makeOR(x, y), z), z}, makeOR(makeOR(x, y), z), RewriteDone},
		// Original only replaces args if flattened argument count changes.
		{"equal-count-discarded", []ast.ASTHandle{makeOR(x), y}, ast.ASTHandle{}, RewriteFailed},
		{"concat-disjoint", []ast.ASTHandle{a, b}, makeConcat(extract(15, 8, a), extract(7, 0, b)), Rewrite2},
		{"concat-both-zero", []ast.ASTHandle{zeroConcat, otherZeroConcat}, makeConcat(extract(15, 0, otherZeroConcat)), Rewrite2},
		{"concat-overlap", []ast.ASTHandle{a, makeConcat(y, bvAddNum(t, m, 0, 8))}, ast.ASTHandle{}, RewriteFailed},
		{"mask", []ast.ASTHandle{x, bvAddNum(t, m, 9, 8)}, makeConcat(extract(7, 4, x), oneBit, extract(2, 1, x), oneBit), Rewrite2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, e := m.NewRef(z)
			astTestMust(t, e)
			defer func() { astTestMust(t, m.DelRef(&out)) }()
			st, e := p.RewriteOR(ast.SliceArguments(tc.args), out)
			astTestMust(t, e)
			got, e := m.RefGet(out)
			astTestMust(t, e)
			want := tc.want
			if tc.st == RewriteFailed {
				want = z
			}
			if st != tc.st || got != want || p.bvor != nil {
				t.Fatal("original result/status", st, tc.st, got, want)
			}
		})
	}
	// FAILED must leave an absent output unread as well.
	st, e := p.RewriteOR(ast.SliceArguments([]ast.ASTHandle{x, y}), ast.ASTRef{})
	astTestMust(t, e)
	if st != RewriteFailed {
		t.Fatal("false branch", st)
	}
}

func TestBVRewriterOREveryColdReservationAndPublication(t *testing.T) {
	for _, mode := range []string{"duplicate", "negative", "opposite", "opposite-reverse", "numeric", "zero", "mask", "concat", "flat", "spill", "failed"} {
		complete, published := false, false
		for ordinal := int64(0); ordinal < 1024 && !complete; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", mode, ordinal), func(t *testing.T) {
				m, _ := bvAddFixture(t)
				p := bvNotTestOwner(t, m, false, false)
				xs := bvMergeVars(t, m, 40)
				x, y := xs[0], xs[1]
				args := []ast.ASTHandle{x, x}
				wantStatus := RewriteDone
				switch mode {
				case "negative", "opposite", "opposite-reverse":
					n, e := m.MkBitVectorNot(x)
					astTestMust(t, e)
					astTestPin(t, m, n)
					args = []ast.ASTHandle{n, n}
					if mode == "opposite" {
						args = []ast.ASTHandle{x, n}
					}
					if mode == "opposite-reverse" {
						args = []ast.ASTHandle{n, x}
					}
				case "numeric":
					args = []ast.ASTHandle{bvAddNum(t, m, 1, 65), bvAddNum(t, m, 2, 65)}
				case "zero":
					args = []ast.ASTHandle{bvAddNum(t, m, 0, 65), bvAddNum(t, m, 0, 65)}
				case "mask":
					args = []ast.ASTHandle{x, bvAddNum(t, m, 85, 8)}
					wantStatus = Rewrite2
				case "concat":
					a, e := m.MkBitVectorConcat([]ast.ASTHandle{x, bvAddNum(t, m, 0, 8)})
					astTestMust(t, e)
					astTestPin(t, m, a)
					b, e := m.MkBitVectorConcat([]ast.ASTHandle{bvAddNum(t, m, 0, 8), y})
					astTestMust(t, e)
					astTestPin(t, m, b)
					args = []ast.ASTHandle{a, b}
					wantStatus = Rewrite2
				case "flat":
					n, e := m.MkBitVectorOr(xs)
					astTestMust(t, e)
					astTestPin(t, m, n)
					args = []ast.ASTHandle{n, x}
				case "spill":
					args = append(append([]ast.ASTHandle{}, xs...), x)
				case "failed":
					args = []ast.ASTHandle{x, y}
					wantStatus = RewriteFailed
				}
				// Actual old BOR32 destruction can fail after publishing the result.
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
				st, body := p.RewriteOR(ast.SliceArguments(args), out)
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
					if st != wantStatus || p.bvor != nil || mode == "failed" && got != old {
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
					_, replay := p.RewriteOR(ast.SliceArguments(args), out)
					if replay != body {
						t.Fatal("stopped replay", replay, body)
					}
				}
				cause := p.cause
				astTestMust(t, p.Close())
				astTestMust(t, p.Close())
				if p.cause != cause {
					t.Fatal("owning cleanup changed history")
				}
			})
		}
		if !complete || mode != "failed" && !published {
			t.Fatal("cold campaign incomplete", mode, complete, published)
		}
	}
}

func TestBVRewriterORSortingFlatConfigurationAndErasedSingleton(t *testing.T) {
	m, _ := bvAddFixture(t)
	p := bvNotTestOwner(t, m, false, false)
	xs := bvMergeVars(t, m, 3)
	out, e := m.NewRef(xs[2])
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	local, e := p.context.NewValue()
	astTestMust(t, e)
	defer func() { astTestMust(t, p.context.Del(&local)) }()
	key := "bv_sort_ac"
	astTestMust(t, p.context.SetBoolName(&local, &key, true))
	astTestMust(t, p.Update(&local))
	st, e := p.RewriteOR(ast.SliceArguments([]ast.ASTHandle{xs[1], xs[0]}), out)
	astTestMust(t, e)
	got, e := m.RefGet(out)
	astTestMust(t, e)
	want, e := m.MkBitVectorOr(xs[:2])
	astTestMust(t, e)
	astTestPin(t, m, want)
	if st != RewriteDone || got != want {
		t.Fatal("source ast_to_lt sort", st, got, want)
	}
	st, e = p.RewriteOR(ast.SliceArguments(xs[:2]), ast.ASTRef{})
	astTestMust(t, e)
	if st != RewriteFailed {
		t.Fatal("already sorted should fail", st)
	}
	key = "flat"
	astTestMust(t, p.context.SetBoolName(&local, &key, false))
	key = "bv_sort_ac"
	astTestMust(t, p.context.SetBoolName(&local, &key, false))
	astTestMust(t, p.Update(&local))
	st, e = p.RewriteOR(ast.SliceArguments([]ast.ASTHandle{want, xs[2]}), ast.ASTRef{})
	astTestMust(t, e)
	if st != RewriteFailed {
		t.Fatal("flat disabled", st)
	}
	// Original singleton assignment precedes even get_bv_size/default values.
	truth, e := m.MkTrue()
	astTestMust(t, e)
	astTestPin(t, m, truth)
	astTestMust(t, m.SetRef(out, xs[2]))
	before := m.Stats()
	m.Budget().FailAfter(0)
	st, e = p.RewriteOR(ast.SliceArguments([]ast.ASTHandle{truth}), out)
	m.Budget().FailAfter(-1)
	astTestMust(t, e)
	got, e = m.RefGet(out)
	astTestMust(t, e)
	if st != RewriteDone || got != truth || m.Stats() != before || p.bvor != nil {
		t.Fatal("singleton read BV sort or placed scopes")
	}
	// The input view may become stale at publication; source return must not
	// revisit that app after its output reference has been replaced.
	duplicate, e := m.MkBitVectorOr([]ast.ASTHandle{xs[0], xs[0]})
	astTestMust(t, e)
	astTestMust(t, m.SetRef(out, duplicate))
	view, e := m.AppArgumentView(duplicate)
	astTestMust(t, e)
	st, e = p.RewriteOR(view, out)
	astTestMust(t, e)
	got, e = m.RefGet(out)
	astTestMust(t, e)
	if st != RewriteDone || got != xs[0] {
		t.Fatal("borrowed view publication", st, got)
	}
}

func TestBVRewriterORXORNativeMembershipPreservesForeignMarkPlanes(t *testing.T) {
	m, _ := bvAddFixture(t)
	p := bvNotTestOwner(t, m, false, false)
	defer func() { astTestMust(t, p.Close()) }()
	xs := bvMergeVars(t, m, 2)
	x, y := xs[0], xs[1]
	not, e := m.MkBitVectorNot(x)
	astTestMust(t, e)
	astTestPin(t, m, not)
	one, e := ast.NewASTFastMark1(m)
	astTestMust(t, e)
	defer func() { astTestMust(t, one.Close()) }()
	two, e := ast.NewASTFastMark2(m)
	astTestMust(t, e)
	defer func() { astTestMust(t, two.Close()) }()
	for _, h := range xs {
		astTestMust(t, one.Mark(h, true))
		astTestMust(t, two.Mark(h, true))
	}
	out, e := m.NewRef(ast.ASTHandle{})
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	for _, tc := range []struct {
		args []ast.ASTHandle
		want ast.ASTHandle
		xor  bool
	}{
		{[]ast.ASTHandle{x, x}, x, false},
		{[]ast.ASTHandle{not, not}, not, false},
		{[]ast.ASTHandle{x, x, y}, y, true},
		{[]ast.ASTHandle{not, not, y}, y, true},
	} {
		var st RewriteStatus
		if tc.xor {
			st, e = p.RewriteXOR(ast.SliceArguments(tc.args), out)
		} else {
			st, e = p.RewriteOR(ast.SliceArguments(tc.args), out)
		}
		astTestMust(t, e)
		got, e := m.RefGet(out)
		astTestMust(t, e)
		if st != RewriteDone || got != tc.want {
			t.Fatal("foreign marks influenced membership", tc.xor, st, got, tc.want)
		}
	}
	astTestMust(t, p.Close())
	for _, h := range xs {
		a, e := one.IsMarked(h)
		astTestMust(t, e)
		b, e := two.IsMarked(h)
		astTestMust(t, e)
		if !a || !b {
			t.Fatal("rewriter changed another caller's marks")
		}
	}
}
