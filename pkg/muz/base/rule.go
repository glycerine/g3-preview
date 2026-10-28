package base

// Source: src/muz/base/dl_rule.h rule and rule_manager;
// src/muz/base/dl_rule.cpp rule_manager::mk, check_valid_head, and rule
// ownership/query operations.

import (
	"errors"
	"fmt"

	"github.com/glycerine/g3/pkg/ast"
)

var ErrRuleOwner = errors.New("muz/base: invalid or closed rule owner")

type ruleTail struct {
	handle    ast.ASTHandle
	reference ast.ASTRef
	negated   bool
}

// Rule is an owning Horn-rule representation. Tail order matches the native
// rule layout: positive predicates, negative predicates, then interpreted
// formulas.
type Rule struct {
	manager       *ast.ASTManager
	head          ast.ASTHandle
	headReference ast.ASTRef
	tail          []ruleTail
	positive      int
	uninterpreted int
	name          string
	closed        bool
}

// RuleManager owns the registered predicate declarations used to classify
// rule heads and tails. Rules own their AST references independently.
type RuleManager struct {
	manager    *ast.ASTManager
	predicates map[ast.ASTHandle]ast.ASTRef
	closed     bool
}

func NewRuleManager(manager *ast.ASTManager) (*RuleManager, error) {
	if manager == nil {
		return nil, ErrRuleOwner
	}
	return &RuleManager{manager: manager, predicates: make(map[ast.ASTHandle]ast.ASTRef)}, nil
}

func (m *RuleManager) open() error {
	if m == nil || m.closed || m.manager == nil {
		return ErrRuleOwner
	}
	return nil
}

// RegisterPredicate retains a declaration as a recursive Horn predicate.
func (m *RuleManager) RegisterPredicate(declaration ast.ASTHandle) error {
	if err := m.open(); err != nil {
		return err
	}
	kind, err := m.manager.GetKind(declaration)
	if err != nil {
		return err
	}
	if kind != ast.ASTFuncDecl {
		return fmt.Errorf("muz/base: predicate is not a function declaration")
	}
	if _, found := m.predicates[declaration]; found {
		return nil
	}
	reference, err := m.manager.NewRef(declaration)
	if err != nil {
		return err
	}
	m.predicates[declaration] = reference
	return nil
}

func (m *RuleManager) isPredicate(application ast.ASTHandle) (bool, error) {
	kind, err := m.manager.GetKind(application)
	if err != nil {
		return false, err
	}
	if kind != ast.ASTApp {
		return false, fmt.Errorf("muz/base: expected application in rule")
	}
	declaration, err := m.manager.GetDecl(application)
	if err != nil {
		return false, err
	}
	_, found := m.predicates[declaration]
	return found, nil
}

func (m *RuleManager) validHead(head ast.ASTHandle) error {
	predicate, err := m.isPredicate(head)
	if err != nil {
		return err
	}
	if !predicate {
		return fmt.Errorf("muz/base: illegal head; predicate must be registered")
	}
	count, err := m.manager.GetNumArgs(head)
	if err != nil {
		return err
	}
	for i := uint32(0); i < count; i++ {
		argument, e := m.manager.GetArg(head, i)
		if e != nil {
			return e
		}
		kind, e := m.manager.GetKind(argument)
		if e != nil {
			return e
		}
		if kind == ast.ASTVar {
			continue
		}
		value, e := m.manager.IsValue(argument)
		if e != nil {
			return e
		}
		if !value {
			return fmt.Errorf("muz/base: illegal non-variable argument in rule head")
		}
	}
	return nil
}

