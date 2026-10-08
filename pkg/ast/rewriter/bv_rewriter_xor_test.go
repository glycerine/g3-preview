// Source: src/ast/rewriter/bv_rewriter.cpp:1736-1903.
package rewriter

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestBVRewriterXOROriginalBranchesAndStatuses(t *testing.T) {
	m, _ := bvAddFixture(t)
	p := bvNotTestOwner(t, m, false, false)
	xs := bvMergeVars(t, m, 3)
	x, y, z := xs[0], xs[1], xs[2]
	not := func(h ast.ASTHandle) ast.ASTHandle {
		v, e := m.MkBitVectorNot(h)
		astTestMust(t, e)
		astTestPin(t, m, v)
		return v
	}
	xor := func(hs ...ast.ASTHandle) ast.ASTHandle {
		v, e := m.MkBitVectorXor(hs)
		astTestMust(t, e)
		astTestPin(t, m, v)
		return v
	}
	extract := func(hi, lo uint32, h ast.ASTHandle) ast.ASTHandle {
		v, e := m.MkBitVectorExtract(hi, lo, h)
		astTestMust(t, e)
		astTestPin(t, m, v)
		return v
	}
	nx, ny := not(x), not(y)
	mask, e := m.MkBitVectorConcat([]ast.ASTHandle{extract(7, 4, x), extract(3, 3, nx), extract(2, 1, x), extract(0, 0, nx)})
	astTestMust(t, e)
	astTestPin(t, m, mask)
	zero, all := bvAddNum(t, m, 0, 8), bvAddNum(t, m, 255, 8)
	for _, tc := range []struct {
		name string
		args []ast.ASTHandle
		want ast.ASTHandle
		st   RewriteStatus
	}{
		{"singleton", []ast.ASTHandle{x}, x, RewriteDone},
		{"duplicate", []ast.ASTHandle{x, x}, zero, RewriteDone},
		{"negative-duplicate", []ast.ASTHandle{nx, nx}, zero, RewriteDone},
		{"odd-duplicate", []ast.ASTHandle{x, x, x}, x, RewriteDone},
		{"opposite", []ast.ASTHandle{x, nx}, all, RewriteDone},
		{"opposite-reverse", []ast.ASTHandle{nx, x}, all, RewriteDone},
		{"two-opposites", []ast.ASTHandle{x, nx, y, ny}, zero, RewriteDone},
		{"opposite-plus-positive", []ast.ASTHandle{x, nx, x}, nx, RewriteDone},
		{"opposite-plus-negative", []ast.ASTHandle{nx, x, nx}, not(nx), RewriteDone},
		{"first-survivor-order", []ast.ASTHandle{x, y, x, x}, xor(x, y), RewriteDone},
		{"numerals", []ast.ASTHandle{bvAddNum(t, m, 3, 8), bvAddNum(t, m, 2, 8)}, bvAddNum(t, m, 1, 8), RewriteDone},
		{"zero", []ast.ASTHandle{zero, zero}, zero, RewriteDone},
		{"erase-zero", []ast.ASTHandle{x, zero}, x, RewriteDone},
		{"coefficients-cancel", []ast.ASTHandle{x, bvAddNum(t, m, 1, 8), bvAddNum(t, m, 1, 8)}, x, RewriteDone},
		{"failed", []ast.ASTHandle{x, y}, ast.ASTHandle{}, RewriteFailed},
		{"single-coefficient-failed", []ast.ASTHandle{x, y, bvAddNum(t, m, 1, 8)}, ast.ASTHandle{}, RewriteFailed},
		{"all-one-with-two-symbols", []ast.ASTHandle{x, y, all}, xor(all, x, y), RewriteDone},
		{"mask-all-one-direct", []ast.ASTHandle{x, all}, extract(7, 0, nx), Rewrite3},
		{"mask", []ast.ASTHandle{x, bvAddNum(t, m, 9, 8)}, mask, Rewrite3},
		{"flatten-merge", []ast.ASTHandle{xor(x, y), x}, y, RewriteDone},
		{"flatten-one-level", []ast.ASTHandle{xor(xor(x, y), z), z}, xor(x, y), RewriteDone},
		{"equal-count-discarded", []ast.ASTHandle{xor(x), y}, ast.ASTHandle{}, RewriteFailed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, e := m.NewRef(z)
			astTestMust(t, e)
			defer func() { astTestMust(t, m.DelRef(&out)) }()
			st, e := p.RewriteXOR(ast.SliceArguments(tc.args), out)
			astTestMust(t, e)
			got, e := m.RefGet(out)
			astTestMust(t, e)
			want := tc.want
			if tc.st == RewriteFailed {
				want = z
			}
			if st != tc.st || got != want || p.bvxor != nil {
				t.Fatal("source XOR result/status", st, tc.st, got, want)
			}
		})
	}
	st, e := p.RewriteXOR(ast.SliceArguments(xs[:2]), ast.ASTRef{})
	astTestMust(t, e)
	if st != RewriteFailed {
		t.Fatal("false branch read absent output")
	}
}

