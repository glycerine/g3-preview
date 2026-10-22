// Source checkpoint: src/interp/iz3mgr.cpp; src/interp/iz3mgr.h.
package interp

import (
	"github.com/glycerine/g3/pkg/ast"
)

// Operator is the interpolation wrapper's family-independent AST vocabulary.
type Operator uint8

const (
	OperatorTrue Operator = iota
	OperatorFalse
	OperatorAnd
	OperatorOr
	OperatorNot
	OperatorIff
	OperatorIte
	OperatorEqual
	OperatorImplies
	OperatorDistinct
	OperatorXor
	OperatorOEq
	OperatorInterp
	OperatorLE
	OperatorGE
	OperatorLT
	OperatorGT
	OperatorPlus
	OperatorSub
	OperatorUnaryMinus
	OperatorTimes
	OperatorDiv
	OperatorIDiv
	OperatorRem
	OperatorMod
	OperatorPower
	OperatorToReal
	OperatorToInt
	OperatorIsInt
	OperatorSelect
	OperatorStore
	OperatorConstArray
	OperatorArrayDefault
	OperatorArrayMap
	OperatorSetUnion
	OperatorSetIntersect
	OperatorSetDifference
	OperatorSetComplement
	OperatorSetSubset
	OperatorAsArray
	OperatorNumeral
	OperatorForall
	OperatorExists
	OperatorVariable
	OperatorUninterpreted
	OperatorOther
)

// Manager is the retained typed convenience layer over ASTManager. It borrows
// the AST manager; callers pin handles that outlive an immediate operation.
type Manager struct {
	self       *Manager
	ast        *ast.ASTManager
	arithmetic ast.FamilyID
	array      ast.FamilyID
	closed     bool
}

func NewManager(m *ast.ASTManager) (*Manager, error) {
	if m == nil {
		return nil, ast.ErrASTHandle
	}
	if _, err := m.ProofsEnabled(); err != nil {
		return nil, err
	}
	arithmetic, err := m.ArithmeticFamilyID()
	if err != nil {
		return nil, err
	}
	array, err := m.ArrayFamilyID()
	if err != nil {
		return nil, err
	}
	r := &Manager{ast: m, arithmetic: arithmetic, array: array}
	r.self = r
	return r, nil
}

func (m *Manager) open() error {
	if m == nil || m.self != m || m.closed || m.ast == nil {
		return ast.ErrASTHandle
	}
	_, err := m.ast.ProofsEnabled()
	return err
}

func (m *Manager) Close() error {
	if m == nil || m.self != m {
		return ast.ErrASTHandle
	}
	m.closed = true
	return nil
}

func (m *Manager) ASTManager() (*ast.ASTManager, error) {
	if err := m.open(); err != nil {
		return nil, err
	}
	return m.ast, nil
}

var basicOperators = map[int32]Operator{
	ast.ASTOpTrue: OperatorTrue, ast.ASTOpFalse: OperatorFalse,
	ast.ASTOpEq: OperatorEqual, ast.ASTOpDistinct: OperatorDistinct,
	ast.ASTOpIte: OperatorIte, ast.ASTOpAnd: OperatorAnd,
	ast.ASTOpOr: OperatorOr, ast.ASTOpIff: OperatorIff,
	ast.ASTOpXor: OperatorXor, ast.ASTOpNot: OperatorNot,
	ast.ASTOpImplies: OperatorImplies, ast.ASTOpOEq: OperatorOEq,
	ast.ASTOpInterp: OperatorInterp,
}

var arithmeticOperators = map[int32]Operator{
	2: OperatorLE, 3: OperatorGE, 4: OperatorLT, 5: OperatorGT,
	6: OperatorPlus, 7: OperatorSub, 8: OperatorUnaryMinus, 9: OperatorTimes,
	10: OperatorDiv, 11: OperatorIDiv, 12: OperatorRem, 13: OperatorMod,
	14: OperatorToReal, 15: OperatorToInt, 16: OperatorIsInt, 18: OperatorPower,
}

var arrayOperators = map[int32]Operator{
	ast.ASTArrayOpStore: OperatorStore, ast.ASTArrayOpSelect: OperatorSelect,
	ast.ASTArrayOpConst: OperatorConstArray, ast.ASTArrayOpDefault: OperatorArrayDefault,
	ast.ASTArrayOpMap: OperatorArrayMap, ast.ASTArrayOpSetUnion: OperatorSetUnion,
	ast.ASTArrayOpSetIntersect: OperatorSetIntersect, ast.ASTArrayOpSetDifference: OperatorSetDifference,
	ast.ASTArrayOpSetComplement: OperatorSetComplement, ast.ASTArrayOpSetSubset: OperatorSetSubset,
	ast.ASTArrayOpAsArray: OperatorAsArray,
}

