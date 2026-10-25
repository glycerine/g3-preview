package smtcore

import (
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/util/lbool"
)

func smtPBTestValue(l SMTLiteral, bits uint32) bool {
	if l == SMTTrueLiteral {
		return true
	}
	if l == SMTFalseLiteral {
		return false
	}
	v := bits&(1<<uint(l.Var()-1)) != 0
	if l.Sign() {
		return !v
	}
	return v
}

func smtPBTestHolds(a *smtPBArgs, bits uint32, equality bool) bool {
	sum := new(big.Rat)
	for _, term := range a.terms {
		if smtPBTestValue(term.literal, bits) {
			sum.Add(sum, &term.coeff)
		}
	}
	if equality {
		return sum.Cmp(&a.bound) == 0
	}
	return sum.Cmp(&a.bound) >= 0
}

func TestSMTPBLiteralNormalizationIndependentValuationOracle(t *testing.T) {
	a, b, c := NewSMTLiteral(1, false), NewSMTLiteral(2, false), NewSMTLiteral(3, false)
	for _, eq := range []bool{false, true} {
		for _, test := range []struct {
			literals     []SMTLiteral
			coefficients []string
		}{
			{[]SMTLiteral{a, b, c}, []string{"4", "3", "2"}},
			{[]SMTLiteral{a, a.Negated(), b, SMTTrueLiteral, SMTFalseLiteral, a, c}, []string{"2", "3", "-4", "1", "10", "0", "1"}},
			{[]SMTLiteral{a, b, c.Negated()}, []string{"1/2", "3/4", "-5/6"}},
			{[]SMTLiteral{a, a.Negated(), b}, []string{"2", "2", "0"}},
		} {
			for k := int64(-5); k <= 12; k++ {
				var original smtPBArgs
				original.bound.SetFrac64(k, 2)
				for i, l := range test.literals {
					q, ok := new(big.Rat).SetString(test.coefficients[i])
					if !ok {
						t.Fatal("bad fixture rational")
					}
					original.terms = append(original.terms, smtPBTerm{l, *q})
				}
				normal := original.copy()
				normal.unique()
				truth := normal.normalize(eq)
				normal.prune(eq)
				for bits := uint32(0); bits < 8; bits++ {
					want := smtPBTestHolds(&original, bits, eq)
					got := smtPBTestHolds(&normal, bits, eq)
					if truth == lbool.True {
						got = true
					}
					if truth == lbool.False {
						got = false
					}
					if got != want {
						t.Fatalf("normalization changed valuation: eq=%v coeff=%v bound=%v bits=%d truth=%v", eq, test.coefficients, &original.bound, bits, truth)
					}
				}
				if !eq && truth == lbool.Undef && len(normal.terms) > 0 {
					complement := normal.copy()
					astTestMust(t, complement.negate())
					for bits := uint32(0); bits < 8; bits++ {
						if smtPBTestHolds(&complement, bits, false) == smtPBTestHolds(&original, bits, false) {
							t.Fatal("strict integer complement lost polarity")
						}
					}
				}
			}
		}
	}
}