func bvXORColdInput(t *testing.T, m *ast.ASTManager, xs []ast.ASTHandle, mode string) ([]ast.ASTHandle, RewriteStatus) {
	t.Helper()
	x, y := xs[0], xs[1]
	switch mode {
	case "negative", "opposite", "opposite-reverse", "opposite-survivor":
		n, e := m.MkBitVectorNot(x)
		astTestMust(t, e)
		astTestPin(t, m, n)
		if mode == "negative" {
			return []ast.ASTHandle{n, n}, RewriteDone
		}
		if mode == "opposite" {
			return []ast.ASTHandle{x, n}, RewriteDone
		}
		if mode == "opposite-reverse" {
			return []ast.ASTHandle{n, x}, RewriteDone
		}
		return []ast.ASTHandle{x, n, x}, RewriteDone
	case "numeric":
		return []ast.ASTHandle{bvAddNum(t, m, 3, 65), bvAddNum(t, m, 2, 65)}, RewriteDone
	case "mask", "single-piece":
		v := uint32(85)
		if mode == "single-piece" {
			v = 255
		}
		return []ast.ASTHandle{x, bvAddNum(t, m, v, 8)}, Rewrite3
	case "flat":
		n, e := m.MkBitVectorXor(xs)
		astTestMust(t, e)
		astTestPin(t, m, n)
		return []ast.ASTHandle{n, x}, RewriteDone
	case "spill":
		return append(append([]ast.ASTHandle{}, xs...), x), RewriteDone
	case "failed":
		return []ast.ASTHandle{x, y}, RewriteFailed
	case "zero":
		return []ast.ASTHandle{x, bvAddNum(t, m, 0, 8)}, RewriteDone
	default:
		return []ast.ASTHandle{x, x}, RewriteDone
	}
}

func TestBVRewriterXOREveryColdReservationAndPublication(t *testing.T) {
	for _, mode := range []string{"duplicate", "negative", "opposite", "opposite-reverse", "opposite-survivor", "numeric", "zero", "mask", "single-piece", "flat", "spill", "failed"} {
		complete, published := false, false
		unusedConstant := false
		for ordinal := int64(0); ordinal < 1024 && !complete; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", mode, ordinal), func(t *testing.T) {
				m, _ := bvAddFixture(t)
				xs := bvMergeVars(t, m, 40)
				args, wantStatus := bvXORColdInput(t, m, xs, mode)
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
				st, body := p.RewriteXOR(ast.SliceArguments(args), out)
				m.Budget().FailAfter(-1)
				got, e := m.RefGet(out)
				astTestMust(t, e)
				if body == nil {
					complete = true
					if st != wantStatus || p.bvxor != nil || mode == "failed" && got != old {
						t.Fatal("source status/end", st, wantStatus)
					}
				} else {
					if !numericHoistRefusal(body) {
						t.Fatal(body)
					}
					if got != old {
						published = true
						if st != wantStatus {
							t.Fatal("published status", st, wantStatus)
						}
					}
					if p.bvxor != nil && p.bvxor.constant != (ast.ASTRef{}) {
						value, e := m.RefGet(p.bvxor.constant)
						astTestMust(t, e)
						if value.IsNull() && len(p.bvxor.args.Slice()) == 0 {
							unusedConstant = true
						}
					}
					_, replay := p.RewriteXOR(ast.SliceArguments(args), out)
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
			t.Fatal("cold campaign incomplete", mode, complete, published)
		}
		if mode == "duplicate" && !unusedConstant {
			t.Fatal("original unused c constructor not reached before zero factory")
		}
	}
}

func TestBVRewriterXORSortingFlatConfigurationAndErasedSingleton(t *testing.T) {
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
	st, e := p.RewriteXOR(ast.SliceArguments([]ast.ASTHandle{xs[1], xs[0]}), out)
	astTestMust(t, e)
	got, e := m.RefGet(out)
	astTestMust(t, e)
	want, e := m.MkBitVectorXor(xs[:2])
	astTestMust(t, e)
	astTestPin(t, m, want)
	if st != RewriteDone || got != want {
		t.Fatal("source sorting", st, got, want)
	}
	st, e = p.RewriteXOR(ast.SliceArguments(xs[:2]), ast.ASTRef{})
	astTestMust(t, e)
	if st != RewriteFailed {
		t.Fatal("already sorted", st)
	}
	key = "flat"
	astTestMust(t, p.context.SetBoolName(&local, &key, false))
	key = "bv_sort_ac"
	astTestMust(t, p.context.SetBoolName(&local, &key, false))
	astTestMust(t, p.Update(&local))
	st, e = p.RewriteXOR(ast.SliceArguments([]ast.ASTHandle{want, xs[2]}), ast.ASTRef{})
	astTestMust(t, e)
	if st != RewriteFailed {
		t.Fatal("flat disabled", st)
	}
	truth, e := m.MkTrue()
	astTestMust(t, e)
	astTestPin(t, m, truth)
	astTestMust(t, m.SetRef(out, xs[2]))
	before := m.Stats()
	m.Budget().FailAfter(0)
	st, e = p.RewriteXOR(ast.SliceArguments([]ast.ASTHandle{truth}), out)
	m.Budget().FailAfter(-1)
	astTestMust(t, e)
	got, e = m.RefGet(out)
	astTestMust(t, e)
	if st != RewriteDone || got != truth || m.Stats() != before || p.bvxor != nil {
		t.Fatal("singleton placed numerical/mark scopes")
	}
	triple, e := m.MkBitVectorXor([]ast.ASTHandle{xs[0], xs[0], xs[0]})
	astTestMust(t, e)
	astTestMust(t, m.SetRef(out, triple))
	view, e := m.AppArgumentView(triple)
	astTestMust(t, e)
	st, e = p.RewriteXOR(view, out)
	astTestMust(t, e)
	got, e = m.RefGet(out)
	astTestMust(t, e)
	if st != RewriteDone || got != xs[0] {
		t.Fatal("source view invalidated at publication", st, got)
	}
}
