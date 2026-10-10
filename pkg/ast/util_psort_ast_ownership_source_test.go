// Source: src/util/sorting_network.h:190-260,553-564,597-1046; src/util/vector.h:174-189.
package ast

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func psortNamedGraph(t *testing.T, s *PsortASTNetwork, c *PsortASTContext, in, out *ASTLiteralVector, other uint64) {
	t.Helper()
	m := c.manager
	cold := other + m.Stats().AllocatedBytes + astTestSymbolGraph(m.Symbols) + m.Symbols.Stats().Table.RetainedBytes + s.header + c.header
	vectors := []*ASTLiteralVector{in}
	if out != nil {
		vectors = append(vectors, out)
	}
	for f := s.frames; f != nil; f = f.next {
		if f.owner != s || f.invocation == 0 || f.invocation > s.serial || f.cursor > uint32(len(f.vectors)) {
			t.Fatal("actual lexical invocation")
		}
		cold += f.header
		for i := range f.vectors {
			v := &f.vectors[i]
			if v.header != 0 || v.self != v {
				t.Fatal("inline vector header double charge/identity")
			}
			vectors = append(vectors, v)
		}
	}
	astLiteralNamedGraph(t, s.budget, s.runtime, cold, vectors)
}
func TestPsortASTNetworkEveryActualBoundReservationNamedBeforeFaultLift(t *testing.T) {
	for _, bits := range []uint8{32, 64} {
		for _, mode := range []int{0, 1, 2} {
			first := int64(-1)
			for ordinal := int64(0); ordinal < 1024; ordinal++ {
				m := astTestManager(t)
				c, e := NewPsortASTContext(m)
				astTestMust(t, e)
				r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: m.BudgetField, SizeTBits: bits})
				astTestMust(t, e)
				s, e := NewPsortASTNetwork(c, r, m.BudgetField, nil)
				astTestMust(t, e)
				in, e := NewASTLiteralVector(m.BudgetField)
				astTestMust(t, e)
				sort, e := m.MkBoolSort()
				astTestMust(t, e)
				for i := uint32(0); i < 4; i++ {
					h, e := m.MkFreshConst("a", sort)
					astTestMust(t, e)
					astTestPin(t, m, h)
					astTestMust(t, in.Push(h))
				}
				span, e := in.Span()
				astTestMust(t, e)
				other := m.BudgetField.Used() - m.Stats().AllocatedBytes - astTestSymbolGraph(m.Symbols) - m.Symbols.Stats().Table.RetainedBytes - r.Stats().TotalBytes - s.header - c.header - in.header
				m.BudgetField.FailAfter(ordinal)
				var result ASTHandle
				switch mode {
				case 0:
					result, e = s.Ge(true, 2, 4, span)
				case 1:
					result, e = s.Le(true, 2, 4, span)
				case 2:
					result, e = s.Eq(true, 2, 4, span)
				}
				cause := e
				psortNamedGraph(t, s, c, in, nil, other)
				sz, e := in.Size()
				astTestMust(t, e)
				if sz != 4 {
					t.Fatal("borrowed input logical size modified", ordinal, sz)
				}
				if cause != nil && !errors.Is(cause, arena.ErrInjectedFailure) {
					var failure umemory.Failure
					if !errors.As(cause, &failure) || failure.Kind != umemory.FailureMemout {
						t.Fatal("actual reservation cause", bits, mode, ordinal, cause)
					}
				}
				if cause == nil {
					first = ordinal
					if result == (ASTHandle{}) {
						t.Fatal("success lost actual result")
					}
				}
				// Owning source scopes are released only after their
				// immediate named graphs have been checked while refusal remains active.
				m.BudgetField.FailAfter(-1)
				astTestMust(t, s.Close())
				astTestMust(t, in.Close())
				astTestMust(t, r.Close())
				for attempt := 0; attempt < 3; attempt++ {
					e = c.Close()
					if e == nil {
						break
					}
					if attempt == 2 {
						t.Fatal("owning child failed to progress", e)
					}
				}
				if cause == nil {
					break
				}
			}
			if first < 0 {
				t.Fatal("actual first success not reached", bits, mode)
			}
			t.Log("every actual bound reservation", bits, mode, first)
		}
	}
}