// NewRule constructs and owns a rule. negated may be nil; otherwise it must
// have one entry per tail. Negated interpreted formulas are materialized as
// Boolean not applications, matching the native representation.
func (m *RuleManager) NewRule(head ast.ASTHandle, tail []ast.ASTHandle, negated []bool, name string) (_ *Rule, resultErr error) {
	if err := m.open(); err != nil {
		return nil, err
	}
	if negated != nil && len(negated) != len(tail) {
		return nil, fmt.Errorf("muz/base: negation count does not match tail size")
	}
	if err := m.validHead(head); err != nil {
		return nil, err
	}
	r := &Rule{manager: m.manager, head: head, name: name}
	var err error
	r.headReference, err = m.manager.NewRef(head)
	if err != nil {
		return nil, err
	}
	defer func() {
		if resultErr != nil {
			resultErr = errors.Join(resultErr, r.Close())
		}
	}()
	uninterpreted := make([]ruleTail, 0, len(tail))
	interpreted := make([]ruleTail, 0, len(tail))
	for i, expression := range tail {
		isPredicate, e := m.isPredicate(expression)
		if e != nil {
			return nil, e
		}
		isNegated := negated != nil && negated[i]
		stored := expression
		if isNegated && !isPredicate {
			stored, e = m.manager.MkNot(expression)
			if e != nil {
				return nil, e
			}
			isNegated = false
		}
		reference, e := m.manager.NewRef(stored)
		if e != nil {
			return nil, e
		}
		entry := ruleTail{handle: stored, reference: reference, negated: isNegated}
		// Keep every acquired reference reachable by Close during construction.
		r.tail = append(r.tail, entry)
		if isPredicate {
			uninterpreted = append(uninterpreted, entry)
		} else {
			interpreted = append(interpreted, entry)
		}
	}
	// Native rule_manager::mk fills the interpreted segment downward.
	for left, right := 0, len(interpreted)-1; left < right; left, right = left+1, right-1 {
		interpreted[left], interpreted[right] = interpreted[right], interpreted[left]
	}
	// Mirror the source's in-place two-ended partition of predicate tails.
	end := len(uninterpreted)
	for i := 0; i < end; {
		if uninterpreted[i].negated {
			end--
			uninterpreted[i], uninterpreted[end] = uninterpreted[end], uninterpreted[i]
		} else {
			i++
		}
	}
	r.positive = end
	r.uninterpreted = len(uninterpreted)
	r.tail = append(uninterpreted, interpreted...)
	return r, nil
}

func (r *Rule) open() error {
	if r == nil || r.closed || r.manager == nil {
		return ErrRuleOwner
	}
	return nil
}

func (r *Rule) Head() (ast.ASTHandle, error) {
	if err := r.open(); err != nil {
		return ast.ASTHandle{}, err
	}
	return r.head, nil
}

func (r *Rule) Name() (string, error) {
	if err := r.open(); err != nil {
		return "", err
	}
	return r.name, nil
}

func (r *Rule) TailSize() (int, error) {
	if err := r.open(); err != nil {
		return 0, err
	}
	return len(r.tail), nil
}

func (r *Rule) PositiveTailSize() (int, error) {
	if err := r.open(); err != nil {
		return 0, err
	}
	return r.positive, nil
}

func (r *Rule) UninterpretedTailSize() (int, error) {
	if err := r.open(); err != nil {
		return 0, err
	}
	return r.uninterpreted, nil
}

func (r *Rule) Tail(index int) (ast.ASTHandle, bool, error) {
	if err := r.open(); err != nil {
		return ast.ASTHandle{}, false, err
	}
	if index < 0 || index >= len(r.tail) {
		return ast.ASTHandle{}, false, fmt.Errorf("muz/base: tail index out of bounds")
	}
	return r.tail[index].handle, r.tail[index].negated, nil
}

func (r *Rule) HasNegation() (bool, error) {
	if err := r.open(); err != nil {
		return false, err
	}
	return r.positive != r.uninterpreted, nil
}

func (r *Rule) IsInTail(declaration ast.ASTHandle, onlyPositive bool) (bool, error) {
	if err := r.open(); err != nil {
		return false, err
	}
	end := r.uninterpreted
	if onlyPositive {
		end = r.positive
	}
	for i := 0; i < end; i++ {
		candidate, err := r.manager.GetDecl(r.tail[i].handle)
		if err != nil {
			return false, err
		}
		if candidate == declaration {
			return true, nil
		}
	}
	return false, nil
}

func (r *Rule) Close() (result error) {
	if r == nil || r.manager == nil {
		return ErrRuleOwner
	}
	if r.closed {
		return nil
	}
	for i := len(r.tail) - 1; i >= 0; i-- {
		result = errors.Join(result, r.manager.DelRef(&r.tail[i].reference))
	}
	result = errors.Join(result, r.manager.DelRef(&r.headReference))
	r.tail = nil
	r.closed = true
	return result
}

func (m *RuleManager) Close() (result error) {
	if m == nil || m.manager == nil {
		return ErrRuleOwner
	}
	if m.closed {
		return nil
	}
	for declaration, reference := range m.predicates {
		ref := reference
		result = errors.Join(result, m.manager.DelRef(&ref))
		delete(m.predicates, declaration)
	}
	m.closed = true
	return result
}
