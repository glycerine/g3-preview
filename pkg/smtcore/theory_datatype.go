// Source: src/smt/theory_datatype.cpp and src/smt/theory_datatype.h retained
// constructor, accessor and recognizer axioms.
package smtcore

import (
	"fmt"

	"github.com/glycerine/g3/pkg/ast"
	protomodel "github.com/glycerine/g3/pkg/smtcore/proto_model"
	"github.com/glycerine/g3/pkg/util/statistics"
)

type smtDatatypeScope struct {
	initialized, reconstructed, recognizers, splits int
	unsupported                                     bool
}

type smtDatatypeStatistics struct {
	constructorAxioms, accessorAxioms, updateAxioms, splits, occursChecks uint64
}

// SMTDatatypeTheory encodes constructor cases with recognizer clauses and
// publishes reconstruction and accessor equalities through the shared egraph.
type SMTDatatypeTheory struct {
	*SMTTheoryBase
	initialized        map[ast.ASTHandle]bool
	initializedTrail   []ast.ASTHandle
	reconstructed      map[ast.ASTHandle]bool
	reconstructedTrail []ast.ASTHandle
	recognizerValues   map[ast.ASTHandle]bool
	recognizerTrail    []ast.ASTHandle
	caseSplits         map[ast.ASTHandle]bool
	caseSplitTrail     []ast.ASTHandle
	scopes             []smtDatatypeScope
	unsupported        bool
	statistics         smtDatatypeStatistics
}

func NewSMTDatatypeTheory(family ast.FamilyID) (*SMTDatatypeTheory, error) {
	base, err := NewSMTTheoryBase(family)
	if err != nil {
		return nil, err
	}
	return &SMTDatatypeTheory{
		SMTTheoryBase:    base,
		initialized:      make(map[ast.ASTHandle]bool),
		reconstructed:    make(map[ast.ASTHandle]bool),
		recognizerValues: make(map[ast.ASTHandle]bool),
		caseSplits:       make(map[ast.ASTHandle]bool),
	}, nil
}

func (t *SMTDatatypeTheory) operation(expression ast.ASTHandle) (int32, bool, error) {
	state, err := t.State()
	if err != nil {
		return 0, false, err
	}
	if kind, kindErr := state.manager.GetKind(expression); kindErr != nil || kind != ast.ASTApp {
		return 0, false, kindErr
	}
	family, err := state.manager.GetFamilyID(expression)
	if err != nil || family != t.FamilyID() {
		return 0, false, err
	}
	op, err := state.manager.GetDeclKind(expression)
	return op, err == nil, err
}

func (t *SMTDatatypeTheory) equalityAxiom(left, right ast.ASTHandle, antecedent SMTLiteral) error {
	if left == right {
		return nil
	}
	equality, err := t.MkEqualityLiteral(left, right, true)
	if err != nil {
		return err
	}
	clause := []SMTLiteral{equality}
	if antecedent != SMTNullLiteral {
		clause = []SMTLiteral{antecedent.Negated(), equality}
	}
	state, _ := t.State()
	return state.addTheoryAxiom(t.FamilyID(), clause)
}

func (t *SMTDatatypeTheory) recognizerLiteral(constructor, value ast.ASTHandle) (SMTLiteral, ast.ASTHandle, error) {
	state, _ := t.State()
	parameter, err := state.manager.NewParameterAST(constructor)
	if err != nil {
		return SMTNullLiteral, ast.ASTHandle{}, err
	}
	application, err := state.manager.MkFamilyApp(t.FamilyID(), ast.ASTDatatypeIs, []ast.ASTParameter{parameter}, []ast.ASTHandle{value})
	deleteErr := state.manager.DelParameter(&parameter)
	if err != nil {
		return SMTNullLiteral, ast.ASTHandle{}, err
	}
	if deleteErr != nil {
		return SMTNullLiteral, ast.ASTHandle{}, deleteErr
	}
	if err = state.Internalize(application, true); err != nil {
		return SMTNullLiteral, ast.ASTHandle{}, err
	}
	literal, err := state.Literal(application)
	return literal, application, err
}

