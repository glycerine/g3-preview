package spacer

// Source: src/muz/spacer/spacer_context.cpp context::mk_reach_fact.

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/muz/base"
)

// MakeReachFact constructs a concrete reach fact from a rule path selected by
// md. The returned fact owns retained references to every child justification.
func (p *PredicateTransformer) MakeReachFact(obligation *ProofObligation, rule *base.Rule, md *model.ModelCore, eliminateAuxiliaries, reachDNF bool) (result *ReachFact, resultErr error) {
	if err := p.open(); err != nil {
		return nil, err
	}
	defer func() {
		if resultErr != nil && result != nil {
			resultErr = errors.Join(resultErr, result.Close())
			result = nil
		}
	}()
	if obligation == nil || obligation.valid() != nil || obligation.transformer != p || rule == nil || md == nil || md.Manager() != p.manager {
		return nil, ErrManagerOwner
	}
	transition, err := p.RuleTransition(rule)
	if err != nil {
		return nil, err
	}
	path := []ast.ASTHandle{transition}
	predecessors, err := p.RulePredecessors(rule)
	if err != nil {
		return nil, err
	}
	justifications := make([]*ReachFact, 0, len(predecessors))
	defer func() {
		for _, justification := range justifications {
			resultErr = errors.Join(resultErr, justification.Close())
		}
	}()
	var variables []ast.ASTHandle
	for index, predecessor := range predecessors {
		fact, found, err := predecessor.UsedOriginReachFact(md, uint32(index))
		if err != nil || !found {
			if err == nil {
				err = ast.ErrASTDomain
			}
			return nil, err
		}
		justifications = append(justifications, fact)
		formula, err := fact.Fact()
		if err != nil {
			return nil, err
		}
		originRef, err := p.owner.FormulaNewToOld(formula, uint32(index), false)
		if err != nil {
			return nil, err
		}
		origin, err := p.manager.RefGet(originRef)
		if err == nil {
			path = append(path, origin)
		}
		err = errors.Join(err, p.manager.DelRef(&originRef))
		if err != nil {
			return nil, err
		}
		signature, err := predecessor.Signature()
		if err != nil {
			return nil, err
		}
		for _, declaration := range signature {
			originDeclaration, err := p.owner.OldToOld(declaration, 0, uint32(index))
			if err != nil {
				return nil, err
			}
			value, err := p.manager.MkApp(originDeclaration, nil)
			if err != nil {
				return nil, err
			}
			variables = append(variables, value)
		}
		auxiliaries, err := fact.Auxiliaries()
		if err != nil {
			return nil, err
		}
		for _, auxiliary := range auxiliaries {
			declaration, err := p.manager.GetDecl(auxiliary)
			if err != nil {
				return nil, err
			}
			originDeclaration, err := p.owner.NewToOld(declaration, uint32(index))
			if err != nil {
				return nil, err
			}
			value, err := p.manager.MkApp(originDeclaration, nil)
			if err != nil {
				return nil, err
			}
			variables = append(variables, value)
		}
	}
	ruleAuxiliaries, err := p.RuleAuxiliaries(rule)
	if err != nil {
		return nil, err
	}
	if eliminateAuxiliaries {
		variables = append(variables, ruleAuxiliaries...)
	}
	formula, err := p.manager.MkAndArray(path)
	if err != nil {
		return nil, err
	}
	temporary, err := NewDerivation(obligation, rule, formula, nil)
	if err != nil {
		return nil, err
	}
	defer func() { resultErr = errors.Join(resultErr, temporary.Close()) }()
	var implicantRef ast.ASTRef
	if reachDNF {
		implicantRef, err = temporary.modelImplicant(md, formula)
		if err != nil {
			return nil, err
		}
		formula, err = p.manager.RefGet(implicantRef)
		if err != nil {
			return nil, err
		}
		defer func() { resultErr = errors.Join(resultErr, p.manager.DelRef(&implicantRef)) }()
	}
	projectedRef, remaining, err := temporary.project(false, md, variables, formula)
	if err != nil {
		return nil, err
	}
	defer func() {
		resultErr = errors.Join(resultErr, p.manager.DelRef(&projectedRef), releaseNormalizeRefs(p.manager, remaining))
	}()
	if len(remaining) != 0 {
		return nil, ast.ErrASTDomain
	}
	projected, err := p.manager.RefGet(projectedRef)
	if err != nil {
		return nil, err
	}
	retainedAuxiliaries := ruleAuxiliaries
	if eliminateAuxiliaries {
		retainedAuxiliaries = nil
	}
	result, err = NewReachFact(p.manager, rule, projected, retainedAuxiliaries, false)
	if err != nil {
		return nil, err
	}
	for _, justification := range justifications {
		if err = result.AddJustification(justification); err != nil {
			return nil, errors.Join(err, result.Close())
		}
	}
	return result, nil
}
