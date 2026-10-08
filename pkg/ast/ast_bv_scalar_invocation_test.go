// Source: src/ast/rewriter/bv_rewriter.cpp:2048-2056;
// src/ast/rewriter/bv_rewriter.h:177.
package ast

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
	"testing"
)

func TestASTBitVectorReductionANDCallerEveryActualReservation(t *testing.T) {
	for _, mode := range []string{"allones", "noncanonical", "negative", "symbolic"} {
		complete := false
		for ordinal := int64(0); ordinal < 512 && !complete; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", mode, ordinal), func(t *testing.T) {
				m := bvOffsetRuntimeTestManager(t)
				_, e := m.RegisterArithmeticPlugin()
				astTestMust(t, e)
				fid, e := m.RegisterBitVectorPlugin()
				astTestMust(t, e)
				sort, e := m.BitVectorSort(65)
				astTestMust(t, e)
				text := "36893488147419103231"
				if mode == "noncanonical" {
					text = "73786976294838206463"
				}
				if mode == "negative" {
					text = "-1"
				}
				h := bvRawOffsetTestNumeral(t, m, fid, sort, text, 65)
				x, e := m.MkVar(0, sort)
				astTestMust(t, e)
				astTestPin(t, m, x)
				if mode == "symbolic" {
					h = x
				}
				out, e := m.NewRef(x)
				astTestMust(t, e)
				defer func() { astTestMust(t, m.DelRef(&out)) }()
				var call ASTBitVectorNumeralInvocation
				m.Budget().FailAfter(ordinal)
				n, body := NewBitVectorMinusOneNumerals(m)
				var width uint32
				numeral, all := false, false
				if body == nil {
					numeral, body = n.Read(h, &width)
				}
				if body == nil && numeral {
					all, body = n.Compare(width)
				}
				var node ASTHandle
				if body == nil && numeral {
					bit := uint32(0)
					if all {
						bit = 1
					}
					node, body = m.BitVectorNumeralUnsignedWithInvocation(bit, 1, &call)
				}
				m.Budget().FailAfter(-1)
				if body == nil {
					complete = true
					if all != (mode == "allones") {
						t.Fatal("stored value comparison", all, mode)
					}
					if !n.placed[0] {
						t.Fatal("default r missing")
					}
					for i := 1; i < 4; i++ {
						if n.placed[i] != numeral {
							t.Fatal("comparison expression lifetime", i)
						}
					}
					if numeral {
						original, e := m.numerals.String(&n.values[0])
						astTestMust(t, e)
						if original != text {
							t.Fatal("normalized stored input", original)
						}
						astTestMust(t, m.SetRef(out, node))
						for i := 1; i < 4; i++ {
							if !n.placed[i] {
								t.Fatal("temporary ended before assignment", i)
							}
						}
					} else {
						got, e := m.RefGet(out)
						astTestMust(t, e)
						if got != x {
							t.Fatal("FAILED touched output")
						}
					}
					astTestMust(t, n.SourceEnd())
				} else {
					var f util.Failure
					if !errors.Is(body, arena.ErrInjectedFailure) && (!errors.As(body, &f) || f.Kind != util.FailureMemout) {
						t.Fatal(body)
					}
				}
				if call.self != nil {
					copy := call
					if _, e = m.BitVectorNumeralUnsignedWithInvocation(0, 1, &copy); e != ErrASTHandle {
						t.Fatal("copied factory receipt", e)
					}
				}
				cb, cs := call.receipt.bodyCause, call.receipt.sourceCause
				astTestMust(t, m.CleanupBitVectorNumeralInvocation(&call))
				if cb != call.receipt.bodyCause || cs != call.receipt.sourceCause {
					t.Fatal("cleanup changed factory history")
				}
				if call.receipt.frame != nil && !call.receipt.retired && !call.receipt.frame.ownDone {
					t.Fatal("factory child not joined")
				}
				if cb != nil {
					if _, e = m.BitVectorNumeralUnsignedWithInvocation(0, 1, &call); e != ErrASTPending {
						t.Fatal("stopped factory replay", e)
					}
				}
				if n != nil {
					astTestMust(t, n.Close())
					astTestMust(t, n.Close())
				}
			})
		}
		if !complete {
			t.Fatal("actual AND campaign incomplete", mode)
		}
	}
}
