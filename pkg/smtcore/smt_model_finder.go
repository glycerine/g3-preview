// Source: src/smt/smt_model_finder.h; src/smt/smt_model_finder.cpp model_finder
// registration, scope, search and candidate-model lifecycle.
package smtcore

import (
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
	scopes      []int
	closed      bool
}

func NewSMTModelFinder(state *SMTInternalizerState) (*SMTModelFinder, error) {
	if state == nil || state.manager == nil || state.context == nil {
		return nil, ast.ErrASTHandle
	}
	return &SMTModelFinder{manager: state.manager, state: state, registered: make(map[ast.ASTHandle]struct{})}, nil
}

func (f *SMTModelFinder) open() error {
	if f == nil || f.manager == nil || f.state == nil || f.registered == nil || f.closed {
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
	return nil
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
		delete(f.registered, quantifier)
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

// FixModel is the production entry used during candidate-model construction.
// Quantifier-specific macro and AUF passes progressively extend this method.
func (f *SMTModelFinder) FixModel(model *protomodel.ProtoModel) error {
	if err := f.open(); err != nil {
		return err
	}
	if model == nil || model.Core() == nil {
		return ast.ErrASTHandle
	}
	_, err := f.relevantQuantifiers()
	return err
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
	f.closed = true
	f.quantifiers = nil
	f.scopes = nil
	f.registered = nil
	f.state = nil
	f.manager = nil
	return nil
}
