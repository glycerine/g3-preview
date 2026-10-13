// Source: src/ast/pattern/pattern_inference.cpp:326-393.
package pattern

import "github.com/glycerine/g3/pkg/ast"

func (c *patternCandidates) saveSubpattern(n ast.ASTHandle) {
	if _, seen := c.visited[n]; seen {
		return
	}
	c.todo = append(c.todo, n)
	c.visited[n] = struct{}{}
}

// containsSubpattern searches the root's DAG in last-child-first order. A
// proper subapplication is relevant only when it remains in the candidate
// metadata map and has the same free-variable set as the root.
func (c *patternCandidates) containsSubpattern(root ast.ASTHandle) (bool, error) {
	if c == nil || c.closed {
		return false, ast.ErrASTHandle
	}
	rootInfo, found := c.info[root]
	if !found {
		return false, ast.ErrASTDomain
	}
	clear(c.visited)
	c.todo = c.todo[:0]
	c.saveSubpattern(root)
	for len(c.todo) != 0 {
		i := len(c.todo) - 1
		curr := c.todo[i]
		c.todo[i] = ast.ASTHandle{}
		c.todo = c.todo[:i]
		kind, err := c.manager.GetKind(curr)
		if err != nil {
			return false, err
		}
		switch kind {
		case ast.ASTApp:
			if curr != root {
				if info, found := c.info[curr]; found && sameFreeVars(rootInfo.freeVars, info.freeVars) {
					return true, nil
				}
			}
			count, err := c.manager.GetNumArgs(curr)
			if err != nil {
				return false, err
			}
			for j := uint32(0); j < count; j++ {
				child, err := c.manager.GetArg(curr, j)
				if err != nil {
					return false, err
				}
				c.saveSubpattern(child)
			}
		case ast.ASTVar:
		default:
			return false, ast.ErrASTDomain
		}
	}
	return false, nil
}

// filterBigger is pattern_inference_cfg::filter_bigger_patterns. It borrows
// the supplied candidates and returns them in order, omitting those that
// contain a same-variable candidate subapplication.
func (c *patternCandidates) filterBigger(patterns []ast.ASTHandle) ([]ast.ASTHandle, error) {
	if c == nil || c.closed {
		return nil, ast.ErrASTHandle
	}
	result := make([]ast.ASTHandle, 0, len(patterns))
	for _, curr := range patterns {
		bigger, err := c.containsSubpattern(curr)
		if err != nil {
			return nil, err
		}
		if !bigger {
			result = append(result, curr)
		}
	}
	return result, nil
}
