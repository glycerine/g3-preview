// Source: src/ast/pattern/pattern_inference.cpp:31-93.
package pattern

import "github.com/glycerine/g3/pkg/ast"

type expressionPair struct{ left, right ast.ASTHandle }

// smallerPattern is the concrete borrowed expression-pair cache used while
// pattern inference compares two candidate patterns. The AST manager owns the
// expressions; this matcher retains no references after a call.
type smallerPattern struct {
	manager  *ast.ASTManager
	bindings []ast.ASTHandle
	cache    map[expressionPair]struct{}
	todo     []expressionPair
}

func newSmallerPattern(m *ast.ASTManager) (*smallerPattern, error) {
	if m == nil {
		return nil, ast.ErrASTHandle
	}
	return &smallerPattern{manager: m, cache: make(map[expressionPair]struct{})}, nil
}

func (s *smallerPattern) save(left, right ast.ASTHandle) {
	pair := expressionPair{left, right}
	if _, seen := s.cache[pair]; seen {
		return
	}
	s.cache[pair] = struct{}{}
	s.todo = append(s.todo, pair)
}

// smaller implements smaller_pattern::operator(). Pattern variables below
// numBindings bind once to any instance expression; external variables must
// retain their exact AST identity.
func (s *smallerPattern) smaller(numBindings uint32, left, right ast.ASTHandle) (bool, error) {
	if s == nil || s.manager == nil {
		return false, ast.ErrASTHandle
	}
	if uint64(numBindings) > uint64(^uint(0)>>1) {
		return false, ast.ErrASTDomain
	}
	if cap(s.bindings) < int(numBindings) {
		s.bindings = make([]ast.ASTHandle, numBindings)
	} else {
		s.bindings = s.bindings[:numBindings]
		clear(s.bindings)
	}
	clear(s.cache)
	s.todo = s.todo[:0]
	s.save(left, right)
	for len(s.todo) != 0 {
		i := len(s.todo) - 1
		pair := s.todo[i]
		s.todo[i] = expressionPair{}
		s.todo = s.todo[:i]
		leftKind, err := s.manager.GetKind(pair.left)
		if err != nil {
			return false, err
		}
		if leftKind != ast.ASTVar {
			rightKind, err := s.manager.GetKind(pair.right)
			if err != nil {
				return false, err
			}
			if leftKind != rightKind {
				return false, nil
			}
		}
		switch leftKind {
		case ast.ASTApp:
			leftCount, err := s.manager.GetNumArgs(pair.left)
			if err != nil {
				return false, err
			}
			rightCount, err := s.manager.GetNumArgs(pair.right)
			if err != nil {
				return false, err
			}
			leftDecl, err := s.manager.GetDecl(pair.left)
			if err != nil {
				return false, err
			}
			rightDecl, err := s.manager.GetDecl(pair.right)
			if err != nil {
				return false, err
			}
			if leftCount != rightCount || leftDecl != rightDecl {
				return false, nil
			}
			for j := uint32(0); j < leftCount; j++ {
				childLeft, err := s.manager.GetArg(pair.left, j)
				if err != nil {
					return false, err
				}
				childRight, err := s.manager.GetArg(pair.right, j)
				if err != nil {
					return false, err
				}
				s.save(childLeft, childRight)
			}
		case ast.ASTVar:
			index, err := s.manager.GetVarIndex(pair.left)
			if err != nil {
				return false, err
			}
			if index < numBindings {
				if s.bindings[index].IsNull() {
					s.bindings[index] = pair.right
				} else if s.bindings[index] != pair.right {
					return false, nil
				}
			} else if pair.left != pair.right {
				return false, nil
			}
		default:
			if pair.left != pair.right {
				return false, nil
			}
		}
	}
	return true, nil
}
