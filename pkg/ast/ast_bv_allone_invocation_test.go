package ast

// Source: src/ast/bv_decl_plugin.cpp:794-803.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestASTBitVectorAllOneExactInvocationEveryReservation(t *testing.T) {
	complete := false
	for ordinal := int64(0); ordinal < 128 && !complete; ordinal++ {
		m := bvOffsetRuntimeTestManager(t)
		_, e := m.RegisterArithmeticPlugin()
		astTestMust(t, e)
		fid, e := m.RegisterBitVectorPlugin()
		astTestMust(t, e)
		s, e := m.BitVectorSort(65)
		astTestMust(t, e)
		value := bvRawOffsetTestNumeral(t, m, fid, s, "36893488147419103231", 65)
		var call ASTBitVectorNumeralInvocation
		m.Budget().FailAfter(ordinal)
		yes, body := m.IsBitVectorAllOneWithInvocation(value, &call)
		m.Budget().FailAfter(-1)
		if body == nil {
			complete = true
			if !yes {
				t.Fatal(ordinal)
			}
		} else {
			var failure umemory.Failure
			if !errors.Is(body, arena.ErrInjectedFailure) && (!errors.As(body, &failure) || failure.Kind != umemory.FailureMemout) {
				t.Fatal(body)
			}
		}
		before := m.BitVectorNumeralInvocationSourceReady(&call)
		copied := call
		if _, e = m.IsBitVectorAllOneWithInvocation(value, &copied); !errors.Is(e, ErrASTHandle) {
			t.Fatal("copied invocation", e)
		}
		astTestMust(t, m.CleanupBitVectorNumeralInvocation(&call))
		if after := m.BitVectorNumeralInvocationSourceReady(&call); after != before {
			t.Fatal("cleanup changed source history", before, after)
		}
		if body != nil {
			if _, e = m.IsBitVectorAllOneWithInvocation(value, &call); !errors.Is(e, ErrASTPending) {
				t.Fatal("failed call replay", e)
			}
		}
	}
	if !complete {
		t.Fatal("no completion")
	}
}
