// Source: src/smt/smt_model_checker.cpp finite-universe counterexample
// discovery and delayed instance publication.
package smtcore

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/ast/rewriter"
	coremodel "github.com/glycerine/g3/pkg/model"
	protomodel "github.com/glycerine/g3/pkg/smtcore/proto_model"
	"github.com/glycerine/g3/pkg/util/lbool"
)

type smtAuxiliarySolverFactory func() (*SMTSolver, error)

// SMTModelChecker checks universal quantifiers over finite proto-model
// universes and uses a fresh SMT solver for infinite domains. Counterexamples
// enter the ordinary quantifier queue so proof construction and
// internalization remain centralized in SMTQIInstanceBuilder.
type SMTModelChecker struct {
	manager        *ast.ASTManager
	state          *SMTInternalizerState
	registry       *SMTQuantifierRegistry
	modelFinder    *SMTModelFinder
	initialMaxCexs uint32
	maxCexs        uint32
	maxIterations  uint32
	iteration      uint32
	exhausted      bool
	valueTerms     map[ast.ASTHandle]ast.ASTHandle
	auxiliary      smtAuxiliarySolverFactory
	closed         bool
}

func (c *SMTModelChecker) SetModelFinder(finder *SMTModelFinder) error {
	if err := c.open(); err != nil {
		return err
	}
	if finder == nil || finder.manager != c.manager || finder.state != c.state {
		return ast.ErrASTDomain
	}
	c.modelFinder = finder
	return nil
}

