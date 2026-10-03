// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
package g3

import (
	"errors"
	"testing"
)

func TestASTDependencyRefVectorSourceCallbacksAndFullRoutes(t *testing.T) {
	m := astTestManager(t)
	d, e := m.DependencyLeaf(astDependencyExpr(t, m, 7100))
	astTestMust(t, e)
	pin, e := m.NewDependencyRef(d)
	astTestMust(t, e)
	v, e := m.NewDependencyRefVectorFrom([]ASTDependency{d, d, {}})
	astTestMust(t, e)
	c, e := m.CopyDependencyRefVector(v)
	astTestMust(t, e)
	if n, e := m.DependencyRefVectorCapacity(c); e != nil || n != 3 {
		t.Fatal(n, e)
	}
	astTestMust(t, m.DependencyRefVectorSet(v, 0, d))
	astTestMust(t, m.DependencyRefVectorResizeFill(v, 5, d))
	astTestMust(t, m.DependencyRefVectorReserve(v, 7))
	astTestMust(t, m.DependencyRefVectorShrink(v, 5))
	astTestMust(t, m.DependencyRefVectorResize(v, 3))
	astTestMust(t, m.DependencyRefVectorPush(v, d))
	astTestMust(t, m.DependencyRefVectorPop(v))
	astTestMust(t, m.DependencyRefVectorSwapElements(v, 0, 1))
	astTestMust(t, m.DependencyRefVectorReverse(v))
	if yes, e := m.DependencyRefVectorContains(v, d); e != nil || !yes {
		t.Fatal(yes, e)
	}
	astTestMust(t, m.DependencyRefVectorEraseValue(v, d))
	astTestMust(t, m.DependencyRefVectorErase(v, 0))
	astTestMust(t, m.DependencyRefVectorAppendValues(v, []ASTDependency{d}))
	astTestMust(t, m.DependencyRefVectorAppend(v, c))
	astTestMust(t, m.DependencyRefVectorAssign(v, c))
	moved, e := m.MoveDependencyRefVector(c)
	astTestMust(t, e)
	astTestMust(t, m.SwapDependencyRefVectors(v, moved))
	it, e := m.DependencyRefVectorBegin(v)
	astTestMust(t, e)
	for i := 0; i < 3; i++ {
		_, ok, e := m.DependencyRefVectorNext(&it)
		astTestMust(t, e)
		if !ok {
			t.Fatal(i)
		}
	}
	el, e := m.DependencyRefVectorElement(v, 0)
	astTestMust(t, e)
	astTestMust(t, m.DependencyRefElementSet(el, d))
	astTestMust(t, m.DependencyRefElementAssign(el, el))
	if yes, e := m.DependencyRefElementEqual(el, d); e != nil || !yes {
		t.Fatal(yes, e)
	}
	if got, e := m.DependencyRefElementGet(el); e != nil || got != d {
		t.Fatal(got, e)
	}
	if got, e := m.DependencyRefVectorGetDefault(v, 99, d); e != nil || got != d {
		t.Fatal(got, e)
	}
	_, e = m.DependencyRefVectorBack(v)
	astTestMust(t, e)
	if owner, e := m.DependencyRefVectorManager(v); e != nil || owner != m {
		t.Fatal(owner, e)
	}
	astTestMust(t, m.DependencyRefVectorReset(v))
	astTestMust(t, m.DependencyRefVectorFinalize(v))
	if yes, e := m.DependencyRefVectorEmpty(v); e != nil || !yes {
		t.Fatal(yes, e)
	}
	for _, x := range []*ASTDependencyRefVector{&v, &c, &moved} {
		astTestMust(t, m.DelDependencyRefVector(x))
	}
	if rc, e := m.DependencyRefCount(d); e != nil || rc != 1 {
		t.Fatal(rc, e)
	}
	astTestMust(t, m.DelDependencyRef(&pin))
	if _, e = m.DependencyIsLeaf(d); !errors.Is(e, ErrASTHandle) {
		t.Fatal(e)
	}
}
