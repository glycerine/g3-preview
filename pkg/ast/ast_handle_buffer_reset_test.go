// Source: src/util/buffer.h:98-104.
package ast

import "testing"

func TestASTHandleBufferResetRetainsSpillCapacity(t *testing.T) {
	m := astTestManager(t)
	var buffer ASTHandleBuffer
	defer buffer.Close(m.Budget())
	for i := 0; i < 20; i++ {
		astTestMust(t, buffer.Push(m.Budget(), ASTHandle{}))
	}
	baseline := m.Budget().Used()
	buffer.Reset()
	if len(buffer.Slice()) != 0 || m.Budget().Used() != baseline {
		t.Fatal("reset discarded capacity")
	}
	m.Budget().FailAfter(0)
	for i := 0; i < 20; i++ {
		if e := buffer.Push(m.Budget(), ASTHandle{}); e != nil {
			m.Budget().FailAfter(-1)
			t.Fatal("reset buffer reserved again", e)
		}
	}
	m.Budget().FailAfter(-1)
	if len(buffer.Slice()) != 20 || m.Budget().Used() != baseline {
		t.Fatal("reset reuse changed charge")
	}
}
