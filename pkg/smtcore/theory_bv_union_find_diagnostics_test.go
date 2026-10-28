package smtcore

// Source: src/smt/theory_bv.cpp:1416,1447,1463-1499.

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestSMTBVZeroOneMergeSourceAssertionSites(t *testing.T) {
	for _, mode := range []string{"width", "aux", "complement", "literal-width"} {
		t.Run(mode, func(t *testing.T) {
			_, report := newLRASourceAssertionContext(t)
			m, state, theory := bvOccurrenceFixture(t)
			truth, err := m.MkTrue()
			astTestMust(t, err)
			falsity, err := m.MkFalse()
			astTestMust(t, err)
			_, root := bvNativeWord(t, m, state, theory, truth)
			var child SMTTheoryVar
			line := 0
			condition := ""
			switch mode {
			case "width":
				_, child = bvNativeWord(t, m, state, theory, truth, truth)
				line, condition = 1463, "bv_size == get_bv_size(r2)"
			case "aux":
				child = root
				theory.mergeAux[0] = []SMTTheoryVar{root, SMTNullTheoryVar}
				theory.mergeAux[1] = []SMTTheoryVar{child, SMTNullTheoryVar}
				line, condition = 1472, "m_merge_aux[0][i] == null_theory_var || m_merge_aux[1][i] == null_theory_var"
			case "complement":
				_, child = bvNativeWord(t, m, state, theory, falsity)
				theory.bitLiterals[child][0] = SMTTrueLiteral
				line, condition = 1486, "m_bits[v1][it->m_idx] == ~(m_bits[v2][it->m_idx])"
			case "literal-width":
				_, child = bvNativeWord(t, m, state, theory, falsity)
				theory.bitLiterals[child] = append(theory.bitLiterals[child], SMTFalseLiteral)
				line, condition = 1487, "m_bits[v1].size() == m_bits[v2].size()"
			}
			if mode == "aux" {
				err = theory.assertBVMergeAux(1472, 1)
			} else {
				_, err = theory.mergeZeroOneBits(root, child)
			}
			bvInvariantFailure(t, err, report, line, condition)
		})
	}
}

func TestSMTBVMergeComplementConditionalSourceTrace(t *testing.T) {
	_, _ = newLRASourceAssertionContext(t)
	var output bytes.Buffer
	m, state, theory := bvOccurrenceFixture(t)
	p := smtRelevancyBool(t, m, "trace-complement-p")
	q := smtRelevancyBool(t, m, "trace-complement-q")
	_, left := bvNativeWord(t, m, state, theory, p)
	_, right := bvNativeWord(t, m, state, theory, q)
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("bv_bug")
	theory.trace = tags
	bit := theory.bitLiterals[left][0]
	astTestMust(t, theory.traceBVMergeComplement(left, right, 0, bit, bit.Negated()))
	if diagnostic.DebugBuild && tags.ShouldTrace("bv_bug") {
		if !strings.Contains(output.String(), "-------- [bv_bug] merge_eh src/smt/theory_bv.cpp:1416 ---------\n") || !strings.Contains(output.String(), "idx: 0\n") {
			t.Fatal("missing source conditional merge trace", output.String())
		}
	} else if output.Len() != 0 {
		t.Fatal("erased source conditional merge trace executed", output.String())
	}
}
