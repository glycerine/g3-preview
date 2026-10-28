package pdr

// Source: context::solve_impl, check_reachability, expand_node,
// expand_state, propagate, close_node, and the child-obligation portion of
// create_children in src/muz/pdr/pdr_context.cpp.

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func (c *Context) closeNode(node *ModelNode) error {
	if err := node.SetClosed(); err != nil {
		return err
	}
	for parent := node.parent; parent != nil && parent.IsOneClosed(); parent = parent.parent {
		if err := parent.SetClosed(); err != nil {
			return err
		}
	}
	return nil
}

func (c *Context) createChildrenExact(node *ModelNode) (resultErr error) {
	modelCore, found, err := node.Model()
	if err != nil {
		return err
	}
	if !found {
		return ast.ErrASTDomain
	}
	rule, err := node.transformer.selectRule(modelCore)
	if err != nil {
		return err
	}
	if err = node.SetRule(rule.rule); err != nil {
		return err
	}
	out, err := c.manager.NewRef(ast.ASTHandle{})
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, c.manager.DelRef(&out)) }()
	references := make([]ast.ASTRef, 0)
	defer func() {
		for index := len(references) - 1; index >= 0; index-- {
			resultErr = errors.Join(resultErr, c.manager.DelRef(&references[index]))
		}
	}()
	for predecessorIndex, predecessor := range rule.predecessors {
		equalities := make([]ast.ASTHandle, 0, len(predecessor.signature))
		for _, signature := range predecessor.signature {
			oldDeclaration, x := c.owner.OldToOld(signature, 0, uint32(predecessorIndex))
			if x != nil {
				return x
			}
			oldState, x := c.manager.MkApp(oldDeclaration, nil)
			if x != nil {
				return x
			}
			found, x := predecessor.modelConstant(modelCore, oldState, out)
			if x != nil {
				return x
			}
			if !found {
				return ast.ErrASTDomain
			}
			value, x := c.manager.RefGet(out)
			if x != nil {
				return x
			}
			newDeclaration, x := c.owner.OldToNew(signature, 0)
			if x != nil {
				return x
			}
			newState, x := c.manager.MkApp(newDeclaration, nil)
			if x != nil {
				return x
			}
			equality, x := c.manager.MkEq(newState, value)
			if x != nil {
				return x
			}
			reference, x := c.manager.NewRef(equality)
			if x != nil {
				return x
			}
			references = append(references, reference)
			equalities = append(equalities, equality)
		}
		cube, x := c.manager.MkAndArray(equalities)
		if x != nil {
			return x
		}
		child, x := NewModelNode(node, cube, predecessor, node.level-1)
		if x != nil {
			return x
		}
		c.nodes++
		if child.depth > c.maxDepth {
			c.maxDepth = child.depth
		}
		if x = c.search.AddLeaf(child); x != nil {
			return x
		}
	}
	return node.CheckPreClosed()
}

func (c *Context) expandNode(node *ModelNode) (resultStatus lbool.LBool, resultErr error) {
	if node == nil || !node.IsOpen() {
		return lbool.Undef, ast.ErrASTDomain
	}
	if node.level < c.expanded {
		c.expanded = node.level
	}
	cached, err := node.transformer.IsCachedReachable(node.state)
	if err != nil {
		return lbool.Undef, err
	}
	if cached {
		return lbool.True, c.closeNode(node)
	}
	check, err := node.transformer.CheckReachable(node.level, node.state, true, true)
	if err != nil {
		return lbool.Undef, err
	}
	if check.Status == lbool.True {
		if check.Model == nil {
			return lbool.Undef, ast.ErrASTDomain
		}
		if err = node.SetModel(check.Model); err != nil {
			return lbool.Undef, errors.Join(err, check.Model.DecRef())
		}
		if err = check.Model.DecRef(); err != nil {
			return lbool.Undef, err
		}
		if node.level == 0 {
			return lbool.True, c.closeNode(node)
		}
		return lbool.True, c.createChildrenExact(node)
	}
	if check.Status == lbool.False {
		lemma, x := c.owner.NotAnd(check.Core)
		if x != nil {
			return lbool.Undef, x
		}
		level := InfinityLevel
		if check.AssumesLevel {
			level = node.level
		}
		if _, x = node.transformer.AddProperty(lemma, level); x != nil {
			return lbool.Undef, x
		}
		if x = c.search.BacktrackLevel(false, node); x != nil {
			return lbool.Undef, x
		}
		return lbool.False, nil
	}
	return lbool.Undef, nil
}

func (c *Context) checkReachability(level uint32) (bool, error) {
	truth, err := c.manager.MkTrue()
	if err != nil {
		return false, err
	}
	root, err := NewModelNode(nil, truth, c.query, level)
	if err != nil {
		return false, err
	}
	if err = c.search.SetRoot(root); err != nil {
		return false, err
	}
	for {
		node, found, x := c.search.Next()
		if x != nil {
			return false, x
		}
		if !found {
			break
		}
		status, x := c.expandNode(node)
		if x != nil {
			return false, x
		}
		if status == lbool.Undef {
			return false, nil
		}
	}
	return root.IsClosed(), nil
}

func (c *Context) propagate(maximum uint32) (bool, error) {
	for level := c.expanded; level <= maximum; level++ {
		all := true
		for _, declaration := range c.order {
			empty, err := c.transformers[declaration].PropagateToNextLevel(level)
			if err != nil {
				return false, err
			}
			all = empty && all
		}
		if all && level == maximum {
			c.inductive = level
			return true, nil
		}
		if level == InfinityLevel {
			break
		}
	}
	return false, nil
}

// Solve runs the retained frame-expansion and propagation loop for the query
// predicate. Child projection currently uses exact model values; native
// implicant minimization and QE generalization are added on top of this loop.
func (c *Context) Solve(queryDeclaration ast.ASTHandle) (lbool.LBool, error) {
	if err := c.open(); err != nil {
		return lbool.Undef, err
	}
	if !c.initialized || c.search == nil {
		return lbool.Undef, ast.ErrASTDomain
	}
	query := c.transformers[queryDeclaration]
	if query == nil {
		c.lastResult = lbool.False
		return c.lastResult, nil
	}
	c.query = query
	c.lastResult = lbool.Undef
	c.expanded, c.inductive = 0, 0
	for level := uint32(0); ; level++ {
		c.expanded = level
		reachable, err := c.checkReachability(level)
		if err != nil {
			return lbool.Undef, err
		}
		if reachable {
			c.lastResult = lbool.True
			return c.lastResult, nil
		}
		if level != 0 {
			fixed, x := c.propagate(level)
			if x != nil {
				return lbool.Undef, x
			}
			if fixed {
				c.lastResult = lbool.False
				for _, declaration := range c.order {
					if c.transformers[declaration] != c.query && c.inductive > 0 {
						if x = c.transformers[declaration].PropagateToInfinity(c.inductive); x != nil {
							return lbool.Undef, x
						}
					}
				}
				return c.lastResult, nil
			}
		}
		if level == InfinityLevel-1 {
			return lbool.Undef, nil
		}
	}
}

func (c *Context) LastResult() (lbool.LBool, error) {
	if err := c.open(); err != nil {
		return lbool.Undef, err
	}
	return c.lastResult, nil
}
