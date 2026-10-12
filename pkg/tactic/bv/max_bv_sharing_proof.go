// Source: src/ast/rewriter/rewriter_def.h:245-292,474-541;
// src/tactic/bv/max_bv_sharing_tactic.cpp:245-264.
package bv

import "github.com/glycerine/g3/pkg/ast"

type sharingProofResult struct {
	expr, proof ast.ASTHandle
}

func (p *SharingGoalPass) rewriteWithProof(h ast.ASTHandle, r *SharingReducer, memo map[ast.ASTHandle]sharingProofResult, walk *sharingWalk) (sharingProofResult, error) {
	if e := walk.advance(); e != nil {
		return sharingProofResult{}, e
	}
	if found, ok := memo[h]; ok {
		return found, nil
	}
	kind, e := p.manager.GetKind(h)
	if e != nil {
		return sharingProofResult{}, e
	}
	if kind == ast.ASTQuantifier {
		return p.rewriteQuantifierWithProof(h, r, memo, walk)
	}
	if kind != ast.ASTApp {
		v := sharingProofResult{expr: h}
		memo[h] = v
		return v, nil
	}
	decl, e := p.manager.GetDecl(h)
	if e != nil {
		return sharingProofResult{}, e
	}
	n, e := p.manager.GetNumArgs(h)
	if e != nil {
		return sharingProofResult{}, e
	}
	args := make([]ast.ASTHandle, n)
	var childProofs []ast.ASTHandle
	for i := range args {
		old, e := p.manager.GetArg(h, uint32(i))
		if e != nil {
			return sharingProofResult{}, e
		}
		child, e := p.rewriteWithProof(old, r, memo, walk)
		if e != nil {
			return sharingProofResult{}, e
		}
		if e := walk.advance(); e != nil {
			return sharingProofResult{}, e
		}
		args[i] = child.expr
		if !child.proof.IsNull() {
			childProofs = append(childProofs, child.proof)
		}
	}
	built := h
	var proof ast.ASTHandle
	if len(childProofs) != 0 {
		built, e = p.manager.MkApp(decl, args)
		if e != nil {
			return sharingProofResult{}, e
		}
		if e = r.keep(built); e != nil {
			return sharingProofResult{}, e
		}
		proof, e = p.manager.MkProofCongruence(h, built, childProofs)
		if e != nil {
			return sharingProofResult{}, e
		}
		if e = r.keep(proof); e != nil {
			return sharingProofResult{}, e
		}
	}
	replacement, done, e := r.ReduceApp(decl, args)
	if e != nil {
		return sharingProofResult{}, e
	}
	if done {
		local, e := p.manager.MkProofRewrite(built, replacement)
		if e != nil {
			return sharingProofResult{}, e
		}
		if e = r.keep(local); e != nil {
			return sharingProofResult{}, e
		}
		proof, e = p.manager.MkProofTransitivity(proof, local)
		if e != nil {
			return sharingProofResult{}, e
		}
		if proof != local {
			if e = r.keep(proof); e != nil {
				return sharingProofResult{}, e
			}
		}
		built = replacement
	}
	v := sharingProofResult{expr: built, proof: proof}
	memo[h] = v
	return v, nil
}

func (p *SharingGoalPass) rewriteQuantifierWithProof(q ast.ASTHandle, r *SharingReducer, memo map[ast.ASTHandle]sharingProofResult, walk *sharingWalk) (sharingProofResult, error) {
	body, e := p.manager.GetQuantifierBody(q)
	if e != nil {
		return sharingProofResult{}, e
	}
	newBody, e := p.rewriteWithProof(body, r, memo, walk)
	if e != nil {
		return sharingProofResult{}, e
	}
	if e := walk.advance(); e != nil {
		return sharingProofResult{}, e
	}
	changed := newBody.expr != body
	nPatterns, e := p.manager.GetNumPatterns(q)
	if e != nil {
		return sharingProofResult{}, e
	}
	patterns := make([]ast.ASTHandle, nPatterns)
	for i := range patterns {
		old, e := p.manager.GetPattern(q, uint32(i))
		if e != nil {
			return sharingProofResult{}, e
		}
		rewritten, e := p.rewriteWithProof(old, r, memo, walk)
		if e != nil {
			return sharingProofResult{}, e
		}
		if e := walk.advance(); e != nil {
			return sharingProofResult{}, e
		}
		patterns[i] = rewritten.expr
		changed = changed || rewritten.expr != old
	}
	nNoPatterns, e := p.manager.GetNumNoPatterns(q)
	if e != nil {
		return sharingProofResult{}, e
	}
	noPatterns := make([]ast.ASTHandle, nNoPatterns)
	for i := range noPatterns {
		old, e := p.manager.GetNoPattern(q, uint32(i))
		if e != nil {
			return sharingProofResult{}, e
		}
		rewritten, e := p.rewriteWithProof(old, r, memo, walk)
		if e != nil {
			return sharingProofResult{}, e
		}
		if e := walk.advance(); e != nil {
			return sharingProofResult{}, e
		}
		noPatterns[i] = rewritten.expr
		changed = changed || rewritten.expr != old
	}
	if !changed {
		v := sharingProofResult{expr: q}
		memo[q] = v
		return v, nil
	}
	updated, e := p.manager.UpdateQuantifierAnnotations(q, patterns, noPatterns, newBody.expr)
	if e != nil {
		return sharingProofResult{}, e
	}
	if e = r.keep(updated); e != nil {
		return sharingProofResult{}, e
	}
	bodyProof := newBody.proof
	if bodyProof.IsNull() {
		bodyProof, e = p.manager.MkProofReflexivity(body)
		if e != nil {
			return sharingProofResult{}, e
		}
		if e = r.keep(bodyProof); e != nil {
			return sharingProofResult{}, e
		}
	}
	proof, e := p.manager.MkProofQuantIntro(q, updated, bodyProof)
	if e != nil {
		return sharingProofResult{}, e
	}
	if e = r.keep(proof); e != nil {
		return sharingProofResult{}, e
	}
	v := sharingProofResult{expr: updated, proof: proof}
	memo[q] = v
	return v, nil
}
