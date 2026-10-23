package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSMTQFLRAAutomaticSetupRationalThresholdsAndCNF(t *testing.T) {
	for _, tc := range []struct {
		name, constant              string
		nonCNF, firstPush, disabled bool
		relevancy                   uint32
	}{
		{"cnf-small", "1/2", false, false, false, 0},
		{"large-numerator-only", "2000003/500", false, false, false, 0},
		{"large-denominator-only", "1999999/503", false, false, false, 0},
		{"numerator-boundary", "2000000/503", false, false, false, 0},
		{"both-large", "2000003/503", false, false, false, 2},
		{"non-cnf", "1/2", true, false, false, 0},
		{"non-cnf-both-large", "2000003/503", true, false, false, 2},
		{"first-push", "2000003/503", false, true, false, 0},
		{"disabled", "1/2", false, false, true, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, cfg := smtModelGeneratorFixture(t)
			solver, err := NewSMTSolver(m, nil, "QF_LRA", SMTSolverConfig{Params: cfg.Context, ModelEvaluator: cfg})
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, solver.Close()) })
			if tc.disabled {
				smtDisableAutoConfig(t, solver)
			}
			constant, err := m.ArithmeticRealNumeralText(tc.constant)
			astTestMust(t, err)
			real, err := m.GetSort(constant)
			astTestMust(t, err)
			x, err := m.MkConst(astTestSymbol(t, m, "setup-real-x"), real)
			astTestMust(t, err)
			family, err := m.ArithmeticFamilyID()
			astTestMust(t, err)
			formula, err := m.MkFamilyApp(family, smtArithOpGE, nil, []ast.ASTHandle{x, constant})
			astTestMust(t, err)
			if tc.nonCNF {
				p := smtRelevancyBool(t, m, "setup-real-p")
				q := smtRelevancyBool(t, m, "setup-real-q")
				both, err := m.MkAnd(p, q)
				astTestMust(t, err)
				formula, err = m.MkOr(both, formula)
				astTestMust(t, err)
			}
			astTestMust(t, solver.Assert(formula))
			if tc.firstPush {
				astTestMust(t, solver.Push())
			}
			status, err := solver.CheckSat(nil)
			astTestMust(t, err)
			if status != lbool.True {
				t.Fatal("real setup changed satisfiability", status)
			}
			k := solver.kernel
			if k.smtState.RelevancyLvl != tc.relevancy || k.smtParams.state.RelevancyLvl != tc.relevancy || k.internalizer.relevancyLevel != tc.relevancy || k.relevancy.enabled != (tc.relevancy > 0) {
				t.Fatal("rational-size relevance guard", k.smtState.RelevancyLvl, tc.relevancy)
			}
			owner, err := k.smtParams.ArithmeticState()
			astTestMust(t, err)
			if tc.disabled {
				if owner.Eq2Ineq || !owner.Reflect || !owner.PropagateEqs {
					t.Fatal("disabled auto configuration applied a real profile", owner)
				}
				return
			}
			if owner != k.arithState || !owner.Eq2Ineq || owner.Reflect || owner.PropagateEqs || k.preprocessState.NNFCNF || !k.preprocessState.EliminateTermITE {
				t.Fatal("real setup owner and consumers diverged", owner)
			}
			if tc.firstPush {
				if k.smtState.PhaseSelection != 3 || owner.SmallLemmaSize == 32 {
					t.Fatal("first push ran the static real profile")
				}
				astTestMust(t, solver.Pop(1))
			} else if tc.nonCNF {
				if k.smtState.PhaseSelection != 0 || k.smtState.RestartAdaptive || k.smtState.RestartStrategy != 0 || owner.StrongerLemmas || owner.SmallLemmaSize != 32 {
					t.Fatal("non-CNF real profile", owner, k.smtState)
				}
			} else if k.smtState.PhaseSelection != 4 || owner.SmallLemmaSize != 32 {
				t.Fatal("CNF real profile", owner, k.smtState)
			}
		})
	}
}
