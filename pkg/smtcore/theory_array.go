// Source: retained store/select axiom fragment of src/smt/theory_array_base.h,
// src/smt/theory_array_base.cpp, src/smt/theory_array.h, and
// src/smt/theory_array.cpp.
package smtcore

import (
	"errors"
	"fmt"

	"github.com/glycerine/g3/pkg/ast"
	protomodel "github.com/glycerine/g3/pkg/smtcore/proto_model"
	"github.com/glycerine/g3/pkg/util/statistics"
)

type smtArrayScope struct {
	stores, selects, defaults, epsilons int
	selectAxioms, defaultAxioms         int
	unsupported                         bool
}

type smtArrayAxiomPair struct {
	observer ast.ASTHandle
	array    ast.ASTHandle
}

type smtArrayStatistics struct {
	axiom1, axiom2, extensionality                        uint64
	mapAxiom, defaultMap, selectConst, defaultStore       uint64
	defaultConst, defaultAsArray, selectAsArray, eqSplits uint64
}

// SMTArrayTheory internalizes read-over-write, constant-array, map, default,
// as-array, and extensionality axioms and constructs retained array models.
type SMTArrayTheory struct {
	*SMTTheoryBase
	stores, selects, defaults             map[ast.ASTHandle]bool
	storeTrail, selectTrail, defaultTrail []ast.ASTHandle
	selectAxioms, defaultAxioms           map[smtArrayAxiomPair]bool
	selectAxiomTrail, defaultAxiomTrail   []smtArrayAxiomPair
	epsilons                              map[ast.ASTHandle]ast.ASTHandle
	epsilonTrail                          []ast.ASTHandle
	scopes                                []smtArrayScope
	unsupported                           bool
	statistics                            smtArrayStatistics
}

func NewSMTArrayTheory(family ast.FamilyID) (*SMTArrayTheory, error) {
	base, err := NewSMTTheoryBase(family)
	if err != nil {
		return nil, err
	}
	return &SMTArrayTheory{
		SMTTheoryBase: base,
		stores:        make(map[ast.ASTHandle]bool),
		selects:       make(map[ast.ASTHandle]bool),
		defaults:      make(map[ast.ASTHandle]bool),
		selectAxioms:  make(map[smtArrayAxiomPair]bool),
		defaultAxioms: make(map[smtArrayAxiomPair]bool),
		epsilons:      make(map[ast.ASTHandle]ast.ASTHandle),
	}, nil
}

func (t *SMTArrayTheory) arrayOperation(expression ast.ASTHandle) (int32, bool, error) {
	state, err := t.State()
	if err != nil {
		return 0, false, err
	}
	kind, err := state.manager.GetKind(expression)
	if err != nil || kind != ast.ASTApp {
		return 0, false, err
	}
	family, err := state.manager.GetFamilyID(expression)
	if err != nil || family != t.FamilyID() {
		return 0, false, err
	}
	op, err := state.manager.GetDeclKind(expression)
	return op, err == nil, err
}

func (t *SMTArrayTheory) InternalizeAtom(expression ast.ASTHandle, _ bool) (bool, error) {
	if op, array, err := t.arrayOperation(expression); err != nil {
		return false, err
	} else if array {
		if op == ast.ASTArrayOpSelect || op == ast.ASTArrayOpDefault {
			_, termErr := t.InternalizeTerm(expression)
			if termErr != nil {
				return false, fmt.Errorf("array select atom: %w", termErr)
			}
			state, _ := t.State()
			variable, found := state.boolVar[expression]
			if !found {
				variable, termErr = state.MkBoolVar(expression)
				if termErr != nil {
					return false, fmt.Errorf("array select Boolean variable: %w", termErr)
				}
			}
			data, dataErr := state.BooleanData(variable)
			if dataErr != nil {
				return false, dataErr
			}
			if !data.IsTheoryAtom() {
				if dataErr = state.SetVariableTheory(variable, t.FamilyID()); dataErr != nil {
					return false, fmt.Errorf("array select Boolean theory: %w", dataErr)
				}
			}
			return true, nil
		}
		t.unsupported = true
	}
	return false, nil
}

