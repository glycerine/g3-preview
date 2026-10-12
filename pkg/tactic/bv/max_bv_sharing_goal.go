// Source: src/tactic/bv/max_bv_sharing_tactic.cpp:238-271.
package bv

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/tactic"
)

var ErrSharingProofs = errors.New("g3: max BV sharing proof traversal is not yet ported")

// SharingGoalPass applies the concrete BV sharing reducer to goal formulas.
// Proof production remains in the enclosing source tactic's unfinished scope.
type SharingGoalPass struct {
	manager *ast.ASTManager
	family  ast.FamilyID
	maxArgs uint32
}

func NewSharingGoalPass(m *ast.ASTManager, maxArgs uint32) (*SharingGoalPass, error) {
	if m == nil {
		return nil, ast.ErrASTHandle
	}
	fid, e := m.RegisterBitVectorPlugin()
	if e != nil {
		return nil, e
	}
	return &SharingGoalPass{manager: m, family: fid, maxArgs: maxArgs}, nil
}

// Apply follows the source's sequential goal updates: pairs found in an
// earlier formula can guide the next formula. The result is the input goal.
func (p *SharingGoalPass) Apply(goal *tactic.Goal) (_ *tactic.Goal, result error) {
	if p == nil || p.manager == nil || goal == nil || goal.Manager() != p.manager {
		return nil, ast.ErrASTHandle
	}
	if e := goal.CheckOwner(); e != nil {
		return nil, e
	}
	proofs, e := goal.ProofsEnabled()
	if e != nil {
		return nil, e
	}
	if proofs {
		return nil, ErrSharingProofs
	}
	r, e := NewSharingReducer(p.manager, p.family, p.maxArgs)
	if e != nil {
		return nil, e
	}
	defer func() { result = errors.Join(result, r.Close()) }()
	memo := make(map[ast.ASTHandle]ast.ASTHandle)
	n, e := goal.Size()
	if e != nil {
		return nil, e
	}
	for i := uint32(0); i < n; i++ {
		current, e := goal.Size()
		if e != nil {
			return nil, e
		}
		if i >= current { // A preceding update may have made the goal inconsistent.
			break
		}
		form, e := goal.Form(i)
		if e != nil {
			return nil, e
		}
		dep, e := goal.Dependency(i)
		if e != nil {
			return nil, e
		}
		rewritten, e := p.rewriteApplication(form, r, memo)
		if e != nil {
			return nil, e
		}
		if e := goal.Update(i, rewritten, ast.ASTHandle{}, dep); e != nil {
			return nil, e
		}
	}
	if e := goal.IncDepth(); e != nil {
		return nil, e
	}
	return goal, nil
}

func (p *SharingGoalPass) rewriteApplication(h ast.ASTHandle, r *SharingReducer, memo map[ast.ASTHandle]ast.ASTHandle) (ast.ASTHandle, error) {
	original := h
	if mapped, ok := memo[h]; ok {
		return mapped, nil
	}
	kind, e := p.manager.GetKind(h)
	if e != nil {
		return ast.ASTHandle{}, e
	}
	if kind != ast.ASTApp {
		if kind == ast.ASTQuantifier {
			return p.rewriteQuantifier(h, r, memo)
		}
		memo[h] = h
		return h, nil
	}
	decl, e := p.manager.GetDecl(h)
	if e != nil {
		return ast.ASTHandle{}, e
	}
	n, e := p.manager.GetNumArgs(h)
	if e != nil {
		return ast.ASTHandle{}, e
	}
	args := make([]ast.ASTHandle, n)
	changed := false
	for i := range args {
		arg, e := p.manager.GetArg(h, uint32(i))
		if e != nil {
			return ast.ASTHandle{}, e
		}
		args[i], e = p.rewriteApplication(arg, r, memo)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		changed = changed || args[i] != arg
	}
	if replacement, done, e := r.ReduceApp(decl, args); e != nil {
		return ast.ASTHandle{}, e
	} else if done {
		memo[h] = replacement
		return replacement, nil
	}
	if changed {
		h, e = p.manager.MkApp(decl, args)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		if e := r.keep(h); e != nil {
			return ast.ASTHandle{}, e
		}
	}
	memo[original] = h
	return h, nil
}

func (p *SharingGoalPass) rewriteQuantifier(q ast.ASTHandle, r *SharingReducer, memo map[ast.ASTHandle]ast.ASTHandle) (ast.ASTHandle, error) {
	body, e := p.manager.GetQuantifierBody(q)
	if e != nil {
		return ast.ASTHandle{}, e
	}
	newBody, e := p.rewriteApplication(body, r, memo)
	if e != nil {
		return ast.ASTHandle{}, e
	}
	changed := newBody != body
	nPatterns, e := p.manager.GetNumPatterns(q)
	if e != nil {
		return ast.ASTHandle{}, e
	}
	patterns := make([]ast.ASTHandle, nPatterns)
	for i := range patterns {
		old, e := p.manager.GetPattern(q, uint32(i))
		if e != nil {
			return ast.ASTHandle{}, e
		}
		patterns[i], e = p.rewriteApplication(old, r, memo)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		changed = changed || patterns[i] != old
	}
	nNoPatterns, e := p.manager.GetNumNoPatterns(q)
	if e != nil {
		return ast.ASTHandle{}, e
	}
	noPatterns := make([]ast.ASTHandle, nNoPatterns)
	for i := range noPatterns {
		old, e := p.manager.GetNoPattern(q, uint32(i))
		if e != nil {
			return ast.ASTHandle{}, e
		}
		noPatterns[i], e = p.rewriteApplication(old, r, memo)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		changed = changed || noPatterns[i] != old
	}
	if !changed {
		memo[q] = q
		return q, nil
	}
	updated, e := p.manager.UpdateQuantifierAnnotations(q, patterns, noPatterns, newBody)
	if e != nil {
		return ast.ASTHandle{}, e
	}
	if e := r.keep(updated); e != nil {
		return ast.ASTHandle{}, e
	}
	memo[q] = updated
	return updated, nil
}
