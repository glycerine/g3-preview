// Source: src/util/vector.h:102-120,159-167,191-202.
package ast

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestASTLiteralVectorExactCopyCapacityAndRawTokens(t *testing.T) {
	m := astTestManager(t)
	sort, e := m.MkBoolSort()
	astTestMust(t, e)
	h, e := m.MkFreshConst("x", sort)
	astTestMust(t, e)
	astTestPin(t, m, h)
	r, e := util.NewRuntime(util.RuntimeConfig{Budget: m.BudgetField})
	astTestMust(t, e)
	original, e := NewRuntimeASTLiteralVector(r, m.BudgetField, nil)
	astTestMust(t, e)
	astTestMust(t, original.Resize(3, h))
	clone, e := original.Clone()
	astTestMust(t, e)
	cap, e := clone.Capacity()
	astTestMust(t, e)
	if cap != 3 {
		t.Fatal("exact source copy cap", cap)
	}
	if clone.raw.Block == original.raw.Block {
		t.Fatal("copy points at original array")
	}
	for i := uint32(0); i < 3; i++ {
		got, e := clone.At(i)
		astTestMust(t, e)
		if got != h {
			t.Fatal("source raw AST token", i)
		}
	}
	astTestMust(t, original.Reset())
	astTestMust(t, clone.Assign(original))
	cap, e = clone.Capacity()
	astTestMust(t, e)
	if cap != 3 {
		t.Fatal("non-null empty source retains cap", cap)
	}
	size, e := clone.Size()
	astTestMust(t, e)
	if size != 0 {
		t.Fatal("logical source empty", size)
	}
	astTestMust(t, clone.Close())
	astTestMust(t, original.Close())
	astTestMust(t, r.Close())
}

func TestASTLiteralVectorEveryCopyReservationAndUnfinishedOwner(t *testing.T) {
	for _, bits := range []uint8{32, 64} {
		success := int64(-1)
		for ordinal := int64(0); ordinal < 16; ordinal++ {
			m := astTestManager(t)
			sort, e := m.MkBoolSort()
			astTestMust(t, e)
			h, e := m.MkFreshConst("x", sort)
			astTestMust(t, e)
			astTestPin(t, m, h)
			r, e := util.NewRuntime(util.RuntimeConfig{Budget: m.BudgetField, SizeTBits: bits})
			astTestMust(t, e)
			original, e := NewRuntimeASTLiteralVector(r, m.BudgetField, nil)
			astTestMust(t, e)
			astTestMust(t, original.Resize(3, h))
			external := m.BudgetField.Used() - r.Stats().TotalBytes - original.header
			m.BudgetField.FailAfter(ordinal)
			copy, cause := original.Clone()
			owners := []*RuntimeASTLiteralVector{original}
			if copy != nil {
				owners = append(owners, copy)
			}
			astLiteralNamedGraph(t, m.BudgetField, r, external, owners)
			if cause != nil && copy != nil && copy.constructed {
				t.Fatal("failed source constructor completed", ordinal)
			}
			if cause == nil {
				success = ordinal
				cap, e := copy.Capacity()
				astTestMust(t, e)
				if cap != 3 {
					t.Fatal(cap)
				}
			}
			m.BudgetField.FailAfter(-1)
			if copy != nil {
				astTestMust(t, copy.Close())
			}
			astLiteralNamedGraph(t, m.BudgetField, r, external, []*RuntimeASTLiteralVector{original})
			astTestMust(t, original.Close())
			astTestMust(t, r.Close())
			if success >= 0 {
				t.Log("actual literal copy first success", bits, success)
				break
			}
		}
		if success < 0 {
			t.Fatal("copy horizon", bits)
		}
	}
}

