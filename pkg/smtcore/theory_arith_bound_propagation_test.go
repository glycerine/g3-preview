package smtcore

// Source: src/smt/theory_arith_core.h:2546-3029 row bound propagation.

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSMTArithmeticRowsPropagateBooleanBounds(t *testing.T) {
	for _, strict := range []bool{false, true} {
		for _, small := range []uint32{0, 16} {
			for _, proofs := range []bool{false, true} {
				t.Run(fmt.Sprint(strict, "/", small, "/", proofs), func(t *testing.T) {
					m := astTestManager(t)
					if proofs {
						astTestMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
					}
					family, err := m.RegisterArithmeticPlugin()
					astTestMust(t, err)
					_, _, state := smtInternalizerFixture(t, m, nil)
					theory, err := NewSMTLinearArithmeticTheory(family)
					astTestMust(t, err)
					astTestMust(t, theory.ConfigureArithmetic(SMTArithState{
						BoundProp: 2, PropagationThreshold: ^uint32(0),
						MaxLemmaSize: 128, SmallLemmaSize: small,
						PropagateEqs: true, BlandsRuleThreshold: 1000,
					}))
					astTestMust(t, state.RegisterTheory(theory))
					sort, err := m.MkFamilySort(family, 0, nil)
					astTestMust(t, err)
					variable := func(name string) ast.ASTHandle {
						h, err := m.MkConst(astTestSymbol(t, m, name), sort)
						astTestMust(t, err)
						return h
					}
					number := func(text string) ast.ASTHandle {
						h, err := m.ArithmeticRealNumeralText(text)
						astTestMust(t, err)
						return h
					}
					app := func(op int32, arguments ...ast.ASTHandle) ast.ASTHandle {
						h, err := m.MkFamilyApp(family, op, nil, arguments)
						astTestMust(t, err)
						return h
					}
					x, y, z := variable("row-prop-x"), variable("row-prop-y"), variable("row-prop-z")
					xop, targetOp := int32(smtArithOpGE), int32(smtArithOpGE)
					if strict {
						xop, targetOp = smtArithOpGT, smtArithOpGT
					}
					premises := []ast.ASTHandle{
						app(xop, x, number("1")),
						app(smtArithOpGE, y, number("2")),
					}
					unrelated := app(smtArithOpGE, z, number("10"))
					target := app(targetOp, app(smtArithOpAdd, x, y), number("3"))
					for _, expression := range append(append([]ast.ASTHandle{}, premises...), unrelated, target) {
						astTestMust(t, state.Internalize(expression, true))
					}
					astTestMust(t, theory.Propagate())
					astTestMust(t, state.PushScope())
					var premiseLiterals []SMTLiteral
					for _, expression := range append(append([]ast.ASTHandle{}, premises...), unrelated) {
						literal, err := state.Literal(expression)
						astTestMust(t, err)
						astTestMust(t, state.AssignLiteral(literal, NewSMTBooleanAxiom()))
						premiseLiterals = append(premiseLiterals, literal)
					}
					astTestMust(t, state.PropagateTheoryAssignments())
					astTestMust(t, theory.Propagate())
					targetLiteral, err := state.Literal(target)
					astTestMust(t, err)
					assignment, err := state.Assignment(targetLiteral)
					astTestMust(t, err)
					if assignment != lbool.True || theory.statistics.boundPropagations != 1 {
						t.Fatal("row bound was not propagated", assignment, theory.statistics.boundPropagations)
					}
					if small == 0 {
						data, err := state.BooleanData(targetLiteral.Var())
						astTestMust(t, err)
						object, found := data.Justification().Justification()
						owner, ok := object.(*SMTExtTheoryPropagationJustification)
						if !found || !ok || owner.Consequent != targetLiteral || len(owner.Literals) != 2 || owner.Literals[0] != premiseLiterals[0] || owner.Literals[1] != premiseLiterals[1] {
							t.Fatalf("external row reason: %T %#v", object, owner)
						}
						for _, literal := range owner.Literals {
							if literal == premiseLiterals[2] {
								t.Fatal("row reason retained unrelated bound")
							}
						}
						wantParameters := 0
						if proofs {
							wantParameters = 3
						}
						if len(owner.Parameters()) != wantParameters {
							t.Fatal("row proof parameter count", len(owner.Parameters()), wantParameters)
						}
					} else {
						want := boundClauseSet([]SMTLiteral{premiseLiterals[0].Negated(), premiseLiterals[1].Negated(), targetLiteral})
						found := false
						for _, record := range state.clauses {
							if record.clause != nil && record.clause.IsAuxLemma() && boundClauseSet(record.Literals) == want {
								found = true
							}
						}
						if !found {
							t.Fatal("small row lemma lost exact premises", want)
						}
					}
					astTestMust(t, state.PopScopes(1))
					assignment, err = state.Assignment(targetLiteral)
					astTestMust(t, err)
					if assignment != lbool.Undef {
						t.Fatal("row consequence survived scope pop", assignment)
					}
				})
			}
		}
	}
}

