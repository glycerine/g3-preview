package smtcore

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	protomodel "github.com/glycerine/g3/pkg/smtcore/proto_model"
	"github.com/glycerine/g3/pkg/util/lbool"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestSMTModelCheckerBooleanSkolemsAvoidCartesianEnumeration(t *testing.T) {
	m, config := smtModelGeneratorFixture(t)
	s, err := NewSMTSolver(m, nil, "AUFLIA", SMTSolverConfig{Params: config.Context, ModelEvaluator: config})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, s.Close()) })
	status, err := s.CheckSatCore(nil)
	astTestMust(t, err)
	if status != lbool.True {
		t.Fatal("source solver setup", status)
	}
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	const count = 22
	sorts := make([]ast.ASTHandle, count)
	names := make([]symbol.Symbol, count)
	terms := make([]ast.ASTHandle, count)
	for i := range terms {
		sorts[i] = boolean
		names[i] = astTestSymbol(t, m, fmt.Sprint("skolem-bool-", i))
		variable, err := m.MkVar(uint32(i), boolean)
		astTestMust(t, err)
		terms[i], err = m.MkNot(variable)
		astTestMust(t, err)
	}
	body, err := m.MkOrArgs(terms)
	astTestMust(t, err)
	q, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: sorts, Names: names, Body: body})
	astTestMust(t, err)
	state := s.kernel.internalizer
	astTestMust(t, state.Internalize(q, true))
	proto, err := protomodel.NewProtoModel(m, config)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, proto.Close()) })
	checker := state.quantifierManager.modelChecker
	astTestMust(t, state.quantifierManager.modelFinder.FixModel(proto))
	astTestMust(t, m.Limit.Push(50000))
	defer m.Limit.Pop()
	satisfied, err := checker.Check(proto, []ast.ASTHandle{q})
	astTestMust(t, err)
	if satisfied || checker.Exhausted() || len(checker.newInstances) != 1 {
		t.Fatal("bounded Boolean counterexample", satisfied, checker.Exhausted(), len(checker.newInstances))
	}
	for _, value := range checker.newInstances[0].bindings {
		if !m.IsTrue(value) {
			t.Fatal("counterexample must set every Boolean true", value)
		}
	}
	if checker.auxiliaryContext == nil || checker.auxiliaryContext.kernel.Statistics().NumChecks == 0 {
		t.Fatal("Boolean counterexample bypassed auxiliary SMT")
	}
}

func TestSMTModelCheckerMissingInverseUsesOneCompleteCounterexample(t *testing.T) {
	for _, unknownRestricted := range []bool{false, true} {
		t.Run(fmt.Sprint(unknownRestricted), func(t *testing.T) {
			m, config := smtModelGeneratorFixture(t)
			s, err := NewSMTSolver(m, nil, "AUFLIA", SMTSolverConfig{Params: config.Context, ModelEvaluator: config})
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, s.Close()) })
			boolean, err := m.MkBoolSort()
			astTestMust(t, err)
			x, err := m.MkVar(1, boolean)
			astTestMust(t, err)
			y, err := m.MkVar(0, boolean)
			astTestMust(t, err)
			body, err := m.MkAnd(x, y)
			astTestMust(t, err)
			q, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{boolean, boolean}, Names: []symbol.Symbol{astTestSymbol(t, m, "complete-x"), astTestSymbol(t, m, "complete-y")}, Body: body})
			astTestMust(t, err)
			state := s.kernel.internalizer
			astTestMust(t, state.Internalize(q, true))
			proto, err := protomodel.NewProtoModel(m, config)
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, proto.Close()) })
			manager := state.quantifierManager
			astTestMust(t, manager.modelFinder.FixModel(proto))
			for i := uint32(0); i < 2; i++ {
				values, err := manager.modelFinder.InstantiationValues(q, i)
				astTestMust(t, err)
				if len(values) != 0 {
					t.Fatal("fixture unexpectedly has inverse restrictions")
				}
			}
			checker := manager.modelChecker
			astTestMust(t, checker.ConfigureLifecycle(3, 10))
			astTestMust(t, checker.InitSearch())
			if unknownRestricted {
				astTestMust(t, checker.initAuxiliaryContext())
				base, err := NewSMTTheoryBase(77)
				astTestMust(t, err)
				probe := &smtRestrictedCheckUnknownTheory{smtInternalizerTestTheory: &smtInternalizerTestTheory{SMTTheoryBase: base}}
				astTestMust(t, checker.auxiliaryContext.kernel.RegisterTheory(probe))
			}
			satisfied, err := checker.Check(proto, []ast.ASTHandle{q})
			astTestMust(t, err)
			if satisfied || checker.Exhausted() || len(checker.newInstances) != 1 {
				t.Fatal("missing inverse must use one saved complete witness", satisfied, checker.Exhausted(), len(checker.newInstances))
			}
			if checker.auxiliaryContext.kernel.Statistics().NumChecks != 2 {
				t.Fatal("missing inverse skipped native second auxiliary check")
			}
		})
	}
}

