// Source: src/ast/rewriter/bv_rewriter.cpp:1513-1523,1555-1566.
package rewriter

import (
	"testing"

	"fmt"
	"github.com/glycerine/g3/pkg/ast"
)

func TestBVRewriterDerivedRepeatAndZeroExtend(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	satOK(t, e)
	_, e = m.RegisterBitVectorPlugin()
	satOK(t, e)
	c, g, _ := bvPolynomialParamsFixture(t)
	p, e := NewBVRewriter(m, c, g, nil)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	sort, e := m.BitVectorSort(8)
	satOK(t, e)
	x, e := m.MkVar(0, sort)
	satOK(t, e)
	astTestPin(t, m, x)
	out := astTestPin(t, m, x)
	for _, n := range []uint32{1, 2, 17, 40} {
		status, e := p.RewriteRepeat(n, x, out)
		satOK(t, e)
		got, e := m.RefGet(out)
		satOK(t, e)
		if n == 1 {
			if status != RewriteDone || got != x {
				t.Fatal("repeat identity")
			}
			continue
		}
		if status != Rewrite1 {
			t.Fatal("repeat rewrite depth", status)
		}
		size, e := m.BitVectorSize(got)
		satOK(t, e)
		num, e := m.GetNumArgs(got)
		satOK(t, e)
		kind, e := m.GetDeclKind(got)
		satOK(t, e)
		if size != n*8 || num != n || kind != 37 {
			t.Fatal("repeat shape", size, num, kind)
		}
		for i := uint32(0); i < n; i++ {
			arg, e := m.GetArg(got, i)
			satOK(t, e)
			if arg != x {
				t.Fatal("repeat operand", i)
			}
		}
	}
	for _, n := range []uint32{0, 1, 8, 65, 257} {
		status, e := p.RewriteZeroExtend(n, x, out)
		satOK(t, e)
		got, e := m.RefGet(out)
		satOK(t, e)
		if n == 0 {
			if status != RewriteDone || got != x {
				t.Fatal("zero extension identity")
			}
			continue
		}
		if status != Rewrite1 {
			t.Fatal("zero extension rewrite depth", status)
		}
		first, e := m.GetArg(got, 0)
		satOK(t, e)
		second, e := m.GetArg(got, 1)
		satOK(t, e)
		zero, e := m.IsBitVectorZero(first)
		satOK(t, e)
		width, e := m.BitVectorSize(first)
		satOK(t, e)
		size, e := m.BitVectorSize(got)
		satOK(t, e)
		if !zero || width != n || size != 8+n || second != x {
			t.Fatal("zero extension source shape", zero, width, size, second)
		}
	}
	// Both identity paths erase all buffer/numerical/factory work.
	satOK(t, m.SetRef(out, x))
	before := m.Stats()
	m.Budget().FailAfter(0)
	_, e = p.RewriteRepeat(1, x, out)
	satOK(t, e)
	_, e = p.RewriteZeroExtend(0, x, out)
	satOK(t, e)
	m.Budget().FailAfter(-1)
	if m.Stats() != before {
		t.Fatal("identity allocated")
	}
}

func TestBVRewriterDerivedExtendRepeatEveryColdReservation(t *testing.T) {
	for _, tc := range []struct {
		repeat bool
		n      uint32
	}{{true, 1}, {true, 17}, {true, 40}, {false, 0}, {false, 65}, {false, 257}} {
		complete, published := false, false
		for ordinal := int64(0); ordinal < 512 && !complete; ordinal++ {
			t.Run(fmt.Sprintf("%t/%d/%d", tc.repeat, tc.n, ordinal), func(t *testing.T) {
				m, _, _, _ := boolRewriterConfigFixture(t)
				_, e := m.RegisterArithmeticPlugin()
				astTestMust(t, e)
				_, e = m.RegisterBitVectorPlugin()
				astTestMust(t, e)
				c, g, _ := bvPolynomialParamsFixture(t)
				p, e := NewBVRewriter(m, c, g, nil)
				astTestMust(t, e)
				sort, e := m.BitVectorSort(8)
				astTestMust(t, e)
				x, e := m.MkVar(0, sort)
				astTestMust(t, e)
				astTestPin(t, m, x)
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
				var st RewriteStatus
				var body error
				if tc.repeat {
					st, body = p.RewriteRepeat(tc.n, x, out)
				} else {
					st, body = p.RewriteZeroExtend(tc.n, x, out)
				}
				m.Budget().FailAfter(-1)
				got, e := m.RefGet(out)
				astTestMust(t, e)
				if body == nil {
					complete = true
					want := Rewrite1
					if tc.repeat && tc.n == 1 || !tc.repeat && tc.n == 0 {
						want = RewriteDone
					}
					if st != want {
						t.Fatal("status", st, want)
					}
					if len(p.repeatArgs.Slice()) != 0 {
						t.Fatal("source buffer survived return")
					}
				} else {
					if !numericHoistRefusal(body) {
						t.Fatal(body)
					}
					if got != old {
						published = true
						if st == RewriteFailed {
							t.Fatal("publication status")
						}
					}
					var replay error
					if tc.repeat {
						_, replay = p.RewriteRepeat(tc.n, x, out)
					} else {
						_, replay = p.RewriteZeroExtend(tc.n, x, out)
					}
					if replay != body {
						t.Fatal("stopped replay", replay, body)
					}
				}
				cause := p.cause
				astTestMust(t, p.Close())
				astTestMust(t, p.Close())
				if cause != p.cause {
					t.Fatal("history changed")
				}
			})
		}
		if !complete || !published {
			t.Fatal("campaign incomplete or old DEC absent", tc, complete, published)
		}
	}
}