func (t *SMTDatatypeTheory) publishCases(value, sort ast.ASTHandle) error {
	if t.caseSplits[value] {
		return nil
	}
	state, _ := t.State()
	constructors, err := state.manager.DatatypeConstructors(sort)
	if err != nil || len(constructors) == 0 {
		if err != nil {
			return err
		}
		return ast.ErrASTDomain
	}
	literals := make([]SMTLiteral, len(constructors))
	for i, constructor := range constructors {
		literals[i], _, err = t.recognizerLiteral(constructor, value)
		if err != nil {
			return err
		}
	}
	if err = state.addTheoryAxiom(t.FamilyID(), literals); err != nil {
		return err
	}
	for i := range literals {
		for j := i + 1; j < len(literals); j++ {
			if err = state.addTheoryAxiom(t.FamilyID(), []SMTLiteral{literals[i].Negated(), literals[j].Negated()}); err != nil {
				return err
			}
		}
	}
	t.caseSplits[value] = true
	t.caseSplitTrail = append(t.caseSplitTrail, value)
	t.statistics.splits++
	return nil
}

func (t *SMTDatatypeTheory) publishConstructor(value, constructor ast.ASTHandle) error {
	state, _ := t.State()
	accessors, err := state.manager.DatatypeConstructorAccessors(constructor)
	if err != nil {
		return err
	}
	count, err := state.manager.GetNumArgs(value)
	if err != nil || int(count) != len(accessors) {
		if err != nil {
			return err
		}
		return ast.ErrASTDomain
	}
	for i, accessor := range accessors {
		selected, selectErr := state.manager.MkApp(accessor, []ast.ASTHandle{value})
		if selectErr != nil {
			return selectErr
		}
		argument, argumentErr := state.manager.GetArg(value, uint32(i))
		if argumentErr != nil {
			return argumentErr
		}
		if argumentErr = t.equalityAxiom(selected, argument, SMTNullLiteral); argumentErr != nil {
			return argumentErr
		}
		t.statistics.accessorAxioms++
	}
	literal, _, err := t.recognizerLiteral(constructor, value)
	if err != nil {
		return err
	}
	if err = state.addTheoryAxiom(t.FamilyID(), []SMTLiteral{literal}); err != nil {
		return err
	}
	t.statistics.constructorAxioms++
	return nil
}

func (t *SMTDatatypeTheory) publishUpdate(node *SMTEnode) error {
	if len(node.arguments) != 2 {
		return ast.ErrASTDomain
	}
	state, _ := t.State()
	declaration, err := state.manager.GetDecl(node.owner)
	if err != nil {
		return fmt.Errorf("datatype update declaration: %w", err)
	}
	updatedAccessor, matched, err := state.manager.DatatypeUpdateAccessor(declaration)
	if err != nil || !matched {
		if err != nil {
			return fmt.Errorf("datatype update accessor: %w", err)
		}
		return ast.ErrASTDomain
	}
	sort, err := state.manager.GetSort(node.arguments[0].owner)
	if err != nil {
		return fmt.Errorf("datatype update base sort: %w", err)
	}
	constructors, err := state.manager.DatatypeConstructors(sort)
	if err != nil {
		return fmt.Errorf("datatype update constructors: %w", err)
	}
	var constructor ast.ASTHandle
	for _, candidate := range constructors {
		_, same, indexErr := state.manager.DatatypeAccessorIndex(updatedAccessor, candidate)
		if indexErr != nil {
			return fmt.Errorf("datatype update accessor index: %w", indexErr)
		}
		if same {
			constructor = candidate
			break
		}
	}
	if constructor.IsNull() {
		return ast.ErrASTDomain
	}
	recognizer, _, err := t.recognizerLiteral(constructor, node.arguments[0].owner)
	if err != nil {
		return fmt.Errorf("datatype update recognizer: %w", err)
	}
	accessors, err := state.manager.DatatypeConstructorAccessors(constructor)
	if err != nil {
		return fmt.Errorf("datatype update constructor accessors: %w", err)
	}
	for _, accessor := range accessors {
		selected, selectErr := state.manager.MkApp(accessor, []ast.ASTHandle{node.owner})
		if selectErr != nil {
			return fmt.Errorf("datatype update result accessor: %w", selectErr)
		}
		right := node.arguments[1].owner
		if accessor != updatedAccessor {
			right, selectErr = state.manager.MkApp(accessor, []ast.ASTHandle{node.arguments[0].owner})
			if selectErr != nil {
				return fmt.Errorf("datatype update base accessor: %w", selectErr)
			}
		}
		if err = t.equalityAxiom(selected, right, recognizer); err != nil {
			return fmt.Errorf("datatype update field equality: %w", err)
		}
	}
	if err = t.equalityAxiom(node.owner, node.arguments[0].owner, recognizer.Negated()); err != nil {
		return fmt.Errorf("datatype update identity: %w", err)
	}
	t.statistics.updateAxioms++
	return nil
}

