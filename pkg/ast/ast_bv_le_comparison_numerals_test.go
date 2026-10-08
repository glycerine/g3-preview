// Source: src/ast/rewriter/bv_rewriter.cpp:436-517,560-568.
package ast

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
	"math/big"
	"testing"
)

func TestASTBitVectorLEComparisonEveryActualReservation(t *testing.T) {
	for _, tc := range []struct {
		mode   string
		width  uint32
		signed bool
	}{
		{"bounds", 65, false}, {"bounds", 257, true}, {"normalize", 65, true}, {"sum", 65, true}, {"zero", 257, false},
	} {
		complete, failures := false, 0
		for ordinal := int64(0); ordinal < 2048 && !complete; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d/%d", tc.mode, tc.width, ordinal), func(t *testing.T) {
				m := bvOffsetRuntimeTestManager(t)
				_, e := m.RegisterArithmeticPlugin()
				astTestMust(t, e)
				fid, e := m.RegisterBitVectorPlugin()
				astTestMust(t, e)
				sort, e := m.BitVectorSort(tc.width)
				astTestMust(t, e)
				text := "-3"
				if tc.mode == "sum" {
					text = "18446744073709551617"
				}
				a := bvRawOffsetTestNumeral(t, m, fid, sort, text, tc.width)
				b := bvRawOffsetTestNumeral(t, m, fid, sort, "18446744073709551617", tc.width)
				m.Budget().FailAfter(ordinal)
				n, body := NewBitVectorLEComparisonNumerals(m)
				var width uint32
				if body == nil {
					_, body = n.Read(0, a, &width)
				}
				if body == nil {
					_, body = n.Read(1, b, &width)
				}
				if body == nil && tc.mode == "normalize" {
					body = n.Normalize(0, width, tc.signed)
				}
				if body == nil {
					body = n.PlaceBounds()
				}
				if body == nil && tc.mode == "bounds" {
					body = n.Bounds(width, tc.signed)
				}
				var fits bool
				if body == nil && tc.mode == "sum" {
					_, body = n.RemainderIsZero()
				}
				if body == nil && tc.mode == "sum" {
					fits, body = n.SumFitsSigned(width)
				}
				if body == nil && tc.mode == "sum" {
					body = n.SumMinusOne()
				}
				var node ASTHandle
				if body == nil && tc.mode == "sum" {
					node, body = n.Constant(8, width)
				}
				if body == nil && tc.mode == "zero" {
					node, body = n.ZeroConstant(width)
				}
				m.Budget().FailAfter(-1)
				if body == nil {
					complete = true
					for i := 0; i < 5; i++ {
						if !n.placed[i] {
							t.Fatal("missing original default", i)
						}
					}
					zero, e := m.numerals.IsZero(&n.values[2])
					astTestMust(t, e)
					if !zero {
						t.Fatal("original unused r3 changed")
					}
					if tc.mode == "bounds" {
						lower := new(big.Int)
						upper := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), uint(width)), big.NewInt(1))
						if tc.signed {
							lower.Neg(new(big.Int).Lsh(big.NewInt(1), uint(width-1)))
							upper.Sub(new(big.Int).Lsh(big.NewInt(1), uint(width-1)), big.NewInt(1))
						}
						for i, want := range []*big.Int{lower, upper} {
							text, e := m.numerals.String(&n.values[3+i])
							astTestMust(t, e)
							if text != want.String() {
								t.Fatal("independent numeric bounds", text, want)
							}
						}
					}
					if tc.mode == "normalize" {
						text, e := m.numerals.String(&n.values[0])
						astTestMust(t, e)
						if text != "-3" {
							t.Fatal("signed normalization", text)
						}
					}
					if tc.mode == "sum" {
						if fits {
							t.Fatal("sum at/above signed power accepted")
						}
						value, e := m.numerals.String(&n.values[8])
						astTestMust(t, e)
						if value != "36893488147419103233" || !n.placed[11] || n.placed[5] {
							t.Fatal("sum expression lifetime/value", value, n.placed)
						}
					}
					var ref ASTRef
					if !node.IsNull() {
						ref, e = m.NewRef(node)
						astTestMust(t, e)
						defer func() { astTestMust(t, m.DelRef(&ref)) }()
					}
					astTestMust(t, n.CloseExpression())
					for i := 0; i < 5; i++ {
						if !n.placed[i] {
							t.Fatal("owning expression cleanup consumed older default", i)
						}
					}
					if e = n.SourceEnd(); e != ErrASTPending {
						t.Fatal("cleanup credited unreached source return", e)
					}
				} else {
					failures++
					var f util.Failure
					if !errors.Is(body, arena.ErrInjectedFailure) && (!errors.As(body, &f) || f.Kind != util.FailureMemout) {
						t.Fatal(body)
					}
					if n != nil {
						_, e = n.Read(0, a, &width)
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
			t.Fatal("actual LE numerals incomplete", tc, complete, failures)
		}
	}
}
