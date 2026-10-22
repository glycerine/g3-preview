package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
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

func TestSMTActivityCaseSplitQueueHonorsConfiguredAndCachedPhases(t *testing.T) {
	m := astTestManager(t)
	_, _, state := smtInternalizerFixture(t, m, nil)
	astTestMust(t, state.SetBinaryClauseOptimization(true))
	a := smtRelevancyBool(t, m, "case-split-configured-a")
	astTestMust(t, state.Internalize(a, true))
	variable, found := state.BoolVar(a)
	if !found {
		t.Fatal("missing configured-phase variable")
	}
	queue, err := NewSMTActivityCaseSplitQueue(state)
	astTestMust(t, err)
	config := SMTParamsState{PhaseSelection: 1, PhaseCachingOn: 400, PhaseCachingOff: 100}
	astTestMust(t, queue.ConfigurePhases(config))
	selected, phase, err := queue.NextCaseSplit()
	astTestMust(t, err)
	if selected != variable || phase != lbool.True {
		t.Fatal("always-true phase", selected, phase)
	}

	config.PhaseSelection = 0
	astTestMust(t, queue.ConfigurePhases(config))
	selected, phase, err = queue.NextCaseSplit()
	astTestMust(t, err)
	if selected != variable || phase != lbool.False {
		t.Fatal("always-false phase", selected, phase)
	}

	astTestMust(t, state.PushScope())
	astTestMust(t, state.AssignLiteral(NewSMTLiteral(variable, false), NewSMTBooleanAxiom()))
	astTestMust(t, state.PopScopes(1))
	if !state.booleanData[variable].PhaseAvailable || !state.booleanData[variable].Phase {
		t.Fatal("assignment phase was not cached across rollback")
	}
	config.PhaseSelection = 2
	astTestMust(t, queue.ConfigurePhases(config))
	selected, phase, err = queue.NextCaseSplit()
	astTestMust(t, err)
	if selected != variable || phase != lbool.True {
		t.Fatal("cached phase", selected, phase)
	}

	config.PhaseSelection = 5
	config.RandomSeed = 0
	astTestMust(t, queue.ConfigurePhases(config))
	selected, phase, err = queue.NextCaseSplit()
	astTestMust(t, err)
	if selected != variable || phase != lbool.True {
		t.Fatal("source-random phase", selected, phase)
	}

	b := smtRelevancyBool(t, m, "case-split-configured-b")
	astTestMust(t, state.Internalize(b, true))
	bVariable, found := state.BoolVar(b)
	if !found {
		t.Fatal("missing occurrence-phase peer")
	}
	state.activity[variable] = 2
	positive := NewSMTLiteral(variable, false)
	peer := NewSMTLiteral(bVariable, false)
	astTestMust(t, state.addClause([]SMTLiteral{positive, peer}, false, ast.ASTHandle{}))
	astTestMust(t, state.addClause([]SMTLiteral{positive, peer.Negated()}, false, ast.ASTHandle{}))
	config.PhaseSelection = 6
	astTestMust(t, queue.ConfigurePhases(config))
	selected, phase, err = queue.NextCaseSplit()
	astTestMust(t, err)
	if selected != variable || phase != lbool.True {
		t.Fatal("occurrence phase", selected, phase)
	}

	config.PhaseSelection = 7
	if err = queue.ConfigurePhases(config); err != ast.ErrASTDomain {
		t.Fatal("invalid phase strategy", err)
	}
}

func TestSMTActivityCaseSplitQueueForgetsConservativeConflictPhases(t *testing.T) {
	m := astTestManager(t)
	_, _, state := smtInternalizerFixture(t, m, nil)
	expression := smtRelevancyBool(t, m, "case-split-conservative")
	astTestMust(t, state.Internalize(expression, true))
	variable, _ := state.BoolVar(expression)
	queue, err := NewSMTActivityCaseSplitQueue(state)
	astTestMust(t, err)
	config := SMTParamsState{PhaseSelection: 3, PhaseCachingOn: 400, PhaseCachingOff: 100}
	astTestMust(t, queue.ConfigurePhases(config))
	astTestMust(t, state.PushScope())
	astTestMust(t, state.AssignLiteral(NewSMTLiteral(variable, false), NewSMTBooleanAxiom()))
	if !state.booleanData[variable].PhaseAvailable {
		t.Fatal("assignment did not cache its phase")
	}
	astTestMust(t, queue.PrepareConflict())
	if state.booleanData[variable].PhaseAvailable {
		t.Fatal("conservative conflict retained current-level phase")
	}

	state.booleanData[variable].PhaseAvailable = true
	config.PhaseSelection = 2
	astTestMust(t, queue.ConfigurePhases(config))
	astTestMust(t, queue.PrepareConflict())
	if !state.booleanData[variable].PhaseAvailable {
		t.Fatal("ordinary caching discarded current-level phase")
	}
}
