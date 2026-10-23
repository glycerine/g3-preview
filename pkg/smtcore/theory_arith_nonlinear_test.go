package smtcore

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/math/simplex"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSMTNonlinearRelaxationScopesProofsAndModel(t *testing.T) {
	for _, proofs := range []bool{false, true} {
		t.Run(fmt.Sprint(proofs), func(t *testing.T) {
			m, cfg := smtModelGeneratorFixture(t)
			if proofs {
				astTestMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
			}
			s, err := NewSMTSolver(m, nil, "QF_NRA", SMTSolverConfig{Params: cfg.Context, ModelEvaluator: cfg})
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, s.Close()) })
			s.kernel.preprocess = false
			astTestMust(t, s.SetProduceModels(true))
			family, err := m.ArithmeticFamilyID()
			astTestMust(t, err)
			sort, err := m.MkFamilySort(family, 0, nil)
			astTestMust(t, err)
			x, err := m.MkConst(astTestSymbol(t, m, "nl-x"), sort)
			astTestMust(t, err)
			y, err := m.MkConst(astTestSymbol(t, m, "nl-y"), sort)
			astTestMust(t, err)
			unrelated, err := m.MkConst(astTestSymbol(t, m, "nl-unrelated"), sort)
			astTestMust(t, err)
			product, err := m.MkArithmeticMul(x, y)
			astTestMust(t, err)
			refs := []ast.ASTHandle{x, y, product, unrelated}
			var base []ast.ASTHandle
			for _, spec := range []struct {
				term  ast.ASTHandle
				op    int32
				value string
			}{{x, smtArithOpGT, "2"}, {x, smtArithOpLE, "3"}, {y, smtArithOpGE, "4"}, {y, smtArithOpLE, "5"}, {unrelated, smtArithOpGE, "999"}} {
				value, err := m.ArithmeticRealNumeralText(spec.value)
				astTestMust(t, err)
				bound, err := m.MkFamilyApp(family, spec.op, nil, []ast.ASTHandle{spec.term, value})
				astTestMust(t, err)
				base = append(base, bound)
				refs = append(refs, bound)
			}
			var comparisons []ast.ASTHandle
			for _, op := range []int32{smtArithOpGT, smtArithOpLE} {
				eight, err := m.ArithmeticRealNumeralText("8")
				astTestMust(t, err)
				bound, err := m.MkFamilyApp(family, op, nil, []ast.ASTHandle{product, eight})
				astTestMust(t, err)
				comparisons = append(comparisons, bound)
				refs = append(refs, bound)
			}
			for _, h := range refs {
				ref, err := m.NewRef(h)
				astTestMust(t, err)
				t.Cleanup(func() { astTestMust(t, m.DelRef(&ref)) })
			}
			for _, bound := range base {
				astTestMust(t, s.Assert(bound))
			}
			for repeat := 0; repeat < 2; repeat++ {
				for index, bound := range comparisons {
					astTestMust(t, s.Push())
					astTestMust(t, s.Assert(bound))
					status, err := s.CheckSat(nil)
					astTestMust(t, err)
					want := lbool.True
					if index == 1 {
						want = lbool.False
					}
					if status != want {
						t.Fatal("nonlinear bound", repeat, index, status)
					}
					theory := s.kernel.internalizer.theories[family].(*SMTLinearArithmeticTheory)
					if index == 0 {
						model, err := s.AcquireModel()
						astTestMust(t, err)
						values := make([]*big.Rat, 0, 2)
						for _, term := range []ast.ASTHandle{x, y} {
							decl, err := m.GetDecl(term)
							astTestMust(t, err)
							h, err := model.ConstantInterpretation(decl)
							astTestMust(t, err)
							text, _, numeral, err := m.ArithmeticNumeralText(h)
							astTestMust(t, err)
							value, ok := new(big.Rat).SetString(text)
							if !numeral || !ok {
								t.Fatal("nonlinear model numeral", text)
							}
							values = append(values, value)
						}
						actual := new(big.Rat).Mul(values[0], values[1])
						if actual.Cmp(big.NewRat(8, 1)) <= 0 || values[0].Cmp(big.NewRat(3, 1)) > 0 || values[1].Cmp(big.NewRat(5, 1)) > 0 {
							t.Fatal("model does not satisfy original product bounds", values)
						}
						valid, err := theory.checkNonlinearMonomialAssignments()
						astTestMust(t, err)
						if !valid {
							t.Fatal("SAT model contains a relaxed product")
						}
						astTestMust(t, model.DecRef())
					} else if proofs {
						proof, err := s.Proof()
						astTestMust(t, err)
						fact, err := m.GetProofFact(proof)
						astTestMust(t, err)
						if !m.IsFalse(fact) {
							t.Fatal("nonlinear contradiction proof")
						}
						lemma := smtProofTheoryRuleNode(t, m, proof, "farkas")
						if lemma.IsNull() {
							t.Fatal("interval bound contradiction lost source farkas rule")
						}
						if smtTestArithmeticExpressionContains(t, m, lemma, unrelated) {
							t.Fatal("interval conflict cites unrelated assertion")
						}
						decl, err := m.GetDecl(lemma)
						astTestMust(t, err)
						parameters, err := m.GetDeclParametersCopy(decl)
						astTestMust(t, err)
						if len(parameters) < 3 {
							t.Fatal("interval conflict lost bound coefficients")
						}
						for i := range parameters {
							kind, err := m.ParameterKind(parameters[i])
							astTestMust(t, err)
							if i > 1 && kind != ast.ASTParameterRational {
								t.Fatal("non-rational interval conflict weight", kind)
							}
							astTestMust(t, m.DelParameter(&parameters[i]))
						}
					}
					astTestMust(t, s.Pop(1))
					if len(theory.nonlinearColumns) != 0 || len(theory.nonlinearFactors) != 0 {
						t.Fatal("nonlinear monomial survived pop")
					}
				}
			}
		})
	}
}

