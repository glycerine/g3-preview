// Source: src/ast/rewriter/bv_rewriter.cpp:358-393.
package ast

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
	"math/big"
	"testing"
)

func TestASTBitVectorConcatComparisonEveryActualReservation(t *testing.T) {
	for _, tc := range []struct {
		a, b   string
		as, bs uint32
		signed bool
	}{
		{"73786976294838206467", "36893488147419103233", 67, 66, false},
		{"-3", "-2", 257, 65, true},
		{"18446744073709551616", "18446744073709551617", 65, 65, true},
	} {
		complete, failures := false, 0
		for ordinal := int64(0); ordinal < 1024 && !complete; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", tc.a, ordinal), func(t *testing.T) {
				m := bvOffsetRuntimeTestManager(t)
				_, e := m.RegisterArithmeticPlugin()
				astTestMust(t, e)
				fid, e := m.RegisterBitVectorPlugin()
				astTestMust(t, e)
				sa, e := m.BitVectorSort(tc.as)
				astTestMust(t, e)
				sb, e := m.BitVectorSort(tc.bs)
				astTestMust(t, e)
				a := bvRawOffsetTestNumeral(t, m, fid, sa, tc.a, tc.as)
				b := bvRawOffsetTestNumeral(t, m, fid, sb, tc.b, tc.bs)
				m.Budget().FailAfter(ordinal)
				n, body := NewBitVectorConcatComparisonNumerals(m)
				var as, bs uint32
				if body == nil {
					_, body = n.Read(0, a, &as)
				}
				if body == nil {
					_, body = n.Read(1, b, &bs)
				}
				small := as
				if bs < small {
					small = bs
				}
				if body == nil {
					body = n.High(0, as, small, tc.signed)
				}
				if body == nil {
					body = n.High(1, bs, small, tc.signed)
				}
				var equal, less bool
				if body == nil {
					equal, less, body = n.CompareHigh()
				}
				m.Budget().FailAfter(-1)
				if body == nil {
					complete = true
					oracle := func(text string, width uint32) *big.Int {
						v, _ := new(big.Int).SetString(text, 10)
						if width > small {
							v.Div(v, new(big.Int).Lsh(big.NewInt(1), uint(width-small)))
						}
						modulus := new(big.Int).Lsh(big.NewInt(1), uint(small))
						v.Mod(v, modulus)
						if tc.signed && v.Bit(int(small-1)) != 0 {
							v.Sub(v, modulus)
						}
						return v
					}
					aa, bb := oracle(tc.a, tc.as), oracle(tc.b, tc.bs)
					if equal != (aa.Cmp(bb) == 0) || less != (aa.Cmp(bb) < 0) || n.placed[4] || n.placed[5] || !n.placed[2] || !n.placed[3] {
						t.Fatal("source high-prefix comparison", equal, less, aa, bb, n.placed)
					}
					// Owning child cleanup preserves all four older source locals.
					astTestMust(t, n.CloseExpression())
					if !n.placed[0] || !n.placed[1] || !n.placed[2] || !n.placed[3] {
						t.Fatal("older prefix locals consumed")
					}
					if e = n.SourceEnd(); e != ErrASTPending {
						t.Fatal("cleanup credited source return", e)
					}
				} else {
					failures++
					var f util.Failure
					if !errors.Is(body, arena.ErrInjectedFailure) && (!errors.As(body, &f) || f.Kind != util.FailureMemout) {
						t.Fatal(body)
					}
					if n != nil {
						_, e = n.Read(0, a, &as)
						if e != ErrASTPending {
							t.Fatal("stopped replay", e)
						}
					}
				}
				if n != nil {
					copied := *n
					if e = copied.Close(); e != ErrASTHandle {
						t.Fatal("copied owner", e)
					}
					beforeB, beforeS, _ := n.scalar.Causes()
					astTestMust(t, n.Close())
					astTestMust(t, n.Close())
					afterB, afterS, _ := n.scalar.Causes()
					if beforeB != afterB || beforeS != afterS {
						t.Fatal("history changed")
					}
				}
			})
		}
		if !complete || failures == 0 {
			t.Fatal("actual numerical comparison incomplete", tc, complete, failures)
		}
	}
}
