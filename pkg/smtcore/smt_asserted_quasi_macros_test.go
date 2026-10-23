package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/util/lbool"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestSMTAssertedQuasiMacrosPublishConditionalModel(t *testing.T) {
	for _, scenario := range []string{"enabled", "proofs", "quasi-only", "disabled", "repeated"} {
		t.Run(scenario, func(t *testing.T) {
			m, solver, config := smtAssertedMacroFixture(t, scenario == "proofs", scenario != "disabled")
			if scenario == "quasi-only" {
				params, err := config.Context.NewValue()
				astTestMust(t, err)
				finder := "macro_finder"
				astTestMust(t, config.Context.SetBoolName(&params, &finder, false))
				astTestMust(t, solver.UpdateParams(&params))
				astTestMust(t, config.Context.Del(&params))
			}
			boolean, err := m.MkBoolSort()
			astTestMust(t, err)
			x, err := m.MkVar(0, boolean)
			astTestMust(t, err)
			ground, err := m.MkConst(astTestSymbol(t, m, "quasi-model-ground"), boolean)
			astTestMust(t, err)
			f, err := m.MkFuncDecl(astTestSymbol(t, m, "quasi-model-f"), []ast.ASTHandle{boolean, boolean}, boolean, ast.ASTFuncDeclInfo{})
			astTestMust(t, err)
			head, err := m.MkApp(f, []ast.ASTHandle{x, ground})
			astTestMust(t, err)
			body, err := m.MkIff(head, x)
			astTestMust(t, err)
			for _, name := range []string{"first", "second"} {
				if name == "second" && scenario != "repeated" {
					break
				}
				q, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{boolean}, Names: []symbol.Symbol{astTestSymbol(t, m, name+"-quasi-x")}, Body: body, QID: astTestSymbol(t, m, name+"-quasi")})
				astTestMust(t, err)
				astTestMust(t, solver.Assert(q))
			}
			astTestMust(t, solver.kernel.internalizePending())
			if scenario == "disabled" || scenario == "repeated" {
				if solver.kernel.assertedMacros.HasMacros() || solver.kernel.internalizer.quantifiers.Empty() {
					t.Fatal("disabled or repeated quasi-macro was extracted")
				}
				return
			}
			if solver.kernel.assertedMacros.NumMacros() != 1 {
				t.Fatal("unique quasi-macro was not extracted")
			}
			status, err := solver.CheckSat(nil)
			astTestMust(t, err)
			if status != lbool.True {
				t.Fatal("quasi-macro model was rejected", status)
			}
			generated, err := solver.AcquireModel()
			astTestMust(t, err)
			defer func() { astTestMust(t, generated.DecRef()) }()
			evaluator, err := model.NewModelEvaluator(generated, config)
			astTestMust(t, err)
			defer func() { astTestMust(t, evaluator.Close()) }()
			astTestMust(t, evaluator.SetModelCompletion(true))
			yes, err := m.MkTrue()
			astTestMust(t, err)
			no, err := m.MkFalse()
			astTestMust(t, err)
			out, err := m.NewRef(ast.ASTHandle{})
			astTestMust(t, err)
			defer func() { astTestMust(t, m.DelRef(&out)) }()
			for _, input := range []ast.ASTHandle{yes, no} {
				application, err := m.MkApp(f, []ast.ASTHandle{input, ground})
				astTestMust(t, err)
				astTestMust(t, evaluator.Evaluate(application, out))
				value, err := m.RefGet(out)
				astTestMust(t, err)
				if value != input {
					t.Fatal("conditional quasi-macro interpretation lost its defining slice", input, value)
				}
			}
			application, err := m.MkApp(f, []ast.ASTHandle{yes, ground})
			astTestMust(t, err)
			contradiction, err := m.MkNot(application)
			astTestMust(t, err)
			astTestMust(t, solver.Assert(contradiction))
			status, err = solver.CheckSat(nil)
			astTestMust(t, err)
			if status != lbool.False {
				t.Fatal("later assertion did not expand the quasi-macro", status)
			}
			if scenario == "proofs" {
				proof, err := solver.Proof()
				astTestMust(t, err)
				fact, err := m.GetProofFact(proof)
				astTestMust(t, err)
				if !m.IsFalse(fact) {
					t.Fatal("quasi-macro contradiction lost its proof", fact)
				}
			}
		})
	}
}
