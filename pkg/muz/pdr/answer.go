package pdr

// Source: model_search::get_trace, context::get_answer and
// context::mk_sat_answer and context::mk_unsat_answer in
// src/muz/pdr/pdr_context.cpp, plus inductive_property::to_expr in
// src/muz/pdr/pdr_manager.cpp.

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/ast/rewriter"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func (c *Context) instantiateTraceExpression(node *ModelNode, rule *normalizedRule, expression ast.ASTHandle) (result ast.ASTRef, resultErr error) {
	modelCore, found, err := node.Model()
	if err != nil {
		return result, err
	}
	if !found {
		return result, ast.ErrASTDomain
	}
	used, err := ast.NewASTUsedVars(c.manager)
	if err != nil {
		return result, err
	}
	if err = used.Collect(expression); err != nil {
		return result, err
	}
	arguments := make([]ast.ASTHandle, len(rule.instances))
	references := make([]ast.ASTRef, 0, len(arguments))
	defer func() {
		for index := len(references) - 1; index >= 0; index-- {
			resultErr = errors.Join(resultErr, c.manager.DelRef(&references[index]))
		}
		if resultErr != nil && result != (ast.ASTRef{}) {
			resultErr = errors.Join(resultErr, c.manager.DelRef(&result))
			result = ast.ASTRef{}
		}
	}()
	valueRef, err := c.manager.NewRef(ast.ASTHandle{})
	if err != nil {
		return result, err
	}
	defer func() { resultErr = errors.Join(resultErr, c.manager.DelRef(&valueRef)) }()
	for index, instance := range rule.instances {
		if instance.IsNull() {
			sort := used.Contains(uint32(index))
			if sort.IsNull() {
				continue
			}
			arguments[index], err = c.manager.MkVar(uint32(index), sort)
		} else {
			var available bool
			available, err = node.transformer.modelConstant(modelCore, instance, valueRef)
			if err == nil && !available {
				err = ast.ErrASTDomain
			}
			if err == nil {
				arguments[index], err = c.manager.RefGet(valueRef)
			}
		}
		if err != nil {
			return result, err
		}
		if arguments[index].IsNull() {
			continue
		}
		reference, x := c.manager.NewRef(arguments[index])
		if x != nil {
			return result, x
		}
		references = append(references, reference)
	}
	substitution, err := rewriter.NewVarSubstitution(c.manager, false)
	if err != nil {
		return result, err
	}
	defer func() { resultErr = errors.Join(resultErr, substitution.Close()) }()
	result, err = c.manager.NewRef(ast.ASTHandle{})
	if err != nil {
		return result, err
	}
	if err = substitution.Substitute(expression, arguments, result); err != nil {
		return result, err
	}
	return result, nil
}

func (c *Context) selectedTraceRule(node *ModelNode) (*normalizedRule, error) {
	if node.selected != nil {
		return node.selected, nil
	}
	initial := make([]*normalizedRule, 0)
	tags := make([]ast.ASTHandle, 0)
	for _, rule := range node.transformer.rules {
		if len(rule.predecessors) != 0 {
			continue
		}
		initial = append(initial, rule)
		if !rule.tag.IsNull() {
			tags = append(tags, rule.tag)
		}
	}
	if len(initial) == 1 {
		return initial[0], nil
	}
	if len(initial) == 0 || len(tags) != len(initial) {
		return nil, ast.ErrASTDomain
	}
	selection, err := c.manager.MkOrArray(tags)
	if err != nil {
		return nil, err
	}
	check, err := node.transformer.solver.CheckAtLevel(0, []ast.ASTHandle{selection, node.state}, false, true)
	if err != nil {
		return nil, err
	}
	if check.Status != lbool.True || check.Model == nil {
		if check.Model != nil {
			err = check.Model.DecRef()
		}
		return nil, errors.Join(ast.ErrASTDomain, err)
	}
	rule, selectErr := node.transformer.selectInitialRule(check.Model)
	if selectErr == nil {
		selectErr = node.SetModel(check.Model)
	}
	if selectErr == nil {
		selectErr = node.SetRule(rule.rule)
	}
	if selectErr == nil {
		node.selected = rule
	}
	selectErr = errors.Join(selectErr, check.Model.DecRef())
	if selectErr != nil {
		return nil, selectErr
	}
	return rule, nil
}

func traceTransformerReaches(current, target *PredicateTransformer, seen map[*PredicateTransformer]bool) bool {
	if seen[current] {
		return false
	}
	seen[current] = true
	for _, rule := range current.rules {
		for _, predecessor := range rule.predecessors {
			if predecessor == target || traceTransformerReaches(predecessor, target, seen) {
				return true
			}
		}
	}
	return false
}

