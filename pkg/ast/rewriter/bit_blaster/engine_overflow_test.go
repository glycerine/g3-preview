package bit_blaster

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestBitBlasterMultiplicationBoundsExhaustiveConstants(t *testing.T) {
	m, engine := arithmeticEngine(t)
	out, err := m.NewRef(ast.ASTHandle{})
	paramsMust(t, err)
	defer func() { paramsMust(t, m.DelRef(&out)) }()
	for width := 1; width <= 5; width++ {
		limit := int64(1 << width)
		maximum := int64(1<<(width-1)) - 1
		minimum := -int64(1 << (width - 1))
		for rawA := int64(0); rawA < limit; rawA++ {
			aRefs, aBits := logicBits(t, engine, rawA, width)
			a := signedBitValue(rawA, width)
			for rawB := int64(0); rawB < limit; rawB++ {
				bRefs, bBits := logicBits(t, engine, rawB, width)
				b := signedBitValue(rawB, width)
				paramsMust(t, engine.MkUnsignedMulNoOverflow(aBits, bBits, out))
				h, err := m.RefGet(out)
				paramsMust(t, err)
				if got, want := m.IsTrue(h), rawA*rawB < limit; got != want {
					t.Fatalf("width %d: unsigned-safe(%d*%d)=%v, want %v", width, rawA, rawB, got, want)
				}
				paramsMust(t, engine.MkSignedMulNoOverflow(aBits, bBits, out))
				h, err = m.RefGet(out)
				paramsMust(t, err)
				if got, want := m.IsTrue(h), a*b <= maximum; got != want {
					t.Fatalf("width %d: no-overflow(%d*%d)=%v, want %v", width, a, b, got, want)
				}
				paramsMust(t, engine.MkSignedMulNoUnderflow(aBits, bBits, out))
				h, err = m.RefGet(out)
				paramsMust(t, err)
				if got, want := m.IsTrue(h), a*b >= minimum; got != want {
					t.Fatalf("width %d: no-underflow(%d*%d)=%v, want %v", width, a, b, got, want)
				}
				arithmeticRelease(t, m, bRefs)
			}
			arithmeticRelease(t, m, aRefs)
		}
	}
}
