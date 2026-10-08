// Source: src/ast/rewriter/bv_rewriter.cpp:255-346.
package rewriter

import (
	"fmt"
	"github.com/glycerine/g3/pkg/ast"
	"testing"
)

func TestBVRewriterLEOverflowOriginalBranchesAndSumGuards(t *testing.T) {
	m, _ := bvAddFixture(t)
	p := bvNotTestOwner(t, m, false, false)
	xs := bvMergeVars(t, m, 4)
	x, y, z := xs[0], xs[1], xs[2]
	add := func(hs ...ast.ASTHandle) ast.ASTHandle { return bvPropagationOp(t, m, p, 4, hs...) }
	n := func(v uint32) ast.ASTHandle { return bvAddNum(t, m, v, 8) }
	cmp := func(a, b ast.ASTHandle) ast.ASTHandle {
		h, e := m.MkBitVectorULE(a, b)
		astTestMust(t, e)
		astTestPin(t, m, h)
		return h
	}
	eq := func(a, b ast.ASTHandle) ast.ASTHandle {
		h, e := m.MkEq(a, b)
		astTestMust(t, e)
		astTestPin(t, m, h)
		return h
	}
	and := func(a, b ast.ASTHandle) ast.ASTHandle {
		h, e := m.MkAndArgs([]ast.ASTHandle{a, b})
		astTestMust(t, e)
		astTestPin(t, m, h)
		return h
	}
	yes, e := m.MkTrue()
	astTestMust(t, e)
	astTestPin(t, m, yes)
	plain := add(x, y)
	for _, tc := range []struct {
		name       string
		a, b, want ast.ASTHandle
		st         RewriteStatus
	}{
		{"same-nonadd", x, x, yes, RewriteDone},
		{"equal-offset", add(n(5), x), add(n(5), x), yes, RewriteDone},
		{"less-offset", add(n(2), x), add(n(5), x), cmp(n(3), add(n(5), x)), Rewrite2},
		{"one-point", add(n(6), x), add(n(5), x), eq(x, n(250)), Rewrite2},
		{"zero-right-offset", add(n(5), x, y), plain, cmp(n(251), plain), Rewrite2},
		{"interval", add(n(9), x, y), add(n(5), x, y), and(cmp(n(247), plain), cmp(plain, n(250))), Rewrite2},
		{"missing-left-coefficient", plain, add(n(3), x, y), cmp(n(3), add(n(3), x, y)), Rewrite2},
		{"both-missing-coefficients", plain, plain, yes, RewriteDone},
		{"different-common", add(n(2), x), add(n(5), y), ast.ASTHandle{}, RewriteFailed},
		{"different-order", add(n(2), x, y), add(n(5), y, x), ast.ASTHandle{}, RewriteFailed},
		{"different-count", add(n(2), x, y), add(n(5), x), ast.ASTHandle{}, RewriteFailed},
		{"non-leading-coefficient", add(x, n(2)), add(x, n(5)), ast.ASTHandle{}, RewriteFailed},
		{"different-nonadd", x, y, ast.ASTHandle{}, RewriteFailed},
		{"mixed-sum-source-guard", bvPropagationOp(t, m, p, 6, x, y), add(n(5), bvPropagationOp(t, m, p, 6, x, y)), ast.ASTHandle{}, RewriteFailed},
		{"reverse-mixed-source-guard", add(n(5), bvPropagationOp(t, m, p, 6, x, y)), bvPropagationOp(t, m, p, 6, x, y), ast.ASTHandle{}, RewriteFailed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, e := m.NewRef(z)
			astTestMust(t, e)
			defer func() { astTestMust(t, m.DelRef(&out)) }()
			st, e := p.RewriteLEOverflow(false, tc.a, tc.b, out)
			astTestMust(t, e)
			got, e := m.RefGet(out)
			astTestMust(t, e)
			want := tc.want
			if tc.st == RewriteFailed {
				want = z
			}
			if st != tc.st || got != want || p.leOverflow != nil {
				t.Fatal("source overflow rewrite", st, tc.st, got, want)
			}
		})
	}
	// Signed rejection precedes even argument validation and default common/a0/b0.
	m.Budget().FailAfter(0)
	st, e := p.RewriteLEOverflow(true, ast.ASTHandle{}, ast.ASTHandle{}, ast.ASTRef{})
	m.Budget().FailAfter(-1)
	astTestMust(t, e)
	if st != RewriteFailed {
		t.Fatal("signed early guard")
	}
}

