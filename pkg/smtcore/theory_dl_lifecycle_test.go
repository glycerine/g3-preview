package smtcore

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

// Datalog has no captured parameter owner. The destination's constructor is
// already its source mk_fresh equivalent; verify that retained solver paths
// preserve that fresh owner instead of adding a redundant plugin adapter.
func TestSMTDatalogFreshTranslationAndAuxiliaryOwners(t *testing.T) {
	for _, same := range []bool{true, false} {
		t.Run(fmt.Sprint(same), func(t *testing.T) {
			m, cfg := smtModelGeneratorFixture(t)
			target, tcfg := m, cfg
			if !same {
				target, tcfg = smtModelGeneratorFixture(t)
			}
			s, err := NewSMTSolver(m, nil, "QF_FD", SMTSolverConfig{Params: cfg.Context, ModelEvaluator: cfg, TargetConfig: func(manager *ast.ASTManager) (SMTSolverConfig, error) {
				if manager != target {
					return SMTSolverConfig{}, ast.ErrASTHandle
				}
				return SMTSolverConfig{Params: tcfg.Context, ModelEvaluator: tcfg}, nil
			}})
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, s.Close()) })
			astTestMust(t, s.SetProduceModels(true))
			sort, err := m.MkDatalogFiniteSort(astTestSymbol(t, m, "fresh-datalog-domain"), 4)
			astTestMust(t, err)
			x, err := m.MkConst(astTestSymbol(t, m, "fresh-datalog-x"), sort)
			astTestMust(t, err)
			astTestPin(t, m, x)
			two, err := m.MkDatalogNumeral(2, sort)
			astTestMust(t, err)
			eq, err := m.MkEq(x, two)
			astTestMust(t, err)
			astTestMust(t, s.Assert(eq))
			status, err := s.CheckSat(nil)
			astTestMust(t, err)
			if status != lbool.True {
				t.Fatal("source finite-domain model", status)
			}
			family, err := m.RegisterDatalogPlugin()
			astTestMust(t, err)
			previous := s.kernel.internalizer.theories[family].(*SMTDatalogTheory)
			if previous.NumVariables() == 0 || len(previous.declarations) == 0 {
				t.Fatal("source theory was not populated")
			}
			value, err := s.Translate(target, nil)
			astTestMust(t, err)
			copied := value.(*SMTSolver)
			t.Cleanup(func() { astTestMust(t, copied.Close()) })
			targetFamily, err := target.RegisterDatalogPlugin()
			astTestMust(t, err)
			next := copied.kernel.internalizer.theories[targetFamily].(*SMTDatalogTheory)
			if next == previous || next.NumVariables() != 0 || len(next.declarations) != 0 || len(next.declarationRefs) != 0 {
				t.Fatal("translation borrowed populated source theory state")
			}
			status, err = copied.CheckSat(nil)
			astTestMust(t, err)
			if status != lbool.True {
				t.Fatal("translated finite-domain model", status)
			}
			if next.NumVariables() == 0 || len(next.declarations) == 0 {
				t.Fatal("fresh destination theory did not own new state")
			}
			model, err := copied.AcquireModel()
			astTestMust(t, err)
			if model == nil {
				t.Fatal("translated model missing")
			}
			defer func() { astTestMust(t, model.DecRef()) }()
			tx := x
			if !same {
				translation, err := ast.NewASTTranslation(m, target, false)
				astTestMust(t, err)
				tx, err = translation.Translate(x)
				astTestMust(t, err)
				astTestMust(t, translation.Close())
			}
			declaration, err := target.GetDecl(tx)
			astTestMust(t, err)
			result, err := model.ConstantInterpretation(declaration)
			astTestMust(t, err)
			n, matched, err := target.DatalogNumeralValue(result)
			astTestMust(t, err)
			if !matched || n != 2 {
				t.Fatal("translated finite-domain interpretation", n, matched)
			}
			auxiliary, err := s.newAuxiliarySolver()
			astTestMust(t, err)
			defer func() { astTestMust(t, auxiliary.Close()) }()
			fresh := auxiliary.kernel.internalizer.theories[family].(*SMTDatalogTheory)
			if fresh == previous || fresh.NumVariables() != 0 || len(fresh.declarations) != 0 {
				t.Fatal("auxiliary did not construct an independent datalog owner")
			}
			if previous.NumVariables() == 0 || len(previous.declarations) == 0 {
				t.Fatal("fresh construction consumed source theory state")
			}
		})
	}
}
