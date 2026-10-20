package bit_blaster

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func signedEncoding(value, mask int64) int64 { return value & mask }

func TestBitBlasterSignedDivisionRemainderModuloExhaustiveConstants(t *testing.T) {
	m, engine := arithmeticEngine(t)
	for width := 1; width <= 5; width++ {
		limit := int64(1 << width)
		mask := limit - 1
		for rawA := int64(0); rawA < limit; rawA++ {
			aRefs, aBits := logicBits(t, engine, rawA, width)
			a := signedBitValue(rawA, width)
			for rawB := int64(0); rawB < limit; rawB++ {
				bRefs, bBits := logicBits(t, engine, rawB, width)
				b := signedBitValue(rawB, width)

				division, err := engine.MkSignedDiv(aBits, bBits)
				paramsMust(t, err)
				wantDivision := int64(-1)
				if b == 0 {
					if a < 0 {
						wantDivision = 1
					}
				} else {
					wantDivision = a / b
				}
				wantDivision = signedEncoding(wantDivision, mask)
				if got := arithmeticValue(t, engine, division); got != wantDivision {
					t.Fatalf("width %d: sdiv(%d,%d)=%d, want %d", width, a, b, got, wantDivision)
				}
				arithmeticRelease(t, m, division)

				remainder, err := engine.MkSignedRem(aBits, bBits)
				paramsMust(t, err)
				wantRemainder := a
				if b != 0 {
					wantRemainder = a % b
				}
				wantRemainder = signedEncoding(wantRemainder, mask)
				if got := arithmeticValue(t, engine, remainder); got != wantRemainder {
					t.Fatalf("width %d: srem(%d,%d)=%d, want %d", width, a, b, got, wantRemainder)
				}
				arithmeticRelease(t, m, remainder)

				modulo, err := engine.MkSignedMod(aBits, bBits)
				paramsMust(t, err)
				wantModulo := a
				if b != 0 {
					u := a
					if u < 0 {
						u = -u
					}
					v := b
					if v < 0 {
						v = -v
					}
					u %= v
					switch {
					case u == 0:
						wantModulo = 0
					case a >= 0 && b >= 0:
						wantModulo = u
					case a < 0 && b >= 0:
						wantModulo = -u + b
					case a >= 0 && b < 0:
						wantModulo = u + b
					default:
						wantModulo = -u
					}
				}
				wantModulo = signedEncoding(wantModulo, mask)
				if got := arithmeticValue(t, engine, modulo); got != wantModulo {
					t.Fatalf("width %d: smod(%d,%d)=%d, want %d", width, a, b, got, wantModulo)
				}
				arithmeticRelease(t, m, modulo)
				arithmeticRelease(t, m, bRefs)
			}
			arithmeticRelease(t, m, aRefs)
		}
	}
}

func containsHandle(t *testing.T, m *ast.ASTManager, root, wanted ast.ASTHandle, seen map[ast.ASTHandle]bool) bool {
	t.Helper()
	if root == wanted {
		return true
	}
	if seen[root] {
		return false
	}
	seen[root] = true
	count, err := m.GetNumArgs(root)
	if err != nil {
		return false
	}
	for i := uint32(0); i < count; i++ {
		arg, err := m.GetArg(root, i)
		paramsMust(t, err)
		if containsHandle(t, m, arg, wanted, seen) {
			return true
		}
	}
	return false
}

func TestBitBlasterSignedOperationsRetainSymbolicSignMuxes(t *testing.T) {
	m, engine := arithmeticEngine(t)
	boolSort, err := m.MkBoolSort()
	paramsMust(t, err)
	aSign, err := m.MkConst(usymbol.Numerical(831), boolSort)
	paramsMust(t, err)
	gatePin(t, m, aSign)
	bSign, err := m.MkConst(usymbol.Numerical(832), boolSort)
	paramsMust(t, err)
	gatePin(t, m, bSign)
	truth, err := m.MkTrue()
	paramsMust(t, err)
	falsehood, err := m.MkFalse()
	paramsMust(t, err)
	a := []ast.ASTHandle{falsehood, truth, aSign}
	b := []ast.ASTHandle{truth, falsehood, bSign}
	operations := []struct {
		name string
		call func([]ast.ASTHandle, []ast.ASTHandle) ([]ast.ASTRef, error)
	}{
		{"sdiv", engine.MkSignedDiv},
		{"srem", engine.MkSignedRem},
		{"smod", engine.MkSignedMod},
	}
	for _, operation := range operations {
		refs, err := operation.call(a, b)
		paramsMust(t, err)
		foundA, foundB := false, false
		for _, h := range arithmeticHandles(t, engine, refs) {
			foundA = foundA || containsHandle(t, m, h, aSign, make(map[ast.ASTHandle]bool))
			foundB = foundB || containsHandle(t, m, h, bSign, make(map[ast.ASTHandle]bool))
		}
		if !foundA || !foundB {
			t.Fatalf("%s lost symbolic signs: a=%v b=%v", operation.name, foundA, foundB)
		}
		arithmeticRelease(t, m, refs)
	}
}