func traceRecursiveAnchor(root *ModelNode) (*PredicateTransformer, error) {
	stack := []*ModelNode{root}
	for len(stack) != 0 {
		last := len(stack) - 1
		node := stack[last]
		stack = stack[:last]
		if traceTransformerReaches(node.transformer, node.transformer, make(map[*PredicateTransformer]bool)) {
			return node.transformer, nil
		}
		children, err := node.Children()
		if err != nil {
			return nil, err
		}
		stack = append(stack, children...)
	}
	return nil, nil
}

func traceHasNestedPredecessor(root *ModelNode) (bool, error) {
	children, err := root.Children()
	if err != nil {
		return false, err
	}
	for _, child := range children {
		grandchildren, x := child.Children()
		if x != nil {
			return false, x
		}
		if len(grandchildren) != 0 {
			return true, nil
		}
	}
	return false, nil
}

func updateTraceModels(root *ModelNode) error {
	byState := make(map[ast.ASTHandle]*ModelNode)
	nodes := []*ModelNode{root}
	for index := 0; index < len(nodes); index++ {
		node := nodes[index]
		_, found, err := node.Model()
		if err != nil {
			return err
		}
		if found {
			byState[node.state] = node
		}
		children, err := node.Children()
		if err != nil {
			return err
		}
		nodes = append(nodes, children...)
	}
	for _, node := range nodes {
		_, found, err := node.Model()
		if err != nil {
			return err
		}
		if found {
			continue
		}
		donor := byState[node.state]
		if donor == nil {
			return ast.ErrASTDomain
		}
		modelCore, _, err := donor.Model()
		if err != nil {
			return err
		}
		if err = node.SetModel(modelCore); err != nil {
			return err
		}
		node.selected = donor.selected
	}
	return nil
}

// traceHasNonlinearRule identifies witnesses for which the retained native
// fixedpoint frontend reduces the printed trace to true. The linear recursive
// trace below must not try to recover models or select initial rules for every
// branch of such a witness.
func traceHasNonlinearRule(root *ModelNode) (bool, error) {
	stack := []*ModelNode{root}
	for len(stack) != 0 {
		last := len(stack) - 1
		node := stack[last]
		stack = stack[:last]
		if node.selected != nil {
			uninterpreted, err := node.selected.rule.UninterpretedTailSize()
			if err != nil {
				return false, err
			}
			if uninterpreted > 1 {
				return true, nil
			}
		}
		children, err := node.Children()
		if err != nil {
			return false, err
		}
		stack = append(stack, children...)
	}
	return false, nil
}

func (c *Context) unsatAnswer(queryDeclaration ast.ASTHandle) (result ast.ASTRef, resultErr error) {
	references := make([]ast.ASTRef, 0)
	defer func() {
		for index := len(references) - 1; index >= 0; index-- {
			resultErr = errors.Join(resultErr, c.manager.DelRef(&references[index]))
		}
		if resultErr != nil && result != (ast.ASTRef{}) {
			resultErr = errors.Join(resultErr, c.manager.DelRef(&result))
			result = ast.ASTRef{}
		}
	}()
	pin := func(handle ast.ASTHandle) error {
		reference, err := c.manager.NewRef(handle)
		if err == nil {
			references = append(references, reference)
		}
		return err
	}
	abstractor, err := ast.NewExprAbstractor(c.manager)
	if err != nil {
		return result, err
	}
	defer func() { resultErr = errors.Join(resultErr, abstractor.Close()) }()
	definitions := make([]ast.ASTHandle, 0, len(c.order))
	for _, declaration := range c.order {
		if declaration == queryDeclaration || c.auxiliary[declaration] {
			continue
		}
		transformer := c.transformers[declaration]
		body, x := transformer.Formulas(c.inductive, false)
		if x != nil {
			return result, x
		}
		if x = pin(body); x != nil {
			return result, x
		}
		body, x = c.owner.FormulaNewToOld(body, 0, false)
		if x != nil {
			return result, x
		}
		if x = pin(body); x != nil {
			return result, x
		}
		arguments := make([]ast.ASTHandle, len(transformer.signature))
		for index, signature := range transformer.signature {
			arguments[index], x = c.manager.MkApp(signature, nil)
			if x != nil {
				return result, x
			}
			if x = pin(arguments[index]); x != nil {
				return result, x
			}
		}
		application, x := c.manager.MkApp(declaration, arguments)
		if x != nil {
			return result, x
		}
		if x = pin(application); x != nil {
			return result, x
		}
		equality, x := c.manager.MkEq(application, body)
		if x != nil {
			return result, x
		}
		if x = pin(equality); x != nil {
			return result, x
		}
		definition, x := c.manager.NewRef(ast.ASTHandle{})
		if x != nil {
			return result, x
		}
		references = append(references, definition)
		if x = abstractor.MkForall(arguments, equality, definition); x != nil {
			return result, x
		}
		definitionHandle, x := c.manager.RefGet(definition)
		if x != nil {
			return result, x
		}
		definitions = append(definitions, definitionHandle)
	}
	formula, err := c.manager.MkAndArray(definitions)
	if err != nil {
		return result, err
	}
	return c.manager.NewRef(formula)
}

