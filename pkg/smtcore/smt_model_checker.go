// Source: src/smt/smt_model_checker.cpp finite-universe counterexample
// discovery and delayed instance publication.
package smtcore

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/ast/rewriter"
	protomodel "github.com/glycerine/g3/pkg/smtcore/proto_model"
)

// SMTModelChecker checks universal quantifiers over the finite universes
// exposed by a candidate proto-model. Counterexamples are inserted into the
// ordinary quantifier queue, so proof construction and internalization remain
// centralized in SMTQIInstanceBuilder.
type SMTModelChecker struct {
	manager        *ast.ASTManager
	state          *SMTInternalizerState
	registry       *SMTQuantifierRegistry
	initialMaxCexs uint32
	maxCexs        uint32
	maxIterations  uint32
	iteration      uint32
	exhausted      bool
	valueTerms     map[ast.ASTHandle]ast.ASTHandle
	closed         bool
}

func (c *SMTModelChecker) SetCandidateTerms(rootValues map[*SMTEnode]ast.ASTHandle) error {
	if err := c.open(); err != nil {
		return err
	}
	c.valueTerms = make(map[ast.ASTHandle]ast.ASTHandle, len(rootValues))
	for root, value := range rootValues {
		if root != nil && !value.IsNull() {
			c.valueTerms[value] = root.owner
		}
	}
	return nil
}

func NewSMTModelChecker(state *SMTInternalizerState, maxCexs uint32) (*SMTModelChecker, error) {
	if state == nil || state.manager == nil || state.quantifiers == nil {
		return nil, ast.ErrASTHandle
	}
	return &SMTModelChecker{
		manager: state.manager, state: state, registry: state.quantifiers,
		initialMaxCexs: maxCexs, maxCexs: maxCexs, maxIterations: ^uint32(0),
	}, nil
}

func (c *SMTModelChecker) ConfigureLifecycle(maxIterations uint32) error {
	if err := c.open(); err != nil {
		return err
	}
	c.maxIterations = maxIterations
	return nil
}

func (c *SMTModelChecker) InitSearch() error {
	if err := c.open(); err != nil {
		return err
	}
	c.maxCexs = c.initialMaxCexs
	c.iteration = 0
	c.exhausted = false
	return nil
}

func (c *SMTModelChecker) Exhausted() bool {
	return c != nil && !c.closed && c.exhausted
}

func (c *SMTModelChecker) open() error {
	if c == nil || c.manager == nil || c.state == nil || c.registry == nil || c.closed {
		return ast.ErrASTHandle
	}
	return nil
}

func (c *SMTModelChecker) universe(model *protomodel.ProtoModel, sort ast.ASTHandle) ([]ast.ASTHandle, error) {
	boolean, err := c.manager.MkBoolSort()
	if err != nil {
		return nil, err
	}
	if sort == boolean {
		f, err := c.manager.MkFalse()
		if err != nil {
			return nil, err
		}
		t, err := c.manager.MkTrue()
		if err != nil {
			return nil, err
		}
		return []ast.ASTHandle{f, t}, nil
	}
	values, err := model.KnownUniverse(sort)
	if err != nil {
		return nil, err
	}
	if len(values) != 0 {
		return values, nil
	}
	value, err := model.SomeValue(sort)
	if err != nil {
		return nil, err
	}
	return []ast.ASTHandle{value}, nil
}

func (c *SMTModelChecker) bindingNode(value ast.ASTHandle) (*SMTEnode, error) {
	if term := c.valueTerms[value]; !term.IsNull() {
		value = term
	}
	if node := c.state.context.enodes[value]; node != nil {
		return node.root, nil
	}
	kind, err := c.manager.GetKind(value)
	if err != nil || kind != ast.ASTApp {
		if err != nil {
			return nil, err
		}
		return nil, ast.ErrASTDomain
	}
	if err = c.state.internalizeTerm(value); err != nil {
		return nil, err
	}
	if node := c.state.context.enodes[value]; node != nil {
		return node.root, nil
	}
	return c.state.MkEnode(value, false, false, true)
}

