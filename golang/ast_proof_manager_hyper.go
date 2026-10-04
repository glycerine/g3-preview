// Licensed under the MIT license.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"math"
)

type ASTProofPosition struct{ First, Second uint32 }

// MkProofHyperResolve preserves all four lexical source vectors, including
// fmls and sorts, whose values are otherwise unused. Inputs are borrowed.
func (m *ASTManager) MkProofHyperResolve(premises []ASTHandle, conclusion ASTHandle, positions []ASTProofPosition, substitutions []ASTRefVector) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	if uint64(len(premises)) > math.MaxUint32 || uint64(len(positions)) > math.MaxUint32 || uint64(len(substitutions)) > math.MaxUint32 {
		return ASTHandle{}, arena.ErrBounds
	}
	var fmls astProofRawArguments
	defer fmls.close(m)
	if astSourceDebug && uint64(len(positions))+1 != uint64(len(substitutions)) {
		return ASTHandle{}, ErrASTDomain
	}
	for _, p := range premises {
		f, e := m.proofFact(p)
		if e != nil {
			return ASTHandle{}, e
		}
		if e = fmls.push(m, f); e != nil {
			return ASTHandle{}, e
		}
	}
	if e := m.proofCheckBool(conclusion); e != nil {
		return ASTHandle{}, e
	}
	var params astProofParameters
	defer params.close(m)
	for i, v := range substitutions {
		s, e := m.vectorState(v)
		if e != nil {
			return ASTHandle{}, e
		}
		for j := uint32(0); j < s.size; j++ {
			r, e := m.vectorValue(s, j)
			if e != nil {
				return ASTHandle{}, e
			}
			p, e := m.newParameterAST(m.external(r))
			if e != nil {
				return ASTHandle{}, e
			}
			if e = params.pushOwned(m, p); e != nil {
				return ASTHandle{}, e
			}
		}
		if i+1 < len(substitutions) {
			if i >= len(positions) {
				return ASTHandle{}, arena.ErrBounds
			}
			p, e := m.newParameterInt(int32(positions[i].First))
			if e != nil {
				return ASTHandle{}, e
			}
			if e = params.pushOwned(m, p); e != nil {
				return ASTHandle{}, e
			}
			p, e = m.newParameterInt(int32(positions[i].Second))
			if e != nil {
				return ASTHandle{}, e
			}
			if e = params.pushOwned(m, p); e != nil {
				return ASTHandle{}, e
			}
		}
	}
	var sorts astProofRawArguments
	defer sorts.close(m)
	var args astProofRawArguments
	defer args.close(m)
	for _, p := range premises {
		if e := sorts.push(m, m.external(m.proofSort)); e != nil {
			return ASTHandle{}, e
		}
		if e := args.push(m, p); e != nil {
			return ASTHandle{}, e
		}
	}
	if e := sorts.push(m, m.external(m.boolSort)); e != nil {
		return ASTHandle{}, e
	}
	if e := args.push(m, conclusion); e != nil {
		return ASTHandle{}, e
	}
	r, e := m.mkFamilyApp(0, ASTPrHyperResolve, params.values, args.values)
	if e != nil {
		return ASTHandle{}, e
	}
	if astSourceDebug {
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
