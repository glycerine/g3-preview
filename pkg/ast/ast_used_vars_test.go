// Source: src/ast/used_vars.cpp:22-113; src/ast/used_vars.h:25-56.
package ast

import (
	"errors"
	"testing"

	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestASTUsedVarsCompleteTraversalAndQueries(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	vars := make([]ASTHandle, 4)
	for i := range vars {
		vars[i], e = m.MkVar(uint32(i), b)
		astTestMust(t, e)
		astTestPin(t, m, vars[i])
	}
	d, e := m.MkFuncDecl(usymbol.Numerical(40), []ASTHandle{b}, b, ASTFuncDeclInfo{})
	astTestMust(t, e)
	astTestPin(t, m, d)
	noPattern, e := m.MkApp(d, []ASTHandle{vars[0]})
	astTestMust(t, e)
	astTestPin(t, m, noPattern)
	q, e := m.MkQuantifier(ASTQuantifierSpec{
		Forall: true, Sorts: []ASTHandle{b}, Names: []usymbol.Symbol{usymbol.Numerical(41)},
		Body: vars[3], NoPatterns: []ASTHandle{noPattern},
	})
	astTestMust(t, e)
	astTestPin(t, m, q)
	root, e := m.MkAnd(vars[1], q)
	astTestMust(t, e)
	astTestPin(t, m, root)
	u, e := NewASTUsedVars(m)
	astTestMust(t, e)
	astTestMust(t, u.Collect(root))
	// Direct var 1 and quantified body var 3 map to free indices 1/2.
	// The no-pattern variable 0 is bound.
	if u.MaxFoundVarIndexPlusOne() != 3 || u.NumVars() != 2 || u.UsesAllVars(3) ||
		u.UsesAVar(1) || !u.UsesAVar(2) || !u.Contains(0).IsNull() {
		t.Fatal("source free-variable set", u.MaxFoundVarIndexPlusOne(), u.NumVars())
	}
	for _, i := range []uint32{1, 2} {
		sort, err := u.Get(i)
		astTestMust(t, err)
		if sort != b || u.Contains(i) != b {
			t.Fatal("found sort", i, sort, u.Contains(i), b)
		}
	}
	if _, e = u.Get(3); !errors.Is(e, ErrASTDomain) {
		t.Fatal("unchecked source get boundary", e)
	}
	astTestMust(t, u.Process(vars[0]))
	if u.NumVars() != 3 || !u.UsesAllVars(3) || !u.Contains(3).IsNull() {
		t.Fatal("process accumulation", u.NumVars(), u.MaxFoundVarIndexPlusOne())
	}
	u.Reset()
	if u.NumVars() != 0 || u.MaxFoundVarIndexPlusOne() != 0 || u.UsesAVar(4) {
		t.Fatal("reset retained variables")
	}
	copyOwner := *u
	if e = copyOwner.Process(root); !errors.Is(e, ErrASTHandle) {
		t.Fatal("copied collector accepted", e)
	}
}

func TestASTUsedVarsSharedNodeCacheIncludesBinderDepth(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	v, e := m.MkVar(1, b)
	astTestMust(t, e)
	astTestPin(t, m, v)
	q, e := m.MkQuantifier(ASTQuantifierSpec{Forall: true, Sorts: []ASTHandle{b}, Names: []usymbol.Symbol{usymbol.Numerical(42)}, Body: v})
	astTestMust(t, e)
	astTestPin(t, m, q)
	root, e := m.MkAnd(v, q)
	astTestMust(t, e)
	astTestPin(t, m, root)
	u, e := NewASTUsedVars(m)
	astTestMust(t, e)
	astTestMust(t, u.Collect(root))
	if !u.UsesAllVars(2) || u.NumVars() != 2 {
		t.Fatal("same shared node at distinct binder depths", u.NumVars(), u.MaxFoundVarIndexPlusOne())
	}
}
