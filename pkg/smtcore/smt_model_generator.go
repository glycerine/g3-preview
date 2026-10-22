// Source: src/smt/smt_model_generator.cpp dependency ordering and fresh-value
// source management.
package smtcore

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	protomodel "github.com/glycerine/g3/pkg/smtcore/proto_model"
	"github.com/glycerine/g3/pkg/util/lbool"
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

func (g *SMTModelGenerator) discoverValueProcedures(state *SMTInternalizerState) ([]*SMTEnode, map[*SMTEnode]SMTModelValueProcedure, error) {
	if err := g.open(); err != nil {
		return nil, nil, err
	}
	if state == nil || state.manager != g.manager || state.context != g.context || g.model == nil {
		return nil, nil, ast.ErrASTDomain
	}
	for _, theory := range state.theoryOrder {
		if modelTheory, ok := theory.(SMTModelValueTheory); ok {
			if err := modelTheory.InitializeModel(g); err != nil {
				return nil, nil, err
			}
		}
	}
	booleanSort, err := g.manager.MkBoolSort()
	if err != nil {
		return nil, nil, err
	}
	roots := make([]*SMTEnode, 0, len(state.enodeStack))
	procedures := make(map[*SMTEnode]SMTModelValueProcedure)
	for _, node := range state.enodeStack {
		if node == nil || node.root != node || !g.context.relevant[node.owner] {
			continue
		}
		roots = append(roots, node)
		sort, sortErr := g.manager.GetSort(node.owner)
		if sortErr != nil {
			return nil, nil, sortErr
		}
		var procedure SMTModelValueProcedure
		if sort == booleanSort {
			assignment := g.context.assignments[node.owner]
			if assignment == lbool.Undef {
				return nil, nil, ast.ErrASTDomain
			}
			value, valueErr := g.manager.MkFalse()
			if assignment == lbool.True {
				value, valueErr = g.manager.MkTrue()
			}
			if valueErr != nil {
				return nil, nil, valueErr
			}
			procedure, valueErr = NewSMTExpressionValueProcedure(value)
			if valueErr != nil {
				return nil, nil, valueErr
			}
		} else {
			family, familyErr := g.manager.GetFamilyID(sort)
			if familyErr != nil {
				return nil, nil, familyErr
			}
			theory := state.theories[family]
			if modelTheory, ok := theory.(SMTModelValueTheory); ok {
				if node.TheoryVar(family) != SMTNullTheoryVar {
					procedure, familyErr = modelTheory.ModelValue(node, g)
				} else {
					fresh, freshErr := g.MkExtraFreshValue(sort)
					if freshErr != nil {
						return nil, nil, freshErr
					}
					procedure, familyErr = NewSMTFreshValueProcedure(fresh)
				}
				if familyErr != nil {
					return nil, nil, familyErr
				}
			} else {
				unique, uniqueErr := g.manager.IsUniqueValue(node.owner)
				if uniqueErr != nil {
					return nil, nil, uniqueErr
				}
				value := node.owner
				if !unique {
					value, uniqueErr = g.model.FreshValue(sort)
					if uniqueErr != nil || value.IsNull() {
						if uniqueErr != nil {
							return nil, nil, uniqueErr
						}
						return nil, nil, ast.ErrASTDomain
					}
				}
				procedure, uniqueErr = NewSMTExpressionValueProcedure(value)
				if uniqueErr != nil {
					return nil, nil, uniqueErr
				}
			}
		}
		procedures[node] = procedure
	}
	return roots, procedures, nil
}

func (g *SMTModelGenerator) BuildStateValues(state *SMTInternalizerState) error {
	roots, procedures, err := g.discoverValueProcedures(state)
	if err != nil {
		return err
	}
	if err = g.registerExistingModelValues(state); err != nil {
		return err
	}
	if err = g.BuildValues(roots, procedures); err != nil {
		return err
	}
	if err = g.publishConstantInterpretations(state); err != nil {
		return err
	}
	if err = g.publishFunctionInterpretations(state); err != nil {
		return err
	}
	return g.finalizeTheoryModels(state)
}

func (g *SMTModelGenerator) registerExistingModelValues(state *SMTInternalizerState) error {
	if g.model == nil {
		return ast.ErrASTDomain
	}
	for _, node := range state.enodeStack {
		if node == nil || node.root != node || !g.context.relevant[node.owner] {
			continue
		}
		modelValue, err := g.manager.IsModelValue(node.owner)
		if err != nil {
			return err
		}
		if modelValue {
			if err = g.model.RegisterValue(node.owner); err != nil {
				return err
			}
		}
	}
	return nil
}