func TestSMTNonlinearUpwardDownwardAndIntegerIntervals(t *testing.T) {
	theory, err := NewSMTLinearArithmeticTheory(1)
	astTestMust(t, err)
	// Exercise the concrete interval/tableau boundary independently of
	// preprocessing and SAT's choice of arithmetic bound literals.
	solver := simplex.NewSolver(nil)
	solver.EnsureVariable(2)
	astTestMust(t, solver.SetLower(1, simplex.RationalInfinitesimal(big.NewRat(2, 1))))
	astTestMust(t, solver.SetUpper(1, simplex.RationalInfinitesimal(big.NewRat(3, 1))))
	astTestMust(t, solver.SetLower(2, simplex.RationalInfinitesimal(big.NewRat(8, 1))))
	astTestMust(t, solver.SetUpper(2, simplex.RationalInfinitesimal(big.NewRat(12, 1))))
	var manager smtArithmeticBoundDependencyManager
	product, err := theory.nonlinearColumnInterval(solver, 2, &manager)
	astTestMust(t, err)
	factor, err := theory.nonlinearColumnInterval(solver, 1, &manager)
	astTestMust(t, err)
	astTestMust(t, product.div(factor))
	changed, conflict := theory.updateNonlinearBounds(solver, 0, product)
	if !changed || conflict {
		t.Fatal("downward interval division")
	}
	lower, _ := solver.Lower(0)
	upper, _ := solver.Upper(0)
	if lower.Main.Cmp(big.NewRat(8, 3)) != 0 || upper.Main.Cmp(big.NewRat(6, 1)) != 0 {
		t.Fatal("downward bounds", lower, upper)
	}
	theory.integerColumns[0] = true
	interval, err := newSMTOldIntervalBounds(&manager, big.NewRat(-2, 1), true, nil, big.NewRat(5, 2), false, nil)
	astTestMust(t, err)
	integers := simplex.NewSolver(nil)
	changed, conflict = theory.updateNonlinearBounds(integers, 0, interval)
	if !changed || conflict {
		t.Fatal("integer interval normalization")
	}
	lower, _ = integers.Lower(0)
	upper, _ = integers.Upper(0)
	if lower.Main.Cmp(big.NewRat(-1, 1)) != 0 || upper.Main.Cmp(big.NewRat(2, 1)) != 0 || lower.Epsilon.Sign() != 0 || upper.Epsilon.Sign() != 0 {
		t.Fatal("integer bounds", lower, upper)
	}
}

func TestSMTNonlinearCandidateRejectsRelaxedProduct(t *testing.T) {
	m, cfg := smtModelGeneratorFixture(t)
	s, err := NewSMTSolver(m, nil, "QF_NRA", SMTSolverConfig{Params: cfg.Context, ModelEvaluator: cfg})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, s.Close()) })
	s.kernel.preprocess = false
	family, err := m.ArithmeticFamilyID()
	astTestMust(t, err)
	sort, err := m.MkFamilySort(family, 0, nil)
	astTestMust(t, err)
	x, err := m.MkConst(astTestSymbol(t, m, "nonlinear-irrational"), sort)
	astTestMust(t, err)
	square, err := m.MkArithmeticMul(x, x)
	astTestMust(t, err)
	two, err := m.ArithmeticRealNumeralText("2")
	astTestMust(t, err)
	equality, err := m.MkEq(square, two)
	astTestMust(t, err)
	astTestMust(t, s.Assert(equality))
	status, err := s.CheckSat(nil)
	astTestMust(t, err)
	// This rational tableau cannot represent sqrt(2). Accepting its free
	// product column as a genuine square would report a spurious model.
	if status != lbool.Undef {
		t.Fatal("unvalidated nonlinear product accepted", status)
	}
}