func (t *SMTDatatypeTheory) attachVariable(node *SMTEnode) error {
	if t.IsAttachedToVariable(node) {
		return nil
	}
	_, err := t.CreateVariable(node)
	return err
}

func (t *SMTDatatypeTheory) ensureVariable(node *SMTEnode) error {
	if t.initialized[node.owner] {
		return nil
	}
	if err := t.attachVariable(node); err != nil {
		return err
	}
	t.initialized[node.owner] = true
	t.initializedTrail = append(t.initializedTrail, node.owner)
	state, _ := t.State()
	sort, err := state.manager.GetSort(node.owner)
	if err != nil {
		return err
	}
	constructor, matched, err := state.manager.DatatypeConstructorApp(node.owner)
	if err != nil {
		return err
	}
	if matched {
		return t.publishConstructor(node.owner, constructor)
	}
	constructors, err := state.manager.DatatypeConstructors(sort)
	if err != nil {
		return err
	}
	if len(constructors) == 1 {
		// Source mk_var asserts reconstruction unconditionally for a
		// one-constructor datatype. Waiting for a recognizer assignment can
		// omit it when that auxiliary recognizer is not relevant.
		return t.assertConstructor(node.owner, constructors[0], SMTNullLiteral)
	}
	infinite, err := state.manager.SortIsInfinite(sort)
	if err != nil {
		return err
	}
	if !infinite {
		return t.publishCases(node.owner, sort)
	}
	return nil
}

func (t *SMTDatatypeTheory) InternalizeAtom(expression ast.ASTHandle, _ bool) (bool, error) {
	op, own, err := t.operation(expression)
	if err != nil || !own || op != ast.ASTDatatypeRecognizer && op != ast.ASTDatatypeIs {
		return false, err
	}
	state, _ := t.State()
	variable, found := state.boolVar[expression]
	if !found {
		variable, err = state.MkBoolVar(expression)
		if err != nil {
			return false, err
		}
	}
	data, err := state.BooleanData(variable)
	if err != nil {
		return false, err
	}
	if !data.IsTheoryAtom() {
		if err = state.SetVariableTheory(variable, t.FamilyID()); err != nil {
			return false, err
		}
	}
	return t.InternalizeTerm(expression)
}

func (t *SMTDatatypeTheory) InternalizeTerm(expression ast.ASTHandle) (bool, error) {
	operation, own, err := t.operation(expression)
	if err != nil || !own {
		return false, err
	}
	if operation < ast.ASTDatatypeConstructor || operation > ast.ASTDatatypeUpdateField {
		return false, nil
	}
	state, _ := t.State()
	count, err := state.manager.GetNumArgs(expression)
	if err != nil {
		return false, err
	}
	for i := uint32(0); i < count; i++ {
		argument, argumentErr := state.manager.GetArg(expression, i)
		if argumentErr != nil {
			return false, argumentErr
		}
		if argumentErr = state.Internalize(argument, false); argumentErr != nil {
			return false, argumentErr
		}
		if state.context.enodes[argument] == nil && (state.manager.IsTrue(argument) || state.manager.IsFalse(argument)) {
			if _, argumentErr = state.MkEnode(argument, false, true, false); argumentErr != nil {
				return false, fmt.Errorf("datatype Boolean value enode: %w", argumentErr)
			}
		}
	}
	node := state.context.enodes[expression]
	if node == nil {
		boolean, booleanErr := state.isBoolean(expression)
		if booleanErr != nil {
			return false, booleanErr
		}
		node, err = state.MkEnode(expression, false, boolean, true)
		if err != nil {
			return false, fmt.Errorf("datatype enode: %w", err)
		}
	}
	if operation == ast.ASTDatatypeConstructor || operation == ast.ASTDatatypeUpdateField {
		for _, argument := range node.arguments {
			sort, sortErr := state.manager.GetSort(argument.owner)
			if sortErr != nil {
				return false, sortErr
			}
			if family, familyErr := state.manager.GetFamilyID(sort); familyErr == nil && family == t.FamilyID() {
				if sortErr = t.ensureVariable(argument); sortErr != nil {
					return false, fmt.Errorf("datatype argument variable: %w", sortErr)
				}
			}
		}
		if err = t.ensureVariable(node); err != nil {
			return false, fmt.Errorf("datatype result variable: %w", err)
		}
		if operation == ast.ASTDatatypeUpdateField {
			if err = t.publishUpdate(node); err != nil {
				return false, err
			}
		}
	} else if len(node.arguments) == 1 {
		if err = t.ensureVariable(node.arguments[0]); err != nil {
			return false, err
		}
		if operation == ast.ASTDatatypeRecognizer || operation == ast.ASTDatatypeIs {
			value := node.arguments[0].owner
			if !t.recognizerValues[value] {
				t.recognizerValues[value] = true
				t.recognizerTrail = append(t.recognizerTrail, value)
			}
		}
		if operation == ast.ASTDatatypeAccessor {
			if err = t.attachVariable(node); err != nil {
				return false, err
			}
		}
	}
	return true, nil
}

