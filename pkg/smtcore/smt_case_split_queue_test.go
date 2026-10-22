package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSMTActivityCaseSplitQueueUsesActivityAssignmentAndPhase(t *testing.T) {
	m := astTestManager(t)
	_, _, state := smtInternalizerFixture(t, m, nil)
	a := smtRelevancyBool(t, m, "case-split-a")
	b := smtRelevancyBool(t, m, "case-split-b")
	astTestMust(t, state.Internalize(a, true))
	astTestMust(t, state.Internalize(b, true))
	av, found := state.BoolVar(a)
	if !found {
		t.Fatal("missing first variable")
	}
	bv, found := state.BoolVar(b)
	if !found {
		t.Fatal("missing second variable")
	}
	state.activity[bv] = 4
	state.booleanData[bv].SetTrueFirst()
	queue, err := NewSMTActivityCaseSplitQueue(state)
	astTestMust(t, err)
	variable, phase, err := queue.NextCaseSplit()
	astTestMust(t, err)
	if variable != bv || phase != lbool.True {
		t.Fatal("activity/phase selection", variable, phase, bv)
	}
	astTestMust(t, state.AssignLiteral(NewSMTLiteral(bv, false), NewSMTBooleanAxiom()))
	variable, phase, err = queue.NextCaseSplit()
	astTestMust(t, err)
	if variable != av || phase != lbool.False {
		t.Fatal("assigned variable was not skipped", variable, phase, av)
	}
}
