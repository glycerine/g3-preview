package ast

// Source: src/util/sorting_network.h:125-147,167-180,593-595,968-978,1003-1013.

import (
	umemory "github.com/glycerine/g3/pkg/util/memory"
	"math"
	"testing"
)

func TestPsortASTCostLiteralWrapTieAndReset(t *testing.T) {
	a, b := psortCost{1, 0}, psortCost{0, 5}
	if psortCostMinimum(a, b) != b {
		t.Fatal("literal strict tie selects second cost")
	}
	huge := psortCost{math.MaxUint32, math.MaxUint32}
	if huge.value() != math.MaxUint32-5 || huge.plus(psortCost{2, 3}) != (psortCost{1, 2}) || huge.times(2) != (psortCost{math.MaxUint32 - 1, math.MaxUint32 - 1}) {
		t.Fatal("source intermediate unsigned wraps")
	}
	st := PsortASTStats{17, 23, 91}
	st.Reset()
	if st != (PsortASTStats{}) {
		t.Fatal("source memset statistics")
	}
}
func TestPsortASTCostActualBoundModeSelection(t *testing.T) {
	for mode := 0; mode < 5; mode++ {
		m := astTestManager(t)
		c, e := NewPsortASTContext(m)
		astTestMust(t, e)
		r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: m.BudgetField})
		astTestMust(t, e)
		s, e := NewPsortASTNetwork(c, r, m.BudgetField, nil)
		astTestMust(t, e)
		in, e := NewASTLiteralVector(m.BudgetField)
		astTestMust(t, e)
		sort, e := m.MkBoolSort()
		astTestMust(t, e)
		for i := 0; i < 2; i++ {
			h, e := m.MkFreshConst("a", sort)
			astTestMust(t, e)
			astTestPin(t, m, h)
			astTestMust(t, in.Push(h))
		}
		p, e := in.Span()
		astTestMust(t, e)
		switch mode {
		case 0:
			_, e = s.Le(false, 0, 1, p)
		case 1:
			_, e = s.Ge(false, 1, 2, p)
		case 2:
			_, e = s.Eq(false, 0, 1, p)
		case 3:
			_, e = s.Ge(true, 1, 2, p)
		case 4:
			_, e = s.Le(true, 0, 1, p)
		}
		astTestMust(t, e)
		cmp, e := s.costCompare()
		astTestMust(t, e)
		want := uint32(3)
		if mode == 2 {
			want = 6
		}
		if cmp != (psortCost{2, want}) {
			t.Fatal("source EQ-only comparator cost", mode, cmp)
		}
		if mode == 1 {
			// vc_card_rec calls vc_smerge(k,l,n-l) with (a,b,c),
			// unlike executable smerge(c,a,b). For k=1,n=5 this is
			// (1,2,3): costs (1,1)+(1,4)+(3,1), hence (5,6).
			card, err := s.costCardRecursive(1, 5)
			astTestMust(t, err)
			if card != (psortCost{5, 6}) {
				t.Fatal("source vc_smerge argument order in cardinality cost", card)
			}
		}
		d, e := s.costDirectSort(3, 4)
		astTestMust(t, e)
		clauses := uint32(8)
		if mode >= 2 {
			clauses = 16
		}
		if d != (psortCost{3, clauses}) {
			t.Fatal("source full modes enable both direct-sort terms", mode, d)
		}
		d, e = s.costDirectMerge(2, 3, 4)
		astTestMust(t, e)
		clauses = uint32(11)
		if mode == 0 {
			clauses = 8
		} else if mode == 1 {
			clauses = 3
		}
		if d != (psortCost{4, clauses}) {
			t.Fatal("literal integer multiplication then division", mode, d)
		}
		astTestMust(t, s.Close())
		astTestMust(t, in.Close())
		astTestMust(t, r.Close())
		astTestMust(t, c.Close())
	}
}
