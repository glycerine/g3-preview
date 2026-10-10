// Source: src/ast/ast.cpp:1781-1844.
package ast

import (
	"testing"
)

func TestASTDecRefConsumedNativeWideDeletion(t *testing.T) {
	m := astTestManager(t)
	sort, e := m.MkBoolSort()
	astTestMust(t, e)
	var args [32]ASTHandle
	for i := range args {
		args[i], e = m.MkVar(uint32(i), sort)
		astTestMust(t, e)
	}
	h, e := m.MkFamilyApp(0, ASTOpAnd, nil, args[:])
	astTestMust(t, e)
	astTestMust(t, m.IncRef(h))
	m.Budget().FailAfter(0)
	consumed, body := m.DecRefConsumed(h)
	m.Budget().FailAfter(-1)
	if !consumed || body != nil {
		t.Fatal("actual deletion consumption", consumed, body)
	}
	// Native worklist growth completes the graph without budget reservations.
	if _, e = m.internal(h); e != ErrASTHandle {
		t.Fatal("consumed parent still live", e)
	}
	for _, arg := range args {
		if _, e = m.internal(arg); e != ErrASTHandle {
			t.Fatal("reached child not deleted", e)
		}
	}
	consumed, e = m.DecRefConsumed(h)
	if consumed || e != ErrASTHandle {
		t.Fatal("stale token must not report consumption", consumed, e)
	}
	consumed, e = m.DecRefConsumed(ASTHandle{})
	astTestMust(t, e)
	if !consumed {
		t.Fatal("null reference consumption")
	}
}
