package spacer

// Source: src/muz/spacer/spacer_context.h and spacer_context.cpp derivation
// and derivation::premise ownership and state updates.

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/muz/base"
)

type DerivationPremise struct {
	transformer *PredicateTransformer
	oldIndex    uint32
	summary     ast.ASTRef
	variables   []ast.ASTRef
	must        bool
}

func newDerivationPremise(transformer *PredicateTransformer, oldIndex uint32, summary ast.ASTHandle, must bool, auxiliaries []ast.ASTHandle) (_ *DerivationPremise, resultErr error) {
	if transformer == nil || transformer.open() != nil {
		return nil, ErrManagerOwner
	}
	premise := &DerivationPremise{transformer: transformer, oldIndex: oldIndex, must: must}
	defer func() {
		if resultErr != nil {
			resultErr = errors.Join(resultErr, premise.close())
		}
	}()
	var err error
	premise.summary, err = transformer.manager.NewRef(summary)
	if err != nil {
		return nil, err
	}
	if err = premise.rebuildVariables(auxiliaries); err != nil {
		return nil, err
	}
	return premise, nil
}

func (p *DerivationPremise) rebuildVariables(auxiliaries []ast.ASTHandle) (resultErr error) {
	if p == nil || p.transformer == nil {
		return ErrManagerOwner
	}
	manager := p.transformer.manager
	var next []ast.ASTRef
	defer func() {
		if resultErr != nil {
			resultErr = errors.Join(resultErr, releaseNormalizeRefs(manager, next))
		}
	}()
	signature, err := p.transformer.Signature()
	if err != nil {
		return err
	}
	for _, declaration := range signature {
		origin, err := p.transformer.owner.OldToOld(declaration, 0, p.oldIndex)
		if err != nil {
			return err
		}
		value, err := manager.MkApp(origin, nil)
		if err != nil {
			return err
		}
		reference, err := manager.NewRef(value)
		if err != nil {
			return err
		}
		next = append(next, reference)
	}
	for _, auxiliary := range auxiliaries {
		declaration, err := manager.GetDecl(auxiliary)
		if err != nil {
			return err
		}
		origin, err := p.transformer.owner.NewToOld(declaration, p.oldIndex)
		if err != nil {
			return err
		}
		value, err := manager.MkApp(origin, nil)
		if err != nil {
			return err
		}
		reference, err := manager.NewRef(value)
		if err != nil {
			return err
		}
		next = append(next, reference)
	}
	if err = releaseNormalizeRefs(manager, p.variables); err != nil {
		return err
	}
	p.variables = next
	next = nil
	return nil
}

func (p *DerivationPremise) SetSummary(summary ast.ASTHandle, must bool, auxiliaries []ast.ASTHandle) error {
	if p == nil || p.transformer == nil {
		return ErrManagerOwner
	}
	converted, err := p.transformer.owner.FormulaNewToOld(summary, p.oldIndex, false)
	if err != nil {
		return err
	}
	if err = p.rebuildVariables(auxiliaries); err != nil {
		return errors.Join(err, p.transformer.manager.DelRef(&converted))
	}
	if err = p.transformer.manager.DelRef(&p.summary); err != nil {
		return errors.Join(err, p.transformer.manager.DelRef(&converted))
	}
	p.summary = converted
	p.must = must
	return nil
}

func (p *DerivationPremise) Summary() (ast.ASTHandle, error) {
	if p == nil || p.transformer == nil {
		return ast.ASTHandle{}, ErrManagerOwner
	}
	return p.transformer.manager.RefGet(p.summary)
}

func (p *DerivationPremise) Variables() ([]ast.ASTHandle, error) {
	if p == nil || p.transformer == nil {
		return nil, ErrManagerOwner
	}
	result := make([]ast.ASTHandle, len(p.variables))
	for i := range p.variables {
		var err error
		result[i], err = p.transformer.manager.RefGet(p.variables[i])
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (p *DerivationPremise) Transformer() *PredicateTransformer { return p.transformer }
func (p *DerivationPremise) OldIndex() uint32                   { return p.oldIndex }
func (p *DerivationPremise) Must() bool                         { return p.must }

func (p *DerivationPremise) close() error {
	if p == nil || p.transformer == nil {
		return nil
	}
	manager := p.transformer.manager
	result := releaseNormalizeRefs(manager, p.variables)
	if p.summary != (ast.ASTRef{}) {
		result = errors.Join(result, manager.DelRef(&p.summary))
	}
	if result == nil {
		p.variables = nil
		p.transformer = nil
	}
	return result
}

type Derivation struct {
	parent       *ProofObligation
	rule         *base.Rule
	premises     []*DerivationPremise
	active       uint32
	transition   ast.ASTRef
	existentials []ast.ASTRef
	closed       bool
}

func NewDerivation(parent *ProofObligation, rule *base.Rule, transition ast.ASTHandle, existentials []ast.ASTHandle) (_ *Derivation, resultErr error) {
	if parent == nil || parent.valid() != nil || rule == nil {
		return nil, ErrManagerOwner
	}
	ownedRule, err := rule.Retain()
	if err != nil {
		return nil, err
	}
	result := &Derivation{parent: parent, rule: ownedRule}
	defer func() {
		if resultErr != nil {
			resultErr = errors.Join(resultErr, result.Close())
		}
	}()
	result.transition, err = parent.transformer.manager.NewRef(transition)
	if err != nil {
		return nil, err
	}
	result.existentials, err = refsFromHandles(parent.transformer.manager, existentials)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (d *Derivation) open() error {
	if d == nil || d.closed || d.parent == nil || d.parent.valid() != nil || d.rule == nil {
		return ErrManagerOwner
	}
	return nil
}

func (d *Derivation) AddPremise(transformer *PredicateTransformer, oldIndex uint32, summary ast.ASTHandle, must bool, auxiliaries []ast.ASTHandle) error {
	if err := d.open(); err != nil {
		return err
	}
	if transformer == nil || transformer.owner != d.parent.transformer.owner {
		return ErrManagerOwner
	}
	premise, err := newDerivationPremise(transformer, oldIndex, summary, must, auxiliaries)
	if err != nil {
		return err
	}
	d.premises = append(d.premises, premise)
	return nil
}

func (d *Derivation) Premises() ([]*DerivationPremise, error) {
	if err := d.open(); err != nil {
		return nil, err
	}
	return append([]*DerivationPremise(nil), d.premises...), nil
}

func (d *Derivation) Parent() (*ProofObligation, error) {
	if err := d.open(); err != nil {
		return nil, err
	}
	return d.parent, nil
}

func (d *Derivation) Rule() (*base.Rule, error) {
	if err := d.open(); err != nil {
		return nil, err
	}
	return d.rule.Retain()
}

func (d *Derivation) Transition() (ast.ASTHandle, error) {
	if err := d.open(); err != nil {
		return ast.ASTHandle{}, err
	}
	return d.parent.transformer.manager.RefGet(d.transition)
}

func (d *Derivation) Close() (result error) {
	if d == nil || d.closed {
		return nil
	}
	manager := d.parent.transformer.manager
	for i := len(d.premises) - 1; i >= 0; i-- {
		result = errors.Join(result, d.premises[i].close())
	}
	result = errors.Join(result, releaseNormalizeRefs(manager, d.existentials))
	if d.transition != (ast.ASTRef{}) {
		result = errors.Join(result, manager.DelRef(&d.transition))
	}
	result = errors.Join(result, d.rule.Close())
	if result == nil {
		d.premises = nil
		d.existentials = nil
		d.rule = nil
		d.parent = nil
		d.closed = true
	}
	return result
}
