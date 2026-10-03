// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
package g3

import (
	"errors"
	"testing"
)

func TestASTDependencyRefActualOwnerCopyMoveStealAndAssignment(t *testing.T) {
	m := astTestManager(t)
	d, e := m.DependencyLeaf(astDependencyExpr(t, m, 7000))
	astTestMust(t, e)
	base := m.allocator.Stats().Live
	r, e := m.NewDependencyRef(d)
	astTestMust(t, e)
	if rc, e := m.DependencyRefCount(d); e != nil || rc != 1 {
		t.Fatal(rc, e)
	}
	c, e := m.CopyDependencyRef(r)
	astTestMust(t, e)
	astTestMust(t, m.DependencyRefAssign(r, r))
	if rc, e := m.DependencyRefCount(d); e != nil || rc != 2 {
		t.Fatal(rc, e)
	}
	moved, e := m.MoveDependencyRef(c)
	astTestMust(t, e)
	if yes, e := m.DependencyRefPresent(c); e != nil || yes {
		t.Fatal(yes, e)
	}
	if equal, e := m.DependencyRefEqual(r, moved); e != nil || !equal {
		t.Fatal(equal, e)
	}
	if owner, e := m.DependencyRefManager(r); e != nil || owner != m {
		t.Fatal(owner, e)
	}
	got, e := m.DependencyRefSteal(moved)
	astTestMust(t, e)
	if got != d {
		t.Fatal(got)
	}
	astTestMust(t, m.DependencyRefReset(r))
	astTestMust(t, m.DependencyDec(got))
	astTestMust(t, m.DelDependencyRef(&r))
	astTestMust(t, m.DelDependencyRef(&c))
	astTestMust(t, m.DelDependencyRef(&moved))
	if m.allocator.Stats().Live != base-1 {
		t.Fatal("actual owner cleanup before Close")
	}
	if _, e = m.DependencyRefGet(r); !errors.Is(e, ErrASTHandle) {
		t.Fatal(e)
	}
}
