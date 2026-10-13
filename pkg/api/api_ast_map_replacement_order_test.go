package api

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
)

func TestAPIASTMapReplacementReleasesOldBeforePublishingNew(t *testing.T) {
	m := astTestManager(t)
	c, err := newAPIContextResults(m, false)
	astTestMust(t, err)
	a, err := newAPIASTMapObject(c)
	astTestMust(t, err)
	astTestMust(t, c.saveObject(&a.object))
	sort, err := m.MkBoolSort()
	astTestMust(t, err)
	key, err := m.MkVar(33101, sort)
	astTestMust(t, err)
	old, err := m.MkVar(33102, sort)
	astTestMust(t, err)
	incoming, err := m.MkVar(33103, sort)
	astTestMust(t, err)
	for _, h := range []ast.ASTHandle{sort, key, incoming} {
		astTestMust(t, m.IncRef(h))
	}
	astTestMust(t, a.insert(key, old))
	m.ExprIDs.Cleanup(m.ExprIDs.Range())
	m.Budget().FailAfter(0)
	err = a.insert(key, incoming)
	m.Budget().FailAfter(-1)
	if !errors.Is(err, arena.ErrInjectedFailure) {
		t.Fatalf("old-value deletion refusal: %v", err)
	}
	got, err := a.find(key)
	if err != nil || got != old || a.entries[key].valueOwned {
		t.Fatalf("source map publication after stopped DEC: got=%v, err=%v, entry=%+v", got, err, a.entries[key])
	}
	if _, err := m.GetKind(old); err == nil || len(a.pending) != 1 {
		t.Fatalf("deleted source pointer and incoming owning receipt: old error=%v, pending=%d", err, len(a.pending))
	}
	astTestMust(t, a.reset())
	if rc, err := m.GetRefCount(incoming); err != nil || rc != 1 {
		t.Fatalf("incoming reference cleanup: rc=%d, err=%v", rc, err)
	}
	astTestMust(t, c.Close())
	for _, h := range []ast.ASTHandle{incoming, key, sort} {
		astTestMust(t, m.DecRef(h))
	}
}
