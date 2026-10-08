// Source: src/util/buffer.h:146-175; src/ast/rewriter/poly_rewriter_def.h:159-172.
package ast

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestASTHandleBufferPopRetainsSpillAndBorrowedReferences(t *testing.T) {
	m := astTestManager(t)
	s, e := m.MkBoolSort()
	astTestMust(t, e)
	x, e := m.MkVar(0, s)
	astTestMust(t, e)
	astTestPin(t, m, x)
	refs, e := m.GetRefCount(x)
	astTestMust(t, e)
	before := m.Budget().Used()
	var buffer ASTHandleBuffer
	for i := 0; i < 40; i++ {
		astTestMust(t, buffer.Push(m.Budget(), x))
	}
	used := m.Budget().Used()
	if used <= before {
		t.Fatal("spill not exercised")
	}
	m.Budget().FailAfter(0)
	for i := 0; i < 40; i++ {
		astTestMust(t, buffer.PopBack())
	}
	if e = buffer.PopBack(); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("empty checked pop", e)
	}
	astTestMust(t, buffer.Push(m.Budget(), x))
	m.Budget().FailAfter(-1)
	if m.Budget().Used() != used || len(buffer.Slice()) != 1 || buffer.Slice()[0] != x {
		t.Fatal("pop released or lost reusable backing")
	}
	now, e := m.GetRefCount(x)
	astTestMust(t, e)
	if now != refs {
		t.Fatal("buffer owned borrowed expression")
	}
	buffer.Close(m.Budget())
	if m.Budget().Used() != before {
		t.Fatal("spill retained after close")
	}
}
