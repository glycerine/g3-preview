// Copyright (c) 2011 Microsoft Corporation. Licensed under the MIT license.
// Source mpn.cpp:371 and buffer.h:84-95; independent bounded lifetime oracle.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestNaturalFormattingLexicalSourceLocals(t *testing.T) {
	b := arena.NewBudget(1 << 22)
	m, e := NewNaturalManager(arena.VectorConfig{Descriptors: arena.Config{Budget: b}})
	if e != nil {
		t.Fatal(e)
	}
	before := m.runtime.Stats()
	if e = m.beginFormattingLocals(33); e != nil {
		t.Fatal(e)
	}
	if e = m.beginFormattingLocals(1); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("active formatting reuse guard", e)
	}
	for i, n := range []uint32{33, 34, 1} {
		s := &m.formatting.locals[i]
		if s.pos != n || s.capacity() != []uint32{64, 64, 16}[i] {
			t.Fatal(i, s.pos, s.capacity())
		}
		for j := uint32(0); j < n; j++ {
			v, e := s.read(m.runtime, j)
			if e != nil || v != 0 {
				t.Fatal(i, j, v, e)
			}
		}
	}
	if m.runtime.Stats().LiveBlocks != before.LiveBlocks+2 || m.words.Stats().Live != 0 || m.overflowPayloads() != 2 {
		t.Fatal("actual lexical roots")
	}
	if e = m.finishFormattingLocals(); e != nil {
		t.Fatal(e)
	}
	if m.runtime.Stats().LiveBlocks != before.LiveBlocks || m.formatting.active || m.formatting.pending {
		t.Fatal("completed reverse destruction")
	}
	m.Close()
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestNaturalFormattingFailedConstructorRetainsRealRoot(t *testing.T) {
	found := [2]bool{}
	completed := false
	for fault := int64(0); fault < 100; fault++ {
		b := arena.NewBudget(1 << 22)
		m, e := NewNaturalManager(arena.VectorConfig{Descriptors: arena.Config{Budget: b}})
		if e != nil {
			t.Fatal(e)
		}
		b.FailAfter(fault)
		e = m.beginFormattingLocals(65)
		if e != nil {
			if !m.formatting.pending || m.formatting.active {
				t.Fatal("retained failed constructor", fault)
			}
			// A refused later expansion leaves the failed object's old physical
			// allocation alive. Its destructor did not run; earlier completed
			// locals have actually unwound.
			s := &m.formatting.locals[m.formatting.failedLocal]
			if s.raw != (memory.Block{}) {
				found[m.formatting.failedLocal] = true
				if m.runtime.Stats().LiveBlocks == 0 {
					t.Fatal("failed local raw lost", fault)
				}
				if ordinary := m.beginFormattingLocals(1); ordinary == nil {
					t.Fatal("formatting reuses pending local ownership")
				}
			}

			if !numericPhysicalReservationRefusal(e) {
				t.Fatal("actual refusal category", fault, e)
			}
			if m.formatting.completed != 0 {
				t.Fatal("earlier source local destructor not completed", fault, m.formatting.completed)
			}
			roots := uint32(0)
			for i := range m.formatting.locals {
				local := &m.formatting.locals[i]
				if uint8(i) != m.formatting.failedLocal && (local.raw != (memory.Block{}) || local.pending != (memory.Block{})) {
					t.Fatal("unwound/never constructed root", fault, i)
				}
				if local.raw != (memory.Block{}) {
					roots++
				}
				if local.pending != (memory.Block{}) {
					roots++
				}
			}
			if roots != m.runtime.Stats().LiveBlocks {
				t.Fatal("immediate named root graph", fault, roots, m.runtime.Stats())
			}
			ws := m.words.Stats()
			owned := m.headerBytes + m.runtime.Stats().TotalBytes + ws.OwnerBytes + ws.PayloadBytes + ws.BlockMetadataBytes + ws.DirectoryBytes + ws.Descriptors.OwnerBytes + ws.Descriptors.RecordBytes + ws.Descriptors.MetadataBytes + ws.Descriptors.DirectoryBytes
			if owned != b.Used() {
				t.Fatal("immediate real charge graph", fault, owned, b.Used())
			}
		} else {
			completed = true
			if e = m.finishFormattingLocals(); e != nil {
				t.Fatal(e)
			}
		}
		count, counterError := m.runtime.AllocationCount()
		if counterError != nil {
			t.Fatal(counterError)
		}
		b.FailAfter(-1)
		if e = m.CleanupFormatting(); e != nil {
			t.Fatal(fault, e)
		}
		after, counterError := m.runtime.AllocationCount()
		if counterError != nil || after != count {
			t.Fatal("cleanup invented source allocation", count, after, counterError)
		}
		if m.runtime.Stats().LiveBlocks != 0 {
			t.Fatal("cleanup actual roots", fault)
		}
		m.Close()
		if b.Used() != 0 {
			t.Fatal("refund", fault, b.Used())
		}
		if found[0] && found[1] && completed {
			break
		}
	}
	if !found[0] || !found[1] || !completed {
		t.Fatal("no later expansion refusal exercised")
	}
}

