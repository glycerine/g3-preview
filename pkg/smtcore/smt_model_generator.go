// Source: src/smt/smt_model_generator.cpp dependency ordering and fresh-value
// source management.
package smtcore

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	protomodel "github.com/glycerine/g3/pkg/smtcore/proto_model"
)

type SMTModelGenerator struct {
	manager    *ast.ASTManager
	context    *SMTCheckerContext
	model      *protomodel.ProtoModel
	extraFresh []*SMTExtraFreshValue
	rootValues map[*SMTEnode]ast.ASTHandle
	refs       []ast.ASTRef
	freshIndex uint32
	closed     bool
}

func NewSMTModelGenerator(manager *ast.ASTManager) (*SMTModelGenerator, error) {
	if manager == nil {
		return nil, ast.ErrASTHandle
	}
	return &SMTModelGenerator{manager: manager, rootValues: make(map[*SMTEnode]ast.ASTHandle), freshIndex: 1}, nil
}

func (g *SMTModelGenerator) open() error {
	if g == nil || g.manager == nil || g.closed {
		return ast.ErrASTHandle
	}
	return nil
}

func (g *SMTModelGenerator) SetContext(context *SMTCheckerContext) error {
	if err := g.open(); err != nil {
		return err
	}
	if context == nil || context.manager != g.manager || g.context != nil {
		return ast.ErrASTDomain
	}
	g.context = context
	return nil
}

func (g *SMTModelGenerator) InitializeModel(config model.ModelEvaluatorConfig) error {
	if err := g.open(); err != nil {
		return err
	}
	if g.context == nil || g.model != nil {
		return ast.ErrASTDomain
	}
	value, err := protomodel.NewProtoModel(g.manager, config)
	if err != nil {
		return err
	}
	g.model = value
	return nil
}

func (g *SMTModelGenerator) RegisterFactory(factory protomodel.ValueFactory) error {
	if err := g.open(); err != nil {
		return err
	}
	if g.model == nil {
		return ast.ErrASTDomain
	}
	return g.model.RegisterFactory(factory)
}

func (g *SMTModelGenerator) SomeValue(sort ast.ASTHandle) (ast.ASTHandle, error) {
	if err := g.open(); err != nil {
		return ast.ASTHandle{}, err
	}
	if g.model == nil {
		return ast.ASTHandle{}, ast.ErrASTDomain
	}
	return g.model.SomeValue(sort)
}

func (g *SMTModelGenerator) pin(value ast.ASTHandle) error {
	ref, err := g.manager.NewRef(value)
	if err != nil {
		return err
	}
	g.refs = append(g.refs, ref)
	return nil
}

func (g *SMTModelGenerator) MkExtraFreshValue(sort ast.ASTHandle) (*SMTExtraFreshValue, error) {
	if err := g.open(); err != nil {
		return nil, err
	}
	infinite, err := g.manager.SortIsInfinite(sort)
	if err != nil {
		return nil, err
	}
	if !infinite {
		return nil, ast.ErrASTDomain
	}
	value, err := NewSMTExtraFreshValue(sort, g.freshIndex)
	if err != nil {
		return nil, err
	}
	g.freshIndex++
	g.extraFresh = append(g.extraFresh, value)
	return value, nil
}

func (g *SMTModelGenerator) BuildValues(roots []*SMTEnode, procedures map[*SMTEnode]SMTModelValueProcedure) error {
	if err := g.open(); err != nil {
		return err
	}
	if g.model == nil || len(g.rootValues) != 0 {
		return ast.ErrASTDomain
	}
	ordered, err := g.topSortSources(roots, procedures)
	if err != nil {
		return err
	}
	for _, source := range ordered {
		if source.IsFreshValue() {
			fresh, e := source.FreshValue()
			if e != nil {
				return e
			}
			value, e := g.model.FreshValue(fresh.sort)
			if e != nil || value.IsNull() {
				if e != nil {
					return e
				}
				return ast.ErrASTDomain
			}
			if e = g.pin(value); e != nil {
				return e
			}
			if e = fresh.SetValue(value); e != nil {
				return e
			}
			continue
		}
		node, e := source.Enode()
		if e != nil {
			return e
		}
		procedure := procedures[node]
		dependencies := procedure.Dependencies()
		values := make([]ast.ASTHandle, 0, len(dependencies))
		for _, dependency := range dependencies {
			if dependency.IsFreshValue() {
				fresh, depErr := dependency.FreshValue()
				if depErr != nil {
					return depErr
				}
				value, depErr := fresh.Value()
				if depErr != nil {
					return depErr
				}
				values = append(values, value)
			} else {
				child, depErr := dependency.Enode()
				if depErr != nil {
					return depErr
				}
				value := g.rootValues[child]
				if value.IsNull() {
					return ast.ErrASTDomain
				}
				values = append(values, value)
			}
		}
		value, e := procedure.MakeValue(g, values)
		if e != nil || value.IsNull() {
			if e != nil {
				return e
			}
			return ast.ErrASTDomain
		}
		if e = g.model.RegisterValue(value); e != nil {
			return e
		}
		if e = g.pin(value); e != nil {
			return e
		}
		g.rootValues[node] = value
	}
	return nil
}

