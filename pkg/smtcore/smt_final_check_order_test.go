package smtcore

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func smtFinalCheckTestTheory(t *testing.T, s *SMTInternalizerState) *smtInternalizerTestTheory {
	t.Helper()
	family := ast.FamilyID(71 + len(s.theoryOrder))
	base, err := NewSMTTheoryBase(family)
	astTestMust(t, err)
	theory := &smtInternalizerTestTheory{SMTTheoryBase: base}
	astTestMust(t, s.RegisterTheory(theory))
	return theory
}

func TestSMTFinalCheckQuantifierSlotResumesBeforeTheories(t *testing.T) {
	m, state, registry, manager := smtQuantifierManagerFixture(t, 10, 20, SMTQuantifierManagerConfig{QuickUnsat: true})
	theory := smtFinalCheckTestTheory(t, state)
	theory.finalStatus = SMTFinalCheckContinue
	quantifier, predicate, sort := smtManagedQuantifier(t, m, "final-check-order-quantifier")
	astTestMust(t, state.Internalize(quantifier, true))
	_, binding := smtEnodeConstant(t, m, 2610, sort, 1)
	astTestMust(t, state.context.RegisterEnode(binding, true))
	application, _ := smtQuickApplicationNode(t, m, state.context, predicate, binding, true)
	astTestMust(t, state.context.SetRelevant(quantifier, true))
	astTestMust(t, state.context.SetAssignment(quantifier, lbool.True))
	astTestMust(t, state.context.SetAssignment(application, lbool.False))
	status, err := state.FinalCheckTheories()
	astTestMust(t, err)
	if status != SMTFinalCheckContinue || theory.finalCheckCalls != 1 || state.finalCheckIndex != 1 || registry.schedule.Stats().Instances != 0 {
		t.Fatal("first continuing theory must leave quantifier slot next", status, state.finalCheckIndex, theory.finalCheckCalls)
	}
	theory.finalStatus = SMTFinalCheckDone
	status, err = state.FinalCheckTheories()
	astTestMust(t, err)
	if status != SMTFinalCheckContinue || theory.finalCheckCalls != 1 || state.finalCheckIndex != 0 || registry.schedule.Stats().Instances != 1 {
		t.Fatal("quantifier slot did not run before resuming theory", status, state.finalCheckIndex, theory.finalCheckCalls, registry.schedule.Stats())
	}
	if manager.acceptedModel != nil {
		t.Fatal("instantiation round published a candidate model")
	}
}

func TestSMTQuantifierFinalCheckDefersCandidateModelValidation(t *testing.T) {
	m, state, _, manager := smtQuantifierManagerFixture(t, 10, 20, SMTQuantifierManagerConfig{MBQI: true, MaxIterations: 1})
	calls := 0
	poison := errors.New("candidate model observed")
	manager.candidateModel = func() (*SMTModelGenerator, error) {
		calls++
		return nil, poison
	}
	status, err := manager.CheckModel()
	astTestMust(t, err)
	if status != SMTQuantifierModelSAT || calls != 0 {
		t.Fatal("empty quantifier registry should accept without a model", status, calls)
	}
	quantifier, _, _ := smtManagedQuantifier(t, m, "final-model-phase")
	astTestMust(t, state.Internalize(quantifier, true))
	astTestMust(t, state.context.SetAssignment(quantifier, lbool.True))
	final, err := manager.FinalCheck(true)
	astTestMust(t, err)
	if final != SMTFinalCheckDone || calls != 0 {
		t.Fatal("full instantiation check ran candidate-model validation", final, calls)
	}
	status, err = manager.CheckModel()
	if status != SMTQuantifierModelUnknown || !errors.Is(err, poison) || calls != 1 {
		t.Fatal("separate model phase did not run", status, err, calls)
	}
	manager.mbqi = false
	status, err = manager.CheckModel()
	astTestMust(t, err)
	if status != SMTQuantifierModelUnknown || calls != 1 {
		t.Fatal("disabled MBQI must report UNKNOWN without constructing a candidate", status, calls)
	}
}

type smtFinalCheckAssignTheory struct {
	*smtInternalizerTestTheory
	literal  SMTLiteral
	assigned bool
}

func (t *smtFinalCheckAssignTheory) FinalCheck() (SMTFinalCheckStatus, error) {
	if !t.assigned {
		if err := t.state.AssignLiteral(t.literal, NewSMTBooleanAxiom()); err != nil {
			return SMTFinalCheckGiveUp, err
		}
		t.assigned = true
	}
	return t.smtInternalizerTestTheory.FinalCheck()
}

func TestSMTFinalCheckPendingAssignmentOverridesGiveUp(t *testing.T) {
	m := astTestManager(t)
	_, _, state := smtInternalizerFixture(t, m, nil)
	family, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	base, err := NewSMTTheoryBase(family)
	astTestMust(t, err)
	a := smtRelevancyBool(t, m, "final-check-pending-a")
	astTestMust(t, state.Internalize(a, true))
	literal, err := state.Literal(a)
	astTestMust(t, err)
	theory := &smtFinalCheckAssignTheory{smtInternalizerTestTheory: &smtInternalizerTestTheory{SMTTheoryBase: base, finalStatus: SMTFinalCheckGiveUp}, literal: literal}
	astTestMust(t, state.RegisterTheory(theory))
	status, err := state.FinalCheckTheories()
	astTestMust(t, err)
	if status != SMTFinalCheckContinue || len(state.IncompleteTheories()) != 1 || !state.canPropagate() {
		t.Fatal("pending assignment was hidden by theory give-up", status, state.IncompleteTheories())
	}
	consistent, err := state.PropagateCore()
	astTestMust(t, err)
	if !consistent || state.canPropagate() {
		t.Fatal("pending assignment did not drain", consistent)
	}
	status, err = state.FinalCheckTheories()
	astTestMust(t, err)
	if status != SMTFinalCheckGiveUp {
		t.Fatal("give-up did not resume after propagation", status)
	}
}

