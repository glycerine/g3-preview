// Source: src/util/mpz.cpp:283-303; actual first BIG digit allocation transport.
package util

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	"testing"
)

func TestIntegerStringSetSourceActualAllocationTransport(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	var m *ArbitraryFloatManager
	a := NewInt(7)
	armed, calls := false, 0
	r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f umemory.Failure) error {
		if armed {
			calls++
			z := m.integer
			s := &z.stringSet
			if !s.executing || !s.declared || s.endReached || s.stage != 1 || s.index != 11 || a != NewInt(1234567890) {
				t.Fatal("first BIG digit source prefix", s.index, s.stage, a)
			}
			tmp, old, inv, used, stage := s.tmp, a, s.invocation, b.Used(), s.stage
			for i, call := range []func() error{m.Close, m.SourceFinalize, z.CloseChecked, z.SourceFinalize, m.rational.CloseChecked, m.rational.SourceFinalize, z.CleanupStringSet, func() error { return z.SetString(&a, "1") }, func() error { return z.SetInt32(&a, 1) }, func() error { return z.CleanupRawNumeral(&a) }, func() error { return z.Del(&s.tmp) }} {
				if e := call(); !errors.Is(e, ErrNumericPending) {
					t.Fatal("actual string callback guard", i, e)
				}
			}
			if a != old || s.tmp != tmp || s.invocation != inv || s.stage != stage || b.Used() != used || s.endReached {
				t.Fatal("callback changed prefix")
			}
		}
		return f
	}})
	serviceMust(t, e)
	m, e = NewArbitraryFloatManager(IntegerConfig{Memory: r, Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	pad, e := r.Allocate(uint64(99990 - r.State().LocalSize - 8))
	serviceMust(t, e)
	serviceMust(t, r.SetMaxSize(1))
	armed = true
	body := m.integer.SetString(&a, "-12345678901234567890")
	armed = false
	if !errors.Is(body, umemory.Failure{Kind: umemory.FailureMemout}) || calls != 1 {
		t.Fatal("actual parse transport", body, calls)
	}
	integerPhysicalAuditWithRaw(t, m.integer, []umemory.Block{pad}, &a)
	if b.Used() != m.OwnedStorageBytes()+r.Stats().TotalBytes {
		t.Fatal("callback exact graph")
	}
	serviceMust(t, r.SetMaxSize(0))
	serviceMust(t, m.integer.CleanupStringSet())
	serviceMust(t, m.integer.CleanupRawNumeral(&a))
	serviceMust(t, m.Close())
	serviceMust(t, r.Deallocate(&pad))
	serviceMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("callback refunds")
	}
}

func TestIntegerStringSetSourceResetReleaseBeforeDeclarations(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	m, e := NewArbitraryFloatManager(IntegerConfig{Memory: r, Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	z := m.integer
	var a Int
	digits := make([]uint32, 125)
	digits[124] = 1
	serviceMust(t, z.SetLimbs(&a, digits))
	old := a
	contextTLSNearMinimum(t, r)
	lane, e := r.Allocate(8)
	serviceMust(t, e)
	delta := int64(-99950) - r.State().LocalSize
	if e = r.Reallocate(&lane, uint64(16)+uint64(delta)-8); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	body := z.SetString(&a, "-123")
	s := &z.stringSet
	if !errors.Is(body, arena.ErrBounds) || s.sourceCause == nil || a != old || !z.Cell(a.Cell).Released || s.declared || s.ten != NewInt(0) || s.tmp != NewInt(0) || s.endReached || s.index != 0 {
		t.Fatal("RESET release crossed declarations", body, a, s.declared)
	}
	used, state, inv := b.Used(), r.State(), s.invocation
	if e = z.SetString(&a, "1"); e != arena.ErrInvalidSlot || a != old || b.Used() != used || r.State() != state || s.invocation != inv {
		t.Fatal("released RESET replay", e)
	}
	if e = m.SourceFinalize(); !errors.Is(e, arena.ErrBounds) || m.sourceCause != s.sourceCause || z.SourceFinalizing || b.Used() != used {
		t.Fatal("RESET stop crossed older class", e)
	}
	integerPhysicalAuditWithRaw(t, z, []umemory.Block{lane}, &a)
	if b.Used() != m.OwnedStorageBytes()+r.Stats().TotalBytes {
		t.Fatal("RESET stopped graph")
	}
	recovery, e := r.Allocate(200000)
	serviceMust(t, e)
	serviceMust(t, z.CleanupStringSet())
	if a != old || s.declared || s.endReached {
		t.Fatal("owning cleanup stole input or invented declarations")
	}
	serviceMust(t, z.CleanupRawNumeral(&a))
	contextTLSFinish(t, r, m.Close)
	for _, block := range []*umemory.Block{&lane, &recovery} {
		contextTLSFinish(t, r, func() error {
			if *block == (umemory.Block{}) {
				return nil
			}
			return r.Deallocate(block)
		})
	}
	contextTLSFinish(t, r, r.Close)
	if b.Used() != 0 {
		t.Fatal("RESET source refunds")
	}
}
