package rewriter

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestTheoryRewriterFreshDefaultIsolatesLocalSettingsAndLifetime(t *testing.T) {
	m, context, globals, local := boolRewriterConfigFixture(t)
	name := "elim_and"
	astTestMust(t, context.SetBoolName(local, &name, true))
	original, err := NewTheoryRewriter(m, context, globals, local)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, original.Close()) })
	fresh, err := original.FreshDefault()
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, fresh.Close()) })
	a, b := goalTestAtom(t, m, "fresh-default-a"), goalTestAtom(t, m, "fresh-default-b")
	conjunction, err := m.MkAnd(a, b)
	astTestMust(t, err)
	astTestPin(t, m, conjunction)
	out := astTestPin(t, m, ast.ASTHandle{})
	astTestMust(t, original.Rewrite(conjunction, out))
	value, err := m.RefGet(out)
	astTestMust(t, err)
	if value == conjunction {
		t.Fatal("fixture did not enable configured conjunction elimination")
	}
	astTestMust(t, original.Close())
	astTestMust(t, fresh.Rewrite(conjunction, out))
	value, err = m.RefGet(out)
	astTestMust(t, err)
	if value != conjunction {
		t.Fatal("fresh default rewriter inherited local settings or depended on the original owner")
	}
	if _, err := original.FreshDefault(); err != ast.ErrASTHandle {
		t.Fatal("closed original created another default rewriter", err)
	}
}
