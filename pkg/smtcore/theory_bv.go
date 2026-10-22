// Source: src/smt/theory_bv.h and src/smt/theory_bv.cpp bit-vector
// variables, numeral bits, bit2bool atoms, and equality expansion.
package smtcore

import (
	"math/big"

	"github.com/glycerine/g3/pkg/ast"
)

const (
	smtBVOpNumeral  int32 = 0
	smtBVOpBit2Bool int32 = 55
)

type smtBVScope struct {
	bits        int
	unsupported bool
}

// SMTBitVectorTheory retains the source theory's little-endian Boolean-bit
// representation. Operations are added incrementally; any reached operation
// without a bit definition makes final-check report incompleteness.
type SMTBitVectorTheory struct {
	*SMTTheoryBase
	bits        map[ast.ASTHandle][]ast.ASTHandle
	bitTrail    []ast.ASTHandle
	scopes      []smtBVScope
	unsupported bool
}

func NewSMTBitVectorTheory(family ast.FamilyID) (*SMTBitVectorTheory, error) {
	base, err := NewSMTTheoryBase(family)
	if err != nil {
		return nil, err
	}
	return &SMTBitVectorTheory{SMTTheoryBase: base, bits: make(map[ast.ASTHandle][]ast.ASTHandle)}, nil
}

