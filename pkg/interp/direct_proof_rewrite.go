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

func (p *DirectProofInterpolator) chainParts(chain ast.ASTHandle) (rest, last ast.ASTHandle, err error) {
	op, err := p.base.manager.Operator(chain)
	if err != nil || op != OperatorUninterpreted {
		return ast.ASTHandle{}, ast.ASTHandle{}, ProofError{}
	}
	declaration, err := p.base.manager.Symbol(chain)
	if err != nil || declaration != p.concat {
		return ast.ASTHandle{}, ast.ASTHandle{}, ProofError{}
	}
	rest, err = p.base.manager.Arg(chain, 0)
	if err != nil {
		return ast.ASTHandle{}, ast.ASTHandle{}, err
	}
	last, err = p.base.manager.Arg(chain, 1)
	return rest, last, err
}

func (p *DirectProofInterpolator) reverseRewrite(rewrite ast.ASTHandle) (ast.ASTHandle, error) {
	declaration, err := p.base.manager.Symbol(rewrite)
	if err != nil || (declaration != p.rewriteA && declaration != p.rewriteB) {
		return ast.ASTHandle{}, ProofError{}
	}
	position, err := p.base.manager.Arg(rewrite, 0)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	condition, err := p.base.manager.Arg(rewrite, 1)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	equality, err := p.base.manager.Arg(rewrite, 2)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	op, err := p.base.manager.Operator(equality)
	if err != nil || (op != OperatorEqual && op != OperatorIff) {
		return ast.ASTHandle{}, ProofError{}
	}
	left, err := p.base.manager.Arg(equality, 0)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	right, err := p.base.manager.Arg(equality, 1)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	equality, err = p.base.manager.Make(op, right, left)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	return p.makeSymbol(declaration, position, condition, equality)
}

func (p *DirectProofInterpolator) reverseChain(chain ast.ASTHandle) (ast.ASTHandle, error) {
	result, err := p.base.manager.Make(OperatorTrue)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	for {
		empty, e := p.base.manager.IsTrue(chain)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		if empty {
			return result, nil
		}
		rest, last, e := p.chainParts(chain)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		last, e = p.reverseRewrite(last)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		result, e = p.chainCons(result, last)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		chain = rest
	}
}

func (p *DirectProofInterpolator) isEquivalenceChain(chain ast.ASTHandle) (bool, error) {
	for {
		empty, err := p.base.manager.IsTrue(chain)
		if err != nil || empty {
			return empty, err
		}
		rest, last, err := p.chainParts(chain)
		if err != nil {
			return false, err
		}
		restEmpty, err := p.base.manager.IsTrue(rest)
		if err != nil {
			return false, err
		}
		if restEmpty {
			equality, e := p.base.manager.Arg(last, 2)
			if e != nil {
				return false, e
			}
			left, e := p.base.manager.Arg(equality, 0)
			if e != nil {
				return false, e
			}
			leftTrue, e := p.base.manager.IsTrue(left)
			return !leftTrue, e
		}
		chain = rest
	}
}

func (p *DirectProofInterpolator) simplifySymmetry(args []ast.ASTHandle) (ast.ASTHandle, error) {
	if len(args) != 1 {
		return ast.ASTHandle{}, ast.ErrASTDomain
	}
	truth, err := p.base.manager.Make(OperatorTrue)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	condition := truth
	chain := args[0]
	op, err := p.base.manager.Operator(chain)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if op == OperatorTrue {
		return truth, nil
	}
	if op == OperatorImplies {
		condition, err = p.base.manager.Arg(chain, 0)
		if err != nil {
			return ast.ASTHandle{}, err
		}
		chain, err = p.base.manager.Arg(chain, 1)
		if err != nil {
			return ast.ASTHandle{}, err
		}
	}
	equivalence, err := p.isEquivalenceChain(chain)
	if err != nil || !equivalence {
		return ast.ASTHandle{}, ProofError{}
	}
	chain, err = p.reverseChain(chain)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	return p.base.manager.MkImplies(condition, chain)
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
