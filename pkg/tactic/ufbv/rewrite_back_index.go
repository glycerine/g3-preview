// Source: src/tactic/ufbv/ufbv_rewriter.cpp:468-537.
package ufbv

import "github.com/glycerine/g3/pkg/ast"

// rewriteBackIndex is the concrete UFBV m_back_idx/m_processed/m_todo state.
// Forward and backward object sets use native Go maps of AST identities.
type rewriteBackIndex struct {
	manager   *ast.ASTManager
	byDecl    map[ast.ASTHandle]map[ast.ASTHandle]struct{}
	processed map[ast.ASTHandle]struct{}
	todo      []ast.ASTHandle
	owned     []ast.ASTHandle
	closed    bool
}

func newRewriteBackIndex(m *ast.ASTManager) (*rewriteBackIndex, error) {
	if m == nil {
		return nil, ast.ErrASTHandle
	}
	return &rewriteBackIndex{
		manager:   m,
		byDecl:    make(map[ast.ASTHandle]map[ast.ASTHandle]struct{}),
		processed: make(map[ast.ASTHandle]struct{}),
	}, nil
}

func (r *rewriteBackIndex) check() error {
	if r == nil || r.manager == nil || r.closed {
		return ast.ErrASTHandle
	}
	return nil
}

// declarations visits app arguments and quantifier annotations/body as the
// source for_each_expr call does. Only nonconstant uninterpreted applications
// contribute declarations to m_back_idx.
func (r *rewriteBackIndex) declarations(root ast.ASTHandle) (map[ast.ASTHandle]struct{}, error) {
	stack := []ast.ASTHandle{root}
	seen := make(map[ast.ASTHandle]struct{})
	decls := make(map[ast.ASTHandle]struct{})
	for len(stack) != 0 {
		i := len(stack) - 1
		h := stack[i]
		stack = stack[:i]
		if _, present := seen[h]; present {
			continue
		}
		seen[h] = struct{}{}
		kind, e := r.manager.GetKind(h)
		if e != nil {
			return nil, e
		}
		switch kind {
		case ast.ASTVar:
		case ast.ASTApp:
			n, e := r.manager.GetNumArgs(h)
			if e != nil {
				return nil, e
			}
			if n > 0 {
				family, e := r.manager.GetFamilyID(h)
				if e != nil {
					return nil, e
				}
				if family == ast.NullFamilyID {
					f, e := r.manager.GetDecl(h)
					if e != nil {
						return nil, e
					}
					decls[f] = struct{}{}
				}
			}
			for j := n; j > 0; j-- {
				child, e := r.manager.GetArg(h, j-1)
				if e != nil {
					return nil, e
				}
				stack = append(stack, child)
			}
		case ast.ASTQuantifier:
			body, e := r.manager.GetQuantifierBody(h)
			if e != nil {
				return nil, e
			}
			stack = append(stack, body)
			n, e := r.manager.GetNumPatterns(h)
			if e != nil {
				return nil, e
			}
			for j := n; j > 0; j-- {
				child, e := r.manager.GetPattern(h, j-1)
				if e != nil {
					return nil, e
				}
				stack = append(stack, child)
			}
			n, e = r.manager.GetNumNoPatterns(h)
			if e != nil {
				return nil, e
			}
			for j := n; j > 0; j-- {
				child, e := r.manager.GetNoPattern(h, j-1)
				if e != nil {
					return nil, e
				}
				stack = append(stack, child)
			}
		default:
			return nil, ast.ErrASTDomain
		}
	}
	return decls, nil
}

func (r *rewriteBackIndex) addProcessed(expr ast.ASTHandle) error {
	if e := r.check(); e != nil {
		return e
	}
	decls, e := r.declarations(expr)
	if e != nil {
		return e
	}
	if _, present := r.processed[expr]; present {
		return nil
	}
	// The source borrows m_processed roots from the enclosing rewriter's
	// argument/result owners. This concrete owner retains them until queued.
	if e := r.manager.IncRef(expr); e != nil {
		return e
	}
	r.owned = append(r.owned, expr)
	r.processed[expr] = struct{}{}
	for f := range decls {
		set := r.byDecl[f]
		if set == nil {
			set = make(map[ast.ASTHandle]struct{})
			r.byDecl[f] = set
		}
		set[expr] = struct{}{}
	}
	return nil
}

// rescheduleProcessed copies the selected f-occurrence set before removing
// all back-index memberships, since removal may mutate that same set.
func (r *rewriteBackIndex) rescheduleProcessed(f ast.ASTHandle) error {
	if e := r.check(); e != nil {
		return e
	}
	selected := make([]ast.ASTHandle, 0, len(r.byDecl[f]))
	for expr := range r.byDecl[f] {
		if _, active := r.processed[expr]; active {
			selected = append(selected, expr)
		}
	}
	for _, expr := range selected {
		decls, e := r.declarations(expr)
		if e != nil {
			return e
		}
		// m_todo takes a reference before m_processed's concrete owner is
		// retired, so the queued expression remains live across transfer.
		if e := r.manager.IncRef(expr); e != nil {
			return e
		}
		r.todo = append(r.todo, expr)
		delete(r.processed, expr)
		for d := range decls {
			delete(r.byDecl[d], expr)
		}
		for i, h := range r.owned {
			if h == expr {
				r.owned[i] = ast.ASTHandle{}
				consumed, e := r.manager.DecRefConsumed(h)
				if !consumed {
					r.owned[i] = h
				}
				if e != nil {
					return e
				}
				if !consumed {
					return ast.ErrASTPending
				}
				break
			}
		}
	}
	return nil
}

func (r *rewriteBackIndex) close() error {
	if r == nil || r.closed {
		return nil
	}
	for len(r.todo) != 0 {
		h := r.todo[0]
		consumed, e := r.manager.DecRefConsumed(h)
		if consumed {
			r.todo[0] = ast.ASTHandle{}
			r.todo = r.todo[1:]
		}
		if e != nil {
			return e
		}
		if !consumed {
			return ast.ErrASTPending
		}
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
	r.byDecl = nil
	r.processed = nil
	r.closed = true
	return nil
}
