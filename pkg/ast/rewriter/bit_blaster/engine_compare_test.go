package bit_blaster

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func signedBitValue(value int64, width int) int64 {
	sign := int64(1 << (width - 1))
	if value&sign != 0 {
		return value - int64(1<<width)
	}
	return value
}

func TestBitBlasterComparisonsExhaustiveConstants(t *testing.T) {
	m, engine := arithmeticEngine(t)
	out, err := m.NewRef(ast.ASTHandle{})
	paramsMust(t, err)
	defer func() { paramsMust(t, m.DelRef(&out)) }()
	for width := 1; width <= 5; width++ {
		limit := int64(1 << width)
		for a := int64(0); a < limit; a++ {
			aRefs, aBits := logicBits(t, engine, a, width)
			for b := int64(0); b < limit; b++ {
				bRefs, bBits := logicBits(t, engine, b, width)
				paramsMust(t, engine.MkUnsignedLE(aBits, bBits, out))
				got, err := m.RefGet(out)
				paramsMust(t, err)
				if actual, want := m.IsTrue(got), a <= b; actual != want {
					t.Fatalf("width %d: unsigned %d <= %d is %v, want %v", width, a, b, actual, want)
				}
				paramsMust(t, engine.MkSignedLE(aBits, bBits, out))
				got, err = m.RefGet(out)
				paramsMust(t, err)
				signedA, signedB := signedBitValue(a, width), signedBitValue(b, width)
				if actual, want := m.IsTrue(got), signedA <= signedB; actual != want {
					t.Fatalf("width %d: signed %d <= %d is %v, want %v", width, signedA, signedB, actual, want)
				}
				arithmeticRelease(t, m, bRefs)
			}
			arithmeticRelease(t, m, aRefs)
		}
	}
}