func TestNaturalFormattingPendingLocalDoesNotBlockDivision(t *testing.T) {
	for fault := int64(0); fault < 50; fault++ {
		b := arena.NewBudget(1 << 22)
		m, e := NewNaturalManager(arena.VectorConfig{Descriptors: arena.Config{Budget: b}})
		if e != nil {
			t.Fatal(e)
		}
		b.FailAfter(fault)
		e = m.beginFormattingLocals(65)
		retained := e != nil && m.formatting.pending && m.formatting.locals[m.formatting.failedLocal].raw != (memory.Block{})
		b.FailAfter(-1)
		if retained {
			root := m.formatting.locals[m.formatting.failedLocal].raw
			q, r := [1]uint32{}, [1]uint32{}
			_, e = m.Div([]uint32{10}, []uint32{2}, q[:], r[:])
			if e != nil || q[0] != 5 || r[0] != 0 || m.formatting.locals[m.formatting.failedLocal].raw != root || !m.formatting.pending {
				t.Fatal("unrelated source division with retained lexical root", e, q, r)
			}
		}
		if m.formatting.active {
			if e = m.finishFormattingLocals(); e != nil {
				t.Fatal(e)
			}
		}
		m.Close()
		if b.Used() != 0 {
			t.Fatal("actual owning cleanup", b.Used())
		}
		if retained {
			return
		}
	}
	t.Fatal("no actual retained failed local")
}

func TestNaturalFormattingLeadingZeroSourceDomain(t *testing.T) {
	m := naturalManager(t)
	for _, c := range []struct {
		a    []uint32
		want string
	}{{[]uint32{0, 0}, "0"}, {[]uint32{1, 0}, "1"}, {[]uint32{0, 1, 0}, "4294967296"}} {
		got, e := m.String(c.a)
		if e != nil || got != c.want {
			t.Fatal(c.a, got, e)
		}
	}
}

func TestNaturalFormattingReusableInlineScopeWarm(t *testing.T) {
	m := naturalManager(t)
	input := []uint32{1, 2, 3, 4}
	dst := make([]byte, 100)
	if _, e := m.FormatInto(input, dst); e != nil {
		t.Fatal(e)
	}
	used := m.budget.Used()
	count, e := m.runtime.AllocationCount()
	if e != nil {
		t.Fatal(e)
	}
	var problem error
	allocations := testing.AllocsPerRun(100, func() { _, problem = m.FormatInto(input, dst) })
	after, e := m.runtime.AllocationCount()
	if problem != nil || e != nil || allocations != 0 || used != m.budget.Used() || count != after || m.words.Stats().Live != 0 || m.formatting.active || m.formatting.pending {
		t.Fatal("warm actual inline scope", problem, e, allocations, count, after)
	}
}
func TestNaturalFormattingEmptyGuardAfterLocalActions(t *testing.T) {
	m := naturalManager(t)
	if e := m.beginFormattingLocals(0); e != nil {
		t.Fatal(e)
	}
	if e := formattingReversalGuard(0); !errors.Is(e, arena.ErrBounds) || !m.formatting.active || m.formatting.completed != 7 || m.formatting.locals[1].pos != 1 || m.formatting.locals[2].pos != 1 {
		t.Fatal("guard while completed locals live", e, m.formatting)
	}
	// Source lexical exception unwind is subsequent to the guarded access.
	if e := m.finishFormattingLocals(); e != nil {
		t.Fatal(e)
	}
	dst := []byte{'!', '!'}
	_, e := m.FormatInto(nil, dst)
	if !errors.Is(e, arena.ErrBounds) || string(dst) != "!!" || m.formatting.active || m.formatting.pending || m.runtime.Stats().LiveBlocks != 0 {
		t.Fatal("explicit guard then unwind", e, dst)
	}
}

