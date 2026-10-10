// Source: src/ast/bv_decl_plugin.cpp:26-178; src/ast/bv_decl_plugin.h:24-104,143-208.
package ast

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestASTBitVectorSourceEagerSortsAndPinnedDeclarations(t *testing.T) {
	m := astTestManager(t)
	arith, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	id, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	if id == arith {
		t.Fatal("separate source family")
	}
	for bits := uint32(1); bits <= 64; bits++ {
		h, e := m.BitVectorSort(bits)
		astTestMust(t, e)
		r, e := m.internal(h)
		astTestMust(t, e)
		info, e := m.sortInfo(m.sortAt(r.slot).info)
		astTestMust(t, e)
		if info.family != id || info.kind != 0 || info.count != 1 {
			t.Fatal("actual sort_info", bits, *info)
		}
		param, e := m.infoParameter(info, 0)
		astTestMust(t, e)
		value, e := m.parameter(param)
		astTestMust(t, e)
		if value.Kind != ASTParameterInt || uint32(value.integer) != bits {
			t.Fatal("source width parameter", bits, *value)
		}
		if bits < 64 {
			size, e := info.size.Size()
			astTestMust(t, e)
			if size != uint64(1)<<bits {
				t.Fatal(bits, size)
			}
		} else if !info.size.IsVeryBig() {
			t.Fatal("source base2 cutoff", info.size)
		}
	}
}

func TestASTBitVectorSourcePinnedKindsAndEveryASTReservation(t *testing.T) {
	sy, q, _ := astTestOwners(t)
	success := false
	for ordinal := int64(0); ordinal < 256; ordinal++ {
		b := arena.NewBudget(1 << 30)
		m, e := NewASTManager(ASTManagerConfig{Symbols: sy, Numerals: q, Budget: b})
		astTestMust(t, e)
		_, e = m.RegisterArithmeticPlugin()
		astTestMust(t, e)
		b.FailAfter(ordinal)
		_, call := m.RegisterBitVectorPlugin()
		if b.Used() != m.Stats().AllocatedBytes {
			t.Fatal("named actual wrapper/cache/AST graph before lift", ordinal, b.Used(), m.Stats())
		}
		if call != nil && !errors.Is(call, arena.ErrInjectedFailure) {
			t.Fatal(ordinal, call)
		}
		if call == nil {
			success = true
			p := m.bitVectorPlugin
			for j, kind := range []int32{2, 1, 59, 60} {
				d := m.declAt(p.fields[j].slot)
				info, e := m.funcInfo(d.info)
				astTestMust(t, e)
				if info.family != p.family || info.kind != kind {
					t.Fatal("literal pinned declaration kind", j, info.kind, kind)
				}
			}
		}
		b.FailAfter(-1)
		astTestMust(t, m.Close())
		if b.Used() != 0 {
			t.Fatal("exact owning context end", ordinal, b.Used())
		}
		if success {
			t.Log("first AST reservation success", ordinal)
			break
		}
	}
	if !success {
		t.Fatal("missing completed source prefix")
	}
}

// Reusing an actual hash-consed sort is source-defined. The maximum reference
// counter seed is a Go/source DEBUG precondition fixture, not an ordinary
// allocation-refusal recipe or source algorithm failure.
func TestASTBitVectorReusedSortActualINCAcquisitionGuard(t *testing.T) {
	sy, q, _ := astTestOwners(t)
	b := arena.NewBudget(1 << 30)
	m, e := NewASTManager(ASTManagerConfig{Symbols: sy, Numerals: q, Budget: b})
	astTestMust(t, e)
	t.Cleanup(func() {
		astTestMust(t, m.Close())
		if b.Used() != 0 {
			t.Fatal("remaining actual AST owner", b.Used())
		}
	})
	_, e = m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	name, e := m.Symbols.Intern("bv")
	astTestMust(t, e)
	family, e := m.families.MkID(name)
	astTestMust(t, e)
	param, e := m.NewParameterUint(1)
	astTestMust(t, e)
	info, e := m.NewSortInfo(family, 0, []ASTParameter{param}, FiniteSortSize(2), false)
	astTestMust(t, e)
	sort, e := m.MkSort(name, info)
	astTestMust(t, e)
	astTestMust(t, m.DelSortInfo(&info))
	astTestMust(t, m.DelParameter(&param))
	ref, e := m.NewRef(sort)
	astTestMust(t, e)
	node, e := m.internal(sort)
	astTestMust(t, e)
	original := m.nodeAt(node.slot).rc
	if ASTSourceDebug {
		m.nodeAt(node.slot).rc = ^uint32(0)
	}
	_, call := m.RegisterBitVectorPlugin()
	p := m.bitVectorPlugin
	if p == nil {
		t.Fatal("actual registration publication absent", call)
	}
	captured := p.sorts[1]
	if captured != node {
		t.Fatal("actual reused sort identity", captured, node)
	}
	if ASTSourceDebug {
		if call != ErrASTDomain || p.sortAcquired[1] != 0 || m.nodeAt(node.slot).rc != ^uint32(0) {
			t.Fatal("published pointer before rejected INC", call, p.sortAcquired, m.nodeAt(node.slot).rc)
		}
		if m.BudgetField.Used() != m.Stats().AllocatedBytes {
			t.Fatal("actual retained graph", m.Stats())
		}
		astTestMust(t, m.finalizeBitVectorPlugin())
		if m.nodeAt(node.slot).rc != ^uint32(0) || p.sortAcquired[1] != 0 {
			t.Fatal("own abort decremented foreign reference")
		}
		m.nodeAt(node.slot).rc = original
	} else {
		astTestMust(t, call)
		if p.sortAcquired[1] != 1 || m.nodeAt(node.slot).rc != original+3 {
			t.Fatal("actual successful cache INC plus bit0/bit1 range pins", original, m.nodeAt(node.slot).rc)
		}
	}
	astTestMust(t, m.DelRef(&ref))
}
