package smtcore

import "testing"

func TestSMTBooleanActivityIncrementDecayAndRescale(t *testing.T) {
	m := astTestManager(t)
	_, _, state := smtInternalizerFixture(t, m, nil)
	expression := smtRelevancyBool(t, m, "boolean-activity")
	astTestMust(t, state.Internalize(expression, true))
	variable, _ := state.BoolVar(expression)
	astTestMust(t, state.incrementBooleanActivity(variable))
	if state.activity[variable] != 1 {
		t.Fatal("initial activity increment", state.activity[variable])
	}
	state.decayBooleanActivity()
	astTestMust(t, state.incrementBooleanActivity(variable))
	if state.activity[variable] != 2.052 || state.activityIncrement != 1.052 {
		t.Fatal("decayed activity increment", state.activity[variable], state.activityIncrement)
	}
	state.activity[variable] = smtBooleanActivityLimit * 2
	state.activityIncrement = 2
	astTestMust(t, state.incrementBooleanActivity(variable))
	if state.activity[variable] != (smtBooleanActivityLimit*2+2)*smtBooleanInverseActivityLimit || state.activityIncrement != 2*smtBooleanInverseActivityLimit {
		t.Fatal("activity rescale", state.activity[variable], state.activityIncrement)
	}
}

func TestSMTBooleanRandomInitialActivityModes(t *testing.T) {
	m := astTestManager(t)
	_, _, state := smtInternalizerFixture(t, m, nil)
	params := SMTParamsState{InvDecay: 1.052, RandomInitialActivity: 2}
	astTestMust(t, state.configureBooleanHeuristics(params))
	first := smtRelevancyBool(t, m, "random-activity-always")
	astTestMust(t, state.Internalize(first, true))
	firstVar, _ := state.BoolVar(first)
	if state.activity[firstVar] != -0.038 {
		t.Fatal("always-random initial activity", state.activity[firstVar])
	}

	params.RandomInitialActivity = 1
	astTestMust(t, state.configureBooleanHeuristics(params))
	second := smtRelevancyBool(t, m, "random-activity-idle")
	astTestMust(t, state.Internalize(second, true))
	secondVar, _ := state.BoolVar(second)
	if state.activity[secondVar] != 0 {
		t.Fatal("search-only activity randomized while idle", state.activity[secondVar])
	}
	astTestMust(t, state.beginSearchPersistence(0, params))
	third := smtRelevancyBool(t, m, "random-activity-search")
	astTestMust(t, state.Internalize(third, true))
	thirdVar, _ := state.BoolVar(third)
	if state.activity[thirdVar] != -0.038 {
		t.Fatal("search activity was not randomized", state.activity[thirdVar])
	}
	astTestMust(t, state.endSearchPersistence())
}
