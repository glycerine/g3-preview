package ast

// Source: src/ast/rewriter/bv_rewriter.cpp:262-263,298-303,315-346.

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	"math/big"
	"testing"
)

func TestASTBitVectorOverflowComparisonEveryActualReservation(t *testing.T) {
	for _, width := range []uint32{65, 257} {
		for _, mode := range []string{"difference", "bounds"} {
			complete, failures := false, 0
			for ordinal := int64(0); ordinal < 1024 && !complete; ordinal++ {
				t.Run(fmt.Sprintf("%s/%d/%d", mode, width, ordinal), func(t *testing.T) {
					m := bvOffsetRuntimeTestManager(t)
					_, e := m.RegisterArithmeticPlugin()
					astTestMust(t, e)
					fid, e := m.RegisterBitVectorPlugin()
					astTestMust(t, e)
					sort, e := m.BitVectorSort(width)
					astTestMust(t, e)
					av := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 64), big.NewInt(9))
					bv := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 64), big.NewInt(5))
					if mode == "difference" {
						av, bv = bv, av
					}
					a := bvRawOffsetTestNumeral(t, m, fid, sort, av.String(), width)
					b := bvRawOffsetTestNumeral(t, m, fid, sort, bv.String(), width)
					m.Budget().FailAfter(ordinal)
					n, body := NewBitVectorOverflowComparisonNumerals(m)
					if body == nil {
						body = n.Zero(0)
					}
					if body == nil {
						body = n.Zero(1)
					}
					var bits uint32
					if body == nil {
						_, body = n.Read(0, a, &bits)
					}
					if body == nil {
						_, body = n.Read(1, b, &bits)
					}
					if body == nil {
						if mode == "difference" {
							body = n.Difference()
						} else {
							body = n.Bounds(width)
						}
					}
					var node ASTHandle
					if body == nil {
						node, body = n.Constant(2, width)
					}
					m.Budget().FailAfter(-1)
					if body == nil {
						complete = true
						if n.placed[4] || n.placed[5] || !n.placed[0] || !n.placed[1] || !n.placed[2] || mode == "bounds" && !n.placed[3] {
							t.Fatal("source numerical lifetimes", n.placed)
						}
						power := new(big.Int).Lsh(big.NewInt(1), uint(width))
						want := new(big.Int).Sub(power, av)
						if mode == "difference" {
							want.Sub(bv, av)
						}
						text, e := m.numerals.String(&n.values[2])
						astTestMust(t, e)
						if text != want.String() {
							t.Fatal("independent lower/difference", text, want)
						}
						if mode == "bounds" {
							upper := new(big.Int).Sub(new(big.Int).Sub(power, bv), big.NewInt(1))
							text, e = m.numerals.String(&n.values[3])
							astTestMust(t, e)
							if text != upper.String() {
								t.Fatal("independent upper", text, upper)
							}
						}
						ref, e := m.NewRef(node)
						astTestMust(t, e)
						defer func() { astTestMust(t, m.DelRef(&ref)) }()
						astTestMust(t, n.SourceEnd())
					} else {
						failures++
						var f umemory.Failure
						if !errors.Is(body, arena.ErrInjectedFailure) && (!errors.As(body, &f) || f.Kind != umemory.FailureMemout) {
							t.Fatal(body)
						}
						if n != nil {
							_, e = n.Read(0, a, &bits)
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
				t.Fatal("actual overflow numerals incomplete", mode, width, complete, failures)
			}
		}
	}
}
