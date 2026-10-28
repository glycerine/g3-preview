package base

// Source: src/muz/base/dl_rule_set.h rule_dependencies and rule_set;
// src/muz/base/dl_rule_set.cpp grouped storage, dependency population,
// reopen/finalize lifecycle, and stratified-negation validation.

import (
	"errors"
	"fmt"

	"github.com/glycerine/g3/pkg/ast"
)

var ErrRuleSetOwner = errors.New("muz/base: invalid or closed rule set")

type ruleDependencyEdge struct {
	master ast.ASTHandle
	strict bool
}

// RuleSet owns retained references to a Horn program's rules. Finalize builds
// dependencies and validates stratified negation; Reopen discards that derived
// state before mutation.
type RuleSet struct {
	manager      *RuleManager
	rules        []*Rule
	byHead       map[ast.ASTHandle][]*Rule
	dependencies map[ast.ASTHandle][]ast.ASTHandle
	strata       map[ast.ASTHandle]uint32
	finalized    bool
	closed       bool
}

func NewRuleSet(manager *RuleManager) (*RuleSet, error) {
	if manager == nil {
		return nil, ErrRuleSetOwner
	}
	if err := manager.open(); err != nil {
		return nil, err
	}
	return &RuleSet{manager: manager, byHead: make(map[ast.ASTHandle][]*Rule)}, nil
}

func (s *RuleSet) open() error {
	if s == nil || s.closed || s.manager == nil {
		return ErrRuleSetOwner
	}
	return s.manager.open()
}

func (s *RuleSet) ruleHeadDeclaration(rule *Rule) (ast.ASTHandle, error) {
	state, err := rule.open()
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if state.manager != s.manager.manager {
		return ast.ASTHandle{}, fmt.Errorf("muz/base: rule belongs to another AST manager")
	}
	return state.manager.GetDecl(state.head)
}

// Add retains rule and indexes it under its head predicate.
func (s *RuleSet) Add(rule *Rule) error {
	if err := s.open(); err != nil {
		return err
	}
	if s.finalized {
		return fmt.Errorf("muz/base: reopen rule set before mutation")
	}
	declaration, err := s.ruleHeadDeclaration(rule)
	if err != nil {
		return err
	}
	owned, err := rule.Retain()
	if err != nil {
		return err
	}
	s.rules = append(s.rules, owned)
	s.byHead[declaration] = append(s.byHead[declaration], owned)
	return nil
}

// Remove drops the first rule with the same native rule state. Like the
// source vectors, removal swaps in the final entry and does not preserve order.
func (s *RuleSet) Remove(rule *Rule) error {
	if err := s.open(); err != nil {
		return err
	}
	if s.finalized {
		return fmt.Errorf("muz/base: reopen rule set before mutation")
	}
	state, err := rule.open()
	if err != nil {
		return err
	}
	declaration, err := s.ruleHeadDeclaration(rule)
	if err != nil {
		return err
	}
	index := -1
	for i, candidate := range s.rules {
		if candidate.state == state {
			index = i
			break
		}
	}
	if index < 0 {
		return fmt.Errorf("muz/base: rule is not in rule set")
	}
	owned := s.rules[index]
	s.rules[index] = s.rules[len(s.rules)-1]
	s.rules = s.rules[:len(s.rules)-1]
	bucket := s.byHead[declaration]
	for i, candidate := range bucket {
		if candidate.state == state {
			bucket[i] = bucket[len(bucket)-1]
			bucket = bucket[:len(bucket)-1]
			break
		}
	}
	if len(bucket) == 0 {
		delete(s.byHead, declaration)
	} else {
		s.byHead[declaration] = bucket
	}
	return owned.Close()
}

func (s *RuleSet) Len() (int, error) {
	if err := s.open(); err != nil {
		return 0, err
	}
	return len(s.rules), nil
}

// Rule returns a new owning reference to the indexed rule.
func (s *RuleSet) Rule(index int) (*Rule, error) {
	if err := s.open(); err != nil {
		return nil, err
	}
	if index < 0 || index >= len(s.rules) {
		return nil, fmt.Errorf("muz/base: rule index out of bounds")
	}
	return s.rules[index].Retain()
}

