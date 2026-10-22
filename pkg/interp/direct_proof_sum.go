// Source checkpoint: src/interp/iz3proof_itp.cpp ordinary conditional
// inequality summation.
package interp

import (
	"errors"
	"math/big"

	"github.com/glycerine/g3/pkg/ast"
)

func (p *DirectProofInterpolator) ownedIntegerDivision(left, right ast.ASTHandle) (ast.ASTHandle, error) {
	ref, err := p.base.manager.MakeIDiv(left, right)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	result, err := p.base.manager.ast.RefGet(ref)
	if err != nil {
		return ast.ASTHandle{}, errors.Join(err, p.base.manager.ast.DelRef(&ref))
	}
	p.owned = append(p.owned, ref)
	return result, nil
}

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

func (p *DirectProofInterpolator) divideInequality(inequality, divisor ast.ASTHandle) (ast.ASTHandle, error) {
	value, numeral, err := p.base.manager.Numeral(divisor)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if numeral && value.Cmp(big.NewRat(1, 1)) == 0 {
		return inequality, nil
	}
	op, err := p.base.manager.Operator(inequality)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if op == OperatorLT {
		left, e := p.base.manager.Arg(inequality, 0)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		right, e := p.base.manager.Arg(inequality, 1)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		one, e := p.base.manager.MakeIntText("1")
		if e != nil {
			return ast.ASTHandle{}, e
		}
		right, e = p.base.manager.Make(OperatorSub, right, one)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		inequality, e = p.base.manager.Make(OperatorLE, left, right)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		inequality, e = p.base.manager.SimplifyInequality(inequality)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		op = OperatorLE
	}
	if op != OperatorLE {
		return ast.ASTHandle{}, ProofError{}
	}
	left, err := p.base.manager.Arg(inequality, 0)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	right, err := p.base.manager.Arg(inequality, 1)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	left, err = p.ownedIntegerDivision(left, divisor)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	right, err = p.ownedIntegerDivision(right, divisor)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	return p.base.manager.Make(op, left, right)
}

func (p *DirectProofInterpolator) simplifyRotatedSum(args []ast.ASTHandle) (ast.ASTHandle, error) {
	if len(args) != 2 {
		return ast.ASTHandle{}, ast.ErrASTDomain
	}
	placeholder, err := p.placeholder(args[0])
	if err != nil {
		return ast.ASTHandle{}, err
	}
	zero, err := p.base.manager.MakeIntText("0")
	if err != nil {
		return ast.ASTHandle{}, err
	}
	inequality, err := p.base.manager.Make(OperatorLE, zero, zero)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	aProof, err := p.base.manager.Make(OperatorTrue)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	bProof := aProof
	var rotate func(ast.ASTHandle, ast.ASTHandle) (ast.ASTHandle, error)
	rotate = func(proof, current ast.ASTHandle) (ast.ASTHandle, error) {
		if proof == placeholder {
			return p.base.manager.SimplifyInequality(current)
		}
		op, e := p.base.manager.Operator(proof)
		if e != nil || op != OperatorUninterpreted {
			return ast.ASTHandle{}, ProofError{}
		}
		declaration, e := p.base.manager.Symbol(proof)
		if e != nil || declaration != p.sum {
			return ast.ASTHandle{}, ProofError{}
		}
		first, e := p.base.manager.Arg(proof, 0)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		coefficient, e := p.base.manager.Arg(proof, 1)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		third, e := p.base.manager.Arg(proof, 2)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		if third == placeholder {
			one, e := p.base.manager.MakeIntText("1")
			if e != nil {
				return ast.ASTHandle{}, e
			}
			current, e = p.sumConditionalInequality(current, one, first, &aProof, &bProof)
			if e != nil {
				return ast.ASTHandle{}, e
			}
			return p.divideInequality(current, coefficient)
		}
		current, e = p.sumConditionalInequality(current, coefficient, third, &aProof, &bProof)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		return rotate(first, current)
	}
	inequality, err = rotate(args[1], inequality)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	result, err := p.base.manager.MkImplies(bProof, inequality)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	return p.base.manager.MkAnd(aProof, result)
}
