package smtcore

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSMTTheoryAwareSelectionConsumesQueueEntry(t *testing.T) {
	m := astTestManager(t)
	_, _, state := smtInternalizerFixture(t, m, nil)
	a := smtRelevancyBool(t, m, "aware-consume-a")
	b := smtRelevancyBool(t, m, "aware-consume-b")
	astTestMust(t, state.Internalize(a, true))
	astTestMust(t, state.Internalize(b, true))
	av, _ := state.BoolVar(a)
	bv, _ := state.BoolVar(b)
	state.activity[av], state.activity[bv] = 2, 1
	params := SMTParamsState{InvDecay: 1.052, CaseSplitStrategy: 6, PhaseSelection: 0}
	astTestMust(t, state.configureBooleanHeuristics(params))
	queue, err := NewSMTActivityCaseSplitQueue(state)
	astTestMust(t, err)
	astTestMust(t, queue.ConfigurePhases(params))
	first, _, err := queue.NextCaseSplit()
	astTestMust(t, err)
	second, _, err := queue.NextCaseSplit()
	astTestMust(t, err)
	if first != av || second != bv {
		t.Fatal("strategy 6 did not consume the selected queue entry", first, second, av, bv)
	}
}

func TestSMTTheoryAwarePriorityActivityPhaseAndUnassign(t *testing.T) {
	m := astTestManager(t)
	_, _, state := smtInternalizerFixture(t, m, nil)
	var variables []int32
	for _, name := range []string{"aware-a", "aware-b", "aware-c"} {
		expression := smtRelevancyBool(t, m, name)
		astTestMust(t, state.Internalize(expression, true))
		variable, _ := state.BoolVar(expression)
		variables = append(variables, variable)
	}
	a, b, c := variables[0], variables[1], variables[2]
	state.activity[a], state.activity[b] = 2, 3
	params := SMTParamsState{InvDecay: 1.052, CaseSplitStrategy: 6, PhaseSelection: 0}
	astTestMust(t, state.configureBooleanHeuristics(params))
	queue, err := NewSMTActivityCaseSplitQueue(state)
	astTestMust(t, err)
	astTestMust(t, queue.ConfigurePhases(params))
	astTestMust(t, state.AddTheoryAwareBranchingInfo(a, 2, lbool.True))
	astTestMust(t, state.PushScope())
	selected, phase, err := queue.NextCaseSplit()
	astTestMust(t, err)
	if selected != a || phase != lbool.False {
		t.Fatal("priority+activity score or native ignored phase hint changed", selected, phase)
	}
	astTestMust(t, state.AssignLiteral(NewSMTLiteral(a, false), NewSMTBooleanAxiom()))
	state.activityIncrement = 5
	astTestMust(t, state.incrementBooleanActivity(c))
	astTestMust(t, state.AddTheoryAwareBranchingInfo(b, -2, lbool.True))
	selected, _, err = queue.NextCaseSplit()
	astTestMust(t, err)
	if selected != c {
		t.Fatal("activity-increased event did not update the native heap", selected)
	}
	astTestMust(t, state.AssignLiteral(NewSMTLiteral(c, true), NewSMTBooleanAxiom()))
	// Updating an extracted variable does not insert it until unassignment.
	astTestMust(t, state.AddTheoryAwareBranchingInfo(a, 100, lbool.True))
	selected, _, err = queue.NextCaseSplit()
	astTestMust(t, err)
	if selected != b {
		t.Fatal("priority update reinserted an extracted variable", selected)
	}
	astTestMust(t, state.AssignLiteral(NewSMTLiteral(b, false), NewSMTBooleanAxiom()))
	selected, _, err = queue.NextCaseSplit()
	astTestMust(t, err)
	if selected != SMTNullBoolVar {
		t.Fatal("assigned/extracted variables remained selectable", selected)
	}
	astTestMust(t, state.PopScopes(1))
	selected, _, err = queue.NextCaseSplit()
	astTestMust(t, err)
	if selected != a {
		t.Fatal("unassignment failed to reinsert the variable with its persistent priority", selected)
	}
}

