// Source checkpoint: src/interp/iz3proof_itp.cpp local rewrite construction
// and trivial mixed-rule interpolation.
package interp

import "github.com/glycerine/g3/pkg/ast"

func (p *DirectProofInterpolator) isEquivalence(term ast.ASTHandle) (bool, error) {
	op, err := p.base.manager.Operator(term)
	return op == OperatorEqual || op == OperatorIff, err
}

func (p *DirectProofInterpolator) makeRewrite(side TermSide, position, condition, equality ast.ASTHandle) (ast.ASTHandle, error) {
	equivalence, err := p.isEquivalence(equality)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if !equivalence {
		return ast.ASTHandle{}, ProofError{}
	}
	var declaration ast.ASTHandle
	switch side {
	case TermA:
		declaration = p.rewriteA
	case TermB:
		declaration = p.rewriteB
	default:
		return ast.ASTHandle{}, ProofError{}
	}
	return p.makeSymbol(declaration, position, condition, equality)
}

func (p *DirectProofInterpolator) chainCons(chain, element ast.ASTHandle) (ast.ASTHandle, error) {
	return p.makeSymbol(p.concat, chain, element)
}

func (p *DirectProofInterpolator) makeLocalRewrite(side TermSide, formula ast.ASTHandle) (ast.ASTHandle, error) {
	equivalence, err := p.isEquivalence(formula)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if !equivalence {
		truth, e := p.base.manager.Make(OperatorTrue)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		formula, err = p.base.manager.Make(OperatorIff, truth, formula)
		if err != nil {
			return ast.ASTHandle{}, err
		}
	}
	truth, err := p.base.manager.Make(OperatorTrue)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	rewrite, err := p.makeRewrite(side, p.topPosition, truth, formula)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	return p.chainCons(truth, rewrite)
}

func (p *DirectProofInterpolator) trivialInterpolation(rule ast.ASTHandle, premises []ast.ASTHandle, inputMask int) (ast.ASTHandle, error) {
	proofs := make([]ast.ASTHandle, len(premises))
	conjuncts := make([]ast.ASTHandle, 0, len(premises))
	mask := 0
	for i, premise := range premises {
		side, err := p.TermSide(premise)
		if err != nil {
			return ast.ASTHandle{}, err
		}
		if side == TermA || side == TermB {
			proofs[i], err = p.makeLocalRewrite(side, premise)
			if err != nil {
				return ast.ASTHandle{}, err
			}
			continue
		}
		proofs[i], err = p.placeholder(premise)
		if err != nil {
			return ast.ASTHandle{}, err
		}
		if inputMask&(1<<i) != 0 {
			mask |= 1 << len(conjuncts)
		}
		conjuncts = append(conjuncts, premise)
	}
	proof, err := p.makeSymbol(rule, proofs...)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	return p.makeContraNode(proof, conjuncts, mask)
}