func (t *SMTArrayTheory) InternalizeTerm(expression ast.ASTHandle) (bool, error) {
	state, err := t.State()
	if err != nil {
		return false, err
	}
	op, array, err := t.arrayOperation(expression)
	if err != nil || !array {
		return false, err
	}
	count, err := state.manager.GetNumArgs(expression)
	if err != nil {
		return false, err
	}
	for index := uint32(0); index < count; index++ {
		argument, argErr := state.manager.GetArg(expression, index)
		if argErr != nil {
			return false, argErr
		}
		if argErr = state.Internalize(argument, false); argErr != nil {
			return false, fmt.Errorf("array term argument %d: %w", index, argErr)
		}
	}
	sort, err := state.manager.GetSort(expression)
	if err != nil {
		return false, err
	}
	handled := false
	if family, familyErr := state.manager.GetFamilyID(sort); familyErr == nil && family == t.FamilyID() {
		handled = true
		node := state.context.enodes[expression]
		if node == nil {
			node, err = state.MkEnode(expression, false, false, true)
			if err != nil {
				return false, err
			}
		}
		if !t.IsAttachedToVariable(node) {
			if _, err = t.CreateVariable(node); err != nil {
				return false, err
			}
		}
	}
	switch op {
	case ast.ASTArrayOpStore:
		if err = t.assertStoreAxiom(expression); err != nil {
			return false, fmt.Errorf("array store axiom: %w", err)
		}
	case ast.ASTArrayOpSelect:
		if err = t.assertSelectStoreAxioms(expression); err != nil {
			return false, fmt.Errorf("array select-store axiom: %w", err)
		}
	case ast.ASTArrayOpConst, ast.ASTArrayOpExt, ast.ASTArrayOpMap:
	case ast.ASTArrayOpAsArray:
		t.unsupported = true
	case ast.ASTArrayOpDefault:
		if err = t.assertDefaultAxiom(expression); err != nil {
			return false, fmt.Errorf("array default axiom: %w", err)
		}
	default:
		t.unsupported = true
	}
	return handled, nil
}

func (t *SMTArrayTheory) assertStoreAxiom(store ast.ASTHandle) error {
	if t.stores[store] {
		return nil
	}
	t.stores[store] = true
	t.storeTrail = append(t.storeTrail, store)
	state, _ := t.State()
	count, err := state.manager.GetNumArgs(store)
	if err != nil || count < 3 {
		return err
	}
	arguments := make([]ast.ASTHandle, 0, count-1)
	arguments = append(arguments, store)
	for index := uint32(1); index+1 < count; index++ {
		argument, argErr := state.manager.GetArg(store, index)
		if argErr != nil {
			return argErr
		}
		arguments = append(arguments, argument)
	}
	selected, err := state.manager.MkFamilyApp(t.FamilyID(), ast.ASTArrayOpSelect, nil, arguments)
	if err != nil {
		return err
	}
	value, err := state.manager.GetArg(store, count-1)
	if err != nil {
		return err
	}
	literal, err := t.MkEqualityLiteral(selected, value, true)
	if err != nil {
		return err
	}
	if err = state.addTheoryAxiom(t.FamilyID(), []SMTLiteral{literal}); err != nil {
		return fmt.Errorf("array store unit axiom: %w", err)
	}
	t.statistics.axiom1++
	return nil
}

func (t *SMTArrayTheory) assertSelectStoreAxioms(selectExpression ast.ASTHandle) error {
	if !t.selects[selectExpression] {
		t.selects[selectExpression] = true
		t.selectTrail = append(t.selectTrail, selectExpression)
	}
	state, _ := t.State()
	array, err := state.manager.GetArg(selectExpression, 0)
	if err != nil {
		return err
	}
	return t.assertSelectAxiom(selectExpression, array)
}

