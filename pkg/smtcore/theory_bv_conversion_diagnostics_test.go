package smtcore

// Source: src/smt/theory_bv.cpp:518,555,571-578,623-645.

import (
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestSMTBVConversionSourceAssertionSites(t *testing.T) {
	if !diagnostic.DebugBuild {
		return
	}
	for _, mode := range []string{"bv2int-enode", "bv2int-kind", "int2bv-enode", "int2bv-kind"} {
		t.Run(mode, func(t *testing.T) {
			_, report := newLRASourceAssertionContext(t)
			m, solver, theory := bvConversionFixture(t, 0)
			state := solver.kernel.internalizer
			zero := bvConversionInt(t, m, "0")
			intSort, err := m.GetSort(zero)
			astTestMust(t, err)
			x, err := m.MkConst(astTestSymbol(t, m, "conversion-diagnostic-x"), intSort)
			astTestMust(t, err)
			int2bv, err := m.MkIntToBitVector(2, x)
			astTestMust(t, err)
			bvSort, err := m.BitVectorSort(2)
			astTestMust(t, err)
			word, err := m.MkConst(astTestSymbol(t, m, "conversion-diagnostic-w"), bvSort)
			astTestMust(t, err)
			bv2int, err := m.MkBitVectorToInt(word)
			astTestMust(t, err)
			line := 0
			condition := ""
			switch mode {
			case "bv2int-enode":
				err = theory.assertBVToIntAxiom(bv2int)
				line, condition = 571, "get_context().e_internalized(n)"
			case "bv2int-kind":
				astTestMust(t, state.Internalize(int2bv, false))
				err = theory.assertBVToIntAxiom(int2bv)
				line, condition = 572, "m_util.is_bv2int(n)"
			case "int2bv-enode":
				err = theory.assertIntToBVAxiom(int2bv)
				line, condition = 644, "get_context().e_internalized(n)"
			case "int2bv-kind":
				astTestMust(t, state.Internalize(bv2int, false))
				err = theory.assertIntToBVAxiom(bv2int)
				line, condition = 645, "m_util.is_int2bv(n)"
			}
			bvInvariantFailure(t, err, report, line, condition)
		})
	}
}