// PredicateRules returns new owning references in the rule set's current
// grouped order. The caller must close each result.
func (s *RuleSet) PredicateRules(declaration ast.ASTHandle) ([]*Rule, error) {
	if err := s.open(); err != nil {
		return nil, err
	}
	source := s.byHead[declaration]
	result := make([]*Rule, 0, len(source))
	for _, rule := range source {
		owned, err := rule.Retain()
		if err != nil {
			for _, prior := range result {
				err = errors.Join(err, prior.Close())
			}
			return nil, err
		}
		result = append(result, owned)
	}
	return result, nil
}

func (s *RuleSet) addDependency(order *[]ast.ASTHandle, seen map[ast.ASTHandle]bool, declaration ast.ASTHandle) {
	if !seen[declaration] {
		seen[declaration] = true
		*order = append(*order, declaration)
	}
}

func (s *RuleSet) expressionPredicates(root ast.ASTHandle, visit func(ast.ASTHandle)) error {
	manager := s.manager.manager
	stack := []ast.ASTHandle{root}
	seen := make(map[ast.ASTHandle]bool)
	for len(stack) != 0 {
		last := len(stack) - 1
		current := stack[last]
		stack = stack[:last]
		if seen[current] {
			continue
		}
		seen[current] = true
		kind, err := manager.GetKind(current)
		if err != nil {
			return err
		}
		switch kind {
		case ast.ASTApp:
			declaration, e := manager.GetDecl(current)
			if e != nil {
				return e
			}
			if _, predicate := s.manager.predicates[declaration]; predicate {
				visit(declaration)
			}
			count, e := manager.GetNumArgs(current)
			if e != nil {
				return e
			}
			for i := count; i > 0; i-- {
				argument, x := manager.GetArg(current, i-1)
				if x != nil {
					return x
				}
				stack = append(stack, argument)
			}
		case ast.ASTQuantifier:
			body, e := manager.GetQuantifierBody(current)
			if e != nil {
				return e
			}
			stack = append(stack, body)
		}
	}
	return nil
}

