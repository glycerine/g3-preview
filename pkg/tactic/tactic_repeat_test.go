package tactic

// Source: src/tactic/tactical.cpp:927-1075.

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

type rewriteSequenceTactic struct {
	*primitiveTactic
	from, middle, final ast.ASTHandle
	applications        uint32
	converters          []ModelConverter
}

func (t *rewriteSequenceTactic) Apply(g *Goal, out *TacticResult) error {
	t.applications++
	current, err := g.Form(0)
	if err != nil {
		return err
	}
	switch current {
	case t.from:
		err = g.Update(0, t.middle, ast.ASTHandle{}, ast.ASTDependency{})
	case t.middle:
		err = g.Update(0, t.final, ast.ASTHandle{}, ast.ASTDependency{})
	}
	if err != nil {
		return err
	}
	if err = t.primitiveTactic.Apply(g, out); err != nil {
		return err
	}
	if int(t.applications) <= len(t.converters) {
		return out.SetModelConverter(t.converters[t.applications-1])
	}
	return nil
}

func TestRepeatComposesModelConvertersInApplicationOrder(t *testing.T) {
	m := astTestManager(t)
	p := goalTestAtom(t, m, "repeat-model-p")
	q := goalTestAtom(t, m, "repeat-model-q")
	r := goalTestAtom(t, m, "repeat-model-r")
	g := goalTestNew(t, m, false, false)
	astTestMust(t, g.Assert(p))
	calls := []string{}
	converters := []ModelConverter{
		&modelConverterTestLeaf{name: "first", calls: &calls},
		&modelConverterTestLeaf{name: "second", calls: &calls},
		&modelConverterTestLeaf{name: "fixpoint", calls: &calls},
	}
	child := &rewriteSequenceTactic{
		primitiveTactic: newPrimitiveTactic(primitiveSkip, "", 0),
		from:            p,
		middle:          q,
		final:           r,
		converters:      converters,
	}
	repeat, err := NewRepeatTactic(child, ^uint32(0))
	astTestMust(t, err)
	tacticOwnForTest(t, repeat)
	out := &TacticResult{}
	astTestMust(t, repeat.Apply(g, out))
	ref := &ModelRef{}
	astTestMust(t, out.ModelConverter.ApplyModel(ref, 7))
	if len(calls) != 3 || calls[0] != "model:first:7" || calls[1] != "model:second:0" || calls[2] != "model:fixpoint:0" {
		t.Fatal("repeat converter order", calls)
	}
	astTestMust(t, ref.Close())
	astTestMust(t, out.Close())
}

func TestRepeatStopsAtGoalFixpointAndMaximumDepth(t *testing.T) {
	m := astTestManager(t)
	p := goalTestAtom(t, m, "repeat-p")
	q := goalTestAtom(t, m, "repeat-q")
	r := goalTestAtom(t, m, "repeat-r")
	g := goalTestNew(t, m, false, false)
	astTestMust(t, g.Assert(p))
	child := &rewriteSequenceTactic{primitiveTactic: newPrimitiveTactic(primitiveSkip, "", 0), from: p, middle: q, final: r}
	repeat, err := NewRepeatTactic(child, ^uint32(0))
	astTestMust(t, err)
	tacticOwnForTest(t, repeat)
	out := &TacticResult{}
	astTestMust(t, repeat.Apply(g, out))
	if child.applications != 3 {
		t.Fatal("fixpoint application count", child.applications)
	}
	goalTestForms(t, g, r)
	astTestMust(t, out.Close())

	limitedGoal := goalTestNew(t, m, false, false)
	astTestMust(t, limitedGoal.Assert(p))
	limitedChild := &rewriteSequenceTactic{primitiveTactic: newPrimitiveTactic(primitiveSkip, "", 0), from: p, middle: q, final: r}
	limited, err := NewRepeatTactic(limitedChild, 0)
	astTestMust(t, err)
	tacticOwnForTest(t, limited)
	out = &TacticResult{}
	astTestMust(t, limited.Apply(limitedGoal, out))
	if limitedChild.applications != 1 {
		t.Fatal("maximum depth application count", limitedChild.applications)
	}
	goalTestForms(t, limitedGoal, q)
	astTestMust(t, out.Close())
}

type repeatBranchTactic struct {
	*primitiveTactic
	trigger, left, right ast.ASTHandle
	applications         uint32
}

func (t *repeatBranchTactic) Apply(g *Goal, out *TacticResult) error {
	t.applications++
	current, err := g.Form(0)
	if err != nil {
		return err
	}
	if current != t.trigger {
		return t.primitiveTactic.Apply(g, out)
	}
	if err = g.Update(0, t.left, ast.ASTHandle{}, ast.ASTDependency{}); err != nil {
		return err
	}
	if err = out.prepare(g.manager); err != nil {
		return err
	}
	leftGoal, err := NewGoalConfigCopy(g)
	if err != nil {
		return err
	}
	if err = g.CopyTo(leftGoal); err != nil {
		_ = leftGoal.Close()
		return err
	}
	rightGoal, err := NewGoalConfigCopy(g)
	if err != nil {
		_ = leftGoal.Close()
		return err
	}
	if err = g.CopyTo(rightGoal); err != nil {
		_ = rightGoal.Close()
		_ = leftGoal.Close()
		return err
	}
	if err = rightGoal.Update(0, t.right, ast.ASTHandle{}, ast.ASTDependency{}); err != nil {
		_ = rightGoal.Close()
		_ = leftGoal.Close()
		return err
	}
	if err = out.TakeGoal(leftGoal); err != nil {
		_ = rightGoal.Close()
		_ = leftGoal.Close()
		return err
	}
	if err = out.TakeGoal(rightGoal); err != nil {
		_ = rightGoal.Close()
		return err
	}
	return nil
}

func TestRepeatRecursesAcrossProducedSubgoals(t *testing.T) {
	m := astTestManager(t)
	p := goalTestAtom(t, m, "repeat-branch-p")
	q := goalTestAtom(t, m, "repeat-branch-q")
	r := goalTestAtom(t, m, "repeat-branch-r")
	g := goalTestNew(t, m, false, false)
	astTestMust(t, g.Assert(p))
	child := &repeatBranchTactic{primitiveTactic: newPrimitiveTactic(primitiveSkip, "", 0), trigger: p, left: q, right: r}
	repeat, err := NewRepeatTactic(child, 4)
	astTestMust(t, err)
	tacticOwnForTest(t, repeat)
	out := &TacticResult{}
	astTestMust(t, repeat.Apply(g, out))
	if child.applications != 3 || len(out.Goals) != 2 {
		t.Fatal("repeat fanout", child.applications, len(out.Goals))
	}
	first, err := out.Goals[0].Form(0)
	astTestMust(t, err)
	second, err := out.Goals[1].Form(0)
	astTestMust(t, err)
	if first != q || second != r {
		t.Fatal("repeat branch formulas", first, second)
	}
	astTestMust(t, out.Close())
}
