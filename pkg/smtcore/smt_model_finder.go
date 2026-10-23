// Source: src/smt/smt_model_finder.h; src/smt/smt_model_finder.cpp model_finder
// registration, scope, search and candidate-model lifecycle.
package smtcore

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
	protomodel "github.com/glycerine/g3/pkg/smtcore/proto_model"
	"github.com/glycerine/g3/pkg/util/lbool"
)

// SMTModelFinder owns the quantified-formula analysis used to extend a ground
// candidate model before MBQI checks it. The macro and AUF algorithms attach
// their per-quantifier state to this lifecycle.
type SMTModelFinder struct {
	manager     *ast.ASTManager
	state       *SMTInternalizerState
	quantifiers []ast.ASTHandle
	registered  map[ast.ASTHandle]struct{}
	sets        map[ast.ASTHandle][]*smtModelFinderInstantiationSet
	scopes      []int
	closed      bool
}

func NewSMTModelFinder(state *SMTInternalizerState) (*SMTModelFinder, error) {
	if state == nil || state.manager == nil || state.context == nil {
		return nil, ast.ErrASTHandle
	}
	return &SMTModelFinder{
		manager: state.manager, state: state, registered: make(map[ast.ASTHandle]struct{}),
		sets: make(map[ast.ASTHandle][]*smtModelFinderInstantiationSet),
	}, nil
}

func (f *SMTModelFinder) open() error {
	if f == nil || f.manager == nil || f.state == nil || f.registered == nil || f.sets == nil || f.closed {
		return ast.ErrASTHandle
	}
	return nil
}

func (f *SMTModelFinder) RegisterQuantifier(quantifier ast.ASTHandle) error {
	if err := f.open(); err != nil {
		return err
	}
	kind, err := f.manager.GetKind(quantifier)
	if err != nil {
		return err
	}
	if kind != ast.ASTQuantifier {
		return ast.ErrASTDomain
	}
	if _, found := f.registered[quantifier]; found {
		return ast.ErrASTDomain
	}
	f.registered[quantifier] = struct{}{}
	f.quantifiers = append(f.quantifiers, quantifier)
	count, err := f.manager.GetNumDeclarations(quantifier)
	if err != nil {
		return err
	}
	f.sets[quantifier] = make([]*smtModelFinderInstantiationSet, count)
	return nil
}

func (f *SMTModelFinder) instantiationSet(quantifier ast.ASTHandle, variable uint32, create bool) (*smtModelFinderInstantiationSet, error) {
	if err := f.open(); err != nil {
		return nil, err
	}
	sets, found := f.sets[quantifier]
	if !found || variable >= uint32(len(sets)) {
		return nil, ast.ErrASTDomain
	}
	if sets[variable] == nil && create {
		value, err := newSMTModelFinderInstantiationSet(f.manager)
		if err != nil {
			return nil, err
		}
		sets[variable] = value
	}
	return sets[variable], nil
}

func (f *SMTModelFinder) InsertInstantiationTerm(quantifier ast.ASTHandle, variable uint32, term ast.ASTHandle, generation uint32) (bool, error) {
	set, err := f.instantiationSet(quantifier, variable, true)
	if err != nil {
		return false, err
	}
	return set.Insert(term, generation)
}

func (f *SMTModelFinder) Inverse(quantifier ast.ASTHandle, variable uint32, value ast.ASTHandle) (term ast.ASTHandle, generation uint32, found bool, err error) {
	set, err := f.instantiationSet(quantifier, variable, false)
	if err != nil || set == nil {
		return term, 0, false, err
	}
	return set.Inverse(value)
}

func (f *SMTModelFinder) InstantiationValues(quantifier ast.ASTHandle, variable uint32) ([]ast.ASTHandle, error) {
	set, err := f.instantiationSet(quantifier, variable, false)
	if err != nil || set == nil {
		return nil, err
	}
	return set.Values()
}

func (f *SMTModelFinder) releaseQuantifier(quantifier ast.ASTHandle) (result error) {
	for _, set := range f.sets[quantifier] {
		if set != nil {
			result = errors.Join(result, set.Close())
		}
	}
	if result == nil {
		delete(f.sets, quantifier)
		delete(f.registered, quantifier)
	}
	return result
}

func (f *SMTModelFinder) PushScope() error {
	if err := f.open(); err != nil {
		return err
	}
	f.scopes = append(f.scopes, len(f.quantifiers))
	return nil
}