func (*SMTDatatypeTheory) InternalizeEquality(ast.ASTHandle, int32) error { return nil }

func (t *SMTDatatypeTheory) ApplySortConstraint(node *SMTEnode, sort ast.ASTHandle) error {
	state, err := t.State()
	if err != nil {
		return err
	}
	family, err := state.manager.GetFamilyID(sort)
	if err != nil || family != t.FamilyID() || t.IsAttachedToVariable(node) {
		return err
	}
	infinite, err := state.manager.SortIsInfinite(sort)
	if err != nil || infinite {
		return err
	}
	return t.ensureVariable(node)
}

func (*SMTDatatypeTheory) UseDisequalities() bool                          { return false }
func (*SMTDatatypeTheory) NewEquality(SMTTheoryVar, SMTTheoryVar) error    { return nil }
func (*SMTDatatypeTheory) NewDisequality(SMTTheoryVar, SMTTheoryVar) error { return ast.ErrASTDomain }

func (t *SMTDatatypeTheory) Assign(variable int32, isTrue bool) error {
	if !isTrue {
		return nil
	}
	state, err := t.State()
	if err != nil {
		return err
	}
	expression, err := state.BoolExpression(variable)
	if err != nil || t.reconstructed[expression] {
		return err
	}
	op, own, err := t.operation(expression)
	if err != nil || !own || op != ast.ASTDatatypeRecognizer && op != ast.ASTDatatypeIs {
		return err
	}
	declaration, err := state.manager.GetDecl(expression)
	if err != nil {
		return err
	}
	constructor, matched, err := state.manager.DatatypeRecognizerConstructor(declaration)
	if err != nil || !matched {
		if err != nil {
			return err
		}
		return ast.ErrASTDomain
	}
	value, err := state.manager.GetArg(expression, 0)
	if err != nil {
		return err
	}
	valueConstructor, isConstructor, err := state.manager.DatatypeConstructorApp(value)
	if err != nil {
		return err
	}
	if isConstructor && valueConstructor == constructor {
		t.reconstructed[expression] = true
		t.reconstructedTrail = append(t.reconstructedTrail, expression)
		return nil
	}
	if err = t.assertConstructor(value, constructor, NewSMTLiteral(variable, false)); err != nil {
		return err
	}
	t.reconstructed[expression] = true
	t.reconstructedTrail = append(t.reconstructedTrail, expression)
	return nil
}

// Source assert_is_constructor_axiom; antecedent is null for the eager
// single-constructor case and the recognizer literal for assigned cases.
func (t *SMTDatatypeTheory) assertConstructor(value, constructor ast.ASTHandle, antecedent SMTLiteral) error {
	state, err := t.State()
	if err != nil {
		return err
	}
	accessors, err := state.manager.DatatypeConstructorAccessors(constructor)
	if err != nil {
		return err
	}
	arguments := make([]ast.ASTHandle, len(accessors))
	for i, accessor := range accessors {
		arguments[i], err = state.manager.MkApp(accessor, []ast.ASTHandle{value})
		if err != nil {
			return err
		}
	}
	constructed, err := state.manager.MkApp(constructor, arguments)
	if err != nil {
		return err
	}
	return t.equalityAxiom(value, constructed, antecedent)
}

func (t *SMTDatatypeTheory) PushScope() error {
	if err := t.SMTTheoryBase.PushScope(); err != nil {
		return err
	}
	t.scopes = append(t.scopes, smtDatatypeScope{
		initialized: len(t.initializedTrail), reconstructed: len(t.reconstructedTrail),
		recognizers: len(t.recognizerTrail), splits: len(t.caseSplitTrail), unsupported: t.unsupported,
	})
	return nil
}