func TestBVRewriterLEOverflowEveryColdReservationAndPublication(t *testing.T) {
	for _, mode := range []string{"equal", "less", "point", "zero-right", "interval", "wide-common", "mismatch"} {
		complete, published := false, false
		for ordinal := int64(0); ordinal < 2048 && !complete; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", mode, ordinal), func(t *testing.T) {
				m, _ := bvAddFixture(t)
				p := bvNotTestOwner(t, m, false, false)
				xs := bvMergeVars(t, m, 40)
				add := func(hs ...ast.ASTHandle) ast.ASTHandle { return bvPropagationOp(t, m, p, 4, hs...) }
				left, right := add(bvAddNum(t, m, 9, 8), xs[0]), add(bvAddNum(t, m, 5, 8), xs[0])
				want := Rewrite2
				switch mode {
				case "equal":
					right = left
					want = RewriteDone
				case "less":
					left, right = right, left
				case "point":
					left = add(bvAddNum(t, m, 6, 8), xs[0])
				case "zero-right":
					right = add(xs[0], xs[1])
					left = add(bvAddNum(t, m, 9, 8), xs[0], xs[1])
				case "wide-common":
					left = add(append([]ast.ASTHandle{bvAddNum(t, m, 9, 8)}, xs...)...)
					right = add(append([]ast.ASTHandle{bvAddNum(t, m, 5, 8)}, xs...)...)
				case "mismatch":
					right = add(bvAddNum(t, m, 5, 8), xs[1])
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
				st, body := p.RewriteLEOverflow(false, left, right, out)
				m.Budget().FailAfter(-1)
				got, e := m.RefGet(out)
				astTestMust(t, e)
				if body == nil {
					complete = true
					if st != want || p.leOverflow != nil || want == RewriteFailed && got != old {
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
					_, replay := p.RewriteLEOverflow(false, left, right, out)
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
		if !complete || mode != "mismatch" && !published {
			t.Fatal("cold overflow incomplete", mode, complete, published)
		}
	}
}

func TestBVRewriterLEOverflowIndependentWraparoundEnumeration(t *testing.T) {
	m, q, _ := bvOrRuntimeFixture(t)
	p := bvNotTestOwner(t, m, false, false)
	x := bvPropagationVar(t, m, 0, 4)
	yes, e := m.MkTrue()
	astTestMust(t, e)
	no, e := m.MkFalse()
	astTestMust(t, e)
	var compileBV func(ast.ASTHandle) func(uint32) uint32
	compileBV = func(h ast.ASTHandle) func(uint32) uint32 {
		t.Helper()
		kind, e := m.GetKind(h)
		astTestMust(t, e)
		if kind == ast.ASTVar {
			return func(v uint32) uint32 { return v & 15 }
		}
		numeric, e := m.IsBitVectorNumeral(h)
		astTestMust(t, e)
		if numeric {
			n, e := q.NewValue()
			astTestMust(t, e)
			var width uint32
			_, e = m.BitVectorNumeralValue(h, &n, &width)
			astTestMust(t, e)
			value, e := q.Uint64(&n)
			astTestMust(t, e)
			astTestMust(t, q.Del(&n))
			return func(uint32) uint32 { return uint32(value) }
		}
		k, e := m.GetDeclKind(h)
		astTestMust(t, e)
		if k != 4 {
			t.Fatal("unsupported independent BV operation", k)
		}
		args, e := m.AppArgumentView(h)
		astTestMust(t, e)
		fs := make([]func(uint32) uint32, args.Count())
		for i := uint32(0); i < args.Count(); i++ {
			child, e := args.At(i)
			astTestMust(t, e)
			fs[i] = compileBV(child)
		}
		return func(v uint32) uint32 {
			var sum uint32
			for _, f := range fs {
				sum += f(v)
			}
			return sum & 15
		}
	}
	var compileBool func(ast.ASTHandle) func(uint32) bool
	compileBool = func(h ast.ASTHandle) func(uint32) bool {
		t.Helper()
		if h == yes {
			return func(uint32) bool { return true }
		}
		if h == no {
			return func(uint32) bool { return false }
		}
		k, e := m.GetDeclKind(h)
		astTestMust(t, e)
		a, e := m.GetArg(h, 0)
		astTestMust(t, e)
		b, e := m.GetArg(h, 1)
		astTestMust(t, e)
		fid, e := m.GetFamilyID(h)
		astTestMust(t, e)
		if fid == p.bvFamily && k == 22 {
			af, bf := compileBV(a), compileBV(b)
			return func(v uint32) bool { return af(v) <= bf(v) }
		}
		if fid == m.BasicFamilyID() && k == ast.ASTOpEq {
			af, bf := compileBV(a), compileBV(b)
			return func(v uint32) bool { return af(v) == bf(v) }
		}
		if fid == m.BasicFamilyID() && k == ast.ASTOpAnd {
			af, bf := compileBool(a), compileBool(b)
			return func(v uint32) bool { return af(v) && bf(v) }
		}
		t.Fatal("unsupported independent Boolean operation", fid, k)
		return nil
	}
	for a := uint32(0); a < 16; a++ {
		for b := uint32(0); b < 16; b++ {
			left := bvPropagationOp(t, m, p, 4, bvAddNum(t, m, a, 4), x)
			right := bvPropagationOp(t, m, p, 4, bvAddNum(t, m, b, 4), x)
			out, e := m.NewRef(ast.ASTHandle{})
			astTestMust(t, e)
			_, e = p.RewriteLEOverflow(false, left, right, out)
			astTestMust(t, e)
			got, e := m.RefGet(out)
			astTestMust(t, e)
			f := compileBool(got)
			for v := uint32(0); v < 16; v++ {
				want := ((a + v) & 15) <= ((b + v) & 15)
				if f(v) != want {
					t.Fatal("independent unsigned wraparound", a, b, v, f(v), want)
				}
			}
			astTestMust(t, m.DelRef(&out))
		}
	}
}
