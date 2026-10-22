// Source checkpoint: src/interp/iz3proof_itp.cpp arithmetic normalization
// proof vocabulary and basic chain accessors.
package interp

import "github.com/glycerine/g3/pkg/ast"

func (p *DirectProofInterpolator) makeNormalStep(left, right, proof ast.ASTHandle) (ast.ASTHandle, error) {
	equality, err := p.makeEquivalence(left, right)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	return p.makeSymbol(p.normalStep, equality, proof)
}

func (p *DirectProofInterpolator) consNormal(first, rest ast.ASTHandle) (ast.ASTHandle, error) {
	return p.makeSymbol(p.normalChain, first, rest)
}

func (p *DirectProofInterpolator) normalChainParts(chain ast.ASTHandle) (first, rest ast.ASTHandle, err error) {
	op, err := p.base.manager.Operator(chain)
	if err != nil || op != OperatorUninterpreted {
		return ast.ASTHandle{}, ast.ASTHandle{}, ProofError{}
	}
	declaration, err := p.base.manager.Symbol(chain)
	if err != nil || declaration != p.normalChain {
		return ast.ASTHandle{}, ast.ASTHandle{}, ProofError{}
	}
	first, err = p.base.manager.Arg(chain, 0)
	if err != nil {
		return ast.ASTHandle{}, ast.ASTHandle{}, err
	}
	rest, err = p.base.manager.Arg(chain, 1)
	return first, rest, err
}

func (p *DirectProofInterpolator) normalStepParts(step ast.ASTHandle) (left, right, proof ast.ASTHandle, err error) {
	op, err := p.base.manager.Operator(step)
	if err != nil || op != OperatorUninterpreted {
		return ast.ASTHandle{}, ast.ASTHandle{}, ast.ASTHandle{}, ProofError{}
	}
	declaration, err := p.base.manager.Symbol(step)
	if err != nil || declaration != p.normalStep {
		return ast.ASTHandle{}, ast.ASTHandle{}, ast.ASTHandle{}, ProofError{}
	}
	equality, err := p.base.manager.Arg(step, 0)
	if err != nil {
		return ast.ASTHandle{}, ast.ASTHandle{}, ast.ASTHandle{}, err
	}
	left, err = p.base.manager.Arg(equality, 0)
	if err != nil {
		return ast.ASTHandle{}, ast.ASTHandle{}, ast.ASTHandle{}, err
	}
	right, err = p.base.manager.Arg(equality, 1)
	if err != nil {
		return ast.ASTHandle{}, ast.ASTHandle{}, ast.ASTHandle{}, err
	}
	proof, err = p.base.manager.Arg(step, 1)
	return left, right, proof, err
}

func (p *DirectProofInterpolator) makeNormal(inequality, chain ast.ASTHandle) (ast.ASTHandle, error) {
	valid, err := p.isInequality(inequality)
	if err != nil || !valid {
		return ast.ASTHandle{}, ProofError{}
	}
	return p.makeSymbol(p.normal, inequality, chain)
}

func (p *DirectProofInterpolator) destructNormal(proof ast.ASTHandle) (inequality, chain ast.ASTHandle, err error) {
	op, err := p.base.manager.Operator(proof)
	if err != nil {
		return ast.ASTHandle{}, ast.ASTHandle{}, err
	}
	if op == OperatorUninterpreted {
		declaration, e := p.base.manager.Symbol(proof)
		if e != nil {
			return ast.ASTHandle{}, ast.ASTHandle{}, e
		}
		if declaration == p.normal {
			inequality, e = p.base.manager.Arg(proof, 0)
			if e != nil {
				return ast.ASTHandle{}, ast.ASTHandle{}, e
			}
			chain, e = p.base.manager.Arg(proof, 1)
			return inequality, chain, e
		}
	}
	chain, err = p.base.manager.Make(OperatorTrue)
	return proof, chain, err
}

func (p *DirectProofInterpolator) isNormalInequality(proof ast.ASTHandle) (bool, error) {
	op, err := p.base.manager.Operator(proof)
	if err != nil {
		return false, err
	}
	if op == OperatorUninterpreted {
		declaration, e := p.base.manager.Symbol(proof)
		if e != nil {
			return false, e
		}
		if declaration == p.normal {
			inequality, e := p.base.manager.Arg(proof, 0)
			if e != nil {
				return false, e
			}
			return p.isInequality(inequality)
		}
	}
	return p.isInequality(proof)
}
