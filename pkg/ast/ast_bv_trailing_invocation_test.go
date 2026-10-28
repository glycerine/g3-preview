package ast

// Source: src/ast/rewriter/bv_trailing.cpp:213-234,311-327.

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestASTBitVectorTrailingInvocationNamesActualChild(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	input, e := m.BitVectorNumeralUnsigned(12, 8)
	astTestMust(t, e)
	out := astTestPin(t, m, input)
	var call ASTBitVectorNumeralInvocation
	yes, count, e := m.CountBitVectorNumeralTrailingZerosWithInvocation(input, &call)
	astTestMust(t, e)
	if !yes || count != 2 || call.receipt.frame == nil || !call.receipt.retired {
		t.Fatal("missing exact count invocation")
	}
	astTestMust(t, m.BitVectorNumeralInvocationSourceReady(&call))
	astTestMust(t, m.CleanupBitVectorNumeralInvocation(&call))
	yes, count, body := m.RemoveBitVectorNumeralTrailingZerosWithInvocation(input, 1, ASTRef{}, &call)
	if !yes || count != 1 || !errors.Is(body, ErrASTHandle) || call.receipt.frame == nil || call.receipt.retired || !call.receipt.sourceDone {
		t.Fatal("missing reached failed publication", yes, count, body)
	}
	astTestMust(t, m.BitVectorNumeralInvocationSourceReady(&call))
	copy := call
	if e := m.CleanupBitVectorNumeralInvocation(&copy); !errors.Is(e, ErrASTHandle) {
		t.Fatal("copied child accepted", e)
	}
	astTestMust(t, m.CleanupBitVectorNumeralInvocation(&call))
	if _, _, e := m.CountBitVectorNumeralTrailingZerosWithInvocation(input, &call); !errors.Is(e, ErrASTPending) {
		t.Fatal("failed call site reused", e)
	}
	if _, _, e := m.RemoveBitVectorNumeralTrailingZerosWithInvocation(input, 1, out, nil); !errors.Is(e, ErrASTHandle) {
		t.Fatal("nil child accepted", e)
	}
}

func TestASTBitVectorTrailingInvocationEveryActualReservation(t *testing.T) {
	for _, remove := range []bool{false, true} {
		success, failures, named := false, 0, false
		for ordinal := int64(0); ordinal < 256; ordinal++ {
			t.Run(fmt.Sprint(remove, "/", ordinal), func(t *testing.T) {
				m := bvOffsetRuntimeTestManager(t)
				_, e := m.RegisterArithmeticPlugin()
				astTestMust(t, e)
				fid, e := m.RegisterBitVectorPlugin()
				astTestMust(t, e)
				sort, e := m.BitVectorSort(8)
				astTestMust(t, e)
				input := bvTrailingColdTestNumeral(t, m, fid, sort)
				out := astTestPin(t, m, input)
				one, e := m.BitVectorNumeralUnsigned(1, 8)
				astTestMust(t, e)
				astTestPin(t, m, one)
				var older ASTBitVectorNumeralScope
				_, _, _, e = m.ReadBitVectorNumeralScope(one, &older)
				astTestMust(t, e)
				var call ASTBitVectorNumeralInvocation
				m.Budget().FailAfter(ordinal)
				var count uint32
				var body error
				if remove {
					_, count, body = m.RemoveBitVectorNumeralTrailingZerosWithInvocation(input, 2, out, &call)
				} else {
					_, count, body = m.CountBitVectorNumeralTrailingZerosWithInvocation(input, &call)
				}
				m.Budget().FailAfter(-1)
				ready := m.BitVectorNumeralInvocationSourceReady(&call)
				before := call.receipt
				if body != nil {
					failures++
					if !errors.Is(body, arena.ErrInjectedFailure) {
						var f umemory.Failure
						if !errors.As(body, &f) || f.Kind != umemory.FailureMemout {
							t.Fatal("unexpected actual Runtime refusal", body)
						}
					}
					if before.frame != nil {
						named = true
						if before.frame == older.frame || before.invocation != before.frame.invocation {
							t.Fatal("ambient older frame adopted")
						}
					}
					got, e := m.RefGet(out)
					astTestMust(t, e)
					if got != input {
						t.Fatal("failed child published")
					}
				} else {
					success = true
					want := uint32(8)
					if remove {
						want = 2
					}
					if count != want || !before.retired {
						t.Fatal("completed invocation", count, want, before.retired)
					}
				}
				astTestMust(t, m.CleanupBitVectorNumeralInvocation(&call))
				astTestMust(t, m.CleanupBitVectorNumeralInvocation(&call))
				if call.receipt.frame != before.frame || call.receipt.invocation != before.invocation || call.receipt.sourceDone != before.sourceDone || call.receipt.retired != before.retired {
					t.Fatal("owning cleanup rewrote invocation history")
				}
				after := m.BitVectorNumeralInvocationSourceReady(&call)
				if (ready == nil) != (after == nil) || ready != nil && !errors.Is(after, ready) {
					t.Fatal("owning cleanup granted source DT", ready, after)
				}
				if older.frame.consumed[0] {
					t.Fatal("child cleanup consumed the older caller's numeral")
				}
				// A retained child is independently named by call, not adopted
				// into older's source scope. Its header remains plugin-owned.
				end := m.EndBitVectorNumeralScope(&older, body)
				if body == nil {
					astTestMust(t, end)
				}
				astTestMust(t, m.CleanupBitVectorNumeralScope(&older))
			})
			if success {
				break
			}
		}
		if !success || failures == 0 || !named {
			t.Fatal("actual invocation horizon", remove, success, failures, named)
		}
	}
}
