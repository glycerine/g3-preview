// Source: src/smt/theory_arith_aux.h antecedents_t and bound explanations.
package smtcore

import (
	"errors"
	"math/big"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/math/simplex"
)

type smtArithmeticBoundExplanation struct {
	dependencies []simplex.BoundDependency
}

// mk_bound_from_row uses justified_derived_bound: combine contributions to
// repeated literals while preserving their first occurrence in the row.
func smtArithmeticWeightedBoundExplanation(solver *simplex.Solver, uses []simplex.ArithmeticBoundUse) *smtArithmeticBoundExplanation {
	result := &smtArithmeticBoundExplanation{}
	positions := make(map[int32]int)
	for _, use := range uses {
		if use.Coefficient.Sign() == 0 {
			continue
		}
		dependencies, known := solver.BoundDependencies(use.Column, use.Lower)
		if !known {
			return nil
		}
		for _, dependency := range dependencies {
			coefficient := new(big.Rat).Mul(&use.Coefficient, &dependency.Coefficient)
			if position, found := positions[dependency.Literal]; found {
				old := &result.dependencies[position].Coefficient
				old.Add(old, coefficient)
			} else {
				positions[dependency.Literal] = len(result.dependencies)
				result.dependencies = append(result.dependencies, simplex.BoundDependency{Literal: dependency.Literal, Coefficient: *coefficient})
			}
		}
	}
	return result
}

func smtArithmeticCopyBoundExplanation(solver *simplex.Solver, column uint32, lower bool) *smtArithmeticBoundExplanation {
	dependencies, known := solver.BoundDependencies(column, lower)
	if !known {
		return nil
	}
	return &smtArithmeticBoundExplanation{dependencies: dependencies}
}

// Euclidean mk_bound creates a plain derived_bound. Its accumulator deduplicates
// premises and discards their old multipliers; a later push_justification uses
// the caller's coefficient once for each retained premise.
func smtArithmeticPlainBoundExplanation(solver *simplex.Solver, uses []simplex.ArithmeticBoundUse) *smtArithmeticBoundExplanation {
	result := &smtArithmeticBoundExplanation{}
	seen := make(map[int32]bool)
	for _, use := range uses {
		dependencies, known := solver.BoundDependencies(use.Column, use.Lower)
		if !known {
			return nil
		}
		for _, dependency := range dependencies {
			if seen[dependency.Literal] {
				continue
			}
			seen[dependency.Literal] = true
			result.dependencies = append(result.dependencies, simplex.BoundDependency{Literal: dependency.Literal, Coefficient: *big.NewRat(1, 1)})
		}
	}
	return result
}

func smtTightenReplaySimplexBound(solver *simplex.Solver, column uint32, lower bool, value simplex.Infinitesimal, explanation *smtArithmeticBoundExplanation) (bool, error) {
	if explanation == nil {
		return smtTightenSimplexBound(solver, column, lower, value)
	}
	return smtTightenExplainedSimplexBound(solver, column, lower, value, explanation.dependencies)
}

func smtTightenExplainedSimplexBound(solver *simplex.Solver, column uint32, lower bool, value simplex.Infinitesimal, dependencies []simplex.BoundDependency) (bool, error) {
	if lower {
		if old, found := solver.Lower(column); found && value.Cmp(&old) <= 0 {
			return false, nil
		}
		return true, solver.SetLowerWithDependencies(column, value, dependencies)
	}
	if old, found := solver.Upper(column); found && value.Cmp(&old) >= 0 {
		return false, nil
	}
	return true, solver.SetUpperWithDependencies(column, value, dependencies)
}

// Each endpoint expands independently, as bound::push_justification does in
// the source. Repeated premises retain their separate proof coefficients.
func smtArithmeticExplainBounds(solver *simplex.Solver, uses []simplex.ArithmeticBoundUse, proofs bool) (smtArithmeticAntecedents, bool, error) {
	var result smtArithmeticAntecedents
	for _, use := range uses {
		dependencies, known := solver.BoundDependencies(use.Column, use.Lower)
		if !known {
			return smtArithmeticAntecedents{}, false, nil
		}
		for _, dependency := range dependencies {
			coefficient := new(big.Rat).Mul(&use.Coefficient, &dependency.Coefficient)
			if err := result.pushLiteral(SMTLiteralFromIndex(dependency.Literal), coefficient, proofs); err != nil {
				return result, false, err
			}
		}
	}
	return result, true, nil
}

// parameters returns owning copies of the source rule symbol followed by
// literal coefficients and then equality coefficients. Callers release them
// after their justification has taken its own copies.
func (a *smtArithmeticAntecedents) parameters(m *ast.ASTManager, rule string) (result []ast.ASTParameter, err error) {
	if a.numParameters() == 0 {
		return nil, nil
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, smtArithmeticReleaseParameters(m, result))
			result = nil
		}
	}()
	symbol, err := m.Symbols.Intern(rule)
	if err != nil {
		return nil, err
	}
	p, err := m.NewParameterSymbol(symbol)
	if err != nil {
		return nil, err
	}
	result = append(result, p)
	for _, coefficients := range [][]*big.Rat{a.literalCoefficients, a.equalityCoefficients} {
		for _, coefficient := range coefficients {
			p, err = m.NewParameterRat(coefficient)
			if err != nil {
				return result, err
			}
			result = append(result, p)
		}
	}
	return result, nil
}

func smtArithmeticReleaseParameters(m *ast.ASTManager, parameters []ast.ASTParameter) (err error) {
	for i := range parameters {
		err = errors.Join(err, m.DelParameter(&parameters[i]))
	}
	return
}
