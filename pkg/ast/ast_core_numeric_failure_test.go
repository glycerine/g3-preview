package ast

import (
	"testing"
)

func TestASTRationalInfoFactoryFailureHasImmediateRootAccounting(t *testing.T) {
	m := astTestManager(t)
	s := m.numerals
	numeral, e := s.NewValue()
	astTestMust(t, e)
	astTestMust(t, s.SetString(&numeral, "123456789012345678901234567890123456789012345678901234567890/7"))
	defer s.Del(&numeral)
	p, e := m.NewParameterRational(&numeral)
	astTestMust(t, e)
	defer m.DelParameter(&p)
	info, e := m.NewSortInfo(4, 10, []ASTParameter{p}, InfiniteSortSize(), false)
	astTestMust(t, e)
	defer m.DelSortInfo(&info)
	expected, e := s.Stats()
	astTestMust(t, e)
	baseline := m.liveNodes
	name := astTestSymbol(t, m, "BudgetedRationalSort")
	succeeded := false
	var failures int
	for after := int64(0); after < 64; after++ {
		m.BudgetField.FailAfter(after)
		h, e := m.MkSort(name, info)
		if e == nil {
			m.BudgetField.FailAfter(-1)
			if m.liveNodes != baseline+1 {
				t.Fatal("unique source sort missing")
			}
			ref, e := m.NewRef(h)
			astTestMust(t, e)
			astTestMust(t, m.DelRef(&ref))
			succeeded = true
			break
		}
		failures++
		stats, e := s.Stats()
		astTestMust(t, e)
		if stats.LiveCells != expected.LiveCells || stats.LivePayloads != expected.LivePayloads || m.liveNodes != baseline {
			t.Fatal("factory failed prefix retained temporary numeric/node ownership", after, stats, expected, m.liveNodes, baseline)
		}
		if m.pendingDeletionCurrent != (astNodeRef{}) || m.pendingDeletionHead != (astNodeRef{}) {
			t.Fatal("lost cleanup continuation")
		}
	}
	m.BudgetField.FailAfter(-1)
	if !succeeded || failures == 0 {
		t.Fatal("did not traverse exact factory allocation boundaries", failures)
	}
}
