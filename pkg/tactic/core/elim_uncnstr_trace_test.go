package core

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestEliminateUnconstrainedExactTraceAndStreamRefusal(t *testing.T) {
	m, c, g, services := solveEqFixture(t)
	runner, err := NewEliminateUnconstrainedTactic(m, c, g, nil, services)
	simplifyMust(t, err)
	t.Cleanup(func() { simplifyMust(t, runner.Close()) })
	a, _, _ := propagateAtoms(t, m)
	term, err := m.MkNot(a)
	simplifyMust(t, err)
	simplifyPin(t, m, term)
	goal, err := tactic.NewGoal(m, false, false, false)
	simplifyMust(t, err)
	t.Cleanup(func() { simplifyMust(t, goal.Close()) })
	simplifyMust(t, goal.Assert(term))
	var before, trace bytes.Buffer
	simplifyMust(t, goal.Display(&before))
	tags := diagnostic.NewTraceTagsWithWriter(&trace)
	for _, tag := range []string{"elim_uncnstr", "elim_uncnstr_bug", "elim_uncnstr_bug_ll"} {
		tags.Enable(tag)
	}
	simplifyMust(t, runner.BindTrace(tags))
	var result tactic.TacticResult
	t.Cleanup(func() { simplifyMust(t, result.Close()) })
	simplifyMust(t, runner.Apply(goal, &result))
	if tags.ShouldTrace("elim_uncnstr") {
		record := func(tag, function string, line int, body string) string {
			return fmt.Sprintf("-------- [%s] %s src/tactic/core/elim_uncnstr_tactic.cpp:%d ---------\n%s------------------------------------------------\n", tag, function, line, body)
		}
		want := record("elim_uncnstr_bug", "operator()", 828, before.String()) +
			record("elim_uncnstr", "operator()", 838, "unconstrained variables...\na \n") +
			record("elim_uncnstr_bug", "mk_fresh_uncnstr_var_for", 99, "eliminating:\n(not a)\n") +
			record("elim_uncnstr_bug_ll", "mk_fresh_uncnstr_var_for", 100, "eliminating:\n(not a)\n") +
			record("elim_uncnstr", "operator()", 891, "")
		if trace.String() != want {
			t.Fatalf("source trace differs\nGo: %s\nwant: %s", trace.String(), want)
		}
	} else if trace.Len() != 0 {
		t.Fatal("erased traces wrote output")
	}
	refused := diagnostic.NewTraceTagsWithWriter(simplifyRefusedWriter{})
	refused.EnableAll(true)
	simplifyMust(t, runner.BindTrace(refused))
	simplifyMust(t, runner.Apply(goal, &result))
	size, err := goal.Size()
	simplifyMust(t, err)
	if size != 1 {
		t.Fatal("trace refusal changed the goal")
	}
}