func TestSMTSearchInitializationResetsFinalCheckRotationAndIncompleteTheories(t *testing.T) {
	m := astTestManager(t)
	_, _, state := smtInternalizerFixture(t, m, nil)
	first := smtFinalCheckTestTheory(t, state)
	second := smtFinalCheckTestTheory(t, state)
	first.finalStatus = SMTFinalCheckGiveUp
	second.finalStatus = SMTFinalCheckContinue
	status, err := state.FinalCheckTheories()
	astTestMust(t, err)
	if status != SMTFinalCheckContinue || state.finalCheckIndex != 2 || len(state.IncompleteTheories()) != 1 {
		t.Fatal("fixture did not leave a partial round", status, state.finalCheckIndex)
	}
	astTestMust(t, state.InitializeTheorySearch())
	if state.finalCheckIndex != 0 || len(state.IncompleteTheories()) != 0 {
		t.Fatal("new search reused previous final-check state", state.finalCheckIndex, state.IncompleteTheories())
	}
	first.finalStatus = SMTFinalCheckContinue
	secondCalls := second.finalCheckCalls
	status, err = state.FinalCheckTheories()
	astTestMust(t, err)
	if status != SMTFinalCheckContinue || first.finalCheckCalls != 2 || second.finalCheckCalls != secondCalls || state.finalCheckIndex != 1 {
		t.Fatal("new search did not start at first theory", status, first.finalCheckCalls, second.finalCheckCalls, state.finalCheckIndex)
	}
}

func TestSMTSearchDefersMBQIUntilAllTheoriesFinish(t *testing.T) {
	m, cfg := smtModelGeneratorFixture(t)
	params, err := cfg.Context.NewValue()
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, cfg.Context.Del(&params)) })
	key := "preprocess"
	astTestMust(t, cfg.Context.SetBoolName(&params, &key, false))
	s, err := NewSMTSolver(m, &params, "AUFLIA", SMTSolverConfig{Params: cfg.Context, ModelEvaluator: cfg})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, s.Close()) })
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	x, err := m.MkVar(0, boolean)
	astTestMust(t, err)
	body, err := m.MkIff(x, x)
	astTestMust(t, err)
	quantifier, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{boolean}, Names: []symbol.Symbol{astTestSymbol(t, m, "final-mbqi-x")}, Body: body})
	astTestMust(t, err)
	astTestMust(t, s.Assert(quantifier))
	theory := smtFinalCheckTestTheory(t, s.kernel.internalizer)
	theory.finalStatus = SMTFinalCheckGiveUp
	manager := s.kernel.internalizer.quantifierManager
	poison := errors.New("candidate-model phase entered")
	builds := 0
	manager.candidateModel = func() (*SMTModelGenerator, error) {
		builds++
		return nil, poison
	}
	status, err := s.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.Undef || s.kernel.lastFailure != SMTFailureTheory || builds != 0 {
		t.Fatal("incomplete theory must prevent candidate construction", status, s.kernel.lastFailure, builds)
	}
	theory.finalStatus = SMTFinalCheckDone
	status, err = s.CheckSat(nil)
	if status != lbool.Undef || !errors.Is(err, poison) || builds != 1 {
		t.Fatal("completed theory round must enter model validation", status, err, builds)
	}
}

type smtRestartConflictTheory struct{ *smtInternalizerTestTheory }

func (t *smtRestartConflictTheory) Restart() error {
	t.restartCalls++
	return t.state.AssignLiteral(SMTFalseLiteral, NewSMTBooleanAxiom())
}

func TestSMTTheoryRestartStopsAfterNewConflict(t *testing.T) {
	m, state, _, manager := smtQuantifierManagerFixture(t, 10, 20, SMTQuantifierManagerConfig{})
	base, err := NewSMTTheoryBase(71)
	astTestMust(t, err)
	first := &smtRestartConflictTheory{&smtInternalizerTestTheory{SMTTheoryBase: base}}
	astTestMust(t, state.RegisterTheory(first))
	second := smtFinalCheckTestTheory(t, state)
	falsity, err := m.MkFalse()
	astTestMust(t, err)
	ref, err := m.NewRef(falsity)
	astTestMust(t, err)
	manager.modelFinder.pendingConstraints = append(manager.modelFinder.pendingConstraints, ref)
	astTestMust(t, state.RestartTheories())
	if !state.Inconsistent() || first.restartCalls != 1 || second.restartCalls != 0 || len(manager.modelFinder.pendingConstraints) != 1 {
		t.Fatal("restart continued after a theory made context inconsistent", first.restartCalls, second.restartCalls, len(manager.modelFinder.pendingConstraints))
	}
}