func (t *SMTArrayTheory) assertSelectAxiom(selectExpression, array ast.ASTHandle) error {
	state, err := t.State()
	if err != nil {
		return err
	}
	pair := smtArrayAxiomPair{observer: selectExpression, array: array}
	if t.selectAxioms[pair] {
		return nil
	}
	op, isArray, err := t.arrayOperation(array)
	if err != nil || !isArray {
		return err
	}
	if op != ast.ASTArrayOpConst && op != ast.ASTArrayOpMap && op != ast.ASTArrayOpAsArray && op != ast.ASTArrayOpStore {
		return nil
	}
	t.selectAxioms[pair] = true
	t.selectAxiomTrail = append(t.selectAxiomTrail, pair)
	observedArray, err := state.manager.GetArg(selectExpression, 0)
	if err != nil {
		return err
	}
	guard := make([]SMTLiteral, 0, 1)
	if observedArray != array {
		equal, equalityErr := t.MkEqualityLiteral(observedArray, array, true)
		if equalityErr != nil {
			return equalityErr
		}
		guard = append(guard, equal.Negated())
	}
	selectCount, err := state.manager.GetNumArgs(selectExpression)
	if err != nil || selectCount < 2 {
		return err
	}
	if op == ast.ASTArrayOpConst {
		value, argErr := state.manager.GetArg(array, 0)
		if argErr != nil {
			return argErr
		}
		literal, equalityErr := t.MkEqualityLiteral(selectExpression, value, true)
		if equalityErr != nil {
			return equalityErr
		}
		clause := append(guard, literal)
		if equalityErr = state.addTheoryAxiom(t.FamilyID(), clause); equalityErr != nil {
			return fmt.Errorf("constant array select axiom: %w", equalityErr)
		}
		t.statistics.selectConst++
		return nil
	}
	if op == ast.ASTArrayOpMap {
		return t.assertSelectMapAxiom(selectExpression, array, guard)
	}
	if op == ast.ASTArrayOpAsArray {
		function, functionErr := state.manager.AsArrayFunction(array)
		if functionErr != nil {
			return functionErr
		}
		count, countErr := state.manager.GetNumArgs(selectExpression)
		if countErr != nil || count < 2 {
			return countErr
		}
		arguments := make([]ast.ASTHandle, 0, count-1)
		for index := uint32(1); index < count; index++ {
			argument, argErr := state.manager.GetArg(selectExpression, index)
			if argErr != nil {
				return argErr
			}
			arguments = append(arguments, argument)
		}
		value, appErr := state.manager.MkApp(function, arguments)
		if appErr != nil {
			return appErr
		}
		literal, equalityErr := t.MkEqualityLiteral(selectExpression, value, true)
		if equalityErr != nil {
			return equalityErr
		}
		clause := append(guard, literal)
		if equalityErr = state.addTheoryAxiom(t.FamilyID(), clause); equalityErr != nil {
			return fmt.Errorf("as-array select axiom: %w", equalityErr)
		}
		t.statistics.selectAsArray++
		return nil
	}
	if op != ast.ASTArrayOpStore {
		return nil
	}
	storeCount, err := state.manager.GetNumArgs(array)
	if err != nil || storeCount != selectCount+1 {
		return err
	}
	allEqual := true
	added := false
	for index := uint32(1); index < selectCount; index++ {
		storedIndex, argErr := state.manager.GetArg(array, index)
		if argErr != nil {
			return argErr
		}
		selectedIndex, argErr := state.manager.GetArg(selectExpression, index)
		if argErr != nil {
			return argErr
		}
		if storedIndex != selectedIndex {
			allEqual = false
			break
		}
	}
	if allEqual {
		return nil
	}
	base, err := state.manager.GetArg(array, 0)
	if err != nil {
		return err
	}
	fallbackArguments := make([]ast.ASTHandle, 0, selectCount)
	fallbackArguments = append(fallbackArguments, base)
	for index := uint32(1); index < selectCount; index++ {
		argument, argErr := state.manager.GetArg(selectExpression, index)
		if argErr != nil {
			return argErr
		}
		fallbackArguments = append(fallbackArguments, argument)
	}
	fallback, err := state.manager.MkFamilyApp(t.FamilyID(), ast.ASTArrayOpSelect, nil, fallbackArguments)
	if err != nil {
		return err
	}
	consequence, err := t.MkEqualityLiteral(selectExpression, fallback, true)
	if err != nil {
		return err
	}
	for index := uint32(1); index < selectCount; index++ {
		storedIndex, argErr := state.manager.GetArg(array, index)
		if argErr != nil {
			return argErr
		}
		selectedIndex, argErr := state.manager.GetArg(selectExpression, index)
		if argErr != nil {
			return argErr
		}
		if storedIndex == selectedIndex {
			continue
		}
		antecedent, equalityErr := t.MkEqualityLiteral(storedIndex, selectedIndex, true)
		if equalityErr != nil {
			return equalityErr
		}
		clause := append(append(guard[:0:0], guard...), antecedent, consequence)
		if err = state.addTheoryAxiom(t.FamilyID(), clause); err != nil {
			return fmt.Errorf("array conditional store axiom: %w", err)
		}
		added = true
	}
	if added {
		t.statistics.axiom2++
	}
	return nil
}

