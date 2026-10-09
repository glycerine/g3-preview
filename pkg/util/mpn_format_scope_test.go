// Source mpn.cpp:371 and buffer.h:84-95; independent bounded lifetime oracle.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
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
		s := &m.Formatting.Locals[i]
		if uint32(len(*s)) != n {
			t.Fatal(i, len(*s), n)
		}
		for j := uint32(0); j < n; j++ {
			if v := (*s)[j]; v != 0 {
				t.Fatal(i, j, v)
			}
		}
	}
	if m.runtime.Stats().LiveBlocks != before.LiveBlocks {
		t.Fatal("actual lexical roots")
	}
	if e = m.finishFormattingLocals(); e != nil {
		t.Fatal(e)
	}
	if m.runtime.Stats().LiveBlocks != before.LiveBlocks || m.Formatting.Active || m.Formatting.PendingField {
		t.Fatal("completed reverse destruction")
	}
	m.Close()
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
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

func TestNaturalFormattingReusableNativeScopeWarm(t *testing.T) {
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
	if problem != nil || e != nil || allocations != 0 || used != m.budget.Used() || count != after || m.Formatting.Active || m.Formatting.PendingField {
		t.Fatal("warm reusable native scope", problem, e, allocations, count, after)
	}
}
func TestNaturalFormattingEmptyGuardAfterLocalActions(t *testing.T) {
	m := naturalManager(t)
	if e := m.beginFormattingLocals(0); e != nil {
		t.Fatal(e)
	}
	if e := formattingReversalGuard(0); !errors.Is(e, arena.ErrBounds) || !m.Formatting.Active || m.Formatting.CompletedField != 7 || len(m.Formatting.Locals[1]) != 1 || len(m.Formatting.Locals[2]) != 1 {
		t.Fatal("guard while completed locals live", e, m.Formatting)
	}
	// Source lexical exception unwind is subsequent to the guarded access.
	if e := m.finishFormattingLocals(); e != nil {
		t.Fatal(e)
	}
	dst := []byte{'!', '!'}
	_, e := m.FormatInto(nil, dst)
	if !errors.Is(e, arena.ErrBounds) || string(dst) != "!!" || m.Formatting.Active || m.Formatting.PendingField || m.runtime.Stats().LiveBlocks != 0 {
		t.Fatal("explicit guard then unwind", e, dst)
	}
}

func TestNaturalSourceFinalizeColdLifecycleGuards(t *testing.T) {
	var absent *NaturalManager
	if e := absent.SourceFinalize(); !errors.Is(e, ErrRuntimeOwner) {
		t.Fatal("nil owner", e)
	}
	m := naturalManager(t)
	copied := *m
	if e := copied.SourceFinalize(); !errors.Is(e, ErrRuntimeOwner) {
		t.Fatal("copied owner", e)
	}
	// These scalar state fixtures test exact lifecycle entry guards. They do
	// not manufacture a physical allocation/deallocation refusal.
	m.originalCause = arena.ErrInjectedFailure
	if e := m.SourceFinalize(); !errors.Is(e, arena.ErrClosed) || m.SourceFinalizing {
		t.Fatal("failed constructor source body", e)
	}
	m.originalCause = nil
	m.Closing = true
	if e := m.SourceFinalize(); !errors.Is(e, arena.ErrClosed) || m.SourceFinalizing {
		t.Fatal("half owned destructor source body", e)
	}
	if e := m.check(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("owning closing ordinary read", e)
	}
	m.Closing = false
	if e := m.SourceFinalize(); e != nil {
		t.Fatal(e)
	}
	m.Close()
	if e := m.SourceFinalize(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("closed source owner", e)
	}
}

// Reuse must clear old scratch, and a guarded output failure must leave the
// formatter ready for the next invocation. Runtime allocation faults do not
// apply to native scratch storage.
func TestNaturalFormattingNativeScratchReuseAndCleanup(t *testing.T) {
	m := naturalManager(t)
	before := m.runtime.State()
	used := m.budget.Used()
	m.budget.FailAfter(0)
	for _, n := range []uint32{65, 2, 129, 33} {
		serviceMust(t, m.beginFormattingLocals(n))
		for _, words := range m.Formatting.Locals {
			for i, v := range words {
				if v != 0 {
					t.Fatal("stale scratch", n, i, v)
				}
				words[i] = 0xffffffff
			}
		}
		serviceMust(t, m.finishFormattingLocals())
	}
	if m.runtime.State() != before || m.budget.Used() != used {
		t.Fatal("native scratch changed runtime accounting")
	}
	serviceMust(t, m.CleanupFormatting())
	for _, words := range m.Formatting.Locals {
		if words != nil {
			t.Fatal("explicit cleanup retained capacity")
		}
	}
	// Bounded destination writes fail after inner scratch construction.
	e := m.sourceToCString(limbView{external: []uint32{0, 1}, size: 2}, sourceDecimalDestination{external: []byte{'!'}}, 100)
	if !errors.Is(e, arena.ErrBounds) || m.Formatting.Active || m.Formatting.PendingField {
		t.Fatal("failure cleanup", e)
	}
	var dst [32]byte
	serviceMust(t, m.sourceToCString(limbView{external: []uint32{0, 1}, size: 2}, sourceDecimalDestination{external: dst[:]}, uint64(len(dst))))
	if string(dst[:10]) != "4294967296" || dst[10] != 0 {
		t.Fatal(dst)
	}
	m.budget.FailAfter(-1)
}