func (m *Manager) Operator(h ast.ASTHandle) (Operator, error) {
	if err := m.open(); err != nil {
		return OperatorOther, err
	}
	kind, err := m.ast.GetKind(h)
	if err != nil {
		return OperatorOther, err
	}
	switch kind {
	case ast.ASTQuantifier:
		forall, e := m.ast.IsForall(h)
		if e != nil {
			return OperatorOther, e
		}
		if forall {
			return OperatorForall, nil
		}
		return OperatorExists, nil
	case ast.ASTVar:
		return OperatorVariable, nil
	case ast.ASTApp:
		family, e := m.ast.GetFamilyID(h)
		if e != nil {
			return OperatorOther, e
		}
		if family == ast.NullFamilyID {
			return OperatorUninterpreted, nil
		}
		declKind, e := m.ast.GetDeclKind(h)
		if e != nil {
			return OperatorOther, e
		}
		if family == m.ast.BasicFamilyID() {
			if op, found := basicOperators[declKind]; found {
				return op, nil
			}
			return OperatorOther, nil
		}
		if family == m.arithmetic {
			if op, found := arithmeticOperators[declKind]; found {
				return op, nil
			}
			if _, _, numeral, e := m.ast.ArithmeticNumeralText(h); e != nil {
				return OperatorOther, e
			} else if numeral {
				return OperatorNumeral, nil
			}
			return OperatorOther, nil
		}
		if family == m.array {
			if op, found := arrayOperators[declKind]; found {
				return op, nil
			}
		}
	}
	return OperatorOther, nil
}

func (m *Manager) NumArgs(h ast.ASTHandle) (uint32, error) {
	if err := m.open(); err != nil {
		return 0, err
	}
	kind, err := m.ast.GetKind(h)
	if err != nil {
		return 0, err
	}
	switch kind {
	case ast.ASTApp:
		return m.ast.GetNumArgs(h)
	case ast.ASTQuantifier:
		return 1, nil
	case ast.ASTVar:
		return 0, nil
	default:
		return 0, ast.ErrASTDomain
	}
}

func (m *Manager) Arg(h ast.ASTHandle, index uint32) (ast.ASTHandle, error) {
	if err := m.open(); err != nil {
		return ast.ASTHandle{}, err
	}
	kind, err := m.ast.GetKind(h)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if kind == ast.ASTQuantifier {
		if index != 0 {
			return ast.ASTHandle{}, ast.ErrASTDomain
		}
		return m.ast.GetQuantifierBody(h)
	}
	return m.ast.GetArg(h, index)
}

