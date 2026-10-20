package bit_blaster

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func bitBlasterBooleanEval(t *testing.T, m *ast.ASTManager, h ast.ASTHandle, values map[ast.ASTHandle]bool) bool {
	t.Helper()
	if value, ok := values[h]; ok {
		return value
	}
	op, err := m.GetDeclKind(h)
	paramsMust(t, err)
	count, err := m.GetNumArgs(h)
	paramsMust(t, err)
	child := func(index uint32) bool {
		arg, err := m.GetArg(h, index)
		paramsMust(t, err)
		return bitBlasterBooleanEval(t, m, arg, values)
	}
	switch op {
	case ast.ASTOpTrue:
		return true
	case ast.ASTOpFalse:
		return false
	case ast.ASTOpNot:
		return !child(0)
	case ast.ASTOpIff, ast.ASTOpEq:
		return child(0) == child(1)
	case ast.ASTOpXor:
		return child(0) != child(1)
	case ast.ASTOpIte:
		if child(0) {
			return child(1)
		}
		return child(2)
	case ast.ASTOpAnd:
		for i := uint32(0); i < count; i++ {
			if !child(i) {
				return false
			}
		}
		return true
	case ast.ASTOpOr:
		for i := uint32(0); i < count; i++ {
			if child(i) {
				return true
			}
		}
		return false
	case 59:
		return child(0) && child(1) || child(0) && child(2) || child(1) && child(2)
	case 60:
		return (child(0) != child(1)) != child(2)
	default:
		t.Fatalf("unsupported bit-blaster Boolean operation %d", op)
		return false
	}
}

func evaluatedBits(t *testing.T, m *ast.ASTManager, handles []ast.ASTHandle, values map[ast.ASTHandle]bool) int64 {
	t.Helper()
	var result int64
	for i, handle := range handles {
		if bitBlasterBooleanEval(t, m, handle, values) {
			result |= int64(1) << i
		}
	}
	return result
}

func TestBitBlasterExtendedRotationsExhaustiveConstants(t *testing.T) {
	m, engine := arithmeticEngine(t)
	for width := 1; width <= 5; width++ {
		limit := int64(1 << width)
		mask := limit - 1
		for value := int64(0); value < limit; value++ {
			valueRefs, valueBits := logicBits(t, engine, value, width)
			for amount := int64(0); amount < limit; amount++ {
				amountRefs, amountBits := logicBits(t, engine, amount, width)
				left, err := engine.MkExtendedRotateLeft(valueBits, amountBits)
				paramsMust(t, err)
				n := amount % int64(width)
				wantLeft := value
				if n != 0 {
					wantLeft = ((value << n) | (value >> (int64(width) - n))) & mask
				}
				if got := arithmeticValue(t, engine, left); got != wantLeft {
					t.Fatalf("width %d: ext-rol(%d,%d)=%d, want %d", width, value, amount, got, wantLeft)
				}
				arithmeticRelease(t, m, left)
				right, err := engine.MkExtendedRotateRight(valueBits, amountBits)
				paramsMust(t, err)
				wantRight := value
				if n != 0 {
					wantRight = ((value >> n) | (value << (int64(width) - n))) & mask
				}
				if got := arithmeticValue(t, engine, right); got != wantRight {
					t.Fatalf("width %d: ext-ror(%d,%d)=%d, want %d", width, value, amount, got, wantRight)
				}
				arithmeticRelease(t, m, right)
				arithmeticRelease(t, m, amountRefs)
			}
			arithmeticRelease(t, m, valueRefs)
		}
	}
}

func TestBitBlasterExtendedRotationSymbolicModuloCircuit(t *testing.T) {
	m, engine := arithmeticEngine(t)
	boolSort, err := m.MkBoolSort()
	paramsMust(t, err)
	condition, err := m.MkConst(usymbol.Numerical(841), boolSort)
	paramsMust(t, err)
	gatePin(t, m, condition)
	falsehood, err := m.MkFalse()
	paramsMust(t, err)
	valueRefs, valueBits := logicBits(t, engine, 5, 3)
	amount := []ast.ASTHandle{condition, falsehood, falsehood}
	left, err := engine.MkExtendedRotateLeft(valueBits, amount)
	paramsMust(t, err)
	handles := arithmeticHandles(t, engine, left)
	if got := evaluatedBits(t, m, handles, map[ast.ASTHandle]bool{condition: false}); got != 5 {
		t.Fatal("symbolic rotate amount zero", got)
	}
	if got := evaluatedBits(t, m, handles, map[ast.ASTHandle]bool{condition: true}); got != 3 {
		t.Fatal("symbolic rotate amount one", got)
	}
	arithmeticRelease(t, m, left)
	arithmeticRelease(t, m, valueRefs)
}