func (g *SMTModelGenerator) Value(node *SMTEnode) (ast.ASTHandle, error) {
	if err := g.open(); err != nil {
		return ast.ASTHandle{}, err
	}
	if node == nil || node.root == nil {
		return ast.ASTHandle{}, ast.ErrASTDomain
	}
	value := g.rootValues[node.root]
	if value.IsNull() {
		return ast.ASTHandle{}, ast.ErrASTDomain
	}
	return value, nil
}

func (g *SMTModelGenerator) Close() error {
	if g == nil || g.manager == nil {
		return ast.ErrASTHandle
	}
	if g.closed {
		return nil
	}
	var result error
	if g.model != nil {
		result = errors.Join(result, g.model.Close())
		g.model = nil
	}
	for index := len(g.refs); index > 0; index-- {
		result = errors.Join(result, g.manager.DelRef(&g.refs[index-1]))
	}
	g.refs = nil
	if result == nil {
		g.closed = true
		g.context = nil
		g.manager = nil
	}
	return result
}

type smtModelSourceKey struct {
	node  *SMTEnode
	fresh *SMTExtraFreshValue
}

func smtModelDependencyKey(dependency SMTModelValueDependency) (smtModelSourceKey, error) {
	if dependency.IsFreshValue() {
		value, err := dependency.FreshValue()
		return smtModelSourceKey{fresh: value}, err
	}
	node, err := dependency.Enode()
	return smtModelSourceKey{node: node}, err
}

func smtModelKeyDependency(key smtModelSourceKey) (SMTModelValueDependency, error) {
	if key.fresh != nil {
		return NewSMTFreshValueDependency(key.fresh)
	}
	return NewSMTEnodeValueDependency(key.node)
}

func (g *SMTModelGenerator) topSortSources(roots []*SMTEnode, procedures map[*SMTEnode]SMTModelValueProcedure) ([]SMTModelValueDependency, error) {
	if g == nil || g.manager == nil {
		return nil, ast.ErrASTHandle
	}
	colors := make(map[smtModelSourceKey]uint8)
	result := make([]SMTModelValueDependency, 0, len(roots)+len(g.extraFresh))
	var visit func(smtModelSourceKey) error
	visit = func(key smtModelSourceKey) error {
		switch colors[key] {
		case 1:
			return ast.ErrASTDomain
		case 2:
			return nil
		}
		colors[key] = 1
		if key.fresh != nil {
			for _, root := range roots {
				if root == nil || root.root != root {
					return ast.ErrASTDomain
				}
				procedure := procedures[root]
				if procedure == nil {
					return ast.ErrASTDomain
				}
				sort, err := g.manager.GetSort(root.owner)
				if err != nil {
					return err
				}
				if sort == key.fresh.sort && !procedure.IsFresh() {
					if err = visit(smtModelSourceKey{node: root}); err != nil {
						return err
					}
				}
			}
		} else {
			if key.node == nil || key.node.root != key.node {
				return ast.ErrASTDomain
			}
			procedure := procedures[key.node]
			if procedure == nil {
				return ast.ErrASTDomain
			}
			for _, dependency := range procedure.Dependencies() {
				child, err := smtModelDependencyKey(dependency)
				if err != nil {
					return err
				}
				if err = visit(child); err != nil {
					return err
				}
			}
		}
		colors[key] = 2
		dependency, err := smtModelKeyDependency(key)
		if err != nil {
			return err
		}
		result = append(result, dependency)
		return nil
	}
	for _, fresh := range g.extraFresh {
		if err := visit(smtModelSourceKey{fresh: fresh}); err != nil {
			return nil, err
		}
	}
	for _, root := range roots {
		if procedure := procedures[root]; procedure == nil {
			return nil, ast.ErrASTDomain
		} else if procedure.IsFresh() {
			if err := visit(smtModelSourceKey{node: root}); err != nil {
				return nil, err
			}
		}
	}
	for _, root := range roots {
		if err := visit(smtModelSourceKey{node: root}); err != nil {
			return nil, err
		}
	}
	return result, nil
}
