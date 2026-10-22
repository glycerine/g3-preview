// Source: src/smt/smt_model_generator.cpp dependency ordering and fresh-value
// source management.
package smtcore

import "github.com/glycerine/g3/pkg/ast"

type SMTModelGenerator struct {
	manager    *ast.ASTManager
	extraFresh []*SMTExtraFreshValue
	freshIndex uint32
}

func NewSMTModelGenerator(manager *ast.ASTManager) (*SMTModelGenerator, error) {
	if manager == nil {
		return nil, ast.ErrASTHandle
	}
	return &SMTModelGenerator{manager: manager, freshIndex: 1}, nil
}

func (g *SMTModelGenerator) MkExtraFreshValue(sort ast.ASTHandle) (*SMTExtraFreshValue, error) {
	if g == nil || g.manager == nil {
		return nil, ast.ErrASTHandle
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