func TestSMTArithmeticRowExplanationRelaxesStrongBounds(t *testing.T) {
	for _, relax := range []bool{false, true} {
		for _, proofs := range []bool{false, true} {
			t.Run(fmt.Sprint(relax, "/", proofs), func(t *testing.T) {
				m := astTestManager(t)
				if proofs {
					astTestMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
				}
				family, err := m.RegisterArithmeticPlugin()
				astTestMust(t, err)
				_, _, state := smtInternalizerFixture(t, m, nil)
				theory, err := NewSMTLinearArithmeticTheory(family)
				astTestMust(t, err)
				astTestMust(t, theory.ConfigureArithmetic(SMTArithState{
					BoundProp: 2, PropagationThreshold: ^uint32(0), MaxLemmaSize: 128,
					PropagateEqs: true, BlandsRuleThreshold: 1000, StrongerLemmas: relax,
				}))
				astTestMust(t, state.RegisterTheory(theory))
				sort, err := m.MkFamilySort(family, 0, nil)
				astTestMust(t, err)
				variable := func(name string) ast.ASTHandle {
					h, makeErr := m.MkConst(astTestSymbol(t, m, name), sort)
					astTestMust(t, makeErr)
					return h
				}
				number := func(text string) ast.ASTHandle {
					h, makeErr := m.ArithmeticRealNumeralText(text)
					astTestMust(t, makeErr)
					return h
				}
				app := func(op int32, arguments ...ast.ASTHandle) ast.ASTHandle {
					h, makeErr := m.MkFamilyApp(family, op, nil, arguments)
					astTestMust(t, makeErr)
					return h
				}
				x, y := variable("relax-x"), variable("relax-y")
				weak := app(smtArithOpGE, x, number("1"))
				strong := app(smtArithOpGE, x, number("2"))
				yBound := app(smtArithOpGE, y, number("2"))
				target := app(smtArithOpGE, app(smtArithOpAdd, x, y), number("3"))
				for _, expression := range []ast.ASTHandle{weak, strong, yBound, target} {
					astTestMust(t, state.Internalize(expression, true))
				}
				astTestMust(t, theory.Propagate())
				literals := make([]SMTLiteral, 3)
				for index, expression := range []ast.ASTHandle{weak, strong, yBound} {
					literals[index], err = state.Literal(expression)
					astTestMust(t, err)
					astTestMust(t, state.AssignLiteral(literals[index], NewSMTBooleanAxiom()))
				}
				astTestMust(t, state.PropagateTheoryAssignments())
				astTestMust(t, theory.Propagate())
				targetLiteral, err := state.Literal(target)
				astTestMust(t, err)
				data, err := state.BooleanData(targetLiteral.Var())
				astTestMust(t, err)
				object, found := data.Justification().Justification()
				owner, ok := object.(*SMTExtTheoryPropagationJustification)
				if !found || !ok || len(owner.Literals) != 2 || owner.Literals[1] != literals[2] {
					t.Fatalf("relaxed external reason: %T %#v", object, owner)
				}
				wantX := literals[1]
				if relax {
					wantX = literals[0]
				}
				if owner.Literals[0] != wantX {
					t.Fatal("wrong x premise", owner.Literals[0], wantX, owner.Literals)
				}
				wantParameters := 0
				if proofs {
					wantParameters = 3
				}
				if len(owner.Parameters()) != wantParameters {
					t.Fatal("relaxed proof parameter count", len(owner.Parameters()), wantParameters)
				}
			})
		}
	}
}

func TestSMTArithmeticRowPropagationConfigurationGates(t *testing.T) {
	tests := []struct {
		name      string
		boundProp uint32
		threshold uint32
		skipBig   bool
		want      lbool.LBool
	}{
		{"enabled", 2, ^uint32(0), false, lbool.True},
		{"mode-none", 0, ^uint32(0), false, lbool.Undef},
		{"conflict-threshold", 2, 0, false, lbool.Undef},
		{"skip-big-coefficients", 2, ^uint32(0), true, lbool.Undef},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			m := astTestManager(t)
			family, err := m.RegisterArithmeticPlugin()
			astTestMust(t, err)
			_, _, state := smtInternalizerFixture(t, m, nil)
			theory, err := NewSMTLinearArithmeticTheory(family)
			astTestMust(t, err)
			astTestMust(t, theory.ConfigureArithmetic(SMTArithState{
				BoundProp: test.boundProp, PropagationThreshold: test.threshold,
				SkipRowsWithBigCoeffs: test.skipBig, MaxLemmaSize: 128,
				PropagateEqs: true, BlandsRuleThreshold: 1000,
			}))
			astTestMust(t, state.RegisterTheory(theory))
			sort, err := m.MkFamilySort(family, 0, nil)
			astTestMust(t, err)
			variable := func(name string) ast.ASTHandle {
				h, makeErr := m.MkConst(astTestSymbol(t, m, name), sort)
				astTestMust(t, makeErr)
				return h
			}
			number := func(text string) ast.ASTHandle {
				h, makeErr := m.ArithmeticRealNumeralText(text)
				astTestMust(t, makeErr)
				return h
			}
			app := func(op int32, arguments ...ast.ASTHandle) ast.ASTHandle {
				h, makeErr := m.MkFamilyApp(family, op, nil, arguments)
				astTestMust(t, makeErr)
				return h
			}
			x, y := variable("gate-x"), variable("gate-y")
			premises := []ast.ASTHandle{
				app(smtArithOpGE, x, number("1")),
				app(smtArithOpGE, y, number("2")),
			}
			largeX := app(smtArithOpMul, number("2147483648"), x)
			target := app(smtArithOpGE, app(smtArithOpAdd, largeX, y), number("2147483650"))
			for _, expression := range append(append([]ast.ASTHandle{}, premises...), target) {
				astTestMust(t, state.Internalize(expression, true))
			}
			astTestMust(t, theory.Propagate())
			for _, expression := range premises {
				literal, literalErr := state.Literal(expression)
				astTestMust(t, literalErr)
				astTestMust(t, state.AssignLiteral(literal, NewSMTBooleanAxiom()))
			}
			astTestMust(t, state.PropagateTheoryAssignments())
			astTestMust(t, theory.Propagate())
			targetLiteral, err := state.Literal(target)
			astTestMust(t, err)
			assignment, err := state.Assignment(targetLiteral)
			astTestMust(t, err)
			if assignment != test.want {
				t.Fatal("row propagation gate", assignment, test.want)
			}
		})
	}
}
