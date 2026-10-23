// Source: src/smt/theory_arith_nl.h mk_gb_monomial, add_row_to_gb,
// add_monomial_def_to_gb, init_grobner_var_order, and compute_grobner.
package smtcore

import (
	"errors"
	"math/big"
	"sort"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/math/grobner"
	"github.com/glycerine/g3/pkg/math/simplex"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func smtFixedSimplexColumn(solver *simplex.Solver, column uint32) (*big.Rat, bool) {
	lower, hasLower := solver.Lower(column)
	upper, hasUpper := solver.Upper(column)
	if !hasLower || !hasUpper || lower.Epsilon.Sign() != 0 || upper.Epsilon.Sign() != 0 || lower.Main.Cmp(&upper.Main) != 0 {
		return nil, false
	}
	return new(big.Rat).Set(&lower.Main), true
}

func (t *SMTLinearArithmeticTheory) grobnerMonomial(gb *grobner.Solver, solver *simplex.Solver, coefficient *big.Rat, column uint32) (*grobner.Monomial, error) {
	coeff := new(big.Rat).Set(coefficient)
	var variables []ast.ASTHandle
	var factor func(uint32, bool) error
	factor = func(current uint32, expand bool) error {
		if current == 0 {
			return nil // The distinguished constant column is one.
		}
		if expand && len(t.nonlinearFactors[current]) != 0 {
			for _, argument := range t.nonlinearFactors[current] {
				if err := factor(argument, false); err != nil {
					return err
				}
			}
			return nil
		}
		if value, fixed := smtFixedSimplexColumn(solver, current); fixed {
			coeff.Mul(coeff, value)
			return nil
		}
		// The source preprocessor flattens products. Preserve the same
		// polynomial if a raw Go caller supplies a nested multiplication.
		if len(t.nonlinearFactors[current]) != 0 {
			return factor(current, true)
		}
		variable, found := t.theoryVariable(current)
		if !found {
			return ast.ErrASTDomain
		}
		node, err := t.Enode(variable)
		if err != nil {
			return err
		}
		if value, numeral, err := t.numeral(node.owner); err != nil {
			return err
		} else if numeral {
			coeff.Mul(coeff, value)
		} else {
			variables = append(variables, node.owner)
		}
		return nil
	}
	if err := factor(column, true); err != nil {
		return nil, err
	}
	return gb.NewMonomial(coeff, variables)
}

func (t *SMTLinearArithmeticTheory) assertGrobnerForm(gb *grobner.Solver, solver *simplex.Solver, form *smtLinearForm) (err error) {
	monomials := make([]*grobner.Monomial, 0, len(form.terms)+1)
	defer func() {
		if err != nil {
			for _, monomial := range monomials {
				err = errors.Join(err, gb.DelMonomial(monomial))
			}
		}
	}()
	columns := make([]uint32, 0, len(form.terms))
	for column := range form.terms {
		columns = append(columns, column)
	}
	sort.Slice(columns, func(i, j int) bool { return columns[i] < columns[j] })
	for _, column := range columns {
		monomial, err := t.grobnerMonomial(gb, solver, form.terms[column], column)
		if err != nil {
			return err
		}
		monomials = append(monomials, monomial)
	}
	if form.constant.Sign() != 0 {
		monomial, err := gb.NewMonomial(&form.constant, nil)
		if err != nil {
			return err
		}
		monomials = append(monomials, monomial)
	}
	return gb.AssertMonomialsZero(monomials, ast.ASTDependency{})
}

func (t *SMTLinearArithmeticTheory) checkNonlinearGrobner() (conflict bool, err error) {
	// The last integer-feasible candidate may contain speculative branch
	// bounds. Reconstruct the unconditional tableau before substituting
	// fixed variables or using their intervals to justify a global conflict.
	solver, status, err := t.buildSimplex(nil)
	if err != nil || status != lbool.True {
		return false, err
	}
	gb, err := grobner.New(t.state.manager)
	if err != nil {
		return false, err
	}
	defer func() { err = errors.Join(err, gb.Close()) }()
	columns := make(map[ast.ASTHandle]uint32)
	for variable, column := range t.simplexVariables {
		node, err := t.Enode(SMTTheoryVar(variable))
		if err != nil {
			return false, err
		}
		columns[node.owner] = column
		_, lower := solver.Lower(column)
		_, upper := solver.Upper(column)
		weight := 6 // FREE
		if _, fixed := smtFixedSimplexColumn(solver, column); fixed {
			weight = 0
		} else if lower && upper {
			weight = 2
		} else if lower || upper {
			weight = 4
		}
		if len(t.nonlinearFactors[column]) != 0 {
			weight++ // Source QUOTED_* ordering for product aliases.
		}
		if err = gb.SetWeight(node.owner, weight); err != nil {
			return false, err
		}
	}
	// Definition rows are unconditional equations. Explicit opposite
	// bounds provide the remaining fixed rows in this replay representation.
	definitionVariables := make([]int, 0, len(t.definitions))
	for variable := range t.definitions {
		definitionVariables = append(definitionVariables, int(variable))
	}
	sort.Ints(definitionVariables)
	for _, rawVariable := range definitionVariables {
		variable := SMTTheoryVar(rawVariable)
		definition := t.definitions[variable]
		form := definition.clone()
		form.addTerm(t.simplexVariables[variable], big.NewRat(-1, 1))
		if err = t.assertGrobnerForm(gb, solver, &form); err != nil {
			return false, err
		}
	}
	for i, first := range t.assertions {
		left, ok := smtLinearNormalizedLE(&t.atoms[first.atom], first)
		if !ok {
			continue
		}
		for _, second := range t.assertions[i+1:] {
			right, ok := smtLinearNormalizedLE(&t.atoms[second.atom], second)
			if ok && smtLinearFormsOpposite(&left, &right) {
				if err = t.assertGrobnerForm(gb, solver, &left); err != nil {
					return false, err
				}
				break
			}
		}
	}
	for _, column := range t.nonlinearColumns {
		if value, fixed := smtFixedSimplexColumn(solver, column); fixed {
			form := newSMTLinearForm()
			form.addTerm(column, big.NewRat(1, 1))
			form.constant.Neg(value)
			if err = t.assertGrobnerForm(gb, solver, &form); err != nil {
				return false, err
			}
		}
	}
	t.statistics.grobner++
	if _, err = gb.ComputeBasis(t.nonlinearGBLimit); err != nil {
		return false, err
	}
	conflict = gb.Inconsistent()
	var manager smtArithmeticBoundDependencyManager
	for _, equation := range gb.Equations() {
		if conflict {
			break
		}
		sum, err := newSMTOldIntervalPoint(&manager, new(big.Rat), nil, nil)
		if err != nil {
			return false, err
		}
		for _, monomial := range equation.Monomials() {
			interval, err := newSMTOldIntervalPoint(&manager, monomial.Coefficient(), nil, nil)
			if err != nil {
				return false, err
			}
			for i := uint32(0); i < monomial.Degree(); {
				h := monomial.Variable(i)
				j := i + 1
				for j < monomial.Degree() && monomial.Variable(j) == h {
					j++
				}
				column, found := columns[h]
				if !found {
					return false, ast.ErrASTDomain
				}
				bound, err := t.nonlinearColumnInterval(solver, column, &manager)
				if err != nil {
					return false, err
				}
				bound.power(j - i)
				if err = interval.mul(bound); err != nil {
					return false, err
				}
				i = j
			}
			if err = sum.add(interval); err != nil {
				return false, err
			}
		}
		conflict = !sum.containsZero()
	}
	if conflict {
		literals := make([]SMTLiteral, 0, len(t.assertions))
		for _, assertion := range t.assertions {
			literals = append(literals, smtLinearAssertionLiteral(&t.atoms[assertion.atom], assertion))
		}
		t.statistics.conflicts++
		err = t.state.SetTheoryConflict(t.FamilyID(), literals)
	}
	return conflict, err
}
