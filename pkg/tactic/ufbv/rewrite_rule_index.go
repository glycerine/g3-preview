// Source: src/tactic/ufbv/ufbv_rewriter.cpp:226-267,287-317.
package ufbv

import "github.com/glycerine/g3/pkg/ast"

type demodulatorRule struct {
	function, quantifier, lhs, rhs ast.ASTHandle
}

// rewriteRuleIndex is the concrete UFBV forward index and demodulator map.
// Each indexed rule owns one reference to its quantifier, LHS and RHS.
type rewriteRuleIndex struct {
	manager *ast.ASTManager
	matcher *matchSubst
	byFunc  map[ast.ASTHandle][]ast.ASTHandle
	rules   map[ast.ASTHandle]demodulatorRule
	owned   []ast.ASTHandle
	closed  bool
}

func newRewriteRuleIndex(m *ast.ASTManager) (*rewriteRuleIndex, error) {
	matcher, e := newMatchSubst(m)
	if e != nil {
		return nil, e
	}
	return &rewriteRuleIndex{
		manager: m, matcher: matcher,
		byFunc: make(map[ast.ASTHandle][]ast.ASTHandle),
		rules:  make(map[ast.ASTHandle]demodulatorRule),
	}, nil
}

func (r *rewriteRuleIndex) check() error {
	if r == nil || r.closed || r.manager == nil || r.matcher == nil {
		return ast.ErrASTHandle
	}
	return nil
}

// insert follows insert_fwd_idx after is_demodulator has supplied its large
// application and small RHS. The index borrows f from the LHS declaration.
func (r *rewriteRuleIndex) insert(quantifier, lhs, rhs ast.ASTHandle) error {
	if e := r.check(); e != nil {
		return e
	}
	if _, present := r.rules[quantifier]; present {
		return ast.ErrASTDomain
	}
	kind, e := r.manager.GetKind(quantifier)
	if e != nil {
		return e
	}
	if kind != ast.ASTQuantifier {
		return ast.ErrASTDomain
	}
	f, e := r.manager.GetDecl(lhs)
	if e != nil {
		return e
	}
	if e = r.manager.IncRef(quantifier); e != nil {
		return e
	}
	r.owned = append(r.owned, quantifier)
	if e = r.manager.IncRef(lhs); e != nil {
		return e
	}
	r.owned = append(r.owned, lhs)
	if e = r.manager.IncRef(rhs); e != nil {
		return e
	}
	r.owned = append(r.owned, rhs)
	r.byFunc[f] = append(r.byFunc[f], quantifier)
	r.rules[quantifier] = demodulatorRule{f, quantifier, lhs, rhs}
	return nil
}

// tryRewrite is rewrite1: it skips arity mismatches and tries each indexed
// rule until the concrete matcher produces an owned rewritten expression.
func (r *rewriteRuleIndex) tryRewrite(f ast.ASTHandle, args []ast.ASTHandle) (ast.ASTRef, bool, error) {
	if e := r.check(); e != nil {
		return ast.ASTRef{}, false, e
	}
	for _, q := range r.byFunc[f] {
		rule, found := r.rules[q]
		if !found {
			return ast.ASTRef{}, false, ast.ErrASTDomain
		}
		count, e := r.manager.GetNumArgs(rule.lhs)
		if e != nil {
			return ast.ASTRef{}, false, e
		}
		if int(count) != len(args) {
			continue
		}
		decl, e := r.manager.GetDecl(rule.lhs)
		if e != nil {
			return ast.ASTRef{}, false, e
		}
		if decl != f {
			return ast.ASTRef{}, false, ast.ErrASTDomain
		}
		out, matched, e := r.matcher.rewriteRule(rule.lhs, rule.rhs, args)
		if e != nil || matched {
			return out, matched, e
		}
	}
	return ast.ASTRef{}, false, nil
}

func (r *rewriteRuleIndex) close() error {
	if r == nil || r.closed {
		return nil
	}
	if e := r.matcher.close(); e != nil {
		return e
	}
	for len(r.owned) != 0 {
		h := r.owned[0]
		consumed, e := r.manager.DecRefConsumed(h)
		if consumed {
			r.owned[0] = ast.ASTHandle{}
			r.owned = r.owned[1:]
		}
		if e != nil {
			return e
		}
		if !consumed {
			return ast.ErrASTPending
		}
	}
	r.byFunc = nil
	r.rules = nil
	r.closed = true
	return nil
}
