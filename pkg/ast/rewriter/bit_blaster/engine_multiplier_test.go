package bit_blaster

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestBitBlasterMultiplierExhaustiveSymbolicCircuit(t *testing.T) {
	m, engine := arithmeticEngine(t)
	boolSort, err := m.MkBoolSort()
	paramsMust(t, err)
	for width := 1; width <= 4; width++ {
		a := make([]ast.ASTHandle, width)
		b := make([]ast.ASTHandle, width)
		for i := 0; i < width; i++ {
			a[i], err = m.MkConst(usymbol.Numerical(uint32(900+width*20+i)), boolSort)
			paramsMust(t, err)
			gatePin(t, m, a[i])
			b[i], err = m.MkConst(usymbol.Numerical(uint32(910+width*20+i)), boolSort)
			paramsMust(t, err)
			gatePin(t, m, b[i])
		}
		product, err := engine.MkMultiplier(a, b)
		paramsMust(t, err)
		productBits := arithmeticHandles(t, engine, product)
		limit := 1 << width
		for av := 0; av < limit; av++ {
			for bv := 0; bv < limit; bv++ {
				values := make(map[ast.ASTHandle]bool, 2*width)
				for i := 0; i < width; i++ {
					values[a[i]] = av&(1<<i) != 0
					values[b[i]] = bv&(1<<i) != 0
				}
				if got, want := evaluatedBits(t, m, productBits, values), int64((av*bv)&(limit-1)); got != want {
					t.Fatalf("width %d: %d*%d=%d, want %d", width, av, bv, got, want)
				}
			}
		}
		arithmeticRelease(t, m, product)
	}
}

func TestBitBlasterMultiplierNumeralAndMinusOnePaths(t *testing.T) {
	m, engine := arithmeticEngine(t)
	for width := 1; width <= 5; width++ {
		limit := int64(1 << width)
		for a := int64(0); a < limit; a++ {
			aRefs, aBits := logicBits(t, engine, a, width)
			for b := int64(0); b < limit; b++ {
				bRefs, bBits := logicBits(t, engine, b, width)
				product, err := engine.MkMultiplier(aBits, bBits)
				paramsMust(t, err)
				if got, want := arithmeticValue(t, engine, product), (a*b)&(limit-1); got != want {
					t.Fatalf("width %d: constant %d*%d=%d, want %d", width, a, b, got, want)
				}
				arithmeticRelease(t, m, product)
				arithmeticRelease(t, m, bRefs)
			}
			arithmeticRelease(t, m, aRefs)
		}
	}
}

func TestBitBlasterReachedDiscardedConstantCaseCircuit(t *testing.T) {
	m, engine := arithmeticEngine(t)
	boolSort, err := m.MkBoolSort()
	paramsMust(t, err)
	constantRefs, constant := logicBits(t, engine, 3, 3)
	symbolic := make([]ast.ASTHandle, 3)
	for i := range symbolic {
		symbolic[i], err = m.MkConst(usymbol.Numerical(uint32(1050+i)), boolSort)
		paramsMust(t, err)
		gatePin(t, m, symbolic[i])
	}
	circuit, err := engine.constCaseMultiplier(constant, symbolic)
	paramsMust(t, err)
	if len(circuit) != 3 {
		t.Fatal("constant-case circuit width", len(circuit))
	}
	handles := arithmeticHandles(t, engine, circuit)
	for value := 0; value < 8; value++ {
		assignment := make(map[ast.ASTHandle]bool, 3)
		for i := range symbolic {
			assignment[symbolic[i]] = value&(1<<i) != 0
		}
		if got, want := evaluatedBits(t, m, handles, assignment), int64((3*value)&7); got != want {
			t.Fatalf("constant-case 3*%d=%d, want %d", value, got, want)
		}
	}
	arithmeticRelease(t, m, circuit)
	arithmeticRelease(t, m, constantRefs)
}
