// Source: src/tactic/ufbv/ufbv_rewriter.cpp:759-902,919-927.
package ufbv

import "github.com/glycerine/g3/pkg/ast"

type matchPair struct{ pattern, instance ast.ASTHandle }

// matchSubst is the concrete expression-pair cache and variable binding state
// used by UFBV's instance check. The AST manager owns every borrowed handle.
type matchSubst struct {
	manager  *ast.ASTManager
	cache    map[matchPair]struct{}
	bindings map[ast.ASTHandle]ast.ASTHandle
	todo     []matchPair
	allEqual bool
}

func newMatchSubst(m *ast.ASTManager) (*matchSubst, error) {
	if m == nil {
		return nil, ast.ErrASTDomain
	}
	return &matchSubst{manager: m, cache: make(map[matchPair]struct{}), bindings: make(map[ast.ASTHandle]ast.ASTHandle)}, nil
}

func (s *matchSubst) sameDecl(a, b ast.ASTHandle) (bool, error) {
	da, e := s.manager.GetDecl(a)
	if e != nil {
		return false, e
	}
	db, e := s.manager.GetDecl(b)
	return da == db, e
}

// bindIdentical traverses an equal, non-ground application. A variable may be
// left unbound or already bound to itself at offset 1; another binding fails.
func (s *matchSubst) bindIdentical(root ast.ASTHandle) (bool, error) {
	stack := []ast.ASTHandle{root}
	seen := make(map[ast.ASTHandle]struct{})
	for len(stack) != 0 {
		i := len(stack) - 1
		h := stack[i]
		stack = stack[:i]
		if _, done := seen[h]; done {
			continue
		}
		seen[h] = struct{}{}
		kind, e := s.manager.GetKind(h)
		if e != nil {
			return false, e
		}
		switch kind {
		case ast.ASTVar:
			if bound, found := s.bindings[h]; found && bound != h {
				return false, nil
			}
			s.bindings[h] = h
		case ast.ASTQuantifier:
			return false, nil
		case ast.ASTApp:
			count, e := s.manager.GetNumArgs(h)
			if e != nil {
				return false, e
			}
			for j := count; j > 0; j-- {
				child, e := s.manager.GetArg(h, j-1)
				if e != nil {
					return false, e
				}
				stack = append(stack, child)
			}
		default:
			return false, ast.ErrASTDomain
		}
	}
	return true, nil
}

// matchArgs follows match_subst::match_args, including its quick equal and
// quick declaration-failure paths. args are the proposed instance arguments.
func (s *matchSubst) matchArgs(lhs ast.ASTHandle, args []ast.ASTHandle) (bool, error) {
	if s == nil || s.manager == nil {
		return false, ast.ErrASTDomain
	}
	clear(s.cache)
	s.todo = s.todo[:0]
	count, e := s.manager.GetNumArgs(lhs)
	if e != nil {
		return false, e
	}
	if int(count) != len(args) {
		return false, ast.ErrASTDomain
	}
	s.allEqual = true
	for i, instance := range args {
		pattern, e := s.manager.GetArg(lhs, uint32(i))
		if e != nil {
			return false, e
		}
		if pattern != instance {
			s.allEqual = false
		}
		pk, e := s.manager.GetKind(pattern)
		if e != nil {
			return false, e
		}
		ik, e := s.manager.GetKind(instance)
		if e != nil {
			return false, e
		}
		if pk == ast.ASTApp && ik == ast.ASTApp {
			equal, e := s.sameDecl(pattern, instance)
			if e != nil || !equal {
				return false, e
			}
		}
		s.todo = append(s.todo, matchPair{pattern, instance})
	}
	if s.allEqual {
		return true, nil
	}
	clear(s.bindings)
	for len(s.todo) != 0 {
		i := len(s.todo) - 1
		p := s.todo[i]
		pk, e := s.manager.GetKind(p.pattern)
		if e != nil {
			return false, e
		}
		if pk == ast.ASTVar {
			if bound, found := s.bindings[p.pattern]; found && bound != p.instance {
				return false, nil
			}
			s.bindings[p.pattern] = p.instance
			s.todo = s.todo[:i]
			continue
		}
		ik, e := s.manager.GetKind(p.instance)
		if e != nil {
			return false, e
		}
		if ik == ast.ASTVar || pk == ast.ASTQuantifier || ik == ast.ASTQuantifier {
			return false, nil
		}
		if pk != ast.ASTApp || ik != ast.ASTApp {
			return false, ast.ErrASTDomain
		}
		groundPattern, e := s.manager.IsGround(p.pattern)
		if e != nil {
			return false, e
		}
		groundInstance, e := s.manager.IsGround(p.instance)
		if e != nil {
			return false, e
		}
		if groundPattern && !groundInstance {
			return false, nil
		}
		if p.pattern == p.instance && groundPattern {
			s.todo = s.todo[:i]
			continue
		}
		if _, seen := s.cache[p]; seen {
			s.todo = s.todo[:i]
			continue
		}
		if p.pattern == p.instance {
			ok, e := s.bindIdentical(p.pattern)
			if e != nil || !ok {
				return false, e
			}
			s.todo = s.todo[:i]
			s.cache[p] = struct{}{}
			continue
		}
		equal, e := s.sameDecl(p.pattern, p.instance)
		if e != nil || !equal {
			return false, e
		}
		pn, e := s.manager.GetNumArgs(p.pattern)
		if e != nil {
			return false, e
		}
		in, e := s.manager.GetNumArgs(p.instance)
		if e != nil {
			return false, e
		}
		if pn != in {
			return false, nil
		}
		s.todo = s.todo[:i]
		if pn == 0 {
			continue
		}
		s.cache[p] = struct{}{}
		for j := pn; j > 0; j-- {
			left, e := s.manager.GetArg(p.pattern, j-1)
			if e != nil {
				return false, e
			}
			right, e := s.manager.GetArg(p.instance, j-1)
			if e != nil {
				return false, e
			}
			s.todo = append(s.todo, matchPair{left, right})
		}
	}
	return true, nil
}

// isInstance is match_subst::operator()(expr*,expr*), which only checks
// matching and does not construct a rewritten RHS.
func (s *matchSubst) isInstance(pattern, instance ast.ASTHandle) (bool, error) {
	if s == nil || s.manager == nil {
		return false, ast.ErrASTDomain
	}
	clear(s.cache)
	s.todo = s.todo[:0]
	pk, e := s.manager.GetKind(pattern)
	if e != nil {
		return false, e
	}
	if pk == ast.ASTVar {
		return true, nil
	}
	ik, e := s.manager.GetKind(instance)
	if e != nil {
		return false, e
	}
	if pk != ast.ASTApp || ik != ast.ASTApp {
		return false, nil
	}
	equal, e := s.sameDecl(pattern, instance)
	if e != nil || !equal {
		return false, e
	}
	pn, e := s.manager.GetNumArgs(pattern)
	if e != nil {
		return false, e
	}
	in, e := s.manager.GetNumArgs(instance)
	if e != nil || pn != in {
		return false, e
	}
	args := make([]ast.ASTHandle, in)
	for i := range args {
		args[i], e = s.manager.GetArg(instance, uint32(i))
		if e != nil {
			return false, e
		}
	}
	return s.matchArgs(pattern, args)
}
