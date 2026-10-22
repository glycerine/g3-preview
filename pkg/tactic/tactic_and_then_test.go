// Source: src/tactic/tactical.cpp:104-271.
package tactic

import "testing"

type countingSkipTactic struct {
	*primitiveTactic
	applications uint32
}

type converterSkipTactic struct {
	*primitiveTactic
	converter ModelConverter
}

func (t *converterSkipTactic) Apply(g *Goal, out *TacticResult) error {
	if err := t.primitiveTactic.Apply(g, out); err != nil {
		return err
	}
	return out.SetModelConverter(t.converter)
}

func (t *countingSkipTactic) Apply(g *Goal, out *TacticResult) error {
	t.applications++
	return t.primitiveTactic.Apply(g, out)
}

func TestAndThenSingleGoalAndDecidedShortcut(t *testing.T) {
	m := astTestManager(t)
	first := &countingSkipTactic{primitiveTactic: newPrimitiveTactic(primitiveSkip, "", 0)}
	second := &countingSkipTactic{primitiveTactic: newPrimitiveTactic(primitiveSkip, "", 0)}
	sequence, err := NewAndThenTactic(first, second)
	astTestMust(t, err)
	tacticOwnForTest(t, sequence)

	nondecided := goalTestNew(t, m, false, false)
	p := goalTestAtom(t, m, "and-then-p")
	astTestMust(t, nondecided.Assert(p))
	out := &TacticResult{}
	astTestMust(t, sequence.Apply(nondecided, out))
	if first.applications != 1 || second.applications != 1 || len(out.Goals) != 1 || out.Goals[0] != nondecided {
		t.Fatal("single-goal sequence", first.applications, second.applications, len(out.Goals))
	}
	astTestMust(t, out.Close())

	decided, err := NewGoal(m, false, true, false)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, decided.Close()) })
	out = &TacticResult{}
	astTestMust(t, sequence.Apply(decided, out))
	if first.applications != 2 || second.applications != 1 || len(out.Goals) != 1 || out.Goals[0] != decided {
		t.Fatal("decided shortcut", first.applications, second.applications, len(out.Goals))
	}
	astTestMust(t, out.Close())
}

func TestAndThenNarySourceOrder(t *testing.T) {
	m := astTestManager(t)
	g := goalTestNew(t, m, false, false)
	astTestMust(t, g.Assert(goalTestAtom(t, m, "and-then-order")))
	children := []*countingSkipTactic{
		{primitiveTactic: newPrimitiveTactic(primitiveSkip, "", 0)},
		{primitiveTactic: newPrimitiveTactic(primitiveSkip, "", 0)},
		{primitiveTactic: newPrimitiveTactic(primitiveSkip, "", 0)},
	}
	sequence, err := NewAndThenTactic(children[0], children[1], children[2])
	astTestMust(t, err)
	tacticOwnForTest(t, sequence)
	out := &TacticResult{}
	astTestMust(t, sequence.Apply(g, out))
	for i, child := range children {
		if child.applications != 1 {
			t.Fatalf("child %d applications %d", i, child.applications)
		}
	}
	astTestMust(t, out.Close())
}

func TestAndThenComposesSingleGoalModelConverters(t *testing.T) {
	m := astTestManager(t)
	g := goalTestNew(t, m, false, false)
	astTestMust(t, g.Assert(goalTestAtom(t, m, "and-then-model")))
	calls := []string{}
	firstConverter := &modelConverterTestLeaf{name: "first", calls: &calls}
	secondConverter := &modelConverterTestLeaf{name: "second", calls: &calls}
	first := &converterSkipTactic{primitiveTactic: newPrimitiveTactic(primitiveSkip, "", 0), converter: firstConverter}
	second := &converterSkipTactic{primitiveTactic: newPrimitiveTactic(primitiveSkip, "", 0), converter: secondConverter}
	sequence, err := NewAndThenTactic(first, second)
	astTestMust(t, err)
	tacticOwnForTest(t, sequence)
	out := &TacticResult{}
	astTestMust(t, sequence.Apply(g, out))
	if _, ok := out.ModelConverter.(*tacticConcatModelConverter); !ok {
		t.Fatalf("converter composition %T", out.ModelConverter)
	}
	if firstConverter.refs != 1 || secondConverter.refs != 1 {
		t.Fatal("composite converter ownership", firstConverter.refs, secondConverter.refs)
	}
	astTestMust(t, out.Close())
	if !firstConverter.closed || !secondConverter.closed {
		t.Fatal("result did not release converter composition")
	}
}
