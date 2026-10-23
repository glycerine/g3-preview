package smtcore

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestSMTAssertedBoundEliminationGuardsProofsAndScopes(t *testing.T) {
	for _, proofs := range []bool{false, true} {
		for _, mode := range []string{"enabled", "disabled", "no-preprocessing", "used-variable"} {
			t.Run(fmt.Sprint(proofs, "/", mode), func(t *testing.T) {
				m, solver := smtAssertedNNFFixture(t, proofs)
				astTestMust(t, solver.Push())
				five, err := m.ArithmeticIntegerNumeralText("5")
				astTestMust(t, err)
				s, err := m.GetSort(five)
				astTestMust(t, err)
				x, err := m.MkVar(0, s)
				astTestMust(t, err)
				family, err := m.ArithmeticFamilyID()
				astTestMust(t, err)
				body, err := m.MkFamilyApp(family, 2, nil, []ast.ASTHandle{x, five})
				astTestMust(t, err)
				if mode == "used-variable" {
					b, err := m.MkBoolSort()
					astTestMust(t, err)
					p := smtQuickPredicate(t, m, "bound-use-p", s, b)
					px, err := m.MkApp(p, []ast.ASTHandle{x})
					astTestMust(t, err)
					body, err = m.MkOr(body, px)
					astTestMust(t, err)
				}
				q, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{s}, Names: []symbol.Symbol{symbol.Numerical(0)}, Body: body, Weight: 1})
				astTestMust(t, err)
				astTestMust(t, solver.Assert(q))
				k := solver.kernel
				k.preprocessState.EliminateBounds = mode != "disabled"
				k.preprocess = mode != "no-preprocessing"
				before := k.assertions[0].Formula
				astTestMust(t, k.eliminatePendingBounds())
				row := k.assertions[0]
				if mode == "enabled" {
					if !m.IsFalse(row.Formula) {
						t.Fatal("unbounded universal inequality did not reduce to false")
					}
				} else if row.Formula != before {
					t.Fatal("guarded or used variable was eliminated")
				}
				if proofs {
					fact, err := m.GetProofFact(row.Proof)
					astTestMust(t, err)
					if fact != row.Formula {
						t.Fatal("bound elimination assertion proof mismatch")
					}
				}
				if mode == "enabled" {
					status, err := solver.CheckSat(nil)
					astTestMust(t, err)
					if status != lbool.False {
						t.Fatal("bound elimination conflict", status)
					}
				}
				astTestMust(t, solver.Pop(1))
				status, err := solver.CheckSat(nil)
				astTestMust(t, err)
				if status != lbool.True {
					t.Fatal("scope retained an eliminated bound", status)
				}
			})
		}
	}
}

func TestSMTAssertedBoundsSourceSetupAndUpdates(t *testing.T) {
	cases := []struct {
		logic                                     string
		pushFirst, checkEmptyFirst, autoOff, want bool
	}{
		{logic: "AUFLIA", want: true},
		{logic: "AUFLIRA", want: true},
		{logic: "AUFLIA", pushFirst: true, want: true},
		{logic: "AUFLIA", autoOff: true},
		{logic: "ALL", want: true},
		{logic: "ALL", pushFirst: true},
		{logic: "ALL", checkEmptyFirst: true},
		{logic: "UFLRA", want: true},
		{logic: "LRA"},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprint(tc), func(t *testing.T) {
			m, cfg := smtModelGeneratorFixture(t)
			params, err := cfg.Context.NewValue()
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, cfg.Context.Del(&params)) })
			if tc.autoOff {
				key := "auto_config"
				astTestMust(t, cfg.Context.SetBoolName(&params, &key, false))
			}
			backendLogic := tc.logic
			if tc.logic == "ALL" {
				// General mixed arithmetic setup is still partial. Hold the
				// concrete integer backend constant while checking setup modes.
				backendLogic = "AUFLIA"
			}
			solver, err := NewSMTSolver(m, &params, backendLogic, SMTSolverConfig{Params: cfg.Context, ModelEvaluator: cfg})
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, solver.Close()) })
			solver.kernel.setupLogic = tc.logic
			if tc.pushFirst {
				astTestMust(t, solver.Push())
			}
			if tc.checkEmptyFirst {
				status, err := solver.CheckSat(nil)
				astTestMust(t, err)
				if status != lbool.True {
					t.Fatal("empty prefix", status)
				}
			}
			five, err := m.ArithmeticIntegerNumeralText("5")
			astTestMust(t, err)
			if tc.logic == "LRA" || tc.logic == "UFLRA" {
				five, err = m.ArithmeticRealNumeralText("5")
				astTestMust(t, err)
			}
			s, err := m.GetSort(five)
			astTestMust(t, err)
			x, err := m.MkVar(0, s)
			astTestMust(t, err)
			family, err := m.ArithmeticFamilyID()
			astTestMust(t, err)
			body, err := m.MkFamilyApp(family, 2, nil, []ast.ASTHandle{x, five})
			astTestMust(t, err)
			q, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{s}, Names: []symbol.Symbol{symbol.Numerical(0)}, Body: body, Weight: 1})
			astTestMust(t, err)
			astTestMust(t, solver.Assert(q))
			status, err := solver.CheckSat(nil)
			astTestMust(t, err)
			if status != lbool.False {
				reason, _ := solver.ReasonUnknown()
				t.Fatal("unbounded universal", status, reason)
			}
			k := solver.kernel
			if k.preprocessState.EliminateBounds != tc.want {
				t.Fatal("source setup bound policy", k.preprocessState.EliminateBounds, tc.want)
			}
			astTestMust(t, solver.UpdateParams(&params))
			if k.preprocessState.EliminateBounds != tc.want {
				t.Fatal("parameter update erased setup policy")
			}
		})
	}
}
