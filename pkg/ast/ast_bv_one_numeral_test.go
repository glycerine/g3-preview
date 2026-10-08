// Source: src/ast/rewriter/bv_rewriter.cpp:1960-1963;
// src/util/rational.h:43,306-308.
package ast

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestASTBitVectorBorrowedOneEveryActualReservation(t *testing.T) {
	for _, width := range []uint32{1, 65, 257} {
		complete, failures := false, 0
		for ordinal := int64(0); ordinal < 256 && !complete; ordinal++ {
			m := bvOffsetRuntimeTestManager(t)
			_, e := m.RegisterArithmeticPlugin()
			astTestMust(t, e)
			_, e = m.RegisterBitVectorPlugin()
			astTestMust(t, e)
			oneBefore := m.numerals.OneField
			var call ASTBitVectorNumeralInvocation
			m.Budget().FailAfter(ordinal)
			h, body := m.BitVectorNumeralOneWithInvocation(width, &call)
			m.Budget().FailAfter(-1)
			if body == nil {
				complete = true
				astTestMust(t, m.BitVectorNumeralInvocationSourceReady(&call))
				astTestPin(t, m, h)
				v, e := m.numerals.NewValue()
				astTestMust(t, e)
				var gotWidth uint32
				yes, e := m.BitVectorNumeralValue(h, &v, &gotWidth)
				astTestMust(t, e)
				text, e := m.numerals.String(&v)
				astTestMust(t, e)
				if !yes || gotWidth != width || text != "1" {
					t.Fatal("static one numeral", text, gotWidth)
				}
				astTestMust(t, m.numerals.Del(&v))
			} else {
				failures++
				var f util.Failure
				if !errors.Is(body, arena.ErrInjectedFailure) && (!errors.As(body, &f) || f.Kind != util.FailureMemout) {
					t.Fatal(width, ordinal, body)
				}
				if _, e = m.BitVectorNumeralOneWithInvocation(width, &call); e == nil {
					t.Fatal("replayed stopped factory")
				}
			}
			if m.numerals.OneField != oneBefore {
				t.Fatal("borrowed constant changed")
			}
			if f := call.receipt.frame; f != nil && (f.placed[5] || f.children[9] != (util.RationalServiceInvocation{})) {
				t.Fatal("invented outer literal constructor")
			}
			copy := call
			if e = m.CleanupBitVectorNumeralInvocation(&copy); e != ErrASTHandle {
				t.Fatal("copied receipt", e)
			}
			beforeBody, beforeSource := call.receipt.bodyCause, call.receipt.sourceCause
			astTestMust(t, m.CleanupBitVectorNumeralInvocation(&call))
			if call.receipt.bodyCause != beforeBody || call.receipt.sourceCause != beforeSource {
				t.Fatal("cleanup changed source history")
			}
		}
		if !complete || failures == 0 {
			t.Fatal("reservation campaign incomplete", width, complete, failures)
		}
	}
}

func TestASTBitVectorBorrowedOneWidthGuard(t *testing.T) {
	m := bvOffsetRuntimeTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	var call ASTBitVectorNumeralInvocation
	m.Budget().FailAfter(0)
	_, body := m.BitVectorNumeralOneWithInvocation(0, &call)
	m.Budget().FailAfter(-1)
	var source ASTException
	if !errors.As(body, &source) {
		t.Fatal("width guard", body)
	}
	f := call.receipt.frame
	if f == nil || !f.complete[0] || f.placed[5] {
		t.Fatal("borrowed one parameter construction")
	}
	astTestMust(t, m.CleanupBitVectorNumeralInvocation(&call))
}