func TestNaturalSourceFinalizeRetainsFailedLexicalLocal(t *testing.T) {
	found := false
	for fault := int64(0); fault < 50; fault++ {
		b := arena.NewBudget(1 << 22)
		m, e := NewNaturalManager(arena.VectorConfig{Descriptors: arena.Config{Budget: b}})
		if e != nil {
			t.Fatal(e)
		}
		if e = m.division[0].resize(m.runtime, 33); e != nil {
			t.Fatal(e)
		}
		member := m.division[0].raw
		b.FailAfter(fault)
		e = m.beginFormattingLocals(65)
		if e == nil {
			if e = m.finishFormattingLocals(); e != nil {
				t.Fatal(e)
			}
		}
		retained := m.formatting.pending && m.formatting.locals[m.formatting.failedLocal].raw != (memory.Block{})
		b.FailAfter(-1)
		if retained {
			found = true
			failed := m.formatting.locals[m.formatting.failedLocal].raw
			if e = m.SourceFinalize(); e != nil {
				t.Fatal(e)
			}
			if !m.sourceFinalized || m.headerBytes == 0 || !m.formatting.pending || m.runtime.Stats().LiveBlocks != 1 {
				t.Fatal("source member destruction consumed failed local", m.runtime.Stats())
			}
			if _, e = m.runtime.ReadUint32(member, 0); e == nil {
				t.Fatal("published member survived")
			}
			if _, e = m.runtime.ReadUint32(failed, 0); e != nil {
				t.Fatal("failed ctor oldraw consumed", e)
			}
			state := m.runtime.State()
			if e = m.SourceFinalize(); e != nil || m.runtime.State() != state {
				t.Fatal("source repeat", e)
			}
			if e = m.check(); !errors.Is(e, arena.ErrClosed) {
				t.Fatal("source destroyed readiness", e)
			}
		}
		m.Close()
		if b.Used() != 0 {
			t.Fatal("owning physical cleanup", b.Used())
		}
		if found {
			break
		}
	}
	if !found {
		t.Fatal("no failed ctor root")
	}
}

func TestNaturalSourceFinalizeRetainsUnpublishedMemberAllocation(t *testing.T) {
	b := arena.NewBudget(1 << 23)
	rt, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64})
	if e != nil {
		t.Fatal(e)
	}
	c := arena.VectorConfig{Descriptors: arena.Config{Budget: b}}
	words, e := arena.NewWords(c)
	if e != nil {
		t.Fatal(e)
	}
	m, e := newNaturalManagerWithMemory(c, words, true, rt)
	if e != nil {
		t.Fatal(e)
	}
	if e = rt.SetMaxCount(1); e != nil {
		t.Fatal(e)
	}
	e = m.division[0].resize(rt, 40000)
	var refusal memory.Failure
	if !errors.As(e, &refusal) || refusal.Kind != memory.FailureAllocExceeded || m.division[0].raw == (memory.Block{}) || m.division[0].pending == (memory.Block{}) || rt.Stats().LiveBlocks != 2 {
		t.Fatal("actual late TLS allocation prefix", e, rt.Stats())
	}
	published, pending := m.division[0].raw, m.division[0].pending
	if e = m.SourceFinalize(); e != nil {
		t.Fatal(e)
	}
	if rt.Stats().LiveBlocks != 1 || m.division[0].pending != pending || m.headerBytes == 0 {
		t.Fatal("source destructor consumed orphan allocation", rt.Stats())
	}
	if _, e = rt.ReadUint32(published, 0); e == nil {
		t.Fatal("published buffer survived")
	}
	if _, e = rt.ReadUint32(pending, 0); !errors.Is(e, memory.ErrRuntimeUninitialized) {
		t.Fatal("actual pending digit init", e)
	}
	if e = m.CloseChecked(); e != nil {
		t.Fatal(e)
	}
	if rt.Stats().LiveBlocks != 0 || b.Used() != rt.Stats().TotalBytes {
		t.Fatal("owning roots", b.Used(), rt.Stats())
	}
	if e = rt.Close(); e != nil || b.Used() != 0 {
		t.Fatal(e, b.Used())
	}
}

