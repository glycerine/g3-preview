package tactic

// Source: src/tactic/tactical.cpp:366-459,1385-1387.

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

type mutatingFailureTactic struct {
	*primitiveTactic
	formula ast.ASTHandle
}

func (t *mutatingFailureTactic) Apply(g *Goal, _ *TacticResult) error {
	if err := g.Assert(t.formula); err != nil {
		return err
	}
	return NewTacticException("mutated and failed")
}

func TestOrElseRestoresGoalAfterTacticFailure(t *testing.T) {
	m := astTestManager(t)
	p := goalTestAtom(t, m, "or-else-p")
	q := goalTestAtom(t, m, "or-else-q")
	g := goalTestNew(t, m, false, false)
	astTestMust(t, g.Assert(p))
	failing := &mutatingFailureTactic{primitiveTactic: newPrimitiveTactic(primitiveSkip, "", 0), formula: q}
	choice, err := NewOrElseTactic(failing, NewSkipTactic())
	astTestMust(t, err)
	tacticOwnForTest(t, choice)
	out := &TacticResult{}
	astTestMust(t, choice.Apply(g, out))
	if len(out.Goals) != 1 || out.Goals[0] != g {
		t.Fatal("fallback did not return restored input")
	}
	goalTestForms(t, g, p)
	astTestMust(t, out.Close())
	if failing.references != 1 || failing.closed {
		t.Fatal("choice did not retain failing child", failing.references, failing.closed)
	}
}

func TestOrElseLastFailureAndSkipIfFailed(t *testing.T) {
	m := astTestManager(t)
	g := goalTestNew(t, m, false, false)
	choice, err := NewOrElseTactic(NewFailTactic(), NewFailTactic())
	astTestMust(t, err)
	tacticOwnForTest(t, choice)
	if err = choice.Apply(g, &TacticResult{}); err == nil || err.Error() != "fail tactic" {
		t.Fatal("last failure", err)
	}

	fallback, err := NewSkipIfFailedTactic(NewFailTactic())
	astTestMust(t, err)
	tacticOwnForTest(t, fallback)
	out := &TacticResult{}
	astTestMust(t, fallback.Apply(g, out))
	if len(out.Goals) != 1 || out.Goals[0] != g {
		t.Fatal("skip-if-failed did not return input")
	}
	astTestMust(t, out.Close())
}