func TestASTLiteralVectorAllocationCallbacksCannotRetireExactCopyOwner(t *testing.T) {
	m := astTestManager(t)
	sort, e := m.MkBoolSort()
	astTestMust(t, e)
	h, e := m.MkFreshConst("x", sort)
	astTestMust(t, e)
	astTestPin(t, m, h)
	var dst *RuntimeASTLiteralVector
	var closeCause, dtCause error
	r, e := util.NewRuntime(util.RuntimeConfig{Budget: m.BudgetField, ThreadLocal: true, Transport: func(f util.Failure) error { closeCause = dst.Close(); dtCause = dst.SourceFinalize(); return f }})
	astTestMust(t, e)
	src, e := NewRuntimeASTLiteralVector(r, m.BudgetField, nil)
	astTestMust(t, e)
	astTestMust(t, src.Push(h))
	dst, e = NewRuntimeASTLiteralVector(r, m.BudgetField, nil)
	astTestMust(t, e)
	external := m.BudgetField.Used() - r.Stats().TotalBytes - src.header - dst.header
	anchor := astUnsignedGrowthPrime(t, r)
	cause := dst.Assign(src)
	if cause == nil || closeCause != util.ErrRuntimeUnsignedGrowthExecuting || dtCause != util.ErrRuntimeUnsignedGrowthExecuting || dst.closed || dst.sourceDone {
		t.Fatal("real copying allocation callback", cause, closeCause, dtCause)
	}
	astLiteralNamedGraph(t, m.BudgetField, r, external, []*RuntimeASTLiteralVector{src, dst}, anchor)
	astTestMust(t, r.SetMaxSize(0))
	astTestMust(t, dst.Assign(src))
	astLiteralNamedGraph(t, m.BudgetField, r, external, []*RuntimeASTLiteralVector{src, dst}, anchor)
	got, e := dst.At(0)
	astTestMust(t, e)
	if got != h {
		t.Fatal("NULL assignment retry")
	}
	astTestMust(t, dst.SourceFinalize())
	astLiteralNamedGraph(t, m.BudgetField, r, external, []*RuntimeASTLiteralVector{src, dst}, anchor)
	astTestMust(t, dst.Close())
	astTestMust(t, src.Close())
	astTestMust(t, r.Deallocate(&anchor))
	astTestMust(t, r.Close())
}

func TestASTLiteralVectorEveryDestructiveAssignReservationAndDanglingDestructorGuard(t *testing.T) {
	for _, bits := range []uint8{32, 64} {
		success := int64(-1)
		for ordinal := int64(0); ordinal < 16; ordinal++ {
			m := astTestManager(t)
			sort, e := m.MkBoolSort()
			astTestMust(t, e)
			h, e := m.MkFreshConst("x", sort)
			astTestMust(t, e)
			astTestPin(t, m, h)
			r, e := util.NewRuntime(util.RuntimeConfig{Budget: m.BudgetField, SizeTBits: bits})
			astTestMust(t, e)
			src, e := NewRuntimeASTLiteralVector(r, m.BudgetField, nil)
			astTestMust(t, e)
			dst, e := NewRuntimeASTLiteralVector(r, m.BudgetField, nil)
			astTestMust(t, e)
			astTestMust(t, src.Resize(3, h))
			astTestMust(t, dst.Push(h))
			old := dst.raw.Block
			external := m.BudgetField.Used() - r.Stats().TotalBytes - src.header - dst.header
			m.BudgetField.FailAfter(ordinal)
			cause := dst.Assign(src)
			astLiteralNamedGraph(t, m.BudgetField, r, external, []*RuntimeASTLiteralVector{src, dst})
			if _, e = r.Size(old); e == nil {
				t.Fatal("source destroy must precede every copy reservation", ordinal)
			}
			if cause != nil {
				if !dst.assignmentDestroyed || dst.raw.Block != old || dst.assignmentCause != cause {
					t.Fatal("literal dangling source pointer and first body cause", ordinal, cause)
				}
				dt := dst.SourceFinalize()
				if dt == nil || dst.destructorCause != dt || dst.assignmentCause != cause {
					t.Fatal("reached dangling DT is guarded undefined source domain", dt)
				}
				before := r.Stats()
				if dst.SourceFinalize() != dt || r.Stats() != before {
					t.Fatal("source stopped DT replay")
				}
				astLiteralNamedGraph(t, m.BudgetField, r, external, []*RuntimeASTLiteralVector{src, dst})
			} else {
				success = ordinal
				cap, e := dst.Capacity()
				astTestMust(t, e)
				if cap != 3 {
					t.Fatal("exact source capacity", cap)
				}
			}
			m.BudgetField.FailAfter(-1)
			if cause != nil {
				astTestMust(t, dst.CleanupStopped())
			} else {
				astTestMust(t, dst.Close())
			}
			astLiteralNamedGraph(t, m.BudgetField, r, external, []*RuntimeASTLiteralVector{src})
			astTestMust(t, src.Close())
			astTestMust(t, r.Close())
			if success >= 0 {
				break
			}
		}
		if success < 0 {
			t.Fatal("actual assignment first success")
		}
		t.Log("destructive literal assignment horizon", bits, success)
	}
}
