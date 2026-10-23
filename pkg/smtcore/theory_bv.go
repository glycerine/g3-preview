// Source: src/smt/theory_bv.h and src/smt/theory_bv.cpp bit-vector
// variables, numeral bits, bit2bool atoms, and equality expansion.
package smtcore

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/ast/rewriter"
	protomodel "github.com/glycerine/g3/pkg/smtcore/proto_model"
	"github.com/glycerine/g3/pkg/util/lbool"
	"github.com/glycerine/g3/pkg/util/statistics"
)

const (
	smtBVOpNumeral  int32 = 0
	smtBVOpBit2Bool int32 = 55
)

type smtBVScope struct {
	bits, lazyBridges, fixedEqualities int
	unsupported                        bool
}

type smtBVFixedPair struct{ left, right ast.ASTHandle }

type smtBVStatistics struct {
	conflicts, diseqStatic, diseqDynamic uint64
	bit2core, theoryToCoreEq, eqDynamic  uint64
}

// SMTBitVectorTheory retains the source theory's little-endian Boolean-bit
// representation. Operations are added incrementally; any reached operation
// without a bit definition makes final-check report incompleteness.
type SMTBitVectorTheory struct {
	*SMTTheoryBase
	bits             map[ast.ASTHandle][]ast.ASTHandle
	bitExpressions   map[ast.ASTHandle][]ast.ASTHandle
	bitOwners        map[ast.ASTHandle]ast.ASTRef
	bitTrail         []ast.ASTHandle
	lazyBridges      map[ast.ASTHandle]ast.ASTRef
	lazyBridgeTrail  []ast.ASTHandle
	fixedEqualities  map[smtBVFixedPair]bool
	fixedTrail       []smtBVFixedPair
	temporary        []ast.ASTRef
	scopes           []smtBVScope
	unsupported      bool
	statistics       smtBVStatistics
	config           SMTBVState
	discoveringFixed bool
}

func (t *SMTBitVectorTheory) ConfigureBitVector(config SMTBVState) error {
	if t == nil {
		return ast.ErrASTDomain
	}
	t.config = config
	return nil
}

