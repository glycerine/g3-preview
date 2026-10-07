package ast

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestASTDependencyRefBufferInlineSourceRoutesAndCapturedAppend(t *testing.T) {
	m := astTestManager(t)
	d, e := m.DependencyLeaf(astDependencyExpr(t, m, 7200))
	astTestMust(t, e)
	pin, e := m.NewDependencyRef(d)
	astTestMust(t, e)
	b, e := m.NewDependencyRefBuffer()
	astTestMust(t, e)
	for i := 0; i < 10; i++ {
		astTestMust(t, m.DependencyRefBufferPush(b, d))
	}
	astTestMust(t, m.DependencyRefBufferAppend(b, b))
	if n, e := m.DependencyRefBufferSize(b); e != nil || n != 20 {
		t.Fatal(n, e)
	}
	if n, e := m.DependencyRefBufferCapacity(b); e != nil || n != 32 {
		t.Fatal(n, e)
	}
	c, e := m.NewDependencyRefBuffer()
	astTestMust(t, e)
	astTestMust(t, m.DependencyRefBufferAssign(c, b))
	astTestMust(t, m.DependencyRefBufferSet(c, 0, d))
	astTestMust(t, m.DependencyRefBufferSetX(c, 22, d))
	astTestMust(t, m.DependencyRefBufferShrink(c, 2))
	astTestMust(t, m.DependencyRefBufferResize(c, 3))
	astTestMust(t, m.DependencyRefBufferPop(c))
	astTestMust(t, m.DependencyRefBufferAppendValues(c, []ASTDependency{d}))
	_, e = m.DependencyRefBufferBack(c)
	astTestMust(t, e)
	el, e := m.DependencyRefBufferElement(c, 0)
	astTestMust(t, e)
	got, e := m.DependencyRefBufferBorrowGet(el)
	astTestMust(t, e)
	if got != d {
		t.Fatal(got)
	}
	astTestMust(t, m.DependencyRefBufferBorrowWriteRaw(el, d))
	it, e := m.DependencyRefBufferBegin(c)
	astTestMust(t, e)
	_, ok, e := m.DependencyRefBufferNext(&it)
	astTestMust(t, e)
	if !ok {
		t.Fatal(ok)
	}
	if owner, e := m.DependencyRefBufferManager(c); e != nil || owner != m {
		t.Fatal(owner, e)
	}
	astTestMust(t, m.DependencyRefBufferReset(c))
	astTestMust(t, m.DependencyRefBufferFinalize(c))
	if yes, e := m.DependencyRefBufferEmpty(c); e != nil || !yes {
		t.Fatal(yes, e)
	}
	if n, e := m.DependencyRefBufferCapacity(c); e != nil || n != 16 {
		t.Fatal(n, e)
	}
	astTestMust(t, m.DelDependencyRefBuffer(&b))
	astTestMust(t, m.DelDependencyRefBuffer(&c))
	if rc, e := m.DependencyRefCount(d); e != nil || rc != 1 {
		t.Fatal(rc, e)
	}
	astTestMust(t, m.DelDependencyRef(&pin))
	if _, e = m.DependencyRefBufferGet(b, 0); !errors.Is(e, ErrASTHandle) {
		t.Fatal(e)
	}
}

func TestASTDependencyRefBufferMutableBackUsesSourceAssertion(t *testing.T) {
	m := astTestManager(t)
	b, e := m.NewDependencyRefBuffer()
	astTestMust(t, e)
	_, e = m.DependencyRefBufferBackBorrow(b)
	if ASTSourceDebug {
		if !errors.Is(e, ErrASTDomain) {
			t.Fatal(e)
		}
	} else if !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	astTestMust(t, m.DependencyRefBufferResize(b, 1))
	el, e := m.DependencyRefBufferBackBorrow(b)
	astTestMust(t, e)
	astTestMust(t, m.DependencyRefBufferBorrowWriteRaw(el, ASTDependency{}))
	astTestMust(t, m.DelDependencyRefBuffer(&b))
}
