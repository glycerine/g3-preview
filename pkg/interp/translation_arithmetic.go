// Source checkpoint: src/interp/iz3translate.cpp arithmetic normalization
// helpers used by retained proof-rule translation.
package interp

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
)

func (t *FullTranslation) ownedSimplify(term ast.ASTHandle) (ast.ASTHandle, error) {
	ref, err := t.base.manager.ReallySimplify(term)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	result, err := t.base.manager.ast.RefGet(ref)
	if err != nil {
		return ast.ASTHandle{}, errors.Join(err, t.base.manager.ast.DelRef(&ref))
	}
	t.owned = append(t.owned, ref)
	return result, nil
}

func (t *FullTranslation) RHSNormalizeInequality(inequality ast.ASTHandle) (ast.ASTHandle, error) {
	if err := t.open(); err != nil {
		return ast.ASTHandle{}, err
	}
	zero, err := t.base.manager.MakeIntText("0")
	if err != nil {
		return ast.ASTHandle{}, err
	}
	one, err := t.base.manager.MakeIntText("1")
	if err != nil {
		return ast.ASTHandle{}, err
	}
	result, err := t.base.manager.Make(OperatorLE, zero, zero)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	result, err = t.base.manager.LinearCombination(result, one, inequality, false)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	return t.base.manager.SimplifyInequality(result)
}

func (t *FullTranslation) CheckFarkas(premises []ast.ASTHandle, conclusion ast.ASTHandle) (bool, error) {
	if err := t.open(); err != nil {
		return false, err
	}
	one, err := t.base.manager.MakeIntText("1")
	if err != nil {
		return false, err
	}
	negativeOne, err := t.base.manager.MakeIntText("-1")
	if err != nil {
		return false, err
	}
	coefficients := make([]ast.ASTHandle, len(premises)+1)
	inequalities := make([]ast.ASTHandle, len(premises)+1)
	for i := range premises {
		coefficients[i] = one
		inequalities[i] = premises[i]
	}
	coefficients[len(premises)] = negativeOne
	inequalities[len(premises)] = conclusion
	combined, err := t.base.manager.SumInequalities(coefficients, inequalities, false)
	if err != nil {
		return false, err
	}
	right, err := t.base.manager.Arg(combined, 1)
	if err != nil {
		return false, err
	}
	right, err = t.ownedSimplify(right)
	if err != nil {
		return false, err
	}
	value, numeral, err := t.base.manager.Numeral(right)
	return numeral && value.Sign() == 0, err
}

func (t *FullTranslation) NormalizeInequality(inequality ast.ASTHandle) (ast.ASTHandle, error) {
	if err := t.open(); err != nil {
		return ast.ASTHandle{}, err
	}
	result, err := t.RHSNormalizeInequality(inequality)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	op, err := t.base.manager.Operator(result)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	left, err := t.base.manager.Arg(result, 0)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	right, err := t.base.manager.Arg(result, 1)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	_, numeral, err := t.base.manager.Numeral(right)
	if err != nil || numeral {
		return result, err
	}
	zero, err := t.base.manager.MakeIntText("0")
	if err != nil {
		return ast.ASTHandle{}, err
	}
	rightOp, err := t.base.manager.Operator(right)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if rightOp == OperatorPlus {
		arguments, e := t.base.manager.Args(right)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		constant := zero
		firstNonconstant := 0
		if len(arguments) != 0 {
			if _, matched, e := t.base.manager.Numeral(arguments[0]); e != nil {
				return ast.ASTHandle{}, e
			} else if matched {
				constant = arguments[0]
				firstNonconstant = 1
			}
		}
		if firstNonconstant < len(arguments) {
			nonconstant, e := t.base.manager.Make(OperatorPlus, arguments[firstNonconstant:]...)
			if e != nil {
				return ast.ASTHandle{}, e
			}
			left, e = t.base.manager.Make(OperatorSub, left, nonconstant)
			if e != nil {
				return ast.ASTHandle{}, e
			}
		}
		right = constant
	} else {
		positive, e := t.base.manager.Make(OperatorPlus, right)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		left, e = t.base.manager.Make(OperatorSub, left, positive)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		right = zero
	}
	left, err = t.ownedSimplify(left)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	right, err = t.ownedSimplify(right)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	return t.base.manager.Make(op, left, right)
}
