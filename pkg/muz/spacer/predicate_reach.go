package spacer

// Source: src/muz/spacer/spacer_context.cpp pred_transformer reach facts,
// reach-case variables, init_reach_facts, and extend_initial.

import (
	"errors"
	"fmt"

	"github.com/glycerine/g3/pkg/ast"
)

func (p *PredicateTransformer) SetReachAsInitial(value bool) error {
	if err := p.open(); err != nil {
		return err
	}
	p.reachAsInitial = value
	return nil
}

func (p *PredicateTransformer) freshReachCaseVariable() (ast.ASTRef, error) {
	boolean, err := p.manager.MkBoolSort()
	if err != nil {
		return ast.ASTRef{}, err
	}
	symbol, err := p.manager.Symbols.Intern(fmt.Sprintf("%s#reach_case_%d", p.name, len(p.reachCaseVariables)))
	if err != nil {
		return ast.ASTRef{}, err
	}
	declaration, err := p.manager.MkFuncDecl(symbol, nil, boolean, ast.ASTFuncDeclInfo{})
	if err != nil {
		return ast.ASTRef{}, err
	}
	declaration, err = p.owner.NewPredicate(declaration)
	if err != nil {
		return ast.ASTRef{}, err
	}
	value, err := p.manager.MkApp(declaration, nil)
	if err != nil {
		return ast.ASTRef{}, err
	}
	return p.manager.NewRef(value)
}

// ExtendInitial admits fact through a fresh current-state extension literal.
func (p *PredicateTransformer) ExtendInitial(fact ast.ASTHandle) (_ ast.ASTHandle, resultErr error) {
	if err := p.open(); err != nil {
		return ast.ASTHandle{}, err
	}
	boolean, err := p.manager.MkBoolSort()
	if err != nil {
		return ast.ASTHandle{}, err
	}
	fresh, err := p.manager.MkFreshConst(p.name+"_ext", boolean)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	declaration, err := p.manager.GetDecl(fresh)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	declaration, err = p.owner.NewPredicate(declaration)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	value, err := p.manager.MkApp(declaration, nil)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	currentExtension, err := p.manager.RefGet(p.extendLiteral)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	formula, err := p.manager.MkOrArray([]ast.ASTHandle{currentExtension, fact, value})
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if err = p.solver.Assert(formula); err != nil {
		return ast.ASTHandle{}, err
	}
	negated, err := p.manager.MkNot(value)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if err = p.manager.SetRef(p.extendLiteral, negated); err != nil {
		return ast.ASTHandle{}, err
	}
	return negated, nil
}