func (t *SMTArrayTheory) assertSelectMapAxiom(selectExpression, arrayMap ast.ASTHandle, guard []SMTLiteral) error {
	state, _ := t.State()
	mapDeclaration, err := state.manager.GetDecl(arrayMap)
	if err != nil {
		return err
	}
	function, err := state.manager.ArrayMapFunction(mapDeclaration)
	if err != nil {
		return err
	}
	mapCount, err := state.manager.GetNumArgs(arrayMap)
	if err != nil {
		return err
	}
	selectCount, err := state.manager.GetNumArgs(selectExpression)
	if err != nil || selectCount < 2 {
		return err
	}
	indices := make([]ast.ASTHandle, 0, selectCount-1)
	for index := uint32(1); index < selectCount; index++ {
		argument, argErr := state.manager.GetArg(selectExpression, index)
		if argErr != nil {
			return argErr
		}
		indices = append(indices, argument)
	}
	values := make([]ast.ASTHandle, 0, mapCount)
	for index := uint32(0); index < mapCount; index++ {
		input, argErr := state.manager.GetArg(arrayMap, index)
		if argErr != nil {
			return argErr
		}
		arguments := make([]ast.ASTHandle, 0, selectCount)
		arguments = append(arguments, input)
		arguments = append(arguments, indices...)
		value, selectErr := state.manager.MkFamilyApp(t.FamilyID(), ast.ASTArrayOpSelect, nil, arguments)
		if selectErr != nil {
			return selectErr
		}
		values = append(values, value)
	}
	mapped, err := state.manager.MkApp(function, values)
	if err != nil {
		return err
	}
	literal, err := t.MkEqualityLiteral(selectExpression, mapped, true)
	if err != nil {
		return err
	}
	clause := append(guard, literal)
	if err = state.addTheoryAxiom(t.FamilyID(), clause); err != nil {
		return fmt.Errorf("array map select axiom: %w", err)
	}
	t.statistics.mapAxiom++
	return nil
}

func (t *SMTArrayTheory) assertDefaultAxiom(defaultExpression ast.ASTHandle) error {
	if !t.defaults[defaultExpression] {
		t.defaults[defaultExpression] = true
		t.defaultTrail = append(t.defaultTrail, defaultExpression)
	}
	state, _ := t.State()
	array, err := state.manager.GetArg(defaultExpression, 0)
	if err != nil {
		return err
	}
	return t.assertDefaultAxiomForArray(defaultExpression, array)
}

