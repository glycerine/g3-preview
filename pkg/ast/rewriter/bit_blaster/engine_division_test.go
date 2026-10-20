package bit_blaster

import "testing"

func TestBitBlasterUnsignedDivisionExhaustiveConstants(t *testing.T) {
	m, engine := arithmeticEngine(t)
	for width := 1; width <= 5; width++ {
		limit := int64(1 << width)
		for dividend := int64(0); dividend < limit; dividend++ {
			dividendRefs, dividendBits := logicBits(t, engine, dividend, width)
			for divisor := int64(0); divisor < limit; divisor++ {
				divisorRefs, divisorBits := logicBits(t, engine, divisor, width)
				quotient, remainder, err := engine.MkUnsignedDivRem(dividendBits, divisorBits)
				paramsMust(t, err)
				wantQuotient, wantRemainder := limit-1, dividend
				if divisor != 0 {
					wantQuotient, wantRemainder = dividend/divisor, dividend%divisor
				}
				if got := arithmeticValue(t, engine, quotient); got != wantQuotient {
					t.Fatalf("width %d: %d udiv %d = %d, want %d", width, dividend, divisor, got, wantQuotient)
				}
				if got := arithmeticValue(t, engine, remainder); got != wantRemainder {
					t.Fatalf("width %d: %d urem %d = %d, want %d", width, dividend, divisor, got, wantRemainder)
				}
				arithmeticRelease(t, m, quotient)
				arithmeticRelease(t, m, remainder)
				arithmeticRelease(t, m, divisorRefs)
			}
			arithmeticRelease(t, m, dividendRefs)
		}
	}
}

func TestBitBlasterUnsignedDivisionWrappersOwnOnlyTheirResult(t *testing.T) {
	m, engine := arithmeticEngine(t)
	dividendRefs, dividendBits := logicBits(t, engine, 13, 4)
	divisorRefs, divisorBits := logicBits(t, engine, 3, 4)
	quotient, err := engine.MkUnsignedDiv(dividendBits, divisorBits)
	paramsMust(t, err)
	if got := arithmeticValue(t, engine, quotient); got != 4 {
		t.Fatal("udiv wrapper", got)
	}
	remainder, err := engine.MkUnsignedRem(dividendBits, divisorBits)
	paramsMust(t, err)
	if got := arithmeticValue(t, engine, remainder); got != 1 {
		t.Fatal("urem wrapper", got)
	}
	arithmeticRelease(t, m, remainder)
	arithmeticRelease(t, m, quotient)
	arithmeticRelease(t, m, divisorRefs)
	arithmeticRelease(t, m, dividendRefs)
}