func NewSMTBitVectorTheory(family ast.FamilyID) (*SMTBitVectorTheory, error) {
	base, err := NewSMTTheoryBase(family)
	if err != nil {
		return nil, err
	}
	return &SMTBitVectorTheory{
		SMTTheoryBase:   base,
		bits:            make(map[ast.ASTHandle][]ast.ASTHandle),
		bitExpressions:  make(map[ast.ASTHandle][]ast.ASTHandle),
		bitOwners:       make(map[ast.ASTHandle]ast.ASTRef),
		lazyBridges:     make(map[ast.ASTHandle]ast.ASTRef),
		fixedEqualities: make(map[smtBVFixedPair]bool),
		config:          SMTBVState{Mode: 1, HiDiv0: true, Reflect: true, BlastMaxSize: 2147483647, EnableInt2BV: true},
	}, nil
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

func (t *SMTBitVectorTheory) argumentBits(expression ast.ASTHandle) ([][]ast.ASTHandle, error) {
	state, _ := t.State()
	count, err := state.manager.GetNumArgs(expression)
	if err != nil {
		return nil, err
	}
	result := make([][]ast.ASTHandle, count)
	for i := range result {
		argument, argErr := state.manager.GetArg(expression, uint32(i))
		if argErr != nil {
			return nil, argErr
		}
		result[i], argErr = t.bitsFor(argument)
		if argErr != nil {
			return nil, argErr
		}
	}
	return result, nil
}

func (t *SMTBitVectorTheory) retainTemporary(bits []ast.ASTHandle) error {
	if len(bits) == 0 {
		return ast.ErrASTDomain
	}
	state, _ := t.State()
	assembled, err := state.manager.MkBitVectorFromBits(bits)
	if err != nil {
		return err
	}
	owner, err := state.manager.NewRef(assembled)
	if err != nil {
		return err
	}
	t.temporary = append(t.temporary, owner)
	return nil
}

func (t *SMTBitVectorTheory) releaseTemporary() (result error) {
	state, err := t.State()
	if err != nil {
		return err
	}
	for i := range t.temporary {
		result = errors.Join(result, state.manager.DelRef(&t.temporary[i]))
	}
	t.temporary = t.temporary[:0]
	return result
}

func (t *SMTBitVectorTheory) mapUnary(bits []ast.ASTHandle, operation func(ast.ASTHandle) (ast.ASTHandle, error)) ([]ast.ASTHandle, error) {
	result := make([]ast.ASTHandle, len(bits))
	for i := range bits {
		var err error
		result[i], err = operation(bits[i])
		if err != nil {
			return nil, err
		}
	}
	return result, t.retainTemporary(result)
}

func (t *SMTBitVectorTheory) mapBinary(left, right []ast.ASTHandle, operation func(ast.ASTHandle, ast.ASTHandle) (ast.ASTHandle, error)) ([]ast.ASTHandle, error) {
	if len(left) != len(right) {
		return nil, ast.ErrASTDomain
	}
	result := make([]ast.ASTHandle, len(left))
	for i := range left {
		var err error
		result[i], err = operation(left[i], right[i])
		if err != nil {
			return nil, err
		}
	}
	return result, t.retainTemporary(result)
}

func (t *SMTBitVectorTheory) boolNot(value ast.ASTHandle) (ast.ASTHandle, error) {
	state, _ := t.State()
	if state.manager.IsTrue(value) {
		return state.manager.MkFalse()
	}
	if state.manager.IsFalse(value) {
		return state.manager.MkTrue()
	}
	if kind, err := state.manager.GetKind(value); err != nil {
		return ast.ASTHandle{}, err
	} else if kind == ast.ASTApp {
		family, err := state.manager.GetFamilyID(value)
		if err != nil {
			return ast.ASTHandle{}, err
		}
		op, err := state.manager.GetDeclKind(value)
		if err != nil {
			return ast.ASTHandle{}, err
		}
		if family == state.manager.BasicFamilyID() && op == ast.ASTOpNot {
			return state.manager.GetArg(value, 0)
		}
	}
	return state.manager.MkNot(value)
}

func (t *SMTBitVectorTheory) boolAnd(left, right ast.ASTHandle) (ast.ASTHandle, error) {
	state, _ := t.State()
	if state.manager.IsFalse(left) || state.manager.IsFalse(right) {
		return state.manager.MkFalse()
	}
	if state.manager.IsTrue(left) || left == right {
		return right, nil
	}
	if state.manager.IsTrue(right) {
		return left, nil
	}
	return state.manager.MkAnd(left, right)
}

func (t *SMTBitVectorTheory) boolOr(left, right ast.ASTHandle) (ast.ASTHandle, error) {
	state, _ := t.State()
	if state.manager.IsTrue(left) || state.manager.IsTrue(right) {
		return state.manager.MkTrue()
	}
	if state.manager.IsFalse(left) || left == right {
		return right, nil
	}
	if state.manager.IsFalse(right) {
		return left, nil
	}
	return state.manager.MkOr(left, right)
}

func (t *SMTBitVectorTheory) boolXor(left, right ast.ASTHandle) (ast.ASTHandle, error) {
	state, _ := t.State()
	if left == right {
		return state.manager.MkFalse()
	}
	if state.manager.IsFalse(left) {
		return right, nil
	}
	if state.manager.IsFalse(right) {
		return left, nil
	}
	if state.manager.IsTrue(left) {
		return t.boolNot(right)
	}
	if state.manager.IsTrue(right) {
		return t.boolNot(left)
	}
	return state.manager.MkXor(left, right)
}

func (t *SMTBitVectorTheory) boolIff(left, right ast.ASTHandle) (ast.ASTHandle, error) {
	state, _ := t.State()
	if left == right {
		return state.manager.MkTrue()
	}
	if state.manager.IsTrue(left) {
		return right, nil
	}
	if state.manager.IsTrue(right) {
		return left, nil
	}
	if state.manager.IsFalse(left) {
		return t.boolNot(right)
	}
	if state.manager.IsFalse(right) {
		return t.boolNot(left)
	}
	return state.manager.MkIff(left, right)
}

func (t *SMTBitVectorTheory) boolIte(condition, thenValue, elseValue ast.ASTHandle) (ast.ASTHandle, error) {
	state, _ := t.State()
	out, err := state.manager.NewRef(ast.ASTHandle{})
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if err = rewriter.SimplifyBooleanGateConfigured(
		state.manager, ast.ASTOpIte,
		[]ast.ASTHandle{condition, thenValue, elseValue}, out, true, false,
	); err != nil {
		_ = state.manager.DelRef(&out)
		return ast.ASTHandle{}, err
	}
	value, err := state.manager.RefGet(out)
	if err != nil {
		_ = state.manager.DelRef(&out)
		return ast.ASTHandle{}, err
	}
	t.temporary = append(t.temporary, out)
	return value, nil
}

func (t *SMTBitVectorTheory) reduceBits(bits []ast.ASTHandle, and bool) (ast.ASTHandle, error) {
	if len(bits) == 0 {
		return ast.ASTHandle{}, ast.ErrASTDomain
	}
	result := bits[0]
	var err error
	for i := 1; i < len(bits); i++ {
		if and {
			result, err = t.boolAnd(result, bits[i])
		} else {
			result, err = t.boolOr(result, bits[i])
		}
		if err != nil {
			return ast.ASTHandle{}, err
		}
	}
	return result, t.retainTemporary([]ast.ASTHandle{result})
}

func (t *SMTBitVectorTheory) unsignedLE(left, right []ast.ASTHandle) (ast.ASTHandle, error) {
	if len(left) == 0 || len(left) != len(right) {
		return ast.ASTHandle{}, ast.ErrASTDomain
	}
	state, _ := t.State()
	result, err := state.manager.MkTrue()
	if err != nil {
		return ast.ASTHandle{}, err
	}
	for i := range left {
		equal, gateErr := t.boolIff(left[i], right[i])
		if gateErr != nil {
			return ast.ASTHandle{}, gateErr
		}
		notLeft, gateErr := t.boolNot(left[i])
		if gateErr != nil {
			return ast.ASTHandle{}, gateErr
		}
		less, gateErr := t.boolAnd(notLeft, right[i])
		if gateErr != nil {
			return ast.ASTHandle{}, gateErr
		}
		result, gateErr = t.boolIte(equal, result, less)
		if gateErr != nil {
			return ast.ASTHandle{}, gateErr
		}
	}
	return result, t.retainTemporary([]ast.ASTHandle{result})
}

func (t *SMTBitVectorTheory) addBits(left, right []ast.ASTHandle) ([]ast.ASTHandle, error) {
	if len(left) == 0 || len(left) != len(right) {
		return nil, ast.ErrASTDomain
	}
	state, _ := t.State()
	carry, err := state.manager.MkFalse()
	if err != nil {
		return nil, err
	}
	result := make([]ast.ASTHandle, len(left))
	for i := range left {
		partial, gateErr := t.boolXor(left[i], right[i])
		if gateErr != nil {
			return nil, gateErr
		}
		result[i], gateErr = t.boolXor(partial, carry)
		if gateErr != nil {
			return nil, gateErr
		}
		if i+1 == len(left) {
			continue
		}
		first, gateErr := t.boolAnd(left[i], right[i])
		if gateErr != nil {
			return nil, gateErr
		}
		second, gateErr := t.boolAnd(left[i], carry)
		if gateErr != nil {
			return nil, gateErr
		}
		third, gateErr := t.boolAnd(right[i], carry)
		if gateErr != nil {
			return nil, gateErr
		}
		carry, gateErr = t.boolOr(first, second)
		if gateErr == nil {
			carry, gateErr = t.boolOr(carry, third)
		}
		if gateErr != nil {
			return nil, gateErr
		}
	}
	return result, t.retainTemporary(result)
}

func (t *SMTBitVectorTheory) subtractBits(left, right []ast.ASTHandle) ([]ast.ASTHandle, ast.ASTHandle, error) {
	if len(left) == 0 || len(left) != len(right) {
		return nil, ast.ASTHandle{}, ast.ErrASTDomain
	}
	state, _ := t.State()
	carry, err := state.manager.MkTrue()
	if err != nil {
		return nil, ast.ASTHandle{}, err
	}
	result := make([]ast.ASTHandle, len(left))
	for i := range left {
		notRight, gateErr := t.boolNot(right[i])
		if gateErr != nil {
			return nil, ast.ASTHandle{}, gateErr
		}
		partial, gateErr := t.boolXor(left[i], notRight)
		if gateErr != nil {
			return nil, ast.ASTHandle{}, gateErr
		}
		result[i], gateErr = t.boolXor(partial, carry)
		if gateErr != nil {
			return nil, ast.ASTHandle{}, gateErr
		}
		first, gateErr := t.boolAnd(left[i], notRight)
		if gateErr != nil {
			return nil, ast.ASTHandle{}, gateErr
		}
		second, gateErr := t.boolAnd(left[i], carry)
		if gateErr != nil {
			return nil, ast.ASTHandle{}, gateErr
		}
		third, gateErr := t.boolAnd(notRight, carry)
		if gateErr != nil {
			return nil, ast.ASTHandle{}, gateErr
		}
		carry, gateErr = t.boolOr(first, second)
		if gateErr == nil {
			carry, gateErr = t.boolOr(carry, third)
		}
		if gateErr != nil {
			return nil, ast.ASTHandle{}, gateErr
		}
	}
	retained := append(append([]ast.ASTHandle(nil), result...), carry)
	return result, carry, t.retainTemporary(retained)
}

func (t *SMTBitVectorTheory) unsignedDivRem(left, right []ast.ASTHandle) ([]ast.ASTHandle, []ast.ASTHandle, error) {
	if len(left) == 0 || len(left) != len(right) {
		return nil, nil, ast.ErrASTDomain
	}
	state, _ := t.State()
	width := len(left)
	partial := make([]ast.ASTHandle, width)
	partial[0] = left[width-1]
	var err error
	for i := 1; i < width; i++ {
		partial[i], err = state.manager.MkFalse()
		if err != nil {
			return nil, nil, err
		}
	}
	quotient := make([]ast.ASTHandle, width)
	for stage := 0; stage < width; stage++ {
		difference, carry, subtractErr := t.subtractBits(partial, right)
		if subtractErr != nil {
			return nil, nil, subtractErr
		}
		quotient[width-stage-1] = carry
		next := make([]ast.ASTHandle, width)
		if stage+1 < width {
			next[0] = left[width-stage-2]
			for i := 1; i < width; i++ {
				next[i], err = t.boolIte(carry, difference[i-1], partial[i-1])
				if err != nil {
					return nil, nil, err
				}
			}
		} else {
			for i := range next {
				next[i], err = t.boolIte(carry, difference[i], partial[i])
				if err != nil {
					return nil, nil, err
				}
			}
		}
		if err = t.retainTemporary(next); err != nil {
			return nil, nil, err
		}
		partial = next
	}
	return quotient, partial, nil
}

func (t *SMTBitVectorTheory) muxBits(condition ast.ASTHandle, whenTrue, whenFalse []ast.ASTHandle) ([]ast.ASTHandle, error) {
	if len(whenTrue) == 0 || len(whenTrue) != len(whenFalse) {
		return nil, ast.ErrASTDomain
	}
	notCondition, err := t.boolNot(condition)
	if err != nil {
		return nil, err
	}
	result := make([]ast.ASTHandle, len(whenTrue))
	for i := range result {
		selectedTrue, gateErr := t.boolAnd(condition, whenTrue[i])
		if gateErr != nil {
			return nil, gateErr
		}
		selectedFalse, gateErr := t.boolAnd(notCondition, whenFalse[i])
		if gateErr != nil {
			return nil, gateErr
		}
		result[i], gateErr = t.boolOr(selectedTrue, selectedFalse)
		if gateErr != nil {
			return nil, gateErr
		}
	}
	return result, t.retainTemporary(result)
}

func (t *SMTBitVectorTheory) absoluteBits(bits []ast.ASTHandle) ([]ast.ASTHandle, error) {
	if len(bits) == 0 {
		return nil, ast.ErrASTDomain
	}
	negated, err := t.negateBits(bits)
	if err != nil {
		return nil, err
	}
	return t.muxBits(bits[len(bits)-1], negated, bits)
}

func (t *SMTBitVectorTheory) signedDivRemMod(left, right []ast.ASTHandle) ([]ast.ASTHandle, []ast.ASTHandle, []ast.ASTHandle, error) {
	if len(left) == 0 || len(left) != len(right) {
		return nil, nil, nil, ast.ErrASTDomain
	}
	absLeft, err := t.absoluteBits(left)
	if err != nil {
		return nil, nil, nil, err
	}
	absRight, err := t.absoluteBits(right)
	if err != nil {
		return nil, nil, nil, err
	}
	unsignedQuotient, unsignedRemainder, err := t.unsignedDivRem(absLeft, absRight)
	if err != nil {
		return nil, nil, nil, err
	}

	negatedQuotient, err := t.negateBits(unsignedQuotient)
	if err != nil {
		return nil, nil, nil, err
	}
	sameSign, err := t.boolIff(left[len(left)-1], right[len(right)-1])
	if err != nil {
		return nil, nil, nil, err
	}
	quotient, err := t.muxBits(sameSign, unsignedQuotient, negatedQuotient)
	if err != nil {
		return nil, nil, nil, err
	}

	negatedRemainder, err := t.negateBits(unsignedRemainder)
	if err != nil {
		return nil, nil, nil, err
	}
	remainder, err := t.muxBits(left[len(left)-1], negatedRemainder, unsignedRemainder)
	if err != nil {
		return nil, nil, nil, err
	}

	negativeRemainderPlusDivisor, err := t.addBits(negatedRemainder, right)
	if err != nil {
		return nil, nil, nil, err
	}
	remainderPlusDivisor, err := t.addBits(unsignedRemainder, right)
	if err != nil {
		return nil, nil, nil, err
	}
	negativeDivisorCases, err := t.muxBits(right[len(right)-1], negatedRemainder, negativeRemainderPlusDivisor)
	if err != nil {
		return nil, nil, nil, err
	}
	positiveDividendCases, err := t.muxBits(right[len(right)-1], remainderPlusDivisor, unsignedRemainder)
	if err != nil {
		return nil, nil, nil, err
	}
	modulusBody, err := t.muxBits(left[len(left)-1], negativeDivisorCases, positiveDividendCases)
	if err != nil {
		return nil, nil, nil, err
	}
	state, _ := t.State()
	zero := make([]ast.ASTHandle, len(left))
	for i := range zero {
		zero[i], err = state.manager.MkFalse()
		if err != nil {
			return nil, nil, nil, err
		}
	}
	remainderIsZero, err := t.equalBits(unsignedRemainder, zero)
	if err != nil {
		return nil, nil, nil, err
	}
	modulus, err := t.muxBits(remainderIsZero, unsignedRemainder, modulusBody)
	return quotient, remainder, modulus, err
}

func (t *SMTBitVectorTheory) unsignedBits(value uint64, width int) ([]ast.ASTHandle, error) {
	state, _ := t.State()
	result := make([]ast.ASTHandle, width)
	for i := range result {
		var err error
		result[i], err = state.manager.MkBoolVal(value&(uint64(1)<<uint(i)) != 0)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (t *SMTBitVectorTheory) equalBits(left, right []ast.ASTHandle) (ast.ASTHandle, error) {
	if len(left) == 0 || len(left) != len(right) {
		return ast.ASTHandle{}, ast.ErrASTDomain
	}
	equalities := make([]ast.ASTHandle, len(left))
	for i := range equalities {
		var err error
		equalities[i], err = t.boolIff(left[i], right[i])
		if err != nil {
			return ast.ASTHandle{}, err
		}
	}
	return t.reduceBits(equalities, true)
}

func (t *SMTBitVectorTheory) rotateBits(bits []ast.ASTHandle, amount uint64, left bool) ([]ast.ASTHandle, error) {
	if len(bits) == 0 {
		return nil, ast.ErrASTDomain
	}
	width := uint64(len(bits))
	amount %= width
	result := make([]ast.ASTHandle, len(bits))
	for i := range result {
		source := (uint64(i) + amount) % width
		if left {
			source = (width + uint64(i) - amount) % width
		}
		result[i] = bits[source]
	}
	return result, nil
}

func (t *SMTBitVectorTheory) extendedRotateBits(bits, amountBits []ast.ASTHandle, left bool) ([]ast.ASTHandle, error) {
	if len(bits) == 0 || len(bits) != len(amountBits) {
		return nil, ast.ErrASTDomain
	}
	widthBits, err := t.unsignedBits(uint64(len(bits)), len(bits))
	if err != nil {
		return nil, err
	}
	_, masked, err := t.unsignedDivRem(amountBits, widthBits)
	if err != nil {
		return nil, err
	}
	equalities := make([]ast.ASTHandle, len(bits))
	for amount := range equalities {
		value, valueErr := t.unsignedBits(uint64(amount), len(bits))
		if valueErr != nil {
			return nil, valueErr
		}
		equalities[amount], valueErr = t.equalBits(masked, value)
		if valueErr != nil {
			return nil, valueErr
		}
	}
	result := make([]ast.ASTHandle, len(bits))
	for index := range result {
		result[index] = bits[index]
		for amount := 1; amount < len(bits); amount++ {
			source := (index + amount) % len(bits)
			if left {
				source = (len(bits) + index - amount) % len(bits)
			}
			result[index], err = t.boolIte(equalities[amount], bits[source], result[index])
			if err != nil {
				return nil, err
			}
		}
	}
	return result, t.retainTemporary(result)
}

func (t *SMTBitVectorTheory) negateBits(bits []ast.ASTHandle) ([]ast.ASTHandle, error) {
	state, _ := t.State()
	inverted, err := t.mapUnary(bits, t.boolNot)
	if err != nil {
		return nil, err
	}
	one := make([]ast.ASTHandle, len(bits))
	for i := range one {
		one[i], err = state.manager.MkBoolVal(i == 0)
		if err != nil {
			return nil, err
		}
	}
	return t.addBits(inverted, one)
}

func (t *SMTBitVectorTheory) multiplyBits(left, right []ast.ASTHandle) ([]ast.ASTHandle, error) {
	if len(left) == 0 || len(left) != len(right) {
		return nil, ast.ErrASTDomain
	}
	state, _ := t.State()
	result := make([]ast.ASTHandle, len(left))
	var err error
	for i := range result {
		result[i], err = state.manager.MkFalse()
		if err != nil {
			return nil, err
		}
	}
	for shift := range right {
		row := make([]ast.ASTHandle, len(left))
		for i := range row {
			if i < shift {
				row[i], err = state.manager.MkFalse()
			} else {
				row[i], err = t.boolAnd(left[i-shift], right[shift])
			}
			if err != nil {
				return nil, err
			}
		}
		result, err = t.addBits(result, row)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (t *SMTBitVectorTheory) unsignedMulNoOverflow(left, right []ast.ASTHandle) (ast.ASTHandle, error) {
	if len(left) == 0 || len(left) != len(right) {
		return ast.ASTHandle{}, ast.ErrASTDomain
	}
	state, _ := t.State()
	width := len(left)
	extendedLeft := append([]ast.ASTHandle(nil), left...)
	extendedRight := append([]ast.ASTHandle(nil), right...)
	for i := 0; i < width; i++ {
		zero, err := state.manager.MkFalse()
		if err != nil {
			return ast.ASTHandle{}, err
		}
		extendedLeft = append(extendedLeft, zero)
		extendedRight = append(extendedRight, zero)
	}
	product, err := t.multiplyBits(extendedLeft, extendedRight)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	overflow, err := t.reduceBits(product[width:], false)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	return t.boolNot(overflow)
}

func (t *SMTBitVectorTheory) signedMulBound(left, right []ast.ASTHandle, positiveOverflow bool) (ast.ASTHandle, error) {
	if len(left) == 0 || len(left) != len(right) {
		return ast.ASTHandle{}, ast.ErrASTDomain
	}
	width := len(left)
	extendedLeft := append([]ast.ASTHandle(nil), left...)
	extendedRight := append([]ast.ASTHandle(nil), right...)
	for i := 0; i < width; i++ {
		extendedLeft = append(extendedLeft, left[width-1])
		extendedRight = append(extendedRight, right[width-1])
	}
	product, err := t.multiplyBits(extendedLeft, extendedRight)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	mismatches := make([]ast.ASTHandle, width)
	for i := range mismatches {
		mismatches[i], err = t.boolXor(product[width+i], product[width-1])
		if err != nil {
			return ast.ASTHandle{}, err
		}
	}
	overflow, err := t.reduceBits(mismatches, false)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	var signCondition ast.ASTHandle
	if positiveOverflow {
		signCondition, err = t.boolIff(left[width-1], right[width-1])
	} else {
		signCondition, err = t.boolXor(left[width-1], right[width-1])
	}
	if err != nil {
		return ast.ASTHandle{}, err
	}
	violation, err := t.boolAnd(signCondition, overflow)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	return t.boolNot(violation)
}

func (t *SMTBitVectorTheory) shiftBits(bits, amount []ast.ASTHandle, operation int32) ([]ast.ASTHandle, error) {
	if len(bits) == 0 || len(amount) == 0 {
		return nil, ast.ErrASTDomain
	}
	state, _ := t.State()
	result := append([]ast.ASTHandle(nil), bits...)
	for stage, condition := range amount {
		shift := len(bits)
		if stage < 63 {
			candidate := uint64(1) << uint(stage)
			if candidate < uint64(len(bits)) {
				shift = int(candidate)
			}
		}
		shifted := make([]ast.ASTHandle, len(bits))
		for i := range shifted {
			source := i - shift
			if operation != 45 {
				source = i + shift
			}
			if source >= 0 && source < len(result) {
				shifted[i] = result[source]
				continue
			}
			var err error
			if operation == 47 {
				shifted[i] = result[len(result)-1]
			} else {
				shifted[i], err = state.manager.MkFalse()
			}
			if err != nil {
				return nil, err
			}
		}
		for i := range result {
			var err error
			result[i], err = t.boolIte(condition, shifted[i], result[i])
			if err != nil {
				return nil, err
			}
		}
	}
	return result, nil
}

func (t *SMTBitVectorTheory) acBits(arguments [][]ast.ASTHandle, operation func([]ast.ASTHandle, []ast.ASTHandle) ([]ast.ASTHandle, error)) ([]ast.ASTHandle, error) {
	if len(arguments) < 2 {
		return nil, ast.ErrASTDomain
	}
	result := arguments[0]
	var err error
	for i := 1; i < len(arguments); i++ {
		result, err = operation(result, arguments[i])
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (t *SMTBitVectorTheory) parameter(expression ast.ASTHandle, index int) (value uint32, result error) {
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
			result = errors.Join(result, state.manager.DelParameter(&parameters[i]))
		}
	}()
	if index < 0 || index >= len(parameters) {
		return 0, ast.ErrASTDomain
	}
	integer, err := state.manager.ParameterInt(parameters[index])
	if err != nil || integer < 0 {
		if err != nil {
			return 0, err
		}
		return 0, ast.ErrASTDomain
	}
	return uint32(integer), nil
}

func (t *SMTBitVectorTheory) operationBits(expression ast.ASTHandle, operation int32) ([]ast.ASTHandle, bool, error) {
	state, _ := t.State()
	if operation == 56 {
		count, err := state.manager.GetNumArgs(expression)
		if err != nil {
			return nil, true, err
		}
		bits := make([]ast.ASTHandle, count)
		for i := range bits {
			bits[i], err = state.manager.GetArg(expression, uint32(i))
			if err != nil {
				return nil, true, err
			}
		}
		return bits, true, nil
	}
	arguments, err := t.argumentBits(expression)
	if err != nil {
		return nil, false, err
	}
	binary := func(gate func(ast.ASTHandle, ast.ASTHandle) (ast.ASTHandle, error)) func([]ast.ASTHandle, []ast.ASTHandle) ([]ast.ASTHandle, error) {
		return func(a, b []ast.ASTHandle) ([]ast.ASTHandle, error) { return t.mapBinary(a, b, gate) }
	}
	switch operation {
	case 3:
		if len(arguments) != 1 {
			return nil, true, ast.ErrASTDomain
		}
		bits, err := t.negateBits(arguments[0])
		return bits, true, err
	case 4:
		bits, err := t.acBits(arguments, t.addBits)
		return bits, true, err
	case 5:
		if len(arguments) != 2 {
			return nil, true, ast.ErrASTDomain
		}
		negated, err := t.negateBits(arguments[1])
		if err != nil {
			return nil, true, err
		}
		bits, err := t.addBits(arguments[0], negated)
		return bits, true, err
	case 6:
		bits, err := t.acBits(arguments, t.multiplyBits)
		return bits, true, err
	case 17, 19, 21:
		if len(arguments) != 2 {
			return nil, true, ast.ErrASTDomain
		}
		quotient, remainder, modulus, err := t.signedDivRemMod(arguments[0], arguments[1])
		switch operation {
		case 17:
			return quotient, true, err
		case 19:
			return remainder, true, err
		default:
			return modulus, true, err
		}
	case 18, 20:
		if len(arguments) != 2 {
			return nil, true, ast.ErrASTDomain
		}
		quotient, remainder, err := t.unsignedDivRem(arguments[0], arguments[1])
		if operation == 20 {
			return remainder, true, err
		}
		return quotient, true, err
	case 30, 31, 33:
		gate := t.boolAnd
		if operation == 31 {
			gate = t.boolOr
		} else if operation == 33 {
			gate = t.boolXor
		}
		bits, err := t.acBits(arguments, binary(gate))
		return bits, true, err
	case 32:
		if len(arguments) != 1 {
			return nil, true, ast.ErrASTDomain
		}
		bits, err := t.mapUnary(arguments[0], t.boolNot)
		return bits, true, err
	case 34, 35, 36:
		if len(arguments) != 2 {
			return nil, true, ast.ErrASTDomain
		}
		gate := t.boolAnd
		if operation == 35 {
			gate = t.boolOr
		} else if operation == 36 {
			gate = t.boolXor
		}
		bits, err := t.mapBinary(arguments[0], arguments[1], func(a, b ast.ASTHandle) (ast.ASTHandle, error) {
			value, gateErr := gate(a, b)
			if gateErr != nil {
				return ast.ASTHandle{}, gateErr
			}
			return t.boolNot(value)
		})
		return bits, true, err
	case 37:
		var bits []ast.ASTHandle
		for i := len(arguments); i > 0; i-- {
			bits = append(bits, arguments[i-1]...)
		}
		return bits, true, nil
	case 38, 39:
		if len(arguments) != 1 || len(arguments[0]) == 0 {
			return nil, true, ast.ErrASTDomain
		}
		extra, err := t.parameter(expression, 0)
		if err != nil {
			return nil, true, err
		}
		bits := append([]ast.ASTHandle(nil), arguments[0]...)
		fill, err := state.manager.MkFalse()
		if operation == 38 {
			fill = arguments[0][len(arguments[0])-1]
		}
		if err != nil {
			return nil, true, err
		}
		for i := uint32(0); i < extra; i++ {
			bits = append(bits, fill)
		}
		return bits, true, nil
	case 40:
		if len(arguments) != 1 {
			return nil, true, ast.ErrASTDomain
		}
		high, err := t.parameter(expression, 0)
		if err != nil {
			return nil, true, err
		}
		low, err := t.parameter(expression, 1)
		if err != nil || low > high || uint64(high) >= uint64(len(arguments[0])) {
			if err != nil {
				return nil, true, err
			}
			return nil, true, ast.ErrASTDomain
		}
		return append([]ast.ASTHandle(nil), arguments[0][low:high+1]...), true, nil
	case 42, 43:
		if len(arguments) != 1 {
			return nil, true, ast.ErrASTDomain
		}
		bit, err := t.reduceBits(arguments[0], operation == 43)
		return []ast.ASTHandle{bit}, true, err
	case 44:
		if len(arguments) != 2 || len(arguments[0]) != len(arguments[1]) {
			return nil, true, ast.ErrASTDomain
		}
		equalities := make([]ast.ASTHandle, len(arguments[0]))
		for i := range equalities {
			equalities[i], err = t.boolIff(arguments[0][i], arguments[1][i])
			if err != nil {
				return nil, true, err
			}
		}
		bit, err := t.reduceBits(equalities, true)
		return []ast.ASTHandle{bit}, true, err
	case 45, 46, 47:
		if len(arguments) != 2 || len(arguments[0]) != len(arguments[1]) {
			return nil, true, ast.ErrASTDomain
		}
		bits, err := t.shiftBits(arguments[0], arguments[1], operation)
		return bits, true, err
	case 48, 49:
		if len(arguments) != 1 {
			return nil, true, ast.ErrASTDomain
		}
		amount, err := t.parameter(expression, 0)
		if err != nil {
			return nil, true, err
		}
		bits, err := t.rotateBits(arguments[0], uint64(amount), operation == 48)
		return bits, true, err
	case 50, 51:
		if len(arguments) != 2 || len(arguments[0]) != len(arguments[1]) {
			return nil, true, ast.ErrASTDomain
		}
		bits, err := t.extendedRotateBits(arguments[0], arguments[1], operation == 50)
		return bits, true, err
	default:
		return nil, false, nil
	}
}

func (t *SMTBitVectorTheory) publishBridge(expression, definition ast.ASTHandle) error {
	state, _ := t.State()
	variable, found := state.boolVar[expression]
	if !found {
		return ast.ErrASTDomain
	}
	defined, err := state.Literal(definition)
	if err != nil {
		return fmt.Errorf("read bit-vector atom definition literal: %w", err)
	}
	literal := NewSMTLiteral(variable, false)
	if err = state.addTheoryAxiom(t.FamilyID(), []SMTLiteral{literal.Negated(), defined}); err != nil {
		return err
	}
	if err = state.addTheoryAxiom(t.FamilyID(), []SMTLiteral{literal, defined.Negated()}); err != nil {
		return err
	}
	if state.relevancy != nil {
		if err = state.relevancy.AddDependency(expression, definition); err != nil {
			return err
		}
	}
	return nil
}

func (t *SMTBitVectorTheory) bridgeAtom(expression, definition ast.ASTHandle, lazy bool) (bool, error) {
	state, _ := t.State()
	if err := state.Internalize(definition, true); err != nil {
		return false, fmt.Errorf("internalize bit-vector atom definition: %w", errors.Join(err, t.releaseTemporary()))
	}
	if err := t.releaseTemporary(); err != nil {
		return false, fmt.Errorf("release bit-vector atom definition: %w", err)
	}
	variable, found := state.boolVar[expression]
	var err error
	if !found {
		variable, err = state.MkBoolVar(expression)
		if err != nil {
			return false, fmt.Errorf("create bit-vector atom variable: %w", err)
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
	if lazy && t.config.LazyLE && state.relevancy != nil {
		if _, exists := t.lazyBridges[expression]; !exists {
			owner, ownerErr := state.manager.NewRef(definition)
			if ownerErr != nil {
				return false, ownerErr
			}
			t.lazyBridges[expression] = owner
			t.lazyBridgeTrail = append(t.lazyBridgeTrail, expression)
		}
		return true, nil
	}
	if err = t.publishBridge(expression, definition); err != nil {
		return false, err
	}
	return true, nil
}

// booleanGate3 expands the Boolean-valued BV helper gates emitted by the
// extended bit-blaster configuration. Carry is the majority of its inputs;
// xor3 is their odd parity.
func (t *SMTBitVectorTheory) booleanGate3(expression ast.ASTHandle, operation int32) (ast.ASTHandle, error) {
	state, _ := t.State()
	if count, err := state.manager.GetNumArgs(expression); err != nil || count != 3 {
		if err != nil {
			return ast.ASTHandle{}, err
		}
		return ast.ASTHandle{}, ast.ErrASTDomain
	}
	arguments := [3]ast.ASTHandle{}
	for i := range arguments {
		argument, err := state.manager.GetArg(expression, uint32(i))
		if err != nil {
			return ast.ASTHandle{}, err
		}
		arguments[i] = argument
	}
	if operation == 60 {
		partial, err := t.boolXor(arguments[0], arguments[1])
		if err != nil {
			return ast.ASTHandle{}, err
		}
		if err = t.retainTemporary([]ast.ASTHandle{partial}); err != nil {
			return ast.ASTHandle{}, err
		}
		return t.boolXor(partial, arguments[2])
	}
	first, err := t.boolAnd(arguments[0], arguments[1])
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if err = t.retainTemporary([]ast.ASTHandle{first}); err != nil {
		return ast.ASTHandle{}, err
	}
	second, err := t.boolAnd(arguments[0], arguments[2])
	if err != nil {
		return ast.ASTHandle{}, errors.Join(err, t.releaseTemporary())
	}
	if err = t.retainTemporary([]ast.ASTHandle{second}); err != nil {
		return ast.ASTHandle{}, errors.Join(err, t.releaseTemporary())
	}
	third, err := t.boolAnd(arguments[1], arguments[2])
	if err != nil {
		return ast.ASTHandle{}, errors.Join(err, t.releaseTemporary())
	}
	if err = t.retainTemporary([]ast.ASTHandle{third}); err != nil {
		return ast.ASTHandle{}, errors.Join(err, t.releaseTemporary())
	}
	partial, err := t.boolOr(first, second)
	if err != nil {
		return ast.ASTHandle{}, errors.Join(err, t.releaseTemporary())
	}
	if err = t.retainTemporary([]ast.ASTHandle{partial}); err != nil {
		return ast.ASTHandle{}, errors.Join(err, t.releaseTemporary())
	}
	return t.boolOr(partial, third)
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
	} else if own && op == 57 {
		bits, err = t.symbolicBits(expression, width)
	} else if own {
		var handled bool
		bits, handled, err = t.operationBits(expression, op)
		if err == nil && !handled {
			bits, err = t.symbolicBits(expression, width)
			// The source leaves its five division-by-zero value functions
			// uninterpreted and includes them in the generated model.
			if op < 12 || op > 16 {
				t.unsupported = true
			}
		}
	} else {
		bits, err = t.symbolicBits(expression, width)
	}
	if err != nil {
		return nil, errors.Join(err, t.releaseTemporary())
	}
	assembled, err := state.manager.MkBitVectorFromBits(bits)
	if err != nil {
		return nil, errors.Join(err, t.releaseTemporary())
	}
	owner, err := state.manager.NewRef(assembled)
	if err != nil {
		return nil, errors.Join(err, t.releaseTemporary())
	}
	if err = t.releaseTemporary(); err != nil {
		_ = state.manager.DelRef(&owner)
		return nil, err
	}
	// Publish before internalizing bit2bool atoms, whose owner lookup comes
	// back through this method.
	t.bits[expression] = bits
	for _, bit := range bits {
		bit, err = t.bitAtom(bit)
		if err != nil {
			return nil, err
		}
		if !state.manager.IsTrue(bit) && !state.manager.IsFalse(bit) {
			t.bitExpressions[bit] = append(t.bitExpressions[bit], expression)
		}
	}
	t.bitOwners[expression] = owner
	t.bitTrail = append(t.bitTrail, expression)
	return bits, nil
}

func (t *SMTBitVectorTheory) publishStaticDisequalities(expression ast.ASTHandle) error {
	state, err := t.State()
	if err != nil {
		return err
	}
	bits := t.bits[expression]
	if len(bits) == 0 {
		return nil
	}
	for other, otherBits := range t.bits {
		if other == expression || len(otherBits) != len(bits) {
			continue
		}
		complementary := false
		available := true
		for i := range bits {
			left, literalErr := state.Literal(bits[i])
			if literalErr != nil {
				if errors.Is(literalErr, ast.ErrASTDomain) {
					available = false
					break
				}
				return literalErr
			}
			right, literalErr := state.Literal(otherBits[i])
			if literalErr != nil {
				if errors.Is(literalErr, ast.ErrASTDomain) {
					available = false
					break
				}
				return literalErr
			}
			if left == right.Negated() {
				complementary = true
				break
			}
		}
		if !available || !complementary {
			continue
		}
		equality, equalityErr := t.MkEqualityLiteral(expression, other, true)
		if equalityErr != nil {
			return equalityErr
		}
		if equalityErr = state.addTheoryAxiom(t.FamilyID(), []SMTLiteral{equality.Negated()}); equalityErr != nil {
			return equalityErr
		}
		t.statistics.diseqStatic++
	}
	return nil
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
		if op == 59 || op == 60 {
			definition, gateErr := t.booleanGate3(expression, op)
			if gateErr != nil {
				return false, fmt.Errorf("expand bit-vector Boolean gate: %w", gateErr)
			}
			handled, bridgeErr := t.bridgeAtom(expression, definition, false)
			if bridgeErr != nil {
				return false, fmt.Errorf("bridge bit-vector Boolean gate: %w", bridgeErr)
			}
			return handled, nil
		}
		if op != 22 && op != 23 && op != 52 && op != 53 && op != 54 {
			t.unsupported = true
			return false, nil
		}
		arguments, argErr := t.argumentBits(expression)
		if argErr != nil {
			return false, argErr
		}
		if len(arguments) != 2 || len(arguments[0]) == 0 || len(arguments[0]) != len(arguments[1]) {
			return false, ast.ErrASTDomain
		}
		left, right := arguments[0], arguments[1]
		if op == 52 {
			definition, overflowErr := t.unsignedMulNoOverflow(left, right)
			if overflowErr != nil {
				return false, overflowErr
			}
			return t.bridgeAtom(expression, definition, true)
		}
		if op == 53 || op == 54 {
			definition, overflowErr := t.signedMulBound(left, right, op == 53)
			if overflowErr != nil {
				return false, overflowErr
			}
			return t.bridgeAtom(expression, definition, true)
		}
		if op == 23 {
			left = append([]ast.ASTHandle(nil), left...)
			right = append([]ast.ASTHandle(nil), right...)
			left[len(left)-1], argErr = t.boolNot(left[len(left)-1])
			if argErr == nil {
				right[len(right)-1], argErr = t.boolNot(right[len(right)-1])
			}
			if argErr != nil {
				return false, argErr
			}
		}
		definition, argErr := t.unsignedLE(left, right)
		if argErr != nil {
			return false, argErr
		}
		return t.bridgeAtom(expression, definition, true)
	}
	state, _ := t.State()
	owner, err := state.manager.GetArg(expression, 0)
	if err != nil {
		return false, err
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
	// Publish this atom before internalizing its owner. If the owner creates
	// this same bit while building its circuit, the recursive visit observes
	// the existing Boolean variable and terminates.
	if err = state.Internalize(owner, false); err != nil {
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
		t.statistics.bit2core++
		if state.relevancy != nil {
			if err = state.relevancy.AddDependency(expression, definition); err != nil {
				return false, err
			}
		}
	}
	return true, nil
}

func (t *SMTBitVectorTheory) InternalizeTerm(expression ast.ASTHandle) (bool, error) {
	state, err := t.State()
	if err != nil {
		return false, err
	}
	op, own, err := t.operation(expression)
	if err != nil || !own {
		return false, err
	}
	if (op == 57 || op == 58) && !t.config.EnableInt2BV {
		return false, nil
	}
	count, err := state.manager.GetNumArgs(expression)
	if err != nil {
		return false, err
	}
	for i := uint32(0); i <= count; i++ {
		candidate := expression
		if i < count {
			candidate, err = state.manager.GetArg(expression, i)
			if err != nil {
				return false, err
			}
		}
		isBV, candidateErr := state.manager.IsBitVector(candidate)
		if candidateErr != nil {
			return false, candidateErr
		}
		if isBV {
			width, widthErr := state.manager.BitVectorSize(candidate)
			if widthErr != nil {
				return false, widthErr
			}
			if width > t.config.BlastMaxSize {
				t.unsupported = true
				return false, nil
			}
		}
	}
	for i := uint32(0); i < count; i++ {
		argument, argErr := state.manager.GetArg(expression, i)
		if argErr != nil {
			return false, argErr
		}
		if isBV, bvErr := state.manager.IsBitVector(argument); bvErr != nil {
			return false, bvErr
		} else if isBV || op == 57 {
			if argErr = state.Internalize(argument, false); argErr != nil {
				return false, argErr
			}
		}
	}
	node := state.context.enodes[expression]
	if node == nil {
		node, err = state.MkEnode(expression, !t.config.Reflect, false, t.config.CC)
		if err != nil {
			return false, err
		}
	}
	if !t.IsAttachedToVariable(node) {
		if _, err = t.CreateVariable(node); err != nil {
			return false, err
		}
	}
	if op == 58 {
		argument, argErr := state.manager.GetArg(expression, 0)
		if argErr != nil {
			return false, argErr
		}
		bits, argErr := t.bitsFor(argument)
		if argErr != nil || len(bits) == 0 {
			if argErr != nil {
				return false, argErr
			}
			return false, ast.ErrASTDomain
		}
		zero, argErr := state.manager.ArithmeticIntegerNumeralText("0")
		if argErr != nil {
			return false, argErr
		}
		var sum ast.ASTHandle
		for i, bit := range bits {
			weight := new(big.Int).Lsh(big.NewInt(1), uint(i))
			numeral, numeralErr := state.manager.ArithmeticIntegerNumeralText(weight.String())
			if numeralErr != nil {
				return false, numeralErr
			}
			term, termErr := state.manager.MkIte(bit, numeral, zero)
			if termErr != nil {
				return false, termErr
			}
			if i == 0 {
				sum = term
			} else {
				sum, termErr = state.manager.MkArithmeticAdd(sum, term)
				if termErr != nil {
					return false, termErr
				}
			}
		}
		axiom, argErr := state.manager.MkEq(expression, sum)
		if argErr != nil {
			return false, argErr
		}
		if argErr = state.Internalize(axiom, true); argErr != nil {
			return false, argErr
		}
		literal, argErr := state.Literal(axiom)
		if argErr != nil {
			return false, argErr
		}
		if argErr = state.addTheoryAxiom(t.FamilyID(), []SMTLiteral{literal}); argErr != nil {
			return false, argErr
		}
		return true, nil
	}
	bits, err := t.bitsFor(expression)
	if err != nil {
		return false, err
	}
	for index, bit := range bits {
		if err = state.Internalize(bit, true); err != nil {
			return false, fmt.Errorf("bit-vector bit %d: %w", index, err)
		}
		literal, literalErr := state.Literal(bit)
		if literalErr != nil {
			return false, literalErr
		}
		if literal.Var() != SMTTrueBoolVar {
			data, dataErr := state.BooleanData(literal.Var())
			if dataErr != nil {
				return false, dataErr
			}
			// Source add_bit subscribes gate literals as well as bit2bool
			// atoms, so completion of an interpreted term triggers equality.
			if !data.IsTheoryAtom() {
				if err = state.SetVariableTheory(literal.Var(), t.FamilyID()); err != nil {
					return false, err
				}
			}
		}
	}
	if err = t.publishStaticDisequalities(expression); err != nil {
		return false, err
	}
	if op == 57 {
		argument, argErr := state.manager.GetArg(expression, 0)
		if argErr != nil {
			return false, argErr
		}
		converted, argErr := state.manager.MkBitVectorToInt(expression)
		if argErr != nil {
			return false, argErr
		}
		integerSort, argErr := state.manager.GetSort(argument)
		if argErr != nil {
			return false, argErr
		}
		// The source states bv2int(result) = argument mod 2^width. The
		// equivalent linear-integer form below introduces the quotient so the
		// retained arithmetic solver can enforce the same relation directly.
		quotient, argErr := state.manager.MkFreshConst("int2bv-quotient", integerSort)
		if argErr != nil {
			return false, argErr
		}
		modulus := new(big.Int).Lsh(big.NewInt(1), uint(len(bits)))
		modulusNumeral, argErr := state.manager.ArithmeticIntegerNumeralText(modulus.String())
		if argErr != nil {
			return false, argErr
		}
		multiple, argErr := state.manager.MkArithmeticMul(modulusNumeral, quotient)
		if argErr != nil {
			return false, argErr
		}
		sum, argErr := state.manager.MkArithmeticAdd(converted, multiple)
		if argErr != nil {
			return false, argErr
		}
		axiom, argErr := state.manager.MkEq(argument, sum)
		if argErr != nil {
			return false, argErr
		}
		if argErr = state.Internalize(axiom, true); argErr != nil {
			return false, argErr
		}
		literal, argErr := state.Literal(axiom)
		if argErr != nil {
			return false, argErr
		}
		if argErr = state.addTheoryAxiom(t.FamilyID(), []SMTLiteral{literal}); argErr != nil {
			return false, argErr
		}
	}
	// Source init_bits calls find_wpos immediately. All bits may already be
	// fixed when a new compound term reuses an existing circuit.
	if err = t.discoverFixedEqualities(); err != nil {
		return false, err
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
	if err = state.addTheoryAxiom(t.FamilyID(), []SMTLiteral{literal, defined.Negated()}); err != nil {
		return err
	}
	t.statistics.eqDynamic++
	return nil
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

func (t *SMTBitVectorTheory) Relevant(expression ast.ASTHandle) error {
	state, err := t.State()
	if err != nil || state.relevancy == nil {
		return err
	}
	if owner, found := t.lazyBridges[expression]; found {
		definition, refErr := state.manager.RefGet(owner)
		if refErr != nil {
			return refErr
		}
		if err = state.relevancy.MarkRelevant(definition); err != nil {
			return err
		}
		if err = t.publishBridge(expression, definition); err != nil {
			return err
		}
		if err = state.manager.DelRef(&owner); err != nil {
			return err
		}
		delete(t.lazyBridges, expression)
	}
	isBV, err := state.manager.IsBitVector(expression)
	if err != nil {
		return err
	}
	if !isBV {
		return nil
	}
	bits, err := t.bitsFor(expression)
	if err != nil {
		return err
	}
	for _, bit := range bits {
		if err = state.relevancy.MarkRelevant(bit); err != nil {
			return err
		}
	}
	return nil
}

func (t *SMTBitVectorTheory) NewEquality(left, right SMTTheoryVar) error {
	leftNode, err := t.Enode(left)
	if err != nil {
		return err
	}
	rightNode, err := t.Enode(right)
	if err != nil {
		return err
	}
	_, err = t.MkEqualityLiteral(leftNode.owner, rightNode.owner, true)
	return err
}

func (t *SMTBitVectorTheory) NewDisequality(left, right SMTTheoryVar) error {
	leftNode, err := t.Enode(left)
	if err != nil {
		return err
	}
	rightNode, err := t.Enode(right)
	if err != nil {
		return err
	}
	leftBits, err := t.bitsFor(leftNode.owner)
	if err != nil {
		return err
	}
	rightBits, err := t.bitsFor(rightNode.owner)
	if err != nil || len(leftBits) == 0 || len(leftBits) != len(rightBits) {
		if err != nil {
			return err
		}
		return ast.ErrASTDomain
	}
	state, _ := t.State()
	differences := make([]ast.ASTHandle, len(leftBits))
	for i := range differences {
		differences[i], err = t.boolXor(leftBits[i], rightBits[i])
		if err != nil {
			return errors.Join(err, t.releaseTemporary())
		}
	}
	if err = t.retainTemporary(differences); err != nil {
		return errors.Join(err, t.releaseTemporary())
	}
	clause := make([]SMTLiteral, 1, len(differences)+1)
	clause[0], err = t.MkEqualityLiteral(leftNode.owner, rightNode.owner, true)
	if err != nil {
		return errors.Join(err, t.releaseTemporary())
	}
	for _, difference := range differences {
		if err = state.Internalize(difference, true); err != nil {
			return errors.Join(err, t.releaseTemporary())
		}
		literal, literalErr := state.Literal(difference)
		if literalErr != nil {
			return errors.Join(literalErr, t.releaseTemporary())
		}
		clause = append(clause, literal)
	}
	if err = t.releaseTemporary(); err != nil {
		return err
	}
	t.statistics.diseqDynamic++
	return state.addTheoryAxiom(t.FamilyID(), clause)
}

// Source bit occurrences are indexed by Boolean variable, independently of
// literal polarity. Keep the expression index on the same underlying atom.
func (t *SMTBitVectorTheory) bitAtom(bit ast.ASTHandle) (ast.ASTHandle, error) {
	for {
		not, child, err := t.state.manager.IsNot(bit)
		if err != nil || !not {
			return bit, err
		}
		bit = child
	}
}

func (t *SMTBitVectorTheory) fixedAssignment(bits []ast.ASTHandle) (*big.Int, []SMTLiteral, bool, error) {
	state, err := t.State()
	if err != nil {
		return nil, nil, false, err
	}
	value := new(big.Int)
	antecedents := make([]SMTLiteral, 0, len(bits))
	for i, bit := range bits {
		atom, atomErr := t.bitAtom(bit)
		if atomErr != nil {
			return nil, nil, false, atomErr
		}
		if !state.manager.IsTrue(atom) && !state.manager.IsFalse(atom) {
			if _, internalized := state.BoolVar(atom); !internalized {
				return nil, nil, false, nil
			}
		}
		literal, literalErr := state.Literal(bit)
		if literalErr != nil {
			return nil, nil, false, literalErr
		}
		assignment, assignmentErr := state.Assignment(literal)
		if assignmentErr != nil {
			return nil, nil, false, assignmentErr
		}
		switch assignment {
		case lbool.Undef:
			return nil, nil, false, nil
		case lbool.True:
			value.SetBit(value, i, 1)
			antecedents = append(antecedents, literal)
		case lbool.False:
			antecedents = append(antecedents, literal.Negated())
		}
	}
	return value, antecedents, true, nil
}

func (t *SMTBitVectorTheory) Assign(variable int32, _ bool) (result error) {
	if t.discoveringFixed {
		return nil
	}
	state, err := t.State()
	if err != nil {
		return err
	}
	bit, err := state.BoolExpression(variable)
	if err != nil {
		return err
	}
	completed := false
	for _, expression := range t.bitExpressions[bit] {
		_, _, fixed, fixedErr := t.fixedAssignment(t.bits[expression])
		if fixedErr != nil {
			return fmt.Errorf("bit-vector completion check for %d: %w", expression.Slot(), fixedErr)
		}
		if fixed {
			completed = true
			break
		}
	}
	if !completed {
		return nil
	}
	return t.discoverFixedEqualities()
}

func (t *SMTBitVectorTheory) discoverFixedEqualities() error {
	if t.discoveringFixed {
		return nil
	}
	t.discoveringFixed = true
	defer func() { t.discoveringFixed = false }()
	state, err := t.State()
	if err != nil {
		return err
	}
	type fixedTerm struct {
		expression  ast.ASTHandle
		antecedents []SMTLiteral
	}
	fixed := make(map[string]fixedTerm)
	for expression, bits := range t.bits {
		node := state.context.enodes[expression]
		if node == nil || !t.IsAttachedToVariable(node) {
			continue
		}
		value, antecedents, complete, fixedErr := t.fixedAssignment(bits)
		if fixedErr != nil {
			return fmt.Errorf("bit-vector fixed assignment for %d: %w", expression.Slot(), fixedErr)
		}
		if !complete {
			continue
		}
		key := fmt.Sprintf("%d:%s", len(bits), value.String())
		prior, found := fixed[key]
		if !found {
			fixed[key] = fixedTerm{expression, antecedents}
			continue
		}
		priorNode := state.context.enodes[prior.expression]
		if priorNode == nil || priorNode.root == node.root {
			continue
		}
		pair := smtBVFixedPair{left: prior.expression, right: expression}
		if pair.left.Slot() > pair.right.Slot() {
			pair.left, pair.right = pair.right, pair.left
		}
		if t.fixedEqualities[pair] {
			continue
		}
		equality, equalityErr := t.MkEqualityLiteral(prior.expression, expression, true)
		if equalityErr != nil {
			return fmt.Errorf("bit-vector fixed-value equality %d %d: %w", prior.expression.Slot(), expression.Slot(), equalityErr)
		}
		clause := make([]SMTLiteral, 1, 1+len(prior.antecedents)+len(antecedents))
		clause[0] = equality
		for _, antecedent := range prior.antecedents {
			clause = append(clause, antecedent.Negated())
		}
		for _, antecedent := range antecedents {
			clause = append(clause, antecedent.Negated())
		}
		if equalityErr = state.addTheoryAxiom(t.FamilyID(), clause); equalityErr != nil {
			return fmt.Errorf("bit-vector fixed-value axiom %d %d: %w", prior.expression.Slot(), expression.Slot(), equalityErr)
		}
		t.fixedEqualities[pair] = true
		t.fixedTrail = append(t.fixedTrail, pair)
		t.statistics.theoryToCoreEq++
		if state.inconsistent {
			return nil
		}
	}
	return nil
}

func (t *SMTBitVectorTheory) PushScope() error {
	if err := t.SMTTheoryBase.PushScope(); err != nil {
		return err
	}
	t.scopes = append(t.scopes, smtBVScope{
		bits: len(t.bitTrail), lazyBridges: len(t.lazyBridgeTrail),
		fixedEqualities: len(t.fixedTrail), unsupported: t.unsupported,
	})
	return nil
}

func (t *SMTBitVectorTheory) PopScopes(count uint32) error {
	if count == 0 || count > uint32(len(t.scopes)) {
		return ast.ErrASTDomain
	}
	state, err := t.State()
	if err != nil {
		return err
	}
	level := len(t.scopes) - int(count)
	scope := t.scopes[level]
	for _, expression := range t.bitTrail[scope.bits:] {
		for _, bit := range t.bits[expression] {
			bit, err = t.bitAtom(bit)
			if err != nil {
				return err
			}
			owners := t.bitExpressions[bit]
			for i := len(owners) - 1; i >= 0; i-- {
				if owners[i] == expression {
					owners = append(owners[:i], owners[i+1:]...)
				}
			}
			if len(owners) == 0 {
				delete(t.bitExpressions, bit)
			} else {
				t.bitExpressions[bit] = owners
			}
		}
		delete(t.bits, expression)
		owner := t.bitOwners[expression]
		if err := state.manager.DelRef(&owner); err != nil {
			return err
		}
		delete(t.bitOwners, expression)
	}
	t.bitTrail = t.bitTrail[:scope.bits]
	for _, expression := range t.lazyBridgeTrail[scope.lazyBridges:] {
		owner, found := t.lazyBridges[expression]
		if !found {
			continue
		}
		if err := state.manager.DelRef(&owner); err != nil {
			return err
		}
		delete(t.lazyBridges, expression)
	}
	t.lazyBridgeTrail = t.lazyBridgeTrail[:scope.lazyBridges]
	for _, pair := range t.fixedTrail[scope.fixedEqualities:] {
		delete(t.fixedEqualities, pair)
	}
	t.fixedTrail = t.fixedTrail[:scope.fixedEqualities]
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

func (t *SMTBitVectorTheory) CollectStatistics(target *statistics.Statistics) error {
	if t == nil || target == nil {
		return ast.ErrASTHandle
	}
	target.UpdateUint64("bv conflicts", t.statistics.conflicts)
	target.UpdateUint64("bv diseqs", t.statistics.diseqStatic)
	target.UpdateUint64("bv dynamic diseqs", t.statistics.diseqDynamic)
	target.UpdateUint64("bv bit2core", t.statistics.bit2core)
	target.UpdateUint64("bv->core eq", t.statistics.theoryToCoreEq)
	target.UpdateUint64("bv dynamic eqs", t.statistics.eqDynamic)
	return nil
}

func (t *SMTBitVectorTheory) ResetStatistics() {
	if t != nil {
		t.statistics = smtBVStatistics{}
	}
}

func (t *SMTBitVectorTheory) Close() (result error) {
	if t == nil {
		return ast.ErrASTHandle
	}
	state, err := t.State()
	if err != nil {
		return err
	}
	result = errors.Join(result, t.releaseTemporary())
	for expression, owner := range t.bitOwners {
		result = errors.Join(result, state.manager.DelRef(&owner))
		delete(t.bitOwners, expression)
		delete(t.bits, expression)
	}
	t.bitExpressions = nil
	t.fixedEqualities = nil
	t.fixedTrail = nil
	for expression, owner := range t.lazyBridges {
		result = errors.Join(result, state.manager.DelRef(&owner))
		delete(t.lazyBridges, expression)
	}
	t.bitTrail = t.bitTrail[:0]
	t.lazyBridgeTrail = t.lazyBridgeTrail[:0]
	t.scopes = t.scopes[:0]
	return result
}

func (t *SMTBitVectorTheory) InitializeModel(generator *SMTModelGenerator) error {
	state, err := t.State()
	if err != nil || generator == nil || generator.manager != state.manager || generator.model == nil {
		if err != nil {
			return err
		}
		return ast.ErrASTDomain
	}
	factory, err := protomodel.NewBitVectorFactory(generator.manager)
	if err != nil {
		return err
	}
	return generator.RegisterFactory(factory)
}

// IncludeFunctionInterpretation retains the source model hook for the five
// unspecified division-by-zero functions.
func (t *SMTBitVectorTheory) IncludeFunctionInterpretation(declaration ast.ASTHandle) bool {
	state, err := t.State()
	if err != nil {
		return false
	}
	family, err := state.manager.GetFamilyID(declaration)
	if err != nil || family != t.FamilyID() {
		return false
	}
	op, err := state.manager.GetDeclKind(declaration)
	return err == nil && op >= 12 && op <= 16
}

func (t *SMTBitVectorTheory) ModelValue(node *SMTEnode, generator *SMTModelGenerator) (SMTModelValueProcedure, error) {
	if node == nil || generator == nil || !t.IsAttachedToVariable(node) || generator.model == nil {
		return nil, ast.ErrASTDomain
	}
	state, err := t.State()
	if err != nil {
		return nil, err
	}
	bits, err := t.bitsFor(node.owner)
	if err != nil {
		return nil, err
	}
	value := new(big.Int)
	for i, bit := range bits {
		if state.manager.IsFalse(bit) {
			continue
		}
		if state.manager.IsTrue(bit) {
			value.SetBit(value, i, 1)
			continue
		}
		if _, found := state.BoolVar(bit); !found {
			// A late theory witness can expose a symbolic bit after relevancy
			// propagation has finished. It is unconstrained, so choose zero.
			continue
		}
		literal, literalErr := state.Literal(bit)
		if literalErr != nil {
			return nil, literalErr
		}
		assignment, assignmentErr := state.Assignment(literal)
		if assignmentErr != nil {
			return nil, assignmentErr
		}
		if assignment == lbool.True {
			value.SetBit(value, i, 1)
		}
	}
	factory, ok := generator.model.Factory(t.FamilyID()).(*protomodel.BitVectorFactory)
	if !ok || factory == nil {
		return nil, ast.ErrASTDomain
	}
	numeral, err := factory.MkNumValue(new(big.Rat).SetInt(value), uint32(len(bits)))
	if err != nil {
		return nil, err
	}
	return NewSMTExpressionValueProcedure(numeral)
}

var _ SMTTheoryInternalizer = (*SMTBitVectorTheory)(nil)