// Answer reconstructs the selected satisfying rule trace or the direct
// predicate inductive certificate for the latest query and returns one owning
// reference. The caller releases it through ASTManager.DelRef.
func (c *Context) Answer(queryDeclaration ast.ASTHandle) (result ast.ASTRef, resultErr error) {
	if err := c.open(); err != nil {
		return result, err
	}
	if c.queryDecl != queryDeclaration {
		return result, ast.ErrASTDomain
	}
	if c.lastResult == lbool.False {
		return c.unsatAnswer(queryDeclaration)
	}
	if c.lastResult != lbool.True || c.query == nil || c.query.head != queryDeclaration {
		return result, ast.ErrASTDomain
	}
	root, found, err := c.search.Root()
	if err != nil || !found {
		if err != nil {
			return result, err
		}
		return result, ast.ErrASTDomain
	}
	nonlinear, err := traceHasNonlinearRule(root)
	if err != nil {
		return result, err
	}
	if nonlinear {
		truth, x := c.manager.MkTrue()
		if x != nil {
			return result, x
		}
		return c.manager.NewRef(truth)
	}
	if err = updateTraceModels(root); err != nil {
		return result, err
	}
	anchor, err := traceRecursiveAnchor(root)
	if err != nil {
		return result, err
	}
	linearPredicates, err := traceHasNestedPredecessor(root)
	if err != nil {
		return result, err
	}
	linearPredicates = linearPredicates && anchor == nil
	constraints := make([]ast.ASTHandle, 0)
	references := make([]ast.ASTRef, 0)
	defer func() {
		for index := len(references) - 1; index >= 0; index-- {
			resultErr = errors.Join(resultErr, c.manager.DelRef(&references[index]))
		}
		if resultErr != nil && result != (ast.ASTRef{}) {
			resultErr = errors.Join(resultErr, c.manager.DelRef(&result))
			result = ast.ASTRef{}
		}
	}()
	stack := []*ModelNode{root}
	for len(stack) != 0 {
		last := len(stack) - 1
		node := stack[last]
		stack = stack[:last]
		rule, x := c.selectedTraceRule(node)
		if x != nil {
			return result, x
		}
		uninterpreted, x := rule.rule.UninterpretedTailSize()
		if x != nil {
			return result, x
		}
		tailSize, x := rule.rule.TailSize()
		if x != nil {
			return result, x
		}
		if node != root && linearPredicates {
			head, y := rule.rule.Head()
			if y != nil {
				return result, y
			}
			instantiated, y := c.instantiateTraceExpression(node, rule, head)
			if y != nil {
				return result, y
			}
			references = append(references, instantiated)
			handle, y := c.manager.RefGet(instantiated)
			if y != nil {
				return result, y
			}
			if !c.manager.IsTrue(handle) {
				constraints = append(constraints, handle)
			}
		}
		start := uninterpreted
		if node != root {
			start = 0
		}
		for index := start; index < tailSize; index++ {
			tail, _, y := rule.rule.Tail(index)
			if y != nil {
				return result, y
			}
			if index < uninterpreted {
				if anchor == nil || index >= len(rule.predecessors) || rule.predecessors[index] != anchor {
					continue
				}
			}
			instantiated, y := c.instantiateTraceExpression(node, rule, tail)
			if y != nil {
				return result, y
			}
			references = append(references, instantiated)
			handle, y := c.manager.RefGet(instantiated)
			if y != nil {
				return result, y
			}
			if index >= uninterpreted && c.implicant != nil {
				modelCore, present, z := node.Model()
				if z != nil {
					return result, z
				}
				if present && modelCore.DerivedModel() != nil {
					evaluated, z := c.implicant.EvalExpr(modelCore.DerivedModel(), handle)
					if z != nil {
						return result, z
					}
					references = append(references, evaluated)
					handle, z = c.manager.RefGet(evaluated)
					if z != nil {
						return result, z
					}
				}
			}
			if !c.manager.IsTrue(handle) {
				constraints = append(constraints, handle)
			}
		}
		children, x := node.Children()
		if x != nil {
			return result, x
		}
		stack = append(stack, children...)
	}
	var formula ast.ASTHandle
	if len(constraints) == 0 {
		formula, err = c.manager.MkTrue()
	} else {
		formula, err = c.manager.MkAndArgs(constraints)
	}
	if err != nil {
		return result, err
	}
	return c.manager.NewRef(formula)
}
