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
