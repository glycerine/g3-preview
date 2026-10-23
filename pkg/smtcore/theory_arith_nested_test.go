package smtcore

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/math/simplex"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSMTCrossNestedArithmeticScopedProofs(t *testing.T) {
	for _, integer := range []bool{false, true} {
		for _, proofs := range []bool{false, true} {
			for _, preprocess := range []bool{false, true} {
				for _, kind := range []string{"horner-bound", "completed-square", "multivariate"} {
					t.Run(fmt.Sprint(integer, "/", proofs, "/", preprocess, "/", kind), func(t *testing.T) {
						m, cfg := smtModelGeneratorFixture(t)
						if proofs {
							astTestMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
						}
						logic, sortKind := "QF_NRA", int32(0)
						if integer {
							logic, sortKind = "QF_NIA", 1
						}
						s, err := NewSMTSolver(m, nil, logic, SMTSolverConfig{Params: cfg.Context, ModelEvaluator: cfg})
						astTestMust(t, err)
						t.Cleanup(func() { astTestMust(t, s.Close()) })
						s.kernel.preprocess = preprocess
						family, err := m.ArithmeticFamilyID()
						astTestMust(t, err)
						sort, err := m.MkFamilySort(family, sortKind, nil)
						astTestMust(t, err)
						x, err := m.MkConst(astTestSymbol(t, m, "nested-x"), sort)
						astTestMust(t, err)
						y, err := m.MkConst(astTestSymbol(t, m, "nested-y"), sort)
						astTestMust(t, err)
						unrelated, err := m.MkConst(astTestSymbol(t, m, "nested-unrelated"), sort)
						astTestMust(t, err)
						numeral := func(value int64) ast.ASTHandle {
							h, err := m.ArithmeticNumeralRat(big.NewRat(value, 1), integer)
							astTestMust(t, err)
							return h
						}
						app := func(op int32, args ...ast.ASTHandle) ast.ASTHandle {
							h, err := m.MkFamilyApp(family, op, nil, args)
							astTestMust(t, err)
							return h
						}
						var polynomial ast.ASTHandle
						formulas := []ast.ASTHandle{app(smtArithOpGE, unrelated, numeral(77))}
						switch kind {
						case "horner-bound":
							polynomial = app(smtArithOpAdd, app(smtArithOpMul, x, x), app(smtArithOpMul, numeral(-1), x))
							formulas = append(formulas, app(smtArithOpGT, x, numeral(1)))
						case "completed-square":
							polynomial = app(smtArithOpAdd, app(smtArithOpMul, x, x), app(smtArithOpMul, numeral(-2), x), numeral(2))
						case "multivariate":
							polynomial = app(smtArithOpAdd, app(smtArithOpMul, x, x, y), app(smtArithOpMul, numeral(-2), x, y), app(smtArithOpMul, numeral(2), y))
							formulas = append(formulas, app(smtArithOpGT, y, numeral(0)))
						}
						formulas = append(formulas, app(smtArithOpLE, polynomial, numeral(0)))
						for _, formula := range formulas {
							ref, err := m.NewRef(formula)
							astTestMust(t, err)
							t.Cleanup(func() { astTestMust(t, m.DelRef(&ref)) })
						}
						for repeat := 0; repeat < 2; repeat++ {
							astTestMust(t, s.Push())
							theory := s.kernel.internalizer.theories[family].(*SMTLinearArithmeticTheory)
							config, err := s.kernel.smtParams.ArithmeticState()
							astTestMust(t, err)
							config.NLArithGB, config.NLArithBranching = false, false
							astTestMust(t, theory.ConfigureArithmetic(config))
							for _, formula := range formulas {
								astTestMust(t, s.Assert(formula))
							}
							before := theory.statistics.crossNested
							status, err := s.CheckSat(nil)
							astTestMust(t, err)
							if status != lbool.False {
								t.Fatal("cross-nested polynomial conflict", status)
							}
							if theory.statistics.crossNested == before {
								t.Fatal("nested polynomial strategy was not exercised")
							}
							if proofs {
								proof, err := s.Proof()
								astTestMust(t, err)
								fact, err := m.GetProofFact(proof)
								astTestMust(t, err)
								if !m.IsFalse(fact) {
									t.Fatal("nested polynomial conflict proof")
								}
								if smtTestArithmeticExpressionContains(t, m, proof, unrelated) {
									t.Fatal("nested polynomial proof cites unrelated assertion")
								}
							}
							astTestMust(t, s.Pop(1))
							status, err = s.CheckSat(nil)
							astTestMust(t, err)
							if status != lbool.True {
								t.Fatal("nested polynomial conflict survived pop", status)
							}
						}
					})
				}
			}
		}
	}
}

func TestSMTNestedPolynomialIdentityAtRationalPoints(t *testing.T) {
	m, cfg := smtModelGeneratorFixture(t)
	s, err := NewSMTSolver(m, nil, "QF_NRA", SMTSolverConfig{Params: cfg.Context, ModelEvaluator: cfg})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, s.Close()) })
	astTestMust(t, s.Push())
	family, err := m.ArithmeticFamilyID()
	astTestMust(t, err)
	theory := s.kernel.internalizer.theories[family].(*SMTLinearArithmeticTheory)
	sort, err := m.MkFamilySort(family, 0, nil)
	astTestMust(t, err)
	x, err := m.MkConst(astTestSymbol(t, m, "nested-identity-x"), sort)
	astTestMust(t, err)
	_, err = theory.InternalizeTerm(x)
	astTestMust(t, err)
	n := smtNestedArithmetic{theory: theory, solver: simplex.NewSolver(nil), refs: smtArithmeticDivisionCall{theory: theory}}
	defer func() { astTestMust(t, n.refs.close()) }()
	column, found := n.column(x)
	if !found {
		t.Fatal("nested polynomial variable")
	}
	for _, degrees := range [][4]int64{{3, -2, 6, 1}, {3, 1, 4, 1}, {2, 2, 5, 1}, {2, -3, 4, -2}, {2, -1, 3, 1}} {
		var polynomial []smtNestedTerm
		for _, spec := range [][2]int64{{degrees[0], degrees[1]}, {degrees[2], degrees[3]}} {
			expression, err := n.power(x, uint32(spec[0]))
			astTestMust(t, err)
			term := smtNestedTerm{coefficient: *big.NewRat(spec[1], 1), expression: expression}
			for i := int64(0); i < spec[0]; i++ {
				term.factors = append(term.factors, x)
			}
			polynomial = append(polynomial, term)
		}
		expression, err := n.crossNested(polynomial, x)
		astTestMust(t, err)
		for _, text := range []string{"-2", "-1", "0", "1/2", "2"} {
			point, _ := new(big.Rat).SetString(text)
			n.solver = simplex.NewSolver(nil)
			bound := simplex.RationalInfinitesimal(point)
			astTestMust(t, n.solver.SetLower(column, bound))
			astTestMust(t, n.solver.SetUpper(column, bound))
			interval, err := n.evaluate(expression)
			astTestMust(t, err)
			want := new(big.Rat)
			for _, spec := range [][2]int64{{degrees[0], degrees[1]}, {degrees[2], degrees[3]}} {
				term := big.NewRat(spec[1], 1)
				for i := int64(0); i < spec[0]; i++ {
					term.Mul(term, point)
				}
				want.Add(want, term)
			}
			if interval.minusInfinity() || interval.plusInfinity() || interval.lower.value.Cmp(want) != 0 || interval.upper.value.Cmp(want) != 0 {
				t.Fatal("nested polynomial identity", degrees, text, interval, want)
			}
		}
	}
}