func (t *SMTArrayTheory) assertDefaultAxiomForArray(defaultExpression, array ast.ASTHandle) error {
	state, err := t.State()
	if err != nil {
		return err
	}
	pair := smtArrayAxiomPair{observer: defaultExpression, array: array}
	if t.defaultAxioms[pair] {
		return nil
	}
	op, isArray, err := t.arrayOperation(array)
	if err != nil || !isArray {
		return err
	}
	if op != ast.ASTArrayOpConst && op != ast.ASTArrayOpMap && op != ast.ASTArrayOpStore && op != ast.ASTArrayOpAsArray {
		return nil
	}
	t.defaultAxioms[pair] = true
	t.defaultAxiomTrail = append(t.defaultAxiomTrail, pair)
	observedArray, err := state.manager.GetArg(defaultExpression, 0)
	if err != nil {
		return err
	}
	guard := make([]SMTLiteral, 0, 1)
	if observedArray != array {
		equal, equalityErr := t.MkEqualityLiteral(observedArray, array, true)
		if equalityErr != nil {
			return equalityErr
		}
		guard = append(guard, equal.Negated())
	}
	var value ast.ASTHandle
	switch op {
	case ast.ASTArrayOpConst:
		value, err = state.manager.GetArg(array, 0)
	case ast.ASTArrayOpMap:
		declaration, declErr := state.manager.GetDecl(array)
		if declErr != nil {
			return declErr
		}
		function, functionErr := state.manager.ArrayMapFunction(declaration)
		if functionErr != nil {
			return functionErr
		}
		count, countErr := state.manager.GetNumArgs(array)
		if countErr != nil {
			return countErr
		}
		arguments := make([]ast.ASTHandle, 0, count)
		for index := uint32(0); index < count; index++ {
			input, argErr := state.manager.GetArg(array, index)
			if argErr != nil {
				return argErr
			}
			inputDefault, defaultErr := state.manager.MkFamilyApp(t.FamilyID(), ast.ASTArrayOpDefault, nil, []ast.ASTHandle{input})
			if defaultErr != nil {
				return defaultErr
			}
			arguments = append(arguments, inputDefault)
		}
		value, err = state.manager.MkApp(function, arguments)
	case ast.ASTArrayOpStore:
		large, largeErr := t.arrayHasLargeDomain(array)
		if largeErr != nil {
			return largeErr
		}
		base, argErr := state.manager.GetArg(array, 0)
		if argErr != nil {
			return argErr
		}
		baseDefault, defaultErr := state.manager.MkFamilyApp(t.FamilyID(), ast.ASTArrayOpDefault, nil, []ast.ASTHandle{base})
		if defaultErr != nil {
			return defaultErr
		}
		if large {
			value = baseDefault
			break
		}
		count, countErr := state.manager.GetNumArgs(array)
		if countErr != nil || count < 3 {
			return countErr
		}
		conditions := make([]ast.ASTHandle, 0, count-2)
		for index := uint32(1); index+1 < count; index++ {
			location, locationErr := state.manager.GetArg(array, index)
			if locationErr != nil {
				return locationErr
			}
			sort, sortErr := state.manager.GetSort(location)
			if sortErr != nil {
				return sortErr
			}
			epsilon, epsilonErr := t.arrayEpsilon(sort)
			if epsilonErr != nil {
				return epsilonErr
			}
			equal, equalErr := state.manager.MkEq(epsilon, location)
			if equalErr != nil {
				return equalErr
			}
			conditions = append(conditions, equal)
		}
		condition, conditionErr := state.manager.MkAndArray(conditions)
		if conditionErr != nil {
			return conditionErr
		}
		storedValue, valueErr := state.manager.GetArg(array, count-1)
		if valueErr != nil {
			return valueErr
		}
		value, err = state.manager.MkIte(condition, storedValue, baseDefault)
	case ast.ASTArrayOpAsArray:
		function, functionErr := state.manager.AsArrayFunction(array)
		if functionErr != nil {
			return functionErr
		}
		arity, arityErr := state.manager.GetArity(function)
		if arityErr != nil {
			return arityErr
		}
		arguments := make([]ast.ASTHandle, 0, arity)
		for index := uint32(0); index < arity; index++ {
			sort, domainErr := state.manager.GetDomain(function, index)
			if domainErr != nil {
				return domainErr
			}
			epsilon, epsilonErr := t.arrayEpsilon(sort)
			if epsilonErr != nil {
				return epsilonErr
			}
			arguments = append(arguments, epsilon)
		}
		value, err = state.manager.MkApp(function, arguments)
	default:
		return nil
	}
	if err != nil {
		return err
	}
	literal, err := t.MkEqualityLiteral(defaultExpression, value, true)
	if err != nil {
		return err
	}
	clause := append(guard, literal)
	if err = state.addTheoryAxiom(t.FamilyID(), clause); err != nil {
		return fmt.Errorf("array default value axiom: %w", err)
	}
	switch op {
	case ast.ASTArrayOpConst:
		t.statistics.defaultConst++
	case ast.ASTArrayOpMap:
		t.statistics.defaultMap++
	case ast.ASTArrayOpStore:
		t.statistics.defaultStore++
	case ast.ASTArrayOpAsArray:
		t.statistics.defaultAsArray++
	}
	return nil
}