func (g *SMTModelGenerator) uninterpretedConstant(expression ast.ASTHandle) (ast.ASTHandle, bool, error) {
	kind, err := g.manager.GetKind(expression)
	if err != nil || kind != ast.ASTApp {
		return ast.ASTHandle{}, false, err
	}
	count, err := g.manager.GetNumArgs(expression)
	if err != nil || count != 0 {
		return ast.ASTHandle{}, false, err
	}
	family, err := g.manager.GetFamilyID(expression)
	if err != nil || family != ast.NullFamilyID {
		return ast.ASTHandle{}, false, err
	}
	declaration, err := g.manager.GetDecl(expression)
	return declaration, err == nil, err
}

func (g *SMTModelGenerator) publishConstantInterpretations(state *SMTInternalizerState) error {
	if g.model == nil {
		return ast.ErrASTDomain
	}
	for variable := int32(SMTFirstBoolVar); int(variable) < len(state.boolExpr); variable++ {
		expression := state.boolExpr[variable]
		if expression.IsNull() || !g.context.relevant[expression] {
			continue
		}
		declaration, constant, err := g.uninterpretedConstant(expression)
		if err != nil {
			return err
		}
		if !constant {
			continue
		}
		assignment := state.assignment[NewSMTLiteral(variable, false).Index()]
		if assignment == lbool.Undef {
			return ast.ErrASTDomain
		}
		value, err := g.manager.MkFalse()
		if assignment == lbool.True {
			value, err = g.manager.MkTrue()
		}
		if err != nil {
			return err
		}
		if err = g.model.Core().RegisterConstant(declaration, value); err != nil {
			return err
		}
	}
	for _, node := range state.enodeStack {
		if node == nil || node.root != node || !g.context.relevant[node.owner] {
			continue
		}
		value := g.rootValues[node]
		if value.IsNull() {
			continue
		}
		sort, err := g.manager.GetSort(node.owner)
		if err != nil {
			return err
		}
		booleanSort, err := g.manager.MkBoolSort()
		if err != nil {
			return err
		}
		if sort == booleanSort {
			continue
		}
		declaration, constant, err := g.uninterpretedConstant(node.owner)
		if err != nil {
			return err
		}
		if constant {
			if err = g.model.Core().RegisterConstant(declaration, value); err != nil {
				return err
			}
		}
	}
	return nil
}

func (g *SMTModelGenerator) includeFunctionInterpretation(state *SMTInternalizerState, declaration ast.ASTHandle) (bool, error) {
	family, err := g.manager.GetFamilyID(declaration)
	if err != nil {
		return false, err
	}
	if family == ast.NullFamilyID {
		return true, nil
	}
	if family == g.manager.BasicFamilyID() {
		return false, nil
	}
	theory := state.theories[family]
	if theory == nil {
		return true, nil
	}
	modelTheory, ok := theory.(SMTModelFunctionTheory)
	return ok && modelTheory.IncludeFunctionInterpretation(declaration), nil
}

func (g *SMTModelGenerator) publishFunctionInterpretations(state *SMTInternalizerState) error {
	if g.model == nil {
		return ast.ErrASTDomain
	}
	for _, node := range state.enodeStack {
		if node == nil || node.NumArgs() == 0 || !node.IsCongruenceRoot() || !g.context.relevant[node.owner] {
			continue
		}
		include, err := g.includeFunctionInterpretation(state, node.declaration)
		if err != nil {
			return err
		}
		if !include {
			continue
		}
		result := g.rootValues[node.root]
		if result.IsNull() {
			return ast.ErrASTDomain
		}
		arguments := make([]ast.ASTHandle, len(node.arguments))
		for index, argument := range node.arguments {
			if argument == nil || argument.root == nil {
				return ast.ErrASTDomain
			}
			arguments[index] = g.rootValues[argument.root]
			if arguments[index].IsNull() {
				return ast.ErrASTDomain
			}
		}
		interpretation, err := g.model.Core().FunctionInterpretation(node.declaration)
		if err != nil {
			return err
		}
		if interpretation == nil {
			interpretation, err = model.NewFunctionInterpretation(g.manager, node.NumArgs())
			if err != nil {
				return err
			}
			if err = g.model.Core().RegisterFunction(node.declaration, interpretation); err != nil {
				_ = interpretation.Close()
				return err
			}
		}
		if err = interpretation.InsertNewEntry(arguments, result); err != nil {
			return err
		}
	}
	return nil
}

func (g *SMTModelGenerator) finalizeTheoryModels(state *SMTInternalizerState) error {
	for _, theory := range state.theoryOrder {
		if finalizer, ok := theory.(SMTModelFinalizer); ok {
			if err := finalizer.FinalizeModel(g); err != nil {
				return err
			}
		}
	}
	return nil
}

func (g *SMTModelGenerator) FinalModel() (*model.Model, error) {
	if err := g.open(); err != nil {
		return nil, err
	}
	if g.model == nil {
		return nil, ast.ErrASTDomain
	}
	result, err := g.model.FinalModel()
	if err != nil {
		return nil, err
	}
	g.model = nil
	return result, nil
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
