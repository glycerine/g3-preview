package smtcore

// Source: src/smt/theory_bv.cpp:77-156,217,244,280-313,529-537.

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestSMTBVConstructionSourceAssertionSites(t *testing.T) {
	t.Run("bit2bool-entry", func(t *testing.T) {
		_, report := newLRASourceAssertionContext(t)
		m, state, theory := bvOccurrenceFixture(t)
		p := smtRelevancyBool(t, m, "construction-bit2bool-p")
		word, _ := bvNativeWord(t, m, state, theory, p)
		projection, err := m.MkBitVectorBit2Bool(0, word.owner)
		astTestMust(t, err)
		astTestMust(t, state.Internalize(projection, true))
		_, err = theory.internalizeBitProjection(projection)
		bvInvariantFailure(t, err, report, 77, "!ctx.b_internalized(n)")
	})

	t.Run("numeral-width", func(t *testing.T) {
		_, report := newLRASourceAssertionContext(t)
		m, _, theory := bvOccurrenceFixture(t)
		numeral, err := m.BitVectorNumeralUint64(1, 2)
		astTestMust(t, err)
		truth, err := m.MkTrue()
		astTestMust(t, err)
		theory.bits[numeral] = []ast.ASTHandle{truth}
		_, err = theory.InternalizeTerm(numeral)
		bvInvariantFailure(t, err, report, 529, "bits.size() == sz")
	})

	t.Run("init-bits-width", func(t *testing.T) {
		_, report := newLRASourceAssertionContext(t)
		m, state, theory := bvOccurrenceFixture(t)
		left, _ := bvFixedSymbolic(t, m, state, theory, "construction-width-left", 2)
		right, _ := bvFixedSymbolic(t, m, state, theory, "construction-width-right", 2)
		sum, err := m.MkBitVectorAdd(left.owner, right.owner)
		astTestMust(t, err)
		truth, err := m.MkTrue()
		astTestMust(t, err)
		theory.bits[sum] = []ast.ASTHandle{truth}
		_, err = theory.InternalizeTerm(sum)
		bvInvariantFailure(t, err, report, 313, "get_bv_size(n) == sz")
	})
}
