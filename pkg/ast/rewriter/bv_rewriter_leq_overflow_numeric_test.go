package rewriter

// Source: src/ast/rewriter/bv_rewriter.cpp:255-346.

import (
	"fmt"
	"github.com/glycerine/g3/pkg/ast"
	"math/big"
	"testing"
)

func TestBVRewriterLEOverflowEveryActualNumericReservation(t *testing.T) {
	for _, tc := range []struct {
		mode  string
		width uint32
	}{
		{"interval", 65}, {"interval", 257}, {"point", 65}, {"raw-difference", 65},
	} {
		complete, failures := false, 0
		for ordinal := int64(0); ordinal < 2048 && !complete; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d/%d", tc.mode, tc.width, ordinal), func(t *testing.T) {
				m, q, fid := bvOrRuntimeFixture(t)
				p := bvNotTestOwner(t, m, false, false)
				power := new(big.Int).Lsh(big.NewInt(1), uint(tc.width))
				av := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 64), big.NewInt(9))
				bv := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 64), big.NewInt(5))
				if tc.mode == "point" {
					av.Add(bv, big.NewInt(1))
				}
				if tc.mode == "raw-difference" {
					av.Add(power, big.NewInt(5))
					bv.Add(power, big.NewInt(9))
				}
				x := bvPropagationVar(t, m, 0, tc.width)
				a := bvPropagationOp(t, m, p, 4, bvOrRawNumeral(t, m, q, fid, tc.width, av), x)
				b := bvPropagationOp(t, m, p, 4, bvOrRawNumeral(t, m, q, fid, tc.width, bv), x)
				out, e := m.NewRef(ast.ASTHandle{})
				astTestMust(t, e)
				defer func() { astTestMust(t, m.DelRef(&out)) }()
				m.Budget().FailAfter(ordinal)
				st, body := p.RewriteLEOverflow(false, a, b, out)
				m.Budget().FailAfter(-1)
				if body != nil {
					failures++
					if !numericHoistRefusal(body) {
						t.Fatal(body)
					}
					_, replay := p.RewriteLEOverflow(false, a, b, out)
					if replay != body {
						t.Fatal("stopped replay", replay, body)
					}
				} else {
					complete = true
					if st != Rewrite2 || p.leOverflow != nil {
						t.Fatal("source return", st)
					}
					got, e := m.RefGet(out)
					astTestMust(t, e)
					kind, e := m.GetDeclKind(got)
					astTestMust(t, e)
					read := func(h ast.ASTHandle, want *big.Int) {
						t.Helper()
						v, e := q.NewValue()
						astTestMust(t, e)
						var bits uint32
						_, e = m.BitVectorNumeralValue(h, &v, &bits)
						astTestMust(t, e)
						text, e := q.String(&v)
						astTestMust(t, e)
						astTestMust(t, q.Del(&v))
						if text != want.String() || bits != tc.width {
							t.Fatal("independent overflow boundary", text, want, bits)
						}
					}
					if tc.mode == "raw-difference" {
						c, e := m.GetArg(got, 0)
						astTestMust(t, e)
						right, e := m.GetArg(got, 1)
						astTestMust(t, e)
						if kind != 22 || right != b {
							t.Fatal("raw coefficient comparison was normalized/rebuilt")
						}
						read(c, new(big.Int).Sub(bv, av))
					} else if tc.mode == "point" {
						left, e := m.GetArg(got, 0)
						astTestMust(t, e)
						c, e := m.GetArg(got, 1)
						astTestMust(t, e)
						if kind != ast.ASTOpEq || left != x {
							t.Fatal("point shape")
						}
						read(c, new(big.Int).Sub(power, av))
					} else {
						if kind != ast.ASTOpAnd {
							t.Fatal("interval shape", kind)
						}
						lo, e := m.GetArg(got, 0)
						astTestMust(t, e)
						hi, e := m.GetArg(got, 1)
						astTestMust(t, e)
						lower, e := m.GetArg(lo, 0)
						astTestMust(t, e)
						lc, e := m.GetArg(lo, 1)
						astTestMust(t, e)
						uc, e := m.GetArg(hi, 0)
						astTestMust(t, e)
						upper, e := m.GetArg(hi, 1)
						astTestMust(t, e)
						if lc != x || uc != x {
							t.Fatal("common expression altered")
						}
						read(lower, new(big.Int).Sub(power, av))
						read(upper, new(big.Int).Sub(new(big.Int).Sub(power, bv), big.NewInt(1)))
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
		if !complete || failures == 0 {
			t.Fatal("actual overflow comparison incomplete", tc, complete, failures)
		}
	}
}
