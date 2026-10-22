// Source: retained store/select axiom fragment of src/smt/theory_array_base.h,
// src/smt/theory_array_base.cpp, src/smt/theory_array.h, and
// src/smt/theory_array.cpp.
package smtcore

import (
	"errors"
	"fmt"

	"github.com/glycerine/g3/pkg/ast"
)

type smtArrayScope struct {
	stores, selects int
	unsupported     bool
}

// SMTArrayTheory internalizes core read-over-write, constant-array, and
// extensionality axioms. Maps, defaults, and model construction are added by
// later retained slices.
type SMTArrayTheory struct {
	*SMTTheoryBase
	stores, selects         map[ast.ASTHandle]bool
	storeTrail, selectTrail []ast.ASTHandle
	scopes                  []smtArrayScope
	unsupported             bool
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
		if op == ast.ASTArrayOpSelect {
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
	case ast.ASTArrayOpConst, ast.ASTArrayOpExt:
		// Const arrays need no axiom until selected; that rule is added with
		// the retained constant-array fragment.
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
	return nil
}

func (t *SMTArrayTheory) assertSelectStoreAxioms(selectExpression ast.ASTHandle) error {
	if t.selects[selectExpression] {
		return nil
	}
	t.selects[selectExpression] = true
	t.selectTrail = append(t.selectTrail, selectExpression)
	state, _ := t.State()
	array, err := state.manager.GetArg(selectExpression, 0)
	if err != nil {
		return err
	}
	op, isArray, err := t.arrayOperation(array)
	if err != nil || !isArray {
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
		if equalityErr = state.addTheoryAxiom(t.FamilyID(), []SMTLiteral{literal}); equalityErr != nil {
			return fmt.Errorf("constant array select axiom: %w", equalityErr)
		}
		return nil
	}
	if op != ast.ASTArrayOpStore {
		return nil
	}
	selectCount, err := state.manager.GetNumArgs(selectExpression)
	if err != nil {
		return err
	}
	storeCount, err := state.manager.GetNumArgs(array)
	if err != nil || storeCount != selectCount+1 {
		return err
	}
	allEqual := true
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
		if err = state.addTheoryAxiom(t.FamilyID(), []SMTLiteral{antecedent, consequence}); err != nil {
			return fmt.Errorf("array conditional store axiom: %w", err)
		}
	}
	return nil
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
	return state.addTheoryAxiom(t.FamilyID(), []SMTLiteral{arrayEquality, selectEquality.Negated()})
}
func (*SMTArrayTheory) NewEquality(SMTTheoryVar, SMTTheoryVar) error    { return nil }
func (*SMTArrayTheory) NewDisequality(SMTTheoryVar, SMTTheoryVar) error { return nil }
func (*SMTArrayTheory) Assign(int32, bool) error                        { return nil }

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
	t.scopes = append(t.scopes, smtArrayScope{len(t.storeTrail), len(t.selectTrail), t.unsupported})
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
	t.storeTrail = t.storeTrail[:scope.stores]
	t.selectTrail = t.selectTrail[:scope.selects]
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
