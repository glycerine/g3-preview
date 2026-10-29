package qe

// Source: src/muz/spacer/spacer_qe_project.cpp
// arith_project_util::project, integer coefficient/divisor normalization.

import (
	"math/big"

	"github.com/glycerine/g3/pkg/ast"
)

func legacyIntegerLCM(left, right *big.Rat) (*big.Rat, error) {
	if left == nil || right == nil || !left.IsInt() || !right.IsInt() {
		return nil, ast.ErrASTDomain
	}
	a := new(big.Int).Abs(left.Num())
	b := new(big.Int).Abs(right.Num())
	if a.Sign() == 0 || b.Sign() == 0 {
		return new(big.Rat), nil
	}
	gcd := new(big.Int).GCD(nil, nil, a, b)
	value := new(big.Int).Quo(a, gcd)
	value.Mul(value, b)
	return new(big.Rat).SetInt(value), nil
}

func (p *legacyArithmeticProjection) replaceLegacyTerm(literal *legacyLinearLiteral, term ast.ASTHandle) error {
	replacement, err := p.syntax.manager.NewRef(term)
	if err != nil {
		return err
	}
	if err = p.syntax.manager.DelRef(&literal.term); err != nil {
		_ = p.syntax.manager.DelRef(&replacement)
		return err
	}
	literal.term = replacement
	return nil
}

func (p *legacyArithmeticProjection) normalizeIntegers(literals []*legacyLinearLiteral) (lcmCoefficients, lcmDivisors *big.Rat, resultErr error) {
	if p == nil || p.syntax == nil {
		return nil, nil, ast.ErrASTHandle
	}
	lcmCoefficients = big.NewRat(1, 1)
	for _, literal := range literals {
		if literal == nil || literal.manager != p.syntax.manager || literal.coefficient.Sign() == 0 || !literal.coefficient.IsInt() {
			return nil, nil, ast.ErrASTDomain
		}
		lcmCoefficients, resultErr = legacyIntegerLCM(lcmCoefficients, literal.coefficient)
		if resultErr != nil {
			return nil, nil, resultErr
		}
	}
	lcmDivisors = big.NewRat(1, 1)
	for _, literal := range literals {
		factor := new(big.Rat).Quo(lcmCoefficients, new(big.Rat).Abs(literal.coefficient))
		if factor.Cmp(big.NewRat(1, 1)) != 0 {
			term, err := p.syntax.manager.RefGet(literal.term)
			if err != nil {
				return nil, nil, err
			}
			scaled, err := p.multiplyLegacy(factor, term)
			if err != nil {
				return nil, nil, err
			}
			if err = p.replaceLegacyTerm(literal, scaled); err != nil {
				return nil, nil, err
			}
		}
		if literal.coefficient.Sign() > 0 {
			literal.coefficient.Set(lcmCoefficients)
		} else {
			literal.coefficient.Neg(lcmCoefficients)
		}
		if literal.divisor.Sign() != 0 {
			literal.divisor.Mul(literal.divisor, factor)
			lcmDivisors, resultErr = legacyIntegerLCM(lcmDivisors, literal.divisor)
			if resultErr != nil {
				return nil, nil, resultErr
			}
		}
	}
	lcmDivisors, resultErr = legacyIntegerLCM(lcmDivisors, lcmCoefficients)
	if resultErr != nil {
		return nil, nil, resultErr
	}
	return lcmCoefficients, lcmDivisors, nil
}
