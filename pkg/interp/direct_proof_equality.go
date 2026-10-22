// Source checkpoint: src/interp/iz3proof_itp.cpp local modus-ponens,
// congruence, and transitivity rules.
package interp

import "github.com/glycerine/g3/pkg/ast"

func (p *DirectProofInterpolator) makeEquivalence(left, right ast.ASTHandle) (ast.ASTHandle, error) {
	sort, err := p.base.manager.Type(left)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	boolean, err := p.base.manager.IsBoolType(sort)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if boolean {
		return p.base.manager.Make(OperatorIff, left, right)
	}
	return p.base.manager.Make(OperatorEqual, left, right)
}

func (p *DirectProofInterpolator) MakeModusPonens(equivalence, premise1, premise2 ast.ASTHandle) (ast.ASTHandle, error) {
	if err := p.open(); err != nil {
		return ast.ASTHandle{}, err
	}
	side, err := p.TermSide(equivalence)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if side == TermMixed {
		return ast.ASTHandle{}, ProofError{}
	}
	left, err := p.base.manager.Arg(equivalence, 0)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	right, err := p.base.manager.Arg(equivalence, 1)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	leftSide, err := p.TermSide(left)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	rightSide, err := p.TermSide(right)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	var interpolation ast.ASTHandle
	if leftSide == TermA {
		if rightSide == TermA {
			rightOp, e := p.base.manager.Operator(right)
			if e != nil {
				return ast.ASTHandle{}, e
			}
			if rightOp == OperatorOr {
				args, e := p.base.manager.Args(right)
				if e != nil {
					return ast.ASTHandle{}, e
				}
				interpolation, err = p.MakeAssumption(p.scope.Hi, args)
			} else {
				interpolation, err = p.base.manager.Make(OperatorFalse)
			}
		} else if side == TermA {
			interpolation = right
		} else {
			return ast.ASTHandle{}, ProofError{}
		}
	} else if rightSide == TermA {
		comparison, e := p.base.manager.Make(OperatorEqual, left, right)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		comparisonSide, e := p.TermSide(comparison)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		if comparisonSide != TermA {
			return ast.ASTHandle{}, ProofError{}
		}
		interpolation, err = p.base.manager.MkNot(left)
	} else {
		interpolation, err = p.base.manager.Make(OperatorTrue)
	}
	if err != nil {
		return ast.ASTHandle{}, err
	}
	negatedEquivalence, err := p.base.manager.MkNot(equivalence)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	interpolation, err = p.MakeResolution(left, []ast.ASTHandle{right, negatedEquivalence}, interpolation, premise1)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	return p.MakeResolution(equivalence, []ast.ASTHandle{right}, interpolation, premise2)
}

func (p *DirectProofInterpolator) MakeCongruence(premiseConclusion, conclusion, premise ast.ASTHandle) (ast.ASTHandle, error) {
	if err := p.open(); err != nil {
		return ast.ASTHandle{}, err
	}
	premiseSide, err := p.TermSide(premiseConclusion)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	conclusionSide, err := p.TermSide(conclusion)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if premiseSide == TermMixed || conclusionSide == TermMixed {
		return ast.ASTHandle{}, ProofError{}
	}
	var interpolation ast.ASTHandle
	if premiseSide == TermA {
		if conclusionSide == TermA {
			interpolation, err = p.base.manager.Make(OperatorFalse)
		} else {
			interpolation = premiseConclusion
		}
	} else if conclusionSide == TermA {
		interpolation, err = p.base.manager.MkNot(premiseConclusion)
	} else {
		interpolation, err = p.base.manager.Make(OperatorTrue)
	}
	if err != nil {
		return ast.ASTHandle{}, err
	}
	return p.MakeResolution(premiseConclusion, []ast.ASTHandle{conclusion}, interpolation, premise)
}

func (p *DirectProofInterpolator) MakeTransitivity(x, y, z, premise1, premise2 ast.ASTHandle) (ast.ASTHandle, error) {
	left, err := p.makeEquivalence(x, y)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	right, err := p.makeEquivalence(y, z)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	conclusion, err := p.makeEquivalence(x, z)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	equivalence, err := p.base.manager.Make(OperatorIff, left, conclusion)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	interpolation, err := p.MakeCongruence(right, equivalence, premise2)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	return p.MakeModusPonens(equivalence, premise1, interpolation)
}