func (f *SMTModelFinder) PopScopes(count uint32) error {
	if err := f.open(); err != nil {
		return err
	}
	if count == 0 || count > uint32(len(f.scopes)) {
		return ast.ErrASTDomain
	}
	level := len(f.scopes) - int(count)
	limit := f.scopes[level]
	for _, quantifier := range f.quantifiers[limit:] {
		if err := f.releaseQuantifier(quantifier); err != nil {
			return err
		}
	}
	f.quantifiers = f.quantifiers[:limit]
	f.scopes = f.scopes[:level]
	return nil
}

func (f *SMTModelFinder) InitSearch() error { return f.open() }

func (f *SMTModelFinder) relevantQuantifiers() ([]ast.ASTHandle, error) {
	if err := f.open(); err != nil {
		return nil, err
	}
	result := make([]ast.ASTHandle, 0, len(f.quantifiers))
	for _, quantifier := range f.quantifiers {
		if f.state.context.relevant[quantifier] && f.state.context.assignments[quantifier] == lbool.True {
			result = append(result, quantifier)
		}
	}
	return result, nil
}

func (f *SMTModelFinder) collectPatternTerm(quantifier, term ast.ASTHandle, declarations uint32) error {
	kind, err := f.manager.GetKind(term)
	if err != nil {
		return err
	}
	if kind != ast.ASTApp {
		return nil
	}
	declaration, err := f.manager.GetDecl(term)
	if err != nil {
		return err
	}
	count, err := f.manager.GetNumArgs(term)
	if err != nil {
		return err
	}
	ground := f.state.context.byDeclaration[declaration]
	for position := uint32(0); position < count; position++ {
		argument, err := f.manager.GetArg(term, position)
		if err != nil {
			return err
		}
		argumentKind, err := f.manager.GetKind(argument)
		if err != nil {
			return err
		}
		if argumentKind == ast.ASTVar {
			index, err := f.manager.GetVarIndex(argument)
			if err != nil {
				return err
			}
			if index < declarations {
				variable := declarations - index - 1
				for _, application := range ground {
					if application == nil || position >= uint32(len(application.arguments)) {
						continue
					}
					binding := application.arguments[position].root
					if _, err = f.InsertInstantiationTerm(quantifier, variable, binding.owner, binding.Generation()); err != nil {
						return err
					}
				}
			}
		}
		if err = f.collectPatternTerm(quantifier, argument, declarations); err != nil {
			return err
		}
	}
	return nil
}

func (f *SMTModelFinder) collectPatternInstantiationTerms(quantifier ast.ASTHandle) error {
	declarations, err := f.manager.GetNumDeclarations(quantifier)
	if err != nil {
		return err
	}
	patterns, err := f.manager.GetNumPatterns(quantifier)
	if err != nil {
		return err
	}
	for i := uint32(0); i < patterns; i++ {
		pattern, err := f.manager.GetPattern(quantifier, i)
		if err != nil {
			return err
		}
		if err = f.collectPatternTerm(quantifier, pattern, declarations); err != nil {
			return err
		}
	}
	return nil
}

// FixModel is the production entry used during candidate-model construction.
// Quantifier-specific macro and AUF passes progressively extend this method.
func (f *SMTModelFinder) FixModel(model *protomodel.ProtoModel) error {
	if err := f.open(); err != nil {
		return err
	}
	if model == nil || model.Core() == nil {
		return ast.ErrASTHandle
	}
	relevant, err := f.relevantQuantifiers()
	if err != nil {
		return err
	}
	for _, quantifier := range relevant {
		if err = f.collectPatternInstantiationTerms(quantifier); err != nil {
			return err
		}
		for _, set := range f.sets[quantifier] {
			if set != nil {
				if err = set.MakeInverse(model); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (f *SMTModelFinder) Close() error {
	if f == nil {
		return ast.ErrASTHandle
	}
	if f.closed {
		return nil
	}
	if f.manager == nil {
		return ast.ErrASTHandle
	}
	var result error
	for _, quantifier := range f.quantifiers {
		result = errors.Join(result, f.releaseQuantifier(quantifier))
	}
	if result != nil {
		return result
	}
	f.closed = true
	f.quantifiers = nil
	f.scopes = nil
	f.registered = nil
	f.sets = nil
	f.state = nil
	f.manager = nil
	return result
}