func (m *Manager) Args(h ast.ASTHandle) ([]ast.ASTHandle, error) {
	n, err := m.NumArgs(h)
	if err != nil {
		return nil, err
	}
	result := make([]ast.ASTHandle, n)
	for i := range result {
		result[i], err = m.Arg(h, uint32(i))
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (m *Manager) Symbol(h ast.ASTHandle) (ast.ASTHandle, error) {
	if err := m.open(); err != nil {
		return ast.ASTHandle{}, err
	}
	if kind, err := m.ast.GetKind(h); err != nil {
		return ast.ASTHandle{}, err
	} else if kind != ast.ASTApp {
		return ast.ASTHandle{}, nil
	}
	return m.ast.GetDecl(h)
}

func (m *Manager) Type(h ast.ASTHandle) (ast.ASTHandle, error) {
	if err := m.open(); err != nil {
		return ast.ASTHandle{}, err
	}
	return m.ast.GetSort(h)
}

func (m *Manager) MakeVariable(name string, sort ast.ASTHandle) (ast.ASTHandle, error) {
	if err := m.open(); err != nil {
		return ast.ASTHandle{}, err
	}
	symbol, err := m.ast.Symbols.Intern(name)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	return m.ast.MkConst(symbol, sort)
}

func (m *Manager) MakeDeclaration(declaration ast.ASTHandle, args ...ast.ASTHandle) (ast.ASTHandle, error) {
	if err := m.open(); err != nil {
		return ast.ASTHandle{}, err
	}
	return m.ast.MkApp(declaration, args)
}

func (m *Manager) familyOperator(op Operator) (ast.FamilyID, int32, bool) {
	for kind, candidate := range basicOperators {
		if candidate == op {
			return m.ast.BasicFamilyID(), kind, true
		}
	}
	for kind, candidate := range arithmeticOperators {
		if candidate == op {
			return m.arithmetic, kind, true
		}
	}
	for kind, candidate := range arrayOperators {
		if candidate == op {
			return m.array, kind, true
		}
	}
	return ast.NullFamilyID, 0, false
}

func (m *Manager) Make(op Operator, args ...ast.ASTHandle) (ast.ASTHandle, error) {
	if err := m.open(); err != nil {
		return ast.ASTHandle{}, err
	}
	family, kind, found := m.familyOperator(op)
	if !found {
		return ast.ASTHandle{}, ast.ErrASTDomain
	}
	return m.ast.MkFamilyApp(family, kind, nil, args)
}

func (m *Manager) IsTrue(h ast.ASTHandle) (bool, error) {
	op, err := m.Operator(h)
	return op == OperatorTrue, err
}
func (m *Manager) IsFalse(h ast.ASTHandle) (bool, error) {
	op, err := m.Operator(h)
	return op == OperatorFalse, err
}
func (m *Manager) IsIff(h ast.ASTHandle) (bool, error) {
	op, err := m.Operator(h)
	return op == OperatorIff, err
}
func (m *Manager) IsOr(h ast.ASTHandle) (bool, error) {
	op, err := m.Operator(h)
	return op == OperatorOr, err
}
func (m *Manager) IsNot(h ast.ASTHandle) (bool, error) {
	op, err := m.Operator(h)
	return op == OperatorNot, err
}

func (m *Manager) MkNot(x ast.ASTHandle) (ast.ASTHandle, error) {
	op, err := m.Operator(x)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	switch op {
	case OperatorTrue:
		return m.Make(OperatorFalse)
	case OperatorFalse:
		return m.Make(OperatorTrue)
	case OperatorNot:
		return m.Arg(x, 0)
	default:
		return m.Make(OperatorNot, x)
	}
}

func (m *Manager) MkAnd(x, y ast.ASTHandle) (ast.ASTHandle, error) {
	ox, err := m.Operator(x)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	oy, err := m.Operator(y)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	switch {
	case ox == OperatorTrue:
		return y, nil
	case oy == OperatorTrue:
		return x, nil
	case ox == OperatorFalse:
		return x, nil
	case oy == OperatorFalse, x == y:
		return y, nil
	default:
		return m.Make(OperatorAnd, x, y)
	}
}

func (m *Manager) MkOr(x, y ast.ASTHandle) (ast.ASTHandle, error) {
	ox, err := m.Operator(x)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	oy, err := m.Operator(y)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	switch {
	case ox == OperatorFalse:
		return y, nil
	case oy == OperatorFalse:
		return x, nil
	case ox == OperatorTrue:
		return x, nil
	case oy == OperatorTrue, x == y:
		return y, nil
	default:
		return m.Make(OperatorOr, x, y)
	}
}

func (m *Manager) MkImplies(x, y ast.ASTHandle) (ast.ASTHandle, error) {
	ox, err := m.Operator(x)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	oy, err := m.Operator(y)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	switch {
	case ox == OperatorTrue:
		return y, nil
	case oy == OperatorFalse:
		return m.MkNot(x)
	case ox == OperatorFalse, x == y:
		return m.Make(OperatorTrue)
	case oy == OperatorTrue:
		return y, nil
	default:
		return m.Make(OperatorImplies, x, y)
	}
}

func (m *Manager) MkAndSlice(values []ast.ASTHandle) (ast.ASTHandle, error) {
	filtered := make([]ast.ASTHandle, 0, len(values))
	for _, value := range values {
		op, err := m.Operator(value)
		if err != nil {
			return ast.ASTHandle{}, err
		}
		if op == OperatorFalse {
			return m.Make(OperatorFalse)
		}
		if op != OperatorTrue {
			filtered = append(filtered, value)
		}
	}
	switch len(filtered) {
	case 0:
		return m.Make(OperatorTrue)
	case 1:
		return filtered[0], nil
	default:
		return m.Make(OperatorAnd, filtered...)
	}
}

func (m *Manager) MkOrSlice(values []ast.ASTHandle) (ast.ASTHandle, error) {
	result, err := m.Make(OperatorFalse)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	for _, value := range values {
		result, err = m.MkOr(result, value)
		if err != nil {
			return ast.ASTHandle{}, err
		}
	}
	return result, nil
}

func (m *Manager) MkEqual(x, y ast.ASTHandle) (ast.ASTHandle, error) {
	if x == y {
		return m.Make(OperatorTrue)
	}
	ox, err := m.Operator(x)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	oy, err := m.Operator(y)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	switch {
	case ox == OperatorTrue:
		return y, nil
	case oy == OperatorTrue:
		return x, nil
	case ox == OperatorFalse:
		return m.MkNot(y)
	case oy == OperatorFalse:
		return m.MkNot(x)
	default:
		return m.Make(OperatorEqual, x, y)
	}
}

func (m *Manager) MkIte(condition, thenValue, elseValue ast.ASTHandle) (ast.ASTHandle, error) {
	oc, err := m.Operator(condition)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if oc == OperatorTrue {
		return thenValue, nil
	}
	if oc == OperatorFalse {
		return elseValue, nil
	}
	if thenValue == elseValue {
		return thenValue, nil
	}
	ot, err := m.Operator(thenValue)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	oe, err := m.Operator(elseValue)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if ot == OperatorTrue && oe == OperatorFalse {
		return condition, nil
	}
	if oe == OperatorTrue && ot == OperatorFalse {
		return m.MkNot(condition)
	}
	return m.Make(OperatorIte, condition, thenValue, elseValue)
}