// Finalize computes the dependency graph and minimal predicate strata. It
// returns false for a dependency cycle containing a negative predicate edge.
func (s *RuleSet) Finalize() (bool, error) {
	if err := s.open(); err != nil {
		return false, err
	}
	if s.finalized {
		return false, fmt.Errorf("muz/base: rule set is already finalized")
	}
	order := make([]ast.ASTHandle, 0, len(s.byHead))
	seenNodes := make(map[ast.ASTHandle]bool)
	edges := make(map[ast.ASTHandle][]ruleDependencyEdge)
	edgeIndex := make(map[ast.ASTHandle]map[ast.ASTHandle]int)
	addEdge := func(depending, master ast.ASTHandle, strict bool) {
		s.addDependency(&order, seenNodes, depending)
		s.addDependency(&order, seenNodes, master)
		indices := edgeIndex[depending]
		if indices == nil {
			indices = make(map[ast.ASTHandle]int)
			edgeIndex[depending] = indices
		}
		if index, found := indices[master]; found {
			if strict {
				edges[depending][index].strict = true
			}
			return
		}
		indices[master] = len(edges[depending])
		edges[depending] = append(edges[depending], ruleDependencyEdge{master: master, strict: strict})
	}
	for _, rule := range s.rules {
		state, err := rule.open()
		if err != nil {
			return false, err
		}
		head, err := state.manager.GetDecl(state.head)
		if err != nil {
			return false, err
		}
		s.addDependency(&order, seenNodes, head)
		for i, tail := range state.tail {
			strict := i >= state.positive && i < state.uninterpreted && tail.negated
			if err = s.expressionPredicates(tail.handle, func(master ast.ASTHandle) { addEdge(head, master, false) }); err != nil {
				return false, err
			}
			if strict {
				master, e := state.manager.GetDecl(tail.handle)
				if e != nil {
					return false, e
				}
				addEdge(head, master, true)
			}
		}
	}
	// Compute native predicate strata: strongly connected components followed
	// by a dependency-first topological order of the component graph.
	nextIndex := 0
	indices := make(map[ast.ASTHandle]int, len(order))
	lowLink := make(map[ast.ASTHandle]int, len(order))
	onStack := make(map[ast.ASTHandle]bool, len(order))
	stack := make([]ast.ASTHandle, 0, len(order))
	components := make([][]ast.ASTHandle, 0, len(order))
	componentOf := make(map[ast.ASTHandle]int, len(order))
	var visitNode func(ast.ASTHandle)
	visitNode = func(node ast.ASTHandle) {
		indices[node] = nextIndex + 1
		lowLink[node] = nextIndex + 1
		nextIndex++
		stack = append(stack, node)
		onStack[node] = true
		for _, edge := range edges[node] {
			if indices[edge.master] == 0 {
				visitNode(edge.master)
				if lowLink[edge.master] < lowLink[node] {
					lowLink[node] = lowLink[edge.master]
				}
			} else if onStack[edge.master] && indices[edge.master] < lowLink[node] {
				lowLink[node] = indices[edge.master]
			}
		}
		if lowLink[node] != indices[node] {
			return
		}
		component := make([]ast.ASTHandle, 0, 1)
		for {
			last := len(stack) - 1
			member := stack[last]
			stack = stack[:last]
			onStack[member] = false
			componentOf[member] = len(components)
			component = append(component, member)
			if member == node {
				break
			}
		}
		components = append(components, component)
	}
	for _, node := range order {
		if indices[node] == 0 {
			visitNode(node)
		}
	}
	componentDeps := make([][]int, len(components))
	componentSeen := make([]map[int]bool, len(components))
	for _, depending := range order {
		list := edges[depending]
		from := componentOf[depending]
		for _, edge := range list {
			to := componentOf[edge.master]
			if from == to {
				if edge.strict {
					return false, nil
				}
				continue
			}
			if componentSeen[from] == nil {
				componentSeen[from] = make(map[int]bool)
			}
			if !componentSeen[from][to] {
				componentSeen[from][to] = true
				componentDeps[from] = append(componentDeps[from], to)
			}
		}
	}
	componentOrder := make([]int, 0, len(components))
	visitedComponents := make([]bool, len(components))
	var visitComponent func(int)
	visitComponent = func(component int) {
		if visitedComponents[component] {
			return
		}
		visitedComponents[component] = true
		for _, dependency := range componentDeps[component] {
			visitComponent(dependency)
		}
		componentOrder = append(componentOrder, component)
	}
	for _, node := range order {
		visitComponent(componentOf[node])
	}
	strata := make(map[ast.ASTHandle]uint32, len(order))
	for index, component := range componentOrder {
		for _, node := range components[component] {
			strata[node] = uint32(index)
		}
	}
	dependencies := make(map[ast.ASTHandle][]ast.ASTHandle, len(edges))
	for depending, list := range edges {
		dependencies[depending] = make([]ast.ASTHandle, len(list))
		for i := range list {
			dependencies[depending][i] = list[i].master
		}
	}
	s.dependencies = dependencies
	s.strata = strata
	s.finalized = true
	return true, nil
}

func (s *RuleSet) Reopen() error {
	if err := s.open(); err != nil {
		return err
	}
	s.dependencies = nil
	s.strata = nil
	s.finalized = false
	return nil
}

func (s *RuleSet) IsFinalized() (bool, error) {
	if err := s.open(); err != nil {
		return false, err
	}
	return s.finalized, nil
}

func (s *RuleSet) Dependencies(declaration ast.ASTHandle) ([]ast.ASTHandle, error) {
	if err := s.open(); err != nil {
		return nil, err
	}
	if !s.finalized {
		return nil, fmt.Errorf("muz/base: finalize rule set before reading dependencies")
	}
	return append([]ast.ASTHandle(nil), s.dependencies[declaration]...), nil
}

func (s *RuleSet) PredicateStratum(declaration ast.ASTHandle) (uint32, error) {
	if err := s.open(); err != nil {
		return 0, err
	}
	if !s.finalized {
		return 0, fmt.Errorf("muz/base: finalize rule set before reading strata")
	}
	return s.strata[declaration], nil
}

func (s *RuleSet) Close() (result error) {
	if s == nil || s.manager == nil {
		return ErrRuleSetOwner
	}
	if s.closed {
		return nil
	}
	for i := len(s.rules) - 1; i >= 0; i-- {
		result = errors.Join(result, s.rules[i].Close())
	}
	s.rules = nil
	s.byHead = nil
	s.dependencies = nil
	s.strata = nil
	s.closed = true
	return result
}
