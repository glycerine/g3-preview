package bit_blaster

import "testing"

func TestConcreteBitBlasterConstructorAndParameterSnapshot(t *testing.T) {
	m, _ := arithmeticEngine(t)
	params := SMTBitBlasterParams{ExtGates: true, Quantifiers: true}
	blaster, err := NewBitBlaster(m, params)
	paramsMust(t, err)
	if got := blaster.Params(); got != params {
		t.Fatal("parameter snapshot", got, params)
	}
	aRefs, a := logicBits(t, blaster.Engine, 3, 3)
	bRefs, b := logicBits(t, blaster.Engine, 5, 3)
	product, err := blaster.MkMultiplier(a, b)
	paramsMust(t, err)
	if got := arithmeticValue(t, blaster.Engine, product); got != 7 {
		t.Fatal("promoted concrete engine", got)
	}
	arithmeticRelease(t, m, product)
	arithmeticRelease(t, m, bRefs)
	arithmeticRelease(t, m, aRefs)
}