func TestIntegerSourceFinalizeRetainsFailedNaturalFormatting(t *testing.T) {
	found := false
	for fault := int64(0); fault < 50; fault++ {
		b := arena.NewBudget(1 << 23)
		m, e := NewIntegerManager(IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: memory.SmallAllocatorProfile{PointerBits: 64}})
		if e != nil {
			t.Fatal(e)
		}
		n := m.natural
		b.FailAfter(fault)
		e = n.beginFormattingLocals(65)
		if e == nil {
			if e = n.finishFormattingLocals(); e != nil {
				t.Fatal(e)
			}
		}
		retained := n.formatting.pending && n.formatting.locals[n.formatting.failedLocal].raw != (memory.Block{})
		b.FailAfter(-1)
		if retained {
			found = true
			failed := n.formatting.locals[n.formatting.failedLocal].raw
			if e = m.SourceFinalize(); e != nil {
				t.Fatal(e)
			}
			if m.natural != n || !n.sourceFinalized || n.sourceStage != 4 || n.headerBytes == 0 || !n.formatting.pending || m.memory.Stats().LiveBlocks != 1 {
				t.Fatal("parent lost real lexical cleanup owner", m.memory.Stats())
			}
			if _, e = m.memory.ReadUint32(failed, 0); e != nil {
				t.Fatal("source allocator consumed independent failed local", e)
			}
			value := NewInt(1)
			if e = m.check(&value); !errors.Is(e, arena.ErrClosed) {
				t.Fatal("destroyed parent remains readable", e)
			}
			state := m.memory.State()
			used := b.Used()
			if e = m.SourceFinalize(); e != nil || state != m.memory.State() || used != b.Used() {
				t.Fatal("source repeated retained scope cleanup", e)
			}
		}
		if e = m.CloseChecked(); e != nil || b.Used() != 0 {
			t.Fatal("actual owning cleanup", e, b.Used())
		}
		if found {
			break
		}
	}
	if !found {
		t.Fatal("missing real failed constructor parent path")
	}
}

func TestNaturalSourceFinalizeColdLifecycleGuards(t *testing.T) {
	var absent *NaturalManager
	if e := absent.SourceFinalize(); !errors.Is(e, memory.ErrRuntimeOwner) {
		t.Fatal("nil owner", e)
	}
	m := naturalManager(t)
	copied := *m
	if e := copied.SourceFinalize(); !errors.Is(e, memory.ErrRuntimeOwner) {
		t.Fatal("copied owner", e)
	}
	// These scalar state fixtures test exact lifecycle entry guards. They do
	// not manufacture a physical allocation/deallocation refusal.
	m.originalCause = arena.ErrInjectedFailure
	if e := m.SourceFinalize(); !errors.Is(e, arena.ErrClosed) || m.sourceFinalizing {
		t.Fatal("failed constructor source body", e)
	}
	m.originalCause = nil
	m.closing = true
	if e := m.SourceFinalize(); !errors.Is(e, arena.ErrClosed) || m.sourceFinalizing {
		t.Fatal("half owned destructor source body", e)
	}
	if e := m.check(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("owning closing ordinary read", e)
	}
	m.closing = false
	if e := m.SourceFinalize(); e != nil {
		t.Fatal(e)
	}
	m.Close()
	if e := m.SourceFinalize(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("closed source owner", e)
	}
}
