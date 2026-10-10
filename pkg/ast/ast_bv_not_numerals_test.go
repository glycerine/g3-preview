// Source: src/ast/rewriter/bv_rewriter.cpp:1911-1949.
package ast

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	"math/big"
	"testing"
)

func TestASTBitVectorNotNumeralsEveryActualReservation(t *testing.T) {
	for _, mode := range []string{"numeric", "literal", "borrowed"} {
		for _, width := range []uint32{65, 257} {
			complete, failures := false, 0
			for ordinal := int64(0); ordinal < 512 && !complete; ordinal++ {
				t.Run(fmt.Sprintf("%s/%d/%d", mode, width, ordinal), func(t *testing.T) {
					m := bvOffsetRuntimeTestManager(t)
					_, e := m.RegisterArithmeticPlugin()
					astTestMust(t, e)
					fid, e := m.RegisterBitVectorPlugin()
					astTestMust(t, e)
					sort, e := m.BitVectorSort(width)
					astTestMust(t, e)
					arg := bvRawOffsetTestNumeral(t, m, fid, sort, "73786976294838206467", width)
					m.Budget().FailAfter(ordinal)
					n, body := NewBitVectorNotNumerals(m)
					slot := 4
					if body == nil {
						if mode == "numeric" {
							slot = 0
							var readWidth uint32
							_, body = n.Read(arg, &readWidth)
							if body == nil {
								body = n.Complement(readWidth)
							}
						} else {
							body = n.MinusOne(width, mode == "borrowed")
						}
					}
					var node ASTHandle
					if body == nil {
						node, body = n.Constant(slot, width)
					}
					m.Budget().FailAfter(-1)
					if body == nil {
						complete = true
						// Source full-expression values have ended; the named local survives
						// through factory publication, and older default val is still placed.
						if !n.placed[0] || n.placed[1] || n.placed[2] || n.placed[3] || mode != "numeric" && !n.placed[4] {
							t.Fatal("source expression lifetimes", n.placed)
						}
						if mode == "borrowed" && n.values[3] != (util.OwnedRational{}) {
							t.Fatal("borrowed one constructed literal")
						}
						ref, e := m.NewRef(node)
						astTestMust(t, e)
						defer func() { astTestMust(t, m.DelRef(&ref)) }()
						astTestMust(t, n.SourceEnd())
						astTestMust(t, n.Close())
						value, e := m.numerals.NewValue()
						astTestMust(t, e)
						var actualWidth uint32
						yes, e := m.BitVectorNumeralValue(node, &value, &actualWidth)
						astTestMust(t, e)
						text, e := m.numerals.String(&value)
						astTestMust(t, e)
						want := bvNotNumeralExpected(mode, width)
						if !yes || actualWidth != width || text != want {
							t.Fatal("result survived source locals", text, want, actualWidth)
						}
						astTestMust(t, m.numerals.Del(&value))
					} else {
						failures++
						var f umemory.Failure
						if !errors.Is(body, arena.ErrInjectedFailure) && (!errors.As(body, &f) || f.Kind != umemory.FailureMemout) {
							t.Fatal(body)
						}
					}
					if n != nil {
						copy := *n
						if e = copy.Close(); e != ErrASTHandle {
							t.Fatal("copied owner", e)
						}
						bodyBefore, sourceBefore, _ := n.scalar.Causes()
						factoryBody, factorySource := n.call.receipt.bodyCause, n.call.receipt.sourceCause
						astTestMust(t, n.Close())
						astTestMust(t, n.Close())
						bodyAfter, sourceAfter, _ := n.scalar.Causes()
						if bodyBefore != bodyAfter || sourceBefore != sourceAfter || factoryBody != n.call.receipt.bodyCause || factorySource != n.call.receipt.sourceCause {
							t.Fatal("own cleanup changed source history")
						}
					}
				})
			}
			if !complete || failures == 0 {
				t.Fatal("reservation campaign incomplete", mode, width, complete, failures)
			}
		}
	}
}

func bvNotNumeralExpected(mode string, width uint32) string {
	mask := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), uint(width)), big.NewInt(1))
	if mode != "numeric" {
		return mask.String()
	}
	value, _ := new(big.Int).SetString("73786976294838206467", 10)
	return new(big.Int).And(new(big.Int).Not(value), mask).String()
}
