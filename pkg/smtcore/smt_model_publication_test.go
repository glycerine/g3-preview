package smtcore

import (
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestSMTSolverPublishesAcceptedMBQICandidateWithoutRebuilding(t *testing.T) {
	m, config := smtModelGeneratorFixture(t)
	local, err := config.Context.NewValue()
	astTestMust(t, err)
	preprocess := "preprocess"
	astTestMust(t, config.Context.SetBoolName(&local, &preprocess, false))
	solver, err := NewSMTSolver(m, &local, "AUFLIA", SMTSolverConfig{Params: config.Context, ModelEvaluator: config})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, solver.Close()); astTestMust(t, config.Context.Del(&local)) })
	astTestMust(t, solver.SetProduceModels(true))
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	x, err := m.MkVar(0, boolean)
	astTestMust(t, err)
	body, err := m.MkIff(x, x)
	astTestMust(t, err)
	quantifier, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{boolean}, Names: []symbol.Symbol{astTestSymbol(t, m, "publish-candidate-x")}, Body: body})
	astTestMust(t, err)
	astTestMust(t, solver.Assert(quantifier))
	zero, err := m.ArithmeticIntegerNumeralText("0")
	astTestMust(t, err)
	integer, err := m.GetSort(zero)
	astTestMust(t, err)
	marker, err := m.MkFuncDecl(astTestSymbol(t, m, "accepted-candidate-marker"), nil, integer, ast.ASTFuncDeclInfo{})
	astTestMust(t, err)
	markerRef, err := m.NewRef(marker)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.DelRef(&markerRef)) })
	manager := solver.kernel.internalizer.quantifierManager
	build := manager.candidateModel
	var candidates []*SMTModelGenerator
	var expected ast.ASTHandle
	manager.candidateModel = func() (*SMTModelGenerator, error) {
		generator, err := build()
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, generator)
		value, err := m.ArithmeticIntegerNumeralText(strconv.Itoa(40 + len(candidates)))
		if err == nil {
			err = generator.model.Core().RegisterConstant(marker, value)
		}
		if err != nil {
			_ = generator.Close()
			return nil, err
		}
		expected = value
		return generator, nil
	}
	for attempt := 1; attempt <= 2; attempt++ {
		status, err := solver.CheckSat(nil)
		astTestMust(t, err)
		if status != lbool.True || len(candidates) != attempt {
			t.Fatal("accepted candidate was not checked once per search", status, len(candidates), attempt)
		}
		generated, err := solver.AcquireModel()
		astTestMust(t, err)
		if generated == nil {
			t.Fatal("accepted model was not published")
		}
		value, err := generated.ConstantInterpretation(marker)
		astTestMust(t, err)
		if value != expected {
			t.Fatal("publication rebuilt or reused an obsolete candidate", value, expected)
		}
		astTestMust(t, generated.DecRef())
		if manager.acceptedModel != nil || !candidates[attempt-1].closed {
			t.Fatal("publication did not consume accepted generator ownership")
		}
	}
	astTestMust(t, solver.SetProduceModels(false))
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.True || len(candidates) != 3 || !candidates[2].closed || manager.acceptedModel != nil {
		t.Fatal("model-disabled search retained accepted candidate", status, len(candidates))
	}
}
