package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestSMTGeneratedModelCleansMacroFallbackConstants(t *testing.T) {
	m, solver, _ := smtAssertedMacroFixture(t, false, false)
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	x, err := m.MkVar(0, boolean)
	astTestMust(t, err)
	c := smtRelevancyBool(t, m, "model-cleanup-constant")
	f, err := m.MkFuncDecl(astTestSymbol(t, m, "model-cleanup-function"), []ast.ASTHandle{boolean}, boolean, ast.ASTFuncDeclInfo{})
	astTestMust(t, err)
	head, err := m.MkApp(f, []ast.ASTHandle{x})
	astTestMust(t, err)
	body, err := m.MkAnd(x, c)
	astTestMust(t, err)
	equality, err := m.MkEq(head, body)
	astTestMust(t, err)
	q, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{boolean}, Names: []symbol.Symbol{astTestSymbol(t, m, "model-cleanup-x")}, Body: equality})
	astTestMust(t, err)
	astTestMust(t, solver.Assert(q))
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.True || solver.kernel.assertedMacros.NumMacros() != 1 {
		t.Fatal("macro cleanup fixture did not extract its definition", status, solver.kernel.assertedMacros.NumMacros())
	}
	generated, err := solver.AcquireModel()
	astTestMust(t, err)
	defer func() { astTestMust(t, generated.DecRef()) }()
	interpretation, err := generated.FunctionInterpretation(f)
	astTestMust(t, err)
	if interpretation == nil {
		t.Fatal("published macro interpretation is missing")
	}
	fallback, err := interpretation.Else()
	astTestMust(t, err)
	no, err := m.MkFalse()
	astTestMust(t, err)
	if fallback != no {
		t.Fatal("published macro fallback retained an unassigned model constant instead of simplifying and(x,false)", fallback, no)
	}
	declaration, err := m.GetDecl(c)
	astTestMust(t, err)
	completed, err := generated.ConstantInterpretation(declaration)
	astTestMust(t, err)
	if completed != no {
		t.Fatal("cleanup did not publish the constant interpretation used by the fallback")
	}
}

func TestSMTGeneratedModelCompressionHonorsLiveParameter(t *testing.T) {
	m, config := smtModelGeneratorFixture(t)
	local, err := config.Context.NewValue()
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, config.Context.Del(&local)) })
	key := "auto_config"
	astTestMust(t, config.Context.SetBoolName(&local, &key, false))
	solver, err := NewSMTSolver(m, &local, "QF_UF", SMTSolverConfig{Params: config.Context, ModelEvaluator: config})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, solver.Close()) })
	astTestMust(t, solver.SetProduceModels(true))
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	f, err := m.MkFuncDecl(astTestSymbol(t, m, "model-compress-function"), []ast.ASTHandle{boolean}, boolean, ast.ASTFuncDeclInfo{})
	astTestMust(t, err)
	yes, err := m.MkTrue()
	astTestMust(t, err)
	no, err := m.MkFalse()
	astTestMust(t, err)
	for _, value := range []ast.ASTHandle{no, yes} {
		application, err := m.MkApp(f, []ast.ASTHandle{value})
		astTestMust(t, err)
		negated, err := m.MkNot(application)
		astTestMust(t, err)
		astTestMust(t, solver.Assert(negated))
	}
	for _, compact := range []bool{false, true, false} {
		key = "compact"
		astTestMust(t, config.Context.SetBoolName(&local, &key, compact))
		astTestMust(t, solver.UpdateParams(&local))
		solver.kernel.preprocess = false
		status, err := solver.CheckSat(nil)
		astTestMust(t, err)
		if status != lbool.True || solver.kernel.smtState.ModelCompact != compact {
			t.Fatal("model compression fixture or parameter update", status, compact)
		}
		generated, err := solver.AcquireModel()
		astTestMust(t, err)
		func() {
			defer func() { astTestMust(t, generated.DecRef()) }()
			interpretation, err := generated.FunctionInterpretation(f)
			astTestMust(t, err)
			if interpretation == nil {
				t.Fatal("missing compressed function interpretation")
			}
			wantEntries := uint32(2)
			if compact {
				wantEntries = 0
			}
			if interpretation.NumEntries() != wantEntries {
				t.Fatal("model.compact did not control graph compression", compact, interpretation.NumEntries(), wantEntries)
			}
			fallback, err := interpretation.Else()
			astTestMust(t, err)
			if fallback != no {
				t.Fatal("compression changed the completed fallback")
			}
		}()
	}
}
