// Source: src/ast/bv_decl_plugin.cpp:767-773.
package ast

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/pkg/util"
	"testing"
)

func TestASTBitVectorSignBitInvocationEveryActualReservation(t *testing.T) {
	complete := false
	for ordinal := int64(0); ordinal < 512 && !complete; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m := bvOffsetRuntimeTestManager(t)
			_, e := m.RegisterArithmeticPlugin()
			astTestMust(t, e)
			_, e = m.RegisterBitVectorPlugin()
			astTestMust(t, e)
			v, e := m.numerals.NewValue()
			astTestMust(t, e)
			defer func() { astTestMust(t, m.numerals.CleanupValue(&v)) }()
			astTestMust(t, m.numerals.SetString(&v, "-36893488147419103233"))
			var call ASTBitVectorNumeralInvocation
			m.Budget().FailAfter(ordinal)
			yes, body := m.BitVectorHasSignBitWithInvocation(&v, 65, &call)
			m.Budget().FailAfter(-1)
			if body == nil {
				complete = true
				if !yes {
					t.Fatal("original normalized sign")
				}
				astTestMust(t, m.BitVectorNumeralInvocationSourceReady(&call))
			} else if _, e = m.BitVectorHasSignBitWithInvocation(&v, 65, &call); e == nil {
				t.Fatal("stopped replay")
			}
			copy := call
			if e = m.CleanupBitVectorNumeralInvocation(&copy); e != ErrASTHandle {
				t.Fatal("copied exact receipt", e)
			}
			b, s := call.receipt.bodyCause, call.receipt.sourceCause
			astTestMust(t, m.CleanupBitVectorNumeralInvocation(&call))
			astTestMust(t, m.CleanupBitVectorNumeralInvocation(&call))
			if r := &call.receipt; r.frame != nil && !r.retired && !r.frame.ownDone {
				t.Fatal("actual sign frame not joined")
			}
			if b != call.receipt.bodyCause || s != call.receipt.sourceCause {
				t.Fatal("source history changed")
			}
			after, e := m.numerals.String(&v)
			astTestMust(t, e)
			if after != "-36893488147419103233" {
				t.Fatal("borrowed sign input modified", after)
			}
		})
	}
	if !complete {
		t.Fatal("actual sign invocation campaign incomplete")
	}
}

func TestASTBitVectorSignBitInvocationAssertionBeforeInput(t *testing.T) {
	if !ASTSourceDebug {
		return
	}
	m := astTestManager(t)
	var call ASTBitVectorNumeralInvocation
	yes, e := m.BitVectorHasSignBitWithInvocation(nil, 0, &call)
	if yes || !errors.Is(e, util.ErrGlobalParamsAssertion) || call.receipt.frame != nil {
		t.Fatal("assertion before input/frame", yes, e)
	}
	astTestMust(t, m.CleanupBitVectorNumeralInvocation(&call))
}
