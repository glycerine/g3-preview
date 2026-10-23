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

func TestSMTArithmeticPlainDerivedBoundDeduplicatesAndReplays(t *testing.T) {
	solver := simplex.NewSolver(nil)
	low := simplex.RationalInfinitesimal(big.NewRat(1, 1))
	high := simplex.RationalInfinitesimal(big.NewRat(3, 1))
	literal := NewSMTLiteral(7, false).Index()
	dependencies := []simplex.BoundDependency{{Literal: literal, Coefficient: *big.NewRat(-5, 3)}}
	astTestMust(t, solver.SetLowerWithDependencies(1, low, dependencies))
	astTestMust(t, solver.SetUpperWithDependencies(1, high, dependencies))
	uses := []simplex.ArithmeticBoundUse{{Column: 1, Lower: true}, {Column: 1, Lower: false}}
	explanation := smtArithmeticPlainBoundExplanation(solver, uses)
	if explanation == nil || len(explanation.dependencies) != 1 || explanation.dependencies[0].Literal != literal || explanation.dependencies[0].Coefficient.Cmp(big.NewRat(1, 1)) != 0 {
		t.Fatal("plain derived reason", explanation)
	}
	// The derived bound owns flattened premises independently of its old
	// endpoints, and applies a later caller coefficient exactly once.
	solver.UnsetLower(1)
	solver.UnsetUpper(1)
	changed, err := smtTightenReplaySimplexBound(solver, 2, true, high, explanation)
	astTestMust(t, err)
	if !changed {
		t.Fatal("derived bound not installed")
	}
	ante, known, err := smtArithmeticExplainBounds(solver, []simplex.ArithmeticBoundUse{{Column: 2, Lower: true, Coefficient: *big.NewRat(11, 13)}}, true)
	astTestMust(t, err)
	if !known || len(ante.literals) != 1 || ante.literalCoefficients[0].Cmp(big.NewRat(11, 13)) != 0 {
		t.Fatal("derived-bound caller coefficient", ante, known)
	}
	if reason := smtArithmeticPlainBoundExplanation(solver, uses); reason != nil {
		t.Fatal("unexplained endpoint became a logical premise")
	}
}

func TestSMTArithmeticRowBoundExplanationsChooseEndpointsAndMergeWeights(t *testing.T) {
	for _, repeated := range []bool{false, true} {
		theory := &SMTLinearArithmeticTheory{}
		solver := simplex.NewSolver(nil)
		_, err := solver.AddRationalRow(1, []uint32{1, 2, 3}, []big.Rat{*big.NewRat(2, 1), *big.NewRat(-3, 1), *big.NewRat(4, 1)})
		astTestMust(t, err)
		// x = 3/2*y - 2*z. The lower endpoint uses y's lower bound and
		// z's upper bound; the upper endpoint uses the opposite pair.
		literals := []int32{10, 12, 14, 16}
		if repeated {
			literals[3] = literals[0]
		}
		for i, entry := range []struct {
			column             uint32
			lower              bool
			value, coefficient *big.Rat
		}{
			{2, true, big.NewRat(1, 1), big.NewRat(2, 3)},
			{2, false, big.NewRat(2, 1), big.NewRat(4, 5)},
			{3, true, big.NewRat(-1, 1), big.NewRat(-3, 7)},
			{3, false, big.NewRat(3, 1), big.NewRat(-1, 2)},
		} {
			_, err := smtTightenExplainedSimplexBound(solver, entry.column, entry.lower, simplex.RationalInfinitesimal(entry.value), []simplex.BoundDependency{{Literal: literals[i], Coefficient: *entry.coefficient}})
			astTestMust(t, err)
		}
		conflict, err := theory.propagateArithmeticRowBounds(solver)
		astTestMust(t, err)
		if conflict {
			t.Fatal("consistent row rejected")
		}
		lower, _ := solver.Lower(1)
		upper, _ := solver.Upper(1)
		if lower.Main.Cmp(big.NewRat(-9, 2)) != 0 || upper.Main.Cmp(big.NewRat(5, 1)) != 0 {
			t.Fatal(lower, upper)
		}
		dependencies, known := solver.BoundDependencies(1, true)
		if !known {
			t.Fatal("unexplained row lower bound")
		}
		if repeated {
			if len(dependencies) != 1 || dependencies[0].Literal != 10 || dependencies[0].Coefficient.Cmp(big.NewRat(2, 1)) != 0 {
				t.Fatal("repeated row literal weights", dependencies)
			}
		} else {
			if len(dependencies) != 2 {
				t.Fatal(dependencies)
			}
			for i, want := range []int32{10, 16} {
				if dependencies[i].Literal != want || dependencies[i].Coefficient.Cmp(big.NewRat(1, 1)) != 0 {
					t.Fatal("lower endpoint reason", dependencies)
				}
			}
		}
		dependencies, known = solver.BoundDependencies(1, false)
		if !known || len(dependencies) != 2 {
			t.Fatal(dependencies, known)
		}
		for i, want := range []*big.Rat{big.NewRat(6, 5), big.NewRat(6, 7)} {
			if dependencies[i].Literal != []int32{12, 14}[i] || dependencies[i].Coefficient.Cmp(want) != 0 {
				t.Fatal("upper endpoint reason", dependencies)
			}
		}
	}
}

func TestSMTArithmeticAffineDefinitionBoundExplanations(t *testing.T) {
	form := smtLinearForm{terms: map[uint32]*big.Rat{2: big.NewRat(2, 1), 3: big.NewRat(-3, 1)}}
	form.constant.SetInt64(5)
	theory := &SMTLinearArithmeticTheory{simplexVariables: []uint32{1}, definitions: map[SMTTheoryVar]smtLinearForm{0: form}}
	solver := simplex.NewSolver(nil)
	for i, entry := range []struct {
		column uint32
		lower  bool
		value  int64
	}{{2, true, 1}, {2, false, 2}, {3, true, -1}, {3, false, 3}} {
		_, err := smtTightenExplainedSimplexBound(solver, entry.column, entry.lower, simplex.RationalInfinitesimal(big.NewRat(entry.value, 1)), []simplex.BoundDependency{{Literal: int32(10 + 2*i), Coefficient: *big.NewRat(1, 1)}})
		astTestMust(t, err)
	}
	changed, conflict, err := theory.propagateArithmeticDefinitionBounds(solver)
	astTestMust(t, err)
	if !changed || conflict {
		t.Fatal(changed, conflict)
	}
	for i, lower := range []bool{true, false} {
		value, _ := solver.Upper(1)
		if lower {
			value, _ = solver.Lower(1)
		}
		if value.Main.Cmp(big.NewRat([]int64{-2, 12}[i], 1)) != 0 {
			t.Fatal(value)
		}
		dependencies, known := solver.BoundDependencies(1, lower)
		if !known || len(dependencies) != 2 {
			t.Fatal(dependencies, known)
		}
		for j, want := range []int64{2, -3} {
			if dependencies[j].Literal != [][]int32{{10, 16}, {12, 14}}[i][j] || dependencies[j].Coefficient.Cmp(big.NewRat(want, 1)) != 0 {
				t.Fatal("affine definition premise order or multiplier", dependencies)
			}
		}
	}
}