func smtFiniteSkolemFixture(t *testing.T) (*ast.ASTManager, *SMTModelChecker, *protomodel.ProtoModel, ast.ASTHandle) {
	t.Helper()
	m, config := smtModelGeneratorFixture(t)
	s, err := NewSMTSolver(m, nil, "AUFLIA", SMTSolverConfig{Params: config.Context, ModelEvaluator: config})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, s.Close()) })
	status, err := s.CheckSatCore(nil)
	astTestMust(t, err)
	if status != lbool.True {
		t.Fatal("source solver setup", status)
	}
	sort, err := m.MkUninterpretedSort(astTestSymbol(t, m, "finite-skolem-sort"), nil)
	astTestMust(t, err)
	c, err := m.MkConst(astTestSymbol(t, m, "finite-skolem-c"), sort)
	astTestMust(t, err)
	x, err := m.MkVar(0, sort)
	astTestMust(t, err)
	body, err := m.MkEq(x, c)
	astTestMust(t, err)
	q, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{sort}, Names: []symbol.Symbol{astTestSymbol(t, m, "finite-skolem-x")}, Body: body})
	astTestMust(t, err)
	state := s.kernel.internalizer
	astTestMust(t, state.Internalize(q, true))
	proto, err := protomodel.NewProtoModel(m, config)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, proto.Close()) })
	value, err := proto.SomeValue(sort)
	astTestMust(t, err)
	declaration, err := m.GetDecl(c)
	astTestMust(t, err)
	astTestMust(t, proto.Core().RegisterConstant(declaration, value))
	astTestMust(t, proto.FreezeUniverse(sort))
	checker, err := NewSMTModelChecker(state, 1)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, checker.Close()) })
	astTestMust(t, checker.SetAuxiliarySolverFactory(s.newAuxiliarySolver))
	return m, checker, proto, q
}

func TestSMTModelCheckerFiniteSkolemUniverseAndScopeRecovery(t *testing.T) {
	_, checker, proto, q := smtFiniteSkolemFixture(t)
	for repeat := 0; repeat < 2; repeat++ {
		satisfied, err := checker.Check(proto, []ast.ASTHandle{q})
		astTestMust(t, err)
		if !satisfied || checker.Exhausted() || len(checker.newInstances) != 0 {
			t.Fatal("finite singleton universe admitted a foreign Skolem value", satisfied, checker.Exhausted())
		}
		aux := checker.auxiliaryContext
		if aux == nil || aux.kernel.Statistics().NumChecks != uint32(repeat+1) {
			t.Fatal("finite universe bypassed auxiliary SMT")
		}
		level, err := aux.ScopeLevel()
		astTestMust(t, err)
		if level != 0 || len(aux.kernel.assertions) != 0 {
			t.Fatal("finite-universe restrictions survived query pop", level)
		}
	}
}

func TestSMTModelCheckerFiniteSkolemAllocationFrontiers(t *testing.T) {
	succeeded := false
	for stop := int64(0); stop < 512 && !succeeded; stop++ {
		t.Run(fmt.Sprint(stop), func(t *testing.T) {
			m, checker, proto, q := smtFiniteSkolemFixture(t)
			astTestMust(t, checker.initAuxiliaryContext())
			sort, err := m.GetDeclarationSort(q, 0)
			astTestMust(t, err)
			universe, err := proto.KnownUniverse(sort)
			astTestMust(t, err)
			skolem, err := m.MkFreshConst("finite-frontier", sort)
			astTestMust(t, err)
			skolemRef, err := m.NewRef(skolem)
			astTestMust(t, err)
			defer func() { astTestMust(t, m.DelRef(&skolemRef)) }()
			aux := checker.auxiliaryContext
			astTestMust(t, aux.PushCore())
			// Isolate the new restriction owner. An interrupted model evaluator
			// has a separate explicit reset/resume contract, like native rewriting.
			m.Budget().FailAfter(stop)
			err = checker.restrictSkolemToUniverse(aux, skolem, universe)
			m.Budget().FailAfter(-1)
			if err == nil {
				succeeded = true
			} else if !errors.Is(err, arena.ErrInjectedFailure) {
				t.Fatal("unexpected finite restriction failure", err)
			}
			astTestMust(t, aux.PopCore(1))
			level, scopeErr := aux.ScopeLevel()
			astTestMust(t, scopeErr)
			if level != 0 || len(aux.kernel.assertions) != 0 {
				t.Fatal("failed restriction retained scope or assertions", level)
			}
			satisfied, err := checker.Check(proto, []ast.ASTHandle{q})
			astTestMust(t, err)
			if !satisfied || checker.Exhausted() {
				t.Fatal("auxiliary did not recover after restriction pop")
			}
		})
		if t.Failed() {
			t.Fatalf("finite Skolem frontier %d", stop)
		}
	}
	if !succeeded {
		t.Fatal("no successful finite Skolem allocation frontier")
	}
}

// The first query has a complete witness; the restricted query is incomplete.
type smtRestrictedCheckUnknownTheory struct{ *smtInternalizerTestTheory }

func (t *smtRestrictedCheckUnknownTheory) FinalCheck() (SMTFinalCheckStatus, error) {
	t.finalCheckCalls++
	if t.finalCheckCalls >= 2 {
		return SMTFinalCheckGiveUp, nil
	}
	return SMTFinalCheckDone, nil
}
