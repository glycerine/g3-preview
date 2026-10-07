// Licensed under the MIT license.
package ast

import (
	"testing"
)

func TestASTProofTheoryRationalFailureOwnership(t *testing.T) {
	m := astTestManager(t)
	astTestMust(t, m.ToggleProofMode(ASTProofEnabled))
	s := m.numerals
	value, e := s.NewValue()
	astTestMust(t, e)
	astTestMust(t, s.SetString(&value, "123456789012345678901234567890123456789012345678901234567891/987654321098765432109876543210987654321098765432109876543211"))
	defer s.Del(&value)
	base, e := s.Stats()
	astTestMust(t, e)
	p, e := m.NewParameterRational(&value)
	astTestMust(t, e)
	defer m.DelParameter(&p)
	copied, e := s.Stats()
	astTestMust(t, e)
	if copied.LiveCells != base.LiveCells+2 || copied.LivePayloads != base.LivePayloads+2 {
		t.Fatal("fixture must own two large integer cells and payloads", base, copied)
	}
	f := proofTestBool(t, m, "theory-large-parameter")
	premise, e := m.MkProofHypothesis(f)
	astTestMust(t, e)
	// Input ownership plus every real info-owned parameter must account for all
	// live parameter records immediately, before rendering, deleting or closing.
	audit := func(after int64) {
		roots := map[ASTParameter]bool{p: true}
		infos := uint32(0)
		for i := uint32(1); i < m.nextInfo; i++ {
			r := m.infoAt(i)
			if !r.live {
				continue
			}
			infos++
			for j := uint32(0); j < r.count; j++ {
				param, err := m.infoParameter(r, j)
				astTestMust(t, err)
				if roots[param] {
					t.Fatal("duplicate owning parameter token", after, param)
				}
				roots[param] = true
			}
		}
		var live, big uint32
		for i := uint32(1); i < m.nextParameter; i++ {
			r := m.parameterAt(i)
			if !r.Live {
				continue
			}
			live++
			if !roots[ASTParameter{m.origin, i, r.generation}] {
				t.Fatal("unreachable temporary parameter", after, i)
			}
			if r.Kind == ASTParameterRational {
				big++
				size, err := s.StorageSize(&r.Rational)
				astTestMust(t, err)
				if size == 0 {
					t.Fatal("lost large rational payload", after, i)
				}
			}
		}
		stats, err := s.Stats()
		astTestMust(t, err)
		if infos != m.liveInfos || live != m.liveParameters || len(roots) != int(live) || stats.LiveCells != base.LiveCells+2*big || stats.LivePayloads != base.LivePayloads+2*big {
			t.Fatal("immediate owner graph mismatch", after, infos, m.liveInfos, live, m.liveParameters, stats, base, big)
		}
		if m.pendingDeletionCurrent != (astNodeRef{}) || m.pendingDeletionHead != (astNodeRef{}) {
			t.Fatal("unconsumed cleanup continuation", after)
		}
	}
	var failures int
	succeeded := false
	for after := int64(0); after < 128; after++ {
		m.BudgetField.FailAfter(after)
		out, err := m.MkProofTheoryLemma(2, f, []ASTHandle{premise}, []ASTParameter{p, p, p})
		audit(after)
		m.BudgetField.FailAfter(-1)
		if err == nil {
			proofTestFact(t, m, out, f)
			succeeded = true
			break
		}
		if !numericStorageReservationRefusal(err) {
			t.Fatal("unexpected failure", after, err)
		}
		failures++
	}
	if !succeeded || failures < 4 {
		t.Fatal("missing real parameter copy/factory failure boundaries", failures, succeeded)
	}
}