func TestSMTTheoryAwareDeletedSlotPriorityAndRandomSelection(t *testing.T) {
	m := astTestManager(t)
	_, _, state := smtInternalizerFixture(t, m, nil)
	params := SMTParamsState{InvDecay: 1.052, CaseSplitStrategy: 6, PhaseSelection: 0}
	astTestMust(t, state.configureBooleanHeuristics(params))
	queue, err := NewSMTActivityCaseSplitQueue(state)
	astTestMust(t, err)
	astTestMust(t, queue.ConfigurePhases(params))
	astTestMust(t, state.PushScope())
	child := smtRelevancyBool(t, m, "aware-deleted-child")
	astTestMust(t, state.Internalize(child, true))
	old, _ := state.BoolVar(child)
	astTestMust(t, state.AddTheoryAwareBranchingInfo(old, 20, lbool.Undef))
	astTestMust(t, state.PopScopes(1))
	replacement := smtRelevancyBool(t, m, "aware-replacement")
	astTestMust(t, state.Internalize(replacement, true))
	newVariable, _ := state.BoolVar(replacement)
	if newVariable != old {
		t.Fatal("fixture did not reuse the deleted Boolean slot")
	}
	selected, _, err := queue.NextCaseSplit()
	astTestMust(t, err)
	if selected != newVariable {
		t.Fatal("new variable was not inserted into the retained queue")
	}
	if state.theoryAwareBranching.priorities[newVariable] != 20 {
		t.Fatal("source del_var_eh priority persistence was lost")
	}
	// Separate context: seed zero chooses variable 3 from four slots.
	m = astTestManager(t)
	_, _, state = smtInternalizerFixture(t, m, nil)
	var variables []int32
	for _, name := range []string{"aware-random-a", "aware-random-b", "aware-random-c"} {
		expression := smtRelevancyBool(t, m, name)
		astTestMust(t, state.Internalize(expression, true))
		variable, _ := state.BoolVar(expression)
		variables = append(variables, variable)
	}
	state.activity[variables[0]] = 100
	params.RandomVarFreq = 1
	astTestMust(t, state.configureBooleanHeuristics(params))
	queue, err = NewSMTActivityCaseSplitQueue(state)
	astTestMust(t, err)
	astTestMust(t, queue.ConfigurePhases(params))
	selected, _, err = queue.NextCaseSplit()
	astTestMust(t, err)
	if selected != variables[2] {
		t.Fatal("native random choice did not override priority/activity", selected)
	}
	astTestMust(t, state.AddTheoryAwareBranchingInfo(selected, 200, lbool.Undef))
	params.RandomVarFreq = 0
	astTestMust(t, queue.ConfigurePhases(params))
	next, _, err := queue.NextCaseSplit()
	astTestMust(t, err)
	if next != selected {
		t.Fatal("random selection incorrectly removed its heap entry", next, selected)
	}
}

func TestSMTTheoryAwareInfoIsNoopForOtherStrategies(t *testing.T) {
	m := astTestManager(t)
	_, _, state := smtInternalizerFixture(t, m, nil)
	a := smtRelevancyBool(t, m, "unaware-a")
	astTestMust(t, state.Internalize(a, true))
	variable, _ := state.BoolVar(a)
	astTestMust(t, state.AddTheoryAwareBranchingInfo(variable, 100, lbool.True))
	if state.theoryAwareBranching != nil {
		t.Fatal("an inactive callback allocated the theory-aware queue")
	}
}

func TestSMTTheoryAwareQueueDisplayRetainsNativeOrder(t *testing.T) {
	m := astTestManager(t)
	_, _, state := smtInternalizerFixture(t, m, nil)
	a := smtRelevancyBool(t, m, "aware-display-a")
	b := smtRelevancyBool(t, m, "aware-display-b")
	astTestMust(t, state.Internalize(a, true))
	astTestMust(t, state.Internalize(b, true))
	av, _ := state.BoolVar(a)
	bv, _ := state.BoolVar(b)
	state.activity[av], state.activity[bv] = 2, 1
	params := SMTParamsState{InvDecay: 1.052, CaseSplitStrategy: 6, PhaseSelection: 0}
	astTestMust(t, state.configureBooleanHeuristics(params))
	queue, err := NewSMTActivityCaseSplitQueue(state)
	astTestMust(t, err)
	astTestMust(t, queue.ConfigurePhases(params))
	aid, err := m.GetASTID(a)
	astTestMust(t, err)
	bid, err := m.GetASTID(b)
	astTestMust(t, err)
	var out bytes.Buffer
	astTestMust(t, state.theoryAwareBranching.display(&out))
	if out.String() != fmt.Sprintf("remaining case-splits:\n#%d #%d \n", aid, bid) {
		t.Fatal("native queue display order/header changed", out.String())
	}
	selected, _, err := queue.NextCaseSplit()
	astTestMust(t, err)
	if selected != av {
		t.Fatal("display consumed or reordered the queue")
	}
	out.Reset()
	astTestMust(t, state.theoryAwareBranching.display(&out))
	if out.String() != fmt.Sprintf("remaining case-splits:\n#%d \n", bid) {
		t.Fatal("display did not omit an extracted entry", out.String())
	}
	_, _, err = queue.NextCaseSplit()
	astTestMust(t, err)
	out.Reset()
	astTestMust(t, state.theoryAwareBranching.display(&out))
	if out.Len() != 0 {
		t.Fatal("empty display emitted a header or newline", out.String())
	}
}

func TestSMTTheoryAwareAvailabilityDoesNotConsumeChoice(t *testing.T) {
	m := astTestManager(t)
	_, _, state := smtInternalizerFixture(t, m, nil)
	for _, name := range []string{"aware-peek-a", "aware-peek-b"} {
		astTestMust(t, state.Internalize(smtRelevancyBool(t, m, name), true))
	}
	params := SMTParamsState{InvDecay: 1.052, CaseSplitStrategy: 6, PhaseSelection: 0}
	astTestMust(t, state.configureBooleanHeuristics(params))
	queue, err := NewSMTActivityCaseSplitQueue(state)
	astTestMust(t, err)
	astTestMust(t, queue.ConfigurePhases(params))
	for i := 0; i < 2; i++ {
		for probe := 0; probe < 3; probe++ {
			available, err := queue.hasCaseSplit()
			astTestMust(t, err)
			if !available {
				t.Fatal("availability probe consumed a pending decision")
			}
		}
		selected, _, err := queue.NextCaseSplit()
		astTestMust(t, err)
		if selected == SMTNullBoolVar {
			t.Fatal("probe left no decision for the search loop")
		}
	}
	if available, err := queue.hasCaseSplit(); err != nil || available {
		t.Fatal("extracted choices remained available", available, err)
	}
}
