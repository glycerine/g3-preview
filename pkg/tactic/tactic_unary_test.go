package tactic

// Source: src/tactic/tactical.cpp:1077-1128.

import "testing"

type branchingTestTactic struct{ *primitiveTactic }

func (t *branchingTestTactic) Apply(g *Goal, out *TacticResult) error {
	if err := out.prepare(g.manager); err != nil {
		return err
	}
	if err := out.AddGoal(g); err != nil {
		return err
	}
	return out.AddGoal(g)
}

type cleanupTestTactic struct {
	*primitiveTactic
	cleanups uint32
}

func (t *cleanupTestTactic) Cleanup() error {
	t.cleanups++
	return nil
}

func TestFailIfBranchingAndCleanTacticals(t *testing.T) {
	m := astTestManager(t)
	g := goalTestNew(t, m, false, false)
	branch := &branchingTestTactic{newPrimitiveTactic(primitiveSkip, "", 0)}
	guard, err := NewFailIfBranchingTactic(branch, 1)
	astTestMust(t, err)
	tacticOwnForTest(t, guard)
	out := &TacticResult{}
	if err = guard.Apply(g, out); err == nil || err.Error() != "failed-if-branching tactical" {
		t.Fatal("branching guard", err)
	}
	if len(out.Goals) != 0 {
		t.Fatal("branching refusal retained results")
	}

	allowed, err := NewFailIfBranchingTactic(branch, 2)
	astTestMust(t, err)
	tacticOwnForTest(t, allowed)
	astTestMust(t, allowed.Apply(g, out))
	if len(out.Goals) != 2 {
		t.Fatal("allowed branches", len(out.Goals))
	}
	astTestMust(t, out.Close())

	child := &cleanupTestTactic{primitiveTactic: newPrimitiveTactic(primitiveSkip, "", 0)}
	clean, err := NewCleanTactic(child)
	astTestMust(t, err)
	tacticOwnForTest(t, clean)
	out = &TacticResult{}
	astTestMust(t, clean.Apply(g, out))
	if child.cleanups != 1 {
		t.Fatal("clean tactical did not clean after apply", child.cleanups)
	}
	astTestMust(t, out.Close())
}
