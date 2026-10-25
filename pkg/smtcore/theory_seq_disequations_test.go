package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestSMTSequenceDisequationQueueScopesAndOwnedCases(t *testing.T) {
	m := astTestManager(t)
	left, err := m.MkString("left")
	astTestMust(t, err)
	right, err := m.MkString("right")
	astTestMust(t, err)
	queue, err := newSMTSequenceDisequationQueue(m)
	astTestMust(t, err)
	t.Cleanup(func() {
		if !queue.closed {
			astTestMust(t, queue.Close())
		}
	})
	dependency := (&smtSequenceDependencyManager{}).Leaf(smtSequenceAssumption{literal: NewSMTLiteral(2, true)})
	astTestMust(t, queue.Push(left, right, dependency))
	astTestMust(t, queue.PushScope())
	astTestMust(t, queue.Push(right, left, nil))
	astTestMust(t, queue.PushScope())
	astTestMust(t, queue.Push(left, left, nil))
	astTestMust(t, queue.PopScopes(2))
	if queue.Len() != 1 {
		t.Fatal("disequation scope restore", queue.Len())
	}
	l, r, ls, rs, literals, dep, err := queue.At(0)
	astTestMust(t, err)
	if l != left || r != right || len(ls) != 1 || len(rs) != 1 || len(ls[0]) != 1 || ls[0][0] != left || len(rs[0]) != 1 || rs[0][0] != right || len(literals) != 0 || dep != dependency {
		t.Fatal("disequation restored value", l, r, ls, rs, literals, dep)
	}
	astTestMust(t, queue.Close())
}

func TestSMTSequenceDisequationReplacementSwapAndNestedRestore(t *testing.T) {
	m := astTestManager(t)
	var words []ast.ASTHandle
	for _, text := range []string{"first", "middle", "last"} {
		word, err := m.MkString(text)
		astTestMust(t, err)
		words = append(words, word)
	}
	queue, err := newSMTSequenceDisequationQueue(m)
	astTestMust(t, err)
	defer func() { astTestMust(t, queue.Close()) }()
	for _, word := range words {
		astTestMust(t, queue.Push(word, word, nil))
	}
	astTestMust(t, queue.PushScope())
	guards := []SMTLiteral{NewSMTLiteral(2, false)}
	astTestMust(t, queue.Set(0, words[0], words[1], [][]ast.ASTHandle{{words[0], words[1]}}, [][]ast.ASTHandle{{words[2]}}, guards, nil))
	guards[0] = SMTNullLiteral
	astTestMust(t, queue.PushScope())
	astTestMust(t, queue.EraseSwapBack(0))
	first, _, _, _, _, _, err := queue.At(0)
	astTestMust(t, err)
	if queue.Len() != 2 || first != words[2] {
		t.Fatal("source swap-with-last removal changed queue order")
	}
	astTestMust(t, queue.PopScopes(1))
	_, _, ls, rs, restored, _, err := queue.At(0)
	astTestMust(t, err)
	if queue.Len() != 3 || len(ls) != 1 || len(ls[0]) != 2 || ls[0][1] != words[1] || rs[0][0] != words[2] || restored[0] != NewSMTLiteral(2, false) {
		t.Fatal("replacement cases or copied guard did not survive nested scope restoration")
	}
	astTestMust(t, queue.PopScopes(1))
	for i, word := range words {
		left, right, ls, rs, guards, _, err := queue.At(i)
		astTestMust(t, err)
		if left != word || right != word || len(ls[0]) != 1 || ls[0][0] != word || rs[0][0] != word || len(guards) != 0 {
			t.Fatal("base disequation was not restored after replacement and removal", i)
		}
	}
}
