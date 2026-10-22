package smtcore

import "testing"

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