// AddReachFact retains a new, non-duplicate fact and publishes its monotone
// reach-case implication to this transformer and all rule users.
func (p *PredicateTransformer) AddReachFact(fact *ReachFact) (_ bool, resultErr error) {
	if err := p.open(); err != nil {
		return false, err
	}
	if fact == nil {
		return false, ErrManagerOwner
	}
	value, err := fact.Fact()
	if err != nil {
		return false, err
	}
	for _, prior := range p.reachFacts {
		priorValue, getErr := prior.Fact()
		if getErr != nil {
			return false, getErr
		}
		if priorValue == value {
			return false, nil
		}
	}
	initial, err := fact.IsInitial()
	if err != nil {
		return false, err
	}
	if initial && len(p.reachFacts) != 0 {
		lastInitial, lastErr := p.reachFacts[len(p.reachFacts)-1].IsInitial()
		if lastErr != nil {
			return false, lastErr
		}
		if !lastInitial {
			return false, ast.ErrASTDomain
		}
	}
	var last ast.ASTHandle
	if len(p.reachCaseVariables) != 0 {
		last, err = p.manager.RefGet(p.reachCaseVariables[len(p.reachCaseVariables)-1])
		if err != nil {
			return false, err
		}
	}
	var variableRef ast.ASTRef
	if initial || !p.reachAsInitial {
		variableRef, err = p.freshReachCaseVariable()
	} else {
		var extension ast.ASTHandle
		extension, err = p.ExtendInitial(value)
		if err == nil {
			_, extension, err = p.manager.IsNot(extension)
		}
		if err == nil {
			variableRef, err = p.manager.NewRef(extension)
		}
	}
	if err != nil {
		return false, err
	}
	committedVariable := false
	defer func() {
		if !committedVariable {
			resultErr = errors.Join(resultErr, p.manager.DelRef(&variableRef))
		}
	}()
	variable, err := p.manager.RefGet(variableRef)
	if err != nil {
		return false, err
	}
	parts := []ast.ASTHandle{value, variable}
	if !last.IsNull() {
		notLast, makeErr := p.manager.MkNot(last)
		if makeErr != nil {
			return false, makeErr
		}
		parts = append([]ast.ASTHandle{notLast}, parts...)
	}
	formula, err := p.manager.MkOrArray(parts)
	if err != nil {
		return false, err
	}
	if err = p.reachSolver.Assert(formula); err != nil {
		return false, err
	}
	owned, err := fact.Retain()
	if err != nil {
		return false, err
	}
	p.reachFacts = append(p.reachFacts, owned)
	p.reachCaseVariables = append(p.reachCaseVariables, variableRef)
	committedVariable = true
	if initial {
		p.initialReachFacts++
	}
	lemma, err := NewLemma(p.owner, formula, InfinityLevel)
	if err != nil {
		return true, err
	}
	defer func() { resultErr = errors.Join(resultErr, lemma.Close()) }()
	for _, user := range p.uses {
		if err = user.addLemmaFromChild(p, lemma, InfinityLevel); err != nil {
			return true, err
		}
	}
	return true, nil
}

// InitReachFacts publishes every normalized initial rule transition once.
func (p *PredicateTransformer) InitReachFacts() (resultErr error) {
	if err := p.open(); err != nil {
		return err
	}
	for _, rule := range p.rules {
		if !rule.initial {
			continue
		}
		fact, err := NewReachFact(p.manager, rule.rule, rule.transition, rule.auxiliaries, true)
		if err != nil {
			return err
		}
		_, addErr := p.AddReachFact(fact)
		closeErr := fact.Close()
		if addErr != nil || closeErr != nil {
			return errors.Join(addErr, closeErr)
		}
	}
	return nil
}

func (p *PredicateTransformer) HasReachFacts() (bool, error) {
	if err := p.open(); err != nil {
		return false, err
	}
	return len(p.reachFacts) != 0, nil
}

func (p *PredicateTransformer) ReachFacts() ([]*ReachFact, error) {
	if err := p.open(); err != nil {
		return nil, err
	}
	result := make([]*ReachFact, 0, len(p.reachFacts))
	for _, fact := range p.reachFacts {
		owned, err := fact.Retain()
		if err != nil {
			for _, prior := range result {
				_ = prior.Close()
			}
			return nil, err
		}
		result = append(result, owned)
	}
	return result, nil
}

func (p *PredicateTransformer) ReachCaseVariable(index uint32) (ast.ASTHandle, error) {
	if err := p.open(); err != nil {
		return ast.ASTHandle{}, err
	}
	if index >= uint32(len(p.reachCaseVariables)) {
		return ast.ASTHandle{}, ast.ErrASTDomain
	}
	return p.manager.RefGet(p.reachCaseVariables[index])
}

func (p *PredicateTransformer) LastReachCaseVariable() (ast.ASTHandle, bool, error) {
	if err := p.open(); err != nil {
		return ast.ASTHandle{}, false, err
	}
	if len(p.reachCaseVariables) == 0 {
		return ast.ASTHandle{}, false, nil
	}
	value, err := p.manager.RefGet(p.reachCaseVariables[len(p.reachCaseVariables)-1])
	return value, err == nil, err
}
