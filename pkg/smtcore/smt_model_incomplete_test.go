package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSMTIncompleteTheoryCandidateKeepsFirstFunctionEntry(t *testing.T) {
	m, config := smtModelGeneratorFixture(t)
	params, err := config.Context.NewValue()
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, config.Context.Del(&params)) })
	key := "preprocess"
	astTestMust(t, config.Context.SetBoolName(&params, &key, false))
	key = "compact"
	astTestMust(t, config.Context.SetBoolName(&params, &key, false))
	s, err := NewSMTSolver(m, &params, "QF_UFLRA", SMTSolverConfig{Params: config.Context, ModelEvaluator: config})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, s.Close()) })
	astTestMust(t, s.SetProduceModels(true))
	zero, err := m.ArithmeticRealNumeralText("0")
	astTestMust(t, err)
	real, err := m.GetSort(zero)
	astTestMust(t, err)
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	f, err := m.MkFuncDecl(astTestSymbol(t, m, "incomplete-function"), []ast.ASTHandle{real}, boolean, ast.ASTFuncDeclInfo{})
	astTestMust(t, err)
	var arguments []ast.ASTHandle
	for _, name := range []string{"incomplete-a", "incomplete-b"} {
		argument, err := m.MkConst(astTestSymbol(t, m, name), real)
		astTestMust(t, err)
		arguments = append(arguments, argument)
		application, err := m.MkApp(f, []ast.ASTHandle{argument})
		astTestMust(t, err)
		if len(arguments) == 2 {
			application, err = m.MkNot(application)
			astTestMust(t, err)
		}
		astTestMust(t, s.Assert(application))
	}
	// Symbolic exponentiation makes the actual arithmetic theory incomplete
	// before it mutates shared assignments or assumes their equalities.
	power, err := m.MkArithmeticPower(arguments[0], arguments[1])
	astTestMust(t, err)
	application, err := m.MkApp(f, []ast.ASTHandle{power})
	astTestMust(t, err)
	astTestMust(t, s.Assert(application))
	one, err := m.ArithmeticRealNumeralText("1")
	astTestMust(t, err)
	distinct, err := m.MkApp(f, []ast.ASTHandle{one})
	astTestMust(t, err)
	distinct, err = m.MkNot(distinct)
	astTestMust(t, err)
	astTestMust(t, s.Assert(distinct))
	status, err := s.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.Undef || s.kernel.lastFailure != SMTFailureTheory {
		t.Fatal("incomplete-theory search", status, s.kernel.lastFailure)
	}
	ok, err := s.PrepareCandidateModel()
	astTestMust(t, err)
	if !ok {
		t.Fatal("candidate model unavailable")
	}
	generated, err := s.AcquireModel()
	astTestMust(t, err)
	if generated == nil {
		t.Fatal("candidate model missing")
	}
	defer func() { astTestMust(t, generated.DecRef()) }()
	interpretation, err := generated.FunctionInterpretation(f)
	astTestMust(t, err)
	if interpretation == nil || interpretation.NumEntries() != 2 {
		t.Fatal("duplicate candidate function entries", interpretation)
	}
	first, err := interpretation.FindEntry([]ast.ASTHandle{zero})
	astTestMust(t, err)
	second, err := interpretation.FindEntry([]ast.ASTHandle{one})
	astTestMust(t, err)
	if first == nil || second == nil {
		t.Fatal("missing distinct candidate arguments")
	}
	value, err := first.Result()
	astTestMust(t, err)
	if !m.IsTrue(value) {
		t.Fatal("duplicate replaced the first result")
	}
	value, err = second.Result()
	astTestMust(t, err)
	if !m.IsFalse(value) {
		t.Fatal("distinct argument entry lost")
	}
}
