// Source: src/tactic/tactical.cpp:462-884; src/tactic/tactical.h:55-61.
package tactic

import "testing"

func TestParallelCombinatorsUseSingleGoroutineSourceBranches(t *testing.T) {
	m := astTestManager(t)
	g := goalTestNew(t, m, false, false)
	astTestMust(t, g.Assert(goalTestAtom(t, m, "parallel-fallback")))
	fail := NewFailTactic()
	first := &countingSkipTactic{primitiveTactic: newPrimitiveTactic(primitiveSkip, "", 0)}
	second := &countingSkipTactic{primitiveTactic: newPrimitiveTactic(primitiveSkip, "", 0)}
	choice, err := NewParTactic(fail, first)
	astTestMust(t, err)
	tacticOwnForTest(t, choice)
	out := &TacticResult{}
	astTestMust(t, choice.Apply(g, out))
	if first.applications != 1 || len(out.Goals) != 1 {
		t.Fatal("parallel fallback branch", first.applications, len(out.Goals))
	}
	astTestMust(t, out.Close())

	sequence, err := NewParAndThenTactic(first, second)
	astTestMust(t, err)
	tacticOwnForTest(t, sequence)
	out = &TacticResult{}
	astTestMust(t, sequence.Apply(g, out))
	if first.applications != 2 || second.applications != 1 || len(out.Goals) != 1 {
		t.Fatal("parallel and-then branch", first.applications, second.applications, len(out.Goals))
	}
	astTestMust(t, out.Close())
}
