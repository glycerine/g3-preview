// Source: src/ast/num_occurs.cpp; src/ast/num_occurs.h.
package ast

import (
	"errors"
	"testing"

	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestASTNumOccursSharedEdgesRootsAndReset(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	x, e := m.MkVar(0, b)
	astTestMust(t, e)
	f, e := m.MkFuncDecl(astTestSymbol(t, m, "num-occurs-f"), []ASTHandle{b}, b, ASTFuncDeclInfo{})
	astTestMust(t, e)
	shared, e := m.MkApp(f, []ASTHandle{x})
	astTestMust(t, e)
	root, e := m.MkAnd(shared, shared)
	astTestMust(t, e)
	astTestPin(t, m, root)

	n, e := NewNumOccurs(m, false, false)
	astTestMust(t, e)
	astTestMust(t, n.Count(root))
	for _, row := range []struct {
		h    ASTHandle
		want uint32
	}{{root, 1}, {shared, 2}, {x, 1}} {
		if got := n.GetNumOccurrences(row.h); got != row.want {
			t.Fatalf("occurrences of %v = %d, want %d", row.h, got, row.want)
		}
	}

	n.Reset()
	astTestMust(t, n.CountAll([]ASTHandle{shared, shared}))
	if got := n.GetNumOccurrences(shared); got != 2 {
		t.Fatal("each repeated root must be counted", got)
	}
	if got := n.GetNumOccurrences(x); got != 1 {
		t.Fatal("a repeated root's children must be traversed once", got)
	}
}

func TestASTNumOccursRefCountAndQuantifierModes(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	x, e := m.MkVar(0, b)
	astTestMust(t, e)
	f, e := m.MkFuncDecl(astTestSymbol(t, m, "num-occurs-mode-f"), []ASTHandle{b}, b, ASTFuncDeclInfo{})
	astTestMust(t, e)
	shared, e := m.MkApp(f, []ASTHandle{x})
	astTestMust(t, e)
	body, e := m.MkAnd(shared, shared)
	astTestMust(t, e)
	q, e := m.MkQuantifier(ASTQuantifierSpec{
		Forall: true, Sorts: []ASTHandle{b}, Names: []usymbol.Symbol{astTestSymbol(t, m, "num-occurs-q")}, Body: body,
	})
	astTestMust(t, e)
	astTestPin(t, m, q)

	sharedOnly, e := NewNumOccurs(m, true, false)
	astTestMust(t, e)
	astTestMust(t, sharedOnly.Count(q))
	if got := sharedOnly.GetNumOccurrences(shared); got != 2 {
		t.Fatal("ref-count filter lost shared incoming edges", got)
	}
	if got := sharedOnly.GetNumOccurrences(x); got != 0 {
		t.Fatal("ref-count-one child was counted", got)
	}

	shallow, e := NewNumOccurs(m, false, true)
	astTestMust(t, e)
	astTestMust(t, shallow.Count(q))
	if shallow.GetNumOccurrences(q) != 1 || shallow.GetNumOccurrences(body) != 0 {
		t.Fatal("ignore-quantifiers mode traversed the body")
	}
	if e = shallow.Count(b); !errors.Is(e, ErrASTDomain) {
		t.Fatal("non-expression root", e)
	}
	if _, e = NewNumOccurs(nil, false, false); !errors.Is(e, ErrASTHandle) {
		t.Fatal("nil manager", e)
	}
}