func (t *SMTDatatypeTheory) PopScopes(count uint32) error {
	if count == 0 || count > uint32(len(t.scopes)) {
		return ast.ErrASTDomain
	}
	level := len(t.scopes) - int(count)
	scope := t.scopes[level]
	for _, expression := range t.initializedTrail[scope.initialized:] {
		delete(t.initialized, expression)
	}
	for _, expression := range t.reconstructedTrail[scope.reconstructed:] {
		delete(t.reconstructed, expression)
	}
	for _, expression := range t.recognizerTrail[scope.recognizers:] {
		delete(t.recognizerValues, expression)
	}
	for _, expression := range t.caseSplitTrail[scope.splits:] {
		delete(t.caseSplits, expression)
	}
	t.initializedTrail = t.initializedTrail[:scope.initialized]
	t.reconstructedTrail = t.reconstructedTrail[:scope.reconstructed]
	t.recognizerTrail = t.recognizerTrail[:scope.recognizers]
	t.caseSplitTrail = t.caseSplitTrail[:scope.splits]
	t.unsupported = scope.unsupported
	t.scopes = t.scopes[:level]
	return t.SMTTheoryBase.PopScopes(count)
}

func (t *SMTDatatypeTheory) FinalCheck() (SMTFinalCheckStatus, error) {
	if t.unsupported {
		return SMTFinalCheckGiveUp, nil
	}
	state, err := t.State()
	if err != nil {
		return SMTFinalCheckGiveUp, err
	}
	constructors := make(map[*SMTEnode]*SMTEnode)
	declarations := make(map[*SMTEnode]ast.ASTHandle)
	for expression := range t.initialized {
		node := state.context.enodes[expression]
		if node == nil {
			continue
		}
		declaration, matched, declarationErr := state.manager.DatatypeConstructorApp(expression)
		if declarationErr != nil {
			return SMTFinalCheckGiveUp, declarationErr
		}
		if !matched {
			continue
		}
		root := node.root
		if prior := constructors[root]; prior != nil && declarations[root] != declaration {
			equality, equalityErr := t.MkEqualityLiteral(prior.owner, node.owner, true)
			if equalityErr != nil {
				return SMTFinalCheckGiveUp, equalityErr
			}
			if equalityErr = state.addTheoryAxiom(t.FamilyID(), []SMTLiteral{equality.Negated()}); equalityErr != nil {
				return SMTFinalCheckGiveUp, equalityErr
			}
			return SMTFinalCheckContinue, nil
		}
		constructors[root], declarations[root] = node, declaration
	}
	for expression := range t.recognizerValues {
		node := state.context.enodes[expression]
		if node == nil || constructors[node.root] != nil || t.caseSplits[expression] {
			continue
		}
		sort, sortErr := state.manager.GetSort(expression)
		if sortErr != nil {
			return SMTFinalCheckGiveUp, sortErr
		}
		if splitErr := t.publishCases(expression, sort); splitErr != nil {
			return SMTFinalCheckGiveUp, splitErr
		}
		return SMTFinalCheckContinue, nil
	}
	t.statistics.occursChecks++
	visiting := make(map[*SMTEnode]int)
	finished := make(map[*SMTEnode]bool)
	incoming := make(map[*SMTEnode]*SMTEnode)
	path := make([]*SMTEnode, 0)
	var cycle []*SMTEnode
	var visit func(*SMTEnode) error
	visit = func(root *SMTEnode) error {
		if finished[root] || cycle != nil {
			return nil
		}
		if start, found := visiting[root]; found {
			cycle = append([]*SMTEnode(nil), path[start:]...)
			return nil
		}
		constructor := constructors[root]
		if constructor == nil {
			finished[root] = true
			return nil
		}
		visiting[root] = len(path)
		path = append(path, root)
		for _, argument := range constructor.arguments {
			sort, sortErr := state.manager.GetSort(argument.owner)
			if sortErr != nil {
				return sortErr
			}
			family, familyErr := state.manager.GetFamilyID(sort)
			if familyErr != nil {
				return familyErr
			}
			if family == t.FamilyID() {
				target := argument.root
				if _, active := visiting[target]; active || !finished[target] {
					incoming[target] = argument
				}
				if visitErr := visit(target); visitErr != nil {
					return visitErr
				}
			}
			if cycle != nil {
				break
			}
		}
		path = path[:len(path)-1]
		delete(visiting, root)
		finished[root] = true
		return nil
	}
	for root := range constructors {
		if err = visit(root); err != nil {
			return SMTFinalCheckGiveUp, err
		}
		if cycle != nil {
			break
		}
	}
	if cycle != nil {
		clause := make([]SMTLiteral, 0, len(cycle))
		for _, root := range cycle {
			constructor := constructors[root]
			argument := incoming[root]
			if constructor == nil || argument == nil {
				return SMTFinalCheckGiveUp, ast.ErrASTDomain
			}
			equality, equalityErr := t.MkEqualityLiteral(argument.owner, constructor.owner, true)
			if equalityErr != nil {
				return SMTFinalCheckGiveUp, equalityErr
			}
			clause = append(clause, equality.Negated())
		}
		if err = state.addTheoryAxiom(t.FamilyID(), clause); err != nil {
			return SMTFinalCheckGiveUp, err
		}
		return SMTFinalCheckContinue, nil
	}
	return SMTFinalCheckDone, nil
}

