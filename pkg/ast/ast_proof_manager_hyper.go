package ast

import (
	"math"

	"github.com/glycerine/g3/pkg/arena"
)

type ASTProofPosition struct{ First, Second uint32 }

// MkProofHyperResolve validates premise facts and assembles native argument/
// parameter slices. Inputs are borrowed; unused source scratch is omitted.
func (m *ASTManager) MkProofHyperResolve(premises []ASTHandle, conclusion ASTHandle, positions []ASTProofPosition, substitutions [][]ASTHandle) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	if uint64(len(premises)) > math.MaxUint32 || uint64(len(positions)) > math.MaxUint32 || uint64(len(substitutions)) > math.MaxUint32 {
		return ASTHandle{}, arena.ErrBounds
	}
	if ASTSourceDebug && uint64(len(positions))+1 != uint64(len(substitutions)) {
		return ASTHandle{}, ErrASTDomain
	}
	for _, p := range premises {
		_, e := m.proofFact(p)
		if e != nil {
			return ASTHandle{}, e
		}
	}
	if e := m.proofCheckBool(conclusion); e != nil {
		return ASTHandle{}, e
	}
	var params []ASTParameter
	defer func() { m.dropProofParameters(params) }()
	for i, v := range substitutions {
		for _, h := range v {
			p, e := m.newParameterAST(h)
			if e != nil {
				return ASTHandle{}, e
			}
			params = append(params, p)
		}
		if i+1 < len(substitutions) {
			if i >= len(positions) {
				return ASTHandle{}, arena.ErrBounds
			}
			p, e := m.newParameterInt(int32(positions[i].First))
			if e != nil {
				return ASTHandle{}, e
			}
			params = append(params, p)
			p, e = m.newParameterInt(int32(positions[i].Second))
			if e != nil {
				return ASTHandle{}, e
			}
			params = append(params, p)
		}
	}
	args := append([]ASTHandle(nil), premises...)
	args = append(args, conclusion)
	r, e := m.mkFamilyApp(0, ASTPrHyperResolve, params, args)
	if e != nil {
		return ASTHandle{}, e
	}
	if ASTSourceDebug {
		fid, k, e := m.declIdentity(m.appAt(r.slot).decl)
		if e != nil {
			return ASTHandle{}, e
		}
		if fid != 0 || k != ASTPrHyperResolve {
			return ASTHandle{}, ErrASTDomain
		}
	}
	return m.external(r), nil
}
