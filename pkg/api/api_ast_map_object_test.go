package api

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestAPIASTMapConcreteReferenceCleanup(t *testing.T) {
	m := astTestManager(t)
	c, e := newAPIContextResults(m, false)
	astTestMust(t, e)
	a, e := newAPIASTMapObject(c)
	astTestMust(t, e)
	astTestMust(t, c.saveObject(&a.object))
	sort, e := m.MkBoolSort()
	astTestMust(t, e)
	key, e := m.MkVar(33001, sort)
	astTestMust(t, e)
	value, e := m.MkVar(33002, sort)
	astTestMust(t, e)
	for _, h := range []ast.ASTHandle{key, value, sort} {
		astTestMust(t, m.IncRef(h))
	}
	checkRefs := func(h ast.ASTHandle, want uint32) {
		t.Helper()
		rc, e := m.GetRefCount(h)
		if e != nil || rc != want {
			t.Fatalf("AST map ref count = %d, want %d: %v", rc, want, e)
		}
	}
	keyBase, e := m.GetRefCount(key)
	astTestMust(t, e)
	valueBase, e := m.GetRefCount(value)
	astTestMust(t, e)
	sortBase, e := m.GetRefCount(sort)
	astTestMust(t, e)
	astTestMust(t, a.insert(key, value))
	checkRefs(key, keyBase+1)
	checkRefs(value, valueBase+1)
	astTestMust(t, a.insert(key, value)) // replacement INC precedes DEC
	checkRefs(value, valueBase+1)
	astTestMust(t, a.insert(key, sort)) // any AST kind is a valid value
	checkRefs(value, valueBase)
	checkRefs(sort, sortBase+1)
	if got, e := a.find(key); e != nil || got != sort {
		t.Fatal("replacement lookup", got, e)
	}
	if size, e := a.size(); e != nil || size != 1 {
		t.Fatal("map size", size, e)
	}
	if keys, e := a.keys(); e != nil || len(keys) != 1 || keys[0] != key {
		t.Fatal("borrowed keys", keys, e)
	}
	astTestMust(t, a.erase(key))
	checkRefs(key, keyBase)
	checkRefs(sort, sortBase)
	if _, e := a.find(key); !errors.Is(e, errAPIASTMapMissing) {
		t.Fatal("missing key category", e)
	}
	astTestMust(t, a.insert(key, value))
	astTestMust(t, a.reset())
	checkRefs(key, keyBase)
	checkRefs(value, valueBase)
	astTestMust(t, a.insert(key, value))
	astTestMust(t, c.Close()) // derived destructor runs dec_ref_key_values
	checkRefs(key, keyBase)
	checkRefs(value, valueBase)
	for _, h := range []ast.ASTHandle{key, value, sort} {
		astTestMust(t, m.DecRef(h))
	}
}

func TestAPIASTMapExpressionPairTransfer(t *testing.T) {
	m := astTestManager(t)
	c, e := newAPIContextResults(m, false)
	astTestMust(t, e)
	a, e := newAPIASTMapObject(c)
	astTestMust(t, e)
	astTestMust(t, c.saveObject(&a.object))
	sort, e := m.MkBoolSort()
	astTestMust(t, e)
	key, e := m.MkVar(33011, sort)
	astTestMust(t, e)
	value, e := m.MkVar(33012, sort)
	astTestMust(t, e)
	for _, h := range []ast.ASTHandle{key, value} {
		astTestMust(t, m.IncRef(h))
	}
	s, e := ast.NewASTExprMapWithProofs(m, false)
	astTestMust(t, e)
	astTestMust(t, s.Insert(key, value, ast.ASTHandle{}))
	astTestMust(t, a.copyExprPairs(s))
	astTestMust(t, s.Close())
	if got, e := a.find(key); e != nil || got != value {
		t.Fatal("API map lost source-local pair", got, e)
	}
	for _, h := range []ast.ASTHandle{key, value} {
		rc, e := m.GetRefCount(h)
		if e != nil || rc != 2 {
			t.Fatal("API map failed to retain pair", rc, e)
		}
	}
	astTestMust(t, c.Close())
	for _, h := range []ast.ASTHandle{key, value} {
		astTestMust(t, m.DecRef(h))
	}
}
