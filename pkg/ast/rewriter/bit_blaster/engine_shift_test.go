package bit_blaster

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestBitBlasterShiftsExhaustiveConstantAmounts(t *testing.T) {
	m, engine := arithmeticEngine(t)
	for width := 1; width <= 5; width++ {
		limit := int64(1 << width)
		mask := limit - 1
		for value := int64(0); value < limit; value++ {
			valueRefs, valueBits := logicBits(t, engine, value, width)
			for amount := int64(0); amount < limit; amount++ {
				amountRefs, amountBits := logicBits(t, engine, amount, width)
				left, err := engine.MkShiftLeft(valueBits, amountBits)
				paramsMust(t, err)
				wantLeft := int64(0)
				if amount < int64(width) {
					wantLeft = (value << amount) & mask
				}
				if got := arithmeticValue(t, engine, left); got != wantLeft {
					t.Fatalf("width %d: %d << %d = %d, want %d", width, value, amount, got, wantLeft)
				}
				arithmeticRelease(t, m, left)

				logical, err := engine.MkLogicalShiftRight(valueBits, amountBits)
				paramsMust(t, err)
				wantLogical := int64(0)
				if amount < int64(width) {
					wantLogical = value >> amount
				}
				if got := arithmeticValue(t, engine, logical); got != wantLogical {
					t.Fatalf("width %d: lshr(%d,%d) = %d, want %d", width, value, amount, got, wantLogical)
				}
				arithmeticRelease(t, m, logical)

				arithmetic, err := engine.MkArithmeticShiftRight(valueBits, amountBits)
				paramsMust(t, err)
				signed := signedBitValue(value, width)
				shift := amount
				if shift > int64(width) {
					shift = int64(width)
				}
				wantArithmetic := (signed >> shift) & mask
				if got := arithmeticValue(t, engine, arithmetic); got != wantArithmetic {
					t.Fatalf("width %d: ashr(%d,%d) = %d, want %d", width, value, amount, got, wantArithmetic)
				}
				arithmeticRelease(t, m, arithmetic)
				arithmeticRelease(t, m, amountRefs)
			}
			arithmeticRelease(t, m, valueRefs)
		}
	}
}

func TestBitBlasterSymbolicShiftUsesSourceBarrelStage(t *testing.T) {
	m, engine := arithmeticEngine(t)
	boolSort, err := m.MkBoolSort()
	paramsMust(t, err)
	condition, err := m.MkConst(usymbol.Numerical(821), boolSort)
	paramsMust(t, err)
	gatePin(t, m, condition)
	falsehood, err := m.MkFalse()
	paramsMust(t, err)
	valueRefs, valueBits := logicBits(t, engine, 11, 4)
	amount := []ast.ASTHandle{condition, falsehood, falsehood, falsehood}
	shifted, err := engine.MkShiftLeft(valueBits, amount)
	paramsMust(t, err)
	got := arithmeticHandles(t, engine, shifted)
	for index := range got {
		thenBit := falsehood
		if index > 0 {
			thenBit = valueBits[index-1]
		}
		expected, err := engine.gateRef(ast.ASTOpIte, condition, thenBit, valueBits[index])
		paramsMust(t, err)
		want, err := m.RefGet(expected)
		paramsMust(t, err)
		if got[index] != want {
			t.Fatalf("barrel bit %d = %v, want %v", index, got[index], want)
		}
		paramsMust(t, m.DelRef(&expected))
	}
	arithmeticRelease(t, m, shifted)
	arithmeticRelease(t, m, valueRefs)
}
