// Source checkpoint: src/interp/iz3proof_itp.cpp ordinary conditional
// inequality summation.
package interp

import (
	"math/big"

	"github.com/glycerine/g3/pkg/ast"
)

func (p *DirectProofInterpolator) isInequality(term ast.ASTHandle) (bool, error) {
	op, err := p.base.manager.Operator(term)
	if err != nil {
		return false, err
	}
	if op == OperatorNot {
		term, err = p.base.manager.Arg(term, 0)
		if err != nil {
			return false, err
		}
		op, err = p.base.manager.Operator(term)
		if err != nil {
			return false, err
		}
	}
	return op == OperatorLE || op == OperatorLT || op == OperatorGE || op == OperatorGT, nil
}

func (p *DirectProofInterpolator) destructConditionalInequality(term ast.ASTHandle, aProof, bProof *ast.ASTHandle) (ast.ASTHandle, error) {
	op, err := p.base.manager.Operator(term)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if op == OperatorAnd {
		condition, e := p.base.manager.Arg(term, 0)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		*aProof, e = p.base.manager.MkAnd(*aProof, condition)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		term, e = p.base.manager.Arg(term, 1)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		op, err = p.base.manager.Operator(term)
		if err != nil {
			return ast.ASTHandle{}, err
		}
	}
	if op == OperatorImplies {
		condition, e := p.base.manager.Arg(term, 0)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		*bProof, e = p.base.manager.MkAnd(*bProof, condition)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		return p.base.manager.Arg(term, 1)
	}
	return term, nil
}

func (p *DirectProofInterpolator) sumConditionalInequality(current, coefficient, addition ast.ASTHandle, aProof, bProof *ast.ASTHandle) (ast.ASTHandle, error) {
	op, err := p.base.manager.Operator(addition)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if op == OperatorAnd || op == OperatorImplies {
		condition, e := p.base.manager.Arg(addition, 0)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		body, e := p.base.manager.Arg(addition, 1)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		current, e = p.sumConditionalInequality(current, coefficient, body, aProof, bProof)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		if op == OperatorAnd {
			*aProof, e = p.base.manager.MkAnd(*aProof, condition)
		} else {
			*bProof, e = p.base.manager.MkAnd(*bProof, condition)
		}
		return current, e
	}
	inequality, err := p.isInequality(addition)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if !inequality {
		return ast.ASTHandle{}, ProofError{}
	}
	return p.base.manager.LinearCombination(current, coefficient, addition, false)
}

func (p *DirectProofInterpolator) simplifySum(args []ast.ASTHandle) (ast.ASTHandle, error) {
	if len(args) != 3 {
		return ast.ASTHandle{}, ast.ErrASTDomain
	}
	coefficient, numeral, err := p.base.manager.Numeral(args[1])
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if !numeral || coefficient.Cmp(big.NewRat(1, 1)) != 0 {
		thirdOp, e := p.base.manager.Operator(args[2])
		if e != nil {
			return ast.ASTHandle{}, e
		}
		if thirdOp == OperatorUninterpreted {
			declaration, e := p.base.manager.Symbol(args[2])
			if e != nil {
				return ast.ASTHandle{}, e
			}
			if declaration == p.sum {
				one, e := p.base.manager.MakeIntText("1")
				if e != nil {
					return ast.ASTHandle{}, e
				}
				distributed, e := p.distributeSumCoefficient(args[1], args[2])
				if e != nil {
					return ast.ASTHandle{}, e
				}
				return p.makeSymbol(p.sum, args[0], one, distributed)
			}
		}
	}
	aProof, err := p.base.manager.Make(OperatorTrue)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	bProof := aProof
	inequality, err := p.destructConditionalInequality(args[0], &aProof, &bProof)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	valid, err := p.isInequality(inequality)
	if err != nil || !valid {
		if err != nil {
			return ast.ASTHandle{}, err
		}
		return ast.ASTHandle{}, ProofError{}
	}
	inequality, err = p.sumConditionalInequality(inequality, args[1], args[2], &aProof, &bProof)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	conditional, err := p.base.manager.MkImplies(bProof, inequality)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	return p.base.manager.MkAnd(aProof, conditional)
}

func (p *DirectProofInterpolator) distributeSumCoefficient(coefficient, expression ast.ASTHandle) (ast.ASTHandle, error) {
	op, err := p.base.manager.Operator(expression)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if op == OperatorUninterpreted {
		declaration, e := p.base.manager.Symbol(expression)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		if declaration == p.sum {
			first, e := p.base.manager.Arg(expression, 0)
			if e != nil {
				return ast.ASTHandle{}, e
			}
			innerCoefficient, e := p.base.manager.Arg(expression, 1)
			if e != nil {
				return ast.ASTHandle{}, e
			}
			third, e := p.base.manager.Arg(expression, 2)
			if e != nil {
				return ast.ASTHandle{}, e
			}
			first, e = p.distributeSumCoefficient(coefficient, first)
			if e != nil {
				return ast.ASTHandle{}, e
			}
			thirdOp, e := p.base.manager.Operator(third)
			if e != nil {
				return ast.ASTHandle{}, e
			}
			if thirdOp == OperatorUninterpreted {
				thirdDeclaration, symbolErr := p.base.manager.Symbol(third)
				if symbolErr != nil {
					return ast.ASTHandle{}, symbolErr
				}
				if thirdDeclaration == p.sum {
					innerCoefficient, e = p.base.manager.Make(OperatorTimes, coefficient, innerCoefficient)
					if e != nil {
						return ast.ASTHandle{}, e
					}
					third, e = p.distributeSumCoefficient(innerCoefficient, third)
					if e != nil {
						return ast.ASTHandle{}, e
					}
					one, e := p.base.manager.MakeIntText("1")
					if e != nil {
						return ast.ASTHandle{}, e
					}
					return p.makeSymbol(p.sum, first, one, third)
				}
			}
			innerCoefficient, e = p.base.manager.Make(OperatorTimes, coefficient, innerCoefficient)
			if e != nil {
				return ast.ASTHandle{}, e
			}
			return p.makeSymbol(p.sum, first, innerCoefficient, third)
		}
	}
	if op == OperatorLE {
		left, e := p.base.manager.Arg(expression, 0)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		right, e := p.base.manager.Arg(expression, 1)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		leftValue, leftNumeral, e := p.base.manager.Numeral(left)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		rightValue, rightNumeral, e := p.base.manager.Numeral(right)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		if leftNumeral && rightNumeral && leftValue.Sign() == 0 && rightValue.Sign() == 0 {
			return expression, nil
		}
	}
	zero, err := p.base.manager.MakeIntText("0")
	if err != nil {
		return ast.ASTHandle{}, err
	}
	base, err := p.base.manager.Make(OperatorLE, zero, zero)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	return p.makeSymbol(p.sum, base, coefficient, expression)
}