func (t *SMTArrayTheory) arrayEpsilon(sort ast.ASTHandle) (ast.ASTHandle, error) {
	if epsilon := t.epsilons[sort]; !epsilon.IsNull() {
		return epsilon, nil
	}
	state, _ := t.State()
	epsilon, err := state.manager.MkFreshConst("epsilon", sort)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	t.epsilons[sort] = epsilon
	t.epsilonTrail = append(t.epsilonTrail, sort)
	return epsilon, nil
}

func (t *SMTArrayTheory) arrayHasLargeDomain(array ast.ASTHandle) (bool, error) {
	state, _ := t.State()
	sort, err := state.manager.GetSort(array)
	if err != nil {
		return false, err
	}
	cardinality := uint64(1)
	for index := uint32(0); ; index++ {
		declaration, declErr := state.manager.MkArrayExtDecl(sort, index)
		if errors.Is(declErr, ast.ErrASTDomain) {
			break
		}
		if declErr != nil {
			return false, declErr
		}
		domain, rangeErr := state.manager.GetRange(declaration)
		if rangeErr != nil {
			return false, rangeErr
		}
		size, sizeErr := state.manager.GetSortSize(domain)
		if sizeErr != nil {
			return false, sizeErr
		}
		if size.IsInfinite() || size.IsVeryBig() {
			return true, nil
		}
		value, valueErr := size.Size()
		if valueErr != nil {
			return false, valueErr
		}
		if value == 0 {
			return false, nil
		}
		if cardinality >= 1<<20 || value >= (1<<20+cardinality-1)/cardinality {
			return true, nil
		}
		cardinality *= value
	}
	return cardinality >= 1<<20, nil
}

func (t *SMTArrayTheory) InternalizeEquality(expression ast.ASTHandle, variable int32) error {
	state, err := t.State()
	if err != nil {
		return err
	}
	left, err := state.manager.GetArg(expression, 0)
	if err != nil {
		return err
	}
	right, err := state.manager.GetArg(expression, 1)
	if err != nil {
		return err
	}
	sort, err := state.manager.GetSort(left)
	if err != nil {
		return err
	}
	leftArguments := []ast.ASTHandle{left}
	rightArguments := []ast.ASTHandle{right}
	for index := uint32(0); ; index++ {
		declaration, declErr := state.manager.MkArrayExtDecl(sort, index)
		if errors.Is(declErr, ast.ErrASTDomain) {
			break
		}
		if declErr != nil {
			return declErr
		}
		witness, appErr := state.manager.MkApp(declaration, []ast.ASTHandle{left, right})
		if appErr != nil {
			return appErr
		}
		leftArguments = append(leftArguments, witness)
		rightArguments = append(rightArguments, witness)
	}
	if len(leftArguments) == 1 {
		return ast.ErrASTDomain
	}
	leftSelect, err := state.manager.MkFamilyApp(t.FamilyID(), ast.ASTArrayOpSelect, nil, leftArguments)
	if err != nil {
		return err
	}
	rightSelect, err := state.manager.MkFamilyApp(t.FamilyID(), ast.ASTArrayOpSelect, nil, rightArguments)
	if err != nil {
		return err
	}
	selectEquality, err := t.MkEqualityLiteral(leftSelect, rightSelect, true)
	if err != nil {
		return err
	}
	arrayEquality := NewSMTLiteral(variable, false)
	if err = state.addTheoryAxiom(t.FamilyID(), []SMTLiteral{arrayEquality, selectEquality.Negated()}); err != nil {
		return err
	}
	t.statistics.extensionality++
	return nil
}
func (t *SMTArrayTheory) NewEquality(left, right SMTTheoryVar) error {
	if _, err := t.Enode(left); err != nil {
		return err
	}
	if _, err := t.Enode(right); err != nil {
		return err
	}
	return t.instantiateEquivalentArrayAxioms()
}
func (*SMTArrayTheory) NewDisequality(SMTTheoryVar, SMTTheoryVar) error { return nil }
func (*SMTArrayTheory) Assign(int32, bool) error                        { return nil }

