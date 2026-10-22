// Source checkpoint: src/interp/iz3proof_itp.cpp local resolution dispatch.
package interp

import "github.com/glycerine/g3/pkg/ast"

// MakeResolution applies the source Boolean interpolation rule for local A or
// B pivots. Mixed pivots are handled by the arithmetic cut-elimination path,
// which remains a separate porting stage.
func (p *DirectProofInterpolator) MakeResolution(pivot ast.ASTHandle, conclusion []ast.ASTHandle, premise1, premise2 ast.ASTHandle) (ast.ASTHandle, error) {
	if err := p.open(); err != nil {
		return ast.ASTHandle{}, err
	}
	side, err := p.TermSide(pivot)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	switch side {
	case TermA:
		return p.keep(p.base.manager.MkOr(premise1, premise2))
	case TermB:
		return p.keep(p.base.manager.MkAnd(premise1, premise2))
	default:
		return ast.ASTHandle{}, ProofError{}
	}
}

// MakeSymmetry preserves local equality proofs unchanged. Mixed symmetry uses
// the mixed-resolution machinery and is completed with that rule family.
func (p *DirectProofInterpolator) MakeSymmetry(conclusion, premiseConclusion, premise ast.ASTHandle) (ast.ASTHandle, error) {
	if err := p.open(); err != nil {
		return ast.ASTHandle{}, err
	}
	side, err := p.TermSide(conclusion)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if side == TermMixed {
		return ast.ASTHandle{}, ProofError{}
	}
	return p.keep(premise, nil)
}