func (*SMTDatatypeTheory) Name() string { return "datatype" }

func (t *SMTDatatypeTheory) CollectStatistics(target *statistics.Statistics) error {
	if t == nil || target == nil {
		return ast.ErrASTHandle
	}
	target.UpdateUint64("datatype splits", t.statistics.splits)
	target.UpdateUint64("datatype constructor ax", t.statistics.constructorAxioms)
	target.UpdateUint64("datatype accessor ax", t.statistics.accessorAxioms)
	target.UpdateUint64("datatype update field ax", t.statistics.updateAxioms)
	target.UpdateUint64("datatype occurs check", t.statistics.occursChecks)
	return nil
}

func (t *SMTDatatypeTheory) ResetStatistics() {
	if t != nil {
		t.statistics = smtDatatypeStatistics{}
	}
}

func (t *SMTDatatypeTheory) InitializeModel(generator *SMTModelGenerator) error {
	state, err := t.State()
	if err != nil || generator == nil || generator.manager != state.manager || generator.model == nil {
		if err != nil {
			return err
		}
		return ast.ErrASTDomain
	}
	factory, err := protomodel.NewDatatypeFactory(generator.model)
	if err != nil {
		return err
	}
	return generator.RegisterFactory(factory)
}

type smtDatatypeValueProcedure struct {
	constructor  ast.ASTHandle
	dependencies []SMTModelValueDependency
}

func (p *smtDatatypeValueProcedure) Dependencies() []SMTModelValueDependency {
	if p == nil {
		return nil
	}
	return append([]SMTModelValueDependency(nil), p.dependencies...)
}

func (p *smtDatatypeValueProcedure) MakeValue(generator *SMTModelGenerator, values []ast.ASTHandle) (ast.ASTHandle, error) {
	if p == nil || generator == nil || p.constructor.IsNull() || len(values) != len(p.dependencies) {
		return ast.ASTHandle{}, ast.ErrASTDomain
	}
	return generator.manager.MkApp(p.constructor, values)
}

func (*smtDatatypeValueProcedure) IsFresh() bool { return false }

func (t *SMTDatatypeTheory) ModelValue(node *SMTEnode, generator *SMTModelGenerator) (SMTModelValueProcedure, error) {
	if node == nil || generator == nil || !t.IsAttachedToVariable(node) || generator.model == nil {
		return nil, ast.ErrASTDomain
	}
	state, err := t.State()
	if err != nil {
		return nil, err
	}
	for expression := range t.initialized {
		candidate := state.context.enodes[expression]
		if candidate == nil || candidate.root != node.root {
			continue
		}
		constructor, matched, constructorErr := state.manager.DatatypeConstructorApp(expression)
		if constructorErr != nil {
			return nil, constructorErr
		}
		if !matched {
			continue
		}
		dependencies := make([]SMTModelValueDependency, 0, len(candidate.arguments))
		for _, argument := range candidate.arguments {
			dependency, dependencyErr := NewSMTEnodeValueDependency(argument)
			if dependencyErr != nil {
				return nil, dependencyErr
			}
			dependencies = append(dependencies, dependency)
		}
		return &smtDatatypeValueProcedure{constructor: constructor, dependencies: dependencies}, nil
	}
	sort, err := state.manager.GetSort(node.owner)
	if err != nil {
		return nil, err
	}
	value, err := generator.SomeValue(sort)
	if err != nil {
		return nil, err
	}
	return NewSMTExpressionValueProcedure(value)
}
