// Source: src/ast/rewriter/bv_rewriter.cpp:436-584.
package rewriter

import (
	"fmt"
	"github.com/glycerine/g3/pkg/ast"
	"math/big"
	"testing"
)

func TestBVRewriterLEEveryActualNumericReservation(t *testing.T) {
	for _, tc := range []struct {
		mode  string
		width uint32
	}{
		{"numeric", 65}, {"signed-bounds", 257}, {"remainder", 65}, {"leading", 65},
	} {
		complete, failures := false, 0
		for ordinal := int64(0); ordinal < 4096 && !complete; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d/%d", tc.mode, tc.width, ordinal), func(t *testing.T) {
				m, q, fid := bvOrRuntimeFixture(t)
				p := bvNotTestOwner(t, m, false, false)
				width := tc.width
				x := bvPropagationVar(t, m, 0, width)
				var a, b ast.ASTHandle
				signed := false
				want := RewriteDone
				switch tc.mode {
				case "numeric":
					av := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), uint(width+2)), big.NewInt(3))
					a = bvOrRawNumeral(t, m, q, fid, width, av)
					b = bvOrRawNumeral(t, m, q, fid, width, big.NewInt(-1))
				case "signed-bounds":
					a = x
					b = bvOrRawNumeral(t, m, q, fid, width, new(big.Int).Lsh(big.NewInt(1), uint(width-1)))
					signed = true
					want = Rewrite1
				case "remainder":
					divisor := bvOrRawNumeral(t, m, q, fid, width, big.NewInt(4))
					rem := bvPropagationOp(t, m, p, 19, x, divisor)
					mul := bvPropagationOp(t, m, p, 6, bvOrRawNumeral(t, m, q, fid, width, big.NewInt(-1)), rem)
					a = bvPropagationOp(t, m, p, 4, x, mul)
					signed = true
					want = Rewrite2
					// width65 max_int is 2^64-1; c1+c2-1 must be below 2^64.
					b = bvOrRawNumeral(t, m, q, fid, width, new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 64), big.NewInt(4)))
				case "leading":
					a = x
					b = bvOrRawNumeral(t, m, q, fid, width, new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 64), big.NewInt(1)))
					want = Rewrite3
				}
				out, e := m.NewRef(ast.ASTHandle{})
				astTestMust(t, e)
				defer func() { astTestMust(t, m.DelRef(&out)) }()
				m.Budget().FailAfter(ordinal)
				st, body := p.RewriteLE(signed, a, b, out)
				m.Budget().FailAfter(-1)
				if body != nil {
					failures++
					if !numericHoistRefusal(body) {
						t.Fatal(body)
					}
					_, replay := p.RewriteLE(signed, a, b, out)
					if replay != body {
						t.Fatal("stopped replay", replay, body)
					}
				} else {
					complete = true
					if st != want || p.leq != nil {
						t.Fatal("source return", st, want)
					}
					got, e := m.RefGet(out)
					astTestMust(t, e)
					if tc.mode == "numeric" {
						yes, e := m.MkTrue()
						astTestMust(t, e)
						if got != yes {
							t.Fatal("unsigned normalization comparison")
						}
					}
					if tc.mode == "signed-bounds" {
						eq, e := m.MkEq(x, b)
						astTestMust(t, e)
						if got != eq {
							t.Fatal("signed lower equality")
						}
					}
					if tc.mode == "remainder" {
						c, e := m.GetArg(got, 1)
						astTestMust(t, e)
						n, e := q.NewValue()
						astTestMust(t, e)
						var bits uint32
						_, e = m.BitVectorNumeralValue(c, &n, &bits)
						astTestMust(t, e)
						text, e := q.String(&n)
						astTestMust(t, e)
						astTestMust(t, q.Del(&n))
						want := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 64), big.NewInt(1))
						if text != want.String() || bits != width {
							t.Fatal("independent special-rule sum", text, want, bits)
						}
					}
					if tc.mode == "leading" {
						eq, e := m.GetArg(got, 0)
						astTestMust(t, e)
						hi, e := m.GetArg(eq, 0)
						astTestMust(t, e)
						high, e := m.BitVectorExtractHigh(hi)
						astTestMust(t, e)
						low, e := m.BitVectorExtractLow(hi)
						astTestMust(t, e)
						if high != 64 || low != 64 {
							t.Fatal("independent leading-zero interval", high, low)
						}
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
			t.Fatal("actual LE incomplete", tc, complete, failures)
		}
	}
}
