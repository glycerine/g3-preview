// Source: src/util/sorting_network.h:190-260,531-594,645-653,749-777.
package ast

import (
	"errors"
	"testing"

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
	in, e := NewRuntimeASTLiteralVector(r, m.BudgetField, nil)
	astTestMust(t, e)
	out, e := NewRuntimeASTLiteralVector(r, m.BudgetField, nil)
	astTestMust(t, e)
	sort, e := m.MkBoolSort()
	astTestMust(t, e)
	for i := 0; i < 2; i++ {
		h, e := m.MkFreshConst("a", sort)
		astTestMust(t, e)
		astTestPin(t, m, h)
		astTestMust(t, in.Push(h))
	}
	p, e := in.CPtr()
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
	trail, e := m.RefVectorSize(c.trail)
	astTestMust(t, e)
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
func TestPsortASTNetworkRealErrorTransportCannotRetireExecutingLexicalOwner(t *testing.T) {
	for _, bits := range []uint8{32, 64} {
		m := astTestManager(t)
		c, e := NewPsortASTContext(m)
		astTestMust(t, e)
		var s *PsortASTNetwork
		var closeCause, sourceCause error
		var frame *psortASTVectorFrame
		r, e := util.NewRuntime(util.RuntimeConfig{Budget: m.BudgetField, ThreadLocal: true, SizeTBits: bits, Transport: func(f util.Failure) error {
			frame = s.frames
			closeCause = s.Close()
			sourceCause = s.SourceFinalize()
			return f
		}})
		astTestMust(t, e)
		s, e = NewPsortASTNetwork(c, r, m.BudgetField, nil)
		astTestMust(t, e)
		in, e := NewRuntimeASTLiteralVector(r, m.BudgetField, nil)
		astTestMust(t, e)
		sort, e := m.MkBoolSort()
		astTestMust(t, e)
		for i := 0; i < 4; i++ {
			h, e := m.MkFreshConst("a", sort)
			astTestMust(t, e)
			astTestPin(t, m, h)
			astTestMust(t, in.Push(h))
		}
		p, e := in.CPtr()
		astTestMust(t, e)
		anchor := astUnsignedGrowthPrime(t, r)
		other := m.BudgetField.Used() - m.Stats().AllocatedBytes - astTestSymbolGraph(m.Symbols) - r.Stats().TotalBytes - s.header - c.header - in.header
		astTestMust(t, r.SetMaxSize(1))
		_, cause := s.Le(true, 1, 4, p)
		var f util.Failure
		if !errors.As(cause, &f) || f.Kind != util.FailureMemout || !errors.Is(closeCause, ErrASTPending) || !errors.Is(sourceCause, ErrASTPending) || frame == nil || frame.owner != s {
			t.Fatal("actual postphysical callback owner guard", bits, cause, closeCause)
		}
		if frame.cursor != 0 || frame.vectors[0].constructed || frame.vectors[0].sourceDone || frame.vectors[0].closed || frame.vectors[1].constructed || s.stats.CompiledVariables != 0 {
			t.Fatal("incomplete range ctor must not run vector DT or later fresh", frame.cursor, frame.vectors[0].constructed, frame.vectors[0].sourceDone, frame.vectors[1].constructed, s.stats)
		}
		// The incomplete constructor retains its actual returned raw allocation.
		cold := other + m.Stats().AllocatedBytes + astTestSymbolGraph(m.Symbols) + s.header + c.header
		vectors := []*RuntimeASTLiteralVector{in}
		for f := s.frames; f != nil; f = f.next {
			cold += f.header
			for i := range f.vectors {
				vectors = append(vectors, &f.vectors[i])
			}
		}
		for f := c.pending; f != nil; f = f.next {
			cold += f.header
		}
		astLiteralNamedGraph(t, m.BudgetField, r, cold, vectors, anchor)
		astTestMust(t, s.Close())
		if !frame.vectors[0].closed {
			t.Fatal("owning abort did not close failed constructor")
		}
		astTestMust(t, r.SetMaxSize(0))
		s, e = NewPsortASTNetwork(c, r, m.BudgetField, nil)
		astTestMust(t, e)
		_, e = s.Le(true, 1, 4, p)
		astTestMust(t, e)
		astTestMust(t, s.Close())
		astTestMust(t, in.Close())
		astTestMust(t, r.Deallocate(&anchor))
		astTestMust(t, r.Close())
		astTestMust(t, c.Close())
	}
}

// Actual destructive Assign failure supplies a reached younger vector DT stop;
// this is a guarded dangling-pointer domain, not native valid destructor credit.
func TestPsortASTNetworkYoungerStoppedDestructorLeavesOuterUnread(t *testing.T) {
	m := astTestManager(t)
	c, e := NewPsortASTContext(m)
	astTestMust(t, e)
	r, e := util.NewRuntime(util.RuntimeConfig{Budget: m.BudgetField})
	astTestMust(t, e)
	s, e := NewPsortASTNetwork(c, r, m.BudgetField, nil)
	astTestMust(t, e)
	outer, e := s.beginFrame(1)
	astTestMust(t, e)
	astTestMust(t, s.defaultVector(outer, 0))
	astTestMust(t, outer.vectors[0].Push(ASTHandle{}))
	old := outer.vectors[0].raw.Block
	younger, e := s.beginFrame(1)
	astTestMust(t, e)
	astTestMust(t, s.defaultVector(younger, 0))
	astTestMust(t, younger.vectors[0].Push(ASTHandle{}))
	src, e := NewRuntimeASTLiteralVector(r, m.BudgetField, nil)
	astTestMust(t, e)
	astTestMust(t, src.Push(ASTHandle{}))
	other := m.BudgetField.Used() - m.Stats().AllocatedBytes - astTestSymbolGraph(m.Symbols) - m.Symbols.Stats().Table.RetainedBytes - r.Stats().TotalBytes - s.header - c.header - src.header - outer.header - younger.header
	m.BudgetField.FailAfter(0)
	body := younger.vectors[0].Assign(src)
	if body == nil {
		t.Fatal("actual copy reservation should refuse")
	}
	cause := s.finishFrame(younger, body)
	if younger.destructorCause == nil {
		t.Fatal("actual dangling pointer DT guard", cause)
	}
	outerCause := s.finishFrame(outer, cause)
	if !errors.Is(outerCause, younger.destructorCause) || outer.cursor != 1 || outer.sourceDone || outer.vectors[0].sourceDone || outer.vectors[0].raw.Block != old {
		t.Fatal("older source member read/destruction after younger stop", outerCause, outer.cursor)
	}
	if _, e = r.Size(old); e != nil {
		t.Fatal("older physical payload consumed", e)
	}
	psortNamedGraph(t, s, c, src, nil, other)
	m.BudgetField.FailAfter(-1)
	astTestMust(t, s.Close())
	astTestMust(t, src.Close())
	astTestMust(t, r.Close())
	astTestMust(t, c.Close())
}