func (t *SMTBitVectorTheory) operation(expression ast.ASTHandle) (int32, bool, error) {
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

func (t *SMTBitVectorTheory) symbolicBits(expression ast.ASTHandle, width uint32) ([]ast.ASTHandle, error) {
	state, _ := t.State()
	result := make([]ast.ASTHandle, width)
	for i := range result {
		bit, err := state.manager.MkBitVectorBit2Bool(uint32(i), expression)
		if err != nil {
			return nil, err
		}
		result[i] = bit
	}
	return result, nil
}

func (t *SMTBitVectorTheory) numeralBits(expression ast.ASTHandle, width uint32) ([]ast.ASTHandle, error) {
	state, _ := t.State()
	text, matched, err := state.manager.BitVectorNumeralText(expression)
	if err != nil || !matched {
		if err != nil {
			return nil, err
		}
		return nil, ast.ErrASTDomain
	}
	value, ok := new(big.Int).SetString(text, 10)
	if !ok || value.Sign() < 0 {
		return nil, ast.ErrASTDomain
	}
	result := make([]ast.ASTHandle, width)
	for i := range result {
		result[i], err = state.manager.MkBoolVal(value.Bit(i) != 0)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (t *SMTBitVectorTheory) bitsFor(expression ast.ASTHandle) ([]ast.ASTHandle, error) {
	if bits, found := t.bits[expression]; found {
		return bits, nil
	}
	state, err := t.State()
	if err != nil {
		return nil, err
	}
	width, err := state.manager.BitVectorSize(expression)
	if err != nil {
		return nil, err
	}
	op, own, err := t.operation(expression)
	if err != nil {
		return nil, err
	}
	var bits []ast.ASTHandle
	if own && op == smtBVOpNumeral {
		bits, err = t.numeralBits(expression, width)
	} else {
		bits, err = t.symbolicBits(expression, width)
		if own {
			t.unsupported = true
		}
	}
	if err != nil {
		return nil, err
	}
	// Publish before internalizing bit2bool atoms, whose owner lookup comes
	// back through this method.
	t.bits[expression] = bits
	t.bitTrail = append(t.bitTrail, expression)
	return bits, nil
}

func (t *SMTBitVectorTheory) bit2BoolIndex(expression ast.ASTHandle) (uint32, error) {
	state, _ := t.State()
	declaration, err := state.manager.GetDecl(expression)
	if err != nil {
		return 0, err
	}
	parameters, err := state.manager.GetDeclParametersCopy(declaration)
	if err != nil {
		return 0, err
	}
	defer func() {
		for i := range parameters {
			_ = state.manager.DelParameter(&parameters[i])
		}
	}()
	if len(parameters) != 1 {
		return 0, ast.ErrASTDomain
	}
	value, err := state.manager.ParameterInt(parameters[0])
	if err != nil || value < 0 {
		if err != nil {
			return 0, err
		}
		return 0, ast.ErrASTDomain
	}
	return uint32(value), nil
}

func (t *SMTBitVectorTheory) InternalizeAtom(expression ast.ASTHandle, _ bool) (bool, error) {
	op, own, err := t.operation(expression)
	if err != nil || !own {
		return false, err
	}
	if op != smtBVOpBit2Bool {
		t.unsupported = true
		return false, nil
	}
	state, _ := t.State()
	owner, err := state.manager.GetArg(expression, 0)
	if err != nil {
		return false, err
	}
	bits, err := t.bitsFor(owner)
	if err != nil {
		return false, err
	}
	index, err := t.bit2BoolIndex(expression)
	if err != nil || uint64(index) >= uint64(len(bits)) {
		if err != nil {
			return false, err
		}
		return false, ast.ErrASTDomain
	}
	definition := bits[index]
	if definition != expression {
		if err = state.Internalize(definition, true); err != nil {
			return false, err
		}
	}
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
	if definition != expression {
		defined, err := state.Literal(definition)
		if err != nil {
			return false, err
		}
		literal := NewSMTLiteral(variable, false)
		if err = state.addTheoryAxiom(t.FamilyID(), []SMTLiteral{literal.Negated(), defined}); err != nil {
			return false, err
		}
		if err = state.addTheoryAxiom(t.FamilyID(), []SMTLiteral{literal, defined.Negated()}); err != nil {
			return false, err
		}
	}
	return true, nil
}

func (t *SMTBitVectorTheory) InternalizeTerm(expression ast.ASTHandle) (bool, error) {
	state, err := t.State()
	if err != nil {
		return false, err
	}
	if _, own, err := t.operation(expression); err != nil || !own {
		return false, err
	}
	count, err := state.manager.GetNumArgs(expression)
	if err != nil {
		return false, err
	}
	for i := uint32(0); i < count; i++ {
		argument, argErr := state.manager.GetArg(expression, i)
		if argErr != nil {
			return false, argErr
		}
		if isBV, bvErr := state.manager.IsBitVector(argument); bvErr != nil {
			return false, bvErr
		} else if isBV {
			if argErr = state.Internalize(argument, false); argErr != nil {
				return false, argErr
			}
		}
	}
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
	bits, err := t.bitsFor(expression)
	if err != nil {
		return false, err
	}
	for _, bit := range bits {
		if err = state.Internalize(bit, true); err != nil {
			return false, err
		}
	}
	return true, nil
}

func (t *SMTBitVectorTheory) InternalizeEquality(expression ast.ASTHandle, variable int32) error {
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
	leftBits, err := t.bitsFor(left)
	if err != nil {
		return err
	}
	rightBits, err := t.bitsFor(right)
	if err != nil || len(leftBits) != len(rightBits) {
		if err != nil {
			return err
		}
		return ast.ErrASTDomain
	}
	equalities := make([]ast.ASTHandle, len(leftBits))
	for i := range equalities {
		equalities[i], err = state.manager.MkIff(leftBits[i], rightBits[i])
		if err != nil {
			return err
		}
	}
	definition, err := state.manager.MkAndArgs(equalities)
	if err != nil {
		return err
	}
	if err = state.Internalize(definition, true); err != nil {
		return err
	}
	defined, err := state.Literal(definition)
	if err != nil {
		return err
	}
	literal := NewSMTLiteral(variable, false)
	if err = state.addTheoryAxiom(t.FamilyID(), []SMTLiteral{literal.Negated(), defined}); err != nil {
		return err
	}
	return state.addTheoryAxiom(t.FamilyID(), []SMTLiteral{literal, defined.Negated()})
}

func (t *SMTBitVectorTheory) ApplySortConstraint(node *SMTEnode, sort ast.ASTHandle) error {
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

func (*SMTBitVectorTheory) NewEquality(SMTTheoryVar, SMTTheoryVar) error    { return nil }
func (*SMTBitVectorTheory) NewDisequality(SMTTheoryVar, SMTTheoryVar) error { return nil }
func (*SMTBitVectorTheory) Assign(int32, bool) error                        { return nil }

func (t *SMTBitVectorTheory) PushScope() error {
	if err := t.SMTTheoryBase.PushScope(); err != nil {
		return err
	}
	t.scopes = append(t.scopes, smtBVScope{len(t.bitTrail), t.unsupported})
	return nil
}

func (t *SMTBitVectorTheory) PopScopes(count uint32) error {
	if count == 0 || count > uint32(len(t.scopes)) {
		return ast.ErrASTDomain
	}
	level := len(t.scopes) - int(count)
	scope := t.scopes[level]
	for _, expression := range t.bitTrail[scope.bits:] {
		delete(t.bits, expression)
	}
	t.bitTrail = t.bitTrail[:scope.bits]
	t.unsupported = scope.unsupported
	t.scopes = t.scopes[:level]
	return t.SMTTheoryBase.PopScopes(count)
}

func (t *SMTBitVectorTheory) FinalCheck() (SMTFinalCheckStatus, error) {
	if t.unsupported {
		return SMTFinalCheckGiveUp, nil
	}
	return SMTFinalCheckDone, nil
}

func (*SMTBitVectorTheory) Name() string { return "bv" }

var _ SMTTheoryInternalizer = (*SMTBitVectorTheory)(nil)
