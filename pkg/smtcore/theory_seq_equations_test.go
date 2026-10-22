package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestSMTSequenceEquationQueueMutationAndScopedRestore(t *testing.T) {
	m := astTestManager(t)
	left, err := m.MkString("left")
	astTestMust(t, err)
	right, err := m.MkString("right")
	astTestMust(t, err)
	third, err := m.MkString("third")
	astTestMust(t, err)
	manager := &smtSequenceDependencyManager{}
	dependency := manager.Leaf(smtSequenceAssumption{literal: NewSMTLiteral(2, false)})
	queue, err := newSMTSequenceEquationQueue(m)
	astTestMust(t, err)
	t.Cleanup(func() {
		if !queue.closed {
			astTestMust(t, queue.Close())
		}
	})
	astTestMust(t, queue.Push(7, []ast.ASTHandle{left}, []ast.ASTHandle{right}, dependency))
	astTestMust(t, queue.PushScope())
	astTestMust(t, queue.Set(0, 8, []ast.ASTHandle{third, left}, []ast.ASTHandle{right}, nil))
	astTestMust(t, queue.Push(9, []ast.ASTHandle{right}, []ast.ASTHandle{third}, dependency))
	if queue.Len() != 2 {
		t.Fatal("scoped mutation size", queue.Len())
	}
	astTestMust(t, queue.PushScope())
	astTestMust(t, queue.PopBack())
	astTestMust(t, queue.PopScopes(1))
	if id, lhs, rhs, dep, e := queue.At(0); e != nil || id != 8 || len(lhs) != 2 || lhs[0] != third || lhs[1] != left || len(rhs) != 1 || rhs[0] != right || dep != nil {
		t.Fatal("nested restore", id, lhs, rhs, dep, e)
	}
	astTestMust(t, queue.PopScopes(1))
	if queue.Len() != 1 {
		t.Fatal("outer restore size", queue.Len())
	}
	if id, lhs, rhs, dep, e := queue.At(0); e != nil || id != 7 || len(lhs) != 1 || lhs[0] != left || len(rhs) != 1 || rhs[0] != right || dep != dependency {
		t.Fatal("outer restore", id, lhs, rhs, dep, e)
	}
	if _, _, _, _, e := queue.At(1); e == nil {
		t.Fatal("out-of-range equation accepted")
	}
	astTestMust(t, queue.Close())
}