func (c *SMTModelChecker) addCounterexample(quantifier ast.ASTHandle, values []ast.ASTHandle) (bool, error) {
	nodes := make([]*SMTEnode, len(values))
	var maxGeneration uint32
	for i, value := range values {
		node, err := c.bindingNode(value)
		if err != nil {
			return false, err
		}
		nodes[i] = node
		if node.Generation() > maxGeneration {
			maxGeneration = node.Generation()
		}
	}
	return c.registry.AddInstance(quantifier, ast.ASTHandle{}, nodes, maxGeneration, 0, 0)
}

func (c *SMTModelChecker) checkQuantifier(model *protomodel.ProtoModel, quantifier ast.ASTHandle, remaining *uint32) (bool, error) {
	count, err := c.manager.GetNumDeclarations(quantifier)
	if err != nil {
		return false, err
	}
	universes := make([][]ast.ASTHandle, count)
	for i := uint32(0); i < count; i++ {
		sort, err := c.manager.GetDeclarationSort(quantifier, i)
		if err != nil {
			return false, err
		}
		universes[i], err = c.universe(model, sort)
		if err != nil {
			return false, err
		}
	}
	values := make([]ast.ASTHandle, count)
	found := false
	var visit func(uint32) error
	visit = func(index uint32) (result error) {
		if *remaining == 0 {
			return nil
		}
		if index < count {
			for _, value := range universes[index] {
				values[index] = value
				if err := visit(index + 1); err != nil || *remaining == 0 {
					return err
				}
			}
			return nil
		}
		instance, err := c.manager.NewRef(ast.ASTHandle{})
		if err != nil {
			return err
		}
		defer func() { result = errors.Join(result, c.manager.DelRef(&instance)) }()
		if err = rewriter.InstantiateQuantifier(c.manager, quantifier, values, instance); err != nil {
			return err
		}
		term, err := c.manager.RefGet(instance)
		if err != nil {
			return err
		}
		evaluated, err := c.manager.NewRef(ast.ASTHandle{})
		if err != nil {
			return err
		}
		defer func() { result = errors.Join(result, c.manager.DelRef(&evaluated)) }()
		if _, err = model.Eval(term, evaluated, true); err != nil {
			return err
		}
		value, err := c.manager.RefGet(evaluated)
		if err != nil {
			return err
		}
		if !c.manager.IsFalse(value) {
			return nil
		}
		inserted, err := c.addCounterexample(quantifier, values)
		if err != nil {
			return err
		}
		if inserted {
			found = true
			*remaining--
		}
		return nil
	}
	return found, visit(0)
}

// Check returns true when every supplied quantifier holds in the candidate
// finite model. A false result means at least one new instance was queued.
func (c *SMTModelChecker) Check(model *protomodel.ProtoModel, quantifiers []ast.ASTHandle) (bool, error) {
	if err := c.open(); err != nil {
		return false, err
	}
	if model == nil || model.Core() == nil {
		return false, ast.ErrASTHandle
	}
	if len(quantifiers) != 0 && c.iteration >= c.maxIterations {
		c.exhausted = true
		return false, nil
	}
	c.exhausted = false
	remaining := c.maxCexs
	if remaining == 0 {
		remaining = 1
	}
	found := false
	if len(quantifiers) != 0 {
		c.iteration++
	}
	for _, quantifier := range quantifiers {
		added, err := c.checkQuantifier(model, quantifier, &remaining)
		if err != nil {
			return false, err
		}
		found = found || added
		if remaining == 0 {
			break
		}
	}
	c.maxCexs += c.initialMaxCexs
	return !found, nil
}

func (c *SMTModelChecker) Close() error {
	if c == nil || c.manager == nil {
		return ast.ErrASTHandle
	}
	if c.closed {
		return nil
	}
	c.closed = true
	c.state = nil
	c.registry = nil
	c.valueTerms = nil
	c.manager = nil
	return nil
}
