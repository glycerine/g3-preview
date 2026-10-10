// Source: src/util/sorting_network.h:190-260,531-594,645-653,749-777.
package ast

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestPsortASTNetworkUninitializedModeReachedAfterActualComparatorPrefix(t *testing.T) {
	m := astTestManager(t)
	c, e := NewPsortASTContext(m)
	astTestMust(t, e)
	r, e := util.NewRuntime(util.RuntimeConfig{Budget: m.BudgetField})
	astTestMust(t, e)
	s, e := NewPsortASTNetwork(c, r, m.BudgetField, nil)
	astTestMust(t, e)
	in, e := NewASTLiteralVector(m.BudgetField)
	astTestMust(t, e)
	out, e := NewASTLiteralVector(m.BudgetField)
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
	astTestMust(t, s.Sorting(0, ASTLiteralVectorSpan{}, out))
	astTestMust(t, s.Sorting(1, p, out))
	astTestMust(t, out.Reset())
	cause := s.Sorting(2, p, out)
	if !errors.Is(cause, ErrPsortUninitializedComparison) || s.modeInitialized {
		t.Fatal("actual uninitialized source mode access", cause)
	}
	sz, e := out.Size()
	astTestMust(t, e)
	trail := len(c.trail)
	st, e := s.Stats()
	astTestMust(t, e)
	if sz != 2 || trail != 2 || st.CompiledVariables != 2 || st.CompiledClauses != 0 {
		t.Fatal("max/min then output publication BEFORE cmp", sz, trail, st)
	}
	// Only an actual bound method assignment initializes m_t. A later direct
	// sorting call uses that retained mode, without clearing the old output.
	_, e = s.Eq(true, 0, 2, p)
	astTestMust(t, e)
	old, e := out.Size()
	astTestMust(t, e)
	astTestMust(t, s.Sorting(2, p, out))
	sz, e = out.Size()
	astTestMust(t, e)
	if sz != old+2 {
		t.Fatal("source direct sorting appends", old, sz)
	}
	astTestMust(t, s.Close())
	astTestMust(t, in.Close())
	astTestMust(t, out.Close())
	astTestMust(t, r.Close())
	astTestMust(t, c.Close())
}

func TestPsortASTNetworkNativeRangeFailureUnwindsFrames(t *testing.T) {
	m := astTestManager(t)
	c, e := NewPsortASTContext(m)
	astTestMust(t, e)
	r, e := util.NewRuntime(util.RuntimeConfig{Budget: m.BudgetField})
	astTestMust(t, e)
	s, e := NewPsortASTNetwork(c, r, m.BudgetField, nil)
	astTestMust(t, e)
	in, e := NewASTLiteralVector(m.BudgetField)
	astTestMust(t, e)
	sort, e := m.MkBoolSort()
	astTestMust(t, e)
	x, e := m.MkFreshConst("a", sort)
	astTestMust(t, e)
	astTestPin(t, m, x)
	astTestMust(t, in.Push(x))
	span, e := in.Span()
	astTestMust(t, e)
	other := m.BudgetField.Used() - m.Stats().AllocatedBytes - astTestSymbolGraph(m.Symbols) - m.Symbols.Stats().Table.RetainedBytes - r.Stats().TotalBytes - s.header - c.header - in.header
	_, cause := s.Ge(true, 2, 4, span)
	if !errors.Is(cause, arena.ErrBounds) || s.frames != nil {
		t.Fatal("short native input must unwind recursive frame storage", cause, s.frames)
	}
	psortNamedGraph(t, s, c, in, nil, other)
	astTestMust(t, s.Close())
	astTestMust(t, in.Close())
	astTestMust(t, r.Close())
	astTestMust(t, c.Close())
}
