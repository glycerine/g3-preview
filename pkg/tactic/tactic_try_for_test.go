// Source: src/tactic/tactical.cpp:1130-1156.
package tactic

import (
	"runtime"
	"testing"
	"time"

	"github.com/glycerine/g3/pkg/util/messages"
)

type cancellationPollingTactic struct {
	*primitiveTactic
	canceled bool
}

func (t *cancellationPollingTactic) Apply(g *Goal, out *TacticResult) error {
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		canceled, err := g.manager.Canceled()
		if err != nil {
			return err
		}
		if canceled {
			t.canceled = true
			return NewTacticException(messages.Canceled)
		}
		runtime.Gosched()
	}
	return t.primitiveTactic.Apply(g, out)
}

func TestTryForCancelsChildAndRestoresResourceLimit(t *testing.T) {
	m := astTestManager(t)
	g := goalTestNew(t, m, false, false)
	astTestMust(t, g.Assert(goalTestAtom(t, m, "try-for-p")))
	child := &cancellationPollingTactic{primitiveTactic: newPrimitiveTactic(primitiveSkip, "", 0)}
	limited, err := NewTryForTactic(child, 1)
	astTestMust(t, err)
	tacticOwnForTest(t, limited)
	out := &TacticResult{}
	if err = limited.Apply(g, out); err == nil || err.Error() != messages.Canceled {
		t.Fatal("timeout result", err)
	}
	if !child.canceled || m.ResourceLimit().CancelFlag() {
		t.Fatal("timeout cancellation lifetime", child.canceled, m.ResourceLimit().CancelCount())
	}
	astTestMust(t, out.Close())
}

func TestTryForDisabledIntervalsApplyNormally(t *testing.T) {
	m := astTestManager(t)
	g := goalTestNew(t, m, false, false)
	astTestMust(t, g.Assert(goalTestAtom(t, m, "try-for-disabled")))
	for _, interval := range []uint32{0, ^uint32(0)} {
		child := &countingSkipTactic{primitiveTactic: newPrimitiveTactic(primitiveSkip, "", 0)}
		limited, err := NewTryForTactic(child, interval)
		astTestMust(t, err)
		tacticOwnForTest(t, limited)
		out := &TacticResult{}
		astTestMust(t, limited.Apply(g, out))
		if child.applications != 1 || m.ResourceLimit().CancelFlag() {
			t.Fatal("disabled try-for", interval, child.applications)
		}
		astTestMust(t, out.Close())
	}
}
