package smtcore

import (
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/math/simplex"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSMTArithmeticReplayBoundDependenciesAndScaling(t *testing.T) {
	theory := &SMTLinearArithmeticTheory{nextSimplex: 3}
	for i, rhs := range []int64{6, 3} {
		form := smtLinearForm{terms: map[uint32]*big.Rat{1: big.NewRat(-3, 2)}}
		form.constant.SetInt64(-rhs)
		op, truth := int32(smtArithOpLT), false
		if i == 1 {
			op, truth = smtArithOpGE, true
		}
		theory.atoms = append(theory.atoms, smtLinearArithmeticAtom{variable: int32(10 + i), form: form, op: op, supported: true})
		theory.assertions = append(theory.assertions, smtIntegerDiffAssertion{atom: i, truth: truth})
	}
	solver, status, err := theory.buildSimplex()
	astTestMust(t, err)
	if status != lbool.True {
		t.Fatal(status)
	}
	dependencies, known := solver.BoundDependencies(1, false)
	if !known || len(dependencies) != 1 || dependencies[0].Literal != NewSMTLiteral(10, true).Index() || dependencies[0].Coefficient.Cmp(big.NewRat(-2, 3)) != 0 {
		t.Fatal("weaker assertion replaced the strong negative-scaled bound", dependencies, known)
	}
	uses := []simplex.ArithmeticBoundUse{{Column: 1, Lower: false, Coefficient: *big.NewRat(9, 7)}}
	ante, known, err := smtArithmeticExplainBounds(solver, uses, true)
	astTestMust(t, err)
	if !known || len(ante.literalCoefficients) != 1 || ante.literalCoefficients[0].Cmp(big.NewRat(-6, 7)) != 0 {
		t.Fatal(ante, known)
	}
	// Temporary model values carry no logical explanation, even if they
	// coincide with a previously justified endpoint.
	value, _ := solver.Upper(1)
	astTestMust(t, solver.SetUpper(1, value))
	if _, known, err = smtArithmeticExplainBounds(solver, uses, true); err != nil || known {
		t.Fatal("model premise", known, err)
	}
}

func TestSMTArithmeticGCDExplanationExcludesUnrelatedAssertions(t *testing.T) {
	for _, proofs := range []bool{false, true} {
		theory := &SMTLinearArithmeticTheory{integer: true, nextSimplex: 4, simplexVariables: []uint32{1, 2, 3}}
		for i, op := range []int32{smtArithOpGE, smtArithOpLE, smtArithOpGE} {
			form := smtLinearForm{terms: map[uint32]*big.Rat{1: big.NewRat(2, 1), 2: big.NewRat(4, 1)}}
			form.constant.SetInt64(-1)
			if i == 2 {
				form.terms = map[uint32]*big.Rat{3: big.NewRat(1, 1)}
				form.constant.SetInt64(0)
			}
			theory.atoms = append(theory.atoms, smtLinearArithmeticAtom{variable: int32(10 + i), form: form, op: op, supported: true})
			theory.assertions = append(theory.assertions, smtIntegerDiffAssertion{atom: i, truth: true})
		}
		solver, status, err := theory.buildSimplex()
		astTestMust(t, err)
		if status != lbool.True {
			t.Fatal(status)
		}
		result, _, uses, err := solver.ArithmeticGCDWithExplanation(theory.arithmeticIntegerMask())
		astTestMust(t, err)
		if result != simplex.ArithmeticGCDConflict {
			t.Fatal(result)
		}
		ante, known, err := smtArithmeticExplainBounds(solver, uses, proofs)
		astTestMust(t, err)
		if !known || len(ante.literals) != 2 {
			t.Fatal(ante, known)
		}
		for _, literal := range ante.literals {
			if literal != NewSMTLiteral(10, false) && literal != NewSMTLiteral(11, false) {
				t.Fatal("irrelevant premise", literal)
			}
		}
		if proofs {
			if len(ante.literalCoefficients) != 2 {
				t.Fatal("missing proof weights", ante)
			}
			for _, coefficient := range ante.literalCoefficients {
				if coefficient.Cmp(big.NewRat(-1, 2)) != 0 {
					t.Fatal("normalized row weight", coefficient)
				}
			}
		} else if ante.numParameters() != 0 {
			t.Fatal("proofs disabled", ante)
		}
		// Replay after removing the contradictory scoped assertions.
		theory.assertions = theory.assertions[2:]
		solver, status, err = theory.buildSimplex()
		astTestMust(t, err)
		if status != lbool.True {
			t.Fatal(status)
		}
		result, _, _, err = solver.ArithmeticGCDWithExplanation(theory.arithmeticIntegerMask())
		astTestMust(t, err)
		if result != simplex.ArithmeticGCDPass {
			t.Fatal("conflict survived assertion removal", result)
		}
	}
}

func TestSMTArithmeticAntecedentParameterOrderAndOwnership(t *testing.T) {
	m, _ := smtModelGeneratorFixture(t)
	var ante smtArithmeticAntecedents
	astTestMust(t, ante.pushLiteral(NewSMTLiteral(4, false), big.NewRat(-2, 3), true))
	astTestMust(t, ante.pushLiteral(NewSMTLiteral(4, false), big.NewRat(5, 7), true))
	astTestMust(t, ante.pushEquality(SMTEnodePair{First: &SMTEnode{}, Second: &SMTEnode{}}, big.NewRat(11, 13), true))
	// Parameter copying is independent of the lifetime of the accumulator.
	parameters, err := ante.parameters(m, "gcd-test")
	astTestMust(t, err)
	ante.literalCoefficients[0].SetInt64(99)
	if len(parameters) != 4 {
		t.Fatal(len(parameters))
	}
	symbol, err := m.ParameterSymbol(parameters[0])
	astTestMust(t, err)
	text, _, err := m.Symbols.BareString(symbol)
	astTestMust(t, err)
	if text != "gcd-test" {
		t.Fatal(text)
	}
	for i, value := range []*big.Rat{big.NewRat(-2, 3), big.NewRat(5, 7), big.NewRat(11, 13)} {
		want, err := m.NewParameterRat(value)
		astTestMust(t, err)
		equal, err := m.ParameterEqual(parameters[i+1], want)
		astTestMust(t, err)
		if !equal {
			t.Fatal("proof coefficient order", i)
		}
		astTestMust(t, m.DelParameter(&want))
	}
	astTestMust(t, smtArithmeticReleaseParameters(m, parameters))
	if _, err := m.ParameterKind(parameters[1]); err != ast.ErrASTHandle {
		t.Fatal("released parameter remained live", err)
	}
}
