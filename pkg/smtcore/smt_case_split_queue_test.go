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
	if selected != variable || phase != lbool.False {
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

func TestSMTActivityCaseSplitQueueHonorsRandomVariableFrequency(t *testing.T) {
	m := astTestManager(t)
	_, _, state := smtInternalizerFixture(t, m, nil)
	variables := make([]int32, 3)
	for index := range variables {
		expression := smtRelevancyBool(t, m, "random-variable-"+string(rune('a'+index)))
		astTestMust(t, state.Internalize(expression, true))
		variables[index], _ = state.BoolVar(expression)
	}
	state.activity[variables[0]] = 100
	params := SMTParamsState{InvDecay: 1.052, RandomInitialActivity: 0, RandomVarFreq: 1, PhaseSelection: 0}
	astTestMust(t, state.configureBooleanHeuristics(params))
	queue, err := NewSMTActivityCaseSplitQueue(state)
	astTestMust(t, err)
	astTestMust(t, queue.ConfigurePhases(params))
	selected, _, err := queue.NextCaseSplit()
	astTestMust(t, err)
	if selected != variables[2] {
		t.Fatal("random split did not override activity", selected, variables)
	}
}

func TestSMTActivityCaseSplitQueueDelaysSearchVariables(t *testing.T) {
	m := astTestManager(t)
	_, _, state := smtInternalizerFixture(t, m, nil)
	base := smtRelevancyBool(t, m, "delayed-variable-base")
	astTestMust(t, state.Internalize(base, true))
	baseVar, _ := state.BoolVar(base)
	params := SMTParamsState{InvDecay: 1.052, CaseSplitStrategy: 1, PhaseSelection: 0}
	astTestMust(t, state.configureBooleanHeuristics(params))
	queue, err := NewSMTActivityCaseSplitQueue(state)
	astTestMust(t, err)
	astTestMust(t, queue.ConfigurePhases(params))
	astTestMust(t, state.beginSearchPersistence(0, params))
	defer func() { astTestMust(t, state.endSearchPersistence()) }()
	astTestMust(t, state.PushScope())
	search := smtRelevancyBool(t, m, "delayed-variable-search")
	astTestMust(t, state.Internalize(search, true))
	searchVar, _ := state.BoolVar(search)
	state.activity[searchVar] = 100

	selected, _, err := queue.NextCaseSplit()
	astTestMust(t, err)
	if selected != baseVar {
		t.Fatal("search variable bypassed delayed pool", selected, baseVar)
	}
	astTestMust(t, state.AssignLiteral(NewSMTLiteral(baseVar, false), NewSMTBooleanAxiom()))
	selected, _, err = queue.NextCaseSplit()
	astTestMust(t, err)
	if selected != searchVar {
		t.Fatal("delayed pool was not promoted", selected, searchVar)
	}

	astTestMust(t, state.PopScopes(1))
	astTestMust(t, state.PushScope())
	replacement := smtRelevancyBool(t, m, "delayed-variable-replacement")
	astTestMust(t, state.Internalize(replacement, true))
	replacementVar, _ := state.BoolVar(replacement)
	state.activity[replacementVar] = 200
	selected, _, err = queue.NextCaseSplit()
	astTestMust(t, err)
	if selected != baseVar {
		t.Fatal("reused variable index bypassed delayed pool", selected, baseVar)
	}
	astTestMust(t, state.PopScopes(1))
}

func TestSMTActivityCaseSplitQueueCachesReinternalizedActivity(t *testing.T) {
	m := astTestManager(t)
	_, _, state := smtInternalizerFixture(t, m, nil)
	params := SMTParamsState{InvDecay: 1.052, CaseSplitStrategy: 2}
	astTestMust(t, state.configureBooleanHeuristics(params))
	expression := smtRelevancyBool(t, m, "cached-reinternalized-activity")
	astTestMust(t, state.PushScope())
	astTestMust(t, state.Internalize(expression, true))
	variable, _ := state.BoolVar(expression)
	state.activity[variable] = 17
	astTestMust(t, state.PopScopes(1))
	if _, err := m.GetKind(expression); err != nil {
		t.Fatal("activity cache did not retain expression", err)
	}
	astTestMust(t, state.Internalize(expression, true))
	variable, _ = state.BoolVar(expression)
	if state.activity[variable] != 17 {
		t.Fatal("reinternalized variable lost cached activity", state.activity[variable])
	}
}