func (c *SMTModelChecker) SetAuxiliarySolverFactory(factory smtAuxiliarySolverFactory) error {
	if err := c.open(); err != nil {
		return err
	}
	c.auxiliary = factory
	return nil
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

func (c *SMTModelChecker) ConfigureLifecycle(maxCexs, maxIterations uint32) error {
	if err := c.open(); err != nil {
		return err
	}
	c.initialMaxCexs = maxCexs
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

func (c *SMTModelChecker) addAuxiliaryBlockingClause(auxiliary *SMTSolver, skolems, values []ast.ASTHandle, finite []bool) (result error) {
	disequalities := make([]ast.ASTHandle, 0, len(skolems))
	refs := make([]ast.ASTRef, 0, len(skolems))
	defer func() {
		for i := len(refs) - 1; i >= 0; i-- {
			result = errors.Join(result, c.manager.DelRef(&refs[i]))
		}
	}()
	for i, skolem := range skolems {
		if finite[i] || skolem.IsNull() || values[i].IsNull() {
			continue
		}
		equal, err := c.manager.MkEq(skolem, values[i])
		if err != nil {
			return err
		}
		disequal, err := c.manager.MkNot(equal)
		if err != nil {
			return err
		}
		ref, err := c.manager.NewRef(disequal)
		if err != nil {
			return err
		}
		refs = append(refs, ref)
		disequalities = append(disequalities, disequal)
	}
	if len(disequalities) == 0 {
		return ast.ErrASTPending
	}
	clause, err := c.manager.MkOrArgs(disequalities)
	if err != nil {
		return err
	}
	return auxiliary.Assert(clause)
}

func (c *SMTModelChecker) addInstantiationSetRestrictions(auxiliary *SMTSolver, quantifier ast.ASTHandle, skolems []ast.ASTHandle, finite []bool) (asserted bool, result error) {
	if c.modelFinder == nil {
		return false, nil
	}
	for i, skolem := range skolems {
		if finite[i] || skolem.IsNull() {
			continue
		}
		values, err := c.modelFinder.InstantiationValues(quantifier, uint32(i))
		if err != nil {
			return false, err
		}
		if len(values) == 0 {
			continue
		}
		equalities := make([]ast.ASTHandle, 0, len(values))
		refs := make([]ast.ASTRef, 0, len(values))
		for _, value := range values {
			equal, e := c.manager.MkEq(skolem, value)
			if e != nil {
				for j := range refs {
					result = errors.Join(result, c.manager.DelRef(&refs[j]))
				}
				return false, errors.Join(e, result)
			}
			ref, e := c.manager.NewRef(equal)
			if e != nil {
				for j := range refs {
					result = errors.Join(result, c.manager.DelRef(&refs[j]))
				}
				return false, errors.Join(e, result)
			}
			refs = append(refs, ref)
			equalities = append(equalities, equal)
		}
		clause, err := c.manager.MkOrArgs(equalities)
		if err == nil {
			err = auxiliary.Assert(clause)
		}
		for j := range refs {
			err = errors.Join(err, c.manager.DelRef(&refs[j]))
		}
		if err != nil {
			return false, err
		}
		asserted = true
	}
	return asserted, nil
}

func (c *SMTModelChecker) auxiliaryBindings(cex *coremodel.ModelCore, quantifier ast.ASTHandle, values []ast.ASTHandle, finite []bool, skolemDecls []ast.ASTHandle, requireInverse bool) (raw, bindings []ast.ASTHandle, inverseOK bool, err error) {
	raw = append([]ast.ASTHandle(nil), values...)
	for i, declaration := range skolemDecls {
		if declaration.IsNull() {
			continue
		}
		value, e := cex.ConstantInterpretation(declaration)
		if e != nil {
			return nil, nil, false, e
		}
		if value.IsNull() {
			sort, e := c.manager.GetDeclarationSort(quantifier, uint32(i))
			if e != nil {
				return nil, nil, false, e
			}
			value, e = cex.GetSomeValue(sort)
			if e != nil {
				return nil, nil, false, e
			}
		}
		raw[i] = value
	}
	bindings = append([]ast.ASTHandle(nil), raw...)
	inverseOK = true
	if c.modelFinder == nil {
		return raw, bindings, !requireInverse, nil
	}
	for i := range bindings {
		if finite[i] {
			continue
		}
		term, _, found, e := c.modelFinder.Inverse(quantifier, uint32(i), bindings[i])
		if e != nil {
			return nil, nil, false, e
		}
		if found {
			bindings[i] = term
		} else if requireInverse {
			inverseOK = false
		}
	}
	return raw, bindings, inverseOK, nil
}

func (c *SMTModelChecker) auxiliaryCounterexamples(model *protomodel.ProtoModel, quantifier ast.ASTHandle, values []ast.ASTHandle, finite []bool, remaining *uint32) (found bool, result error) {
	if c.auxiliary == nil {
		return false, nil
	}
	body, err := c.manager.GetQuantifierBody(quantifier)
	if err != nil {
		return false, err
	}
	evaluated, err := c.manager.NewRef(ast.ASTHandle{})
	if err != nil {
		return false, err
	}
	defer func() { result = errors.Join(result, c.manager.DelRef(&evaluated)) }()
	if _, err = model.Eval(body, evaluated, true); err != nil {
		return false, err
	}
	evaluatedBody, err := c.manager.RefGet(evaluated)
	if err != nil {
		return false, err
	}
	skolems := append([]ast.ASTHandle(nil), values...)
	skolemDecls := make([]ast.ASTHandle, len(skolems))
	skolemRefs := make([]ast.ASTRef, 0, len(skolems))
	defer func() {
		for i := len(skolemRefs) - 1; i >= 0; i-- {
			result = errors.Join(result, c.manager.DelRef(&skolemRefs[i]))
		}
	}()
	for i := range skolems {
		if finite[i] {
			continue
		}
		sort, e := c.manager.GetDeclarationSort(quantifier, uint32(i))
		if e != nil {
			return false, e
		}
		skolems[i], e = c.manager.MkFreshConst("mbqi", sort)
		if e != nil {
			return false, e
		}
		ref, e := c.manager.NewRef(skolems[i])
		if e != nil {
			return false, e
		}
		skolemRefs = append(skolemRefs, ref)
		skolemDecls[i], e = c.manager.GetDecl(skolems[i])
		if e != nil {
			return false, e
		}
	}
	substitution, err := rewriter.NewVarSubstitution(c.manager, true)
	if err != nil {
		return false, err
	}
	defer func() { result = errors.Join(result, substitution.Close()) }()
	ground, err := c.manager.NewRef(ast.ASTHandle{})
	if err != nil {
		return false, err
	}
	defer func() { result = errors.Join(result, c.manager.DelRef(&ground)) }()
	if err = substitution.Substitute(evaluatedBody, skolems, ground); err != nil {
		return false, err
	}
	groundBody, err := c.manager.RefGet(ground)
	if err != nil {
		return false, err
	}
	negated, err := c.manager.MkNot(groundBody)
	if err != nil {
		return false, err
	}
	auxiliary, err := c.auxiliary()
	if err != nil {
		return false, err
	}
	defer func() { result = errors.Join(result, auxiliary.Close()) }()
	if err = auxiliary.Assert(negated); err != nil {
		return false, err
	}
	status, err := auxiliary.CheckSat(nil)
	if err != nil {
		return false, err
	}
	if status == lbool.False {
		return false, nil
	}
	if status != lbool.True {
		c.exhausted = true
		return false, nil
	}
	complete, err := auxiliary.AcquireModel()
	if err != nil {
		return false, err
	}
	if complete == nil {
		return false, ast.ErrASTPending
	}
	defer func() { result = errors.Join(result, complete.DecRef()) }()
	restricted, err := c.addInstantiationSetRestrictions(auxiliary, quantifier, skolems, finite)
	if err != nil {
		return false, err
	}
	first := complete
	for *remaining != 0 {
		cex := first
		first = nil
		if restricted || cex == nil {
			status, err = auxiliary.CheckSat(nil)
			if err != nil {
				return false, err
			}
			if status == lbool.False {
				break
			}
			if status != lbool.True {
				c.exhausted = true
				return found, nil
			}
			cex, err = auxiliary.AcquireModel()
			if err != nil {
				return false, err
			}
			if cex == nil {
				return false, ast.ErrASTPending
			}
		}
		raw, bindings, inverseOK, err := c.auxiliaryBindings(cex, quantifier, values, finite, skolemDecls, restricted)
		if err != nil {
			if cex != complete {
				err = errors.Join(err, cex.DecRef())
			}
			return false, err
		}
		if restricted && !inverseOK {
			if cex != complete {
				if err = cex.DecRef(); err != nil {
					return false, err
				}
			}
			break
		}
		inserted, err := c.addCounterexample(quantifier, bindings)
		if err != nil {
			if cex != complete {
				err = errors.Join(err, cex.DecRef())
			}
			return false, err
		}
		if inserted {
			found = true
			*remaining--
		}
		if *remaining != 0 {
			if err = c.addAuxiliaryBlockingClause(auxiliary, skolems, raw, finite); err != nil {
				if cex != complete {
					err = errors.Join(err, cex.DecRef())
				}
				return false, err
			}
		}
		if cex != complete {
			if err = cex.DecRef(); err != nil {
				return false, err
			}
		}
	}
	if restricted && !found && *remaining != 0 {
		_, bindings, _, err := c.auxiliaryBindings(complete, quantifier, values, finite, skolemDecls, false)
		if err != nil {
			return false, err
		}
		inserted, err := c.addCounterexample(quantifier, bindings)
		if err != nil {
			return false, err
		}
		if inserted {
			found = true
			*remaining--
		}
	}
	return found, nil
}

func (c *SMTModelChecker) checkQuantifier(model *protomodel.ProtoModel, quantifier ast.ASTHandle, remaining *uint32) (bool, error) {
	count, err := c.manager.GetNumDeclarations(quantifier)
	if err != nil {
		return false, err
	}
	universes := make([][]ast.ASTHandle, count)
	finite := make([]bool, count)
	boolean, err := c.manager.MkBoolSort()
	if err != nil {
		return false, err
	}
	hasInfinite := false
	for i := uint32(0); i < count; i++ {
		sort, err := c.manager.GetDeclarationSort(quantifier, i)
		if err != nil {
			return false, err
		}
		finite[i] = sort == boolean
		if !finite[i] {
			finite[i], err = model.IsFinite(sort)
			if err != nil {
				return false, err
			}
		}
		if finite[i] || c.auxiliary == nil {
			universes[i], err = c.universe(model, sort)
			if err != nil {
				return false, err
			}
		} else {
			hasInfinite = true
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
			if !finite[index] && c.auxiliary != nil {
				return visit(index + 1)
			}
			for _, value := range universes[index] {
				values[index] = value
				if err := visit(index + 1); err != nil || *remaining == 0 {
					return err
				}
			}
			return nil
		}
		if hasInfinite {
			inserted, err := c.auxiliaryCounterexamples(model, quantifier, values, finite, remaining)
			if err != nil {
				return err
			}
			found = found || inserted
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
	c.modelFinder = nil
	c.valueTerms = nil
	c.auxiliary = nil
	c.manager = nil
	return nil
}
