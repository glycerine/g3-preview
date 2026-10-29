package spacer

// Source: src/muz/spacer/spacer_context.cpp pred_transformer rule maps,
// find_predecessors, remove_predecessors, and add_premises.

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/muz/base"
)

func (p *PredicateTransformer) normalizedRule(rule *base.Rule) (*spacerNormalizedRule, error) {
	if err := p.open(); err != nil {
		return nil, err
	}
	if rule == nil {
		return nil, base.ErrRuleOwner
	}
	for _, candidate := range p.rules {
		same, err := candidate.rule.Same(rule)
		if err != nil {
			return nil, err
		}
		if same {
			return candidate, nil
		}
	}
	return nil, ast.ErrASTDomain
}

// RuleTag returns the retained rule-selection literal used in the transition.
func (p *PredicateTransformer) RuleTag(rule *base.Rule) (ast.ASTHandle, error) {
	normalized, err := p.normalizedRule(rule)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if normalized.tag.IsNull() {
		return ast.ASTHandle{}, ast.ErrASTDomain
	}
	return normalized.tag, nil
}

// RuleTransition returns the normalized, unguarded transition for rule.
func (p *PredicateTransformer) RuleTransition(rule *base.Rule) (ast.ASTHandle, error) {
	normalized, err := p.normalizedRule(rule)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	return normalized.transition, nil
}

// RuleAuxiliaries returns the current-state constants introduced for variables
// that occur only in interpreted rule tails.
func (p *PredicateTransformer) RuleAuxiliaries(rule *base.Rule) ([]ast.ASTHandle, error) {
	normalized, err := p.normalizedRule(rule)
	if err != nil {
		return nil, err
	}
	return append([]ast.ASTHandle(nil), normalized.auxiliaries...), nil
}

// RulePredecessors returns predicate transformers in uninterpreted-tail order.
func (p *PredicateTransformer) RulePredecessors(rule *base.Rule) ([]*PredicateTransformer, error) {
	normalized, err := p.normalizedRule(rule)
	if err != nil {
		return nil, err
	}
	return append([]*PredicateTransformer(nil), normalized.predecessors...), nil
}

// TaggedRule returns an owning rule reference for a selection literal.
func (p *PredicateTransformer) TaggedRule(tag ast.ASTHandle) (*base.Rule, bool, error) {
	if err := p.open(); err != nil {
		return nil, false, err
	}
	for _, normalized := range p.rules {
		if normalized.tag == tag {
			rule, err := normalized.rule.Retain()
			return rule, err == nil, err
		}
	}
	return nil, false, nil
}

// RemovePredecessors removes positive or negative rule-selection literals.
func (p *PredicateTransformer) RemovePredecessors(literals []ast.ASTHandle) ([]ast.ASTHandle, error) {
	if err := p.open(); err != nil {
		return nil, err
	}
	result := make([]ast.ASTHandle, 0, len(literals))
	for _, literal := range literals {
		original := literal
		negated, child, err := p.manager.IsNot(literal)
		if err != nil {
			return nil, err
		}
		if negated {
			literal = child
		}
		isTag := false
		for _, normalized := range p.rules {
			if normalized.tag == literal {
				isTag = true
				break
			}
		}
		if !isTag {
			result = append(result, original)
		}
	}
	return result, nil
}

func (p *PredicateTransformer) appendRulePremises(level uint32, normalized *spacerNormalizedRule, result *[]ast.ASTRef) error {
	for index, predecessor := range normalized.predecessors {
		formula, err := predecessor.Formulas(level, false)
		if err != nil {
			return err
		}
		if p.manager.IsTrue(formula) {
			continue
		}
		converted, err := p.owner.FormulaNewToOld(formula, uint32(index), true)
		if err != nil {
			return err
		}
		*result = append(*result, converted)
	}
	return nil
}

// RulePremises returns owned predecessor-frame formulas in old-state order.
func (p *PredicateTransformer) RulePremises(level uint32, rule *base.Rule) (_ []ast.ASTRef, resultErr error) {
	normalized, err := p.normalizedRule(rule)
	if err != nil {
		return nil, err
	}
	var result []ast.ASTRef
	defer func() {
		if resultErr != nil {
			resultErr = errors.Join(resultErr, releaseNormalizeRefs(p.manager, result))
		}
	}()
	if err = p.appendRulePremises(level, normalized, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// Premises returns owned background, state, and predecessor-frame formulas.
func (p *PredicateTransformer) Premises(level uint32) (_ []ast.ASTRef, resultErr error) {
	if err := p.open(); err != nil {
		return nil, err
	}
	if !p.initialized {
		return nil, ast.ErrASTDomain
	}
	var result []ast.ASTRef
	defer func() {
		if resultErr != nil {
			resultErr = errors.Join(resultErr, releaseNormalizeRefs(p.manager, result))
		}
	}()
	background, err := p.owner.Background()
	if err != nil {
		return nil, err
	}
	backgroundRef, err := p.manager.NewRef(background)
	if err != nil {
		return nil, err
	}
	result = append(result, backgroundRef)
	state := p.transition
	if level == 0 {
		state = p.initial
	}
	stateRef, err := p.manager.CopyRef(state)
	if err != nil {
		return nil, err
	}
	result = append(result, stateRef)
	for _, normalized := range p.rules {
		if err = p.appendRulePremises(level, normalized, &result); err != nil {
			return nil, err
		}
	}
	return result, nil
}