func (t *SMTArrayTheory) instantiateEquivalentArrayAxioms() error {
	state, err := t.State()
	if err != nil {
		return err
	}
	for _, observer := range t.selectTrail {
		array, argErr := state.manager.GetArg(observer, 0)
		if argErr != nil {
			return argErr
		}
		arrayNode := state.context.enodes[array]
		if arrayNode == nil {
			continue
		}
		for _, candidate := range t.variables {
			if candidate != nil && candidate.root == arrayNode.root {
				if err = t.assertSelectAxiom(observer, candidate.owner); err != nil {
					return err
				}
			}
		}
	}
	for _, observer := range t.defaultTrail {
		array, argErr := state.manager.GetArg(observer, 0)
		if argErr != nil {
			return argErr
		}
		arrayNode := state.context.enodes[array]
		if arrayNode == nil {
			continue
		}
		for _, candidate := range t.variables {
			if candidate != nil && candidate.root == arrayNode.root {
				if err = t.assertDefaultAxiomForArray(observer, candidate.owner); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (t *SMTArrayTheory) ApplySortConstraint(node *SMTEnode, sort ast.ASTHandle) error {
	state, err := t.State()
	if err != nil {
		return err
	}
	family, err := state.manager.GetFamilyID(sort)
	if err != nil || family != t.FamilyID() || t.IsAttachedToVariable(node) {
		return err
	}
	_, err = t.CreateVariable(node)
	return err
}

func (t *SMTArrayTheory) PushScope() error {
	if err := t.SMTTheoryBase.PushScope(); err != nil {
		return err
	}
	t.scopes = append(t.scopes, smtArrayScope{
		stores:        len(t.storeTrail),
		selects:       len(t.selectTrail),
		defaults:      len(t.defaultTrail),
		epsilons:      len(t.epsilonTrail),
		selectAxioms:  len(t.selectAxiomTrail),
		defaultAxioms: len(t.defaultAxiomTrail),
		unsupported:   t.unsupported,
	})
	return nil
}

func (t *SMTArrayTheory) PopScopes(count uint32) error {
	if count == 0 || count > uint32(len(t.scopes)) {
		return ast.ErrASTDomain
	}
	level := len(t.scopes) - int(count)
	scope := t.scopes[level]
	for _, expression := range t.storeTrail[scope.stores:] {
		delete(t.stores, expression)
	}
	for _, expression := range t.selectTrail[scope.selects:] {
		delete(t.selects, expression)
	}
	for _, expression := range t.defaultTrail[scope.defaults:] {
		delete(t.defaults, expression)
	}
	for _, sort := range t.epsilonTrail[scope.epsilons:] {
		delete(t.epsilons, sort)
	}
	for _, pair := range t.selectAxiomTrail[scope.selectAxioms:] {
		delete(t.selectAxioms, pair)
	}
	for _, pair := range t.defaultAxiomTrail[scope.defaultAxioms:] {
		delete(t.defaultAxioms, pair)
	}
	t.storeTrail = t.storeTrail[:scope.stores]
	t.selectTrail = t.selectTrail[:scope.selects]
	t.defaultTrail = t.defaultTrail[:scope.defaults]
	t.epsilonTrail = t.epsilonTrail[:scope.epsilons]
	t.selectAxiomTrail = t.selectAxiomTrail[:scope.selectAxioms]
	t.defaultAxiomTrail = t.defaultAxiomTrail[:scope.defaultAxioms]
	t.unsupported = scope.unsupported
	t.scopes = t.scopes[:level]
	return t.SMTTheoryBase.PopScopes(count)
}

func (t *SMTArrayTheory) FinalCheck() (SMTFinalCheckStatus, error) {
	if t.unsupported {
		return SMTFinalCheckGiveUp, nil
	}
	return SMTFinalCheckDone, nil
}

func (*SMTArrayTheory) Name() string { return "array" }

func (t *SMTArrayTheory) CollectStatistics(target *statistics.Statistics) error {
	if t == nil || target == nil {
		return ast.ErrASTHandle
	}
	target.UpdateUint64("array ax1", t.statistics.axiom1)
	target.UpdateUint64("array ax2", t.statistics.axiom2)
	target.UpdateUint64("array exp ax2", 0)
	target.UpdateUint64("array ext ax", t.statistics.extensionality)
	target.UpdateUint64("array splits", t.statistics.eqSplits)
	target.UpdateUint64("array map ax", t.statistics.mapAxiom)
	target.UpdateUint64("array def const", t.statistics.defaultConst)
	target.UpdateUint64("array sel const", t.statistics.selectConst)
	target.UpdateUint64("array def store", t.statistics.defaultStore)
	target.UpdateUint64("array def as-array", t.statistics.defaultAsArray)
	target.UpdateUint64("array sel as-array", t.statistics.selectAsArray)
	return nil
}

func (t *SMTArrayTheory) ResetStatistics() {
	if t != nil {
		t.statistics = smtArrayStatistics{}
	}
}

func (t *SMTArrayTheory) InitializeModel(generator *SMTModelGenerator) error {
	state, err := t.State()
	if err != nil || generator == nil || generator.manager != state.manager || generator.model == nil {
		if err != nil {
			return err
		}
		return ast.ErrASTDomain
	}
	factory, err := protomodel.NewArrayFactory(generator.model)
	if err != nil {
		return err
	}
	return generator.RegisterFactory(factory)
}

type smtArrayConstructorValueProcedure struct {
	declaration  ast.ASTHandle
	dependencies []SMTModelValueDependency
}

func (p *smtArrayConstructorValueProcedure) Dependencies() []SMTModelValueDependency {
	if p == nil {
		return nil
	}
	return append([]SMTModelValueDependency(nil), p.dependencies...)
}

func (p *smtArrayConstructorValueProcedure) MakeValue(generator *SMTModelGenerator, values []ast.ASTHandle) (ast.ASTHandle, error) {
	if p == nil || generator == nil || p.declaration.IsNull() || len(values) != len(p.dependencies) {
		return ast.ASTHandle{}, ast.ErrASTDomain
	}
	return generator.manager.MkApp(p.declaration, values)
}

func (*smtArrayConstructorValueProcedure) IsFresh() bool { return false }

func (t *SMTArrayTheory) ModelValue(node *SMTEnode, generator *SMTModelGenerator) (SMTModelValueProcedure, error) {
	if node == nil || generator == nil || !t.IsAttachedToVariable(node) || generator.model == nil {
		return nil, ast.ErrASTDomain
	}
	sort, err := generator.manager.GetSort(node.owner)
	if err != nil {
		return nil, err
	}
	if op, array, operationErr := t.arrayOperation(node.owner); operationErr != nil {
		return nil, operationErr
	} else if array && (op == ast.ASTArrayOpConst || op == ast.ASTArrayOpStore || op == ast.ASTArrayOpMap) {
		declaration, declarationErr := generator.manager.GetDecl(node.owner)
		if declarationErr != nil {
			return nil, declarationErr
		}
		arguments := node.Arguments()
		dependencies := make([]SMTModelValueDependency, 0, len(arguments))
		for _, argument := range arguments {
			dependency, dependencyErr := NewSMTEnodeValueDependency(argument)
			if dependencyErr != nil {
				return nil, dependencyErr
			}
			dependencies = append(dependencies, dependency)
		}
		return &smtArrayConstructorValueProcedure{declaration: declaration, dependencies: dependencies}, nil
	}
	value, err := generator.model.FreshValue(sort)
	if err != nil {
		return nil, err
	}
	if value.IsNull() {
		value, err = generator.SomeValue(sort)
		if err != nil {
			return nil, err
		}
	}
	return NewSMTExpressionValueProcedure(value)
}
