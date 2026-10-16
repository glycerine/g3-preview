// Source: src/tactic/ufbv/ufbv_rewriter.cpp:226-267,287-317.
package ufbv

import "github.com/glycerine/g3/pkg/ast"

type rewriteForwardSet struct {
	values    []ast.ASTHandle
	header    uint64
	traceLive bool
}

type demodulatorRule struct {
	function, quantifier, lhs, rhs ast.ASTHandle
	ownedStart                     int
	removing                       bool
}

// rewriteRuleIndex is the concrete UFBV forward index and demodulator map.
// Each indexed rule owns one reference to its quantifier, LHS and RHS.
type rewriteRuleIndex struct {
	manager *ast.ASTManager
	matcher *matchSubst
	byFunc  map[ast.ASTHandle]*rewriteForwardSet
	rules   map[ast.ASTHandle]demodulatorRule
	owned   []ast.ASTHandle
	closing bool
	closed  bool
}

func newRewriteRuleIndex(m *ast.ASTManager) (*rewriteRuleIndex, error) {
	matcher, e := newMatchSubst(m)
	if e != nil {
		return nil, e
	}
	return &rewriteRuleIndex{
		manager: m, matcher: matcher,
		byFunc: make(map[ast.ASTHandle]*rewriteForwardSet),
		rules:  make(map[ast.ASTHandle]demodulatorRule),
	}, nil
}

func (r *rewriteRuleIndex) check() error {
	if r == nil || r.closing || r.closed || r.manager == nil || r.matcher == nil {
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
	assertions, e := forwardAssertionEnabled()
	if e != nil {
		return e
	}
	if assertions {
		kind, e := r.manager.GetKind(lhs)
		if e != nil {
			return e
		}
		if kind != ast.ASTApp {
			if e := forwardAssertionFailure(217, "large->get_kind() == AST_APP"); e != nil {
				return e
			}
		}
		if quantifier.IsNull() {
			if e := forwardAssertionFailure(218, "demodulator"); e != nil {
				return e
			}
		}
		if lhs.IsNull() || rhs.IsNull() {
			if e := forwardAssertionFailure(219, "large && small"); e != nil {
				return e
			}
		}
	}
	if _, present := r.rules[quantifier]; present {
		return ast.ErrASTDomain
	}
	if rhs.IsNull() {
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
	set, found := r.byFunc[f]
	if found && set == nil {
		if assertions {
			if e := forwardAssertionFailure(231, "it->m_value"); e != nil {
				return e
			}
		}
		return ast.ErrASTDomain
	}
	if !found {
		set, e = r.newForwardSet()
		if e != nil {
			return e
		}
		r.byFunc[f] = set
	}
	start := len(r.owned)
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
	set.values = append(set.values, quantifier)
	r.rules[quantifier] = demodulatorRule{function: f, quantifier: quantifier, lhs: lhs, rhs: rhs, ownedStart: start}
	return nil
}

// remove follows remove_fwd_idx's LHS, RHS, quantifier release order. A
// stopped release leaves its current reference and rule reachable for retry.
// As in the source, the function entry remains after its last rule is erased.
func (r *rewriteRuleIndex) remove(f, quantifier ast.ASTHandle) error {
	if e := r.check(); e != nil {
		return e
	}
	set, inForward := r.byFunc[f]
	if !inForward {
		if enabled, e := forwardAssertionEnabled(); e != nil {
			return e
		} else if enabled {
			if _, present := r.rules[quantifier]; !present {
				if e := forwardAssertionFailure(252, "m_demodulator2lhs_rhs.contains(demodulator)"); e != nil {
					return e
				}
			}
		}
		return ast.ErrASTDomain
	}
	if set == nil {
		return ast.ErrASTDomain
	}
	rule, found := r.rules[quantifier]
	if !found || rule.function != f {
		return ast.ErrASTDomain
	}
	rule.removing = true
	r.rules[quantifier] = rule
	for _, offset := range []int{1, 2, 0} {
		at := rule.ownedStart + offset
		if at >= len(r.owned) {
			return ast.ErrASTDomain
		}
		h := r.owned[at]
		if h.IsNull() {
			continue
		}
		consumed, e := r.manager.DecRefConsumed(h)
		if consumed {
			r.owned[at] = ast.ASTHandle{}
		}
		if e != nil {
			return e
		}
		if !consumed {
			return ast.ErrASTPending
		}
	}
	delete(r.rules, quantifier)
	for i, q := range set.values {
		if q == quantifier {
			copy(set.values[i:], set.values[i+1:])
			set.values[len(set.values)-1] = ast.ASTHandle{}
			set.values = set.values[:len(set.values)-1]
			return nil
		}
	}
	return ast.ErrASTDomain
}

func (r *rewriteRuleIndex) consistent() bool {
	if r.check() != nil {
		return false
	}
	for f, set := range r.byFunc {
		if set == nil {
			return false
		}
		for _, q := range set.values {
			rule, found := r.rules[q]
			if !found || rule.removing || rule.function != f {
				return false
			}
		}
	}
	return true
}

// tryRewrite is rewrite1: it skips arity mismatches and tries each indexed
// rule until the concrete matcher produces an owned rewritten expression.
func (r *rewriteRuleIndex) tryRewrite(f ast.ASTHandle, args []ast.ASTHandle) (ast.ASTRef, bool, error) {
	if e := r.check(); e != nil {
		return ast.ASTRef{}, false, e
	}
	set := r.byFunc[f]
	if set == nil {
		return ast.ASTRef{}, false, nil
	}
	for _, q := range set.values {
		rule, found := r.rules[q]
		if !found {
			if enabled, e := forwardAssertionEnabled(); e != nil {
				return ast.ASTRef{}, false, e
			} else if enabled {
				if e := forwardAssertionFailure(297, "m_demodulator2lhs_rhs.contains(d)"); e != nil {
					return ast.ASTRef{}, false, e
				}
			}
			return ast.ASTRef{}, false, ast.ErrASTDomain
		}
		if rule.removing {
			return ast.ASTRef{}, false, ast.ErrASTPending
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
			if enabled, e := forwardAssertionEnabled(); e != nil {
				return ast.ASTRef{}, false, e
			} else if enabled {
				if e := forwardAssertionFailure(307, "large->get_decl() == f"); e != nil {
					return ast.ASTRef{}, false, e
				}
			}
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
	r.closing = true
	if e := r.matcher.close(); e != nil {
		return e
	}
	for f, set := range r.byFunc {
		r.releaseForwardSet(set)
		delete(r.byFunc, f)
	}
	for len(r.owned) != 0 {
		h := r.owned[0]
		if h.IsNull() {
			r.owned = r.owned[1:]
			continue
		}
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
